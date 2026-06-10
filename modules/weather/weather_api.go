package weather

import (
	"github.com/GoMudEngine/GoMud/modules/weather/engine"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// registerExports exposes the module API to other modules and JS scripts via
// plugin.ExportFunction (spec §9.7). All exports guard simReady so callers
// during boot (or in degraded mode) get empty-but-valid answers.
// Mutating exports rely on the engine invoking exported functions on the
// MainWorker goroutine (the same single-goroutine guarantee as commands and
// events).
func (m *weatherModule) registerExports() {
	m.plug.ExportFunction(`GetWeather`, m.exportGetWeather)
	m.plug.ExportFunction(`GetFronts`, m.exportGetFronts)
	m.plug.ExportFunction(`SpawnFront`, m.exportSpawnFront)
}

// exportGetWeather reports a zone's current weather: {"type": string,
// "intensity": float64}. type is "" when the sim isn't running or the zone is
// unknown; intensity is the strongest effective front projection (0 for calm).
func (m *weatherModule) exportGetWeather(zone string) map[string]any {
	out := map[string]any{"type": "", "intensity": 0.0}
	if !m.simReady {
		return out
	}
	canonical, ok := m.graph.FindZone(zone)
	if !ok {
		return out
	}
	w := m.state.Weather[canonical]
	if w == "" {
		w = sim.Clear
	}
	out["type"] = string(w)
	if covers := sim.Covering(m.graph, m.state.Fronts, m.simCfg, canonical); len(covers) > 0 {
		out["intensity"] = covers[0].Effective
	}
	return out
}

// exportGetFronts lists active fronts as plain maps (id, type, zone,
// intensity, moisture, age).
func (m *weatherModule) exportGetFronts() []map[string]any {
	if !m.simReady {
		return nil
	}
	out := make([]map[string]any, 0, len(m.state.Fronts))
	for _, f := range m.state.Fronts {
		out = append(out, map[string]any{
			"id": uint64(f.Id), "type": string(f.Type), "zone": f.Zone,
			"intensity": f.Intensity, "moisture": f.Moisture, "age": f.Age,
		})
	}
	return out
}

// exportSpawnFront programmatically spawns a front (e.g. a quest summoning a
// storm). Returns false when the sim isn't running or the zone is unknown.
func (m *weatherModule) exportSpawnFront(wtype string, zone string, intensity float64) bool {
	if !m.simReady {
		return false
	}
	canonical, ok := m.graph.FindZone(zone)
	if !ok {
		return false
	}
	next, _, ok := sim.ForceSpawn(m.state, m.graph, m.simCfg, sim.WeatherType(wtype), canonical, intensity, sim.Clock{Round: engine.CurrentRound()})
	if !ok {
		return false
	}
	m.state = next
	engine.Reconcile(m.state.Weather)
	m.persistState()
	return true
}
