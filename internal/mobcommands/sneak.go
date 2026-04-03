package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Sneak(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Can't sneak while in combat
	if mob.Character.Aggro != nil {
		return true, nil
	}

	// Already sneaking
	if mob.Character.HasBuffFlag(buffs.Hidden) {
		return true, nil
	}

	mob.AddBuff(9, `skill`)

	return true, nil
}
