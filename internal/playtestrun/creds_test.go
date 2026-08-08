package playtestrun

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/GoMudEngine/GoMud/internal/playtestprofiles"
)

func writeCreds(t *testing.T, file playtestprofiles.CredsFile) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "creds.json")
	raw, err := json.Marshal(file)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
	return path
}

func TestSelectCredsPlayer_MatchesProfile(t *testing.T) {
	path := writeCreds(t, playtestprofiles.CredsFile{
		RunID: "r1",
		Players: []playtestprofiles.PlayerCreds{
			{Profile: "early", Username: "pt_early_aaa", Password: "secret1", UserID: 1, RoomID: 10},
			{Profile: "veteran", Username: "pt_veteran_bbb", Password: "secret2", UserID: 2, RoomID: 20},
		},
	})
	user, pass, err := SelectCredsPlayer(path, "veteran")
	require.NoError(t, err)
	require.Equal(t, "pt_veteran_bbb", user)
	require.Equal(t, "secret2", pass)
}

func TestSelectCredsPlayer_MissingProfile(t *testing.T) {
	path := writeCreds(t, playtestprofiles.CredsFile{
		Players: []playtestprofiles.PlayerCreds{
			{Profile: "fresh", Username: "u", Password: "s3cret-fresh-only", UserID: 1, RoomID: 1},
		},
	})
	_, _, err := SelectCredsPlayer(path, "veteran")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "s3cret-fresh-only")
}

func TestSelectCredsPlayer_AmbiguousProfile(t *testing.T) {
	path := writeCreds(t, playtestprofiles.CredsFile{
		Players: []playtestprofiles.PlayerCreds{
			{Profile: "fresh", Username: "u1", Password: "s3cret-one", UserID: 1, RoomID: 1},
			{Profile: "fresh", Username: "u2", Password: "s3cret-two", UserID: 2, RoomID: 2},
		},
	})
	_, _, err := SelectCredsPlayer(path, "fresh")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "s3cret-one")
	require.NotContains(t, err.Error(), "s3cret-two")
}

func TestSelectCredsPlayer_MissingFile(t *testing.T) {
	_, _, err := SelectCredsPlayer(filepath.Join(t.TempDir(), "nope.json"), "fresh")
	require.Error(t, err)
}

func TestSelectCredsByActorID_Matches(t *testing.T) {
	path := writeCreds(t, playtestprofiles.CredsFile{
		RunID: "r1",
		Players: []playtestprofiles.PlayerCreds{
			{Profile: "early", ActorID: "leader", Username: "pt_early_aaa", Password: "secret1", UserID: 1, RoomID: 10},
			{Profile: "early", ActorID: "joiner", Username: "pt_early_bbb", Password: "secret2", UserID: 2, RoomID: 20},
		},
	})
	user, pass, err := SelectCredsByActorID(path, "joiner")
	require.NoError(t, err)
	require.Equal(t, "pt_early_bbb", user)
	require.Equal(t, "secret2", pass)
}

func TestSelectCredsByActorID_Missing(t *testing.T) {
	path := writeCreds(t, playtestprofiles.CredsFile{
		Players: []playtestprofiles.PlayerCreds{
			{Profile: "early", ActorID: "leader", Username: "u", Password: "s3cret-leader", UserID: 1, RoomID: 1},
		},
	})
	_, _, err := SelectCredsByActorID(path, "joiner")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "s3cret-leader")
}

func TestSelectCredsPlayer_StillAmbiguousWithActorIDs(t *testing.T) {
	// Profile-only helper remains single-agent: duplicate profiles still error
	// even when actor_id disambiguates for SelectCredsByActorID.
	path := writeCreds(t, playtestprofiles.CredsFile{
		Players: []playtestprofiles.PlayerCreds{
			{Profile: "early", ActorID: "leader", Username: "u1", Password: "s3cret-one", UserID: 1, RoomID: 1},
			{Profile: "early", ActorID: "joiner", Username: "u2", Password: "s3cret-two", UserID: 2, RoomID: 2},
		},
	})
	_, _, err := SelectCredsPlayer(path, "early")
	require.Error(t, err)
	require.Contains(t, err.Error(), "ambiguous")
}
