package knowledge

import (
	"testing"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	resetCache()

	fc := &ObserverFile{
		ObserverMobId: 99,
		ObserverName:  "records_clerk_pell",
		Records: []*Record{
			{
				Subject:       PlayerSubject(17),
				HasMet:        true,
				NameLearned:   "smoketester",
				Source:        SourceWitnessed,
				Confidence:    ConfidenceHigh,
				LastSeenRoom:  462,
				LastSeenRound: 100,
				Observations: []Observation{
					{Room: 462, Round: 100},
				},
				CrimesWitnessed:  []int{1, 2},
				LearnedRound:     50,
				LastUpdatedRound: 100,
			},
		},
	}

	if err := saveObserverFile(fc); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded := loadObserverFileFromDisk(99, "records_clerk_pell")
	if loaded == nil {
		t.Fatalf("expected non-nil load")
	}
	if loaded.ObserverMobId != 99 {
		t.Errorf("ObserverMobId mismatch: got %d", loaded.ObserverMobId)
	}
	if len(loaded.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(loaded.Records))
	}
	r := loaded.Records[0]
	if r.Subject != PlayerSubject(17) {
		t.Errorf("subject mismatch: got %+v", r.Subject)
	}
	if r.NameLearned != "smoketester" {
		t.Errorf("name mismatch: got %q", r.NameLearned)
	}
	if len(r.CrimesWitnessed) != 2 || r.CrimesWitnessed[0] != 1 || r.CrimesWitnessed[1] != 2 {
		t.Errorf("crimes mismatch: got %v", r.CrimesWitnessed)
	}
}

func TestLoadMissingFileReturnsNil(t *testing.T) {
	resetCache()
	if loadObserverFileFromDisk(404, "ghost") != nil {
		t.Errorf("expected nil for missing file")
	}
}
