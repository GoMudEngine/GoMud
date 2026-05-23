package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// victimEngagedWithActor reports whether the given victim mob's current combat
// target is the actor. Handles both player and mob actors correctly:
//   - For player actors, matches on UserId (non-zero).
//   - For mob actors, matches on MobInstanceId (non-zero). This is necessary
//     because MobActor.GetUserId() always returns 0, so a bare UserId comparison
//     would incorrectly match every mob engaged with another mob (UserId == 0
//     for mob-vs-mob fights).
//
// Uses CurrentCombatTarget (not EngagedTarget) so the Aggro field fallback
// works in test fixtures where CombatPhase is nil.
func victimEngagedWithActor(victim *characters.Character, actor Actor) bool {
	target := victim.CurrentCombatTarget()
	if actor.IsPlayer() {
		return target.UserId != 0 && target.UserId == actor.GetUserId()
	}
	// Mob actor — match on mob instance id.
	return target.MobInstanceId != 0 && target.MobInstanceId == actor.GetMobInstanceId()
}

// TriggerPacifismAura fires the pacifism-aura mutation for any Actor (player
// or mob). AoE de-aggro: any mob in the room that is attacking the actor has
// its Aggro cleared. The actor always receives ConditionRecoveryPenalty
// (duration 3, magnitude 1.0) — the pacifism wash prevents the actor from
// attacking while the aura's influence lingers. The actor's own Aggro is also
// cleared.
//
// Gates: must have "pacifism-aura" mutation + pass the shared special-move
// cooldown + have at least 12 stamina. Combat is NOT required (matches original
// player command which has no IsInCombat gate).
//
// Behavioral note: the original player command emitted a numeric count of
// de-aggroed creatures ("N creature(s) lose interest in attacking you.").
// This violates the no-hard-numbers rule (CLAUDE.md). The count message is
// replaced with a descriptive variant when at least one creature is affected.
//
// Skill note: the original player command fires a Skullduggery skill-use event.
// Preserved verbatim — pacifism-aura is a stealth/deception mutation, not a
// combat one, so Skullduggery is intentional.
func TriggerPacifismAura(actor Actor, opts MutationOpts) MutationResult {
	pre := mutationPreamble(actor, "pacifism-aura", false /* combatRequired=false */, 12)
	if !pre.OK {
		return MutationResult{BlockReason: pre.BlockReason}
	}

	char := actor.GetCharacter()

	room := actor.GetRoom()
	if room == nil {
		return MutationResult{BlockReason: "no-room"}
	}

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`<ansi fg="cyan-bold">A wave of calming energy pulses outward from you. Hostile intent fades from nearby creatures.</ansi>`)
	}
	room.SendTextVisual(
		messaging.CategoryMutation,
		fmt.Sprintf(
			`<ansi fg="cyan">A soothing aura radiates from <ansi fg="username">%s</ansi>. Nearby creatures seem to lose their aggression.</ansi>`,
			actor.GetName(),
		),
		actor.GetUserId(),
	)

	// AoE de-aggro: clear Aggro on any room mob that is targeting the actor.
	// victimEngagedWithActor handles both player and mob actors correctly:
	// a bare UserId comparison would match every mob-vs-mob fight in the
	// room when actor is a mob (GetUserId() == 0 for all mob actors).
	deAggroCount := 0
	for _, mobInstId := range room.GetMobs(rooms.FindAll) {
		victim := mobs.GetInstance(mobInstId)
		if victim == nil || !victim.Character.IsInCombat() {
			continue
		}
		if victimEngagedWithActor(&victim.Character, actor) {
			victim.Character.EndAggro()
			deAggroCount++
		}
	}

	if deAggroCount > 0 && actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`<ansi fg="cyan">Nearby creatures lose interest in attacking you.</ansi>`)
	}

	// Self-penalty: the pacifism wash prevents the actor from attacking.
	char.AddCondition(characters.ConditionRecoveryPenalty, 3, 1.0, "pacifism-aura")
	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`<ansi fg="yellow">The aura's peaceful influence washes over you as well. You cannot bring yourself to attack.</ansi>`)
	}

	// Clear the actor's own Aggro.
	char.EndAggro()

	actor.OnSkillUse(string(skills.Skullduggery))

	return MutationResult{Triggered: true, AffectedCount: deAggroCount}
}
