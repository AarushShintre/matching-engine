package strategy

import "testing"

func fixedCfg(seed *int) Config {
	return Config{
		Symbol: "DEMO", QuoteSize: 10, SpreadMode: SpreadFixed,
		FixedHalfSpread: 5, MovementThreshold: 1, Enabled: true,
		SeedReference: seed, OrderIDNamespace: 1,
	}
}

func TestInitialQuotesFixedSpread(t *testing.T) {
	cfg := fixedCfg(nil)
	rec := &IngressRecorder{}
	r, err := NewRunner(cfg, rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 100, 1))}); err != nil {
		t.Fatal(err)
	}
	if rec.CountNew() != 2 {
		t.Fatalf("want 2 new orders, got %d", rec.CountNew())
	}
	orders := rec.NewOrders()
	if orders[0].Side != "buy" || orders[0].Price != 95 || orders[0].Quantity != 10 {
		t.Fatalf("bid want buy 95x10, got %+v", orders[0])
	}
	if orders[1].Side != "sell" || orders[1].Price != 105 || orders[1].Quantity != 10 {
		t.Fatalf("ask want sell 105x10, got %+v", orders[1])
	}
	if rec.CountCancel() != 0 {
		t.Fatalf("unexpected cancels: %d", rec.CountCancel())
	}
}

func TestNoQuotesWithoutReference(t *testing.T) {
	cfg := fixedCfg(nil)
	st := NewState()
	d := SeedTick(cfg, st, 1, 2)
	if d.Action != ActionHold {
		t.Fatalf("want hold without seed/trade, got %s", d.Action)
	}
	rec := &IngressRecorder{}
	r, err := NewRunner(cfg, rec)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.StartSeed()
	if rec.CountNew() != 0 {
		t.Fatalf("want zero ingress, got %d", rec.CountNew())
	}
}

func TestDisabledConfigNoQuotes(t *testing.T) {
	seed := 100
	cfg := fixedCfg(&seed)
	cfg.Enabled = false
	rec := &IngressRecorder{}
	r, err := NewRunner(cfg, rec)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.StartSeed()
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 100, 1))})
	if rec.CountNew() != 0 || rec.CountCancel() != 0 {
		t.Fatalf("disabled strategy must not submit; new=%d cancel=%d", rec.CountNew(), rec.CountCancel())
	}
	d := Decide(cfg, NewState(), Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 100, 1))}, 1, 2)
	if d.Action != ActionHold {
		t.Fatalf("want hold when disabled, got %s", d.Action)
	}
}

func TestSeedReferenceUntilFirstTrade(t *testing.T) {
	seed := 100
	cfg := fixedCfg(&seed)
	rec := &IngressRecorder{}
	r, err := NewRunner(cfg, rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.StartSeed(); err != nil {
		t.Fatal(err)
	}
	if rec.CountNew() != 2 {
		t.Fatalf("seed should quote, got %d news", rec.CountNew())
	}
	if r.State.Ref.Source != RefSeed {
		t.Fatalf("want seed source, got %s", r.State.Ref.Source)
	}
	orders := rec.NewOrders()
	if orders[0].Price != 95 || orders[1].Price != 105 {
		t.Fatalf("seed quotes around 100: got %d / %d", orders[0].Price, orders[1].Price)
	}
	// First trade takes over reference; threshold move triggers requote.
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 110, 1))})
	if r.State.Ref.Source != RefLastTrade {
		t.Fatalf("want last_trade after first trade, got %s", r.State.Ref.Source)
	}
	if r.State.Quotes.CenteredOn != 110 {
		t.Fatalf("want centered on 110, got %d", r.State.Quotes.CenteredOn)
	}
}

func TestRequoteOnThreshold(t *testing.T) {
	cfg := fixedCfg(nil)
	rec := &IngressRecorder{}
	r, err := NewRunner(cfg, rec)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 100, 1))})
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 110, 1))})
	if rec.CountCancel() < 2 {
		t.Fatalf("want cancels on requote, got %d", rec.CountCancel())
	}
	orders := rec.NewOrders()
	if len(orders) < 4 {
		t.Fatalf("want initial+requote news, got %d", len(orders))
	}
	lastBid, lastAsk := orders[len(orders)-2], orders[len(orders)-1]
	if lastBid.Price != 105 || lastAsk.Price != 115 {
		t.Fatalf("requote around 110 S=5: want 105/115, got %d/%d", lastBid.Price, lastAsk.Price)
	}
}

func TestHoldBelowMovementThreshold(t *testing.T) {
	cfg := fixedCfg(nil)
	cfg.MovementThreshold = 5
	rec := &IngressRecorder{}
	r, err := NewRunner(cfg, rec)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 100, 1))})
	newsBefore := rec.CountNew()
	cancelsBefore := rec.CountCancel()
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 101, 1))})
	if rec.CountNew() != newsBefore || rec.CountCancel() != cancelsBefore {
		t.Fatal("sub-threshold move must hold quotes")
	}
}

func TestIgnoreOwnFillLastTradeForRequote(t *testing.T) {
	cfg := fixedCfg(nil)
	rec := &IngressRecorder{}
	r, err := NewRunner(cfg, rec)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 100, 1))})
	bidID := *r.State.Quotes.BidOrderID
	newsBefore := rec.CountNew()
	cancelsBefore := rec.CountCancel()
	// Large price move but attributable only to own resting fill.
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(OwnFillTrade("DEMO", 120, 1, bidID))})
	if rec.CountNew() != newsBefore || rec.CountCancel() != cancelsBefore {
		t.Fatal("own-fill last trade must not trigger requote")
	}
	if r.State.Quotes.CenteredOn != 100 {
		t.Fatalf("centered_on should remain 100, got %d", r.State.Quotes.CenteredOn)
	}
}

func TestRequoteAfterPartialFill(t *testing.T) {
	cfg := fixedCfg(nil)
	rec := &IngressRecorder{}
	r, err := NewRunner(cfg, rec)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 100, 1))})
	bidID := *r.State.Quotes.BidOrderID
	askID := *r.State.Quotes.AskOrderID
	// Partial: mark bid filled via own-fill (hold), then external move requotes remaining.
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(OwnFillTrade("DEMO", 100, 5, bidID))})
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 110, 1))})
	// Should cancel remaining owned (ask at least) and post fresh two-sided set.
	if rec.CountCancel() < 1 {
		t.Fatal("expected cancel of remaining owned order(s)")
	}
	foundAskCancel := false
	for _, s := range rec.Submitted {
		if c, ok := s.(CancelOrder); ok && c.OrderID == askID {
			foundAskCancel = true
		}
	}
	if !foundAskCancel {
		t.Fatal("expected cancel of remaining ask")
	}
	orders := rec.NewOrders()
	lastBid, lastAsk := orders[len(orders)-2], orders[len(orders)-1]
	if lastBid.Quantity != 10 || lastAsk.Quantity != 10 {
		t.Fatal("fresh two-sided set must use full QuoteSize (no inventory skew)")
	}
}

func ptrTrade(t TradeEvent) *TradeEvent { return &t }
