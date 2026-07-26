package marketdata

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestBusSubscribeShape(t *testing.T) {
	b := NewBus(8)
	ch := b.Subscribe()
	if ch == nil {
		t.Fatal("Subscribe returned nil")
	}
	err := b.Publish(Event{Kind: KindTrade, Trade: &TradeEvent{Symbol: "TEST", Price: 1, Quantity: 1}})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Publish: want ErrNotImplemented until TODO(spec-5), got %v", err)
	}
}

func TestLogConsumerStub(t *testing.T) {
	c := NewLogConsumer(io.Discard)
	err := c.Run(context.Background(), make(<-chan Event))
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Run: want ErrNotImplemented until TODO(spec-5), got %v", err)
	}
}
