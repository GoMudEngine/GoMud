package playtestrun

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/playtestenv"
)

const (
	leaseBuffer      = 5 * time.Minute
	stopPollInterval = 200 * time.Millisecond
)

// envSupervisor is the narrow playtestenv surface used by playtestrun.
type envSupervisor interface {
	Start(context.Context, playtestenv.StartOptions) (playtestenv.Result, error)
	Stop(context.Context, playtestenv.RunOptions) (playtestenv.Result, error)
	Status(context.Context, playtestenv.RunOptions) (playtestenv.Result, error)
}

// Clock abstracts time for wall-clock watchdog tests.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

type realClock struct{}

func (realClock) Now() time.Time                         { return time.Now() }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// RunParams configures a blocking playtestrun session.
type RunParams struct {
	Checkout          string
	GoalsPath         string
	Personality       string
	WallClockOverride time.Duration // 0 = use goals binding
	Env               envSupervisor
	Clock             Clock
	Stdout            io.Writer
	Stderr            io.Writer
	Commit            string // optional override (tests); empty ⇒ probe git
	Dirty             *bool  // optional override (tests)
	StopPollInterval  time.Duration
}

// ReadyPayload is the one-line JSON printed when the session is ready.
type ReadyPayload struct {
	Endpoint   *playtestenv.Endpoint `json:"endpoint"`
	Creds      *string               `json:"creds"` // null when creation-flow
	RunID      string                `json:"run_id"`
	Checkout   string                `json:"checkout"`
	Commit     string                `json:"commit"`
	Dirty      bool                  `json:"dirty"`
	DeadlineAt time.Time             `json:"deadline_at"`
	Sidecar    string                `json:"sidecar"`
	BridgeDir  string                `json:"bridge_dir"`
}

// Run starts playtestenv, watches the wall-clock / stop signal, then stops.
// Returns a non-zero-style error for environment_failed and incomplete_wallclock
// (callers map err != nil to exit code != 0).
func Run(ctx context.Context, p RunParams) error {
	if err := validateRunParams(p); err != nil {
		return err
	}
	if p.Clock == nil {
		p.Clock = realClock{}
	}
	if p.Stdout == nil {
		p.Stdout = io.Discard
	}
	if p.Stderr == nil {
		p.Stderr = io.Discard
	}
	if p.StopPollInterval <= 0 {
		p.StopPollInterval = stopPollInterval
	}

	binding, err := ParseGoalsEphemeral(p.GoalsPath)
	if err != nil {
		return err
	}
	wallClock := binding.WallClock
	if p.WallClockOverride > 0 {
		wallClock = p.WallClockOverride
	}

	lease := wallClock + leaseBuffer
	startOpts := playtestenv.StartOptions{
		Checkout: p.Checkout,
		Lease:    lease,
	}
	if !binding.CreationFlow {
		startOpts.Profiles = []playtestenv.ProfileRequest{{
			Profile:   binding.Profile,
			StartRoom: binding.StartRoom,
			Overlays:  binding.Overlays,
		}}
	}

	startRes, startErr := p.Env.Start(ctx, startOpts)
	commit, dirty := resolveGitMeta(p)

	if startErr != nil {
		runID := startRes.RunID
		if runID == "" {
			runID = "failed-" + fmt.Sprintf("%d", p.Clock.Now().UnixNano())
		}
		bridge := BridgeDirPath(p.Checkout, runID)
		_ = os.MkdirAll(bridge, 0o755)
		sc := baseSidecar(p, binding, wallClock, runID, commit, dirty, bridge)
		sc.Status = StatusEnvironmentFailed
		sc.EnvironmentReport = startRes.Report
		if startRes.Endpoint != nil {
			sc.Endpoint = startRes.Endpoint
		}
		if startRes.Artifacts != nil {
			sc.Creds = startRes.Artifacts.Creds
		}
		_, _ = WriteSidecar(p.Checkout, sc)
		return fmt.Errorf("playtestrun: environment failed: %w", startErr)
	}

	runID := startRes.RunID
	bridge := BridgeDirPath(p.Checkout, runID)
	credsPath := ""
	if startRes.Artifacts != nil {
		credsPath = startRes.Artifacts.Creds
	}

	failPostStart := func(cause error) error {
		stopCtx := ctx
		if stopCtx.Err() != nil {
			stopCtx = context.Background()
		}
		_, _ = p.Env.Stop(stopCtx, playtestenv.RunOptions{Checkout: p.Checkout, RunID: runID})
		_ = os.MkdirAll(bridge, 0o755)
		sc := baseSidecar(p, binding, wallClock, runID, commit, dirty, bridge)
		sc.Status = StatusEnvironmentFailed
		sc.Endpoint = startRes.Endpoint
		sc.Creds = credsPath
		sc.EnvironmentReport = startRes.Report
		_, _ = WriteSidecar(p.Checkout, sc)
		return cause
	}

	if err := os.MkdirAll(bridge, 0o755); err != nil {
		return failPostStart(fmt.Errorf("playtestrun: mkdir bridge: %w", err))
	}

	if !binding.CreationFlow {
		if credsPath == "" {
			return failPostStart(fmt.Errorf("playtestrun: profile run missing creds artifact"))
		}
		if _, _, err := SelectCredsPlayer(credsPath, binding.Profile); err != nil {
			return failPostStart(fmt.Errorf("playtestrun: creds profile match: %w", err))
		}
	} else if credsPath != "" {
		return failPostStart(fmt.Errorf("playtestrun: creation_flow must not produce creds"))
	}

	started := p.Clock.Now()
	deadline := started.Add(wallClock)
	sc := baseSidecar(p, binding, wallClock, runID, commit, dirty, bridge)
	sc.StartedAt = started
	sc.DeadlineAt = deadline
	sc.Status = StatusReady
	sc.Endpoint = startRes.Endpoint
	sc.Creds = credsPath
	sidecarPath, err := WriteSidecar(p.Checkout, sc)
	if err != nil {
		return failPostStart(err)
	}

	ready := ReadyPayload{
		Endpoint:   startRes.Endpoint,
		RunID:      runID,
		Checkout:   p.Checkout,
		Commit:     commit,
		Dirty:      dirty,
		DeadlineAt: deadline,
		Sidecar:    sidecarPath,
		BridgeDir:  bridge,
	}
	if credsPath != "" {
		creds := credsPath
		ready.Creds = &creds
	}
	if err := json.NewEncoder(p.Stdout).Encode(ready); err != nil {
		return failPostStart(fmt.Errorf("playtestrun: write ready JSON: %w", err))
	}

	stopPath := filepath.Join(bridge, "stop")
	reason, waitErr := waitForDeadlineOrStop(ctx, p.Clock, deadline, stopPath, p.StopPollInterval)
	if waitErr != nil && ctx.Err() == nil {
		_, _ = p.Env.Stop(context.Background(), playtestenv.RunOptions{Checkout: p.Checkout, RunID: runID})
		return waitErr
	}

	stopCtx := ctx
	if ctx.Err() != nil {
		stopCtx = context.Background()
	}
	_, stopErr := p.Env.Stop(stopCtx, playtestenv.RunOptions{Checkout: p.Checkout, RunID: runID})

	finalStatus := StatusStopped
	var retErr error
	switch reason {
	case "deadline":
		finalStatus = StatusIncompleteWallclock
		retErr = fmt.Errorf("playtestrun: wall-clock budget exceeded (%s)", wallClock)
	case "cancel":
		finalStatus = StatusInterrupted
		retErr = fmt.Errorf("playtestrun: interrupted: %w", waitErr)
	}
	_ = UpdateSidecarStatus(p.Checkout, runID, finalStatus)
	if stopErr != nil && retErr == nil {
		return fmt.Errorf("playtestrun: stop: %w", stopErr)
	}
	return retErr
}

func validateRunParams(p RunParams) error {
	if strings.TrimSpace(p.Checkout) == "" {
		return fmt.Errorf("playtestrun: --checkout is required")
	}
	if strings.TrimSpace(p.GoalsPath) == "" {
		return fmt.Errorf("playtestrun: --goals is required")
	}
	if strings.TrimSpace(p.Personality) == "" {
		return fmt.Errorf("playtestrun: --personality is required")
	}
	if p.Env == nil {
		return fmt.Errorf("playtestrun: env supervisor is required")
	}
	return nil
}

func baseSidecar(p RunParams, binding EphemeralBinding, wallClock time.Duration, runID, commit string, dirty bool, bridge string) SessionSidecar {
	sc := SessionSidecar{
		RunID:       runID,
		Checkout:    p.Checkout,
		Commit:      commit,
		Dirty:       dirty,
		GoalsPath:   p.GoalsPath,
		Personality: p.Personality,
		Budgets:     SessionBudgets{WallClock: wallClock.String()},
		BridgeDir:   bridge,
		Status:      StatusStarting,
	}
	if binding.CreationFlow {
		sc.CreationFlow = true
		sc.CreationRationale = binding.CreationRationale
	} else {
		sc.Profile = binding.Profile
	}
	return sc
}

func resolveGitMeta(p RunParams) (commit string, dirty bool) {
	if p.Dirty != nil {
		dirty = *p.Dirty
	}
	if p.Commit != "" || p.Dirty != nil {
		return p.Commit, dirty
	}
	return probeGit(p.Checkout)
}

func probeGit(checkout string) (commit string, dirty bool) {
	out, err := exec.Command("git", "-C", checkout, "rev-parse", "HEAD").Output()
	if err == nil {
		commit = strings.TrimSpace(string(out))
	}
	st, err := exec.Command("git", "-C", checkout, "status", "--porcelain").Output()
	if err == nil {
		dirty = strings.TrimSpace(string(st)) != ""
	}
	return commit, dirty
}

func waitForDeadlineOrStop(ctx context.Context, clock Clock, deadline time.Time, stopPath string, poll time.Duration) (reason string, err error) {
	for {
		if ctx.Err() != nil {
			return "cancel", ctx.Err()
		}
		if fileExists(stopPath) {
			return "stop", nil
		}
		now := clock.Now()
		if !now.Before(deadline) {
			return "deadline", nil
		}
		remaining := deadline.Sub(now)
		sleep := poll
		if remaining < sleep {
			sleep = remaining
		}
		if sleep <= 0 {
			return "deadline", nil
		}
		select {
		case <-ctx.Done():
			return "cancel", ctx.Err()
		case <-clock.After(sleep):
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// WriteStopSignal creates stop signal files (idempotent). Writes the run-level
// stop (scenario + shared) and the single-agent bridge stop for compatibility.
func WriteStopSignal(checkout, runID string) error {
	runDir := RunDir(checkout, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fmt.Errorf("playtestrun: mkdir run dir: %w", err)
	}
	if err := writeStopFile(StopSignalPath(checkout, runID)); err != nil {
		return err
	}
	bridge := BridgeDirPath(checkout, runID)
	if err := os.MkdirAll(bridge, 0o755); err != nil {
		return fmt.Errorf("playtestrun: mkdir bridge: %w", err)
	}
	return writeStopFile(filepath.Join(bridge, "stop"))
}

func writeStopFile(path string) error {
	if fileExists(path) {
		return nil
	}
	if err := os.WriteFile(path, []byte("stop\n"), 0o644); err != nil {
		return fmt.Errorf("playtestrun: write stop signal: %w", err)
	}
	return nil
}

// MarkScenarioAbort is a driver-contract helper: set scenario status to
// incomplete_abort and peer actors (except stoppedActorID) to aborted_peer.
// Go wall-clock/stop paths do not call this; abort is driver-initiated.
func MarkScenarioAbort(checkout, runID, stoppedActorID string) error {
	sc, err := ReadSidecar(checkout, runID)
	if err != nil {
		return err
	}
	sc.Status = StatusIncompleteAbort
	for i := range sc.Actors {
		if sc.Actors[i].ID == stoppedActorID {
			if sc.Actors[i].Status == ActorStatusReady || sc.Actors[i].Status == ActorStatusPending {
				sc.Actors[i].Status = ActorStatusFailed
			}
			continue
		}
		if sc.Actors[i].Status != ActorStatusStopped && sc.Actors[i].Status != ActorStatusFailed {
			sc.Actors[i].Status = ActorStatusAbortedPeer
		}
	}
	_, err = WriteSidecar(checkout, sc)
	return err
}
