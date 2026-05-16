package items

// DefaultReachForSubtype returns the canonical reach (meters) for
// items of a given weapon subtype. Authors who leave the per-item
// Reach field at zero get this value via ResolveReach.
//
// Subtypes not in this map return 0 — typically non-weapon subtypes
// (BlobContent, etc.) where reach is meaningless.
//
// See docs/superpowers/specs/2026-05-16-state-chunk-4c-position-weapon-utility-design.md
// for the full taxonomy table and reasoning.
func DefaultReachForSubtype(s ItemSubType) float64 {
	switch s {
	// Natural attacks
	case Fist:
		return 0.1
	case Claws:
		return 0.15
	case Bite:
		return 0.15
	case Sting:
		return 0.2
	case Slam:
		return 0.3
	case Gore:
		return 0.4
	case Whipping:
		return 0.5 // hand-held whip; mob tail-weapon would override

	// Melee weapon subtypes
	case Stabbing:
		return 0.3 // dagger / shiv family
	case Slashing:
		return 1.0 // sword family
	case Cleaving:
		return 0.9 // axe family
	case Bludgeoning:
		return 0.8 // mace / hammer family

	// Ranged (melee-fallback reach when used as a club)
	case Shooting:
		return 1.0 // bow/crossbow average; per-item overrides for compacts

	// Caster
	case Wand:
		return 0.4
	case Sceptre:
		return 0.6
	case Staff:
		return 1.5

	default:
		return 0 // non-weapon subtype or unknown
	}
}

// ResolveReach returns the effective reach for a weapon: explicit
// per-item Reach if set, otherwise the subtype default. Zero means
// "no reach data" — treated as no penalty by the combat path
// (combat.ReachUtility handles zero gracefully).
func ResolveReach(spec *ItemSpec) float64 {
	if spec == nil {
		return 0
	}
	if spec.Reach > 0 {
		return spec.Reach
	}
	return DefaultReachForSubtype(spec.Subtype)
}

// ResolveNaturalReach returns reach for a mob's natural attack
// (claws/bite/etc.) where there's no ItemSpec to inspect. Calls
// straight through to DefaultReachForSubtype; provided as a sibling
// helper to make caller intent explicit.
func ResolveNaturalReach(subtype ItemSubType) float64 {
	return DefaultReachForSubtype(subtype)
}
