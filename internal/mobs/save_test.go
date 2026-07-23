package mobs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/species"
)

// pointMobDataFilesAt redirects mob file I/O (and the behavior_archetype
// os.Stat check) at a temp dir for the test's duration, and seeds a minimal
// species so ValidateMobSpec's speciesid check doesn't fail every save test.
//
// This overrides the mobsDataRoot package var directly rather than going
// through configs.SetVal("FilePaths.DataFiles", dir): SetVal persists the
// override to the real config-overrides.yaml on disk (see
// configs.overridePath, which resolves relative to the CURRENT DataFiles
// value) — running that in a unit test risks clobbering the developer's real
// override file with a path that points at a since-deleted t.TempDir(). The
// package-var seam mirrors items.itemsBasePath's test-injection pattern
// (internal/items/save_test.go) and has no on-disk side effects outside the
// temp dir itself.
func pointMobDataFilesAt(t *testing.T, dir string) {
	t.Helper()

	prevRoot := mobsDataRoot
	mobsDataRoot = func() string { return dir }
	t.Cleanup(func() { mobsDataRoot = prevRoot })

	for _, sub := range []string{"mobs/testzone", "behaviors/archetypes"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cleanupSpecies := species.SeedSpeciesForTest(map[int]*species.Species{
		1: {SpeciesId: 1, Name: "Test Species"},
	})
	t.Cleanup(cleanupSpecies)
}

// seedMob installs a minimal template into the mobs + mobNameCache maps
// (mirroring what LoadDataFiles would have populated at boot) and registers
// cleanup. Returns a value copy for the test to mutate freely.
func seedMob(t *testing.T, id MobId, name string) Mob {
	t.Helper()

	m := Mob{MobId: id, Zone: "Testzone", StatPool: 10, ActivityLevel: 5}
	m.Character = characters.Character{Name: name, Description: "A test mob.", SpeciesId: 1}

	mobsMu.Lock()
	cp := m
	mobs[int(id)] = &cp
	mobsMu.Unlock()

	mobNameCacheMu.Lock()
	mobNameCache[id] = name
	mobNameCacheMu.Unlock()

	t.Cleanup(func() {
		mobsMu.Lock()
		delete(mobs, int(id))
		mobsMu.Unlock()
		mobNameCacheMu.Lock()
		delete(mobNameCache, id)
		mobNameCacheMu.Unlock()
	})

	return m
}

func TestSaveMobSpec_RelocatesFileOnRename(t *testing.T) {
	dir := t.TempDir()
	pointMobDataFilesAt(t, dir)
	m := seedMob(t, 99901, "Test Grunt")

	if err := SaveMobSpec(m); err != nil {
		t.Fatalf("initial save: %v", err)
	}
	oldPath := filepath.Join(dir, "mobs", "testzone", "99901-test_grunt.yaml")
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("expected %s to exist: %v", oldPath, err)
	}

	// Rename -> filename changes -> old file removed, new written.
	renamed := m
	renamed.Character.Name = "Test Bruiser"
	if err := SaveMobSpec(renamed); err != nil {
		t.Fatalf("rename save: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old file %s should be gone after rename", oldPath)
	}
	newPath := filepath.Join(dir, "mobs", "testzone", "99901-test_bruiser.yaml")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("new file %s should exist: %v", newPath, err)
	}

	// Bare reads of mobs/mobNameCache below (here and throughout this file)
	// skip mobsMu/mobNameCacheMu: safe only because this test binary has no
	// concurrent writers (no t.Parallel, no server goroutines touching these
	// maps during `go test`).
	if mobNameCache[99901] != "Test Bruiser" {
		t.Errorf("name cache not updated: %q", mobNameCache[99901])
	}
	if mobs[99901].Character.Name != "Test Bruiser" {
		t.Errorf("template cache not updated: %q", mobs[99901].Character.Name)
	}
}

// A non-canonical name must be normalized on save, or LoadDataFiles' next
// boot panics via casing.AssertCanonical. This is the failure mode the web
// builder must never be able to trigger.
func TestSaveMobSpec_CanonicalizesName(t *testing.T) {
	dir := t.TempDir()
	pointMobDataFilesAt(t, dir)
	m := seedMob(t, 99904, "lowercase grunt")

	if err := SaveMobSpec(m); err != nil {
		t.Fatalf("save: %v", err)
	}
	got := mobs[99904]
	if got.Character.Name != "Lowercase Grunt" {
		t.Errorf("name not canonicalized: %q", got.Character.Name)
	}
	if mobNameCache[99904] != "Lowercase Grunt" {
		t.Errorf("name cache not canonicalized: %q", mobNameCache[99904])
	}
}

func TestSaveMobSpec_RejectsDanglingRefs(t *testing.T) {
	dir := t.TempDir()
	pointMobDataFilesAt(t, dir)
	m := seedMob(t, 99902, "Ref Tester")

	bad := m
	bad.ScheduleId = "no_such_schedule"
	if err := SaveMobSpec(bad); err == nil {
		t.Error("expected rejection: dangling schedule_id")
	}

	bad = m
	bad.BuffIds = []int{999999}
	if err := SaveMobSpec(bad); err == nil {
		t.Error("expected rejection: dangling buff id")
	}

	bad = m
	bad.LootPool = []int{999999}
	if err := SaveMobSpec(bad); err == nil {
		t.Error("expected rejection: dangling loot item id")
	}

	if _, err := os.Stat(filepath.Join(dir, "mobs", "testzone", "99902-ref_tester.yaml")); !os.IsNotExist(err) {
		t.Error("no file should have been written for rejected saves")
	}
}

func TestDeleteMobSpec_RemovesFileAndCaches(t *testing.T) {
	dir := t.TempDir()
	pointMobDataFilesAt(t, dir)
	m := seedMob(t, 99903, "Doomed Mob")

	if err := SaveMobSpec(m); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "mobs", "testzone", "99903-doomed_mob.yaml")
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}

	if err := DeleteMobSpec(99903); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("file %s should be gone", p)
	}
	if _, ok := mobs[99903]; ok {
		t.Error("template cache entry should be gone")
	}
	if _, ok := mobNameCache[99903]; ok {
		t.Error("name cache entry should be gone")
	}
}

// DeleteMobSpec must still succeed (and clear both caches) when the on-disk
// file is already gone — e.g. a prior manual cleanup, or a delete retried
// after a partial failure. os.Remove's IsNotExist case must be swallowed.
func TestDeleteMobSpec_IdempotentWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	pointMobDataFilesAt(t, dir)
	m := seedMob(t, 99905, "Already Gone")

	if err := SaveMobSpec(m); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "mobs", "testzone", "99905-already_gone.yaml")
	if err := os.Remove(p); err != nil {
		t.Fatalf("manual pre-removal: %v", err)
	}

	if err := DeleteMobSpec(99905); err != nil {
		t.Fatalf("delete should succeed even when the file is already gone: %v", err)
	}
	if _, ok := mobs[99905]; ok {
		t.Error("template cache entry should be gone")
	}
	if _, ok := mobNameCache[99905]; ok {
		t.Error("name cache entry should be gone")
	}
}

func TestCreateNewMobFile_StubIsBootSafe(t *testing.T) {
	dir := t.TempDir()
	pointMobDataFilesAt(t, dir)

	id, err := CreateNewMobFile("Testzone")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		mobsMu.Lock()
		delete(mobs, int(id))
		mobsMu.Unlock()
		mobNameCacheMu.Lock()
		delete(mobNameCache, id)
		mobNameCacheMu.Unlock()
	})

	tmpl := GetMobSpec(id)
	if tmpl == nil {
		t.Fatal("stub not in cache")
	}
	if err := ValidateMobSpec(tmpl); err != nil {
		t.Errorf("fresh stub must be boot-safe, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mobs", "testzone", tmpl.Filename())); err != nil {
		t.Errorf("stub file missing: %v", err)
	}
}

func TestCreateNewMobFile_RejectsEmptyZone(t *testing.T) {
	dir := t.TempDir()
	pointMobDataFilesAt(t, dir)

	if _, err := CreateNewMobFile(""); err == nil {
		t.Error("expected rejection of an empty zone")
	}
}
