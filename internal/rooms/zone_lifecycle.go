package rooms

import (
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ZoneFolderCollision reports the first zone in existing whose sanitized
// folder name matches newName's, or "" when the folder is free.
//
// ZoneNameSanitize only lowercases and converts spaces to underscores, so
// "Amber Valley", "amber valley" and "Amber_Valley" all map to the folder
// amber_valley. CreateZone's duplicate check compares DISPLAY names and so
// misses this, reaching os.Mkdir on a live zone's folder.
func ZoneFolderCollision(newName string, existing []string) string {
	folder := ZoneNameSanitize(newName)
	if folder == "" {
		return ""
	}
	for _, z := range existing {
		if ZoneNameSanitize(z) == folder {
			return z
		}
	}
	return ""
}

// ZoneBlocker is one reason a zone cannot be deleted.
type ZoneBlocker struct {
	Kind string `json:"kind"` // room | content | inbound-exit | player
	Id   string `json:"id"`   // human-readable identifier
}

// zoneBlockerSources injects every world lookup the scan needs so the policy
// is testable without a filesystem or a loaded world.
type zoneBlockerSources struct {
	roomIdsInZone  func(zone string) []int
	zoneRootRoomId func(zone string) int
	contentFiles   func(zone string) []string
	inboundExits   func(zone string) []string
	playersInZone  func(zone string) []string
}

// ZoneDeletionBlockersWith applies the deletion policy: a zone may be deleted
// only when it holds nothing but its root room, owns no authored content, has
// no exits pointing into it from other zones, and has no players inside.
//
// Shops and the two .instances/ trees are deliberately NOT blockers — they are
// regenerable living state, not authored work.
func ZoneDeletionBlockersWith(zone string, src zoneBlockerSources) []ZoneBlocker {
	out := []ZoneBlocker{}

	root := src.zoneRootRoomId(zone)
	for _, id := range src.roomIdsInZone(zone) {
		if id == root {
			continue
		}
		out = append(out, ZoneBlocker{Kind: "room", Id: fmt.Sprintf("room %d", id)})
	}
	for _, f := range src.contentFiles(zone) {
		out = append(out, ZoneBlocker{Kind: "content", Id: f})
	}
	for _, e := range src.inboundExits(zone) {
		out = append(out, ZoneBlocker{Kind: "inbound-exit", Id: e})
	}
	for _, p := range src.playersInZone(zone) {
		out = append(out, ZoneBlocker{Kind: "player", Id: p})
	}
	return out
}

// zoneContentDirs are the trees holding AUTHORED zone content. Presence of any
// file here blocks deletion.
func zoneContentDirs() []string {
	return []string{"mobs", "dialogue", "behaviors", "schedules", "caravans", "foragers"}
}

// zoneAllDirs is every tree a zone owns, including regenerable living state
// (shops + the two .instances trees). Deletion removes all of them.
func zoneAllDirs() []string {
	return append(zoneContentDirs(), "rooms", "rooms.instances", "mobs.instances", "shops")
}

// ZoneDeletionBlockers is the production wiring of the scan.
func ZoneDeletionBlockers(zone string) []ZoneBlocker {
	return ZoneDeletionBlockersWith(zone, zoneBlockerSources{
		roomIdsInZone: func(z string) []int {
			out := []int{}
			for _, id := range GetAllRoomIds() {
				if r := LoadRoomTemplate(id); r != nil && r.Zone == z {
					out = append(out, id)
				}
			}
			return out
		},
		zoneRootRoomId: func(z string) int {
			id, _ := GetZoneRoot(z)
			return id
		},
		contentFiles: func(z string) []string {
			out := []string{}
			base := configs.GetFilePathsConfig().DataFiles.String()
			for _, d := range zoneContentDirs() {
				dir := util.FilePath(base, "/", d, "/", ZoneNameSanitize(z))
				entries, err := os.ReadDir(dir)
				if err != nil {
					continue // tree absent for this zone
				}
				for _, e := range entries {
					if !e.IsDir() {
						out = append(out, d+"/"+ZoneNameSanitize(z)+"/"+e.Name())
					}
				}
			}
			return out
		},
		inboundExits: func(z string) []string {
			out := []string{}
			for _, id := range GetAllRoomIds() {
				src := LoadRoomTemplate(id)
				if src == nil || src.Zone == z {
					continue // same-zone exits die with the zone
				}
				for name, ex := range src.Exits {
					dst := LoadRoomTemplate(ex.RoomId)
					if dst != nil && dst.Zone == z {
						out = append(out, fmt.Sprintf("room %d (%s) %s", id, src.Zone, name))
					}
				}
			}
			return out
		},
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

// DeleteZone removes every directory a zone owns and drops it from the room
// manager. It re-checks the blockers itself — never trust the caller.
//
// The caller is responsible for invalidating the mapper cache afterwards;
// internal/rooms cannot import internal/mapper (mapper imports rooms).
func DeleteZone(zone string) error {
	if _, ok := roomManager.zones[zone]; !ok {
		return fmt.Errorf("zone %q does not exist", zone)
	}
	if b := ZoneDeletionBlockers(zone); len(b) > 0 {
		return fmt.Errorf("zone %q is not empty (%d blockers)", zone, len(b))
	}

	// Collect the zone's rooms BEFORE touching the filesystem. LoadRoomTemplate
	// reads from DISK (loadRoomFromFile), so once the directories are gone it
	// returns nil for every room and the cache purge below would silently do
	// nothing, leaving stale rooms in roomManager.
	doomed := []int{}
	for _, id := range GetAllRoomIds() {
		if r := LoadRoomTemplate(id); r != nil && r.Zone == zone {
			doomed = append(doomed, id)
		}
	}

	base := configs.GetFilePathsConfig().DataFiles.String()
	folder := ZoneNameSanitize(zone)
	for _, d := range zoneAllDirs() {
		dir := util.FilePath(base, "/", d, "/", folder)
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("removing %s: %w", dir, err)
		}
	}

	// ClearRoomCache purges rooms, roomsWithUsers, roomsWithMobs AND
	// roomIdToFileCache — the last matters because GetAllRoomIds iterates that
	// map, so a stale entry would keep reporting a deleted room forever. It
	// errors when the room was never cached in memory, which is fine here.
	for _, id := range doomed {
		_ = ClearRoomCache(id)
		delete(roomManager.roomIdToFileCache, id)
	}
	delete(roomManager.zones, zone)
	return nil
}
