package strategy

// StrategyStatus is the minimum observability surface (FR-009).
type StrategyStatus struct {
	Phase           Phase
	Reference       *int
	ReferenceSource *RefSource
	QuoteSet        *QuoteSnapshot
	LastAction      *Action
	NonClaim        string
}

// QuoteSnapshot is a status view of the current quote set.
type QuoteSnapshot struct {
	BidPx int
	AskPx int
	BidID *OrderID
	AskID *OrderID
}

// FeedbackLoopEvidence records feed → decision → ingress → follow-on feed.
type FeedbackLoopEvidence struct {
	FeedEvent   Event
	Decision    Decision
	IngressOps  []any // NewLimitOrder or CancelOrder
	FollowOnFeed []any // TradeEvent or BookDepthEvent
}

// StatusFromState builds a StrategyStatus snapshot.
func StatusFromState(st State, last *Action) StrategyStatus {
	var ref *int
	var src *RefSource
	if st.Ref.Valid {
		v := st.Ref.Value
		ref = &v
		s := st.Ref.Source
		src = &s
	}
	var qs *QuoteSnapshot
	if st.Quotes.State != QuoteEmpty {
		qs = &QuoteSnapshot{
			BidPx: st.Quotes.BidPrice,
			AskPx: st.Quotes.AskPrice,
			BidID: st.Quotes.BidOrderID,
			AskID: st.Quotes.AskOrderID,
		}
	}
	return StrategyStatus{
		Phase:           st.Phase,
		Reference:       ref,
		ReferenceSource: src,
		QuoteSet:        qs,
		LastAction:      last,
		NonClaim:        NonClaimBanner,
	}
}
