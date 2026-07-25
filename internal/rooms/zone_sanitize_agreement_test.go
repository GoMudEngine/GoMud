package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// internal/mobs carries its OWN ZoneNameSanitize (mobs.go:1212), used by
// Mob.Filepath to choose the folder a mob template is written to.
// rooms.ZoneNameSanitize chooses the folder everything else uses. If the two
// ever drift, a zone rename moves the rooms one way and the mobs another, and
// the mobs silently vanish from the zone — no error, no log line, just an
// empty-feeling zone.
//
// This test lives in package rooms rather than package mobs on purpose: rooms
// already imports mobs (roomdetails.go), so the reverse import is a cycle.
// Comparing the two implementations directly is worth more than asserting the
// mobs one against a hardcoded table, which could itself drift from rooms.
func TestZoneNameSanitize_MobsAgreesWithRooms(t *testing.T) {
	for _, in := range []string{
		"Stillwater", "Amber Valley", "amber valley", "Amber_Valley",
		"New Plymouth Common", "A", "", "Zone With  Double Space",
		"MiXeD CaSe Zone", "trailing space ", " leading space",
	} {
		if got, want := mobs.ZoneNameSanitize(in), ZoneNameSanitize(in); got != want {
			t.Errorf("ZoneNameSanitize(%q): mobs=%q rooms=%q — the two must agree", in, got, want)
		}
	}
}
