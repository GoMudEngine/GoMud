package inputhandlers

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/term"
)

func hasField(fields []term.MSSPField, name string) (term.MSSPField, bool) {
	for _, f := range fields {
		if f.Name == name {
			return f, true
		}
	}
	return term.MSSPField{}, false
}

func baseInputs() MSSPInputs {
	return MSSPInputs{
		Enabled: true, Name: "Delusions of Grandeur", Website: "https://www.dogmud.org",
		Genre: "Fantasy", Gameplay: []string{"Adventure", "Roleplaying"}, Status: "Open Beta",
		Language: "English", Family: "Custom", Location: "United States", Created: "2026",
		Contact: "", Hostname: "", Port: "",
		Players: 3, UptimeUnix: 1_700_000_000,
		Rooms: 1383, Mobiles: 640, Objects: 429, Skills: 10, Races: 42,
	}
}

func TestBuildMSSPFields_CoreAndLive(t *testing.T) {
	fields := buildMSSPFields(baseInputs())

	if f, ok := hasField(fields, "PLAYERS"); !ok || f.Values[0] != "3" {
		t.Fatalf("PLAYERS = %v (ok=%v)", f.Values, ok)
	}
	if f, ok := hasField(fields, "UPTIME"); !ok || f.Values[0] != "1700000000" {
		t.Fatalf("UPTIME = %v (ok=%v)", f.Values, ok)
	}
	if f, ok := hasField(fields, "NAME"); !ok || f.Values[0] != "Delusions of Grandeur" {
		t.Fatalf("NAME = %v (ok=%v)", f.Values, ok)
	}
	if f, ok := hasField(fields, "GAMEPLAY"); !ok || len(f.Values) != 2 {
		t.Fatalf("GAMEPLAY = %v (ok=%v)", f.Values, ok)
	}
	if f, ok := hasField(fields, "GMCP"); !ok || f.Values[0] != "1" {
		t.Fatalf("GMCP flag = %v (ok=%v)", f.Values, ok)
	}
	if f, ok := hasField(fields, "ROOMS"); !ok || f.Values[0] != "1383" {
		t.Fatalf("ROOMS = %v (ok=%v)", f.Values, ok)
	}
}

func TestBuildMSSPFields_OmitsEmptyAndZero(t *testing.T) {
	in := baseInputs()
	in.Objects = 0 // a zero world count must be omitted
	fields := buildMSSPFields(in)

	if _, ok := hasField(fields, "CONTACT"); ok {
		t.Error("empty CONTACT must be omitted")
	}
	if _, ok := hasField(fields, "HOSTNAME"); ok {
		t.Error("empty HOSTNAME must be omitted")
	}
	if _, ok := hasField(fields, "OBJECTS"); ok {
		t.Error("zero OBJECTS count must be omitted")
	}
	// PLAYERS = 0 is still meaningful and must be sent.
	in2 := baseInputs()
	in2.Players = 0
	if f, ok := hasField(buildMSSPFields(in2), "PLAYERS"); !ok || f.Values[0] != "0" {
		t.Fatalf("PLAYERS=0 should still be sent, got %v (ok=%v)", f.Values, ok)
	}
}

func TestBuildMSSPFields_DisabledReturnsNil(t *testing.T) {
	in := baseInputs()
	in.Enabled = false
	if fields := buildMSSPFields(in); fields != nil {
		t.Fatalf("disabled must return nil, got %d fields", len(fields))
	}
}
