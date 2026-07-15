package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// dropMobLootAndSetCorpse drops a dead mob's carried + equipped
// items and creates a corpse in the room. Called from
// Death_MobInstanceCleanup.go BEFORE the mob instance is destroyed
// — the consolidated ordering avoids a registration-order bug where
// the instance cleanup (filename "I" < "L") would otherwise run
// first via init() ordering and wipe the instance before this
// function could read its data.
//
// Carried items always drop (100% base). Equipped items gate on
// mob.ItemDropChance. Gold always drops. Corpse is added when
// CorpsesEnabled config is set.
func dropMobLootAndSetCorpse(m *mobs.Mob, room *rooms.Room) {
	currentRound := util.GetRoundCount()

	// Build the loot container from the dead mob's carried + equipped items
	// and gold, applying the same gating rules as before. Where the loot ends
	// up (corpse container vs. room floor) is decided below by CorpsesEnabled.
	var loot rooms.Container

	if !m.Character.HasBuffFlag(buffs.PermaGear) {

		// Carried items: 100% base drop chance (per-item DropChance
		// still applies via ShouldDrop).
		for _, item := range m.Character.Items {
			if !item.ShouldDrop(100) {
				continue
			}
			item.Validate() // ensure a distinct UUID (round-robin keys loot by UUID)
			loot.AddItem(item)
		}

		// Equipped items: gate on mob.ItemDropChance unless the item
		// has its own per-instance DropChance.
		for _, item := range m.Character.Equipment.GetAllItems() {
			// NeverDrops (e.g. boss-only stat-boost gear like the Core
			// Guardian's Hull Plating / Core Matrix) is skipped entirely —
			// distinct from PermaGear, which also suppresses this mob's
			// carried Items + Gold. NeverDrops only touches equipped gear,
			// leaving the mob's intended carried-item loot untouched.
			if item.GetSpec().NeverDrops {
				continue
			}
			if !item.ShouldDrop(m.ItemDropChance) {
				continue
			}
			item.Validate() // ensure a distinct UUID (round-robin keys loot by UUID)
			loot.AddItem(item)
		}

		if m.Character.Gold > 0 {
			loot.Gold += m.Character.Gold
		}
	}

	// Auto-loot the mob's gold to the killer at death: a solo killer takes it
	// straight to purse, a party pools it (settled/split on membership change or
	// `party gold split`). Items are still looted manually, respecting loot mode.
	// Removing the collected gold from `loot` keeps it out of the corpse / off
	// the floor (no double-loot).
	if collected := autoLootGold(m.Character.PlayerDamage, loot.Gold); collected > 0 {
		loot.Gold -= collected
	}

	config := configs.GetGamePlayConfig()
	if config.Death.CorpsesEnabled {
		// Loot goes into the corpse container — nothing drops to the floor.
		owners := computeCorpseOwners(m.Character.PlayerDamage, room.RoomId)
		room.AddCorpse(rooms.Corpse{
			MobId:             int(m.MobId),
			Character:         m.Character,
			RoundCreated:      currentRound,
			WasCharmed:        m.Character.IsCharmed() || m.Character.EverCharmed,
			CorpseName:        m.CorpseName,
			CorpseDescription: m.CorpseDescription,
			Loot:              loot,
			OwnerUserIds:      owners,
			LootMode:          corpseLootMode(m.Character.PlayerDamage),
			RoundOwnedUntil:   lootTimeoutRound(currentRound, config.Death.CorpseLootTimeout.String()),
			RRAssignee:        assignCorpseLoot(loot, owners, m.Character.PlayerDamage),
		})
		return
	}

	// Corpses disabled — fall back to the legacy floor-drop behavior,
	// preserving the exact player-facing messages.
	lootDropped := false

	for _, item := range loot.Items {
		msg := fmt.Sprintf(
			`<ansi fg="item">%s</ansi> drops to the ground.`,
			item.DisplayName(),
		)
		room.SendTextVisual(messaging.CategoryLoot, msg)
		room.AddItem(item, false)
		lootDropped = true
	}

	if loot.Gold > 0 {
		msg := fmt.Sprintf(
			`<ansi fg="yellow-bold">%d gold</ansi> drops to the ground.`,
			loot.Gold,
		)
		room.SendTextVisual(messaging.CategoryLoot, msg)
		room.Gold += loot.Gold
		lootDropped = true
	}

	// Dark-room fallback sound for loot drops.
	if lootDropped && room.GetVisibility() < 1 {
		room.SendText(messaging.CategoryLoot, `You hear something clatter to the ground.`)
	}
}

// autoGoldPlan is the routing decision for a killed mob's gold: into the killer
// party's shared pool, or straight to a solo killer's purse.
type autoGoldPlan struct {
	toPartyPool bool
	userId      int // solo recipient (top damager) when !toPartyPool
	amount      int
}

// planAutoLootGold decides where a killed mob's gold goes. Pure over the damage
// map + the party registry: a killer in a party pools it; a solo killer takes it
// to purse. Returns a zero plan when no player killed the mob or there's no gold.
func planAutoLootGold(playerDamage map[int]int, gold int) autoGoldPlan {
	if gold <= 0 || len(playerDamage) == 0 {
		return autoGoldPlan{}
	}
	if killerParty(playerDamage) != nil {
		return autoGoldPlan{toPartyPool: true, amount: gold}
	}
	return autoGoldPlan{userId: topDamager(playerDamage), amount: gold}
}

// autoLootGold applies planAutoLootGold: pools to the killer party (notifying
// online members) or credits a solo killer's purse. Returns the amount actually
// collected (0 if the recipient is offline/unloaded, so the gold stays in the
// corpse rather than vanishing).
func autoLootGold(playerDamage map[int]int, gold int) int {
	plan := planAutoLootGold(playerDamage, gold)
	if plan.amount <= 0 {
		return 0
	}

	if plan.toPartyPool {
		p := killerParty(playerDamage)
		if p == nil {
			return 0
		}
		p.AddGold(plan.amount)
		// Tell just the killer their find pooled — messaging every member on
		// every kill would spam a grinding party. The pool pays out on the next
		// membership change or `party gold split`.
		if u := users.GetByUserId(topDamager(playerDamage)); u != nil {
			u.SendText(messaging.CategoryLoot,
				fmt.Sprintf(`<ansi fg="yellow-bold">%d gold</ansi> goes into the party pool.`, plan.amount))
		}
		return plan.amount
	}

	u := users.GetByUserId(plan.userId)
	if u == nil {
		return 0
	}
	u.Character.Gold += plan.amount
	events.AddToQueue(events.EquipmentChange{UserId: plan.userId, GoldChange: plan.amount})
	u.SendText(messaging.CategoryLoot,
		fmt.Sprintf(`You collect <ansi fg="yellow-bold">%d gold</ansi>.`, plan.amount))
	return plan.amount
}

