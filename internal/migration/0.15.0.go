package migration

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// Description:
// Backfill authored x/y/z/plane onto every non-instance room. Geometry moves
// from crawl-derived (an emergent property of the exit graph) to authored (the
// room owns its coordinate). We crawl SPATIAL exit deltas per connected
// component — portal/named exits do NOT merge components, so interiors and
// instanced areas fall into their own component and get their own plane. The
// largest component is plane 0 (the overworld). Within a component the frame is
// re-based so the lowest room id sits at the origin (matching the mapper). The
// rendered map is unchanged because within-zone relative layout is offset-
// invariant (verified by the map-parity spot-check at apply time); pre-existing
// overlaps on a Euclidean plane are reported, not silently accepted.
//
// Idempotent: a room already carrying any nonzero coord/plane is left untouched.
// Spec: docs/superpowers/specs/2026-07-22-authored-coordinate-model-design.md.

type coordEdge struct {
	to  int
	dir string // resolved direction name (mapdirection override, else exit key)
}

type coordRoom struct {
	id       int
	path     string
	zoneDir  string
	edges    []coordEdge
	rawBytes []byte
	migrated bool
}

func migrate_BackfillCoords(dryRun bool) error {
	c := configs.GetConfig()
	return backfillCoordsInDir(filepath.Join(string(c.FilePaths.DataFiles), "rooms"), dryRun)
}

func backfillCoordsInDir(roomsDir string, dryRun bool) error {
	mode := "APPLY"
	if dryRun {
		mode = "DRY-RUN"
	}
	rms, err := loadCoordRooms(roomsDir)
	if err != nil {
		return err
	}
	nonCart := loadNonCartesianZones(roomsDir)
	coords, planes, collisions, mixed := crawlAndAssign(rms, nonCart)

	planeSet := map[int]bool{}
	for _, p := range planes {
		planeSet[p] = true
	}
	mudlog.Info("Migration 0.15.0 coords", "mode", mode, "rooms", len(rms),
		"planes", len(planeSet), "collisions", collisions, "mixedComponents", mixed)
	if collisions > 0 {
		mudlog.Warn("Migration 0.15.0 coords",
			"message", "pre-existing overlaps on a Euclidean plane — mark the zone non_cartesian or fix geometry",
			"count", collisions)
	}
	if mixed > 0 {
		mudlog.Warn("Migration 0.15.0 coords",
			"message", "component mixes non_cartesian and normal zones — its whole plane will be non-Euclidean",
			"count", mixed)
	}

	for id, r := range rms {
		if r.migrated {
			continue
		}
		co := coords[id]
		if dryRun {
			continue
		}
		if err := writeCoordsPreservingOrder(r.path, r.rawBytes, co[0], co[1], co[2], planes[id]); err != nil {
			return fmt.Errorf("room %d (%s): %w", id, r.path, err)
		}
	}
	return nil
}

// --- loading ---

type parsedRoom struct {
	RoomId int `yaml:"roomid"`
	X      int `yaml:"x"`
	Y      int `yaml:"y"`
	Z      int `yaml:"z"`
	Plane  int `yaml:"plane"`
	Exits  map[string]struct {
		RoomId       int    `yaml:"roomid"`
		MapDirection string `yaml:"mapdirection"`
	} `yaml:"exits"`
}

func loadCoordRooms(roomsDir string) (map[int]*coordRoom, error) {
	out := map[int]*coordRoom{}
	zoneDirs, err := os.ReadDir(roomsDir)
	if err != nil {
		return nil, err
	}
	for _, zd := range zoneDirs {
		if !zd.IsDir() {
			continue
		}
		zoneDir := zd.Name()
		// Instances are runtime clones with their own planes; never author them.
		if strings.HasPrefix(zoneDir, "instance_") || strings.HasPrefix(strings.ToLower(zoneDir), "instance ") {
			continue
		}
		files, _ := filepath.Glob(filepath.Join(roomsDir, zoneDir, "*.yaml"))
		for _, path := range files {
			if filepath.Base(path) == "zone-config.yaml" {
				continue
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			var pr parsedRoom
			if err := yaml.Unmarshal(raw, &pr); err != nil {
				mudlog.Warn("Migration 0.15.0 coords", "file", path, "error", err)
				continue
			}
			if pr.RoomId == 0 {
				continue
			}
			r := &coordRoom{
				id:       pr.RoomId,
				path:     path,
				zoneDir:  zoneDir,
				rawBytes: raw,
				migrated: pr.X != 0 || pr.Y != 0 || pr.Z != 0 || pr.Plane != 0,
			}
			for name, e := range pr.Exits {
				dir := name
				if e.MapDirection != "" {
					dir = e.MapDirection
				}
				r.edges = append(r.edges, coordEdge{to: e.RoomId, dir: dir})
			}
			// stable edge order (map iteration is random) so crawls are deterministic
			sort.Slice(r.edges, func(i, j int) bool { return r.edges[i].to < r.edges[j].to })
			out[pr.RoomId] = r
		}
	}
	return out, nil
}

// --- crawl + plane assignment ---

// loadNonCartesianZones returns the set of zone folder names flagged
// non_cartesian in their zone-config.yaml.
func loadNonCartesianZones(roomsDir string) map[string]bool {
	out := map[string]bool{}
	zoneDirs, err := os.ReadDir(roomsDir)
	if err != nil {
		return out
	}
	for _, zd := range zoneDirs {
		if !zd.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(roomsDir, zd.Name(), "zone-config.yaml"))
		if err != nil {
			continue
		}
		var zc struct {
			NonCartesian bool `yaml:"non_cartesian"`
		}
		if yaml.Unmarshal(raw, &zc) == nil && zc.NonCartesian {
			out[zd.Name()] = true
		}
	}
	return out
}

// crawlAndAssign returns per-room {x,y,z}, per-room plane, the count of overlapping
// cells on EUCLIDEAN planes only, and the count of components mixing zone kinds.
func crawlAndAssign(rms map[int]*coordRoom, nonCart map[string]bool) (map[int][3]int, map[int]int, int, int) {
	// Union-find over SPATIAL edges only (portals don't merge components).
	parent := map[int]int{}
	var find func(int) int
	find = func(a int) int {
		if parent[a] == 0 {
			parent[a] = a
		}
		if parent[a] != a {
			parent[a] = find(parent[a])
		}
		return parent[a]
	}
	union := func(a, b int) { parent[find(a)] = find(b) }
	for id, r := range rms {
		find(id)
		for _, e := range r.edges {
			dst, ok := rms[e.to]
			if !ok {
				continue // dangling exit target
			}
			if !mapper.IsValidExitDirection(e.dir) {
				continue // portal: never merges components
			}
			// A spatial exit across a normal<->non_cartesian boundary does NOT
			// merge planes: the non-Euclidean area (maze/warped space) keeps its
			// own plane so it can't drag a normal plane non-Euclidean. You still
			// walk through the boundary at runtime (it's a cross-plane exit).
			if nonCart[r.zoneDir] != nonCart[dst.zoneDir] {
				continue
			}
			union(id, e.to)
		}
	}

	comps := map[int][]int{}
	for id := range rms {
		root := find(id)
		comps[root] = append(comps[root], id)
	}

	// Order components: largest first, then lowest member id — deterministic.
	roots := make([]int, 0, len(comps))
	for root := range comps {
		roots = append(roots, root)
		sort.Ints(comps[root])
	}
	sort.Slice(roots, func(i, j int) bool {
		if len(comps[roots[i]]) != len(comps[roots[j]]) {
			return len(comps[roots[i]]) > len(comps[roots[j]])
		}
		return comps[roots[i]][0] < comps[roots[j]][0]
	})

	coords := map[int][3]int{}
	planes := map[int]int{}
	collisions, mixed := 0, 0
	for pi, root := range roots {
		members := comps[root]
		plane := pi // largest => 0
		compCoords := crawlComponent(rms, members)
		for id, c := range compCoords {
			coords[id] = c
			planes[id] = plane
		}
		nc, normal := false, false
		for _, id := range members {
			if nonCart[rms[id].zoneDir] {
				nc = true
			} else {
				normal = true
			}
		}
		if !nc {
			collisions += countCollisions(compCoords) // overlaps only matter on Euclidean planes
		}
		if nc && normal {
			mixed++ // a non_cartesian zone spatially bleeding into a normal plane
		}
	}
	return coords, planes, collisions, mixed
}

// crawlComponent BFS-walks a component from its lowest room id at the origin,
// combining spatial deltas (first-visit-wins), then re-bases to the lowest id.
// Traversal is UNDIRECTED over spatial edges (each A->B edge also yields B->A
// with the negated delta) so that every component member is positioned — a
// member reachable only via an incoming edge would otherwise default to the
// origin and produce a false collision. In a consistent Cartesian component a
// room's position is path-independent, so this matches the mapper's directed
// crawl for every Euclidean plane.
func crawlComponent(rms map[int]*coordRoom, members []int) map[int][3]int {
	memberSet := make(map[int]bool, len(members))
	for _, id := range members {
		memberSet[id] = true
	}
	type nbr struct{ to, dx, dy, dz int }
	adj := map[int][]nbr{}
	for _, id := range members {
		for _, e := range rms[id].edges {
			if !memberSet[e.to] || !mapper.IsValidExitDirection(e.dir) {
				continue
			}
			dx, dy, dz := mapper.GetDelta(e.dir)
			adj[id] = append(adj[id], nbr{e.to, dx, dy, dz})
			adj[e.to] = append(adj[e.to], nbr{id, -dx, -dy, -dz})
		}
	}

	lowest := members[0] // members is sorted ascending
	pos := map[int][3]int{lowest: {0, 0, 0}}
	queue := []int{lowest}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		base := pos[cur]
		for _, n := range adj[cur] {
			if _, seen := pos[n.to]; seen {
				continue
			}
			pos[n.to] = [3]int{base[0] + n.dx, base[1] + n.dy, base[2] + n.dz}
			queue = append(queue, n.to)
		}
	}
	// Any member still unpositioned (disconnected via spatial edges — e.g. joined
	// only by a portal that union-find shouldn't have merged) falls to the origin.
	for _, id := range members {
		if _, ok := pos[id]; !ok {
			pos[id] = [3]int{0, 0, 0}
		}
	}
	// Re-base to the lowest id (already at origin here, but keep explicit for parity).
	off := pos[lowest]
	for id, c := range pos {
		pos[id] = [3]int{c[0] - off[0], c[1] - off[1], c[2] - off[2]}
	}
	return pos
}

func countCollisions(coords map[int][3]int) int {
	byCell := map[[3]int]int{}
	for _, c := range coords {
		byCell[c]++
	}
	n := 0
	for _, cnt := range byCell {
		if cnt > 1 {
			n += cnt - 1
		}
	}
	return n
}

// --- order-preserving write-back ---

// writeCoordsPreservingOrder appends the nonzero coord keys as top-level YAML
// lines at the END of the file — a plain TEXT append that touches nothing above,
// so all existing prose wrapping, quoting, and line endings are preserved (a
// re-marshal would reflow every description/nouns block). Matches the file's
// existing EOL. Rooms are only processed when they carry no coords yet
// (idempotency guard upstream), so there are never existing coord keys to
// duplicate. An origin room (all-zero on plane 0) writes nothing.
func writeCoordsPreservingOrder(path string, raw []byte, x, y, z, plane int) error {
	nl := "\n"
	if bytes.Contains(raw, []byte("\r\n")) {
		nl = "\r\n"
	}
	var add strings.Builder
	appendIf := func(k string, v int) {
		if v != 0 {
			fmt.Fprintf(&add, "%s: %d%s", k, v, nl)
		}
	}
	appendIf("x", x)
	appendIf("y", y)
	appendIf("z", z)
	appendIf("plane", plane)
	if add.Len() == 0 {
		return nil
	}

	body := raw
	if len(body) > 0 && !bytes.HasSuffix(body, []byte("\n")) {
		body = append(body, []byte(nl)...)
	}
	body = append(body, []byte(add.String())...)
	return os.WriteFile(path, body, 0644)
}
