package playtestrun

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeGoals(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "goals.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func TestParseGoalsEphemeral_ProfileHappy(t *testing.T) {
	path := writeGoals(t, `
goals:
  - do a thing
ephemeral:
  profile: veteran
  start_room: 5455
  budgets:
    wall_clock: 45m
`)
	got, err := ParseGoalsEphemeral(path)
	require.NoError(t, err)
	require.Equal(t, "veteran", got.Profile)
	require.Equal(t, 5455, got.StartRoom)
	require.False(t, got.CreationFlow)
	require.Equal(t, 45*time.Minute, got.WallClock)
}

func TestParseGoalsEphemeral_OverlaysRoundTrip(t *testing.T) {
	path := writeGoals(t, `
ephemeral:
  profile: veteran
  start_room: 462
  overlays:
    grant_spells:
      heal: 1
    set_gold: 500
`)
	got, err := ParseGoalsEphemeral(path)
	require.NoError(t, err)
	require.Equal(t, 1, got.Overlays.GrantSpells["heal"])
	require.NotNil(t, got.Overlays.SetGold)
	require.Equal(t, 500, *got.Overlays.SetGold)
}

func TestParseGoalsEphemeral_CreationFlowHappy(t *testing.T) {
	path := writeGoals(t, `
ephemeral:
  creation_flow: true
  creation_rationale: >
    Brand-new character required for this run.
`)
	got, err := ParseGoalsEphemeral(path)
	require.NoError(t, err)
	require.True(t, got.CreationFlow)
	require.Contains(t, got.CreationRationale, "Brand-new character")
	require.Empty(t, got.Profile)
	require.Equal(t, 0, got.StartRoom)
	require.Equal(t, 30*time.Minute, got.WallClock)
}

func TestParseGoalsEphemeral_MissingEphemeral(t *testing.T) {
	path := writeGoals(t, `
goals:
  - only goals
`)
	_, err := ParseGoalsEphemeral(path)
	require.Error(t, err)
}

func TestParseGoalsEphemeral_UnknownKey(t *testing.T) {
	path := writeGoals(t, `
ephemeral:
  profile: fresh
  start_room: 1
  mystery: true
`)
	_, err := ParseGoalsEphemeral(path)
	require.Error(t, err)
}

func TestParseGoalsEphemeral_BothProfileAndCreation(t *testing.T) {
	path := writeGoals(t, `
ephemeral:
  profile: fresh
  start_room: 1
  creation_flow: true
  creation_rationale: nope
`)
	_, err := ParseGoalsEphemeral(path)
	require.Error(t, err)
}

func TestParseGoalsEphemeral_NeitherProfileNorCreation(t *testing.T) {
	path := writeGoals(t, `
ephemeral:
  budgets:
    wall_clock: 10m
`)
	_, err := ParseGoalsEphemeral(path)
	require.Error(t, err)
}

func TestParseGoalsEphemeral_CreationWithoutRationale(t *testing.T) {
	path := writeGoals(t, `
ephemeral:
  creation_flow: true
`)
	_, err := ParseGoalsEphemeral(path)
	require.Error(t, err)
}

func TestParseGoalsEphemeral_CreationEmptyRationale(t *testing.T) {
	path := writeGoals(t, `
ephemeral:
  creation_flow: true
  creation_rationale: "   "
`)
	_, err := ParseGoalsEphemeral(path)
	require.Error(t, err)
}

func TestParseGoalsEphemeral_CreationForbidsProfileStartRoomOverlays(t *testing.T) {
	cases := []string{
		`
ephemeral:
  creation_flow: true
  creation_rationale: why
  profile: fresh
`,
		`
ephemeral:
  creation_flow: true
  creation_rationale: why
  start_room: 1
`,
		`
ephemeral:
  creation_flow: true
  creation_rationale: why
  overlays:
    set_gold: 1
`,
	}
	for i, body := range cases {
		_, err := ParseGoalsEphemeral(writeGoals(t, body))
		require.Error(t, err, "case %d", i)
	}
}

func TestParseGoalsEphemeral_ProfileRequiresPositiveStartRoom(t *testing.T) {
	cases := []string{
		`
ephemeral:
  profile: fresh
`,
		`
ephemeral:
  profile: fresh
  start_room: 0
`,
		`
ephemeral:
  profile: fresh
  start_room: -3
`,
	}
	for i, body := range cases {
		_, err := ParseGoalsEphemeral(writeGoals(t, body))
		require.Error(t, err, "case %d", i)
	}
}

func TestParseGoalsEphemeral_UnknownProfileID(t *testing.T) {
	path := writeGoals(t, `
ephemeral:
  profile: not-a-real-profile
  start_room: 1
`)
	_, err := ParseGoalsEphemeral(path)
	require.Error(t, err)
}

func TestParseGoalsEphemeral_DefaultWallClock(t *testing.T) {
	path := writeGoals(t, `
ephemeral:
  profile: early
  start_room: 100
`)
	got, err := ParseGoalsEphemeral(path)
	require.NoError(t, err)
	require.Equal(t, 30*time.Minute, got.WallClock)
}

func TestParseGoalsEphemeral_MissingPath(t *testing.T) {
	_, err := ParseGoalsEphemeral(filepath.Join(t.TempDir(), "missing.yaml"))
	require.Error(t, err)
}

func TestParseGoalsEphemeral_UnreadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "goals.yaml")
	require.NoError(t, os.Mkdir(path, 0o755)) // directory, not a file
	_, err := ParseGoalsEphemeral(path)
	require.Error(t, err)
}

func TestParseGoalsEphemeral_IgnoresLegacyMaxRounds(t *testing.T) {
	path := writeGoals(t, `
session:
  max_rounds: 40
ephemeral:
  profile: mid
  start_room: 200
`)
	got, err := ParseGoalsEphemeral(path)
	require.NoError(t, err)
	require.Equal(t, "mid", got.Profile)
	require.Equal(t, 200, got.StartRoom)
}
