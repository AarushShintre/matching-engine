package marketdata

// Side is bid or ask for depth events (Spec 5 / Spec 6 contract).
type Side string

const (
	SideBid Side = "bid"
	SideAsk Side = "ask"
)

// Kind discriminates outbound market-data events.
type Kind int

const (
	KindTrade Kind = iota
	KindBookDepth
)

// Event is one item on the outbound market-data stream.
type Event struct {
	Kind      Kind
	Trade     *TradeEvent
	BookDepth *BookDepthEvent
}

// TradeEvent is a Spec 5 trade published by the matching owner.
type TradeEvent struct {
	Symbol           string
	Price            int
	Quantity         int
	RestingOrderID   *uint64 // maker / resting side if known
	AggressorOrderID *uint64 // taker / aggressor side if known
}

// BookDepthEvent is a Spec 5 depth change at one price level.
type BookDepthEvent struct {
	Symbol   string
	Side     Side
	Price    int
	Quantity int // resting depth at level after change
}
