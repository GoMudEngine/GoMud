package weather

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/modules/weather/engine"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

const adminUsage = "Weather admin subcommands: zones | fronts | spawn <type> <zone> [intensity 0..1] | clear [zone] | graph [zone] | rebuild | status"

// cmdWeather is the weather command. Bare `weather` shows local conditions to
// any player; everything else is admin/mod gated (HasRolePermission: admins
// always pass, mods need the granted "weather" permission key).
func (m *weatherModule) cmdWeather(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if !m.cfg.Enabled {
		m.printLocalWeather(user, room) // module disabled: handler exists but stays inert
		return true, nil
	}

	args := strings.Fields(rest)
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}

	if sub == "" || !user.HasRolePermission(`weather`, true) {
		m.printLocalWeather(user, room)
		return true, nil
	}

	switch sub {
	case "zones":
		m.printZones(user)
	case "fronts":
		m.printFronts(user)
	case "spawn":
		m.cmdSpawn(user, args[1:])
	case "clear":
		m.cmdClear(user, args[1:])
	case "graph":
		zone := strings.Join(args[1:], " ")
		if zone == "" && room != nil {
			zone = room.Zone
		}
		m.printGraphForZone(user, zone)
	case "rebuild":
		m.rebuildGraph()
		if m.graph == nil {
			sendLine(user, "Weather: graph rebuild failed (see server log).")
			return true, nil
		}
		sendLine(user, fmt.Sprintf("Weather: rebuilt graph — %d zones, %d edges, %d components.",
			len(m.graph.Nodes), len(m.graph.Edges), m.graph.Components))
	case "status":
		m.printStatus(user)
	default:
		sendLine(user, adminUsage)
	}
	return true, nil
}

// printLocalWeather shows the weather where the user stands (the player view).
func (m *weatherModule) printLocalWeather(user *users.UserRecord, room *rooms.Room) {
	if !m.simReady || room == nil {
		sendLine(user, "The weather seems entirely unremarkable.")
		return
	}
	w := m.state.Weather[room.Zone]
	if w == "" {
		w = sim.Clear
	}
	sendLine(user, fmt.Sprintf("The weather in %s is %s.", room.Zone, w))
	if covers := sim.Covering(m.graph, m.state.Fronts, m.simCfg, room.Zone); len(covers) > 0 {
		c := covers[0]
		where := "centered here"
		if c.Hops > 0 {
			where = fmt.Sprintf("%d zone(s) away", c.Hops)
		}
		sendLine(user, fmt.Sprintf("  A %s system (front #%d) is %s — intensity %.2f, felt here at %.2f.",
			c.Front.Type, c.Front.Id, where, c.Front.Intensity, c.Effective))
	}
}

func (m *weatherModule) printZones(user *users.UserRecord) {
	if !m.simReady {
		sendLine(user, "Weather: simulation not running.")
		return
	}
	zones := m.graph.Zones()
	sendLine(user, fmt.Sprintf("Current weather (%d zones):", len(zones)))
	for _, z := range zones {
		sendLine(user, fmt.Sprintf("  %-30s %s", z, m.state.Weather[z]))
	}
}

func (m *weatherModule) printFronts(user *users.UserRecord) {
	if !m.simReady {
		sendLine(user, "Weather: simulation not running.")
		return
	}
	if len(m.state.Fronts) == 0 {
		sendLine(user, "No active weather fronts.")
		return
	}
	sendLine(user, fmt.Sprintf("Active fronts (%d):", len(m.state.Fronts)))
	for _, f := range m.state.Fronts {
		sendLine(user, fmt.Sprintf("  #%-3d %-10s @ %-25s intensity %.2f moisture %.2f age %d/%d",
			f.Id, f.Type, f.Zone, f.Intensity, f.Moisture, f.Age, f.MaxAge))
	}
}

// cmdSpawn: weather spawn <type> <zone words...> [intensity]. Zone names may
// contain spaces; a trailing float is taken as intensity only when at least
// one zone word remains.
func (m *weatherModule) cmdSpawn(user *users.UserRecord, parts []string) {
	if !m.simReady {
		sendLine(user, "Weather: simulation not running.")
		return
	}
	if len(parts) < 2 {
		sendLine(user, "Usage: weather spawn <type> <zone> [intensity 0..1]")
		return
	}
	wtype := strings.ToLower(parts[0])
	rest := parts[1:]
	intensity := 0.0
	if len(rest) > 1 {
		if f, err := strconv.ParseFloat(rest[len(rest)-1], 64); err == nil {
			intensity = f
			rest = rest[:len(rest)-1]
		}
	}
	zone, ok := m.graph.FindZone(strings.Join(rest, " "))
	if !ok {
		sendLine(user, fmt.Sprintf("Weather: zone %q is not in the graph.", strings.Join(rest, " ")))
		return
	}
	next, _, ok := sim.ForceSpawn(m.state, m.graph, m.simCfg, sim.WeatherType(wtype), zone, intensity, sim.Clock{Round: engine.CurrentRound()})
	if !ok {
		sendLine(user, "Weather: spawn failed.")
		return
	}
	m.state = next
	engine.Reconcile(m.state.Weather)
	m.persistState()
	f := m.state.Fronts[len(m.state.Fronts)-1]
	sendLine(user, fmt.Sprintf("Spawned front #%d: %s @ %s, intensity %.2f.", f.Id, f.Type, f.Zone, f.Intensity))
}

// cmdClear: weather clear [zone words...]. No zone = clear everything.
func (m *weatherModule) cmdClear(user *users.UserRecord, parts []string) {
	if !m.simReady {
		sendLine(user, "Weather: simulation not running.")
		return
	}
	var zones []sim.ZoneId
	if len(parts) > 0 {
		zone, ok := m.graph.FindZone(strings.Join(parts, " "))
		if !ok {
			sendLine(user, fmt.Sprintf("Weather: zone %q is not in the graph.", strings.Join(parts, " ")))
			return
		}
		zones = []sim.ZoneId{zone}
	}
	before := len(m.state.Fronts)
	next, diff := sim.ClearZones(m.state, m.graph, m.simCfg, zones, sim.Clock{Round: engine.CurrentRound()})
	m.state = next
	engine.Reconcile(m.state.Weather)
	m.persistState()
	sendLine(user, fmt.Sprintf("Cleared %d front(s); %d zone change(s).", before-len(m.state.Fronts), len(diff.Changes)))
}

func (m *weatherModule) printStatus(user *users.UserRecord) {
	if m.graph == nil {
		sendLine(user, "Weather: no geography graph yet. Try 'weather rebuild'.")
		return
	}
	g := m.graph
	sendLine(user, fmt.Sprintf("Geography: %d zones, %d edges, %d components (built round %d).",
		len(g.Nodes), len(g.Edges), g.Components, g.BuiltAtRound))
	if !m.simReady {
		sendLine(user, "Simulation: NOT running.")
		return
	}
	sendLine(user, fmt.Sprintf("Simulation: %d active front(s); state round %d; next tick at round %d (every %d game hour(s)).",
		len(m.state.Fronts), m.state.Round, m.nextTick, m.cfg.TickEveryGameHours))
	sendLine(user, fmt.Sprintf("Emotes: mode=%s every ~%d rounds; buffs=%v; persist=%v.",
		m.cfg.EmoteMode, m.cfg.EmoteEveryRounds, m.cfg.BuffsEnabled, m.cfg.Persist))
}

// printGraphForZone prints a zone's neighbors (crawler spot-check). NOTE: the
// Neighbors result is a shared index slice — copy before sorting.
func (m *weatherModule) printGraphForZone(user *users.UserRecord, zone string) {
	if m.graph == nil {
		sendLine(user, "Weather: no geography graph yet (built on the first round). Try 'weather rebuild'.")
		return
	}
	canonical, ok := m.graph.FindZone(zone)
	if !ok {
		sendLine(user, fmt.Sprintf("Weather: zone %q is not in the graph.", zone))
		return
	}
	node := m.graph.Nodes[canonical]
	sendLine(user, fmt.Sprintf(
		"Zone %s [biome=%s rooms=%d outdoor=%v]:", node.Zone, node.Biome, node.Rooms, node.HasOutdoor))

	neighbors := append([]sim.Edge(nil), m.graph.Neighbors(canonical)...)
	if len(neighbors) == 0 {
		sendLine(user, "  (no adjacent zones)")
		return
	}
	sort.Slice(neighbors, func(i, j int) bool { return neighbors[i].B < neighbors[j].B })
	for _, e := range neighbors {
		sendLine(user, fmt.Sprintf("  -> %s (weight %d)", e.B, e.Weight))
	}
}
