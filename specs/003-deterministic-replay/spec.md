# Feature Specification: Deterministic Replay Test Suite

**Feature Branch**: `003-deterministic-replay`

**Created**: 2026-07-25

**Status**: Draft

**Input**: User description: "Spec 3 — Deterministic Replay Test Suite. Capture an input order sequence + concurrent submission timing. Replay it N times, assert identical trade output every time. This is where you prove principle 2, not just assert it."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Capture a Concurrent Submission Scenario (Priority: P1)

A correctness verifier records a realistic matching scenario that includes both
the ordered list of order operations (submits and cancels) and the relative
timing of concurrent submissions — when multiple clients submit near the same
moment through the concurrent ingestion path. The capture is durable enough to
be replayed later without re-running live concurrent clients.

**Why this priority**: Without a capture that includes concurrent timing, replay
only re-proves single-threaded matching (already covered by Spec 1). Principle II
requires proof under concurrent scheduling, not merely sequential unit scenarios.

**Independent Test**: Produce one captured scenario artifact from a concurrent
submission session; inspect it and confirm it contains the operation sequence
and timing relationships needed for replay, without requiring live clients.

**Acceptance Scenarios**:

1. **Given** a concurrent multi-client submission session against the ingestion
   path, **When** capture completes, **Then** the recorded scenario includes
   every submitted operation (limit, market, cancel) with identifiers and
   parameters sufficient to re-apply matching identically.
2. **Given** two or more operations submitted concurrently (overlapping in
   wall-clock time), **When** capture completes, **Then** the scenario records
   relative submission timing (or equivalent ordering constraints) so replay
   can recreate the same concurrency pattern.
3. **Given** a captured scenario, **When** it is loaded for replay,
   **Then** no live concurrent clients are required — the capture alone drives
   the replay harness.

---

### User Story 2 - Replay N Times With Identical Trade Output (Priority: P1)

A verifier replays a captured concurrent scenario N times. Each run may
experience different internal scheduling of concurrent work, but the ordered
trade output MUST be identical across all N runs. Divergence fails the suite.

**Why this priority**: This is the proof of Constitution Principle II — same
ordered inputs (including concurrent timing) always yield byte-identical trade
output, run after run, regardless of scheduling. The feature is not done without
this gate.

**Independent Test**: Load one captured scenario, run replay N times under
conditions that allow scheduling variation, and compare stable-serialized trade
outputs for exact equality across all runs.

**Acceptance Scenarios**:

1. **Given** a captured concurrent scenario and a configured replay count N
   (default N ≥ 100), **When** the scenario is replayed N times, **Then** every
   run produces the same ordered trade list (identical trade quantity, price,
   maker id, taker id, and sequence position for each trade).
2. **Given** N successful identical replays, **When** any single trade field
   differs on a subsequent run, **Then** the suite reports failure with enough
   detail to identify which run diverged and which trade positions differ.
3. **Given** the same captured scenario, **When** replayed on different days
   or process starts, **Then** trade output remains identical to the first
   accepted baseline for that scenario (replay is stable across process
   lifetimes, not only within one process).

---

### User Story 3 - Suite Covers Discriminative Concurrent Cases (Priority: P2)

The replay suite includes multiple named scenarios that exercise concurrent
submission patterns where incorrect concurrency would change match outcomes
(e.g., racing submits at the same price, cancel racing a cross, multi-client
bursts). The suite is discriminative: a concurrency bug that breaks ordering
guarantees causes at least one scenario to fail.

**Why this priority**: Vacuous green tests do not prove Principle II. Coverage
must include cases where scheduling-sensitive bugs would surface as divergent
trade sequences.

**Independent Test**: Run the full named scenario suite for N replays each;
confirm each scenario has a documented concurrency intent and that deliberate
violation of ingestion ordering (in a faulty test double) fails the suite.

**Acceptance Scenarios**:

1. **Given** the suite catalog, **When** reviewed, **Then** it includes at
   least: concurrent same-price submits, cancel racing a crossing order, and
   multi-client burst into a non-empty book.
2. **Given** each named scenario, **When** replayed N times, **Then** that
   scenario alone passes only if all N runs agree on trade output.
3. **Given** an intentionally ordering-broken ingestion substitute,
   **When** the suite runs, **Then** at least one scenario fails (suite is not
   vacuous).

---

### User Story 4 - Baseline Lock and Regression Gate (Priority: P2)

Once a scenario's trade output is accepted, it becomes the regression baseline.
Later engine or ingestion changes that alter trade sequences for the same
capture fail the gate unless the baseline is explicitly updated as part of an
intentional, reviewed behavior change.

**Why this priority**: Principle II must remain green as Specs 4+ land; replay
is the ongoing correctness net, not a one-shot demo.

**Independent Test**: Establish a baseline from N identical runs; mutate
matching or ingestion behavior in a controlled way and confirm the suite fails
against the locked baseline; restore behavior and confirm green.

**Acceptance Scenarios**:

1. **Given** a scenario with an accepted baseline trade sequence, **When**
   replay produces a different trade sequence, **Then** the run fails against
   the baseline (not only against cross-run equality within that invocation).
2. **Given** Spec 1 matching scenarios that remain valid under concurrent
   ingestion, **When** the deterministic replay suite runs, **Then** those
   earlier sequential correctness expectations stay green (no silent regress
   of Spec 1 behavior).

### Edge Cases

- Empty capture (no operations) — replay succeeds with empty identical trade
  output across N runs; suite still records the scenario as covered.
- Capture with only non-crossing rests (no trades) — N replays agree on empty
  trade lists and identical final book snapshots.
- Extremely tight concurrent timing (near-simultaneous submits) — replay still
  yields identical trades across N runs.
- Capture that includes rejected/invalid operations — rejections and resulting
  (non-)trades are part of the compared output sequence; N runs agree.
- Very large N or long scenarios — suite remains usable as a regression gate
  (completes in a time budget acceptable for routine verification; exact budget
  set at planning, not a product claim here).
- Partial suite run (single scenario) — supported so failures can be isolated
  without requiring the full catalog every time.
- Missing or corrupt capture artifact — replay does not silently invent inputs;
  the run fails with a clear capture-load error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support capturing a scenario that includes the full
  sequence of order operations (limit submit, market submit, cancel) applied
  through the concurrent ingestion path.
- **FR-002**: Captured scenarios MUST include concurrent submission timing
  (relative timing or equivalent ordering constraints among overlapping
  submissions) so replay recreates the same concurrency pattern.
- **FR-003**: System MUST replay a captured scenario without requiring live
  concurrent clients — the capture alone drives submissions into the ingestion
  and matching path.
- **FR-004**: System MUST run each captured scenario N times in one suite
  invocation, with a configurable N and a default of at least 100.
- **FR-005**: For each scenario, all N replay runs MUST produce identical
  ordered trade output; any divergence MUST fail the suite.
- **FR-006**: Compared trade output MUST include, for each trade, quantity,
  price, maker order id, taker order id, and stable position in the trade
  sequence (canonical comparison form fixed for the suite).
- **FR-007**: The suite MUST support locking an accepted trade sequence as a
  regression baseline and failing later runs that differ from that baseline.
- **FR-008**: The suite MUST include multiple named concurrent scenarios that
  exercise scheduling-sensitive patterns (at minimum: same-price concurrent
  submits, cancel-vs-cross race, multi-client burst into a non-empty book).
- **FR-009**: The suite MUST be discriminative: incorrect concurrent ordering
  behavior MUST cause at least one scenario to fail under verification.
- **FR-010**: Replay MUST apply Spec 1 matching semantics and Spec 2 concurrent
  ingestion behavior without redefining either; this feature adds capture,
  replay, and identity assertions only.
- **FR-011**: On divergence or baseline mismatch, the suite MUST report which
  scenario failed, which run(s) disagreed, and which trade positions differ.
- **FR-012**: Corrupted or missing capture inputs MUST fail fast with a clear
  error; the suite MUST NOT invent or silently skip required scenario data.

*Constitution-aligned requirements:*

- **FR-C01**: Replay exercises the Spec 2 single-owner, message-based ingress
  model; this feature MUST NOT introduce shared-memory locking on book state.
- **FR-C02**: This feature IS the project proof of deterministic replay:
  matching behavior under concurrent submission MUST produce identical trade
  output across N replays of the same captured inputs (Principle II).
- **FR-C03**: This feature MUST NOT claim throughput or latency numbers; those
  belong to a later benchmark feature.
- **FR-C04**: Scope is capture, multi-run replay, and identity/baseline
  assertions for concurrent scenarios — no persistence product, multi-node
  distribution, exchange wire protocols, market-data subscribers, or strategy
  layer.
- **FR-C05**: Structured event fan-out for market data remains out of scope;
  trade sequences used for comparison MUST remain ordered and stable so later
  observability features can consume them without changing match results.

### Key Entities

- **Captured Scenario**: Named artifact holding the operation sequence,
  concurrent submission timing (or equivalent constraints), and metadata
  needed for replay.
- **Operation Record**: One submit or cancel in the capture — type, identifiers,
  side/price/quantity as applicable, and timing relation to other operations.
- **Replay Run**: One full application of a captured scenario producing an
  ordered trade sequence (and optional book snapshot for diagnostics).
- **Trade Sequence**: Ordered list of trades used for cross-run and baseline
  comparison.
- **Baseline**: Accepted trade sequence for a named scenario used as a
  regression oracle.
- **Suite Result**: Pass/fail across scenarios and runs, with divergence
  details when failed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For every named scenario in the suite, 100% of N replay runs
  (N ≥ 100 by default) produce identical ordered trade sequences within a
  single suite invocation — zero tolerated divergence.
- **SC-002**: Across at least three separate suite invocations (separate
  process starts), each named scenario's trade output matches its locked
  baseline with zero mismatches.
- **SC-003**: The suite catalog includes at least three concurrent-pattern
  scenarios (same-price race, cancel-vs-cross race, multi-client burst), each
  of which passes SC-001.
- **SC-004**: When concurrent ordering guarantees are deliberately broken in a
  verification substitute, the suite fails at least one scenario (discriminative
  coverage demonstrated).
- **SC-005**: Spec 1 sequential matching expectations that remain applicable
  under concurrent ingestion stay green when the deterministic replay suite
  runs (no regression of core matching outcomes).
- **SC-006**: A reviewer can demonstrate Principle II compliance using only
  suite results (N identical runs + baseline lock) without relying on manual
  inspection of internal scheduling or informal "it looks deterministic"
  claims.

## Assumptions

- Spec 1 (`001-core-order-book`) matching semantics (limit, market, cancel,
  price-time priority, trade fields) are already defined and remain unchanged.
- Spec 2 concurrent ingestion (single-owner book with message-based ingress,
  multi-client submission) exists as a dependency and is the path exercised by
  capture and replay; this spec does not redefine that model.
- "Identical trade output" means equality of a stable, canonical comparison
  form of the ordered trade list (fixed field set and ordering), not reliance
  on non-deterministic formatting.
- Default N is 100 unless overridden for a specific run; higher N is allowed.
- Concurrent submission timing in the capture is sufficient to recreate the
  concurrency pattern under test; exact wall-clock reproduction is not required
  if relative ordering constraints preserve the intended races.
- Book snapshots may be used as diagnostic aids on failure but the pass/fail
  oracle for Principle II is the ordered trade sequence (and baseline).
- Benchmarks, persistence, networking products, market-data subscribers, and
  strategy logic are out of scope.
- Implementation technology choices are deferred to planning; this
  specification states only observable capture/replay/identity behavior.
