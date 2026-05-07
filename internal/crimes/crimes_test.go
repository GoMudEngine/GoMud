package crimes

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
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
