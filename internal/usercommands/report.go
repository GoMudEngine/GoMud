package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Report(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	c := user.Character

	hpBar := users.RenderVitalBar(c.Health, c.HealthMax.Value,
		c.GetPoolReservation("health", c.HealthMax.Value))
	spBar := users.RenderVitalBar(c.Stamina, c.StaminaMax.Value,
		c.GetPoolReservation("stamina", c.StaminaMax.Value))
	cpBar := users.RenderVitalBar(c.Conviction, c.ConvictionMax.Value,
		c.GetPoolReservation("conviction", c.ConvictionMax.Value))

	barText := fmt.Sprintf(`HP %s  SP %s  CP %s`, hpBar, spBar, cpBar)

	rest = strings.TrimSpace(strings.ToLower(rest))

	// Mode 1: party rep — send to party chat
	if rest == "party" {
		currentParty := parties.Get(user.UserId)
		if currentParty == nil {
			user.SendText("You are not in a party.")
			return true, nil
		}

		for _, uId := range currentParty.GetMembers() {
			if uId == user.UserId {
				continue
			}
			if u := users.GetByUserId(uId); u != nil {
				u.SendText(fmt.Sprintf(
					`<ansi fg="magenta">(party)</ansi> <ansi fg="username">%s</ansi> reports: %s`,
					c.Name, barText))
			}
		}
		user.SendText(fmt.Sprintf(`<ansi fg="magenta">(party)</ansi> You report: %s`, barText))
		return true, nil
	}

	// Mode 2: rep <target> — whisper report to a specific player
	if rest == "me" || rest == "self" {
		user.SendText(fmt.Sprintf(`You check yourself: %s`, barText))
		return true, nil
	}

	if rest != "" {
		// Find target player in room
		target, err := actions.ResolveTargetActor(room, rest)
		if err == actions.ErrTargetVanished {
			user.SendText("They are no longer here.")
			return true, nil
		}
		if err != nil || !target.IsPlayer() {
			user.SendText(fmt.Sprintf(`You don't see "%s" here.`, rest))
			return true, nil
		}

		targetUser := target.(*actions.UserActor).User
		if targetUser.UserId == user.UserId {
			user.SendText(fmt.Sprintf(`You check yourself: %s`, barText))
			return true, nil
		}

		targetUser.SendText(fmt.Sprintf(
			`<ansi fg="whisper"><ansi fg="username">%s</ansi> reports to you: %s</ansi>`,
			c.Name, barText))
		user.SendText(fmt.Sprintf(
			`<ansi fg="whisper">You report to <ansi fg="username">%s</ansi>: %s</ansi>`,
			targetUser.Character.Name, barText))
		return true, nil
	}

	// Mode 3 (default): rep — broadcast to room
	user.SendText(fmt.Sprintf(`You report: %s`, barText))
	room.SendTextVisual(fmt.Sprintf(
		`<ansi fg="username">%s</ansi> reports: %s`,
		c.Name, barText), user.UserId)

	return true, nil
}
