package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

const ruleNameAggressiveActionToRevenge = "aggressive_action_to_revenge"
const aggressiveVictimRevengePriority = 75
const aggressiveWitnessRevengePriority = 50

func init() {
	Register(ruleNameAggressiveActionToRevenge, aggressiveActionToRevenge, "PlayerAttackedMob")
}

// aggressiveActionToRevenge: on PlayerAttackedMob, seed revenge-mob into
// the attacked mob (priority 75) and non-hostile witnesses in the room
// (priority 50). Skips already-auto-hostile mobs (revenge would be
// redundant noise — they already attack on sight).
//
// Shares the PlayerAttackedMob event with rule 9 (combat_assist_to_opinion_boost).
// Both rules fire on the same event; their logic is independent.
//
// Firer: internal/actions/attack.go + internal/actions/taunt.go (Task 3).
func aggressiveActionToRevenge(event events.Event) {
	pa, ok := event.(events.PlayerAttackedMob)
	if !ok {
		return
	}
	if pa.UserId == 0 || pa.MobInstanceId == 0 {
		return
	}

	attackedMob := mobs.GetInstance(pa.MobInstanceId)
	if attackedMob == nil {
		return
	}
	// AutoAggro mobs already attack players on sight — seeding revenge
	// into them is redundant noise; skip.
	if attackedMob.AutoAggro {
		return
	}

	// Classify and respond for the attacked mob at victim priority.
	seedWitnessResponse(attackedMob, pa.UserId, aggressiveVictimRevengePriority)

	// Seed revenge into non-hostile witnesses sharing the same room at
	// witness priority (lower than the direct victim — they aren't the
	// immediate target of the attack).
	room := rooms.LoadRoom(attackedMob.Character.RoomId)
	if room == nil {
		return
	}
	for _, witnessInstId := range room.GetMobs() {
		if witnessInstId == attackedMob.InstanceId {
			continue
		}
		witness := mobs.GetInstance(witnessInstId)
		if witness == nil || witness.AutoAggro {
			continue
		}
		seedWitnessResponse(witness, pa.UserId, aggressiveWitnessRevengePriority)
	}
}
