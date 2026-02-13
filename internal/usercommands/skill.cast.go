package usercommands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
Cast Skill
Level 1 - You can cast spells
Level 2 - Become proficient in a spell at 125% rate
Level 3 - Become proficient in a spell at 175% rate
Level 4 - Become proficient in a spell at 250% rate
*/
func Cast(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Cast)

	if skillLevel == 0 {
		user.SendText("You don't know how to cast spells yet.")
		return true, errors.New(`you don't know how to cast spells yet`)
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 1 {
		user.SendText("Cast What? At Whom?")
		return true, nil
	}

	spellName := args[0]
	args = args[1:]

	if len(args) > 1 {
		if args[0] == `on` {
			args = args[1:]
		}
	}
	spellArg := strings.Join(args, ` `)

	spellInfo := spells.GetSpell(spellName)

	if spellInfo == nil || !user.Character.HasSpell(spellName) {
		user.SendText(fmt.Sprintf(`You don't know a spell called <ansi fg="spellname">%s</ansi>.`, spellName))
		return true, nil
	}

	// Calculate actual costs with config multipliers
	gameConfig := configs.GetGamePlayConfig()
	convictionCost := spellInfo.GetTotalConvictionCost(float64(gameConfig.SpellConvictionCostMultiplier))
	healthCost := spellInfo.GetTotalHealthCost(float64(gameConfig.SpellHealthCostMultiplier))

	if user.Character.Conviction < convictionCost {
		user.SendText(fmt.Sprintf(`You don't have enough conviction to cast <ansi fg="spellname">%s</ansi>.`, spellName))
		return true, nil
	}

	// Check if spell requires health sacrifice (life-force magic)
	if healthCost > 0 {
		if user.Character.Health <= healthCost {
			user.SendText(fmt.Sprintf(`Casting <ansi fg="spellname">%s</ansi> would cost you your life. You need more than %d health.`, spellName, healthCost))
			return true, nil
		}
	}

	targetPlayerId := 0
	targetMobInstanceId := 0

	if spellArg != `` {
		targetPlayerId, targetMobInstanceId = room.FindByName(spellArg)
	}

	spellAggro := characters.SpellAggroInfo{
		SpellId:              spellInfo.SpellId,
		SpellRest:            ``,
		TargetUserIds:        make([]int, 0),
		TargetMobInstanceIds: make([]int, 0),
	}

	if spellInfo.Type == spells.Neutral {

		spellAggro.SpellRest = spellArg

	} else if spellInfo.Type == spells.HelpSingle {

		if spellArg == `` {

			// No target specified? Default to self
			spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, user.UserId)

		} else {

			if targetPlayerId > 0 {
				spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, targetPlayerId)
			} else if targetMobInstanceId > 0 {
				spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, targetMobInstanceId)
			}

		}

	} else if spellInfo.Type == spells.HarmSingle {

		if targetPlayerId > 0 {
			if u := users.GetByUserId(targetPlayerId); u != nil {
				if pvpErr := room.CanPvp(user, u); pvpErr != nil {
					user.SendText(pvpErr.Error())
					return true, nil
				}
			}
		}

		if spellArg == `` {

			if user.Character.Aggro != nil {
				// No target specified? Default to aggro target
				if user.Character.Aggro.UserId > 0 {
					spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, user.Character.Aggro.UserId)
				} else if user.Character.Aggro.MobInstanceId > 0 {
					spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, user.Character.Aggro.MobInstanceId)
				}
			} else {

				fightingMobs := room.GetMobs(rooms.FindFightingPlayer)
				if len(fightingMobs) > 0 {

					for _, mobInstId := range fightingMobs {

						if mob := mobs.GetInstance(mobInstId); mob != nil {
							if mob.Character.IsAggro(user.UserId, 0) {
								spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, mobInstId)
								break
							}
						}

					}

				}

				// If no mobs found, try finding an aggro player
				if len(spellAggro.TargetMobInstanceIds) < 1 {
					fightingPlayers := room.GetPlayers(rooms.FindFightingPlayer)
					if len(fightingPlayers) > 0 {

						for _, fUserId := range fightingPlayers {

							if u := users.GetByUserId(fUserId); u != nil {
								if u.Character.IsAggro(user.UserId, 0) {
									spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, fUserId)
									break
								}
							}

						}

					}
				}

			}

		} else {

			if targetPlayerId > 0 {
				spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, targetPlayerId)
			} else if targetMobInstanceId > 0 {
				spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, targetMobInstanceId)
			}

		}

	} else if spellInfo.Type == spells.HelpMulti {

		spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, user.UserId)

		// Targets self and all in party
		if p := parties.Get(user.UserId); p != nil {

			for _, partyUserId := range p.GetMembers() {

				if partyUserId == user.UserId {
					continue
				}

				if partyUser := users.GetByUserId(partyUserId); partyUser != nil {
					spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, partyUserId)
					spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, partyUser.Character.GetCharmIds()...)
				}

			}

		}

	} else if spellInfo.Type == spells.HarmMulti {

		// Targets all mobs aggro towards player
		// Targets all players aggro towards player and their parties

		// If not currently aggro, only targets all mobs in the room

		if targetMobInstanceId > 0 {

			// target all mobs
			for _, mobInstId := range room.GetMobs() {
				if m := mobs.GetInstance(mobInstId); m != nil {
					spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, mobInstId)
				}
			}

		} else if targetPlayerId > 0 {

			// make sure they can Pvp the player being targetted
			if u := users.GetByUserId(targetPlayerId); u != nil {
				if pvpErr := room.CanPvp(user, u); pvpErr != nil {
					user.SendText(pvpErr.Error())
					return true, nil
				}
			}

			for _, uId := range room.GetPlayers() {
				if uId == user.UserId {
					continue
				}
				if u := users.GetByUserId(uId); u != nil {
					if pvpErr := room.CanPvp(user, u); pvpErr != nil {
						spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, uId)
					}
				}
			}

		} else {

			fightingMobs := room.GetMobs(rooms.FindFightingPlayer)
			for _, mobInstId := range fightingMobs {
				if m := mobs.GetInstance(mobInstId); m != nil {
					if m.Character.IsAggro(user.UserId, 0) || m.HatesSpecies(user.Character.Species()) {
						spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, mobInstId)
					}
				}
			}

			fightingPlayers := room.GetPlayers(rooms.FindFightingPlayer)
			for _, uId := range fightingPlayers {
				if u := users.GetByUserId(uId); u != nil {
					if u.Character.IsAggro(user.UserId, 0) {
						spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, uId)
					}
				}
			}

		}

		if len(spellAggro.TargetUserIds) < 1 && len(spellAggro.TargetMobInstanceIds) < 1 {
			// No targets found, default to all mobs in the room
			spellAggro.TargetMobInstanceIds = room.GetMobs(rooms.FindFightingPlayer)
		}

	} else if spellInfo.Type == spells.HelpArea {

		spellAggro.TargetUserIds = room.GetPlayers()
		spellAggro.TargetMobInstanceIds = room.GetMobs()

	} else if spellInfo.Type == spells.HarmArea {

		// make sure they can Pvp the player being hit
		for _, uId := range room.GetPlayers() {
			if u := users.GetByUserId(uId); u != nil {
				if err := room.CanPvp(user, u); err == nil {
					spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, uId)
				}

			}
		}

		spellAggro.TargetMobInstanceIds = room.GetMobs()

	}

	if len(spellAggro.TargetUserIds) > 0 || len(spellAggro.TargetMobInstanceIds) > 0 || len(spellAggro.SpellRest) > 0 {

		continueCasting := true
		if handled, err := scripting.TrySpellScriptEvent(`onCast`, user.UserId, 0, spellAggro); err == nil {
			continueCasting = handled
		}

		if continueCasting {

			// Fire an event that a skill has been used
			events.AddToQueue(events.SkillUsed{user.UserId, skills.Cast, spellInfo.SpellId})

			// Deduct conviction cost (with config multiplier applied)
			user.Character.Conviction -= convictionCost

			// Deduct health cost if spell requires life-force sacrifice (with config multiplier applied)
			if healthCost > 0 {
				user.Character.Health -= healthCost
				user.SendText(fmt.Sprintf(`<ansi fg="red">You sacrifice %d health to fuel the spell.</ansi>`, healthCost))
			}

			events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
			user.Character.SetCast(spellInfo.WaitRounds, spellAggro)
		}

	} else {

		user.SendText(`Couldn't find a target for the spell.`)

	}

	return true, nil
}
