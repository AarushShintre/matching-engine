package strategy

import (
	"context"
	"testing"
	"time"
)

func TestFeedbackLoop(t *testing.T) {
	cfg := fixedCfg(nil)
	rec := &IngressRecorder{}
	r, err := NewRunner(cfg, rec)
	if err != nil {
		t.Fatal(err)
	}
	feed := &FakeFeed{}
	feed.PushTrade(Trade("DEMO", 100, 1))

	for _, ev := range feed.Events {
		if err := r.HandleEvent(ev); err != nil {
			t.Fatal(err)
		}
	}
	if rec.CountNew() != 2 {
		t.Fatalf("want strategy quotes via ingress, got %d", rec.CountNew())
	}
	if len(r.Evidence) == 0 {
		t.Fatal("want feedback loop evidence recorded")
	}
	ev := r.Evidence[len(r.Evidence)-1]
	if ev.Decision.Action != ActionQuoteInitial {
		t.Fatalf("want quote_initial, got %s", ev.Decision.Action)
	}
	if len(ev.IngressOps) == 0 {
		t.Fatal("want ingress ops in evidence")
	}
	if len(ev.FollowOnFeed) == 0 {
		t.Fatal("want follow-on feed-visible book-depth evidence from harness")
	}
	// No book mutation APIs exist on Runner — only Ingress.
	if _, ok := r.Ingress.(*IngressRecorder); !ok {
		t.Fatal("ingress must be recorder (no book API)")
	}
}

func TestShutdownCancelsOwnedOrders(t *testing.T) {
	cfg := fixedCfg(nil)
	rec := &IngressRecorder{}
	r, err := NewRunner(cfg, rec)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 100, 1))})
	bid, ask := *r.State.Quotes.BidOrderID, *r.State.Quotes.AskOrderID
	if err := r.Shutdown(); err != nil {
		t.Fatal(err)
	}
	if r.State.Phase != PhaseStopped {
		t.Fatalf("want stopped, got %s", r.State.Phase)
	}
	canceled := map[OrderID]bool{}
	for _, s := range rec.Submitted {
		if c, ok := s.(CancelOrder); ok {
			canceled[c.OrderID] = true
		}
	}
	if !canceled[bid] || !canceled[ask] {
		t.Fatalf("shutdown must cancel owned ids; canceled=%v bid=%d ask=%d", canceled, bid, ask)
	}
}

func TestRunContextShutdown(t *testing.T) {
	cfg := fixedCfg(nil)
	rec := &IngressRecorder{}
	r, err := NewRunner(cfg, rec)
	if err != nil {
		t.Fatal(err)
	}
	ch := make(chan Event, 1)
	ch <- Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 100, 1))}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx, ch) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for Run exit")
	}
	if r.State.Phase != PhaseStopped {
		t.Fatalf("want stopped after ctx cancel, got %s", r.State.Phase)
	}
}

func TestGapPausesQuoting(t *testing.T) {
	cfg := fixedCfg(nil)
	rec := &IngressRecorder{}
	r, err := NewRunner(cfg, rec)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 100, 1))})
	_ = r.SignalGap()
	if r.State.Phase != PhasePaused {
		t.Fatalf("want paused, got %s", r.State.Phase)
	}
	news := rec.CountNew()
	// While invalid, Decide on trade should resume — trade restores reference.
	_ = r.HandleEvent(Event{Kind: EventTrade, Trade: ptrTrade(Trade("DEMO", 100, 1))})
	if r.State.Ref.Valid != true {
		t.Fatal("trade after gap should restore valid reference")
	}
	_ = news
}
