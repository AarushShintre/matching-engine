package book

import "time"

type Side int

const (
	Buy Side = iota
	Sell
)

type Order struct {
	ID        int
	Side      Side
	Price     int
	Quantity  int
	Timestamp time.Time
}

type Trade struct {
	BuyOrderID  int
	SellOrderID int
	Price       int
	Quantity    int
}
