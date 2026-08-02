# Feature Specification: Benchmark Suite

**Feature Branch**: `004-benchmark-suite`

**Created**: 2026-07-25

**Status**: Complete

**Input**: User description: "Spec 4 — Benchmark Suite. go test -bench harness simulating realistic concurrent load. Report throughput (orders/sec) and p50/p99 latency. Document results (this becomes your resume numbers) — re-run and update after any architecture change."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Measure Realistic Concurrent Load (Priority: P1)

An engineer (or reviewer) runs the project's performance measurement suite against
the concurrent matching path under a workload that resembles meaningful
contention — multiple producers submitting orders concurrently into the live
ingestion path — and receives a clear report of how many orders were processed
per second and how long order handling took at the median and at the high
percentile (p50 and p99).

**Why this priority**: Without a realistic concurrent load measurement that
includes both throughput and tail latency, the project cannot make any
performance claim under Constitution Principle III. This story is the core
deliverable.

**Independent Test**: Run the documented measurement suite once on a machine
with a known concurrent matching build; confirm output includes orders
processed per unit time plus p50 and p99 latency for the measured path.

**Acceptance Scenarios**:

1. **Given** a build that includes concurrent order ingestion and matching,
   **When** the performance measurement suite is executed, **Then** it
   completes successfully and reports throughput in orders per second.
2. **Given** the same run, **When** latency statistics are inspected,
   **Then** both p50 and p99 latency are present (not only an average or
   single summary number).
3. **Given** the suite's workload description, **When** a reviewer inspects
   it, **Then** the load is concurrent (multiple submitters) and exercises
   the real matching path — not a trivial no-op or empty-book micro-stub
   that cannot represent realistic contention.
4. **Given** two consecutive runs of the same suite on the same machine and
   build without code changes, **When** results are compared, **Then** both
   runs produce the same metric categories (orders/sec, p50, p99) so results
   are comparable over time.

---

### User Story 2 - Document Claimable Performance Numbers (Priority: P1)

A portfolio reader, hiring manager, or future contributor opens the project's
documented benchmark results and sees the latest recorded throughput and
p50/p99 latency, plus enough context (workload character, concurrency shape,
and when the numbers were captured) to interpret them honestly.

**Why this priority**: Documented, attributable numbers are the constitutionally
allowed form of performance communication (resume, README, PR discussion).
Undocumented or hand-waved speed claims are forbidden.

**Independent Test**: Open the documented results artifact; verify it contains
orders/sec, p50, p99, a short workload description, and a capture date or
equivalent provenance — without requiring the reader to re-run the suite.

**Acceptance Scenarios**:

1. **Given** a completed measurement run, **When** results are recorded in
   the project documentation for this feature, **Then** the record includes
   throughput (orders/sec), p50 latency, and p99 latency from that run.
2. **Given** the documented results, **When** a reader evaluates a performance
   claim in project materials, **Then** every numeric throughput or latency
   claim can be traced to a recorded suite run (no orphaned numbers).
3. **Given** project prose (README, commit messages, or similar public
   claims), **When** no recorded suite run exists yet, **Then** those
   materials MUST NOT state specific throughput or latency figures.

---

### User Story 3 - Refresh Numbers After Architecture Changes (Priority: P2)

After a change that alters matching, concurrency, or ingestion architecture,
an engineer re-runs the same measurement suite and updates the documented
results so published numbers still reflect the current system.

**Why this priority**: Stale resume/docs numbers after an architecture change
violate the spirit of Principle III (claims must come from a real run against
the system as it exists). Refresh is the maintenance contract for this feature.

**Independent Test**: Treat an architecture-touching change as a trigger;
re-run the suite; replace the documented figures and provenance with the new
run's output.

**Acceptance Scenarios**:

1. **Given** a merged or proposed change that affects the concurrent matching
   or ingestion path, **When** the change is considered complete for
   performance claims, **Then** the measurement suite has been re-run and
   documented results updated (or an explicit note states that claims are
   withdrawn until a new run is recorded).
2. **Given** updated documented results, **When** compared to the prior
   record, **Then** the new record shows a newer capture identity (date,
   commit, or equivalent) so readers can tell which numbers are current.
3. **Given** an architecture change with no re-run, **When** someone would
   cite the old numbers as current performance, **Then** project rules treat
   that citation as invalid until a fresh recorded run exists.

### Edge Cases

- Suite run on a machine under heavy unrelated load (results may be noisy;
  documentation should note that numbers are machine- and load-dependent).
- Comparison across different hardware without noting the environment
  (forbidden as an apples-to-apples claim; same-suite metrics only).
- Reporting only average latency or only throughput (incomplete; does not
  satisfy the feature).
- Trivial workload (single producer, empty book, or no matching) presented as
  "realistic concurrent load" (does not satisfy Story 1).
- Architecture change that does not touch matching/ingestion (refresh
  optional; not required solely by this feature).
- Failed or incomplete suite run (MUST NOT publish partial numbers as official
  results).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a repeatable performance measurement suite
  that exercises concurrent order submission into the live matching path.
- **FR-002**: The suite MUST simulate realistic concurrent load: multiple
  concurrent producers and enough matching activity that results reflect
  contention on the concurrent ingestion design — not a vacuous micro-bench.
- **FR-003**: Each successful suite run MUST report throughput as orders
  processed per second.
- **FR-004**: Each successful suite run MUST report latency at least at p50
  and p99 for the measured operation path (order submit through handling
  completion as defined by the suite).
- **FR-005**: Average-only or throughput-only reporting is insufficient;
  both throughput and p50/p99 MUST appear together for a run to count as a
  valid performance record.
- **FR-006**: Project documentation MUST include a durable record of the
  latest valid suite results (orders/sec, p50, p99) with enough workload and
  provenance context for a reader to interpret the numbers.
- **FR-007**: Any throughput or latency number used in documentation, commit
  messages, or external claims (including resume material) MUST come from a
  recorded valid suite run; unverified speed claims are forbidden.
- **FR-008**: After architecture changes that affect concurrent matching or
  ingestion, maintainers MUST re-run the suite and update the documented
  results (or withdraw numeric claims) before treating prior numbers as
  current.
- **FR-009**: The suite MUST be runnable by a contributor following project
  documentation without inventing a one-off harness.
- **FR-010**: This feature MUST NOT redefine matching rules, concurrency
  ownership, or replay correctness — those remain owned by earlier specs;
  this feature only measures and documents performance of the concurrent
  system those specs deliver.

*Constitution-aligned requirements:*

- **FR-C01**: Measurement MUST target the single-writer, channel-ingress
  concurrent design (no introducing book-level locks to "make the bench
  look better").
- **FR-C02**: Deterministic replay / correctness suites from earlier features
  MUST remain green; this feature adds performance measurement and MUST NOT
  weaken correctness gates.
- **FR-C03**: Performance numbers MUST come from `go test -bench` runs that
  report orders/sec and p50/p99 — satisfying Constitution Principle III.
- **FR-C04**: Scope is measurement and documentation of the existing
  in-memory concurrent matcher — no persistence, multi-node distribution,
  or exchange protocols.
- **FR-C05**: Prefer Go standard library / built-in testing and benchmarking
  facilities; any third-party bench dependency requires one-sentence
  justification at plan time.

### Key Entities

- **Benchmark Run**: One execution of the measurement suite — timestamp or
  commit identity, environment notes (optional but recommended), and the
  metric set produced.
- **Throughput Metric**: Orders processed per second for the run.
- **Latency Percentiles**: p50 and p99 (and optionally others) for the
  measured per-order path.
- **Workload Profile**: Description of concurrency (producer count / shape),
  order mix, and book conditions that define "realistic" for the suite.
- **Documented Results Record**: Published snapshot of the latest valid run
  used for claims.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A single documented suite run produces all three required
  figures together: orders per second, p50 latency, and p99 latency.
- **SC-002**: 100% of numeric throughput or latency claims in project
  documentation and public materials map to a recorded suite run; materials
  with no recorded run contain zero specific performance numbers.
- **SC-003**: A reviewer can re-run the suite using only project
  documentation and obtain the same metric categories (orders/sec, p50, p99)
  without inventing a custom harness.
- **SC-004**: The documented workload is recognizably concurrent and
  matching-heavy (multiple producers into the live path); a trivial
  single-threaded empty-book stub cannot satisfy the documented suite.
- **SC-005**: After an architecture change that touches concurrent matching
  or ingestion, published numbers are either refreshed from a new run or
  explicitly marked withdrawn within the same change cycle that would
  otherwise cite them.
- **SC-006**: A hiring manager or portfolio reader can cite the documented
  orders/sec and p50/p99 figures with clear provenance (what was measured
  and when) without needing to read implementation source.

## Assumptions

- Concurrent ingestion and matching (channel-based single-writer per symbol)
  already exist from earlier features; this feature measures that design
  rather than introducing it.
- Core matching semantics and deterministic replay remain owned by prior
  specs; benchmark failures do not redefine correctness — they report
  performance.
- "Realistic concurrent load" means multiple concurrent submitters and a mix
  that causes actual matching work (resting liquidity and crossing interest),
  not a precise industry FIX/market replay (out of scope).
- Absolute target numbers (e.g. "must exceed N orders/sec") are not mandated
  by this feature; the requirement is honest measurement and documentation.
  Specific targets may be set later once a baseline run exists.
- Results are machine-dependent; documentation may note hardware/OS at a
  high level but need not be a full lab report.
- Measurement method is constrained by the constitution to `go test -bench`
  with orders/sec and p50/p99; harness internals (timers, histogram approach,
  package layout) are left to planning.
- Market-data fan-out and strategy layers are out of scope for this feature;
  benches may ignore subscribers or use a minimal discard consumer if events
  are already emitted.
- No persistence, networking protocols, or multi-node benches in this feature.
