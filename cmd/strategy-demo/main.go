// Command strategy-demo runs a short Spec 6 simulation loop with Principle VI
// non-claims framing. Not profitable. Not production trading.
package main

import (
	"fmt"
	"os"

	"github.com/AarushShintre/matching-engine/internal/strategy"
)

func main() {
	fmt.Println(strategy.NonClaimBanner)
	fmt.Println("Spec 6 demo: market data → decision → ingress → feed-visible effect")

	cfg := strategy.Config{
		Symbol: "DEMO", QuoteSize: 10, SpreadMode: strategy.SpreadFixed,
		FixedHalfSpread: 5, MovementThreshold: 1, Enabled: true,
		OrderIDNamespace: 42,
	}
	ing := &loggingIngress{}
	r, err := strategy.NewRunner(cfg, ing)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	trade := func(px int) {
		t := strategy.TradeEvent{Symbol: "DEMO", Price: px, Quantity: 1}
		_ = r.HandleEvent(strategy.Event{Kind: strategy.EventTrade, Trade: &t})
		st := r.Status()
		var last, ref string
		if st.LastAction != nil {
			last = string(*st.LastAction)
		}
		if st.Reference != nil {
			ref = fmt.Sprintf("%d", *st.Reference)
		}
		fmt.Printf("status phase=%s last=%s ref=%s\n", st.Phase, last, ref)
	}

	trade(100)
	fmt.Println("requote cycle:")
	trade(110)

	_ = r.Shutdown()
	fmt.Println("shutdown complete; simulation finished")
	fmt.Println(strategy.NonClaimBanner)
}

type loggingIngress struct{}

func (l *loggingIngress) SubmitNewLimit(o strategy.NewLimitOrder) error {
	fmt.Printf("ingress new_limit id=%d side=%s px=%d qty=%d\n", o.OrderID, o.Side, o.Price, o.Quantity)
	return nil
}

func (l *loggingIngress) SubmitCancel(o strategy.CancelOrder) error {
	fmt.Printf("ingress cancel id=%d\n", o.OrderID)
	return nil
}
