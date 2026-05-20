package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wireLifeCrossMachineCascades registers cascade handlers on
// each character's Life machine. On Alive→Dead, force other
// state machines to terminal states and clear scattered state.
// On Dead→Respawning, reset resources + apply grace buff.
//
// Per-actor concrete effects (loot, teleport, decay, KD
// tracking) live in their own observer files
// (Death_*.go, Respawn_*.go) in chunk-2 Tasks 6-8.
//
// Chunks 3 and 4 will repoint the Activity and Position
// pre-wires (currently raw field manipulation) to proper
// machine queries once those machines land.
func wireLifeCrossMachineCascades(c *characters.Character) {
	c.Life.Inner().AfterTransition("life_cross_machine",
		func(from, to life.State, r state.TransitionReason) {
			switch {
			case from == life.Alive && to == life.Dead:
				// Cross-machine cleanup on death.

				// 1. Combat Phase → Idle
				c.CombatPhase.ForceIdle(state.TransitionReason{
					Trigger: combatphase.TriggerForceIdle,
				})

				// 2. Awareness → Visible (via ForceVisible)
				c.Awareness.ForceVisible(state.TransitionReason{
					Trigger: awareness.TriggerDeath,
				})

				// 3. Activity → Free is now handled by Activity_Cascades.go
				//    (chunk 3). The legacy CastingState/CraftingState pointer
				//    nil-outs that lived here are gone — Activity.TransitionToFree
				//    (fired by the activity_life_dead AfterTransition observer)
				//    clears the per-state data slots inside the machine.
				//
				// Transition-window note: legacy CastingState/CraftingState
				// fields are still being written by skill.cast.go and craft.go
				// until Tasks 6-7 add parallel-writes to the Activity machine.
				// During that window, a player who dies mid-activity will have
				// stale legacy pointer state. Task 11 deletes the legacy fields
				// entirely; Tasks 6-7 keep them in sync via parallel-write.

				// 4. Position → Standing is now handled by the
				//    Position FSM cascade (Position_Cascades.go) which
				//    fires on Life Alive→Dead. The legacy pre-wire that
				//    lived here is removed in chunk-4b R4 now that all
				//    production readers go through the FSM predicates.

				// 5. Buffs (non-permanent) → cancel all.
				c.CancelBuffsWithFlag(buffs.All)

				// 6. Conditions slice → clear.
				c.Conditions = nil

			case from == life.Dead && to == life.Respawning:
				// Player respawn cascade: resource reset + grace buff.
				// (Mobs don't reach Respawning; their instances
				// get cleaned up by the despawn observer.)
				c.Health = c.HealthMax.Value / 20         // 5%
				c.Stamina = c.StaminaMax.Value / 20       // 5%
				c.Conviction = c.ConvictionMax.Value / 20 // 5%
				_ = c.AddBuff(81, false)                  // NoAggroTarget grace

				// Clear damage ledger and notify vitals subscribers.
				clear(c.PlayerDamage)
				if uid := c.GetUserId(); uid != 0 {
					events.AddToQueue(events.CharacterVitalsChanged{UserId: uid})
				}
			}
		})
}

func init() {
	characters.OnCharacterCreated(wireLifeCrossMachineCascades)
}
