package strategy

// IsOwnFill reports whether the trade fills against a strategy-owned order (FR-013).
func IsOwnFill(t TradeEvent, owned map[OrderID]OwnedOrder) bool {
	if owned == nil {
		return false
	}
	if t.RestingOrderID != nil {
		if _, ok := owned[*t.RestingOrderID]; ok {
			return true
		}
	}
	if t.AggressorOrderID != nil {
		if _, ok := owned[*t.AggressorOrderID]; ok {
			return true
		}
	}
	return false
}

// OwnedIDs returns currently tracked owned order ids (bid and/or ask).
func OwnedIDs(q QuoteSet) []OrderID {
	var ids []OrderID
	if q.BidOrderID != nil {
		ids = append(ids, *q.BidOrderID)
	}
	if q.AskOrderID != nil {
		ids = append(ids, *q.AskOrderID)
	}
	return ids
}

// MarkFilled updates owned status when a trade hits a strategy order.
func MarkFilled(owned map[OrderID]OwnedOrder, t TradeEvent) {
	if owned == nil {
		return
	}
	mark := func(id *OrderID) {
		if id == nil {
			return
		}
		if o, ok := owned[*id]; ok {
			o.Status = OwnedFilled
			owned[*id] = o
		}
	}
	mark(t.RestingOrderID)
	mark(t.AggressorOrderID)
}
