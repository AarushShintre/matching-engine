# Replay fixtures

The Spec 3 regression catalog contains three scheduling-sensitive captures:

- `same_price_race.json` — concurrent same-price submits followed by a market order
- `cancel_vs_cross.json` — cancel wins the captured order against a crossing limit
- `multi_client_burst.json` — a same-wave burst walks a seeded multi-level book

`delay_ns` groups overlapping calls into replay waves. `sequence` records the
observed total ingress order within and across waves. Replay launches each wave
with concurrent goroutines, while sequence gates their handoff to Spec 2 so the
captured winner order is reproduced deterministically.

Every fixture locks an accepted ordered trade baseline.
