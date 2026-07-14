package configs

import "testing"

func TestMSSPConfigDefaults(t *testing.T) {
	s := Server{}
	s.Validate()

	if s.MSSP.Website != "https://www.dogmud.org" {
		t.Errorf("Website default = %q", s.MSSP.Website)
	}
	if s.MSSP.Genre != "Fantasy" {
		t.Errorf("Genre default = %q", s.MSSP.Genre)
	}
	if s.MSSP.Status != "Open Beta" {
		t.Errorf("Status default = %q", s.MSSP.Status)
	}
	if len(s.MSSP.Gameplay) != 2 || s.MSSP.Gameplay[0] != "Adventure" || s.MSSP.Gameplay[1] != "Roleplaying" {
		t.Errorf("Gameplay default = %v", s.MSSP.Gameplay)
	}
	// Privacy: these must stay empty unless the operator sets them.
	if s.MSSP.Contact != "" {
		t.Errorf("Contact should default empty, got %q", s.MSSP.Contact)
	}
	if s.MSSP.Hostname != "" || s.MSSP.Port != "" {
		t.Errorf("Hostname/Port should default empty, got %q/%q", s.MSSP.Hostname, s.MSSP.Port)
	}
}
