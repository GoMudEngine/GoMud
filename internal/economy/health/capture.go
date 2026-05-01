package health

import (
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CaptureSnapshot walks every live shop, caravan leader, and forager
// and produces a Snapshot suitable for serialization or scoring.
//
// Caravans and foragers are populated by separate helpers (see
// captureCaravans, captureForagers in this file).
func CaptureSnapshot() Snapshot {
	now := time.Now().UTC()
	snap := Snapshot{
		Timestamp: now.Format(time.RFC3339),
		UnixTs:    now.Unix(),
		Round:     util.GetRoundCount(),
	}
	snap.Shops = captureShops()
	snap.Caravans = captureCaravans()
	snap.Foragers = captureForagers()
	return snap
}

func captureShops() []ShopSnapshot {
	all := shops.AllShops()
	out := make([]ShopSnapshot, 0, len(all))
	for _, inv := range all {
		ss := ShopSnapshot{
			Zone:             inv.Zone,
			MobId:            inv.MobId,
			RoomId:           inv.RoomId,
			CraftSupport:     inv.CraftSupport,
			Gold:             inv.Gold,
			StartingGold:     inv.StartingGold,
			LastRestockRound: inv.LastRestock,
			Stock:            make([]StockSnapshot, 0, len(inv.Stock)),
			Name:             lookupShopMobName(inv.MobId, inv.RoomId),
		}
		for _, e := range inv.Stock {
			ss.Stock = append(ss.Stock, StockSnapshot{
				ItemId:     e.ItemId,
				Bucket:     economy.BucketFor(e.ItemId),
				Current:    e.Current,
				Max:        e.MaxStock,
				RestockQty: e.RestockQty,
			})
		}
		out = append(out, ss)
	}
	return out
}

// captureCaravans walks every live mob instance and emits one
// CaravanSnapshot per mob whose BTreeState has a non-empty
// "caravan_state" key (the convention used by actions_caravan.go).
// Cargo is read from the wagon mob co-located in the same room.
func captureCaravans() []CaravanSnapshot {
	out := []CaravanSnapshot{}
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		bs, ok := m.BTreeState.(*behaviortree.BehaviorState)
		if !ok || bs == nil {
			continue
		}
		stateName := bs.GetString("caravan_state")
		if stateName == "" {
			continue
		}

		startedRound, _ := strconv.ParseUint(bs.GetString("caravan_state_started_round"), 10, 64)

		cs := CaravanSnapshot{
			InstId:            instId,
			Name:              m.Character.Name,
			State:             stateName,
			StateEnteredRound: startedRound,
			RoomId:            m.Character.RoomId,
			CargoByBucket:     map[string]int{},
		}

		// Wagon co-located with the leader is the cargo source.
		// Both CargoWeight and CargoCapacity are pounds — that's what
		// actually limits the wagon, and the dashboard's "is the
		// wagon filling up?" question reads honestly as a weight ratio.
		wagon := caravan.FindWagonInRoom(m.Character.RoomId)
		if wagon != nil {
			cs.CargoWeight = int(wagon.Character.GetCarriedWeight())
			cs.CargoCapacity = int(wagon.Character.CarryCapacity())
			for _, it := range wagon.Character.Items {
				bucket := economy.BucketFor(it.ItemId)
				if bucket == "" {
					continue
				}
				w := int(it.GetSpec().GetWeight())
				if w > 0 {
					cs.CargoByBucket[bucket] += w
				}
			}
		}

		out = append(out, cs)
	}
	return out
}

// captureForagers walks every live mob instance and emits one
// ForagerSnapshot per mob whose BTreeState has a non-empty
// "forager_state" key. Foragers have no separate wagon — cargo lives
// on the forager's own Character.Items (plus ComponentItems and
// PotionItems if a component bag or bandolier is equipped).
//
// After the live pass, a placeholder row (State="(not active)") is
// appended for each forager profile in forager.AllProfiles() whose
// mob isn't currently spawned, so the dashboard always shows the full
// set of 3 foragers.
func captureForagers() []ForagerSnapshot {
	out := []ForagerSnapshot{}
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		bs, ok := m.BTreeState.(*behaviortree.BehaviorState)
		if !ok || bs == nil {
			continue
		}
		stateName := bs.GetString("forager_state")
		if stateName == "" {
			continue
		}

		startedRound, _ := strconv.ParseUint(bs.GetString("forager_state_started_round"), 10, 64)

		fs := ForagerSnapshot{
			InstId:            instId,
			MobId:             int(m.MobId),
			Name:              m.Character.Name,
			Territory:         territoryFor(int(m.MobId)),
			State:             stateName,
			StateEnteredRound: startedRound,
			RoomId:            m.Character.RoomId,
			CargoByBucket:     map[string]int{},
			CargoWeight:       int(m.Character.GetCarriedWeight()),
			CargoCapacity:     int(m.Character.CarryCapacity()),
		}
		// Per-bucket: sum item weights across all inventory lists
		// (backpack + component bag + bandolier). Skip items with no
		// bucket or zero weight. Same convention as captureCaravans.
		// Note: captureCaravans only walks wagon.Character.Items because
		// wagons never equip bags; foragers can (e.g. Halix wields a spear).
		inventories := [][]items.Item{m.Character.Items, m.Character.ComponentItems, m.Character.PotionItems}
		for _, list := range inventories {
			for _, it := range list {
				bucket := economy.BucketFor(it.ItemId)
				if bucket == "" {
					continue
				}
				w := int(it.GetSpec().GetWeight())
				if w > 0 {
					fs.CargoByBucket[bucket] += w
				}
			}
		}
		out = append(out, fs)
	}

	// Emit placeholder rows for profiles whose mob isn't currently live.
	// This ensures the dashboard always shows all 3 foragers.
	for _, p := range forager.AllProfiles() {
		active := false
		for i := range out {
			if out[i].MobId == p.MobId {
				active = true
				break
			}
		}
		if !active {
			out = append(out, ForagerSnapshot{
				InstId:        0,
				MobId:         p.MobId,
				Name:          p.Name,
				Territory:     territoryFor(p.MobId),
				State:         "(not active)",
				RoomId:        p.SanctuaryRoom,
				CargoByBucket: map[string]int{},
			})
		}
	}

	return out
}

// territoryFor returns the stable string label for a forager's
// territory, derived from forager.ProfileFor(mobId).Kind. Returns ""
// for non-foragers or unrecognized profiles.
func territoryFor(mobId int) string {
	p := forager.ProfileFor(mobId)
	if p == nil {
		return ""
	}
	switch p.Kind {
	case forager.KindMarsh:
		return "stillwater_marsh"
	case forager.KindSteppe:
		return "thornwall_steppe"
	case forager.KindFernway:
		return "fernway"
	}
	return ""
}

// lookupShopMobName resolves a shop's display name by walking live
// mob instances for one matching mobId+roomId. Returns "" if the mob
// is not currently spawned (shouldn't happen for registered shops, but
// gracefully degrades).
func lookupShopMobName(mobId, roomId int) string {
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		if int(m.MobId) == mobId && m.HomeRoomId == roomId {
			return m.Character.Name
		}
	}
	return ""
}
