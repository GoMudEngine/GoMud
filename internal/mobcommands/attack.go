package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
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
			if m.Character.IsInCombat() && m.Character.CurrentCombatTarget().MobInstanceId == mob.InstanceId {
				attackMobInstanceId = m.InstanceId
				break
			}
		}

		if attackMobInstanceId == 0 {
			for _, uId := range room.GetPlayers(rooms.FindFightingMob) {
				u := users.GetByUserId(uId)
				if u.Character.IsInCombat() && u.Character.CurrentCombatTarget().MobInstanceId == mob.InstanceId {
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

	isSneaking := mob.Character.IsHidden()

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

			// Hidden mobs get a surprise attack on their first strike.
			// Don't clear the Hidden buff here — leave it for the combat
			// loop's CancelIfCombat pass so the surprise attack resolves
			// with the mob still hidden (backstab crit bonus).
			// Type the engagement from whether the burst actually fired, not
			// merely from being hidden: SurpriseAttack also gates on the
			// special-move cooldown, so a hidden-but-on-cooldown opener is an
			// ordinary attack.
			aggroType := characters.DefaultAttack
			if mob.Character.IsHidden() {
				// Pre-combat burst: fire per-weapon strikes from stealth.
				if targetUser := users.GetByUserId(attackPlayerId); targetUser != nil {
					mobActor := actions.NewMobActorInRoom(mob, room)
					targetActor := actions.NewUserActorInRoom(targetUser, room)
					if res := actions.SurpriseAttack(mobActor, actions.SurpriseAttackOpts{Target: targetActor}); res.Triggered {
						aggroType = characters.SurpriseAttack
					}
				}
			}
			// Only announce if not already fighting this target
			alreadyFighting := mob.Character.IsInCombat() && mob.Character.CurrentCombatTarget().UserId == attackPlayerId
			mob.Character.SetAggro(attackPlayerId, 0, aggroType)

			if !isSneaking && !alreadyFighting {

				if canSeeInDark(u, room) {
					u.SendText(messaging.CategoryHitMelee, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> prepares to fight you!`, mob.Character.Name))
				} else {
					u.SendText(messaging.CategoryHitMelee, `Something prepares to fight you!`)
				}

				sendRoomText(room, messaging.CategoryHitMelee,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> prepares to fight <ansi fg="username">%s</ansi>`, mob.Character.Name, u.Character.Name),
					u.UserId)

			}
		}

		return true, nil

	} else if attackMobInstanceId > 0 {

		m := mobs.GetInstance(attackMobInstanceId)

		if m != nil {

			// See above: keyed off the result, not IsHidden alone.
			mobAggroType := characters.DefaultAttack
			if mob.Character.IsHidden() {
				mob.Character.Validate(true)
				// Pre-combat burst: fire per-weapon strikes from stealth.
				mobActor := actions.NewMobActorInRoom(mob, room)
				targetActor := actions.NewMobActorInRoom(m, room)
				if res := actions.SurpriseAttack(mobActor, actions.SurpriseAttackOpts{Target: targetActor}); res.Triggered {
					mobAggroType = characters.SurpriseAttack
				}
			}
			mob.Character.SetAggro(0, attackMobInstanceId, mobAggroType)

			if !isSneaking {
				sendRoomText(room, messaging.CategoryHitMelee,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> prepares to fight <ansi fg="mobname">%s</ansi>`, mob.Character.Name, m.Character.Name))
			}

		}

		return true, nil
	}

	if !isSneaking {
		sendRoomText(room, messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> looks confused and upset.`, mob.Character.Name))
	}

	return true, nil
}
