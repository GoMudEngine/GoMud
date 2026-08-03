package rooms

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const defuseTrapBuffId = 4242

// writeTrappedRoomTemplate writes a template for roomId with a trapped `north`
// exit and an untrapped `south` exit, and registers its path.
func writeTrappedRoomTemplate(t *testing.T, tempDir string, roomId int, northTitle string) string {
	t.Helper()

	tpl := &Room{
		RoomId:      roomId,
		Zone:        "test_zone",
		Title:       northTitle,
		Description: "A room with a trapped door.",
		Exits: map[string]exit.RoomExit{
			"north": {
				RoomId: 200,
				Lock: gamelock.Lock{
					Difficulty:  30,
					TrapBuffIds: []int{defuseTrapBuffId},
				},
			},
			"south": {RoomId: 300},
		},
	}

	data, err := yaml.Marshal(tpl)
	require.NoError(t, err)

	rel := filepath.Join("test_zone", "500.yaml")
	templatePath := filepath.Join(tempDir, "rooms", rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(templatePath), 0755))
	require.NoError(t, os.WriteFile(templatePath, data, 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "rooms.instances", "test_zone"), 0755))

	roomManager.roomIdToFileCache[roomId] = "test_zone/500.yaml"
	t.Cleanup(func() { delete(roomManager.roomIdToFileCache, roomId) })

	return templatePath
}

// TestDefusedExit_SurvivesSaveAndReload is the regression test for exit-trap
// disarms not persisting.
//
// Room.Exits is tagged instance:"skip", so SaveRoomInstance never wrote the
// cleared trap and restoreSkipTaggedFields overwrote Exits from the template on
// every load — the trap came back after any restart or copyover. The container
// branch of the same defuse command persisted correctly because Room.Containers
// is not skip-tagged: two branches of one command with opposite guarantees.
//
// The second half of this test is the important half: it proves the fix did NOT
// buy persistence by un-skipping Exits. An authored edit to the template still
// takes effect on the next load, which is the whole reason the skip tag exists.
func TestDefusedExit_SurvivesSaveAndReload(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	const roomId = 500
	tempDir := useTempDataFiles(t, false)
	templatePath := writeTrappedRoomTemplate(t, tempDir, roomId, "Original Title")

	// --- Session 1: load, defuse, save. ---
	live := LoadRoomInstance(roomId)
	require.NotNil(t, live)
	require.Equal(t, []int{defuseTrapBuffId}, live.Exits["north"].Lock.TrapBuffIds,
		"setup precondition: the north exit should start trapped")

	live.MarkExitTrapDefused("north")
	assert.Nil(t, live.Exits["north"].Lock.TrapBuffIds,
		"defusing must clear the trap in the live room")
	assert.Equal(t, []string{"north"}, live.DefusedExits)

	require.NoError(t, SaveRoomInstance(*live))

	instancePath := filepath.Join(tempDir, "rooms.instances", "test_zone", "500.yaml")
	saved, err := os.ReadFile(instancePath)
	require.NoError(t, err, "the disarm must produce an instance save")
	assert.Contains(t, string(saved), "defusedexits",
		"the instance save must carry the defused-exit record")

	// --- Session 2: a fresh load, as after a restart or copyover. ---
	reloaded := LoadRoomInstance(roomId)
	require.NotNil(t, reloaded)
	assert.Nil(t, reloaded.Exits["north"].Lock.TrapBuffIds,
		"the disarmed trap must stay disarmed across a reload")

	// Everything else about the exit is still template-owned.
	assert.Equal(t, 200, reloaded.Exits["north"].RoomId)
	assert.Equal(t, uint8(30), reloaded.Exits["north"].Lock.Difficulty)
	assert.Len(t, reloaded.Exits, 2, "both authored exits must survive")

	// --- Shadowing check: re-author the template, reload, confirm it wins. ---
	// A builder renames the room, redirects north, and adds an east exit. If
	// DefusedExits had been bought by un-skipping Exits, the stale instance save
	// would shadow all of this.
	edited := &Room{
		RoomId:      roomId,
		Zone:        "test_zone",
		Title:       "Re-Authored Title",
		Description: "The builder edited this room.",
		Exits: map[string]exit.RoomExit{
			"north": {
				RoomId: 999,
				Lock: gamelock.Lock{
					Difficulty:  55,
					TrapBuffIds: []int{defuseTrapBuffId},
				},
			},
			"south": {RoomId: 300},
			"east":  {RoomId: 400},
		},
	}
	editedYAML, err := yaml.Marshal(edited)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(templatePath, editedYAML, 0644))

	afterEdit := LoadRoomInstance(roomId)
	require.NotNil(t, afterEdit)

	assert.Equal(t, "Re-Authored Title", afterEdit.Title,
		"authored title edits must still take effect (skip tag intact)")
	assert.Equal(t, "The builder edited this room.", afterEdit.Description)
	assert.Len(t, afterEdit.Exits, 3, "the newly authored east exit must appear")
	assert.Equal(t, 400, afterEdit.Exits["east"].RoomId)
	assert.Equal(t, 999, afterEdit.Exits["north"].RoomId,
		"an authored exit redirect must still take effect")
	assert.Equal(t, uint8(55), afterEdit.Exits["north"].Lock.Difficulty,
		"an authored lock-difficulty edit must still take effect")

	// The disarm is the ONE thing the instance file overrides, by design.
	assert.Nil(t, afterEdit.Exits["north"].Lock.TrapBuffIds,
		"the player's disarm still applies to the re-authored exit")
}

// TestApplyDefusedExits_StaleNameIsNoOp verifies that a defused-exit record for
// an exit the builder has since renamed or deleted does nothing at all, rather
// than resurrecting a phantom exit.
func TestApplyDefusedExits_StaleNameIsNoOp(t *testing.T) {
	r := &Room{
		Exits: map[string]exit.RoomExit{
			"south": {RoomId: 300, Lock: gamelock.Lock{TrapBuffIds: []int{defuseTrapBuffId}}},
		},
		DefusedExits: []string{"north"}, // no longer exists in the template
	}

	r.applyDefusedExits()

	assert.Len(t, r.Exits, 1, "a stale name must not create an exit")
	assert.Equal(t, []int{defuseTrapBuffId}, r.Exits["south"].Lock.TrapBuffIds,
		"a stale name must not clear an unrelated exit's trap")
}

// TestMarkExitTrapDefused_UnknownExitAndDedupe covers the two guards on the
// recording path.
func TestMarkExitTrapDefused_UnknownExitAndDedupe(t *testing.T) {
	r := &Room{
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 200, Lock: gamelock.Lock{TrapBuffIds: []int{defuseTrapBuffId}}},
		},
	}

	r.MarkExitTrapDefused("nowhere")
	assert.Empty(t, r.DefusedExits, "an unknown exit must not be recorded")

	r.MarkExitTrapDefused("north")
	r.MarkExitTrapDefused("north")
	assert.Equal(t, []string{"north"}, r.DefusedExits, "repeat defuses must not duplicate")
}
