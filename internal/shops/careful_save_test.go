package shops

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tornWriteSentinel stands in for the garbage a crashed/interrupted write
// leaves behind in the scratch file.
const tornWriteSentinel = "TORN PARTIAL WRITE - MUST NOT SURVIVE\n"

// useTempShopDataFiles points FilePaths.DataFiles at a temp tree and sets
// FilePaths.CarefulSaveFiles, restoring both when the test ends.
func useTempShopDataFiles(t *testing.T, careful bool) {
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
}

// TestSaveShop_HonoursCarefulSaveFiles pins that shop saves respect
// FilePaths.CarefulSaveFiles.
//
// SaveShop called os.WriteFile directly while items, mobs, users and alts all
// wrote <path>.new then renamed. Shop files are the living economy — stock
// levels, NPC gold, restock timers — and unlike room instance files they are
// NOT regenerable from a template once a merchant has traded.
//
// The observable difference between the two paths is the scratch file: the
// careful path writes <path>.new and renames it onto the primary, so a
// pre-planted scratch file is consumed; the direct path never touches it. That
// verifies both legs of the flag without simulating a crash, and proves the
// primary is only ever replaced by an atomic rename of a fully-written file.
func TestSaveShop_HonoursCarefulSaveFiles(t *testing.T) {
	const (
		zone   = "carefulzone"
		mobId  = 9101
		roomId = 7
	)

	useTempShopDataFiles(t, true)
	ClearCache()
	t.Cleanup(ClearCache)

	RegisterShop(zone, mobId, roomId, makeTemplate())

	savePath := shopPath(zone, mobId, roomId)
	scratchPath := savePath + ".new"

	// The directory is created by SaveShop's MkdirAll, so do a first save to
	// establish it, then plant the torn scratch file for the real assertion.
	require.NoError(t, SaveShop(zone, mobId, roomId))
	require.NoError(t, os.WriteFile(scratchPath, []byte(tornWriteSentinel), 0644))

	require.NoError(t, SaveShop(zone, mobId, roomId))

	_, err := os.Stat(scratchPath)
	assert.True(t, os.IsNotExist(err),
		"careful save must consume <path>.new via rename, leaving no scratch file")

	written, err := os.ReadFile(savePath)
	require.NoError(t, err)
	assert.NotContains(t, string(written), "TORN PARTIAL WRITE",
		"the primary must hold the real save, never the torn scratch content")
	assert.Contains(t, string(written), "gold",
		"the primary must hold the fully-written shop inventory")
}

// TestSaveShop_CarefulSaveFilesOffWritesDirectly is the control leg: it pins
// that the flag is actually READ rather than hardcoded on.
func TestSaveShop_CarefulSaveFilesOffWritesDirectly(t *testing.T) {
	const (
		zone   = "carefulzoneoff"
		mobId  = 9102
		roomId = 7
	)

	useTempShopDataFiles(t, false)
	ClearCache()
	t.Cleanup(ClearCache)

	RegisterShop(zone, mobId, roomId, makeTemplate())

	savePath := shopPath(zone, mobId, roomId)
	scratchPath := savePath + ".new"

	require.NoError(t, SaveShop(zone, mobId, roomId))
	require.NoError(t, os.WriteFile(scratchPath, []byte(tornWriteSentinel), 0644))

	require.NoError(t, SaveShop(zone, mobId, roomId))

	leftover, err := os.ReadFile(scratchPath)
	require.NoError(t, err, "with the flag off the scratch file should be untouched")
	assert.Equal(t, tornWriteSentinel, string(leftover),
		"direct write must not go anywhere near <path>.new")
}
