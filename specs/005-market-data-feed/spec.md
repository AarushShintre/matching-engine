# Feature Specification: Market Data Feed

**Feature Branch**: `005-market-data-feed`

**Created**: 2026-07-25

**Status**: Draft

**Input**: User description: "Spec 5 — Market Data Feed. Trade and book-depth events published from the matching goroutine over a channel. At minimum: structured log/consumer of events. Stretch: WebSocket broadcast for a live demo."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Observe Trades and Book Changes as Events (Priority: P1)

An operator or downstream consumer needs a reliable stream of what the matcher
just did: every trade that occurred and every meaningful change to book depth.
Those facts leave the matching owner as structured events on a dedicated outbound
path so observers never inspect or lock the live book.

**Why this priority**: Constitution Principle VII requires structured emission of
trade and book-state changes; without this stream, logging, demos, and later
strategy layers cannot observe the engine without re-architecting matching.

**Independent Test**: Drive a known sequence of orders that produce trades and
depth changes; attach a test consumer to the outbound event path and assert the
expected trade and book-depth events appear, in the order matching produced them,
without reading book state directly from the matcher.

**Acceptance Scenarios**:

1. **Given** a resting ask and a crossing buy that fully fills it, **When**
   matching completes, **Then** at least one structured trade event is emitted
   with trade price, quantity, and enough identity to relate buyer and seller
   sides of that fill.
2. **Given** a non-crossing limit that rests on an empty book, **When** the
   order is accepted, **Then** a structured book-depth event is emitted that
   reflects the new resting liquidity at that price/side.
3. **Given** a cancel that removes resting quantity, **When** the cancel is
   applied, **Then** a structured book-depth event is emitted that reflects the
   reduced or removed level.
4. **Given** a sequence that produces multiple trades and depth updates,
   **When** a consumer reads events in arrival order, **Then** the event
   sequence matches the matcher's processing order for that symbol (no
   reordering relative to how matching applied the inputs).
5. **Given** matching is processing orders, **When** events are emitted,
   **Then** emission does not require any shared-memory lock on the order book
   and does not require a second writer on book state.

---

### User Story 2 - Structured Log Consumer (Priority: P2)

An operator runs the engine locally or in a demo and wants human- and
machine-readable visibility into live activity. A first-party consumer attaches
to the outbound event stream and writes structured log lines for each trade and
book-depth event so activity is auditable without a UI.

**Why this priority**: Principle VII notes emission is independent of whether a
consumer exists, but the feature's MVP value is proven only when at least one
real consumer demonstrates the feed; structured logging is the narrowest useful
consumer and does not expand into exchange protocols.

**Independent Test**: Enable the log consumer, run a short order sequence with
known trades and depth changes, and verify structured log output contains one
record per expected event with the key fields present and distinguishable by
event kind (trade vs book-depth).

**Acceptance Scenarios**:

1. **Given** the log consumer is attached and matching emits a trade event,
   **When** the consumer processes that event, **Then** a structured log record
   appears identifying the event as a trade and including price and quantity.
2. **Given** the log consumer is attached and matching emits a book-depth
   event, **When** the consumer processes that event, **Then** a structured log
   record appears identifying the event as a book-depth update and including
   side, price, and resulting depth (or equivalent depth delta) for that level.
3. **Given** no consumer is attached, **When** matching still emits events,
   **Then** matching behavior and earlier deterministic correctness guarantees
   remain intact (emission does not depend on a live consumer to be correct).
4. **Given** a burst of events faster than log output can keep up briefly,
   **When** the consumer falls behind, **Then** matching is not blocked by the
   consumer's slowness in a way that violates single-writer progress expectations
   documented for this feature (backpressure policy per Assumptions).

---

### User Story 3 - Live Demo Broadcast (Priority: P3)

A presenter wants a live demo where external viewers see trades and depth
updates update in near real time. An optional broadcast path fans the same
outbound events out to connected demo clients.

**Why this priority**: Stretch only — useful for demos and portfolio storytelling,
but not part of the core exchange protocol surface and must not block MVP
delivery of emission + structured log (Principle IV: stay narrow).

**Independent Test**: With the optional broadcast path enabled, connect one or
more demo clients, drive matching activity, and confirm clients receive trade
and book-depth updates corresponding to the same event stream the log consumer
sees. Feature acceptance for Spec 5 MUST NOT require this story.

**Acceptance Scenarios**:

1. **Given** the optional broadcast path is enabled and a demo client is
   connected, **When** a trade event is emitted, **Then** the client receives a
   representation of that trade without needing direct access to the book.
2. **Given** the optional broadcast path is enabled and a demo client is
   connected, **When** a book-depth event is emitted, **Then** the client
   receives a representation of that depth change.
3. **Given** the optional broadcast path is disabled or unavailable, **When**
   matching runs with the log consumer only, **Then** P1 and P2 still succeed
   fully — broadcast is not required for Spec 5 MVP.

---

### Edge Cases

- What happens when matching produces multiple trades from a single incoming
  order (walking the book)? Each fill MUST produce its own trade event, and
  depth events MUST reflect intermediate or final depth consistently with the
  documented event granularity (see Assumptions).
- What happens when an incoming order rests with no trade? A book-depth event
  MUST still be emitted; no trade event is required.
- What happens when a market order finds no liquidity? No trade event is
  required; book-depth MUST remain unchanged and MUST NOT spuriously emit a
  depth change for that attempt.
- What happens when a cancel targets a missing or already-filled order? No
  spurious trade event; book-depth events MUST NOT invent liquidity that was
  never removed.
- What happens if the outbound event path has no consumer? Matching MUST still
  emit (or attempt to publish) per Principle VII without failing the match path
  solely because nobody is listening, subject to the documented backpressure
  assumption.
- What happens if a demo broadcast client disconnects mid-stream? Matching and
  the log consumer MUST continue; reconnecting clients are not required to
  receive historical gap-fill in MVP (live-forward only unless clarified later).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The matching owner for a symbol MUST publish a structured trade
  event for every fill that matching produces, including price and quantity and
  sufficient identity to correlate the two sides of the fill.
- **FR-002**: The matching owner for a symbol MUST publish a structured
  book-depth event whenever resting depth at a price level changes because of
  rest, partial fill, full fill, or cancel (including level creation and
  removal).
- **FR-003**: Trade and book-depth events MUST leave the matching owner via an
  outbound channel (or equivalent single-owner publish path) so consumers never
  take a lock on book state and never become a second writer of the book.
- **FR-004**: Event order observed by a single consumer MUST match the order in
  which the matching owner applied the corresponding effects for that symbol.
- **FR-005**: The system MUST provide at least one first-party consumer that
  writes structured log output for trade and book-depth events (event kind and
  key fields distinguishable).
- **FR-006**: Emission of events MUST NOT depend on a consumer being attached;
  matching correctness MUST hold with zero consumers.
- **FR-007**: This feature MUST NOT re-specify core matching rules, concurrent
  ingestion, deterministic replay harnesses, or the benchmark suite; it builds on
  those capabilities and MUST keep prior deterministic guarantees green.
- **FR-008**: Live demo broadcast of the same event stream to external clients
  is OPTIONAL (P3). Spec 5 MVP is complete when FR-001–FR-007 are met even if
  broadcast is absent.
- **FR-009**: Optional demo broadcast, if implemented, MUST consume the same
  outbound event stream as other consumers and MUST NOT read or lock the live
  book directly.
- **FR-010**: Optional demo broadcast MUST NOT be treated as a production
  exchange market-data protocol (no FIX/ITCH/OUCH or equivalent protocol
  obligation in this spec).

*Constitution-aligned requirements:*

- **FR-C01**: Matching for a symbol MUST use a single owner goroutine with
  channel ingress only (no book-level locks). Outbound market-data publication
  MUST preserve that single-writer ownership.
- **FR-C02**: Matching behavior covered by earlier deterministic replay tests
  MUST remain byte-identical for trade outcomes; market-data consumers MUST NOT
  alter match results.
- **FR-C03**: This feature MUST NOT introduce unverified throughput/latency
  claims for the feed; any published performance numbers MUST come from
  `go test -bench` with orders/sec and p50/p99, or MUST NOT be claimed.
- **FR-C04**: Scope MUST stay within market-data observation and demo broadcast
  stretch; persistence, multi-node distribution, and exchange protocols remain
  out of scope.

### Key Entities

- **Trade Event**: A structured record of one fill — price, quantity, and
  correlatable identifiers for the aggressing and resting sides (and symbol if
  multi-symbol appears later; single-symbol assumed for now).
- **Book-Depth Event**: A structured record that a price level's resting
  quantity (or presence) changed on bid or ask, sufficient for a consumer to
  update an external view of depth without querying the book.
- **Market Data Stream**: The ordered outbound sequence of trade and book-depth
  events published by the matching owner for consumption by loggers, demos, and
  future strategy layers.
- **Event Consumer**: A component that receives stream events and acts on them
  (structured logger required; optional live demo broadcaster).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For a fixed scenario with N expected fills, a test consumer
  receives exactly N trade events with prices and quantities matching the
  scenario's expected fills, in the same order as those fills.
- **SC-002**: For a fixed scenario with M expected depth-changing operations
  (rests, fills that change resting size, cancels), a test consumer receives
  book-depth events that allow reconstructing those M depth outcomes without
  inspecting the live book.
- **SC-003**: With the structured log consumer enabled, 100% of trade and
  book-depth events from a short scripted run appear as structured log records
  with distinguishable event kinds and the required key fields.
- **SC-004**: Running the scripted scenario with no consumers attached still
  produces the same match outcomes as with consumers attached (prior
  deterministic trade results unchanged).
- **SC-005**: MVP acceptance does not require any live demo client; if the
  optional broadcast path is built, at least one connected demo client can
  observe the same trades and depth updates as the log consumer for a live run.
- **SC-006**: Reviewers can confirm that market-data observation does not
  introduce book-level locking or a second writer on book state.

## Assumptions

- Prior specs deliver a working matcher with single-writer concurrent ingestion
  and deterministic replay tests; this feature only adds outbound observation.
- Single in-memory symbol (or the active symbol under test) is sufficient; multi-
  symbol fan-out is out of scope unless a later spec opens it.
- "Book-depth event" granularity defaults to per meaningful level change after
  each matching action (rest, each fill's effect on resting depth, cancel), not
  a full-book snapshot on every tick. Snapshot/diff encoding is a planning
  choice; consumers must be able to follow depth from the event stream alone.
- Backpressure default: the matching owner MUST NOT block indefinitely on a slow
  consumer in a way that stalls the book; if the outbound path is bounded,
  drop-newest, drop-oldest, or a documented non-blocking publish policy is
  acceptable for MVP as long as it is explicit and testable. Exact policy is
  deferred to planning with a bias toward protecting matcher progress.
- Structured log format may use the project's existing logging approach; fields
  need only be machine-parsable and stable enough for tests to assert presence.
- Optional live demo broadcast to connected clients is stretch/P3,
  local/demo-oriented, and not a commitment to authenticated multi-tenant
  market data.
- Historical replay of the market-data stream to late joiners is out of scope
  for MVP; live-forward observation is enough.
- Strategy-layer consumption of this feed belongs to a later spec; this feature
  only ensures the stream exists and is consumable.

## Out of Scope

- Re-implementing or changing price-time matching rules from Spec 1.
- Concurrent ingress design from Spec 2 (consumed as given).
- Deterministic replay harness design from Spec 3 (must stay green).
- Benchmark suite work from Spec 4 (except: no unverified feed performance claims).
- Persistence of event history, durable market-data stores, or recovery journals.
- Production exchange protocols (FIX, binary MD feeds, regulatory audit trails).
- Multi-node distribution, fan-out clusters, or cross-symbol aggregation services.
- Authenticated entitlement, throttling SLAs, or paid market-data products.
- Automated trading / strategy logic (later spec); demos must not claim
  profitability (Principle VI).
