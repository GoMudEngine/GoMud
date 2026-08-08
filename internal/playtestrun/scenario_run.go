package playtestrun

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/playtestenv"
)

// ScenarioParams configures a blocking multi-agent playtestrun scenario.
type ScenarioParams struct {
	Checkout          string
	ScenarioPath      string
	WallClockOverride time.Duration // 0 = use scenario file
	Force             bool          // MaxAIConnections size bypass only
	Env               envSupervisor
	Clock             Clock
	Stdout            io.Writer
	Stderr            io.Writer
	Commit            string
	Dirty             *bool
	StopPollInterval  time.Duration
	PlaytestRoot      string // empty ⇒ <checkout>/tools/playtest
	MaxAIConnections  int    // 0 ⇒ probe checkout config
}

// ScenarioReadyActor is one actor entry in the scenario ready JSON line.
type ScenarioReadyActor struct {
	ID           string  `json:"id"`
	Personality  string  `json:"personality"`
	GoalsPath    string  `json:"goals_path"`
	BridgeDir    string  `json:"bridge_dir"`
	Creds        *string `json:"creds"`
	Username     string  `json:"username,omitempty"`
	Profile      *string `json:"profile"`
	CreationFlow bool    `json:"creation_flow"`
	Status       string  `json:"status"`
}

// ScenarioReadyPayload is the one-line JSON printed when a scenario is ready.
type ScenarioReadyPayload struct {
	Endpoint      *playtestenv.Endpoint `json:"endpoint"`
	RunID         string                `json:"run_id"`
	Checkout      string                `json:"checkout"`
	Commit        string                `json:"commit"`
	Dirty         bool                  `json:"dirty"`
	DeadlineAt    time.Time             `json:"deadline_at"`
	Sidecar       string                `json:"sidecar"`
	BlackboardDir string                `json:"blackboard_dir"`
	OnActorStop   string                `json:"on_actor_stop"`
	Actors        []ScenarioReadyActor  `json:"actors"`
}

// RunScenario starts one shared playtestenv, materializes roster profiles with
// actor_id stamps, creates per-actor bridges + blackboard, watches wall-clock /
// stop, then stops the env.
func RunScenario(ctx context.Context, p ScenarioParams) error {
	if err := validateScenarioParams(p); err != nil {
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

	playtestRoot := p.PlaytestRoot
	if strings.TrimSpace(playtestRoot) == "" {
		playtestRoot = filepath.Join(p.Checkout, "tools", "playtest")
	}

	scenario, err := ParseScenario(p.ScenarioPath, playtestRoot, ScenarioParseOpts{
		Force:            p.Force,
		MaxAIConnections: p.MaxAIConnections,
		Checkout:         p.Checkout,
	})
	if err != nil {
		return err
	}

	wallClock := scenario.WallClock
	if p.WallClockOverride > 0 {
		wallClock = p.WallClockOverride
	}
	lease := wallClock + leaseBuffer

	var profiles []playtestenv.ProfileRequest
	for _, actor := range scenario.Roster {
		if actor.Binding.CreationFlow {
			continue
		}
		profiles = append(profiles, playtestenv.ProfileRequest{
			Profile:   actor.Binding.Profile,
			StartRoom: actor.Binding.StartRoom,
			ActorID:   actor.ID,
			Overlays:  actor.Binding.Overlays,
		})
	}

	startRes, startErr := p.Env.Start(ctx, playtestenv.StartOptions{
		Checkout: p.Checkout,
		Lease:    lease,
		Profiles: profiles,
	})
	commit, dirty := resolveScenarioGitMeta(p)

	if startErr != nil {
		runID := startRes.RunID
		if runID == "" {
			runID = "failed-" + fmt.Sprintf("%d", p.Clock.Now().UnixNano())
		}
		_ = prepareScenarioDirs(p.Checkout, runID, scenario)
		sc := baseScenarioSidecar(p, scenario, wallClock, runID, commit, dirty, "")
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
		_ = prepareScenarioDirs(p.Checkout, runID, scenario)
		sc := baseScenarioSidecar(p, scenario, wallClock, runID, commit, dirty, credsPath)
		sc.Status = StatusEnvironmentFailed
		sc.Endpoint = startRes.Endpoint
		sc.Creds = credsPath
		sc.EnvironmentReport = startRes.Report
		_, _ = WriteSidecar(p.Checkout, sc)
		return cause
	}

	if err := prepareScenarioDirs(p.Checkout, runID, scenario); err != nil {
		return failPostStart(err)
	}

	actors, err := buildScenarioActors(p.Checkout, runID, scenario, credsPath)
	if err != nil {
		return failPostStart(err)
	}

	started := p.Clock.Now()
	deadline := started.Add(wallClock)
	sc := baseScenarioSidecar(p, scenario, wallClock, runID, commit, dirty, credsPath)
	sc.StartedAt = started
	sc.DeadlineAt = deadline
	sc.Status = StatusReady
	sc.Endpoint = startRes.Endpoint
	sc.Actors = actors
	sidecarPath, err := WriteSidecar(p.Checkout, sc)
	if err != nil {
		return failPostStart(err)
	}

	readyActors := make([]ScenarioReadyActor, 0, len(actors))
	for _, a := range actors {
		ra := ScenarioReadyActor{
			ID:           a.ID,
			Personality:  a.Personality,
			GoalsPath:    a.GoalsPath,
			BridgeDir:    a.BridgeDir,
			Creds:        a.Creds,
			Username:     a.Username,
			CreationFlow: a.CreationFlow,
			Status:       a.Status,
		}
		if a.Profile != "" {
			prof := a.Profile
			ra.Profile = &prof
		}
		readyActors = append(readyActors, ra)
	}
	ready := ScenarioReadyPayload{
		Endpoint:      startRes.Endpoint,
		RunID:         runID,
		Checkout:      p.Checkout,
		Commit:        commit,
		Dirty:         dirty,
		DeadlineAt:    deadline,
		Sidecar:       sidecarPath,
		BlackboardDir: BlackboardDirPath(p.Checkout, runID),
		OnActorStop:   scenario.OnActorStop,
		Actors:        readyActors,
	}
	if err := json.NewEncoder(p.Stdout).Encode(ready); err != nil {
		return failPostStart(fmt.Errorf("playtestrun: write ready JSON: %w", err))
	}

	stopPath := StopSignalPath(p.Checkout, runID)
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

func validateScenarioParams(p ScenarioParams) error {
	if strings.TrimSpace(p.Checkout) == "" {
		return fmt.Errorf("playtestrun: --checkout is required")
	}
	if strings.TrimSpace(p.ScenarioPath) == "" {
		return fmt.Errorf("playtestrun: --scenario is required")
	}
	if p.Env == nil {
		return fmt.Errorf("playtestrun: env supervisor is required")
	}
	return nil
}

func prepareScenarioDirs(checkout, runID string, scenario ScenarioFile) error {
	bb := BlackboardDirPath(checkout, runID)
	if err := os.MkdirAll(bb, 0o755); err != nil {
		return fmt.Errorf("playtestrun: mkdir blackboard: %w", err)
	}
	for _, actor := range scenario.Roster {
		bridge := ActorBridgeDirPath(checkout, runID, actor.ID)
		if err := os.MkdirAll(bridge, 0o755); err != nil {
			return fmt.Errorf("playtestrun: mkdir actor bridge %s: %w", actor.ID, err)
		}
	}
	return nil
}

func buildScenarioActors(checkout, runID string, scenario ScenarioFile, credsPath string) ([]ActorSidecar, error) {
	out := make([]ActorSidecar, 0, len(scenario.Roster))
	for _, actor := range scenario.Roster {
		bridge := ActorBridgeDirPath(checkout, runID, actor.ID)
		entry := ActorSidecar{
			ID:           actor.ID,
			Personality:  actor.Personality,
			GoalsPath:    actor.GoalsPath,
			BridgeDir:    bridge,
			CreationFlow: actor.Binding.CreationFlow,
			Status:       ActorStatusReady,
		}
		if actor.Binding.CreationFlow {
			entry.Creds = nil
			out = append(out, entry)
			continue
		}
		if credsPath == "" {
			return nil, fmt.Errorf("playtestrun: profile actor %q missing creds artifact", actor.ID)
		}
		user, _, err := SelectCredsByActorID(credsPath, actor.ID)
		if err != nil {
			return nil, fmt.Errorf("playtestrun: actor %q creds: %w", actor.ID, err)
		}
		cp := credsPath
		entry.Creds = &cp
		entry.Username = user
		entry.Profile = actor.Binding.Profile
		out = append(out, entry)
	}
	return out, nil
}

func baseScenarioSidecar(p ScenarioParams, scenario ScenarioFile, wallClock time.Duration, runID, commit string, dirty bool, credsPath string) SessionSidecar {
	actors := make([]ActorSidecar, 0, len(scenario.Roster))
	for _, actor := range scenario.Roster {
		a := ActorSidecar{
			ID:           actor.ID,
			Personality:  actor.Personality,
			GoalsPath:    actor.GoalsPath,
			BridgeDir:    ActorBridgeDirPath(p.Checkout, runID, actor.ID),
			CreationFlow: actor.Binding.CreationFlow,
			Status:       ActorStatusPending,
			Profile:      actor.Binding.Profile,
		}
		if !actor.Binding.CreationFlow && credsPath != "" {
			cp := credsPath
			a.Creds = &cp
		}
		actors = append(actors, a)
	}
	return SessionSidecar{
		RunID:         runID,
		Checkout:      p.Checkout,
		Commit:        commit,
		Dirty:         dirty,
		ScenarioPath:  scenario.Path,
		OnActorStop:   scenario.OnActorStop,
		BlackboardDir: BlackboardDirPath(p.Checkout, runID),
		Creds:         credsPath,
		Budgets:       SessionBudgets{WallClock: wallClock.String()},
		Status:        StatusStarting,
		Actors:        actors,
	}
}

func resolveScenarioGitMeta(p ScenarioParams) (commit string, dirty bool) {
	rp := RunParams{Checkout: p.Checkout, Commit: p.Commit, Dirty: p.Dirty}
	return resolveGitMeta(rp)
}
