package sim

import (
	"reflect"
	"testing"
)

// twoZoneGraph: A(plains) <-> B(mountain), one edge weight 1.
func twoZoneGraph() *Graph {
	return &Graph{
		Nodes: map[string]ZoneNode{
			"A": {Zone: "A", Biome: "plains", Rooms: 3, HasOutdoor: true},
			"B": {Zone: "B", Biome: "mountain", Rooms: 3, HasOutdoor: true},
		},
		Edges: []Edge{{A: "A", B: "B", Weight: 1}},
	}
}

// These tests exercise the tick HELPERS directly (not the full Step), so they
// stay valid as later tasks add movement/spawning to Step.

func TestAgeAndFeedback_DrainsAndClamps(t *testing.T) {
	g := twoZoneGraph()
	// Front in mountain zone B (influence intensityDelta -0.15).
	fronts := []Front{{Id: 1, Type: "storm", Zone: "B", Intensity: 0.1, Moisture: 0.5, MaxAge: 24}}
	out := ageAndFeedback(fronts, g, DefaultClimate())
	if out[0].Intensity != 0 {
		t.Errorf("mountain feedback should drain 0.1 to 0 (clamped), got %v", out[0].Intensity)
	}
	if out[0].Age != 1 {
		t.Errorf("age should increment to 1, got %d", out[0].Age)
	}
}

func TestRemoveDead_DropsZeroIntensityAndAged(t *testing.T) {
	cfg := DefaultConfig()
	fronts := []Front{
		{Id: 1, Intensity: 0},                              // dead: no intensity
		{Id: 2, Intensity: 0.5, Age: cfg.FrontHardAge + 1}, // dead: too old
		{Id: 3, Intensity: 0.5, Age: 1},                    // alive
	}
	out := removeDead(fronts, cfg)
	if len(out) != 1 || out[0].Id != 3 {
		t.Errorf("only front 3 should survive, got %+v", out)
	}
}

func TestResolveWeather_AreaCoverageScalesWithIntensity(t *testing.T) {
	// Line A-B-C-D. Defaults: falloff 0.5, minProjected 0.15, maxRadius 2.
	g := &Graph{
		Nodes: map[string]ZoneNode{"A": {Zone: "A"}, "B": {Zone: "B"}, "C": {Zone: "C"}, "D": {Zone: "D"}},
		Edges: []Edge{{A: "A", B: "B", Weight: 1}, {A: "B", B: "C", Weight: 1}, {A: "C", B: "D", Weight: 1}},
	}
	cfg := DefaultConfig()

	// Strong storm at B (0.9): B=0.9, A&C=0.45, D=0.225 — all four covered (<= 2 hops).
	strong := resolveWeather(g, []Front{{Id: 1, Type: "storm", Zone: "B", Intensity: 0.9}}, cfg)
	for _, z := range []ZoneId{"A", "B", "C", "D"} {
		if strong[z] != "storm" {
			t.Errorf("strong storm should cover %s, got %q", z, strong[z])
		}
	}

	// Weak front at B (0.2): neighbors project 0.1 < 0.15 — only B covered.
	weak := resolveWeather(g, []Front{{Id: 1, Type: "fog", Zone: "B", Intensity: 0.2}}, cfg)
	if weak["B"] != "fog" {
		t.Errorf("weak front should hold B, got %q", weak["B"])
	}
	if weak["A"] != Clear || weak["C"] != Clear {
		t.Errorf("weak front (0.2) should NOT spread; A=%q C=%q", weak["A"], weak["C"])
	}
}

func TestResolveWeather_StrongerEffectiveIntensityWins(t *testing.T) {
	g := &Graph{Nodes: map[string]ZoneNode{"A": {Zone: "A"}, "B": {Zone: "B"}}, Edges: []Edge{{A: "A", B: "B", Weight: 1}}}
	cfg := DefaultConfig()
	// storm centered at A (0.9 -> projects 0.45 onto B); fog local at B (0.5).
	fronts := []Front{{Id: 1, Type: "storm", Zone: "A", Intensity: 0.9}, {Id: 2, Type: "fog", Zone: "B", Intensity: 0.5}}
	w := resolveWeather(g, fronts, cfg)
	if w["A"] != "storm" {
		t.Errorf("A should be storm (0.9), got %q", w["A"])
	}
	if w["B"] != "fog" {
		t.Errorf("B: local fog 0.5 should beat storm's projected 0.45, got %q", w["B"])
	}
}

func TestResolveWeather_FrontlessZonesAreClear(t *testing.T) {
	g := twoZoneGraph()
	w := resolveWeather(g, nil, DefaultConfig())
	if w["A"] != Clear || w["B"] != Clear {
		t.Errorf("no fronts -> all zones Clear, got %+v", w)
	}
}

func TestDiffWeather_ReportsOnlyChanges(t *testing.T) {
	prev := map[ZoneId]WeatherType{"A": Clear, "B": "rain"}
	next := map[ZoneId]WeatherType{"A": "storm", "B": "rain"}
	diff := diffWeather(prev, next)
	if len(diff.Changes) != 1 || diff.Changes[0].Zone != "A" ||
		diff.Changes[0].From != Clear || diff.Changes[0].To != "storm" {
		t.Errorf("expected one A: clear->storm change, got %+v", diff.Changes)
	}
}

func TestStep_DeterministicAndCoversAllZones(t *testing.T) {
	g := twoZoneGraph()
	cfg := DefaultConfig()
	st := State{RNGState: 7, NextID: 1, Weather: map[ZoneId]WeatherType{}}
	a, _ := Step(st, g, DefaultClimate(), cfg, Clock{Round: 1})
	b, _ := Step(st, g, DefaultClimate(), cfg, Clock{Round: 1})
	if a.RNGState != b.RNGState || len(a.Fronts) != len(b.Fronts) {
		t.Fatal("Step must be deterministic for identical inputs")
	}
	for _, z := range g.Zones() {
		if _, ok := a.Weather[z]; !ok {
			t.Errorf("zone %q missing from resolved weather", z)
		}
	}
}

func TestMoveFronts_HighResistanceZoneLingers(t *testing.T) {
	// Mountain (resistance 0.5) vs plains (0.05). Run many independent single
	// fronts one tick each; the mountain front should move far less often.
	g := twoZoneGraph()
	climate := DefaultClimate()
	cfg := DefaultConfig()

	movesFrom := func(zone ZoneId, seed uint64) int {
		moved := 0
		for i := 0; i < 200; i++ {
			rng := NewRNG(seed + uint64(i))
			f := []Front{{Id: 1, Type: "storm", Zone: zone, Intensity: 0.8, Moisture: 0.5, MaxAge: 24}}
			out := moveFronts(f, g, climate, cfg, rng)
			if out[0].Zone != zone {
				moved++
			}
		}
		return moved
	}
	mountainMoves := movesFrom("B", 1000)
	plainsMoves := movesFrom("A", 1000)
	if !(plainsMoves > mountainMoves) {
		t.Errorf("plains front should move more often than mountain front; plains=%d mountain=%d",
			plainsMoves, mountainMoves)
	}
}

func TestMoveFronts_RecordsHistoryOnMove(t *testing.T) {
	// A 3-zone line A-B-C; a front in B that moves should append its old zone to
	// History. Force movement by using zero resistance everywhere via a custom climate.
	g := &Graph{
		Nodes: map[string]ZoneNode{
			"A": {Zone: "A", Biome: "flat"}, "B": {Zone: "B", Biome: "flat"}, "C": {Zone: "C", Biome: "flat"},
		},
		Edges: []Edge{{A: "A", B: "B", Weight: 1}, {A: "B", B: "C", Weight: 1}},
	}
	climate := Climate{"flat": {Weather: map[WeatherType]float64{"clear": 1}, Influence: WeatherInfluence{MovementResistance: 0}, SpawnWeight: 1}, "default": {Weather: map[WeatherType]float64{"clear": 1}}}
	cfg := DefaultConfig()

	movedSomewhere := false
	for i := 0; i < 50; i++ {
		rng := NewRNG(uint64(i) + 1)
		f := []Front{{Id: 1, Type: "storm", Zone: "B", Intensity: 0.8, MaxAge: 24}}
		out := moveFronts(f, g, climate, cfg, rng)
		if out[0].Zone != "B" {
			movedSomewhere = true
			if len(out[0].History) == 0 || out[0].History[len(out[0].History)-1] != "B" {
				t.Fatalf("on move, History should end with the departed zone 'B'; got %v", out[0].History)
			}
		}
	}
	if !movedSomewhere {
		t.Fatal("with zero resistance, the front should sometimes move")
	}
}

func TestEvolveTypes_AdaptsToNewBiome(t *testing.T) {
	// A front that just moved INTO tundra (history shows it came from elsewhere)
	// should, over many seeds, sometimes become a tundra-typical type (snow).
	g := &Graph{Nodes: map[string]ZoneNode{
		"T": {Zone: "T", Biome: "tundra"}, "P": {Zone: "P", Biome: "plains"},
	}}
	climate := DefaultClimate()
	becameSnowy := 0
	for i := 0; i < 300; i++ {
		rng := NewRNG(uint64(i) + 1)
		// Type "storm" entering tundra (storm is not in tundra's profile).
		f := []Front{{Id: 1, Type: "storm", Zone: "T", Intensity: 0.8, History: []ZoneId{"P"}}}
		out := evolveTypes(f, g, climate, rng)
		if out[0].Type == "snow" || out[0].Type == "blizzard" {
			becameSnowy++
		}
	}
	if becameSnowy == 0 {
		t.Error("a storm entering tundra should sometimes evolve toward snow/blizzard")
	}
}

func TestEvolveTypes_StationaryFrontKeepsType(t *testing.T) {
	// A front whose current zone == last history entry did NOT move this tick;
	// it should keep its type (no re-roll).
	g := &Graph{Nodes: map[string]ZoneNode{"T": {Zone: "T", Biome: "tundra"}}}
	for i := 0; i < 50; i++ {
		rng := NewRNG(uint64(i) + 1)
		f := []Front{{Id: 1, Type: "storm", Zone: "T", Intensity: 0.8, History: []ZoneId{"T"}}}
		out := evolveTypes(f, g, DefaultClimate(), rng)
		if out[0].Type != "storm" {
			t.Fatalf("stationary front should keep its type, got %q", out[0].Type)
		}
	}
}

func TestEvolveTypes_LingeringFrontKeepsType(t *testing.T) {
	// Pipeline check: a front in a max-resistance zone always lingers, so after
	// moveFronts + evolveTypes its type must be unchanged. "storm" is NOT in the
	// stuck biome's weather set, so a wrong re-roll would change it.
	g := &Graph{
		Nodes: map[string]ZoneNode{"S": {Zone: "S", Biome: "stuck"}, "X": {Zone: "X", Biome: "stuck"}},
		Edges: []Edge{{A: "S", B: "X", Weight: 1}},
	}
	climate := Climate{
		"stuck":   {Weather: map[WeatherType]float64{"clear": 5, "snow": 5}, Influence: WeatherInfluence{MovementResistance: 1.0}, SpawnWeight: 1},
		"default": {Weather: map[WeatherType]float64{"clear": 1}},
	}
	cfg := DefaultConfig()
	for i := 0; i < 50; i++ {
		rng := NewRNG(uint64(i) + 1)
		f := []Front{{Id: 1, Type: "storm", Zone: "S", Intensity: 0.8, MaxAge: 24}}
		f = moveFronts(f, g, climate, cfg, rng)
		f = evolveTypes(f, g, climate, rng)
		if f[0].Zone != "S" {
			t.Fatalf("resistance 1.0 should always linger; moved to %s (seed %d)", f[0].Zone, i)
		}
		if f[0].Type != "storm" {
			t.Fatalf("lingering front must keep its type, got %q (seed %d)", f[0].Type, i)
		}
	}
}

func TestMoveFronts_NoBacktrackToLastZone(t *testing.T) {
	// 3-zone chain A-B-C, zero resistance (always moves). A front at B that came
	// from A (History ["A"]) must move to C, never back to A.
	g := &Graph{
		Nodes: map[string]ZoneNode{"A": {Zone: "A", Biome: "flat"}, "B": {Zone: "B", Biome: "flat"}, "C": {Zone: "C", Biome: "flat"}},
		Edges: []Edge{{A: "A", B: "B", Weight: 1}, {A: "B", B: "C", Weight: 1}},
	}
	climate := Climate{
		"flat":    {Weather: map[WeatherType]float64{"clear": 1}, Influence: WeatherInfluence{MovementResistance: 0}, SpawnWeight: 1},
		"default": {Weather: map[WeatherType]float64{"clear": 1}},
	}
	cfg := DefaultConfig()
	for i := 0; i < 50; i++ {
		rng := NewRNG(uint64(i) + 1)
		f := []Front{{Id: 1, Type: "storm", Zone: "B", Intensity: 0.8, MaxAge: 24, History: []ZoneId{"A"}}}
		f = moveFronts(f, g, climate, cfg, rng)
		if f[0].Zone == "A" {
			t.Fatalf("front should not backtrack to A (its last zone); seed %d", i)
		}
	}
}

func TestSpawnFronts_RespectsBudget(t *testing.T) {
	g := twoZoneGraph()
	climate := DefaultClimate()
	cfg := DefaultConfig()
	cfg.MaxActiveFronts = 2
	cfg.SpawnChance = 1.0 // always try to spawn

	st := State{RNGState: 1, NextID: 1, Weather: map[ZoneId]WeatherType{}}
	// Run many ticks; fronts accumulate but never exceed the budget.
	for i := 0; i < 50; i++ {
		var diff StateDiff
		st, diff = Step(st, g, climate, cfg, Clock{Round: uint64(i + 1)})
		_ = diff
		if len(st.Fronts) > cfg.MaxActiveFronts {
			t.Fatalf("front count %d exceeded budget %d at tick %d", len(st.Fronts), cfg.MaxActiveFronts, i)
		}
	}
}

func TestSpawnFronts_AssignsIncrementingIDs(t *testing.T) {
	g := twoZoneGraph()
	cfg := DefaultConfig()
	cfg.SpawnChance = 1.0
	rng := NewRNG(5)
	fronts, nextID := spawnFronts(nil, g, DefaultClimate(), cfg, rng, FrontId(10))
	if len(fronts) != 1 {
		t.Fatalf("expected one spawned front, got %d", len(fronts))
	}
	if fronts[0].Id != 10 {
		t.Errorf("spawned front should take nextID 10, got %d", fronts[0].Id)
	}
	if nextID != 11 {
		t.Errorf("nextID should advance to 11, got %d", nextID)
	}
	if fronts[0].Intensity <= 0 {
		t.Error("spawned front should start with positive intensity")
	}
}

func runTicks(seed uint64, n int) State {
	g := twoZoneGraph()
	climate := DefaultClimate()
	cfg := DefaultConfig()
	st := State{RNGState: seed, NextID: 1, Weather: map[ZoneId]WeatherType{}}
	for i := 0; i < n; i++ {
		st, _ = Step(st, g, climate, cfg, Clock{Round: uint64(i + 1)})
	}
	return st
}

func TestStep_Deterministic(t *testing.T) {
	a := runTicks(12345, 40)
	b := runTicks(12345, 40)
	// Same seed + world + tick count => byte-for-byte identical State
	// (fronts incl. History, weather, RNG cursor, NextID, Round).
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("identical runs diverged:\n a=%+v\n b=%+v", a, b)
	}
}

func TestStep_StormDiesCrossingMountains(t *testing.T) {
	// A long mountain chain. A strong storm seeded at one end must lose intensity
	// as it crosses and eventually die — terrain is not a passive recipient.
	nodes := map[string]ZoneNode{}
	var edges []Edge
	const N = 6
	prev := ""
	for i := 0; i < N; i++ {
		z := string(rune('A' + i))
		nodes[z] = ZoneNode{Zone: z, Biome: "mountain", HasOutdoor: true}
		if prev != "" {
			edges = append(edges, Edge{A: prev, B: z, Weight: 1})
		}
		prev = z
	}
	g := &Graph{Nodes: nodes, Edges: edges}
	climate := DefaultClimate()
	cfg := DefaultConfig()
	cfg.SpawnChance = 0 // isolate the seeded storm

	st := State{
		RNGState: 99, NextID: 2,
		Fronts:  []Front{{Id: 1, Type: "storm", Zone: "A", Intensity: 0.9, Moisture: 0.9, MaxAge: 100}},
		Weather: map[ZoneId]WeatherType{},
	}
	start := st.Fronts[0].Intensity
	died := false
	minSeen := start
	for i := 0; i < 40; i++ {
		st, _ = Step(st, g, climate, cfg, Clock{Round: uint64(i + 1)})
		if len(st.Fronts) == 0 {
			died = true
			break
		}
		if st.Fronts[0].Intensity < minSeen {
			minSeen = st.Fronts[0].Intensity
		}
	}
	if !died {
		t.Fatalf("storm should die crossing the mountains; final intensity %v (min seen %v)",
			lastIntensity(st), minSeen)
	}
}

func lastIntensity(st State) float64 {
	if len(st.Fronts) == 0 {
		return 0
	}
	return st.Fronts[0].Intensity
}
