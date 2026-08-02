# Tasks: Benchmark Suite

**Input**: [`spec.md`](./spec.md), [`plan.md`](./plan.md)

## Phase 1: Foundation

- [x] T001 Define the benchmark workload and required metric units in `specs/004-benchmark-suite/plan.md`
- [x] T002 Add percentile calculation coverage in `internal/bench/bench_test.go`

## Phase 2: User Story 1 — Measure Concurrent Load

- [x] T003 [US1] Implement at least four concurrent producers using the real ingress client in `internal/bench/bench_test.go`
- [x] T004 [US1] Exercise resting limits, matching market orders, and cancels in `internal/bench/bench_test.go`
- [x] T005 [US1] Report orders/sec and submit-to-outcome p50/p99 in `internal/bench/bench_test.go`

## Phase 3: User Story 2 — Document Results

- [x] T006 [US2] Document the reproducible benchmark command in `specs/004-benchmark-suite/quickstart.md`
- [x] T007 [US2] Record the post-implementation benchmark run and provenance in `internal/bench/RESULTS.md`

## Phase 4: User Story 3 — Refresh Contract

- [x] T008 [US3] Document rerun requirements after matcher or ingestion changes in `internal/bench/RESULTS.md`
- [x] T009 Run all correctness, race, and benchmark validation commands and update current results
