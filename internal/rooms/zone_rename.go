package rooms

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

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

// ValidateZoneRename checks a proposed new zone name against the existing set.
//
// The folder check matters as much as the name check: ZoneNameSanitize only
// lowercases and turns spaces into underscores, so "Amber Valley" and
// "Amber_Valley" are different display names occupying the SAME directory.
// Renaming onto a live zone's folder would collide on disk.
func ValidateZoneRename(oldName, newName string, existing []string) error {
	newName = strings.TrimSpace(newName)
	if len(newName) < 2 {
		return errors.New("zone name must be at least 2 characters")
	}
	if err := ValidateZoneName(newName); err != nil {
		return err
	}
	if newName == oldName {
		return errors.New("new name is the same as the current name")
	}
	for _, z := range existing {
		if z == newName {
			return fmt.Errorf("zone %q already exists", newName)
		}
	}
	// Exclude the zone being renamed: its own folder is the one moving, so
	// re-casing a name (Stillwater -> StillWater) is legal.
	others := make([]string, 0, len(existing))
	for _, z := range existing {
		if z != oldName {
			others = append(others, z)
		}
	}
	if clash := ZoneFolderCollision(newName, others); clash != "" {
		return fmt.Errorf("zone folder %q is already used by zone %q", ZoneNameSanitize(newName), clash)
	}
	return nil
}

// rewriteZoneField replaces the value of the top-level `zone:` key in a YAML
// file, leaving every other byte alone.
//
// This is deliberately a text edit, not a load-and-re-marshal. Round-tripping
// a room through yaml reflows every description and noun block (`>-` becomes
// `|`, wrapping changes), producing an enormous diff for a one-word change.
//
// "Top-level" means column zero: a `zone:` inside a description block scalar,
// or a nested `subzone:`, must survive untouched. Only the first match is
// replaced — a YAML document cannot legally have two top-level `zone:` keys,
// and stopping at the first avoids corrupting anything odd further down.
//
// A file with no top-level `zone:` is not an error: five of the ten zone trees
// (behaviors, schedules, shops, and both .instances) are folder-only.
func rewriteZoneField(path, newZone string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(raw)

	eol := "\n"
	if strings.Contains(content, "\r\n") {
		eol = "\r\n"
	}

	lines := strings.Split(content, eol)
	for i, line := range lines {
		// Column zero only — no leading whitespace.
		if strings.HasPrefix(line, "zone:") {
			lines[i] = "zone: " + newZone
			return os.WriteFile(path, []byte(strings.Join(lines, eol)), 0644)
		}
	}
	return nil // folder-only tree; nothing to rewrite
}
