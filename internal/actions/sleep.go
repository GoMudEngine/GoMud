package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// SleepOptions is reserved for future authoring knobs (bed-item
// bonus, custom emote prose, etc.). Empty for chunk 3.3.
type SleepOptions struct{}

// SleepResult is the structured outcome of a Sleep call.
type SleepResult struct {
	Success bool   // true if the buff was applied (or was already applied — idempotent)
	Reason  string // empty on success; populated on failure (e.g., "in combat")
}

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
func Sleep(actor Actor, opts SleepOptions) SleepResult {
	c := actor.GetCharacter()
	if c == nil {
		return SleepResult{Success: false, Reason: "no character"}
	}

	// Idempotent: already sleeping — nothing to do.
	if c.HasBuffFlag(buffs.Sleeping) {
		return SleepResult{Success: true}
	}

	// Combat gate.
	if c.IsInCombat() {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				"You can't sleep right now.")
		}
		return SleepResult{Success: false, Reason: "in combat"}
	}

	// Apply buff 15 (Sleeping). The buff YAML includes the room-broadcast
	// "You are getting some much needed rest." message; we only need to emit
	// the third-person room visual here.
	if err := c.AddBuff(15, false); err != nil {
		// Never surface the raw internal error (it leaks the buff id). Log it
		// for ops and give the player clean flavor.
		mudlog.Error("Sleep", "msg", "AddBuff(15 Sleeping) failed", "actor", actor.GetName(), "error", err)
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				"You can't seem to settle into sleep right now.")
		}
		return SleepResult{Success: false, Reason: err.Error()}
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

	return SleepResult{Success: true}
}
