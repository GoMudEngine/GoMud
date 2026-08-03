package rooms

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// tornWriteSentinel stands in for the garbage a crashed/interrupted write
// leaves behind in the scratch file.
const tornWriteSentinel = "TORN PARTIAL WRITE - MUST NOT SURVIVE\n"

// useTempDataFiles points FilePaths.DataFiles at a temp tree and sets
// FilePaths.CarefulSaveFiles, restoring both when the test ends. Returns the
// temp root.
func useTempDataFiles(t *testing.T, careful bool) string {
	t.Helper()

	tempDir := t.TempDir()
	prev := configs.GetFilePathsConfig()

	require.NoError(t, configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles":        tempDir,
		"FilePaths.CarefulSaveFiles": careful,
	}))
	t.Cleanup(func() {
		_ = configs.AddOverlayOverrides(map[string]any{
			"FilePaths.DataFiles":        prev.DataFiles.String(),
			"FilePaths.CarefulSaveFiles": bool(prev.CarefulSaveFiles),
		})
	})

	require.Equal(t, careful, bool(configs.GetFilePathsConfig().CarefulSaveFiles),
		"test setup: CarefulSaveFiles override did not take effect")

	return tempDir
}

// seedTemplateRoom writes a room template into the temp DataFiles tree and
// registers its path so LoadRoomTemplate can find it. Returns the instance
// save path SaveRoomInstance will target.
func seedTemplateRoom(t *testing.T, tempDir string, roomId int) string {
	t.Helper()

	tpl := &Room{
		RoomId:      roomId,
		Zone:        "test_zone",
		Title:       "Careful Save Room",
		Description: "A room used to exercise the careful-save path.",
	}
	data, err := yaml.Marshal(tpl)
	require.NoError(t, err)

	templatePath := filepath.Join(tempDir, "rooms", "test_zone", "100.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(templatePath), 0755))
	require.NoError(t, os.WriteFile(templatePath, data, 0644))

	instanceDir := filepath.Join(tempDir, "rooms.instances", "test_zone")
	require.NoError(t, os.MkdirAll(instanceDir, 0755))

	roomManager.roomIdToFileCache[roomId] = "test_zone/100.yaml"
	t.Cleanup(func() { delete(roomManager.roomIdToFileCache, roomId) })

	return filepath.Join(instanceDir, "100.yaml")
}

// TestSaveRoomInstance_HonoursCarefulSaveFiles pins that room instance saves
// respect FilePaths.CarefulSaveFiles.
//
// Rooms (and shops) called os.WriteFile directly while items, mobs, users and
// alts all wrote <path>.new then renamed. Prod sets CarefulSaveFiles: true, so
// the operator had opted into torn-file protection that two of five subsystems
// silently ignored — and a truncated room YAML panics the server at startup.
//
// The observable difference between the two paths is the scratch file. A
// pre-existing <path>.new is CONSUMED by the careful path (written over, then
// renamed onto the primary) and left completely untouched by the direct path.
// That makes both legs of the flag verifiable without simulating a crash, and
// proves the primary is only ever replaced by an atomic rename of a
// fully-written file.
func TestSaveRoomInstance_HonoursCarefulSaveFiles(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	tempDir := useTempDataFiles(t, true)
	instancePath := seedTemplateRoom(t, tempDir, 100)
	scratchPath := instancePath + ".new"

	// Plant a torn scratch file, as an interrupted earlier save would leave.
	require.NoError(t, os.WriteFile(scratchPath, []byte(tornWriteSentinel), 0644))

	// Gold is NOT instance:"skip", so it differs from the template and forces
	// a non-empty instance save.
	live := Room{RoomId: 100, Zone: "test_zone", Gold: 42}
	require.NoError(t, SaveRoomInstance(live))

	_, err := os.Stat(scratchPath)
	assert.True(t, os.IsNotExist(err),
		"careful save must consume <path>.new via rename, leaving no scratch file")

	written, err := os.ReadFile(instancePath)
	require.NoError(t, err)
	assert.NotContains(t, string(written), "TORN PARTIAL WRITE",
		"the primary must hold the real save, never the torn scratch content")
	assert.Contains(t, string(written), "gold",
		"the primary must hold the fully-written instance data")
}

// TestSaveRoomInstance_CarefulSaveFilesOffWritesDirectly is the control leg: it
// pins that the flag is actually READ rather than hardcoded on. With the flag
// off the scratch file is never touched.
func TestSaveRoomInstance_CarefulSaveFilesOffWritesDirectly(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	tempDir := useTempDataFiles(t, false)
	instancePath := seedTemplateRoom(t, tempDir, 100)
	scratchPath := instancePath + ".new"

	require.NoError(t, os.WriteFile(scratchPath, []byte(tornWriteSentinel), 0644))

	live := Room{RoomId: 100, Zone: "test_zone", Gold: 42}
	require.NoError(t, SaveRoomInstance(live))

	leftover, err := os.ReadFile(scratchPath)
	require.NoError(t, err, "with the flag off the scratch file should be untouched")
	assert.Equal(t, tornWriteSentinel, string(leftover),
		"direct write must not go anywhere near <path>.new")
}

// TestSaveRoomTemplate_HonoursCarefulSaveFiles covers the second rooms write
// site. Templates are the authored content files — the ones whose truncation
// panics the server at boot.
func TestSaveRoomTemplate_HonoursCarefulSaveFiles(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	tempDir := useTempDataFiles(t, true)

	// SaveRoomTemplate reads GetZoneConfig(zone) to queue the map rebuild, so
	// the zone must be registered.
	roomManager.zones["test_zone"] = &ZoneConfig{
		Name:    "test_zone",
		RoomId:  777,
		RoomIds: map[int]struct{}{},
	}

	zoneFolder := ZoneToFolder("test_zone")
	roomsDir := filepath.Join(tempDir, "rooms", zoneFolder)
	require.NoError(t, os.MkdirAll(roomsDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "rooms.instances", zoneFolder), 0755))

	templatePath := filepath.Join(roomsDir, "777.yaml")
	scratchPath := templatePath + ".new"
	require.NoError(t, os.WriteFile(scratchPath, []byte(tornWriteSentinel), 0644))

	tpl := Room{
		RoomId:      777,
		Zone:        "test_zone",
		Title:       "Careful Template Room",
		Description: "Written through the careful-save path.",
	}
	require.NoError(t, SaveRoomTemplate(tpl))

	_, err := os.Stat(scratchPath)
	assert.True(t, os.IsNotExist(err),
		"careful save must consume <path>.new via rename, leaving no scratch file")

	written, err := os.ReadFile(templatePath)
	require.NoError(t, err)
	assert.NotContains(t, string(written), "TORN PARTIAL WRITE")
	assert.Contains(t, string(written), "Careful Template Room")
}
