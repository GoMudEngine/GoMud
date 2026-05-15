package usercommands

import (
	"errors"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Suicide(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Character.Zone == `Shadow Realm` {
		user.SendText(`You're already dead!`)
		return true, errors.New(`already dead`)
	}

	// Dedupe double-fire: the combat loop can queue `suicide` multiple
	// times before the first one executes (damage from multiple
	// sources in the same round, or rapid re-checks). Skip the second
	// and subsequent fires in the same round to avoid stacking stat
	// decay + skill rust.
	currentRound := util.GetRoundCount()
	if user.Character.LastSuicideRound == currentRound {
		return true, nil
	}
	user.Character.LastSuicideRound = currentRound

	// Revive-on-death buff: heal + clear buff, no death.
	// This path must NEVER reach the Life machine — it keeps the
	// character Alive.
	if user.Character.HasBuffFlag(buffs.ReviveOnDeath) {
		user.Character.Health = user.Character.HealthMax.Value
		user.SendText(`You are revived in a shower of magical sparks!`)
		room.SendTextVisual(
			`<ansi fg="username">`+user.Character.Name+`</ansi> is suddenly revived in a shower of sparks!`,
			user.UserId,
		)
		user.Character.CancelBuffsWithFlag(buffs.ReviveOnDeath)
		return true, nil
	}

	// Trigger the Life cascade. All cleanup, announcements, corpse
	// creation, teleport, decay, and respawn happen via observers
	// wired to the three transitions below.
	damageSnapshot := copyPlayerDamage(user.Character.PlayerDamage)
	_ = user.Character.Life.TransitionToDead(
		life.DeadData{
			DamageMap: damageSnapshot,
		},
		state.TransitionReason{
			Trigger: life.TriggerSuicide,
			Actor:   state.ActorRef{UserId: user.UserId},
		},
	)
	_ = user.Character.Life.TransitionToRespawning(
		life.RespawningData{DestRoomId: user.Character.ResolveRespawnRoom()},
		state.TransitionReason{Trigger: life.TriggerRespawnReady},
	)
	_ = user.Character.Life.TransitionToAlive(
		state.TransitionReason{Trigger: life.TriggerRespawnComplete},
	)

	return true, nil
}

// copyPlayerDamage returns a shallow copy of the PlayerDamage map so
// observers receive a stable snapshot that isn't modified by
// subsequent clear() calls in Life_Cascades.go.
func copyPlayerDamage(src map[int]int) map[int]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[int]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
