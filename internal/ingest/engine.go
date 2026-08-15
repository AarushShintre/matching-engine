package ingest

// core idea within Spec 2: ingress is a single writer with multiple clients, completely avoids race conditions
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
	bus    *marketdata.Bus

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

// SetBus attaches the optional Spec 5 outbound bus. Call it before Start.
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
		before := e.snapshotForEvents()
		result, err := e.book.SubmitLimit(book.LimitOrder{
			ID:       int(o.limit.OrderID),
			Side:     toBookSide(o.limit.Side),
			Price:    o.limit.Price,
			Quantity: o.limit.Quantity,
		})
		if e.bus != nil && err == nil && result.Accepted {
			e.emitMatchEvents(before, result.Trades, toBookSide(o.limit.Side))
			if result.Remaining > 0 {
				e.emitDepth(toBookSide(o.limit.Side), o.limit.Price, e.levelDepth(
					e.book.Snapshot(), toBookSide(o.limit.Side), o.limit.Price,
				))
			}
		}
		return outcomeFromBook(result), err
	case opMarket:
		if err := e.validateOperation(o.market.Symbol, o.market.Side, o.market.OrderID); err != nil {
			return Outcome{}, err
		}
		before := e.snapshotForEvents()
		result, err := e.book.SubmitMarket(book.MarketOrder{
			ID:       int(o.market.OrderID),
			Side:     toBookSide(o.market.Side),
			Quantity: o.market.Quantity,
		})
		if e.bus != nil && err == nil && result.Accepted {
			e.emitMatchEvents(before, result.Trades, toBookSide(o.market.Side))
		}
		return outcomeFromBook(result), err
	case opCancel:
		if err := e.validateSymbol(o.cancel.Symbol); err != nil {
			return Outcome{}, err
		}
		if err := validateOrderID(o.cancel.OrderID); err != nil {
			return Outcome{}, err
		}
		before := e.snapshotForEvents()
		side, price, found := findOrderLevel(before, int(o.cancel.OrderID))
		result, err := e.book.Cancel(int(o.cancel.OrderID))
		if e.bus != nil && err == nil && result.Accepted && found {
			e.emitDepth(side, price, e.levelDepth(e.book.Snapshot(), side, price))
		}
		return outcomeFromBook(result), err
	default:
		return Outcome{}, fmt.Errorf("ingest: unknown operation kind %d", o.kind)
	}
}

func (e *Engine) snapshotForEvents() book.BookSnapshot {
	if e.bus == nil {
		return book.BookSnapshot{}
	}
	return e.book.Snapshot()
}

func (e *Engine) emitMatchEvents(
	before book.BookSnapshot,
	trades []book.Trade,
	aggressorSide book.Side,
) {
	if e.bus == nil {
		return
	}
	restingSide := book.Buy
	if aggressorSide == book.Buy {
		restingSide = book.Sell
	}
	removedByPrice := make(map[int]int)
	for _, trade := range trades {
		makerID := uint64(trade.MakerID)
		takerID := uint64(trade.TakerID)
		_ = e.bus.Publish(marketdata.Event{
			Kind: marketdata.KindTrade,
			Trade: &marketdata.TradeEvent{
				Symbol:           e.symbol,
				Price:            trade.Price,
				Quantity:         trade.Quantity,
				RestingOrderID:   &makerID,
				AggressorOrderID: &takerID,
			},
		})

		removedByPrice[trade.Price] += trade.Quantity
		depth := e.levelDepth(before, restingSide, trade.Price) - removedByPrice[trade.Price]
		e.emitDepth(restingSide, trade.Price, depth)
	}
}

func (e *Engine) emitDepth(side book.Side, price, quantity int) {
	if e.bus == nil {
		return
	}
	_ = e.bus.Publish(marketdata.Event{
		Kind: marketdata.KindBookDepth,
		BookDepth: &marketdata.BookDepthEvent{
			Symbol:   e.symbol,
			Side:     toMarketDataSide(side),
			Price:    price,
			Quantity: quantity,
		},
	})
}

func (e *Engine) levelDepth(snapshot book.BookSnapshot, side book.Side, price int) int {
	levels := snapshot.Bids
	if side == book.Sell {
		levels = snapshot.Asks
	}
	for _, level := range levels {
		if level.Price != price {
			continue
		}
		quantity := 0
		for _, order := range level.Orders {
			quantity += order.Quantity
		}
		return quantity
	}
	return 0
}

func findOrderLevel(snapshot book.BookSnapshot, id int) (book.Side, int, bool) {
	for _, level := range snapshot.Bids {
		for _, order := range level.Orders {
			if order.ID == id {
				return book.Buy, level.Price, true
			}
		}
	}
	for _, level := range snapshot.Asks {
		for _, order := range level.Orders {
			if order.ID == id {
				return book.Sell, level.Price, true
			}
		}
	}
	return book.Buy, 0, false
}

func toMarketDataSide(side book.Side) marketdata.Side {
	if side == book.Buy {
		return marketdata.SideBid
	}
	return marketdata.SideAsk
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

// Edit #2: Optimized reply channel allocation
// reply channel now comes from a sync.Pool instead of make() per call.
// Under steady, sustained load this cuts new-channel allocations close to
// zero, since Get() usually returns a previously-Put channel. It does NOT
// guarantee zero allocation — sync.Pool can be cleared by the GC (with a
// one-cycle "victim cache" grace period) and is sharded per-P, so bursty
// load or scheduling churn can still cause fallback allocations via New().
var replyPool = sync.Pool{
	New: func() any { return make(chan response, 1) },
}

func (c *client) submit(request op) (Outcome, error) {
	switch engineState(c.eng.state.Load()) {
	case stateNew:
		return Outcome{}, ErrNotRunning
	case stateStopping, stateStopped:
		return Outcome{}, ErrStopped
	}

	reply := replyPool.Get().(chan response)
	request.reply = reply
	defer replyPool.Put(reply)

	select {
	case c.eng.ingress <- request:
	case <-c.eng.done:
		return Outcome{}, ErrStopped
	}

	select {
	case r := <-reply:
		return r.outcome, r.err
	case <-c.eng.done:
		select {
		case r := <-reply:
			return r.outcome, r.err
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
