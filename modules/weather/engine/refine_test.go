package engine

import (
	"reflect"
	"testing"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

func TestRoomWantId(t *testing.T) {
	cases := []struct {
		weather sim.WeatherType
		indoor  bool
		want    string
	}{
		{"storm", false, "weather-storm"},
		{"storm", true, "weather-storm-indoor"},
		{sim.Clear, false, ""},
		{sim.Clear, true, ""},
		{"", true, ""},
	}
	for _, c := range cases {
		if got := roomWantId(c.weather, c.indoor); got != c.want {
			t.Errorf("(%q,%v): got %q want %q", c.weather, c.indoor, got, c.want)
		}
	}
}

func TestRefineRoomList(t *testing.T) {
	// Tests the composition RefineRoom uses in production:
	// reconcileList(ms, current, roomWantId(w, indoor)).

	// Outdoor room moving storm -> rain (stale storm stripped, rain added).
	f := newFake()
	reconcileList(f, []string{"weather-storm"}, roomWantId("rain", false))
	want := []string{"remove:weather-storm", "add:weather-rain"}
	if !reflect.DeepEqual(f.ops, want) {
		t.Errorf("ops = %v, want %v", f.ops, want)
	}

	// Indoor room with a stale OUTDOOR id heals to the indoor variant.
	f = newFake()
	reconcileList(f, []string{"weather-rain"}, roomWantId("rain", true))
	want = []string{"remove:weather-rain", "add:weather-rain-indoor"}
	if !reflect.DeepEqual(f.ops, want) {
		t.Errorf("ops = %v, want %v", f.ops, want)
	}

	// Clear strips everything.
	f = newFake()
	reconcileList(f, []string{"weather-rain-indoor"}, roomWantId(sim.Clear, true))
	want = []string{"remove:weather-rain-indoor"}
	if !reflect.DeepEqual(f.ops, want) {
		t.Errorf("ops = %v, want %v", f.ops, want)
	}

	// Steady state, outdoor: the mutator is already correct -> ZERO ops.
	// Re-add would reset spawn timing and re-fire the entry message every
	// tick a player stands in the room. The fake never appends, so ops
	// stays nil; len covers nil and empty alike.
	f = newFake()
	reconcileList(f, []string{"weather-rain"}, roomWantId("rain", false))
	if len(f.ops) != 0 {
		t.Errorf("steady-state outdoor: ops = %v, want none", f.ops)
	}

	// Steady state, indoor twin.
	f = newFake()
	reconcileList(f, []string{"weather-rain-indoor"}, roomWantId("rain", true))
	if len(f.ops) != 0 {
		t.Errorf("steady-state indoor: ops = %v, want none", f.ops)
	}
}
