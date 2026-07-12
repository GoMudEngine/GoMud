package behaviortree

// actions_mutation.go — btree action primitives for chunk 2.10+
// mutation_* actives: try_mutation_active, try_any_active_mutation.
//
// try_mutation_active dispatches to the per-mutation TriggerXxx function in
// the actions package. Accepts either `key: <mutation-key>` (single) or
// `keys: [<key1>, <key2>, ...]` (ordered preference list). At least one of
// the two must be set; nodes with neither are rejected at call time with a
// log + Failure.
//
// try_any_active_mutation enumerates the mob's current self/AoE mutations at
// tick time, sorted by rarity descending (alphabetical key tiebreak), and
// fires the first one that successfully triggers. Coexists with
// try_mutation_active (chunk 2.10's explicit-keys version). Intended for
// autonomous "use whatever you have" archetypes that should not require manual
// archetype edits when a mob evolves a new mutation at runtime.
//
// Single-target mutations (blinding-spit, toxic-bite) are intentionally
// excluded from try_mutation_active / try_any_active_mutation. They require
// a resolved target actor before dispatch; without one the mutationPreamble
// succeeds (consuming stamina and the special-move cooldown) and then
// TriggerXxx immediately returns BlockReason="no-target", wasting both
// resources. try_mutation_active_at_target dispatches them instead: it
// resolves the mob's engaged combat target (actions.ResolveEngagedTarget —
// CombatPhase FSM + Aggro fallback) BEFORE dispatch and fails fast when
// there is none.

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// mutationTriggers maps mutation key → action invocation. Add a row
// when lifting a new SELF or AoE mutation into the actions package.
//
// Single-target mutations (blinding-spit, toxic-bite) are intentionally
// excluded — they need a target-resolving primitive that does not exist
// yet. See the file-level godoc above for the full rationale.
// Empty since the legacy active-ability mutations were removed (2026-07-12
// NPC migration). The dispatch machinery is retained generically — add a row
// when lifting a new SELF/AoE active mutation into the actions package.
var mutationTriggers = map[string]func(actions.Actor, actions.MutationOpts) actions.MutationResult{}

// mutationTriggersAtTarget maps single-target mutation key → trigger.
// These need a resolved target actor; actTryMutationActiveAtTarget
// resolves the mob's engaged combat target before dispatching. Add a row
// when lifting a new single-target mutation into the actions package.
// Empty since the legacy single-target active mutations were removed
// (2026-07-12 NPC migration). Add a row when lifting a new single-target
// active mutation into the actions package.
var mutationTriggersAtTarget = map[string]func(actions.Actor, actions.MutationOpts) actions.MutationResult{}

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

// actTryAnyActiveMutation fires the rarest available self/AoE mutation the
// mob currently owns, falling through to less-rare ones if the rarer ones
// cannot fire (on cooldown, low stamina, not in combat, etc.). Returns Failure
// if no candidate fires.
//
// Single-target mutations (blinding-spit, toxic-bite) are excluded: they
// require target resolution that this action does not perform. A future
// try_mutation_active_at_target primitive will dispatch them with target
// resolution.
//
// Determinism: candidates are sorted by rarity descending, with alphabetical
// key tiebreak, so ordering is reproducible regardless of Go map iteration
// order.
//
// This action coexists with try_mutation_active (chunk 2.10's explicit-keys
// version): use try_mutation_active for curated archetype preference lists,
// use try_any_active_mutation for autonomous "use whatever I have" archetypes
// that should pick up newly evolved mutations without manual YAML edits.
func actTryAnyActiveMutation(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}

	// Enumerate mutations the mob owns that are in the self/AoE dispatch map.
	candidates := make([]string, 0, len(mob.Character.Mutations))
	for key := range mob.Character.Mutations {
		if _, ok := mutationTriggers[key]; ok {
			candidates = append(candidates, key)
		}
	}
	if len(candidates) == 0 {
		return Failure
	}

	// Sort: rarity descending, alphabetical key tiebreak for determinism.
	sort.SliceStable(candidates, func(i, j int) bool {
		ri := mutationRarity(candidates[i])
		rj := mutationRarity(candidates[j])
		if ri != rj {
			return ri > rj // higher rarity fires first
		}
		return candidates[i] < candidates[j] // alphabetical tiebreak
	})

	actor := actions.NewMobActorInRoom(mob, room)
	for _, key := range candidates {
		trigger := mutationTriggers[key]
		res := trigger(actor, actions.MutationOpts{})
		if res.Triggered {
			return Success
		}
		// BlockReason in {no-mutation, on-cooldown, low-stamina,
		// not-in-combat, no-target} — fall through to next candidate.
	}
	return Failure
}

// mutationRarity returns the rarity value for a mutation key by consulting the
// mutations registry. Returns 0 for unknown keys so they sort last.
func mutationRarity(key string) int {
	spec := mutations.GetMutation(key)
	if spec == nil {
		return 0
	}
	return spec.Rarity
}

// actTryMutationActiveAtTarget fires a single-target mutation
// (blinding-spit, toxic-bite) at the mob's engaged combat target.
//
// Target resolution is implicit: actions.ResolveEngagedTarget consults
// CurrentCombatTarget (CombatPhase FSM + Aggro fallback). Resolution
// happens BEFORE any trigger dispatch — the trigger preamble deducts
// stamina and arms the shared special-move cooldown ahead of its own
// no-target check, so dispatching without a target would waste both.
// No resolvable target → immediate Failure, resources untouched.
//
// Keys behavior: `key`/`keys` select a curated preference list (same
// shape as try_mutation_active). With neither set, the action enumerates
// the mob's owned single-target mutations sorted by rarity descending
// (alphabetical tiebreak) — the try_any_active_mutation analog, so
// autonomous archetypes pick up runtime-evolved mutations without YAML
// edits.
func actTryMutationActiveAtTarget(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}

	keys := collectMutationKeys(params)
	if len(keys) == 0 {
		for key := range mob.Character.Mutations {
			if _, ok := mutationTriggersAtTarget[key]; ok {
				keys = append(keys, key)
			}
		}
		sort.SliceStable(keys, func(i, j int) bool {
			ri := mutationRarity(keys[i])
			rj := mutationRarity(keys[j])
			if ri != rj {
				return ri > rj // higher rarity fires first
			}
			return keys[i] < keys[j] // alphabetical tiebreak
		})
	}
	if len(keys) == 0 {
		return Failure
	}

	actor := actions.NewMobActorInRoom(mob, room)
	target := actions.ResolveEngagedTarget(actor, room)
	if target == nil {
		return Failure
	}

	for _, key := range keys {
		trigger, ok := mutationTriggersAtTarget[key]
		if !ok {
			mudlog.Warn("try_mutation_active_at_target",
				"warn", "unknown single-target mutation key (no actions.TriggerXxx or not single-target)",
				"key", key, "instance_id", ctx.InstanceId)
			continue
		}
		res := trigger(actor, actions.MutationOpts{TargetActor: target})
		if res.Triggered {
			return Success
		}
		// BlockReason in {no-mutation, on-cooldown, low-stamina,
		// not-in-combat} — fall through to next candidate.
	}
	return Failure
}
