package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Attack(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	attackPlayerId := 0
	attackMobInstanceId := 0

	if rest == `` {
		// If no argument supplied, attack whoever is attacking the player currently.
		for _, mId := range room.GetMobs(rooms.FindFightingMob) {
			m := mobs.GetInstance(mId)
			if m.Character.Aggro != nil && m.Character.Aggro.MobInstanceId == mob.InstanceId {
				attackMobInstanceId = m.InstanceId
				break
			}
		}

		if attackMobInstanceId == 0 {
			for _, uId := range room.GetPlayers(rooms.FindFightingMob) {
				u := users.GetByUserId(uId)
				if u.Character.Aggro != nil && u.Character.Aggro.MobInstanceId == mob.InstanceId {
					attackPlayerId = u.UserId
					break
				}
			}
		}
	} else {
		// Wildcard and named-target resolution delegated to shared helper.
		t := actions.FindAttackTarget(rest, room, 0, mob.InstanceId)
		attackPlayerId = t.UserId
		attackMobInstanceId = t.MobInstanceId
	}

	isSneaking := mob.Character.HasBuffFlag(buffs.Hidden)

	/*
		combatAddlWaitRounds := mob.Character.Equipment.Weapon.GetSpec().WaitRounds + mob.Character.Equipment.Weapon.GetSpec().WaitRounds
		attkType := characters.DefaultAttack
		if mob.Character.Equipment.Weapon.GetSpec().Subtype == items.Shooting {
			attkType = characters.Shooting
		}
	*/

	if attackPlayerId > 0 {

		u := users.GetByUserId(attackPlayerId)

		if u != nil {

			// Track that they've attacked this player
			mob.PlayerAttacked(attackPlayerId)

			// Hidden mobs get a surprise attack on their first strike
			aggroType := characters.DefaultAttack
			if mob.Character.HasBuffFlag(buffs.Hidden) {
				aggroType = characters.SurpriseAttack
				// Clear hidden: remove from permabuffs first so Validate
				// doesn't re-add it, then expire the active buff.
				mob.Character.RemovePermaBuff(9)
				mob.Character.CancelBuffsWithFlag(buffs.Hidden)
				mob.Character.Buffs.RemoveBuff(9)
				mob.Character.Validate(true)
			}
			mob.Character.SetAggro(attackPlayerId, 0, aggroType)

			if !isSneaking {

				if canSeeInDark(u, room) {
					u.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> prepares to fight you!`, mob.Character.Name))
				} else {
					u.SendText(`Something prepares to fight you!`)
				}

				sendRoomText(room,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> prepares to fight <ansi fg="username">%s</ansi>`, mob.Character.Name, u.Character.Name),
					u.UserId)

			}
		}

		return true, nil

	} else if attackMobInstanceId > 0 {

		m := mobs.GetInstance(attackMobInstanceId)

		if m != nil {

			mobAggroType := characters.DefaultAttack
			if mob.Character.HasBuffFlag(buffs.Hidden) {
				mobAggroType = characters.SurpriseAttack
				mob.Character.RemovePermaBuff(9)
				mob.Character.CancelBuffsWithFlag(buffs.Hidden)
				mob.Character.Buffs.RemoveBuff(9)
				mob.Character.Validate(true)
			}
			mob.Character.SetAggro(0, attackMobInstanceId, mobAggroType)

			if !isSneaking {
				sendRoomText(room,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> prepares to fight <ansi fg="mobname">%s</ansi>`, mob.Character.Name, m.Character.Name))
			}

		}

		return true, nil
	}

	if !isSneaking {
		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> looks confused and upset.`, mob.Character.Name))
	}

	return true, nil
}
