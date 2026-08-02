package replay

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/AarushShintre/matching-engine/internal/book"
	"github.com/AarushShintre/matching-engine/internal/ingest"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "testdata")
}

func TestCatalogReplaysDefaultCount(t *testing.T) {
	scenarios, err := LoadCatalog(testdataDir(t))
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if len(scenarios) != 3 {
		t.Fatalf("catalog has %d scenarios, want 3", len(scenarios))
	}

	names := map[string]bool{}
	for _, scenario := range scenarios {
		names[scenario.Name] = true
		if err := scenario.Replay(DefaultN); err != nil {
			t.Fatalf("Replay(%q): %v", scenario.Name, err)
		}
	}
	for _, required := range []string{
		"concurrent_same_price_submits",
		"cancel_racing_cross",
		"multi_client_burst_into_non_empty_book",
	} {
		if !names[required] {
			t.Errorf("catalog missing required scenario %q", required)
		}
	}
}

func TestRunSuiteSupportsFullAndPartialRuns(t *testing.T) {
	dir := testdataDir(t)
	if err := RunSuite(dir, 3); err != nil {
		t.Fatalf("full suite: %v", err)
	}
	if err := RunSuite(dir, 3, "cancel_racing_cross"); err != nil {
		t.Fatalf("partial suite: %v", err)
	}
	err := RunSuite(dir, 1, "missing_scenario")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing scenario: got %v", err)
	}
	if err := RunSuite(dir, -1); !errors.Is(err, ErrInvalidRuns) {
		t.Fatalf("negative run count: got %v, want ErrInvalidRuns", err)
	}
}

func TestLoadMissingAndCorruptFailClearly(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("Load missing file succeeded")
	}

	path := filepath.Join(t.TempDir(), "corrupt.json")
	if err := os.WriteFile(path, []byte(`{"name":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("Load corrupt file: got %v", err)
	}
}

func TestScenarioValidation(t *testing.T) {
	valid := &Scenario{
		Name:     "empty",
		Symbol:   "TEST",
		Ops:      []TimedOp{},
		Baseline: []book.Trade{},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("empty scenario must be valid: %v", err)
	}
	if err := valid.Replay(3); err != nil {
		t.Fatalf("Replay empty scenario: %v", err)
	}
	if err := valid.Replay(0); !errors.Is(err, ErrInvalidRuns) {
		t.Fatalf("Replay(0): got %v, want ErrInvalidRuns", err)
	}

	invalid := *valid
	invalid.Ops = []TimedOp{{Sequence: 0}}
	if err := invalid.Validate(); !errors.Is(err, ErrInvalidScenario) {
		t.Fatalf("payload-less op: got %v, want ErrInvalidScenario", err)
	}
}

func TestBaselineMismatchReportsTradePosition(t *testing.T) {
	scenario, err := Load(filepath.Join(testdataDir(t), "same_price_race.json"))
	if err != nil {
		t.Fatal(err)
	}
	scenario.Baseline[0].Quantity++

	err = scenario.Replay(1)
	var divergence *DivergenceError
	if !errors.As(err, &divergence) {
		t.Fatalf("got %v, want DivergenceError", err)
	}
	if divergence.Scenario != scenario.Name || divergence.Run != 0 ||
		divergence.Against != "baseline" || len(divergence.Diffs) != 1 ||
		divergence.Diffs[0].Position != 0 {
		t.Fatalf("incomplete divergence details: %#v", divergence)
	}
}

func TestCanonicalTrades(t *testing.T) {
	data, err := CanonicalTrades([]book.Trade{
		{Quantity: 2, Price: 100, MakerID: 1, TakerID: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"position":0,"quantity":2,"price":100,"maker_id":1,"taker_id":2}]`
	if string(data) != want {
		t.Fatalf("canonical output\n got: %s\nwant: %s", data, want)
	}
}

func TestRecorderCapturesConcurrentSessionAndSaves(t *testing.T) {
	engine := ingest.NewEngine("TEST", nil)
	if err := engine.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer engine.Stop()

	recorder := NewRecorder(engine.Client(), "recorded_burst", "TEST")
	var wg sync.WaitGroup
	for id := 1; id <= 4; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := recorder.SubmitNewLimit(ingest.NewLimitOrder{
				Symbol: "TEST", OrderID: ingest.OrderID(id), Side: "buy", Price: 100, Quantity: 1,
			})
			if err != nil {
				t.Errorf("SubmitNewLimit(%d): %v", id, err)
			}
		}(id)
	}
	wg.Wait()

	if _, err := recorder.SubmitMarket(ingest.MarketOrder{
		Symbol: "TEST", OrderID: 10, Side: "sell", Quantity: 4,
	}); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "recorded.json")
	if err := recorder.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load saved capture: %v", err)
	}
	if len(loaded.Ops) != 5 || len(loaded.Baseline) != 4 {
		t.Fatalf("saved capture incomplete: ops=%d trades=%d", len(loaded.Ops), len(loaded.Baseline))
	}
	if err := loaded.Replay(10); err != nil {
		t.Fatalf("Replay saved capture: %v", err)
	}
}

func TestRejectedOperationsReplayDeterministically(t *testing.T) {
	scenario := &Scenario{
		Name:   "rejected_operation",
		Symbol: "TEST",
		Ops: []TimedOp{
			{
				Sequence: 0,
				DelayNS:  0,
				Limit: &ingest.NewLimitOrder{
					Symbol: "TEST", OrderID: 1, Side: "invalid", Price: 100, Quantity: 1,
				},
			},
			{
				Sequence: 1,
				DelayNS:  1,
				Limit: &ingest.NewLimitOrder{
					Symbol: "TEST", OrderID: 2, Side: "buy", Price: 100, Quantity: 1,
				},
			},
		},
		Baseline: []book.Trade{},
	}
	if err := scenario.Replay(10); err != nil {
		t.Fatalf("Replay rejected operation: %v", err)
	}
}

func TestSuiteDetectsOrderingBrokenRunner(t *testing.T) {
	scenario := &Scenario{
		Name:     "discriminative_check",
		Symbol:   "TEST",
		Ops:      []TimedOp{},
		Baseline: nil,
	}
	run := 0
	err := scenario.replayWith(2, func() (runOutput, error) {
		run++
		makerID := 1
		if run == 2 {
			// Simulate a faulty ingestion path choosing a different same-price
			// winner on the second run.
			makerID = 2
		}
		trade := book.Trade{Quantity: 1, Price: 100, MakerID: makerID, TakerID: 3}
		return runOutput{
			Trades: []book.Trade{trade},
			Outcomes: []operationOutput{
				{Sequence: 0, Accepted: true, Trades: []book.Trade{trade}},
			},
		}, nil
	})

	var divergence *DivergenceError
	if !errors.As(err, &divergence) {
		t.Fatalf("ordering-broken runner passed; got %v, want DivergenceError", err)
	}
	if divergence.Run != 1 || divergence.Against != "run 0" ||
		len(divergence.Diffs) != 1 || divergence.Diffs[0].Position != 0 {
		t.Fatalf("incomplete divergence report: %#v", divergence)
	}
}

func TestSpec1MatchingExpectationsThroughIngress(t *testing.T) {
	cases := []*Scenario{
		{
			Name:   "spec1_full_fill",
			Symbol: "TEST",
			Ops: []TimedOp{
				{Sequence: 0, DelayNS: 0, Limit: &ingest.NewLimitOrder{
					Symbol: "TEST", OrderID: 1, Side: "sell", Price: 100, Quantity: 10,
				}},
				{Sequence: 1, DelayNS: 1, Limit: &ingest.NewLimitOrder{
					Symbol: "TEST", OrderID: 2, Side: "buy", Price: 100, Quantity: 10,
				}},
			},
			Baseline: []book.Trade{
				{MakerID: 1, TakerID: 2, Price: 100, Quantity: 10},
			},
		},
		{
			Name:   "spec1_partial_fill_and_fifo",
			Symbol: "TEST",
			Ops: []TimedOp{
				{Sequence: 0, DelayNS: 0, Limit: &ingest.NewLimitOrder{
					Symbol: "TEST", OrderID: 1, Side: "sell", Price: 100, Quantity: 3,
				}},
				{Sequence: 1, DelayNS: 1, Limit: &ingest.NewLimitOrder{
					Symbol: "TEST", OrderID: 2, Side: "sell", Price: 100, Quantity: 3,
				}},
				{Sequence: 2, DelayNS: 2, Limit: &ingest.NewLimitOrder{
					Symbol: "TEST", OrderID: 3, Side: "buy", Price: 100, Quantity: 4,
				}},
			},
			Baseline: []book.Trade{
				{MakerID: 1, TakerID: 3, Price: 100, Quantity: 3},
				{MakerID: 2, TakerID: 3, Price: 100, Quantity: 1},
			},
		},
		{
			Name:   "spec1_market_thin_book",
			Symbol: "TEST",
			Ops: []TimedOp{
				{Sequence: 0, DelayNS: 0, Limit: &ingest.NewLimitOrder{
					Symbol: "TEST", OrderID: 1, Side: "sell", Price: 100, Quantity: 2,
				}},
				{Sequence: 1, DelayNS: 1, Market: &ingest.MarketOrder{
					Symbol: "TEST", OrderID: 2, Side: "buy", Quantity: 5,
				}},
			},
			Baseline: []book.Trade{
				{MakerID: 1, TakerID: 2, Price: 100, Quantity: 2},
			},
		},
		{
			Name:   "spec1_cancel_before_match",
			Symbol: "TEST",
			Ops: []TimedOp{
				{Sequence: 0, DelayNS: 0, Limit: &ingest.NewLimitOrder{
					Symbol: "TEST", OrderID: 1, Side: "sell", Price: 100, Quantity: 1,
				}},
				{Sequence: 1, DelayNS: 1, Cancel: &ingest.CancelOrder{
					Symbol: "TEST", OrderID: 1,
				}},
				{Sequence: 2, DelayNS: 2, Limit: &ingest.NewLimitOrder{
					Symbol: "TEST", OrderID: 2, Side: "buy", Price: 100, Quantity: 1,
				}},
			},
			Baseline: []book.Trade{},
		},
	}

	for _, scenario := range cases {
		t.Run(scenario.Name, func(t *testing.T) {
			if err := scenario.Replay(10); err != nil {
				t.Fatal(err)
			}
		})
	}
}
