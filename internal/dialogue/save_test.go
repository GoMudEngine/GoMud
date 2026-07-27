package dialogue

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	yaml "gopkg.in/yaml.v2"
)

// overrideDataFilesDir points configs.DataFiles at a temp dir for the
// duration of a test, using the codebase's supported mechanism (precedent:
// internal/web/auth_test.go): chdir to repo root (ReloadConfig reads
// _datafiles/config.yaml from CWD), write a config-overrides.yaml naming the
// temp dir, set CONFIG_PATH, reload. Cleanup restores CWD and reloads the
// real config.
func overrideDataFilesDir(t *testing.T) string {
	t.Helper()
	// ReloadConfig logs; package tests have no logger configured.
	mudlog.SetupLogger(nil, `LOW`, ``, false)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir(%q): %v", repoRoot, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

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

// The writer owns the loader's contract: path from the loader's own
// sanitizer, cache REPLACED on save, sentinel CLEARED on create and SET on
// delete. The sentinel rule is the one that burns people: Load caches
// "no dialogue forever" per (mob,zone), so a create that forgets to clear it
// ships a file the running server can never see.
func TestWriter_CacheAndSentinelContract(t *testing.T) {
	dir := overrideDataFilesDir(t)

	const mobId, zone = 424242, "Writer Probe Zone"
	key := fmt.Sprintf("%d:%s", mobId, zone)
	delete(dialogueCache, key)
	delete(nilSentinel, key)
	t.Cleanup(func() { delete(dialogueCache, key); delete(nilSentinel, key) })

	// Simulate the burn: Load before the file exists → sentinel set.
	if Load(mobId, zone) != nil {
		t.Fatal("no file yet — Load must return nil")
	}
	if !nilSentinel[key] {
		t.Fatal("Load must set the nil sentinel (loader contract)")
	}

	if err := CreateNewDialogueFile(mobId, zone); err != nil {
		t.Fatalf("create: %v", err)
	}
	if nilSentinel[key] {
		t.Error("create must CLEAR the nil sentinel or the new file is invisible until reboot")
	}
	if df := Load(mobId, zone); df == nil {
		t.Fatal("Load must now see the created file")
	}

	// Save replaces the cache entry in place.
	df := *Load(mobId, zone)
	df.DefaultMood = "hostile"
	if err := SaveDialogueFile(df); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got := Load(mobId, zone); got.DefaultMood != "hostile" {
		t.Errorf("live cache must serve the edit immediately, got mood %q", got.DefaultMood)
	}

	// The file landed at the loader's exact path.
	p := filepath.Join(dir, "dialogue", zoneNameSanitize(zone), fmt.Sprintf("%d.yaml", mobId))
	if _, err := os.Stat(p); err != nil {
		t.Errorf("file not at the loader's path: %v", err)
	}

	// Delete removes the file, drops the cache, and SETS the sentinel.
	if err := DeleteDialogueFile(mobId, zone); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := dialogueCache[key]; ok {
		t.Error("delete must drop the cache entry")
	}
	if !nilSentinel[key] {
		t.Error("delete must set the sentinel — the mob genuinely has no dialogue now")
	}
	if Load(mobId, zone) != nil {
		t.Error("Load after delete must return nil")
	}
}

func TestWriter_CreateRefusesWhenFileExists(t *testing.T) {
	overrideDataFilesDir(t)
	const mobId, zone = 424243, "Writer Probe Zone"
	key := fmt.Sprintf("%d:%s", mobId, zone)
	t.Cleanup(func() { delete(dialogueCache, key); delete(nilSentinel, key) })

	if err := CreateNewDialogueFile(mobId, zone); err != nil {
		t.Fatal(err)
	}
	if err := CreateNewDialogueFile(mobId, zone); err == nil {
		t.Error("second create must refuse — the file exists")
	}
}

// The writer must be lossless over the entire live corpus before it ever
// touches a real file. Marshal-level round trip: bytes -> struct -> marshal
// -> struct -> DeepEqual. Any failure here is a struct-shape gap (the
// greetings incident class), not a test to weaken.
func TestWriter_RoundTripsEveryLiveFile(t *testing.T) {
	root := filepath.Join("..", "..", "_datafiles", "world", "dogmud", "dialogue")
	checked := 0
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		raw, _ := os.ReadFile(path)
		var orig DialogueFile
		if yaml.Unmarshal(raw, &orig) != nil {
			return nil // the parse sweep owns malformed files
		}
		out, err := yaml.Marshal(&orig)
		if err != nil {
			t.Errorf("%s: marshal: %v", path, err)
			return nil
		}
		var back DialogueFile
		if err := yaml.Unmarshal(out, &back); err != nil {
			t.Errorf("%s: re-unmarshal: %v", path, err)
			return nil
		}
		// DeepEqual is the WRONG criterion here: a root-only tree unmarshals
		// with Nodes == nil, marshals as `nodes: []`, and reparses as an
		// EMPTY slice — semantically identical, DeepEqual-different. The
		// meaningful invariant for a canonicalizing writer is that its
		// canonical form is a FIXED POINT: re-reading and re-marshalling its
		// own output is byte-stable (no churn on every open+save). Gross
		// content loss is guarded separately by the section counts.
		out2, err := yaml.Marshal(&back)
		if err != nil {
			t.Errorf("%s: re-marshal: %v", path, err)
			return nil
		}
		if string(out) != string(out2) {
			t.Errorf("%s: marshal is not a fixed point — writer output would churn on every save", path)
		}
		if len(back.Patterns) != len(orig.Patterns) || len(back.Greetings) != len(orig.Greetings) {
			t.Errorf("%s: section counts changed through round trip", path)
		}
		if (orig.Tree == nil) != (back.Tree == nil) {
			t.Errorf("%s: tree presence changed through round trip", path)
		} else if orig.Tree != nil {
			if len(back.Tree.Nodes) != len(orig.Tree.Nodes) || len(back.Tree.Root.Variants) != len(orig.Tree.Root.Variants) {
				t.Errorf("%s: tree node/variant counts changed through round trip", path)
			}
			if !reflect.DeepEqual(orig.Tree.Root.Text, back.Tree.Root.Text) {
				t.Errorf("%s: root text changed through round trip", path)
			}
		}
		checked++
		return nil
	})
	if checked < 300 {
		t.Errorf("round-tripped only %d files — wrong path?", checked)
	}
	t.Logf("round-tripped %d live dialogue files losslessly", checked)
}
