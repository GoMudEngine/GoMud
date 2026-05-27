package goals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func resetCache() {
	cacheMu.Lock()
	cache = map[int]*MobGoals{}
	nameByMobId = map[int]string{}
	cacheMu.Unlock()
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	resetCache()
	mg := &MobGoals{
		MobId:      371,
		NextGoalId: 2,
		Goals: []*Goal{
			{Id: "g1", Type: "revenge", Priority: 70,
				CreatedAt: time.Now().UTC().Truncate(time.Second)},
		},
	}
	cacheStoreForTest("tova", mg)
	if err := saveToDisk(371, "tova"); err != nil {
		t.Fatalf("save: %v", err)
	}
	resetCache()
	got := loadFromDisk(371, "tova")
	if got == nil {
		t.Fatal("load returned nil after save")
	}
	if got.MobId != 371 || got.NextGoalId != 2 || len(got.Goals) != 1 {
		t.Fatalf("round trip lost data: %+v", got)
	}
	if got.Goals[0].Id != "g1" {
		t.Errorf("goal id lost: %q", got.Goals[0].Id)
	}
}

func TestLoadFromDisk_MissingFile(t *testing.T) {
	resetCache()
	if got := loadFromDisk(99999, "nobody"); got != nil {
		t.Errorf("expected nil for missing file, got %+v", got)
	}
}

func TestLoadFromDisk_CorruptYAML(t *testing.T) {
	resetCache()
	path := goalPath(99998, "broken")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not: valid: yaml: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadFromDisk(99998, "broken"); got != nil {
		t.Errorf("expected nil for corrupt file, got %+v", got)
	}
}

func TestSaveToDisk_AtomicRename(t *testing.T) {
	resetCache()
	mg := &MobGoals{MobId: 200, NextGoalId: 1, Goals: nil}
	cacheStoreForTest("phantom", mg)
	if err := saveToDisk(200, "phantom"); err != nil {
		t.Fatalf("save: %v", err)
	}
	path := goalPath(200, "phantom")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("final file missing: %v", err)
	}
	tmp := path + ".tmp"
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("expected .tmp removed after rename, stat err=%v", err)
	}
}

func TestGoalPath_UsesOverride(t *testing.T) {
	override := os.Getenv("DOGMUD_GOALS_DIR_OVERRIDE")
	if override == "" {
		t.Skip("DOGMUD_GOALS_DIR_OVERRIDE unset; TestMain misconfigured?")
	}
	got := goalPath(371, "tova")
	if !strings.HasPrefix(got, override) {
		t.Errorf("goalPath = %q, want prefix %q", got, override)
	}
}
