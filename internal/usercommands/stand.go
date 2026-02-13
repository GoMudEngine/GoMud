package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Stand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Check if character is prone
	if !user.Character.Prone {
		user.SendText("You're already standing.")
		return true, nil
	}

	cfg := configs.GetGamePlayConfig()

	// Calculate stamina cost and requirement
	staminaCost := int(float64(user.Character.StaminaMax.Value) * float64(cfg.StandStaminaCost))
	minStamina := int(float64(user.Character.StaminaMax.Value) * float64(cfg.StandMinStamina))

	// Check if player has enough stamina
	if user.Character.Stamina < minStamina {
		needed := minStamina - user.Character.Stamina
		user.SendText(fmt.Sprintf("You're too exhausted to stand! (need %d more stamina)", needed))
		return true, nil
	}

	// Deduct stamina
	user.Character.Stamina -= staminaCost
	if user.Character.Stamina < 0 {
		user.Character.Stamina = 0
	}

	// Remove prone status (bypasses minimum duration)
	user.Character.Prone = false
	user.Character.ProneRoundsRemaining = 0

	// Send messages
	user.SendText("You struggle to your feet!")

	room.SendText(
		fmt.Sprintf(`<ansi fg="username">%s</ansi> struggles to their feet.`, user.Character.Name),
		user.UserId,
	)

	// If in combat, standing costs the current round
	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
