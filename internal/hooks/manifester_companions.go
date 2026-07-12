package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const empoweredByBondBuff = 105

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
