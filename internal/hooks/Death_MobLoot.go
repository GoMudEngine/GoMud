package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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

	if !m.Character.HasBuffFlag(buffs.PermaGear) {

		lootDropped := false

		// Carried items: 100% base drop chance (per-item DropChance
		// still applies via ShouldDrop).
		for _, item := range m.Character.Items {
			if !item.ShouldDrop(100) {
				continue
			}
			msg := fmt.Sprintf(
				`<ansi fg="item">%s</ansi> drops to the ground.`,
				item.DisplayName(),
			)
			room.SendTextVisual(messaging.CategoryLoot, msg)
			room.AddItem(item, false)
			lootDropped = true
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
			msg := fmt.Sprintf(
				`<ansi fg="item">%s</ansi> drops to the ground.`,
				item.DisplayName(),
			)
			room.SendTextVisual(messaging.CategoryLoot, msg)
			room.AddItem(item, false)
			lootDropped = true
		}

		if m.Character.Gold > 0 {
			msg := fmt.Sprintf(
				`<ansi fg="yellow-bold">%d gold</ansi> drops to the ground.`,
				m.Character.Gold,
			)
			room.SendTextVisual(messaging.CategoryLoot, msg)
			room.Gold += m.Character.Gold
			lootDropped = true
		}

		// Dark-room fallback sound for loot drops.
		if lootDropped && room.GetVisibility() < 1 {
			room.SendText(messaging.CategoryLoot, `You hear something clatter to the ground.`)
		}
	}

	config := configs.GetGamePlayConfig()
	if config.Death.CorpsesEnabled {
		room.AddCorpse(rooms.Corpse{
			MobId:             int(m.MobId),
			Character:         m.Character,
			RoundCreated:      currentRound,
			WasCharmed:        m.Character.IsCharmed() || m.Character.EverCharmed,
			CorpseName:        m.CorpseName,
			CorpseDescription: m.CorpseDescription,
		})
	}
}

