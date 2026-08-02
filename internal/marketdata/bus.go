package marketdata

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

var ErrInvalidEvent = errors.New("marketdata: invalid event")

// Bus is the outbound event path from the matching owner (Principle VII).
// Consumers never lock or write the book. Each subscriber receives its own
// ordered stream.
type Bus struct {
	buffer      int
	mu          sync.RWMutex
	subscribers []chan Event
	dropped     atomic.Uint64
}

// NewBus creates a market-data bus. A non-positive buffer uses 64 events.
func NewBus(buffer int) *Bus {
	if buffer < 1 {
		buffer = 64
	}
	return &Bus{buffer: buffer}
}

// Subscribe creates a receive-only outbound stream.
func (b *Bus) Subscribe() <-chan Event {
	out := make(chan Event, b.buffer)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, out)
	b.mu.Unlock()
	return out
}

// Publish attempts to emit an event without becoming a second book writer.
//
// Publication is non-blocking. If a subscriber's buffer is full, that
// subscriber drops the newest event and Dropped is incremented. A lack of
// subscribers is valid: matching must not depend on a consumer being attached.
func (b *Bus) Publish(ev Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, subscriber := range b.subscribers {
		select {
		case subscriber <- ev:
		default:
			b.dropped.Add(1)
		}
	}
	return nil
}

// Dropped reports the total number of per-subscriber events dropped because a
// subscriber buffer was full.
func (b *Bus) Dropped() uint64 {
	return b.dropped.Load()
}

// Consumer receives market-data events.
type Consumer interface {
	Run(ctx context.Context, in <-chan Event) error
}

// LogConsumer writes structured log lines for trade and book-depth events.
type LogConsumer struct {
	w io.Writer
}

// NewLogConsumer returns a first-party structured log consumer (FR-005).
func NewLogConsumer(w io.Writer) *LogConsumer {
	if w == nil {
		w = io.Discard
	}
	return &LogConsumer{w: w}
}

// Run consumes events until ctx is done or the input channel is closed. It
// writes one JSON object per line.
func (c *LogConsumer) Run(ctx context.Context, in <-chan Event) error {
	encoder := json.NewEncoder(c.w)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-in:
			if !ok {
				return nil
			}
			record, err := logRecord(event)
			if err != nil {
				return err
			}
			if err := encoder.Encode(record); err != nil {
				return err
			}
		}
	}
}

func logRecord(event Event) (any, error) {
	switch {
	case event.Kind == KindTrade && event.Trade != nil:
		return struct {
			Kind             string  `json:"kind"`
			Symbol           string  `json:"symbol"`
			Price            int     `json:"price"`
			Quantity         int     `json:"quantity"`
			RestingOrderID   *uint64 `json:"resting_order_id,omitempty"`
			AggressorOrderID *uint64 `json:"aggressor_order_id,omitempty"`
		}{
			Kind:             "trade",
			Symbol:           event.Trade.Symbol,
			Price:            event.Trade.Price,
			Quantity:         event.Trade.Quantity,
			RestingOrderID:   event.Trade.RestingOrderID,
			AggressorOrderID: event.Trade.AggressorOrderID,
		}, nil
	case event.Kind == KindBookDepth && event.BookDepth != nil:
		return struct {
			Kind   string `json:"kind"`
			Symbol string `json:"symbol"`
			Side   Side   `json:"side"`
			Price  int    `json:"price"`
			Depth  int    `json:"depth"`
		}{
			Kind:   "book_depth",
			Symbol: event.BookDepth.Symbol,
			Side:   event.BookDepth.Side,
			Price:  event.BookDepth.Price,
			Depth:  event.BookDepth.Quantity,
		}, nil
	default:
		return nil, ErrInvalidEvent
	}
}
