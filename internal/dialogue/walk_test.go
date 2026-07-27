package dialogue

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWalkAllFiles_VisitsEveryZoneFile(t *testing.T) {
	dir := overrideDataFilesDir(t)

	write := func(zone string, mobId int, body string) {
		p := filepath.Join(dir, "dialogue", zone)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, fmt.Sprintf("%d.yaml", mobId)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("townzone", 9, "mobid: 9\nzone: townzone\npatterns:\n  - keywords: [\"hello\"]\n    responses: [\"Hi.\"]\n")
	write("wildzone", 12, "mobid: 12\nzone: wildzone\npatterns:\n  - keywords: [\"snake\"]\n    responses: [\"Hiss.\"]\n")

	seen := map[int]string{}
	WalkAllFiles(func(mobId int, zone string, df *DialogueFile) {
		seen[mobId] = zone
		if df == nil || len(df.Patterns) != 1 {
			t.Errorf("mob %d: file not parsed", mobId)
		}
	})

	if seen[9] != "townzone" || seen[12] != "wildzone" {
		t.Fatalf("walker missed files: %v", seen)
	}
}
