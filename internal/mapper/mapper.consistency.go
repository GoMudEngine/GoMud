package mapper

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// ExitKind classifies a spatial exit by comparing its nominal compass delta to
// the actual placed coordinate delta between the two crawled rooms.
type ExitKind string

const (
	ExitNormal   ExitKind = "normal"   // unit cardinal/diagonal, placed exactly as nominal
	ExitLong     ExitKind = "long"     // multi-cell (-x2/-x3), placed exactly as nominal
	ExitVertical ExitKind = "vertical" // up/down (dz != 0)
	ExitWrap     ExitKind = "wrap"     // actual delta != nominal (toroidal/torn edge)
)

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// samePos compares only x/y/z (NOT arrow — positionDelta.arrow differs by direction).
func samePos(a, b positionDelta) bool {
	return a.x == b.x && a.y == b.y && a.z == b.z
}

// classifyKind decides the render/consistency kind from nominal vs actual delta.
func classifyKind(nominal, actual positionDelta) ExitKind {
	if !samePos(nominal, actual) {
		return ExitWrap
	}
	if nominal.z != 0 {
		return ExitVertical
	}
	if absInt(nominal.x) > 1 || absInt(nominal.y) > 1 {
		return ExitLong
	}
	return ExitNormal
}

// findCollisions returns groups of >=2 roomIds that share the same (x,y,z).
func findCollisions(nodes map[int]*mapNode) [][]int {
	byCell := map[[3]int][]int{}
	for id, n := range nodes {
		key := [3]int{n.Pos.x, n.Pos.y, n.Pos.z}
		byCell[key] = append(byCell[key], id)
	}
	groups := [][]int{}
	for _, ids := range byCell {
		if len(ids) >= 2 {
			sort.Ints(ids)
			groups = append(groups, ids)
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i][0] < groups[j][0] })
	return groups
}

// Finding is one consistency problem.
type Finding struct {
	Severity string // "error" (collision/reciprocity/delta) | "warn" (long-crossing)
	Kind     string // "collision" | "noreciprocal" | "deltamismatch" | "longcrossing"
	Zone     string
	RoomId   int
	ExitName string
	Detail   string
}

func (f Finding) String() string {
	loc := fmt.Sprintf("zone=%s room=%d", f.Zone, f.RoomId)
	if f.ExitName != "" {
		loc += " exit=" + f.ExitName
	}
	return fmt.Sprintf("[%s] %s: %s (%s)", f.Severity, f.Kind, f.Detail, loc)
}

// hasReturnExit reports whether dst has any spatial exit leading back to srcId.
func hasReturnExit(dst *mapNode, srcId int) bool {
	for _, e := range dst.Exits {
		if e.RoomId == srcId {
			return true
		}
	}
	return false
}

// roomCrawlable skips ephemeral/instance rooms (they don't exist at boot;
// guards the live cartcheck command).
func roomCrawlable(roomId int) bool {
	return !rooms.IsEphemeralRoomId(roomId)
}

// CheckConsistency walks the crawled rooms of this mapper and returns findings.
// nonCartesian=true (zone flag) suppresses collision/reciprocity/deltamismatch
// (the zone is intentionally non-Euclidean); the long-crossing warning still runs.
func (r *mapper) CheckConsistency(zone string, nonCartesian bool) []Finding {
	findings := []Finding{}

	if !nonCartesian {
		for _, group := range findCollisions(r.crawledRooms) {
			findings = append(findings, Finding{
				Severity: "error", Kind: "collision", Zone: zone, RoomId: group[0],
				Detail: fmt.Sprintf("rooms %v occupy the same coordinate", group),
			})
		}
	}

	for srcId, src := range r.crawledRooms {
		if !roomCrawlable(srcId) {
			continue
		}
		for exitName, e := range src.Exits {
			dst, ok := r.crawledRooms[e.RoomId]
			if !ok {
				continue // cross-zone or uncrawled — not part of this coordinate space
			}
			actual := positionDelta{x: dst.Pos.x - src.Pos.x, y: dst.Pos.y - src.Pos.y, z: dst.Pos.z - src.Pos.z}

			if !nonCartesian {
				if !samePos(e.Direction, actual) {
					findings = append(findings, Finding{
						Severity: "error", Kind: "deltamismatch", Zone: zone, RoomId: srcId, ExitName: exitName,
						Detail: fmt.Sprintf("nominal delta (%d,%d,%d) != actual (%d,%d,%d) — wrap exit detected in a Cartesian zone; set non_cartesian: true on the zone or fix the geometry",
							e.Direction.x, e.Direction.y, e.Direction.z, actual.x, actual.y, actual.z),
					})
				}
				if !e.OneWay && !hasReturnExit(dst, srcId) {
					findings = append(findings, Finding{
						Severity: "error", Kind: "noreciprocal", Zone: zone, RoomId: srcId, ExitName: exitName,
						Detail: fmt.Sprintf("exit to room %d has no return exit (use oneway: true if intentional)", e.RoomId),
					})
				}
			}

			// Soft, always-on: long exit whose straight span crosses an occupied cell.
			if samePos(e.Direction, actual) && (absInt(actual.x) > 1 || absInt(actual.y) > 1) {
				if crossed := r.longSpanCrossesRoom(src.Pos, actual, srcId, e.RoomId); crossed != 0 {
					findings = append(findings, Finding{
						Severity: "warn", Kind: "longcrossing", Zone: zone, RoomId: srcId, ExitName: exitName,
						Detail: fmt.Sprintf("long exit connector passes over room %d", crossed),
					})
				}
			}
		}
	}
	return findings
}

// longSpanCrossesRoom returns the roomId of an intervening occupied cell on the
// straight line from start by delta (exclusive of endpoints), or 0 if none.
// Only handles axis-aligned and pure-diagonal spans (the only shapes posDeltas produce).
func (r *mapper) longSpanCrossesRoom(start, delta positionDelta, srcId, dstId int) int {
	steps := max(absInt(delta.x), absInt(delta.y))
	if steps <= 1 {
		return 0
	}
	sx, sy := sign(delta.x), sign(delta.y)
	byCell := map[[3]int]int{}
	for id, n := range r.crawledRooms {
		byCell[[3]int{n.Pos.x, n.Pos.y, n.Pos.z}] = id
	}
	for i := 1; i < steps; i++ {
		cell := [3]int{start.x + sx*i, start.y + sy*i, start.z}
		if id, ok := byCell[cell]; ok && id != srcId && id != dstId {
			return id
		}
	}
	return 0
}

func sign(v int) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}
