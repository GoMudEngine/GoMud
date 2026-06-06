package mapper

import "github.com/GoMudEngine/GoMud/internal/rooms"

// SnapshotExit is one classified spatial edge for the web map renderer.
type SnapshotExit struct {
	ToRoomId int      `json:"to"`
	DX       int      `json:"dx"`
	DY       int      `json:"dy"`
	DZ       int      `json:"dz"`
	Kind     ExitKind `json:"kind"`
	Locked   bool     `json:"locked,omitempty"`
	Secret   bool     `json:"secret,omitempty"`
	OneWay   bool     `json:"oneway,omitempty"`
	Gate     bool     `json:"gate,omitempty"`
	Stub     bool     `json:"stub,omitempty"`   // destination not a visited same-zone room
	ToZone   string   `json:"tozone,omitempty"` // set when the destination is in a different zone
}

// SnapshotRoom is one room placed in the zone coordinate space.
type SnapshotRoom struct {
	RoomId int            `json:"num"`
	X      int            `json:"x"`
	Y      int            `json:"y"`
	Z      int            `json:"z"`
	Symbol string         `json:"symbol"`
	Biome  string         `json:"biome"`
	Name   string         `json:"name"`           // room title (client tooltip / identify)
	Tags   []string       `json:"tags,omitempty"` // service tags: bank | storage | trainer | shop
	Exits  []SnapshotExit `json:"exits"`
}

// Snapshot returns the visited rooms of this zone with classified exits to
// other visited rooms. Exits to unvisited or uncrawled rooms are omitted (fog
// of war). The exit Kind drives client rendering (normal/long/wrap/vertical).
func (r *mapper) Snapshot(visited map[int]struct{}) []SnapshotRoom {
	out := make([]SnapshotRoom, 0, len(visited))

	// Iteration order is nondeterministic; that's fine — the client places rooms
	// by (x,y,z), not by slice order.
	for id, n := range r.crawledRooms {
		if _, ok := visited[id]; !ok {
			continue
		}

		sym := n.Symbol
		if sym == 0 {
			sym = defaultMapSymbol
		}
		sr := SnapshotRoom{
			RoomId: id,
			X:      n.Pos.x,
			Y:      n.Pos.y,
			Z:      n.Pos.z,
			Symbol: string(sym),
		}
		// Biome name comes from the room's biome (not n.Legend, which may hold a
		// per-room MapLegend override like "Townsquare"); the client uses it for tinting.
		srcZone := ""
		if room := rooms.LoadRoom(id); room != nil {
			sr.Name = room.Title
			srcZone = room.Zone
			if b := room.GetBiome(); b != nil {
				sr.Biome = b.Name
			}
			// Service tags so the client can highlight important rooms.
			if room.IsBank {
				sr.Tags = append(sr.Tags, "bank")
			}
			if room.IsStorage {
				sr.Tags = append(sr.Tags, "storage")
			}
			if len(room.SkillTraining) > 0 {
				sr.Tags = append(sr.Tags, "trainer")
			}
			if len(room.GetMobs(rooms.FindMerchant)) > 0 {
				sr.Tags = append(sr.Tags, "shop")
			}
		}

		for _, e := range n.Exits {
			se := SnapshotExit{
				ToRoomId: e.RoomId,
				DX:       e.Direction.x,
				DY:       e.Direction.y,
				DZ:       e.Direction.z,
				Locked:   e.LockDifficulty > 0,
				Secret:   e.Secret,
				OneWay:   e.OneWay,
				Gate:     e.Gate,
			}
			dst, crawled := r.crawledRooms[e.RoomId]
			_, vis := visited[e.RoomId]
			if crawled && vis {
				actual := positionDelta{x: dst.Pos.x - n.Pos.x, y: dst.Pos.y - n.Pos.y, z: dst.Pos.z - n.Pos.z}
				se.Kind = classifyKind(e.Direction, actual)
			} else {
				se.Stub = true
				se.Kind = classifyKind(e.Direction, e.Direction) // nominal placement
			}
			if dr := rooms.LoadRoom(e.RoomId); dr != nil && dr.Zone != "" && dr.Zone != srcZone {
				se.ToZone = dr.Zone
			}
			sr.Exits = append(sr.Exits, se)
		}
		out = append(out, sr)
	}
	return out
}
