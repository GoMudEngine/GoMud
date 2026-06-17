package actions

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// ReadyStatus classifies whether a command can fire now.
type ReadyStatus int

const (
	ActionReady    ReadyStatus = iota // fire it now
	ActionDeferred                    // transiently blocked — retry later, no player text
	ActionRejected                    // permanently invalid — surface the real error, then drop
)

// ReadinessResult is the verdict from ActionReadiness.
type ReadinessResult struct {
	Status ReadyStatus
	Reason string // short tag for logging/telemetry; never shown to the player
}

// specialMoveVerbs mirrors the switch in CommandIsReady (command_readiness.go).
// SYNC POINT: keep in lockstep with that switch; TestActionReadinessDrift enforces it.
var specialMoveVerbs = map[string]bool{
	"taunt": true, "rally": true, "warcry": true, "trip": true, "bash": true,
	"grapple": true, "kick": true, "rake": true, "maul": true, "throttle": true,
	"pounce": true, "gore": true, "hamstring": true, "drain": true,
}

func splitVerb(cmd string) (verb, rest string) {
	parts := strings.SplitN(strings.TrimSpace(cmd), " ", 2)
	verb = strings.ToLower(parts[0])
	if len(parts) > 1 {
		rest = strings.TrimSpace(parts[1])
	}
	return verb, rest
}

// ActionReadiness reports whether cmd can fire right now for actor, and whether
// any blocker is transient (Deferred) or permanent (Rejected). Commands it does
// not specifically gate return ActionReady and execute verbatim.
func ActionReadiness(actor Actor, cmd string) ReadinessResult {
	if actor == nil {
		return ReadinessResult{ActionRejected, "no actor"}
	}
	char := actor.GetCharacter()
	if char == nil {
		return ReadinessResult{ActionRejected, "no character"}
	}

	verb, rest := splitVerb(cmd)

	if verb == "cast" {
		return castReadiness(actor, rest)
	}

	if specialMoveVerbs[verb] {
		if CommandIsReady(actor, verb) {
			return ReadinessResult{ActionReady, ""}
		}
		// Transient: shared cooldown or an active Activity (cast/craft/salvage).
		if char.IsActing() || char.GetCooldown("special-move") > 0 {
			return ReadinessResult{ActionDeferred, "special-move busy"}
		}
		// Structural: missing body part / wrong species / no valid target.
		return ReadinessResult{ActionRejected, "special-move unavailable"}
	}

	return ReadinessResult{ActionReady, ""}
}

// castReadiness is a READ-ONLY probe mirroring the player cast gates in
// internal/usercommands/skill.cast.go. It MUST NOT call InitiateCast or
// TryCooldown — those have side effects (consuming cooldowns).
func castReadiness(actor Actor, rest string) ReadinessResult {
	char := actor.GetCharacter()

	// Gate 1: Spell lookup (structural — wrong name never becomes valid).
	spellName, _ := splitVerb(rest)
	spellInfo := spells.GetSpell(spellName)
	if spellInfo == nil {
		spellInfo = spells.FindSpellByName(spellName)
	}
	if spellInfo == nil {
		return ReadinessResult{ActionRejected, "unknown spell"}
	}

	// Gate 2: Skill check — school-specific (mirrors cast.go lines 40-45 + 82-93).
	// Manifestation spells require the manifestation skill; all others need
	// spellcasting. Either condition being absent is structural (train first).
	skillLevel := char.GetSkillLevel(skills.Spellcasting)
	manifestLevel := char.GetSkillLevel(skills.Manifestation)
	if spellInfo.HasSchool(spells.SchoolManifestation) {
		if manifestLevel == 0 {
			return ReadinessResult{ActionRejected, "no skill"}
		}
	} else {
		if skillLevel == 0 {
			return ReadinessResult{ActionRejected, "no skill"}
		}
	}

	// Gate 3: Spell known (structural — must learn the spell first).
	if !characterKnowsSpell(char, spellInfo) {
		return ReadinessResult{ActionRejected, "spell not known"}
	}

	// Gate 4: Already casting (transient — wait for current cast to complete).
	if char.IsCasting() {
		return ReadinessResult{ActionDeferred, "already casting"}
	}

	// Gate 5: Busy with another activity — crafting, salvaging, etc. (transient).
	if !char.IsFree() {
		return ReadinessResult{ActionDeferred, "busy"}
	}

	// Gate 6: Conviction cost (transient — CP regenerates over time).
	convMult := 1.0 + mutations.GetConvictionCostMultiplier(char.Mutations)
	cost := spellInfo.GetTotalConvictionCost(convMult)
	if cost > 0 && char.Conviction < cost {
		return ReadinessResult{ActionDeferred, "insufficient conviction"}
	}

	// Gate 7: Cast-init cooldown (transient — set when a prior initiation failed).
	if char.GetCooldown("cast-init") > 0 {
		return ReadinessResult{ActionDeferred, "cast-init cooldown"}
	}

	// Gate 8: Shared special-move cooldown (transient).
	if char.GetCooldown("special-move") > 0 {
		return ReadinessResult{ActionDeferred, "special-move cooldown"}
	}

	// Gate 9: Required components (structural — must acquire the item first).
	if !castComponentsPresent(char, spellInfo) {
		return ReadinessResult{ActionRejected, "missing component"}
	}

	return ReadinessResult{ActionReady, ""}
}

// characterKnowsSpell reports whether char has the spell in their SpellBook
// with a positive cast count (negative = disabled).
func characterKnowsSpell(char *characters.Character, spellInfo *spells.SpellData) bool {
	return char.HasSpell(spellInfo.SpellId)
}

// castComponentsPresent mirrors the two component checks in skill.cast.go
// (lines 131-165): generic component_tag scan and summon_component_id scan.
// Both must pass if set on the spell.
func castComponentsPresent(char *characters.Character, spellInfo *spells.SpellData) bool {
	if spellInfo.ComponentTag != "" {
		found := false
		for _, itm := range char.Items {
			if itm.GetSpec().ComponentTag == spellInfo.ComponentTag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if spellInfo.SummonComponentId > 0 {
		found := false
		for _, itm := range char.Items {
			if itm.ItemId == spellInfo.SummonComponentId {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
