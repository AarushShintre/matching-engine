package strategy

import "math"

// Decide is a pure function of (config, state, event[, nextIDs]).
// nextBid/nextAsk are pre-allocated order IDs for quote actions (from IDGen).
func Decide(cfg Config, st State, ev Event, nextBid, nextAsk OrderID) Decision {
	cfg = cfg.WithDefaults()

	if ev.Kind == EventShutdown {
		return shutdownDecision(st)
	}

	if !cfg.Enabled {
		return Decision{Action: ActionHold}
	}

	if ev.Kind == EventGap {
		return Decision{Action: ActionPause}
	}

	if ev.Kind != EventTrade || ev.Trade == nil {
		return Decision{Action: ActionHold}
	}

	t := *ev.Trade
	if t.Symbol != "" && cfg.Symbol != "" && t.Symbol != cfg.Symbol {
		return Decision{Action: ActionHold}
	}

	// Apply reference update from this trade into a working copy for the decision.
	ref := st.Ref
	price := t.Price
	ref.LastTradePrice = &price
	ref.Value = price
	ref.Source = RefLastTrade
	ref.Valid = true

	ownFill := IsOwnFill(t, st.Owned)

	activity := len(st.TradeCursor)
	if cfg.SpreadMode == SpreadDynamic {
		// Include this trade in the activity window for spread computation.
		activity = activity + 1
		if cfg.ActivityWindow > 0 && activity > cfg.ActivityWindow {
			activity = cfg.ActivityWindow
		}
	}
	s := HalfSpread(cfg, activity)
	if s < 1 {
		return Decision{Action: ActionHold}
	}

	hasQuotes := st.Quotes.State == QuoteResting || st.Quotes.State == QuotePending ||
		st.Quotes.BidOrderID != nil || st.Quotes.AskOrderID != nil

	if !hasQuotes {
		return quoteDecision(ActionQuoteInitial, cfg, ref.Value, s, nextBid, nextAsk, nil)
	}

	// Own-fill: hold for requote even if |ΔL| meets threshold (FR-013).
	if ownFill {
		return Decision{Action: ActionHold}
	}

	delta := absInt(ref.Value - st.Quotes.CenteredOn)
	if delta < cfg.MovementThreshold {
		return Decision{Action: ActionHold}
	}

	cancels := OwnedIDs(st.Quotes)
	return quoteDecision(ActionRequote, cfg, ref.Value, s, nextBid, nextAsk, cancels)
}

// DecideWithSeed evaluates startup/seed path when no trade event yet.
// Call with Event{Kind: EventTrade, Trade: nil} is not used; instead use SeedTick.
func SeedTick(cfg Config, st State, nextBid, nextAsk OrderID) Decision {
	cfg = cfg.WithDefaults()
	if !cfg.Enabled {
		return Decision{Action: ActionHold}
	}
	if st.Ref.Valid || (st.Quotes.BidOrderID != nil || st.Quotes.AskOrderID != nil) {
		return Decision{Action: ActionHold}
	}
	if cfg.SeedReference == nil {
		return Decision{Action: ActionHold}
	}
	s := HalfSpread(cfg, len(st.TradeCursor))
	if s < 1 {
		return Decision{Action: ActionHold}
	}
	return quoteDecision(ActionQuoteInitial, cfg, *cfg.SeedReference, s, nextBid, nextAsk, nil)
}

func shutdownDecision(st State) Decision {
	return Decision{
		Action:    ActionShutdownCancel,
		CancelIDs: OwnedIDs(st.Quotes),
	}
}

func quoteDecision(action Action, cfg Config, ref, s int, bidID, askID OrderID, cancels []OrderID) Decision {
	bidPx, askPx := QuotePrices(ref, s)
	refCopy := ref
	sCopy := s
	return Decision{
		Action:     action,
		Reference:  &refCopy,
		HalfSpread: &sCopy,
		CancelIDs:  cancels,
		NewOrders: []NewLimitOrder{
			{Symbol: cfg.Symbol, OrderID: bidID, Side: "buy", Price: bidPx, Quantity: cfg.QuoteSize},
			{Symbol: cfg.Symbol, OrderID: askID, Side: "sell", Price: askPx, Quantity: cfg.QuoteSize},
		},
	}
}

func absInt(x int) int {
	return int(math.Abs(float64(x)))
}

// ApplyDecision updates state after a decision is computed (before/after ingress).
// For trade events, pass the trade so reference and activity window advance.
func ApplyDecision(st *State, cfg Config, d Decision, ev Event) {
	cfg = cfg.WithDefaults()

	switch d.Action {
	case ActionPause:
		st.Phase = PhasePaused
		st.Ref.Valid = false
		return
	case ActionShutdownCancel:
		st.Phase = PhaseStopped
		st.Quotes = QuoteSet{State: QuoteEmpty}
		return
	}

	if ev.Kind == EventTrade && ev.Trade != nil {
		price := ev.Trade.Price
		st.Ref.LastTradePrice = &price
		st.Ref.Value = price
		st.Ref.Source = RefLastTrade
		st.Ref.Valid = true
		if st.Phase == PhasePaused || st.Phase == PhaseWaiting {
			st.Phase = PhaseQuoting
		}
		// Advance activity cursor (event-count based).
		st.TradeCursor = append(st.TradeCursor, struct{}{})
		if cfg.ActivityWindow > 0 && len(st.TradeCursor) > cfg.ActivityWindow {
			st.TradeCursor = st.TradeCursor[len(st.TradeCursor)-cfg.ActivityWindow:]
		}
		MarkFilled(st.Owned, *ev.Trade)
		// Clear filled ids from QuoteSet if they match.
		if ev.Trade.RestingOrderID != nil || ev.Trade.AggressorOrderID != nil {
			clearIfFilled := func(id **OrderID) {
				if *id == nil {
					return
				}
				if o, ok := st.Owned[**id]; ok && o.Status == OwnedFilled {
					*id = nil
				}
			}
			clearIfFilled(&st.Quotes.BidOrderID)
			clearIfFilled(&st.Quotes.AskOrderID)
		}
	}

	switch d.Action {
	case ActionHold:
		if !st.Ref.Valid && st.Phase != PhasePaused {
			st.Phase = PhaseWaiting
		}
	case ActionQuoteInitial, ActionRequote:
		if d.Reference != nil {
			st.Quotes.CenteredOn = *d.Reference
			st.Ref.Value = *d.Reference
			st.Ref.Valid = true
			if st.Ref.Source == "" {
				st.Ref.Source = RefSeed
			}
		}
		if d.HalfSpread != nil {
			st.Quotes.HalfSpreadUsed = *d.HalfSpread
		}
		st.Quotes.BidSize = cfg.QuoteSize
		st.Quotes.AskSize = cfg.QuoteSize
		if len(d.NewOrders) >= 2 {
			st.Quotes.BidPrice = d.NewOrders[0].Price
			st.Quotes.AskPrice = d.NewOrders[1].Price
			bid := d.NewOrders[0].OrderID
			ask := d.NewOrders[1].OrderID
			st.Quotes.BidOrderID = &bid
			st.Quotes.AskOrderID = &ask
			st.Quotes.State = QuoteResting
			if st.Owned == nil {
				st.Owned = make(map[OrderID]OwnedOrder)
			}
			// Drop canceled/filled prior ids from owned map for clean attribution.
			for _, id := range d.CancelIDs {
				delete(st.Owned, id)
			}
			st.Owned[bid] = OwnedOrder{OrderID: bid, Side: SideBid, Price: d.NewOrders[0].Price, Size: cfg.QuoteSize, Status: OwnedResting}
			st.Owned[ask] = OwnedOrder{OrderID: ask, Side: SideAsk, Price: d.NewOrders[1].Price, Size: cfg.QuoteSize, Status: OwnedResting}
		}
		st.Phase = PhaseQuoting
	}
}

// ApplySeedDecision applies an initial seed quote decision.
func ApplySeedDecision(st *State, cfg Config, d Decision) {
	if d.Action != ActionQuoteInitial {
		return
	}
	st.Ref.Source = RefSeed
	st.Ref.Valid = true
	if d.Reference != nil {
		st.Ref.Value = *d.Reference
	}
	ApplyDecision(st, cfg, d, Event{})
}
