package opinions

// pull moves score toward def by exactly N integer steps without
// overshooting. n=0 is identity. n<0 is treated as 0.
func pull(score, def, n int) int {
	if n <= 0 || score == def {
		return score
	}
	if score < def {
		next := score + n
		if next > def {
			return def
		}
		return next
	}
	// score > def
	next := score - n
	if next < def {
		return def
	}
	return next
}

// decayedScore returns the score after applying integer-step decay
// from anchor to now, using the given half-life in rounds. A
// half-life of 0 disables decay (returns score unchanged).
func decayedScore(score, def int, anchorRound, nowRound, halfLifeRounds uint64) int {
	if halfLifeRounds == 0 || nowRound <= anchorRound {
		return score
	}
	steps := int((nowRound - anchorRound) / halfLifeRounds)
	return pull(score, def, steps)
}
