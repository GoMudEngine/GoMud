package behaviortree

// actions_forager_verbs.go — btree action primitives for chunk 2.9
// forager-suite verbs: try_forage, try_salvage, wander_territory.
//
// try_forage / try_salvage delegate to actions.Forage / actions.Salvage.
// wander_territory delegates to npcWanderTerritoryNeighbor (the
// existing helper in actions_forager.go) so territory-aware
// movement is preserved when the foraging tick loop runs in YAML.

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// actTryForage runs one forage attempt via actions.Forage.
// Returns Success on item found; Failure on miss/cooldown/wrong-biome.
func actTryForage(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Forage(actor, actions.ForageOptions{})
	if result.Found {
		return Success
	}
	return Failure
}

// actTrySalvage runs one salvage tick via actions.Salvage. Default
// mode targets the first eligible corpse in the room.
//
// Optional param "item_uuid" overrides to item-target mode (string).
func actTrySalvage(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}

	opts := actions.SalvageOptions{TargetCorpse: true}
	if uuid := getStringParam(params, "item_uuid"); uuid != "" {
		opts = actions.SalvageOptions{TargetItemUuid: uuid}
	}

	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Salvage(actor, opts)
	if result.Succeeded {
		return Success
	}
	return Failure
}

// actWanderTerritory delegates to the existing
// npcWanderTerritoryNeighbor helper in actions_forager.go for
// territory-aware adjacent-room movement. Returns Failure on
// mobs without a forager profile.
func actWanderTerritory(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	profile := forager.ProfileFor(int(mob.MobId))
	if profile == nil {
		return Failure
	}
	npcWanderTerritoryNeighbor(profile, mob, ctx)
	return Success
}
