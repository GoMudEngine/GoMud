package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// writeFactionDef writes a faction definition YAML to the override dir.
func writeFactionDef(t *testing.T, slug, body string) {
	t.Helper()
	dir := os.Getenv("DOGMUD_FACTIONS_DIR_OVERRIDE")
	if err := os.WriteFile(filepath.Join(dir, slug+".yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// setupFactionsForHookTest wires temp dirs + loads thornwall_citizens.
func setupFactionsForHookTest(t *testing.T) {
	t.Helper()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE", t.TempDir())
	writeFactionDef(t, "thornwall_citizens", `
faction_id: thornwall_citizens
display_name: "Thornwall Citizenry"
description: "x"
default_rep: 0
allies: []
enemies: []
`)
	if err := factions.LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	factions.ClearCache()
	crimes.ClearCache()
}

// seedHookRoom registers a minimal room via SeedRoomsForTest and
// returns a cleanup func.
func seedHookRoom(t *testing.T, roomId int) (*rooms.Room, func()) {
	t.Helper()
	room := &rooms.Room{
		RoomId:      roomId,
		Zone:        "Thornwall City",
		Title:       "Test Room",
		Description: "A test room.",
		Exits:       map[string]exit.RoomExit{},
		Biome:       "city",
	}
	cleanupRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{roomId: room},
		map[string]*rooms.ZoneConfig{
			"Thornwall City": {
				Name:    "Thornwall City",
				RoomId:  roomId,
				RoomIds: map[int]struct{}{roomId: {}},
			},
		},
	)
	cleanupBiomes := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"city": {
			BiomeId:      "city",
			Name:         "City",
			Symbol:       "#",
			LitArea:      true,
			MovementCost: 1.0,
		},
	})
	return room, func() {
		cleanupBiomes()
		cleanupRooms()
	}
}

// makeCitizenMob builds a citizen mob spec + instance pair.
func makeCitizenMob(mobId, instId, roomId int, name string) (*mobs.Mob, *mobs.Mob) {
	spec := &mobs.Mob{
		MobId:  mobs.MobId(mobId),
		Groups: []string{"thornwall_citizens"},
		Zone:   "Thornwall City",
		Character: characters.Character{
			Name:  name,
			Buffs: buffs.New(),
		},
	}
	inst := &mobs.Mob{
		MobId:      mobs.MobId(mobId),
		InstanceId: instId,
		Groups:     []string{"thornwall_citizens"},
		Zone:       "Thornwall City",
		Character: characters.Character{
			Name:   name,
			RoomId: roomId,
			Buffs:  buffs.New(),
		},
	}
	return spec, inst
}

// TestMobDeathFactionRep_MurderWithWitness verifies that when a
// citizen is killed with a witness present the crime is recorded
// with PerpPlayer and the rep delta is applied.
func TestMobDeathFactionRep_MurderWithWitness(t *testing.T) {
	setupFactionsForHookTest(t)

	const roomId = 467

	room, cleanupRoom := seedHookRoom(t, roomId)
	defer cleanupRoom()

	// Victim: citizen mob 100, instance 200 (the one being killed).
	victimSpec, victimInst := makeCitizenMob(100, 200, roomId, "innocent citizen")
	// Witness: citizen mob 101, instance 201 (alive in the room).
	witnessSpec, witnessInst := makeCitizenMob(101, 201, roomId, "city guard")

	cleanupMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{100: victimSpec, 101: witnessSpec},
		map[int]*mobs.Mob{200: victimInst, 201: witnessInst},
	)
	defer cleanupMobs()

	// Only the witness is in the room (victim is already dead /
	// excluded by instance id).
	room.AddMob(201)

	evt := events.MobDeath{
		MobId:        100,
		InstanceId:   200,
		RoomId:       roomId,
		PlayerDamage: map[int]int{17: 50},
	}
	MobDeathFactionRep(evt)

	// Witness present: should record PerpPlayer crime.
	got := crimes.AllForFaction("thornwall_citizens", true)
	if len(got) != 1 {
		t.Fatalf("expected 1 crime, got %d: %+v", len(got), got)
	}
	c := got[0]
	if c.Kind != crimes.KindMurder {
		t.Errorf("kind = %s, want murder", c.Kind)
	}
	if c.Perpetrator.Type != crimes.PerpPlayer || c.Perpetrator.Id != 17 {
		t.Errorf("perpetrator = %+v, want {player, 17}", c.Perpetrator)
	}

	// Rep should have been bumped.
	want := int(configs.GetBalanceConfig().CrimeRepDeltaMurder)
	if got := factions.GetRep("thornwall_citizens", 17); got != want {
		t.Errorf("rep = %d, want %d", got, want)
	}
}

// TestMobDeathFactionRep_LoneMurderUnknownPerp verifies that when
// there are no witnesses the crime records PerpUnknown and NO rep
// bump is applied.
func TestMobDeathFactionRep_LoneMurderUnknownPerp(t *testing.T) {
	setupFactionsForHookTest(t)

	const roomId = 467

	room, cleanupRoom := seedHookRoom(t, roomId)
	defer cleanupRoom()
	_ = room // no mobs added — room is empty

	victimSpec, victimInst := makeCitizenMob(100, 200, roomId, "innocent citizen")
	cleanupMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{100: victimSpec},
		map[int]*mobs.Mob{200: victimInst},
	)
	defer cleanupMobs()

	// Victim instance excluded; room has no other mobs → no witnesses.
	evt := events.MobDeath{
		MobId:        100,
		InstanceId:   200,
		RoomId:       roomId,
		PlayerDamage: map[int]int{17: 50},
	}
	MobDeathFactionRep(evt)

	got := crimes.AllForFaction("thornwall_citizens", true)
	if len(got) != 1 {
		t.Fatalf("expected 1 crime, got %d", len(got))
	}
	c := got[0]
	if c.Kind != crimes.KindMurder {
		t.Errorf("kind = %s, want murder", c.Kind)
	}
	if c.Perpetrator.Type != crimes.PerpUnknown {
		t.Errorf("perpetrator type = %s, want unknown", c.Perpetrator.Type)
	}

	// No rep bump for lone/unwitnessed murder.
	if rep := factions.GetRep("thornwall_citizens", 17); rep != 0 {
		t.Errorf("rep = %d, want 0 (lone murder, no witness)", rep)
	}
}

// TestMobDeathFactionRep_UpgradesAssaultToMurder verifies that when a
// recent assault row exists it is upgraded in-place and only the
// incremental delta is applied.
func TestMobDeathFactionRep_UpgradesAssaultToMurder(t *testing.T) {
	setupFactionsForHookTest(t)

	const roomId = 467

	room, cleanupRoom := seedHookRoom(t, roomId)
	defer cleanupRoom()

	// Witness present so perp is identified.
	witnessSpec, witnessInst := makeCitizenMob(101, 201, roomId, "witness")
	victimSpec, victimInst := makeCitizenMob(100, 200, roomId, "victim")
	cleanupMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{100: victimSpec, 101: witnessSpec},
		map[int]*mobs.Mob{200: victimInst, 201: witnessInst},
	)
	defer cleanupMobs()
	room.AddMob(201)

	// Pre-seed an assault row (as if the first-aggression hook fired).
	assaultVictim := &mobs.Mob{MobId: 100}
	crimes.Record([]string{"thornwall_citizens"}, crimes.KindAssault,
		crimes.Perpetrator{Type: crimes.PerpPlayer, Id: 17},
		assaultVictim, 200, roomId, "Thornwall City")
	// Apply the assault delta so we can track the incremental.
	cfg := configs.GetBalanceConfig()
	factions.BumpRep("thornwall_citizens", 17, int(cfg.CrimeRepDeltaAssault))

	repAfterAssault := factions.GetRep("thornwall_citizens", 17)

	evt := events.MobDeath{
		MobId:        100,
		InstanceId:   200,
		RoomId:       roomId,
		PlayerDamage: map[int]int{17: 50},
	}
	MobDeathFactionRep(evt)

	// Still ONE crime row (upgraded, not a new row).
	got := crimes.AllForFaction("thornwall_citizens", true)
	if len(got) != 1 {
		t.Fatalf("expected 1 crime after upgrade, got %d", len(got))
	}
	if got[0].Kind != crimes.KindMurder {
		t.Errorf("kind after upgrade = %s, want murder", got[0].Kind)
	}

	// Rep should have moved by the incremental difference only.
	wantDelta := int(cfg.CrimeRepDeltaMurder) - int(cfg.CrimeRepDeltaAssault)
	wantRep := repAfterAssault + wantDelta
	if rep := factions.GetRep("thornwall_citizens", 17); rep != wantRep {
		t.Errorf("rep = %d, want %d (repAfterAssault=%d, increment=%d)",
			rep, wantRep, repAfterAssault, wantDelta)
	}
}

// TestMobDeathFactionRep_NoFactionsNoChange verifies the early-exit
// path when the victim mob has no defined factions.
func TestMobDeathFactionRep_NoFactionsNoChange(t *testing.T) {
	setupFactionsForHookTest(t)

	mob := &mobs.Mob{
		MobId:      999,
		InstanceId: 201,
		Groups:     []string{"humanoid"}, // no defined faction
		Character:  characters.Character{Name: "x", RoomId: 467, Buffs: buffs.New()},
	}
	cleanup := mobs.SeedMobsForTest(map[int]*mobs.Mob{999: mob}, map[int]*mobs.Mob{201: mob})
	defer cleanup()

	evt := events.MobDeath{
		MobId:        999,
		InstanceId:   201,
		RoomId:       467,
		PlayerDamage: map[int]int{17: 50},
	}
	MobDeathFactionRep(evt)

	if got := crimes.AllForFaction("thornwall_citizens", true); len(got) != 0 {
		t.Errorf("non-faction kill recorded a crime: %+v", got)
	}
	if rep := factions.GetRep("thornwall_citizens", 17); rep != 0 {
		t.Errorf("rep changed for non-faction kill: %d", rep)
	}
}

// TestMobDeathFactionRep_NoPlayersNoChange verifies the early-exit
// path when PlayerDamage is empty (environmental / mob death).
func TestMobDeathFactionRep_NoPlayersNoChange(t *testing.T) {
	setupFactionsForHookTest(t)

	mob := &mobs.Mob{
		MobId:      100,
		InstanceId: 202,
		Groups:     []string{"thornwall_citizens"},
		Character:  characters.Character{Name: "x", RoomId: 467, Buffs: buffs.New()},
	}
	cleanup := mobs.SeedMobsForTest(map[int]*mobs.Mob{100: mob}, map[int]*mobs.Mob{202: mob})
	defer cleanup()

	evt := events.MobDeath{
		MobId:        100,
		InstanceId:   202,
		RoomId:       467,
		PlayerDamage: map[int]int{}, // no damager
	}
	MobDeathFactionRep(evt)

	if got := crimes.AllForFaction("thornwall_citizens", true); len(got) != 0 {
		t.Errorf("crime recorded with no damager: %+v", got)
	}
}

// configsAlive avoids unused-import warning if configs isn't
// referenced in an active test body.
var _ = configs.GetBalanceConfig
