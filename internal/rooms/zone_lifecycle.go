package rooms

import "fmt"

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
