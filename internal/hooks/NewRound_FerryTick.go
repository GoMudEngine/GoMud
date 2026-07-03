package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/ferry"
)

// FerryTick reconciles every ferry vessel to its clock-derived state.
// The FerriesEnabled gate lives inside ferry.Tick().
func FerryTick(e events.Event) events.ListenerReturn {
	ferry.Tick()
	return events.Continue
}
