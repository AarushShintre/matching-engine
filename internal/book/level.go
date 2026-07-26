package book

// PriceLevel holds the FIFO queue of resting orders at one price.
type PriceLevel struct {
	Orders []*Order
}

func (l *PriceLevel) enqueue(o *Order) {
	l.Orders = append(l.Orders, o)
}

func (l *PriceLevel) front() *Order {
	return l.Orders[0]
}

func (l *PriceLevel) popFront() {
	l.Orders = l.Orders[1:]
}

func (l *PriceLevel) empty() bool {
	return len(l.Orders) == 0
}
