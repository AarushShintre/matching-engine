package ingest

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/AarushShintre/matching-engine/internal/book"
)

func TestEngineClientShape(t *testing.T) {
	e := NewEngine("TEST", book.New())
	cl := e.Client()
	if cl == nil {
		t.Fatal("Client returned nil")
	}

	_, err := cl.SubmitNewLimit(NewLimitOrder{
		Symbol: "TEST", OrderID: 1, Side: "buy", Price: 100, Quantity: 1,
	})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("SubmitNewLimit: want ErrNotImplemented until TODO(spec-2), got %v", err)
	}
}

func TestStartStopStubs(t *testing.T) {
	e := NewEngine("TEST", nil)
	if err := e.Start(context.Background()); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Start: want ErrNotImplemented until TODO(spec-2), got %v", err)
	}
	if err := e.Stop(); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Stop: want ErrNotImplemented until TODO(spec-2), got %v", err)
	}
}

// Concurrent clients compile and hit the shared Client; they must not touch the book.
// Unlock by implementing TODO(spec-2): Start + channel drain.
func TestConcurrentClientsShell(t *testing.T) {
	e := NewEngine("TEST", book.New())
	cl := e.Client()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id OrderID) {
			defer wg.Done()
			_, err := cl.SubmitNewLimit(NewLimitOrder{
				Symbol: "TEST", OrderID: id, Side: "buy", Price: 100, Quantity: 1,
			})
			if err != nil && !errors.Is(err, ErrNotImplemented) {
				t.Errorf("client %d: unexpected error %v", id, err)
			}
		}(OrderID(i + 1))
	}
	wg.Wait()

	t.Log("TODO(spec-2): after Start works, assert each accepted op applied exactly once")
}
