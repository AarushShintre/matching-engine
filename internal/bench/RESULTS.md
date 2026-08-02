# Concurrent Ingress Benchmark Results

Official results are populated only from a successful command documented in
`specs/004-benchmark-suite/quickstart.md`.

## Current Result

Captured 2026-08-01 from revision `064ba01` with the Spec 4/5 working tree
changes present.

- Machine: MacBook Pro, Apple M3 Pro (11 cores), 18 GB RAM
- Runtime: `go1.26.5 darwin/arm64`
- Command:

  ```bash
  go test ./internal/bench -run '^$' \
    -bench '^BenchmarkConcurrentIngress$' \
    -benchmem -benchtime=3s -count=3
  ```

| Run | orders/sec | p50 | p99 | Go ns/op |
|-----|-----------:|----:|----:|---------:|
| 1 | 638,084 | 12.750 µs | 100.208 µs | 1,567 |
| 2 | 636,437 | 12.791 µs | 101.084 µs | 1,571 |
| 3 | 648,268 | 12.625 µs | 98.958 µs | 1,543 |

**Representative result (median-throughput run): 638,084 orders/sec,
12.750 µs p50, and 100.208 µs p99 submit-to-outcome latency.**

Across the three runs, throughput ranged from 636,437 to 648,268 orders/sec.
The arithmetic mean was 640,930 orders/sec.

## Interpretation

The benchmark measures the real channel-ingress, single-writer matching path
with concurrent producers. Latency starts immediately before a client submit
and ends when that submit receives its outcome. Results are machine- and
load-dependent and should not be compared across hardware as if equivalent.

## Refresh Policy

Re-run and replace the current result after changes to matching, ingestion,
concurrency, or this benchmark. If no fresh run is available, withdraw numeric
performance claims until one is recorded.
