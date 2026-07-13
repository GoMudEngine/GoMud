package migration

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"
)

// repoRoot walks up from the test's cwd to the directory holding _archive.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "_archive")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("_archive/prod-users not present — skipping prod-copy migration test")
		}
		dir = parent
	}
}

// migratedMutations re-reads a user file's character.mutations after migration.
func migratedMutations(t *testing.T, path string) (map[string]int, bool) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	ch, _ := m["character"].(map[interface{}]interface{})
	migrated, _ := ch["mutation_migrated"].(bool)
	return yamlIntMap(ch["mutations"]), migrated
}

// TestReclassifyMigration_EndToEnd copies representative prod accounts (+ a
// freshie) into a disposable temp dir, runs the migration, and verifies each
// lands per spec §4 with retired mutations gone and the marker set. t.TempDir is
// auto-removed — nothing touches the real users dir or the prod archive.
func TestReclassifyMigration_EndToEnd(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "_archive", "prod-users", "users")
	tmp := t.TempDir()

	// Representative accounts: heavy veterans across classes + an early/light player.
	reps := map[string]string{ // prod-file -> expected cluster
		"24.yaml": "manifester", // meirok (2 companions)
		"17.yaml": "chrysifier", // Duard (crafter)
		"14.yaml": "ethereal",   // Deios (caster)
		"30.yaml": "ravener",    // Saphira (unarmed 27)
		"4.yaml":  "generalist", // Calabe (light)
		"1.yaml":  "admin",      // Megalomania (role admin)
	}
	for f := range reps {
		data, err := os.ReadFile(filepath.Join(src, f))
		if err != nil {
			t.Skipf("prod file %s missing: %v", f, err)
		}
		if err := os.WriteFile(filepath.Join(tmp, "mtest_"+f), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	// A freshie: minimal new character (no skills/companions) -> generalist.
	freshie := "userid: 999\nrole: user\ncharacter:\n  name: Freshie\n  skills: {}\n  mutations: {}\n"
	if err := os.WriteFile(filepath.Join(tmp, "mtest_freshie.yaml"), []byte(freshie), 0644); err != nil {
		t.Fatal(err)
	}

	if err := reclassifyUsersInDir(tmp, false); err != nil {
		t.Fatalf("migration: %v", err)
	}

	// Freshie -> generalist seed.
	if muts, migrated := migratedMutations(t, filepath.Join(tmp, "mtest_freshie.yaml")); !migrated || muts["keen-senses"] != 1 {
		t.Errorf("freshie: migrated=%v muts=%v, want generalist keen-senses", migrated, muts)
	}

	// Each rep: marker set, retired gone, expected seed present.
	for f, cluster := range reps {
		muts, migrated := migratedMutations(t, filepath.Join(tmp, "mtest_"+f))
		if !migrated {
			t.Errorf("%s: mutation_migrated not set", f)
		}
		for id := range muts {
			if retiredMutations[id] {
				t.Errorf("%s: retired mutation %q survived migration", f, id)
			}
		}
		for id, rank := range SeedForCluster(cluster) {
			if muts[id] != rank {
				t.Errorf("%s (%s): missing seed %q=%d (got %v)", f, cluster, id, rank, muts)
			}
		}
	}

	// Idempotency: re-run must not change an already-migrated file.
	before, _ := migratedMutations(t, filepath.Join(tmp, "mtest_24.yaml"))
	if err := reclassifyUsersInDir(tmp, false); err != nil {
		t.Fatalf("re-run: %v", err)
	}
	after, _ := migratedMutations(t, filepath.Join(tmp, "mtest_24.yaml"))
	if len(before) != len(after) {
		t.Errorf("idempotency broken: %v -> %v", before, after)
	}
}
