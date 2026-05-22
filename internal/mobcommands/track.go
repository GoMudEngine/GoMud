package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Track(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	opts := actions.TrackOptions{TargetNoun: rest}
	if rest == "stop" || rest == "clear" {
		opts = actions.TrackOptions{CancelTracking: true}
	}
	actor := &actions.MobActor{Mob: mob, Room: room}
	_ = actions.Track(actor, opts)
	return true, nil
}
