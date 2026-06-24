package characters

// Bloom-drug addiction helpers. The mechanic (communion high, crash, withdrawal,
// mutation acceleration) is driven by the consume hook + the NewRound_Bloom hook;
// these are the small state helpers on the Character.

// AddBloomAddiction changes the Bloom addiction level, flooring at 0.
func (c *Character) AddBloomAddiction(n int) {
	c.BloomAddiction += n
	if c.BloomAddiction < 0 {
		c.BloomAddiction = 0
	}
}

// BloomAddictionTier returns the descriptive addiction tier:
// 0 none, 1 hooked (1-3), 2 dependent (4-8), 3 enslaved (9+).
func (c *Character) BloomAddictionTier() int {
	switch {
	case c.BloomAddiction >= 9:
		return 3
	case c.BloomAddiction >= 4:
		return 2
	case c.BloomAddiction >= 1:
		return 1
	default:
		return 0
	}
}

// BloomAddictionTierName returns the descriptive word for the current tier.
func (c *Character) BloomAddictionTierName() string {
	switch c.BloomAddictionTier() {
	case 3:
		return "enslaved"
	case 2:
		return "dependent"
	case 1:
		return "hooked"
	default:
		return "clean"
	}
}
