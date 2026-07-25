package strategy

// TradeEvent is a Spec 5 trade consumed by the strategy (in-process contract).
type TradeEvent struct {
	Symbol            string
	Price             int
	Quantity          int
	RestingOrderID    *OrderID
	AggressorOrderID  *OrderID
}

// BookDepthEvent is a Spec 5 book-depth event (optional for quoting; used in evidence).
type BookDepthEvent struct {
	Symbol   string
	Side     Side
	Price    int
	Quantity int
}

// NewLimitOrder is a Spec 2 new-limit ingress operation.
type NewLimitOrder struct {
	Symbol   string
	OrderID  OrderID
	Side     string // "buy" | "sell"
	Price    int
	Quantity int
}

// CancelOrder is a Spec 2 cancel ingress operation.
type CancelOrder struct {
	Symbol  string
	OrderID OrderID
}

// Ingress is the Spec 2 client surface. Strategy MUST NOT mutate the book directly.
type Ingress interface {
	SubmitNewLimit(o NewLimitOrder) error
	SubmitCancel(o CancelOrder) error
}

// EventKind discriminates runner/Decide inputs.
type EventKind int

const (
	EventTrade EventKind = iota
	EventGap
	EventShutdown
)

// Event is a strategy input: trade, feed gap, or shutdown.
type Event struct {
	Kind  EventKind
	Trade *TradeEvent
}

// NonClaimBanner is the mandatory Principle VI framing string.
const NonClaimBanner = "Simulation only — event-driven system-design demo; not profitable or production trading."
