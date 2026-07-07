package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Loot takes everything the looter is entitled to from a corpse:
//
//	loot <corpse>        e.g. "loot wolf corpse", "loot wolf"
//	loot from <corpse>   (leading "from" tolerated)
func Loot(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Can't loot if you can't see.
	if room.GetVisibility() < 1 && !user.Character.HasFlagFromAnySource(buffs.NightVision) {
		user.SendText(messaging.CategorySystem, "You can't see anything to loot!")
		return true, nil
	}

	rest = strings.TrimSpace(rest)
	if rest == `` {
		user.SendText(messaging.CategorySystem, "Loot what?")
		return true, nil
	}

	// Tolerate a leading "from" ("loot from wolf corpse").
	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	if len(args) > 0 && args[0] == "from" {
		rest = strings.Join(args[1:], " ")
	}

	corpseIdx := room.FindCorpseIndex(rest)
	if corpseIdx < 0 {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("You don't see a %s to loot.", rest))
		return true, nil
	}

	lootCorpseAll(user, room, corpseIdx)

	return true, nil
}

// lootCorpseAll transfers every item and all gold the looter is entitled to
// from room.Corpses[corpseIdx] into the looter's inventory. It operates on a
// POINTER into the slice so the loot container mutations persist.
//
// Shared by the `loot` command; Task 10 (round-robin distribution) reuses it.
func lootCorpseAll(user *users.UserRecord, room *rooms.Room, corpseIdx int) {

	if corpseIdx < 0 || corpseIdx >= len(room.Corpses) {
		return
	}

	corpse := &room.Corpses[corpseIdx]

	// Ownership/mode gate — Task 3 stub always allows; Tasks 8/10 gate.
	if !canLootCorpse(user, corpse) {
		user.SendText(messaging.CategorySystem, `This isn't your kill.`)
		return
	}

	if !corpse.HasLoot() {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`There's nothing to loot from the <ansi fg="mob-corpse">%s</ansi>.`, corpse.DisplayName()))
		return
	}

	tookSomething := false

	// Items first. Copy the slice so removal during iteration is safe.
	itemCopies := append([]items.Item{}, corpse.Loot.Items...)
	for _, it := range itemCopies {
		if user.Character.StoreItem(it) {
			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   it,
				Gained: true,
			})
			corpse.Loot.RemoveItem(it)

			user.SendText(messaging.CategorySystem,
				fmt.Sprintf(`You take the <ansi fg="itemname">%s</ansi> from the <ansi fg="mob-corpse">%s</ansi>.`, it.DisplayName(), corpse.DisplayName()),
			)
			tookSomething = true
		} else {
			user.SendText(messaging.CategorySystem,
				fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, it.DisplayName()),
			)
			break
		}
	}

	// Then gold.
	if corpse.Loot.Gold > 0 {
		amt := corpse.Loot.Gold
		corpse.Loot.Gold -= amt
		grantCorpseGold(user, amt)

		user.SendText(messaging.CategorySystem,
			fmt.Sprintf(`You take <ansi fg="gold">%d gold</ansi> from the <ansi fg="mob-corpse">%s</ansi>.`, amt, corpse.DisplayName()),
		)
		tookSomething = true
	}

	if tookSomething {
		user.Character.CancelBuffsWithFlag(buffs.Hidden) // No longer sneaking
		room.SendTextVisual(messaging.CategoryLoot,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> loots the <ansi fg="mob-corpse">%s</ansi>.`, user.Character.Name, corpse.DisplayName()),
			user.UserId,
		)
		sendEncumbranceWarning(user)
	}
}
