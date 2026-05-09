package knowledge

import (
	"testing"

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
