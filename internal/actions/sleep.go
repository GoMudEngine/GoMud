package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
)

// Result is the binary outcome of an actor function call.
// It mirrors the behaviortree Success/Failure vocabulary so that
// mob-schedule callers can forward the value directly without
// importing behaviortree.
type Result int

const (
	// Success indicates the action completed without error.
	Success Result = iota
	// Failure indicates the action was blocked or could not complete.
	Failure
)

// SleepOptions is reserved for future authoring knobs (bed-item
// bonus, custom emote prose, etc.). Empty for chunk 3.3.
type SleepOptions struct{}

// Sleep applies the Sleeping buff (id 15) to the actor's character. Used by
// the player sleep command, the mob sleep command, and the schedule executor
// (via mob.Command("sleep")).
//
// Fails when the actor is in combat or has Aggro. User actors receive a
// player-visible "You can't sleep right now." message; mob actors fail
// silently — the schedule executor retries on the next idle tick after
// combat ends.
//
// Idempotent: if the actor is already sleeping, returns Success without
// re-applying the buff or re-emitting the room emote.
func Sleep(actor Actor, opts SleepOptions) Result {
	c := actor.GetCharacter()
	if c == nil {
		return Failure
	}

	// Idempotent: already sleeping — nothing to do.
	if c.HasBuffFlag(buffs.Sleeping) {
		return Success
	}

	// Combat gate.
	if c.IsInCombat() {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				"You can't sleep right now.")
		}
		return Failure
	}

	// Apply buff 15 (Sleeping). The buff YAML includes the room-broadcast
	// "You are getting some much needed rest." message; we only need to emit
	// the third-person room visual here.
	if err := c.AddBuff(15, false); err != nil {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf("Something prevented you from sleeping: %v", err))
		}
		return Failure
	}

	// Emit third-person room visual to other occupants.
	room := actor.GetRoom()
	if room != nil {
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lies down to sleep.`,
				actor.GetName()),
			actor.GetUserId(),
		)
	}

	return Success
}
