package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
)

// wireCombatPhaseBtreeEvents registers a cascade handler on the
// character's Combat Phase that fires the corresponding btree
// transition event whenever combat-phase state changes.
//
// Events fired (once per transition):
//   - mob_engaging     — Idle → Engaging
//   - mob_engaged      — Engaging → Engaged (after RoundsUntil countdown)
//   - mob_disengaging  — Engaged → Disengaging (flee initiated)
//   - mob_combat_ended — any → Idle (combat ends for any reason)
//
// Tick events (mob_combat_round, mob_idle) fire from the round
// driver via DispatchTickEvent; this handler is for transition
// moments only.
//
// Player characters also have CombatPhase but the btree only
// fires for mob instances. Player-side transition events (if
// later needed) would use a different dispatch path.
func wireCombatPhaseBtreeEvents(c *characters.Character) {
	c.CombatPhase.Inner().AfterTransition("btree_transition_events",
		func(from, to combatphase.State, r state.TransitionReason) {
			// Determine the event name from the transition.
			var eventName string
			switch {
			case from == combatphase.Idle && to == combatphase.Engaging:
				eventName = "mob_engaging"
			case from == combatphase.Engaging && to == combatphase.Engaged:
				eventName = "mob_engaged"
			case from == combatphase.Engaged && to == combatphase.Disengaging:
				eventName = "mob_disengaging"
			case to == combatphase.Idle && from != combatphase.Idle:
				eventName = "mob_combat_ended"
			default:
				return
			}

			// Only fire for mob characters — players don't have btree.
			// Look up the mob instance via the registry. CombatPhase
			// itself doesn't know who owns it; we infer from registry.
			mob := findMobOwningCharacter(c)
			if mob == nil {
				return // player or unmapped character; skip
			}

			ctx := behaviortree.EventContext{
				EventType: eventName,
				RoomId:    mob.Character.RoomId,
			}
			// If the reason carries a player target, surface it.
			if !r.Target.IsZero() && r.Target.IsPlayer() {
				ctx.UserId = r.Target.UserId
			}

			behaviortree.TryMobBehavior(mob.InstanceId, ctx)
		})
}

// findMobOwningCharacter scans mob instances to find the one
// whose Character pointer matches c. Used by btree event wiring.
// Returns nil for player characters or transient/test characters.
//
// O(N) over all mob instances. Acceptable for chunk 0 — transitions
// fire at most once per combat phase change (not per round), so
// even 1 000 concurrent mobs costs < 1 µs per event. Optimize in a
// followup if profiling flags it.
func findMobOwningCharacter(c *characters.Character) *mobs.Mob {
	for _, instanceId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instanceId)
		if m != nil && &m.Character == c {
			return m
		}
	}
	return nil
}

func init() {
	characters.OnCharacterCreated(wireCombatPhaseBtreeEvents)
}
