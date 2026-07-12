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
		// Additive delta: engine applies stat * (1.0 + value), so >0 raises.
		return statPhrase(e.Target, e.Value > 0)
	case "health_multiplier":
		// Additive delta (hpMax * (1.0 + value)); >0 is a bonus, <0 a penalty.
		if e.Value > 0 {
			return "Toughens your body, deepening your reserves of health."
		}
		return "Thins your body's reserves of health."
	case "stamina_regen_multiplier":
		// Additive delta (regen * (1.0 + value)); >0 quickens regen.
		if e.Value > 0 {
			return "Quickens how fast your stamina returns."
		}
		return "Slows how fast your stamina returns."
	case "health_regen_multiplier":
		// Additive delta (regen * (1.0 + value)); >0 quickens healing.
		if e.Value > 0 {
			return "Quickens how fast your wounds close."
		}
		return "Slows how fast your wounds close."
	case "spell_power":
		// Additive delta (spell dmg * (1.0 + value)); >0 strengthens magic.
		if e.Value >= 0 {
			return "Sharpens your spellcraft, lending your magic greater force."
		}
		return "Dulls the force of your spellcraft."
	case "dodge_modifier":
		if e.Value >= 0 {
			return "Sharpens your reflexes, making blows harder to land on you."
		}
		return "Slows your reflexes, making you easier to strike."
	case "on_hit_buff":
		return "Your natural strikes leave a debilitating affliction in the wound."
	case "aura_ally_buff":
		return "Your presence steadies and emboldens allies who fight beside you."
	case "aura_enemy_debuff":
		return "Your presence rattles nearby foes — their aim and focus falter."
	case "conviction_cost_multiplier":
		// Additive delta (cost * (1.0 + value)); <0 is cheaper, >0 dearer.
		if e.Value < 0 {
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
	case "companion_reserve_reduction":
		return "Eases the conviction you must devote to sustaining your companions, letting you field more of them."
	case "forage_yield_multiplier":
		return "Your practiced eye turns up more when you forage the land."
	case "salvage_yield_bonus":
		return "You break things down more thoroughly, recovering more from your salvage."
	case "craft_material_discount":
		return "Your careful hands waste fewer materials when you craft."
	case "craft_quality_bonus":
		return "Your conviction in your craft lends your finished work greater quality."
	case "carry_capacity_multiplier":
		return "You can haul far more than your frame suggests."
	case "companion_empowerment":
		return "The bond you share with your companions makes them fight harder at your side."
	case "reflect_damage":
		return "A share of the harm done to you lashes back into whoever struck you."
	case "on_reflect_buff":
		return "Your backlash leaves a lingering affliction on whoever struck you."
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
	case "battle-frenzy":
		return "When badly wounded you fly into a battle frenzy — you hit harder, but cannot make yourself retreat."
	case "flying":
		return "You take to the air on wings — swift over ground, hard for the earthbound to touch, and free to break away at will."
	case "portable-workshop":
		return "Your body is itself a workshop — you can craft anywhere, needing no station or tools."
	case "homunculus":
		return "You can forge a living copy of yourself — a crafted companion that fights at your side."
	case "companion-cap-raise":
		return "You can keep more companions at your side at once."
	case "brood-mother":
		return "You endlessly birth and sustain a brood — and are never left without a companion."
	case "control-immune":
		return "Nothing can take you off your feet — you cannot be knocked down or grappled."
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
