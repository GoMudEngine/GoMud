package factions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepPath(t *testing.T) {
	got := repPath("warren")
	if !strings.HasSuffix(filepath.ToSlash(got), "factions.rep/warren.yaml") {
		t.Errorf("repPath = %q, want suffix factions.rep/warren.yaml", got)
	}
}

func TestRepSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", dir)
	clearRepCacheForTest()

	fr := &FactionRep{
		FactionId: "warren",
		Players: map[int]*RepEntry{
			17: {Rep: 30, LastUpdatedRound: 1843201},
			92: {Rep: -75, LastUpdatedRound: 1846020},
		},
	}
	repCacheStoreForTest(fr)
	if err := saveRepToDisk("warren"); err != nil {
		t.Fatalf("saveRepToDisk: %v", err)
	}

	clearRepCacheForTest()
	got := loadRepFromDisk("warren")
	if got == nil {
		t.Fatal("loadRepFromDisk returned nil")
	}
	if got.FactionId != "warren" {
		t.Errorf("FactionId = %q", got.FactionId)
	}
	if got.Players[17].Rep != 30 || got.Players[17].LastUpdatedRound != 1843201 {
		t.Errorf("user 17 mismatch: %+v", got.Players[17])
	}
	if got.Players[92].Rep != -75 {
		t.Errorf("user 92 mismatch: %+v", got.Players[92])
	}

	expected := filepath.Join(dir, "warren.yaml")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected file missing: %v", err)
	}
}

func TestLoadRepFromDiskMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", dir)
	clearRepCacheForTest()
	if got := loadRepFromDisk("ghost"); got != nil {
		t.Errorf("loadRepFromDisk on missing file = %+v, want nil", got)
	}
}

func TestLoadRepFromDiskCorruptYAMLReturnsNil(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", dir)
	clearRepCacheForTest()

	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("\t\nplayers:\n  bad: {structure"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := loadRepFromDisk("bad"); got != nil {
		t.Errorf("loadRepFromDisk on corrupt YAML = %+v, want nil", got)
	}
}
