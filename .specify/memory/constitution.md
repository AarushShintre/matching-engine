<!--
Sync Impact Report
- Version change: 1.0.0 → 1.0.1
- Modified principles:
  - I. Single-Writer Concurrency — rationale clarified (interview defendability)
  - VI. Strategy Layer Is Simulation — market-maker named; honesty rationale expanded
- Added sections: none
- Removed sections: none
- Templates requiring updates:
  - ✅ .specify/templates/plan-template.md (Constitution Check gates I–VII; already aligned)
  - ✅ .specify/templates/spec-template.md (added FR-C05, FR-C06)
  - ✅ .specify/templates/tasks-template.md (matcher replay tests mandatory, not optional)
  - ✅ .cursor/skills/speckit-* (reviewed; no outdated agent-specific refs)
  - ⚠ README.md / docs/quickstart.md (not present yet — create when Spec 1 lands)
- Follow-up TODOs: none
-->

# Concurrent Matching Engine (Go) Constitution

## Core Principles

### I. Single-Writer Concurrency (No Shared-Memory Locks)

The order book for a given symbol is owned exclusively by one goroutine. All order
submissions, cancels, and modifications MUST reach that goroutine only through a
channel. Mutexes, RWLocks, and atomic-guarded shared state on the book itself are
FORBIDDEN. If a future spec appears to need a lock on the book, the spec MUST be
revised — an exception to this principle is not warranted.

**Rationale**: This is the entire point of the project. It mirrors real exchange
architecture (single-writer / Disruptor-style) and is the concurrency story that
MUST remain defendable line-by-line in an interview.

### II. Proven Correctness (Deterministic Replay)

Every matching behavior MUST have a deterministic replay test: the same ordered
sequence of inputs MUST always produce byte-identical trade output, run after run,
regardless of goroutine scheduling. No feature is done without such a test.

**Rationale**: Proving correctness under concurrency is the difference between a
toy and infrastructure.

### III. Benchmarked Performance Claims Only

Any throughput or latency number used in documentation, commit messages, or
external claims (including resume material) MUST come from a `go test -bench` run
against realistic — not trivial — load, and MUST report both throughput
(orders/sec) and tail latency (p50/p99), not only an average. Unverified speed
claims are FORBIDDEN.

**Rationale**: A benchmarked number is engineering; an unverified "it's fast" is
marketing.

### IV. Deliberately Narrow Scope

The core engine handles limit orders, market orders, and cancels, for a single
asset class, in-memory only. Persistence, multi-node distribution, and exchange
protocols (e.g. FIX) are OUT OF SCOPE unless a future constitution amendment
explicitly re-opens them. New capability MUST be added by a new feature spec —
not by quietly expanding an existing spec mid-implementation.

**Rationale**: A small, deep, well-tested system is finishable and more valuable
than a wide, shallow one.

### V. Go Standard Library First

Prefer the Go standard library (`container/list`, `container/heap`, channels,
goroutines, `testing`, and related packages) over third-party dependencies unless
a specific gap cannot reasonably be closed without one. Every dependency added
MUST be justifiable in one sentence in the feature plan or PR.

**Rationale**: Keeps the project auditable and the full dependency graph
explainable.

### VI. Strategy Layer Is Simulation, Not Profitability

Any automated trading, market-maker, or signal logic exists only to demonstrate
event-driven system design and the market-data → decision → order feedback loop.
Code comments, docs, demos, and resume material MUST NOT describe that logic as
"profitable" or "production-ready" trading.

**Rationale**: Honesty preserves credibility. Nobody expects a student project to
be a real trading strategy; overclaiming undermines the parts that are genuinely
strong.

### VII. Observability Is First-Class

Every trade match and book-state change MUST emit a structured event over a
channel, independent of whether a consumer exists yet. Market data feeds and demos
MUST build on these events without re-architecting the matcher.

**Rationale**: Observability is what enables the feed and strategy layers without
retrofitting.

## Technology & Runtime Constraints

- **Language**: Go (current stable toolchain for the repo).
- **Concurrency**: Channel-based single-writer per symbol; no book-level locks
  (Principle I).
- **Storage**: In-memory only for the core engine (Principle IV).
- **Dependencies**: Standard library first; third-party packages require
  one-sentence justification (Principle V).
- **Testing**: Deterministic replay tests are mandatory for matching behavior
  (Principle II). Performance numbers require `go test -bench` with
  orders/sec and p50/p99 (Principle III).
- **Events**: Structured trade and book-change events on a channel
  (Principle VII).

## Development Workflow & Quality Gates

- Features progress one numbered spec at a time (core book → concurrency →
  replay → benchmarks → market data → strategy). Later specs MUST keep earlier
  deterministic tests green.
- Each feature follows Spec Kit: specify → plan → tasks → implement. Plans MUST
  pass the Constitution Check before Phase 0 research proceeds.
- Implementation MUST NOT expand scope beyond the active feature spec
  (Principle IV).
- PRs and reviews MUST verify Principles I–VII; lock introduction on book state,
  missing replay tests, unbenchmarked performance claims, unjustified deps,
  overclaimed strategy language, or missing event emission are automatic
  compliance failures.
- Complexity beyond the single-writer channel model MUST be justified in the
  plan's Complexity Tracking table or rejected.

## Governance

This constitution supersedes informal practice and conflicting guidance in
specs, plans, or code comments. Amendments require:

1. Explicit update to `.specify/memory/constitution.md` via `/speckit-constitution`.
2. Semantic version bump:
   - **MAJOR**: Remove or redefine a principle in a backward-incompatible way.
   - **MINOR**: Add a principle/section or materially expand guidance.
   - **PATCH**: Clarifications, wording, or non-semantic refinements.
3. Propagation to dependent Spec Kit templates and a Sync Impact Report.
4. Re-check of in-flight feature specs/plans against amended gates.

Compliance review: every `/speckit-plan` Constitution Check and every PR touching
matching, concurrency, benchmarks, market data, or strategy MUST verify
alignment with this document. Silence or "temporary" exceptions are not allowed
without a constitution amendment.

**Version**: 1.0.1 | **Ratified**: 2026-07-25 | **Last Amended**: 2026-07-25
