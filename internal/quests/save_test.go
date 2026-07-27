package quests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// pointQuestDataFilesAt redirects quest file I/O at a temp dir for the test's
// duration via the questsDataRoot package-var seam (the mobs.mobsDataRoot
// pattern — no configs.SetVal side effects on the real override file).
func pointQuestDataFilesAt(t *testing.T, dir string) {
	t.Helper()
	mudlog.SetupLogger(nil, `LOW`, ``, false)

	prevRoot := questsDataRoot
	questsDataRoot = func() string { return dir }
	t.Cleanup(func() { questsDataRoot = prevRoot })
}

// seedQuest installs a template into the cache (mirroring LoadDataFiles) and
// registers cleanup. Returns a value copy for the test to mutate freely.
func seedQuest(t *testing.T, id int, name string) Quest {
	t.Helper()

	q := Quest{QuestId: id, Name: name, Description: "A test quest.",
		Steps: []QuestStep{{Id: "start", Description: "Begin."}, {Id: "end", Description: "Done."}}}

	cp := q
	quests[id] = &cp

	t.Cleanup(func() {
		if prev, ok := quests[id]; ok {
			for _, f := range prev.Flags {
				delete(flagRegistry, flagKey(id, f.Key))
			}
		}
		delete(quests, id)
	})

	return q
}

func TestSaveQuest_WritesSwapsCacheAndFlags(t *testing.T) {
	dir := t.TempDir()
	pointQuestDataFilesAt(t, dir)
	q := seedQuest(t, 99901, "Test Errand")

	q.Description = "An updated description."
	q.Flags = []QuestFlagDef{{Key: "branch", Values: []string{"a", "b"}}}
	if err := SaveQuest(q); err != nil {
		t.Fatalf("save: %v", err)
	}

	p := filepath.Join(dir, "99901-test_errand.yaml")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected %s to exist: %v", p, err)
	}
	got := GetQuest("99901-start")
	if got == nil || got.Description != "An updated description." {
		t.Errorf("cache not swapped: %+v", got)
	}
	if vals, ok := GetFlagRegistry()["99901-branch"]; !ok || len(vals) != 2 {
		t.Errorf("flags not registered: %v", GetFlagRegistry()["99901-branch"])
	}
}

func TestSaveQuest_RenameMovesFile(t *testing.T) {
	dir := t.TempDir()
	pointQuestDataFilesAt(t, dir)
	q := seedQuest(t, 99902, "Old Name")

	if err := SaveQuest(q); err != nil {
		t.Fatalf("initial save: %v", err)
	}
	oldPath := filepath.Join(dir, "99902-old_name.yaml")
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("expected %s: %v", oldPath, err)
	}

	renamed := q
	renamed.Name = "New Name"
	if err := SaveQuest(renamed); err != nil {
		t.Fatalf("rename save: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old file %s should be gone after rename", oldPath)
	}
	if _, err := os.Stat(filepath.Join(dir, "99902-new_name.yaml")); err != nil {
		t.Errorf("new file should exist: %v", err)
	}
}

func TestSaveQuest_RefusesInvalid(t *testing.T) {
	dir := t.TempDir()
	pointQuestDataFilesAt(t, dir)
	q := seedQuest(t, 99903, "Broken Quest")

	q.Steps = []QuestStep{{Id: "start"}, {Id: "start"}} // duplicate step ids
	if err := SaveQuest(q); err == nil {
		t.Fatal("invalid quest must be refused")
	}
	if _, err := os.Stat(filepath.Join(dir, "99903-broken_quest.yaml")); !os.IsNotExist(err) {
		t.Error("nothing should have been written for a refused save")
	}
	if got := quests[99903]; len(got.Steps) != 2 || got.Steps[1].Id != "end" {
		t.Errorf("cache must be untouched by a refused save: %+v", got.Steps)
	}
}

func TestCreateNewQuestFile_SkeletonIsBootSafe(t *testing.T) {
	dir := t.TempDir()
	pointQuestDataFilesAt(t, dir)
	seed := seedQuest(t, 99904, "Ceiling Marker")

	id, err := CreateNewQuestFile("Fresh Errand")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id <= seed.QuestId {
		t.Fatalf("new id %d should exceed the cache max %d", id, seed.QuestId)
	}
	t.Cleanup(func() { delete(quests, id) })

	tmpl := quests[id]
	if tmpl == nil {
		t.Fatal("skeleton not in cache")
	}
	if err := tmpl.Validate(); err != nil {
		t.Errorf("skeleton must be boot-safe: %v", err)
	}
	if len(tmpl.Steps) != 2 || tmpl.Steps[0].Id != "start" || tmpl.Steps[1].Id != "end" {
		t.Errorf("skeleton steps wrong: %+v", tmpl.Steps)
	}
	if len(tmpl.Triggers) != 0 {
		t.Errorf("skeleton must have zero triggers")
	}
	if _, err := os.Stat(filepath.Join(dir, tmpl.Filename())); err != nil {
		t.Errorf("skeleton file missing: %v", err)
	}
}

func TestDeleteQuest_RemovesFileCacheAndFlags(t *testing.T) {
	dir := t.TempDir()
	pointQuestDataFilesAt(t, dir)
	q := seedQuest(t, 99905, "Doomed Quest")
	q.Flags = []QuestFlagDef{{Key: "fate", Values: []string{"sealed"}}}
	if err := SaveQuest(q); err != nil {
		t.Fatal(err)
	}

	p := filepath.Join(dir, "99905-doomed_quest.yaml")
	if err := DeleteQuest(99905); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("file %s should be gone", p)
	}
	if quests[99905] != nil {
		t.Error("cache entry should be gone")
	}
	if _, ok := GetFlagRegistry()["99905-fate"]; ok {
		t.Error("flag registration should be gone")
	}

	// Idempotent when the file is already missing.
	q2 := seedQuest(t, 99906, "Already Gone")
	if err := SaveQuest(q2); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "99906-already_gone.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := DeleteQuest(99906); err != nil {
		t.Fatalf("delete should succeed with the file already gone: %v", err)
	}
}
