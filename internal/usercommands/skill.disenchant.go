package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/enchantments"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Disenchant handles the `disenchant <item>` command (Stage 31.6).
// Strips a Chrysalis enchantment from an item at an enchanting circle.
func Disenchant(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)
	if rest == "" {
		user.SendText(messaging.CategorySystem, `<ansi fg="command">disenchant <item></ansi> — strip a Chrysalis enchantment from an item.`)
		return true, nil
	}

	// Must be at an enchanting circle
	if room.Station != "enchanting_circle" {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You need to be at an enchanting circle to strip an enchantment.</ansi>`)
		return true, nil
	}

	// Find item in inventory
	pMatch, fMatch := items.FindMatchIn(rest, user.Character.Items...)
	targetItem := fMatch
	if targetItem.ItemId < 1 {
		targetItem = pMatch
	}
	if targetItem.ItemId < 1 {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">You don't have "%s" in your inventory.</ansi>`, rest))
		return true, nil
	}

	// Must have an enchantment
	if !targetItem.HasChrysalisEnchantment() {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">That item has no Chrysalis enchantment to strip.</ansi>`)
		return true, nil
	}

	// Record reservation before stripping
	reservePool := targetItem.ReservePool
	reservePct := enchantments.GetTierReservePct(targetItem.EnchantType, targetItem.EnchantTier, targetItem.GetSpec().Hands)

	// Find the actual item in the slice and strip it
	for i := range user.Character.Items {
		if user.Character.Items[i].ItemId == targetItem.ItemId &&
			user.Character.Items[i].UUID == targetItem.UUID {
			enchantments.StripEnchantment(&user.Character.Items[i])
			break
		}
	}

	// Apply withdrawal condition
	bal := configs.GetBalanceConfig()
	penaltyRounds := int(bal.EnchantRemovalPenaltyRounds)

	// Magnitude stores the pool max reduction as a fraction
	user.Character.AddCondition(
		characters.ConditionEnchantWithdrawal,
		penaltyRounds,
		reservePct,
		reservePool,
	)

	user.SendText(messaging.CategorySystem, `<ansi fg="magenta">You pry the Chrysalis free. It comes away screaming — a ` +
		`soundless wail that reverberates through your bones. The item ` +
		`falls silent, stripped of its living power.</ansi>`)
	user.SendText(messaging.CategorySystem, `<ansi fg="red">A sudden emptiness floods through you. Your body aches ` +
		`for the connection it has lost. The withdrawal will pass... ` +
		`in time.</ansi>`)

	return true, nil
}
