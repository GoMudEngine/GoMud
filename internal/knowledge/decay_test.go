package knowledge

import (
	"testing"
)

func TestFrequentedRooms_Tally(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	r := uint64(0)
	roundForTest = func() uint64 { r++; return r }

	for _, room := range []int{462, 462, 463, 462, 464, 463, 462} {
		RecordObservation(399, PlayerSubject(17), room)
	}

	got := FrequentedRooms(399, PlayerSubject(17), 3)
	if len(got) != 3 {
		t.Fatalf("expected 3 rooms, got %d", len(got))
	}
	// Counts: 462=4, 463=2, 464=1
	if got[0].Room != 462 || got[0].Count != 4 {
		t.Errorf("top: got room=%d count=%d", got[0].Room, got[0].Count)
	}
	if got[1].Room != 463 || got[1].Count != 2 {
		t.Errorf("second: got room=%d count=%d", got[1].Room, got[1].Count)
	}
	if got[2].Room != 464 || got[2].Count != 1 {
		t.Errorf("third: got room=%d count=%d", got[2].Room, got[2].Count)
	}
}

func TestFrequentedRooms_TopKLargerThanUnique(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	r := uint64(0)
	roundForTest = func() uint64 { r++; return r }

	RecordObservation(400, PlayerSubject(17), 462)
	RecordObservation(400, PlayerSubject(17), 463)

	got := FrequentedRooms(400, PlayerSubject(17), 10)
	if len(got) != 2 {
		t.Errorf("expected 2 unique rooms, got %d", len(got))
	}
}

func TestFrequentedRooms_NoRecord(t *testing.T) {
	resetCache()
	got := FrequentedRooms(401, PlayerSubject(9999), 5)
	if len(got) != 0 {
		t.Errorf("expected empty result for unknown subject, got %v", got)
	}
}
