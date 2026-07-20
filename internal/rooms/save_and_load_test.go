package rooms

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

// ─── restoreSkipTaggedFields ────────────────────────────────────────────────

// TestRestoreSkipTaggedFields verifies the helper copies every field
// tagged `instance:"skip"` from src onto dst, leaves non-skip fields
// on dst alone.
func TestRestoreSkipTaggedFields(t *testing.T) {
	// src = "template" values
	src := &Room{
		RoomId:      100,
		Zone:        "template_zone",
		Title:       "Template Title",
		Description: "Template description.",
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 200},
			"south": {RoomId: 300},
		},
		Nouns: map[string]string{"altar": "A stone altar."},
		Gold:  0, // non-skip field; runtime state
	}

	// dst = "post-unmarshal" values with corrupt skip fields + legit
	// non-skip runtime state.
	dst := &Room{
		RoomId:      100,
		Zone:        "corrupt_zone",  // skip-tagged; should be restored
		Title:       "Corrupt Title", // skip-tagged; should be restored
		Description: "Corrupt.",      // skip-tagged; should be restored
		Exits: map[string]exit.RoomExit{
			"east": {RoomId: 999}, // skip-tagged; should be restored
		},
		Nouns: map[string]string{"poison": "Bad data."}, // skip-tagged
		Gold:  42,                                       // NOT skip-tagged; should stay
	}

	restoreSkipTaggedFields(dst, src)

	// Skip-tagged fields now match src
	assert.Equal(t, "template_zone", dst.Zone)
	assert.Equal(t, "Template Title", dst.Title)
	assert.Equal(t, "Template description.", dst.Description)
	assert.Equal(t, 2, len(dst.Exits))
	assert.Equal(t, 200, dst.Exits["north"].RoomId)
	assert.Equal(t, 300, dst.Exits["south"].RoomId)
	_, hasEast := dst.Exits["east"]
	assert.False(t, hasEast, "east should have been replaced by template exits")
	assert.Equal(t, "A stone altar.", dst.Nouns["altar"])
	_, hasPoison := dst.Nouns["poison"]
	assert.False(t, hasPoison, "poison noun should have been replaced")

	// Non-skip field preserved
	assert.Equal(t, 42, dst.Gold)
}

// ─── LoadRoomInstance — skip-tag enforcement ────────────────────────────────

// TestLoadRoomInstance_RestoresSkipTaggedFieldsFromTemplate verifies that
// an instance file containing skip-tagged fields cannot overwrite the
// template values on load. This is an end-to-end test exercising the full
// yaml.Unmarshal → restoreSkipTaggedFields flow on real YAML files.
func TestLoadRoomInstance_RestoresSkipTaggedFieldsFromTemplate(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Set up file paths infrastructure
	tempDir := t.TempDir()
	roomsDir := filepath.Join(tempDir, "rooms")
	instancesDir := filepath.Join(tempDir, "rooms.instances")
	os.MkdirAll(roomsDir, 0755)
	os.MkdirAll(instancesDir, 0755)

	// Mock the config to point to our temp directories
	prev := configs.GetFilePathsConfig()
	configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles": tempDir,
	})
	defer func() {
		configs.AddOverlayOverrides(map[string]any{
			"FilePaths.DataFiles": prev.DataFiles.String(),
		})
	}()

	// Create a template with specific Title, Description, Exits
	templateRoom := &Room{
		RoomId:      100,
		Zone:        "test_zone",
		Title:       "Template Title",
		Description: "A template room.",
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 200},
			"south": {RoomId: 300},
		},
		Gold: 0, // non-skip; will be overwritten by instance
	}

	// Write template to file
	templateYAML, err := yaml.Marshal(templateRoom)
	assert.NoError(t, err)
	templatePath := filepath.Join(roomsDir, "test_zone", "100.yaml")
	os.MkdirAll(filepath.Dir(templatePath), 0755)
	err = os.WriteFile(templatePath, templateYAML, 0644)
	assert.NoError(t, err)

	// Create a corrupt instance file with different Title/Description/Exits
	// (simulating a pre-fix save file that mistakenly wrote skip-tagged fields)
	corruptInstance := map[string]interface{}{
		"gold": 42, // non-skip field; should be applied
		// Skip-tagged fields that should NOT be applied
		"title":       "Corrupt Title",
		"description": "Bad data.",
		"exits": map[string]interface{}{
			"east": map[string]interface{}{
				"roomid": 999,
			},
		},
	}

	instanceYAML, err := yaml.Marshal(corruptInstance)
	assert.NoError(t, err)
	instancePath := filepath.Join(instancesDir, "test_zone", "100.yaml")
	os.MkdirAll(filepath.Dir(instancePath), 0755)
	err = os.WriteFile(instancePath, instanceYAML, 0644)
	assert.NoError(t, err)

	// Register the room file path in the cache
	roomManager.roomIdToFileCache[100] = "test_zone/100.yaml"

	// Call LoadRoomInstance
	loaded := LoadRoomInstance(100)
	assert.NotNil(t, loaded)

	// Verify: skip-tagged fields should come from template
	assert.Equal(t, "Template Title", loaded.Title,
		"Title should be restored from template, not from instance file")
	assert.Equal(t, "A template room.", loaded.Description,
		"Description should be restored from template, not from instance file")
	assert.Equal(t, 2, len(loaded.Exits),
		"Exits should be restored from template, not corrupted by instance")
	assert.Equal(t, 200, loaded.Exits["north"].RoomId)
	assert.Equal(t, 300, loaded.Exits["south"].RoomId)
	_, hasEast := loaded.Exits["east"]
	assert.False(t, hasEast, "corrupt east exit should not appear")

	// Verify: non-skip field should be applied from instance
	assert.Equal(t, 42, loaded.Gold,
		"Gold should be applied from instance file")

	// Cleanup
	delete(roomManager.roomIdToFileCache, 100)
}

// TestLoadRoomInstance_AppliesNonSkipFields is a control case verifying
// that non-skip fields ARE correctly applied from instance files while
// skip-tagged fields are ignored.
func TestLoadRoomInstance_AppliesNonSkipFields(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Set up file paths infrastructure
	tempDir := t.TempDir()
	roomsDir := filepath.Join(tempDir, "rooms")
	instancesDir := filepath.Join(tempDir, "rooms.instances")
	os.MkdirAll(roomsDir, 0755)
	os.MkdirAll(instancesDir, 0755)

	// Mock the config to point to our temp directories
	prev := configs.GetFilePathsConfig()
	configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles": tempDir,
	})
	defer func() {
		configs.AddOverlayOverrides(map[string]any{
			"FilePaths.DataFiles": prev.DataFiles.String(),
		})
	}()

	// Create a template with Gold=0 (non-skip field)
	templateRoom := &Room{
		RoomId:      101,
		Zone:        "test_zone",
		Title:       "Control Title",
		Description: "Control room.",
		Exits:       map[string]exit.RoomExit{},
		Gold:        0,
	}

	// Write template to file
	templateYAML, err := yaml.Marshal(templateRoom)
	assert.NoError(t, err)
	templatePath := filepath.Join(roomsDir, "test_zone", "101.yaml")
	os.MkdirAll(filepath.Dir(templatePath), 0755)
	err = os.WriteFile(templatePath, templateYAML, 0644)
	assert.NoError(t, err)

	// Create an instance file with Gold=99
	instanceData := map[string]interface{}{
		"gold": 99, // non-skip; should be applied
	}

	instanceYAML, err := yaml.Marshal(instanceData)
	assert.NoError(t, err)
	instancePath := filepath.Join(instancesDir, "test_zone", "101.yaml")
	os.MkdirAll(filepath.Dir(instancePath), 0755)
	err = os.WriteFile(instancePath, instanceYAML, 0644)
	assert.NoError(t, err)

	// Register the room file path in the cache
	roomManager.roomIdToFileCache[101] = "test_zone/101.yaml"

	// Call LoadRoomInstance
	loaded := LoadRoomInstance(101)
	assert.NotNil(t, loaded)

	// Verify: non-skip field is applied from instance
	assert.Equal(t, 99, loaded.Gold,
		"non-skip Gold field should be applied from instance")

	// Verify: skip-tagged fields still come from template
	assert.Equal(t, "Control Title", loaded.Title)
	assert.Equal(t, "Control room.", loaded.Description)

	// Cleanup
	delete(roomManager.roomIdToFileCache, 101)
}
