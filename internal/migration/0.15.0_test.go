package migration

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"
)

// writeRoom writes rooms/<zoneFolder>/<id>.yaml with the given exits fragment.
func writeRoom(t *testing.T, root, zoneFolder string, id int, exitsYAML string) {
	t.Helper()
	dir := filepath.Join(root, zoneFolder)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	body := "roomid: " + itoa(id) + "\nzone: " + zoneFolder + "\ntitle: t\ndescription: d\n" + exitsYAML
	if err := os.WriteFile(filepath.Join(dir, itoa(id)+".yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func assertCoord(t *testing.T, root, zoneFolder string, id, x, y, z, plane int) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, zoneFolder, itoa(id)+".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		X, Y, Z, Plane int
	}
	if err := yaml.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.X != x || got.Y != y || got.Z != z || got.Plane != plane {
		t.Errorf("room %d: got (x=%d y=%d z=%d plane=%d), want (%d %d %d %d)\n%s",
			id, got.X, got.Y, got.Z, got.Plane, x, y, z, plane, raw)
	}
}

// A minimal 3-room line: 1 -east-> 2 -east-> 3, all zone "t". After migration
// authored coords equal the crawl (0,0,0),(1,0,0),(2,0,0) on plane 0.
func TestBackfillCoords_LosslessLine(t *testing.T) {
	dir := t.TempDir()
	writeRoom(t, dir, "t", 1, "exits:\n  east:\n    roomid: 2\n")
	writeRoom(t, dir, "t", 2, "exits:\n  west:\n    roomid: 1\n  east:\n    roomid: 3\n")
	writeRoom(t, dir, "t", 3, "exits:\n  west:\n    roomid: 2\n")

	if err := backfillCoordsInDir(dir, false); err != nil {
		t.Fatalf("migration: %v", err)
	}
	assertCoord(t, dir, "t", 1, 0, 0, 0, 0)
	assertCoord(t, dir, "t", 2, 1, 0, 0, 0)
	assertCoord(t, dir, "t", 3, 2, 0, 0, 0)
}

// Two disconnected components (a portal exit does NOT merge them): the larger
// gets plane 0, the smaller a fresh plane.
func TestBackfillCoords_PortalSeparatesPlanes(t *testing.T) {
	dir := t.TempDir()
	// Overworld component (3 rooms), connected spatially.
	writeRoom(t, dir, "over", 1, "exits:\n  east:\n    roomid: 2\n  enter:\n    roomid: 10\n") // 'enter' = portal (non-spatial)
	writeRoom(t, dir, "over", 2, "exits:\n  west:\n    roomid: 1\n  east:\n    roomid: 3\n")
	writeRoom(t, dir, "over", 3, "exits:\n  west:\n    roomid: 2\n")
	// Interior component (2 rooms), reachable only via the portal.
	writeRoom(t, dir, "inn", 10, "exits:\n  north:\n    roomid: 11\n  leave:\n    roomid: 1\n")
	writeRoom(t, dir, "inn", 11, "exits:\n  south:\n    roomid: 10\n")

	if err := backfillCoordsInDir(dir, false); err != nil {
		t.Fatalf("migration: %v", err)
	}
	// Larger component (over, 3 rooms) -> plane 0.
	assertCoord(t, dir, "over", 1, 0, 0, 0, 0)
	// Interior is a separate component -> a different (nonzero) plane. Assert
	// rooms 10 & 11 share a plane that is NOT 0.
	p10 := planeOf(t, dir, "inn", 10)
	p11 := planeOf(t, dir, "inn", 11)
	if p10 == 0 || p10 != p11 {
		t.Errorf("interior rooms should share a nonzero plane; got 10=%d 11=%d", p10, p11)
	}
}

func planeOf(t *testing.T, root, zoneFolder string, id int) int {
	t.Helper()
	raw, _ := os.ReadFile(filepath.Join(root, zoneFolder, itoa(id)+".yaml"))
	var got struct{ Plane int }
	yaml.Unmarshal(raw, &got)
	return got.Plane
}
