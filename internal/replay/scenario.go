package replay

import (
	"errors"

	"github.com/AarushShintre/matching-engine/internal/book"
	"github.com/AarushShintre/matching-engine/internal/ingest"
)

var ErrNotImplemented = errors.New("replay: not implemented")

// DefaultN is the Spec 3 default replay count (N ≥ 100).
const DefaultN = 100

// TimedOp is one captured submit/cancel with relative timing metadata.
type TimedOp struct {
	// DelayNS is offset from scenario start (or relative to prior op).
	// TODO(spec-3): define exact timing semantics for concurrent races.
	DelayNS int64
	Limit   *ingest.NewLimitOrder
	Market  *ingest.MarketOrder
	Cancel  *ingest.CancelOrder
}

// Scenario is a named capture that drives ingest without live clients (FR-003).
//
// Drive replay through ingest.Client — never call book mutation APIs directly.
type Scenario struct {
	Name     string
	Ops      []TimedOp
	Baseline []book.Trade
}

// Load reads a captured scenario artifact from path.
//
// TODO(spec-3): parse capture format (ops + concurrent timing); fail fast on
// missing/corrupt input (FR-012).
func Load(path string) (*Scenario, error) {
	_ = path
	return nil, ErrNotImplemented
}

// Replay runs this scenario n times via Spec 2 ingress and asserts identical
// ordered trade output (and optional baseline lock).
//
// TODO(spec-3): for each run, apply TimedOps through ingest.Client; compare
// canonical trade sequences across all n runs; on divergence report scenario,
// run index, and differing trade positions (FR-005, FR-011).
func (s *Scenario) Replay(n int) error {
	_ = s
	_ = n
	return ErrNotImplemented
}
