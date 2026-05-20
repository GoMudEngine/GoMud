package behaviortree

// actions_skullduggery.go — btree action primitives for the skullduggery suite:
//   try_sneak, try_steal, try_plant, try_shadow, try_defuse
//
// Each primitive resolves its target via resolveSkullduggeryTarget (defined in
// conditions_skullduggery.go) and delegates to the corresponding actions.<Verb>
// function. Returning Success/Failure mirrors the conventions in
// actions_combat.go.

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// actTrySneak invokes actions.Sneak via MobActor.
// Returns Success when the actor entered the hidden state (Success=true) or
// was already hidden (AlreadyHidden=true). Returns Failure for all other
// outcomes (in-combat, spotted by an observer).
func actTrySneak(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Sneak(actor)
	if result.Success || result.AlreadyHidden {
		return Success
	}
	return Failure
}

// actTrySteal runs actions.Steal against the target resolved from ctx.
// Target resolution: Event.UserId → Aggro.UserId → Aggro.MobInstanceId.
// Returns Failure if no target is resolvable.
func actTrySteal(params map[string]any, ctx *EvalContext) Result {
	mob, opts := resolveStealOptsFromContext(ctx)
	if mob == nil {
		return Failure
	}
	if opts.TargetUserId == 0 && opts.TargetMobInstanceId == 0 {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Steal(actor, opts)
	if result.Succeeded {
		return Success
	}
	return Failure
}

// actTryPlant runs actions.Plant on the resolved target.
// Requires an "item_tag" param that names the item to plant from the mob's
// backpack. Returns Failure immediately if item_tag is absent.
func actTryPlant(params map[string]any, ctx *EvalContext) Result {
	itemTag := getStringParam(params, "item_tag")
	if itemTag == "" {
		return Failure
	}
	mob, opts := resolvePlantOptsFromContext(ctx, itemTag)
	if mob == nil {
		return Failure
	}
	if opts.TargetUserId == 0 && opts.TargetMobInstanceId == 0 {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Plant(actor, opts)
	if result.Succeeded {
		return Success
	}
	return Failure
}

// actTryShadow runs actions.Shadow against the resolved target.
// The actions.Shadow implementation enforces the hidden-required precondition
// and returns Reason:"not hidden" when the mob lacks buff 9. Returns Failure
// in that case.
func actTryShadow(params map[string]any, ctx *EvalContext) Result {
	mob, opts := resolveShadowOptsFromContext(ctx)
	if mob == nil {
		return Failure
	}
	if opts.TargetUserId == 0 && opts.TargetMobInstanceId == 0 {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Shadow(actor, opts)
	if result.Succeeded {
		return Success
	}
	return Failure
}

// actTryDefuse scans the room for the first trapped container or exit and
// calls actions.Defuse with that noun as the TargetNoun. Returns Failure if
// no trapped targets exist in the room (saving a wasted Defuse call and kit
// consumption).
func actTryDefuse(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}

	// Find the first trapped container.
	targetNoun := ""
	for name, container := range room.Containers {
		if container.HasLock() && len(container.Lock.TrapBuffIds) > 0 {
			targetNoun = name
			break
		}
	}

	// If no trapped container, try exits.
	if targetNoun == "" {
		for exitName, exitInfo := range room.Exits {
			if exitInfo.HasLock() && len(exitInfo.Lock.TrapBuffIds) > 0 {
				targetNoun = exitName
				break
			}
		}
	}

	if targetNoun == "" {
		return Failure
	}

	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Defuse(actor, actions.DefuseOptions{TargetNoun: targetNoun})
	if result.Succeeded {
		return Success
	}
	return Failure
}

// ─── Target-resolution helpers ───────────────────────────────────────────────
//
// These helpers build the Options structs for each action by calling
// resolveSkullduggeryTarget from conditions_skullduggery.go. Centralising the
// resolution logic here avoids duplicating the fallback chain in every action.

func resolveStealOptsFromContext(ctx *EvalContext) (*mobs.Mob, actions.StealOptions) {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return nil, actions.StealOptions{}
	}
	user, targetMob := resolveSkullduggeryTarget(ctx)
	opts := actions.StealOptions{}
	if user != nil {
		opts.TargetUserId = user.UserId
	} else if targetMob != nil {
		opts.TargetMobInstanceId = targetMob.InstanceId
	}
	return mob, opts
}

func resolvePlantOptsFromContext(ctx *EvalContext, itemTag string) (*mobs.Mob, actions.PlantOptions) {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return nil, actions.PlantOptions{}
	}
	user, targetMob := resolveSkullduggeryTarget(ctx)
	opts := actions.PlantOptions{ItemNoun: itemTag}
	if user != nil {
		opts.TargetUserId = user.UserId
	} else if targetMob != nil {
		opts.TargetMobInstanceId = targetMob.InstanceId
	}
	return mob, opts
}

func resolveShadowOptsFromContext(ctx *EvalContext) (*mobs.Mob, actions.ShadowOptions) {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return nil, actions.ShadowOptions{}
	}
	user, targetMob := resolveSkullduggeryTarget(ctx)
	opts := actions.ShadowOptions{}
	if user != nil {
		opts.TargetUserId = user.UserId
	} else if targetMob != nil {
		opts.TargetMobInstanceId = targetMob.InstanceId
	}
	return mob, opts
}
