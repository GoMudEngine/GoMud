package health_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/economy/health"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/ferry"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// TestCaptureSnapshot_FerryFactors verifies that a live mob instance
// registered as a ferry trade factor (via ferry.SetFactorStateForTest)
// surfaces as a CaravanSnapshot row: state from ferry.FactorPhaseName,
// room/cargo from the mob's own Character (factors carry their own cargo —
// no separate wagon), and throughput from caravan.GetThroughput keyed by
// (m.Zone, m.MobId), the same keys VisitVendorsInRoomOpts writes under.
func TestCaptureSnapshot_FerryFactors(t *testing.T) {
	const roomId = 9600

	r := &rooms.Room{
		RoomId: roomId,
		Zone:   "Stillwater",
		Title:  "Test Dock",
		Exits:  map[string]exit.RoomExit{},
	}
	cleanRoom := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{roomId: r},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRoom()

	const factorMobId = 9577
	const factorInstanceId = 96001
	factor := &mobs.Mob{
		MobId:      mobs.MobId(factorMobId),
		InstanceId: factorInstanceId,
		HomeRoomId: roomId,
		Zone:       "Stillwater", // template-stable zone; matches the throughput key below
	}
	factor.Character.Name = "A Lakeway Factor"
	factor.Character.Buffs = buffs.New()
	factor.Character.RoomId = roomId
	// Deterministic carry capacity — item specs aren't loaded in test, so
	// per-item weights resolve to 0 (mirrors the caravan/forager capture
	// test convention: assert structural wiring, not specific weights).
	characters.ApplyMobOverrides(&factor.Character, 0, 0, 200)
	factor.Character.Items = append(factor.Character.Items, items.Item{ItemId: 40057}) // lake mint, "stillwater" bucket

	mobs.SetInstanceForTest(factor.InstanceId, factor)
	defer mobs.SetInstanceForTest(factor.InstanceId, nil)
	r.AddMob(factor.InstanceId)

	ferry.SetFactorStateForTest(factorInstanceId, ferry.FactorDelivering)
	defer ferry.ClearFactorStateForTest(factorInstanceId)

	// Seed throughput under (m.Zone, m.MobId) — the keys
	// caravan.VisitVendorsInRoomOpts writes deliveries under for factors.
	caravan.ClearThroughputCache()
	t.Cleanup(caravan.ClearThroughputCache)
	caravan.IncrementDelivery("Stillwater", factorMobId, 50)

	snap := health.CaptureSnapshot()

	var got *health.CaravanSnapshot
	for i := range snap.Caravans {
		if snap.Caravans[i].InstId == factorInstanceId {
			got = &snap.Caravans[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("no caravan row found for factor instance %d in %+v", factorInstanceId, snap.Caravans)
	}
	if got.Name != "A Lakeway Factor" {
		t.Errorf("Name: got %q, want %q", got.Name, "A Lakeway Factor")
	}
	if got.State != "delivering" {
		t.Errorf("State: got %q, want delivering", got.State)
	}
	if got.RoomId != roomId {
		t.Errorf("RoomId: got %d, want %d", got.RoomId, roomId)
	}
	if got.CargoCapacity != 200 {
		t.Errorf("CargoCapacity: got %d, want 200 (override set in fixture)", got.CargoCapacity)
	}
	if got.CargoByBucket == nil {
		t.Error("CargoByBucket: got nil map, want initialized map")
	}
	if got.DeliveriesByTier == nil || got.DeliveriesByTier[50] != 1 {
		t.Errorf("DeliveriesByTier[50]: got %v, want 1", got.DeliveriesByTier)
	}
}

// TestCaptureSnapshot_FerryFactors_NonFactorMobsExcluded verifies that a
// live mob instance NOT registered in ferry's factorStates (i.e.
// ferry.FactorPhaseName returns ok=false) does not produce a spurious
// caravan row — captureFerryFactors must gate strictly on that lookup,
// not on mob id or zone.
func TestCaptureSnapshot_FerryFactors_NonFactorMobsExcluded(t *testing.T) {
	const roomId = 9601

	r := &rooms.Room{
		RoomId: roomId,
		Zone:   "Stillwater",
		Title:  "Test Room",
		Exits:  map[string]exit.RoomExit{},
	}
	cleanRoom := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{roomId: r},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRoom()

	const plainInstanceId = 96002
	plain := &mobs.Mob{
		MobId:      mobs.MobId(9577),
		InstanceId: plainInstanceId,
		HomeRoomId: roomId,
		Zone:       "Stillwater",
	}
	plain.Character.Name = "Not A Factor Right Now"
	plain.Character.Buffs = buffs.New()
	plain.Character.RoomId = roomId

	mobs.SetInstanceForTest(plain.InstanceId, plain)
	defer mobs.SetInstanceForTest(plain.InstanceId, nil)
	r.AddMob(plain.InstanceId)

	// Deliberately NOT calling ferry.SetFactorStateForTest.

	snap := health.CaptureSnapshot()

	for _, c := range snap.Caravans {
		if c.InstId == plainInstanceId {
			t.Fatalf("unexpected caravan row for non-factor mob instance %d: %+v", plainInstanceId, c)
		}
	}
}
