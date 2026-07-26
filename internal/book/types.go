package book

import (
	"errors"
	"time"
)

var (
	// ErrNotImplemented marks Spec 1 endpoints still left for you to fill in.
	ErrNotImplemented = errors.New("book: not implemented")
	// ErrRejected is returned when a submit/cancel fails validation.
	ErrRejected = errors.New("book: rejected")
)

type Side int

const (
	Buy Side = iota
	Sell
)

type OrderType int

const (
	TypeLimit OrderType = iota
	TypeMarket
)

type OrderStatus int

const (
	StatusResting OrderStatus = iota
	StatusFilled
	StatusCanceled
	StatusRejected
	StatusPartial // remaining qty still live (aggressor or resting)
)

// Order is the internal resting/working order representation.
type Order struct {
	ID        int
	Side      Side
	Type      OrderType
	Price     int // limit price; unused for market
	Quantity  int // remaining quantity
	Timestamp time.Time
}

// LimitOrder is the Spec 1 public submit for a priced order.
type LimitOrder struct {
	ID       int
	Side     Side
	Price    int
	Quantity int
}

// MarketOrder is the Spec 1 public submit for an unpriced aggressor.
type MarketOrder struct {
	ID       int
	Side     Side
	Quantity int
}

// Trade is one fill (FR-013). Maker is the resting order; Taker is the aggressor.
type Trade struct {
	Quantity int
	Price    int
	MakerID  int
	TakerID  int
}

// Result is the outcome of one book operation.
type Result struct {
	Accepted bool
	Trades   []Trade
	// Remaining is aggressor qty left after the op (0 if fully filled / canceled / rejected).
	Remaining int
}

// LevelSnapshot is FIFO order ids + sizes at one price.
type LevelSnapshot struct {
	Price  int
	Orders []OrderRef
}

// OrderRef is enough of a resting order for harness assertions.
type OrderRef struct {
	ID       int
	Quantity int
}

// BookSnapshot is a read-only view for scenario tests (FR-014).
type BookSnapshot struct {
	Bids []LevelSnapshot // best bid first (high → low)
	Asks []LevelSnapshot // best ask first (low → high)
}
