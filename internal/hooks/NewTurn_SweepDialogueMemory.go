package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/dialogue"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// dialogueMemorySweepTurnInterval is how many turns pass between sweeps. The
// cache is small per entry and the sweep is a full map walk, so there is no
// reason to run it often — the point is only that it runs at all.
const dialogueMemorySweepTurnInterval uint64 = 500

// SweepDialogueMemory drops per-(mob instance, player) conversation memories
// that have gone untouched for a long time.
//
// The cache gains one entry per pair that has ever spoken and, until this hook
// existed, was only emptied by an explicit ResetMemory — so it grew for the
// life of the process, holding entries for mob instances that despawned long
// before. The state is per-instance and already lost on restart, so dropping an
// idle entry costs nothing that a respawn would not.
func SweepDialogueMemory(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.NewTurn)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "NewTurn", "Actual Type", e.Type())
		return events.Cancel
	}

	if evt.TurnNumber%dialogueMemorySweepTurnInterval != 0 {
		return events.Continue
	}

	if removed := dialogue.SweepMemories(); removed > 0 {
		mudlog.Info("SweepDialogueMemory", "removed", removed)
	}

	return events.Continue
}
