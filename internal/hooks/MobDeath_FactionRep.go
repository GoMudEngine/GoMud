package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// MobDeathFactionRep records a murder crime + adjusts faction rep
// when a player kills a faction-member mob. Rewritten in chunk 1.3
// to be crime-aware:
//
//   - Each defined faction the dead mob belonged to gets a crime
//     row recorded.
//   - Witness check determines whether the perpetrator is identified
//     (player) or unknown (lone murder, no rep impact).
//   - When an unresolved assault crime exists for this player +
//     faction within the lookback window, that row is upgraded
//     in-place to murder; rep delta becomes the incremental
//     CrimeRepDeltaMurder - CrimeRepDeltaAssault. Otherwise a new
//     murder row is recorded with the full CrimeRepDeltaMurder.
//   - Party-room propagation from chunk 1.2 is preserved: any party
//     member of a damager who's in the same room at death-time also
//     takes the bump.
func MobDeathFactionRep(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}
	if len(evt.PlayerDamage) == 0 {
		return events.Continue
	}

	// Use the mob TEMPLATE — instance is destroyed by the time this
	// queued event fires (suicide.go calls DestroyInstance after
	// queuing MobDeath).
	spec := mobs.GetMobSpec(mobs.MobId(evt.MobId))
	if spec == nil {
		return events.Continue
	}
	factionIds := factions.FactionsForMob(spec)
	if len(factionIds) == 0 {
		return events.Continue
	}

	room := rooms.LoadRoom(evt.RoomId)
	if room == nil {
		return events.Continue
	}

	// Witnesses: faction-aligned mob instances in the room, EXCLUDING
	// the victim's instance (victim is dead, not a self-witness).
	witnesses := crimes.WitnessesInRoom(factionIds, room, evt.InstanceId)

	// Build the player set: damagers + same-room party members.
	toBump := buildKillerSet(evt, room)

	cfg := configs.GetBalanceConfig()
	deltaMurder := int(cfg.CrimeRepDeltaMurder)
	deltaAssault := int(cfg.CrimeRepDeltaAssault)

	currentExternal := len(witnesses) > 0

	for userId := range toBump {
		perp := crimes.IdentifiedPerp(userId, witnesses)
		for _, fid := range factionIds {
			// Upgrade-in-place if a recent unresolved assault exists.
			if assault := crimes.FindRecentAssault(fid, userId, 100); assault != nil {
				// Four-case model for assault → murder upgrade:
				//
				// Case A — external witness now: upgrade perp to
				//   identified, pay incremental delta.
				// Case B — no external witness now, but assault WAS
				//   externally witnessed: keep the existing identified
				//   perp (assault was seen), no additional rep delta
				//   (no one saw the killing blow).
				// Case C — no external witness now AND assault had no
				//   external witness (lone assault → lone murder): set
				//   perp=unknown and REFUND the assault rep delta.
				if currentExternal {
					// Case A: full identification path.
					crimes.UpgradeAssaultToMurder(fid, assault.Id, perp,
						evt.InstanceId, evt.RoomId, spec.Zone, false)
					if perp.Type == crimes.PerpPlayer {
						// Pay the incremental difference (assault already paid).
						factions.BumpRep(fid, userId, deltaMurder-deltaAssault)
					}
					writeKnowledgeForWitnesses(witnesses, perp, []int{assault.Id}, evt.RoomId)
				} else if assault.HadExternalWitness {
					// Case B: assault was identified, kill was not witnessed.
					// Preserve the existing perp; no new rep impact.
					crimes.UpgradeAssaultToMurder(fid, assault.Id,
						crimes.Perpetrator{}, // ignored: preserveExistingPerp=true
						evt.InstanceId, evt.RoomId, spec.Zone, true)
					// No additional rep bump.
					// No knowledge writes: witnesses is empty (currentExternal==false),
					// so any call here would be a no-op. The assault-time knowledge
					// writes from T15 already covered everyone who saw the first blow.
				} else {
					// Case C: lone assault → lone murder. Nobody survives
					// to identify the killer. Set perp=unknown and refund
					// the assault rep delta.
					crimes.UpgradeAssaultToMurder(fid, assault.Id,
						crimes.Perpetrator{Type: crimes.PerpUnknown},
						evt.InstanceId, evt.RoomId, spec.Zone, false)
					// deltaAssault is negative (e.g. -10); negate to refund.
					factions.BumpRep(fid, userId, -deltaAssault)
					// No knowledge writes: witnesses is empty (currentExternal==false).
					// Perp is PerpUnknown so writeKnowledgeForWitnesses would early-return
					// anyway; remove the call to avoid misleading "what does this do?"
					// confusion during future reads.
				}
				continue
			}

			// Fresh murder record (no prior assault row).
			// hadExternalWitness is false — this is a direct murder,
			// not an assault-that-escalated.
			crimeIds := crimes.Record([]string{fid}, crimes.KindMurder, perp,
				spec, evt.InstanceId, evt.RoomId, spec.Zone, false)
			if perp.Type == crimes.PerpPlayer {
				factions.BumpRep(fid, userId, deltaMurder)
			}
			writeKnowledgeForWitnesses(witnesses, perp, crimeIds, evt.RoomId)
		}
	}

	return events.Continue
}

// writeKnowledgeForWitnesses writes a knowledge record for each witness
// linking them to the given crime IDs and the player perp. Skips when
// perp is not a player (e.g. PerpUnknown in Case C).
func writeKnowledgeForWitnesses(witnesses []int, perp crimes.Perpetrator,
	crimeIds []int, roomId int) {
	if perp.Type != crimes.PerpPlayer {
		return
	}
	subject := knowledge.PlayerSubject(perp.Id)
	for _, witnessInstId := range witnesses {
		w := mobs.GetInstance(witnessInstId)
		if w == nil {
			continue
		}
		for _, crimeId := range crimeIds {
			knowledge.RecordCrimeWitnessed(int(w.MobId), subject, crimeId)
		}
		knowledge.RecordMet(int(w.MobId), subject, roomId, knowledge.SourceWitnessed)
	}
}

// buildKillerSet returns the set of userIds who participated in the
// kill: direct damagers from PlayerDamage, plus any same-room party
// members of those damagers (for pure-support healers etc.).
// Mirrors the chunk 1.2 logic.
func buildKillerSet(evt events.MobDeath, room *rooms.Room) map[int]struct{} {
	out := make(map[int]struct{}, len(evt.PlayerDamage))
	for userId := range evt.PlayerDamage {
		out[userId] = struct{}{}
	}
	for damagerUserId := range evt.PlayerDamage {
		party := parties.Get(damagerUserId)
		if party == nil {
			continue
		}
		for _, memberId := range party.UserIds {
			if _, already := out[memberId]; already {
				continue
			}
			u := users.GetByUserId(memberId)
			if u == nil {
				continue
			}
			if u.Character.RoomId == evt.RoomId {
				out[memberId] = struct{}{}
			}
		}
	}
	return out
}
