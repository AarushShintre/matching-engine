# Concurrent Matching Engine

In-memory matching engine in Go: price-time priority order book, channel-based single-writer concurrency, deterministic replay, and a simulation market-making client that exercises the market-data → decision → ingress feedback loop.

This is a systems-design / interview-defendable project — not a production exchange, and not a profitable trading system.

## Foreword

This project uses GitHub's Spec Kit to scaffold testing and deployment infrastructure, while I wrote all critical trade-execution logic by hand. Spec-Driven Development with the Cursor agent sped up setup of the routine testing harness and model scaffolding, letting me focus on the matching logic itself.
The engine currently implements FIFO (price-time priority) matching only. A Split FIFO/Pro-Rata allocation is planned next — modeled loosely on CME Group's hybrid algorithm, simplified for this project's scope rather than a byte-for-byte reproduction of the exchange spec. The intended design:

For an incoming order O of quantity Q, matched against one price level of the opposite book:
1. FIFO priority slice. A configured percentage of Q is allocated first, in strict time priority, to resting orders on the opposite side (e.g., an incoming bid matches against resting asks at or below its price). Call this slice P.
2. Total opposing liquidity. For the remaining quantity, total resting size at the price level is T = ∑ S(i), where S(i) is the size of each resting order.
3. Pro-rata allocation. Each resting order receives allocation(i) = floor((Q − P) × S(i) / T), rounded down to the nearest whole lot. Orders below a configurable minimum allocation threshold receive zero in this step.
4. Rounding remainder. Because allocations are floored, some quantity is typically left over: remainder = (Q − P) − ∑allocation(i).
5. Remainder distribution. The remainder is distributed FIFO, one lot at a time, across the remaining resting orders in time priority, until exhausted.
   
If the aggressor still has quantity left after exhausting this price level, the process repeats at the next price level.

Some important terms/ rules: 
1. BID : An order to BUY a certain commodity
2. ASK: An order to sell a certain commodity
3. Resting Bid/ Ask: If a bid price is lower than the lowest available ask price, or the ask price is higher than the highest bid price, no trade can occur. In this case, the order must rest inside the book within an Ask or Bid queue.
4. An Agressor is an incoming Order 
5. Matching occurs when an incoming order can be fulfilled by walking across the opposite side of the book
6. In the books, resting bids are maintained in descending order, while resting asks are maintained in ascending order. This means that the highest someone is willing to pay and the lowest someone is willing to sell for are the first entries in the bid and ask respectively.
7. The gap between these two numbers is called spread and is indicative of liquidity. Higher spread indicates lower liquidity, and vice versa.

Why Go?
A few reasons Go was a good fit for this project:

Concurrency that matches the problem. The heart of a matching engine is one simple rule: only one process should ever touch the order book at a time. Go's goroutines and channels make that rule easy to enforce by design — a single goroutine owns the book, and everything else talks to it through channels instead of fighting over locks. The safety comes from the structure of the code, not from careful discipline.

Predictable, repeatable behavior. Go doesn't have hidden async magic or surprise scheduling behavior, which makes it much easier to guarantee that running the same sequence of orders through the engine twice gives you the same result every time. That predictability matters for testing and for trusting the system's output.

Fast enough, simple enough. Go won't match C++ or Rust on raw speed, but it's plenty fast for realistic order-book throughput, and it stays easy to read and reason about. That tradeoff — slightly less raw performance for a lot more clarity — was the right one here.
Solid tooling for concurrency. Built-in race detection, benchmarking, and lightweight goroutines made it straightforward to write tests that actually exercise concurrent behavior instead of faking it.

Easy to follow. Go has a small, clear syntax with no hidden control flow or macros, so anyone reading the code can trace exactly what happens to an order step by step.

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

