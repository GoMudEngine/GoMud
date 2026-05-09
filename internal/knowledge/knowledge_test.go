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
