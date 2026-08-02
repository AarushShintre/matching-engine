package marketdata

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestBusPublishesToEverySubscriberInOrder(t *testing.T) {
	b := NewBus(8)
	first := b.Subscribe()
	second := b.Subscribe()
	events := []Event{
		{Kind: KindTrade, Trade: &TradeEvent{Symbol: "TEST", Price: 100, Quantity: 2}},
		{Kind: KindBookDepth, BookDepth: &BookDepthEvent{
			Symbol: "TEST", Side: SideAsk, Price: 100, Quantity: 3,
		}},
	}
	for _, event := range events {
		if err := b.Publish(event); err != nil {
			t.Fatalf("Publish: %v", err)
		}
	}
	for subscriber, ch := range map[string]<-chan Event{"first": first, "second": second} {
		for i, want := range events {
			select {
			case got := <-ch:
				if got.Kind != want.Kind {
					t.Fatalf("%s event %d kind: got %v, want %v", subscriber, i, got.Kind, want.Kind)
				}
			case <-time.After(time.Second):
				t.Fatalf("%s event %d: timed out", subscriber, i)
			}
		}
	}
}

func TestBusDropsNewestForSlowSubscriber(t *testing.T) {
	b := NewBus(1)
	ch := b.Subscribe()
	first := Event{Kind: KindTrade, Trade: &TradeEvent{Price: 100, Quantity: 1}}
	second := Event{Kind: KindTrade, Trade: &TradeEvent{Price: 101, Quantity: 1}}

	if err := b.Publish(first); err != nil {
		t.Fatalf("first Publish: %v", err)
	}
	if err := b.Publish(second); err != nil {
		t.Fatalf("second Publish: %v", err)
	}
	if got := <-ch; got.Trade == nil || got.Trade.Price != first.Trade.Price {
		t.Fatalf("drop-newest policy retained %#v, want first event", got)
	}
	if got := b.Dropped(); got != 1 {
		t.Fatalf("Dropped: got %d, want 1", got)
	}
}

func TestPublishWithNoSubscribersDoesNotBlock(t *testing.T) {
	b := NewBus(1)
	done := make(chan struct{})
	go func() {
		_ = b.Publish(Event{Kind: KindTrade})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked without a subscriber")
	}
}

func TestLogConsumerWritesJSONLines(t *testing.T) {
	var output bytes.Buffer
	consumer := NewLogConsumer(&output)
	input := make(chan Event, 2)
	input <- Event{Kind: KindTrade, Trade: &TradeEvent{
		Symbol: "TEST", Price: 100, Quantity: 2,
	}}
	input <- Event{Kind: KindBookDepth, BookDepth: &BookDepthEvent{
		Symbol: "TEST", Side: SideAsk, Price: 100, Quantity: 3,
	}}
	close(input)

	if err := consumer.Run(context.Background(), input); err != nil {
		t.Fatalf("Run: %v", err)
	}

	decoder := json.NewDecoder(&output)
	var trade map[string]any
	if err := decoder.Decode(&trade); err != nil {
		t.Fatalf("decode trade: %v", err)
	}
	if trade["kind"] != "trade" || trade["price"] != float64(100) || trade["quantity"] != float64(2) {
		t.Fatalf("trade log: %#v", trade)
	}
	var depth map[string]any
	if err := decoder.Decode(&depth); err != nil {
		t.Fatalf("decode depth: %v", err)
	}
	if depth["kind"] != "book_depth" || depth["side"] != "ask" || depth["depth"] != float64(3) {
		t.Fatalf("depth log: %#v", depth)
	}
}
