package questengine

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// TestRecoverQuestSequence_ContainsPanicInGoroutine locks the fix for the
// 2026-07-20 audit finding 1.3.
//
// QueueSequence delivers delayed dialogue lines and on_complete actions from
// bare timer goroutines. Those goroutines run arbitrary content-authored quest
// actions, and Go cannot recover a child goroutine's panic from its parent — so
// before this fix, a malformed quest sequence killed the whole server. Because
// the trigger is content rather than code, that was a realistic failure mode
// rather than a theoretical one.
//
// This is a real end-to-end check rather than a unit test of the helper: if
// recoverQuestSequence failed to contain the panic, the panic would propagate
// out of the goroutine and take down the entire test binary, so the test cannot
// pass by accident.
func TestRecoverQuestSequence_ContainsPanicInGoroutine(t *testing.T) {
	mudlog.SetupLogger(nil, `LOW`, ``, false)

	done := make(chan struct{})

	go func() {
		defer close(done)
		defer recoverQuestSequence("test stage")

		panic("simulated malformed quest sequence")
	}()

	select {
	case <-done:
		// Survived: the panic was contained inside the goroutine.
	case <-time.After(5 * time.Second):
		t.Fatal("quest sequence goroutine never completed")
	}
}

// TestRecoverQuestSequence_NoPanicIsANoop confirms the helper does not interfere
// with the normal path.
func TestRecoverQuestSequence_NoPanicIsANoop(t *testing.T) {
	mudlog.SetupLogger(nil, `LOW`, ``, false)

	ran := false
	func() {
		defer recoverQuestSequence("test stage")
		ran = true
	}()

	if !ran {
		t.Fatal("the guarded function body did not run")
	}
}
