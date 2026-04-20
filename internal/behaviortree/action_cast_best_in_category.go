package behaviortree

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// actCastBestInCategory is a smart-cast action that picks the best spell from
// the mob's spellbook matching the given category, ranked by base_folds × cost,
// and initiates it on the specified target.
//
// Params:
//
//	category (string, required): the category tag to filter by
//	target (string, optional): "self" for this phase; others reserved
//
// Returns Failure when: category is empty, mob is missing, mob is on the
// special-move cooldown, mob is already casting, or no eligible spell is found.
// Returns Success when it successfully initiates a cast via mob.Command.
func actCastBestInCategory(params map[string]any, ctx *EvalContext) Result {
	category := getStringParam(params, "category")
	if category == "" {
		return Failure
	}
	target := getStringParam(params, "target")
	if target == "" {
		target = "self"
	}

	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}

	// Shared special-move cooldown: cast, bash, kick, trip use the same slot.
	if mob.Character.GetCooldown("special-move") > 0 {
		return Failure
	}

	// Already casting — don't double-initiate.
	if mob.Character.CastingState != nil {
		return Failure
	}

	candidates := collectCategoryCandidates(
		&mob.Character,
		mob.Character.SpellBook,
		category,
		mob.MobId,
		mob.Character.Name,
	)
	if len(candidates) == 0 {
		return Failure
	}

	// Rank: (base_folds × cost) desc, spellid asc for ties.
	sort.SliceStable(candidates, func(i, j int) bool {
		si, sj := candidates[i], candidates[j]
		scoreI := si.BaseFolds * si.Cost
		scoreJ := sj.BaseFolds * sj.Cost
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		return si.SpellId < sj.SpellId
	})

	chosen := candidates[0]

	// target=self is the MVP shape. mob.Command("cast X") with no target
	// resolves HelpSingle-type spells to self via the existing cast pipeline.
	mob.Command("cast " + chosen.SpellId)
	return Success
}

// collectCategoryCandidates returns spells from the spellbook that match the
// category and pass all skip checks. Missing spellids are logged once at Warn
// and treated as skipped. The signature takes explicit Character + Spellbook
// (not the whole Mob) so the helper can be tested in isolation.
func collectCategoryCandidates(
	char *characters.Character,
	spellbook map[string]int,
	category string,
	mobId mobs.MobId,
	mobName string,
) []*spells.SpellData {
	if char == nil || spellbook == nil || category == "" {
		return nil
	}
	cpHave := char.Conviction

	out := make([]*spells.SpellData, 0, len(spellbook))
	for spellId := range spellbook {
		sd := spells.GetSpell(spellId)
		if sd == nil {
			mudlog.Warn("cast_best_in_category", "warning",
				fmt.Sprintf("mob %d (%s) spellbook references deleted spellid %q", mobId, mobName, spellId))
			continue
		}
		if !spellHasCategory(sd, category) {
			continue
		}
		// Component-gated spells: skip (mobs don't carry components).
		if sd.ComponentTag != "" || sd.SummonComponentId != 0 {
			continue
		}
		// Summon / raise / conjure / charm: skip (recursion guard).
		if sd.SummonMobId != 0 || sd.EffectType == "charm" {
			continue
		}
		// Insufficient conviction.
		if cpHave < sd.Cost {
			continue
		}
		// Effect already active on the character.
		if spellEffectAlreadyActive(char, sd) {
			continue
		}
		out = append(out, sd)
	}
	return out
}

func spellHasCategory(sd *spells.SpellData, category string) bool {
	for _, c := range sd.Categories {
		if c == category {
			return true
		}
	}
	return false
}

// spellEffectAlreadyActive returns true when the effect this spell would
// grant is already on the character. Branches:
//   - spell.BuffIds non-empty: skip if any is active (HasBuff)
//   - spell.EffectType == "shield": skip if ConditionShield is already on
//     the target. (Spell resolution lands shield-type casts via
//     AddCondition(ConditionShield, ...) — see spell_resolution.go:758.
//     NOT Character.HasShield(), which checks for equipped shield items
//     or species natural-bash, neither of which is what this spell grants.)
//
// If neither mechanism matches, returns false (conservative — may recast but
// won't silently stall the tree).
func spellEffectAlreadyActive(char *characters.Character, sd *spells.SpellData) bool {
	for _, bid := range sd.BuffIds {
		if char.HasBuff(bid) {
			return true
		}
	}
	if sd.EffectType == "shield" && char.HasCondition(characters.ConditionShield) {
		return true
	}
	return false
}
