package usercommands

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
Skullduggery Skill
Level 3 - Defuse: attempt to disarm a trap on a locked exit or container.
Consumes a disarm kit. Success clears trapbuffids; failure triggers the trap.
*/
func Defuse(rest string, user *users.UserRecord,
	room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)
	if skillLevel < 1 {
		return false, nil
	}

	trapNoun := strings.TrimSpace(strings.ToLower(rest))

	actor := &actions.UserActor{User: user, Room: room}
	actions.Defuse(actor, actions.DefuseOptions{TargetNoun: trapNoun})

	// actions.Defuse emits all player-facing messages directly (rank gate,
	// combat gate, no-target, no-trap, no-kit, success, failure). The wrapper
	// has no additional messages to surface.

	return true, nil
}
