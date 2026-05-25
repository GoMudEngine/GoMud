package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// Stand is the mob-side counterpart to the player stand command. Closes the
// chunk 3.3 T5 parity gap. Cancels Sleeping if present and transitions the
// position FSM from Prone/Supine to Standing.
//
// Unlike the player version, no stamina cost is applied — mobs are not
// expected to be stamina-bound for routine position changes (combat-driven
// position changes are tracked separately).
func Stand(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Cancel Sleeping FIRST so a standing-but-sleeping mob wakes regardless
	// of position-state. Mirrors usercommands/stand.go.
	if mob.Character.HasBuffFlag(buffs.Sleeping) {
		mob.Character.CancelBuffsWithFlag(buffs.Sleeping)
		mobs.OnSleeperWoken(&mob.Character)
	}

	// Position-state bail: already standing → no-op.
	if !mob.Character.IsProne() && !mob.Character.IsSupine() {
		return true, nil
	}

	// Fire the FSM transition. Prone→Standing and Supine→Standing are
	// always valid edges; a failure here is genuinely unexpected.
	if err := mob.Character.Position.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerStandCommand}); err != nil {
		mudlog.Warn("mob Stand: TransitionToStanding failed", "mobInstance", mob.InstanceId, "err", err)
		return true, nil
	}

	room.SendTextVisual(messaging.CategoryMobEmote,
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> struggles to its feet.`, mob.Character.Name),
		0,
	)

	return true, nil
}
