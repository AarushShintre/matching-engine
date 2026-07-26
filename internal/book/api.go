package book

// SubmitLimit matches a limit against the opposite side, then rests any remainder.
//
// TODO(spec-1): reject non-positive price/qty and duplicate order ids (FR-012)
// without mutating the book; return ErrRejected.
func (b *Book) SubmitLimit(o LimitOrder) (Result, error) {
	// TODO(spec-5): after this op, emit trade + book-depth events from the matcher owner
	// (not from clients). Hook lives in ingest once Spec 2 owns the writer.
	trades := b.matchLimit(Order{
		ID:       o.ID,
		Side:     o.Side,
		Type:     TypeLimit,
		Price:    o.Price,
		Quantity: o.Quantity,
	})
	rem := 0
	if lvl := b.findResting(o.ID); lvl != nil {
		rem = lvl.Quantity
	}
	return Result{Accepted: true, Trades: trades, Remaining: rem}, nil
}

// SubmitMarket matches a market order; unfilled qty must not rest (FR-010).
//
// TODO(spec-1): market matching, walk opposite book, discard unfilled remainder (no rest).
func (b *Book) SubmitMarket(o MarketOrder) (Result, error) {
	_ = o
	return Result{}, ErrNotImplemented
}

// Cancel removes a resting order by id (FR-011).
//
// TODO(spec-1): locate id in bid/ask FIFO queues, remove it, drop empty price levels;
// unknown/inactive id → Accepted=false (book unchanged), not an error.
func (b *Book) Cancel(orderID int) (Result, error) {
	_ = orderID
	return Result{}, ErrNotImplemented
}

// Snapshot returns price levels and FIFO order for harness checks (FR-014).
//
// TODO(spec-1): ensure stable ordering and quantities match post-op book state
// for every scenario (including empty levels omitted).
func (b *Book) Snapshot() BookSnapshot {
	out := BookSnapshot{}
	for _, px := range b.BidPrices {
		lvl := b.Bids[px]
		if lvl == nil || lvl.empty() {
			continue
		}
		ls := LevelSnapshot{Price: px}
		for _, o := range lvl.Orders {
			ls.Orders = append(ls.Orders, OrderRef{ID: o.ID, Quantity: o.Quantity})
		}
		out.Bids = append(out.Bids, ls)
	}
	for _, px := range b.AskPrices {
		lvl := b.Asks[px]
		if lvl == nil || lvl.empty() {
			continue
		}
		ls := LevelSnapshot{Price: px}
		for _, o := range lvl.Orders {
			ls.Orders = append(ls.Orders, OrderRef{ID: o.ID, Quantity: o.Quantity})
		}
		out.Asks = append(out.Asks, ls)
	}
	return out
}

func (b *Book) findResting(id int) *Order {
	for _, px := range b.BidPrices {
		if lvl := b.Bids[px]; lvl != nil {
			for _, o := range lvl.Orders {
				if o.ID == id {
					return o
				}
			}
		}
	}
	for _, px := range b.AskPrices {
		if lvl := b.Asks[px]; lvl != nil {
			for _, o := range lvl.Orders {
				if o.ID == id {
					return o
				}
			}
		}
	}
	return nil
}
