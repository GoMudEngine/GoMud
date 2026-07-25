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
