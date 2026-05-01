package health

import (
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/economy"
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
		wagon := caravan.FindWagonInRoom(m.Character.RoomId)
		if wagon != nil {
			cs.CargoCapacity = int(wagon.Character.CarryCapacity())
			for _, it := range wagon.Character.Items {
				bucket := economy.BucketFor(it.ItemId)
				cs.CargoCount++
				if bucket != "" {
					cs.CargoByBucket[bucket]++
				}
			}
		}

		out = append(out, cs)
	}
	return out
}

// captureForagers is implemented in Task 7.
func captureForagers() []ForagerSnapshot { return nil }

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
