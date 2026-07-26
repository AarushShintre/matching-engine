package ingest

import (
	"context"
	"errors"

	"github.com/AarushShintre/matching-engine/internal/book"
	"github.com/AarushShintre/matching-engine/internal/marketdata"
)

var ErrNotImplemented = errors.New("ingest: not implemented")

// Client is the Spec 2 client surface. Clients MUST NOT mutate the book directly.
type Client interface {
	SubmitNewLimit(o NewLimitOrder) (Outcome, error)
	SubmitMarket(o MarketOrder) (Outcome, error)
	SubmitCancel(o CancelOrder) (Outcome, error)
}

// op is one handoff from a client to the exclusive matcher owner.
type op struct {
	kind   opKind
	limit  NewLimitOrder
	market MarketOrder
	cancel CancelOrder
	result chan Outcome
	errc   chan error
}

type opKind int

const (
	opLimit opKind = iota
	opMarket
	opCancel
)

// Engine owns one symbol's book exclusively via a single matcher goroutine.
//
// Principle I: no mutex / RWLock / atomic-guarded fields on book state.
// All mutations MUST run on the matcher owner after channel handoff.
type Engine struct {
	symbol string
	book   *book.Book
	// TODO(spec-5): optional marketdata.Bus; emit trade + depth after each applied op
	bus *marketdata.Bus

	ingress chan op
	// TODO(spec-2): lifecycle signaling for Start/Stop/drain
}

// NewEngine builds an ingress engine for one symbol.
// book may be nil; New() is used in that case.
func NewEngine(symbol string, b *book.Book) *Engine {
	if b == nil {
		b = book.New()
	}
	return &Engine{
		symbol:  symbol,
		book:    b,
		ingress: make(chan op),
	}
}

// SetBus attaches a Spec 5 outbound bus (optional until you implement emission).
func (e *Engine) SetBus(bus *marketdata.Bus) {
	e.bus = bus
}

// Client returns the shared ingress path for concurrent submitters.
func (e *Engine) Client() Client {
	return &client{eng: e}
}

// Start runs the exclusive matching owner.
//
// TODO(spec-2): spawn one goroutine that drains e.ingress sequentially and
// calls only book endpoints (SubmitLimit / SubmitMarket / Cancel). Never
// mutate e.book from client goroutines.
func (e *Engine) Start(ctx context.Context) error {
	_ = ctx
	return ErrNotImplemented
}

// Stop stops accepting new work and finishes in-flight drain policy.
//
// TODO(spec-2): stop the matcher owner; define drain-vs-reject for queued ops
// so the book stays Spec 1–consistent for the applied prefix (FR-008).
func (e *Engine) Stop() error {
	return ErrNotImplemented
}

// processOne applies a single operation on the matcher owner goroutine.
//
// TODO(spec-2): map ingress ops → book.SubmitLimit / SubmitMarket / Cancel;
// return Outcome to the waiting client; optionally Publish market-data events.
func (e *Engine) processOne(o op) (Outcome, error) {
	_ = o
	_ = e.book
	_ = e.bus
	return Outcome{}, ErrNotImplemented
}

type client struct {
	eng *Engine
}

// SubmitNewLimit hands a limit to the matcher owner.
//
// TODO(spec-2): enqueue on e.ingress and wait for Outcome (or ctx/shutdown error).
func (c *client) SubmitNewLimit(o NewLimitOrder) (Outcome, error) {
	_ = o
	return Outcome{}, ErrNotImplemented
}

// SubmitMarket hands a market order to the matcher owner.
//
// TODO(spec-2): enqueue market op; matcher calls book.SubmitMarket.
func (c *client) SubmitMarket(o MarketOrder) (Outcome, error) {
	_ = o
	return Outcome{}, ErrNotImplemented
}

// SubmitCancel hands a cancel to the matcher owner.
//
// TODO(spec-2): enqueue cancel op; matcher calls book.Cancel.
func (c *client) SubmitCancel(o CancelOrder) (Outcome, error) {
	_ = o
	return Outcome{}, ErrNotImplemented
}
