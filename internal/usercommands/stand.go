package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Stand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Chunk 4b W7: gate on the new Position FSM (Prone or Supine).
	if !user.Character.IsProne() && !user.Character.IsSupine() {
		user.SendTextLegacy("You're already standing.")
		return true, nil
	}

	cfg := configs.GetBalanceConfig()

	// Calculate stamina cost and requirement
	staminaCost := int(float64(user.Character.StaminaMax.Value) * float64(cfg.StandStaminaCost))
	minStamina := int(float64(user.Character.StaminaMax.Value) * float64(cfg.StandMinStamina))

	// Check if player has enough stamina
	if user.Character.Stamina < minStamina {
		needed := minStamina - user.Character.Stamina
		user.SendTextLegacy(fmt.Sprintf("You're too exhausted to stand! (need %d more stamina)", needed))
		return true, nil
	}

	// Fire the FSM transition first — bail without charging stamina if it fails
	// (shouldn't happen since Prone→Standing and Supine→Standing are valid edges).
	if err := user.Character.Position.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerStandCommand}); err != nil {
		mudlog.Warn("Stand: TransitionToStanding failed", "user", user.UserId, "err", err)
		user.SendTextLegacy("Something prevents you from standing.")
		return true, nil
	}

	// Deduct stamina
	user.Character.Stamina -= staminaCost
	if user.Character.Stamina < 0 {
		user.Character.Stamina = 0
	}

	// Send messages
	user.SendTextLegacy("You struggle to your feet!")

	room.SendTextVisualLegacy(
		fmt.Sprintf(`<ansi fg="username">%s</ansi> struggles to their feet.`, user.Character.Name),
		user.UserId,
	)

	// If in combat, standing costs the current round
	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
