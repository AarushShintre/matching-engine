# Implementation Plan: Market Data Feed

**Branch**: `005-market-data-feed` | **Date**: 2026-08-01 | **Spec**: [`spec.md`](./spec.md)

## Summary

Publish ordered trade and price-level depth events from the Spec 2 matcher
owner to independent buffered subscriber channels. Protect matcher progress
with a documented non-blocking drop-newest policy and provide a first-party
JSON-lines consumer.

## Technical Context

- **Language**: Go 1.26.5
- **Dependencies**: Standard library only (`context`, `encoding/json`,
  `sync`, `sync/atomic`)
- **Testing**: `go test`, `go test -race`
- **Storage**: None; live-forward in-memory events only
- **Scope**: One symbol; trade/depth stream and structured logger; no WebSocket

## Constitution Check

- **I Single-writer**: PASS — only `ingest.Engine.run` mutates and snapshots
  the book; subscribers receive values.
- **II Replay**: PASS — publication errors/drops never alter match outcomes.
- **III Benchmarks**: PASS — no unmeasured feed-performance claims.
- **IV Narrow scope**: PASS — no persistence or exchange protocol.
- **V Stdlib first**: PASS — no dependencies added.
- **VI Strategy honesty**: N/A.
- **VII Observability**: PASS — every accepted fill and meaningful level change
  is published from the matcher owner.

**Post-design re-check**: PASS.

## Structure

```text
internal/
├── ingest/
│   ├── engine.go
│   └── marketdata_test.go
└── marketdata/
    ├── types.go
    ├── bus.go
    └── bus_test.go
specs/005-market-data-feed/
├── plan.md
├── quickstart.md
└── tasks.md
```

## Design Decisions

- `Subscribe` creates a dedicated buffered channel so consumers do not compete
  for events.
- `Publish` preserves order per subscriber and never blocks. A full subscriber
  buffer drops the newest event and increments `Dropped`.
- With zero subscribers, publication succeeds immediately.
- The owner emits one trade event per fill, followed by the resulting maker
  price-level depth; limit rests and accepted cancels emit resulting depth.
- The logger emits one stable JSON object per line.
- WebSocket broadcasting remains optional and is not implemented for MVP.

## Complexity Tracking

The bus uses a mutex only to protect subscription metadata. It never guards
book state and does not create a second book writer.
