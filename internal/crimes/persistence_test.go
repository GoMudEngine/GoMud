package crimes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrimePath(t *testing.T) {
	got := crimePath("thornwall_citizens")
	if !strings.HasSuffix(filepath.ToSlash(got), "factions.crimes/thornwall_citizens.yaml") {
		t.Errorf("crimePath = %q, want suffix factions.crimes/thornwall_citizens.yaml", got)
	}
}

func TestCrimesSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE", dir)
	clearCrimeCacheForTest()

	fc := &FactionCrimes{
		FactionId: "thornwall_citizens",
		Crimes: []*Crime{
			{Id: 1, Kind: KindMurder, Zone: "Thornwall City", RoomId: 467, Round: 1843201,
				VictimMobId: 100, VictimInstanceId: 250,
				Perpetrator: Perpetrator{Type: PerpPlayer, Id: 17}},
			{Id: 2, Kind: KindTheft, Zone: "Thornwall City", RoomId: 462, Round: 1845102,
				VictimMobId: 102, VictimInstanceId: 312,
				Perpetrator: Perpetrator{Type: PerpUnknown}},
		},
		nextId: 3,
	}
	crimeCacheStoreForTest(fc)
	if err := saveCrimesToDisk("thornwall_citizens"); err != nil {
		t.Fatalf("saveCrimesToDisk: %v", err)
	}

	clearCrimeCacheForTest()
	got := loadCrimesFromDisk("thornwall_citizens")
	if got == nil {
		t.Fatal("loadCrimesFromDisk returned nil")
	}
	if got.FactionId != "thornwall_citizens" {
		t.Errorf("FactionId = %q", got.FactionId)
	}
	if len(got.Crimes) != 2 {
		t.Fatalf("got %d crimes, want 2", len(got.Crimes))
	}
	if got.Crimes[0].Kind != KindMurder || got.Crimes[0].Perpetrator.Id != 17 {
		t.Errorf("crime 0 mismatch: %+v", got.Crimes[0])
	}
	if got.Crimes[1].Perpetrator.Type != PerpUnknown {
		t.Errorf("crime 1 perpetrator: %+v", got.Crimes[1].Perpetrator)
	}
	// nextId is recomputed on load — should be max(ids) + 1 = 3.
	if got.nextId != 3 {
		t.Errorf("nextId after load = %d, want 3", got.nextId)
	}
}

func TestLoadCrimesFromDiskMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE", dir)
	clearCrimeCacheForTest()
	if got := loadCrimesFromDisk("ghost"); got != nil {
		t.Errorf("loadCrimesFromDisk on missing file = %+v, want nil", got)
	}
}

func TestLoadCrimesFromDiskCorruptYAMLReturnsNil(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE", dir)
	clearCrimeCacheForTest()

	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("\t\ncrimes:\n  bad: {structure"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := loadCrimesFromDisk("bad"); got != nil {
		t.Errorf("loadCrimesFromDisk on corrupt YAML = %+v, want nil", got)
	}
}
