package items

// AgingPhase represents a potion's lifecycle phase.
type AgingPhase int

const (
	PhaseFresh AgingPhase = iota
	PhaseFermented
	PhasePeak
	PhaseDeclining
	PhaseSpoiled
)

// AgingThresholds defines the round counts for each phase transition.
type AgingThresholds struct {
	FermentRounds int `yaml:"ferment_rounds,omitempty"`
	PeakRounds    int `yaml:"peak_rounds,omitempty"`
	DecayRounds   int `yaml:"decay_rounds,omitempty"`
	SpoilRounds   int `yaml:"spoil_rounds,omitempty"`
}

// HasAging returns true if aging thresholds are configured.
func (a AgingThresholds) HasAging() bool {
	return a.SpoilRounds > 0
}

// GetAgingPhase returns the current phase and potency multiplier
// for a potion based on elapsed rounds and effective aging speed.
// effectiveSpeed combines bottle multiplier and craft skill:
// effectiveSpeed = bottleMultiplier * (1.0 - craftSkill/200)
func GetAgingPhase(elapsedRounds uint64, thresholds AgingThresholds, effectiveSpeed float64) (AgingPhase, float64) {
	if !thresholds.HasAging() {
		return PhaseFresh, 1.0
	}

	if effectiveSpeed <= 0 {
		effectiveSpeed = 1.0
	}

	// Higher effectiveSpeed = faster aging = shorter phase durations
	ferment := float64(thresholds.FermentRounds) / effectiveSpeed
	peak := float64(thresholds.PeakRounds) / effectiveSpeed
	decay := float64(thresholds.DecayRounds) / effectiveSpeed
	spoil := float64(thresholds.SpoilRounds) / effectiveSpeed

	elapsed := float64(elapsedRounds)

	switch {
	case elapsed < ferment:
		return PhaseFresh, 1.0
	case elapsed < peak:
		return PhaseFermented, 1.15
	case elapsed < decay:
		return PhasePeak, 1.30
	case elapsed < spoil:
		// Linear decline from 1.30 to 0.5
		progress := (elapsed - decay) / (spoil - decay)
		return PhaseDeclining, 1.30 - (0.80 * progress)
	default:
		return PhaseSpoiled, 0.0
	}
}

// GetPhaseDescription returns descriptive text about a potion's age
// based on the viewer's alchemy skill level. No hard numbers shown.
func GetPhaseDescription(phase AgingPhase, alchemySkill int,
	elapsedRounds uint64, thresholds AgingThresholds,
	effectiveSpeed float64) string {

	if !thresholds.HasAging() {
		return ""
	}

	// Skill 0-5: no info
	if alchemySkill < 6 {
		return ""
	}

	// Compute how far into the current phase we are for urgency hints
	if effectiveSpeed <= 0 {
		effectiveSpeed = 1.0
	}
	decay := float64(thresholds.DecayRounds) / effectiveSpeed
	spoil := float64(thresholds.SpoilRounds) / effectiveSpeed
	peak := float64(thresholds.PeakRounds) / effectiveSpeed
	elapsed := float64(elapsedRounds)

	// Skill 6-15: general feel
	if alchemySkill <= 15 {
		switch phase {
		case PhaseFresh:
			return "It smells freshly brewed."
		case PhaseFermented:
			return "This has aged nicely."
		case PhasePeak:
			return "It has a rich, potent aroma."
		case PhaseDeclining:
			return "It's past its prime."
		case PhaseSpoiled:
			return "It smells off."
		}
		return ""
	}

	// Skill 16-29: phase + direction
	if alchemySkill < 30 {
		switch phase {
		case PhaseFresh:
			return "It's still maturing — not yet at its best."
		case PhaseFermented:
			return "It's developing well, nearing its peak."
		case PhasePeak:
			return "It's at peak potency."
		case PhaseDeclining:
			return "It's beginning to fade — use it soon."
		case PhaseSpoiled:
			return "It has turned — drinking this would be unwise."
		}
		return ""
	}

	// Skill 30+: exact phase + urgency
	switch phase {
	case PhaseFresh:
		return "Freshly brewed and still maturing. Give it time."
	case PhaseFermented:
		return "Fermenting nicely — it will reach peak soon."
	case PhasePeak:
		remaining := decay - elapsed
		total := decay - peak
		if total > 0 && remaining/total > 0.5 {
			return "At peak potency with plenty of time left."
		}
		return "At peak potency, but it won't last much longer."
	case PhaseDeclining:
		remaining := spoil - elapsed
		total := spoil - decay
		if total > 0 && remaining/total > 0.5 {
			return "Past its peak and declining — still usable for now."
		}
		return "Declining rapidly — use it very soon or distill it."
	case PhaseSpoiled:
		return "Completely spoiled. Only good for distilling remnants."
	}

	return ""
}

// CalcEffectiveAgingSpeed computes the combined aging speed from bottle
// multiplier and craft skill. Higher = faster aging.
func CalcEffectiveAgingSpeed(bottleMultiplier float64, craftSkill int) float64 {
	if bottleMultiplier <= 0 {
		bottleMultiplier = 1.0
	}
	// Higher craft skill slows aging: skill 30 = 15% slower
	skillFactor := 1.0 - float64(craftSkill)/200.0
	if skillFactor < 0.5 {
		skillFactor = 0.5 // Floor at 50% speed reduction
	}
	return bottleMultiplier * skillFactor
}
