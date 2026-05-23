package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// PacifismAura is a thin wrapper over actions.TriggerPacifismAura.
// All gates, AoE de-aggro logic, self-penalty, and player messages live in the action.
func PacifismAura(rest string, user *users.UserRecord, room *rooms.Room,
	flags events.EventFlag) (bool, error) {

	actor := actions.NewUserActorInRoom(user, room)
	_ = actions.TriggerPacifismAura(actor, actions.MutationOpts{})
	return true, nil
}
