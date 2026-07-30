package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// TargetAsleep reports whether a character is asleep for the purposes of
// player-initiated social and trade interaction.
//
// Sleep is a real state with real consequences — sleepers take an auto-critting
// first round and 5x regen — but until 2026-07-30 only `steal` respected it.
// `ask`, `give`, `buy`, `sell` and `list` all ignored it, so a player could hold
// a full conversation with a sleeping tavern keeper, be granted a quest by him,
// and hand a quest item to a sleeping guard captain. That is not a corner case:
// 113 of 132 authored schedules put the NPC to sleep IN their own workplace, so
// most shopkeepers in the world are asleep on the spot at night.
func TargetAsleep(c *characters.Character) bool {
	return c != nil && c.HasBuffFlag(buffs.Sleeping)
}

// RefuseIfAsleep is the single gate every player-initiated interaction with an
// NPC should pass through. It reports true (and tells the player) when the
// target is asleep, in which case the caller must return WITHOUT changing any
// state.
//
// Deliberately a refusal rather than a wake-up: sleeping is meant to mean
// something, and a shop that trades at 3am with its keeper snoring in the
// corner makes the schedule system decorative. The player is not dead-ended,
// though — `shout` already wakes every sleeper in the room (see
// usercommands/shout.go), so the message points at that. Damage, a failed
// steal, a light source entering, and the end of the sleep schedule segment
// also wake them.
//
// IMPORTANT for `give`: give.go transfers the item to the mob BEFORE any
// handler fires and cannot undo it, so this MUST be called before the transfer,
// not from a handler.
func RefuseIfAsleep(c *characters.Character, name string, user *users.UserRecord) bool {
	if !TargetAsleep(c) {
		return false
	}
	if user != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> is fast asleep. `+
				`You could make some noise -- try <ansi fg="command">shout</ansi>.`, name))
	}
	return true
}

// RefuseMobIfAsleep is the mob-shaped convenience wrapper around
// RefuseIfAsleep, using the mob's own display name.
func RefuseMobIfAsleep(mob *mobs.Mob, user *users.UserRecord) bool {
	if mob == nil {
		return false
	}
	return RefuseIfAsleep(&mob.Character, mob.Character.Name, user)
}

// ShopClosedForSleep returns a sleeping merchant when the room HAS merchants
// and every one of them is asleep — i.e. the shop is shut for the night.
// It returns nil when any merchant is awake (trade proceeds as normal) or when
// there is no merchant at all (the caller's existing "no merchant here" path is
// the right answer, not a sleep message).
//
// Guarding at this level rather than inside merchant resolution keeps buy/sell
// pricing and inventory logic untouched, and keeps the refusal message accurate:
// "Marn is fast asleep" instead of a misleading "there's no merchant here" while
// the keeper snores in plain view.
func ShopClosedForSleep(room *rooms.Room) *mobs.Mob {
	var asleep *mobs.Mob
	found := false
	for _, mobId := range room.GetMobs(rooms.FindMerchant) {
		m := mobs.GetInstance(mobId)
		if m == nil {
			continue
		}
		found = true
		if !TargetAsleep(&m.Character) {
			return nil // someone is awake and open for business
		}
		if asleep == nil {
			asleep = m
		}
	}
	if !found {
		return nil
	}
	return asleep
}

// MerchantAwayAsleep returns a merchant whose HOME room is this one but who is
// currently asleep somewhere else — the shopkeeper who has gone to the back
// room or upstairs for the night.
//
// 19 of 132 schedules move the NPC out of their workplace to sleep (the rest
// sleep on the spot). For those, a player walking into the shop at 2am finds an
// empty room and the stock message "there's no merchant here", which reads as
// though they came to the wrong place. This lets the caller say what is
// actually true: the keeper has shut up shop for the night.
//
// Only consulted on the no-merchant-present path, so the instance scan is rare.
func MerchantAwayAsleep(room *rooms.Room) *mobs.Mob {
	if room == nil {
		return nil
	}
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m == nil || m.HomeRoomId != room.RoomId {
			continue
		}
		if m.Character.RoomId == room.RoomId {
			continue // actually here; not our case
		}
		if len(m.Character.Shop) == 0 {
			continue // not a merchant
		}
		if TargetAsleep(&m.Character) {
			return m
		}
	}
	return nil
}
