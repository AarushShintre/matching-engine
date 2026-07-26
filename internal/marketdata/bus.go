package marketdata

import (
	"context"
	"errors"
	"io"
)

var ErrNotImplemented = errors.New("marketdata: not implemented")

// Bus is the outbound event path from the matching owner (Principle VII).
// Consumers never lock or write the book.
type Bus struct {
	// TODO(spec-5): choose buffer size + drop/non-blocking publish policy
	out chan Event
}

// NewBus creates a market-data bus. Capacity is a scaffold default only.
func NewBus(buffer int) *Bus {
	if buffer < 1 {
		buffer = 64
	}
	return &Bus{out: make(chan Event, buffer)}
}

// Subscribe returns a receive-only view of the outbound stream.
func (b *Bus) Subscribe() <-chan Event {
	return b.out
}

// Publish attempts to emit an event without becoming a second book writer.
//
// TODO(spec-5): non-blocking / drop-oldest / drop-newest policy so a slow
// consumer cannot stall the matcher (see Spec 5 backpressure assumptions).
func (b *Bus) Publish(ev Event) error {
	_ = ev
	return ErrNotImplemented
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
	return &LogConsumer{w: w}
}

// Run consumes events until ctx is done.
//
// TODO(spec-5): emit structured slog (or equivalent) records with distinguishable
// event kinds and key fields (price, qty, side, depth).
func (c *LogConsumer) Run(ctx context.Context, in <-chan Event) error {
	_ = ctx
	_ = in
	_ = c.w
	return ErrNotImplemented
}
