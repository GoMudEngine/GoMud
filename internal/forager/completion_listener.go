package forager

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// foragerDeliveryPatrols is the set of oneshot patrol ids that, when
// completed, should advance a forager's StateDelivering to the next
// state (StateStoring or StateRecalling).
var foragerDeliveryPatrols = map[string]struct{}{
	"marsh_forager_delivery":  {},
	"steppe_forager_delivery": {},
}

// behaviorStateSetter is the minimal interface the completion listener
// needs to mutate the forager's btree state-machine keys. The concrete
// *behaviortree.BehaviorState satisfies this implicitly. Defined here
// to avoid an internal/forager -> internal/behaviortree import cycle
// (behaviortree already imports forager).
type behaviorStateSetter interface {
	Set(key string, value any)
}

// ForagerCompletionListener consumes events.PatrolCompleted for the
// forager-delivery oneshot patrols. When the patrol finishes (forager
// is back at sanctuary), advance the forager's BTreeState[forager_state]
// from delivering to either storing (chest + unsold items) or
// recalling.
//
// Non-forager patrols, unknown mobs, and BTreeState that doesn't
// satisfy the setter interface are silently ignored. Chunk 3.8.
func ForagerCompletionListener(e events.Event) events.ListenerReturn {
	pc, ok := e.(events.PatrolCompleted)
	if !ok {
		return events.Continue
	}
	if _, isForagerPatrol := foragerDeliveryPatrols[pc.PatrolId]; !isForagerPatrol {
		return events.Continue
	}
	forager := mobs.GetInstance(pc.MobInstanceId)
	if forager == nil {
		return events.Continue
	}
	bs, ok := forager.BTreeState.(behaviorStateSetter)
	if !ok {
		return events.Continue
	}

	// Decide next state: Storing if forager has a chest AND carries
	// unsold items, otherwise Recalling.
	nextState := StateRecalling
	if forager.StorageChestRoom > 0 && len(forager.Character.Items) > 0 {
		nextState = StateStoring
	}

	bs.Set("forager_state", nextState.Name())
	bs.Set("forager_state_started_round", strconv.FormatUint(util.GetRoundCount(), 10))
	// Reset the legacy keyVisitIndex counter (set by the pre-3.8
	// delivery loop) so a future cycle restart starts clean.
	bs.Set("forager_visit_index", "0")

	return events.Continue
}
