package strategy

// FakeFeed is a minimal Spec 5 event source for decision/loop tests (T008a).
type FakeFeed struct {
	Events []Event
}

func (f *FakeFeed) PushTrade(t TradeEvent) {
	tt := t
	f.Events = append(f.Events, Event{Kind: EventTrade, Trade: &tt})
}

func (f *FakeFeed) PushGap() {
	f.Events = append(f.Events, Event{Kind: EventGap})
}

func Trade(symbol string, price, qty int) TradeEvent {
	return TradeEvent{Symbol: symbol, Price: price, Quantity: qty}
}

func OwnFillTrade(symbol string, price, qty int, resting OrderID) TradeEvent {
	id := resting
	return TradeEvent{
		Symbol: symbol, Price: price, Quantity: qty,
		RestingOrderID: &id,
	}
}
