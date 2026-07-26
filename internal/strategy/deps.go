package strategy

import (
	"github.com/AarushShintre/matching-engine/internal/ingest"
	"github.com/AarushShintre/matching-engine/internal/marketdata"
)

// Canonical Spec 2 / Spec 5 types (owned by ingest + marketdata packages).
type (
	OrderID        = ingest.OrderID
	NewLimitOrder  = ingest.NewLimitOrder
	CancelOrder    = ingest.CancelOrder
	TradeEvent     = marketdata.TradeEvent
	BookDepthEvent = marketdata.BookDepthEvent
)

// Ingress is the Spec 2 client surface used by the strategy.
// Strategy MUST NOT mutate the book directly.
//
// Narrower than ingest.Client (error-only, no market submit): wrap an
// ingest.Client adapter when the engine is live, or use test fakes.
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
