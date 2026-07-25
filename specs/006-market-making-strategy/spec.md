# Feature Specification: Market-Making Strategy Layer

**Feature Branch**: `006-market-making-strategy`

**Created**: 2026-07-25

**Status**: Draft

**Input**: User description: "Spec 6 — Market-Making Strategy Layer (the \"stock picking\" extension). A strategy goroutine/client that consumes the market data feed. Simple quoting logic: post bid/ask a fixed or dynamic spread around last trade price, update on price movement. Feeds its own orders back into the Spec 2 ingestion channel — closes the loop. Explicitly scoped per constitution principle 6: this proves system design, not profitability."

## Non-Claims *(mandatory for this feature)*

This feature exists **only** to demonstrate an event-driven feedback loop:
market data → decision → order submission → book update → market data again.

- It is **not** a profitable trading strategy.
- It is **not** production-ready automated trading.
- It is **not** investment advice, alpha research, or a claim of edge.
- Documentation, demos, comments, and resume material for this feature MUST
  describe it as a **simulation / system-design demo** (Constitution Principle VI).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Quote Around Last Trade from the Feed (Priority: P1)

An operator starts the strategy client against a live (or test) market-data
feed for one symbol. After the feed provides a last-trade price, the strategy
posts a resting bid and ask around that price using a configured spread and
size, submitting those orders only through the existing concurrent ingestion
path (never by locking or writing the book directly).

**Why this priority**: Initial two-sided quoting from feed-derived last trade
is the minimum proof that the strategy consumes Spec 5 events and submits via
Spec 2.

**Independent Test**: Drive a feed with a known last-trade price; start the
strategy with a fixed spread and size; assert a bid and ask appear on the book
at the expected quote prices and quantities, and that submissions entered only
via the ingestion path.

**Acceptance Scenarios**:

1. **Given** a market-data feed that has published a last-trade price L and a
   strategy configured with fixed half-spread S and size Q, **When** the
   strategy starts (or first receives L), **Then** it submits a buy limit at
   L−S and a sell limit at L+S each for quantity Q through the ingestion
   channel.
2. **Given** no last-trade price has been observed yet, **When** the strategy
   is running, **Then** it does not submit quotes until a last-trade price is
   available (or an explicitly configured seed reference price is provided).
3. **Given** the strategy has submitted its initial quotes, **When** the book
   is inspected via normal engine visibility, **Then** both the bid and ask
   rest at the quoted prices unless immediately matched by other flow.
4. **Given** any strategy order submission, **When** it is accepted, **Then**
   it is indistinguishable from any other client order at the matching layer
   (same order types and ingestion path; no privileged book access).

---

### User Story 2 - Cancel/Replace Quotes When Price Moves (Priority: P1)

While the strategy is running, the last-trade price on the feed moves. The
strategy cancels its previous resting quotes and replaces them with a new
bid/ask around the updated last-trade price (fixed or dynamic spread), so
displayed quotes track the reference without leaving stale prices indefinitely.

**Why this priority**: Continuous requoting is what makes the feedback loop
visible over time; a one-shot quote does not demonstrate the design.

**Independent Test**: Start with quotes around L1; publish a new last trade at
L2 that crosses the configured movement threshold; assert prior strategy orders
are canceled (or no longer resting) and new quotes appear around L2.

**Acceptance Scenarios**:

1. **Given** active strategy quotes centered on last trade L1, **When** the
   feed publishes a new last trade L2 that differs from L1 by at least the
   configured movement threshold, **Then** the strategy cancels the prior
   bid and ask (via ingestion cancel requests) and submits new quotes around
   L2.
2. **Given** a last-trade update that does **not** meet the movement
   threshold, **When** the feed event arrives, **Then** existing quotes remain
   unchanged.
3. **Given** dynamic spread mode is enabled, **When** a requote occurs,
   **Then** the half-spread used is derived from the configured dynamic rule
   (for example, a function of recent trade activity or a bounded range) and
   still produces one bid and one ask around the current last trade.
4. **Given** a strategy quote is partially or fully filled before a requote,
   **When** a requote is triggered, **Then** the strategy cancels any remaining
   resting strategy order it still owns and posts a fresh two-sided quote
   set for the new reference (it does not attempt to "manage inventory" as a
   profit feature).

---

### User Story 3 - Close the Loop: Strategy Orders Appear on the Feed (Priority: P2)

Strategy-originated orders that rest or trade cause the same book-change and
trade events as any other client. Those events flow through the market-data
feed, so an observer can see the full cycle: feed → strategy decision →
ingestion → match/book update → feed again.

**Why this priority**: Closing the loop is the point of Spec 6; it depends on
Stories 1–2 and on Specs 2 and 5 already working.

**Independent Test**: Run matcher + ingestion + feed + strategy with a
deterministic scripted trade that establishes last price; observe strategy
quotes; optionally aggress one side; confirm subsequent feed events reflect
strategy-driven book or trade changes.

**Acceptance Scenarios**:

1. **Given** strategy quotes resting on the book, **When** a market-data
   subscriber reads the feed, **Then** book-change (or equivalent depth)
   events show the strategy's bid and ask levels.
2. **Given** an external order that trades against a strategy quote, **When**
   the trade occurs, **Then** a trade event appears on the feed and the
   strategy's subsequent behavior (if any) is driven only by feed-visible
   information plus its own configuration — not by private book locks.
3. **Given** the strategy is stopped, **When** shutdown completes, **Then** it
   cancels any remaining resting strategy orders it owns (best effort) so the
   book is not left with orphaned demo quotes without an owner process.

---

### User Story 4 - Honest Demo Framing (Optional Stretch) (Priority: P3)

An operator or reviewer can run a short demonstration that clearly labels the
strategy as a simulation of event-driven design. A dedicated WebSocket or rich
UI is **not** required; console output, a test harness log, or a minimal
status line is sufficient.

**Why this priority**: Framing prevents overclaiming; presentation polish is
secondary to the loop itself.

**Independent Test**: Run the demo path and confirm output includes explicit
simulation / non-profitability language and shows at least one requote cycle.

**Acceptance Scenarios**:

1. **Given** the demo entrypoint is run, **When** it starts the strategy,
   **Then** startup messaging states that the strategy is a system-design
   simulation and not profitable or production trading.
2. **Given** at least one last-trade change that triggers a requote, **When**
   the demo runs to completion of that cycle, **Then** an observer can see
   evidence of cancel/replace around the new last trade without needing a
   separate UI product.

### Edge Cases

- Feed reconnect or temporary gap: strategy pauses new quotes until a valid
  last-trade reference is available again; it does not invent prices.
- Last trade moves by less than the movement threshold: no requote.
- Cancel of a strategy order that already filled: unsuccessful cancel is
  tolerated; strategy continues with a fresh quote set on next requote.
- Ingestion backpressure or reject (duplicate id, invalid size): strategy
  logs/records the failure and retries with a new order id on the next quote
  cycle rather than writing the book directly.
- Strategy quote crosses the opposite side immediately (spread too tight vs
  book): resulting trades are normal engine behavior; strategy does not claim
  this as intentional "edge."
- Multiple strategy instances on the same symbol: out of scope for this
  feature (assume at most one demo strategy per symbol).
- Dynamic spread producing zero or negative half-spread: rejected by
  configuration validation; strategy does not start or does not requote until
  configuration is valid.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a strategy client that consumes market-data
  events from the Spec 5 feed for a single configured symbol.
- **FR-002**: System MUST derive its quote reference from the feed's last-trade
  price (or an explicitly configured seed reference used only until the first
  trade is observed).
- **FR-003**: System MUST support a fixed half-spread mode that places a bid
  at reference−S and an ask at reference+S for configured size Q.
- **FR-004**: System MUST support a dynamic half-spread mode whose spread is
  computed from a documented, deterministic rule over feed-visible inputs and
  configuration bounds (still centered on the last-trade reference).
- **FR-005**: System MUST submit all strategy new orders and cancels exclusively
  through the Spec 2 concurrent ingestion path; the strategy MUST NOT lock,
  mutate, or read the order book via shared memory.
- **FR-006**: System MUST cancel and replace resting strategy quotes when the
  last-trade reference moves by at least a configured movement threshold.
- **FR-007**: System MUST use unique order identifiers for each new strategy
  order and MUST track which resting orders it currently owns for cancel on
  requote or shutdown.
- **FR-008**: System MUST treat strategy fills as ordinary matches; it MUST NOT
  implement inventory-based profitability logic, predictive signals, or
  portfolio optimization as part of this feature.
- **FR-009**: System MUST expose enough observability (logs, structured status,
  or test hooks) for a reviewer to verify: feed event in → quote decision →
  ingestion submit/cancel → subsequent feed-visible effect.
- **FR-010**: Documentation and demo copy for this feature MUST include explicit
  non-claims: not profitable, not production trading, simulation for
  event-driven design only.
- **FR-011**: A WebSocket or graphical demo UI is OPTIONAL (stretch only); the
  feature is complete without it if FR-001–FR-010 are met.
- **FR-012**: This feature MUST NOT redefine matching rules, ingestion
  ownership, deterministic replay of the core matcher, or the market-data
  event schema beyond consuming what Spec 5 already provides.

*Constitution-aligned requirements:*

- **FR-C01**: Strategy-originated orders MUST reach the symbol's single-writer
  owner only through the established message-based ingestion channel (no
  book-level locks; Principle I).
- **FR-C02**: Any automated tests that assert quote placement or requote
  behavior MUST be deterministic given a fixed feed event sequence and
  configuration (Principle II applied to strategy decision outputs, without
  claiming matcher replay belongs to this feature).
- **FR-C03**: This feature MUST NOT publish throughput or latency claims for
  the strategy; any performance numbers require Spec 4-style benchmarks and
  MUST NOT be framed as trading performance or PnL (Principle III + VI).
- **FR-C04**: Scope MUST remain a single-symbol demo quoting loop in-process
  with the existing engine — no persistence, multi-node distribution, exchange
  wire protocols, brokerage connectivity, or live-market capital deployment
  (Principle IV).
- **FR-C05**: Strategy behavior and documentation MUST comply with Principle VI
  (simulation, not profitability) in all user-facing and developer-facing text.
- **FR-C06**: The strategy MUST build on Spec 5 structured market-data events
  rather than requiring matcher re-architecture (Principle VII).

### Key Entities

- **Strategy Configuration**: Symbol, quote size, spread mode (fixed/dynamic),
  fixed half-spread or dynamic rule parameters, movement threshold, optional
  seed reference price, enable/disable flag.
- **Quote Set**: The strategy's current intended bid and ask (prices, sizes,
  and owned resting order identifiers).
- **Reference Price**: Last-trade price from the feed (or seed until first
  trade), used as the center of the quote set.
- **Strategy Decision**: For a given feed event and configuration, whether to
  hold, or cancel/replace quotes, and at which prices/sizes.
- **Feedback Loop Evidence**: Ordered observation that a feed event led to
  ingestion activity and a subsequent feed-visible book or trade effect.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In a scripted demo or test with a known last-trade L and fixed
  half-spread S, the strategy places resting bid L−S and ask L+S at the
  configured size within one quote cycle after L is observed (100% of runs in
  a 10-run suite).
- **SC-002**: When last trade moves from L1 to L2 beyond the movement
  threshold, prior strategy orders are no longer resting and new quotes around
  L2 are present after the requote cycle (100% of runs in a 10-run suite).
- **SC-003**: 100% of strategy new-order and cancel requests in the test suite
  enter only through the Spec 2 ingestion path (verified by test instrumentation
  or equivalent ingress accounting — zero direct book mutations by the
  strategy).
- **SC-004**: A reviewer can point to at least one end-to-end sequence where a
  feed event is followed by a strategy submit/cancel and then a feed-visible
  book or trade event attributable to that action.
- **SC-005**: Demo or README text for this feature contains explicit
  non-claims (not profitable / not production trading / simulation) and
  contains zero sentences that describe the strategy as profitable, alpha-
  generating, or production-ready trading.
- **SC-006**: Feature is usable for its demonstration purpose without a
  WebSocket or graphical UI; optional UI stretch, if present, does not become
  a blocker for declaring the loop complete.

## Assumptions

- Spec 2 (concurrent ingestion) and Spec 5 (market data feed) are available
  dependencies; this feature consumes them and does not re-specify them.
- One demo strategy instance per symbol is sufficient; multi-strategy
  coordination is out of scope.
- Default configuration uses fixed half-spread; dynamic mode is a supported
  alternative with a simple deterministic rule documented at plan time.
- Movement threshold defaults to one price tick (any last-trade change)
  unless configured higher to reduce churn.
- Quote size is a positive constant from configuration (no size skew by side
  for inventory management).
- Strategy order identifiers are allocated from a dedicated id space or
  generator so they do not collide with scripted test client ids.
- "Stock picking" in the product nickname refers only to this demo extension's
  place in the portfolio of specs — not to equity selection or investment
  skill.
- Optional stretch items (rich UI, WebSocket dashboards) are non-blocking.
- Persistence, risk checks, real-money venues, and third-party market-data
  vendors remain out of scope.
- Third-party libraries are avoided unless the later plan justifies a gap;
  constitution prefers standard-library building blocks at implementation time.
