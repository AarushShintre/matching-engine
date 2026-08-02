# Benchmark Suite Quickstart

Run from the repository root:

```bash
go test ./...
go test -race ./...
go test ./internal/bench -run '^$' \
  -bench '^BenchmarkConcurrentIngress$' \
  -benchmem -benchtime=3s -count=3
```

Each benchmark row must contain:

- `orders/sec` — completed ingress operations divided by measured wall time
- `p50-ns/op` — median submit-to-outcome latency
- `p99-ns/op` — 99th-percentile submit-to-outcome latency

The workload uses at least four concurrent producers. Each producer cycles
through a resting ask, a crossing market buy, a resting bid, and a cancel.
This exercises channel contention, matching, resting liquidity, and removal.

Record successful runs in `internal/bench/RESULTS.md`, including date, command,
OS/architecture, CPU, Go version, and revision state. Re-run after changes to
matching, ingress, or benchmark architecture before citing old figures.
