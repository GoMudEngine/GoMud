package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// --- Self-position queries (7) ---

func condMobIsStanding(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsStanding() {
		return Success
	}
	return Failure
}

func condMobIsProne(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsProne() {
		return Success
	}
	return Failure
}

func condMobIsGrappling(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsGrappling() {
		return Success
	}
	return Failure
}

func condMobInMount(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsMount() {
		return Success
	}
	return Failure
}

func condMobInGuard(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsGuard() {
		return Success
	}
	return Failure
}

func condMobInClinch(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsClinch() {
		return Success
	}
	return Failure
}

func condMobInTopDominant(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsTopDominant() {
		return Success
	}
	return Failure
}

// --- Target-position queries (3) ---

func condTargetIsStanding(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.IsStanding() {
		return Success
	}
	return Failure
}

func condTargetIsProne(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.IsProne() {
		return Success
	}
	return Failure
}

func condTargetIsGrappled(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.IsGrappling() {
		return Success
	}
	return Failure
}

// --- Control-axis queries (chunk 4b) ---

func condMobIsInControl(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsController() {
		return Success
	}
	return Failure
}

func condMobIsBeingControlled(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsBeingControlled() {
		return Success
	}
	return Failure
}

// condMobControlAtLeast was a ControlLevel-gradient check that compared
// the mob's per-round drift needle to a named threshold. The drift
// needle (ControlLevel) is removed in chunk 4b-fixup T18.
//
// TODO: T19 — sunset/convert. The condition registry key
// "mob_control_at_least" is kept so existing behavior-tree YAML
// does not crash on load, but the function always returns Failure
// until T19 re-engineers it against the new per-round drift snapshot
// (Character.LastDriftRoll.MarginAttacker) or removes the key
// from the registry entirely.
func condMobControlAtLeast(params map[string]any, ctx *EvalContext) Result {
	// TODO: T19 — re-engineer or remove
	return Failure
}

func condMobLowGrappleStamina(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsLowGrappleStamina() {
		return Success
	}
	return Failure
}

func condTargetIsInControl(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.IsController() {
		return Success
	}
	return Failure
}

func condTargetIsBeingControlled(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.IsBeingControlled() {
		return Success
	}
	return Failure
}

func init() {
	conditionRegistry["mob_is_standing"] = condMobIsStanding
	conditionRegistry["mob_is_prone"] = condMobIsProne
	conditionRegistry["mob_is_grappling"] = condMobIsGrappling
	conditionRegistry["mob_in_mount"] = condMobInMount
	conditionRegistry["mob_in_guard"] = condMobInGuard
	conditionRegistry["mob_in_clinch"] = condMobInClinch
	conditionRegistry["mob_in_top_dominant"] = condMobInTopDominant
	conditionRegistry["target_is_standing"] = condTargetIsStanding
	conditionRegistry["target_is_prone"] = condTargetIsProne
	conditionRegistry["target_is_grappled"] = condTargetIsGrappled
	conditionRegistry["mob_is_in_control"] = condMobIsInControl
	conditionRegistry["mob_is_being_controlled"] = condMobIsBeingControlled
	conditionRegistry["mob_control_at_least"] = condMobControlAtLeast
	conditionRegistry["mob_low_grapple_stamina"] = condMobLowGrappleStamina
	conditionRegistry["target_is_in_control"] = condTargetIsInControl
	conditionRegistry["target_is_being_controlled"] = condTargetIsBeingControlled
}
