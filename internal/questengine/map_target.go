package questengine

import (
	"fmt"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// mobRoomLookup returns the current room of the single live instance of the
// given mob template, and whether exactly one was found.
//
// Zero instances (dead, or not spawned yet) and two-or-more (a generic template
// with no single correct answer) both report false — the caller then falls back
// rather than guessing. A marker pointing at the wrong one of five identical
// mobs is worse than no marker at all.
//
// It is a package-level var so tests can stub it without a running world.
var mobRoomLookup = func(mobId int) (int, bool) {
	roomId, found := 0, 0
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m == nil || int(m.MobId) != mobId {
			continue
		}
		found++
		if found > 1 {
			return 0, false // ambiguous — decline
		}
		roomId = m.Character.RoomId
	}
	if found != 1 || roomId < 1 {
		return 0, false
	}
	return roomId, true
}

// ResolveQuestTarget returns the room id the given quest points at during
// currentStep, for the minimap destination marker. It returns 0 when there is
// no resolvable target (the client then draws no marker — no guessing).
//
// Resolution order:
//  1. map_target: -1 — a deliberate "no marker". Wins over everything,
//     including map_target_mob, so an author can keep a marker switched off.
//  2. map_target_mob: >0 — the CURRENT room of that NPC. This is the right
//     choice for "go see <named NPC>", because a fixed room is wrong for part
//     of every day for any NPC on a schedule. Declines (falls through) when the
//     NPC is unspawned or the template has multiple live instances.
//  3. map_target: >0 — a fixed room. Also serves as the fallback for (2).
//  4. Inference: a room_enter trigger gated on the current step token
//     (conditions.has contains "{questId}-{currentStep}") — its room.
//  5. 0.
func (e *Engine) ResolveQuestTarget(questId int, currentStep string) int {
	if currentStep == "" || currentStep == "end" {
		return 0
	}
	q := e.quests[questId]
	if q == nil {
		return 0
	}

	for _, step := range q.Steps {
		if step.Id != currentStep {
			continue
		}

		// 1. Deliberate no-marker beats every other source.
		if step.MapTarget == -1 {
			return 0
		}

		// 2. Follow the NPC, when we can name exactly one.
		if step.MapTargetMob > 0 {
			if roomId, ok := mobRoomLookup(step.MapTargetMob); ok {
				return roomId
			}
			// Unresolvable: fall through to the static fallback / inference.
		}

		// 3. Fixed room.
		if step.MapTarget > 0 {
			return step.MapTarget
		}

		break // nothing on the step → fall through to room_enter inference
	}

	// 4. room_enter trigger gated on the current step.
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
