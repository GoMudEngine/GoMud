package hooks

import (
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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

	// Post-combat retarget pass: players with no aggro who have mobs
	// attacking them (or their companions) should pick up a target.
	// This catches cases where mobs initiated combat during handleMobCombat
	// but the player's retarget scan in handlePlayerCombat ran too early.
	for _, userId := range users.GetOnlineUserIds() {
		user := users.GetByUserId(userId)
		if user == nil || user.Character.Aggro != nil {
			continue
		}
		uRoom := rooms.LoadRoom(user.Character.RoomId)
		if uRoom == nil {
			continue
		}
		if RetargetOrEnd(user.Character, uRoom, user.UserId, 0) {
			targetName := "something"
			if user.Character.Aggro.MobInstanceId > 0 {
				if m := mobs.GetInstance(user.Character.Aggro.MobInstanceId); m != nil {
					targetName = m.Character.Name
				}
			} else if user.Character.Aggro.UserId > 0 {
				if u := users.GetByUserId(user.Character.Aggro.UserId); u != nil {
					targetName = u.Character.Name
				}
			}
			user.SendText(fmt.Sprintf("You shift your focus to <ansi fg=\"mobname\">%s</ansi>!", targetName))
		}
	}

	return events.Continue
}

func handlePlayerCombat(evt events.NewRound) (affectedPlayerIds []int, affectedMobInstanceIds []int) {

	moonMod := float64(configs.GetBalanceConfig().MoonStatModMax)

	tStart := time.Now()

	for _, userId := range users.GetOnlineUserIds() {

		user := users.GetByUserId(userId)

		if user == nil {
			continue
		}

		if user.Character.HasBuffFlag(buffs.NoCombat) {
			continue
		}

		handlePlayerShieldDecay(user)

		if handlePlayerFoldCasting(user, userId) {
			continue
		}

		if user.Character.Aggro == nil {
			continue
		}

		// Validate aggro target still exists and is alive; retarget if possible
		if !ValidateAggro(user.Character) {
			uRoom := rooms.LoadRoom(user.Character.RoomId)
			if uRoom != nil {
				if RetargetOrEnd(user.Character, uRoom, user.UserId, 0) {
					if mob := mobs.GetInstance(user.Character.Aggro.MobInstanceId); mob != nil {
						user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"mobname\">%s</ansi>!", mob.Character.Name))
					} else if defUser := users.GetByUserId(user.Character.Aggro.UserId); defUser != nil {
						user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"username\">%s</ansi>!", defUser.Character.Name))
					}
				}
			}
			if user.Character.Aggro == nil {
				continue
			}
		}

		user.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

		uRoom := rooms.LoadRoom(user.Character.RoomId)
		if uRoom == nil {
			continue
		}

		if handlePlayerFlee(user, uRoom, userId) {
			continue
		}

		// PvP combat
		if user.Character.Aggro != nil && user.Character.Aggro.UserId > 0 {
			handlePlayerVsPlayer(user, uRoom, evt, &affectedPlayerIds)
		}

		// PvM combat
		if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId > 0 {
			handlePlayerVsMob(user, uRoom, evt, moonMod, &affectedPlayerIds, &affectedMobInstanceIds)
		}
	}

	util.TrackTime(`DoCombat::handlePlayerCombat()`, time.Since(tStart).Seconds())

	return affectedPlayerIds, affectedMobInstanceIds
}

func handleMobCombat(evt events.NewRound) (affectedPlayerIds []int, affectedMobInstanceIds []int) {

	moonMod := float64(configs.GetBalanceConfig().MoonStatModMax)
	tStart := time.Now()

	for _, mobId := range mobs.GetAllMobInstanceIds() {

		mob := mobs.GetInstance(mobId)

		if mob == nil || mob.Character.Health <= 0 {
			continue
		}

		if mob.Character.HasBuffFlag(buffs.NoCombat) {
			continue
		}

		// Non-combatant mobs (merchants, quest NPCs) should never fight.
		// If they somehow got aggro (e.g., from pack scatter), clear it.
		if mob.IsNonCombatant() {
			if mob.Character.Aggro != nil {
				mob.Character.EndAggro()
			}
			continue
		}

		mobRoom := rooms.LoadRoom(mob.Character.RoomId)
		if mobRoom == nil {
			if mob.Character.Aggro != nil {
				mob.Character.EndAggro()
			}
			continue
		}

		// Only run the full combat prep (buff stripping, shield decay, etc.) when
		// actually fighting or when the mob might enter combat this round.
		if mob.Character.Aggro != nil {
			// Strip combat-cancelling buffs (Hidden, etc.)
			// Also remove Hidden from permabuffs so Validate doesn't re-add it
			if mob.Character.HasBuffFlag(buffs.Hidden) {
				mob.Character.RemovePermaBuff(9)
			}
			mob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

			// Mob shield decay (symmetric with handlePlayerShieldDecay)
			if mob.Character.HasCondition(characters.ConditionShield) {
				if mob.Character.GetConditionDuration(characters.ConditionShield) <= 1 {
					mob.Character.RemoveCondition(characters.ConditionShield)
					mobRoom.SendText(fmt.Sprintf(
						`<ansi fg="blue"><ansi fg="mobname">%s</ansi>'s Minor Shield dissipates.</ansi>`,
						mob.Character.Name))
				} else {
					mob.Character.DecrementCondition(characters.ConditionShield)
				}
			}

			if handleMobFoldCasting(mob, mobRoom) {
				continue
			}

			// Validate mob's aggro target still exists and is alive; retarget if possible
			if !ValidateAggro(&mob.Character) {
				if mobRoom != nil {
					RetargetOrEnd(&mob.Character, mobRoom, 0, mob.InstanceId)
				}
			}
		}

		// Idle companions with autoassist scan for threats to owner
		if mob.Character.Aggro == nil && mob.Character.IsCharmed() {
			CompanionAutoTarget(mob, mobRoom)
			if mob.Character.Aggro == nil {
				continue
			}
		} else if mob.Character.Aggro == nil {
			continue
		}

		c := configs.GetConfig()
		if handleMobAIDecision(mob, c) {
			continue
		}

		affectedMobInstanceIds = append(affectedMobInstanceIds, mob.InstanceId)

		// MvP combat
		if mob.Character.Aggro != nil && mob.Character.Aggro.UserId > 0 {
			handleMobVsPlayer(mob, mobRoom, evt, moonMod, &affectedPlayerIds)
		}

		// MvM combat
		if mob.Character.Aggro != nil && mob.Character.Aggro.MobInstanceId > 0 {
			handleMobVsMob(mob, mobRoom, evt, &affectedMobInstanceIds)
		}
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

// applyMoonMods temporarily adds moon phase stat deltas to a mutated character's
// six combat stats (DEX/STR via Swiftmoon, VIT/WIL via Wanderer, PER/CHA via The Eye).
// Returns a no-op restore function when the character has no mutations or moonMod is zero.
// Always call the returned function immediately after the combat roll to undo the changes.
func applyMoonMods(ch *characters.Character, moonMod float64) func() {
	if len(ch.Mutations) == 0 || moonMod == 0 {
		return func() {}
	}
	swift, wander, eye := gametime.GetAllPhases()
	dex := gametime.MoonStatDelta(swift, moonMod, ch.Stats.Dexterity.ValueAdj)
	str := gametime.MoonStatDelta(swift, moonMod, ch.Stats.Strength.ValueAdj)
	vit := gametime.MoonStatDelta(wander, moonMod, ch.Stats.Vitality.ValueAdj)
	wil := gametime.MoonStatDelta(wander, moonMod, ch.Stats.Willpower.ValueAdj)
	per := gametime.MoonStatDelta(eye, moonMod, ch.Stats.Perception.ValueAdj)
	cha := gametime.MoonStatDelta(eye, moonMod, ch.Stats.Charisma.ValueAdj)
	ch.Stats.Dexterity.ValueAdj += dex
	ch.Stats.Strength.ValueAdj += str
	ch.Stats.Vitality.ValueAdj += vit
	ch.Stats.Willpower.ValueAdj += wil
	ch.Stats.Perception.ValueAdj += per
	ch.Stats.Charisma.ValueAdj += cha
	return func() {
		ch.Stats.Dexterity.ValueAdj -= dex
		ch.Stats.Strength.ValueAdj -= str
		ch.Stats.Vitality.ValueAdj -= vit
		ch.Stats.Willpower.ValueAdj -= wil
		ch.Stats.Perception.ValueAdj -= per
		ch.Stats.Charisma.ValueAdj -= cha
	}
}
