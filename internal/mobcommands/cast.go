package mobcommands

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Cast(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 1 {
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

	if spellInfo == nil {
		return true, nil
	}
	/*
		if mob.Character.Conviction < spellInfo.Cost {
			return true, nil
		}
	*/
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
			spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, mob.InstanceId)

		} else {

			if targetPlayerId > 0 {
				spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, targetPlayerId)
			} else if targetMobInstanceId > 0 {
				spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, targetMobInstanceId)
			}

		}

	} else if spellInfo.Type == spells.HarmSingle {

		if spellArg == `` {

			if mob.Character.Aggro != nil {
				// No target specified? Default to self
				if mob.Character.Aggro.UserId > 0 {
					spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, mob.Character.Aggro.UserId)
				} else if mob.Character.Aggro.MobInstanceId > 0 {
					spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, mob.Character.Aggro.MobInstanceId)
				}
			} else {

				playersFightingMobs := room.GetPlayers(rooms.FindFightingMob)
				if len(playersFightingMobs) > 0 {

					for _, pUserId := range playersFightingMobs {

						if u := users.GetByUserId(pUserId); u != nil {
							if u.Character.IsAggro(0, mob.InstanceId) {
								spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, pUserId)
								break
							}
						}

					}

				}

				// If no mobs found, try finding an aggro player
				if len(spellAggro.TargetMobInstanceIds) < 1 {
					mobsFightingMobs := room.GetMobs(rooms.FindFightingMob)
					if len(mobsFightingMobs) > 0 {

						for _, mId := range mobsFightingMobs {

							if m := mobs.GetInstance(mId); m != nil {
								if m.Character.IsAggro(0, mob.InstanceId) {
									spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, mId)
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

		spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, mob.InstanceId)

		if !mob.Character.IsCharmed() {

			for _, mobInstId := range room.GetMobs() {

				if m := mobs.GetInstance(mobInstId); m != nil {
					// Cast on same kind
					if m.MobId == mob.MobId {
						spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, mobInstId)
					}
				}
			}

		} else {

			spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, mob.Character.Charmed.UserId)

			// Targets self and all in party
			if p := parties.Get(mob.Character.Charmed.UserId); p != nil {

				for _, partyUserId := range p.GetMembers() {

					if partyUserId == mob.Character.Charmed.UserId {
						continue
					}

					if partyUser := users.GetByUserId(partyUserId); partyUser != nil {
						spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, partyUserId)
						spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, partyUser.Character.GetCharmIds()...)
					}

				}

			}

		}

	} else if spellInfo.Type == spells.HarmMulti {

		// Targets all mobs aggro towards player
		// Targets all players aggro towards player and their parties

		// If not currently aggro, only targets all mobs in the room

		mobsFightingMobs := room.GetMobs(rooms.FindFightingMob)
		for _, mobInstId := range mobsFightingMobs {
			if m := mobs.GetInstance(mobInstId); m != nil {
				if m.Character.IsAggro(0, mob.InstanceId) || m.HatesRace(m.Character.Race()) {
					spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, mobInstId)
				}
			}
		}

		playersFightingMobs := room.GetPlayers(rooms.FindFightingMob)
		for _, uId := range playersFightingMobs {
			if u := users.GetByUserId(uId); u != nil {
				if u.Character.IsAggro(0, mob.InstanceId) {
					spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, uId)
				}
			}
		}

	} else if spellInfo.Type == spells.HelpArea || spellInfo.Type == spells.HarmArea {

		spellAggro.TargetUserIds = room.GetPlayers()
		spellAggro.TargetMobInstanceIds = room.GetMobs()

	}

	if len(spellAggro.TargetUserIds) > 0 || len(spellAggro.TargetMobInstanceIds) > 0 || len(spellAggro.SpellRest) > 0 {

		continueCasting := true
		if allowContinueCasting, err := scripting.TrySpellScriptEvent(`onCast`, 0, mob.InstanceId, spellAggro); err == nil {
			continueCasting = allowContinueCasting
		}

		if continueCasting {
			mob.Character.Conviction -= spellInfo.Cost
			mob.Character.SetCast(spellInfo.WaitRounds, spellAggro)
		}

	}

	return true, nil
}
