package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const ruleNameFactionRepCounter = "faction_rep_counter"

func init() {
	// Registered for Communication — the closest available event that
	// involves a mob-to-player or player-to-mob social interaction.
	//
	// STUB NOTICE (chunk 4.5): events.Communication has CommType (say,
	// whisper, shout, party, broadcast) and no subtype distinguishing
	// "positive interaction" from neutral or hostile speech. This rule
	// therefore EARLY-RETURNS on every event — it is effectively a no-op
	// until a richer signal is available.
	//
	// TODO-ADAPT (future chunk): wire a dedicated event (e.g. GiftAccepted,
	// a TradeCompleted, or a QuestTurnedIn carrying the quest-giver mob
	// instance id) that unambiguously signals a positive player→mob
	// interaction. Then replace the early-return body with:
	//   1. Extract (playerUserId, targetMobInstanceId) from the event.
	//   2. Resolve targetMob via mobs.GetInstance.
	//   3. Walk factions.FactionsForMob(targetMob) and bump
	//      "faction_rep_built_with:<factionId>" on the targetMob.
	//
	// The 4.3 catalog's befriend-faction Predicate reads this counter
	// (docs/superpowers/specs/…-reactive-goal-generation-design.md §3.5)
	// and will produce zero until this stub is completed.
	Register(ruleNameFactionRepCounter, factionRepCounter, "Communication")
}

// factionRepCounter is the chunk-4.5 stub for bumping
// faction_rep_built_with:<factionId> on a mob when a player engages
// in a positive interaction with that mob (gift, trade, quest turn-in).
//
// Registered for events.Communication but early-returns on all events
// because Communication carries no "positive interaction" signal —
// CommType values are say/whisper/shout/party/broadcast. The references
// to mobs.GetInstance and factions.FactionsForMob are preserved so that
// a future implementer only needs to fill in the event-extraction step.
//
// See the init() comment above for the full TODO-ADAPT path.
func factionRepCounter(event events.Event) {
	_, ok := event.(events.Communication)
	if !ok {
		return
	}
	// No CommType value cleanly signals a positive player→mob interaction.
	// Early-return until a richer event type is available (see init comment).
	//
	// Future body shape (do NOT fill in CommType checks — they would
	// fire on all chats and produce noisy, incorrect reputation bumps):
	//   targetMob := mobs.GetInstance(positiveInteractionMobId)
	//   for _, fid := range factions.FactionsForMob(targetMob) {
	//       bumpMiscInt(targetMob, "faction_rep_built_with:"+fid, 1)
	//   }
	_ = mobs.GetInstance
	_ = factions.FactionsForMob
}
