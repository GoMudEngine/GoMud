package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/users"
)

// actGrantProgression forces exactly one real skill-progression event for the
// triggering player, emitting the standard SKILL ADVANCEMENT banner. It calls the
// genuine CheckSkillProgression with a large bonus multiplier: the chance clamps
// to 1.0 (progression.go), so the roll is guaranteed, and the real path does the
// IncreaseSkill + tier bookkeeping and queues the banner. The tutorial guards this
// to fire once via a room set_state flag.
//
// params: skill (string) — the skill tag to advance (e.g. "spellcasting").
func actGrantProgression(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	skill := getStringParam(params, "skill")
	if skill == "" {
		return Failure
	}
	user.Character.CheckSkillProgression(skill, user.UserId, 1000.0)
	return Success
}
