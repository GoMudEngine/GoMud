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
