package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// runnerCircuitPatrols is the set of patrol ids that, when completed,
// trigger the runner → wagon residual-cargo return.
var runnerCircuitPatrols = map[string]struct{}{
	"thornwall_runner_circuit":  {},
	"stillwater_runner_circuit": {},
}

// CaravanRunnerCompletionListener consumes events.PatrolCompleted for
// the runner-circuit oneshot patrols. When Lars finishes his circuit
// (back at the depot terminal waypoint), transfer any residual cargo
// from his inventory back to the wagon.
//
// Non-runner patrols, missing instances, mob/template mismatch, and
// runner-without-wagon cases are silently ignored. Chunk 3.8.
func CaravanRunnerCompletionListener(e events.Event) events.ListenerReturn {
	pc, ok := e.(events.PatrolCompleted)
	if !ok {
		return events.Continue
	}
	if _, isRunnerCircuit := runnerCircuitPatrols[pc.PatrolId]; !isRunnerCircuit {
		return events.Continue
	}
	runner := mobs.GetInstance(pc.MobInstanceId)
	if runner == nil || int(runner.MobId) != RunnerMobId {
		return events.Continue
	}
	wagon := FindWagonInRoom(runner.Character.RoomId)
	if wagon == nil {
		// Wagon not co-located — caravan already departed or some
		// other anomaly. Cargo stays on Lars; the 5.3 depot-arrival
		// safety in handleDepotArrival (added in T15) will catch
		// this on the next caravan depot visit.
		return events.Continue
	}
	TransferAllCargoBack(runner, wagon)
	return events.Continue
}
