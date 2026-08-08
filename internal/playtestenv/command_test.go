package playtestenv

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// dogmudExecHelperModeVar selects which helper behavior the re-exec'd test
// binary performs. It is only ever set on the child process's environment,
// never read from the parent's ambient environment.
const dogmudExecHelperModeVar = "DOGMUD_EXEC_HELPER_MODE"

// execHelperLargeOutputSize is chosen to be many times larger than any
// common OS pipe buffer (historically 64KiB on Linux) so that a runner which
// drains stdout and stderr serially - instead of concurrently - deadlocks:
// the child blocks writing to the second stream's full pipe while the parent
// is still waiting to finish reading the first.
const execHelperLargeOutputSize = 3 * 1024 * 1024

// TestExecRunnerHandlesLargeConcurrentStdoutAndStderrWithoutDeadlock proves
// execRunner attaches stdout/stderr writers before Start and drains both
// streams concurrently, rather than serially via StdoutPipe/StderrPipe. A
// serial drain would deadlock once either stream exceeds one OS pipe
// buffer.
func TestExecRunnerHandlesLargeConcurrentStdoutAndStderrWithoutDeadlock(t *testing.T) {
	if os.Getenv(dogmudExecHelperModeVar) == "emit-large" {
		execHelperEmitLargeConcurrentOutput()
		return
	}

	runner := execRunner{}
	var stdout, stderr bytes.Buffer
	spec := CommandSpec{
		Name:   os.Args[0],
		Args:   []string{"-test.run=^TestExecRunnerHandlesLargeConcurrentStdoutAndStderrWithoutDeadlock$"},
		Env:    append(append([]string{}, os.Environ()...), dogmudExecHelperModeVar+"=emit-large"),
		Stdout: &stdout,
		Stderr: &stderr,
	}

	done := make(chan error, 1)
	go func() { done <- runner.Run(context.Background(), spec) }()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(20 * time.Second):
		t.Fatal("execRunner.Run deadlocked draining large concurrent stdout/stderr")
	}

	wantStdout := bytes.Repeat([]byte("O"), execHelperLargeOutputSize)
	wantStderr := bytes.Repeat([]byte("E"), execHelperLargeOutputSize)
	require.True(t, bytes.Equal(wantStdout, stdout.Bytes()), "stdout was truncated or corrupted")
	require.True(t, bytes.Equal(wantStderr, stderr.Bytes()), "stderr was truncated or corrupted")
}

// execHelperEmitLargeConcurrentOutput is the child-process body invoked via
// re-exec. It writes to stdout and stderr from separate goroutines so both
// streams fill concurrently, then exits immediately via os.Exit so the
// testing package's own "PASS" summary line is never appended to stdout,
// which would corrupt the parent's exact byte-for-byte comparison.
func execHelperEmitLargeConcurrentOutput() {
	stdoutData := bytes.Repeat([]byte("O"), execHelperLargeOutputSize)
	stderrData := bytes.Repeat([]byte("E"), execHelperLargeOutputSize)

	done := make(chan struct{}, 2)
	go func() {
		_, _ = os.Stdout.Write(stdoutData)
		done <- struct{}{}
	}()
	go func() {
		_, _ = os.Stderr.Write(stderrData)
		done <- struct{}{}
	}()
	<-done
	<-done
	os.Exit(0)
}

// TestExecRunnerWaitDelayPreventsHangOnOrphanedGrandchildPipes proves Run
// sets a bounded WaitDelay so it cannot hang forever when its immediate
// child exits but leaves behind a grandchild that inherited - and is still
// holding open - the same stdout/stderr handles. Without a WaitDelay,
// exec.Cmd.Wait blocks until every holder of those handles closes them,
// even though the process exec.Cmd itself is tracking has already exited.
func TestExecRunnerWaitDelayPreventsHangOnOrphanedGrandchildPipes(t *testing.T) {
	if os.Getenv(dogmudExecHelperModeVar) == "spawn-grandchild" {
		execHelperSpawnGrandchildHoldingPipes()
		return
	}

	runner := execRunner{}
	var stdout, stderr bytes.Buffer
	spec := CommandSpec{
		Name:   os.Args[0],
		Args:   []string{"-test.run=^TestExecRunnerWaitDelayPreventsHangOnOrphanedGrandchildPipes$"},
		Env:    append(append([]string{}, os.Environ()...), dogmudExecHelperModeVar+"=spawn-grandchild"),
		Stdout: &stdout,
		Stderr: &stderr,
	}

	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- runner.Run(context.Background(), spec) }()

	select {
	case err := <-done:
		elapsed := time.Since(start)
		require.Less(t, elapsed, 15*time.Second,
			"Run must return within its bounded WaitDelay even though a grandchild still holds stdout/stderr open")
		require.Error(t, err, "an orphaned grandchild holding the pipes open must surface as an error, not a silent success")
		require.True(t, errors.Is(err, exec.ErrWaitDelay), "expected exec.ErrWaitDelay in the error chain, got %v (%T)", err, err)
	case <-time.After(20 * time.Second):
		t.Fatal("execRunner.Run hung past its WaitDelay bound waiting on an orphaned grandchild's pipes")
	}
}

// execHelperSpawnGrandchildHoldingPipes starts a grandchild sleep process
// that inherits this process's (already-redirected) stdout/stderr, then
// exits immediately without waiting for it. The grandchild keeps running
// and holding those handles open even after this, the immediate child, has
// terminated - the exact scenario execRunnerWaitDelay must recover from.
//
// The grandchild is a trivial external sleep command rather than another
// copy of this test binary: re-executing the test binary itself would keep
// its temp executable file locked on Windows for the sleep's duration,
// causing `go test` to fail deleting it after the run even though every
// test passed.
func execHelperSpawnGrandchildHoldingPipes() {
	var grandchild *exec.Cmd
	if runtime.GOOS == "windows" {
		grandchild = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "Start-Sleep -Seconds 10")
	} else {
		grandchild = exec.Command("sleep", "10")
	}
	grandchild.Stdout = os.Stdout
	grandchild.Stderr = os.Stderr
	if err := grandchild.Start(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

// TestExecRunnerContextCancellationTerminatesChild proves that cancelling
// the context passed to Run terminates the child process instead of leaving
// it running or blocking the caller until natural exit.
func TestExecRunnerContextCancellationTerminatesChild(t *testing.T) {
	if os.Getenv(dogmudExecHelperModeVar) == "sleep-long" {
		time.Sleep(30 * time.Second)
		return
	}

	runner := execRunner{}
	ctx, cancel := context.WithCancel(context.Background())
	spec := CommandSpec{
		Name:   os.Args[0],
		Args:   []string{"-test.run=^TestExecRunnerContextCancellationTerminatesChild$"},
		Env:    append(append([]string{}, os.Environ()...), dogmudExecHelperModeVar+"=sleep-long"),
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx, spec) }()

	time.Sleep(300 * time.Millisecond)
	cancelledAt := time.Now()
	cancel()

	select {
	case err := <-done:
		require.Error(t, err, "a cancelled context must surface as an error from Run")
		require.Less(t, time.Since(cancelledAt), 10*time.Second,
			"child was not terminated promptly after context cancellation")
	case <-time.After(15 * time.Second):
		t.Fatal("execRunner.Run did not return after context cancellation; child was likely not terminated")
	}
}

// TestExecRunnerNonzeroExitStatusIsInspectable proves a nonzero child exit
// status is preserved and inspectable by the caller via errors.As, rather
// than being swallowed or reported only as an opaque error string.
func TestExecRunnerNonzeroExitStatusIsInspectable(t *testing.T) {
	if os.Getenv(dogmudExecHelperModeVar) == "exit-nonzero" {
		os.Exit(7)
	}

	runner := execRunner{}
	spec := CommandSpec{
		Name:   os.Args[0],
		Args:   []string{"-test.run=^TestExecRunnerNonzeroExitStatusIsInspectable$"},
		Env:    append(append([]string{}, os.Environ()...), dogmudExecHelperModeVar+"=exit-nonzero"),
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	err := runner.Run(context.Background(), spec)
	require.Error(t, err)

	var exitErr *ExitError
	require.True(t, errors.As(err, &exitErr), "expected *ExitError, got %T: %v", err, err)
	require.Equal(t, 7, exitErr.ExitCode)

	var stdExitErr *exec.ExitError
	require.True(t, errors.As(err, &stdExitErr), "expected underlying *exec.ExitError to remain inspectable via errors.As")
}

// TestExecRunnerSucceedsWithZeroExit proves the ordinary success path
// returns a nil error and does not misclassify exit code zero.
func TestExecRunnerSucceedsWithZeroExit(t *testing.T) {
	if os.Getenv(dogmudExecHelperModeVar) == "exit-zero" {
		os.Exit(0)
	}

	runner := execRunner{}
	spec := CommandSpec{
		Name:   os.Args[0],
		Args:   []string{"-test.run=^TestExecRunnerSucceedsWithZeroExit$"},
		Env:    append(append([]string{}, os.Environ()...), dogmudExecHelperModeVar+"=exit-zero"),
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	require.NoError(t, runner.Run(context.Background(), spec))
}

// TestExecRunnerDoesNotMutateCallerEnvironment proves that passing an
// explicit Env slice never mutates process-global state such as
// os.Environ(), regardless of how many times Run is called.
func TestExecRunnerDoesNotMutateCallerEnvironment(t *testing.T) {
	if os.Getenv(dogmudExecHelperModeVar) == "exit-zero-quiet" {
		os.Exit(0)
	}

	before := os.Environ()

	runner := execRunner{}
	spec := CommandSpec{
		Name:   os.Args[0],
		Args:   []string{"-test.run=^TestExecRunnerDoesNotMutateCallerEnvironment$"},
		Env:    []string{"PATH=" + os.Getenv("PATH"), dogmudExecHelperModeVar + "=exit-zero-quiet"},
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	_ = runner.Run(context.Background(), spec)

	after := os.Environ()
	require.Equal(t, before, after, "Run must never mutate the caller's ambient environment")
}
