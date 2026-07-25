package strategy

// IDGen allocates unique strategy order IDs in a reserved namespace.
// Layout: namespace in high 16 bits, monotonic sequence in low 48 bits.
type IDGen struct {
	namespace uint64
	seq       uint64
}

// NewIDGen creates a generator for the given namespace.
func NewIDGen(namespace uint64) *IDGen {
	return &IDGen{namespace: namespace & 0xffff}
}

// Next returns the next OrderID.
func (g *IDGen) Next() OrderID {
	g.seq++
	id := (g.namespace << 48) | (g.seq & 0xffffffffffff)
	return OrderID(id)
}
