package book

import (
	"slices"
)

// SubmitLimit matches a limit against the opposite side, then rests any remainder.
//
// TODO(spec-1): DONE reject non-positive price/qty and duplicate order ids (FR-012)
// without mutating the book; return ErrRejected.
func (b *Book) SubmitLimit(o LimitOrder) (Result, error) {
	// Spec 5 trade and depth events are emitted by the ingest matcher owner,
	// after this operation returns; the book remains consumer-independent.
	if o.Price <= 0 || o.Quantity <= 0 {
		return Result{Accepted: false}, nil
	}
	if b.findResting(o.ID) != nil {
		return Result{Accepted: false}, nil
	}
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

// SubmitMarket matches a market order against available opposite liquidity.
// Unfilled quantity is discarded and must not rest (FR-010).
func (b *Book) SubmitMarket(o MarketOrder) (Result, error) {
	qtyLeft := o.Quantity
	var trades []Trade

	if o.Side == Buy {
		for qtyLeft > 0 && len(b.AskPrices) > 0 {
			bestPrice := b.AskPrices[0]
			level := b.Asks[bestPrice]
			resting := level.front()

			fill := qtyLeft
			if resting.Quantity < fill {
				fill = resting.Quantity
			}

			trades = append(trades, Trade{
				MakerID:  resting.ID,
				TakerID:  o.ID,
				Price:    bestPrice,
				Quantity: fill,
			})

			qtyLeft -= fill
			resting.Quantity -= fill

			if resting.Quantity == 0 {
				level.popFront()
			}
			if level.empty() {
				b.removeAskLevel(bestPrice)
			}
		}
	} else {
		for qtyLeft > 0 && len(b.BidPrices) > 0 {
			bestPrice := b.BidPrices[0]
			level := b.Bids[bestPrice]
			resting := level.front()

			fill := qtyLeft
			if resting.Quantity < fill {
				fill = resting.Quantity
			}

			trades = append(trades, Trade{
				MakerID:  resting.ID,
				TakerID:  o.ID,
				Price:    bestPrice,
				Quantity: fill,
			})

			qtyLeft -= fill
			resting.Quantity -= fill

			if resting.Quantity == 0 {
				level.popFront()
			}
			if level.empty() {
				b.removeBidLevel(bestPrice)
			}
		}
	}

	// qtyLeft (if any) is discarded — market orders do not rest.
	return Result{Accepted: true, Trades: trades, Remaining: 0}, nil
}

// Cancel removes a resting order by id (FR-011).
// TODO(spec-1): DONE locate id in bid/ask FIFO queues, remove it, drop empty price levels;
// unknown/inactive id → Accepted=false (book unchanged), not an error.
func (b *Book) Cancel(orderID int) (Result, error) {
	// walk through the each price level, and each order at that price level
	// if order ID found, you can delete the order from that price level
	// if the price level has no other order, you can delete the price level from the book

	for i, price := range b.BidPrices {
		lvl := b.Bids[price]
		for j, order := range lvl.Orders {
			if order.ID == orderID {
				lvl.Orders = slices.Delete(lvl.Orders, j, j+1)
				if len(lvl.Orders) == 0 {
					delete(b.Bids, price)
					b.BidPrices = slices.Delete(b.BidPrices, i, i+1)
				}
				return Result{Accepted: true}, nil
			}
		}
	}
	for i, price := range b.AskPrices {
		lvl := b.Asks[price]
		for j, order := range lvl.Orders {
			if order.ID == orderID {
				lvl.Orders = slices.Delete(lvl.Orders, j, j+1)
				if len(lvl.Orders) == 0 {
					delete(b.Asks, price)
					b.AskPrices = slices.Delete(b.AskPrices, i, i+1)
				}
				return Result{Accepted: true}, nil
			}
		}
	}

	return Result{Accepted: false}, nil
}

// Snapshot returns price levels and FIFO order for harness checks (FR-014).
//
// TODO(spec-1): DONE ensure stable ordering and quantities match post-op book state
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
