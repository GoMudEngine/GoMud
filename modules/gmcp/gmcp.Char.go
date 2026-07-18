package gmcp

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/casing"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// pendingCharSpawnRefresh tracks userIds that spawned but may not have received
// their initial full Char push yet (e.g. a web client whose websocket GMCP frame
// raced the login burst, so isGMCPEnabled was still false at PlayerSpawn time).
// Without this, the Status & Conditions header + conditions only recover when a
// later CharacterChanged fires (e.g. a buff add/refresh). The next NewRound
// re-pushes "Char" once for these users and clears them, guaranteeing delivery
// after the connection is GMCP-ready. Mirrors the pattern in gmcp.Automation.go.
// Steady-state cost is zero once drained.
var (
	pendingCharSpawnRefresh   = map[int]struct{}{}
	pendingCharSpawnRefreshMu sync.Mutex
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
	events.RegisterListener(events.CharacterTrained{}, g.charTrainedHandler)
	events.RegisterListener(GMCPCharUpdate{}, g.buildAndSendGMCPPayload)
	events.RegisterListener(events.CharacterStatsChanged{}, g.statsChangeHandler)
	events.RegisterListener(events.CharacterChanged{}, g.charChangeHandler)
	events.RegisterListener(events.BuffsTriggered{}, g.buffTriggeredHandler)
	// Re-push the full Char payload once on the round following spawn so the
	// initial login push lands reliably even if the first GMCP frame raced the
	// connection setup (otherwise the status panel header + conditions stay
	// blank until a later CharacterChanged event).
	events.RegisterListener(events.NewRound{}, g.newRoundHandler)

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
		Identifier: `Char.Affects, Char.Conditions`,
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
		Identifier: `Char.Vitals, Char.Conditions`,
	})

	return events.Continue
}

func (g *GMCPCharModule) ownershipChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.ItemOwnership)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	// Refresh the whole inventory: get/drop/give can move items between the
	// backpack, the potion bandolier, and the component bag, so the worn
	// sub-containers need to refresh too.
	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Inventory`,
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
func (g *GMCPCharModule) playerSpawnHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Send full update now (covers telnet/Mudlet clients that have already
	// negotiated GMCP)...
	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char`,
	})

	// ...and queue a one-shot re-push for the next round, which guarantees
	// delivery for web clients once the connection is settled.
	pendingCharSpawnRefreshMu.Lock()
	pendingCharSpawnRefresh[evt.UserId] = struct{}{}
	pendingCharSpawnRefreshMu.Unlock()

	return events.Continue
}

func (g *GMCPCharModule) newRoundHandler(e events.Event) events.ListenerReturn {

	if _, typeOk := e.(events.NewRound); !typeOk {
		return events.Continue
	}

	pendingCharSpawnRefreshMu.Lock()
	if len(pendingCharSpawnRefresh) == 0 {
		pendingCharSpawnRefreshMu.Unlock()
		return events.Continue
	}
	userIds := make([]int, 0, len(pendingCharSpawnRefresh))
	for userId := range pendingCharSpawnRefresh {
		userIds = append(userIds, userId)
	}
	pendingCharSpawnRefresh = map[int]struct{}{}
	pendingCharSpawnRefreshMu.Unlock()

	for _, userId := range userIds {
		events.AddToQueue(GMCPCharUpdate{
			UserId:     userId,
			Identifier: `Char`,
		})
	}

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
			Class:     skills.GetTitle(user.Character.Mutations, user.Character.GetAllSkillRanks(), user.Character.Stats),
			Race:      casing.Title(user.Character.Species()),
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
					Max:   int(user.Character.CarryCapacity()),
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
					Max:   int(user.Character.CarryCapacity()),
				},
			},

			Worn:         buildWornSlots(user.Character),
			Bandolier:    buildBandolierContainer(user.Character),
			ComponentBag: buildComponentBagContainer(user.Character),
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

			HpReserved:         user.Character.GetPoolReservation("health", user.Character.HealthMax.Value),
			StaminaReserved:    user.Character.GetPoolReservation("stamina", user.Character.StaminaMax.Value),
			ConvictionReserved: user.Character.GetPoolReservation("conviction", user.Character.ConvictionMax.Value),

			Toxicity: user.Character.ToxicityBandName(),
		}

		if !all {
			return payload.Vitals, `Char.Vitals`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Worth`, gmcpModule) {

		payload.Worth = &GMCPCharModule_Payload_Worth{
			Gold: user.Character.Gold,
			Bank: user.Character.Bank,
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

			// Stealth buffs (Hidden, Empathic Shroud, etc.) are deliberately not
			// surfaced to the client — mirrors the `conditions` command filter and
			// the no-end-message design on the hidden buff. If you can't know who
			// spotted you, you shouldn't be told you're hidden.
			if slices.Contains(buffSpec.Flags, buffs.Hidden) {
				continue
			}

			// Warcry/Rally are surfaced through Char.Conditions (the combat
			// condition list); their mirror buff is skipped here so the web
			// client doesn't render the same effect as both an Affect and a
			// Condition chip.
			if slices.Contains(buffSpec.Flags, buffs.ConditionMirror) {
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

		engine := questengine.GetEngine()
		focusId := user.Character.GetFocusQuestId()

		for questId, questStep := range user.Character.GetQuestProgress() {

			qDef := engine.GetQuest(questId)
			if qDef == nil {
				continue
			}

			// Secret quests are not sent.
			if qDef.Secret {
				continue
			}

			completedSteps := 0
			totalSteps := len(qDef.Steps)

			questPayload := GMCPCharModule_Payload_Quest{
				Id:          questId,
				Name:        qDef.Name,
				Description: qDef.Description,
				Completion:  0,
				Focused:     questId == focusId,
			}

			for _, step := range qDef.Steps {
				completedSteps++
				if step.Id == questStep {
					questPayload.Description = step.Description
					questPayload.Hint = step.Hint
					break
				}
			}

			if totalSteps > 0 {
				questPayload.Completion = int(math.Floor(float64(completedSteps)/float64(totalSteps)) * 100)
			}

			// Marker data is computed for the focused quest only.
			if questPayload.Focused {
				if target := engine.ResolveQuestTarget(questId, questStep); target != 0 {
					questPayload.TargetRoom = target
					if next, dir, ok := mapper.NextStep(user.Character.RoomId, target); ok {
						questPayload.NextRoom = next
						questPayload.NextDir = dir
					}
				}
			}

			// Add to the returned output
			payload.Quests = append(payload.Quests, questPayload)
		}

		if !all {
			return payload.Quests, `Char.Quests`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Skills`, gmcpModule) {

		payload.Skills = map[string]string{}
		for skillName, rank := range user.Character.Skills {
			payload.Skills[skillName] = skills.GetSkillRankDescription(rank)
		}

		if !all {
			return payload.Skills, `Char.Skills`
		}
	}

	if all || g.wantsGMCPPayload(`Char.Conditions`, gmcpModule) {

		payload.Conditions = []GMCPCondition{}
		for _, cond := range user.Character.Conditions {
			payload.Conditions = append(payload.Conditions, GMCPCondition{
				Type:        cond.Type.DisplayName(),
				Description: cond.Type.Description(),
				Duration:    conditionDurationLabel(cond.Duration),
			})
		}

		if !all {
			return payload.Conditions, `Char.Conditions`
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
	Info       *GMCPCharModule_Payload_Info             `json:"Info,omitempty"`
	Affects    map[string]GMCPCharModule_Payload_Affect `json:"Affects,omitempty"`
	Enemies    []GMCPCharModule_Enemy                   `json:"Enemies,omitempty"`
	Inventory  *GMCPCharModule_Payload_Inventory        `json:"Inventory,omitempty"`
	Stats      *GMCPCharModule_Payload_Stats            `json:"Stats,omitempty"`
	Vitals     *GMCPCharModule_Payload_Vitals           `json:"Vitals,omitempty"`
	Worth      *GMCPCharModule_Payload_Worth            `json:"Worth,omitempty"`
	Quests     []GMCPCharModule_Payload_Quest           `json:"Quests,omitempty"`
	Pets       []GMCPCharModule_Payload_Pet             `json:"Pets,omitempty"`
	Skills     map[string]string                        `json:"Skills,omitempty"`
	Conditions []GMCPCondition                          `json:"Conditions,omitempty"`
}

// /////////////////
// Char.Conditions
// /////////////////
type GMCPCondition struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Duration    string `json:"duration"` // "sustained", "briefly", "for a while", "extended"
}

// conditionDurationLabel converts rounds remaining to a qualitative label.
// 0 = permanent/sustained; 1–3 = briefly; 4–10 = for a while; 11+ = extended.
func conditionDurationLabel(rounds int) string {
	switch {
	case rounds == 0:
		return "sustained"
	case rounds <= 3:
		return "briefly"
	case rounds <= 10:
		return "for a while"
	default:
		return "extended"
	}
}

// /////////////////
// Char.Info
// /////////////////
type GMCPCharModule_Payload_Info struct {
	Account   string `json:"account,omitempty"`
	Name      string `json:"name,omitempty"`
	Class     string `json:"class,omitempty"`
	Race      string `json:"race,omitempty"`
}

// /////////////////
// Char.Enemies
// /////////////////
type GMCPCharModule_Enemy struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Hp      int    `json:"hp"`
	MaxHp   int    `json:"hp_max"`
	Engaged bool   `json:"engaged"`
}

// /////////////////
// Char.Inventory
// /////////////////
type GMCPCharModule_Payload_Inventory struct {
	Worn         []GMCPCharModule_Payload_Slot              `json:"Worn"`
	Bandolier    *GMCPCharModule_Payload_Container          `json:"Bandolier,omitempty"`
	ComponentBag *GMCPCharModule_Payload_Container          `json:"ComponentBag,omitempty"`
	Backpack     *GMCPCharModule_Payload_Inventory_Backpack `json:"Backpack,omitempty"`
}

// GMCPCharModule_Payload_Slot is one equipment slot in eq/worn order. Item is
// null when the slot is empty. Mutation-gated slots (extra arms/wrists, tail)
// are only emitted when the character actually has that slot.
type GMCPCharModule_Payload_Slot struct {
	Slot    string                                 `json:"slot"`    // display label e.g. "Weapon","Arm 3"
	SlotKey string                                 `json:"slotKey"` // "weapon","offhand","extraarm1",...
	Item    *GMCPCharModule_Payload_Inventory_Item `json:"item"`    // null when empty
}

// GMCPCharModule_Payload_Container is a sub-inventory carried by a worn item
// (potion bandolier, component bag).
type GMCPCharModule_Payload_Container struct {
	Items   []GMCPCharModule_Payload_Inventory_Item           `json:"items,omitempty"`
	Summary GMCPCharModule_Payload_Inventory_Backpack_Summary `json:"Summary,omitempty"`
}

type GMCPCharModule_Payload_Inventory_Backpack struct {
	Items   []GMCPCharModule_Payload_Inventory_Item           `json:"items,omitempty"`
	Summary GMCPCharModule_Payload_Inventory_Backpack_Summary `json:"Summary,omitempty"`
}

type GMCPCharModule_Payload_Inventory_Backpack_Summary struct {
	Count int `json:"count,omitempty"`
	Max   int `json:"max,omitempty"`
}

type GMCPCharModule_Payload_Inventory_Item struct {
	Id      string   `json:"id"`
	Handle  string   `json:"handle"`  // opaque per-instance target token (item UUID)
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	SubType string   `json:"subtype"`
	Uses    int      `json:"uses"`
	UsesMax int      `json:"usesMax"` // spec starting/max charges; 0 = not a charged item
	Details []string `json:"details"`
}

func newInventory_Item(itm items.Item) GMCPCharModule_Payload_Inventory_Item {

	itmSpec := itm.GetSpec()
	d := GMCPCharModule_Payload_Inventory_Item{
		Id:      itm.ShorthandId(),
		Handle:  itm.UUID.String(),
		Name:    itm.Name(),
		Type:    string(itmSpec.Type),
		SubType: string(itmSpec.Subtype),
		Uses:    itm.Uses,
		UsesMax: itmSpec.Uses, // spec's starting count = max charges
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

// newInventory_ItemPtr returns a pointer to the item payload for an occupied
// slot, or nil when the slot is empty/disabled (ItemId < 1 covers both the
// empty zero-value and the negative ItemDisabledSlot sentinel).
func newInventory_ItemPtr(itm items.Item) *GMCPCharModule_Payload_Inventory_Item {
	if itm.ItemId < 1 {
		return nil
	}
	d := newInventory_Item(itm)
	return &d
}

// buildWornSlots emits the equipment slots in worn.go/eq order. Base slots are
// always present (Item nil when empty); mutation-gated slots (extra arms,
// extra wrists, tail) are only emitted when the character has them, so the
// list stays in eq order.
func buildWornSlots(c *characters.Character) []GMCPCharModule_Payload_Slot {

	slots := make([]GMCPCharModule_Payload_Slot, 0, 26)

	// Mutation-slot predicates: ExtraArms count is derived/capped in
	// validateMutationSlots; tail presence is the "tail" mutation key.
	_, hasTail := c.Mutations["tail"]

	// includeSlot decides whether a given slot key is emitted for this
	// character. Base slots are always present; mutation-gated slots
	// (extra arms/wrists) appear per ExtraArms count, and tail/legs are
	// XOR-gated on the tail mutation. The slot list + labels themselves
	// come from characters.Worn.AllSlots() (single source of truth).
	includeSlot := func(key string) bool {
		switch key {
		case `extraarm1`, `extrawrist1`:
			return c.ExtraArms >= 1
		case `extraarm2`, `extrawrist2`:
			return c.ExtraArms >= 2
		case `extraarm3`, `extrawrist3`:
			return c.ExtraArms >= 3
		case `extraarm4`, `extrawrist4`:
			return c.ExtraArms >= 4
		case `tail`:
			return hasTail
		case `legs`:
			return !hasTail
		default:
			return true
		}
	}

	for _, s := range c.Equipment.AllSlots() {
		if !includeSlot(s.Key) {
			continue
		}
		slots = append(slots, GMCPCharModule_Payload_Slot{
			Slot:    s.Label,
			SlotKey: s.Key,
			Item:    newInventory_ItemPtr(*s.Item),
		})
	}

	return slots
}

// buildBandolierContainer returns the potion bandolier sub-inventory when a
// bandolier belt is worn, or nil otherwise.
func buildBandolierContainer(c *characters.Character) *GMCPCharModule_Payload_Container {

	beltSpec := c.Equipment.Belt.GetSpec()
	if !beltSpec.IsBandolier {
		return nil
	}

	cont := &GMCPCharModule_Payload_Container{
		Items: []GMCPCharModule_Payload_Inventory_Item{},
		Summary: GMCPCharModule_Payload_Inventory_Backpack_Summary{
			Count: len(c.PotionItems),
			Max:   beltSpec.BandolierCapacity,
		},
	}
	for _, itm := range c.PotionItems {
		cont.Items = append(cont.Items, newInventory_Item(itm))
	}
	return cont
}

// buildComponentBagContainer returns the component-bag sub-inventory when a
// component bag is worn, or nil otherwise.
func buildComponentBagContainer(c *characters.Character) *GMCPCharModule_Payload_Container {

	if c.Equipment.ComponentBag.ItemId < 1 {
		return nil
	}

	contents := c.GetComponentBagContents()
	cont := &GMCPCharModule_Payload_Container{
		Items: []GMCPCharModule_Payload_Inventory_Item{},
		Summary: GMCPCharModule_Payload_Inventory_Backpack_Summary{
			Count: len(contents),
			Max:   c.Equipment.ComponentBag.GetSpec().BagCapacity,
		},
	}
	for _, itm := range contents {
		cont.Items = append(cont.Items, newInventory_Item(itm))
	}
	return cont
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

	HpReserved         int `json:"hp_reserved,omitempty"`
	StaminaReserved    int `json:"stamina_reserved,omitempty"`
	ConvictionReserved int `json:"conviction_reserved,omitempty"`

	// Toxicity severity band: "clear", "queasy", "sick", or "critical".
	// Always present so the web client can track transitions.
	Toxicity string `json:"toxicity"`
}

// /////////////////
// Char.Worth
// /////////////////
type GMCPCharModule_Payload_Worth struct {
	Gold int `json:"gold_carry,omitempty"`
	Bank int `json:"gold_bank,omitempty"`
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
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Completion  int    `json:"completion"`
	Hint        string `json:"hint,omitempty"`
	Focused     bool   `json:"focused,omitempty"`
	// Focused quest only; omitted/zero when there is no resolvable target.
	TargetRoom int    `json:"target_room,omitempty"`
	NextRoom   int    `json:"next_room,omitempty"`
	NextDir    string `json:"next_dir,omitempty"`
}

// /////////////////
// Char.Pets
// /////////////////
type GMCPCharModule_Payload_Pet struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Hunger string `json:"hunger"`
}
