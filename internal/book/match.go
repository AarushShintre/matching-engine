package book

import "time"

// matchLimit is the existing price-time matching loop for limits.
// Prefer SubmitLimit as the public endpoint.
func (book *Book) matchLimit(order Order) (trades []Trade) {
	if order.Side == Buy {
		for order.Quantity > 0 &&
			len(book.AskPrices) > 0 &&
			order.Price >= book.AskPrices[0] {

			bestPrice := book.AskPrices[0]
			level := book.Asks[bestPrice]
			resting := level.front()

			qty := order.Quantity
			if resting.Quantity < qty {
				qty = resting.Quantity
			}

			trades = append(trades, Trade{
				MakerID:  resting.ID,
				TakerID:  order.ID,
				Price:    bestPrice,
				Quantity: qty,
			})

			order.Quantity -= qty
			resting.Quantity -= qty

			if resting.Quantity == 0 {
				level.popFront()
			}
			if level.empty() {
				book.removeAskLevel(bestPrice)
			}
		}
	} else {
		for order.Quantity > 0 &&
			len(book.BidPrices) > 0 &&
			order.Price <= book.BidPrices[0] {

			bestPrice := book.BidPrices[0]
			level := book.Bids[bestPrice]
			resting := level.front()

			qty := order.Quantity
			if resting.Quantity < qty {
				qty = resting.Quantity
			}

			trades = append(trades, Trade{
				MakerID:  resting.ID,
				TakerID:  order.ID,
				Price:    bestPrice,
				Quantity: qty,
			})

			order.Quantity -= qty
			resting.Quantity -= qty

			if resting.Quantity == 0 {
				level.popFront()
			}
			if level.empty() {
				book.removeBidLevel(bestPrice)
			}
		}
	}

	if order.Quantity > 0 {
		order.Timestamp = time.Now()
		resting := order
		book.rest(&resting)
	}

	return trades
}
