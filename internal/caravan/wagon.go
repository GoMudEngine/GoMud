package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// WagonMobId is the mob template ID of the caravan wagon — the
// cargo-bearing follower that travels with each caravan leader.
// Its inventory is the source of truth for caravan cargo.
const WagonMobId = 374

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
