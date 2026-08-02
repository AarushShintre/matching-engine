# Market Data Feed Quickstart

Run the feed and owner-path tests:

```bash
go test ./internal/marketdata ./internal/ingest
go test -race ./internal/marketdata ./internal/ingest
```

Attach a consumer before starting activity:

```go
bus := marketdata.NewBus(256)
events := bus.Subscribe()
engine.SetBus(bus) // before engine.Start

logger := marketdata.NewLogConsumer(os.Stdout)
go logger.Run(ctx, events)
```

Trade records contain symbol, price, quantity, maker/resting order ID, and
taker/aggressor order ID. Depth records contain symbol, side, price, and the
resulting aggregate quantity at that level.

Each subscriber has an independent bounded buffer. Publishing never blocks the
matcher. When a subscriber is full, the newest event for that subscriber is
dropped and `bus.Dropped()` increments. Consumers that require gap-free state
must monitor this counter and arrange recovery; snapshot recovery and durable
history are outside MVP scope.
