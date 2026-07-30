package mapper

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Cross-zone routing.
//
// GetPath is per-zone by construction: it resolves its mapper via
// mapperZoneCache[zone] keyed off the START room, so it can never route to a
// room in another zone. That left every cross-zone quest marker inert — no
// destination dot (the room is not in the current zone's snapshot) and no
// next-step arrow. Six quests were affected, worst being the New Plymouth
// questline, whose guidance went dark exactly when a player was hunting for
// the right district.
//
// The fix is deliberately NOT a global room-level pathfinder. Zones are joined
// by ordinary exits, so a coarse zone-level graph is enough: work out which
// neighbouring zone to head for, then reuse the existing in-zone GetPath to
// walk to that border room. The player gets a correct arrow the whole way, one
// zone at a time, and the hot path stays on the cached per-zone mapper.
//
// SEE ALSO — modules/weather/crawler/build.go (buildEdges) builds a SECOND
// zone-adjacency graph by the same crawl, for weather-front movement. The two
// are not interchangeable and this is not an oversight:
//
//   - Its sim.Edge is {A, B, Weight}: undirected, canonicalised (a <= b), and
//     it discards the border room, exit direction and destination room — the
//     only fields a next-hop needs. It answers "do these zones touch", not
//     "which way do I walk".
//   - internal/ never imports modules/ anywhere in this codebase, and
//     cross-zone routing has to live here because it feeds NextStep -> the
//     GMCP quest payload.
//   - sim.Graph is deliberately "pure data, carries no engine types", built
//     through weather's own WorldReader seam and cached to versioned JSON.
//
// The genuinely duplicated part is the crawl itself (walk rooms, inspect
// exits, detect zone changes). If that is ever unified, this is the file that
// should consume the shared crawl. Note one behavioural difference: weather
// can exclude secret exits (Options.IncludeSecretExits); this graph counts
// them, which matches GetPath and the in-zone mapper (neither filters secret
// exits — the mapper only renders them with SecretSymbol). Changing that here
// alone would make cross-zone routing disagree with in-zone routing.

// zoneLink is one ordinary exit that leaves a zone.
//
// exitRoom is in the FROM zone and destRoom is in toZone — crossing means
// taking exitName out of exitRoom.
type zoneLink struct {
	toZone   string
	exitRoom int
	exitName string
	destRoom int
}

var (
	zoneGraphMu    sync.RWMutex
	zoneGraph      map[string][]zoneLink
	zoneGraphBuilt bool
)

// InvalidateZoneGraph drops the cached graph so the next cross-zone query
// rebuilds it. Call after anything that moves rooms between zones (zone
// rename/delete, room re-zone).
func InvalidateZoneGraph() {
	zoneGraphMu.Lock()
	zoneGraph, zoneGraphBuilt = nil, false
	zoneGraphMu.Unlock()
}

// buildZoneGraph walks every room once and records each exit whose destination
// lies in a different zone. Built lazily on first use and cached; the world is
// ~1400 rooms, so this is a one-off scan, not a per-move cost.
func buildZoneGraph() {
	g := map[string][]zoneLink{}
	seen := map[string]map[string]bool{} // fromZone -> toZone -> already linked

	for _, roomId := range rooms.GetAllRoomIds() {
		room := rooms.LoadRoom(roomId)
		if room == nil || room.Zone == "" {
			continue
		}
		for exitName, ex := range room.Exits {
			dest := rooms.LoadRoom(ex.RoomId)
			if dest == nil || dest.Zone == "" || dest.Zone == room.Zone {
				continue
			}
			// One link per zone pair is enough to route; the in-zone leg to
			// that border room is computed by GetPath.
			if seen[room.Zone] == nil {
				seen[room.Zone] = map[string]bool{}
			}
			if seen[room.Zone][dest.Zone] {
				continue
			}
			seen[room.Zone][dest.Zone] = true
			g[room.Zone] = append(g[room.Zone], zoneLink{
				toZone:   dest.Zone,
				exitRoom: roomId,
				exitName: exitName,
				destRoom: ex.RoomId,
			})
		}
	}

	links := 0
	for _, v := range g {
		links += len(v)
	}
	mudlog.Info("mapper.buildZoneGraph", "zones", len(g), "links", links)

	zoneGraph, zoneGraphBuilt = g, true
}

func ensureZoneGraph() {
	zoneGraphMu.RLock()
	built := zoneGraphBuilt
	zoneGraphMu.RUnlock()
	if built {
		return
	}
	zoneGraphMu.Lock()
	defer zoneGraphMu.Unlock()
	if zoneGraphBuilt { // another goroutine won the race
		return
	}
	buildZoneGraph()
}

// nextZoneHop returns the link to take FIRST when travelling from fromZone to
// toZone — i.e. the border out of the caller's current zone, even when the
// destination is several zones away. Breadth-first, so it is the fewest-zones
// route.
func nextZoneHop(fromZone, toZone string) (zoneLink, bool) {
	if fromZone == "" || toZone == "" || fromZone == toZone {
		return zoneLink{}, false
	}
	ensureZoneGraph()

	zoneGraphMu.RLock()
	defer zoneGraphMu.RUnlock()

	// BFS over zones, remembering which first-leg link each frontier came from.
	type node struct {
		zone  string
		first zoneLink
	}
	queue := make([]node, 0, len(zoneGraph[fromZone]))
	seen := map[string]bool{fromZone: true}
	for _, l := range zoneGraph[fromZone] {
		if l.toZone == toZone {
			return l, true // direct neighbour
		}
		if !seen[l.toZone] {
			seen[l.toZone] = true
			queue = append(queue, node{zone: l.toZone, first: l})
		}
	}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, l := range zoneGraph[n.zone] {
			if l.toZone == toZone {
				return n.first, true // keep the FIRST leg, not this one
			}
			if !seen[l.toZone] {
				seen[l.toZone] = true
				queue = append(queue, node{zone: l.toZone, first: n.first})
			}
		}
	}
	return zoneLink{}, false
}

// crossZoneHop returns the next room + exit for a journey that leaves the
// caller's zone. Standing on the border room, that is the crossing exit itself;
// otherwise it is the first in-zone step toward the border room.
//
// Known limitation: buildZoneGraph keeps ONE border per zone pair. If that
// particular border happens to be unreachable from where the player stands
// (behind a locked door, say) this returns false and the player gets no arrow,
// even though a second crossing might have worked. That degrades to today's
// behaviour rather than misleading anyone, so it is left as-is; storing every
// border and trying each is the fix if it ever bites.
func crossZoneHop(fromRoomId int, fromZone, toZone string) (nextRoomId int, exitName string, found bool) {
	link, ok := nextZoneHop(fromZone, toZone)
	if !ok {
		return 0, "", false
	}
	if fromRoomId == link.exitRoom {
		return link.destRoom, link.exitName, true
	}
	path, err := GetPath(fromRoomId, link.exitRoom) // same zone — GetPath can do this
	if err != nil {
		return 0, "", false
	}
	return firstHop(path)
}
