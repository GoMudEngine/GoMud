package rooms

import (
	"testing"

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
