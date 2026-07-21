package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// resolveModTarget finds a player for a moderation action: first in the current
// room (fuzzy), then globally by character name. Returns nil if not found. This
// lets speech-moderation reach a player who is not in the admin's room.
func resolveModTarget(room *rooms.Room, rest string) *users.UserRecord {
	if target, err := actions.ResolveTargetActor(room, rest); err == nil && target.IsPlayer() {
		return target.(*actions.UserActor).User
	}
	return users.GetByCharacterName(rest)
}
