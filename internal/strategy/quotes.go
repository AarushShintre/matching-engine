package strategy

// HalfSpread returns the half-spread S for the current config and activity count.
func HalfSpread(cfg Config, activity int) int {
	switch cfg.SpreadMode {
	case SpreadDynamic:
		return DynamicHalfSpread(cfg, activity)
	default:
		return cfg.FixedHalfSpread
	}
}

// DynamicHalfSpread computes S_dyn = clamp(base + activity*step, min, max).
func DynamicHalfSpread(cfg Config, activity int) int {
	raw := cfg.BaseHalfSpread + activity*cfg.ActivityStep
	if raw < cfg.MinHalfSpread {
		return cfg.MinHalfSpread
	}
	if raw > cfg.MaxHalfSpread {
		return cfg.MaxHalfSpread
	}
	return raw
}

// QuotePrices returns bid and ask around reference L with half-spread S.
func QuotePrices(ref, halfSpread int) (bid, ask int) {
	return ref - halfSpread, ref + halfSpread
}

// SideToBuySell maps QuoteSet side to ingress buy/sell.
func SideToBuySell(s Side) string {
	if s == SideAsk {
		return "sell"
	}
	return "buy"
}
