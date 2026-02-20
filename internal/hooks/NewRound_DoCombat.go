package hooks

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func DoCombat(e events.Event) events.ListenerReturn {

	evt := e.(events.NewRound)

	//
	// Combat rounds
	//
	affectedPlayers1, affectedMobs1 := handlePlayerCombat(evt)

	affectedPlayers2, affectedMobs2 := handleMobCombat(evt)

	// Do any resolution or extra checks based on everyone that has been involved in combat this round.
	handleAffected(append(affectedPlayers1, affectedPlayers2...), append(affectedMobs1, affectedMobs2...))

	return events.Continue
}

func handlePlayerCombat(evt events.NewRound) (affectedPlayerIds []int, affectedMobInstanceIds []int) {

	c := configs.GetConfig()

	tStart := time.Now()

	for _, userId := range users.GetOnlineUserIds() {

		user := users.GetByUserId(userId)

		// If has a buff that prevents combat, skip the player
		if user.Character.HasBuffFlag(buffs.NoCombat) {
			continue
		}

		if user == nil {
			continue
		}

		// Stage 11.4: Minor Shield round expiry
		if user.Character.HasCondition(characters.ConditionShield) {
			if user.Character.GetConditionDuration(characters.ConditionShield) <= 1 {
				user.Character.RemoveCondition(characters.ConditionShield)
				user.SendText(`<ansi fg="blue">Your Minor Shield dissipates.</ansi>`)
			} else {
				user.Character.DecrementCondition(characters.ConditionShield)
			}
		}

		/**************************
		*
		* START HANDLING FOLD CASTING
		*
		**************************/

		if user.Character.CastingState != nil {

			// Stage 11.3: prone = automatic concentration break
			if user.Character.CombatPosition == characters.PositionProne {
				user.Character.CastingState = nil
				user.SendText(`<ansi fg="red">You lose your concentration as you hit the ground!</ansi>`)
				room := rooms.LoadRoom(user.Character.RoomId)
				if room != nil {
					room.SendText(fmt.Sprintf(
						`<ansi fg="username">%s</ansi>'s concentration breaks.`, user.Character.Name), user.UserId)
				}
				continue
			}

			cs := user.Character.CastingState

			spellData := spells.GetSpell(cs.SpellId)
			if spellData == nil {
				// Spell data missing — clear and continue
				user.Character.CastingState = nil
				user.SendText(`<ansi fg="red">The spell dissipates — its data cannot be found.</ansi>`)
				continue
			}

			// Fold accumulation: loop FoldsPerRound times per combat round,
			// doubling on each iteration (mirrors the attacks-per-round pattern).
			// Each iteration emits its own message, like each attack in combat.
			// FoldsPerRound=1: 0→1, 1→2, 2→4  (one message per round)
			// FoldsPerRound=2: 0→1→2 in round 1 (two messages), 2→4 in round 2 (one message)

			// Pre-flight: simulate the loop to calculate foldDelta for the conviction check.
			simFolds := cs.FoldsAccumulated
			for i := 0; i < cs.FoldsPerRound; i++ {
				if simFolds == 0 {
					simFolds = 1
				} else {
					simFolds *= 2
				}
				if simFolds >= cs.FoldsNeeded {
					simFolds = cs.FoldsNeeded
					break
				}
			}
			foldDelta := simFolds - cs.FoldsAccumulated

			// Conviction cost proportional to folds gained this round.
			// Early rounds are cheap; the final doubling costs the most.
			roundCost := 0
			if cs.TotalConvictionCost > 0 && cs.FoldsNeeded > 0 {
				roundCost = (cs.TotalConvictionCost * foldDelta) / cs.FoldsNeeded
				if roundCost < 1 {
					roundCost = 1
				}
			}

			if roundCost > 0 && user.Character.Conviction < roundCost {
				// Out of conviction — auto-cancel
				user.Character.CastingState = nil
				user.SendText(`<ansi fg="red">Your conviction wavers — the fold collapses.</ansi>`)
				continue
			}

			user.Character.Conviction -= roundCost
			cs.ConvictionSpent += roundCost

			// Real loop: advance folds and emit a message per iteration.
			for i := 0; i < cs.FoldsPerRound; i++ {
				if cs.FoldsAccumulated == 0 {
					cs.FoldsAccumulated = 1
				} else {
					cs.FoldsAccumulated *= 2
				}
				if cs.FoldsAccumulated > cs.FoldsNeeded {
					cs.FoldsAccumulated = cs.FoldsNeeded
				}

				if cs.FoldsAccumulated >= cs.FoldsNeeded {
					// Folds complete — Stage 11.4: resolve the spell effect
					resolveRoom := rooms.LoadRoom(user.Character.RoomId)
					if resolveRoom != nil {
						resolveSpell(user, cs, spellData, resolveRoom)
					}
					user.Character.TrackSpellCast(cs.SpellId)
					user.Character.OnSkillUse(string(skills.Spellcasting), userId)
					user.Character.OnStatUse("willpower", userId)
					user.Character.CastingState = nil
					break
				}
				user.SendText(fmt.Sprintf(
					`<ansi fg="cyan">You fold your will deeper. You now hold <ansi fg="cyan-bold">%d/%d</ansi> folds.</ansi>`,
					cs.FoldsAccumulated, cs.FoldsNeeded))
			}

			continue
		}

		/**************************
		*
		* END HANDLING FOLD CASTING
		*
		**************************/

		if user.Character.Aggro == nil {
			continue
		}

		// Disable any buffs that are cancelled by combat
		user.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

		roomId := user.Character.RoomId

		uRoom := rooms.LoadRoom(roomId)
		if uRoom == nil {
			continue
		}

		if user.Character.Aggro.Type == characters.Flee {

			// Revert to Default combat regardless of outcome
			user.Character.SetAggro(user.Character.Aggro.UserId, user.Character.Aggro.MobInstanceId, characters.DefaultAttack)

			blockedByMob := ``
			for _, mobInstId := range uRoom.GetMobs(rooms.FindFighting) {
				if mob := mobs.GetInstance(mobInstId); mob != nil {
					if mob.Character.Aggro == nil || mob.Character.Aggro.UserId != userId {
						continue
					}

					// Stat comparison accounts for up to 70% of chance to flee.
					chanceIn100 := int(float64(user.Character.Stats.Dexterity.ValueAdj) / (float64(user.Character.Stats.Dexterity.ValueAdj) + float64(mob.Character.Stats.Dexterity.ValueAdj)) * 70)
					chanceIn100 += 30

					roll := util.Rand(100)

					util.LogRoll(`Flee`, roll, chanceIn100)

					if roll >= chanceIn100 {
						blockedByMob = mob.Character.Name
						break
					}
				}
			}

			blockedByPlayer := ``
			blockedByPlayerId := 0
			for _, userId := range uRoom.GetPlayers(rooms.FindFighting) {
				if u := users.GetByUserId(userId); u != nil {
					if u.Character.Aggro == nil || u.Character.Aggro.UserId != userId {
						continue
					}

					// if equal, 25% chance of fleeing... at best, 50% chance. Then add 50% on top.
					chanceIn100 := int(float64(user.Character.Stats.Dexterity.ValueAdj) / (float64(user.Character.Stats.Dexterity.ValueAdj) + float64(u.Character.Stats.Dexterity.ValueAdj)) * 70)
					chanceIn100 += 30

					roll := util.Rand(100)

					util.LogRoll(`Flee`, roll, chanceIn100)

					if roll < chanceIn100 {
						blockedByPlayer = u.Character.Name
						blockedByPlayerId = u.UserId
						break
					}
				}
			}

			if blockedByMob != `` {
				user.SendText(fmt.Sprintf(`<ansi fg="red-bold"><ansi fg="mobname">%s</ansi> blocks you from fleeing!</ansi>`, blockedByMob))
				uRoom.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is blocked from fleeing by <ansi fg="mobname">%s</ansi>!`, user.Character.Name, blockedByMob), user.UserId)
				continue
			}

			if blockedByPlayer != `` {
				user.SendText(fmt.Sprintf(`<ansi fg="red-bold"><ansi fg="username">%s</ansi> blocks you from fleeing!</ansi>`, blockedByPlayer))
				uRoom.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is blocked from fleeing by <ansi fg="username">%s</ansi>!`, user.Character.Name, blockedByPlayer), user.UserId, blockedByPlayerId)
				continue
			}

			// Success!
			exitName, exitRoomId := uRoom.GetRandomExit()

			if exitName == `` {
				user.SendText(`You can't find an exit!`)
				continue
			}

			user.SendText(fmt.Sprintf(`You flee to the <ansi fg="exit">%s</ansi> exit!`, exitName))
			uRoom.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> flees to the <ansi fg="exit">%s</ansi> exit!`, user.Character.Name, exitName), user.UserId)

			user.Character.Aggro = nil

			originRoomId := user.Character.RoomId
			if err := rooms.MoveToRoom(user.UserId, exitRoomId); err == nil {

				scripting.TryRoomScriptEvent(`onExit`, user.UserId, originRoomId)

				for _, instId := range uRoom.GetMobs(rooms.FindCharmed) {
					if mob := mobs.GetInstance(instId); mob != nil {
						// Charmed mobs assist
						if mob.Character.IsCharmed(userId) {
							mob.Command(exitName)
						}
					}
				}

				newRoom := rooms.LoadRoom(exitRoomId)

				if doLook, err := scripting.TryRoomScriptEvent(`onEnter`, user.UserId, exitRoomId); err != nil || doLook {
					usercommands.Look(``, user, newRoom, events.CmdSecretly)
				}

			}

			continue
		}

		/**************************
		*
		* START HANDLING PHYSICAL COMBAT
		*
		**************************/

		// In combat with another player
		if user.Character.Aggro != nil && user.Character.Aggro.UserId > 0 {

			defUser := users.GetByUserId(user.Character.Aggro.UserId)

			uRoom := rooms.LoadRoom(roomId)

			if uRoom == nil {
				user.Character.Aggro = nil
				continue
			}

			targetFound := true
			if defUser == nil {
				targetFound = false
			} else if defUser.Character.RoomId != user.Character.RoomId {

				if user.Character.Aggro.ExitName == `` {
					targetFound = false
				} else {
					// If the exitId doesn't match the target room id, can't find em
					if _, exitRoomId := uRoom.FindExitByName(user.Character.Aggro.ExitName); exitRoomId != defUser.Character.RoomId {
						targetFound = false
					}
				}

			}

			if !targetFound {
				user.SendText(`Your target can't be found.`)
				user.Character.Aggro = nil
				continue
			}

			defRoom := rooms.LoadRoom(defUser.Character.RoomId)
			if defRoom == nil {
				user.Character.Aggro = nil
				continue
			}

			defUser.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

			if defUser.Character.Health < 1 {
				user.SendText(`Your rage subsides.`)
				user.Character.Aggro = nil
				continue
			}

			if user.Character.Aggro.RoundsWaiting > 0 {
				mudlog.Debug(`RoundsWaiting`, `User`, user.Character.Name, `Rounds`, user.Character.Aggro.RoundsWaiting)

				user.Character.Aggro.RoundsWaiting--

				roundResult := combat.GetWaitMessages(items.Wait, user.Character, defUser.Character, combat.User, combat.User)

				for _, msg := range roundResult.MessagesToSource {
					user.SendText(msg)
				}

				for _, msg := range roundResult.MessagesToTarget {
					defUser.SendText(msg)
				}

				if len(roundResult.MessagesToSourceRoom) > 0 {
					for _, msg := range roundResult.MessagesToSourceRoom {
						uRoom.SendText(msg, user.UserId, defUser.UserId)
					}
				}

				if len(roundResult.MessagesToTargetRoom) > 0 {
					for _, msg := range roundResult.MessagesToTargetRoom {
						defRoom.SendText(msg, user.UserId, defUser.UserId)
					}
				}

				continue
			}

			// Can't see them, can't fight them.
			if defUser.Character.HasBuffFlag(buffs.Hidden) {
				user.SendText("You can't seem to find your target.")
				continue
			}

			affectedPlayerIds = append(affectedPlayerIds, user.Character.Aggro.UserId)

			// Stage 8.3: Process automatic grapple position progression
			processGrappleProgression(user.Character, defUser.Character, user.Character.Name, defUser.Character.Name, uRoom, user.UserId, defUser.UserId)

			roundResult := combat.AttackPlayerVsPlayer(user, defUser)

			// Stage 8.4: Process crit effects (parry crit disarm, dodge crit grapple opportunity)

			// Debug logging for defense crits (player vs player)
			if roundResult.DefenseZScore > 1.5 && roundResult.DefenseUsed != "" {
				defUser.SendText(fmt.Sprintf(`<ansi fg="cyan">[DEBUG: %s z-score: %.2f %s]</ansi>`,
					roundResult.DefenseUsed,
					roundResult.DefenseZScore,
					map[bool]string{true: "(CRIT!)", false: "(close)"}[roundResult.DefenseZScore > 2.0]))
			}

			// Process parry crit disarm (10% chance on parry crit)
			if roundResult.ParryCritDetected {
				disarmResult := combat.AttemptCritDisarm(defUser.Character, user.Character, 10.0)
				if disarmResult.Success {
					// Drop weapon to room
					uRoom.AddItem(disarmResult.Weapon, false)

					// Send messages to all parties
					defUser.SendText(disarmResult.Message)
					user.SendText(disarmResult.TargetMsg)
					uRoom.SendText(disarmResult.RoomMessage, user.UserId, defUser.UserId)
				}
			}

			// Process dodge crit grapple opportunity
			if roundResult.DodgeCritDetected {
				// Only set opportunity if grapple cooldown is available
				// Check by examining the cooldown without consuming it
				if defUser.Character.Cooldowns["special-move"] <= 0 {
					combat.SetGrappleOpportunity(defUser.Character)

					// Message
					defUser.SendText(fmt.Sprintf(
						`<ansi fg="yellow">You slip inside %s's guard! [Grapple opportunity]</ansi>`,
						user.Character.Name))
					uRoom.SendText(fmt.Sprintf(
						`<ansi fg="combat">%s slips inside %s's guard!</ansi>`,
						defUser.Character.Name, user.Character.Name),
						user.UserId, defUser.UserId)
				}
			}

			// If a mob attacks a player, check whether player has a charmed mob helping them, and if so, they will move to attack back
			room := rooms.LoadRoom(roomId)
			for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
				if charmedMob := mobs.GetInstance(instanceId); charmedMob != nil {
					if charmedMob.Character.IsCharmed(defUser.UserId) && charmedMob.Character.Aggro == nil {

						// Set aggro to something to prevent multiple attack triggers on this conditional
						charmedMob.Character.Aggro = &characters.Aggro{
							Type: characters.DefaultAttack,
						}

						charmedMob.Command(fmt.Sprintf("attack @%d", user.UserId))

					}
				}
			}

			for _, buffId := range roundResult.BuffSource {
				user.AddBuff(buffId, `combat`)
			}

			for _, buffId := range roundResult.BuffTarget {
				defUser.AddBuff(buffId, `combat`)
			}

			for _, msg := range roundResult.MessagesToSource {
				user.SendText(msg)
			}

			for _, msg := range roundResult.MessagesToTarget {
				defUser.SendText(msg)
			}

			for _, msg := range roundResult.MessagesToSourceRoom {
				uRoom.SendText(msg, user.UserId, defUser.UserId)
			}

			for _, msg := range roundResult.MessagesToTargetRoom {
				defRoom.SendText(msg, user.UserId, defUser.UserId)
			}

			// If the attack connected, check for damage to equipment.
			if roundResult.Hit {

				defUser.Character.TrackPlayerDamage(user.UserId, roundResult.DamageToTarget)

				// For now, only focus on offhand items.
				if defUser.Character.Equipment.Offhand.ItemId > 0 {

					modifier := 0
					if roundResult.Crit { // Crits double the chance of breakage for offhand items.
						modifier = int(defUser.Character.Equipment.Offhand.GetSpec().BreakChance)
					}

					if defUser.Character.Equipment.Offhand.BreakTest(modifier) {
						// Send message about the break

						defUser.SendText(`<ansi fg="202">***</ansi>`)
						defUser.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> Your <ansi fg="item">%s</ansi> breaks! <ansi fg="202">***</ansi></ansi>`, defUser.Character.Equipment.Offhand.NameSimple()))
						defUser.SendText(`<ansi fg="202">***</ansi>`)

						defRoom.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> The <ansi fg="item">%s</ansi> <ansi fg="username">%s</ansi> was carrying breaks! <ansi fg="202">***</ansi></ansi>`, defUser.Character.Equipment.Offhand.NameSimple(), defUser.Character.Name), defUser.UserId)

						events.AddToQueue(events.ItemOwnership{
							UserId: defUser.UserId,
							Item:   defUser.Character.Equipment.Offhand,
							Gained: false,
						})

						defUser.Character.RemoveFromBody(defUser.Character.Equipment.Offhand)

						itm := items.New(20) // Broken item
						if !defUser.Character.StoreItem(itm) {
							room.AddItem(itm, false)

							events.AddToQueue(events.ItemOwnership{
								UserId: defUser.UserId,
								Item:   itm,
								Gained: true,
							})
						}
					}
				}
			}

			if user.Character.Health <= 0 || defUser.Character.Health <= 0 {
				defUser.Character.EndAggro()
				user.Character.EndAggro()

				// Auto-retarget: If target died but player is still alive and being attacked, auto-target attacker
				if user.Character.Health > 0 && defUser.Character.Health <= 0 {
					// Check for mobs attacking this player
					for _, mobInstId := range uRoom.GetMobs(rooms.FindFighting) {
						if attackingMob := mobs.GetInstance(mobInstId); attackingMob != nil {
							if attackingMob.Character.Aggro != nil && attackingMob.Character.Aggro.UserId == user.UserId {
								user.Character.SetAggro(0, attackingMob.InstanceId, characters.DefaultAttack)
								user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"mobname\">%s</ansi>!", attackingMob.Character.Name))
								break
							}
						}
					}

					// If no mobs attacking, check for players attacking
					if user.Character.Aggro == nil {
						for _, playerId := range uRoom.GetPlayers(rooms.FindFighting) {
							if attackingPlayer := users.GetByUserId(playerId); attackingPlayer != nil {
								if attackingPlayer.Character.Aggro != nil && attackingPlayer.Character.Aggro.UserId == user.UserId {
									user.Character.SetAggro(attackingPlayer.UserId, 0, characters.DefaultAttack)
									user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"username\">%s</ansi>!", attackingPlayer.Character.Name))
									break
								}
							}
						}
					}
				}
			} else {
				user.Character.SetAggro(defUser.UserId, 0, characters.DefaultAttack)
			}

		}

		// In combat with a mob
		if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId > 0 {

			affectedMobInstanceIds = append(affectedMobInstanceIds, user.Character.Aggro.MobInstanceId)

			defMob := mobs.GetInstance(user.Character.Aggro.MobInstanceId)

			targetFound := true
			if defMob == nil {
				targetFound = false
			} else if defMob.Character.RoomId != user.Character.RoomId {

				if user.Character.Aggro.ExitName == `` {
					targetFound = false
				} else {
					// Make sure the target is still at the exit

					uRoom := rooms.LoadRoom(roomId)
					if uRoom == nil {
						user.Character.Aggro = nil
						continue
					}

					// If the exitId doesn't match the target room id, can't find em
					if _, exitRoomId := uRoom.FindExitByName(user.Character.Aggro.ExitName); exitRoomId != defMob.Character.RoomId {
						targetFound = false
					}

				}

			}

			if !targetFound {
				user.SendText("Your target can't be found.")
				user.Character.Aggro = nil
				continue
			}

			defRoom := rooms.LoadRoom(defMob.Character.RoomId)

			defMob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

			if defMob.Character.Health < 1 {
				user.SendText("Your rage subsides.")
				user.Character.Aggro = nil
				continue
			}

			if user.Character.Aggro.RoundsWaiting > 0 {
				mudlog.Debug(`RoundsWaiting`, `User`, user.Character.Name, `Rounds`, user.Character.Aggro.RoundsWaiting)

				user.Character.Aggro.RoundsWaiting--

				roundResult := combat.GetWaitMessages(items.Wait, user.Character, &defMob.Character, combat.User, combat.Mob)

				for _, msg := range roundResult.MessagesToSource {
					user.SendText(msg)
				}

				for _, msg := range roundResult.MessagesToSourceRoom {
					uRoom.SendText(msg, user.UserId)
				}

				for _, msg := range roundResult.MessagesToTargetRoom {
					defRoom.SendText(msg, user.UserId)
				}

				continue
			}

			// Can't see them, can't fight them.
			if defMob.Character.HasBuffFlag(buffs.Hidden) {
				user.SendText("You can't seem to find your target.")
				continue
			}

			affectedPlayerIds = append(affectedPlayerIds, user.Character.Aggro.UserId)

			// Stage 8.3: Process automatic grapple position progression
			processGrappleProgression(user.Character, &defMob.Character, user.Character.Name, defMob.Character.Name, uRoom, user.UserId, 0)

			var roundResult combat.AttackResult

			roundResult = combat.AttackPlayerVsMob(user, defMob)

			// Stage 8.4: Process crit effects (parry crit disarm, dodge crit grapple opportunity)

			// Debug logging for defense crits (player vs mob - mob defending)
			if roundResult.DefenseZScore > 1.5 && roundResult.DefenseUsed != "" {
				user.SendText(fmt.Sprintf(`<ansi fg="cyan">[DEBUG: Mob %s z-score: %.2f %s]</ansi>`,
					roundResult.DefenseUsed,
					roundResult.DefenseZScore,
					map[bool]string{true: "(CRIT!)", false: "(close)"}[roundResult.DefenseZScore > 2.0]))
			}

			// Process parry crit disarm (10% chance on parry crit) - mob defending
			if roundResult.ParryCritDetected {
				disarmResult := combat.AttemptCritDisarm(&defMob.Character, user.Character, 10.0)
				if disarmResult.Success {
					// Drop weapon to room
					uRoom.AddItem(disarmResult.Weapon, false)

					// Send messages (mob defending, so mob gets credit for disarm)
					user.SendText(disarmResult.TargetMsg)
					uRoom.SendText(disarmResult.RoomMessage, user.UserId)
				}
			}

			// Process dodge crit grapple opportunity - mob defending
			if roundResult.DodgeCritDetected {
				// Mobs don't have cooldown restrictions for grapple opportunity
				combat.SetGrappleOpportunity(&defMob.Character)

				// Message
				uRoom.SendText(fmt.Sprintf(
					`<ansi fg="combat"><ansi fg="mobname">%s</ansi> slips inside %s's guard!</ansi>`,
					defMob.Character.Name, user.Character.Name),
					user.UserId)
			}

			for _, buffId := range roundResult.BuffSource {
				user.AddBuff(buffId, `combat`)
			}

			for _, buffId := range roundResult.BuffTarget {
				defMob.AddBuff(buffId, `combat`)
			}

			for _, msg := range roundResult.MessagesToSource {
				user.SendText(msg)
			}

			for _, msg := range roundResult.MessagesToSourceRoom {
				uRoom.SendText(msg, user.UserId)
			}

			for _, msg := range roundResult.MessagesToTargetRoom {
				defRoom.SendText(msg, user.UserId)
			}

			// Stage 11.5: Mob concentration break when hit
			if defMob.Character.CastingState != nil && roundResult.DamageToTarget > 0 {
				maxHP := defMob.Character.HealthMax.Value
				damagePct := roundResult.DamageToTarget * 100 / maxHP
				if damagePct < 1 {
					damagePct = 1
				}
				chance := characters.CalcConcentrationChance(defMob.Character.Stats.Willpower.ValueAdj, damagePct)
				rollConc := util.Rand(100)
				util.LogRoll(`Mob Concentration`, rollConc, chance)
				if rollConc >= chance {
					defMob.Character.CastingState = nil
					uRoom.SendText(fmt.Sprintf(
						`<ansi fg="mobname">%s</ansi>'s concentration breaks.`, defMob.Character.Name))
				}
			}

						// Handle any scripted behavior now.
			if roundResult.Hit {
				scripting.TryMobScriptEvent(`onHurt`, defMob.InstanceId, user.UserId, `user`, map[string]any{`damage`: roundResult.DamageToTarget, `crit`: roundResult.Crit})
			}

			//
			// Special mob-only reaction/behavior
			//
			// Hostility default to 5 minutes
			for _, groupName := range defMob.Groups {
				mobs.MakeHostile(groupName, user.UserId, c.Timing.MinutesToRounds(2)-user.Character.Stats.Charisma.ValueAdj)
			}

			// Mobs get aggro when attacked
			if defMob.Character.Aggro == nil {
				defMob.PreventIdle = true
				// If not in the same room,
				// find an exit to the room of the player to move to
				if user.Character.RoomId != defMob.Character.RoomId {
					if mobRoom := rooms.LoadRoom(defMob.Character.RoomId); mobRoom != nil {
						for exitName, exitInfo := range mobRoom.Exits {
							if exitInfo.RoomId == user.Character.RoomId {
								defMob.Command(fmt.Sprintf(`go %s`, exitName))
								if actionStr := defMob.GetAngryCommand(); actionStr != `` {
									defMob.Command(actionStr)
								}
								break
							}
						}
					}
				}

				defMob.Command(fmt.Sprintf("attack @%d", user.UserId)) // @ means player
			}

			if user.Character.Health <= 0 || defMob.Character.Health <= 0 {
				defMob.Character.EndAggro()
				user.Character.EndAggro()

				// Auto-retarget: If target died but player is still alive and being attacked, auto-target attacker
				if user.Character.Health > 0 && defMob.Character.Health <= 0 {
					// Check for mobs attacking this player
					for _, mobInstId := range uRoom.GetMobs(rooms.FindFighting) {
						if attackingMob := mobs.GetInstance(mobInstId); attackingMob != nil {
							if attackingMob.Character.Aggro != nil && attackingMob.Character.Aggro.UserId == user.UserId {
								user.Character.SetAggro(0, attackingMob.InstanceId, characters.DefaultAttack)
								user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"mobname\">%s</ansi>!", attackingMob.Character.Name))
								break
							}
						}
					}

					// If no mobs attacking, check for players attacking
					if user.Character.Aggro == nil {
						for _, playerId := range uRoom.GetPlayers(rooms.FindFighting) {
							if attackingPlayer := users.GetByUserId(playerId); attackingPlayer != nil {
								if attackingPlayer.Character.Aggro != nil && attackingPlayer.Character.Aggro.UserId == user.UserId {
									user.Character.SetAggro(attackingPlayer.UserId, 0, characters.DefaultAttack)
									user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"username\">%s</ansi>!", attackingPlayer.Character.Name))
									break
								}
							}
						}
					}
				}
			} else {
				user.Character.SetAggro(0, defMob.InstanceId, characters.DefaultAttack)
			}

		}

		/**************************
		*
		* END HANDLING PHYSICAL COMBAT
		*
		**************************/

	}

	util.TrackTime(`DoCombat::handlePlayerCombat()`, time.Since(tStart).Seconds())

	return affectedPlayerIds, affectedMobInstanceIds
}

func handleMobCombat(evt events.NewRound) (affectedPlayerIds []int, affectedMobInstanceIds []int) {

	tStart := time.Now()

	// Handle mob round of combat
	for _, mobId := range mobs.GetAllMobInstanceIds() {

		mob := mobs.GetInstance(mobId)

		// Only handling combat functions here, so ditch out if not in combat
		if mob == nil || mob.Character.Aggro == nil {
			continue
		}

		// If has a buff that prevents combat, skip the player
		if mob.Character.HasBuffFlag(buffs.NoCombat) {
			continue
		}

		mobRoom := rooms.LoadRoom(mob.Character.RoomId)

		if mobRoom == nil {
			mob.Character.Aggro = nil
			continue
		}

		// Disable any buffs that are cancelled by combat
		mob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

		/**************************
		*
		* START HANDLING FOLD CASTING (MOBS)
		*
		**************************/

		if mob.Character.CastingState != nil {
			if mob.Character.CombatPosition == characters.PositionProne {
				mob.Character.CastingState = nil
				mobRoom.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi>'s concentration breaks.`, mob.Character.Name))
				continue
			}

			cs := mob.Character.CastingState
			spellData := spells.GetSpell(cs.SpellId)
			if spellData == nil {
				mob.Character.CastingState = nil
				continue
			}

			// Simulate fold advance to compute this round's conviction cost
			simFolds := cs.FoldsAccumulated
			for i := 0; i < cs.FoldsPerRound; i++ {
				if simFolds == 0 {
					simFolds = 1
				} else {
					simFolds *= 2
				}
				if simFolds >= cs.FoldsNeeded {
					simFolds = cs.FoldsNeeded
					break
				}
			}
			foldDelta := simFolds - cs.FoldsAccumulated
			roundCost := 0
			if cs.TotalConvictionCost > 0 && cs.FoldsNeeded > 0 {
				roundCost = (cs.TotalConvictionCost * foldDelta) / cs.FoldsNeeded
				if roundCost < 1 {
					roundCost = 1
				}
			}
			if roundCost > 0 && mob.Character.Conviction < roundCost {
				mob.Character.CastingState = nil
				mobRoom.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi>'s spell falters.`, mob.Character.Name))
				continue
			}
			mob.Character.Conviction -= roundCost

			for i := 0; i < cs.FoldsPerRound; i++ {
				if cs.FoldsAccumulated == 0 {
					cs.FoldsAccumulated = 1
				} else {
					cs.FoldsAccumulated *= 2
				}
				if cs.FoldsAccumulated > cs.FoldsNeeded {
					cs.FoldsAccumulated = cs.FoldsNeeded
				}

				if cs.FoldsAccumulated >= cs.FoldsNeeded {
					if resolveRoom := rooms.LoadRoom(mob.Character.RoomId); resolveRoom != nil {
						resolveMobSpell(mob, cs, spellData, resolveRoom)
					}
					mob.Character.CastingState = nil
					break
				}
				mobRoom.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> weaves magic with focused intent.`, mob.Character.Name))
			}
			continue
		}

		/**************************
		*
		* END HANDLING FOLD CASTING (MOBS)
		*
		**************************/

		/**************************
		*
		* START HANDLING PHYSICAL COMBAT
		*
		**************************/
		c := configs.GetConfig()

		// H2H is the base level combat, can do combat commands then
		if mob.Character.Aggro.Type == characters.DefaultAttack {

			// Stage 11.5: Caster AI decision - try spell first, then special move
			var chosenMove string
			if util.Rand(100) < mob.ActivityLevel {
				var targetChar *characters.Character
				if mob.Character.Aggro.UserId > 0 {
					if u := users.GetByUserId(mob.Character.Aggro.UserId); u != nil {
						targetChar = u.Character
					}
				} else if mob.Character.Aggro.MobInstanceId > 0 {
					if tm := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); tm != nil {
						targetChar = &tm.Character
					}
				}
				if targetChar != nil {
					chosenMove = combat.ChooseCastAction(mob)
					if chosenMove == "" {
						chosenMove = combat.ChooseSpecialMove(mob, targetChar)
					}
				}
			}

			// Execute AI-chosen move or fall back to CombatCommands
			if chosenMove != "" {
				mob.Command(chosenMove, 0)
				continue
			}
			// If they have idle commands, maybe do one of them?
			cmdCt := len(mob.CombatCommands)
			if cmdCt > 0 {

				// Each mob has a 10% chance of doing an idle action.
				if util.Rand(100) < mob.ActivityLevel {

					combatAction := mob.CombatCommands[util.Rand(cmdCt)]

					if combatAction == `` { // blank is a no-op
						continue
					}

					var waitTime float64 = 0.0
					allCmds := strings.Split(combatAction, `;`)
					if len(allCmds) >= c.Timing.TurnsPerRound() {
						mob.Command(`say I have a CombatAction that is too long. Please notify an admin.`)
					} else {
						for _, action := range strings.Split(combatAction, `;`) {
							mob.Command(action, waitTime)
							waitTime += 0.1
						}
					}
					continue
				}
			}

		}
		roomId := mob.Character.RoomId

		affectedMobInstanceIds = append(affectedMobInstanceIds, mob.InstanceId)

		// mob attacks player
		if mob.Character.Aggro != nil && mob.Character.Aggro.UserId > 0 {

			defUser := users.GetByUserId(mob.Character.Aggro.UserId)
			if defUser == nil || mob.Character.RoomId != defUser.Character.RoomId {
				mob.Character.Aggro = nil
				continue
			}

			defRoom := rooms.LoadRoom(defUser.Character.RoomId)
			if defRoom == nil {
				mob.Character.Aggro = nil
				continue
			}

			defUser.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

			if defUser.Character.Health < 1 {
				mob.Character.Aggro = nil
				continue
			}

			// Can't see them, can't fight them.
			if defUser.Character.HasBuffFlag(buffs.Hidden) {
				continue
			}

			affectedPlayerIds = append(affectedPlayerIds, mob.Character.Aggro.UserId)

			// Stage 8.8: Players get aggro when attacked by mobs (reciprocal aggro)
			if defUser.Character.Aggro == nil {
				defUser.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
			}

			// Stage 8.3: Process automatic grapple position progression
			processGrappleProgression(&mob.Character, defUser.Character, mob.Character.Name, defUser.Character.Name, mobRoom, 0, defUser.UserId)

			// --- BEGIN MOB TARGET SWITCHING AI (Stage 7.4) ---
			// Mobs with higher combat skill may intelligently switch targets
			// Only consider switching occasionally (10% of rounds)
			if util.Rand(100) < 10 && mob.Character.Aggro.Type == characters.DefaultAttack {

				combatSkill := mob.Character.GetCombatSkillLevel()

				// Only mobs with decent combat skill (30+) can switch targets strategically
				if combatSkill >= 30 {

					// Find all players in the room who are fighting
					potentialTargets := []int{}
					for _, userId := range mobRoom.GetPlayers() {
						if userId == mob.Character.Aggro.UserId {
							continue // Skip current target
						}
						if u := users.GetByUserId(userId); u != nil {
							if u.Character.Health > 0 && !u.Character.HasBuffFlag(buffs.Hidden) {
								// Prefer switching to someone who's attacking this mob
								if u.Character.Aggro != nil && u.Character.Aggro.MobInstanceId == mob.InstanceId {
									potentialTargets = append(potentialTargets, userId)
								}
							}
						}
					}

					// If we found someone attacking us, consider switching to them
					if len(potentialTargets) > 0 {
						switchChance := combat.ChanceToSwitchTarget(&mob.Character)
						roll := util.Rand(100)

						util.LogRoll("Mob Target Switch", roll, switchChance)

						if roll < switchChance {
							// Pick a random attacker to switch to
							newTargetId := potentialTargets[util.Rand(len(potentialTargets))]

							// Switch target with 1 round cost
							mob.Character.SetAggro(newTargetId, 0, mob.Character.Aggro.Type, 1)

							if newTarget := users.GetByUserId(newTargetId); newTarget != nil {
								mobRoom.SendText(
									fmt.Sprintf("<ansi fg=\"mobname\">%s</ansi> shifts focus to <ansi fg=\"username\">%s</ansi>!", mob.Character.Name, newTarget.Character.Name),
								)
							}

							continue // Skip this round due to repositioning
						}
					}
				}
			}
			// --- END MOB TARGET SWITCHING AI ---

			// If no weapon but has stuff in the backpack, look for a weapon
			// Especially useful for when they get disarmed
			if mob.Character.Equipment.Weapon.ItemId == 0 && len(mob.Character.Items) > 0 {

				roll := util.Rand(100)

				util.LogRoll(`Look for weapon`, roll, mob.Character.Stats.Charisma.ValueAdj)

				if roll < mob.Character.Stats.Charisma.ValueAdj {
					possibleWeapons := []string{}
					for _, itm := range mob.Character.Items {
						iSpec := itm.GetSpec()
						if iSpec.Type == items.Weapon {
							possibleWeapons = append(possibleWeapons, itm.DisplayName())
						}
					}

					if len(possibleWeapons) > 0 {
						mob.Command(fmt.Sprintf("equip %s", possibleWeapons[util.Rand(len(possibleWeapons))]))
					}

				}
			}

			if mob.Character.Aggro.RoundsWaiting > 0 {
				mudlog.Debug(`RoundsWaiting`, `User`, mob.Character.Name, `Rounds`, mob.Character.Aggro.RoundsWaiting)

				mob.Character.Aggro.RoundsWaiting--

				roundResult := combat.GetWaitMessages(items.Wait, &mob.Character, defUser.Character, combat.Mob, combat.User)

				for _, msg := range roundResult.MessagesToTarget {
					defUser.SendText(msg)
				}

				for _, msg := range roundResult.MessagesToSourceRoom {
					mobRoom.SendText(msg, defUser.UserId)
				}

				for _, msg := range roundResult.MessagesToTargetRoom {
					defRoom.SendText(msg, defUser.UserId)
				}

				continue
			}

			var roundResult combat.AttackResult

			roundResult = combat.AttackMobVsPlayer(mob, defUser)

			// Stage 11.4: Minor Shield reduces physical weapon damage
			if roundResult.Hit && defUser.Character.HasCondition(characters.ConditionShield) {
				reduction := int(defUser.Character.GetConditionMagnitude(characters.ConditionShield)) / 2
				if roundResult.DamageToTarget > reduction+1 {
					roundResult.DamageToTarget -= reduction
					roundResult.DamageToTargetReduction += reduction
				}
			}

			// Stage 8.4: Process crit effects (parry crit disarm, dodge crit grapple opportunity)

			// Debug logging for defense crits (mob vs player - player defending)
			if roundResult.DefenseZScore > 1.5 && roundResult.DefenseUsed != "" {
				defUser.SendText(fmt.Sprintf(`<ansi fg="cyan">[DEBUG: %s z-score: %.2f %s]</ansi>`,
					roundResult.DefenseUsed,
					roundResult.DefenseZScore,
					map[bool]string{true: "(CRIT!)", false: "(close)"}[roundResult.DefenseZScore > 2.0]))
			}

			// Process parry crit disarm (10% chance on parry crit) - player defending
			if roundResult.ParryCritDetected {
				disarmResult := combat.AttemptCritDisarm(defUser.Character, &mob.Character, 10.0)
				if disarmResult.Success {
					// Drop weapon to room
					mobRoom.AddItem(disarmResult.Weapon, false)

					// Send messages to all parties
					defUser.SendText(disarmResult.Message)
					mobRoom.SendText(disarmResult.RoomMessage, defUser.UserId)
				}
			}

			// Process dodge crit grapple opportunity - player defending
			if roundResult.DodgeCritDetected {
				// Only set opportunity if grapple cooldown is available
				if defUser.Character.Cooldowns["special-move"] <= 0 {
					combat.SetGrappleOpportunity(defUser.Character)

					// Message
					defUser.SendText(fmt.Sprintf(
						`<ansi fg="yellow">You slip inside %s's guard! [Grapple opportunity]</ansi>`,
						mob.Character.Name))
					mobRoom.SendText(fmt.Sprintf(
						`<ansi fg="combat">%s slips inside %s's guard!</ansi>`,
						defUser.Character.Name, mob.Character.Name),
						defUser.UserId)
				}
			}

			// If a mob attacks a player, check whether player has a charmed mob helping them, and if so, they will move to attack back
			room := rooms.LoadRoom(roomId)
			for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
				if charmedMob := mobs.GetInstance(instanceId); charmedMob != nil {
					if charmedMob.Character.IsCharmed(defUser.UserId) && charmedMob.Character.Aggro == nil {
						// This is set to prevent it from triggering more than once
						charmedMob.Character.Aggro = &characters.Aggro{
							Type: characters.DefaultAttack,
						}

						charmedMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId))

					}
				}
			}

			for _, buffId := range roundResult.BuffSource {
				mob.AddBuff(buffId, `combat`)
			}

			for _, buffId := range roundResult.BuffTarget {
				defUser.AddBuff(buffId, `combat`)
			}

			for _, msg := range roundResult.MessagesToTarget {
				defUser.SendText(msg)
			}

			for _, msg := range roundResult.MessagesToSourceRoom {
				mobRoom.SendText(msg, defUser.UserId)
			}

			for _, msg := range roundResult.MessagesToTargetRoom {
				defRoom.SendText(msg, defUser.UserId)
			}

			// Stage 11.3: concentration check when caster takes damage
			if defUser.Character.CastingState != nil && roundResult.DamageToTarget > 0 {
				maxHealth := defUser.Character.HealthMax.Value
				damagePct := roundResult.DamageToTarget * 100 / maxHealth
				if damagePct < 1 {
					damagePct = 1
				}
				chance := characters.CalcConcentrationChance(
					defUser.Character.Stats.Willpower.ValueAdj, damagePct)
				roll := util.Rand(100)
				util.LogRoll(`Concentration`, roll, chance)
				if roll >= chance {
					defUser.Character.CastingState = nil
					defUser.SendText(`<ansi fg="red">The pain shatters your concentration!</ansi>`)
					defRoom.SendText(fmt.Sprintf(
						`<ansi fg="username">%s</ansi>'s concentration breaks.`,
						defUser.Character.Name), defUser.UserId)
				}
			}

			// If the attack connected, check for damage to equipment.
			if roundResult.Hit {

				// For now, only focus on offhand items.
				if defUser.Character.Equipment.Offhand.ItemId > 0 {

					modifier := 0
					if roundResult.Crit { // Crits double the chance of breakage for offhand items.
						modifier = int(defUser.Character.Equipment.Offhand.GetSpec().BreakChance)
					}

					if defUser.Character.Equipment.Offhand.BreakTest(modifier) {
						// Send message about the break

						defUser.SendText(`<ansi fg="202">***</ansi>`)
						defUser.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> Your <ansi fg="item">%s</ansi> breaks! <ansi fg="202">***</ansi></ansi>`, defUser.Character.Equipment.Offhand.NameSimple()))
						defUser.SendText(`<ansi fg="202">***</ansi>`)

						defRoom.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> The <ansi fg="item">%s</ansi> <ansi fg="username">%s</ansi> was carrying breaks! <ansi fg="202">***</ansi></ansi>`, defUser.Character.Equipment.Offhand.NameSimple(), defUser.Character.Name), defUser.UserId)

						events.AddToQueue(events.ItemOwnership{
							UserId: defUser.UserId,
							Item:   defUser.Character.Equipment.Offhand,
							Gained: false,
						})

						defUser.Character.RemoveFromBody(defUser.Character.Equipment.Offhand)

						itm := items.New(20) // Broken item
						if !defUser.Character.StoreItem(itm) {
							room.AddItem(itm, false)

							events.AddToQueue(events.ItemOwnership{
								UserId: defUser.UserId,
								Item:   itm,
								Gained: true,
							})

						}
					}
				}
			}

			if mob.Character.Health <= 0 || defUser.Character.Health <= 0 {
				mob.Character.EndAggro()
				defUser.Character.EndAggro()
			} else {
				mob.Character.SetAggro(defUser.UserId, 0, characters.DefaultAttack)
			}
		}

		// mob attacks mob
		if mob.Character.Aggro != nil && mob.Character.Aggro.MobInstanceId > 0 {

			affectedMobInstanceIds = append(affectedMobInstanceIds, mob.Character.Aggro.MobInstanceId)

			defMob := mobs.GetInstance(mob.Character.Aggro.MobInstanceId)

			if defMob == nil || mob.Character.RoomId != defMob.Character.RoomId {
				mob.Character.Aggro = nil
				continue
			}

			defRoom := rooms.LoadRoom(defMob.Character.RoomId)

			defMob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

			if defMob.Character.Health < 1 {
				mob.Character.Aggro = nil
				continue
			}

			if mob.Character.Aggro.RoundsWaiting > 0 {
				mudlog.Debug(`RoundsWaiting`, `User`, mob.Character.Name, `Rounds`, mob.Character.Aggro.RoundsWaiting)

				mob.Character.Aggro.RoundsWaiting--

				roundResult := combat.GetWaitMessages(items.Wait, &mob.Character, &defMob.Character, combat.Mob, combat.Mob)

				for _, msg := range roundResult.MessagesToSourceRoom {
					mobRoom.SendText(msg)
				}

				for _, msg := range roundResult.MessagesToTargetRoom {
					defRoom.SendText(msg)
				}

				continue
			}

			// Can't see them, can't fight them.
			if defMob.Character.HasBuffFlag(buffs.Hidden) {
				continue
			}

			// Stage 8.3: Process automatic grapple position progression
			processGrappleProgression(&mob.Character, &defMob.Character, mob.Character.Name, defMob.Character.Name, mobRoom, 0, 0)

			var roundResult combat.AttackResult

			roundResult = combat.AttackMobVsMob(mob, defMob)

			for _, buffId := range roundResult.BuffSource {
				mob.AddBuff(buffId, `combat`)
			}

			for _, buffId := range roundResult.BuffTarget {
				defMob.AddBuff(buffId, `combat`)
			}

			for _, msg := range roundResult.MessagesToSourceRoom {
				mobRoom.SendText(msg)
			}

			for _, msg := range roundResult.MessagesToTargetRoom {
				defRoom.SendText(msg)
			}

			// Handle any scripted behavior now.
			if roundResult.Hit {
				scripting.TryMobScriptEvent(`onHurt`, defMob.InstanceId, mob.InstanceId, `mob`, map[string]any{`damage`: roundResult.DamageToTarget, `crit`: roundResult.Crit})
			}

			// Mobs get aggro when attacked
			if defMob.Character.Aggro == nil {
				defMob.PreventIdle = true
				defMob.Character.Aggro = &characters.Aggro{
					Type: characters.DefaultAttack,
				}
				defMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId)) // # means mob
			}

			// If the attack connected, check for damage to equipment.
			if roundResult.Hit {
				// For now, only focus on offhand items.
				if defMob.Character.Equipment.Offhand.ItemId > 0 {

					modifier := 0
					if roundResult.Crit { // Crits double the chance of breakage for offhand items.
						modifier = int(defMob.Character.Equipment.Offhand.GetSpec().BreakChance)
					}

					if defMob.Character.Equipment.Offhand.BreakTest(modifier) {
						// Send message about the break

						if defRoom := rooms.LoadRoom(defMob.Character.RoomId); defRoom != nil {

							defRoom.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> The <ansi fg="item">%s</ansi> <ansi fg="mobname">%s</ansi> was carrying breaks! <ansi fg="202">***</ansi></ansi>`, defMob.Character.Equipment.Offhand.NameSimple(), defMob.Character.Name))

							events.AddToQueue(events.ItemOwnership{
								MobInstanceId: defMob.InstanceId,
								Item:          defMob.Character.Equipment.Offhand,
								Gained:        false,
							})

							defMob.Character.RemoveFromBody(defMob.Character.Equipment.Offhand)
							itm := items.New(20) // Broken item
							if !defMob.Character.StoreItem(itm) {
								defRoom.AddItem(itm, false)

								events.AddToQueue(events.ItemOwnership{
									MobInstanceId: defMob.InstanceId,
									Item:          itm,
									Gained:        true,
								})
							}
						}
					}
				}
			}

			if mob.Character.Health <= 0 || defMob.Character.Health <= 0 {
				mob.Character.EndAggro()
				defMob.Character.EndAggro()
			} else {
				mob.Character.SetAggro(0, defMob.InstanceId, characters.DefaultAttack)
			}

		}

		/**************************
		*
		* END HANDLING PHYSICAL COMBAT
		*
		**************************/

	}

	util.TrackTime(`World::handleMobCombat()`, time.Since(tStart).Seconds())

	return affectedPlayerIds, affectedMobInstanceIds
}

// processGrappleProgression handles automatic position changes for grappled fighters
// Stage 8.3: Clinched → attempts to advance to Grounded or break free
// Grounded → controlled fighter attempts to escape
func processGrappleProgression(char1 *characters.Character, char2 *characters.Character, char1Name string, char2Name string, room *rooms.Room, user1Id int, user2Id int) {
	// Only process if both are in a grapple position
	if !char1.CombatPosition.IsGrapplePosition() || !char2.CombatPosition.IsGrapplePosition() {
		return
	}

	// Determine who is controller and who is controlled
	var controller, controlled *characters.Character
	var controllerName, controlledName string

	if char1.HasCondition(characters.ConditionGrappleController) {
		controller = char1
		controlled = char2
		controllerName = char1Name
		controlledName = char2Name
	} else {
		controller = char2
		controlled = char1
		controllerName = char2Name
		controlledName = char1Name
	}

	var result combat.PositionProgressionResult

	// Check position and perform appropriate progression
	if char1.CombatPosition == characters.PositionClinched {
		result = combat.CheckClinchProgression(controller, controlled)
	} else if char1.CombatPosition == characters.PositionGrounded {
		result = combat.CheckGroundedEscape(controller, controlled)
	} else {
		return // Not in a grapple position that needs processing
	}

	// Apply the result
	combat.ApplyPositionProgression(char1, char2, result)

	// Send messages if position changed
	if result.Changed {
		if result.NewPosition == characters.PositionStanding {
			// Both broke apart
			room.SendText(
				fmt.Sprintf(`<ansi fg="combat">%s</ansi>`, result.RoomMessage),
			)
		} else if result.NewPosition == characters.PositionGrounded {
			// Advanced to grounded
			room.SendText(
				fmt.Sprintf(`<ansi fg="combat"><ansi fg="username">%s</ansi> takes <ansi fg="mobname">%s</ansi> to the ground!</ansi>`,
					controllerName, controlledName),
			)
		}
	}
}

func handleAffected(affectedPlayerIds []int, affectedMobInstanceIds []int) {

	playersHandled := map[int]struct{}{}
	for _, userId := range affectedPlayerIds {
		if _, ok := playersHandled[userId]; ok {
			continue
		}
		playersHandled[userId] = struct{}{}

		if user := users.GetByUserId(userId); user != nil {

			if user.Character.Health <= -10 {

				user.Command(`suicide`) // suicide drops all money/items and transports to land of the dead.

			} else if user.Character.Health < 1 {

				events.AddToQueue(events.PlayerDrop{UserId: user.UserId, RoomId: user.Character.RoomId})

			}

		}
	}

	mobsHandled := map[int]struct{}{}
	for _, mobId := range affectedMobInstanceIds {
		if _, ok := mobsHandled[mobId]; ok {
			continue
		}
		mobsHandled[mobId] = struct{}{}

		if mob := mobs.GetInstance(mobId); mob != nil {
			if mob.Character.Health < 1 {

				mob.Command(`suicide`)

			}
		}

	}

}
