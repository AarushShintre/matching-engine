# Data Model: Market-Making Strategy Layer

**Feature**: `006-market-making-strategy`  
**Date**: 2026-07-25

Entities describe strategy-owned state only. Order book, ingress messages, and
market-data events remain owned by Specs 1, 2, and 5 respectively.

---

## StrategyConfiguration

Runtime parameters for one demo strategy instance (one symbol).

| Field | Type | Rules |
|-------|------|--------|
| `Symbol` | string | Non-empty; must match engine symbol under test |
| `QuoteSize` | positive integer | `Q ≥ 1`; same size both sides |
| `SpreadMode` | enum | `fixed` \| `dynamic` |
| `FixedHalfSpread` | positive integer | Required when mode=`fixed`; `S ≥ 1` (tick) |
| `BaseHalfSpread` | positive integer | Required when mode=`dynamic` |
| `MinHalfSpread` | positive integer | Dynamic only; `≥ 1`; `≤ MaxHalfSpread` |
| `MaxHalfSpread` | positive integer | Dynamic only; `≥ MinHalfSpread` |
| `ActivityWindow` | positive integer | Dynamic only; `W ≥ 1` trade events |
| `ActivityStep` | non-negative integer | Dynamic only; added per trade in window |
| `MovementThreshold` | positive integer | Default `1`; requote when `\|ΔL\| ≥` threshold |
| `SeedReference` | optional positive integer | Used only until first trade-observed `L` |
| `Enabled` | bool | When false, strategy does not quote |
| `OrderIDNamespace` | opaque / uint | Distinct id space for strategy orders |

**Validation**: Reject start (or freeze requotes) if half-spread would be ≤ 0,
dynamic bounds are inverted, or `QuoteSize < 1`. Invalid config never causes
direct book writes.

---

## ReferencePrice

Center of the quote set.

| Field | Type | Rules |
|-------|------|--------|
| `Value` | integer price | Present only when known |
| `Source` | enum | `seed` \| `last_trade` |
| `LastTradePrice` | optional integer | From latest Spec 5 trade event |
| `Valid` | bool | False on startup (no seed) or after feed gap policy pause |

**Transitions**:
- `unknown` → `seed` if `SeedReference` configured and no trade yet
- `seed` \| `unknown` → `last_trade` on first trade event for symbol
- `last_trade` updates on each trade event; validity may clear on reconnect/gap
  until a new trade arrives (do not invent prices)

---

## QuoteSet

Strategy’s currently intended two-sided quote and owned resting ids.

| Field | Type | Rules |
|-------|------|--------|
| `BidPrice` / `AskPrice` | integer | `AskPrice = Ref + S`, `BidPrice = Ref − S` |
| `BidSize` / `AskSize` | positive integer | Equal to `QuoteSize` in this feature |
| `BidOrderID` / `AskOrderID` | optional order id | Set when resting (or in flight); cleared when flat |
| `CenteredOn` | integer | Reference `L` used for this set |
| `HalfSpreadUsed` | positive integer | `S` applied (fixed or dynamic result) |
| `State` | enum | `empty` \| `pending` \| `resting` \| `canceling` |

**Invariants**:
- At most one owned bid and one owned ask at a time for this feature
- Prices always derived from current valid reference + half-spread rule
- No inventory skew fields (forbidden by FR-008)

---

## StrategyDecision

Pure function output for one feed event + config + current QuoteSet.

| Field | Type | Meaning |
|-------|------|---------|
| `Action` | enum | `hold` \| `quote_initial` \| `requote` \| `pause` \| `shutdown_cancel` |
| `Reference` | integer | `L` used if acting |
| `HalfSpread` | positive integer | `S` for this decision |
| `CancelIDs` | list of order id | Prior owned ids to cancel |
| `NewBid` / `NewAsk` | optional limit intents | Price, size, new order id |

**Rules** (deterministic):
- `Enabled=false` → `hold` (no new submits; cancels only on shutdown if already quoting)
- No valid reference → `hold` or `pause` (no submit)
- First valid reference and empty quote set → `quote_initial`
- Valid reference with `|L − CenteredOn| ≥ MovementThreshold` → `requote`
- Sub-threshold trade → `hold`
- Last-trade event attributable only to a fill against a strategy-owned resting
  order (trade’s resting/aggressor id intersects current OwnedOrder set) →
  `hold` for requote purposes even if `|L − CenteredOn| ≥ MovementThreshold`
  (FR-013). Update fill/owned status from the event as needed, but do not
  emit cancel/replace solely because of that trade.
- Shutdown → `shutdown_cancel` then stop

---

## OwnedOrder

Thin tracking record for cancel/replace and shutdown.

| Field | Type | Rules |
|-------|------|--------|
| `OrderID` | order id | From strategy namespace |
| `Side` | bid \| ask | |
| `Price` / `Size` | integer | As submitted |
| `Status` | enum | `submitted` \| `resting` \| `filled` \| `canceled` \| `unknown` |

Fill/cancel awareness comes from feed-visible effects and/or Spec 2 submit
outcomes — not from locking the book. Unsuccessful cancel of an already-filled
order is tolerated.

---

## FeedbackLoopEvidence (observability)

Ordered observation for reviewers/tests (not persisted).

| Field | Type | Meaning |
|-------|------|---------|
| `FeedEvent` | Spec 5 event ref | Trigger (typically trade) |
| `Decision` | StrategyDecision summary | What the strategy chose |
| `IngressOps` | list | New-order / cancel sent via Spec 2 |
| `FollowOnFeed` | optional Spec 5 events | Book-depth / trade attributable to those ops |

---

## State Transitions (strategy run loop)

```text
                    ┌─────────────┐
                    │   stopped   │
                    └──────▲──────┘
                           │ shutdown_cancel done
┌──────────┐  start   ┌────┴─────┐  valid L   ┌──────────┐
│  init    │─────────►│ waiting  │───────────►│ quoting  │◄─┐
└──────────┘          └────┬─────┘            └────┬─────┘  │
                           │ feed gap              │ requote│
                           ▼                       └────────┘
                      ┌─────────┐
                      │ paused  │──new valid L──► quoting
                      └─────────┘
```

- `waiting`: running but no submit until reference valid
- `quoting`: maintains QuoteSet; hold or requote on trades
- `paused`: after gap/reconnect until last trade known again
- `stopped`: best-effort cancels completed; no further ingress

---

## Relationships

```text
StrategyConfiguration ──configures──► Strategy run loop
Spec 5 Trade Event ──updates──► ReferencePrice
ReferencePrice + Configuration ──produce──► StrategyDecision
StrategyDecision ──emits──► Spec 2 NewOrder / Cancel
StrategyDecision ──updates──► QuoteSet / OwnedOrder
Spec 5 BookDepth/Trade ──evidence──► FeedbackLoopEvidence
```

No relationship from strategy to OrderBook memory.
