# Benchmark Results (Spec 4)

**Status**: No valid suite run yet.

Constitution Principle III — do not cite throughput or latency numbers in README,
commits, resume material, or PRs until a row below is filled from
`go test -bench` against the concurrent ingress path.

## Workload profile (target)

| Field | Value |
|-------|--------|
| Producers | TODO (≥ 4 concurrent) |
| Path | `ingest.Client` → single-writer matcher |
| Mix | limits / markets / cancels into non-empty book |
| Notes | Must exercise real matching, not empty-book micro-stub |

## Recorded runs

| Date | Commit | Machine | orders/sec | p50 | p99 | Command |
|------|--------|---------|------------|-----|-----|---------|
| — | — | — | — | — | — | `go test ./internal/bench -bench=BenchmarkConcurrentIngress -benchmem` |

## Refresh policy

After any architecture change that touches concurrent matching or ingestion,
re-run the suite and update this table (or explicitly withdraw prior numbers).
