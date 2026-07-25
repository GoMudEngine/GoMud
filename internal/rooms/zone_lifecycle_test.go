package rooms

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestZoneFolderCollision(t *testing.T) {
	existing := []string{"Amber Valley", "Stillwater"}

	// Same folder (amber_valley), different display name -> collision.
	assert.Equal(t, "Amber Valley", ZoneFolderCollision("Amber_Valley", existing))
	assert.Equal(t, "Amber Valley", ZoneFolderCollision("amber valley", existing))

	// Genuinely new zone -> no collision.
	assert.Equal(t, "", ZoneFolderCollision("Thornwall", existing))

	// A name identical to an existing zone still reports it; CreateZone's
	// own duplicate check runs first, but this must not silently pass.
	assert.Equal(t, "Stillwater", ZoneFolderCollision("Stillwater", existing))
}

func TestZoneDeletionBlockers_ReportsEachKind(t *testing.T) {
	src := zoneBlockerSources{
		roomIdsInZone:  func(z string) []int { return []int{100, 101, 102} },
		zoneRootRoomId: func(z string) int { return 100 },
		contentFiles:   func(z string) []string { return []string{"mobs/testzone/5-guard.yaml"} },
		inboundExits: func(z string) []string {
			return []string{"room 900 (Other Zone) east"}
		},
		playersInZone: func(z string) []string { return []string{"Meirok"} },
	}

	got := ZoneDeletionBlockersWith("Testzone", src)

	kinds := map[string]int{}
	for _, b := range got {
		kinds[b.Kind]++
	}
	// 101 and 102 are non-root rooms; 100 is the root and is NOT a blocker.
	assert.Equal(t, 2, kinds["room"], "root room must not be reported")
	assert.Equal(t, 1, kinds["content"])
	assert.Equal(t, 1, kinds["inbound-exit"])
	assert.Equal(t, 1, kinds["player"])
}

func TestZoneDeletionBlockers_CleanZoneIsEmpty(t *testing.T) {
	src := zoneBlockerSources{
		roomIdsInZone:  func(z string) []int { return []int{100} }, // root only
		zoneRootRoomId: func(z string) int { return 100 },
		contentFiles:   func(z string) []string { return nil },
		inboundExits:   func(z string) []string { return nil },
		playersInZone:  func(z string) []string { return nil },
	}
	assert.Empty(t, ZoneDeletionBlockersWith("Testzone", src))
}

func TestZoneContentDirs_CoversAllAuthoredTrees(t *testing.T) {
	dirs := zoneContentDirs()
	assert.ElementsMatch(t,
		[]string{"mobs", "dialogue", "behaviors", "schedules", "caravans", "foragers"},
		dirs,
		"authored content trees scanned for delete blockers")
}

func TestZoneAllDirs_CoversAllTenTrees(t *testing.T) {
	assert.Len(t, zoneAllDirs(), 10, "a zone owns ten directories")
}

func TestDeleteZone_RemovesEveryTree(t *testing.T) {
	if os.Getenv(`DOGMUD_BOOT_SMOKE`) == `` {
		t.Skip("set DOGMUD_BOOT_SMOKE=1 to run the filesystem zone test")
	}

	// A test binary's CWD is its own package directory, and ReloadConfig reads
	// _datafiles/config.yaml relative to CWD. Without this chdir the lazily
	// validated zero-value config falls back to the built-in default data root
	// (world/default) and CreateZone tries to mkdir under a path that does not
	// exist. Same workaround as internal/web/auth_test.go.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir(%q): %v", repoRoot, err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}

	const zone = "Ziggurat Test Zone"

	// Leave nothing behind if an assertion fails mid-test.
	t.Cleanup(func() {
		base := configs.GetFilePathsConfig().DataFiles.String()
		for _, d := range zoneAllDirs() {
			_ = os.RemoveAll(util.FilePath(base, "/", d, "/", ZoneNameSanitize(zone)))
		}
	})

	roomId, err := CreateZone(zone)
	assert.NoError(t, err)
	assert.NotZero(t, roomId)

	folder := ZoneNameSanitize(zone)
	base := configs.GetFilePathsConfig().DataFiles.String()
	roomsDir := util.FilePath(base, "/", "rooms", "/", folder)
	_, statErr := os.Stat(roomsDir)
	assert.NoError(t, statErr, "zone folder should exist after CreateZone")

	// Root room only, no content -> deletable.
	assert.Empty(t, ZoneDeletionBlockers(zone))
	assert.NoError(t, DeleteZone(zone))

	for _, d := range zoneAllDirs() {
		dir := util.FilePath(base, "/", d, "/", folder)
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("%s still exists after DeleteZone", dir)
		}
	}
	assert.NotContains(t, GetAllZoneNames(), zone)
}
