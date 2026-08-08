package playtestprofiles

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseManifestEmptyEntries(t *testing.T) {
	m, err := ParseManifest([]byte("entries: []\n"))
	require.NoError(t, err)
	require.Empty(t, m.Entries)
}

func TestParseManifestHappy(t *testing.T) {
	raw := `
entries:
  - profile: veteran
    start_room: 462
    overlays:
      grant_spells:
        heal: 1
      set_gold: 50
`
	m, err := ParseManifest([]byte(raw))
	require.NoError(t, err)
	require.Len(t, m.Entries, 1)
	require.Equal(t, "veteran", m.Entries[0].Profile)
	require.Equal(t, 462, m.Entries[0].StartRoom)
	require.Equal(t, 1, m.Entries[0].Overlays.GrantSpells["heal"])
	require.NotNil(t, m.Entries[0].Overlays.SetGold)
	require.Equal(t, 50, *m.Entries[0].Overlays.SetGold)
}

func TestParseManifestRejectsUnknownOverlayKey(t *testing.T) {
	raw := `
entries:
  - profile: fresh
    start_room: 100
    overlays:
      totally_unknown: true
`
	_, err := ParseManifest([]byte(raw))
	require.Error(t, err)
	require.True(t,
		strings.Contains(err.Error(), "totally_unknown") || strings.Contains(err.Error(), "field"),
		"expected unknown-field error, got %v", err)
}

func TestParseManifestRejectsUnknownProfile(t *testing.T) {
	raw := `
entries:
  - profile: not-a-real-profile
    start_room: 100
`
	_, err := ParseManifest([]byte(raw))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown profile")
}

func TestParseManifestRejectsMissingStartRoom(t *testing.T) {
	raw := `
entries:
  - profile: fresh
    start_room: 0
`
	_, err := ParseManifest([]byte(raw))
	require.Error(t, err)
	require.Contains(t, err.Error(), "start_room")
}
