package mobcommands

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Suicide(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	currentRound := util.GetRoundCount()

	if rest != `vanish` && mob.Character.HasBuffFlag(buffs.ReviveOnDeath) {

		mob.Character.Health = mob.Character.HealthMax.Value

		room.SendText(`<ansi fg="mobname">` + mob.Character.Name + `</ansi> is suddenly revived in a shower of sparks!`)

		mob.Character.CancelBuffsWithFlag(buffs.ReviveOnDeath)

		return true, nil
	}

	// Useful to know sometimes
	mobs.TrackRecentDeath(mob.InstanceId)

	mudlog.Debug(`Mob Death`, `name`, mob.Character.Name, `rest`, rest)

	// Make sure to clean up any charm stuff if it's being removed
	if charmedUserId := mob.Character.RemoveCharm(); charmedUserId > 0 {
		if charmedUser := users.GetByUserId(charmedUserId); charmedUser != nil {
			charmedUser.Character.TrackCharmed(mob.InstanceId, false)
		}
	}

	// vanish is meant to remove the mob without any rewards/drops/etc.
	if rest == `vanish` {

		// Destroy any record of this mob.
		mobs.DestroyInstance(mob.InstanceId)

		// Clean up mob from room...
		if r := rooms.LoadRoom(mob.HomeRoomId); r != nil {
			r.CleanupMobSpawns(false)
		}

		// Remove from current room
		room.RemoveMob(mob.InstanceId)

		return true, nil
	}

	// Send a death msg to everyone in the room.
	room.SendText(
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> has died.`, mob.Character.Name),
	)

	// Special handling of "The Guide"
	// Mark this moment to prevent an immediate respawn
	if mob.MobId == 38 {
		if mob.Character.Charmed != nil {
			if tmpU := users.GetByUserId(mob.Character.Charmed.UserId); tmpU != nil {
				tmpU.SetTempData(`lastGuideRound`, currentRound)
			}
		}
	}

	mobXP := mob.Character.XPTL(mob.Character.Level - 1)

	events.AddToQueue(events.MobDeath{
		MobId:         int(mob.MobId),
		InstanceId:    mob.InstanceId,
		RoomId:        room.RoomId,
		CharacterName: mob.Character.Name,
		Level:         mob.Character.Level,
		PlayerDamage:  mob.Character.PlayerDamage,
	})

	xpVal := mobXP / 90

	xpVariation := xpVal / 100
	if xpVariation < 1 {
		xpVariation = 1
	}

	partyTracker := map[int]int{} // key is party leader ID, value is how much will be shared.

	if len(mob.Character.PlayerDamage) > 0 {

		xpVal = xpVal / len(mob.Character.PlayerDamage) // Div by number of players that beat him up
		xpVal += ((util.Rand(3) - 1) * xpVariation)     // a little bit of variation

		totalPlayerLevels := 0
		for uId, _ := range mob.Character.PlayerDamage {
			if user := users.GetByUserId(uId); user != nil {
				totalPlayerLevels += user.Character.Level
			}
		}

		attackerCt := len(mob.Character.PlayerDamage)

		for uId, _ := range mob.Character.PlayerDamage {
			if user := users.GetByUserId(uId); user != nil {

				if user.Character.Aggro != nil {
					if user.Character.Aggro.MobInstanceId == mob.InstanceId {
						user.Character.Aggro = nil
					}
				}

				scripting.TryMobScriptEvent(`onDie`, mob.InstanceId, uId, `user`, map[string]any{`attackerCount`: attackerCt})

				p := parties.Get(user.UserId)

				// Not in a party? Great give them the xp.
				if p == nil {

					if mob.Character.Zone != `Training` { // Don't track any kills in the training zone
						user.Character.KD.AddMobKill(int(mob.MobId))
						// Check for first kill of this mob type
						if user.Character.KD.GetMobKills(int(mob.MobId)) == 1 {
							user.Character.OnFirstMobKill(user.UserId)
						}
					}

					xpScaler := 1.0

					// If there's a level delta of more than 5, apply a scaler
					if math.Abs(float64(mob.Character.Level)-float64(totalPlayerLevels)) > 5 {

						xpScaler = float64(mob.Character.Level) / float64(totalPlayerLevels) // How much of the mobs level is the player?
						if xpScaler > 1.5 {
							xpScaler = 1.5
						} else if xpScaler < 0.25 {
							xpScaler = 0.25
						}

					}

					finalXPVal := int(math.Ceil(float64(xpVal) * xpScaler))

					mudlog.Debug("XP Calculation", "MobLevel", mob.Character.Level, "XPBase", mobXP, "xpVal", xpVal, "xpVariation", xpVariation, "xpScaler", xpScaler, "finalXPVal", finalXPVal)

					user.GrantXP(finalXPVal, `combat`)

					// Apply alignment changes
					alignmentBefore := user.Character.AlignmentName()
					alignmentAdj := combat.AlignmentChange(user.Character.Alignment, mob.Character.Alignment)
					user.Character.UpdateAlignment(alignmentAdj)
					alignmentAfter := user.Character.AlignmentName()

					mudlog.Debug("Alignment", "user Alignment", user.Character.Alignment, "mob Alignment", mob.Character.Alignment, `alignmentAdj`, alignmentAdj, `alignmentBefore`, alignmentBefore, `alignmentAfter`, alignmentAfter)

					if alignmentBefore != alignmentAfter {
						alignmentBefore = fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, alignmentBefore, alignmentBefore)
						alignmentAfter = fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, alignmentAfter, alignmentAfter)
						updateTxt := fmt.Sprintf(`<ansi fg="231">Your alignment has shifted from %s to %s!</ansi>`, alignmentBefore, alignmentAfter)
						user.SendText(updateTxt)
					}

					// Chance to learn to tame the creature.
					levelDelta := user.Character.Level - mob.Character.Level
					if levelDelta < 0 {
						levelDelta = 0
					}
					skillsDelta := int((float64(user.Character.Stats.Charisma.ValueAdj-mob.Character.Stats.Charisma.ValueAdj) + float64(user.Character.Stats.Perception.ValueAdj-mob.Character.Stats.Perception.ValueAdj)) / 2)
					if skillsDelta < 0 {
						skillsDelta = 0
					}
					targetNumber := levelDelta + skillsDelta
					if targetNumber < 1 {
						targetNumber = 1
					}

					mudlog.Debug("Tame Chance", "levelDelta", levelDelta, "skillsDelta", skillsDelta, "targetNumber", targetNumber)

					if util.Rand(1000) < targetNumber {
						if mob.IsTameable() && user.Character.GetSkillLevel(skills.Tame) > 0 {

							currentSkill := user.Character.MobMastery.GetTame(int(mob.MobId))
							if currentSkill < 50 {
								user.Character.MobMastery.SetTame(int(mob.MobId), currentSkill+1)
								if currentSkill == -1 {
									user.SendText(fmt.Sprintf(`<ansi fg="magenta">***</ansi> You've learned how to tame a <ansi fg="mobname">%s</ansi>! <ansi fg="magenta">***</ansi>`, mob.Character.Name))
								} else {
									user.SendText(fmt.Sprintf(`<ansi fg="magenta">***</ansi> Your <ansi fg="mobname">%s</ansi> taming skills get a little better! <ansi fg="magenta">***</ansi>`, mob.Character.Name))
								}
							}

						}
					}

					continue
				}

				if _, ok := partyTracker[p.LeaderUserId]; !ok {
					partyTracker[p.LeaderUserId] = 0
				}
				partyTracker[p.LeaderUserId] += xpVal

			}
		}

	}

	if len(partyTracker) > 0 {

		for leaderId, xp := range partyTracker {
			if p := parties.Get(leaderId); p != nil {

				allMembers := p.GetMembers()
				xpSplit := xp / len(allMembers)

				mudlog.Info(`Party XP`, `totalXP`, xp, `splitXP`, xpSplit, `memberCt`, len(allMembers))

				for _, memberId := range allMembers {

					if user := users.GetByUserId(memberId); user != nil {

						if mob.Character.Zone != `Training` { // Don't track any kills in the training zone
							user.Character.KD.AddMobKill(int(mob.MobId))
							// Check for first kill of this mob type
							if user.Character.KD.GetMobKills(int(mob.MobId)) == 1 {
								user.Character.OnFirstMobKill(user.UserId)
							}
						}

						user.GrantXP(xpSplit, `combat`)

						// Apply alignment changes
						alignmentBefore := user.Character.AlignmentName()
						alignmentAdj := combat.AlignmentChange(user.Character.Alignment, mob.Character.Alignment)
						user.Character.UpdateAlignment(alignmentAdj)
						alignmentAfter := user.Character.AlignmentName()

						mudlog.Debug("Alignment", "user Alignment", user.Character.Alignment, "mob Alignment", mob.Character.Alignment, `alignmentAdj`, alignmentAdj, `alignmentBefore`, alignmentBefore, `alignmentAfter`, alignmentAfter)

						if alignmentBefore != alignmentAfter {
							alignmentBefore = fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, alignmentBefore, alignmentBefore)
							alignmentAfter = fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, alignmentAfter, alignmentAfter)
							updateTxt := fmt.Sprintf(`<ansi fg="231">Your alignment has shifted from %s to %s!</ansi>`, alignmentBefore, alignmentAfter)
							user.SendText(updateTxt)
						}

						// Chance to learn to tame the creature.
						levelDelta := user.Character.Level - mob.Character.Level
						if levelDelta < 0 {
							levelDelta = 0
						}
						skillsDelta := int((float64(user.Character.Stats.Charisma.ValueAdj-mob.Character.Stats.Charisma.ValueAdj) + float64(user.Character.Stats.Perception.ValueAdj-mob.Character.Stats.Perception.ValueAdj)) / 2)
						if skillsDelta < 0 {
							skillsDelta = 0
						}
						targetNumber := levelDelta + skillsDelta
						if targetNumber < 1 {
							targetNumber = 1
						}

						mudlog.Debug("Tame Chance", "levelDelta", levelDelta, "skillsDelta", skillsDelta, "targetNumber", targetNumber)

						if util.Rand(1000) < targetNumber {
							if mob.IsTameable() && user.Character.GetSkillLevel(skills.Tame) > 0 {

								currentSkill := user.Character.MobMastery.GetTame(int(mob.MobId))
								if currentSkill < 50 {
									user.Character.MobMastery.SetTame(int(mob.MobId), currentSkill+1)

									if currentSkill == -1 {
										user.SendText(fmt.Sprintf(`<ansi fg="magenta">***</ansi> You've learned how to tame a <ansi fg="mobname">%s</ansi>! <ansi fg="magenta">***</ansi>`, mob.Character.Name))
									} else {
										user.SendText(fmt.Sprintf(`<ansi fg="magenta">***</ansi> Your <ansi fg="mobname">%s</ansi> taming skills get a little better! <ansi fg="magenta">***</ansi>`, mob.Character.Name))
									}
								}

							}
						}
					}

				}

			}
		}

	}

	if !mob.Character.HasBuffFlag(buffs.PermaGear) {

		// Check for any dropped loot...
		for _, item := range mob.Character.Items {
			msg := fmt.Sprintf(`<ansi fg="item">%s</ansi> drops to the ground.`, item.DisplayName())
			room.SendText(msg)
			room.AddItem(item, false)
		}

		allWornItems := mob.Character.Equipment.GetAllItems()

		for _, item := range allWornItems {

			roll := util.Rand(100)

			util.LogRoll(`Drop Item`, roll, mob.ItemDropChance)

			if roll >= mob.ItemDropChance {
				continue
			}

			msg := fmt.Sprintf(`<ansi fg="item">%s</ansi> drops to the ground.`, item.DisplayName())
			room.SendText(msg)
			room.AddItem(item, false)
		}

		if mob.Character.Gold > 0 {
			msg := fmt.Sprintf(`<ansi fg="yellow-bold">%d gold</ansi> drops to the ground.`, mob.Character.Gold)
			room.SendText(msg)
			room.Gold += mob.Character.Gold
		}

	}

	// Destroy any record of this mob.
	mobs.DestroyInstance(mob.InstanceId)

	// Clean up mob from room...
	if r := rooms.LoadRoom(mob.HomeRoomId); r != nil {
		r.CleanupMobSpawns(false)
	}

	// Remove from current room
	room.RemoveMob(mob.InstanceId)

	config := configs.GetGamePlayConfig()

	if config.Death.CorpsesEnabled {
		room.AddCorpse(rooms.Corpse{
			MobId:        int(mob.MobId),
			Character:    mob.Character,
			RoundCreated: currentRound,
		})
	}

	return true, nil
}
