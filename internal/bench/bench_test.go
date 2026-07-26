package bench

import "testing"

// BenchmarkConcurrentIngress measures realistic multi-producer load into Spec 2 ingest.
//
// TODO(spec-4): start ingest.Engine; N≥4 producers submit mixed limit/market/cancel
// into a non-empty book; measure submit→outcome latency; report:
//   - orders/sec (throughput)
//   - p50 latency
//   - p99 latency
// then record the run in RESULTS.md (Principle III).
func BenchmarkConcurrentIngress(b *testing.B) {
	b.ReportMetric(0, "orders/sec")
	b.ReportMetric(0, "p50-ns/op")
	b.ReportMetric(0, "p99-ns/op")

	b.Skip("TODO(spec-4): multi-producer load into ingest; measure submit→outcome latency")

	for i := 0; i < b.N; i++ {
		// TODO(spec-4): one timed submit through ingest.Client
	}
}
