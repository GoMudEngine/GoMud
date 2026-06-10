package sim

import (
	"reflect"
	"testing"
)

func TestStateJSONRoundTrip(t *testing.T) {
	st := State{
		Round:    7,
		RNGState: 0xDEADBEEF,
		NextID:   3,
		Fronts: []Front{
			{Id: 1, Type: "storm", Zone: "A", Intensity: 0.5, Moisture: 0.6, Age: 4, MaxAge: 24, History: []ZoneId{"B", "A"}},
			{Id: 2, Type: "fog", Zone: "C", Intensity: 0.2, Moisture: 0.9, Age: 1, MaxAge: 12, History: nil},
		},
		Weather: map[ZoneId]WeatherType{"A": "storm", "B": Clear, "C": "fog"},
	}
	b, err := st.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	got, err := StateFromJSON(b)
	if err != nil {
		t.Fatalf("StateFromJSON: %v", err)
	}
	if !reflect.DeepEqual(st, got) {
		t.Errorf("round trip mismatch:\n want %+v\n got  %+v", st, got)
	}
}

func TestStateFromJSONError(t *testing.T) {
	if _, err := StateFromJSON([]byte("{bad")); err == nil {
		t.Error("StateFromJSON should error on malformed JSON")
	}
}

func TestNewState(t *testing.T) {
	s := NewState(99)
	if s.RNGState != 99 {
		t.Errorf("seed not stored: %d", s.RNGState)
	}
	if s.NextID != 1 {
		t.Errorf("NextID should start at 1, got %d", s.NextID)
	}
	if s.Weather == nil || len(s.Weather) != 0 {
		t.Errorf("Weather should be empty non-nil map: %v", s.Weather)
	}
	if len(s.Fronts) != 0 {
		t.Errorf("Fronts should be empty: %v", s.Fronts)
	}
}

func TestDeriveSeedStableAndWorldSensitive(t *testing.T) {
	g1 := &Graph{Nodes: map[string]ZoneNode{"A": {}, "B": {}}}
	g1b := &Graph{Nodes: map[string]ZoneNode{"B": {}, "A": {}}} // same zones, different insertion
	g2 := &Graph{Nodes: map[string]ZoneNode{"A": {}, "C": {}}}
	if DeriveSeed(g1) != DeriveSeed(g1b) {
		t.Error("same zone set must derive the same seed")
	}
	if DeriveSeed(g1) == DeriveSeed(g2) {
		t.Error("different worlds should derive different seeds")
	}
	if DeriveSeed(g1) == 0 {
		t.Error("derived seed should be non-zero for a non-empty world")
	}
}
