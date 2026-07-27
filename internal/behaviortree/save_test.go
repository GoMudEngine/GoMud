package behaviortree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// overrideBTDataFilesDir points configs.DataFiles at a temp dir (the
// dialogue save_test pattern: chdir repo root, config-overrides.yaml,
// CONFIG_PATH, reload; cleanup restores).
func overrideBTDataFilesDir(t *testing.T) string {
	t.Helper()
	mudlog.SetupLogger(nil, `LOW`, ``, false)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir(%q): %v", repoRoot, err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	dir := t.TempDir()
	overridePath := filepath.Join(dir, "config-overrides.yaml")
	overrideBytes := []byte("FilePaths:\n  DataFiles: " + filepath.ToSlash(dir) + "\n  CarefulSaveFiles: false\n")
	if err := os.WriteFile(overridePath, overrideBytes, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("CONFIG_PATH", filepath.ToSlash(overridePath))
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig with override: %v", err)
	}
	t.Cleanup(func() {
		os.Unsetenv("CONFIG_PATH")
		_ = configs.ReloadConfig()
	})
	return dir
}

func validProbeTree() TreeDef {
	return TreeDef{
		Notes: "writer probe",
		Tree: NodeDef{Type: "selector", Children: []NodeDef{
			{Type: "action", Event: "mob_idle", Do: "flee", Note: "example"},
		}},
	}
}

func TestSaveArchetype_WritesValidatesAndReloadsCache(t *testing.T) {
	overrideBTDataFilesDir(t)
	e := GetEngine()
	const name = "writer_probe"
	t.Cleanup(func() { e.EvictArchetype(name) })

	// Simulate the pre-save world state: negative-cached.
	e.mu.Lock()
	e.noArchetype[name] = true
	delete(e.archetypes, name)
	e.mu.Unlock()

	tree := validProbeTree()
	tree.GoalWeights = map[string]float64{"survival": 2.0}
	warns, err := SaveArchetype(name, tree)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	_ = warns

	if _, err := os.Stat(GetArchetypePath(name)); err != nil {
		t.Fatalf("archetype file missing: %v", err)
	}
	e.mu.RLock()
	_, cached := e.archetypes[name]
	neg := e.noArchetype[name]
	e.mu.RUnlock()
	if !cached || neg {
		t.Fatalf("save must reload the cache and clear the negative (cached=%v neg=%v)", cached, neg)
	}
	if w := e.GetArchetypeGoalWeights(name); w["survival"] != 2.0 {
		t.Fatalf("goal weights not reloaded: %v", w)
	}
}

func TestSaveArchetype_RefusesUnknownDoAndWritesNothing(t *testing.T) {
	overrideBTDataFilesDir(t)
	const name = "writer_probe_bad"
	tree := validProbeTree()
	tree.Tree.Children[0].Do = "no_such_action"
	if _, err := SaveArchetype(name, tree); err == nil {
		t.Fatal("unknown do must refuse")
	}
	if _, err := os.Stat(GetArchetypePath(name)); !os.IsNotExist(err) {
		t.Fatal("refused save must write nothing")
	}
}

func TestSaveArchetype_RefusesUnknownEvent(t *testing.T) {
	overrideBTDataFilesDir(t)
	tree := validProbeTree()
	tree.Tree.Children[0].Event = "mob_death" // the caravan-wagon incident
	_, err := SaveArchetype("writer_probe_ev", tree)
	if err == nil || !strings.Contains(err.Error(), "mob_death") {
		t.Fatalf("unknown event must refuse verbatim, got %v", err)
	}
}

func TestDeleteArchetype_EvictsAndSetsNegative(t *testing.T) {
	overrideBTDataFilesDir(t)
	e := GetEngine()
	const name = "writer_probe_del"
	if _, err := SaveArchetype(name, validProbeTree()); err != nil {
		t.Fatal(err)
	}
	if err := DeleteArchetype(name); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(GetArchetypePath(name)); !os.IsNotExist(err) {
		t.Fatal("file should be gone")
	}
	e.mu.RLock()
	_, cached := e.archetypes[name]
	neg := e.noArchetype[name]
	_, weights := e.archetypeGoalWeights[name]
	e.mu.RUnlock()
	if cached || !neg || weights {
		t.Fatalf("delete must evict + set negative + clear goal maps (cached=%v neg=%v weights=%v)", cached, neg, weights)
	}
}

func TestSaveMobTreeAndRoomTree_RoundTripAndCache(t *testing.T) {
	overrideBTDataFilesDir(t)
	e := GetEngine()
	const mobId, roomId = 424242, 535353
	t.Cleanup(func() { e.EvictTree(mobId); e.EvictRoomTree(roomId) })

	if _, err := SaveMobTree(mobId, "Probe Zone", "Writer Probe", validProbeTree()); err != nil {
		t.Fatalf("mob tree save: %v", err)
	}
	if e.GetTree(mobId) == nil || e.HasNoTree(mobId) {
		t.Fatal("mob tree save must load the cache and clear the negative")
	}
	if err := DeleteMobTree(mobId, "Probe Zone", "Writer Probe"); err != nil {
		t.Fatal(err)
	}
	if e.GetTree(mobId) != nil || !e.HasNoTree(mobId) {
		t.Fatal("mob tree delete must evict and set the negative (fallback semantics)")
	}

	if _, err := SaveRoomTree(roomId, "Probe Zone", validProbeTree()); err != nil {
		t.Fatalf("room tree save: %v", err)
	}
	if e.GetRoomTree(roomId) == nil || e.HasNoRoomTree(roomId) {
		t.Fatal("room tree save must load the cache and clear the negative")
	}
	if err := DeleteRoomTree(roomId, "Probe Zone"); err != nil {
		t.Fatal(err)
	}
	if e.GetRoomTree(roomId) != nil || !e.HasNoRoomTree(roomId) {
		t.Fatal("room tree delete must evict and set the negative")
	}
}

func TestCreateArchetype_SkeletonCompiles(t *testing.T) {
	overrideBTDataFilesDir(t)
	e := GetEngine()
	const name = "writer_probe_new"
	t.Cleanup(func() { e.EvictArchetype(name) })

	if err := CreateArchetype(name); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := CreateArchetype(name); err == nil {
		t.Fatal("create must refuse an existing archetype")
	}
	data, err := os.ReadFile(GetArchetypePath(name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadTreeFromBytes(data); err != nil {
		t.Fatalf("skeleton must compile: %v", err)
	}
}

func TestRawFileHasHandComments(t *testing.T) {
	dir := t.TempDir()
	commented := filepath.Join(dir, "a.yaml")
	os.WriteFile(commented, []byte("# rationale\ntree:\n  type: selector\n"), 0644)
	clean := filepath.Join(dir, "b.yaml")
	os.WriteFile(clean, []byte("tree:\n  type: selector\n  children: []\n"), 0644)
	if !RawFileHasHandComments(commented) {
		t.Fatal("comment lines must be detected")
	}
	if RawFileHasHandComments(clean) {
		t.Fatal("clean marshal output must not flag")
	}
}
