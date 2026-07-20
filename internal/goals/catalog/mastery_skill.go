// Package-level note on TODO-ADAPT stubs below:
//
// mobSkillRank delegates to Character.GetSkillLevel — the canonical
// accessor that incorporates equipment/buff StatMod bonuses.
//
// skillTrainingProximity always returns 1 ("in zone, not current room")
// as a deliberate 4.3 heuristic shortcut. The fuller per-skill training-
// context table (combat skills train via fighting, crafting skills train
// at stations, etc.) is deferred to chunk 4.4 / future planner work.
// When 4.4 lands, replace this stub with a map[string]func lookup keyed
// on skill_name.

package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	skillsPkg "github.com/GoMudEngine/GoMud/internal/skills"
)

func init() {
	goals.RegisterGoalType("mastery-skill", goals.GoalTypeMeta{
		Predicate:     masterySkillPredicate,
		ContextScore:  masterySkillContextScore,
		AllowMultiple: true,
		DedupKey:      masterySkillDedupKey,
		Params: []goals.ParamSchema{
			{Key: "skill_name", Required: true, GoType: "string"},
			{Key: "target_rank", Required: true, GoType: "int"},
		},
	})
}

func masterySkillDedupKey(g *goals.Goal) string {
	if name, ok := g.Params["skill_name"].(string); ok {
		return name
	}
	return ""
}

// masterySkillPredicate: satisfied when current rank >= target.
func masterySkillPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	name, _ := g.Params["skill_name"].(string)
	target := paramIntOr(g, "target_rank", 0)
	if name == "" || target == 0 {
		return false
	}
	return mobSkillRank(mob, name) >= target
}

// masterySkillContextScore tiers:
//   - 0 if already at/above target (predicate fires next tick)
//   - 0.2 baseline if no training opportunity in zone (elsewhere)
//   - 1.0 if opportunity in zone
//   - 2.0 if opportunity in current room
//   - Multiplied by (1 - rank/target)*0.5 + 0.5 so closer-to-target is
//     less urgent
func masterySkillContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	name, _ := g.Params["skill_name"].(string)
	target := paramIntOr(g, "target_rank", 0)
	if name == "" || target == 0 {
		return 0
	}
	current := mobSkillRank(mob, name)
	if current >= target {
		return 0
	}
	var base float64
	switch skillTrainingProximity(mob, name) {
	case 0: // in current room
		base = 2.0
	case 1: // in zone
		base = 1.0
	default:
		base = 0.2
	}
	urgency := (1.0-float64(current)/float64(target))*0.5 + 0.5
	return base * urgency
}

// ─── TODO-ADAPT helpers ──────────────────────────────────────────────────────

// mobSkillRank returns the mob's current rank in the named skill,
// including any equipment/buff StatMod bonuses, via the canonical
// Character.GetSkillLevel accessor. Verified against
// internal/characters/skills.go:166.
func mobSkillRank(mob *mobs.Mob, skillName string) int {
	return mob.Character.GetSkillLevel(skillsPkg.SkillTag(skillName))
}

// skillTrainingProximity always returns 1 ("in zone, not current room").
// This is a deliberate 4.3 heuristic shortcut — the per-skill training-
// context table is deferred to chunk 4.4. See file-level note above.
func skillTrainingProximity(_ *mobs.Mob, _ string) int {
	return 1
}
