package ingest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/AarushShintre/matching-engine/internal/book"
	"github.com/AarushShintre/matching-engine/internal/marketdata"
)

var (
	ErrAlreadyStarted = errors.New("ingest: engine already started")
	ErrNotRunning     = errors.New("ingest: engine is not running")
	ErrStopped        = errors.New("ingest: engine is stopped")
	ErrWrongSymbol    = errors.New("ingest: operation symbol does not match engine")
	ErrInvalidSide    = errors.New("ingest: side must be buy or sell")
	ErrInvalidOrderID = errors.New("ingest: order id exceeds book id range")
)

// Client is the Spec 2 client surface. Clients MUST NOT mutate the book directly.
type Client interface {
	SubmitNewLimit(o NewLimitOrder) (Outcome, error)
	SubmitMarket(o MarketOrder) (Outcome, error)
	SubmitCancel(o CancelOrder) (Outcome, error)
}

type response struct {
	outcome Outcome
	err     error
}

// op is one handoff from a client to the exclusive matcher owner.
type op struct {
	kind   opKind
	limit  NewLimitOrder
	market MarketOrder
	cancel CancelOrder
	reply  chan response
}

type opKind int

const (
	opLimit opKind = iota
	opMarket
	opCancel
)

type engineState uint32

const (
	stateNew engineState = iota
	stateRunning
	stateStopping
	stateStopped
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
	stopCh  chan struct{}
	done    chan struct{}

	state    atomic.Uint32
	stopOnce sync.Once
	doneOnce sync.Once
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
		stopCh:  make(chan struct{}),
		done:    make(chan struct{}),
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

// Start launches the one goroutine allowed to mutate this engine's book.
func (e *Engine) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if !e.state.CompareAndSwap(uint32(stateNew), uint32(stateRunning)) {
		if engineState(e.state.Load()) == stateStopped {
			return ErrStopped
		}
		return ErrAlreadyStarted
	}

	go e.run(ctx)
	return nil
}

func (e *Engine) run(ctx context.Context) {
	defer e.finish()

	for {
		select {
		case request := <-e.ingress:
			outcome, err := e.processOne(request)
			request.reply <- response{outcome: outcome, err: err}
		case <-e.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop rejects future submissions and waits for the owner goroutine to exit.
// An operation already received by the owner completes before Stop returns.
func (e *Engine) Stop() error {
	for {
		switch engineState(e.state.Load()) {
		case stateNew:
			if e.state.CompareAndSwap(uint32(stateNew), uint32(stateStopped)) {
				e.finish()
				return nil
			}
		case stateRunning:
			if e.state.CompareAndSwap(uint32(stateRunning), uint32(stateStopping)) {
				e.stopOnce.Do(func() { close(e.stopCh) })
				<-e.done
				return nil
			}
		case stateStopping:
			<-e.done
			return nil
		case stateStopped:
			return nil
		}
	}
}

func (e *Engine) finish() {
	e.state.Store(uint32(stateStopped))
	e.doneOnce.Do(func() { close(e.done) })
}

// processOne applies a single operation on the matcher owner goroutine.
func (e *Engine) processOne(o op) (Outcome, error) {
	switch o.kind {
	case opLimit:
		if err := e.validateOperation(o.limit.Symbol, o.limit.Side, o.limit.OrderID); err != nil {
			return Outcome{}, err
		}
		result, err := e.book.SubmitLimit(book.LimitOrder{
			ID:       int(o.limit.OrderID),
			Side:     toBookSide(o.limit.Side),
			Price:    o.limit.Price,
			Quantity: o.limit.Quantity,
		})
		return outcomeFromBook(result), err
	case opMarket:
		if err := e.validateOperation(o.market.Symbol, o.market.Side, o.market.OrderID); err != nil {
			return Outcome{}, err
		}
		result, err := e.book.SubmitMarket(book.MarketOrder{
			ID:       int(o.market.OrderID),
			Side:     toBookSide(o.market.Side),
			Quantity: o.market.Quantity,
		})
		return outcomeFromBook(result), err
	case opCancel:
		if err := e.validateSymbol(o.cancel.Symbol); err != nil {
			return Outcome{}, err
		}
		if err := validateOrderID(o.cancel.OrderID); err != nil {
			return Outcome{}, err
		}
		result, err := e.book.Cancel(int(o.cancel.OrderID))
		return outcomeFromBook(result), err
	default:
		return Outcome{}, fmt.Errorf("ingest: unknown operation kind %d", o.kind)
	}
}

type client struct {
	eng *Engine
}

// SubmitNewLimit hands a limit to the matcher owner.
func (c *client) SubmitNewLimit(o NewLimitOrder) (Outcome, error) {
	return c.submit(op{kind: opLimit, limit: o})
}

// SubmitMarket hands a market order to the matcher owner.
func (c *client) SubmitMarket(o MarketOrder) (Outcome, error) {
	return c.submit(op{kind: opMarket, market: o})
}

// SubmitCancel hands a cancel to the matcher owner.
func (c *client) SubmitCancel(o CancelOrder) (Outcome, error) {
	return c.submit(op{kind: opCancel, cancel: o})
}

func (c *client) submit(request op) (Outcome, error) {
	switch engineState(c.eng.state.Load()) {
	case stateNew:
		return Outcome{}, ErrNotRunning
	case stateStopping, stateStopped:
		return Outcome{}, ErrStopped
	}

	request.reply = make(chan response, 1)
	select {
	case c.eng.ingress <- request:
	case <-c.eng.done:
		return Outcome{}, ErrStopped
	}

	select {
	case reply := <-request.reply:
		return reply.outcome, reply.err
	case <-c.eng.done:
		// If the owner accepted the request, it puts the response in this
		// buffered channel before it can exit.
		select {
		case reply := <-request.reply:
			return reply.outcome, reply.err
		default:
			return Outcome{}, ErrStopped
		}
	}
}

func (e *Engine) validateOperation(symbol, side string, id OrderID) error {
	if err := e.validateSymbol(symbol); err != nil {
		return err
	}
	if side != "buy" && side != "sell" {
		return ErrInvalidSide
	}
	return validateOrderID(id)
}

func (e *Engine) validateSymbol(symbol string) error {
	if symbol != e.symbol {
		return ErrWrongSymbol
	}
	return nil
}

func validateOrderID(id OrderID) error {
	maxInt := uint64(^uint(0) >> 1)
	if uint64(id) > maxInt {
		return ErrInvalidOrderID
	}
	return nil
}

func toBookSide(side string) book.Side {
	if side == "buy" {
		return book.Buy
	}
	return book.Sell
}

func outcomeFromBook(result book.Result) Outcome {
	return Outcome{
		Accepted:  result.Accepted,
		Trades:    result.Trades,
		Remaining: result.Remaining,
	}
}
