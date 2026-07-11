package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Flee makes a mob disengage from combat and move to a random adjacent room.
// The flee attempt is gated by an opposed roll: mob Dex+Skullduggery vs each
// attacker's Dex+UnarmedCombat. If any attacker wins the roll, the mob is
// blocked and stays in the room.
func Flee(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Non-combatant mobs never flee (they should never be in combat)
	if mob.IsNonCombatant() {
		return true, nil
	}

	// A no-flee state (e.g. the Blood Frenzy mutation state) forbids retreat —
	// mirrors the player-flee gate so a frenzied mob cannot panic-flee.
	if mob.Character.HasBuffFlag(buffs.NoFlee) {
		return true, nil
	}

	// Can't flee while grappled. Chunk 4b R3: FSM-driven — covers all
	// 11 grapple states via the two rollups (legacy enum only knew the
	// Clinched / Grounded buckets). Mirrors the player-flee
	// grapple-block UX in handlePlayerFlee — the player needs to see
	// that their grapple is holding the mob even when the mob tries
	// to break free. Pre-chunk-4b this returned silently, which made
	// the grapple feel passive.
	if mob.Character.IsStandingGrapple() || mob.Character.IsGroundGrapple() {
		sendRoomText(room, messaging.CategoryGrappleFlow,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to break free but you've got them locked down!`, mob.Character.Name))
		return true, nil
	}

	// Shared opposed-roll blocker resolution (combat.ResolveFleeBlockers).
	// Identical math to the player-flee path in handlePlayerFlee; the
	// helper handles both player and mob blockers, the prone penalty,
	// and the targeting filter.
	if combat.ResolveFleeBlockers(&mob.Character, room) != nil {
		sendRoomText(room, messaging.CategoryRoomExit,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to flee but is blocked!`, mob.Character.Name))
		return true, nil
	}

	// If in combat, clear aggro
	if mob.Character.IsInCombat() {
		mob.Character.EndAggro()
	}

	// Get a random exit (skips secret and locked exits)
	exitName, _ := room.GetRandomExit()
	if exitName == "" {
		// Cornered — no exits available. Surface the panic visibly so
		// the room knows the mob tried and failed (pre-fix this was
		// silent, making the mob appear to randomly exit combat).
		sendRoomText(room, messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> looks around frantically for an escape but finds none!`, mob.Character.Name))
		return true, nil
	}

	// Send flee message before moving
	sendRoomText(room, messaging.CategoryRoomExit,
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> flees!`, mob.Character.Name))

	// Move via the existing Go command
	Go(exitName, mob, room)

	behaviortree.TryMobBehavior(mob.InstanceId, behaviortree.EventContext{
		EventType: "mob_flee",
		RoomId:    mob.Character.RoomId,
	})

	return true, nil
}
