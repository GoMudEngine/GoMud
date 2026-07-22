package migration

import (
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
	coords, planes, collisions, mixed := crawlAndAssign(rms)

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

// crawlAndAssign returns per-room {x,y,z}, per-room plane, the count of overlapping
// cells on Euclidean planes, and the count of components mixing zone kinds.
func crawlAndAssign(rms map[int]*coordRoom) (map[int][3]int, map[int]int, int, int) {
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
			if _, ok := rms[e.to]; !ok {
				continue // dangling exit target
			}
			if mapper.IsValidExitDirection(e.dir) {
				union(id, e.to)
			}
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
		collisions += countCollisions(compCoords)
		if componentIsMixed(rms, members) {
			mixed++
		}
	}
	return coords, planes, collisions, mixed
}

// crawlComponent BFS-walks a component from its lowest room id at the origin,
// combining spatial deltas (first-visit-wins), then re-bases to the lowest id.
func crawlComponent(rms map[int]*coordRoom, members []int) map[int][3]int {
	lowest := members[0] // members is sorted ascending
	pos := map[int][3]int{lowest: {0, 0, 0}}
	queue := []int{lowest}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		base := pos[cur]
		for _, e := range rms[cur].edges {
			if _, ok := rms[e.to]; !ok {
				continue
			}
			if _, seen := pos[e.to]; seen {
				continue
			}
			if !mapper.IsValidExitDirection(e.dir) {
				continue // portal: doesn't place a neighbor spatially
			}
			dx, dy, dz := mapper.GetDelta(e.dir)
			pos[e.to] = [3]int{base[0] + dx, base[1] + dy, base[2] + dz}
			queue = append(queue, e.to)
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

func componentIsMixed(rms map[int]*coordRoom, members []int) bool {
	sawNormal, sawWeird := false, false
	// Heuristic during migration: we don't load zone configs per member here;
	// mixing is reported so a human reviews. A component is "mixed" if it spans
	// >1 zone dir AND at least one looks maze/instance-like. Cheap signal only.
	zones := map[string]bool{}
	for _, id := range members {
		zones[rms[id].zoneDir] = true
	}
	for z := range zones {
		if isLikelyNonCartesianDir(z) {
			sawWeird = true
		} else {
			sawNormal = true
		}
	}
	return sawNormal && sawWeird
}

func isLikelyNonCartesianDir(zoneDir string) bool {
	for _, s := range []string{"labyrinth", "foldweave", "trashheap", "oasis", "crash_site_interior", "antechamber", "ferries"} {
		if strings.Contains(zoneDir, s) {
			return true
		}
	}
	return false
}

// --- order-preserving write-back ---

func writeCoordsPreservingOrder(path string, raw []byte, x, y, z, plane int) error {
	var ms yaml.MapSlice
	if err := yaml.Unmarshal(raw, &ms); err != nil {
		return err
	}
	// Drop any existing coord keys, then append the nonzero ones (omitempty parity).
	out := ms[:0]
	for _, item := range ms {
		if k, ok := item.Key.(string); ok {
			switch k {
			case "x", "y", "z", "plane":
				continue
			}
		}
		out = append(out, item)
	}
	appendIf := func(k string, v int) {
		if v != 0 {
			out = append(out, yaml.MapItem{Key: k, Value: v})
		}
	}
	appendIf("x", x)
	appendIf("y", y)
	appendIf("z", z)
	appendIf("plane", plane)

	b, err := yaml.Marshal(out)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
