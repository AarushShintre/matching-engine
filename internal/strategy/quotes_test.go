package strategy

import "testing"

func TestDynamicHalfSpread(t *testing.T) {
	cfg := Config{
		SpreadMode: SpreadDynamic,
		BaseHalfSpread: 2, MinHalfSpread: 2, MaxHalfSpread: 10,
		ActivityWindow: 5, ActivityStep: 1,
	}
	// activity 0 → 2
	if got := DynamicHalfSpread(cfg, 0); got != 2 {
		t.Fatalf("activity 0: want 2, got %d", got)
	}
	// activity 3 → 2+3=5
	if got := DynamicHalfSpread(cfg, 3); got != 5 {
		t.Fatalf("activity 3: want 5, got %d", got)
	}
	// activity 20 → clamp to max 10
	if got := DynamicHalfSpread(cfg, 20); got != 10 {
		t.Fatalf("activity 20: want 10, got %d", got)
	}
	cfg.MinHalfSpread = 4
	cfg.BaseHalfSpread = 1
	cfg.ActivityStep = 0
	if got := DynamicHalfSpread(cfg, 0); got != 4 {
		t.Fatalf("clamp to min: want 4, got %d", got)
	}
}
