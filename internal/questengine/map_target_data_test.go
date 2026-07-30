package questengine

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// repoRootFromPkg returns the repo root from this package's directory. Go test
// binaries run with CWD set to their own package dir, so any test that reads
// _datafiles/ must climb out first (precedent: internal/web/auth_test.go).
func repoRootFromPkg(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

var (
	reRoomId    = regexp.MustCompile(`(?m)^roomid:\s*(\d+)`)
	reMapTarget = regexp.MustCompile(`(?m)^\s*map_target:\s*(-?\d+)`)
	reQuestId   = regexp.MustCompile(`(?m)^questid:\s*(\d+)`)
)

// loadRoomIds collects every authored room id from the world tree.
func loadRoomIds(t *testing.T, root string) map[int]bool {
	t.Helper()
	ids := map[int]bool{}
	roomsDir := filepath.Join(root, "_datafiles", "world", "dogmud", "rooms")
	err := filepath.Walk(roomsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return err
		}
		// zone-config.yaml carries a roomid (the zone entrance) but is not a room.
		if filepath.Base(path) == "zone-config.yaml" {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		if m := reRoomId.FindSubmatch(b); m != nil {
			id, _ := strconv.Atoi(string(m[1]))
			ids[id] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking rooms: %v", err)
	}
	if len(ids) == 0 {
		t.Fatal("no rooms parsed — the room tree moved or the parser broke")
	}
	return ids
}

// TestQuestMapTargetsPointAtRealRooms is a data guard for the minimap quest
// marker. A `map_target` naming a room that does not exist yields a marker the
// client can never draw — silently, with no boot error and no test failure
// anywhere else. This walks every quest file and fails on the first bad id.
//
// Valid values: a real room id, or -1 meaning "deliberately no marker".
// 0/absent is also fine — the engine then infers from a room_enter trigger.
func TestQuestMapTargetsPointAtRealRooms(t *testing.T) {
	root := repoRootFromPkg(t)
	rooms := loadRoomIds(t, root)

	questDir := filepath.Join(root, "_datafiles", "world", "dogmud", "quests")
	entries, err := os.ReadDir(questDir)
	if err != nil {
		t.Fatalf("reading quest dir: %v", err)
	}

	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(questDir, e.Name()))
		if rerr != nil {
			t.Fatalf("reading %s: %v", e.Name(), rerr)
		}
		for _, m := range reMapTarget.FindAllSubmatch(b, -1) {
			v, _ := strconv.Atoi(string(m[1]))
			checked++
			if v == -1 || v == 0 {
				continue // deliberate no-marker / fall through to inference
			}
			if v < -1 {
				t.Errorf("%s: map_target %d is not a valid value (want a room id, -1, or 0)",
					e.Name(), v)
				continue
			}
			if !rooms[v] {
				t.Errorf("%s: map_target %d does not match any authored room — "+
					"the minimap marker can never render", e.Name(), v)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no map_target values found — the quest dir moved or the parser broke")
	}
	t.Logf("checked %d map_target values across %d quest files", checked, len(entries))
}

// TestQuestMapTargetsAreSameZoneAsTheirQuest is a WARNING-ONLY probe, not a
// gate. A marker in a different zone from the rest of its quest renders
// nothing: mapper.Snapshot only emits the current zone's visited rooms (so no
// marker token exists), and mapper.GetPath resolves its mapper from the START
// room's zone (so no next-step arrow either). Cross-zone guidance is a known
// deferred gap, so this only reports — failing here would block legitimate
// multi-zone quests that we cannot fix by authoring.
func TestQuestMapTargetsAreSameZoneAsTheirQuest(t *testing.T) {
	root := repoRootFromPkg(t)

	// room id -> zone
	zoneOf := map[int]string{}
	roomsDir := filepath.Join(root, "_datafiles", "world", "dogmud", "rooms")
	reZone := regexp.MustCompile(`(?m)^zone:\s*(.+)$`)
	_ = filepath.Walk(roomsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") ||
			filepath.Base(path) == "zone-config.yaml" {
			return nil
		}
		b, _ := os.ReadFile(path)
		m := reRoomId.FindSubmatch(b)
		z := reZone.FindSubmatch(b)
		if m != nil && z != nil {
			id, _ := strconv.Atoi(string(m[1]))
			zoneOf[id] = strings.TrimSpace(string(z[1]))
		}
		return nil
	})

	questDir := filepath.Join(root, "_datafiles", "world", "dogmud", "quests")
	entries, _ := os.ReadDir(questDir)
	spanning := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, _ := os.ReadFile(filepath.Join(questDir, e.Name()))
		zones := map[string]bool{}
		for _, m := range reMapTarget.FindAllSubmatch(b, -1) {
			v, _ := strconv.Atoi(string(m[1]))
			if v > 0 {
				if z, ok := zoneOf[v]; ok {
					zones[z] = true
				}
			}
		}
		if len(zones) > 1 {
			spanning++
			names := make([]string, 0, len(zones))
			for z := range zones {
				names = append(names, z)
			}
			qid := "?"
			if m := reQuestId.FindSubmatch(b); m != nil {
				qid = string(m[1])
			}
			t.Logf("quest %s (%s) has markers in %d zones %v — guidance goes dark at each crossing",
				qid, e.Name(), len(zones), names)
		}
	}
	t.Logf("%d quests have cross-zone markers (known deferred gap: no cross-zone pathing)", spanning)
}
