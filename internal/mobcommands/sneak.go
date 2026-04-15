package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Sneak(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	result := actions.ExecuteSneak(&actions.MobActor{Mob: mob, Room: room})

	if result.Success {
		// Track skill use so mobs can progress skullduggery like players.
		mob.Character.OnSkillUse("skullduggery", 0)
	}

	return true, nil
}
