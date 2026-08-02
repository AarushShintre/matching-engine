# Tasks: Market Data Feed

**Input**: [`spec.md`](./spec.md), [`plan.md`](./plan.md)

## Phase 1: Foundation

- [x] T001 Define structured trade and book-depth event types in `internal/marketdata/types.go`
- [x] T002 Define subscriber and consumer contracts in `internal/marketdata/bus.go`

## Phase 2: User Story 1 — Observe Trades and Depth

- [x] T003 [US1] Test ordered rest, fill, walk, and cancel events in `internal/ingest/marketdata_test.go`
- [x] T004 [US1] Implement per-subscriber ordered publication in `internal/marketdata/bus.go`
- [x] T005 [US1] Emit one trade event per fill from the owner path in `internal/ingest/engine.go`
- [x] T006 [US1] Emit resulting level depth for rests, fills, and cancels in `internal/ingest/engine.go`

## Phase 3: User Story 2 — Structured Log Consumer

- [x] T007 [US2] Test stable trade and depth JSON lines in `internal/marketdata/bus_test.go`
- [x] T008 [US2] Implement the context-aware JSON-lines consumer in `internal/marketdata/bus.go`
- [x] T009 [US2] Test and implement non-blocking drop-newest backpressure in `internal/marketdata/bus.go`

## Phase 4: User Story 3 — Optional Broadcast

- [x] T010 [US3] Confirm WebSocket broadcast remains optional and out of MVP in `specs/005-market-data-feed/plan.md`

## Phase 5: Validation

- [x] T011 Run full correctness and race tests for `internal/marketdata/` and `internal/ingest/`
- [x] T012 Document validation and backpressure behavior in `specs/005-market-data-feed/quickstart.md`
