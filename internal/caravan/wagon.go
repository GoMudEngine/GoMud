package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// WagonMobId is the mob template ID of the caravan wagon — the
// cargo-bearing follower that travels with each caravan leader.
// Its inventory is the source of truth for caravan cargo.
const WagonMobId = 374

// LeaderMobId is the mob template ID of the caravan leader (Ketil).
const LeaderMobId = 357

// caravanMobIds is the complete set of mob template IDs that make up
// the caravan crew: leader, wagon, and guards.
var caravanMobIds = map[int]struct{}{
	357: {}, // Ketil (leader)
	358: {}, // Marta (guard)
	359: {}, // Lars (guard)
	374: {}, // caravan wagon
	375: {}, // Hob (guard)
	376: {}, // Bran (guard)
}

// IsCaravanMob reports whether the given mob template ID belongs to the
// caravan crew. Used by the room-change hook to decide whether to record
// knowledge observations for bystanders in the destination room.
func IsCaravanMob(mobTemplateId int) bool {
	_, ok := caravanMobIds[mobTemplateId]
	return ok
}

// FindWagonInRoom returns the wagon mob (WagonMobId) co-located in
// the given room, or nil if the wagon is not present (mid-respawn,
// followers lagging, or wagon wiped).
//
// Callers decide whether nil is fatal or recoverable.
func FindWagonInRoom(roomId int) *mobs.Mob {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return nil
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		if int(m.MobId) == WagonMobId {
			return m
		}
	}
	return nil
}
