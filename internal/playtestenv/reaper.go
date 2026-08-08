package playtestenv

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const managedLabelFilter = "label=" + labelManaged + "=" + labelManagedValue

// Reap enumerates immediate run directories under tools/playtest/.run/,
// cleans only lease-expired runs whose manifest identities unambiguously
// match live Docker labels, and reports all other labelled leftovers as
// diagnostics without deleting them.
//
// Candidates are discovered only from the checkout filesystem — never from
// Docker alone. The local Docker context is validated once before any
// candidate inspection. Per-candidate lock waits use ReaperLockWait.
// Once a candidate's first destructive action begins, cleanup uses
// context.WithoutCancel plus CleanupTimeout so caller cancellation cannot
// interrupt that candidate; later candidates are not started after cancel.
func (s *Supervisor) Reap(ctx context.Context, checkoutPath string) ([]Result, error) {
	var results []Result

	s.event("validate checkout")
	checkout, err := validateCheckoutForHost(ctx, s.deps.runner, checkoutPath)
	if err != nil {
		res := Result{
			Operation: "reap",
			Failure: &FailureRecord{
				Category: FailureInvalidCheckout,
				Phase:    StateValidating,
				Summary:  err.Error(),
			},
		}
		return []Result{res}, err
	}

	dc, err := s.deps.resolveDocker(ctx, s.deps.runner)
	if err != nil {
		res := Result{
			Operation: "reap",
			Failure: &FailureRecord{
				Category: FailureDockerUnavailable,
				Phase:    StateValidating,
				Summary:  err.Error(),
			},
		}
		return []Result{res}, err
	}
	s.event("local Docker preflight")

	candidates, err := listRunCandidates(checkout.Path)
	if err != nil {
		res := Result{
			Operation: "reap",
			Failure: &FailureRecord{
				Category: FailureManifest,
				Phase:    StateValidating,
				Summary:  err.Error(),
			},
		}
		return []Result{res}, err
	}

	known := map[string]*Manifest{} // runID -> last successfully read manifest
	var firstErr error
	recordErr := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	for _, runID := range candidates {
		if err := ctx.Err(); err != nil {
			recordErr(err)
			break
		}
		res, m, err := s.reapCandidate(ctx, checkout.Path, runID, dc)
		results = append(results, res)
		if m != nil {
			known[m.RunID] = m
		}
		recordErr(err)
	}

	orphanResults, err := s.diagnoseOrphans(ctx, dc, known, checkout.Fingerprint)
	results = append(results, orphanResults...)
	recordErr(err)

	return results, firstErr
}

func listRunCandidates(checkout string) ([]string, error) {
	root := filepath.Join(checkout, filepath.FromSlash(runsDirName))
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("playtestenv: list run candidates: %w", err)
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !isValidRunID(name) {
			continue
		}
		ids = append(ids, name)
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *Supervisor) reapCandidate(ctx context.Context, checkoutPath, runID string, dc dockerContext) (Result, *Manifest, error) {
	res := Result{Operation: "reap", RunID: runID}
	runDir := filepath.Join(checkoutPath, filepath.FromSlash(runsDirName), runID)
	lockPath := filepath.Join(runDir, ".lock")
	manifestPath := filepath.Join(runDir, manifestFileName)

	lock, err := s.deps.acquireLock(ctx, lockPath, ReaperLockWait)
	if err != nil {
		if errors.Is(err, ErrLockBusy) {
			res.Failure = &FailureRecord{
				Category:  FailureLockBusy,
				Phase:     StateValidating,
				Summary:   err.Error(),
				Retryable: true,
			}
			return res, nil, nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			res.Failure = &FailureRecord{Category: FailureManifest, Phase: StateValidating, Summary: err.Error()}
			return res, nil, err
		}
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: StateValidating, Summary: err.Error()}
		return res, nil, nil
	}
	defer func() { _ = lock.Close() }()

	m, err := readManifest(manifestPath)
	if err != nil {
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: StateValidating, Summary: err.Error()}
		return res, nil, nil
	}
	if m.Artifacts.Manifest == "" {
		m.Artifacts.Manifest = manifestPath
	}
	if m.RunID != "" && m.RunID != runID {
		res.Failure = &FailureRecord{
			Category: FailureManifest,
			Phase:    m.State,
			Summary:  fmt.Sprintf("manifest run id %q disagrees with directory %q", m.RunID, runID),
		}
		return res, m, nil
	}
	if m.RunID == "" {
		m.RunID = runID
	}
	res.RunID = m.RunID
	res.Project = m.Project
	res.State = m.State
	populateResultArtifacts(&res, m)

	now := s.deps.now()
	// Lease must be strictly before now; equal expiry is not reaped.
	if !m.LeaseExpiresAt.Before(now) {
		return res, m, nil
	}

	identity, err := inspectRunIdentity(ctx, s.deps.runner, dc, m)
	if err != nil {
		res.Failure = &FailureRecord{Category: FailureCleanup, Phase: m.State, Summary: err.Error()}
		return res, m, nil
	}
	// Live resources that disagree with the manifest block deletion. Missing
	// resources alone do not — cleanup may still remove the Compose project /
	// image named in the validated manifest (same rule as Stop).
	if identity.AnyFound && !identity.Agrees() {
		summary := formatIdentityDisagreement(identity)
		res.Failure = &FailureRecord{
			Category: FailureIdentityMismatch,
			Phase:    m.State,
			Summary:  summary,
		}
		return res, m, nil
	}

	// Terminal + nothing live: report already-clean without destructive work.
	if (m.State == StateStopped || m.State == StateFailed) && !identity.AnyFound {
		cleanup := m.Cleanup
		if cleanup == nil || !cleanup.Complete {
			cleanup = &CleanupResult{Complete: true, Summary: "no live resources remained"}
		}
		res.Cleanup = cleanup
		res.Failure = m.Failure
		return res, m, nil
	}

	switch m.State {
	case StateValidating, StateBuilding, StateStarting, StateStopping:
		return s.reapAbandoned(ctx, &res, m, runDir, dc)
	case StateReady:
		return s.reapReady(ctx, &res, m, runDir, dc)
	case StateFailed:
		return s.reapFailed(ctx, &res, m, runDir, dc)
	case StateStopped:
		return s.reapReady(ctx, &res, m, runDir, dc)
	default:
		res.Failure = &FailureRecord{
			Category: FailureManifest,
			Phase:    m.State,
			Summary:  fmt.Sprintf("unsupported run state %q", m.State),
		}
		return res, m, nil
	}
}

func (s *Supervisor) reapAbandoned(callerCtx context.Context, res *Result, m *Manifest, runDir string, dc dockerContext) (Result, *Manifest, error) {
	phase := m.State
	if err := callerCtx.Err(); err != nil {
		res.State = m.State
		res.Failure = m.Failure
		populateResultArtifacts(res, m)
		return *res, m, err
	}
	if m.State != StateFailed {
		if err := transitionManifest(m, StateFailed, s.deps.now()); err != nil {
			res.Failure = &FailureRecord{Category: FailureManifest, Phase: phase, Summary: err.Error()}
			return *res, m, err
		}
	}
	abandoned := &FailureRecord{
		Category: FailureAbandonedRun,
		Phase:    phase,
		Summary:  fmt.Sprintf("abandoned run recovered from state %s", phase),
	}
	m.Failure = abandoned
	_ = writeManifest(m.Artifacts.Manifest, m)

	cleanupBase := context.WithoutCancel(callerCtx)
	cleanupCtx, cancel := context.WithTimeout(cleanupBase, CleanupTimeout)
	defer cancel()

	cleanup := s.cleanupFailedRun(cleanupCtx, m, runDir, m.ContainerID, dc, composeVarsFromManifest(m, runDir), m.Artifacts.Compose)
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
		return *res, m, fmt.Errorf("playtestenv: cleanup incomplete: %s", cleanup.Summary)
	}
	return *res, m, nil
}

func (s *Supervisor) reapReady(callerCtx context.Context, res *Result, m *Manifest, runDir string, dc dockerContext) (Result, *Manifest, error) {
	if err := callerCtx.Err(); err != nil && m.State == StateReady {
		res.State = m.State
		populateResultArtifacts(res, m)
		return *res, m, err
	}
	switch m.State {
	case StateReady:
		if err := transitionManifest(m, StateStopping, s.deps.now()); err != nil {
			res.Failure = &FailureRecord{Category: FailureManifest, Phase: m.State, Summary: err.Error()}
			return *res, m, err
		}
		if err := writeManifest(m.Artifacts.Manifest, m); err != nil {
			res.Failure = &FailureRecord{Category: FailureManifest, Phase: StateStopping, Summary: err.Error()}
			return *res, m, err
		}
		res.State = StateStopping
	case StateStopping, StateStopped:
		// Resume leftover cleanup.
	default:
		err := fmt.Errorf("playtestenv: reapReady called in state %s", m.State)
		res.Failure = &FailureRecord{Category: FailureManifest, Phase: m.State, Summary: err.Error()}
		return *res, m, err
	}

	cleanupBase := context.WithoutCancel(callerCtx)
	cleanupCtx, cancel := context.WithTimeout(cleanupBase, CleanupTimeout)
	defer cancel()

	if m.ContainerID != "" {
		_ = captureServerLogs(cleanupCtx, s.deps.runner, dc, m.ContainerID, m.Artifacts.ServerLog)
		_ = captureInspectEvidence(cleanupCtx, s.deps.runner, dc, m.ContainerID, m.Artifacts.Inspect)
	}

	cleanup := s.cleanupReadyRun(cleanupCtx, m, runDir, dc)
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
		return *res, m, fmt.Errorf("playtestenv: cleanup incomplete: %s", cleanup.Summary)
	}

	if m.State == StateStopping {
		if err := transitionManifest(m, StateStopped, s.deps.now()); err != nil {
			res.Failure = &FailureRecord{Category: FailureManifest, Phase: m.State, Summary: err.Error()}
			_ = writeManifest(m.Artifacts.Manifest, m)
			res.State = m.State
			res.Cleanup = cleanup
			return *res, m, err
		}
	}
	_ = writeManifest(m.Artifacts.Manifest, m)
	res.State = m.State
	res.Cleanup = cleanup
	populateResultArtifacts(res, m)
	return *res, m, nil
}

func (s *Supervisor) reapFailed(callerCtx context.Context, res *Result, m *Manifest, runDir string, dc dockerContext) (Result, *Manifest, error) {
	if err := callerCtx.Err(); err != nil {
		res.Failure = m.Failure
		return *res, m, err
	}
	cleanupBase := context.WithoutCancel(callerCtx)
	cleanupCtx, cancel := context.WithTimeout(cleanupBase, CleanupTimeout)
	defer cancel()

	cleanup := s.cleanupFailedRun(cleanupCtx, m, runDir, m.ContainerID, dc, composeVarsFromManifest(m, runDir), m.Artifacts.Compose)
	m.Cleanup = cleanup
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
		return *res, m, fmt.Errorf("playtestenv: cleanup incomplete: %s", cleanup.Summary)
	}
	return *res, m, nil
}

type orphanResource struct {
	Kind   string
	ID     string
	Labels map[string]string
}

func (s *Supervisor) diagnoseOrphans(ctx context.Context, dc dockerContext, known map[string]*Manifest, checkoutFP string) ([]Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resources, err := listManagedResources(ctx, s.deps.runner, dc)
	if err != nil {
		return []Result{{
			Operation: "reap",
			Failure: &FailureRecord{
				Category: FailureCleanup,
				Phase:    StateValidating,
				Summary:  err.Error(),
			},
		}}, err
	}

	// Known resource IDs from manifests we already inspected — skip those.
	knownIDs := map[string]struct{}{}
	for _, m := range known {
		for _, id := range []string{m.ContainerID, m.Network, m.Volume, m.Image} {
			if id != "" {
				knownIDs[id] = struct{}{}
			}
		}
	}

	var results []Result
	seen := map[string]struct{}{}
	for _, r := range resources {
		key := r.Kind + ":" + r.ID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if _, ok := knownIDs[r.ID]; ok {
			continue
		}
		runID := r.Labels[labelRunID]
		summary := orphanSummary(r, known, checkoutFP)
		res := Result{
			Operation: "reap",
			RunID:     runID,
			Failure: &FailureRecord{
				Category: FailureIdentityMismatch,
				Phase:    StateValidating,
				Summary:  summary,
			},
		}
		results = append(results, res)
	}
	return results, nil
}

func orphanSummary(r orphanResource, known map[string]*Manifest, checkoutFP string) string {
	runID := r.Labels[labelRunID]
	fp := r.Labels[labelCheckout]
	if runID == "" {
		return fmt.Sprintf("labelled %s %s without matching manifest (missing run-id label)", r.Kind, r.ID)
	}
	if _, ok := known[runID]; !ok {
		if fp != "" && fp != checkoutFP {
			return fmt.Sprintf("labelled %s %s for run %q belongs to other checkout (fingerprint %s); no matching manifest", r.Kind, r.ID, runID, fp)
		}
		return fmt.Sprintf("labelled %s %s for run %q without matching manifest", r.Kind, r.ID, runID)
	}
	if fp != "" && fp != checkoutFP {
		return fmt.Sprintf("labelled %s %s for run %q belongs to other checkout", r.Kind, r.ID, runID)
	}
	return fmt.Sprintf("labelled %s %s for run %q cannot be safely reaped", r.Kind, r.ID, runID)
}

func listManagedResources(ctx context.Context, runner Runner, dc dockerContext) ([]orphanResource, error) {
	var out []orphanResource
	queries := []struct {
		kind string
		args []string
	}{
		{"container", []string{"ps", "-a", "--filter", managedLabelFilter, "--format", "{{json .}}"}},
		{"network", []string{"network", "ls", "--filter", managedLabelFilter, "--format", "{{json .}}"}},
		{"volume", []string{"volume", "ls", "--filter", managedLabelFilter, "--format", "{{json .}}"}},
		{"image", []string{"images", "--filter", managedLabelFilter, "--format", "{{json .}}"}},
	}
	for _, q := range queries {
		var stdout, stderr strings.Builder
		spec := dockerCommand(dc, q.args, "", &stdout, &stderr)
		if err := runner.Run(ctx, spec); err != nil {
			return out, fmt.Errorf("playtestenv: list managed %ss: %w", q.kind, err)
		}
		items, err := parseDockerListJSON(stdout.String(), q.kind)
		if err != nil {
			return out, err
		}
		out = append(out, items...)
	}
	return out, nil
}

func parseDockerListJSON(raw, kind string) ([]orphanResource, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []orphanResource
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(line), &doc); err != nil {
			return nil, fmt.Errorf("playtestenv: decode docker %s list: %w", kind, err)
		}
		id := firstString(doc, "ID", "Id", "Name", "Names")
		if id == "" {
			continue
		}
		// docker ps Names can be "/name" or comma-separated; keep raw ID prefer.
		if v, ok := doc["ID"].(string); ok && v != "" {
			id = v
		} else if v, ok := doc["Id"].(string); ok && v != "" {
			id = v
		}
		labels := parseListLabels(doc["Labels"])
		out = append(out, orphanResource{Kind: kind, ID: id, Labels: labels})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func firstString(doc map[string]any, keys ...string) string {
	for _, k := range keys {
		switch v := doc[k].(type) {
		case string:
			if v != "" {
				return v
			}
		}
	}
	return ""
}

func parseListLabels(raw any) map[string]string {
	out := map[string]string{}
	switch v := raw.(type) {
	case map[string]any:
		for k, val := range v {
			out[k] = fmt.Sprint(val)
		}
	case map[string]string:
		for k, val := range v {
			out[k] = val
		}
	case string:
		if v == "" {
			return out
		}
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			k, val, ok := strings.Cut(part, "=")
			if !ok {
				continue
			}
			out[strings.TrimSpace(k)] = strings.TrimSpace(val)
		}
	}
	return out
}
