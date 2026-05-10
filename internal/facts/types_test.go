package facts

import (
	"testing"
)

func TestStatusConstants(t *testing.T) {
	if string(StatusActive) != "active" {
		t.Errorf("StatusActive: got %s", StatusActive)
	}
	if string(StatusWithdrawn) != "withdrawn" {
		t.Errorf("StatusWithdrawn: got %s", StatusWithdrawn)
	}
	if string(StatusExpired) != "expired" {
		t.Errorf("StatusExpired: got %s", StatusExpired)
	}
}

func TestSourceConstants(t *testing.T) {
	if string(SourceWitnessed) != "witnessed" {
		t.Errorf("SourceWitnessed: %s", SourceWitnessed)
	}
	if string(SourceTold) != "told" {
		t.Errorf("SourceTold: %s", SourceTold)
	}
	if string(SourceDeduced) != "deduced" {
		t.Errorf("SourceDeduced: %s", SourceDeduced)
	}
	if string(SourceUnknown) != "unknown" {
		t.Errorf("SourceUnknown: %s", SourceUnknown)
	}
}
