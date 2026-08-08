package playtestprofiles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

func TestMaterializePersistsAndWritesCreds(t *testing.T) {
	tmp := t.TempDir()
	dataFiles := filepath.Join(tmp, "world")
	require.NoError(t, os.MkdirAll(filepath.Join(dataFiles, "users"), 0o755))
	profilesDir := filepath.Join(tmp, "profiles")
	require.NoError(t, os.MkdirAll(profilesDir, 0o755))
	src, err := os.ReadFile(filepath.Join("testdata", "fresh.yaml"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "fresh.yaml"), src, 0o644))

	prev := configs.GetFilePathsConfig().DataFiles.String()
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles":        dataFiles,
		"Validation.PasswordSizeMin": 4,
		"Validation.PasswordSizeMax": 16,
		"Validation.NameSizeMin":     1,
		"Validation.NameSizeMax":     80,
	}))
	t.Cleanup(func() {
		_ = configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": prev})
	})

	credsPath := filepath.Join(tmp, "control", "creds.json")
	m := &Manifest{Entries: []ManifestEntry{{
		Profile:   "fresh",
		StartRoom: 100,
		Overlays:  Overlays{GrantSpells: map[string]int{"heal": 1}},
	}}}
	creds, err := Materialize(m, MaterializeOptions{
		ProfilesDir:  profilesDir,
		World:        testWorld(),
		CredsOutPath: credsPath,
		RunID:        "testrun",
	})
	require.NoError(t, err)
	require.Len(t, creds, 1)
	require.Equal(t, "fresh", creds[0].Profile)
	require.Equal(t, 100, creds[0].RoomID)
	require.NotZero(t, creds[0].UserID)

	st, err := os.Stat(credsPath)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		require.Equal(t, os.FileMode(0o600), st.Mode().Perm())
	}

	raw, err := os.ReadFile(credsPath)
	require.NoError(t, err)
	var file CredsFile
	require.NoError(t, json.Unmarshal(raw, &file))
	require.Equal(t, "testrun", file.RunID)
	require.Equal(t, creds[0].Password, file.Players[0].Password)

	matches, err := filepath.Glob(filepath.Join(dataFiles, "users", "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, matches)
}

func TestMaterializeStampsActorIDOnDuplicateProfiles(t *testing.T) {
	tmp := t.TempDir()
	dataFiles := filepath.Join(tmp, "world")
	require.NoError(t, os.MkdirAll(filepath.Join(dataFiles, "users"), 0o755))
	profilesDir := filepath.Join(tmp, "profiles")
	require.NoError(t, os.MkdirAll(profilesDir, 0o755))
	src, err := os.ReadFile(filepath.Join("testdata", "fresh.yaml"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "early.yaml"), src, 0o644))

	prev := configs.GetFilePathsConfig().DataFiles.String()
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles":        dataFiles,
		"Validation.PasswordSizeMin": 4,
		"Validation.PasswordSizeMax": 16,
		"Validation.NameSizeMin":     1,
		"Validation.NameSizeMax":     80,
		"Validation.NameRejectRegex": `^[a-zA-Z0-9_]+$`,
	}))
	t.Cleanup(func() {
		_ = configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": prev})
	})

	credsPath := filepath.Join(tmp, "control", "creds.json")
	m := &Manifest{Entries: []ManifestEntry{
		{Profile: "early", StartRoom: 100, ActorID: "leader"},
		{Profile: "early", StartRoom: 100, ActorID: "joiner"},
	}}
	creds, err := Materialize(m, MaterializeOptions{
		ProfilesDir:  profilesDir,
		World:        testWorld(),
		CredsOutPath: credsPath,
		RunID:        "scenario-run",
	})
	require.NoError(t, err)
	require.Len(t, creds, 2)
	require.Equal(t, "leader", creds[0].ActorID)
	require.Equal(t, "joiner", creds[1].ActorID)
	require.Equal(t, "early", creds[0].Profile)
	require.Equal(t, "early", creds[1].Profile)

	raw, err := os.ReadFile(credsPath)
	require.NoError(t, err)
	var file CredsFile
	require.NoError(t, json.Unmarshal(raw, &file))
	require.Equal(t, "leader", file.Players[0].ActorID)
	require.Equal(t, "joiner", file.Players[1].ActorID)
}

func TestMaterializeFromConfigNoopWhenEmpty(t *testing.T) {
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{
		"Playtest.ProfilesManifest": "",
	}))
	creds, err := MaterializeFromConfig()
	require.NoError(t, err)
	require.Nil(t, creds)
}

func TestMaterializeSecondEntryFailureReturnsError(t *testing.T) {
	tmp := t.TempDir()
	dataFiles := filepath.Join(tmp, "world")
	require.NoError(t, os.MkdirAll(filepath.Join(dataFiles, "users"), 0o755))
	profilesDir := filepath.Join(tmp, "profiles")
	require.NoError(t, os.MkdirAll(profilesDir, 0o755))
	src, err := os.ReadFile(filepath.Join("testdata", "fresh.yaml"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "fresh.yaml"), src, 0o644))

	prev := configs.GetFilePathsConfig().DataFiles.String()
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles":        dataFiles,
		"Validation.PasswordSizeMin": 4,
		"Validation.PasswordSizeMax": 16,
		"Validation.NameSizeMin":     1,
		"Validation.NameSizeMax":     80,
		"Validation.NameRejectRegex": `^[a-zA-Z0-9_]+$`,
	}))
	t.Cleanup(func() {
		_ = configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": prev})
	})

	m := &Manifest{Entries: []ManifestEntry{
		{Profile: "fresh", StartRoom: 100},
		{Profile: "fresh", StartRoom: 99999}, // bad room via testWorld
	}}
	_, err = Materialize(m, MaterializeOptions{
		ProfilesDir: profilesDir,
		World:       testWorld(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "entries[1]")
	// First entry may have been persisted; fail-closed for the run is enough.
	matches, globErr := filepath.Glob(filepath.Join(dataFiles, "users", "*.yaml"))
	require.NoError(t, globErr)
	require.NotEmpty(t, matches)
}

func TestPersistOfflineUserPreservesAdminRole(t *testing.T) {
	tmp := t.TempDir()
	dataFiles := filepath.Join(tmp, "world")
	require.NoError(t, os.MkdirAll(filepath.Join(dataFiles, "users"), 0o755))
	prev := configs.GetFilePathsConfig().DataFiles.String()
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles": dataFiles,
	}))
	t.Cleanup(func() {
		_ = configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": prev})
	})

	u := &users.UserRecord{
		Role:     users.RoleAdmin,
		Username: "pt-admin-test01",
		Character: &characters.Character{
			Name: "Admin Tester",
		},
	}
	require.NoError(t, u.SetPassword("testpass12"))
	require.NoError(t, PersistOfflineUser(u))
	require.Equal(t, users.RoleAdmin, u.Role)
	require.True(t, u.IsAI)
	require.NotZero(t, u.UserId)
	// Must not appear as an online connection mapping
	require.Nil(t, users.GetByUserId(u.UserId))
}
