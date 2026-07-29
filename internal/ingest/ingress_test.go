package ingest

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/AarushShintre/matching-engine/internal/book"
)

func TestClientRequiresRunningEngine(t *testing.T) {
	e := NewEngine("TEST", book.New())
	cl := e.Client()
	if cl == nil {
		t.Fatal("Client returned nil")
	}

	_, err := cl.SubmitNewLimit(NewLimitOrder{
		Symbol: "TEST", OrderID: 1, Side: "buy", Price: 100, Quantity: 1,
	})
	if !errors.Is(err, ErrNotRunning) {
		t.Fatalf("submit before Start: got %v, want ErrNotRunning", err)
	}
}

func TestStartStopLifecycle(t *testing.T) {
	e := NewEngine("TEST", nil)
	if err := e.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := e.Start(context.Background()); !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("second Start: got %v, want ErrAlreadyStarted", err)
	}
	if err := e.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := e.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}

	_, err := e.Client().SubmitCancel(CancelOrder{Symbol: "TEST", OrderID: 1})
	if !errors.Is(err, ErrStopped) {
		t.Fatalf("submit after Stop: got %v, want ErrStopped", err)
	}
}

func TestContextCancellationStopsEngine(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	e := NewEngine("TEST", nil)
	if err := e.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	cancel()

	select {
	case <-e.done:
	case <-time.After(time.Second):
		t.Fatal("engine did not stop after context cancellation")
	}
}

func TestOperationsPreserveBookSemantics(t *testing.T) {
	e := NewEngine("TEST", book.New())
	if err := e.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer e.Stop()
	cl := e.Client()

	rest, err := cl.SubmitNewLimit(NewLimitOrder{
		Symbol: "TEST", OrderID: 1, Side: "sell", Price: 100, Quantity: 5,
	})
	if err != nil || !rest.Accepted || rest.Remaining != 5 {
		t.Fatalf("resting limit: outcome=%#v err=%v", rest, err)
	}

	cross, err := cl.SubmitNewLimit(NewLimitOrder{
		Symbol: "TEST", OrderID: 2, Side: "buy", Price: 100, Quantity: 3,
	})
	want := book.Trade{MakerID: 1, TakerID: 2, Price: 100, Quantity: 3}
	if err != nil || len(cross.Trades) != 1 || cross.Trades[0] != want {
		t.Fatalf("crossing limit: outcome=%#v err=%v", cross, err)
	}

	canceled, err := cl.SubmitCancel(CancelOrder{Symbol: "TEST", OrderID: 1})
	if err != nil || !canceled.Accepted {
		t.Fatalf("cancel: outcome=%#v err=%v", canceled, err)
	}
}

func TestValidation(t *testing.T) {
	e := NewEngine("TEST", nil)
	if err := e.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer e.Stop()

	_, err := e.Client().SubmitNewLimit(NewLimitOrder{
		Symbol: "OTHER", OrderID: 1, Side: "buy", Price: 100, Quantity: 1,
	})
	if !errors.Is(err, ErrWrongSymbol) {
		t.Fatalf("wrong symbol: got %v", err)
	}

	_, err = e.Client().SubmitMarket(MarketOrder{
		Symbol: "TEST", OrderID: 2, Side: "invalid", Quantity: 1,
	})
	if !errors.Is(err, ErrInvalidSide) {
		t.Fatalf("invalid side: got %v", err)
	}
}

func TestConcurrentClientsAppliedExactlyOnce(t *testing.T) {
	e := NewEngine("TEST", book.New())
	if err := e.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer e.Stop()
	cl := e.Client()

	const (
		clients   = 4
		perClient = 50
	)
	errs := make(chan error, clients*perClient)
	var wg sync.WaitGroup
	for clientID := 0; clientID < clients; clientID++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			for sequence := 0; sequence < perClient; sequence++ {
				id := OrderID(clientID*perClient + sequence + 1)
				out, err := cl.SubmitNewLimit(NewLimitOrder{
					Symbol: "TEST", OrderID: id, Side: "buy", Price: 100, Quantity: 1,
				})
				if err != nil {
					errs <- err
				} else if !out.Accepted || out.Remaining != 1 {
					errs <- errors.New("operation was not applied exactly once")
				}
			}
		}(clientID)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	for id := 1; id <= clients*perClient; id++ {
		out, err := cl.SubmitCancel(CancelOrder{Symbol: "TEST", OrderID: OrderID(id)})
		if err != nil || !out.Accepted {
			t.Fatalf("cancel %d: outcome=%#v err=%v", id, out, err)
		}
	}
}
