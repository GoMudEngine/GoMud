package mapper

import "github.com/GoMudEngine/GoMud/internal/rooms"

// SnapshotExit is one classified spatial edge for the web map renderer.
type SnapshotExit struct {
	ToRoomId int      `json:"to"`
	DX       int      `json:"dx"`
	DY       int      `json:"dy"`
	DZ       int      `json:"dz"`
	Kind     ExitKind `json:"kind"`
}

// SnapshotRoom is one room placed in the zone coordinate space.
type SnapshotRoom struct {
	RoomId int            `json:"num"`
	X      int            `json:"x"`
	Y      int            `json:"y"`
	Z      int            `json:"z"`
	Symbol string         `json:"symbol"`
	Biome  string         `json:"biome"`
	Exits  []SnapshotExit `json:"exits"`
}

// Snapshot returns the visited rooms of this zone with classified exits to
// other visited rooms. Exits to unvisited or uncrawled rooms are omitted (fog
// of war). The exit Kind drives client rendering (normal/long/wrap/vertical).
func (r *mapper) Snapshot(visited map[int]struct{}) []SnapshotRoom {
	out := make([]SnapshotRoom, 0, len(visited))

	for id, n := range r.crawledRooms {
		if _, ok := visited[id]; !ok {
			continue
		}

		sr := SnapshotRoom{
			RoomId: id,
			X:      n.Pos.x,
			Y:      n.Pos.y,
			Z:      n.Pos.z,
			Symbol: string(n.Symbol),
		}
		if room := rooms.LoadRoom(id); room != nil {
			if b := room.GetBiome(); b != nil {
				sr.Biome = b.Name
			}
		}

		for _, e := range n.Exits {
			dst, ok := r.crawledRooms[e.RoomId]
			if !ok {
				continue
			}
			if _, ok := visited[e.RoomId]; !ok {
				continue
			}
			actual := positionDelta{x: dst.Pos.x - n.Pos.x, y: dst.Pos.y - n.Pos.y, z: dst.Pos.z - n.Pos.z}
			sr.Exits = append(sr.Exits, SnapshotExit{
				ToRoomId: e.RoomId,
				DX:       e.Direction.x,
				DY:       e.Direction.y,
				DZ:       e.Direction.z,
				Kind:     classifyKind(e.Direction, actual),
			})
		}
		out = append(out, sr)
	}
	return out
}
