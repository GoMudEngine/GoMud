package util

import (
	"testing"
	"time"
)

// TestMudLockIsNotReentrant documents the invariant that forces copyover to
// have two entry points (triggerCopyover / triggerCopyoverLocked).
//
// The admin `copyover` command runs inside MainWorker's locked EventLoop, while
// the SIGUSR1 handler runs on its own unlocked goroutine. Because this mutex is
// not reentrant, a single entry point that acquired the lock unconditionally
// would deadlock the admin path, and one that never acquired it would let the
// signal path serialise world state to disk while the tick loop mutates it.
//
// If this ever becomes reentrant, that split can be collapsed.
func TestMudLockIsNotReentrant(t *testing.T) {
	LockMud()

	acquired := make(chan struct{})
	go func() {
		LockMud()
		close(acquired)
		UnlockMud()
	}()

	select {
	case <-acquired:
		UnlockMud()
		t.Fatal("a second LockMud succeeded while the lock was held — the mutex is reentrant " +
			"or broken, and copyover's two-entry-point split needs revisiting")
	case <-time.After(50 * time.Millisecond):
		// Correct: the second acquisition is blocked.
	}

	UnlockMud()

	select {
	case <-acquired:
		// Correct: releasing let the waiter through.
	case <-time.After(2 * time.Second):
		t.Fatal("UnlockMud did not release a waiting LockMud")
	}
}
