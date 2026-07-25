# Research: Market-Making Strategy Layer

**Feature**: `006-market-making-strategy`  
**Date**: 2026-07-25

All Technical Context unknowns resolved below. Earlier specs (001–005) are
assumed dependencies; this research only decides Spec 6 behavior and packaging.

---

## R1. Strategy process model

**Decision**: Run the strategy as a dedicated goroutine (or long-lived function
driven by a `context.Context`) that is a **client** of Spec 5 (market-data
consumer) and Spec 2 (ingress submitter). It never shares book memory with the
matcher.

**Rationale**: Matches Constitution Principles I and VI and FR-005 / FR-C01 —
the strategy is another concurrent client, not a privileged book peer.

**Alternatives considered**:
- Inline quoting inside the matcher owner — rejected (second responsibility on
  the single writer; violates narrow ownership and FR-012).
- Separate OS process with sockets — rejected for MVP (out of scope per
  FR-C04; in-process channel wiring is enough to prove the loop).

---

## R2. Reference price source

**Decision**: Primary reference = last-trade price from Spec 5 **trade**
events for the configured symbol. Optional `SeedReference` is used only until
the first trade event is observed, then discarded for quoting.

**Rationale**: Spec FR-002; seed unblocks demos before any fill without
inventing ongoing prices (edge case: feed gap → pause, do not invent).

**Alternatives considered**:
- Mid of best bid/ask from depth events — rejected for MVP (requires a local
  depth reconstruction and is not “last trade” as specified).
- External price feed — rejected (FR-C04; live venues out of scope).

---

## R3. Fixed and dynamic half-spread

**Decision**:
- **Fixed mode**: bid = `L − S`, ask = `L + S`, size `Q` (integer price units
  as used by Spec 1).
- **Dynamic mode** (deterministic, feed-visible only):

  ```text
  activity = number of trade events observed in the last W events (or wall-
             clock-free: last W trade events in the strategy’s event cursor)
  raw = BaseHalfSpread + (activity * ActivityStep)
  S_dyn = clamp(raw, MinHalfSpread, MaxHalfSpread)
  ```

  Quoting still centers on `L`. Configuration validation requires
  `MinHalfSpread ≥ 1` (or project minimum tick), `MaxHalfSpread ≥ MinHalfSpread`,
  and `ActivityStep ≥ 0`. Zero/negative effective half-spread cannot start or
  requote (FR edge case).

**Rationale**: Spec FR-003/FR-004 and Assumptions (“simple deterministic
rule documented at plan time”). Uses only feed-visible trade activity — no
inventory or predictive signal (FR-008).

**Alternatives considered**:
- Spread = function of `|ΔL|` only — viable but noisier on sparse trades;
  activity count is easier to explain in demos.
- Inventory-skewed quotes — rejected (FR-008 / Principle VI).

---

## R4. Movement threshold and requote protocol

**Decision**: Requote when `|L_new − L_quoted| ≥ MovementThreshold` (default
`1` price unit = any last-trade change). Protocol per cycle:

1. Cancel owned resting bid/ask via Spec 2 cancel requests (best effort if
   already filled).
2. Allocate new order IDs.
3. Submit new buy limit at `L−S` and sell limit at `L+S` size `Q` via Spec 2.

Ignore sub-threshold trade updates for quote placement. Partial fills before
requote: cancel whatever still rests; always post a fresh two-sided set (no
inventory management).

**Rationale**: FR-006, FR-007, User Story 2; keeps decision logic deterministic
for a fixed feed sequence (FR-C02).

**Alternatives considered**:
- Amend-in-place / modify order API — rejected (Specs 1–2 expose limit /
  market / cancel only).
- Continuous timer-based requote — rejected (event-driven loop is the point).

---

## R5. Order identity and ownership tracking

**Decision**: Strategy maintains a `QuoteSet` with `BidOrderID` / `AskOrderID`
(optional when flat). IDs come from a strategy-owned monotonic generator with a
reserved high bit or distinct numeric range (e.g. `strategyID << 48 | seq`) so
demo/test client IDs do not collide. On reject/duplicate, log and wait for the
next quote cycle with a new ID (no direct book write).

**Rationale**: FR-007 and Assumptions; supports cancel-on-requote and
shutdown cleanup (User Story 3).

**Alternatives considered**:
- UUID strings — unnecessary if Spec 1 uses integer IDs; prefer matching
  existing order-id type from Spec 1/2 when implemented.
- Matcher-allocated IDs returned on ack — acceptable later if ingress outcomes
  provide them; plan still requires local tracking of “currently owned” IDs.

---

## R6. Feed consumption and backpressure

**Decision**: Strategy subscribes as a Spec 5 consumer (same outbound stream
or a fan-out subscription provided by Spec 5). On reconnect/gap or missing
last trade: **pause** new quotes; do not invent `L`. Strategy must not block
the matcher; if its read loop is slow, it follows Spec 5’s consumer policy
(matcher progress protected). Strategy-side: process events sequentially in
its own goroutine.

**Rationale**: Edge cases in Spec 6; Principles I and VII.

**Alternatives considered**:
- Strategy polls book snapshots — rejected (FR-005 / no shared book read).
- Strategy receives private fill callbacks from matcher — rejected (must be
  feed-visible + config only per User Story 3).

---

## R7. Observability and demo framing

**Decision**: Structured logs (or test hooks) emit at least: feed trade
observed → decision (hold | requote) → ingress submit/cancel → note that
subsequent feed effects are Spec 5’s job. Demo entrypoint (`cmd` or test
harness) prints explicit Principle VI non-claims at startup. No WebSocket UI
required (FR-011).

**Rationale**: FR-009, FR-010, SC-004–SC-006.

**Alternatives considered**:
- Rich dashboard — optional stretch only; non-blocking.
- Silent strategy — rejected (cannot verify the loop).

---

## R8. Dependencies and module layout

**Decision**: Go standard library only for Spec 6 (channels, context, testing,
log/slog or fmt). Package `strategy` (or `internal/strategy`) depends on
ingress client types from Spec 2 and market-data event types from Spec 5 —
no new third-party deps. Repository layout:

```text
cmd/strategy-demo/     # demo main with non-claims banner
internal/strategy/     # config, quote logic, run loop
```

(Exact `internal/` neighbors for book/ingest/marketdata land with Specs 1–5;
Spec 6 only adds the strategy package and demo.)

**Rationale**: Principle V; FR-C04.

**Alternatives considered**:
- External trading SDK — rejected (out of scope / unjustified dependency).

---

## R9. Testing strategy

**Decision**: Deterministic unit/integration tests feed a fixed Spec 5 event
sequence into the strategy with a fake ingress recorder. Assert: quote prices,
cancel/replace on threshold, no ingress traffic before `L` (unless seed),
100% of submits/cancels go through the fake ingress (zero book API calls).
End-to-end loop test (when Specs 2+5 exist): scripted trade establishes `L`,
strategy quotes, aggressor hits a quote, feed shows resulting events.

**Rationale**: FR-C02, SC-001–SC-003; Principle II applied to strategy
outputs without claiming matcher replay ownership.

**Alternatives considered**:
- Timing-based flaky tests — rejected.
- PnL assertions — rejected (Principle VI).

---

## R10. Performance claims

**Decision**: Spec 6 publishes **no** throughput, latency, or trading-
performance numbers. Correctness and loop visibility only.

**Rationale**: FR-C03, Principles III and VI.

**Alternatives considered**:
- Micro-benchmark requote rate — optional later under Spec 4 rules; not part
  of this feature’s success criteria.
