package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
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

	reportLines := fmt.Sprintf(
		`<ansi fg="username">%s</ansi> reports: `+
			`HP %s  `+
			`SP %s  `+
			`CP %s`,
		c.Name, hpBar, spBar, cpBar)

	// Show to self
	selfLines := fmt.Sprintf(
		`You report: `+
			`HP %s  `+
			`SP %s  `+
			`CP %s`,
		hpBar, spBar, cpBar)

	user.SendText(selfLines)
	room.SendText(reportLines, user.UserId)

	return true, nil
}
