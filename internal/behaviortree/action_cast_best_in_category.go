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
//	target (string, optional): "self" (default), "most_wounded_packmate",
//	  or "tanking_packmate". Packmate targets resolve via
//	  mobs.FindPackmatesInRoom and address the cast via "#<instanceId>"
//	  syntax on the cast command.
//
// Returns Failure when: category is empty, mob is missing, mob is on the
// special-move cooldown, mob is already casting, no eligible spell is found,
// or a packmate target is specified but no packmate matches.
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

	// Resolve packmate target before spell selection so we can fail fast
	// if no packmate matches (avoids running the sort when the cast would
	// fail anyway).
	var targetInstId int
	switch target {
	case "self":
		// no target prefix on the cast command — resolves to self.
	case "most_wounded_packmate":
		pm := findMostWoundedPackmate(mob)
		if pm == nil {
			return Failure
		}
		targetInstId = pm.InstanceId
	case "tanking_packmate":
		pm := findTankingPackmate(mob)
		if pm == nil {
			return Failure
		}
		targetInstId = pm.InstanceId
	default:
		mudlog.Warn("cast_best_in_category", "warning",
			fmt.Sprintf("unknown target %q; treating as failure", target))
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

	// Packmate targets are addressed via `cast <spell> #<instanceId>`.
	// target=self issues the cast without a target suffix; HelpSingle-type
	// spells resolve to self via the existing cast pipeline.
	if targetInstId > 0 {
		mob.Command(fmt.Sprintf("cast %s #%d", chosen.SpellId, targetInstId))
	} else {
		mob.Command("cast " + chosen.SpellId)
	}
	return Success
}

// findMostWoundedPackmate returns the same-room packmate with the lowest
// HP ratio. Returns nil if no packmates match or all packmates have
// HealthMax.Value <= 0 (broken state).
func findMostWoundedPackmate(self *mobs.Mob) *mobs.Mob {
	var best *mobs.Mob
	bestRatio := 2.0 // impossible ratio so any candidate wins
	for _, pm := range mobs.FindPackmatesInRoom(self) {
		maxHp := pm.Character.HealthMax.Value
		if maxHp <= 0 {
			continue
		}
		ratio := float64(pm.Character.Health) / float64(maxHp)
		if ratio < bestRatio {
			bestRatio = ratio
			best = pm
		}
	}
	return best
}

// findTankingPackmate returns the first same-room packmate with an
// active Aggro (engaged in combat). Returns nil if no packmate is
// actively tanking.
func findTankingPackmate(self *mobs.Mob) *mobs.Mob {
	for _, pm := range mobs.FindPackmatesInRoom(self) {
		if pm.Character.Aggro != nil {
			return pm
		}
	}
	return nil
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
