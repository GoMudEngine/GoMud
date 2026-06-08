package usercommands

import (
	"slices"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Conditions(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	type buffInfo struct {
		Name        string
		Description string
		RoundsLeft  int
		PermaBuff   bool
	}

	afflictions := []buffInfo{}

	charBuffs := user.Character.GetBuffs()
	for _, buff := range charBuffs {

		spec := buffs.GetBuffSpec(buff.BuffId)
		if spec == nil {
			continue
		}

		// Buffs that hide you (Empathic Shroud, the Hidden stealth state, etc.)
		// are deliberately NOT shown: if you can't know who spotted you, you
		// shouldn't be told you're hidden in the first place. Mirrors the
		// no-end-message design on the hidden buff itself.
		if slices.Contains(spec.Flags, buffs.Hidden) {
			continue
		}

		roundsLeft, _ := buffs.GetDurations(buff, spec)

		newAffliction := buffInfo{
			Name:        spec.Name,
			Description: spec.Description,
			RoundsLeft:  roundsLeft,
			PermaBuff:   buff.PermaBuff,
		}
		newAffliction.Name, newAffliction.Description = spec.VisibleNameDesc()

		afflictions = append(afflictions, newAffliction)
	}

	// Stage 9.8: Append active combat conditions
	for _, cond := range user.Character.Conditions {
		afflictions = append(afflictions, buffInfo{
			Name:        cond.Type.DisplayName(),
			Description: cond.Type.Description(),
			RoundsLeft:  cond.Duration,
			PermaBuff:   cond.Duration == 0,
		})
	}

	tplTxt, _ := templates.Process("character/conditions", afflictions, user.UserId)
	user.SendText(messaging.CategorySystem, tplTxt)

	return true, nil
}
