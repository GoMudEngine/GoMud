package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/parties"
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

	// "loot pass <corpse>" — reassign your round-robin share to the next member.
	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	if len(args) > 0 && args[0] == "pass" {
		return lootPassCorpse(strings.TrimSpace(strings.Join(args[1:], " ")), user, room)
	}

	// Tolerate a leading "from" ("loot from wolf corpse").
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

	// Ownership/mode gate: enforces kill ownership and party loot mode.
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
	now := util.GetRoundCount()
	itemCopies := append([]items.Item{}, corpse.Loot.Items...)
	for _, it := range itemCopies {
		// Loot-mode gate: only take items reserved to this user (or unreserved).
		if !corpse.CanTakeItem(it.UUID.String(), user.UserId, now) {
			continue
		}
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

// lootPassCorpse reassigns the passing user's still-present round-robin loot on
// the named corpse to the NEXT member in the party rotation. Only meaningful in
// round-robin mode; ffa/leader-hold have nothing to pass.
func lootPassCorpse(rest string, user *users.UserRecord, room *rooms.Room) (bool, error) {

	if rest == `` {
		user.SendText(messaging.CategorySystem, "Pass your loot on which corpse?")
		return true, nil
	}

	corpseIdx := room.FindCorpseIndex(rest)
	if corpseIdx < 0 {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("You don't see a %s to loot.", rest))
		return true, nil
	}

	corpse := &room.Corpses[corpseIdx]

	if corpse.LootMode != "roundrobin" || len(corpse.RRAssignee) == 0 {
		user.SendText(messaging.CategorySystem, "There's nothing to pass — loot passing only applies to round-robin looting.")
		return true, nil
	}

	party := parties.Get(user.UserId)
	if party == nil {
		user.SendText(messaging.CategorySystem, "There's nothing to pass.")
		return true, nil
	}

	// Reconstruct the rotation: party join order filtered to the corpse owners.
	ownerSet := make(map[int]struct{}, len(corpse.OwnerUserIds))
	for _, id := range corpse.OwnerUserIds {
		ownerSet[id] = struct{}{}
	}
	members := make([]int, 0, len(party.UserIds))
	for _, id := range party.UserIds {
		if _, ok := ownerSet[id]; ok {
			members = append(members, id)
		}
	}
	if len(members) < 2 {
		user.SendText(messaging.CategorySystem, "There's no one to pass your share to.")
		return true, nil
	}

	selfIdx := -1
	for i, id := range members {
		if id == user.UserId {
			selfIdx = i
			break
		}
	}
	if selfIdx < 0 {
		user.SendText(messaging.CategorySystem, "There's nothing to pass.")
		return true, nil
	}

	next := members[(selfIdx+1)%len(members)]

	reassigned := 0
	for _, it := range corpse.Loot.Items {
		uid := it.UUID.String()
		if owner, ok := corpse.RRAssignee[uid]; ok && owner == user.UserId {
			corpse.RRAssignee[uid] = next
			reassigned++
		}
	}

	if reassigned == 0 {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You have no share to pass on the <ansi fg="mob-corpse">%s</ansi>.`, corpse.DisplayName()))
		return true, nil
	}

	nextName := "the next member"
	if u := users.GetByUserId(next); u != nil {
		nextName = u.Character.Name
	}
	user.SendText(messaging.CategorySystem,
		fmt.Sprintf(`You pass your share of the <ansi fg="mob-corpse">%s</ansi> to <ansi fg="username">%s</ansi>.`, corpse.DisplayName(), nextName),
	)

	return true, nil
}
