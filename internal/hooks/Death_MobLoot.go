package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// wireMobDeathLoot subscribes to Life Alive→Dead transitions on
// mob characters and drops loot + creates a corpse. Player actor
// transitions are skipped (MobInstanceId == 0 guard).
//
// NOTE (Chunk-2): This observer is wired but dormant until
// Task 10 routes mob death paths through Life.TransitionToDead.
// The existing inline loot-drop + corpse logic in
// internal/mobcommands/suicide.go (lines 180-250) is the live
// path today; it will be removed when Task 10 lands.
func wireMobDeathLoot(c *characters.Character) {
	c.Life.Inner().AfterTransition("mob_death_loot",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			// Only fire for mob characters.
			if c.MobInstanceId == 0 {
				return
			}
			m := mobs.GetInstance(c.MobInstanceId)
			if m == nil {
				return
			}
			room := rooms.LoadRoom(m.Character.RoomId)
			if room == nil {
				return
			}
			dropMobLootAndSetCorpse(m, room)
		})
}

// dropMobLootAndSetCorpse is the observer-side port of the
// loot-drop + corpse-creation block from suicide.go (lines
// 180-250). When Task 10 thins suicide.go, the inline block
// there is removed and this function becomes the sole call site.
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
			room.SendTextVisual(msg)
			room.AddItem(item, false)
			lootDropped = true
		}

		// Equipped items: gate on mob.ItemDropChance unless the item
		// has its own per-instance DropChance.
		for _, item := range m.Character.Equipment.GetAllItems() {
			if !item.ShouldDrop(m.ItemDropChance) {
				continue
			}
			msg := fmt.Sprintf(
				`<ansi fg="item">%s</ansi> drops to the ground.`,
				item.DisplayName(),
			)
			room.SendTextVisual(msg)
			room.AddItem(item, false)
			lootDropped = true
		}

		if m.Character.Gold > 0 {
			msg := fmt.Sprintf(
				`<ansi fg="yellow-bold">%d gold</ansi> drops to the ground.`,
				m.Character.Gold,
			)
			room.SendTextVisual(msg)
			room.Gold += m.Character.Gold
			lootDropped = true
		}

		// Dark-room fallback sound for loot drops.
		if lootDropped && room.GetVisibility() < 1 {
			room.SendText(`You hear something clatter to the ground.`)
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

func init() {
	characters.OnCharacterCreated(wireMobDeathLoot)
}
