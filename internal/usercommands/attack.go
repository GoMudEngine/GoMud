package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Attack(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	attackPlayerId := 0
	attackMobInstanceId := 0

	if rest == `` {
		partyInfo := parties.Get(user.UserId)

		// If no argument supplied, attack whoever is attacking the player currently.
		for _, mId := range room.GetMobs(rooms.FindFightingPlayer) {
			m := mobs.GetInstance(mId)
			if m.Character.Aggro == nil {
				continue
			}

			if m.Character.Aggro.UserId == user.UserId {
				attackMobInstanceId = m.InstanceId
				break
			}

			if partyInfo != nil {
				if partyInfo.IsMember(m.Character.Aggro.UserId) {
					attackMobInstanceId = m.InstanceId
					break
				}
			}
		}

		if attackMobInstanceId == 0 {
			for _, uId := range room.GetPlayers(rooms.FindFightingPlayer) {
				u := users.GetByUserId(uId)
				if u.Character.Aggro == nil {
					continue
				}

				if u.Character.Aggro.UserId == user.UserId {
					attackPlayerId = u.UserId
					break
				}

				if partyInfo != nil {
					if partyInfo.IsMember(u.Character.Aggro.UserId) {
						attackPlayerId = u.UserId
						break
					}
				}
			}
		}

		// Finally, if still no targets, check if any party members are aggroed and just glom onto that
		if attackMobInstanceId == 0 && attackPlayerId == 0 {
			if partyInfo != nil {
				for uId := range partyInfo.GetMembers() {
					if partyUser := users.GetByUserId(uId); partyUser != nil {
						if partyUser.Character.Aggro == nil {
							continue
						}

						if partyUser.Character.Aggro.MobInstanceId > 0 {
							attackMobInstanceId = partyUser.Character.Aggro.MobInstanceId
							break
						}

						if partyUser.Character.Aggro.UserId > 0 {
							attackPlayerId = partyUser.Character.Aggro.UserId
							break
						}

					}
				}
			}
		}

	} else {
		// Wildcard and named-target resolution delegated to shared helper.
		t := actions.FindAttackTarget(rest, room, user.UserId, 0)
		attackPlayerId = t.UserId
		attackMobInstanceId = t.MobInstanceId
	}

	if attackMobInstanceId == 0 && attackPlayerId == 0 {
		user.SendText("You attack the darkness!")
		return true, nil
	}

	isSneaking := user.Character.HasBuffFlag(buffs.Hidden)

	/*
		combatAddlWaitRounds := user.Character.Equipment.Weapon.GetSpec().WaitRounds + user.Character.Equipment.Weapon.GetSpec().WaitRounds
		attkType := characters.DefaultAttack
		if user.Character.Equipment.Weapon.GetSpec().Subtype == items.Shooting {
			attkType = characters.Shooting
		}
	*/

	// --- TARGET SWITCHING LOGIC (Stage 7.4) ---
	// If already in combat and trying to attack a different target, use target switching
	if user.Character.Aggro != nil {
		currentTargetUserId := user.Character.Aggro.UserId
		currentTargetMobId := user.Character.Aggro.MobInstanceId

		isDifferentTarget := false
		if attackMobInstanceId > 0 && attackMobInstanceId != currentTargetMobId {
			isDifferentTarget = true
		}
		if attackPlayerId > 0 && attackPlayerId != currentTargetUserId {
			isDifferentTarget = true
		}

		// If switching targets, use the Target command logic instead
		if isDifferentTarget && (user.Character.Aggro.Type == characters.DefaultAttack || user.Character.Aggro.Type == characters.Shooting) {
			// Build target name for the Target command
			targetName := rest
			if targetName == "" || targetName[0] == '*' {
				// For empty or random targets, find the actual name
				if attackMobInstanceId > 0 {
					if m := mobs.GetInstance(attackMobInstanceId); m != nil {
						targetName = m.Character.Name
					}
				} else if attackPlayerId > 0 {
					if p := users.GetByUserId(attackPlayerId); p != nil {
						targetName = p.Character.Name
					}
				}
			}
			// Delegate to Target command for proper switching logic
			return Target(targetName, user, room, flags)
		}
	}
	// --- END TARGET SWITCHING LOGIC ---

	if attackMobInstanceId > 0 {

		m := mobs.GetInstance(attackMobInstanceId)

		if m != nil {
			dupIdx := room.GetMobDuplicateIndex(m.InstanceId)
			mName := m.Character.GetMobNameIndexed(user.UserId, dupIdx).String()

			if m.Character.IsCharmed(user.UserId) {
				user.SendText(fmt.Sprintf(`%s is your friend!`, mName))
				return true, nil
			}

			if m.IsNonCombatant() {
				user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, m.Character.Name))
				return true, nil
			}

			if party := parties.Get(user.UserId); party != nil {
				for _, id := range party.UserIds {
					if id == user.UserId {
						continue
					}
					if partyUser := users.GetByUserId(id); partyUser != nil {
						if partyUser.Character.RoomId == user.Character.RoomId &&
							partyUser.Character.GetSetting("autoattack") != "off" &&
							partyUser.Character.Aggro == nil {
							// Surprise attack for hidden party members before they join combat
							if partyUser.Character.HasBuffFlag(buffs.Hidden) {
								partyCfg := configs.GetBalanceConfig()
								if partyUser.Character.TryCooldown("special-move",
									fmt.Sprintf("%d rounds", partyCfg.SpecialMoveCooldown)) {
									executeSurpriseAttack(partyUser, room, attackMobInstanceId, 0)
								}
							}
							partyUser.Command(fmt.Sprintf(`attack #%d`, attackMobInstanceId))
						}
					}
				}
			}

			// Surprise attack from stealth — fires before normal combat begins
			if isSneaking {
				cfg := configs.GetBalanceConfig()
				if user.Character.TryCooldown("special-move",
					fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
					executeSurpriseAttack(user, room, attackMobInstanceId, 0)
				}
			}

			user.Character.SetAggro(0, attackMobInstanceId, characters.DefaultAttack)

			user.SendText(
				fmt.Sprintf(`You prepare to enter into mortal combat with %s.`, mName),
			)

			if !isSneaking {
				room.SendTextVisual(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to fight %s.`, user.Character.Name, mName),
					user.UserId,
				)
			}

			for _, instId := range room.GetMobs(rooms.FindCharmed) {
				if m := mobs.GetInstance(instId); m != nil {
					if m.Character.Aggro == nil && m.Character.IsCharmed(user.UserId) {
						// Only auto-assist if the companion has AutoAssist enabled
						comp := user.Character.GetCompanionByInstanceId(instId)
						if comp != nil && comp.AutoAssist {
							m.Command(fmt.Sprintf(`attack #%d`, attackMobInstanceId))
						}
					}
				}
			}

		}

	} else if attackPlayerId > 0 {

		if p := users.GetByUserId(attackPlayerId); p != nil {

			if pvpErr := room.CanPvp(user, p); pvpErr != nil {
				user.SendText(pvpErr.Error())
				return true, nil
			}

			if partyInfo := parties.Get(user.UserId); partyInfo != nil {
				if partyInfo.IsMember(attackPlayerId) {
					user.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is in your party!`, p.Character.Name))
					return true, nil
				}
			}

			if party := parties.Get(user.UserId); party != nil {
				if party.IsLeader(user.UserId) {
					for _, id := range party.GetAutoAttackUserIds() {
						if id == user.UserId {
							continue
						}
						if partyUser := users.GetByUserId(id); partyUser != nil {
							if partyUser.Character.RoomId == user.Character.RoomId {
								partyUser.Command(fmt.Sprintf(`attack @%d`, attackPlayerId)) // # denotes a specific mob instanceId
							}
						}
					}
				}
			}

			// Surprise attack from stealth — fires before normal combat begins
			if isSneaking {
				cfg := configs.GetBalanceConfig()
				if user.Character.TryCooldown("special-move",
					fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
					executeSurpriseAttack(user, room, 0, attackPlayerId)
				}
			}

			user.Character.SetAggro(attackPlayerId, 0, characters.DefaultAttack)

			user.SendText(
				fmt.Sprintf(`You prepare to enter into mortal combat with <ansi fg="username">%s</ansi>.`, p.Character.Name),
			)

			if !isSneaking {

				p.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to fight you!`, user.Character.Name),
				)

				room.SendTextVisual(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to fight <ansi fg="mobname">%s</ansi>.`, user.Character.Name, p.Character.Name),
					user.UserId, attackPlayerId)
			}

			for _, instId := range room.GetMobs(rooms.FindCharmed) {
				if m := mobs.GetInstance(instId); m != nil {
					if m.Character.Aggro == nil && m.Character.IsCharmed(user.UserId) { // Charmed mobs help the player

						m.Command(fmt.Sprintf(`attack @%d`, attackPlayerId)) // @ denotes a specific user id

					}
				}
			}

		}

	}

	return true, nil
}
