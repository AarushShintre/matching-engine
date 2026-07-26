# Feature Specification: Concurrent Ingestion Layer

**Feature Branch**: `002-concurrent-ingestion`

**Created**: 2026-07-25

**Status**: Draft

**Input**: User description: "Spec 2 — Concurrent Ingestion Layer. Single matching goroutine per symbol. Order-submission channel feeding that goroutine. Multiple simulated concurrent clients submitting through the channel. No locks anywhere on book state — enforce constitution principle 1."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Exclusive Matcher Owns the Book (Priority: P1)

For a given symbol, exactly one matching owner processes all book-mutating
operations. Submitters never mutate the book directly; they only hand off work
to that owner.

**Why this priority**: Exclusive ownership is constitution Principle I and the
entire concurrency model for the project. Without it, later correctness and
performance work is invalid.

**Independent Test**: Start the matcher for one symbol; confirm book mutations
occur only on the matcher path (submitters cannot call into book mutation APIs
from outside that path), and that shutting down or omitting the matcher means
submissions are not applied to the book.

**Acceptance Scenarios**:

1. **Given** a running matcher for symbol S, **When** a valid limit order is
   submitted for S, **Then** the order is eventually processed by the matcher
   and appears in trades and/or book state consistent with Spec 1 matching
   rules.
2. **Given** a running matcher for symbol S, **When** observers inspect how
   book state is updated, **Then** there is a single exclusive owner for S's
   book — not multiple concurrent writers on that book.
3. **Given** the matcher for symbol S is not running (or has been stopped),
   **When** clients attempt to submit, **Then** submissions are not applied to
   S's book (rejected, failed handoff, or equivalent non-mutation outcome).

---

### User Story 2 - Submit Through a Shared Ingress Path (Priority: P1)

Clients submit limit, market, and cancel requests onto a shared ingress path
that feeds the symbol's matcher. The matcher drains that path and applies
operations one at a time using existing Spec 1 matching behavior.

**Why this priority**: Message-based ingress is how exclusive ownership is
enforced; without it, clients would share the book.

**Independent Test**: Submit a mixed sequence of limits, markets, and cancels
only through the ingress path; assert Spec 1–compatible results and that no
alternate mutation API is required for success.

**Acceptance Scenarios**:

1. **Given** a running matcher, **When** a client submits a limit, market, or
   cancel via the ingress path, **Then** the operation is accepted for
   processing (or rejected with a clear validation/handoff error) without the
   client holding book state.
2. **Given** several operations enqueued in arrival order on the ingress path,
   **When** the matcher processes them, **Then** they are applied sequentially
   and produce the same matching outcomes Spec 1 would produce for that same
   ordered sequence.
3. **Given** a cancel submitted via ingress for a resting order, **When** the
   matcher processes it, **Then** the order is removed per Spec 1 cancel rules.

---

### User Story 3 - Multiple Concurrent Clients (Priority: P1)

Multiple simulated clients submit concurrently toward the same symbol. The
system remains correct under interleaved submission attempts: every accepted
operation is processed exactly once by the matcher, and book state never
requires shared-memory locks.

**Why this priority**: Concurrent clients are the reason Spec 2 exists; Spec 1
already covered single-threaded calls.

**Independent Test**: Run N concurrent clients (N ≥ 4) submitting mixed
operations against one symbol; assert completion without book corruption,
duplicate application of the same client operation, or lock-based book access.

**Acceptance Scenarios**:

1. **Given** N ≥ 4 concurrent clients each submitting a known set of orders,
   **When** all submissions have been processed, **Then** every accepted
   operation has been applied exactly once and final book/trade state is
   internally consistent with Spec 1 rules for some serialization of those
   operations.
2. **Given** concurrent clients submitting cancels and new orders for the same
   symbol, **When** processing completes, **Then** no resting order is both
   fully present and fully absent in a contradictory way; cancels and matches
   reflect a single serial order of application.
3. **Given** concurrent load against one symbol, **When** the implementation is
   reviewed or tested for book synchronization, **Then** book state is not
   guarded by mutexes, read-write locks, or atomic-guarded shared book fields —
   exclusivity comes from the single matcher owner only.

---

### User Story 4 - Spec 1 Behavior Preserved Under Ingestion (Priority: P2)

All Spec 1 matching scenarios still pass when operations are fed through the
concurrent ingestion path (including from a single client using ingress), not
only via the old synchronous direct path.

**Why this priority**: Ingestion must not change matching semantics; regressions
here would break the project’s correctness story.

**Independent Test**: Re-run the Spec 1 deterministic scenario suite by
submitting each operation through ingress and waiting for completion; expect
identical trade sequences and final book snapshots.

**Acceptance Scenarios**:

1. **Given** any Spec 1 recorded scenario sequence, **When** it is submitted
   in order through the ingress path, **Then** trade output and final book
   state match the Spec 1 expected results.
2. **Given** the Spec 1 scenario suite, **When** run under the ingestion
   layer, **Then** 100% of previously passing scenarios still pass.

### Edge Cases

- Ingress backlog grows while the matcher is busy (clients may block, buffer,
  or receive backpressure — outcomes must be defined and non-corrupting).
- Client submits after matcher shutdown during drain of remaining work.
- Duplicate operation identifiers under concurrent clients (same rejection
  semantics as Spec 1, applied by the matcher).
- Burst of cancels for orders not yet processed (cancel of unknown id after
  serialization).
- Very high concurrency with small books (stress consistency, not throughput
  claims).
- Attempt to share or mutate book state from a client path (must be impossible
  or a hard test failure / API absence).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide exactly one exclusive matching owner per
  symbol that alone may mutate that symbol's order book.
- **FR-002**: System MUST accept order submissions, market orders, and cancels
  from clients only via a shared ingress path that delivers work to the
  symbol's matching owner.
- **FR-003**: The matching owner MUST process ingress operations sequentially
  (one completes before the next mutates the book) using Spec 1 matching rules.
- **FR-004**: System MUST support multiple concurrent clients submitting to the
  same symbol's ingress path without requiring clients to coordinate locks with
  each other.
- **FR-005**: Book state for a symbol MUST NOT be protected by shared-memory
  locks (mutexes, read-write locks) or by atomics used as a substitute for
  exclusive ownership of book fields. Exclusivity MUST come from single-owner
  processing only.
- **FR-006**: Each accepted client operation MUST be applied at most once by
  the matcher (no double-processing of the same handoff).
- **FR-007**: Clients MUST be able to learn the outcome of a submission
  (acceptance/rejection, trades produced, and/or completion signal) without
  reading book memory directly.
- **FR-008**: Stopping the matching owner MUST stop further application of new
  submissions to that symbol's book; in-flight drain behavior MUST leave the
  book consistent with Spec 1 rules for the prefix of operations applied.
- **FR-009**: The Spec 1 deterministic scenario suite MUST remain runnable and
  passing when driven through the ingress path.
- **FR-010**: Direct client-side mutation of book structures MUST NOT be part
  of the supported concurrent API (ingress is the only supported write path).

*Constitution-aligned requirements:*

- **FR-C01**: Matching for a symbol MUST use a single owner with message-based
  ingress only — no book-level shared-memory locks (Principle I). This feature
  is the primary enforcement of FR-C01 deferred from Spec 1.
- **FR-C02**: Under concurrent client scheduling, a fixed ordered sequence
  submitted one-at-a-time through ingress MUST still yield identical trade
  output across repeated runs (Principle II baseline). Full capture of
  concurrent timing schedules for replay is deferred to the deterministic
  replay feature (Spec 3).
- **FR-C03**: This feature MUST NOT claim throughput or latency numbers
  (Principle III); stress tests may exist only to show correctness under load.
- **FR-C04**: Scope stays in-memory, single asset class, limit/market/cancel —
  no persistence, multi-node, or exchange protocols (Principle IV).
- **FR-C05**: Structured market-data fan-out to many subscribers remains
  deferred to the market data feature; matcher-owned trade results from Spec 1
  MUST remain available as operation outcomes.

### Key Entities

- **Symbol Matcher**: Exclusive owner responsible for applying operations to
  one symbol's book.
- **Ingress Path**: Shared submission pathway from clients to a symbol matcher
  (ordered handoff of operations).
- **Client Submission**: A limit, market, or cancel request originated by a
  concurrent client, including identity needed for correlation of results.
- **Operation Outcome**: Completion result returned or observed by the client —
  validation status, ordered trades, and relevant order remainders.
- **Order Book**: Same entity as Spec 1; mutated only by the symbol matcher.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With N ≥ 4 concurrent clients each sending ≥ 50 operations to one
  symbol, a correctness run completes with zero book-consistency failures and
  zero double-applied operations across 10 consecutive suite runs.
- **SC-002**: 100% of Spec 1 deterministic scenarios pass when each operation
  is submitted through the ingress path (same expected trades and final book).
- **SC-003**: A design or automated review checkpoint confirms there are no
  shared-memory locks (or lock-equivalent atomics on book fields) on the order
  book path — exclusivity is single-owner only.
- **SC-004**: Under concurrent submission, every accepted operation yields
  exactly one observable completion outcome to its client (no silent drops of
  accepted work).
- **SC-005**: When the matcher is stopped, no new client submission is applied
  to book state after stop completes (verified by post-stop probes).
- **SC-006**: Reviewers can explain, using only this feature's behavior, how
  multiple clients safely share one book without locking it.

## Assumptions

- Spec 1 core matching (limit, market, cancel, price-time priority, trade
  records) is the semantic engine behind the matcher; this feature adds
  concurrent ingress and exclusive ownership, not new order types.
- Implementation of Spec 1 may still be in progress; Spec 2 depends on Spec 1
  being functionally complete before Spec 2 implementation starts, even if
  this specification is written earlier.
- "Simulated concurrent clients" means test or harness actors issuing
  submissions in parallel — not a networked exchange protocol.
- One symbol is sufficient for this feature's acceptance; multi-symbol routing
  may exist as a thin map of symbol → matcher but is not a separate product
  surface.
- Backpressure behavior (block vs bounded buffer vs error) defaults to
  safe blocking or bounded handoff that never corrupts the book; exact policy
  may be chosen at plan time if tests still meet SC-001–SC-004.
- Full deterministic replay of concurrent timing (capture schedule, replay N
  times) is Spec 3; Spec 2 proves lock-free single-writer ingestion and
  preserves Spec 1 sequences through ingress.
- No performance or latency claims are made in this feature.
- Strategy/market-maker clients are out of scope (later feature); any client
  here is a generic submitter.
