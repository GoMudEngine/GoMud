package behaviortree

// actions_mutation.go — btree action primitive for chunk 2.10
// mutation_* actives: try_mutation_active.
//
// Dispatches to the per-mutation TriggerXxx function in the actions
// package. Accepts either `key: <mutation-key>` (single) or
// `keys: [<key1>, <key2>, ...]` (ordered preference list). At least
// one of the two must be set; nodes with neither are rejected at
// call time with a log + Failure.

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// mutationTriggers maps mutation key → action invocation. Add a row
// when lifting a new mutation_* command into the actions package.
//
// Single-target mutations (blinding-spit, toxic-bite) are listed but
// cannot be fully fired through this action — they Failure-out with
// "no-target" because v1 doesn't resolve mob targets here. A future
// chunk will add a target-aware variant.
var mutationTriggers = map[string]func(actions.Actor, actions.MutationOpts) actions.MutationResult{
	"blinding-flash": actions.TriggerBlindingFlash,
	"blinding-spit":  actions.TriggerBlindingSpit,
	"healing-gel":    actions.TriggerHealingGel,
	"pacifism-aura":  actions.TriggerPacifismAura,
	"sonic-shout":    actions.TriggerSonicShout,
	"toxic-bite":     actions.TriggerToxicBite,
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
