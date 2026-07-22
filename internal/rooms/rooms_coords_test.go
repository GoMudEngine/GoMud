package rooms

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestRoom_CoordYamlRoundTrip(t *testing.T) {
	in := Room{RoomId: 5, Zone: "Test", Title: "t", Description: "d", X: 3, Y: -1, Z: 0, Plane: 2}
	b, err := yaml.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Room
	if err := yaml.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.X != 3 || out.Y != -1 || out.Z != 0 || out.Plane != 2 {
		t.Errorf("coords lost: got x=%d y=%d z=%d plane=%d", out.X, out.Y, out.Z, out.Plane)
	}

	// Plane 0 / origin coords should be omitempty (not clutter every YAML).
	zero, _ := yaml.Marshal(Room{RoomId: 1, Zone: "Z", Title: "t", Description: "d"})
	if s := string(zero); strings.Contains(s, "\nx:") || strings.Contains(s, "\nplane:") {
		t.Errorf("zero coords should be omitempty; got:\n%s", s)
	}
}
