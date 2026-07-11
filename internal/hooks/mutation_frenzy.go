package hooks

// bloodFrenzyBuffId is the Blood Frenzy state buff (see 100-blood_frenzy.yaml).
// The buff carries the damage-bonus + no-flee flags and lapses after 2 rounds
// unless the per-round trigger re-applies it.
const bloodFrenzyBuffId = 100

// shouldFrenzy reports whether a bearer of the battle-frenzy mutation flag
// should be in the frenzy state: in combat and below half of max health.
// The maxHealth guard avoids a divide-by-zero on a degenerate character.
func shouldFrenzy(hasFlag bool, health, maxHealth int) bool {
	if !hasFlag || maxHealth <= 0 {
		return false
	}
	return health*2 < maxHealth
}
