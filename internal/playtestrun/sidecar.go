package playtestrun

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GoMudEngine/GoMud/internal/playtestenv"
)

// Sidecar status values written to session.json.
const (
	StatusStarting            = "starting"
	StatusReady               = "ready"
	StatusIncompleteWallclock = "incomplete_wallclock"
	StatusInterrupted         = "interrupted"
	StatusStopped             = "stopped"
	StatusEnvironmentFailed   = "environment_failed"
)

// SessionBudgets nests wall-clock for the approved sidecar schema.
type SessionBudgets struct {
	WallClock string `json:"wall_clock"` // e.g. "30m"
}

// SessionSidecar is the machine-readable session record under
// tools/playtest/.run/<run_id>/session.json.
type SessionSidecar struct {
	RunID             string                `json:"run_id"`
	Checkout          string                `json:"checkout"`
	Commit            string                `json:"commit"`
	Dirty             bool                  `json:"dirty"`
	GoalsPath         string                `json:"goals_path"`
	Personality       string                `json:"personality,omitempty"`
	Endpoint          *playtestenv.Endpoint `json:"endpoint,omitempty"`
	Creds             string                `json:"creds,omitempty"`
	Profile           string                `json:"profile,omitempty"`
	CreationFlow      bool                  `json:"creation_flow,omitempty"`
	CreationRationale string                `json:"creation_rationale,omitempty"`
	Budgets           SessionBudgets        `json:"budgets"`
	StartedAt         time.Time             `json:"started_at"`
	DeadlineAt        time.Time             `json:"deadline_at"`
	Status            string                `json:"status"`
	EnvironmentReport string                `json:"environment_report,omitempty"`
	BridgeDir         string                `json:"bridge_dir"`
}

// RunDir returns <checkout>/tools/playtest/.run/<runID>.
func RunDir(checkout, runID string) string {
	return filepath.Join(checkout, "tools", "playtest", ".run", runID)
}

// SidecarPath returns the session.json path for a run.
func SidecarPath(checkout, runID string) string {
	return filepath.Join(RunDir(checkout, runID), "session.json")
}

// BridgeDirPath returns the run-scoped mudagent bridge directory.
func BridgeDirPath(checkout, runID string) string {
	return filepath.Join(RunDir(checkout, runID), "bridge")
}

// WriteSidecar atomically writes session.json and returns its path.
func WriteSidecar(checkout string, sc SessionSidecar) (string, error) {
	if sc.RunID == "" {
		return "", fmt.Errorf("playtestrun: sidecar run_id is required")
	}
	dir := RunDir(checkout, sc.RunID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("playtestrun: mkdir run dir: %w", err)
	}
	path := SidecarPath(checkout, sc.RunID)
	raw, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("playtestrun: marshal sidecar: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("playtestrun: write sidecar temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("playtestrun: rename sidecar: %w", err)
	}
	return path, nil
}

// ReadSidecar loads session.json for a run.
func ReadSidecar(checkout, runID string) (SessionSidecar, error) {
	path := SidecarPath(checkout, runID)
	raw, err := os.ReadFile(path)
	if err != nil {
		return SessionSidecar{}, fmt.Errorf("playtestrun: read sidecar %s: %w", path, err)
	}
	var sc SessionSidecar
	if err := json.Unmarshal(raw, &sc); err != nil {
		return SessionSidecar{}, fmt.Errorf("playtestrun: parse sidecar %s: %w", path, err)
	}
	return sc, nil
}

// UpdateSidecarStatus loads, mutates status, and rewrites the sidecar.
func UpdateSidecarStatus(checkout, runID, status string) error {
	sc, err := ReadSidecar(checkout, runID)
	if err != nil {
		return err
	}
	sc.Status = status
	_, err = WriteSidecar(checkout, sc)
	return err
}
