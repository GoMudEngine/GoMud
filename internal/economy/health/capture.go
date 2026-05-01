package health

import (
	"time"

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

// captureCaravans is implemented in Task 6.
func captureCaravans() []CaravanSnapshot { return nil }

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
