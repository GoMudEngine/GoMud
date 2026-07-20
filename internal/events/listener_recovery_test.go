package events

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegression_PanickingListenerDoesNotEscape locks the fix for the
// 2026-07-20 audit finding 1.1.
//
// DoListeners is the single dispatch point for every combat round, quest
// event, command execution and mob AI tick — a surface spanning ~150 files. It
// invoked each listener directly with no panic recovery, and nothing higher up
// the chain (ProcessEvents, EventLoop, MainWorker) recovered either. A nil
// dereference, failed type assertion or out-of-range index anywhere in gameplay
// code therefore killed the entire process and disconnected every player.
//
// The project already applied this discipline to individual callbacks
// (behaviortree/actions_goal.go, goals/select.go, goals/store.go), each
// commented about protecting the engine round tick — it had simply never been
// extended to the loop those callbacks run inside.
func TestRegression_PanickingListenerDoesNotEscape(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	t.Run("panic_is_contained", func(t *testing.T) {
		ClearListeners()
		defer ClearListeners()

		RegisterListener(Buff{}, func(e Event) ListenerReturn {
			panic("listener blew up")
		})

		// The bare call must not panic. Without recovery this takes down the
		// process, so a plain assert.NotPanics is the whole point of the test.
		assert.NotPanics(t, func() {
			DoListeners(Buff{BuffId: 1})
		}, "a panicking listener must not escape DoListeners")
	})

	t.Run("sibling_listeners_still_run", func(t *testing.T) {
		ClearListeners()
		defer ClearListeners()

		firstRan, thirdRan := false, false

		RegisterListener(Buff{}, func(e Event) ListenerReturn {
			firstRan = true
			return Continue
		})
		RegisterListener(Buff{}, func(e Event) ListenerReturn {
			panic("middle listener blew up")
		})
		RegisterListener(Buff{}, func(e Event) ListenerReturn {
			thirdRan = true
			return Continue
		})

		require.NotPanics(t, func() {
			DoListeners(Buff{BuffId: 1})
		})

		assert.True(t, firstRan, "listeners before the panicking one must run")
		assert.True(t, thirdRan,
			"a panicking listener must not prevent later listeners from running")
	})

	t.Run("panicking_listener_yields_continue", func(t *testing.T) {
		ClearListeners()
		defer ClearListeners()

		RegisterListener(Buff{}, func(e Event) ListenerReturn {
			panic("boom")
		})

		var got ListenerReturn
		require.NotPanics(t, func() {
			got = DoListeners(Buff{BuffId: 1})
		})

		assert.Equal(t, Continue, got,
			"a panicking listener must be treated as Continue, not silently cancel the event")
	})

	// The lock is held across the whole dispatch. If a recovered panic left it
	// held, every subsequent event would deadlock — so prove dispatch still
	// works afterwards.
	t.Run("dispatch_still_works_after_a_panic", func(t *testing.T) {
		ClearListeners()
		defer ClearListeners()

		RegisterListener(Buff{}, func(e Event) ListenerReturn {
			panic("boom")
		})

		require.NotPanics(t, func() { DoListeners(Buff{BuffId: 1}) })

		// A second, independent dispatch must not deadlock or misbehave.
		ClearListeners()
		ran := false
		RegisterListener(Buff{}, func(e Event) ListenerReturn {
			ran = true
			return Continue
		})

		require.NotPanics(t, func() { DoListeners(Buff{BuffId: 2}) })
		assert.True(t, ran, "the listener lock must not be left held after a recovered panic")
	})

	// Non-panicking behaviour must be untouched: Cancel still short-circuits.
	t.Run("cancel_still_short_circuits", func(t *testing.T) {
		ClearListeners()
		defer ClearListeners()

		laterRan := false
		RegisterListener(Buff{}, func(e Event) ListenerReturn {
			return Cancel
		})
		RegisterListener(Buff{}, func(e Event) ListenerReturn {
			laterRan = true
			return Continue
		})

		got := DoListeners(Buff{BuffId: 1})

		assert.Equal(t, Cancel, got, "Cancel must still propagate")
		assert.False(t, laterRan, "Cancel must still stop later listeners")
	})
}
