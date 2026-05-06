package opinions

// TierOf bucket-maps a raw disposition score to its Tier.
func TierOf(score int) Tier {
	switch {
	case score <= -50:
		return TierHostile
	case score <= -15:
		return TierCold
	case score <= 14:
		return TierNeutral
	case score <= 49:
		return TierWarm
	default:
		return TierFriendly
	}
}
