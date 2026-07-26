package strategy

import "github.com/AarushShintre/matching-engine/internal/ingest"

// ClientAdapter wraps ingest.Client as the narrower strategy Ingress
// (ignores Outcome; errors only). Use when Spec 2 Engine is live.
type ClientAdapter struct {
	Client ingest.Client
}

func (a ClientAdapter) SubmitNewLimit(o NewLimitOrder) error {
	_, err := a.Client.SubmitNewLimit(o)
	return err
}

func (a ClientAdapter) SubmitCancel(o CancelOrder) error {
	_, err := a.Client.SubmitCancel(o)
	return err
}
