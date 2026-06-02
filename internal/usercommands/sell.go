package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Sell(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		user.SendText(messaging.CategorySystem, "What would you like to sell?")
		return true, nil
	}

	// ── Parse optional leading quantity ───────────────────────────────────
	// Supported forms:
	//   sell 5 iron-ore        → quantity=5, itemName="iron-ore"
	//   sell all iron-ore      → quantity=unlimited (math.MaxInt), name="iron-ore"
	//   sell all.iron-ore      → diku-style alias for the above
	//   sell iron-ore          → quantity=1 (existing single-sell)
	//
	// "sell all" with no item name is rejected.
	const unlimited = int(^uint(0) >> 1) // math.MaxInt without import

	quantity := 1
	itemName := rest

	lower := strings.ToLower(strings.TrimSpace(rest))

	// Detect "all.name" prefix (diku-style).
	if strings.HasPrefix(lower, "all.") {
		suffix := strings.TrimPrefix(lower, "all.")
		if strings.TrimSpace(suffix) == "" {
			user.SendText(messaging.CategorySystem, "Specify what you want to sell. (e.g. <ansi fg=\"command\">sell all iron-ore</ansi>)")
			return true, nil
		}
		quantity = unlimited
		// Preserve original casing: skip past "all." (4 chars).
		itemName = rest[4:]
	} else {
		// Check for leading word: a number or the word "all".
		args := strings.SplitN(strings.TrimSpace(rest), " ", 2)
		if len(args) == 2 {
			firstWord := strings.ToLower(args[0])
			if firstWord == "all" {
				if strings.TrimSpace(args[1]) == "" {
					user.SendText(messaging.CategorySystem, "Specify what you want to sell. (e.g. <ansi fg=\"command\">sell all iron-ore</ansi>)")
					return true, nil
				}
				quantity = unlimited
				itemName = args[1]
			} else if n, err := strconv.Atoi(args[0]); err == nil && n >= 1 {
				quantity = n
				itemName = args[1]
			}
		} else if strings.ToLower(strings.TrimSpace(rest)) == "all" {
			// bare "sell all" with no item name
			user.SendText(messaging.CategorySystem, "Specify what you want to sell. (e.g. <ansi fg=\"command\">sell all iron-ore</ansi>)")
			return true, nil
		}
	}

	// ── Delegate the sale to the shared actions.Sell ──────────────────────
	actor := &actions.UserActor{User: user, Room: room}
	res := actions.Sell(actor, actions.SellOptions{ItemName: itemName, Quantity: quantity})

	switch res.Reason {
	case actions.SellStopNoItem:
		return true, nil // actions.Sell already sent "You don't have that item."
	case actions.SellStopNoMerchant:
		user.SendText(messaging.CategorySystem, "There's no merchant here.")
		return true, nil
	}

	if res.Sold == 0 {
		return true, nil // a merchant refusal was already spoken synchronously
	}

	displayName := res.LastItemName
	if res.Sold == 1 {
		user.EventLog.Add(`shop`, fmt.Sprintf(`Sold your <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>`, displayName, res.TotalGold))
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You sell a <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>.`, displayName, res.TotalGold))
		room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> sells a <ansi fg="itemname">%s</ansi>.`, user.Character.Name, displayName), user.UserId)
	} else {
		pluralName := displayName + "s"
		user.EventLog.Add(`shop`, fmt.Sprintf(`Sold %d <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>`, res.Sold, pluralName, res.TotalGold))
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You sell %d <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>.`, res.Sold, pluralName, res.TotalGold))
		room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> sells %d <ansi fg="itemname">%s</ansi>.`, user.Character.Name, res.Sold, pluralName), user.UserId)
	}

	if res.Reason == actions.SellStopMerchantBroke {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="yellow">Sold %d before the merchant ran out of gold.</ansi>`, res.Sold))
	}

	return true, nil
}
