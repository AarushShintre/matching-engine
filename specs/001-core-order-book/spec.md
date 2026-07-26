# Feature Specification: Core Order Book & Matching Logic

**Feature Branch**: `001-core-order-book`

**Created**: 2026-07-25

**Status**: Draft

**Input**: User description: "Spec 1 — Core Order Book & Matching Logic (single-threaded). No concurrency yet. Price-level structure sorted by price. FIFO order queue per price level (price-time priority). Order types: limit, market, cancel. Partial fills, full fills, resting orders. Deterministic unit tests for every matching scenario."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Rest Limit Orders on an Empty Book (Priority: P1)

A trader (or test client) submits a limit buy or sell that does not immediately
cross the opposite side. The order rests on the book at its price, preserving
arrival order among orders at the same price.

**Why this priority**: Resting liquidity is the foundation of every later match;
without a correct book there is nothing to match against.

**Independent Test**: Submit non-crossing limits at several prices and quantities;
inspect book depth and confirm price ordering and FIFO within each price.

**Acceptance Scenarios**:

1. **Given** an empty book, **When** a limit buy is submitted at price P for
   quantity Q, **Then** the book shows one bid level at P with remaining
   quantity Q and no trades occur.
2. **Given** an empty book, **When** a limit sell is submitted at price P for
   quantity Q, **Then** the book shows one ask level at P with remaining
   quantity Q and no trades occur.
3. **Given** two limit buys at the same price submitted in order A then B,
   **When** the book is inspected, **Then** A is ahead of B in that price's
   queue (FIFO / time priority).
4. **Given** bids at multiple prices, **When** the book is inspected,
   **Then** higher bid prices rank ahead of lower bid prices for matching
   priority.
5. **Given** asks at multiple prices, **When** the book is inspected,
   **Then** lower ask prices rank ahead of higher ask prices for matching
   priority.

---

### User Story 2 - Match Crossing Limit Orders (Priority: P1)

A trader submits a limit order that crosses the opposite best price. The engine
matches by price-time priority, producing trades for full or partial fills, and
rests any unfilled remainder of the incoming limit.

**Why this priority**: Crossing limits are the primary continuous matching path;
partial fills and remainder resting must be correct before market orders or
cancels matter.

**Independent Test**: Seed resting opposite-side liquidity, submit crossing
limits of various sizes, and assert trade sequence, remaining book state, and
order remaining quantities.

**Acceptance Scenarios**:

1. **Given** a resting ask of quantity 10 at price 100, **When** a limit buy
   of quantity 10 at price 100 (or higher) arrives, **Then** one trade of
   quantity 10 at price 100 occurs and both sides are fully filled with an
   empty book.
2. **Given** a resting ask of quantity 10 at price 100, **When** a limit buy
   of quantity 4 at price 100 arrives, **Then** one trade of quantity 4 at
   price 100 occurs and the ask remains with quantity 6.
3. **Given** a resting ask of quantity 10 at price 100, **When** a limit buy
   of quantity 15 at price 100 arrives, **Then** one trade of quantity 10 at
   price 100 occurs and the unfilled buy remainder of 15 rests as a bid of
   quantity 5 at price 100.
4. **Given** two resting asks at the same price (older then newer), **When** a
   large crossing buy arrives, **Then** the older ask is fully consumed before
   any quantity is taken from the newer ask.
5. **Given** asks at prices 100 and 101, **When** a buy limit at 101 arrives
   with enough size, **Then** price 100 is fully matched before any fill at
   price 101, and trade prices equal the resting (maker) prices.

---

### User Story 3 - Market Orders Consume Available Liquidity (Priority: P2)

A trader submits a market buy or sell. The engine matches against the best
available opposite prices in priority order until the order is filled or no
liquidity remains. Unfilled market quantity does not rest on the book.

**Why this priority**: Market orders reuse the same matching rules as crossing
limits but add the "do not rest" rule; they depend on Stories 1–2.

**Independent Test**: Seed a multi-level opposite book, submit market orders of
various sizes, and assert trades, empty remainder handling, and final book.

**Acceptance Scenarios**:

1. **Given** sufficient opposite liquidity, **When** a market order for
   quantity Q arrives, **Then** trades totaling quantity Q occur and nothing
   from that order rests on the book.
2. **Given** opposite liquidity totaling less than Q, **When** a market order
   for quantity Q arrives, **Then** all available liquidity is consumed via
   trades, the unfilled market quantity is discarded (not rested), and the
   opposite book for that side is empty.
3. **Given** an empty opposite side, **When** a market order arrives,
   **Then** no trades occur, the book is unchanged, and no resting order is
   created from the market order.

---

### User Story 4 - Cancel Resting Orders (Priority: P2)

A trader cancels a previously resting order by its identifier. The order is
removed from its price queue; already-matched quantity is unaffected.

**Why this priority**: Cancels are required for a usable book and for
cancel-before-match scenarios in deterministic tests.

**Independent Test**: Rest orders, cancel by id (including mid-queue), and
confirm book depth and subsequent match order change accordingly.

**Acceptance Scenarios**:

1. **Given** a resting order with id X and remaining quantity Q, **When** a
   cancel for X is submitted, **Then** X is no longer on the book and no
   trade is produced by the cancel.
2. **Given** three resting orders at the same price (A, B, C in FIFO order),
   **When** B is canceled, **Then** A remains ahead of C and a later crossing
   order matches A before C.
3. **Given** no resting order with id X, **When** a cancel for X is submitted,
   **Then** the book is unchanged and the cancel is reported as not found
   (or equivalent unsuccessful result) without affecting other orders.
4. **Given** a resting order that was partially filled, **When** it is
   canceled, **Then** only the remaining quantity is removed; prior trades
   remain valid history.

---

### User Story 5 - Deterministic Matching Scenarios (Priority: P1)

A verifier replays fixed ordered sequences of submits and cancels and obtains
identical trade and book outcomes every run. Coverage includes crossed book,
partial fill, full fill, market against thin book, and cancel-before-match.

**Why this priority**: Constitution Principle II — matching behavior is not done
without deterministic proof. This story is the quality gate for Stories 1–4.

**Independent Test**: Run the scenario suite repeatedly; assert byte-identical
(or structurally identical, stable-serialized) trade output and final book
snapshots for each named scenario.

**Acceptance Scenarios**:

1. **Given** a recorded input sequence for "crossed book", **When** it is
   applied twice, **Then** both runs produce identical ordered trade lists and
   identical final book state.
2. **Given** a recorded input sequence for "partial fill then rest", **When**
   it is applied twice, **Then** trade quantities/prices/order ids and resting
   remainders match exactly across runs.
3. **Given** a recorded input sequence for "cancel before match", **When** an
   order is canceled before a later crossing order arrives, **Then** the
   canceled order never appears in trades and FIFO among survivors is
   preserved identically across runs.
4. **Given** the full scenario suite, **When** any single matching rule is
   violated in an implementation under test, **Then** at least one scenario
   fails (suite is discriminative, not vacuous).

### Edge Cases

- Incoming limit that exactly equals best opposite price (must match).
- Incoming limit that improves through multiple opposite price levels.
- Cancel of the only order at a price level (price level disappears).
- Cancel after full fill (order already gone — unsuccessful cancel).
- Zero or negative quantity or price on submit (rejected; book unchanged).
- Duplicate order id on submit (rejected; book unchanged).
- Market order that walks the entire opposite book and still has remainder
  (remainder discarded).
- Self-trade (same owner both sides) is out of scope — no special prevention
  unless added in a later spec.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST maintain an in-memory order book for a single symbol
  with distinct bid and ask sides.
- **FR-002**: System MUST accept limit order submissions that specify side,
  price, quantity, and a unique order identifier.
- **FR-003**: System MUST accept market order submissions that specify side,
  quantity, and a unique order identifier (no limit price).
- **FR-004**: System MUST accept cancel requests that reference an order
  identifier.
- **FR-005**: System MUST match using price-time priority: best price first;
  within the same price, earlier resting orders before later ones (FIFO).
- **FR-006**: System MUST execute trades at the resting (maker) order's price.
- **FR-007**: System MUST support partial fills: reduce remaining quantity on
  both aggressor and resting orders and leave unfilled limit remainder resting
  when applicable.
- **FR-008**: System MUST fully fill and remove orders whose remaining quantity
  reaches zero.
- **FR-009**: System MUST rest unfilled quantity of an incoming limit on its
  side at the limit price after matching is exhausted.
- **FR-010**: System MUST NOT rest unfilled quantity of an incoming market
  order; any unfilled market quantity is discarded after matching.
- **FR-011**: System MUST remove a resting order from the book on successful
  cancel and MUST leave the book unchanged on cancel of an unknown or already
  inactive order, while reporting an unsuccessful cancel result.
- **FR-012**: System MUST reject invalid submissions (non-positive price for
  limits, non-positive quantity, missing/duplicate order id) without mutating
  the book.
- **FR-013**: System MUST expose, for each processed operation, an ordered list
  of resulting trades (each with trade quantity, trade price, maker order id,
  taker order id) suitable for deterministic comparison.
- **FR-014**: System MUST expose book state sufficient to verify price levels
  and FIFO order within each level after any operation.
- **FR-015**: Processing MUST be single-threaded / strictly sequential for this
  feature: one operation fully completes before the next begins. Concurrent
  ingestion is out of scope (deferred to a later feature).

*Constitution-aligned requirements:*

- **FR-C01**: Deferred for this feature — exclusive per-symbol book ownership
  with message-based ingress belongs to the concurrent ingestion feature. This
  feature MUST NOT introduce shared-memory locking on book state as a design.
- **FR-C02**: Matching behavior MUST be covered by deterministic scenario tests
  that produce identical trade output for a fixed ordered input sequence across
  repeated runs.
- **FR-C03**: This feature MUST NOT claim throughput or latency numbers.
- **FR-C04**: Scope MUST remain limit, market, and cancel for one symbol,
  in-memory only — no persistence, multi-node distribution, or exchange
  wire protocols.
- **FR-C05**: Structured market-data channel fan-out is deferred to the market
  data feature; this feature MUST still produce ordered trade records per
  FR-013 so later features can emit events without changing match results.

### Key Entities

- **Order**: Identifier, side (buy/sell), type (limit/market), limit price
  (limits only), original quantity, remaining quantity, status
  (resting/filled/canceled/rejected).
- **Price Level**: A price on one side and the FIFO sequence of resting orders
  at that price.
- **Order Book**: Bid side and ask side collections of price levels for one
  symbol.
- **Trade**: Result of a match — quantity, price, maker order id, taker order
  id, stable position in the output sequence.
- **Operation Result**: Outcome of one submit or cancel — validation status,
  trades produced (ordered), and affected order remainders.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of documented matching scenarios (crossed book, partial
  fill, full fill, multi-level walk, market against thin/empty book,
  cancel-before-match, cancel mid-queue) pass with identical trade sequences
  on every repeated run of the suite (at least 10 consecutive runs with no
  divergence).
- **SC-002**: Given any two resting orders at the same price, a later crossing
  order always exhausts the earlier resting order before taking quantity from
  the later one (verified by scenario tests).
- **SC-003**: After any successful cancel, the canceled order never appears as
  maker or taker in subsequent trades from later operations in the same
  sequence.
- **SC-004**: Unfilled market quantity never appears as resting liquidity in
  post-operation book snapshots across the scenario suite.
- **SC-005**: Invalid operations (bad price/quantity/duplicate id) leave the
  prior book snapshot unchanged in 100% of negative test cases.
- **SC-006**: A reviewer can demonstrate correct price-time matching using only
  the scenario suite outputs (trades + book snapshots) without relying on
  concurrency or external services.

## Assumptions

- Single symbol and single in-memory book for this feature; multi-symbol
  routing arrives with concurrent ingestion later.
- Prices and quantities are positive integers in the instrument's native tick
  and lot units (no fractional representation in this feature).
- Buy = bid side; sell = ask side.
- Trade price is always the resting (maker) order's price when an aggressor
  crosses.
- No stop orders, pegged orders, icebergs, hidden liquidity, auctions, or
  fees/commissions.
- No self-trade prevention, credit checks, or account balances.
- No persistence, replication, networking, or market-data subscribers in this
  feature.
- "Byte-identical" trade output may be satisfied by a stable, canonical
  serialization of the trade list (field order and formatting fixed) compared
  across runs.
- Concurrent clients and channel-based ingestion are explicitly out of scope
  until the next feature; this feature processes a synchronous, ordered API.
- Third-party libraries are avoided unless the later plan justifies a gap;
  constitution prefers standard-library building blocks at implementation time.
