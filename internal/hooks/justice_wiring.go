package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/justice"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// init wires justice's guard-speech seam to the actions-based broadcaster. It
// lives in hooks (not justice) so package justice imports no actions, keeping
// the actions->justice direction (theft bounty firing) cycle-free.
func init() {
	justice.SetGuardSay(func(room *rooms.Room, mob *mobs.Mob, line string) {
		if mob == nil || room == nil {
			return
		}
		actor := &actions.MobActor{Mob: mob, Room: room}
		result := actions.Say(actor, line)
		room.SendText(messaging.CategorySpeech,
			actions.FormatSayText(mob.Character.Name, result.Text, false, "mobname", "saytext-mob"))
	})
}
