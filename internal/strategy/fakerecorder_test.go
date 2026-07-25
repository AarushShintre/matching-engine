package strategy

// IngressRecorder is a Spec 2 fake for SC-003 accounting (contracts §6).
type IngressRecorder struct {
	Submitted []any
	RejectNew bool
	RejectCancel map[OrderID]bool
}

func (r *IngressRecorder) SubmitNewLimit(o NewLimitOrder) error {
	if r.RejectNew {
		return errRejected
	}
	r.Submitted = append(r.Submitted, o)
	return nil
}

func (r *IngressRecorder) SubmitCancel(o CancelOrder) error {
	if r.RejectCancel != nil && r.RejectCancel[o.OrderID] {
		r.Submitted = append(r.Submitted, o) // still record attempt
		return errRejected
	}
	r.Submitted = append(r.Submitted, o)
	return nil
}

func (r *IngressRecorder) CountNew() int {
	n := 0
	for _, s := range r.Submitted {
		if _, ok := s.(NewLimitOrder); ok {
			n++
		}
	}
	return n
}

func (r *IngressRecorder) CountCancel() int {
	n := 0
	for _, s := range r.Submitted {
		if _, ok := s.(CancelOrder); ok {
			n++
		}
	}
	return n
}

func (r *IngressRecorder) NewOrders() []NewLimitOrder {
	var out []NewLimitOrder
	for _, s := range r.Submitted {
		if o, ok := s.(NewLimitOrder); ok {
			out = append(out, o)
		}
	}
	return out
}

type rejectError struct{}

func (rejectError) Error() string { return "ingress rejected" }

var errRejected = rejectError{}
