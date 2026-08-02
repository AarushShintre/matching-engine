# Implementation Plan: Benchmark Suite

**Branch**: `004-benchmark-suite` | **Date**: 2026-08-01 | **Spec**: [`spec.md`](./spec.md)

## Summary

Add a standard-library Go benchmark that sends a matching-heavy limit, market,
and cancel workload from at least four concurrent producers through the real
Spec 2 ingress client. Measure wall-clock throughput and per-operation
submit-to-outcome p50/p99 latency, then preserve reproducible results.

## Technical Context

- **Language**: Go 1.26.5
- **Dependencies**: Standard library plus in-repo `internal/ingest`
- **Testing**: `go test`; `go test -bench`
- **Target**: Local macOS/Linux, arm64/amd64
- **Scope**: One symbol, in-memory matcher, no feed consumer
- **Metrics**: `orders/sec`, `p50-ns/op`, `p99-ns/op`

## Constitution Check

- **I Single-writer**: PASS — all work uses `ingest.Client`.
- **II Replay**: PASS — no matching behavior changes.
- **III Benchmarks**: PASS — realistic concurrent `go test -bench` run reports
  throughput and p50/p99 together.
- **IV Narrow scope**: PASS — one in-memory symbol.
- **V Stdlib first**: PASS — no dependencies added.
- **VI Strategy honesty**: N/A.
- **VII Observability**: PASS — feed consumption is intentionally outside this
  matcher-path measurement.

**Post-design re-check**: PASS.

## Structure

```text
internal/bench/
├── bench_test.go
├── doc.go
└── RESULTS.md
specs/004-benchmark-suite/
├── plan.md
├── quickstart.md
└── tasks.md
```

## Design Decisions

- Use a four-operation producer cycle: resting ask, crossing market buy,
  resting bid, cancel.
- Use at least four producers (or `GOMAXPROCS`, when larger).
- Record latency in a per-operation slice and calculate nearest-rank
  percentiles after stopping the benchmark timer.
- Measure throughput from the same timed interval used for all operations.
- Treat results as machine-specific and record command, environment, date, and
  revision state.

## Complexity Tracking

No constitution violations.
