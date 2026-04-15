package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Dismiss severs the player's bond with a companion.
// For living mobs the charm is broken, the mob turns hostile, and summoned /
// conjured / raised companions are queued for a delayed despawn.
// Syntax: dismiss <name>
func Dismiss(rest string, user *users.UserRecord,
	room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)
	if rest == "" {
		user.SendText("Dismiss whom? (dismiss <companion name>)")
		return true, nil
	}

	if len(user.Character.Companions) == 0 {
		user.SendText("You have no companions to dismiss.")
		return true, nil
	}

	comp := user.Character.GetCompanion(rest)
	if comp == nil {
		user.SendText(fmt.Sprintf(
			`You have no companion named "%s".`, rest,
		))
		return true, nil
	}

	compName := comp.Name
	instanceId := comp.InstanceId
	sourceType := comp.SourceType

	mob := mobs.GetInstance(instanceId)

	if mob == nil {
		// Companion is offline / already gone — just clean up the record.
		user.Character.RemoveCompanion(instanceId)
		user.SendText(fmt.Sprintf(
			`Your bond with <ansi fg="mobname">%s</ansi> fades away.`,
			compName,
		))
		return true, nil
	}

	// Break the charm link if one exists.
	if mob.Character.IsCharmed(user.UserId) {
		mob.Character.RemoveCharm()
	}

	// Remove from CharmedMobs tracking on the player.
	user.Character.TrackCharmed(instanceId, false)

	// Remove the companion record before doing anything that might trigger
	// room-wide combat logic.
	user.Character.RemoveCompanion(instanceId)

	// Make the mob hostile toward the player.
	mob.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)

	// Player message.
	user.SendText(fmt.Sprintf(
		`You sever the bond with <ansi fg="mobname">%s</ansi>.`,
		compName,
	))
	user.SendText(fmt.Sprintf(
		`<ansi fg="mobname">%s</ansi> turns on you with fury!`,
		compName,
	))

	// Room message (exclude the dismissing player — they already saw it).
	room.SendTextVisual(
		fmt.Sprintf(
			`<ansi fg="username">%s</ansi> dismisses `+
				`<ansi fg="mobname">%s</ansi>!`,
			user.Character.Name, compName,
		),
		user.UserId,
	)
	room.SendTextVisual(
		fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> turns hostile!`,
			compName,
		),
		user.UserId,
	)

	// Summoned, conjured, and raised companions should not persist in the
	// world indefinitely — queue a 5-minute delayed despawn so they expire
	// naturally rather than wandering as a permanent hostile.
	switch sourceType {
	case characters.CompanionSummoned,
		characters.CompanionConjured,
		characters.CompanionRaised:
		mob.Command("despawn", 300)
	}

	return true, nil
}
