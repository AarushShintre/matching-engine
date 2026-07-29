package ingest

import "github.com/AarushShintre/matching-engine/internal/book"

// OrderID is a client-visible order identifier (Spec 2 / Spec 6).
type OrderID uint64

// NewLimitOrder is a Spec 2 new-limit ingress operation.
type NewLimitOrder struct {
	Symbol   string
	OrderID  OrderID
	Side     string // "buy" | "sell"
	Price    int
	Quantity int
}

// MarketOrder is a Spec 2 market ingress operation.
type MarketOrder struct {
	Symbol   string
	OrderID  OrderID
	Side     string // "buy" | "sell"
	Quantity int
}

// CancelOrder is a Spec 2 cancel ingress operation.
type CancelOrder struct {
	Symbol  string
	OrderID OrderID
}

// Outcome is what a client observes without reading book memory (FR-007).
type Outcome struct {
	Accepted  bool
	Trades    []book.Trade
	Remaining int
}
