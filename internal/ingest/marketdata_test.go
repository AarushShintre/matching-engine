package ingest

import (
	"context"
	"testing"

	"github.com/AarushShintre/matching-engine/internal/marketdata"
)

func TestEngineEmitsOrderedTradeAndDepthEvents(t *testing.T) {
	bus := marketdata.NewBus(16)
	events := bus.Subscribe()
	engine := NewEngine("TEST", nil)
	engine.SetBus(bus)
	if err := engine.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer engine.Stop()
	client := engine.Client()

	if _, err := client.SubmitNewLimit(NewLimitOrder{
		Symbol: "TEST", OrderID: 1, Side: "sell", Price: 100, Quantity: 5,
	}); err != nil {
		t.Fatalf("rest ask: %v", err)
	}
	assertDepthEvent(t, <-events, marketdata.SideAsk, 100, 5)

	if _, err := client.SubmitNewLimit(NewLimitOrder{
		Symbol: "TEST", OrderID: 2, Side: "buy", Price: 100, Quantity: 3,
	}); err != nil {
		t.Fatalf("cross ask: %v", err)
	}
	assertTradeEvent(t, <-events, 1, 2, 100, 3)
	assertDepthEvent(t, <-events, marketdata.SideAsk, 100, 2)

	if _, err := client.SubmitMarket(MarketOrder{
		Symbol: "TEST", OrderID: 3, Side: "buy", Quantity: 2,
	}); err != nil {
		t.Fatalf("market buy: %v", err)
	}
	assertTradeEvent(t, <-events, 1, 3, 100, 2)
	assertDepthEvent(t, <-events, marketdata.SideAsk, 100, 0)

	if _, err := client.SubmitNewLimit(NewLimitOrder{
		Symbol: "TEST", OrderID: 4, Side: "buy", Price: 99, Quantity: 4,
	}); err != nil {
		t.Fatalf("rest bid: %v", err)
	}
	assertDepthEvent(t, <-events, marketdata.SideBid, 99, 4)

	if _, err := client.SubmitCancel(CancelOrder{
		Symbol: "TEST", OrderID: 4,
	}); err != nil {
		t.Fatalf("cancel bid: %v", err)
	}
	assertDepthEvent(t, <-events, marketdata.SideBid, 99, 0)
}

func TestEngineEmitsEveryFillWhenWalkingBook(t *testing.T) {
	bus := marketdata.NewBus(16)
	events := bus.Subscribe()
	engine := NewEngine("TEST", nil)
	engine.SetBus(bus)
	if err := engine.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer engine.Stop()
	client := engine.Client()

	for id, price := range []int{100, 101} {
		if _, err := client.SubmitNewLimit(NewLimitOrder{
			Symbol: "TEST", OrderID: OrderID(id + 1), Side: "sell", Price: price, Quantity: 2,
		}); err != nil {
			t.Fatalf("rest ask %d: %v", price, err)
		}
		<-events
	}
	if _, err := client.SubmitNewLimit(NewLimitOrder{
		Symbol: "TEST", OrderID: 3, Side: "buy", Price: 101, Quantity: 3,
	}); err != nil {
		t.Fatalf("walk asks: %v", err)
	}

	assertTradeEvent(t, <-events, 1, 3, 100, 2)
	assertDepthEvent(t, <-events, marketdata.SideAsk, 100, 0)
	assertTradeEvent(t, <-events, 2, 3, 101, 1)
	assertDepthEvent(t, <-events, marketdata.SideAsk, 101, 1)
}

func TestSlowOrMissingMarketDataConsumerDoesNotChangeMatching(t *testing.T) {
	bus := marketdata.NewBus(1)
	bus.Subscribe() // Deliberately never drained.
	engine := NewEngine("TEST", nil)
	engine.SetBus(bus)
	if err := engine.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer engine.Stop()
	client := engine.Client()

	for id := 1; id <= 10; id++ {
		outcome, err := client.SubmitNewLimit(NewLimitOrder{
			Symbol: "TEST", OrderID: OrderID(id), Side: "buy", Price: 99, Quantity: 1,
		})
		if err != nil || !outcome.Accepted {
			t.Fatalf("submit %d: outcome=%#v err=%v", id, outcome, err)
		}
	}
	if bus.Dropped() == 0 {
		t.Fatal("expected slow subscriber to exercise non-blocking drop policy")
	}
}

func assertTradeEvent(t *testing.T, event marketdata.Event, maker, taker uint64, price, qty int) {
	t.Helper()
	if event.Kind != marketdata.KindTrade || event.Trade == nil {
		t.Fatalf("event: got %#v, want trade", event)
	}
	trade := event.Trade
	if trade.RestingOrderID == nil || *trade.RestingOrderID != maker ||
		trade.AggressorOrderID == nil || *trade.AggressorOrderID != taker ||
		trade.Price != price || trade.Quantity != qty || trade.Symbol != "TEST" {
		t.Fatalf("trade: got %#v", trade)
	}
}

func assertDepthEvent(t *testing.T, event marketdata.Event, side marketdata.Side, price, qty int) {
	t.Helper()
	if event.Kind != marketdata.KindBookDepth || event.BookDepth == nil {
		t.Fatalf("event: got %#v, want book depth", event)
	}
	depth := event.BookDepth
	if depth.Side != side || depth.Price != price || depth.Quantity != qty || depth.Symbol != "TEST" {
		t.Fatalf("depth: got %#v", depth)
	}
}
