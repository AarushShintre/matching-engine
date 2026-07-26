package strategy

import (
	"context"
	"log/slog"

	"github.com/AarushShintre/matching-engine/internal/marketdata"
)

// Runner consumes feed events and submits via Spec 2 ingress only.
type Runner struct {
	Cfg     Config
	Ingress Ingress
	IDs     *IDGen
	Log     *slog.Logger

	State    State
	Evidence []FeedbackLoopEvidence
	lastAct  *Action
}

// NewRunner constructs a strategy runner. Config must Validate() successfully.
func NewRunner(cfg Config, ingress Ingress) (*Runner, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Runner{
		Cfg:     cfg,
		Ingress: ingress,
		IDs:     NewIDGen(cfg.OrderIDNamespace),
		Log:     slog.Default(),
		State:   NewState(),
	}, nil
}

// Status returns the current observability snapshot.
func (r *Runner) Status() StrategyStatus {
	return StatusFromState(r.State, r.lastAct)
}

// StartSeed places initial quotes from SeedReference if configured and Enabled.
func (r *Runner) StartSeed() error {
	if !r.Cfg.Enabled {
		return nil
	}
	bid, ask := r.IDs.Next(), r.IDs.Next()
	d := SeedTick(r.Cfg, r.State, bid, ask)
	r.lastAct = &d.Action
	if d.Action != ActionQuoteInitial {
		return nil
	}
	ops, err := r.applyIngress(d)
	if err != nil {
		// Reject: wait for next cycle with new IDs (FR edge case).
		r.Log.Warn("ingress reject on seed quote; will retry next cycle", "err", err)
		return nil
	}
	ApplySeedDecision(&r.State, r.Cfg, d)
	r.record(Event{}, d, ops, nil)
	return nil
}

// HandleEvent processes one feed/shutdown event.
func (r *Runner) HandleEvent(ev Event) error {
	if r.State.Phase == PhaseStopped {
		return nil
	}

	bid, ask := r.IDs.Next(), r.IDs.Next()
	d := Decide(r.Cfg, r.State, ev, bid, ask)
	r.lastAct = &d.Action

	if r.Log != nil {
		r.Log.Info("strategy decision",
			"action", d.Action,
			"phase", r.State.Phase,
			"non_claim", NonClaimBanner,
		)
	}

	ops, err := r.applyIngress(d)
	if err != nil {
		r.Log.Warn("ingress reject; retry next quote cycle with new ids", "err", err)
		// Still advance reference/own-fill bookkeeping without installing new quotes.
		if d.Action == ActionQuoteInitial || d.Action == ActionRequote {
			hold := Decision{Action: ActionHold}
			ApplyDecision(&r.State, r.Cfg, hold, ev)
			r.record(ev, d, nil, nil)
			return nil
		}
	} else {
		ApplyDecision(&r.State, r.Cfg, d, ev)
	}

	var follow []any
	// Synthesize follow-on book-depth evidence from successful quote submits (demo harness).
	if err == nil && (d.Action == ActionQuoteInitial || d.Action == ActionRequote) {
		for _, o := range d.NewOrders {
			side := marketdata.SideBid
			if o.Side == "sell" {
				side = marketdata.SideAsk
			}
			follow = append(follow, BookDepthEvent{
				Symbol: o.Symbol, Side: side, Price: o.Price, Quantity: o.Quantity,
			})
		}
	}
	r.record(ev, d, ops, follow)
	return nil
}

// Run reads events until ctx is cancelled, then shutdown-cancels owned orders.
func (r *Runner) Run(ctx context.Context, events <-chan Event) error {
	if err := r.StartSeed(); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return r.Shutdown()
		case ev, ok := <-events:
			if !ok {
				return r.Shutdown()
			}
			if err := r.HandleEvent(ev); err != nil {
				return err
			}
		}
	}
}

// Shutdown best-effort cancels owned resting orders and marks stopped.
func (r *Runner) Shutdown() error {
	d := Decide(r.Cfg, r.State, Event{Kind: EventShutdown}, 0, 0)
	r.lastAct = &d.Action
	ops, _ := r.applyIngress(d) // best effort
	ApplyDecision(&r.State, r.Cfg, d, Event{Kind: EventShutdown})
	r.record(Event{Kind: EventShutdown}, d, ops, nil)
	return nil
}

// SignalGap marks feed gap / reconnect pause (no invented prices).
func (r *Runner) SignalGap() error {
	return r.HandleEvent(Event{Kind: EventGap})
}

func (r *Runner) applyIngress(d Decision) ([]any, error) {
	var ops []any
	for _, id := range d.CancelIDs {
		c := CancelOrder{Symbol: r.Cfg.Symbol, OrderID: id}
		if err := r.Ingress.SubmitCancel(c); err != nil {
			// Tolerate cancel-after-fill.
			r.Log.Info("cancel unsuccessful (tolerated)", "order_id", id, "err", err)
		}
		ops = append(ops, c)
	}
	for _, o := range d.NewOrders {
		if err := r.Ingress.SubmitNewLimit(o); err != nil {
			return ops, err
		}
		ops = append(ops, o)
	}
	return ops, nil
}

func (r *Runner) record(ev Event, d Decision, ops []any, follow []any) {
	r.Evidence = append(r.Evidence, FeedbackLoopEvidence{
		FeedEvent:    ev,
		Decision:     d,
		IngressOps:   ops,
		FollowOnFeed: follow,
	})
}
