package mobs

import (
	"testing"
)

func TestApplyScheduleSpawnOverride_PlacesAtCurrentSegment(t *testing.T) {
	s := kerraTestSchedule()
	registerScheduleForTest(s)
	defer unregisterScheduleForTest(s.Id)

	got := applyScheduleSpawnOverride("thornwall_smith", 9999 /* homeRoomId */, 10 /* hour */)
	if got != 5678 {
		t.Errorf("hour 10: expected forge room 5678, got %d", got)
	}

	got = applyScheduleSpawnOverride("thornwall_smith", 9999, 19)
	if got != 9012 {
		t.Errorf("hour 19: expected tavern room 9012, got %d", got)
	}

	got = applyScheduleSpawnOverride("thornwall_smith", 9999, 23)
	if got != 1234 {
		t.Errorf("hour 23: expected loft room 1234, got %d", got)
	}
}

func TestApplyScheduleSpawnOverride_NoScheduleReturnsHome(t *testing.T) {
	got := applyScheduleSpawnOverride("", 9999, 10)
	if got != 9999 {
		t.Errorf("no schedule: expected home %d, got %d", 9999, got)
	}
}

func TestApplyScheduleSpawnOverride_UnknownScheduleReturnsHome(t *testing.T) {
	got := applyScheduleSpawnOverride("definitely_not_real", 9999, 10)
	if got != 9999 {
		t.Errorf("unknown schedule: expected home %d, got %d", 9999, got)
	}
}
