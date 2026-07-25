# Contract: Strategy Client Interfaces

**Feature**: `006-market-making-strategy`  
**Date**: 2026-07-25

This feature does not redefine Spec 2 ingress or Spec 5 market-data schemas.
It consumes them and adds strategy configuration, decision outputs, and
observability hooks. Types below are logical contracts for implementation and
tests (Go structs / test fixtures); wire formats are in-process only.

---

## 1. Consumed: Market-data events (Spec 5)

Strategy MUST handle at least trade events for its symbol. Book-depth events
MAY be ignored for quoting decisions but MAY appear in loop evidence.

```text
TradeEvent {
  kind:        "trade"
  symbol:      string
  price:       int      // last-trade reference candidate
  quantity:    int
  // Minimal correlatable ids for Spec 6 own-fill attribution (FR-013).
  // Align names with Spec 5 when it lands; until then, fixtures MUST populate:
  resting_order_id:   OrderID | null   // maker/resting side if known
  aggressor_order_id: OrderID | null   // taker/aggressor side if known
}

BookDepthEvent {
  kind:        "book_depth"
  symbol:      string
  side:        "bid" | "ask"
  price:       int
  quantity:    int      // resting depth at level after change (or equiv.)
}
```

**Subscription**: Strategy registers as a Spec 5 consumer for `symbol`. It MUST
NOT call into book APIs.

**Own-fill rule for consumers**: A trade is an *own fill* for the strategy iff
`resting_order_id` or `aggressor_order_id` equals an id in the strategy’s
current OwnedOrder set. Spec 6 tests (T021/T026) MUST set these fields
explicitly in fixtures.

---

## 2. Produced: Ingress operations (Spec 2)

All strategy mutations of engine state go through Spec 2 client submission
only.

```text
NewLimitOrder {
  op:       "new_limit"
  symbol:   string
  order_id: OrderID     // from strategy namespace
  side:     "buy" | "sell"
  price:    int
  quantity: int         // QuoteSize
}

CancelOrder {
  op:       "cancel"
  symbol:   string
  order_id: OrderID     // previously owned id
}
```

**Prohibited**: any API that mutates or locks the order book from the strategy
package.

**Outcomes**: Strategy MAY observe Spec 2 completion/reject signals for
logging and OwnedOrder status; on reject it retries with a new `order_id` on
the next quote cycle.

---

## 3. Strategy configuration (this feature)

Logical config object validated before run (see `data-model.md`).

```text
StrategyConfig {
  symbol:              string
  quote_size:          int > 0
  spread_mode:         "fixed" | "dynamic"
  fixed_half_spread:   int > 0          // if fixed
  base_half_spread:    int > 0          // if dynamic
  min_half_spread:     int > 0          // if dynamic
  max_half_spread:     int >= min       // if dynamic
  activity_window:     int > 0          // if dynamic
  activity_step:       int >= 0         // if dynamic
  movement_threshold:  int > 0          // default 1
  seed_reference:      int | null
  enabled:             bool
  order_id_namespace:  opaque
}
```

**Validation errors**: configuration rejected; strategy does not start quoting.

---

## 4. Strategy decision (testable pure surface)

For deterministic tests, expose a pure function (or equivalent) of
`(config, state, event) → Decision`.

```text
Decision {
  action:       "hold" | "quote_initial" | "requote" | "pause" | "shutdown_cancel"
  reference:    int | null
  half_spread:  int | null
  cancel_ids:   [OrderID]
  new_orders:   [NewLimitOrder]    // 0 or 2 (bid+ask) for quote actions
}
```

**Guarantees**: Same `(config, state, event sequence)` ⇒ same decision
sequence (FR-C02).

**Own-fill**: If the trade event’s correlatable order id(s) match a currently
owned strategy order, `action` MUST be `hold` (not `requote`), regardless of
movement magnitude (FR-013).

---

## 5. Observability / status (FR-009)

Minimum log or test-hook records:

```text
StrategyStatus {
  phase:            "waiting" | "quoting" | "paused" | "stopped"
  reference:        int | null
  reference_source: "seed" | "last_trade" | null
  quote_set:        { bid_px, ask_px, bid_id, ask_id } | null
  last_action:      Decision.action | null
  non_claim:        string  // demo banner / constant simulation notice
}
```

Demo entrypoint MUST print non-claim text at startup, e.g.:

> Simulation only — event-driven system-design demo; not profitable or
> production trading.

---

## 6. Ingress accounting (SC-003)

Tests MUST instrument Spec 2 ingress (fake or wrapper) and assert:

- every strategy new-order and cancel increments ingress counters
- strategy package has zero calls to book mutation symbols/APIs

Contract for the fake:

```text
IngressRecorder {
  submitted: [NewLimitOrder | CancelOrder]
  count_new() int
  count_cancel() int
}
```

---

## Out of scope for these contracts

- Matcher replay byte format (Spec 3)
- Benchmark metrics (Spec 4)
- WebSocket demo UI (optional stretch; not required)
- Multi-strategy coordination
