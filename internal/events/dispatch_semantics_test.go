package events

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDispatchRoutesOnlyMatchingTypes documents the dispatch guarantee that
// makes the unchecked `evt := e.(events.X)` form in ~14 hook files safe.
//
// It exists because that form was flagged as "one uncomment away from a
// guaranteed panic": internal/hooks/hooks.go carries a commented-out debug
// listener registered with RegisterListener(nil, ...), and the concern was that
// enabling it would start feeding every event to every listener.
//
// It would not. A nil/`*` registration lands in its own eventListeners["*"]
// bucket, which DoListeners walks SEPARATELY from eventListeners[e.Type()].
// Enabling the wildcard routes all events to the wildcard listener only;
// type-specific listeners keep receiving only their own type. This test pins
// that, so nobody rewrites those 14 files on a false premise — or, worse,
// re-enables the debug hook expecting a crash and concludes something else is
// wrong when it works fine.
func TestDispatchRoutesOnlyMatchingTypes(t *testing.T) {
	mudlog.SetupLogger(nil, `LOW`, ``, false)

	ClearListeners()
	defer ClearListeners()

	var wildcardSaw, buffListenerSaw []string

	// Registered exactly the way the commented-out debug hook does.
	RegisterListener(nil, func(e Event) ListenerReturn {
		wildcardSaw = append(wildcardSaw, e.Type())
		return Continue
	})

	RegisterListener(Buff{}, func(e Event) ListenerReturn {
		buffListenerSaw = append(buffListenerSaw, e.Type())
		return Continue
	})

	// Dispatch an event of a type the Buff listener did not register for.
	DoListeners(Quest{})

	assert.Equal(t, []string{"Quest"}, wildcardSaw,
		"a wildcard listener must receive every event type")
	assert.Empty(t, buffListenerSaw,
		"a type-specific listener must NEVER receive a foreign event type — this is what "+
			"makes the unchecked type assertions in internal/hooks safe")

	// And it still receives its own type.
	DoListeners(Buff{BuffId: 1})
	require.Equal(t, []string{"Buff"}, buffListenerSaw,
		"a type-specific listener must still receive its own type")
}
