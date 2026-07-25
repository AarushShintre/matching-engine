package strategy

import (
	"errors"
	"fmt"
)

// SpreadMode selects fixed or dynamic half-spread quoting.
type SpreadMode string

const (
	SpreadFixed   SpreadMode = "fixed"
	SpreadDynamic SpreadMode = "dynamic"
)

// Config is runtime parameters for one demo strategy instance (one symbol).
type Config struct {
	Symbol            string
	QuoteSize         int
	SpreadMode        SpreadMode
	FixedHalfSpread   int
	BaseHalfSpread    int
	MinHalfSpread     int
	MaxHalfSpread     int
	ActivityWindow    int
	ActivityStep      int
	MovementThreshold int
	SeedReference     *int // optional; used only until first trade
	Enabled           bool
	OrderIDNamespace  uint64
}

// Validate checks configuration rules. Invalid configs must never cause book writes.
func (c Config) Validate() error {
	if c.Symbol == "" {
		return errors.New("symbol must be non-empty")
	}
	if c.QuoteSize < 1 {
		return errors.New("quote_size must be >= 1")
	}
	if c.MovementThreshold < 1 {
		return errors.New("movement_threshold must be >= 1")
	}
	switch c.SpreadMode {
	case SpreadFixed:
		if c.FixedHalfSpread < 1 {
			return errors.New("fixed_half_spread must be >= 1")
		}
	case SpreadDynamic:
		if c.BaseHalfSpread < 1 {
			return errors.New("base_half_spread must be >= 1")
		}
		if c.MinHalfSpread < 1 {
			return errors.New("min_half_spread must be >= 1")
		}
		if c.MaxHalfSpread < c.MinHalfSpread {
			return errors.New("max_half_spread must be >= min_half_spread")
		}
		if c.ActivityWindow < 1 {
			return errors.New("activity_window must be >= 1")
		}
		if c.ActivityStep < 0 {
			return errors.New("activity_step must be >= 0")
		}
	default:
		return fmt.Errorf("unknown spread_mode %q", c.SpreadMode)
	}
	if c.SeedReference != nil && *c.SeedReference < 1 {
		return errors.New("seed_reference must be >= 1 when set")
	}
	return nil
}

// WithDefaults returns a copy with MovementThreshold defaulted to 1 when unset (0).
func (c Config) WithDefaults() Config {
	out := c
	if out.MovementThreshold == 0 {
		out.MovementThreshold = 1
	}
	return out
}
