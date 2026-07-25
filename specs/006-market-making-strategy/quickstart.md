# Quickstart: Market-Making Strategy Layer

**Feature**: `006-market-making-strategy`  
**Date**: 2026-07-25

Validation guide for the event-driven demo loop:
**market data → decision → Spec 2 ingestion → book/trade → market data**.

This is a **simulation / system-design demo**, not profitable or production
trading (Constitution Principle VI).

Prerequisites: Specs 1–2 and 5 implemented enough to run matcher + ingress +
market-data feed for one symbol. See `contracts/strategy-client.md` and
`data-model.md` for shapes.

---

## Prerequisites

- Go toolchain matching the repo
- Working single-symbol matcher with Spec 2 ingress
- Spec 5 outbound trade (and book-depth) events consumable in-process
- Strategy package + optional `cmd/strategy-demo` from this feature

---

## Setup

From repository root (package paths may match final layout in `plan.md`):

```bash
# Fetch modules / build (once Go module exists)
go test ./internal/strategy/...
go build -o bin/strategy-demo ./cmd/strategy-demo
```

Example fixed-spread config (conceptual; wire via flags or test fixture):

| Param | Example |
|-------|---------|
| symbol | `DEMO` |
| quote_size | `10` |
| spread_mode | `fixed` |
| fixed_half_spread | `5` |
| movement_threshold | `1` |
| seed_reference | `100` (optional) |

---

## Scenario A — Initial quotes (SC-001 / Story 1)

1. Start matcher + ingress + market-data feed for `DEMO`.
2. Establish last trade `L = 100` (scripted aggressor trade **or** seed `100`).
3. Start strategy with `S = 5`, `Q = 10`.
4. **Expect**: ingress receives buy limit `95×10` and sell limit `105×10`.
5. **Expect**: book (via normal visibility / feed depth) shows those levels
   unless immediately matched.
6. **Expect**: zero strategy calls that mutate the book directly (ingress
   recorder only).

```bash
go test ./internal/strategy/ -run TestInitialQuotesFixedSpread -count=10
```

---

## Scenario B — Requote on move (SC-002 / Story 2)

1. From Scenario A with quotes centered on `100`.
2. Publish feed trade at `L2 = 110` (`|110−100| ≥` threshold).
3. **Expect**: cancels for prior strategy order ids via ingress.
4. **Expect**: new buy `105×10`, sell `115×10` (if `S` still `5`).
5. Publish trade at `101` with threshold `5` → **Expect**: hold (no cancel/replace).

```bash
go test ./internal/strategy/ -run TestRequoteOnThreshold -count=10
```

---

## Scenario C — Closed loop evidence (SC-004 / Story 3)

1. Run matcher + feed + strategy together.
2. Script a trade so `L` is known; wait for strategy quotes.
3. Optionally aggress one strategy side from a test client via ingress.
4. **Expect**: Spec 5 shows book-depth for strategy quotes and/or a trade
   event when hit.
5. Stop strategy → **Expect**: best-effort cancels of remaining owned orders.

```bash
go test ./internal/strategy/ -run TestFeedbackLoop -count=1
# or
./bin/strategy-demo  # must print simulation non-claims at startup
```

---

## Scenario D — Honest framing (SC-005 / Story 4)

1. Run demo entrypoint or read feature README section.
2. **Expect**: explicit language that this is simulation / not profitable /
   not production trading.
3. **Expect**: no sentences claiming alpha, edge, or production trading.

---

## Dynamic spread smoke (optional)

With `spread_mode=dynamic`, drive `W` trades then requote; confirm
`half_spread` equals
`clamp(base + activity*step, min, max)` per `research.md` / `data-model.md`.

```bash
go test ./internal/strategy/ -run TestDynamicHalfSpread -count=1
```

---

## Success checklist

- [ ] Fixed spread places `L±S` at size `Q` after reference known
- [ ] Requote cancel/replace only when movement threshold met
- [ ] All strategy orders/cancels via Spec 2 ingress instrumentation
- [ ] At least one feed → decision → ingress → feed-visible effect sequence
- [ ] Demo/docs include Principle VI non-claims; no profitability claims
- [ ] Feature usable without WebSocket/UI
