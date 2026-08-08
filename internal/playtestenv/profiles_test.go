package playtestenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/GoMudEngine/GoMud/internal/version"
)

func TestMaterializeRunFilesWritesProfilesManifestAndOverrides(t *testing.T) {
	runDir := t.TempDir()
	controlDir := filepath.Join(runDir, "control")
	require.NoError(t, os.Mkdir(controlDir, 0o755))

	gold := 500
	profiles := []ProfileRequest{
		{
			Profile:   "veteran",
			StartRoom: 462,
			Overlays: ProfileOverlays{
				GrantSpells: map[string]int{"heal": 1},
				SetGold:     &gold,
			},
		},
		{Profile: "fresh", StartRoom: 5200},
	}

	composePath, configPath, profilesPath, err := materializeRunFiles(runDir, controlDir, version.New(0, 16, 0), profiles)
	require.NoError(t, err)
	require.FileExists(t, composePath)
	require.Equal(t, filepath.Join(controlDir, profilesManifestFileName), profilesPath)
	require.FileExists(t, profilesPath)

	configBytes, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var overrides configOverridesDoc
	require.NoError(t, yaml.Unmarshal(configBytes, &overrides))
	require.NotNil(t, overrides.Playtest)
	require.Equal(t, containerProfilesDir, overrides.Playtest.ProfilesDir)
	require.Equal(t, containerProfilesManifest, overrides.Playtest.ProfilesManifest)

	manifestBytes, err := os.ReadFile(profilesPath)
	require.NoError(t, err)
	var doc profilesManifestDoc
	require.NoError(t, yaml.Unmarshal(manifestBytes, &doc))
	require.Len(t, doc.Entries, 2)
	require.Equal(t, "veteran", doc.Entries[0].Profile)
	require.Equal(t, 462, doc.Entries[0].StartRoom)
	require.Equal(t, 1, doc.Entries[0].Overlays.GrantSpells["heal"])
	require.NotNil(t, doc.Entries[0].Overlays.SetGold)
	require.Equal(t, 500, *doc.Entries[0].Overlays.SetGold)
	require.Equal(t, "fresh", doc.Entries[1].Profile)

	// Must never embed plaintext secrets in control YAML the supervisor writes.
	require.NotContains(t, strings.ToLower(string(manifestBytes)), "password")
	require.NotContains(t, strings.ToLower(string(configBytes)), "password")
}

func TestWriteProfilesManifestRejectsInvalidRequests(t *testing.T) {
	controlDir := t.TempDir()
	_, err := writeProfilesManifest(controlDir, nil)
	require.Error(t, err)

	_, err = writeProfilesManifest(controlDir, []ProfileRequest{{Profile: "", StartRoom: 1}})
	require.Error(t, err)

	_, err = writeProfilesManifest(controlDir, []ProfileRequest{{Profile: "fresh", StartRoom: 0}})
	require.Error(t, err)
}

func TestDockerfileDockerignoreKeepsPlaytestProfiles(t *testing.T) {
	root := filepath.Join("..", "..")
	dockerignore := filepath.Join(root, "provisioning", "Dockerfile.dockerignore")
	body, err := os.ReadFile(dockerignore)
	require.NoError(t, err)
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Negation patterns that re-include are fine; only bare exclusions matter.
		if strings.HasPrefix(trimmed, "!") {
			continue
		}
		require.False(t, pathExcludesPlaytestProfiles(trimmed),
			"Dockerfile.dockerignore must not exclude tools/playtest/profiles (line %q)", trimmed)
	}

	dockerfile := filepath.Join(root, "provisioning", "Dockerfile")
	df, err := os.ReadFile(dockerfile)
	require.NoError(t, err)
	require.Contains(t, string(df), "COPY --from=builder /src/tools/playtest/profiles /app/playtest/profiles")
}

func pathExcludesPlaytestProfiles(pattern string) bool {
	p := strings.ReplaceAll(pattern, "\\", "/")
	switch {
	case p == "tools/playtest/profiles",
		p == "tools/playtest/profiles/",
		p == "tools/playtest/profiles/**",
		p == "tools/playtest/profiles/*",
		p == "tools/playtest",
		p == "tools/playtest/",
		p == "tools/playtest/**",
		p == "tools",
		p == "tools/",
		p == "tools/**":
		return true
	default:
		return false
	}
}

func TestFailureReportListsCredsPathWithoutEmbeddingBody(t *testing.T) {
	checkout := t.TempDir()
	reportsDir := filepath.Join(checkout, "tools", "playtest", "reports")
	require.NoError(t, os.MkdirAll(reportsDir, 0o755))

	runDir := filepath.Join(checkout, "tools", "playtest", ".run", "run-creds")
	controlDir := filepath.Join(runDir, "control")
	require.NoError(t, os.MkdirAll(controlDir, 0o755))

	credsPath := filepath.Join(controlDir, credsFileName)
	// Plant a realistic secret body on disk; the report must never copy it.
	secretBody := `{"players":[{"profile":"fresh","username":"pt-fresh-x","password":"super-secret-pass","user_id":1,"room_id":5200}]}`
	require.NoError(t, os.WriteFile(credsPath, []byte(secretBody), 0o600))

	m := &Manifest{
		RunID:    "run-creds",
		Checkout: checkout,
		State:    StateFailed,
		Failure: &FailureRecord{
			Category: FailureBootPanic,
			Phase:    StateStarting,
			Summary:  "materializer failed",
		},
		Artifacts: ArtifactPaths{
			Manifest:  filepath.Join(runDir, "manifest.json"),
			BuildLog:  filepath.Join(runDir, "build.log"),
			ServerLog: filepath.Join(runDir, "server.log"),
			Compose:   filepath.Join(runDir, "compose.resolved.yml"),
			Config:    filepath.Join(controlDir, configOverridesFileName),
			Creds:     credsPath,
		},
	}
	require.NoError(t, os.WriteFile(m.Artifacts.BuildLog, []byte("ok\n"), 0o644))
	require.NoError(t, os.WriteFile(m.Artifacts.ServerLog, []byte("panic\n"), 0o644))

	path, err := writeFailureReport(checkout, m, &CleanupResult{Complete: true},
		time.Date(2026, 8, 8, 15, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	text := string(body)

	require.Contains(t, text, credsPath)
	require.NotContains(t, text, "super-secret-pass")
	require.NotContains(t, text, secretBody)
	require.NotContains(t, strings.ToLower(text), `"password"`)
}
