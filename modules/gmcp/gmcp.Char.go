package gmcp

import (
	"math"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ////////////////////////////////////////////////////////////////////
// NOTE: The init function in Go is a special function that is
// automatically executed before the main function within a package.
// It is used to initialize variables, set up configurations, or
// perform any other setup tasks that need to be done before the
// program starts running.
// ////////////////////////////////////////////////////////////////////
func init() {

	//
	// We can use all functions only, but this demonstrates
	// how to use a struct
	//
	g := GMCPCharModule{
		plug: plugins.New(`gmcp.Char`, `1.0`),
	}

	events.RegisterListener(events.EquipmentChange{}, g.equipmentChangeHandler)
	events.RegisterListener(events.ItemOwnership{}, g.ownershipChangeHandler)

	events.RegisterListener(events.PlayerSpawn{}, g.playerSpawnHandler)
	events.RegisterListener(events.CharacterVitalsChanged{}, g.vitalsChangedHandler)
	events.RegisterListener(events.LevelUp{}, g.levelUpHandler)
	events.RegisterListener(events.CharacterTrained{}, g.charTrainedHandler)
	events.RegisterListener(GMCPCharUpdate{}, g.buildAndSendGMCPPayload)
	events.RegisterListener(events.GainExperience{}, g.xpGainHandler)
	events.RegisterListener(events.CharacterStatsChanged{}, g.statsChangeHandler)
	events.RegisterListener(events.CharacterChanged{}, g.charChangeHandler)
	events.RegisterListener(events.BuffsTriggered{}, g.buffTriggeredHandler)
	events.RegisterListener(events.BuffsAdded{}, g.buffsAddedHandler)
	events.RegisterListener(events.BuffsRemoved{}, g.buffsRemovedHandler)
	events.RegisterListener(events.AlignmentChanged{}, g.alignmentChangedHandler)

	events.RegisterListener(events.Quest{}, g.questProgressHandler, events.Last)

	events.RegisterListener(events.PetLevelChange{}, g.petLevelChangeHandler)
	events.RegisterListener(events.PetFed{}, g.petFedHandler)
	events.RegisterListener(events.PetItemChange{}, g.petItemChangeHandler)

	events.RegisterListener(events.MobDeath{}, g.killsChangedHandler)
	events.RegisterListener(events.PlayerDeath{}, g.killsChangedHandler)

}

type GMCPCharModule struct {
	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
	plug *plugins.Plugin
}

// Tell the system a wish to send specific GMCP Update data
type GMCPCharUpdate struct {
	UserId     int
	Identifier string
}

func (g GMCPCharUpdate) Type() string { return `GMCPCharUpdate` }

func (g *GMCPCharModule) questProgressHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.Quest)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Quests`,
	})

	return events.Continue
}

func (g *GMCPCharModule) killsChangedHandler(e events.Event) events.ListenerReturn {

	var userIds []int

	switch evt := e.(type) {
	case events.MobDeath:
		userIds = evt.KilledByUsers
	case events.PlayerDeath:
		userIds = []int{evt.UserId}
	default:
		return events.Continue
	}

	for _, userId := range userIds {
		if userId == 0 {
			continue
		}
		events.AddToQueue(GMCPCharUpdate{
			UserId:     userId,
			Identifier: `Char.Kills`,
		})
	}

	return events.Continue
}

func (g *GMCPCharModule) buffTriggeredHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.BuffsTriggered)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Affects`,
	})

	return events.Continue
}

func (g *GMCPCharModule) alignmentChangedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.AlignmentChanged)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Info`,
	})

	return events.Continue
}

func (g *GMCPCharModule) buffsAddedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.BuffsAdded)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char`,
	})

	return events.Continue
}

func (g *GMCPCharModule) buffsRemovedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.BuffsRemoved)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char`,
	})

	return events.Continue
}

func (g *GMCPCharModule) petLevelChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PetLevelChange)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Pets`,
	})

	return events.Continue
}

func (g *GMCPCharModule) petItemChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PetItemChange)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Pets`,
	})

	return events.Continue
}

func (g *GMCPCharModule) petFedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PetFed)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Pets`,
	})

	return events.Continue
}

func (g *GMCPCharModule) charChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.CharacterChanged)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char`,
	})

	return events.Continue
}

func (g *GMCPCharModule) vitalsChangedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.CharacterVitalsChanged)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Changing equipment might affect stats, inventory, maxhp/maxmp etc
	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Vitals`,
	})

	return events.Continue
}

func (g *GMCPCharModule) xpGainHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.GainExperience)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Changing equipment might affect stats, inventory, maxhp/maxmp etc
	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Worth`,
	})

	return events.Continue
}

func (g *GMCPCharModule) ownershipChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.ItemOwnership)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Inventory.Backpack`,
	})

	return events.Continue
}

func (g *GMCPCharModule) statsChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.CharacterStatsChanged)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Changing equipment might affect stats, inventory, maxhp/maxmp etc
	events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Stats, Char.Vitals, Char.Inventory.Backpack.Summary`})

	return events.Continue
}

func (g *GMCPCharModule) equipmentChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.EquipmentChange)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	statsToChange := ``

	if len(evt.ItemsRemoved) > 0 || len(evt.ItemsWorn) > 0 {
		statsToChange += `Char.Inventory, Char.Stats, Char.Vitals, Char.Affects`
	}

	// If only gold or bank changed
	if evt.BankChange != 0 || evt.GoldChange != 0 {
		if statsToChange != `` {
			statsToChange += `, `
		}
		statsToChange += `Char.Worth`
	}

	if statsToChange != `` {
		// Changing equipment might affect stats, inventory, maxhp/maxmp etc
		events.AddToQueue(GMCPCharUpdate{
			UserId:     evt.UserId,
			Identifier: statsToChange,
		})
	}

	return events.Continue
}

func (g *GMCPCharModule) charTrainedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.CharacterTrained)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Changing equipment might affect stats, inventory, maxhp/maxmp etc
	events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Info, Char.Stats, Char.Worth, Char.Vitals, Char.Inventory.Backpack.Summary, Char.Skills, Char.Jobs`})

	return events.Continue
}
func (g *GMCPCharModule) levelUpHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.LevelUp)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Changing equipment might affect stats, inventory, maxhp/maxmp etc
	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char`,
	})

	return events.Continue
}

func (g *GMCPCharModule) playerSpawnHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Send full update
	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char`,
	})

	return events.Continue
}

func (g *GMCPCharModule) buildAndSendGMCPPayload(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(GMCPCharUpdate)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "GMCPCharUpdate", "Actual Type", e.Type())
		return events.Cancel
	}

	if evt.UserId < 1 {
		return events.Continue
	}

	// Make sure they have this gmcp module enabled.
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return events.Cancel
	}

	if len(evt.Identifier) >= 4 {

		for _, identifier := range strings.Split(evt.Identifier, `,`) {

			identifier = strings.TrimSpace(identifier)

			identifierParts := strings.Split(strings.ToLower(identifier), `.`)
			for i := 0; i < len(identifierParts); i++ {
				identifierParts[i] = strings.Title(identifierParts[i])
			}

			requestedId := strings.Join(identifierParts, `.`)

			payload, moduleName := g.GetCharNode(user, requestedId)

			events.AddToQueue(GMCPOut{
				UserId:  evt.UserId,
				Module:  moduleName,
				Payload: payload,
			})

		}

	}

	return events.Continue
}

func (g *GMCPCharModule) GetCharNode(user *users.UserRecord, gmcpModule string) (data any, moduleName string) {

	all := gmcpModule == `Char`

	payload := GMCPCharModule_Payload{}

	if all || g.wantsGMCPPayload(`Char.Info`, gmcpModule) {
		payload.Info = &GMCPCharModule_Payload_Info{
			Account:        user.Username,
			Name:           user.Character.Name,
			Class:          skills.GetProfession(user.Character.GetAllSkillRanks()),
			Race:           user.Character.Race(),
			Alignment:      user.Character.AlignmentName(),
			Level:          user.Character.Level,
			Role:           user.Role,
			SkillPoints:    user.Character.StatPoints,
			TrainingPoints: user.Character.TrainingPoints,
		}

		if !all {
			return payload.Info, `Char.Info`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Pets`, gmcpModule) {

		payload.Pets = []GMCPCharModule_Payload_Pet{}

		if user.Character.Pet.Exists() && !user.Character.Pet.IsMissing() {

			p := GMCPCharModule_Payload_Pet{
				Name:     user.Character.Pet.Name,
				Type:     user.Character.Pet.Type,
				Level:    user.Character.Pet.Level,
				Hunger:   user.Character.Pet.Food.String(),
				Capacity: user.Character.Pet.GetEffectiveCapacity(),
				Buffs:    []string{},
				Items:    []GMCPCharModule_Payload_Inventory_Item{},
			}

			if abDisplay := user.Character.Pet.GetCurrentAbilityDisplay(); abDisplay != nil {
				p.Ability = &GMCPCharModule_Payload_Pet_Ability{
					LevelGranted: abDisplay.LevelGranted,
					CombatChance: abDisplay.CombatChance,
					DiceRoll:     abDisplay.DiceRoll,
					StatMods:     abDisplay.StatMods,
				}
				if len(abDisplay.BuffNames) > 0 {
					p.Buffs = abDisplay.BuffNames
				}
			}

			for _, itm := range user.Character.Pet.Items {
				p.Items = append(p.Items, newInventory_Item(itm))
			}

			payload.Pets = append(payload.Pets, p)
		}

		if !all {
			return payload.Pets, `Char.Pets`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Enemies`, gmcpModule) {

		payload.Enemies = []GMCPCharModule_Enemy{}

		aggroMobInstanceId := 0
		if user.Character.Aggro != nil {
			if user.Character.Aggro.MobInstanceId > 0 {
				aggroMobInstanceId = user.Character.Aggro.MobInstanceId
			}
		}

		if roomInfo := rooms.LoadRoom(user.Character.RoomId); roomInfo != nil {

			for _, mobInstanceId := range roomInfo.GetMobs(rooms.FindFighting) {
				mob := mobs.GetInstance(mobInstanceId)
				if mob == nil {
					continue
				}

				e := GMCPCharModule_Enemy{
					Id:      mob.ShorthandId(),
					Name:    mob.Character.Name,
					Level:   mob.Character.Level,
					Hp:      mob.Character.Health,
					MaxHp:   mob.Character.HealthMax.Value,
					Engaged: mob.InstanceId == aggroMobInstanceId,
				}

				payload.Enemies = append(payload.Enemies, e)
			}

		}

		if !all {
			return payload.Enemies, `Char.Enemies`
		}

	}

	// Allow specifically updating the Backpack Summary
	if `Char.Inventory.Backpack.Summary` == gmcpModule {

		payload.Inventory = &GMCPCharModule_Payload_Inventory{
			Backpack: &GMCPCharModule_Payload_Inventory_Backpack{
				Summary: GMCPCharModule_Payload_Inventory_Backpack_Summary{
					Count: len(user.Character.Items),
					Max:   user.Character.CarryCapacity(),
				},
			},
		}

		return payload.Inventory.Backpack.Summary, `Char.Inventory.Backpack.Summary`
	}

	if all || g.wantsGMCPPayload(`Char.Inventory`, gmcpModule) || g.wantsGMCPPayload(`Char.Inventory.Backpack`, gmcpModule) {

		payload.Inventory = &GMCPCharModule_Payload_Inventory{

			Backpack: &GMCPCharModule_Payload_Inventory_Backpack{
				Items: []GMCPCharModule_Payload_Inventory_Item{},
				Summary: GMCPCharModule_Payload_Inventory_Backpack_Summary{
					Count: len(user.Character.Items),
					Max:   user.Character.CarryCapacity(),
				},
			},

			Worn: buildWornPayload(user.Character.Equipment),
		}

		// Fill the items list
		for _, itm := range user.Character.Items {
			payload.Inventory.Backpack.Items = append(payload.Inventory.Backpack.Items, newInventory_Item(itm))
		}

		if !all {

			if `Char.Inventory.Backpack` == gmcpModule {
				return payload.Inventory.Backpack, `Char.Inventory.Backpack`
			}

			return payload.Inventory, `Char.Inventory`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Stats`, gmcpModule) {

		payload.Stats = &GMCPCharModule_Payload_Stats{
			Strength:      user.Character.Stats.Strength.ValueAdj,
			StrengthMod:   user.Character.StatMod(`strength`),
			Speed:         user.Character.Stats.Speed.ValueAdj,
			SpeedMod:      user.Character.StatMod(`speed`),
			Smarts:        user.Character.Stats.Smarts.ValueAdj,
			SmartsMod:     user.Character.StatMod(`smarts`),
			Vitality:      user.Character.Stats.Vitality.ValueAdj,
			VitalityMod:   user.Character.StatMod(`vitality`),
			Mysticism:     user.Character.Stats.Mysticism.ValueAdj,
			MysticismMod:  user.Character.StatMod(`mysticism`),
			Perception:    user.Character.Stats.Perception.ValueAdj,
			PerceptionMod: user.Character.StatMod(`perception`),
		}

		if !all {
			return payload.Stats, `Char.Stats`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Vitals`, gmcpModule) {

		payload.Vitals = &GMCPCharModule_Payload_Vitals{
			Hp:    user.Character.Health,
			HpMax: user.Character.HealthMax.Value,
			Sp:    user.Character.Mana,
			SpMax: user.Character.ManaMax.Value,
		}

		if !all {
			return payload.Vitals, `Char.Vitals`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Worth`, gmcpModule) {

		payload.Worth = &GMCPCharModule_Payload_Worth{
			Gold: user.Character.Gold,
			Bank: user.Character.Bank,
			TNL:  user.Character.XPTL(user.Character.Level),
			XP:   user.Character.Experience,
		}

		if !all {
			return payload.Worth, `Char.Worth`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Affects`, gmcpModule) {

		c := configs.GetTimingConfig()

		payload.Affects = make(map[string]GMCPCharModule_Payload_Affect)

		nameIncrement := 0
		for _, buff := range user.Character.GetBuffs() {

			buffSpec := buffs.GetBuffSpec(buff.BuffId)
			if buffSpec == nil {
				continue
			}

			timeCur, timeMax := -1, -1

			if !buff.PermaBuff {
				roundsLeft, totalRounds := buffs.GetDurations(buff, buffSpec)
				timeMax = c.RoundsToSeconds(totalRounds)
				timeCur = c.RoundsToSeconds(roundsLeft)
				if timeCur < 0 {
					timeCur = 0
				}
			}

			name, desc := buffSpec.VisibleNameDesc()

			buffSource := buff.Source
			if buffSource == `` {
				buffSource = `unknown`
			}
			aff := GMCPCharModule_Payload_Affect{
				Name:         name,
				Description:  desc,
				DurationMax:  timeMax,
				DurationLeft: timeCur,
				Type:         buffSource,
			}

			aff.Mods = make(map[string]int)
			for name, value := range buffSpec.StatMods {
				aff.Mods[name] = value
			}

			if _, ok := payload.Affects[name]; ok {
				nameIncrement++
				name += `#` + strconv.Itoa(nameIncrement)
			}

			payload.Affects[name] = aff
		}

		if !all {
			return payload.Affects, `Char.Affects`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Quests`, gmcpModule) {

		payload.Quests = []GMCPCharModule_Payload_Quest{}

		for questId, questStep := range user.Character.GetQuestProgress() {

			questToken := quests.PartsToToken(questId, questStep)

			questInfo := quests.GetQuest(questToken)
			if questInfo == nil {
				continue
			}

			// Secret quests are not sent
			if questInfo.Secret {
				continue
			}

			completedSteps := 0
			totalSteps := len(questInfo.Steps)

			questPayload := GMCPCharModule_Payload_Quest{
				Name:        questInfo.Name,
				Completion:  0,
				Description: questInfo.Description,
			}

			for _, step := range questInfo.Steps {
				completedSteps++
				if step.Id == questStep {
					questPayload.Description = step.Description
					break
				}
			}

			questPayload.Completion = int(math.Floor(float64(completedSteps) / float64(totalSteps) * 100))

			// Add to the returned output
			payload.Quests = append(payload.Quests, questPayload)
		}

		if !all {
			return payload.Quests, `Char.Quests`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Skills`, gmcpModule) {

		payload.Skills = []GMCPCharModule_Payload_Skill{}

		for skillName, skillLevel := range user.Character.GetSkills() {
			payload.Skills = append(payload.Skills, GMCPCharModule_Payload_Skill{
				Name:    skillName,
				Level:   skillLevel,
				Maximum: skillLevel >= skills.MaxSkillLevel(skillName),
			})
		}

		if !all {
			return payload.Skills, `Char.Skills`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Jobs`, gmcpModule) {

		payload.Jobs = []GMCPCharModule_Payload_Job{}

		allRanks := user.Character.GetAllSkillRanks()
		for _, rank := range skills.GetProfessionRanks(allRanks) {
			payload.Jobs = append(payload.Jobs, GMCPCharModule_Payload_Job{
				Name:        rank.Profession,
				Proficiency: rank.ExperienceTitle,
				Completion:  int(math.Floor(rank.Completion * 100)),
			})
		}

		if !all {
			return payload.Jobs, `Char.Jobs`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Kills`, gmcpModule) {

		mobKills := map[string]int{}
		raceKills := map[string]int{}
		areaKills := map[string]int{}
		eliteKills := map[string]int{}
		charKills := map[string]GMCPCharModule_Payload_Kills_PvpEntry{}

		totalMobKills := 0
		totalPvpKills := 0

		for mid, kCt := range user.Character.KD.Kills {
			if mobSpec := mobs.GetMobSpec(mobs.MobId(mid)); mobSpec != nil {
				totalMobKills += kCt
				mobKills[mobSpec.Character.Name] = mobKills[mobSpec.Character.Name] + kCt
				if raceInfo := races.GetRace(mobSpec.Character.RaceId); raceInfo != nil {
					raceKills[raceInfo.Name] = raceKills[raceInfo.Name] + kCt
				}
				areaKills[mobSpec.Zone] = areaKills[mobSpec.Zone] + kCt
			}
		}

		// Build elite kills by mob name from the "mobId:mobName" keyed map.
		for key, eCt := range user.Character.KD.EliteKills {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 {
				eliteKills[parts[1]] = eliteKills[parts[1]] + eCt
			}
		}

		for userIdNameStr, killCount := range user.Character.KD.PlayerKills {
			parts := strings.Split(userIdNameStr, `:`)
			if len(parts) == 2 {
				charKills[parts[1]] = GMCPCharModule_Payload_Kills_PvpEntry{Count: killCount}
			}
			totalPvpKills++
		}

		mobKDRatio := user.Character.KD.GetMobKDRatio()
		pvpKDRatio := user.Character.KD.GetPvpKDRatio()

		payload.Kills = &GMCPCharModule_Payload_Kills{
			Mob: GMCPCharModule_Payload_Kills_Section{
				Total:      totalMobKills,
				Deaths:     user.Character.KD.GetMobDeaths(),
				KDRatio:    mobKDRatio,
				EliteKills: user.Character.KD.GetEliteKills(),
				ByName:     mobKills,
				ByElite:    eliteKills,
				ByRace:     raceKills,
				ByArea:     areaKills,
			},
			Pvp: GMCPCharModule_Payload_Kills_PvpSection{
				Total:   totalPvpKills,
				Deaths:  user.Character.KD.GetPvpDeaths(),
				KDRatio: pvpKDRatio,
				Players: charKills,
			},
		}

		if !all {
			return payload.Kills, `Char.Kills`
		}
	}

	// If we reached this point and Char wasn't requested, we have a problem.
	if !all {
		mudlog.Error(`gmcp.Char`, `error`, `Bad module requested`, `module`, gmcpModule)
	}

	return payload, `Char`
}

// wantsGMCPPayload(`Char.Info`, `Char`)
func (g *GMCPCharModule) wantsGMCPPayload(packageToConsider string, packageRequested string) bool {

	if packageToConsider == packageRequested {
		return true
	}

	if len(packageToConsider) < len(packageRequested) {
		return false
	}

	if packageToConsider[0:len(packageRequested)] == packageRequested {
		return true
	}

	return false
}

type GMCPCharModule_Payload struct {
	Info      *GMCPCharModule_Payload_Info             `json:"Info,omitempty"`
	Affects   map[string]GMCPCharModule_Payload_Affect `json:"Affects,omitempty"`
	Enemies   []GMCPCharModule_Enemy                   `json:"Enemies,omitempty"`
	Inventory *GMCPCharModule_Payload_Inventory        `json:"Inventory,omitempty"`
	Stats     *GMCPCharModule_Payload_Stats            `json:"Stats,omitempty"`
	Vitals    *GMCPCharModule_Payload_Vitals           `json:"Vitals,omitempty"`
	Worth     *GMCPCharModule_Payload_Worth            `json:"Worth,omitempty"`
	Quests    []GMCPCharModule_Payload_Quest           `json:"Quests,omitempty"`
	Pets      []GMCPCharModule_Payload_Pet             `json:"Pets,omitempty"`
	Skills    []GMCPCharModule_Payload_Skill           `json:"Skills,omitempty"`
	Jobs      []GMCPCharModule_Payload_Job             `json:"Jobs,omitempty"`
	Kills     *GMCPCharModule_Payload_Kills            `json:"Kills,omitempty"`
}

// /////////////////
// Char.Info
// /////////////////
type GMCPCharModule_Payload_Info struct {
	Account        string `json:"account,omitempty"`
	Name           string `json:"name,omitempty"`
	Class          string `json:"class,omitempty"`
	Race           string `json:"race,omitempty"`
	Alignment      string `json:"alignment,omitempty"`
	Level          int    `json:"level,omitempty"`
	Role           string `json:"role"`
	SkillPoints    int    `json:"skillpoints"`
	TrainingPoints int    `json:"trainingpoints"`
}

// /////////////////
// Char.Enemies
// /////////////////
type GMCPCharModule_Enemy struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Level   int    `json:"level"`
	Hp      int    `json:"hp"`
	MaxHp   int    `json:"hp_max"`
	Engaged bool   `json:"engaged"`
}

// /////////////////
// Char.Inventory
// /////////////////
type GMCPCharModule_Payload_Inventory struct {
	Backpack *GMCPCharModule_Payload_Inventory_Backpack `json:"Backpack,omitempty"`
	Worn     GMCPCharModule_Payload_Inventory_Worn      `json:"Worn"`
}

type GMCPCharModule_Payload_Inventory_Backpack struct {
	Items   []GMCPCharModule_Payload_Inventory_Item           `json:"items,omitempty"`
	Summary GMCPCharModule_Payload_Inventory_Backpack_Summary `json:"Summary,omitempty"`
}

type GMCPCharModule_Payload_Inventory_Backpack_Summary struct {
	Count int `json:"count,omitempty"`
	Max   int `json:"max,omitempty"`
}

// GMCPCharModule_Payload_Inventory_Worn is keyed by slot name (e.g. "weapon",
// "head") and serialises to the same JSON object shape that the named-struct
// version produced. Using a map means the field set tracks items.AllEquipSlots()
// automatically; no manual update is needed when slots are added or removed.
type GMCPCharModule_Payload_Inventory_Worn map[string]GMCPCharModule_Payload_Inventory_Item

func buildWornPayload(eq characters.Worn) GMCPCharModule_Payload_Inventory_Worn {
	worn := make(GMCPCharModule_Payload_Inventory_Worn)
	for _, slot := range items.AllEquipSlots() {
		itm := eq.Get(slot)
		if itm.IsDisabled() {
			continue
		}
		worn[string(slot)] = newInventory_Item(*itm)
	}
	return worn
}

type GMCPCharModule_Payload_Inventory_Item struct {
	Id      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	SubType string   `json:"subtype"`
	Uses    int      `json:"uses"`
	Details []string `json:"details"`
	ItemId      int    `json:"itemid,omitempty"`
	SimpleName  string `json:"simple_name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
}

func newInventory_Item(itm items.Item) GMCPCharModule_Payload_Inventory_Item {

	if itm.IsDisabled() {
		return GMCPCharModule_Payload_Inventory_Item{
			Name:    `-disabled-`,
			Details: []string{},
		}
	}

	itmSpec := itm.GetSpec()
	d := GMCPCharModule_Payload_Inventory_Item{
		Id:          itm.ShorthandId(),
		ItemId:      itm.ItemId,
		Name:        itm.Name(),
		SimpleName:  itm.NameSimple(),
		DisplayName: itm.DisplayName(),
		Description: itmSpec.Description,
		Type:        string(itmSpec.Type),
		SubType:     string(itmSpec.Subtype),
		Uses:        itm.Uses,
		Details:     []string{},
	}

	if !itm.Uncursed && itmSpec.Cursed {
		d.Details = append(d.Details, `cursed`)
	}

	if itmSpec.QuestToken != `` {
		d.Details = append(d.Details, `quest`)
	}

	if itmSpec.Type == items.Weapon {
		d.Details = append(d.Details, strconv.Itoa(itmSpec.Hands)+`-handed`)
	}

	return d
}

// /////////////////
// Char.Stats
// /////////////////
type GMCPCharModule_Payload_Stats struct {
	Strength      int `json:"strength,omitempty"`
	StrengthMod   int `json:"strengthmod,omitempty"`
	Speed         int `json:"speed,omitempty"`
	SpeedMod      int `json:"speedmod,omitempty"`
	Smarts        int `json:"smarts,omitempty"`
	SmartsMod     int `json:"smartsmod,omitempty"`
	Vitality      int `json:"vitality,omitempty"`
	VitalityMod   int `json:"vitalitymod,omitempty"`
	Mysticism     int `json:"mysticism,omitempty"`
	MysticismMod  int `json:"mysticismmod,omitempty"`
	Perception    int `json:"perception,omitempty"`
	PerceptionMod int `json:"perceptionmod,omitempty"`
}

// /////////////////
// Char.Vitals
// /////////////////
type GMCPCharModule_Payload_Vitals struct {
	Hp    int `json:"hp,omitempty"`
	HpMax int `json:"hp_max,omitempty"`
	Sp    int `json:"sp,omitempty"`
	SpMax int `json:"sp_max,omitempty"`
}

// /////////////////
// Char.Worth
// /////////////////
type GMCPCharModule_Payload_Worth struct {
	Gold int `json:"gold_carry,omitempty"`
	Bank int `json:"gold_bank,omitempty"`
	TNL  int `json:"tnl,omitempty"`
	XP   int `json:"xp,omitempty"`
}

// /////////////////
// Char.Affects
// /////////////////
type GMCPCharModule_Payload_Affect struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	DurationMax  int            `json:"duration_max"`
	DurationLeft int            `json:"duration_left"`
	Type         string         `json:"type"`
	Mods         map[string]int `json:"affects"`
}

// /////////////////
// Char.Quests
// /////////////////
type GMCPCharModule_Payload_Quest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Completion  int    `json:"completion"`
}

// /////////////////
// Char.Pets
// /////////////////
type GMCPCharModule_Payload_Pet struct {
	Name     string                                  `json:"name"`
	Type     string                                  `json:"type"`
	Level    int                                     `json:"level"`
	Hunger   string                                  `json:"hunger"`
	Ability  *GMCPCharModule_Payload_Pet_Ability     `json:"ability,omitempty"`
	Buffs    []string                                `json:"buffs"`
	Items    []GMCPCharModule_Payload_Inventory_Item `json:"items"`
	Capacity int                                     `json:"capacity"`
}

type GMCPCharModule_Payload_Pet_Ability struct {
	LevelGranted int            `json:"level_granted"`
	CombatChance int            `json:"combat_chance"`
	DiceRoll     string         `json:"dice_roll"`
	StatMods     map[string]int `json:"stat_mods,omitempty"`
}

// /////////////////
// Char.Skills
// /////////////////
type GMCPCharModule_Payload_Skill struct {
	Name    string `json:"name"`
	Level   int    `json:"level"`
	Maximum bool   `json:"maximum,omitempty"`
}

// /////////////////
// Char.Jobs
// /////////////////
type GMCPCharModule_Payload_Job struct {
	Name        string `json:"name"`
	Completion  int    `json:"completion"`
	Proficiency string `json:"proficiency"`
}

// /////////////////
// Char.Kills
// /////////////////
type GMCPCharModule_Payload_Kills struct {
	Mob GMCPCharModule_Payload_Kills_Section    `json:"mob"`
	Pvp GMCPCharModule_Payload_Kills_PvpSection `json:"pvp"`
}

type GMCPCharModule_Payload_Kills_Section struct {
	Total      int            `json:"total"`
	Deaths     int            `json:"deaths"`
	KDRatio    float64        `json:"kd_ratio"`
	EliteKills int            `json:"elite_kills"`
	ByName     map[string]int `json:"by_name"`
	ByElite    map[string]int `json:"by_elite"`
	ByRace     map[string]int `json:"by_race"`
	ByArea     map[string]int `json:"by_area"`
}

type GMCPCharModule_Payload_Kills_PvpSection struct {
	Total   int                                              `json:"total"`
	Deaths  int                                              `json:"deaths"`
	KDRatio float64                                          `json:"kd_ratio"`
	Players map[string]GMCPCharModule_Payload_Kills_PvpEntry `json:"players"`
}

type GMCPCharModule_Payload_Kills_PvpEntry struct {
	Count int `json:"count"`
}
