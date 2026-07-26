package book

import (
	"errors"
	"testing"
)

// scenarioOp is one step in a Spec 1 deterministic harness case.
type scenarioOp struct {
	name string // "limit" | "market" | "cancel"
	limit LimitOrder
	market MarketOrder
	cancelID int
}

type scenarioCase struct {
	name     string
	ops      []scenarioOp
	want     []Trade
	wantSnap *BookSnapshot
	// needTODO skips until the named Spec 1 work is done.
	needTODO string
}

func TestScenarios(t *testing.T) {
	cases := []scenarioCase{
		{
			name: "rest_limit_on_empty_book",
			ops: []scenarioOp{
				{name: "limit", limit: LimitOrder{ID: 1, Side: Buy, Price: 100, Quantity: 5}},
			},
			want: nil,
			wantSnap: &BookSnapshot{
				Bids: []LevelSnapshot{{Price: 100, Orders: []OrderRef{{ID: 1, Quantity: 5}}}},
			},
		},
		{
			name: "crossed_book_full_fill",
			ops: []scenarioOp{
				{name: "limit", limit: LimitOrder{ID: 1, Side: Sell, Price: 100, Quantity: 10}},
				{name: "limit", limit: LimitOrder{ID: 2, Side: Buy, Price: 100, Quantity: 10}},
			},
			want: []Trade{
				{MakerID: 1, TakerID: 2, Price: 100, Quantity: 10},
			},
			wantSnap: &BookSnapshot{},
		},
		{
			name: "partial_fill_resting_ask",
			ops: []scenarioOp{
				{name: "limit", limit: LimitOrder{ID: 1, Side: Sell, Price: 100, Quantity: 10}},
				{name: "limit", limit: LimitOrder{ID: 2, Side: Buy, Price: 100, Quantity: 4}},
			},
			want: []Trade{
				{MakerID: 1, TakerID: 2, Price: 100, Quantity: 4},
			},
			wantSnap: &BookSnapshot{
				Asks: []LevelSnapshot{{Price: 100, Orders: []OrderRef{{ID: 1, Quantity: 6}}}},
			},
		},
		{
			name: "partial_fill_then_rest_aggressor",
			ops: []scenarioOp{
				{name: "limit", limit: LimitOrder{ID: 1, Side: Sell, Price: 100, Quantity: 10}},
				{name: "limit", limit: LimitOrder{ID: 2, Side: Buy, Price: 100, Quantity: 15}},
			},
			want: []Trade{
				{MakerID: 1, TakerID: 2, Price: 100, Quantity: 10},
			},
			wantSnap: &BookSnapshot{
				Bids: []LevelSnapshot{{Price: 100, Orders: []OrderRef{{ID: 2, Quantity: 5}}}},
			},
		},
		{
			name: "fifo_same_price",
			ops: []scenarioOp{
				{name: "limit", limit: LimitOrder{ID: 1, Side: Sell, Price: 100, Quantity: 3}},
				{name: "limit", limit: LimitOrder{ID: 2, Side: Sell, Price: 100, Quantity: 3}},
				{name: "limit", limit: LimitOrder{ID: 3, Side: Buy, Price: 100, Quantity: 4}},
			},
			want: []Trade{
				{MakerID: 1, TakerID: 3, Price: 100, Quantity: 3},
				{MakerID: 2, TakerID: 3, Price: 100, Quantity: 1},
			},
			wantSnap: &BookSnapshot{
				Asks: []LevelSnapshot{{Price: 100, Orders: []OrderRef{{ID: 2, Quantity: 2}}}},
			},
		},
		{
			name: "multi_level_walk",
			ops: []scenarioOp{
				{name: "limit", limit: LimitOrder{ID: 1, Side: Sell, Price: 100, Quantity: 5}},
				{name: "limit", limit: LimitOrder{ID: 2, Side: Sell, Price: 101, Quantity: 5}},
				{name: "limit", limit: LimitOrder{ID: 3, Side: Buy, Price: 101, Quantity: 8}},
			},
			want: []Trade{
				{MakerID: 1, TakerID: 3, Price: 100, Quantity: 5},
				{MakerID: 2, TakerID: 3, Price: 101, Quantity: 3},
			},
			wantSnap: &BookSnapshot{
				Asks: []LevelSnapshot{{Price: 101, Orders: []OrderRef{{ID: 2, Quantity: 2}}}},
			},
		},
		{
			name:     "market_against_liquidity",
			needTODO: "TODO(spec-1): SubmitMarket",
			ops: []scenarioOp{
				{name: "limit", limit: LimitOrder{ID: 1, Side: Sell, Price: 100, Quantity: 10}},
				{name: "market", market: MarketOrder{ID: 2, Side: Buy, Quantity: 10}},
			},
			want: []Trade{
				{MakerID: 1, TakerID: 2, Price: 100, Quantity: 10},
			},
		},
		{
			name:     "market_thin_book_discards_remainder",
			needTODO: "TODO(spec-1): SubmitMarket",
			ops: []scenarioOp{
				{name: "limit", limit: LimitOrder{ID: 1, Side: Sell, Price: 100, Quantity: 3}},
				{name: "market", market: MarketOrder{ID: 2, Side: Buy, Quantity: 10}},
			},
			want: []Trade{
				{MakerID: 1, TakerID: 2, Price: 100, Quantity: 3},
			},
			wantSnap: &BookSnapshot{},
		},
		{
			name:     "cancel_before_match",
			needTODO: "TODO(spec-1): Cancel",
			ops: []scenarioOp{
				{name: "limit", limit: LimitOrder{ID: 1, Side: Sell, Price: 100, Quantity: 10}},
				{name: "cancel", cancelID: 1},
				{name: "limit", limit: LimitOrder{ID: 2, Side: Buy, Price: 100, Quantity: 10}},
			},
			want: nil,
			wantSnap: &BookSnapshot{
				Bids: []LevelSnapshot{{Price: 100, Orders: []OrderRef{{ID: 2, Quantity: 10}}}},
			},
		},
		{
			name:     "cancel_mid_queue",
			needTODO: "TODO(spec-1): Cancel",
			ops: []scenarioOp{
				{name: "limit", limit: LimitOrder{ID: 1, Side: Sell, Price: 100, Quantity: 1}},
				{name: "limit", limit: LimitOrder{ID: 2, Side: Sell, Price: 100, Quantity: 1}},
				{name: "limit", limit: LimitOrder{ID: 3, Side: Sell, Price: 100, Quantity: 1}},
				{name: "cancel", cancelID: 2},
				{name: "limit", limit: LimitOrder{ID: 4, Side: Buy, Price: 100, Quantity: 2}},
			},
			want: []Trade{
				{MakerID: 1, TakerID: 4, Price: 100, Quantity: 1},
				{MakerID: 3, TakerID: 4, Price: 100, Quantity: 1},
			},
		},
		{
			name:     "reject_invalid_qty",
			needTODO: "TODO(spec-1): SubmitLimit validation (FR-012)",
			ops: []scenarioOp{
				{name: "limit", limit: LimitOrder{ID: 1, Side: Buy, Price: 100, Quantity: 0}},
			},
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.needTODO != "" {
				t.Skip(tc.needTODO)
			}
			b := New()
			var got []Trade
			for _, op := range tc.ops {
				switch op.name {
				case "limit":
					res, err := b.SubmitLimit(op.limit)
					if err != nil {
						t.Fatalf("SubmitLimit: %v", err)
					}
					got = append(got, res.Trades...)
				case "market":
					res, err := b.SubmitMarket(op.market)
					if errors.Is(err, ErrNotImplemented) {
						t.Skip("TODO(spec-1): SubmitMarket")
					}
					if err != nil {
						t.Fatalf("SubmitMarket: %v", err)
					}
					got = append(got, res.Trades...)
				case "cancel":
					res, err := b.Cancel(op.cancelID)
					if errors.Is(err, ErrNotImplemented) {
						t.Skip("TODO(spec-1): Cancel")
					}
					if err != nil {
						t.Fatalf("Cancel: %v", err)
					}
					got = append(got, res.Trades...)
				default:
					t.Fatalf("unknown op %q", op.name)
				}
			}
			if !tradesEqual(got, tc.want) {
				t.Fatalf("trades mismatch\ngot  %#v\nwant %#v", got, tc.want)
			}
			if tc.wantSnap != nil {
				snap := b.Snapshot()
				if !snapEqual(snap, *tc.wantSnap) {
					t.Fatalf("snapshot mismatch\ngot  %#v\nwant %#v", snap, *tc.wantSnap)
				}
			}
		})
	}
}

func tradesEqual(a, b []Trade) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func snapEqual(a, b BookSnapshot) bool {
	if len(a.Bids) != len(b.Bids) || len(a.Asks) != len(b.Asks) {
		return false
	}
	for i := range a.Bids {
		if a.Bids[i].Price != b.Bids[i].Price || len(a.Bids[i].Orders) != len(b.Bids[i].Orders) {
			return false
		}
		for j := range a.Bids[i].Orders {
			if a.Bids[i].Orders[j] != b.Bids[i].Orders[j] {
				return false
			}
		}
	}
	for i := range a.Asks {
		if a.Asks[i].Price != b.Asks[i].Price || len(a.Asks[i].Orders) != len(b.Asks[i].Orders) {
			return false
		}
		for j := range a.Asks[i].Orders {
			if a.Asks[i].Orders[j] != b.Asks[i].Orders[j] {
				return false
			}
		}
	}
	return true
}
