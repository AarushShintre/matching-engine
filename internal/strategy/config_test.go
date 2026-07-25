package strategy

import "testing"

func TestConfigValidateRejectsInvalidHalfSpread(t *testing.T) {
	cfg := Config{
		Symbol: "DEMO", QuoteSize: 10, SpreadMode: SpreadFixed,
		FixedHalfSpread: 0, MovementThreshold: 1, Enabled: true,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for fixed_half_spread < 1")
	}
}

func TestConfigValidateRejectsInvertedDynamicBounds(t *testing.T) {
	cfg := Config{
		Symbol: "DEMO", QuoteSize: 10, SpreadMode: SpreadDynamic,
		BaseHalfSpread: 2, MinHalfSpread: 5, MaxHalfSpread: 3,
		ActivityWindow: 3, ActivityStep: 1, MovementThreshold: 1, Enabled: true,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for inverted dynamic bounds")
	}
}

func TestConfigValidateRejectsZeroQuoteSize(t *testing.T) {
	cfg := Config{
		Symbol: "DEMO", QuoteSize: 0, SpreadMode: SpreadFixed,
		FixedHalfSpread: 5, MovementThreshold: 1, Enabled: true,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for quote_size < 1")
	}
}

func TestConfigValidateOK(t *testing.T) {
	cfg := Config{
		Symbol: "DEMO", QuoteSize: 10, SpreadMode: SpreadFixed,
		FixedHalfSpread: 5, MovementThreshold: 1, Enabled: true,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}
