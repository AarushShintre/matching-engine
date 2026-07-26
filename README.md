# Concurrent Matching Engine

In-memory matching engine in Go: price-time priority order book, channel-based single-writer concurrency, deterministic replay, and a simulation market-making client that exercises the market-data → decision → ingress feedback loop.

This is a systems-design / interview-defendable project — not a production exchange, and not a profitable trading system.

## Principles (constitution)

See [`.specify/memory/constitution.md`](.specify/memory/constitution.md).

1. **Single-writer** — one matcher goroutine owns each symbol’s book; no book-level locks
2. **Deterministic replay** — same ordered inputs → identical trade output
3. **Benchmarked claims only** — no throughput/latency numbers without `go test -bench` (orders/sec + p50/p99)
4. **Narrow scope** — limit / market / cancel, in-memory, single asset class
5. **Stdlib first**
6. **Strategy is simulation** — demo feedback loop only; not production trading
7. **Observability first** — trade and book-depth events on a channel

## Layout

```text
internal/
├── book/         # Spec 1 — order book & matching
├── ingest/       # Spec 2 — concurrent ingress (single-writer)
├── replay/       # Spec 3 — capture / deterministic replay
├── bench/        # Spec 4 — go test -bench harness (+ RESULTS.md)
├── marketdata/   # Spec 5 — trade / depth event bus + consumers
└── strategy/     # Spec 6 — simulation market-maker client
cmd/
└── strategy-demo/
specs/            # Feature specs 001–006
```

## Specs

| Spec | Feature | Package |
|------|---------|---------|
| 001 | Core order book & matching | `internal/book` |
| 002 | Concurrent ingestion | `internal/ingest` |
| 003 | Deterministic replay | `internal/replay` |
| 004 | Benchmark suite | `internal/bench` |
| 005 | Market data feed | `internal/marketdata` |
| 006 | Market-making strategy (simulation) | `internal/strategy` |

Specs 1–5 currently ship as a **harness**: clean public APIs, scenario/bench shells, and `TODO(spec-N)` markers for the remaining logic. Spec 6 is implemented as a simulation client.

Search the codebase for `TODO(spec-` to find fill-in points.

## Build & test

Requires Go (see `go.mod`).

```bash
go test ./...
go test ./internal/book/ -v          # Spec 1 scenario harness
go run ./cmd/strategy-demo           # Spec 6 simulation demo
```

Benchmarks (after Spec 4 is filled in):

```bash
go test ./internal/bench -bench=BenchmarkConcurrentIngress -benchmem
```

Record results in [`internal/bench/RESULTS.md`](internal/bench/RESULTS.md) before citing any numbers.

## Performance

No throughput or latency figures are published yet. Per Principle III, do not invent or copy unverified numbers into this README, commits, or resume material.

## Strategy disclaimer

The market-making layer is **simulation only** — an event-driven system-design demo. It is **not** profitable trading and **not** production-ready automated trading.
