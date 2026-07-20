package characters

import (
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// Die routes a character through the Life machine death sequence.
//
// For players (UserId != 0) the full three-step cascade fires:
//
//	Dead → Respawning → Alive
//
// For mobs (UserId == 0) only the Dead transition fires; the
// instance-cleanup observer (Death_MobInstanceCleanup.go) handles
// despawn, and mobs never enter the Respawning state.
//
// Callers MUST pre-check:
//   - ReviveOnDeath buff (already handled at each call site)
//   - LastSuicideRound dedupe (if the call site can double-fire)
//   - Shadow Realm zone guard (player sites only)
//
// Die is idempotent: if the Life machine is already Dead or
// Respawning it returns immediately without firing observers.
func (c *Character) Die(killer state.ActorRef, trigger string) {
	if !c.IsAlive() {
		return
	}

	damageSnapshot := snapshotDamageMap(c.PlayerDamage)

	_ = c.Life.TransitionToDead(
		life.DeadData{
			Killer:    killer,
			DamageMap: damageSnapshot,
		},
		state.TransitionReason{
			Trigger: trigger,
			Actor:   killer,
		},
	)

	// Mobs stay at Dead — the instance cleanup observer fires
	// synchronously in the AfterTransition chain above and
	// despawns the mob.  The Life machine for mob characters
	// does not advance past Dead.
	if c.GetUserId() == 0 {
		return
	}

	_ = c.Life.TransitionToRespawning(
		life.RespawningData{DestRoomId: c.ResolveRespawnRoom()},
		state.TransitionReason{Trigger: life.TriggerRespawnReady},
	)
	_ = c.Life.TransitionToAlive(
		state.TransitionReason{Trigger: life.TriggerRespawnComplete},
	)
}

// snapshotDamageMap returns a shallow copy of src so that
// observers see a stable damage map even if the original is
// cleared by Life_Cascades.go after the transition.
func snapshotDamageMap(src map[int]int) map[int]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[int]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
