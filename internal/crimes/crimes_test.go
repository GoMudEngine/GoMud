package crimes

import "testing"

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
