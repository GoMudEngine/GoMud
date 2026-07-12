package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const (
	empoweredByBondBuff = 105
	broodSpawnMobId     = 9613
	broodFloorReserve   = 200 // cheap floor companion (before reduction); first-pass
)

// tickCompanionEmpowerment refreshes the "Empowered by Bond" buff (105) on the
// owner's in-room companions each round, while the owner holds a
// companion_empowerment mutation (Symbiotic Bond / Spirit Tether / Beast Bond).
//
// This is a DEDICATED empowerment applied by the owner's mutation — it does NOT
// mirror the owner's own buffs. That distinction matters: rally and warcry
// already fan their buffs out to companions (applyRallyToCompanions), so copying
// the owner's buff list here would double-apply them. A separate buff can't.
//
// Called per-round from UserRoundTick. Mirrors applyRallyToCompanions' fan-out.
func tickCompanionEmpowerment(user *users.UserRecord, room *rooms.Room) {
	if room == nil {
		return
	}
	if mutations.GetCompanionEmpowerment(user.Character.Mutations) <= 0 {
		return
	}
	for _, id := range user.Character.GetCharmIds() {
		mob := mobs.GetInstance(id)
		if mob == nil || mob.Character.RoomId != user.Character.RoomId {
			continue
		}
		mob.Character.AddBuff(empoweredByBondBuff, false)
	}
}

// hasAnyLiveCompanion reports whether the character currently has ANY live
// companion (registered + its mob instance still exists).
func hasAnyLiveCompanion(ch *characters.Character) bool {
	for i := range ch.Companions {
		if mobs.GetInstance(ch.Companions[i].InstanceId) != nil {
			return true
		}
	}
	return false
}

// tickBroodMotherFloor ensures a Brood Mother apex owner is NEVER petless: if
// they hold the apex (the "brood-mother" flag) and have no live companion at
// all, it forges a cheap brood spawn. Immediate respawn (no death delay) — the
// whole point is "always have something" — with only a failure back-off so we
// don't retry every round when the owner is at the companion cap. Called
// per-round from UserRoundTick.
func tickBroodMotherFloor(user *users.UserRecord, room *rooms.Room) {
	if room == nil {
		return
	}
	ch := user.Character
	if !mutations.HasMutationFlag(ch.Mutations, "brood-mother") {
		return
	}
	if hasAnyLiveCompanion(ch) {
		return
	}
	if ch.GetCooldown("brood-floor-respawn") > 0 {
		return
	}
	if spawnBroodFloor(user, room) == nil {
		ch.TryCooldown("brood-floor-respawn", "10 rounds")
	}
}

// spawnBroodFloor forges a single cheap brood-spawn companion for a Brood Mother
// who has no companion. Returns the mob, or nil on failure.
func spawnBroodFloor(user *users.UserRecord, room *rooms.Room) *mobs.Mob {
	ch := user.Character

	mob := mobs.NewMobByIdFresh(mobs.MobId(broodSpawnMobId), room.RoomId, 120)
	if mob == nil {
		return nil
	}
	room.AddMob(mob.InstanceId)
	mob.Character.Charm(user.UserId, 99999, "")
	mob.Character.EndAggro()
	ch.TrackCharmed(mob.InstanceId, true)

	reserve := ch.CalcCompanionReserve(broodFloorReserve)
	info := characters.CompanionInfo{
		MobId:             broodSpawnMobId,
		InstanceId:        mob.InstanceId,
		SourceType:        characters.CompanionSummoned,
		Name:              mob.Character.Name,
		BaseName:          mob.Character.Name,
		AutoAssist:        true,
		ConvictionReserve: reserve,
	}
	if !ch.AddCompanion(info) {
		ch.TrackCharmed(mob.InstanceId, false)
		room.RemoveMob(mob.InstanceId)
		mobs.DestroyInstance(mob.InstanceId)
		return nil
	}
	ch.RecalculateStats()
	return mob
}
