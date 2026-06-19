package engine

import (
	"reflect"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// fakeMutatorSet records operations. The live state lives in the `current`
// slice the caller passes to reconcileList, so the fake needs no state of its
// own.
type fakeMutatorSet struct {
	ops []string
}

func newFake() *fakeMutatorSet { return &fakeMutatorSet{} }
func (f *fakeMutatorSet) Add(id string) bool {
	f.ops = append(f.ops, "add:"+id)
	return true
}
func (f *fakeMutatorSet) Remove(id string) bool {
	f.ops = append(f.ops, "remove:"+id)
	return true
}

func TestMutatorIdFor(t *testing.T) {
	if got := MutatorIdFor("storm"); got != "weather-storm" {
		t.Errorf("storm -> %q", got)
	}
	if MutatorIdFor(sim.Clear) != "" || MutatorIdFor("") != "" {
		t.Error("clear and unset must map to no mutator")
	}
}

func TestSeasonMutatorId(t *testing.T) {
	if got := SeasonMutatorId("temperate", "winter"); got != "season-temperate-winter" {
		t.Errorf("got %q", got)
	}
	if SeasonMutatorId("", "winter") != "" || SeasonMutatorId("temperate", "") != "" {
		t.Error("empty track or season must map to no mutator")
	}
}

func TestReconcileSeasonZone(t *testing.T) {
	// Same core as weather: stale season swapped for the current one,
	// weather-* ids untouched because the caller gathers season-* only.
	f := newFake()
	reconcileList(f, []string{"season-temperate-autumn"}, SeasonMutatorId("temperate", "winter"))
	want := []string{"remove:season-temperate-autumn", "add:season-temperate-winter"}
	if !reflect.DeepEqual(f.ops, want) {
		t.Errorf("ops = %v, want %v", f.ops, want)
	}
}

func TestReconcileList(t *testing.T) {
	// Stale storm + stray fog live; target is rain.
	f := newFake()
	reconcileList(f, []string{"weather-storm", "weather-fog"}, MutatorIdFor("rain"))
	want := []string{"remove:weather-storm", "remove:weather-fog", "add:weather-rain"}
	if !reflect.DeepEqual(f.ops, want) {
		t.Errorf("ops = %v, want %v", f.ops, want)
	}

	// Target already live: only the stray is removed.
	f = newFake()
	reconcileList(f, []string{"weather-rain", "weather-fog"}, MutatorIdFor("rain"))
	want = []string{"remove:weather-fog"}
	if !reflect.DeepEqual(f.ops, want) {
		t.Errorf("ops = %v, want %v", f.ops, want)
	}

	// Calm target: everything weather-* goes.
	f = newFake()
	reconcileList(f, []string{"weather-snow"}, MutatorIdFor(sim.Clear))
	want = []string{"remove:weather-snow"}
	if !reflect.DeepEqual(f.ops, want) {
		t.Errorf("ops = %v, want %v", f.ops, want)
	}
}

func TestApplyBuffOverrides(t *testing.T) {
	specs := map[string]*mutators.MutatorSpec{
		"weather-storm":        {MutatorId: "weather-storm", PlayerBuffIds: []int{59002}, MobBuffIds: []int{4}},
		"weather-storm-indoor": {MutatorId: "weather-storm-indoor"},
		"weather-blizzard":     {MutatorId: "weather-blizzard", PlayerBuffIds: []int{59001}},
	}
	lookup := func(id string) *mutators.MutatorSpec { return specs[id] }

	// Pre-mark the unknown-type warn so the warn-once path doesn't hit the
	// engine logger, which is uninitialized under `go test`.
	warnedOverrides["weather-hail"] = true
	t.Cleanup(func() { delete(warnedOverrides, "weather-hail") })

	src := map[string][]int{
		"storm":    {7, 8}, // replace
		"blizzard": {},     // explicit strip (key present, empty value)
		"hail":     {1},    // no such spec: ignored (warn-once)
	}
	if n := applyBuffOverrides(lookup, src); n != 2 {
		t.Fatalf("specs changed = %d, want 2", n)
	}
	if got := specs["weather-storm"].PlayerBuffIds; !reflect.DeepEqual(got, []int{7, 8}) {
		t.Errorf("storm PlayerBuffIds = %v, want [7 8]", got)
	}
	if got := specs["weather-storm"].MobBuffIds; !reflect.DeepEqual(got, []int{4}) {
		t.Errorf("storm MobBuffIds must be untouched: %v", got)
	}
	if got := specs["weather-blizzard"].PlayerBuffIds; len(got) != 0 {
		t.Errorf("blizzard buffs not stripped: %v", got)
	}
	// Indoor variants are buff-free by rule and never overridden.
	if got := specs["weather-storm-indoor"].PlayerBuffIds; got != nil {
		t.Errorf("indoor spec must be untouched: %v", got)
	}
	// The spec must not alias the config map's backing array.
	src["storm"][0] = 99
	if specs["weather-storm"].PlayerBuffIds[0] != 7 {
		t.Error("spec PlayerBuffIds aliases the override map")
	}
}
