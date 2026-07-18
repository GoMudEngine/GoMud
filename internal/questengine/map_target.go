package questengine

import (
	"fmt"
	"slices"
)

// ResolveQuestTarget returns the room id the given quest points at during
// currentStep, for the minimap destination marker. It returns 0 when there is
// no resolvable target (the client then draws no marker — no guessing).
//
// Resolution order:
//  1. The current step's explicit map_target.
//  2. Inference: a room_enter trigger gated on the current step token
//     (conditions.has contains "{questId}-{currentStep}") — its room.
//  3. 0.
func (e *Engine) ResolveQuestTarget(questId int, currentStep string) int {
	if currentStep == "" || currentStep == "end" {
		return 0
	}
	q := e.quests[questId]
	if q == nil {
		return 0
	}

	// 1. Explicit map_target on the current step.
	for _, step := range q.Steps {
		if step.Id == currentStep {
			if step.MapTarget != 0 {
				return step.MapTarget
			}
			break
		}
	}

	// 2. room_enter trigger gated on the current step.
	stepToken := fmt.Sprintf("%d-%s", questId, currentStep)
	for i := range q.Triggers {
		t := &q.Triggers[i]
		if t.Event != "room_enter" || t.Room == 0 {
			continue
		}
		if slices.Contains(t.Conditions.Has, stepToken) {
			return t.Room
		}
	}

	return 0
}
