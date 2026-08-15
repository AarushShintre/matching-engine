package book

import "sort"

// AskPrices is sorted ascending (best ask first).
// BidPrices is sorted descending (best bid first).
type Book struct {
	Asks      map[int]*PriceLevel
	Bids      map[int]*PriceLevel
	AskPrices []int
	BidPrices []int
}

func New() *Book {
	return &Book{
		Asks: make(map[int]*PriceLevel),
		Bids: make(map[int]*PriceLevel),
	}
}

func (b *Book) insertAskPrice(price int) {
	i := sort.Search(len(b.AskPrices), func(i int) bool {
		return b.AskPrices[i] >= price
	})

	if i < len(b.AskPrices) && b.AskPrices[i] == price {
		return
	}

	b.AskPrices = append(b.AskPrices, 0)
	copy(b.AskPrices[i+1:], b.AskPrices[i:])
	b.AskPrices[i] = price
}

func (b *Book) insertBidPrice(price int) {
	i := sort.Search(len(b.BidPrices), func(i int) bool {
		return b.BidPrices[i] <= price
	})

	if i < len(b.BidPrices) && b.BidPrices[i] == price {
		return
	}

	b.BidPrices = append(b.BidPrices, 0)
	copy(b.BidPrices[i+1:], b.BidPrices[i:])
	b.BidPrices[i] = price
}

func (b *Book) rest(order *Order) {
	if order.Side == Buy {
		b.insertBidPrice(order.Price)
		if b.Bids[order.Price] == nil {
			b.Bids[order.Price] = &PriceLevel{}
		}
		b.Bids[order.Price].enqueue(order)
		return
	}

	b.insertAskPrice(order.Price)
	if b.Asks[order.Price] == nil {
		b.Asks[order.Price] = &PriceLevel{}
	}
	b.Asks[order.Price].enqueue(order)
}	

// Edit #1: Bug Fix in removal of price levels
func (b *Book) removeAskLevel(price int) {
	delete(b.Asks, price)
	i := sort.SearchInts(b.AskPrices, price)
	if i < len(b.AskPrices) && b.AskPrices[i] == price {
		b.AskPrices = append(b.AskPrices[:i], b.AskPrices[i+1:]...)
	}
}


func (b *Book) removeBidLevel(price int) {
	delete(b.Bids, price)
	i := sort.SearchInts(b.BidPrices, price)
	if i < len(b.BidPrices) && b.BidPrices[i] == price {
		b.BidPrices = append(b.BidPrices[:i], b.BidPrices[i+1:]...)
	}
}
