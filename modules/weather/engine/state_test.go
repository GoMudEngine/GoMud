package engine

import (
	"testing"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

func TestStateCodecRoundTrip(t *testing.T) {
	s := sim.NewState(42)
	s.Fronts = []sim.Front{{Id: 1, Type: "storm", Zone: "A", Intensity: 0.7, History: []string{"B"}}}
	s.Weather = map[string]sim.WeatherType{"A": "storm", "B": sim.Clear}
	s.Round = 99

	b, err := EncodeState(s)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := DecodeState(b)
	if !ok {
		t.Fatal("round-trip decode failed")
	}
	if got.Round != 99 || got.RNGState != 42 || len(got.Fronts) != 1 || got.Weather["A"] != "storm" {
		t.Fatalf("state mangled: %+v", got)
	}
}

func TestDecodeStateRejectsBadInput(t *testing.T) {
	if _, ok := DecodeState(nil); ok {
		t.Error("nil must not decode")
	}
	if _, ok := DecodeState([]byte("not json")); ok {
		t.Error("garbage must not decode")
	}
	if _, ok := DecodeState([]byte(`{"version":999,"state":{}}`)); ok {
		t.Error("future version must not decode (forces a clean fresh state)")
	}
}
