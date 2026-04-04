package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Flee makes a mob disengage from combat and move to a random adjacent room.
// The flee attempt is gated by an opposed roll: mob Dex+Skullduggery vs each
// attacker's Dex+UnarmedCombat. If any attacker wins the roll, the mob is
// blocked and stays in the room.
func Flee(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Check players fighting this mob
	for _, uId := range room.GetPlayers(rooms.FindFightingMob) {
		u := users.GetByUserId(uId)
		if u == nil {
			continue
		}
		if u.Character.Aggro == nil || u.Character.Aggro.MobInstanceId != mob.InstanceId {
			continue
		}

		fleeScore := float64(mob.Character.Stats.Dexterity.ValueAdj +
			mob.Character.GetSkillLevel(skills.Skullduggery)*25)
		blockScore := float64(u.Character.Stats.Dexterity.ValueAdj +
			u.Character.GetSkillLevel(skills.UnarmedCombat)*25)
		success, _, _, _ := dice.OpposedRollStat(fleeScore, blockScore)
		if !success {
			sendRoomText(room,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to flee but is blocked!`, mob.Character.Name))
			return true, nil
		}
	}

	// Check mobs fighting this mob
	for _, mId := range room.GetMobs(rooms.FindFightingMob) {
		m := mobs.GetInstance(mId)
		if m == nil {
			continue
		}
		if m.Character.Aggro == nil || m.Character.Aggro.MobInstanceId != mob.InstanceId {
			continue
		}

		fleeScore := float64(mob.Character.Stats.Dexterity.ValueAdj +
			mob.Character.GetSkillLevel(skills.Skullduggery)*25)
		blockScore := float64(m.Character.Stats.Dexterity.ValueAdj +
			m.Character.GetSkillLevel(skills.UnarmedCombat)*25)
		success, _, _, _ := dice.OpposedRollStat(fleeScore, blockScore)
		if !success {
			sendRoomText(room,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to flee but is blocked!`, mob.Character.Name))
			return true, nil
		}
	}

	// If in combat, clear aggro
	if mob.Character.Aggro != nil {
		mob.Character.EndAggro()
	}

	// Get a random exit (skips secret and locked exits)
	exitName, _ := room.GetRandomExit()
	if exitName == "" {
		// Cornered — no exits available
		return true, nil
	}

	// Send flee message before moving
	sendRoomText(room,
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> flees!`, mob.Character.Name))

	// Move via the existing Go command
	Go(exitName, mob, room)

	return true, nil
}
