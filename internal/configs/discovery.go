package configs

// DiscoveryParams bundles the inputs for DiscoveryChance.
type DiscoveryParams struct {
	Base       float64 // base chance as a percent (e.g. 5.0 = 5%)
	Decay      float64 // per-known decay rate
	Known      int     // count of already-known spells/recipes
	Perception int     // character Perception stat (adjusted value)
	Skill      int     // relevant skill rank
}

// DiscoveryChance computes the discovery roll chance as a percent
// with a Perception + skill offset reducing the effective decay.
//
// Formula:
//
//	perContrib   = clamp(0, 1, (Perception - 100) / PerceptionScale)
//	skillContrib = clamp(0, 1, Skill / SkillScale)
//	offset       = min(MaxOffset, 1 - (1 - perContrib)(1 - skillContrib))
//	effDecay     = Decay × (1 - offset)
//	chance       = Base / (1 + Known × effDecay)
//
// Returns a value in [0, Base]. Negative Known is clamped to 0.
func DiscoveryChance(p DiscoveryParams) float64 {
	bal := GetBalanceConfig()
	perScale := float64(bal.DiscoveryPerceptionScale)
	skillScale := float64(bal.DiscoverySkillScale)
	maxOffset := float64(bal.DiscoveryMaxDecayOffset)

	perContrib := (float64(p.Perception) - 100.0) / perScale
	if perContrib < 0 {
		perContrib = 0
	}
	if perContrib > 1 {
		perContrib = 1
	}

	skillContrib := float64(p.Skill) / skillScale
	if skillContrib < 0 {
		skillContrib = 0
	}
	if skillContrib > 1 {
		skillContrib = 1
	}

	offset := 1.0 - (1.0-perContrib)*(1.0-skillContrib)
	if offset > maxOffset {
		offset = maxOffset
	}

	effDecay := p.Decay * (1.0 - offset)

	known := p.Known
	if known < 0 {
		known = 0
	}
	return p.Base / (1.0 + float64(known)*effDecay)
}
