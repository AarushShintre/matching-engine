package strategy

// Side is bid or ask for owned tracking.
type Side string

const (
	SideBid Side = "bid"
	SideAsk Side = "ask"
)

// Action is the pure decision output action.
type Action string

const (
	ActionHold           Action = "hold"
	ActionQuoteInitial   Action = "quote_initial"
	ActionRequote        Action = "requote"
	ActionPause          Action = "pause"
	ActionShutdownCancel Action = "shutdown_cancel"
)

// Phase is the strategy run-loop phase.
type Phase string

const (
	PhaseWaiting Phase = "waiting"
	PhaseQuoting Phase = "quoting"
	PhasePaused  Phase = "paused"
	PhaseStopped Phase = "stopped"
)

// RefSource identifies where the reference price came from.
type RefSource string

const (
	RefSeed      RefSource = "seed"
	RefLastTrade RefSource = "last_trade"
)

// QuoteState is QuoteSet lifecycle state.
type QuoteState string

const (
	QuoteEmpty     QuoteState = "empty"
	QuotePending   QuoteState = "pending"
	QuoteResting   QuoteState = "resting"
	QuoteCanceling QuoteState = "canceling"
)

// OwnedStatus tracks a submitted strategy order.
type OwnedStatus string

const (
	OwnedSubmitted OwnedStatus = "submitted"
	OwnedResting   OwnedStatus = "resting"
	OwnedFilled    OwnedStatus = "filled"
	OwnedCanceled  OwnedStatus = "canceled"
	OwnedUnknown   OwnedStatus = "unknown"
)

// LimitIntent is a new limit order the decision wants to submit.
type LimitIntent struct {
	OrderID  OrderID
	Side     Side // bid=buy, ask=sell
	Price    int
	Quantity int
}

// Decision is the pure (config, state, event) → output surface.
type Decision struct {
	Action     Action
	Reference  *int
	HalfSpread *int
	CancelIDs  []OrderID
	NewOrders  []NewLimitOrder
}

// ReferencePrice is the center of the quote set.
type ReferencePrice struct {
	Value          int
	Source         RefSource
	LastTradePrice *int
	Valid          bool
}

// QuoteSet is the strategy's intended two-sided quote and owned resting ids.
type QuoteSet struct {
	BidPrice       int
	AskPrice       int
	BidSize        int
	AskSize        int
	BidOrderID     *OrderID
	AskOrderID     *OrderID
	CenteredOn     int
	HalfSpreadUsed int
	State          QuoteState
}

// OwnedOrder tracks a strategy-owned order for cancel/replace and shutdown.
type OwnedOrder struct {
	OrderID OrderID
	Side    Side
	Price   int
	Size    int
	Status  OwnedStatus
}

// State is mutable strategy state for Decide and the runner.
type State struct {
	Phase        Phase
	Ref          ReferencePrice
	Quotes       QuoteSet
	Owned        map[OrderID]OwnedOrder
	RecentTrades int // count of trade events in the rolling activity window cursor
	TradeCursor  []struct{} // length = activity window occupancy (event-count based)
}

// NewState returns an empty strategy state.
func NewState() State {
	return State{
		Phase: PhaseWaiting,
		Quotes: QuoteSet{State: QuoteEmpty},
		Owned:  make(map[OrderID]OwnedOrder),
	}
}

// EmptyQuotes reports whether the strategy currently has no resting/pending quotes.
func (s State) EmptyQuotes() bool {
	return s.Quotes.State == QuoteEmpty || (s.Quotes.BidOrderID == nil && s.Quotes.AskOrderID == nil && s.Quotes.State != QuotePending)
}
