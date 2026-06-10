package engine

import (
	"reflect"
	"testing"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// fakeMutatorSet records operations; "known:" prefixed ids Add successfully.
type fakeMutatorSet struct {
	ops  []string
	live map[string]bool
}

func newFake(live ...string) *fakeMutatorSet {
	f := &fakeMutatorSet{live: map[string]bool{}}
	for _, id := range live {
		f.live[id] = true
	}
	return f
}
func (f *fakeMutatorSet) Add(id string) bool {
	f.ops = append(f.ops, "add:"+id)
	f.live[id] = true
	return true
}
func (f *fakeMutatorSet) Remove(id string) bool {
	f.ops = append(f.ops, "remove:"+id)
	delete(f.live, id)
	return true
}
func (f *fakeMutatorSet) Has(id string) bool { return f.live[id] }

func TestMutatorIdFor(t *testing.T) {
	if got := MutatorIdFor("storm"); got != "weather-storm" {
		t.Errorf("storm -> %q", got)
	}
	if MutatorIdFor(sim.Clear) != "" || MutatorIdFor("") != "" {
		t.Error("clear and unset must map to no mutator")
	}
}

func TestApplyChange(t *testing.T) {
	cases := []struct {
		name     string
		live     []string
		from, to sim.WeatherType
		wantOps  []string
	}{
		{"calm to storm", nil, "", "storm", []string{"add:weather-storm"}},
		{"clear to rain", nil, sim.Clear, "rain", []string{"add:weather-rain"}},
		{"storm to rain", []string{"weather-storm"}, "storm", "rain", []string{"remove:weather-storm", "add:weather-rain"}},
		{"storm to clear", []string{"weather-storm"}, "storm", sim.Clear, []string{"remove:weather-storm"}},
		{"already live: no duplicate add", []string{"weather-rain"}, "", "rain", nil},
	}
	for _, c := range cases {
		f := newFake(c.live...)
		applyChange(f, c.from, c.to)
		if !reflect.DeepEqual(f.ops, c.wantOps) {
			t.Errorf("%s: ops = %v, want %v", c.name, f.ops, c.wantOps)
		}
	}
}

func TestReconcileZone(t *testing.T) {
	// Stale storm + stray fog live; target is rain.
	f := newFake("weather-storm", "weather-fog")
	reconcileZone(f, []string{"weather-storm", "weather-fog"}, "rain")
	want := []string{"remove:weather-storm", "remove:weather-fog", "add:weather-rain"}
	if !reflect.DeepEqual(f.ops, want) {
		t.Errorf("ops = %v, want %v", f.ops, want)
	}

	// Target already live: only the stray is removed.
	f = newFake("weather-rain", "weather-fog")
	reconcileZone(f, []string{"weather-rain", "weather-fog"}, "rain")
	want = []string{"remove:weather-fog"}
	if !reflect.DeepEqual(f.ops, want) {
		t.Errorf("ops = %v, want %v", f.ops, want)
	}

	// Calm target: everything weather-* goes.
	f = newFake("weather-snow")
	reconcileZone(f, []string{"weather-snow"}, sim.Clear)
	want = []string{"remove:weather-snow"}
	if !reflect.DeepEqual(f.ops, want) {
		t.Errorf("ops = %v, want %v", f.ops, want)
	}
}
