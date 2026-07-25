package rooms

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestZoneRenameBlockers_ReportsPlayersOnly(t *testing.T) {
	// Rooms, mobs and content do NOT block a rename — only players, because
	// their in-memory room pointers and the file move would race.
	src := zoneRenameSources{
		playersInZone: func(z string) []string { return []string{"2 player(s) in room 101"} },
	}
	got := ZoneRenameBlockersWith("Testzone", src)
	assert.Len(t, got, 1)
	assert.Equal(t, "player", got[0].Kind)
}

func TestZoneRenameBlockers_QuietZoneIsRenameable(t *testing.T) {
	src := zoneRenameSources{
		playersInZone: func(z string) []string { return nil },
	}
	assert.Empty(t, ZoneRenameBlockersWith("Testzone", src))
}

func TestValidateZoneRename(t *testing.T) {
	existing := []string{"Amber Valley", "Stillwater"}

	// Happy path.
	assert.NoError(t, ValidateZoneRename("Stillwater", "Quiet Water", existing))

	// Empty / too short. ValidateZoneName returns nil for "", so the emptiness
	// check must be our own.
	assert.Error(t, ValidateZoneRename("Stillwater", "", existing))
	assert.Error(t, ValidateZoneRename("Stillwater", "Q", existing))

	// Illegal characters (ValidateZoneName allows letters/digits/space/_ only).
	assert.Error(t, ValidateZoneRename("Stillwater", "Bad/Name", existing))

	// Renaming to an existing zone.
	assert.Error(t, ValidateZoneRename("Stillwater", "Amber Valley", existing))

	// Different display name that sanitizes onto a LIVE zone's folder.
	assert.Error(t, ValidateZoneRename("Stillwater", "Amber_Valley", existing))

	// Renaming a zone to a different capitalisation of ITSELF is allowed —
	// it collides only with its own folder, which is the one being moved.
	assert.NoError(t, ValidateZoneRename("Stillwater", "StillWater", existing))
}

func TestRewriteZoneField_TouchesOnlyTheTopLevelKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "room.yaml")

	// CRLF, a description mentioning "zone:" inside a block scalar, and a
	// nested key that also ends in "zone:" — none of which may be rewritten.
	original := "roomid: 5\r\n" +
		"zone: Old Name\r\n" +
		"title: A Room\r\n" +
		"description: >-\r\n" +
		"  A sign reads: zone: Old Name. The prose must survive verbatim.\r\n" +
		"nouns:\r\n" +
		"  subzone: something\r\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	if err := rewriteZoneField(path, "New Name"); err != nil {
		t.Fatalf("rewriteZoneField: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)

	assert.Contains(t, s, "zone: New Name\r\n", "top-level zone must be rewritten")
	assert.Contains(t, s, "A sign reads: zone: Old Name.", "prose must be untouched")
	assert.Contains(t, s, "  subzone: something", "nested keys must be untouched")
	assert.Contains(t, s, "\r\n", "CRLF line endings must be preserved")
	assert.NotContains(t, s, "zone: Old Name\r\n", "old top-level value must be gone")
}

func TestRewriteZoneField_NoTopLevelZoneIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schedule.yaml")
	original := "id: arn\r\nsegments: []\r\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	// behaviors/schedules/shops carry no zone: key. Rewriting must be a no-op,
	// not a failure.
	assert.NoError(t, rewriteZoneField(path, "New Name"))
	got, _ := os.ReadFile(path)
	assert.Equal(t, original, string(got))
}

func TestPlanZoneRename_SkipsAbsentTreesAndDetectsCollision(t *testing.T) {
	base := t.TempDir()
	// Only three of the ten trees exist for this zone.
	for _, d := range []string{"rooms", "mobs", "shops"} {
		if err := os.MkdirAll(filepath.Join(base, d, "old_zone"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	mv, err := planZoneRename(base, "Old Zone", "New Zone")
	assert.NoError(t, err)
	assert.Len(t, mv, 3, "only existing trees are planned")

	// A pre-existing target directory must abort the whole plan.
	if err := os.MkdirAll(filepath.Join(base, "rooms", "new_zone"), 0755); err != nil {
		t.Fatal(err)
	}
	_, err = planZoneRename(base, "Old Zone", "New Zone")
	assert.Error(t, err, "an occupied target path must abort before anything moves")
}

func TestRenameZone_MovesRewritesAndRekeys(t *testing.T) {
	if os.Getenv(`DOGMUD_BOOT_SMOKE`) == `` {
		t.Skip("set DOGMUD_BOOT_SMOKE=1 to run the filesystem rename test")
	}
	// Test binaries run with CWD = their own package dir; ReloadConfig reads
	// _datafiles/config.yaml relative to CWD. Same dance as
	// internal/web/auth_test.go and TestDeleteZone_RemovesEveryTree.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := configs.ReloadConfig(); err != nil {
		t.Fatal(err)
	}

	const oldName = "Rename Probe Zone"
	const newName = "Renamed Probe Zone"

	// Scrub BOTH names before and after. Not belt-and-braces: a leftover
	// TARGET directory makes planZoneRename correctly abort the rename, and
	// this test would then fail while reporting nothing about the real cause.
	// (Verified by planting a stale target dir — the guard fires exactly as
	// designed and every downstream assertion collapses.)
	scrub := func() {
		base := configs.GetFilePathsConfig().DataFiles.String()
		for _, d := range zoneAllDirs() {
			_ = os.RemoveAll(util.FilePath(base, "/", d, "/", ZoneNameSanitize(oldName)))
			_ = os.RemoveAll(util.FilePath(base, "/", d, "/", ZoneNameSanitize(newName)))
		}
		delete(roomManager.zones, oldName)
		delete(roomManager.zones, newName)
	}
	scrub()
	t.Cleanup(scrub)

	roomId, err := CreateZone(oldName)
	assert.NoError(t, err)

	assert.NoError(t, RenameZone(oldName, newName))

	base := configs.GetFilePathsConfig().DataFiles.String()
	oldDir := util.FilePath(base, "/", "rooms", "/", ZoneNameSanitize(oldName))
	newDir := util.FilePath(base, "/", "rooms", "/", ZoneNameSanitize(newName))
	_, oldErr := os.Stat(oldDir)
	assert.True(t, os.IsNotExist(oldErr), "old rooms dir must be gone")
	_, newErr := os.Stat(newDir)
	assert.NoError(t, newErr, "new rooms dir must exist")

	// Content rewritten.
	r := LoadRoomTemplate(roomId)
	assert.NotNil(t, r)
	if r != nil {
		assert.Equal(t, newName, r.Zone, "room zone: must be rewritten")
	}

	// Caches rekeyed.
	assert.NotContains(t, GetAllZoneNames(), oldName)
	assert.Contains(t, GetAllZoneNames(), newName)
	if cfg := GetZoneConfig(newName); assert.NotNil(t, cfg) {
		assert.Equal(t, newName, cfg.Name)
	}
}
