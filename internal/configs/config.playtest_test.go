package configs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlaytestValidateDefaultsProfilesDir(t *testing.T) {
	p := Playtest{}
	p.Validate()
	require.Equal(t, ConfigString(`tools/playtest/profiles`), p.ProfilesDir)
	require.Equal(t, ConfigString(``), p.ProfilesManifest)
}

func TestPlaytestValidatePreservesExplicitDir(t *testing.T) {
	p := Playtest{
		ProfilesDir:      `/app/playtest/profiles`,
		ProfilesManifest: `/run/dogmud/profiles-manifest.yaml`,
	}
	p.Validate()
	require.Equal(t, ConfigString(`/app/playtest/profiles`), p.ProfilesDir)
	require.Equal(t, ConfigString(`/run/dogmud/profiles-manifest.yaml`), p.ProfilesManifest)
}
