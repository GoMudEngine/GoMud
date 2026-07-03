// Package ferry implements the scheduled ferry vessels: clock-derived
// vessel state, gangplank reconciliation, and agent-paid boarding.
// Stage 1 of docs/superpowers/specs/2026-07-03-ferry-system-design.md.
package ferry

import (
	"fmt"
	"os"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Port is one end of a route.
type Port struct {
	DockRoom int `yaml:"dock_room"`
}

// Route is one ferry line: a vessel (deck room) shuttling between two ports.
type Route struct {
	RouteId           string `yaml:"routeid"`
	Name              string `yaml:"name"` // display name, e.g. "the Lakewind Packet"
	DeckRoom          int    `yaml:"deck_room"`
	Ports             []Port `yaml:"ports"` // exactly two
	CrossingHours     int    `yaml:"crossing_hours"`
	LayoverHours      int    `yaml:"layover_hours"`
	PhaseOffsetRounds int    `yaml:"phase_offset_rounds"`
	Fare              int    `yaml:"fare"`
}

func (r Route) Id() string       { return r.RouteId }
func (r Route) Filepath() string { return r.RouteId + `.yaml` }

// Validate covers intrinsic checks; world checks (rooms exist) happen in
// LoadDataFiles once room data is loaded.
func (r Route) Validate() error {
	if r.RouteId == `` {
		return fmt.Errorf(`ferry route missing routeid`)
	}
	if r.Name == `` {
		return fmt.Errorf(`ferry route %s missing name`, r.RouteId)
	}
	if r.DeckRoom <= 0 {
		return fmt.Errorf(`ferry route %s missing deck_room`, r.RouteId)
	}
	if len(r.Ports) != 2 {
		return fmt.Errorf(`ferry route %s must have exactly 2 ports, has %d`, r.RouteId, len(r.Ports))
	}
	if r.Ports[0].DockRoom == r.Ports[1].DockRoom {
		return fmt.Errorf(`ferry route %s has the same dock_room at both ports`, r.RouteId)
	}
	for i, p := range r.Ports {
		if p.DockRoom <= 0 {
			return fmt.Errorf(`ferry route %s port %d missing dock_room`, r.RouteId, i)
		}
	}
	if r.CrossingHours <= 0 || r.LayoverHours <= 0 {
		return fmt.Errorf(`ferry route %s needs positive crossing_hours and layover_hours`, r.RouteId)
	}
	if r.LayoverHours >= 24 {
		return fmt.Errorf(`ferry route %s layover_hours must be < 24 (gangplank exit lifetime)`, r.RouteId)
	}
	if r.PhaseOffsetRounds < 0 {
		return fmt.Errorf(`ferry route %s has negative phase_offset_rounds`, r.RouteId)
	}
	if r.Fare < 0 {
		return fmt.Errorf(`ferry route %s has negative fare`, r.RouteId)
	}
	return nil
}

var routes = map[string]Route{}

// RouteFor returns a registered route by id.
func RouteFor(routeId string) (Route, bool) {
	r, ok := routes[routeId]
	return r, ok
}

// AllRoutes returns all registered routes.
func AllRoutes() []Route {
	out := make([]Route, 0, len(routes))
	for _, r := range routes {
		out = append(out, r)
	}
	return out
}

// LoadDataFiles loads + validates every route. Panics on world-integrity
// failures (missing rooms, shared deck rooms, hours that truncate to zero
// rounds at the live RoundsPerDay) — same startup rigor as the
// schedule/patrol validators. Must be called AFTER rooms.LoadDataFiles().
// If the ferries directory does not exist, logs and returns — ferry
// routes are optional content.
func LoadDataFiles() {
	start := time.Now()

	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/ferries`

	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		mudlog.Info(`ferry.LoadDataFiles()`, `loadedCount`, 0,
			`note`, `ferries directory does not exist — skipping`,
			`Time Taken`, time.Since(start))
		return
	}

	loaded, err := fileloader.LoadAllFlatFiles[string, Route](dataPath)
	if err != nil {
		panic(fmt.Sprintf(`ferry.LoadDataFiles: %v`, err))
	}

	rpd := int(configs.GetTimingConfig().RoundsPerDay)

	decksSeen := map[int]string{}
	for id, r := range loaded {
		if hoursToRounds(r.LayoverHours, rpd) == 0 || hoursToRounds(r.CrossingHours, rpd) == 0 {
			panic(fmt.Sprintf(`ferry route %s: crossing/layover hours truncate to 0 rounds at RoundsPerDay=%d`, id, rpd))
		}
		if rooms.LoadRoom(r.DeckRoom) == nil {
			panic(fmt.Sprintf(`ferry route %s: deck_room %d does not exist`, id, r.DeckRoom))
		}
		for i, p := range r.Ports {
			if rooms.LoadRoom(p.DockRoom) == nil {
				panic(fmt.Sprintf(`ferry route %s: port %d dock_room %d does not exist`, id, i, p.DockRoom))
			}
		}
		if other, dup := decksSeen[r.DeckRoom]; dup {
			panic(fmt.Sprintf(`ferry routes %s and %s share deck_room %d`, id, other, r.DeckRoom))
		}
		decksSeen[r.DeckRoom] = id
	}

	routes = loaded
	mudlog.Info(`ferry.LoadDataFiles()`, `loadedCount`, len(routes), `Time Taken`, time.Since(start))
}
