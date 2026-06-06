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
