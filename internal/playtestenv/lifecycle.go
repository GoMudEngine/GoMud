package playtestenv

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// DefaultLease is the lease granted to a newly started run.
	DefaultLease = 2 * time.Hour
	// DefaultReadinessTimeout bounds compound readiness after compose up.
	DefaultReadinessTimeout = 90 * time.Second
	// CleanupTimeout bounds evidence capture and resource cleanup after
	// cancellation or failure.
	CleanupTimeout = 45 * time.Second

	serviceName     = "server"
	buildLogName    = "build.log"
	serverLogName   = "server.log"
	inspectLogName  = "inspect.json"
	controlDirName  = "control"
	imageNamePrefix = "dogmud-playtest:"
)

// supervisorDeps holds injectable dependencies for unit tests. Production
// New() wires real OS/Docker/clock implementations.
type supervisorDeps struct {
	runner        Runner
	now           func() time.Time
	genID         func() (string, error)
	dial          dialFunc
	after         func(time.Duration) <-chan time.Time
	resolveDocker func(context.Context, Runner) (dockerContext, error)
	acquireLock   lockAcquireFunc
	lockWait      time.Duration
	onEvent       func(string)
	// diagnostics receives teed build/subprocess output (defaults to os.Stderr).
	diagnostics io.Writer
}

// Supervisor orchestrates ephemeral local playtest server runs.
type Supervisor struct {
	deps supervisorDeps
}

// New returns a Supervisor with production dependencies.
func New() *Supervisor {
	return newSupervisor(supervisorDeps{})
}

func newSupervisor(deps supervisorDeps) *Supervisor {
	if deps.runner == nil {
		deps.runner = execRunner{}
	}
	if deps.now == nil {
		deps.now = time.Now
	}
	if deps.genID == nil {
		deps.genID = generateRunID
	}
	if deps.dial == nil {
		deps.dial = defaultDial
	}
	if deps.after == nil {
		deps.after = time.After
	}
	if deps.resolveDocker == nil {
		deps.resolveDocker = resolveLocalDockerContextForHost
	}
	if deps.acquireLock == nil {
		deps.acquireLock = acquireRunLock
	}
	if deps.lockWait <= 0 {
		deps.lockWait = DefaultLockWait
	}
	if deps.diagnostics == nil {
		deps.diagnostics = os.Stderr
	}
	return &Supervisor{deps: deps}
}

func (s *Supervisor) event(name string) {
	if s.deps.onEvent != nil {
		s.deps.onEvent(name)
	}
}

// Start validates a checkout, reserves a run, builds and starts the ephemeral
// server, and returns only after readiness succeeds or failure cleanup finishes.
func (s *Supervisor) Start(ctx context.Context, opts StartOptions) (Result, error) {
	res := Result{Operation: "start"}
	lease := opts.Lease
	if lease <= 0 {
		lease = DefaultLease
	}
	readinessTimeout := opts.ReadinessTimeout
	if readinessTimeout <= 0 {
		readinessTimeout = DefaultReadinessTimeout
	}

	s.event("validate checkout")
	checkout, err := validateCheckoutForHost(ctx, s.deps.runner, opts.Checkout)
	if err != nil {
		res.Failure = &FailureRecord{
			Category: FailureInvalidCheckout,
			Phase:    StateValidating,
			Summary:  err.Error(),
		}
		return res, err
	}

	resv, err := reserveRunWithDeps(ctx, checkout.Path, lease, s.deps.lockWait, s.deps.now, s.deps.genID, s.deps.acquireLock, writeManifest)
	if err != nil {
		if errors.Is(err, ErrLockBusy) {
			res.Failure = &FailureRecord{
				Category:  FailureLockBusy,
				Phase:     StateValidating,
				Summary:   err.Error(),
				Retryable: true,
			}
			return res, err
		}
		res.Failure = &FailureRecord{
			Category: FailureManifest,
			Phase:    StateValidating,
			Summary:  err.Error(),
		}
		return res, err
	}
	s.event("reserve and lock run")

	lock := resv.Lock
	defer func() { _ = lock.Close(); s.event("unlock") }()

	m := resv.Manifest
	runDir := resv.RunDir
	res.RunID = resv.RunID
	res.Project = resv.Project
	res.Manifest = filepath.Join(runDir, manifestFileName)
	res.State = m.State

	controlDir := filepath.Join(runDir, controlDirName)
	if err := os.MkdirAll(controlDir, 0o755); err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dockerContext{}, composeRunVars{}, "", StateValidating, FailureManifest, err, false)
	}

	baseline, err := collectGitBaseline(ctx, s.deps.runner, checkout.Path)
	if err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dockerContext{}, composeRunVars{}, "", StateValidating, FailureInvalidCheckout, err, false)
	}

	now := s.deps.now()
	m.Checkout = checkout.Path
	m.CheckoutFingerprint = checkout.Fingerprint
	m.Git = baseline
	m.Image = imageNamePrefix + resv.RunID
	m.Service = serviceName
	m.Network = resv.Project + "_default"
	m.Volume = resv.Project + "_data"
	m.Artifacts = ArtifactPaths{
		Manifest:  filepath.Join(runDir, manifestFileName),
		BuildLog:  filepath.Join(runDir, buildLogName),
		ServerLog: filepath.Join(runDir, serverLogName),
		Inspect:   filepath.Join(runDir, inspectLogName),
		Compose:   filepath.Join(runDir, composeResolvedFileName),
		Config:    filepath.Join(controlDir, configOverridesFileName),
	}
	m.UpdatedAt = now
	if err := writeManifest(m.Artifacts.Manifest, m); err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dockerContext{}, composeRunVars{}, "", StateValidating, FailureManifest, err, false)
	}
	s.event("write validating manifest")
	populateResultArtifacts(&res, m)

	dc, err := s.deps.resolveDocker(ctx, s.deps.runner)
	if err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dockerContext{}, composeRunVars{}, "", StateValidating, FailureDockerUnavailable, err, true)
	}
	s.event("local Docker preflight")

	composePath, configPath, err := materializeRunFiles(runDir, controlDir, checkout.Version)
	if err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dc, composeRunVars{}, "", StateValidating, FailureManifest, err, true)
	}
	m.Artifacts.Compose = composePath
	m.Artifacts.Config = configPath
	if err := writeManifest(m.Artifacts.Manifest, m); err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dc, composeRunVars{}, "", StateValidating, FailureManifest, err, true)
	}
	s.event("write control files")
	populateResultArtifacts(&res, m)

	vars := composeRunVars{
		RunID:               resv.RunID,
		Project:             resv.Project,
		Checkout:            checkout.Path,
		CheckoutFingerprint: checkout.Fingerprint,
		CreatedAt:           m.CreatedAt,
		ControlDir:          controlDir,
	}

	if err := transitionManifest(m, StateBuilding, s.deps.now()); err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dc, vars, composePath, StateValidating, FailureManifest, err, true)
	}
	if err := writeManifest(m.Artifacts.Manifest, m); err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dc, vars, composePath, StateBuilding, FailureManifest, err, true)
	}
	s.event("transition building")
	res.State = m.State

	buildLog, err := os.Create(m.Artifacts.BuildLog)
	if err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dc, vars, composePath, StateBuilding, FailureManifest, err, true)
	}
	// Tee build output to the run artifact and terminal diagnostics concurrently.
	buildOut := io.MultiWriter(buildLog, s.deps.diagnostics)
	buildErr := io.MultiWriter(buildLog, s.deps.diagnostics)
	buildSpec := composeBuildCommand(dc, vars, composePath, runDir, buildOut, buildErr)
	s.event("compose build server")
	buildRunErr := s.deps.runner.Run(ctx, buildSpec)
	_ = buildLog.Close()
	if buildRunErr != nil {
		return s.failStart(ctx, &res, m, runDir, "", dc, vars, composePath, StateBuilding, FailureBuild, buildRunErr, true)
	}

	if err := transitionManifest(m, StateStarting, s.deps.now()); err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dc, vars, composePath, StateBuilding, FailureManifest, err, true)
	}
	if err := writeManifest(m.Artifacts.Manifest, m); err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dc, vars, composePath, StateStarting, FailureManifest, err, true)
	}
	s.event("transition starting")
	res.State = m.State

	upSpec := composeUpNoBuildServerCommand(dc, vars, composePath, runDir, io.Discard, io.Discard)
	s.event("compose up -d --no-build server")
	if err := s.deps.runner.Run(ctx, upSpec); err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dc, vars, composePath, StateStarting, FailureBuild, err, true)
	}

	s.event("resolve container")
	containerID, err := resolveComposeContainerID(ctx, s.deps.runner, dc, vars, composePath, runDir)
	if err != nil {
		return s.failStart(ctx, &res, m, runDir, "", dc, vars, composePath, StateStarting, FailureContainerExited, err, true)
	}
	m.ContainerID = containerID
	if err := writeManifest(m.Artifacts.Manifest, m); err != nil {
		return s.failStart(ctx, &res, m, runDir, containerID, dc, vars, composePath, StateStarting, FailureManifest, err, true)
	}

	s.event("readiness")
	obs, cat, err := waitForReadiness(ctx, s.deps.runner, dc, containerID, readinessTimeout, s.deps.dial, s.deps.now, s.deps.after)
	m.Readiness = obs
	if err != nil {
		if cat == "" {
			cat = FailureReadinessTimeout
		}
		return s.failStart(ctx, &res, m, runDir, containerID, dc, vars, composePath, StateStarting, cat, err, true)
	}

	m.Endpoint = obs.Endpoint
	if err := transitionManifest(m, StateReady, s.deps.now()); err != nil {
		return s.failStart(ctx, &res, m, runDir, containerID, dc, vars, composePath, StateStarting, FailureManifest, err, true)
	}
	if err := writeManifest(m.Artifacts.Manifest, m); err != nil {
		return s.failStart(ctx, &res, m, runDir, containerID, dc, vars, composePath, StateReady, FailureManifest, err, true)
	}
	s.event("transition ready")

	res.State = StateReady
	res.Endpoint = cloneEndpoint(m.Endpoint)
	res.ServerLog = m.Artifacts.ServerLog
	populateResultArtifacts(&res, m)
	return res, nil
}

func populateResultArtifacts(res *Result, m *Manifest) {
	arts := m.Artifacts
	res.Artifacts = &arts
	res.Manifest = arts.Manifest
	res.ServerLog = arts.ServerLog
	if arts.Report != "" {
		res.Report = arts.Report
	}
}

func (s *Supervisor) failStart(
	callerCtx context.Context,
	res *Result,
	m *Manifest,
	runDir string,
	containerID string,
	dc dockerContext,
	vars composeRunVars,
	composePath string,
	phase State,
	category FailureCategory,
	cause error,
	doCleanup bool,
) (Result, error) {
	if category == FailureLockBusy {
		res.Failure = &FailureRecord{Category: category, Phase: phase, Summary: cause.Error(), Retryable: true}
		return *res, cause
	}

	summary := cause.Error()
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		summary = cause.Error()
	}

	// Transition to failed when still nonterminal.
	if m.State != StateFailed && m.State != StateStopped {
		_ = transitionManifest(m, StateFailed, s.deps.now())
	}
	m.Failure = &FailureRecord{
		Category:  category,
		Phase:     phase,
		Summary:   summary,
		Retryable: category == FailureLockBusy,
	}

	var cleanup *CleanupResult
	if doCleanup {
		cleanupBase := context.WithoutCancel(callerCtx)
		cleanupCtx, cancelCleanup := context.WithTimeout(cleanupBase, CleanupTimeout)
		defer cancelCleanup()
		cleanup = s.cleanupFailedRun(cleanupCtx, m, runDir, containerID, dc, vars, composePath)
		m.Cleanup = cleanup
	}

	reportPath, reportErr := writeFailureReport(m.Checkout, m, cleanup, s.deps.now())
	if reportErr == nil {
		m.Artifacts.Report = reportPath
		res.Report = reportPath
	}
	_ = writeManifest(m.Artifacts.Manifest, m)

	res.RunID = m.RunID
	res.Project = m.Project
	res.State = StateFailed
	res.Failure = m.Failure
	res.Cleanup = cleanup
	populateResultArtifacts(res, m)

	// Preserve the original failure even when cleanup/report succeed.
	return *res, cause
}

func (s *Supervisor) cleanupFailedRun(
	ctx context.Context,
	m *Manifest,
	runDir string,
	containerID string,
	dc dockerContext,
	vars composeRunVars,
	composePath string,
) *CleanupResult {
	result := &CleanupResult{Complete: true, Summary: "resources removed"}
	if dc.name == "" {
		// No validated docker context - nothing Docker-side to remove.
		result.Summary = "no docker context; skipped resource cleanup"
		return result
	}
	if vars.Project == "" {
		vars = composeRunVars{
			RunID:               m.RunID,
			Project:             m.Project,
			Checkout:            m.Checkout,
			CheckoutFingerprint: m.CheckoutFingerprint,
			CreatedAt:           m.CreatedAt,
			ControlDir:          filepath.Join(runDir, controlDirName),
		}
	}
	if composePath == "" {
		composePath = m.Artifacts.Compose
	}
	if containerID == "" {
		containerID = m.ContainerID
	}

	// Capture evidence before destructive cleanup.
	if containerID != "" {
		if err := captureServerLogs(ctx, s.deps.runner, dc, containerID, m.Artifacts.ServerLog); err != nil {
			result.Complete = false
			result.Summary = "log capture failed: " + err.Error()
		}
		if err := captureInspectEvidence(ctx, s.deps.runner, dc, containerID, m.Artifacts.Inspect); err != nil {
			result.Complete = false
			if result.Summary == "resources removed" {
				result.Summary = "inspect capture failed: " + err.Error()
			}
		}
	}

	downCleanup := s.removeComposeAndImage(ctx, m, runDir, dc, vars, composePath)
	mergeCleanup(result, downCleanup)

	// Remove control/ and compose.resolved.yml only after complete resource
	// cleanup so a later stop can resume using the same Compose file when
	// leftovers remain.
	if result.Complete {
		if err := removeControlArtifacts(runDir, m); err != nil {
			result.Complete = false
			result.Leftovers = append(result.Leftovers, ResourceRef{Kind: "host-path", ID: err.Error()})
			result.Summary = "control artifact removal failed: " + err.Error()
		}
	}

	if result.Complete && result.Summary == "resources removed" {
		result.Summary = "resources removed"
	}
	return result
}

func captureServerLogs(ctx context.Context, runner Runner, dc dockerContext, containerID, path string) error {
	if path == "" || containerID == "" {
		return nil
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	spec := dockerCommand(dc, []string{"logs", containerID}, "", f, f)
	return runner.Run(ctx, spec)
}

func captureInspectEvidence(ctx context.Context, runner Runner, dc dockerContext, containerID, path string) error {
	if path == "" || containerID == "" {
		return nil
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	spec := dockerCommand(dc, []string{"inspect", containerID}, "", f, io.Discard)
	return runner.Run(ctx, spec)
}

func isBenignImageMissing(err error, stderr string) bool {
	msg := strings.ToLower(err.Error() + " " + stderr)
	return strings.Contains(msg, "no such image") || strings.Contains(msg, "not found")
}

func generateRunID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func defaultDial(ctx context.Context, network, address string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, network, address)
}

func cloneEndpoint(ep *Endpoint) *Endpoint {
	if ep == nil {
		return nil
	}
	cp := *ep
	return &cp
}
