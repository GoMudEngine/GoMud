package crimes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func setupTestCrimes(t *testing.T) {
	t.Helper()
	t.Setenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE", t.TempDir())
	clearCrimeCacheForTest()
	roundForTest = func() uint64 { return 1000 }
	t.Cleanup(func() { roundForTest = nil })
}

func TestRecordReturnsIdAndPersists(t *testing.T) {
	setupTestCrimes(t)

	victim := &mobs.Mob{MobId: 100, Character: characters.Character{Name: "city beggar"}}
	ids := Record(
		[]string{"thornwall_citizens"},
		KindMurder,
		Perpetrator{Type: PerpPlayer, Id: 17},
		victim, 250, 467, "Thornwall City",
	)
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("Record returned %v, want [1]", ids)
	}

	got := AllForFaction("thornwall_citizens", false)
	if len(got) != 1 {
		t.Fatalf("AllForFaction returned %d, want 1", len(got))
	}
	c := got[0]
	if c.Kind != KindMurder || c.VictimMobId != 100 || c.RoomId != 467 ||
		c.Perpetrator.Id != 17 || c.Round != 1000 {
		t.Errorf("crime mismatch: %+v", c)
	}

	// Drop cache, reload from disk.
	ClearCache()
	got = AllForFaction("thornwall_citizens", false)
	if len(got) != 1 || got[0].Id != 1 {
		t.Errorf("after restart-equivalent: %+v", got)
	}
}

func TestRecordMonotonicIds(t *testing.T) {
	setupTestCrimes(t)

	victim := &mobs.Mob{MobId: 100}
	for i := 0; i < 5; i++ {
		Record([]string{"f"}, KindAssault, Perpetrator{Type: PerpPlayer, Id: 17},
			victim, 250, 467, "z")
	}
	got := AllForFaction("f", false)
	for i, c := range got {
		if c.Id != i+1 {
			t.Errorf("crime %d: id = %d, want %d", i, c.Id, i+1)
		}
	}
}

func TestRecordMultipleFactions(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 94}
	ids := Record(
		[]string{"thornwall_guards", "thornwall_citizens"},
		KindMurder,
		Perpetrator{Type: PerpPlayer, Id: 17},
		victim, 251, 467, "Thornwall City",
	)
	if len(ids) != 2 {
		t.Errorf("Record on 2 factions returned %d ids, want 2", len(ids))
	}
	if got := AllForFaction("thornwall_guards", false); len(got) != 1 {
		t.Errorf("guards log: %d, want 1", len(got))
	}
	if got := AllForFaction("thornwall_citizens", false); len(got) != 1 {
		t.Errorf("citizens log: %d, want 1", len(got))
	}
}

func TestResolveMarksRow(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}
	Record([]string{"f"}, KindMurder, Perpetrator{Type: PerpPlayer, Id: 17},
		victim, 250, 467, "z")

	roundForTest = func() uint64 { return 2000 }
	Resolve("f", 1, "fine paid")

	got := AllForFaction("f", true)
	if len(got) != 1 {
		t.Fatalf("expected 1 crime")
	}
	if got[0].ResolvedRound != 2000 || got[0].ResolvedBy != "fine paid" {
		t.Errorf("resolve fields: %+v", got[0])
	}
}

func TestResolveIdempotent(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}
	Record([]string{"f"}, KindMurder, Perpetrator{Type: PerpPlayer, Id: 17},
		victim, 250, 467, "z")
	Resolve("f", 1, "fine paid")
	roundForTest = func() uint64 { return 9999 }
	Resolve("f", 1, "another reason")
	got := AllForFaction("f", true)
	if got[0].ResolvedRound != 1000 {
		t.Errorf("resolve was not idempotent: %+v", got[0])
	}
	if got[0].ResolvedBy != "fine paid" {
		t.Errorf("resolve overwrote reason: %+v", got[0])
	}
}

func TestAllForFactionFiltersResolved(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}
	Record([]string{"f"}, KindMurder, Perpetrator{Type: PerpPlayer, Id: 17},
		victim, 250, 467, "z")
	Record([]string{"f"}, KindMurder, Perpetrator{Type: PerpPlayer, Id: 17},
		victim, 251, 468, "z")
	Resolve("f", 1, "fine")

	if got := AllForFaction("f", false); len(got) != 1 || got[0].Id != 2 {
		t.Errorf("includeResolved=false should hide id=1: %+v", got)
	}
	if got := AllForFaction("f", true); len(got) != 2 {
		t.Errorf("includeResolved=true should show both: %d", len(got))
	}
}

func TestAllForPlayer(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}
	Record([]string{"a"}, KindMurder, Perpetrator{Type: PerpPlayer, Id: 17},
		victim, 250, 467, "z")
	Record([]string{"b"}, KindAssault, Perpetrator{Type: PerpPlayer, Id: 17},
		victim, 250, 468, "z")
	Record([]string{"a"}, KindMurder, Perpetrator{Type: PerpPlayer, Id: 99},
		victim, 251, 469, "z") // different player

	got := AllForPlayer(17, false)
	if len(got) != 2 {
		t.Errorf("AllForPlayer(17) = %d, want 2", len(got))
	}
	got = AllForPlayer(99, false)
	if len(got) != 1 {
		t.Errorf("AllForPlayer(99) = %d, want 1", len(got))
	}
}

func TestAllForPlayerExcludesUnknownPerp(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}
	Record([]string{"a"}, KindMurder, Perpetrator{Type: PerpUnknown},
		victim, 250, 467, "z")
	got := AllForPlayer(17, false)
	if len(got) != 0 {
		t.Errorf("unknown-perp crime should not appear in AllForPlayer: %+v", got)
	}
}

func TestKindConstants(t *testing.T) {
	if KindAssault != "assault" {
		t.Errorf("KindAssault = %q, want assault", KindAssault)
	}
	if KindMurder != "murder" {
		t.Errorf("KindMurder = %q, want murder", KindMurder)
	}
	if KindTheft != "theft" {
		t.Errorf("KindTheft = %q, want theft", KindTheft)
	}
}

func TestPerpTypeConstants(t *testing.T) {
	if PerpPlayer != "player" {
		t.Errorf("PerpPlayer = %q, want player", PerpPlayer)
	}
	if PerpUnknown != "unknown" {
		t.Errorf("PerpUnknown = %q, want unknown", PerpUnknown)
	}
}

func setupFactionsForCrimesTest(t *testing.T) {
	t.Helper()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	dir := os.Getenv("DOGMUD_FACTIONS_DIR_OVERRIDE")
	body := `faction_id: thornwall_citizens
display_name: "Thornwall Citizenry"
description: "x"
default_rep: 0
allies: []
enemies: []
`
	if err := os.WriteFile(filepath.Join(dir, "thornwall_citizens.yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if err := factions.LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
}

func TestIdentifiedPerp_EmptyWitnessesIsUnknown(t *testing.T) {
	got := IdentifiedPerp(17, []int{})
	if got.Type != PerpUnknown {
		t.Errorf("empty witnesses: got type %q, want unknown", got.Type)
	}
}

func TestIdentifiedPerp_NonEmptyIsPlayer(t *testing.T) {
	got := IdentifiedPerp(17, []int{42})
	if got.Type != PerpPlayer || got.Id != 17 {
		t.Errorf("got %+v, want {player, 17}", got)
	}
}

func TestWitnessesInRoom_FiltersByFactionGroup(t *testing.T) {
	setupTestCrimes(t)
	setupFactionsForCrimesTest(t)

	// Build a fake room with three mob instances:
	//   100 — a citizen (faction member, witness)
	//   200 — humanoid only (not a witness)
	//   300 — citizen but the dead victim (excluded)
	room := &rooms.Room{RoomId: 467}
	citizen := &mobs.Mob{MobId: 100, InstanceId: 100, Groups: []string{"thornwall_citizens"}}
	otherMob := &mobs.Mob{MobId: 999, InstanceId: 200, Groups: []string{"humanoid"}}
	victim := &mobs.Mob{MobId: 102, InstanceId: 300, Groups: []string{"thornwall_citizens"}}

	mobs.SetInstanceForTest(100, citizen)
	defer mobs.SetInstanceForTest(100, nil)
	mobs.SetInstanceForTest(200, otherMob)
	defer mobs.SetInstanceForTest(200, nil)
	mobs.SetInstanceForTest(300, victim)
	defer mobs.SetInstanceForTest(300, nil)

	room.AddMob(100)
	room.AddMob(200)
	room.AddMob(300)

	got := WitnessesInRoom([]string{"thornwall_citizens"}, room, 300)
	if len(got) != 1 || got[0] != 100 {
		t.Errorf("WitnessesInRoom = %v, want [100]", got)
	}
}

func TestWitnessesInRoom_VictimIncludedWhenNotExcluded(t *testing.T) {
	setupTestCrimes(t)
	setupFactionsForCrimesTest(t)

	room := &rooms.Room{RoomId: 467}
	victim := &mobs.Mob{MobId: 100, InstanceId: 300, Groups: []string{"thornwall_citizens"}}
	mobs.SetInstanceForTest(300, victim)
	defer mobs.SetInstanceForTest(300, nil)
	room.AddMob(300)

	got := WitnessesInRoom([]string{"thornwall_citizens"}, room, 0) // assault: include victim
	if len(got) != 1 || got[0] != 300 {
		t.Errorf("WitnessesInRoom = %v, want [300] (victim self-witness)", got)
	}
}

func TestFindRecentAssault_ReturnsMatch(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}

	roundForTest = func() uint64 { return 1000 }
	Record([]string{"f"}, KindAssault, Perpetrator{Type: PerpPlayer, Id: 17}, victim, 250, 467, "z")

	got := FindRecentAssault("f", 17, 100)
	if got == nil || got.Id != 1 || got.Kind != KindAssault {
		t.Errorf("FindRecentAssault: %+v", got)
	}
}

func TestFindRecentAssault_NilWhenOutsideWindow(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}

	roundForTest = func() uint64 { return 1000 }
	Record([]string{"f"}, KindAssault, Perpetrator{Type: PerpPlayer, Id: 17}, victim, 250, 467, "z")

	roundForTest = func() uint64 { return 2000 } // 1000 rounds later
	if got := FindRecentAssault("f", 17, 100); got != nil {
		t.Errorf("expected nil for out-of-window; got %+v", got)
	}
}

func TestFindRecentAssault_NilWhenResolved(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}
	Record([]string{"f"}, KindAssault, Perpetrator{Type: PerpPlayer, Id: 17}, victim, 250, 467, "z")
	Resolve("f", 1, "fine")

	if got := FindRecentAssault("f", 17, 100); got != nil {
		t.Errorf("expected nil for resolved assault; got %+v", got)
	}
}

func TestFindRecentAssault_NilWhenAlreadyMurder(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}
	Record([]string{"f"}, KindMurder, Perpetrator{Type: PerpPlayer, Id: 17}, victim, 250, 467, "z")

	if got := FindRecentAssault("f", 17, 100); got != nil {
		t.Errorf("FindRecentAssault should ignore murder rows; got %+v", got)
	}
}

func TestFindRecentAssault_NilWhenWrongPlayer(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}
	Record([]string{"f"}, KindAssault, Perpetrator{Type: PerpPlayer, Id: 99}, victim, 250, 467, "z")

	if got := FindRecentAssault("f", 17, 100); got != nil {
		t.Errorf("FindRecentAssault wrong player: %+v", got)
	}
}

func TestPruneStaleResolvesOldUnresolvedRows(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}

	staleAfterForTest = func() uint64 { return 500 }
	t.Cleanup(func() { staleAfterForTest = nil })

	roundForTest = func() uint64 { return 1000 }
	Record([]string{"f"}, KindMurder, Perpetrator{Type: PerpPlayer, Id: 17}, victim, 250, 467, "z") // id 1, round 1000
	Record([]string{"f"}, KindMurder, Perpetrator{Type: PerpPlayer, Id: 17}, victim, 251, 468, "z") // id 2, round 1000

	roundForTest = func() uint64 { return 1600 } // 600 rounds later — past 500-round threshold
	count := PruneStale("f")
	if count != 2 {
		t.Errorf("PruneStale snapped %d rows, want 2", count)
	}
	got := AllForFaction("f", true)
	for _, c := range got {
		if c.ResolvedRound == 0 || c.ResolvedBy != "stale" {
			t.Errorf("crime %d not snapped: %+v", c.Id, c)
		}
	}
}

func TestPruneStaleSkipsRecentRows(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}

	staleAfterForTest = func() uint64 { return 500 }
	t.Cleanup(func() { staleAfterForTest = nil })

	roundForTest = func() uint64 { return 1000 }
	Record([]string{"f"}, KindMurder, Perpetrator{Type: PerpPlayer, Id: 17}, victim, 250, 467, "z")

	roundForTest = func() uint64 { return 1100 } // 100 rounds later — within threshold
	count := PruneStale("f")
	if count != 0 {
		t.Errorf("PruneStale snapped %d rows, want 0", count)
	}
	if got := AllForFaction("f", false); len(got) != 1 {
		t.Errorf("recent crime should still be unresolved: %+v", got)
	}
}

func TestPruneStaleSkipsAlreadyResolved(t *testing.T) {
	setupTestCrimes(t)
	victim := &mobs.Mob{MobId: 100}

	staleAfterForTest = func() uint64 { return 500 }
	t.Cleanup(func() { staleAfterForTest = nil })

	roundForTest = func() uint64 { return 1000 }
	Record([]string{"f"}, KindMurder, Perpetrator{Type: PerpPlayer, Id: 17}, victim, 250, 467, "z")
	Resolve("f", 1, "fine")

	roundForTest = func() uint64 { return 1600 }
	count := PruneStale("f")
	if count != 0 {
		t.Errorf("PruneStale should skip already-resolved rows; got %d", count)
	}
	got := AllForFaction("f", true)
	if got[0].ResolvedBy != "fine" {
		t.Errorf("PruneStale overwrote existing resolved_by: %+v", got[0])
	}
}
