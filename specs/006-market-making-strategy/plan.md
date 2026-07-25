# Implementation Plan: Market-Making Strategy Layer

**Branch**: `006-market-making-strategy` | **Date**: 2026-07-25 | **Spec**: [`spec.md`](./spec.md)

**Input**: Feature specification from `/specs/006-market-making-strategy/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Add an in-process **simulation** market-making client that consumes Spec 5
market-data trade events, quotes a two-sided limit around last trade (fixed or
documented dynamic half-spread), and submits all new orders/cancels only through
Spec 2 ingestion—proving the feedback loop without profitability claims or
privileged book access.

## Technical Context

**Language/Version**: Go (current stable toolchain for the repo)

**Primary Dependencies**: Go standard library only (`context`, channels,
`testing`, `log/slog` or equivalent). Consumes in-repo Spec 2 ingress client
APIs and Spec 5 market-data event types — no third-party packages.

**Storage**: N/A (in-memory engine only; strategy holds ephemeral QuoteSet /
OwnedOrder state)

**Testing**: `go test` — deterministic decision tests with fixed feed sequences
+ fake ingress recorder; optional end-to-end loop test with matcher + feed

**Target Platform**: Local developer machine (macOS/Linux) — library + CLI demo

**Project Type**: Go module library (`internal/strategy`) + optional
`cmd/strategy-demo` entrypoint

**Performance Goals**: None claimed for this feature (no throughput/latency or
trading-performance numbers)

**Constraints**: Single-writer book via Spec 2 only; no book locks/shared reads;
decisions from feed-visible data + config only; Principle VI non-claims in all
docs/demo copy; one strategy instance per symbol

**Scale/Scope**: Single-symbol demo quoting loop; fixed + simple dynamic spread;
cancel/replace on movement threshold; observability logs/hooks; no UI required

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Verify against `.specify/memory/constitution.md` (v1.0.1+). All items MUST pass
or be recorded under Complexity Tracking with explicit justification.

- **I. Single-writer**: PASS — strategy is an ingress client only; never owns or
  locks book state (FR-005, FR-C01).
- **II. Deterministic replay**: PASS — does not alter matcher semantics; adds
  deterministic strategy-decision tests for fixed feed sequences (FR-C02). Prior
  Spec 3 matcher replay remains owned by Spec 3 and must stay green.
- **III. Benchmarks**: PASS — no performance claims in this feature (FR-C03).
- **IV. Narrow scope**: PASS — in-process single-symbol demo; no persistence,
  multi-node, FIX, or live venues (FR-C04).
- **V. Stdlib first**: PASS — stdlib only; no new third-party deps (research R8).
- **VI. Strategy honesty**: PASS — Non-Claims section, FR-010, demo banner, and
  contracts mandate simulation framing; no inventory/PnL logic (FR-008).
- **VII. Observability**: PASS — builds on Spec 5 structured events; adds
  strategy status/log hooks for loop evidence (FR-009); does not re-architect
  matcher emission (FR-012, FR-C06).

**Post-design re-check**: PASS — `research.md`, `data-model.md`,
`contracts/strategy-client.md`, and `quickstart.md` preserve all gates; no
Complexity Tracking entries required.

## Project Structure

### Documentation (this feature)

```text
specs/006-market-making-strategy/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── strategy-client.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

Greenfield layout expected as Specs 1–6 land (Spec 6 owns only `strategy` + demo):

```text
cmd/
└── strategy-demo/           # Demo main; prints Principle VI non-claims
internal/
├── book/                    # Spec 1 — order book & matching (dependency)
├── ingest/                  # Spec 2 — single-writer ingress (dependency)
├── marketdata/              # Spec 5 — events + consumers (dependency)
└── strategy/                # Spec 6 — config, decision, run loop
    ├── config.go
    ├── decision.go
    ├── quotes.go
    └── runner.go
# tests live as *_test.go beside packages (Go convention)
```

**Structure Decision**: Single Go module at repo root with `internal/` packages
per constitution feature progression. Spec 6 adds `internal/strategy` and
optional `cmd/strategy-demo` only; it depends on ingest + marketdata interfaces
without expanding matcher scope.

## Complexity Tracking

> No constitution violations requiring justification.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
