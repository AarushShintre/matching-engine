package replay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AarushShintre/matching-engine/internal/book"
	"github.com/AarushShintre/matching-engine/internal/ingest"
)

var (
	ErrInvalidScenario = errors.New("replay: invalid scenario")
	ErrInvalidRuns     = errors.New("replay: run count must be positive")
)

// DefaultN is the Spec 3 default replay count (N ≥ 100).
const DefaultN = 100

// TimedOp is one captured submit/cancel. DelayNS is the offset from capture
// start; Sequence is the observed ingress order among overlapping calls.
type TimedOp struct {
	Sequence int                   `json:"sequence"`
	DelayNS  int64                 `json:"delay_ns"`
	Limit    *ingest.NewLimitOrder `json:"limit,omitempty"`
	Market   *ingest.MarketOrder   `json:"market,omitempty"`
	Cancel   *ingest.CancelOrder   `json:"cancel,omitempty"`
}

// Scenario is a named capture that drives ingest without live clients (FR-003).
// Drive replay through ingest.Client — never call book mutation APIs directly.
type Scenario struct {
	Name     string       `json:"name"`
	Symbol   string       `json:"symbol"`
	Ops      []TimedOp    `json:"ops"`
	Baseline []book.Trade `json:"baseline"`
}

// Load reads a captured scenario artifact from path.
func Load(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("replay: load %q: %w", path, err)
	}

	var scenario Scenario
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&scenario); err != nil {
		return nil, fmt.Errorf("replay: decode %q: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("trailing JSON content")
		}
		return nil, fmt.Errorf("replay: decode %q: %w", path, err)
	}
	if err := scenario.Validate(); err != nil {
		return nil, fmt.Errorf("replay: validate %q: %w", path, err)
	}
	return &scenario, nil
}

// Save writes a validated, stable JSON capture that can be replayed later.
func (s *Scenario) Save(path string) error {
	if err := s.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("replay: encode %q: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("replay: save %q: %w", path, err)
	}
	return nil
}

// LoadCatalog loads every JSON scenario in lexical filename order.
func LoadCatalog(dir string) ([]*Scenario, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("replay: catalog %q: %w", dir, err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("replay: catalog %q: no JSON scenarios", dir)
	}

	scenarios := make([]*Scenario, 0, len(paths))
	for _, path := range paths {
		scenario, err := Load(path)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, scenario)
	}
	return scenarios, nil
}

// RunSuite replays all catalog scenarios, or only the requested names. Passing
// n=0 selects DefaultN. Failures are joined so CI reports every broken scenario.
func RunSuite(dir string, n int, names ...string) error {
	if n == 0 {
		n = DefaultN
	}
	if n < 0 {
		return ErrInvalidRuns
	}

	scenarios, err := LoadCatalog(dir)
	if err != nil {
		return err
	}

	requested := make(map[string]struct{}, len(names))
	for _, name := range names {
		requested[name] = struct{}{}
	}
	seen := make(map[string]struct{}, len(names))
	var failures []error
	for _, scenario := range scenarios {
		if len(requested) > 0 {
			if _, ok := requested[scenario.Name]; !ok {
				continue
			}
			seen[scenario.Name] = struct{}{}
		}
		if err := scenario.Replay(n); err != nil {
			failures = append(failures, err)
		}
	}
	for name := range requested {
		if _, ok := seen[name]; !ok {
			failures = append(failures, fmt.Errorf("replay: scenario %q not found in catalog", name))
		}
	}
	return errors.Join(failures...)
}

// Validate checks that the capture is complete and has an unambiguous order.
func (s *Scenario) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("%w: missing name", ErrInvalidScenario)
	}
	if strings.TrimSpace(s.Symbol) == "" {
		return fmt.Errorf("%w: missing symbol", ErrInvalidScenario)
	}
	for i, operation := range s.Ops {
		if operation.Sequence != i {
			return fmt.Errorf("%w: operation %d has sequence %d", ErrInvalidScenario, i, operation.Sequence)
		}
		if operation.DelayNS < 0 {
			return fmt.Errorf("%w: operation %d has negative delay", ErrInvalidScenario, i)
		}
		if i > 0 && operation.DelayNS < s.Ops[i-1].DelayNS {
			return fmt.Errorf("%w: operation %d timing precedes operation %d", ErrInvalidScenario, i, i-1)
		}
		if operation.payloadCount() != 1 {
			return fmt.Errorf("%w: operation %d must contain exactly one payload", ErrInvalidScenario, i)
		}
		if symbol := operation.symbol(); symbol != s.Symbol {
			return fmt.Errorf("%w: operation %d uses symbol %q, want %q", ErrInvalidScenario, i, symbol, s.Symbol)
		}
	}
	return nil
}

// Replay runs this scenario n times via Spec 2 ingress and asserts identical
// ordered trade output (and optional baseline lock).
func (s *Scenario) Replay(n int) error {
	return s.replayWith(n, s.runOnce)
}

func (s *Scenario) replayWith(n int, run func() (runOutput, error)) error {
	if n <= 0 {
		return ErrInvalidRuns
	}
	if err := s.Validate(); err != nil {
		return err
	}

	var first runOutput
	var firstCanonical []byte
	for runIndex := 0; runIndex < n; runIndex++ {
		output, err := run()
		if err != nil {
			return fmt.Errorf("replay: scenario %q run %d: %w", s.Name, runIndex, err)
		}
		if s.Baseline != nil {
			if differences := diffTrades(s.Baseline, output.Trades); len(differences) > 0 {
				return &DivergenceError{
					Scenario: s.Name,
					Run:      runIndex,
					Against:  "baseline",
					Diffs:    differences,
				}
			}
		}
		canonical, err := json.Marshal(output)
		if err != nil {
			return fmt.Errorf("replay: scenario %q run %d canonicalize: %w", s.Name, runIndex, err)
		}
		if runIndex == 0 {
			first = output
			firstCanonical = canonical
			continue
		}
		if !bytes.Equal(firstCanonical, canonical) {
			differences := diffTrades(first.Trades, output.Trades)
			detail := ""
			if len(differences) == 0 {
				detail = "operation outcomes differ"
			}
			return &DivergenceError{
				Scenario: s.Name,
				Run:      runIndex,
				Against:  "run 0",
				Diffs:    differences,
				Detail:   detail,
			}
		}
	}
	return nil
}

type runOutput struct {
	Trades   []book.Trade      `json:"trades"`
	Outcomes []operationOutput `json:"outcomes"`
}

type operationOutput struct {
	Sequence  int          `json:"sequence"`
	Accepted  bool         `json:"accepted"`
	Remaining int          `json:"remaining"`
	Error     string       `json:"error,omitempty"`
	Trades    []book.Trade `json:"trades"`
}

func (s *Scenario) runOnce() (runOutput, error) {
	engine := ingest.NewEngine(s.Symbol, nil)
	if err := engine.Start(context.Background()); err != nil {
		return runOutput{}, err
	}
	defer engine.Stop()

	client := engine.Client()
	output := runOutput{
		Trades:   []book.Trade{},
		Outcomes: []operationOutput{},
	}

	for start := 0; start < len(s.Ops); {
		end := start + 1
		for end < len(s.Ops) && s.Ops[end].DelayNS == s.Ops[start].DelayNS {
			end++
		}

		wave := s.Ops[start:end]
		results := make([]operationResult, len(wave))
		turns := make([]chan struct{}, len(wave)+1)
		for i := range turns {
			turns[i] = make(chan struct{})
		}
		close(turns[0])

		var wg sync.WaitGroup
		for i, operation := range wave {
			wg.Add(1)
			go func(i int, operation TimedOp) {
				defer wg.Done()
				<-turns[i]
				results[i].outcome, results[i].err = submitOperation(client, operation)
				close(turns[i+1])
			}(i, operation)
		}
		wg.Wait()

		for i, result := range results {
			operationOutcome := operationOutput{
				Sequence:  wave[i].Sequence,
				Accepted:  result.outcome.Accepted,
				Remaining: result.outcome.Remaining,
				Trades:    append([]book.Trade{}, result.outcome.Trades...),
			}
			if result.err != nil {
				operationOutcome.Error = result.err.Error()
			}
			output.Outcomes = append(output.Outcomes, operationOutcome)
			output.Trades = append(output.Trades, result.outcome.Trades...)
		}
		start = end
	}
	return output, nil
}

type operationResult struct {
	outcome ingest.Outcome
	err     error
}

func submitOperation(client ingest.Client, operation TimedOp) (ingest.Outcome, error) {
	switch {
	case operation.Limit != nil:
		return client.SubmitNewLimit(*operation.Limit)
	case operation.Market != nil:
		return client.SubmitMarket(*operation.Market)
	case operation.Cancel != nil:
		return client.SubmitCancel(*operation.Cancel)
	default:
		return ingest.Outcome{}, fmt.Errorf("%w: operation %d has no payload", ErrInvalidScenario, operation.Sequence)
	}
}

func (operation TimedOp) payloadCount() int {
	count := 0
	if operation.Limit != nil {
		count++
	}
	if operation.Market != nil {
		count++
	}
	if operation.Cancel != nil {
		count++
	}
	return count
}

func (operation TimedOp) symbol() string {
	switch {
	case operation.Limit != nil:
		return operation.Limit.Symbol
	case operation.Market != nil:
		return operation.Market.Symbol
	case operation.Cancel != nil:
		return operation.Cancel.Symbol
	default:
		return ""
	}
}

// TradeDiff describes one canonical trade position that differs.
type TradeDiff struct {
	Position int
	Expected *book.Trade
	Actual   *book.Trade
}

// DivergenceError identifies the scenario, run, oracle, and trade positions.
type DivergenceError struct {
	Scenario string
	Run      int
	Against  string
	Diffs    []TradeDiff
	Detail   string
}

func (e *DivergenceError) Error() string {
	positions := make([]string, 0, len(e.Diffs))
	for _, difference := range e.Diffs {
		positions = append(positions, fmt.Sprintf("%d", difference.Position))
	}
	message := fmt.Sprintf(
		"replay: scenario %q run %d diverged from %s at trade positions [%s]",
		e.Scenario,
		e.Run,
		e.Against,
		strings.Join(positions, ", "),
	)
	if e.Detail != "" {
		message += ": " + e.Detail
	}
	return message
}

func diffTrades(expected, actual []book.Trade) []TradeDiff {
	length := len(expected)
	if len(actual) > length {
		length = len(actual)
	}
	var differences []TradeDiff
	for position := 0; position < length; position++ {
		var want, got *book.Trade
		if position < len(expected) {
			value := expected[position]
			want = &value
		}
		if position < len(actual) {
			value := actual[position]
			got = &value
		}
		if want == nil || got == nil || *want != *got {
			differences = append(differences, TradeDiff{
				Position: position,
				Expected: want,
				Actual:   got,
			})
		}
	}
	return differences
}

// CanonicalTrades returns stable JSON containing each trade's sequence position
// and all fields required by FR-006.
func CanonicalTrades(trades []book.Trade) ([]byte, error) {
	type canonicalTrade struct {
		Position int `json:"position"`
		Quantity int `json:"quantity"`
		Price    int `json:"price"`
		MakerID  int `json:"maker_id"`
		TakerID  int `json:"taker_id"`
	}
	canonical := make([]canonicalTrade, len(trades))
	for position, trade := range trades {
		canonical[position] = canonicalTrade{
			Position: position,
			Quantity: trade.Quantity,
			Price:    trade.Price,
			MakerID:  trade.MakerID,
			TakerID:  trade.TakerID,
		}
	}
	return json.Marshal(canonical)
}

// Recorder wraps the Spec 2 client and captures overlapping calls in the order
// they are handed to ingress. The lock protects capture metadata, never book state.
type Recorder struct {
	client ingest.Client
	name   string
	symbol string
	start  time.Time

	mu       sync.Mutex
	ops      []TimedOp
	baseline []book.Trade
	lastDone int64
}

func NewRecorder(client ingest.Client, name, symbol string) *Recorder {
	return &Recorder{
		client:   client,
		name:     name,
		symbol:   symbol,
		start:    time.Now(),
		baseline: []book.Trade{},
	}
}

func (r *Recorder) SubmitNewLimit(order ingest.NewLimitOrder) (ingest.Outcome, error) {
	return r.record(TimedOp{Limit: &order})
}

func (r *Recorder) SubmitMarket(order ingest.MarketOrder) (ingest.Outcome, error) {
	return r.record(TimedOp{Market: &order})
}

func (r *Recorder) SubmitCancel(order ingest.CancelOrder) (ingest.Outcome, error) {
	return r.record(TimedOp{Cancel: &order})
}

func (r *Recorder) record(operation TimedOp) (ingest.Outcome, error) {
	calledAt := time.Since(r.start).Nanoseconds()
	r.mu.Lock()
	defer r.mu.Unlock()

	operation.Sequence = len(r.ops)
	operation.DelayNS = calledAt
	if len(r.ops) > 0 && calledAt < r.lastDone {
		// The call overlapped the preceding in-flight submission. Keep it in
		// the same replay wave while Sequence preserves the observed winner.
		operation.DelayNS = r.ops[len(r.ops)-1].DelayNS
	}
	r.ops = append(r.ops, operation)

	outcome, err := submitOperation(r.client, operation)
	r.baseline = append(r.baseline, outcome.Trades...)
	r.lastDone = time.Since(r.start).Nanoseconds()
	return outcome, err
}

func (r *Recorder) Scenario() *Scenario {
	r.mu.Lock()
	defer r.mu.Unlock()
	return &Scenario{
		Name:     r.name,
		Symbol:   r.symbol,
		Ops:      append([]TimedOp(nil), r.ops...),
		Baseline: append([]book.Trade{}, r.baseline...),
	}
}

func (r *Recorder) Save(path string) error {
	return r.Scenario().Save(path)
}
