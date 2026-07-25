# Tasks: Market-Making Strategy Layer

**Input**: Design documents from `/specs/006-market-making-strategy/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Required — FR-C02 / research R9 mandate deterministic decision tests with fixed feed sequences and an ingress recorder (SC-001–SC-003). Matcher replay remains Spec 3’s ownership; this feature does not alter matcher semantics (FR-012). No performance/benchmark claims (FR-C03).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Spec 6 owns `internal/strategy/` and optional `cmd/strategy-demo/`
- Specs 1–2 / 5 packages (`internal/book`, `internal/ingest`, `internal/marketdata`) are dependencies; strategy tests use fakes until those land
- Tests live as `*_test.go` beside packages (Go convention)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize Spec 6 package layout and module scaffolding

- [ ] T001 Create Go module at repository root (`go.mod`) if missing, and directories `internal/strategy/` and `cmd/strategy-demo/` per `plan.md`
- [ ] T002 [P] Add package doc comment in `internal/strategy/doc.go` stating Principle VI simulation / non-profitability framing (FR-010, FR-C05)
- [ ] T003 [P] Add stub README section or `specs/006-market-making-strategy/README.md` with Non-Claims language aligned to `spec.md` (FR-010, SC-005)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared types, validation, dependency interfaces, and test fakes that ALL user stories need

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T004 Implement `StrategyConfig` struct + `Validate()` in `internal/strategy/config.go` per `data-model.md` / `contracts/strategy-client.md` (fixed + dynamic fields, movement threshold default 1, reject ≤0 half-spread / inverted bounds / `QuoteSize < 1`)
- [ ] T005 [P] Define pure decision types (`Decision`, actions, quote intents) and strategy-owned state types (`ReferencePrice`, `QuoteSet`, `OwnedOrder`) in `internal/strategy/types.go` per `data-model.md` and `contracts/strategy-client.md` §4
- [ ] T006 [P] Define Spec 2 ingress client interface (`SubmitNewLimit`, `SubmitCancel`) and Spec 5 trade/book event consumer types used by the strategy in `internal/strategy/deps.go` (no book mutation APIs; FR-005, FR-C01)
- [ ] T007 Implement strategy order-ID generator with reserved namespace in `internal/strategy/ids.go` (FR-007; research R5)
- [ ] T008 [P] Implement `IngressRecorder` fake in `internal/strategy/fakerecorder_test.go` (or `internal/strategy/testing.go` + tests) per `contracts/strategy-client.md` §6 for SC-003 accounting
- [ ] T009 [P] Add config validation unit tests in `internal/strategy/config_test.go` covering invalid half-spread, inverted dynamic bounds, and zero quote size

**Checkpoint**: Foundation ready — user story implementation can now begin

---

## Phase 3: User Story 1 - Quote Around Last Trade from the Feed (Priority: P1) 🎯 MVP

**Goal**: After a last-trade (or seed) reference is known, post resting bid/ask at `L±S` size `Q` only via Spec 2 ingress

**Independent Test**: Drive a fixed last-trade (or seed); assert ingress receives buy `L−S` and sell `L+S` at size `Q`, and no submits before reference is valid

### Tests for User Story 1 ⚠️

> Write tests FIRST; they MUST FAIL before implementation. Deterministic for fixed feed/config (FR-C02).

- [ ] T010 [P] [US1] Add `TestInitialQuotesFixedSpread` in `internal/strategy/decision_test.go` (or `quotes_test.go`) asserting bid `L−S` / ask `L+S` / size `Q` for fixed mode (SC-001; quickstart Scenario A; run with `-count=10`)
- [ ] T011 [P] [US1] Add `TestNoQuotesWithoutReference` in `internal/strategy/decision_test.go` asserting hold/no ingress when no last trade and no seed (spec US1 acceptance #2)
- [ ] T012 [P] [US1] Add `TestSeedReferenceUntilFirstTrade` in `internal/strategy/decision_test.go` asserting seed used only until first trade then last-trade takes over (FR-002)

### Implementation for User Story 1

- [ ] T013 [P] [US1] Implement fixed half-spread quote price computation in `internal/strategy/quotes.go` (`Bid = L−S`, `Ask = L+S`, size `Q`; FR-003)
- [ ] T014 [US1] Implement pure `Decide` for `quote_initial` / `hold` / `pause` paths in `internal/strategy/decision.go` given `(config, state, event)` (contracts §4; FR-C02)
- [ ] T015 [US1] Implement run-loop skeleton in `internal/strategy/runner.go`: consume Spec 5 trade events for one symbol, apply `Decide`, submit new limits only via ingress interface (FR-001, FR-005, FR-C01, FR-C06)
- [ ] T016 [US1] Wire owned-order tracking into `QuoteSet` after successful initial submit in `internal/strategy/runner.go` / `internal/strategy/quotes.go` (FR-007)
- [ ] T017 [US1] Add structured status/log hooks for feed → decision → ingress on initial quote in `internal/strategy/runner.go` (or `internal/strategy/status.go`) (FR-009)
- [ ] T018 [US1] Confirm `TestInitialQuotesFixedSpread` / related US1 tests pass and ingress recorder shows 100% of submits (SC-001, SC-003)

**Checkpoint**: User Story 1 fully functional and testable independently (MVP)

---

## Phase 4: User Story 2 - Cancel/Replace Quotes When Price Moves (Priority: P1)

**Goal**: On non-owned last-trade moves past the movement threshold, cancel prior quotes and replace around the new `L`; support dynamic spread; ignore own-fill requote triggers (FR-013)

**Independent Test**: Quotes centered on L1; publish external L2 beyond threshold → cancels + new quotes around L2; sub-threshold and own-fill trades → hold

### Tests for User Story 2 ⚠️

- [ ] T019 [P] [US2] Add `TestRequoteOnThreshold` in `internal/strategy/decision_test.go` for cancel/replace around L2 when `|L2−L1| ≥` threshold (SC-002; quickstart Scenario B; `-count=10`)
- [ ] T020 [P] [US2] Add `TestHoldBelowMovementThreshold` in `internal/strategy/decision_test.go` asserting no cancel/replace on sub-threshold moves (US2 acceptance #2)
- [ ] T021 [P] [US2] Add `TestIgnoreOwnFillLastTradeForRequote` in `internal/strategy/decision_test.go` asserting own-fill last trades do not trigger requote (FR-013; SC-002)
- [ ] T022 [P] [US2] Add `TestDynamicHalfSpread` in `internal/strategy/quotes_test.go` asserting `S_dyn = clamp(base + activity*step, min, max)` per research R3 (FR-004)
- [ ] T023 [P] [US2] Add `TestRequoteAfterPartialFill` in `internal/strategy/decision_test.go` asserting cancel remaining owned id(s) and fresh two-sided set — no inventory logic (US2 acceptance #5; FR-008)

### Implementation for User Story 2

- [ ] T024 [P] [US2] Implement dynamic half-spread computation in `internal/strategy/quotes.go` using activity window over trade events (research R3; FR-004)
- [ ] T025 [US2] Extend `Decide` in `internal/strategy/decision.go` for `requote` vs `hold` using movement threshold and centered-on reference (FR-006)
- [ ] T026 [US2] Implement own-fill attribution in `internal/strategy/decision.go` (or `internal/strategy/ownership.go`): classify last trades against owned-order set so own fills never alone trigger requote (FR-013)
- [ ] T027 [US2] Implement cancel-then-replace protocol in `internal/strategy/runner.go`: Spec 2 cancels for owned ids, new IDs, fresh bid/ask submits; tolerate cancel-after-fill (research R4; FR-007)
- [ ] T028 [US2] On ingress reject/duplicate, log and retry with new order id on next quote cycle in `internal/strategy/runner.go` (edge case; no direct book write)
- [ ] T029 [US2] Confirm US2 deterministic tests pass including own-fill and dynamic-spread cases (SC-002, FR-C02)

**Checkpoint**: User Stories 1 and 2 both work independently

---

## Phase 5: User Story 3 - Close the Loop: Strategy Orders Appear on the Feed (Priority: P2)

**Goal**: Strategy-driven book/trade effects are visible on Spec 5; shutdown best-effort cancels owned resting orders

**Independent Test**: Matcher + ingest + feed + strategy (or harness): scripted `L` → quotes → optional aggressor → feed shows strategy effects; stop → cancels

### Tests for User Story 3 ⚠️

- [ ] T030 [P] [US3] Add `TestFeedbackLoop` in `internal/strategy/loop_test.go` (or integration harness) asserting feed → decision → ingress → follow-on feed-visible book/trade evidence (SC-004; quickstart Scenario C); skip or stub if Specs 2+5 packages are not yet present
- [ ] T031 [P] [US3] Add `TestShutdownCancelsOwnedOrders` in `internal/strategy/runner_test.go` asserting `shutdown_cancel` emits cancels for remaining owned ids via ingress (US3 acceptance #3)

### Implementation for User Story 3

- [ ] T032 [US3] Emit/record `FeedbackLoopEvidence` (feed event → decision → ingress ops → optional follow-on feed) via status hooks in `internal/strategy/status.go` (or extend `runner.go`) (FR-009; data-model)
- [ ] T033 [US3] Implement graceful shutdown path in `internal/strategy/runner.go`: context cancel → `shutdown_cancel` → best-effort Spec 2 cancels → phase `stopped` (US3; data-model state machine)
- [ ] T034 [US3] Pause quoting on feed gap/reconnect until a valid last-trade reference returns (no invented prices) in `internal/strategy/runner.go` (edge case; research R6)
- [ ] T035 [US3] Confirm loop/shutdown tests and SC-004 reviewer evidence path (logs or test hooks) work without privileged book access (FR-005, FR-C01)

**Checkpoint**: Closed-loop demo path verifiable; Stories 1–3 independently functional

---

## Phase 6: User Story 4 - Honest Demo Framing (Priority: P3)

**Goal**: Demo entrypoint/console path clearly labels the strategy as simulation; shows at least one requote cycle without a UI

**Independent Test**: Run demo; startup includes non-claims; observer sees requote evidence without WebSocket/UI (SC-005, SC-006)

### Tests for User Story 4 ⚠️

- [ ] T036 [P] [US4] Add `TestDemoNonClaimsBanner` in `cmd/strategy-demo/main_test.go` (or `internal/strategy/status_test.go`) asserting required simulation / not-profitable / not-production language is present (FR-010, SC-005)

### Implementation for User Story 4

- [ ] T037 [US4] Implement `cmd/strategy-demo/main.go` that prints Principle VI non-claims at startup and runs a short scripted or harness-driven quote/requote cycle with console/status output (FR-010, FR-011, SC-006)
- [ ] T038 [US4] Audit demo strings, package docs, and feature README for zero profitability / alpha / production-trading claims (FR-C05, SC-005)

**Checkpoint**: Honest framing complete; feature usable without UI

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Docs, constitution compliance, and quickstart validation across stories

- [ ] T039 [P] Align `specs/006-market-making-strategy/quickstart.md` commands with actual test names and `bin/strategy-demo` build path
- [ ] T040 [P] Confirm Spec 3 matcher deterministic replay tests (if present) still green — this feature must not redefine matching (FR-012, FR-C02 note)
- [ ] T041 [P] Audit `internal/strategy/` for book-level locks, shared book reads, unjustified third-party deps, and overclaimed strategy language (Principles I, V, VI)
- [ ] T042 [P] Confirm structured Spec 5 events remain the strategy’s input — no matcher re-architecture (Principle VII, FR-C06)
- [ ] T043 Run full quickstart validation: `go test ./internal/strategy/...` (including `-count=10` where specified) and `go build -o bin/strategy-demo ./cmd/strategy-demo`
- [ ] T044 [P] Document that no strategy throughput/latency/trading-performance numbers are published (FR-C03)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational — MVP; no dependency on US2–US4
- **User Story 2 (Phase 4)**: Depends on Foundational + US1 decision/runner/quote primitives (extends requote on the same loop)
- **User Story 3 (Phase 5)**: Depends on US1+US2 quoting/requote behavior; e2e may additionally need Specs 2+5 packages
- **User Story 4 (Phase 6)**: Depends on US1+US2 for a visible requote cycle; can use harness without full UI
- **Polish (Phase 7)**: Depends on desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: After Foundational — independent MVP
- **User Story 2 (P1)**: After US1 core quote path — independently testable via pure `Decide` + fake ingress
- **User Story 3 (P2)**: After US1+US2 — loop evidence + shutdown
- **User Story 4 (P3)**: After US1+US2 (demo) — framing only; non-blocking for loop completeness per FR-011

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Types/config before decision logic
- Decision before runner integration
- Core quoting before requote/loop/demo polish

### Parallel Opportunities

- T002–T003 (Setup docs) in parallel
- T005–T006, T008–T009 (Foundational) in parallel after T004 where noted
- T010–T012 (US1 tests) in parallel
- T019–T023 (US2 tests) in parallel
- T030–T031 (US3 tests) in parallel
- T039–T042, T044 (Polish) in parallel

---

## Parallel Example: User Story 1

```bash
# Launch US1 tests together (expect FAIL before implementation):
Task: "TestInitialQuotesFixedSpread in internal/strategy/decision_test.go"
Task: "TestNoQuotesWithoutReference in internal/strategy/decision_test.go"
Task: "TestSeedReferenceUntilFirstTrade in internal/strategy/decision_test.go"

# After Decide/quotes stubs exist:
Task: "Fixed half-spread computation in internal/strategy/quotes.go"
# Then runner depends on Decide + quotes:
Task: "Run loop in internal/strategy/runner.go"
```

---

## Parallel Example: User Story 2

```bash
# Launch US2 tests together:
Task: "TestRequoteOnThreshold in internal/strategy/decision_test.go"
Task: "TestHoldBelowMovementThreshold in internal/strategy/decision_test.go"
Task: "TestIgnoreOwnFillLastTradeForRequote in internal/strategy/decision_test.go"
Task: "TestDynamicHalfSpread in internal/strategy/quotes_test.go"
Task: "TestRequoteAfterPartialFill in internal/strategy/decision_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: `go test ./internal/strategy/ -run TestInitialQuotesFixedSpread -count=10`
5. Demo/harness optional until US4

### Incremental Delivery

1. Setup + Foundational → foundation ready
2. US1 → initial quotes via ingress (MVP)
3. US2 → requote + own-fill ignore + dynamic spread
4. US3 → feedback-loop evidence + shutdown cancels
5. US4 → honest demo framing
6. Polish → quickstart + constitution audit

### Parallel Team Strategy

1. Team completes Setup + Foundational together
2. After Foundational:
   - Developer A: User Story 1
   - Developer B: US2 test stubs + dynamic spread helpers (merge after US1 Decide lands)
3. US3/US4 follow once quoting/requote exist

---

## Notes

- [P] tasks = different files, no dependencies on incomplete sibling tasks
- [Story] label maps task to US1–US4 for traceability
- One demo strategy instance per symbol; multi-strategy out of scope
- Strategy MUST NOT lock, mutate, or shared-memory-read the book (Principle I)
- Do not publish strategy performance or PnL claims (Principles III + VI)
- Commit after each task or logical group
- Avoid: vague tasks, same-file conflicts, inventory/profitability logic
`)