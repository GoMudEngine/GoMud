package gmcp

import (
	"math"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/quests"
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

	events.RegisterListener(events.Quest{}, g.questProgressHandler)

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
		statsToChange += `Char.Inventory, Char.Stats, Char.Vitals`
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
	events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Stats, Char.Worth, Char.Vitals, Char.Inventory.Backpack.Summary`})

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
			Account:   user.Username,
			Name:      user.Character.Name,
			Class:     skills.GetProfession(user.Character.GetAllSkillRanks()),
			Race:      user.Character.Race(),
			Alignment: user.Character.AlignmentName(),
			Level:     user.Character.Level,
		}

		if !all {
			return payload.Info, `Char.Info`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Pets`, gmcpModule) {

		payload.Pets = []GMCPCharModule_Payload_Pet{}

		if user.Character.Pet.Exists() {

			p := GMCPCharModule_Payload_Pet{
				Name:   user.Character.Pet.Name,
				Type:   user.Character.Pet.Type,
				Hunger: `full`,
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

			Worn: &GMCPCharModule_Payload_Inventory_Worn{
				Weapon:  newInventory_Item(user.Character.Equipment.Weapon),
				Offhand: newInventory_Item(user.Character.Equipment.Offhand),
				Head:    newInventory_Item(user.Character.Equipment.Head),
				Neck:    newInventory_Item(user.Character.Equipment.Neck),
				Body:    newInventory_Item(user.Character.Equipment.Body),
				Belt:    newInventory_Item(user.Character.Equipment.Belt),
				Gloves:  newInventory_Item(user.Character.Equipment.Gloves),
				Ring:    newInventory_Item(user.Character.Equipment.Ring),
				Legs:    newInventory_Item(user.Character.Equipment.Legs),
				Feet:    newInventory_Item(user.Character.Equipment.Feet),
			},
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
			Strength:   user.Character.Stats.Strength.ValueAdj,
			Dexterity:  user.Character.Stats.Dexterity.ValueAdj,
			Perception: user.Character.Stats.Perception.ValueAdj,
			Vitality:   user.Character.Stats.Vitality.ValueAdj,
			Willpower:  user.Character.Stats.Willpower.ValueAdj,
			Charisma:   user.Character.Stats.Charisma.ValueAdj,
		}

		if !all {
			return payload.Stats, `Char.Stats`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Vitals`, gmcpModule) {

		payload.Vitals = &GMCPCharModule_Payload_Vitals{
			Hp:            user.Character.Health,
			HpMax:         user.Character.HealthMax.Value,
			Stamina:       user.Character.Stamina,
			StaminaMax:    user.Character.StaminaMax.Value,
			Conviction:    user.Character.Conviction,
			ConvictionMax: user.Character.ConvictionMax.Value,
		}

		if !all {
			return payload.Vitals, `Char.Vitals`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Worth`, gmcpModule) {

		payload.Worth = &GMCPCharModule_Payload_Worth{
			Gold:           user.Character.Gold,
			Bank:           user.Character.Bank,
			SkillPoints:    user.Character.StatPoints,
			TrainingPoints: user.Character.TrainingPoints,
			TNL:            user.Character.XPTL(user.Character.Level),
			XP:             user.Character.Experience,
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

			timeLeft, timeMax := -1, -1

			if !buff.PermaBuff {
				roundsLeft, totalRounds := buffs.GetDurations(buff, buffSpec)
				timeMax = c.RoundsToSeconds(totalRounds)
				timeLeft = c.RoundsToSeconds(roundsLeft)
				if timeLeft < 0 {
					timeLeft = 0
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
				DurationLeft: timeLeft,
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

			questPayload.Completion = int(math.Floor(float64(completedSteps)/float64(totalSteps)) * 100)

			// Add to the returned output
			payload.Quests = append(payload.Quests, questPayload)
		}

		if !all {
			return payload.Quests, `Char.Quests`
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
}

// /////////////////
// Char.Info
// /////////////////
type GMCPCharModule_Payload_Info struct {
	Account   string `json:"account,omitempty"`
	Name      string `json:"name,omitempty"`
	Class     string `json:"class,omitempty"`
	Race      string `json:"race,omitempty"`
	Alignment string `json:"alignment,omitempty"`
	Level     int    `json:"level,omitempty"`
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
	Worn     *GMCPCharModule_Payload_Inventory_Worn     `json:"Worn"`
}

type GMCPCharModule_Payload_Inventory_Backpack struct {
	Items   []GMCPCharModule_Payload_Inventory_Item           `json:"items,omitempty"`
	Summary GMCPCharModule_Payload_Inventory_Backpack_Summary `json:"Summary,omitempty"`
}

type GMCPCharModule_Payload_Inventory_Backpack_Summary struct {
	Count int `json:"count,omitempty"`
	Max   int `json:"max,omitempty"`
}

type GMCPCharModule_Payload_Inventory_Worn struct {
	Weapon  GMCPCharModule_Payload_Inventory_Item `json:"weapon,omitempty"`
	Offhand GMCPCharModule_Payload_Inventory_Item `json:"offhand,omitempty"`
	Head    GMCPCharModule_Payload_Inventory_Item `json:"head,omitempty"`
	Neck    GMCPCharModule_Payload_Inventory_Item `json:"neck,omitempty"`
	Body    GMCPCharModule_Payload_Inventory_Item `json:"body,omitempty"`
	Belt    GMCPCharModule_Payload_Inventory_Item `json:"belt,omitempty"`
	Gloves  GMCPCharModule_Payload_Inventory_Item `json:"gloves,omitempty"`
	Ring    GMCPCharModule_Payload_Inventory_Item `json:"ring,omitempty"`
	Legs    GMCPCharModule_Payload_Inventory_Item `json:"legs,omitempty"`
	Feet    GMCPCharModule_Payload_Inventory_Item `json:"feet,omitempty"`
}

type GMCPCharModule_Payload_Inventory_Item struct {
	Id      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	SubType string   `json:"subtype"`
	Uses    int      `json:"uses"`
	Details []string `json:"details"`
}

func newInventory_Item(itm items.Item) GMCPCharModule_Payload_Inventory_Item {

	itmSpec := itm.GetSpec()
	d := GMCPCharModule_Payload_Inventory_Item{
		Id:      itm.ShorthandId(),
		Name:    itm.Name(),
		Type:    string(itmSpec.Type),
		SubType: string(itmSpec.Subtype),
		Uses:    itm.Uses,
		Details: []string{},
	}

	if !itm.Uncursed && itmSpec.Cursed {
		d.Details = append(d.Details, `cursed`)
	}

	if itmSpec.QuestToken != `` {
		d.Details = append(d.Details, `quest`)
	}

	return d
}

// /////////////////
// Char.Stats
// /////////////////
type GMCPCharModule_Payload_Stats struct {
	Strength   int `json:"strength,omitempty"`
	Dexterity  int `json:"dexterity,omitempty"`
	Perception int `json:"perception,omitempty"`
	Vitality   int `json:"vitality,omitempty"`
	Willpower  int `json:"willpower,omitempty"`
	Charisma   int `json:"charisma,omitempty"`
}

// /////////////////
// Char.Vitals
// /////////////////
type GMCPCharModule_Payload_Vitals struct {
	Hp            int `json:"hp,omitempty"`
	HpMax         int `json:"hp_max,omitempty"`
	Stamina       int `json:"stamina,omitempty"`
	StaminaMax    int `json:"stamina_max,omitempty"`
	Conviction    int `json:"conviction,omitempty"`
	ConvictionMax int `json:"conviction_max,omitempty"`
}

// /////////////////
// Char.Worth
// /////////////////
type GMCPCharModule_Payload_Worth struct {
	Gold           int `json:"gold_carry,omitempty"`
	Bank           int `json:"gold_bank,omitempty"`
	SkillPoints    int `json:"skillpoints,omitempty"`
	TrainingPoints int `json:"trainingpoints,omitempty"`
	TNL            int `json:"tnl,omitempty"`
	XP             int `json:"xp,omitempty"`
}

// /////////////////
// Char.Affects
// /////////////////
type GMCPCharModule_Payload_Affect struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	DurationMax  int            `json:"duration_max"`
	DurationLeft int            `json:"duration_cur"`
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
	Name   string `json:"name"`
	Type   string `json:"type"`
	Hunger string `json:"hunger"`
}
