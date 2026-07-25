package rooms

import (
	"testing"

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
