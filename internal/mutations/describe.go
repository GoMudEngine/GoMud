package mutations

// describe.go — render a mutation's structured pro/con effects into short,
// number-free phrases for the `mutations` command. Per the project's
// no-hard-numbers rule we describe direction and kind, never magnitudes.

// DescribeEffect returns a short, player-facing phrase for a single mutation
// effect, or "" if the effect has no meaningful player summary (the caller
// skips empty results).
func DescribeEffect(e MutationEffect) string {
	switch e.Type {
	case "stat_flat":
		return statPhrase(e.Target, e.Value > 0)
	case "stat_multiplier":
		return statPhrase(e.Target, e.Value > 1.0)
	case "health_multiplier":
		if e.Value >= 1.0 {
			return "Toughens your body, deepening your reserves of health."
		}
		return "Thins your body's reserves of health."
	case "stamina_regen_multiplier":
		if e.Value >= 1.0 {
			return "Quickens how fast your stamina returns."
		}
		return "Slows how fast your stamina returns."
	case "conviction_cost_multiplier":
		if e.Value <= 1.0 {
			return "Lessens the conviction your abilities cost."
		}
		return "Raises the conviction your abilities cost."
	case "conviction_damage_reduction":
		return "Steels you against blows to your conviction."
	case "magical_damage_reduction":
		return "Wards you against magical harm."
	case "natural_armor":
		return "Hardens your hide against physical blows."
	case "natural_weapon":
		return "Arms you with a natural weapon."
	case "aggro_magnet":
		if e.Value >= 0 {
			return "Draws hostile attention toward you."
		}
		return "Turns hostile attention away from you."
	case "conditional_damage_low_hp":
		return "You strike harder the more badly wounded you are."
	case "flag":
		return flagPhrase(e.Target)
	}
	return ""
}

// statPhrase renders a stat change as a direction phrase. up=true means the
// stat is raised. Returns "" for non-stat targets.
func statPhrase(target string, up bool) string {
	name := statDisplayName(target)
	if name == "" {
		return ""
	}
	if up {
		return "Heightens your " + name + "."
	}
	return "Dulls your " + name + "."
}

// flagPhrase renders a flag effect by its target.
func flagPhrase(target string) string {
	switch target {
	case "lightsource":
		return "You shed light -- a beacon in the dark, easy to spot."
	case "nightvision":
		return "You see clearly in the dark."
	case "see-hidden":
		return "You notice hidden creatures and things others miss."
	case "active-ability":
		return "Grants a new ability you can use in play."
	case "disable-legs":
		return "You lose the use of your legs."
	case "tail":
		return "You grow a tail, gaining its equipment slot."
	case "movement":
		return "Changes how you move through the world."
	}
	return ""
}

// statDisplayName maps a stat target to its display name, or "" if not a stat.
func statDisplayName(target string) string {
	switch target {
	case "strength":
		return "Strength"
	case "dexterity":
		return "Dexterity"
	case "perception":
		return "Perception"
	case "vitality":
		return "Vitality"
	case "willpower":
		return "Willpower"
	case "charisma":
		return "Charisma"
	}
	return ""
}
