package knowledge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestRecordMet_FreshThenIdempotent(t *testing.T) {
	resetCache()
	originalRound := util.GetRoundCount()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return originalRound + 100 }

	RecordMet(99, PlayerSubject(17), 462, SourceWitnessed)

	r := Get(99, PlayerSubject(17))
	if r == nil {
		t.Fatalf("expected record after RecordMet")
	}
	if !r.HasMet {
		t.Errorf("HasMet should be true")
	}
	if r.LastSeenRoom != 462 {
		t.Errorf("LastSeenRoom mismatch: got %d", r.LastSeenRoom)
	}
	if r.LearnedRound != originalRound+100 {
		t.Errorf("LearnedRound not set: got %d", r.LearnedRound)
	}
	if r.Source != SourceWitnessed || r.Confidence != ConfidenceHigh {
		t.Errorf("source/confidence wrong: %s/%s", r.Source, r.Confidence)
	}

	// Second call — LearnedRound stays, LastSeenRound updates.
	roundForTest = func() uint64 { return originalRound + 200 }
	RecordMet(99, PlayerSubject(17), 463, SourceWitnessed)
	r = Get(99, PlayerSubject(17))
	if r.LearnedRound != originalRound+100 {
		t.Errorf("LearnedRound should not change on second call: got %d", r.LearnedRound)
	}
	if r.LastSeenRoom != 463 {
		t.Errorf("LastSeenRoom should update: got %d", r.LastSeenRoom)
	}
	if r.LastSeenRound != originalRound+200 {
		t.Errorf("LastSeenRound should update: got %d", r.LastSeenRound)
	}
}

func TestRecordObservation_BoundedFIFO(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; observationLogMaxForTest = nil }()
	r := uint64(1000)
	roundForTest = func() uint64 { r++; return r }

	// Force a small bound for the test.
	observationLogMaxForTest = func() int { return 4 }

	for i := 1; i <= 6; i++ {
		RecordObservation(99, PlayerSubject(17), 100+i)
	}

	rec := Get(99, PlayerSubject(17))
	if rec == nil {
		t.Fatalf("expected record")
	}
	if len(rec.Observations) != 4 {
		t.Fatalf("expected bounded log of 4, got %d", len(rec.Observations))
	}
	// Should hold the LAST 4 entries (rooms 103..106).
	wantRooms := []int{103, 104, 105, 106}
	for i, o := range rec.Observations {
		if o.Room != wantRooms[i] {
			t.Errorf("entry %d: room %d, want %d", i, o.Room, wantRooms[i])
		}
	}
}

func TestRecordObservation_SameRoundDedup(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 500 }

	// Use different observer ID to avoid file pollution from previous test.
	RecordObservation(98, PlayerSubject(17), 462)
	RecordObservation(98, PlayerSubject(17), 462) // same room+round

	rec := Get(98, PlayerSubject(17))
	if len(rec.Observations) != 1 {
		t.Errorf("expected dedup at same round, got %d entries", len(rec.Observations))
	}
}

func TestRecordName(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	RecordName(99, PlayerSubject(17), "Bob", SourceWitnessed)
	r := Get(99, PlayerSubject(17))
	if r.NameLearned != "Bob" {
		t.Errorf("name not set: got %q", r.NameLearned)
	}

	// Idempotent on same value.
	RecordName(99, PlayerSubject(17), "Bob", SourceWitnessed)
	if r.NameLearned != "Bob" {
		t.Errorf("name corrupted on re-write: %q", r.NameLearned)
	}
}

func TestRecordCrimeWitnessed_Dedup(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	RecordCrimeWitnessed(99, PlayerSubject(17), 5)
	RecordCrimeWitnessed(99, PlayerSubject(17), 7)
	RecordCrimeWitnessed(99, PlayerSubject(17), 5) // duplicate

	r := Get(99, PlayerSubject(17))
	if len(r.CrimesWitnessed) != 2 {
		t.Errorf("expected dedup, got %v", r.CrimesWitnessed)
	}
	want := map[int]bool{5: true, 7: true}
	for _, id := range r.CrimesWitnessed {
		if !want[id] {
			t.Errorf("unexpected crime id %d", id)
		}
	}
}

func TestForget_DropsRecord(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	RecordMet(99, PlayerSubject(17), 462, SourceWitnessed)
	RecordMet(99, PlayerSubject(18), 463, SourceWitnessed)

	Forget(99, PlayerSubject(17))

	if Get(99, PlayerSubject(17)) != nil {
		t.Errorf("Forget did not drop record for subject 17")
	}
	if Get(99, PlayerSubject(18)) == nil {
		t.Errorf("Forget should have left subject 18 alone")
	}
}

func TestForgetFact(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	RecordMet(99, PlayerSubject(17), 462, SourceWitnessed)
	RecordName(99, PlayerSubject(17), "Bob", SourceWitnessed)
	RecordObservation(99, PlayerSubject(17), 463)
	RecordCrimeWitnessed(99, PlayerSubject(17), 5)

	ForgetFact(99, PlayerSubject(17), "name")
	r := Get(99, PlayerSubject(17))
	if r.NameLearned != "" {
		t.Errorf("expected name cleared, got %q", r.NameLearned)
	}
	if len(r.Observations) == 0 || len(r.CrimesWitnessed) == 0 {
		t.Errorf("ForgetFact name should not touch observations/crimes")
	}

	ForgetFact(99, PlayerSubject(17), "observations")
	r = Get(99, PlayerSubject(17))
	if len(r.Observations) != 0 {
		t.Errorf("expected observations cleared, got %d", len(r.Observations))
	}

	ForgetFact(99, PlayerSubject(17), "crimes")
	r = Get(99, PlayerSubject(17))
	if len(r.CrimesWitnessed) != 0 {
		t.Errorf("expected crimes cleared, got %d", len(r.CrimesWitnessed))
	}
}

func TestForgetFact_UnknownIsNoOp(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	// Use fresh observer ID 700 to avoid disk leakage from earlier tests.
	RecordMet(700, PlayerSubject(17), 462, SourceWitnessed)
	RecordName(700, PlayerSubject(17), "Bob", SourceWitnessed)

	// Calling ForgetFact with an unrecognised key must be a no-op.
	ForgetFact(700, PlayerSubject(17), "bogus")

	r := Get(700, PlayerSubject(17))
	if r == nil {
		t.Fatalf("record should still exist after ForgetFact with unknown key")
	}
	if r.NameLearned != "Bob" {
		t.Errorf("NameLearned should be unchanged: got %q", r.NameLearned)
	}
	if !r.HasMet {
		t.Errorf("HasMet should be unchanged after no-op ForgetFact")
	}
}

func TestReadAPIs(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	// Use fresh observer ID 299 to avoid disk interference from prior tests.
	if HasMet(299, PlayerSubject(17)) {
		t.Errorf("HasMet should be false for unknown subject")
	}
	if _, ok := NameOf(299, PlayerSubject(17)); ok {
		t.Errorf("NameOf should return ok=false for unknown")
	}
	if _, _, ok := LastSeen(299, PlayerSubject(17)); ok {
		t.Errorf("LastSeen should return ok=false for unknown")
	}

	RecordMet(299, PlayerSubject(17), 462, SourceWitnessed)
	RecordName(299, PlayerSubject(17), "Bob", SourceWitnessed)

	if !HasMet(299, PlayerSubject(17)) {
		t.Errorf("HasMet should be true after RecordMet")
	}
	if name, ok := NameOf(299, PlayerSubject(17)); !ok || name != "Bob" {
		t.Errorf("NameOf: got %q ok=%v", name, ok)
	}
	if room, round, ok := LastSeen(299, PlayerSubject(17)); !ok || room != 462 || round != 100 {
		t.Errorf("LastSeen: got room=%d round=%d ok=%v", room, round, ok)
	}
}

// setupFactionsAndCrimesForTest seeds one faction definition (thornwall_citizens)
// into the factions registry and redirects the crimes persistence to a temp dir,
// so WitnessedCrimes tests can exercise the full cross-package join without
// touching real data files.
func setupFactionsAndCrimesForTest(t *testing.T) {
	t.Helper()

	// Point faction definitions at a temp dir with one fixture file.
	facDir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", facDir)

	body := `faction_id: thornwall_citizens
display_name: "Thornwall Citizenry"
description: "Citizens of Thornwall."
default_rep: 0
allies: []
enemies: []
`
	if err := os.WriteFile(filepath.Join(facDir, "thornwall_citizens.yaml"), []byte(body), 0644); err != nil {
		t.Fatalf("write faction fixture: %v", err)
	}
	if err := factions.LoadAllDefinitions(); err != nil {
		t.Fatalf("LoadAllDefinitions: %v", err)
	}

	// Redirect crimes persistence to a separate temp dir.
	t.Setenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE", t.TempDir())
	crimes.ClearCache()
	t.Cleanup(func() { crimes.ClearCache() })
}

func TestWitnessedCrimes_LazyJoin(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	setupFactionsAndCrimesForTest(t)

	// Seed 1.3 with two crimes: one unresolved, one resolved.
	// Use fresh observer/player IDs (599, 9917) to avoid disk interference.
	victim := &mobs.Mob{MobId: 100, Character: characters.Character{Name: "city guard"}}
	ids := crimes.Record([]string{"thornwall_citizens"},
		crimes.KindAssault,
		crimes.Perpetrator{Type: crimes.PerpPlayer, Id: 9917},
		victim, 1, 462, "Thornwall City", false)
	if len(ids) != 1 {
		t.Fatalf("expected 1 crime id, got %d", len(ids))
	}
	crimeIdA := ids[0]

	ids = crimes.Record([]string{"thornwall_citizens"},
		crimes.KindMurder,
		crimes.Perpetrator{Type: crimes.PerpPlayer, Id: 9917},
		victim, 2, 463, "Thornwall City", false)
	if len(ids) != 1 {
		t.Fatalf("expected 1 crime id from second record, got %d", len(ids))
	}
	crimeIdB := ids[0]
	crimes.Resolve("thornwall_citizens", crimeIdB, "fine_paid")

	// Knowledge layer references both crimes.
	RecordCrimeWitnessed(599, PlayerSubject(9917), crimeIdA)
	RecordCrimeWitnessed(599, PlayerSubject(9917), crimeIdB)

	got := WitnessedCrimes(599, PlayerSubject(9917))
	if len(got) != 2 {
		t.Fatalf("expected 2 crimes returned, got %d", len(got))
	}
	for _, w := range got {
		switch w.CrimeId {
		case crimeIdA:
			if w.ResolvedRound != 0 {
				t.Errorf("crime %d should be unresolved (ResolvedRound=%d)", w.CrimeId, w.ResolvedRound)
			}
			if w.Kind != crimes.KindAssault {
				t.Errorf("crime %d: Kind=%q, want assault", w.CrimeId, w.Kind)
			}
		case crimeIdB:
			if w.ResolvedRound == 0 {
				t.Errorf("crime %d should be resolved", w.CrimeId)
			}
			if w.Kind != crimes.KindMurder {
				t.Errorf("crime %d: Kind=%q, want murder", w.CrimeId, w.Kind)
			}
		default:
			t.Errorf("unexpected crime id %d", w.CrimeId)
		}
	}
}

func TestWitnessedCrimes_NilWhenNoRecord(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	// Observer 599 has no record for this subject at all.
	got := WitnessedCrimes(599, PlayerSubject(8888))
	if got != nil {
		t.Errorf("expected nil for unknown observer/subject, got %v", got)
	}
}

func TestWitnessedCrimes_EmptyWhenNoCrimesWitnessed(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	// Record met but no crimes.
	RecordMet(599, PlayerSubject(8889), 462, SourceWitnessed)
	got := WitnessedCrimes(599, PlayerSubject(8889))
	if got != nil {
		t.Errorf("expected nil for subject with no witnessed crimes, got %v", got)
	}
}

func TestRoutineObservers_WritesForOthers(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	// Set up a room with three mobs: a moving forager (template id 50),
	// a citizen observer (template id 99), a guard observer (template id 92).
	room := rooms.LoadRoom(1)
	if room == nil {
		t.Skip("test fixture room missing")
	}
	movingInst := 1001
	observer1Inst := 1002
	observer2Inst := 1003

	moving := mobs.NewMobByIdFresh(50, 0)
	observer1 := mobs.NewMobByIdFresh(99, 0)
	observer2 := mobs.NewMobByIdFresh(92, 0)
	if moving == nil || observer1 == nil || observer2 == nil {
		t.Skip("test fixture mob templates 50/99/92 not registered")
	}
	moving.InstanceId = movingInst
	observer1.InstanceId = observer1Inst
	observer2.InstanceId = observer2Inst

	room.AddMob(movingInst)
	room.AddMob(observer1Inst)
	room.AddMob(observer2Inst)
	defer func() {
		room.RemoveMob(movingInst)
		room.RemoveMob(observer1Inst)
		room.RemoveMob(observer2Inst)
	}()

	RecordRoutineObservers(movingInst, 50, room)

	if Get(99, MobSubject(50)) == nil {
		t.Errorf("observer 99 should have a record about moving mob 50")
	}
	if Get(92, MobSubject(50)) == nil {
		t.Errorf("observer 92 should have a record about moving mob 50")
	}
	if Get(50, MobSubject(50)) != nil {
		t.Errorf("moving mob should not record itself")
	}
}
