package playtestrun

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSidecarWriteReadRoundTrip(t *testing.T) {
	checkout := t.TempDir()
	sc := SessionSidecar{
		RunID:       "run-abc",
		Checkout:    checkout,
		Commit:      "deadbeef",
		Dirty:       true,
		GoalsPath:   "goals.yaml",
		Personality: "bug-finder",
		Profile:     "veteran",
		Budgets:     SessionBudgets{WallClock: "30m"},
		StartedAt:   time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC),
		DeadlineAt:  time.Date(2026, 8, 8, 12, 30, 0, 0, time.UTC),
		Status:      StatusStarting,
		BridgeDir:   filepath.Join(checkout, "tools", "playtest", ".run", "run-abc", "bridge"),
	}
	path, err := WriteSidecar(checkout, sc)
	require.NoError(t, err)
	require.Equal(t, SidecarPath(checkout, "run-abc"), path)
	require.FileExists(t, path)

	got, err := ReadSidecar(checkout, "run-abc")
	require.NoError(t, err)
	require.Equal(t, "run-abc", got.RunID)
	require.Equal(t, "30m", got.Budgets.WallClock)
	require.Equal(t, StatusStarting, got.Status)
	require.Equal(t, "bug-finder", got.Personality)
	require.True(t, got.Dirty)
}

func TestSidecarStatusTransition(t *testing.T) {
	checkout := t.TempDir()
	sc := SessionSidecar{
		RunID:     "run-1",
		Checkout:  checkout,
		Budgets:   SessionBudgets{WallClock: "10m"},
		Status:    StatusStarting,
		BridgeDir: "bridge",
	}
	_, err := WriteSidecar(checkout, sc)
	require.NoError(t, err)

	require.NoError(t, UpdateSidecarStatus(checkout, "run-1", StatusReady))
	got, err := ReadSidecar(checkout, "run-1")
	require.NoError(t, err)
	require.Equal(t, StatusReady, got.Status)

	raw, err := os.ReadFile(SidecarPath(checkout, "run-1"))
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))
	budgets, ok := doc["budgets"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "10m", budgets["wall_clock"])
}
