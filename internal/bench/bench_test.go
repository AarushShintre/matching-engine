package bench

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AarushShintre/matching-engine/internal/ingest"
)

const benchmarkSymbol = "BENCH"

// BenchmarkConcurrentIngress measures submit-to-outcome latency through the
// real single-writer matcher. At least four producers concurrently repeat a
// matching-heavy cycle: rest ask, cross it with a market buy, rest bid, cancel.
func BenchmarkConcurrentIngress(b *testing.B) {
	engine := ingest.NewEngine(benchmarkSymbol, nil)
	if err := engine.Start(context.Background()); err != nil {
		b.Fatal(err)
	}
	defer engine.Stop()

	producerCount := runtime.GOMAXPROCS(0)
	if producerCount < 4 {
		producerCount = 4
	}
	client := engine.Client()
	latencies := make([]int64, b.N)
	errors := make(chan error, producerCount)
	var (
		nextWork atomic.Uint64
		nextID   atomic.Uint64
		wg       sync.WaitGroup
	)

	b.ReportAllocs()
	b.ResetTimer()
	started := time.Now()
	for producer := 0; producer < producerCount; producer++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var (
				step       uint64
				restingBid ingest.OrderID
			)
			for {
				index := int(nextWork.Add(1) - 1)
				if index >= b.N {
					return
				}
				operationStarted := time.Now()
				var (
					outcome ingest.Outcome
					err     error
				)
				switch step % 4 {
				case 0:
					outcome, err = client.SubmitNewLimit(ingest.NewLimitOrder{
						Symbol: benchmarkSymbol, OrderID: ingest.OrderID(nextID.Add(1)),
						Side: "sell", Price: 101, Quantity: 10,
					})
				case 1:
					outcome, err = client.SubmitMarket(ingest.MarketOrder{
						Symbol: benchmarkSymbol, OrderID: ingest.OrderID(nextID.Add(1)),
						Side: "buy", Quantity: 10,
					})
				case 2:
					restingBid = ingest.OrderID(nextID.Add(1))
					outcome, err = client.SubmitNewLimit(ingest.NewLimitOrder{
						Symbol: benchmarkSymbol, OrderID: restingBid,
						Side: "buy", Price: 99, Quantity: 10,
					})
				case 3:
					outcome, err = client.SubmitCancel(ingest.CancelOrder{
						Symbol: benchmarkSymbol, OrderID: restingBid,
					})
				}
				latencies[index] = time.Since(operationStarted).Nanoseconds()
				if err != nil || !outcome.Accepted {
					errors <- fmt.Errorf("producer operation %d: accepted=%v err=%v", step, outcome.Accepted, err)
					return
				}
				step++
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(started)
	b.StopTimer()
	close(errors)
	for err := range errors {
		b.Error(err)
	}

	sorted := append([]int64(nil), latencies...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "orders/sec")
	b.ReportMetric(float64(percentile(sorted, 50)), "p50-ns/op")
	b.ReportMetric(float64(percentile(sorted, 99)), "p99-ns/op")
}

func percentile(sorted []int64, percent int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	index := (len(sorted)*percent + 99) / 100
	if index < 1 {
		index = 1
	}
	return sorted[index-1]
}

func TestPercentile(t *testing.T) {
	samples := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	if got := percentile(samples, 50); got != 50 {
		t.Fatalf("p50: got %d, want 50", got)
	}
	if got := percentile(samples, 99); got != 100 {
		t.Fatalf("p99: got %d, want 100", got)
	}
}
