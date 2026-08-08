package playtestenv

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/flock"
)

// DefaultLockWait is the bounded wait used by direct run operations (start,
// renew, stop) when acquiring a run's advisory lock.
const DefaultLockWait = 5 * time.Second

// ReaperLockWait is the bounded wait used per reaper candidate when
// acquiring a run's advisory lock. It is short so one unresponsive lock does
// not stall reaping of every other candidate.
const ReaperLockWait = 250 * time.Millisecond

// ErrLockBusy is returned when a run's advisory lock could not be acquired
// before the caller-supplied wait elapsed. It carries no PID or age
// information; the operating system - not this package - is responsible for
// releasing a lock left behind by a dead process.
var ErrLockBusy = errors.New("playtestenv: run lock is busy")

// runLock is an OS-native, cross-process advisory lock on one run directory.
// It uses Unix advisory locking and Windows LockFileEx via gofrs/flock. The
// operating system releases the lock automatically when the holding process
// exits, even if Close is never called.
type runLock struct {
	file *flock.Flock
}

// acquireRunLock tries to take the exclusive advisory lock at path, retrying
// every 50ms until either the lock is acquired or wait elapses.
//
// If the internal wait expires while the parent ctx is still live, it returns
// the sentinel ErrLockBusy. If the parent ctx itself is cancelled or expires
// first, that cancellation is preserved and returned unchanged so callers can
// distinguish "someone else holds this lock" from "my own operation was
// cancelled".
func acquireRunLock(ctx context.Context, path string, wait time.Duration) (*runLock, error) {
	lockCtx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()

	f := flock.New(path)
	ok, err := f.TryLockContext(lockCtx, 50*time.Millisecond)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			return nil, ErrLockBusy
		}
		return nil, err
	}
	if !ok {
		return nil, ErrLockBusy
	}
	return &runLock{file: f}, nil
}

// Close releases the advisory lock. It is safe to rely on OS-native release
// instead when a process cannot call Close, such as after a hard kill.
func (l *runLock) Close() error {
	return l.file.Unlock()
}
