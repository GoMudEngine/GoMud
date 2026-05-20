package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Sort(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	hasBag := user.Character.Equipment.ComponentBag.ItemId > 0
	hasBandolier := false
	if user.Character.Equipment.Belt.ItemId > 0 {
		beltSpec := user.Character.Equipment.Belt.GetSpec()
		hasBandolier = beltSpec.IsBandolier
	}

	if !hasBag && !hasBandolier {
		user.SendText(messaging.CategorySystem, `You don't have a component bag or bandolier equipped.`)
		return true, nil
	}

	materialsMoved := 0
	potionsMoved := 0

	if hasBag {
		materialsMoved = user.Character.SortComponentItems()
	}
	if hasBandolier {
		potionsMoved = user.Character.SortPotionItems()
	}

	if materialsMoved == 0 && potionsMoved == 0 {
		user.SendText(messaging.CategorySystem, `No items found to sort.`)
		return true, nil
	}

	parts := make([]string, 0, 2)
	if materialsMoved > 0 {
		parts = append(parts, fmt.Sprintf(
			`%d material(s) into your %s`,
			materialsMoved, user.Character.Equipment.ComponentBag.DisplayName()))
	}
	if potionsMoved > 0 {
		parts = append(parts, fmt.Sprintf(
			`%d potion(s) into your %s`,
			potionsMoved, user.Character.Equipment.Belt.DisplayName()))
	}

	user.SendText(messaging.CategorySystem, fmt.Sprintf(
		`<ansi fg="green">Sorted %s.</ansi>`,
		strings.Join(parts, " and ")))

	return true, nil
}
