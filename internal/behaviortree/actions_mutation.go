package behaviortree

// actions_mutation.go — btree action primitive for chunk 2.10
// mutation_* actives: try_mutation_active.
//
// Dispatches to the per-mutation TriggerXxx function in the actions
// package. Accepts either `key: <mutation-key>` (single) or
// `keys: [<key1>, <key2>, ...]` (ordered preference list). At least
// one of the two must be set; nodes with neither are rejected at
// call time with a log + Failure.
//
// Single-target mutations (blinding-spit, toxic-bite) are intentionally
// excluded from this map. They require a resolved target actor before
// dispatch; without one the mutationPreamble succeeds (consuming stamina
// and the special-move cooldown) and then TriggerXxx immediately returns
// BlockReason="no-target", wasting both resources. A future
// try_mutation_active_at_target primitive will add target resolution
// before dispatch. See the project_mutation_active_runtime_evolution_btree
// memory entry for design details.

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// mutationTriggers maps mutation key → action invocation. Add a row
// when lifting a new SELF or AoE mutation into the actions package.
//
// Single-target mutations (blinding-spit, toxic-bite) are intentionally
// excluded — they need a target-resolving primitive that does not exist
// yet. See the file-level godoc above for the full rationale.
var mutationTriggers = map[string]func(actions.Actor, actions.MutationOpts) actions.MutationResult{
	"blinding-flash": actions.TriggerBlindingFlash,
	"healing-gel":    actions.TriggerHealingGel,
	"pacifism-aura":  actions.TriggerPacifismAura,
	"sonic-shout":    actions.TriggerSonicShout,
	// blinding-spit and toxic-bite intentionally excluded — they require
	// a resolved target. A future try_mutation_active_at_target primitive
	// will dispatch them with target resolution.
}

// actTryMutationActive fires the first available mutation in the
// preference list. Success on a triggered mutation; Failure if no
// candidate fires (missing mutation, on cooldown, low stamina, no
// target, no entry in mutationTriggers).
//
// Validation: rejects nodes with neither `key` nor `keys` set. Logs a
// clear error and returns Failure so the author sees the misconfig.
func actTryMutationActive(params map[string]any, ctx *EvalContext) Result {
	keys := collectMutationKeys(params)
	if len(keys) == 0 {
		mudlog.Error("try_mutation_active",
			"error", "node missing required `key` or `keys` parameter",
			"instance_id", ctx.InstanceId)
		return Failure
	}

	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := actions.NewMobActorInRoom(mob, room)

	for _, key := range keys {
		trigger, ok := mutationTriggers[key]
		if !ok {
			mudlog.Warn("try_mutation_active",
				"warn", "unknown mutation key (no actions.TriggerXxx)",
				"key", key, "instance_id", ctx.InstanceId)
			continue
		}
		res := trigger(actor, actions.MutationOpts{})
		if res.Triggered {
			return Success
		}
		// BlockReason in {no-mutation, on-cooldown, low-stamina,
		// not-in-combat, no-target} — fall through to next candidate.
	}
	return Failure
}

// collectMutationKeys returns the ordered preference list from params.
// `key` (single string) and `keys` ([]string) are both accepted; when
// both are set, `key` takes precedence as the first entry, then `keys`
// are appended in order.
func collectMutationKeys(params map[string]any) []string {
	out := []string{}
	if single := getStringParam(params, "key"); single != "" {
		out = append(out, single)
	}
	if list := getStringListParam(params, "keys"); len(list) > 0 {
		out = append(out, list...)
	}
	return out
}
