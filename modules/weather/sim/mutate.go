package sim

// ForceSpawn injects a front at zone (admin command / exported API), bypassing
// the budget and spawn chance but flowing through the same resolve+diff path as
// Step so the engine applies the result uniformly. intensity <= 0 defaults to
// 0.6. Returns ok=false (state unchanged) for an unknown zone. Pure: the input
// state is not mutated, and no RNG is consumed (admin actions must not perturb
// the deterministic trace).
func ForceSpawn(prev State, g *Graph, cfg Config, wtype WeatherType, zone ZoneId, intensity float64, now Clock) (State, StateDiff, bool) {
	if _, ok := g.Nodes[zone]; !ok {
		return prev, StateDiff{}, false
	}
	if intensity <= 0 {
		intensity = 0.6
	}
	next := prev
	next.Round = now.Round
	next.Fronts = cloneFronts(prev.Fronts)
	next.Fronts = append(next.Fronts, Front{
		Id:        prev.NextID,
		Type:      wtype,
		Zone:      zone,
		Intensity: clamp01(intensity),
		Moisture:  0.5,
		MaxAge:    24,
	})
	next.NextID = prev.NextID + 1
	next.Weather = resolveWeather(g, next.Fronts, cfg)
	return next, diffWeather(prev.Weather, next.Weather), true
}

// ClearZones removes fronts and re-resolves weather. With no zones it removes
// every front. With zones, any front whose coverage projection reaches one of
// them is removed — so the named zone actually clears rather than staying
// covered by a neighboring front. Pure; no RNG consumed.
// Kept fronts share their History backing arrays with prev.Fronts; callers that will later mutate Front fields must pass the result through cloneFronts (Step does).
func ClearZones(prev State, g *Graph, cfg Config, zones []ZoneId, now Clock) (State, StateDiff) {
	next := prev
	next.Round = now.Round

	if len(zones) == 0 {
		next.Fronts = nil
	} else {
		drop := map[FrontId]bool{}
		for _, z := range zones {
			for _, c := range Covering(g, prev.Fronts, cfg, z) {
				drop[c.Front.Id] = true
			}
		}
		keep := make([]Front, 0, len(prev.Fronts))
		for _, f := range prev.Fronts {
			if !drop[f.Id] {
				keep = append(keep, f)
			}
		}
		next.Fronts = keep
	}

	next.Weather = resolveWeather(g, next.Fronts, cfg)
	return next, diffWeather(prev.Weather, next.Weather)
}
