package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// condMobCanSubmitTop reports true when the mob is the controller side
// of a sub-eligible grapple (position has top-attack subs available AND
// ControlLevel is InControl/LosingControl). Used by AI archetypes to
// branch on whether a submission is available, or to avoid stepping on
// the auto-fired submission tick.
func condMobCanSubmitTop(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.Character.Position == nil {
		return Failure
	}
	if !mob.Character.IsController() {
		return Failure
	}
	gd, ok := mob.Character.Position.GrappleData()
	if !ok {
		return Failure
	}
	if position.IsTopSubEligible(mob.Character.Position.State(), gd.ControlLevel) {
		return Success
	}
	return Failure
}

// condMobCanSubmitBottom reports true when the mob is the controlled
// side of a grapple with bottom-attack subs available at the current
// position (e.g., Mount-bottom → Triangle/Armbar reversal).
func condMobCanSubmitBottom(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.Character.Position == nil {
		return Failure
	}
	if !mob.Character.IsBeingControlled() {
		return Failure
	}
	gd, ok := mob.Character.Position.GrappleData()
	if !ok {
		return Failure
	}
	if position.IsBottomSubEligible(mob.Character.Position.State(), gd.ControlLevel) {
		return Success
	}
	return Failure
}

// condMobSubmissionPolicyIs reports true when the mob's current
// SubmissionPolicy matches the params["policy"] string value.
// Used by archetype branches that fork on policy (e.g., a boss
// preset that uses different tactics when set to lethal).
func condMobSubmissionPolicyIs(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	want, ok := params["policy"].(string)
	if !ok {
		return Failure
	}
	wantPolicy, ok := characters.ParseSubmissionPolicy(want)
	if !ok {
		return Failure
	}
	if mob.Character.SubmissionPolicy == wantPolicy {
		return Success
	}
	return Failure
}

func init() {
	conditionRegistry["mob_can_submit_top"] = condMobCanSubmitTop
	conditionRegistry["mob_can_submit_bottom"] = condMobCanSubmitBottom
	conditionRegistry["mob_submission_policy_is"] = condMobSubmissionPolicyIs
}
