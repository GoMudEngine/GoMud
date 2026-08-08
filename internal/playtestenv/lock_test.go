package playtestenv

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/stretchr/testify/require"
)

// TestRunLockContentionSameProcess proves a second acquisition attempt on an
// already-held lock, from the same process, is blocked rather than silently
// succeeding.
func TestRunLockContentionSameProcess(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "run.lock")

	first, err := acquireRunLock(context.Background(), lockPath, time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = first.Close() })

	_, err = acquireRunLock(context.Background(), lockPath, 150*time.Millisecond)
	require.ErrorIs(t, err, ErrLockBusy)
}

// TestRunLockHeldTimeoutReturnsErrLockBusy proves a bounded wait against a
// held lock returns the sentinel busy error rather than hanging forever or
// leaking the underlying context-deadline error.
func TestRunLockHeldTimeoutReturnsErrLockBusy(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "run.lock")

	holder, err := acquireRunLock(context.Background(), lockPath, time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = holder.Close() })

	wait := 200 * time.Millisecond
	start := time.Now()
	_, err = acquireRunLock(context.Background(), lockPath, wait)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, ErrLockBusy)
	require.GreaterOrEqual(t, elapsed, wait)
	require.Less(t, elapsed, wait+2*time.Second, "acquireRunLock must not hang past its bounded wait")
}

// TestRunLockPreservesParentCancellation proves that when the parent context
// is cancelled before the internal wait expires, acquireRunLock surfaces the
// parent's cancellation rather than masking it as ErrLockBusy.
func TestRunLockPreservesParentCancellation(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "run.lock")

	holder, err := acquireRunLock(context.Background(), lockPath, time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = holder.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	_, err = acquireRunLock(ctx, lockPath, 10*time.Second)
	require.ErrorIs(t, err, context.Canceled)
	require.NotErrorIs(t, err, ErrLockBusy)
}

// TestRunLockReleasesAfterProcessExit proves the lock is released by the
// operating system when the holding process exits without ever calling
// Unlock, rather than requiring any PID-file or age-based cleanup.
func TestRunLockReleasesAfterProcessExit(t *testing.T) {
	if os.Getenv("DOGMUD_LOCK_HELPER_RUN") == "1" {
		runLockHelperProcess()
		return
	}

	lockPath := filepath.Join(t.TempDir(), "run.lock")

	cmd := exec.Command(os.Args[0], "-test.run=TestRunLockReleasesAfterProcessExit")
	cmd.Env = append(os.Environ(),
		"DOGMUD_LOCK_HELPER_RUN=1",
		"DOGMUD_LOCK_HELPER_PATH="+lockPath,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "helper process failed: %s", string(out))

	parentLock, err := acquireRunLock(context.Background(), lockPath, 2*time.Second)
	require.NoError(t, err, "parent could not acquire lock after helper process exited without Unlock")
	require.NoError(t, parentLock.Close())
}

// runLockHelperProcess acquires the lock named by DOGMUD_LOCK_HELPER_PATH and
// exits immediately without releasing it, so the OS - not application code -
// is responsible for releasing the lock.
func runLockHelperProcess() {
	path := os.Getenv("DOGMUD_LOCK_HELPER_PATH")
	f := flock.New(path)
	ok, err := f.TryLock()
	if err != nil || !ok {
		os.Exit(1)
	}
	os.Exit(0)
}
