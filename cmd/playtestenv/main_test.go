package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/playtestenv"
)

// fakeSupervisor records calls and returns scripted outcomes.
type fakeSupervisor struct {
	startOpts  playtestenv.StartOptions
	statusOpts playtestenv.RunOptions
	logsOpts   playtestenv.LogsOptions
	renewOpts  playtestenv.RenewOptions
	stopOpts   playtestenv.RunOptions
	reapPath   string

	startCalled  bool
	statusCalled bool
	logsCalled   bool
	renewCalled  bool
	stopCalled   bool
	reapCalled   bool

	startRes  playtestenv.Result
	statusRes playtestenv.Result
	logsRes   playtestenv.Result
	renewRes  playtestenv.Result
	stopRes   playtestenv.Result
	reapRes   []playtestenv.Result

	startErr  error
	statusErr error
	logsErr   error
	renewErr  error
	stopErr   error
	reapErr   error

	// logsFollowText is written to opts.Output when Follow is true.
	logsFollowText string
	// blockUntilCancelled blocks Start until ctx is done.
	blockUntilCancelled bool
}

func (f *fakeSupervisor) Start(ctx context.Context, opts playtestenv.StartOptions) (playtestenv.Result, error) {
	f.startCalled = true
	f.startOpts = opts
	if f.blockUntilCancelled {
		<-ctx.Done()
		return f.startRes, ctx.Err()
	}
	return f.startRes, f.startErr
}

func (f *fakeSupervisor) Status(ctx context.Context, opts playtestenv.RunOptions) (playtestenv.Result, error) {
	f.statusCalled = true
	f.statusOpts = opts
	return f.statusRes, f.statusErr
}

func (f *fakeSupervisor) Logs(ctx context.Context, opts playtestenv.LogsOptions) (playtestenv.Result, error) {
	f.logsCalled = true
	f.logsOpts = opts
	if opts.Follow && opts.Output != nil && f.logsFollowText != "" {
		_, _ = io.WriteString(opts.Output, f.logsFollowText)
	}
	return f.logsRes, f.logsErr
}

func (f *fakeSupervisor) Renew(ctx context.Context, opts playtestenv.RenewOptions) (playtestenv.Result, error) {
	f.renewCalled = true
	f.renewOpts = opts
	return f.renewRes, f.renewErr
}

func (f *fakeSupervisor) Stop(ctx context.Context, opts playtestenv.RunOptions) (playtestenv.Result, error) {
	f.stopCalled = true
	f.stopOpts = opts
	return f.stopRes, f.stopErr
}

func (f *fakeSupervisor) Reap(ctx context.Context, checkoutPath string) ([]playtestenv.Result, error) {
	f.reapCalled = true
	f.reapPath = checkoutPath
	return f.reapRes, f.reapErr
}

func TestRunStartDefaultsCheckoutLeaseAndReadiness(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeSupervisor{
		startRes: playtestenv.Result{
			Operation: "start",
			RunID:     "run-1",
			State:     playtestenv.StateReady,
			Endpoint:  &playtestenv.Endpoint{Host: "127.0.0.1", Port: 55555},
		},
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"start"}, &stdout, &stderr, fake)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if !fake.startCalled {
		t.Fatal("Start not called")
	}
	if fake.startOpts.Checkout != cwd {
		t.Fatalf("checkout=%q want cwd %q", fake.startOpts.Checkout, cwd)
	}
	if fake.startOpts.Lease != 2*time.Hour {
		t.Fatalf("lease=%v want 2h", fake.startOpts.Lease)
	}
	if fake.startOpts.ReadinessTimeout != 90*time.Second {
		t.Fatalf("readiness=%v want 90s", fake.startOpts.ReadinessTimeout)
	}
	out := stdout.String()
	if !strings.Contains(out, "run-1") {
		t.Fatalf("human stdout missing run id: %q", out)
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunStartExplicitCheckoutAndLeaseJSON(t *testing.T) {
	checkout := t.TempDir()
	fake := &fakeSupervisor{
		startRes: playtestenv.Result{
			Operation: "start",
			RunID:     "abc",
			Project:   "pt-abc",
			State:     playtestenv.StateReady,
			Endpoint:  &playtestenv.Endpoint{Host: "127.0.0.1", Port: 12345},
			Manifest:  filepath.Join(checkout, "tools", "playtest", ".run", "abc", "manifest.json"),
			ServerLog: filepath.Join(checkout, "tools", "playtest", ".run", "abc", "server.log"),
		},
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{
		"start", "--checkout", checkout, "--lease", "30m", "--json",
	}, &stdout, &stderr, fake)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if fake.startOpts.Checkout != checkout {
		t.Fatalf("checkout=%q", fake.startOpts.Checkout)
	}
	if fake.startOpts.Lease != 30*time.Minute {
		t.Fatalf("lease=%v", fake.startOpts.Lease)
	}
	if fake.startOpts.ReadinessTimeout != 90*time.Second {
		t.Fatalf("readiness=%v want default 90s", fake.startOpts.ReadinessTimeout)
	}
	assertSingleJSONObject(t, stdout.Bytes())
	var res playtestenv.Result
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.RunID != "abc" || res.Operation != "start" {
		t.Fatalf("json result=%+v", res)
	}
	if strings.Contains(stdout.String(), "build") && strings.Contains(stdout.String(), "Step") {
		t.Fatalf("subprocess text mixed into JSON stdout: %q", stdout.String())
	}
}

func TestRunStatusRequiresCheckoutAndRun(t *testing.T) {
	fake := &fakeSupervisor{}
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"status", "--run", "x"}, &stdout, &stderr, fake); code != 2 {
		t.Fatalf("missing checkout exit=%d want 2", code)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run(context.Background(), []string{"status", "--checkout", t.TempDir()}, &stdout, &stderr, fake); code != 2 {
		t.Fatalf("missing run exit=%d want 2", code)
	}
	if fake.statusCalled {
		t.Fatal("Status must not be called on usage error")
	}
}

func TestRunStatusSuccess(t *testing.T) {
	checkout := t.TempDir()
	fake := &fakeSupervisor{
		statusRes: playtestenv.Result{Operation: "status", RunID: "r1", State: playtestenv.StateReady},
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{
		"status", "--checkout", checkout, "--run", "r1",
	}, &stdout, &stderr, fake)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if fake.statusOpts.Checkout != checkout || fake.statusOpts.RunID != "r1" {
		t.Fatalf("opts=%+v", fake.statusOpts)
	}
	if !strings.Contains(stdout.String(), "r1") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestRunLogsFollowStreamsToStdoutNotJSON(t *testing.T) {
	checkout := t.TempDir()
	fake := &fakeSupervisor{
		logsRes:        playtestenv.Result{Operation: "logs", RunID: "r1", ServerLog: "/tmp/server.log"},
		logsFollowText: "line-one\n",
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{
		"logs", "--checkout", checkout, "--run", "r1", "--follow",
	}, &stdout, &stderr, fake)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if !fake.logsOpts.Follow {
		t.Fatal("Follow not set")
	}
	if !strings.Contains(stdout.String(), "line-one") {
		t.Fatalf("follow stream missing from stdout: %q", stdout.String())
	}
}

func TestRunLogsFollowJSONRejected(t *testing.T) {
	fake := &fakeSupervisor{}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{
		"logs", "--checkout", t.TempDir(), "--run", "r1", "--follow", "--json",
	}, &stdout, &stderr, fake)
	if code != 2 {
		t.Fatalf("exit=%d want 2", code)
	}
	if fake.logsCalled {
		t.Fatal("Logs must not be called when follow+json")
	}
}

func TestRunLogsRequiresCheckoutAndRun(t *testing.T) {
	fake := &fakeSupervisor{}
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"logs", "--run", "r1"}, &stdout, &stderr, fake); code != 2 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRunRenewRequiresLeaseCheckoutAndRun(t *testing.T) {
	fake := &fakeSupervisor{}
	checkout := t.TempDir()
	var stdout, stderr bytes.Buffer
	cases := [][]string{
		{"renew", "--checkout", checkout, "--run", "r1"},
		{"renew", "--checkout", checkout, "--lease", "1h"},
		{"renew", "--run", "r1", "--lease", "1h"},
	}
	for _, args := range cases {
		stdout.Reset()
		stderr.Reset()
		if code := run(context.Background(), args, &stdout, &stderr, fake); code != 2 {
			t.Fatalf("args=%v exit=%d want 2", args, code)
		}
	}
	if fake.renewCalled {
		t.Fatal("Renew must not be called")
	}
}

func TestRunRenewSuccess(t *testing.T) {
	checkout := t.TempDir()
	fake := &fakeSupervisor{
		renewRes: playtestenv.Result{Operation: "renew", RunID: "r1", State: playtestenv.StateReady},
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{
		"renew", "--checkout", checkout, "--run", "r1", "--lease", "45m", "--json",
	}, &stdout, &stderr, fake)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if fake.renewOpts.Lease != 45*time.Minute {
		t.Fatalf("lease=%v", fake.renewOpts.Lease)
	}
	assertSingleJSONObject(t, stdout.Bytes())
}

func TestRunStopRequiresCheckoutAndRun(t *testing.T) {
	fake := &fakeSupervisor{}
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"stop", "--run", "r1"}, &stdout, &stderr, fake); code != 2 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRunStopSuccess(t *testing.T) {
	checkout := t.TempDir()
	fake := &fakeSupervisor{
		stopRes: playtestenv.Result{
			Operation: "stop",
			RunID:     "r1",
			State:     playtestenv.StateStopped,
			Cleanup:   &playtestenv.CleanupResult{Complete: true, Summary: "removed"},
		},
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{
		"stop", "--checkout", checkout, "--run", "r1",
	}, &stdout, &stderr, fake)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if fake.stopOpts.Checkout != checkout || fake.stopOpts.RunID != "r1" {
		t.Fatalf("opts=%+v", fake.stopOpts)
	}
}

func TestRunReapDefaultsCheckout(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeSupervisor{
		reapRes: []playtestenv.Result{
			{Operation: "reap", RunID: "old", State: playtestenv.StateStopped, Cleanup: &playtestenv.CleanupResult{Complete: true}},
		},
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"reap"}, &stdout, &stderr, fake)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if fake.reapPath != cwd {
		t.Fatalf("reap path=%q want cwd %q", fake.reapPath, cwd)
	}
	if !strings.Contains(stdout.String(), "old") {
		t.Fatalf("human reap summary missing run: %q", stdout.String())
	}
}

func TestRunReapJSONWrapperObject(t *testing.T) {
	// Reap returns []Result. JSON mode encodes exactly one wrapper object
	// {"operation":"reap","results":[...]} so stdout stays a single JSON
	// object (not a bare array, not mixed text).
	checkout := t.TempDir()
	fake := &fakeSupervisor{
		reapRes: []playtestenv.Result{
			{Operation: "reap", RunID: "a", State: playtestenv.StateStopped},
			{Operation: "reap", RunID: "b", Failure: &playtestenv.FailureRecord{Category: playtestenv.FailureLockBusy, Retryable: true, Summary: "busy"}},
		},
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"reap", "--checkout", checkout, "--json"}, &stdout, &stderr, fake)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	assertSingleJSONObject(t, stdout.Bytes())
	var wrap struct {
		Operation string               `json:"operation"`
		Results   []playtestenv.Result `json:"results"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &wrap); err != nil {
		t.Fatal(err)
	}
	if wrap.Operation != "reap" || len(wrap.Results) != 2 {
		t.Fatalf("wrap=%+v", wrap)
	}
	if wrap.Results[1].Failure == nil || wrap.Results[1].Failure.Category != playtestenv.FailureLockBusy {
		t.Fatalf("results=%+v", wrap.Results)
	}
}

func TestRunLockBusyJSONExit1(t *testing.T) {
	fake := &fakeSupervisor{
		startRes: playtestenv.Result{
			Operation: "start",
			Failure: &playtestenv.FailureRecord{
				Category:  playtestenv.FailureLockBusy,
				Phase:     playtestenv.StateValidating,
				Summary:   "playtestenv: run lock is busy",
				Retryable: true,
			},
		},
		startErr: playtestenv.ErrLockBusy,
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"start", "--json"}, &stdout, &stderr, fake)
	if code != 1 {
		t.Fatalf("exit=%d want 1", code)
	}
	assertSingleJSONObject(t, stdout.Bytes())
	var res playtestenv.Result
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Failure == nil {
		t.Fatal("missing failure")
	}
	if res.Failure.Category != "lock_busy" {
		t.Fatalf("category=%q", res.Failure.Category)
	}
	if !res.Failure.Retryable {
		t.Fatal("retryable want true")
	}
}

func TestRunOperationFailureHumanDiagnosticsOnStderr(t *testing.T) {
	fake := &fakeSupervisor{
		startRes: playtestenv.Result{
			Operation: "start",
			RunID:     "failed-run",
			Manifest:  "/path/manifest.json",
			ServerLog: "/path/server.log",
			Report:    "/path/report.md",
			Failure: &playtestenv.FailureRecord{
				Category: playtestenv.FailureBuild,
				Phase:    playtestenv.StateBuilding,
				Summary:  "build failed",
			},
			Cleanup: &playtestenv.CleanupResult{
				Complete: false,
				Leftovers: []playtestenv.ResourceRef{
					{Kind: "container", ID: "c1"},
				},
				Summary: "partial cleanup",
			},
		},
		startErr: errors.New("build failed"),
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"start"}, &stdout, &stderr, fake)
	if code != 1 {
		t.Fatalf("exit=%d want 1", code)
	}
	diag := stderr.String()
	for _, want := range []string{"build_failure", "failed-run", "manifest.json", "server.log", "report.md", "container", "c1"} {
		if !strings.Contains(diag, want) {
			t.Fatalf("stderr missing %q: %q", want, diag)
		}
	}
	if strings.Contains(diag, "password") || strings.Contains(diag, "secret") {
		t.Fatalf("secrets leaked: %q", diag)
	}
}

func TestRunUnknownSubcommandExit2(t *testing.T) {
	fake := &fakeSupervisor{}
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"explode"}, &stdout, &stderr, fake); code != 2 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRunUnknownFlagExit2(t *testing.T) {
	fake := &fakeSupervisor{}
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"start", "--not-a-flag"}, &stdout, &stderr, fake); code != 2 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRunRejectsForbiddenFlags(t *testing.T) {
	fake := &fakeSupervisor{}
	forbidden := []string{
		"--host", "x",
		"--target", "x",
		"--context", "x",
		"--compose-file", "x",
		"--source-mount", "x",
		"--export", "x",
	}
	// Each forbidden flag alone on start should exit 2.
	for i := 0; i < len(forbidden); i += 2 {
		args := []string{"start", forbidden[i], forbidden[i+1]}
		var stdout, stderr bytes.Buffer
		if code := run(context.Background(), args, &stdout, &stderr, fake); code != 2 {
			t.Fatalf("args=%v exit=%d want 2 stderr=%q", args, code, stderr.String())
		}
	}
	if fake.startCalled {
		t.Fatal("Start must not run with forbidden flags")
	}
}

func TestRunContextCancellation(t *testing.T) {
	fake := &fakeSupervisor{
		blockUntilCancelled: true,
		startRes:            playtestenv.Result{Operation: "start"},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	var stdout, stderr bytes.Buffer
	go func() {
		done <- run(ctx, []string{"start"}, &stdout, &stderr, fake)
	}()
	// Allow Start to block, then cancel (stands in for SIGINT/SIGTERM via
	// signal.NotifyContext in main).
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case code := <-done:
		if code != 1 {
			t.Fatalf("exit=%d want 1 on cancel", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run did not return after cancel")
	}
}

func TestRunNoArgsExit2(t *testing.T) {
	fake := &fakeSupervisor{}
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), nil, &stdout, &stderr, fake); code != 2 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRunJSONStdoutExactlyOneObjectNoSubprocessText(t *testing.T) {
	fake := &fakeSupervisor{
		statusRes: playtestenv.Result{
			Operation: "status",
			RunID:     "r9",
			State:     playtestenv.StateReady,
		},
	}
	var stdout, stderr bytes.Buffer
	// Diagnostics that must not appear on stdout in JSON mode.
	_, _ = io.WriteString(&stderr, "docker build Step 1/5\n")
	code := run(context.Background(), []string{
		"status", "--checkout", t.TempDir(), "--run", "r9", "--json",
	}, &stdout, &stderr, fake)
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	assertSingleJSONObject(t, stdout.Bytes())
	if strings.Contains(stdout.String(), "docker build") {
		t.Fatalf("subprocess text in stdout: %q", stdout.String())
	}
}

func assertSingleJSONObject(t *testing.T, raw []byte) {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(raw)))
	var first any
	if err := dec.Decode(&first); err != nil {
		t.Fatalf("decode first: %v raw=%q", err, raw)
	}
	obj, ok := first.(map[string]any)
	if !ok {
		t.Fatalf("want JSON object, got %T raw=%q", first, raw)
	}
	if obj == nil {
		t.Fatal("nil object")
	}
	var second any
	if err := dec.Decode(&second); err != io.EOF {
		t.Fatalf("want exactly one JSON value, got trailing %v err=%v raw=%q", second, err, raw)
	}
}
