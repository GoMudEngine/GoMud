package playtestenv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/natefinch/atomic"
)

const (
	labelManaged   = "dogmud.playtest.managed"
	labelRunID     = "dogmud.playtest.run-id"
	labelProject   = "dogmud.playtest.project"
	labelCheckout  = "dogmud.playtest.checkout"
	labelSchema    = "dogmud.playtest.schema"
	labelCreatedAt = "dogmud.playtest.created-at"

	labelManagedValue = "true"
	labelSchemaValue  = "1"

	stopGracePeriod = 10 * time.Second
	stopGracePoll   = 200 * time.Millisecond
)

// ErrResourceIdentityMismatch reports that live Docker resources disagree with
// the run manifest's immutable identity labels, or expected resources are
// missing when agreement is required.
var ErrResourceIdentityMismatch = errors.New("playtestenv: live resource identity does not match manifest")

// ErrLeaseExpired is returned when renew is asked to extend an expired lease.
var ErrLeaseExpired = errors.New("playtestenv: run lease has expired")

// ErrRunNotReady is returned when renew targets a run that is not ready.
var ErrRunNotReady = errors.New("playtestenv: run is not ready")

// ErrInvalidRunID is returned when a caller-supplied run ID fails validation.
var ErrInvalidRunID = errors.New("playtestenv: invalid run id")

type liveResource struct {
	Kind    string
	ID      string
	Found   bool
	Running bool
	Labels  map[string]string
}

type identityReport struct {
	Resources  []liveResource
	Missing    []string
	Mismatched []string
	AnyFound   bool
}

func (r identityReport) Agrees() bool {
	return len(r.Missing) == 0 && len(r.Mismatched) == 0
}

func expectedIdentityLabels(m *Manifest) map[string]string {
	return map[string]string{
		labelManaged:   labelManagedValue,
		labelRunID:     m.RunID,
		labelProject:   m.Project,
		labelCheckout:  m.CheckoutFingerprint,
		labelSchema:    labelSchemaValue,
		labelCreatedAt: m.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type preparedOp struct {
	checkout checkoutValidation
	dc       dockerContext
	runID    string
	runDir   string
	lock     *runLock
	manifest *Manifest
	identity identityReport
}

func (p *preparedOp) close() {
	if p != nil && p.lock != nil {
		_ = p.lock.Close()
		p.lock = nil
	}
}

// Status reports manifest state reconciled against live Docker resources.
func (s *Supervisor) Status(ctx context.Context, opts RunOptions) (Result, error) {
	res := Result{Operation: "status"}
	prep, err := s.prepareOp(ctx, opts.Checkout, opts.RunID, &res)
	if err != nil {
		return res, err
	}
	defer prep.close()

	res.RunID = prep.manifest.RunID
	res.Project = prep.manifest.Project
	res.State = prep.manifest.State
	res.Endpoint = cloneEndpoint(prep.manifest.Endpoint)
	populateResultArtifacts(&res, prep.manifest)

	if !prep.identity.Agrees() {
		summary := formatIdentityDisagreement(prep.identity)
		res.Failure = &FailureRecord{
			Category: FailureIdentityMismatch,
			Phase:    prep.manifest.State,
			Summary:  summary,
		}
		return res, fmt.Errorf("%w: %s", ErrResourceIdentityMismatch, summary)
	}
	return res, nil
}

// Logs refreshes or follows container logs for a run.
func (s *Supervisor) Logs(ctx context.Context, opts LogsOptions) (Result, error) {
	res := Result{Operation: "logs"}
	prep, err := s.prepareOp(ctx, opts.Checkout, opts.RunID, &res)
	if err != nil {
		return res, err
	}
	defer prep.close()

	res.RunID = prep.manifest.RunID
	res.Project = prep.manifest.Project
	res.State = prep.manifest.State
	populateResultArtifacts(&res, prep.manifest)

	if !prep.identity.Agrees() {
		summary := formatIdentityDisagreement(prep.identity)
		res.Failure = &FailureRecord{
			Category: FailureIdentityMismatch,
			Phase:    prep.manifest.State,
			Summary:  summary,
		}
		return res, fmt.Errorf("%w: %s", ErrResourceIdentityMismatch, summary)
	}
	if prep.manifest.ContainerID == "" {
		err := fmt.Errorf("playtestenv: run %s has no container id", prep.manifest.RunID)
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: prep.manifest.State, Summary: err.Error()}
		return res, err
	}

	logPath := prep.manifest.Artifacts.ServerLog
	if logPath == "" {
		logPath = filepath.Join(prep.runDir, serverLogName)
		prep.manifest.Artifacts.ServerLog = logPath
	}

	if opts.Follow {
		if err := s.followServerLogs(ctx, prep.dc, prep.manifest.ContainerID, logPath, opts.Output); err != nil {
			res.ServerLog = logPath
			populateResultArtifacts(&res, prep.manifest)
			return res, err
		}
		res.ServerLog = logPath
		populateResultArtifacts(&res, prep.manifest)
		return res, nil
	}

	if err := refreshServerLogAtomic(ctx, s.deps.runner, prep.dc, prep.manifest.ContainerID, logPath); err != nil {
		res.Failure = &FailureRecord{Category: FailureCleanup, Phase: prep.manifest.State, Summary: err.Error()}
		return res, err
	}
	res.ServerLog = logPath
	populateResultArtifacts(&res, prep.manifest)
	return res, nil
}

// Renew extends the lease of a ready, unexpired, unambiguous run.
func (s *Supervisor) Renew(ctx context.Context, opts RenewOptions) (Result, error) {
	res := Result{Operation: "renew"}
	prep, err := s.prepareOp(ctx, opts.Checkout, opts.RunID, &res)
	if err != nil {
		return res, err
	}
	defer prep.close()

	res.RunID = prep.manifest.RunID
	res.Project = prep.manifest.Project
	res.State = prep.manifest.State
	populateResultArtifacts(&res, prep.manifest)

	// Reread under lock for the lease decision.
	m, err := readManifest(prep.manifest.Artifacts.Manifest)
	if err != nil {
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: StateValidating, Summary: err.Error()}
		return res, err
	}
	prep.manifest = m
	res.State = m.State

	if m.State != StateReady {
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: m.State, Summary: ErrRunNotReady.Error()}
		return res, ErrRunNotReady
	}
	now := s.deps.now()
	if !now.Before(m.LeaseExpiresAt) {
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: m.State, Summary: ErrLeaseExpired.Error()}
		return res, ErrLeaseExpired
	}
	if !prep.identity.Agrees() {
		summary := formatIdentityDisagreement(prep.identity)
		res.Failure = &FailureRecord{
			Category: FailureIdentityMismatch,
			Phase:    m.State,
			Summary:  summary,
		}
		return res, fmt.Errorf("%w: %s", ErrResourceIdentityMismatch, summary)
	}

	lease := opts.Lease
	if lease <= 0 {
		lease = DefaultLease
	}
	m.LeaseExpiresAt = now.Add(lease)
	m.UpdatedAt = now
	if err := writeManifest(m.Artifacts.Manifest, m); err != nil {
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: m.State, Summary: err.Error()}
		return res, err
	}
	res.State = m.State
	res.Endpoint = cloneEndpoint(m.Endpoint)
	populateResultArtifacts(&res, m)
	return res, nil
}

// Stop stops a run, recovers abandoned preterminal runs, and is idempotent for
// already-cleaned terminal runs.
func (s *Supervisor) Stop(ctx context.Context, opts RunOptions) (Result, error) {
	res := Result{Operation: "stop"}
	prep, err := s.prepareOp(ctx, opts.Checkout, opts.RunID, &res)
	if err != nil {
		return res, err
	}
	defer prep.close()

	m := prep.manifest
	res.RunID = m.RunID
	res.Project = m.Project
	res.State = m.State
	populateResultArtifacts(&res, m)

	// Idempotent success: terminal + no live labelled resources.
	if (m.State == StateStopped || m.State == StateFailed) && !prep.identity.AnyFound {
		cleanup := m.Cleanup
		if cleanup == nil {
			cleanup = &CleanupResult{Complete: true, Summary: "no live resources remained"}
		} else if !cleanup.Complete {
			cleanup = &CleanupResult{Complete: true, Summary: "no live resources remained"}
		}
		m.Cleanup = cleanup
		_ = writeManifest(m.Artifacts.Manifest, m)
		res.State = m.State
		res.Cleanup = cleanup
		res.Failure = m.Failure
		populateResultArtifacts(&res, m)
		return res, nil
	}

	if prep.identity.AnyFound && !prep.identity.Agrees() {
		summary := formatIdentityDisagreement(prep.identity)
		res.Failure = &FailureRecord{
			Category: FailureIdentityMismatch,
			Phase:    m.State,
			Summary:  summary,
		}
		return res, fmt.Errorf("%w: %s", ErrResourceIdentityMismatch, summary)
	}

	switch m.State {
	case StateValidating, StateBuilding, StateStarting, StateStopping:
		return s.stopAbandoned(ctx, &res, prep)
	case StateFailed:
		return s.stopFailedResume(ctx, &res, prep)
	case StateReady:
		return s.stopReady(ctx, &res, prep)
	case StateStopped:
		// Live resources remain despite stopped manifest — clean them.
		return s.stopReady(ctx, &res, prep)
	default:
		err := fmt.Errorf("playtestenv: unsupported run state %q", m.State)
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: m.State, Summary: err.Error()}
		return res, err
	}
}

func (s *Supervisor) stopAbandoned(callerCtx context.Context, res *Result, prep *preparedOp) (Result, error) {
	m := prep.manifest
	phase := m.State
	// Cancel before any stop mutation leaves the run untouched.
	if err := callerCtx.Err(); err != nil {
		res.State = m.State
		res.Failure = m.Failure
		populateResultArtifacts(res, m)
		return *res, err
	}
	if m.State != StateFailed {
		if err := transitionManifest(m, StateFailed, s.deps.now()); err != nil {
			res.Failure = &FailureRecord{Category: FailureManifest, Phase: phase, Summary: err.Error()}
			return *res, err
		}
	}
	abandoned := &FailureRecord{
		Category: FailureAbandonedRun,
		Phase:    phase,
		Summary:  fmt.Sprintf("abandoned run recovered from state %s", phase),
	}
	m.Failure = abandoned
	_ = writeManifest(m.Artifacts.Manifest, m)

	// After mutation, finish cleanup even if the caller cancels.
	cleanupBase := context.WithoutCancel(callerCtx)
	cleanupCtx, cancel := context.WithTimeout(cleanupBase, CleanupTimeout)
	defer cancel()

	cleanup := s.cleanupFailedRun(cleanupCtx, m, prep.runDir, m.ContainerID, prep.dc, composeVarsFromManifest(m, prep.runDir), m.Artifacts.Compose)
	m.Cleanup = cleanup
	if !cleanup.Complete {
		m.Failure = &FailureRecord{
			Category: FailureCleanup,
			Phase:    StateFailed,
			Summary:  cleanup.Summary,
		}
	} else {
		m.Failure = abandoned
	}
	_ = writeManifest(m.Artifacts.Manifest, m)

	res.State = StateFailed
	res.Failure = m.Failure
	res.Cleanup = cleanup
	populateResultArtifacts(res, m)
	if !cleanup.Complete {
		return *res, fmt.Errorf("playtestenv: cleanup incomplete: %s", cleanup.Summary)
	}
	return *res, nil
}

func (s *Supervisor) stopFailedResume(callerCtx context.Context, res *Result, prep *preparedOp) (Result, error) {
	m := prep.manifest
	if err := callerCtx.Err(); err != nil {
		res.Failure = m.Failure
		return *res, err
	}
	cleanupBase := context.WithoutCancel(callerCtx)
	cleanupCtx, cancel := context.WithTimeout(cleanupBase, CleanupTimeout)
	defer cancel()

	cleanup := s.cleanupFailedRun(cleanupCtx, m, prep.runDir, m.ContainerID, prep.dc, composeVarsFromManifest(m, prep.runDir), m.Artifacts.Compose)
	m.Cleanup = cleanup
	// failed remains historical state
	if !cleanup.Complete {
		if m.Failure == nil {
			m.Failure = &FailureRecord{Category: FailureCleanup, Phase: StateFailed, Summary: cleanup.Summary}
		}
	}
	_ = writeManifest(m.Artifacts.Manifest, m)
	res.State = StateFailed
	res.Failure = m.Failure
	res.Cleanup = cleanup
	populateResultArtifacts(res, m)
	if !cleanup.Complete {
		return *res, fmt.Errorf("playtestenv: cleanup incomplete: %s", cleanup.Summary)
	}
	return *res, nil
}

func (s *Supervisor) stopReady(callerCtx context.Context, res *Result, prep *preparedOp) (Result, error) {
	m := prep.manifest
	// Cancel before any stop mutation leaves the run untouched.
	if err := callerCtx.Err(); err != nil && m.State == StateReady {
		res.State = m.State
		populateResultArtifacts(res, m)
		return *res, err
	}
	switch m.State {
	case StateReady:
		if err := transitionManifest(m, StateStopping, s.deps.now()); err != nil {
			res.Failure = &FailureRecord{Category: FailureManifest, Phase: m.State, Summary: err.Error()}
			return *res, err
		}
		if err := writeManifest(m.Artifacts.Manifest, m); err != nil {
			res.Failure = &FailureRecord{Category: FailureManifest, Phase: StateStopping, Summary: err.Error()}
			return *res, err
		}
		res.State = StateStopping
	case StateStopping, StateStopped:
		// Resume or sweep leftover resources without an illegal transition.
	default:
		err := fmt.Errorf("playtestenv: stopReady called in state %s", m.State)
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: m.State, Summary: err.Error()}
		return *res, err
	}

	// After mutation (or resume), finish evidence + cleanup even if the caller cancels.
	cleanupBase := context.WithoutCancel(callerCtx)
	cleanupCtx, cancel := context.WithTimeout(cleanupBase, CleanupTimeout)
	defer cancel()

	if m.ContainerID != "" {
		_ = captureServerLogs(cleanupCtx, s.deps.runner, prep.dc, m.ContainerID, m.Artifacts.ServerLog)
		_ = captureInspectEvidence(cleanupCtx, s.deps.runner, prep.dc, m.ContainerID, m.Artifacts.Inspect)
	}

	cleanup := s.cleanupReadyRun(cleanupCtx, m, prep.runDir, prep.dc)
	m.Cleanup = cleanup

	if !cleanup.Complete {
		if m.State == StateStopping {
			_ = transitionManifest(m, StateFailed, s.deps.now())
		}
		m.Failure = &FailureRecord{
			Category: FailureCleanup,
			Phase:    StateStopping,
			Summary:  cleanup.Summary,
		}
		_ = writeManifest(m.Artifacts.Manifest, m)
		res.State = m.State
		res.Failure = m.Failure
		res.Cleanup = cleanup
		populateResultArtifacts(res, m)
		return *res, fmt.Errorf("playtestenv: cleanup incomplete: %s", cleanup.Summary)
	}

	if m.State == StateStopping {
		if err := transitionManifest(m, StateStopped, s.deps.now()); err != nil {
			res.Failure = &FailureRecord{Category: FailureManifest, Phase: m.State, Summary: err.Error()}
			_ = writeManifest(m.Artifacts.Manifest, m)
			res.State = m.State
			res.Cleanup = cleanup
			return *res, err
		}
	}
	_ = writeManifest(m.Artifacts.Manifest, m)
	res.State = m.State
	res.Cleanup = cleanup
	populateResultArtifacts(res, m)
	return *res, nil
}

// gracefulStopContainer sends SIGTERM to a running container, polls up to
// ten seconds for exit, then force-removes it if still running. Missing or
// already-stopped containers are treated as success.
func (s *Supervisor) gracefulStopContainer(ctx context.Context, dc dockerContext, containerID string) *CleanupResult {
	result := &CleanupResult{Complete: true, Summary: "resources removed"}
	if containerID == "" || dc.name == "" {
		return result
	}
	running, err := containerIsRunning(ctx, s.deps.runner, dc, containerID)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if isNotFoundDockerErr(msg) {
			return result
		}
		result.Complete = false
		result.Summary = "container status check failed: " + err.Error()
		result.Leftovers = append(result.Leftovers, ResourceRef{Kind: "container", ID: containerID})
		return result
	}
	if !running {
		return result
	}
	killSpec := dockerCommand(dc, []string{"kill", "--signal=TERM", containerID}, "", io.Discard, io.Discard)
	if err := s.deps.runner.Run(ctx, killSpec); err != nil {
		result.Complete = false
		result.Summary = "docker kill failed: " + err.Error()
		result.Leftovers = append(result.Leftovers, ResourceRef{Kind: "container", ID: containerID})
		return result
	}
	still, _ := waitForContainerStop(ctx, s.deps.runner, dc, containerID, s.deps.now, s.deps.after)
	if !still {
		return result
	}
	var rmErr strings.Builder
	rmSpec := dockerCommand(dc, []string{"rm", "--force", containerID}, "", io.Discard, &rmErr)
	if err := s.deps.runner.Run(ctx, rmSpec); err != nil {
		result.Complete = false
		result.Leftovers = append(result.Leftovers, ResourceRef{Kind: "container", ID: containerID})
		result.Summary = "docker rm --force failed: " + err.Error()
	}
	return result
}

func (s *Supervisor) cleanupReadyRun(ctx context.Context, m *Manifest, runDir string, dc dockerContext) *CleanupResult {
	result := &CleanupResult{Complete: true, Summary: "resources removed"}
	vars := composeVarsFromManifest(m, runDir)
	composePath := m.Artifacts.Compose

	grace := s.gracefulStopContainer(ctx, dc, m.ContainerID)
	mergeCleanup(result, grace)

	downCleanup := s.removeComposeAndImage(ctx, m, runDir, dc, vars, composePath)
	mergeCleanup(result, downCleanup)

	if result.Complete {
		if err := removeControlArtifacts(runDir, m); err != nil {
			result.Complete = false
			result.Leftovers = append(result.Leftovers, ResourceRef{Kind: "host-path", ID: err.Error()})
			result.Summary = "control artifact removal failed: " + err.Error()
			if m.Failure == nil {
				// recorded via leftovers; category applied by caller on incomplete
			}
		}
	}
	return result
}

func mergeCleanup(dst, src *CleanupResult) {
	if src == nil {
		return
	}
	if !src.Complete {
		dst.Complete = false
	}
	dst.Leftovers = append(dst.Leftovers, src.Leftovers...)
	if src.Summary != "" && src.Summary != "resources removed" {
		if dst.Summary == "resources removed" || dst.Summary == "" {
			dst.Summary = src.Summary
		}
	}
}

func (s *Supervisor) removeComposeAndImage(
	ctx context.Context,
	m *Manifest,
	runDir string,
	dc dockerContext,
	vars composeRunVars,
	composePath string,
) *CleanupResult {
	result := &CleanupResult{Complete: true, Summary: "resources removed"}
	if composePath != "" {
		var stderr strings.Builder
		down := composeDownCommand(dc, vars, composePath, runDir, io.Discard, &stderr)
		if err := s.deps.runner.Run(ctx, down); err != nil {
			result.Complete = false
			result.Leftovers = append(result.Leftovers, ResourceRef{Kind: "compose-project", ID: vars.Project})
			result.Summary = "compose down failed: " + err.Error()
		}
	}
	img := m.Image
	if img == "" {
		img = imageNamePrefix + m.RunID
	}
	var rmErr strings.Builder
	rmSpec := dockerCommand(dc, []string{"image", "rm", img}, "", io.Discard, &rmErr)
	if err := s.deps.runner.Run(ctx, rmSpec); err != nil {
		if !isBenignImageMissing(err, rmErr.String()) {
			result.Complete = false
			result.Leftovers = append(result.Leftovers, ResourceRef{Kind: "image", ID: img})
			if result.Summary == "resources removed" {
				result.Summary = "image rm failed: " + err.Error()
			}
		}
	}
	return result
}

func removeControlArtifacts(runDir string, m *Manifest) error {
	controlDir := filepath.Join(runDir, controlDirName)
	if err := os.RemoveAll(controlDir); err != nil {
		return fmt.Errorf("%s: %w", controlDir, err)
	}
	if m.Artifacts.Compose != "" {
		if err := os.Remove(m.Artifacts.Compose); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s: %w", m.Artifacts.Compose, err)
		}
	}
	return nil
}

func composeVarsFromManifest(m *Manifest, runDir string) composeRunVars {
	return composeRunVars{
		RunID:               m.RunID,
		Project:             m.Project,
		Checkout:            m.Checkout,
		CheckoutFingerprint: m.CheckoutFingerprint,
		CreatedAt:           m.CreatedAt,
		ControlDir:          filepath.Join(runDir, controlDirName),
	}
}

func (s *Supervisor) prepareOp(ctx context.Context, checkoutPath, runID string, res *Result) (*preparedOp, error) {
	s.event("validate checkout")
	checkout, err := validateCheckoutForHost(ctx, s.deps.runner, checkoutPath)
	if err != nil {
		res.Failure = &FailureRecord{Category: FailureInvalidCheckout, Phase: StateValidating, Summary: err.Error()}
		return nil, err
	}

	dc, err := s.deps.resolveDocker(ctx, s.deps.runner)
	if err != nil {
		res.Failure = &FailureRecord{Category: FailureDockerUnavailable, Phase: StateValidating, Summary: err.Error()}
		return nil, err
	}
	s.event("local Docker preflight")

	if !isValidRunID(runID) {
		err := fmt.Errorf("%w: %q", ErrInvalidRunID, runID)
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: StateValidating, Summary: err.Error()}
		return nil, err
	}

	runDir := filepath.Join(checkout.Path, filepath.FromSlash(runsDirName), runID)
	lockPath := filepath.Join(runDir, ".lock")
	lock, err := s.deps.acquireLock(ctx, lockPath, s.deps.lockWait)
	if err != nil {
		if errors.Is(err, ErrLockBusy) {
			res.Failure = &FailureRecord{Category: FailureLockBusy, Phase: StateValidating, Summary: err.Error(), Retryable: true}
			return nil, err
		}
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: StateValidating, Summary: err.Error()}
		return nil, err
	}

	manifestPath := filepath.Join(runDir, manifestFileName)
	m, err := readManifest(manifestPath)
	if err != nil {
		_ = lock.Close()
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: StateValidating, Summary: err.Error()}
		return nil, err
	}
	if m.Artifacts.Manifest == "" {
		m.Artifacts.Manifest = manifestPath
	}

	identity, err := inspectRunIdentity(ctx, s.deps.runner, dc, m)
	if err != nil {
		_ = lock.Close()
		res.Failure = &FailureRecord{Category: FailureCleanup, Phase: m.State, Summary: err.Error()}
		return nil, err
	}

	return &preparedOp{
		checkout: checkout,
		dc:       dc,
		runID:    runID,
		runDir:   runDir,
		lock:     lock,
		manifest: m,
		identity: identity,
	}, nil
}

func inspectRunIdentity(ctx context.Context, runner Runner, dc dockerContext, m *Manifest) (identityReport, error) {
	want := expectedIdentityLabels(m)
	targets := []struct {
		kind string
		id   string
	}{
		{"container", m.ContainerID},
		{"network", m.Network},
		{"volume", m.Volume},
		{"image", m.Image},
	}
	report := identityReport{}
	for _, t := range targets {
		if t.id == "" {
			continue
		}
		lr, err := inspectLabelledResource(ctx, runner, dc, t.kind, t.id)
		if err != nil {
			return report, err
		}
		report.Resources = append(report.Resources, lr)
		if !lr.Found {
			report.Missing = append(report.Missing, t.kind+":"+t.id)
			continue
		}
		report.AnyFound = true
		if !labelsMatch(want, lr.Labels) {
			report.Mismatched = append(report.Mismatched, t.kind+":"+t.id)
		}
	}
	return report, nil
}

func inspectLabelledResource(ctx context.Context, runner Runner, dc dockerContext, kind, id string) (liveResource, error) {
	lr := liveResource{Kind: kind, ID: id}
	var stdout, stderr strings.Builder
	spec := dockerCommand(dc, []string{"inspect", id}, "", &stdout, &stderr)
	err := runner.Run(ctx, spec)
	if err != nil {
		msg := strings.ToLower(err.Error() + " " + stderr.String())
		if isNotFoundDockerErr(msg) {
			return lr, nil
		}
		return lr, fmt.Errorf("playtestenv: docker inspect %s: %w", id, err)
	}
	labels, running, err := parseInspectLabels(stdout.String())
	if err != nil {
		return lr, err
	}
	lr.Found = true
	lr.Labels = labels
	lr.Running = running
	return lr, nil
}

func isNotFoundDockerErr(msg string) bool {
	return strings.Contains(msg, "no such") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no such object")
}

func parseInspectLabels(raw string) (map[string]string, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false, fmt.Errorf("playtestenv: empty docker inspect output")
	}
	var docs []map[string]any
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &docs); err != nil {
			return nil, false, fmt.Errorf("playtestenv: decode docker inspect: %w", err)
		}
	} else {
		var doc map[string]any
		if err := json.Unmarshal([]byte(raw), &doc); err != nil {
			return nil, false, fmt.Errorf("playtestenv: decode docker inspect: %w", err)
		}
		docs = []map[string]any{doc}
	}
	if len(docs) != 1 {
		return nil, false, fmt.Errorf("playtestenv: docker inspect returned %d objects, want 1", len(docs))
	}
	doc := docs[0]
	labels := map[string]string{}
	if cfg, ok := doc["Config"].(map[string]any); ok {
		if rawLabels, ok := cfg["Labels"].(map[string]any); ok {
			for k, v := range rawLabels {
				labels[k] = fmt.Sprint(v)
			}
		}
	}
	if rawLabels, ok := doc["Labels"].(map[string]any); ok {
		for k, v := range rawLabels {
			if _, exists := labels[k]; !exists {
				labels[k] = fmt.Sprint(v)
			}
		}
	}
	running := false
	if state, ok := doc["State"].(map[string]any); ok {
		switch v := state["Running"].(type) {
		case bool:
			running = v
		}
	}
	return labels, running, nil
}

func labelsMatch(want, got map[string]string) bool {
	for k, v := range want {
		if got[k] != v {
			return false
		}
	}
	return true
}

func formatIdentityDisagreement(r identityReport) string {
	var parts []string
	if len(r.Missing) > 0 {
		parts = append(parts, "missing "+strings.Join(r.Missing, ", "))
	}
	if len(r.Mismatched) > 0 {
		parts = append(parts, "mismatch "+strings.Join(r.Mismatched, ", "))
	}
	if len(parts) == 0 {
		return "resource identity disagreement"
	}
	return strings.Join(parts, "; ")
}

func refreshServerLogAtomic(ctx context.Context, runner Runner, dc dockerContext, containerID, path string) error {
	var buf bytes.Buffer
	spec := dockerCommand(dc, []string{"logs", containerID}, "", &buf, &buf)
	if err := runner.Run(ctx, spec); err != nil {
		return fmt.Errorf("playtestenv: docker logs: %w", err)
	}
	if err := atomic.WriteFile(path, bytes.NewReader(buf.Bytes())); err != nil {
		return fmt.Errorf("playtestenv: write server log: %w", err)
	}
	return nil
}

func (s *Supervisor) followServerLogs(ctx context.Context, dc dockerContext, containerID, path string, output io.Writer) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var dest io.Writer = f
	if output != nil {
		dest = io.MultiWriter(output, f)
	}
	spec := dockerCommand(dc, []string{"logs", "--follow", containerID}, "", dest, dest)
	err = s.deps.runner.Run(ctx, spec)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ctx.Err()
		}
		return err
	}
	return nil
}

func containerIsRunning(ctx context.Context, runner Runner, dc dockerContext, containerID string) (bool, error) {
	lr, err := inspectLabelledResource(ctx, runner, dc, "container", containerID)
	if err != nil {
		return false, err
	}
	return lr.Found && lr.Running, nil
}

func waitForContainerStop(
	ctx context.Context,
	runner Runner,
	dc dockerContext,
	containerID string,
	now func() time.Time,
	after func(time.Duration) <-chan time.Time,
) (stillRunning bool, err error) {
	deadline := now().Add(stopGracePeriod)
	for {
		running, err := containerIsRunning(ctx, runner, dc, containerID)
		if err != nil {
			return false, err
		}
		if !running {
			return false, nil
		}
		if !now().Before(deadline) {
			return true, nil
		}
		select {
		case <-ctx.Done():
			return true, ctx.Err()
		case <-after(stopGracePoll):
		}
	}
}
