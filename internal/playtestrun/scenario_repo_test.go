package playtestrun

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func repoPlaytestRoot(t *testing.T) (checkout, playtestRoot string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	checkout = filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	playtestRoot = filepath.Join(checkout, "tools", "playtest")
	return checkout, playtestRoot
}

func TestParseScenario_RepoMigratedScenarios(t *testing.T) {
	checkout, playtestRoot := repoPlaytestRoot(t)
	for _, name := range []string{
		"party-formation.yaml",
		"parallel-coverage.yaml",
		"feel-pothole-newbie-veteran.yaml",
	} {
		path := filepath.Join(playtestRoot, "scenarios", name)
		got, err := ParseScenario(path, playtestRoot, ScenarioParseOpts{
			Checkout: checkout,
		})
		require.NoError(t, err, name)
		require.GreaterOrEqual(t, len(got.Roster), 2, name)
		require.Equal(t, onActorStopContinue, got.OnActorStop, name)
		require.Equal(t, defaultScenarioWallClock, got.WallClock, name)
	}
}

func TestParseScenario_RepoAdversarialContestDeferred(t *testing.T) {
	checkout, playtestRoot := repoPlaytestRoot(t)
	path := filepath.Join(playtestRoot, "scenarios", "adversarial-contest.yaml")
	_, err := ParseScenario(path, playtestRoot, ScenarioParseOpts{Checkout: checkout})
	require.Error(t, err)
	require.Contains(t, err.Error(), "pvp")
}
