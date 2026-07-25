package strategy

import (
	"strings"
	"testing"
)

func TestDemoNonClaimsBanner(t *testing.T) {
	s := NonClaimBanner
	lower := strings.ToLower(s)
	needles := []string{"simulation", "not profitable", "production"}
	for _, n := range needles {
		if !strings.Contains(lower, n) {
			t.Fatalf("non-claim banner missing %q: %s", n, s)
		}
	}
	// Must not claim profitability / production trading affirmatively.
	forbidden := []string{"profitable trading", "production-ready", "guaranteed alpha"}
	for _, f := range forbidden {
		if strings.Contains(lower, f) {
			t.Fatalf("banner must not contain %q", f)
		}
	}
}
