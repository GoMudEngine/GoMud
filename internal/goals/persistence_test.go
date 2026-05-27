package goals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
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

func TestMobGoals_NewSelectionFields_RoundTrip(t *testing.T) {
	mg := &MobGoals{
		MobId:             371,
		NextGoalId:        4,
		CurrentGoalId:     "g2",
		CurrentSinceRound: 12450,
		LastSwitchRound:   12450,
		Goals: []*Goal{
			{Id: "g1", Type: "revenge", Priority: 70, CreatedAt: time.Date(2026, 5, 26, 14, 30, 0, 0, time.UTC)},
			{Id: "g2", Type: "wealth-target", Priority: 30, CreatedAt: time.Date(2026, 5, 26, 14, 31, 0, 0, time.UTC)},
		},
	}
	out, err := yaml.Marshal(mg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got MobGoals
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.CurrentGoalId != "g2" {
		t.Errorf("CurrentGoalId: got %q, want %q", got.CurrentGoalId, "g2")
	}
	if got.CurrentSinceRound != 12450 {
		t.Errorf("CurrentSinceRound: got %d, want %d", got.CurrentSinceRound, 12450)
	}
	if got.LastSwitchRound != 12450 {
		t.Errorf("LastSwitchRound: got %d, want %d", got.LastSwitchRound, 12450)
	}
	if len(got.Goals) != 2 {
		t.Fatalf("Goals: got %d, want 2", len(got.Goals))
	}
}

func TestMobGoals_SeededFromArchetype_RoundTrip(t *testing.T) {
	mg := &MobGoals{
		MobId:               371,
		NextGoalId:          2,
		SeededFromArchetype: true,
		Goals: []*Goal{
			{Id: "g1", Type: "survival", Priority: 80,
				CreatedAt: time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)},
		},
	}
	out, err := yaml.Marshal(mg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got MobGoals
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got.SeededFromArchetype {
		t.Errorf("SeededFromArchetype: got false, want true")
	}
}

func TestMobGoals_LegacyFile_LoadsWithSentinelFalse(t *testing.T) {
	legacy := `mob_id: 371
next_goal_id: 2
goals:
  - id: g1
    type: survival
    priority: 80
    created_at: 2026-05-27T10:00:00Z
`
	var got MobGoals
	if err := yaml.Unmarshal([]byte(legacy), &got); err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}
	if got.SeededFromArchetype {
		t.Errorf("SeededFromArchetype: got true, want false (legacy file)")
	}
}

func TestMobGoals_LegacyFile_LoadsWithZeroSelectionFields(t *testing.T) {
	// A 4.1-era file with no selection fields.
	legacy := `mob_id: 371
next_goal_id: 2
goals:
  - id: g1
    type: revenge
    priority: 70
    created_at: 2026-05-26T14:30:00Z
`
	var got MobGoals
	if err := yaml.Unmarshal([]byte(legacy), &got); err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}
	if got.MobId != 371 || got.NextGoalId != 2 {
		t.Errorf("legacy fields: got mob=%d next=%d, want mob=371 next=2", got.MobId, got.NextGoalId)
	}
	if got.CurrentGoalId != "" {
		t.Errorf("CurrentGoalId: got %q, want empty", got.CurrentGoalId)
	}
	if got.CurrentSinceRound != 0 || got.LastSwitchRound != 0 {
		t.Errorf("round fields: got since=%d switch=%d, want 0/0", got.CurrentSinceRound, got.LastSwitchRound)
	}
	if len(got.Goals) != 1 {
		t.Fatalf("Goals: got %d, want 1", len(got.Goals))
	}
}
