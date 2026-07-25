package rooms

import "fmt"

// zoneRenameSources injects the world lookups the rename guard needs, so the
// policy is testable without a loaded world.
type zoneRenameSources struct {
	playersInZone func(zone string) []string
}

// ZoneRenameBlockersWith reports why a zone cannot be renamed right now.
//
// Only players block. Rooms, mobs and authored content are all rewritten by
// the rename itself; a player standing in the zone is different, because the
// files move under their feet and their in-memory room pointer would go stale.
func ZoneRenameBlockersWith(zone string, src zoneRenameSources) []ZoneBlocker {
	out := []ZoneBlocker{}
	for _, p := range src.playersInZone(zone) {
		out = append(out, ZoneBlocker{Kind: "player", Id: p})
	}
	return out
}

// ZoneRenameBlockers is the production wiring.
func ZoneRenameBlockers(zone string) []ZoneBlocker {
	return ZoneRenameBlockersWith(zone, zoneRenameSources{
		playersInZone: func(z string) []string {
			out := []string{}
			for _, id := range GetAllRoomIds() {
				r := LoadRoomTemplate(id)
				if r == nil || r.Zone != z {
					continue
				}
				if n := len(r.GetPlayers()); n > 0 {
					out = append(out, fmt.Sprintf("%d player(s) in room %d", n, id))
				}
			}
			return out
		},
	})
}
