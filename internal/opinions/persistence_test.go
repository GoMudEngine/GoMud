package opinions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpinionPath(t *testing.T) {
	got := opinionPath(41, "lars")
	if !strings.HasSuffix(filepath.ToSlash(got), "opinions/41-lars.yaml") {
		t.Errorf("opinionPath = %q, want suffix opinions/41-lars.yaml", got)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir) // honored by opinionPath in tests

	ClearCache()

	mo := &MobOpinions{
		MobId:              41,
		DefaultDisposition: 5,
		Opinions: map[int]*Opinion{
			17: {Score: -42, LastUpdatedRound: 1843201},
			92: {Score: 28, LastUpdatedRound: 1846020},
		},
	}
	cacheStoreForTest("lars", mo)
	if err := saveToDisk(41, "lars"); err != nil {
		t.Fatalf("saveToDisk: %v", err)
	}

	ClearCache()
	got := loadFromDisk(41, "lars")
	if got == nil {
		t.Fatal("loadFromDisk returned nil for round-tripped file")
	}
	if got.MobId != 41 || got.DefaultDisposition != 5 {
		t.Errorf("round-trip header mismatch: %+v", got)
	}
	if got.Opinions[17].Score != -42 || got.Opinions[17].LastUpdatedRound != 1843201 {
		t.Errorf("round-trip user 17 mismatch: %+v", got.Opinions[17])
	}
	if got.Opinions[92].Score != 28 {
		t.Errorf("round-trip user 92 mismatch: %+v", got.Opinions[92])
	}

	// Sanity: file exists where we expected.
	expected := filepath.Join(dir, "41-lars.yaml")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected file missing: %v", err)
	}
}

func TestLoadFromDiskMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()

	if got := loadFromDisk(999, "ghost"); got != nil {
		t.Errorf("loadFromDisk on missing file = %+v, want nil", got)
	}
}

func TestLoadFromDiskCorruptYAMLReturnsNil(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()

	bad := filepath.Join(dir, "1-bad.yaml")
	if err := os.WriteFile(bad, []byte("\t\nopinions:\n  not_a_number: {bad: structure"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := loadFromDisk(1, "bad"); got != nil {
		t.Errorf("loadFromDisk on corrupt YAML = %+v, want nil", got)
	}
}

func TestRestartPreservesOpinionWithDecay(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()

	defaultProviderForTest = func(mobId int) (string, int, bool) { return "lars", 0, true }
	t.Cleanup(func() { defaultProviderForTest = nil })

	halfLifeForTest = func() uint64 { return 100 }
	t.Cleanup(func() { halfLifeForTest = nil })

	roundForTest = func() uint64 { return 1000 }
	Bump(41, 17, -50)

	// Simulate restart: drop cache, leave file on disk.
	ClearCache()

	// "Restart" — round count advances by five half-lives.
	roundForTest = func() uint64 { return 1500 }
	t.Cleanup(func() { roundForTest = nil })

	got := Get(41, 17)
	// -50 decayed by 5 toward 0 → -45.
	if got != -45 {
		t.Errorf("post-restart Get = %d, want -45 (anchored read with decay)", got)
	}
}
