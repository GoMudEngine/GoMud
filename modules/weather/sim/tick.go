package sim

import "sort"

// Step advances the simulation one coarse tick. It is a pure function of its
// inputs: all randomness comes from prev.RNGState. It returns the next State
// (fronts advanced, weather resolved, RNG cursor written back) and the StateDiff
// of per-zone weather changes the engine layer should apply.
func Step(prev State, g *Graph, climate Climate, cfg Config, now Clock) (State, StateDiff) {
	rng := NewRNG(prev.RNGState)

	fronts := cloneFronts(prev.Fronts)
	fronts = ageAndFeedback(fronts, g, climate)
	fronts = moveFronts(fronts, g, climate, cfg, rng)
	fronts = evolveTypes(fronts, g, climate, rng)
	fronts = removeDead(fronts, cfg)

	nextID := prev.NextID
	fronts, nextID = spawnFronts(fronts, g, climate, cfg, rng, nextID)

	weather := resolveWeather(g, fronts, cfg)
	diff := diffWeather(prev.Weather, weather)

	return State{
		Round:    now.Round,
		RNGState: rng.State(),
		NextID:   nextID,
		Fronts:   fronts,
		Weather:  weather,
	}, diff
}

func cloneFronts(in []Front) []Front {
	out := make([]Front, len(in))
	copy(out, in)
	for i := range out {
		out[i].History = append([]ZoneId(nil), in[i].History...)
	}
	return out
}

// ageAndFeedback ages each front and applies the influence of the biome of the
// zone it currently occupies (the weather <- biome half of the feedback loop),
// plus an age-based decay once past MaxAge. Intensity/Moisture are clamped.
//
// Mutates the elements of fronts in place; callers must pass a copy they own
// (Step does, via cloneFronts).
func ageAndFeedback(fronts []Front, g *Graph, climate Climate) []Front {
	for i := range fronts {
		f := &fronts[i]
		f.Age++
		inf := climate.For(g.Nodes[f.Zone].Biome).Influence
		f.Intensity += inf.IntensityDelta
		f.Moisture += inf.MoistureDelta
		if f.MaxAge > 0 && f.Age > f.MaxAge {
			f.Intensity -= 0.05 * float64(f.Age-f.MaxAge)
		}
		f.Intensity = clamp01(f.Intensity)
		f.Moisture = clamp01(f.Moisture)
	}
	return fronts
}

// removeDead drops fronts whose intensity has reached zero or that exceed the
// hard age cap. It filters in place (reusing the backing array), so callers must
// pass a copy they own (Step does, via cloneFronts).
func removeDead(fronts []Front, cfg Config) []Front {
	out := fronts[:0]
	for _, f := range fronts {
		if f.Intensity <= 0 {
			continue
		}
		if cfg.FrontHardAge > 0 && f.Age > cfg.FrontHardAge {
			continue
		}
		out = append(out, f)
	}
	return out
}

// resolveWeather computes per-zone weather with intensity-scaled area coverage:
// each front projects onto zones within cfg.MaxFrontRadius hops of its center,
// with projected intensity = Intensity * CoverageFalloff^hops; a zone is covered
// only while that stays >= cfg.MinProjected. Per zone, the front projecting the
// highest effective intensity wins (deterministic tie-break by lowest FrontId).
// Zones no front reaches are Clear. Stronger fronts naturally cover more zones.
func resolveWeather(g *Graph, fronts []Front, cfg Config) map[ZoneId]WeatherType {
	type claim struct {
		eff   float64
		id    FrontId
		wtype WeatherType
	}
	best := map[ZoneId]claim{}
	for i := range fronts {
		f := &fronts[i]
		for zone, hops := range zonesWithin(g, f.Zone, cfg.MaxFrontRadius) {
			eff := f.Intensity * pow(cfg.CoverageFalloff, hops)
			if eff < cfg.MinProjected {
				continue
			}
			cur, ok := best[zone]
			if !ok || eff > cur.eff || (eff == cur.eff && f.Id < cur.id) {
				best[zone] = claim{eff: eff, id: f.Id, wtype: f.Type}
			}
		}
	}
	out := make(map[ZoneId]WeatherType, len(g.Nodes))
	for _, z := range g.Zones() {
		if c, ok := best[z]; ok {
			out[z] = c.wtype
		} else {
			out[z] = Clear
		}
	}
	return out
}

// zonesWithin returns each zone reachable from center within maxRadius hops,
// mapped to its hop-distance (center is 0). Deterministic shortest-path BFS;
// its result is order-independent, so resolveWeather stays deterministic.
func zonesWithin(g *Graph, center ZoneId, maxRadius int) map[ZoneId]int {
	dist := map[ZoneId]int{center: 0}
	frontier := []ZoneId{center}
	for d := 0; d < maxRadius; d++ {
		var next []ZoneId
		for _, z := range frontier {
			for _, e := range g.Neighbors(z) {
				if _, seen := dist[e.B]; !seen {
					dist[e.B] = d + 1
					next = append(next, e.B)
				}
			}
		}
		frontier = next
	}
	return dist
}

// pow raises base to a non-negative integer power (small exponents; avoids
// importing math).
func pow(base float64, exp int) float64 {
	r := 1.0
	for i := 0; i < exp; i++ {
		r *= base
	}
	return r
}

// diffWeather returns the zones whose weather changed from prev to next, sorted
// by zone for deterministic output.
func diffWeather(prev, next map[ZoneId]WeatherType) StateDiff {
	zones := make([]ZoneId, 0, len(next))
	for z := range next {
		zones = append(zones, z)
	}
	sort.Strings(zones)

	var changes []ZoneChange
	for _, z := range zones {
		from := prev[z] // "" if previously unknown
		if next[z] != from {
			changes = append(changes, ZoneChange{Zone: z, From: from, To: next[z]})
		}
	}
	return StateDiff{Changes: changes}
}

// moveFronts may advance each front to an adjacent zone. The chance to move is
// (1 - currentResistance); the destination is chosen weighted by edge weight,
// excluding the most recent zone in History (no immediate backtrack) when an
// alternative exists. On a move, the departed zone is appended to History
// (bounded by cfg.HistoryLen).
func moveFronts(fronts []Front, g *Graph, climate Climate, cfg Config, rng *RNG) []Front {
	for i := range fronts {
		f := &fronts[i]
		resistance := climate.For(g.Nodes[f.Zone].Biome).Influence.MovementResistance
		if rng.Float64() < resistance {
			// Record the linger so evolveTypes can tell this front did not move
			// this tick: its current zone becomes the newest History entry.
			f.History = appendBounded(f.History, f.Zone, cfg.HistoryLen)
			continue
		}
		neighbors := g.Neighbors(f.Zone)
		if len(neighbors) == 0 {
			continue
		}
		dest := pickNeighbor(neighbors, lastZone(f.History), rng)
		if dest == "" || dest == f.Zone {
			continue
		}
		f.History = appendBounded(f.History, f.Zone, cfg.HistoryLen)
		f.Zone = dest
	}
	return fronts
}

// pickNeighbor selects a destination weighted by edge weight, excluding `avoid`
// (the last zone) unless it is the only option.
func pickNeighbor(neighbors []Edge, avoid ZoneId, rng *RNG) ZoneId {
	candidates := neighbors
	if avoid != "" {
		filtered := make([]Edge, 0, len(neighbors))
		for _, e := range neighbors {
			if e.B != avoid {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) > 0 {
			candidates = filtered
		}
	}
	total := 0
	for _, e := range candidates {
		w := e.Weight
		if w < 1 {
			w = 1
		}
		total += w
	}
	if total <= 0 {
		return ""
	}
	r := rng.Intn(total)
	for _, e := range candidates {
		w := e.Weight
		if w < 1 {
			w = 1
		}
		if r < w {
			return e.B
		}
		r -= w
	}
	return candidates[len(candidates)-1].B
}

func lastZone(history []ZoneId) ZoneId {
	if len(history) == 0 {
		return ""
	}
	return history[len(history)-1]
}

func appendBounded(history []ZoneId, z ZoneId, max int) []ZoneId {
	history = append(history, z)
	if max > 0 && len(history) > max {
		history = history[len(history)-max:]
	}
	return history
}

// evolveTypes re-rolls the weather type of fronts that moved this tick (current
// zone != last History entry), drawing from the new zone's climate weights. The
// current type gets a bias bonus so changes are gradual, not jarring.
func evolveTypes(fronts []Front, g *Graph, climate Climate, rng *RNG) []Front {
	const keepBias = 3.0 // extra weight on the front's current type if valid here
	for i := range fronts {
		f := &fronts[i]
		if f.Zone == lastZone(f.History) {
			continue // did not move this tick
		}
		profile := climate.For(g.Nodes[f.Zone].Biome)
		if len(profile.Weather) == 0 {
			continue
		}
		f.Type = pickWeatherType(profile, f.Type, keepBias, rng)
	}
	return fronts
}

// pickWeatherType chooses a weather type from a profile's weights, adding
// keepBias to `current` if it appears in the profile. Iteration is over a sorted
// key list so selection is deterministic for a given RNG sequence.
func pickWeatherType(profile ClimateProfile, current WeatherType, keepBias float64, rng *RNG) WeatherType {
	keys := make([]WeatherType, 0, len(profile.Weather))
	for k := range profile.Weather {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool { return keys[a] < keys[b] })

	total := 0.0
	for _, k := range keys {
		w := profile.Weather[k]
		if k == current {
			w += keepBias
		}
		total += w
	}
	if total <= 0 {
		return current
	}
	r := rng.Float64() * total
	for _, k := range keys {
		w := profile.Weather[k]
		if k == current {
			w += keepBias
		}
		if r < w {
			return k
		}
		r -= w
	}
	return keys[len(keys)-1]
}

// spawnFronts may add one new front when under budget. It draws once on
// SpawnChance, then picks an origin zone weighted by climate SpawnWeight and a
// type from that zone's climate. New fronts start at moderate intensity/moisture.
func spawnFronts(fronts []Front, g *Graph, climate Climate, cfg Config, rng *RNG, nextID FrontId) ([]Front, FrontId) {
	if len(fronts) >= cfg.MaxActiveFronts {
		return fronts, nextID
	}
	if rng.Float64() >= cfg.SpawnChance {
		return fronts, nextID
	}
	origin := pickSpawnZone(g, climate, rng)
	if origin == "" {
		return fronts, nextID
	}
	profile := climate.For(g.Nodes[origin].Biome)
	wtype := pickWeatherType(profile, "", 0, rng)
	if wtype == "" {
		return fronts, nextID
	}
	f := Front{
		Id:        nextID,
		Type:      wtype,
		Zone:      origin,
		Intensity: 0.4 + 0.3*rng.Float64(), // 0.4..0.7
		Moisture:  0.5,
		Age:       0,
		MaxAge:    12 + rng.Intn(24), // 12..35 ticks
		History:   nil,
	}
	return append(fronts, f), nextID + 1
}

// pickSpawnZone selects an origin zone weighted by climate SpawnWeight (zones
// iterated in sorted order for determinism).
func pickSpawnZone(g *Graph, climate Climate, rng *RNG) ZoneId {
	zones := g.Zones()
	total := 0.0
	for _, z := range zones {
		w := climate.For(g.Nodes[z].Biome).SpawnWeight
		if w > 0 {
			total += w
		}
	}
	if total <= 0 {
		return ""
	}
	r := rng.Float64() * total
	for _, z := range zones {
		w := climate.For(g.Nodes[z].Biome).SpawnWeight
		if w <= 0 {
			continue
		}
		if r < w {
			return z
		}
		r -= w
	}
	return zones[len(zones)-1]
}
