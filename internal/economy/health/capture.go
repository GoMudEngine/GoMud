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
	"github.com/GoMudEngine/GoMud/internal/mudlog"
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
	currentRound := util.GetRoundCount()
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
			Round:            currentRound,
			Stock:            make([]StockSnapshot, 0, len(inv.Stock)),
			Name:             lookupShopMobName(inv.MobId, inv.RoomId),
		}
		for _, e := range inv.Stock {
			ss.Stock = append(ss.Stock, StockSnapshot{
				ItemId:     e.ItemId,
				Bucket:     economy.BucketFor(e.ItemId),
				Tier:       getRarityTier(e.ItemId),
				Current:    e.Current,
				Max:        e.MaxStock,
				RestockQty: e.RestockQty,
			})
		}
		// Compute StockScore: sum(Current) / sum(MaxStock)
		total, capacity := 0, 0
		for _, e := range ss.Stock {
			total += e.Current
			capacity += e.Max
		}
		if capacity > 0 {
			ss.StockScore = float64(total) / float64(capacity)
		}

		// Copy new Phase-2 counters.
		ss.SalesCount = inv.SalesCount
		ss.BuysCount = inv.BuysCount
		ss.RestockCount = inv.RestockCount
		if len(inv.StockEvents) > 0 {
			ss.StockEvents = make(map[int][]StockEvent, len(inv.StockEvents))
			for k, v := range inv.StockEvents {
				cp := make([]StockEvent, len(v))
				for i, e := range v {
					cp[i] = StockEvent{
						DepletedRound: e.DepletedRound,
						RefilledRound: e.RefilledRound,
					}
				}
				ss.StockEvents[k] = cp
			}
		}
		if len(inv.CurrentDepletion) > 0 {
			ss.CurrentDepletion = make(map[int]uint64, len(inv.CurrentDepletion))
			for k, v := range inv.CurrentDepletion {
				ss.CurrentDepletion[k] = v
			}
		}

		// Phase-4: compute per-tier median TtR and currently-depleted count.
		ss.MedianTtRCommons = computeMedianTtRForTiers(ss, currentRound, true)
		ss.MedianTtRRares = computeMedianTtRForTiers(ss, currentRound, false)
		ss.CurrentlyDepletedCount = len(ss.CurrentDepletion)

		out = append(out, ss)
	}
	return out
}

// computeMedianTtRForTiers computes the median TtR (in rounds) from
// StockEvents for items whose rarity tier qualifies:
//   - commons=true:  tier >= 40 (tier-50 and tier-40 items)
//   - commons=false: tier <= 20 (tier-20 and tier-10 items; tier-30 excluded)
//
// Currently-depleted items contribute their ongoing duration as a
// partial event. Returns 0 if there are no qualifying events.
func computeMedianTtRForTiers(snap ShopSnapshot, currentRound uint64, commons bool) uint64 {
	// Build a set of qualifying item IDs from the stock list.
	qualifying := map[int]bool{}
	for _, e := range snap.Stock {
		if commons && e.Tier >= 40 {
			qualifying[e.ItemId] = true
		} else if !commons && e.Tier <= 20 {
			qualifying[e.ItemId] = true
		}
	}
	if len(qualifying) == 0 {
		return 0
	}

	var durations []uint64
	for itemId, evts := range snap.StockEvents {
		if !qualifying[itemId] {
			continue
		}
		for _, ev := range evts {
			if ev.RefilledRound == 0 {
				continue // still open; handled via CurrentDepletion below
			}
			if ev.RefilledRound > ev.DepletedRound {
				durations = append(durations, ev.RefilledRound-ev.DepletedRound)
			}
		}
	}
	// Currently-depleted items contribute ongoing duration.
	for itemId, depRound := range snap.CurrentDepletion {
		if !qualifying[itemId] {
			continue
		}
		if currentRound > depRound {
			durations = append(durations, currentRound-depRound)
		}
	}
	if len(durations) == 0 {
		return 0
	}
	sortUint64Capture(durations)
	return durations[len(durations)/2]
}

// sortUint64Capture is a simple insertion sort for small slices (median
// helper for capture.go; kept separate from delta.go to avoid package
// naming confusion in tests).
func sortUint64Capture(s []uint64) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
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

		// Populate DeliveriesByTier and LbsDelivered from caravan throughput.
		tp := caravan.GetThroughput(m.Character.Zone, instId)
		if tp != nil {
			cs.LbsDelivered = tp.LbsDelivered
			if tp.DeliveriesByTier != nil {
				cs.DeliveriesByTier = map[int]int{}
				for tier, count := range tp.DeliveriesByTier {
					cs.DeliveriesByTier[tier] = count
				}
			}
		}

		out = append(out, cs)
	}
	return out
}

// captureForagers walks forager.AllProfiles() and, for each profile,
// checks if a live mob instance exists. Emits distinct State strings
// to distinguish despawned (no live mob) from idle-no-state (live mob
// but empty BTreeState). For live mobs with valid forager_state, emits
// the full state snapshot including StuckRounds.
func captureForagers() []ForagerSnapshot {
	out := []ForagerSnapshot{}

	// Build lookup: mobId → live mob instance.
	liveByMobId := map[int]*mobs.Mob{}
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		if forager.ProfileFor(int(m.MobId)) == nil {
			continue
		}
		liveByMobId[int(m.MobId)] = m
	}

	now := util.GetRoundCount()

	for _, p := range forager.AllProfiles() {
		m, alive := liveByMobId[p.MobId]
		if !alive {
			out = append(out, ForagerSnapshot{
				MobId:         p.MobId,
				Name:          p.Name,
				Territory:     territoryFor(p.MobId),
				State:         "(despawned)",
				RoomId:        p.SanctuaryRoom,
				CargoByBucket: map[string]int{},
			})
			continue
		}

		bs, ok := m.BTreeState.(*behaviortree.BehaviorState)
		if !ok || bs == nil {
			out = append(out, ForagerSnapshot{
				InstId:        m.InstanceId,
				MobId:         p.MobId,
				Name:          p.Name,
				Territory:     territoryFor(p.MobId),
				State:         "(idle, no state)",
				RoomId:        m.Character.RoomId,
				CargoByBucket: map[string]int{},
			})
			continue
		}
		stateName := bs.GetString("forager_state")
		if stateName == "" {
			mudlog.Warn("economy/health.captureForagers",
				"warning", "forager state missing on live mob",
				"mobId", p.MobId, "name", p.Name, "roomId", m.Character.RoomId)
			out = append(out, ForagerSnapshot{
				InstId:        m.InstanceId,
				MobId:         p.MobId,
				Name:          p.Name,
				Territory:     territoryFor(p.MobId),
				State:         "(idle, no state)",
				RoomId:        m.Character.RoomId,
				CargoByBucket: map[string]int{},
			})
			continue
		}

		startedRound, _ := strconv.ParseUint(bs.GetString("forager_state_started_round"), 10, 64)
		var stuck uint64
		if now > startedRound {
			stuck = now - startedRound
		}

		fs := ForagerSnapshot{
			InstId:            m.InstanceId,
			MobId:             p.MobId,
			Name:              m.Character.Name,
			Territory:         territoryFor(p.MobId),
			State:             stateName,
			StateEnteredRound: startedRound,
			StuckRounds:       stuck,
			RoomId:            m.Character.RoomId,
			CargoByBucket:     map[string]int{},
			CargoWeight:       int(m.Character.GetCarriedWeight()),
			CargoCapacity:     int(m.Character.CarryCapacity()),
		}
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
		// Populate DeliveriesByTier and LbsDelivered from forager throughput.
		tp := forager.GetThroughput(m.Character.Zone, p.MobId)
		if tp != nil {
			fs.LbsDelivered = tp.LbsDelivered
			if tp.DeliveriesByTier != nil {
				fs.DeliveriesByTier = map[int]int{}
				for tier, count := range tp.DeliveriesByTier {
					fs.DeliveriesByTier[tier] = count
				}
			}
		}
		out = append(out, fs)
	}
	return out
}

// territoryFor returns the stable string label for a forager's
// territory, derived from forager.ProfileFor(mobId).Kind. Returns ""
// for non-foragers; logs a warning and returns "" for foragers whose
// Kind isn't yet mapped here (e.g. a new KindXxx added to
// internal/forager/territory.go without the matching case below).
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
	default:
		mudlog.Warn("economy/health: unmapped ForagerKind in territoryFor",
			"mobId", mobId, "kind", p.Kind, "remediation",
			"add a case to territoryFor in internal/economy/health/capture.go")
		return ""
	}
}

// lookupShopMobName resolves a shop's display name by walking live
// mob instances for one matching mobId+roomId. If the mob is not
// currently spawned, falls back to the mob template (always loaded at
// boot). Returns "" only if neither live instance nor template exists.
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
	// Fallback: template (always loaded at boot).
	if t := mobs.GetMobSpec(mobs.MobId(mobId)); t != nil {
		return t.Character.Name
	}
	return ""
}

// getRarityTier returns the rarity tier of an item from its ItemSpec.
// Returns 0 if the spec doesn't exist.
func getRarityTier(itemId int) int {
	spec := items.GetItemSpec(itemId)
	if spec == nil {
		return 0
	}
	return spec.RarityTier
}
