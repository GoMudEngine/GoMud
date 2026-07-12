package mutations

// manifester.go — effect readers for the Manifester (summoner/broodmaster)
// cluster. The reserve-reducer (companion_reserve_reduction) and the soft-cap
// raise (companion-cap-raise flag) are read elsewhere (GetCompanionReserveRank,
// GetMaxCompanions) — already wired by the Wave-5 companion economy. This file
// adds the one new Manifester numeric effect.

// GetCompanionEmpowerment returns the net companion-empowerment magnitude
// (Symbiotic Bond + the Manifester companion bridges) — how much the owner's
// companions are strengthened. It is a DEDICATED effect, applied to companions
// as its own buff (see hooks.tickCompanionEmpowerment); it must never be built
// by copying the owner's transient buffs, which would double-count the
// rally/warcry buffs those shouts already fan out to companions.
func GetCompanionEmpowerment(owned map[string]int) float64 {
	return sumEffects(owned, "companion_empowerment", "")
}
