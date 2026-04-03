package scripting

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/dop251/goja"
)

func setActorFunctions(vm *goja.Runtime) {
	vm.Set(`GetUser`, GetUser)
	vm.Set(`GetMob`, GetMob)
	vm.Set(`ActorNames`, ActorNames)
}

type ScriptActor struct {
	userId          int
	mobInstanceId   int
	userRecord      *users.UserRecord
	mobRecord       *mobs.Mob
	characterRecord *characters.Character // Lets us bypass the user/mob check in many cases
}

func (a ScriptActor) UserId() int {
	return a.userId
}

func (a ScriptActor) InstanceId() int {
	return a.mobInstanceId
}

func (a ScriptActor) MobTypeId() int {
	if a.mobRecord != nil {
		return int(a.mobRecord.MobId)
	}
	return 0
}

func (a ScriptActor) GetSize() string {
	if r := species.GetSpecies(a.characterRecord.SpeciesId); r != nil {
		return string(r.Size)
	}
	return string(species.Medium)
}

func (a ScriptActor) SendText(msg string) {
	if a.userRecord == nil {
		return
	}

	msg = userTextWrap.Wrap(msg)

	a.userRecord.SendText(msg)
}

func (a ScriptActor) GetStat(statName string) int {

	statName = strings.ToLower(statName)

	if strings.HasPrefix(statName, "st") {
		return a.characterRecord.Stats.Strength.ValueAdj
	}

	if strings.HasPrefix(statName, "de") {
		return a.characterRecord.Stats.Dexterity.ValueAdj
	}

	if strings.HasPrefix(statName, "pe") {
		return a.characterRecord.Stats.Perception.ValueAdj
	}

	if strings.HasPrefix(statName, "vi") {
		return a.characterRecord.Stats.Vitality.ValueAdj
	}

	if strings.HasPrefix(statName, "wi") {
		return a.characterRecord.Stats.Willpower.ValueAdj
	}

	if strings.HasPrefix(statName, "ch") {
		return a.characterRecord.Stats.Charisma.ValueAdj
	}

	return 0
}

func (a ScriptActor) SetTempData(key string, value any) {

	if a.userRecord != nil {
		if userValue, ok := value.(ScriptActor); ok { // Don't store pointer to user data.
			userValue.userRecord = nil
			value = userValue
		}
		a.userRecord.SetTempData(key, value)
		return
	}

	if a.mobRecord != nil {
		if userValue, ok := value.(ScriptActor); ok { // Don't store pointer to user data.
			userValue.mobRecord = nil
			value = userValue
		}
		a.mobRecord.SetTempData(key, value)
		return
	}
}

func (a ScriptActor) GetTempData(key string) any {

	if a.userRecord != nil {
		if value := a.userRecord.GetTempData(key); value != nil {
			if userValue, ok := value.(ScriptActor); ok { // If it was userdata we need to reload the whole thing in case the user isn't around anymore.
				value = GetActor(userValue.userId, 0)
			}
			return value
		}
	} else if a.mobRecord != nil {
		if value := a.mobRecord.GetTempData(key); value != nil {
			if mobValue, ok := value.(ScriptActor); ok { // If it was userdata we need to reload the whole thing in case the user isn't around anymore.
				value = GetActor(0, mobValue.mobInstanceId)
			}
			return value
		}
	}
	return nil
}

func (a ScriptActor) GetTameMastery() map[int]int {
	return a.characterRecord.MobMastery.GetAllTame()
}

func (a ScriptActor) SetTameMastery(mobId int, newSkillLevel int) {
	a.characterRecord.MobMastery.SetTame(mobId, newSkillLevel)
}

func (a ScriptActor) GetChanceToTame(target ScriptActor) int {
	return combat.ChanceToTame(a.userRecord, target.mobRecord)
}

func (a ScriptActor) SetMiscCharacterData(key string, value any) {

	if _, ok := value.(ScriptActor); ok { // Don't store actor data.
		return
	}
	a.characterRecord.SetMiscData(key, value)
}

func (a ScriptActor) GetMiscCharacterData(key string) any {
	if value := a.characterRecord.GetMiscData(key); value != nil {
		return value
	}
	return nil
}

func (a ScriptActor) GetMiscCharacterDataKeys(prefixMatches ...string) []string {
	return a.characterRecord.GetMiscDataKeys(prefixMatches...)
}

func (a ScriptActor) GetCharacterName(wrapInTags bool) string {

	if wrapInTags {
		if a.userRecord != nil {
			return `<ansi fg="username">` + a.characterRecord.Name + `</ansi>`
		} else if a.mobRecord != nil {
			return `<ansi fg="mobname">` + a.characterRecord.Name + `</ansi>`
		}
	}

	return a.characterRecord.Name
}

func (a ScriptActor) SetCharacterName(newName string) {
	a.characterRecord.Name = newName
}

func (a ScriptActor) GetRoomId() int {
	return a.characterRecord.RoomId
}

func (a ScriptActor) HasQuest(questId string) bool {
	return a.characterRecord.HasQuest(questId)
}

func (a ScriptActor) GetQuestFlag(key string) string {
	return a.characterRecord.GetQuestFlag(key)
}

func (a ScriptActor) SetQuestFlag(key, value string) {
	a.characterRecord.SetQuestFlag(key, value)
}

func (a ScriptActor) HasQuestFlag(key string) bool {
	return a.characterRecord.HasQuestFlag(key)
}

func (a ScriptActor) GiveQuest(questId string) {

	if a.userRecord != nil {
		// If in a party, give to all party members.
		if party := parties.Get(a.userId); party != nil {
			for _, userId := range party.GetMembers() {

				events.AddToQueue(events.Quest{
					UserId:     userId,
					QuestToken: questId,
				})

			}
			return
		} else {

			events.AddToQueue(events.Quest{
				UserId:     a.userId,
				QuestToken: questId,
			})

		}
	}
	//a.characterRecord.GiveQuestToken(questId)

}

func (a ScriptActor) GetPartyMembers() []ScriptActor {

	partyMembers := []ScriptActor{}
	partyUserId := 0

	if a.userRecord == nil {
		if a.mobRecord.Character.Charmed == nil {
			return partyMembers
		}

		partyUserId = a.mobRecord.Character.Charmed.UserId
	} else {
		partyUserId = a.userId
	}

	if partyUserId < 1 {
		return partyMembers
	}

	// If in a party, give to all party members.
	if party := parties.Get(partyUserId); party != nil {
		for _, userId := range party.GetMembers() {

			if a := GetActor(userId, 0); a != nil {
				partyMembers = append(partyMembers, *a)
			}

		}
	}

	mobPartyMembers := []ScriptActor{}

	for _, char := range partyMembers {
		for _, mobInstId := range char.characterRecord.GetCharmIds() {
			if a := GetActor(0, mobInstId); a != nil {
				mobPartyMembers = append(mobPartyMembers, *a)
			}
		}
	}

	return append(partyMembers, mobPartyMembers...)
}

func (a ScriptActor) AddGold(amt int, bankAmt ...int) {
	a.characterRecord.Gold += amt
	if a.characterRecord.Gold < 0 {
		a.characterRecord.Gold = 0
	}
	if len(bankAmt) > 0 {
		a.characterRecord.Bank += bankAmt[0]
		if a.characterRecord.Bank < 0 {
			a.characterRecord.Bank = 0
		}
	}
}

func (a ScriptActor) AddHealth(amt int) int {
	ret := a.characterRecord.ApplyHealthChange(amt)

	if ret != 0 && a.userId > 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: a.userId})
	}

	return ret
}

func (a ScriptActor) AddStamina(amt int) int {
	old := a.characterRecord.Stamina
	a.characterRecord.Stamina += amt
	if a.characterRecord.Stamina > a.characterRecord.StaminaMax.Value {
		a.characterRecord.Stamina = a.characterRecord.StaminaMax.Value
	}
	if a.characterRecord.Stamina < 0 {
		a.characterRecord.Stamina = 0
	}
	actual := a.characterRecord.Stamina - old
	if actual != 0 && a.userId > 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: a.userId})
	}
	return actual
}

func (a ScriptActor) AddConviction(amt int) int {
	old := a.characterRecord.Conviction
	a.characterRecord.Conviction += amt
	if a.characterRecord.Conviction > a.characterRecord.ConvictionMax.Value {
		a.characterRecord.Conviction = a.characterRecord.ConvictionMax.Value
	}
	if a.characterRecord.Conviction < 0 {
		a.characterRecord.Conviction = 0
	}
	actual := a.characterRecord.Conviction - old
	if actual != 0 && a.userId > 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: a.userId})
	}
	return actual
}

func (a ScriptActor) Sleep(seconds int) {
	if a.userId == 0 {
		a.mobRecord.Sleep(seconds)
	}
}

func (a ScriptActor) Command(cmd string, waitSeconds ...float64) {
	if len(waitSeconds) < 1 {
		waitSeconds = append(waitSeconds, 0)
	}
	if a.userId > 0 {
		a.userRecord.Command(cmd, waitSeconds[0])
	} else {
		a.mobRecord.Command(cmd, waitSeconds[0])
	}
}

func (a ScriptActor) CommandFlagged(cmd string, flags events.EventFlag, waitSeconds ...float64) {
	if len(waitSeconds) < 1 {
		waitSeconds = append(waitSeconds, 0)
	}
	if a.userId > 0 {
		a.userRecord.CommandFlagged(cmd, flags, waitSeconds[0])
	} else {
		a.mobRecord.Command(cmd, waitSeconds[0])
	}
}

func (a ScriptActor) TrainSkill(skillName string, skillLevel int) bool {

	if a.userRecord == nil {
		return false
	}

	skillName = strings.ToLower(skillName)
	currentLevel := a.characterRecord.GetSkillLevel(skills.SkillTag(skillName))

	if currentLevel < skillLevel {
		newLevel := a.characterRecord.TrainSkill(skillName, skillLevel)

		skillData := struct {
			SkillName  string
			SkillLevel int
		}{
			SkillName:  skillName,
			SkillLevel: newLevel,
		}
		skillUpTxt, _ := templates.Process("character/skillup", skillData, a.userRecord.UserId)
		a.SendText(skillUpTxt)

		return true

	}
	return false
}

func (a ScriptActor) GetSkillLevel(skillName string) int {
	return a.characterRecord.GetSkillLevel(skills.SkillTag(skillName))
}

func (a ScriptActor) MoveRoom(destRoomId int, leaveCharmedMobs ...bool) {

	if a.userRecord != nil {

		rmNow := rooms.LoadRoom(a.characterRecord.RoomId)

		if rmNext := rooms.LoadRoom(destRoomId); rmNext != nil {

			rooms.MoveToRoom(a.userId, destRoomId)

			if len(leaveCharmedMobs) < 1 || !leaveCharmedMobs[0] {
				for _, mobInstId := range a.characterRecord.GetCharmIds() {
					rmNow.RemoveMob(mobInstId)
					rmNext.AddMob(mobInstId)
				}
			}

			if doLook, err := TryRoomScriptEvent(`onEnter`, a.userRecord.UserId, destRoomId); err != nil || doLook {
				a.userRecord.CommandFlagged(`look`, events.CmdSecretly) // Do a secret look.
			}
		}

	} else if a.mobRecord != nil {

		if mobRoom := rooms.LoadRoom(a.characterRecord.RoomId); mobRoom != nil {
			if destRoom := rooms.LoadRoom(destRoomId); destRoom != nil {
				mobRoom.RemoveMob(a.mobInstanceId)
				destRoom.AddMob(a.mobInstanceId)
			}
		}

	}
}

func (a ScriptActor) UpdateItem(itm ScriptItem) {
	a.userRecord.Character.UpdateItem(itm.originalItem, *itm.itemRecord)
}

func (a ScriptActor) AddEventLog(category string, message string) {
	if a.userRecord != nil {
		a.userRecord.EventLog.Add(category, message)
	}
}

func (a ScriptActor) GiveItem(itm any) {

	var sItem *ScriptItem

	if itmScriptItem, ok := itm.(*ScriptItem); ok {
		sItem = itmScriptItem
	} else if itmInt, ok := itm.(int); ok {
		sItem = CreateItem(itmInt)
	} else if itmInt64, ok := itm.(int64); ok {
		sItem = CreateItem(int(itmInt64))
	} else if itmInt32, ok := itm.(int32); ok {
		sItem = CreateItem(int(itmInt32))
	} else if itmFloat64, ok := itm.(float64); ok {
		sItem = CreateItem(int(itmFloat64))
	}

	if sItem != nil {
		iRecord := sItem.itemRecord
		if a.characterRecord.StoreItem(*iRecord) {
			if a.userId > 0 {

				events.AddToQueue(events.ItemOwnership{
					UserId: a.userId,
					Item:   *iRecord,
					Gained: true,
				})

			}
		}
	}

}

func (a ScriptActor) TakeItem(itm ScriptItem) {
	if a.characterRecord.RemoveItem(*itm.itemRecord) {
		if a.userId > 0 {

			events.AddToQueue(events.ItemOwnership{
				UserId: a.userId,
				Item:   *itm.itemRecord,
				Gained: false,
			})

		}
	}
}

func (a ScriptActor) IsTameable() bool {
	if a.mobRecord == nil {
		return false
	}
	return a.mobRecord.IsTameable()
}

func (a ScriptActor) HasBuff(buffId int) bool {
	return a.characterRecord.HasBuff(buffId)
}

func (a ScriptActor) GiveBuff(buffId int, source string) {

	events.AddToQueue(events.Buff{
		UserId:        a.userId,
		MobInstanceId: a.mobInstanceId,
		BuffId:        buffId,
		Source:        source,
	})

}

func (a ScriptActor) GetStatMod(statModName string) int {
	return a.characterRecord.StatMod(statModName)
}

func (a ScriptActor) HasBuffFlag(buffFlag string) bool {
	return a.characterRecord.HasBuffFlag(buffs.Flag(buffFlag))
}

func (a ScriptActor) CancelBuffWithFlag(buffFlag string) bool {

	found := false

	for _, buffId := range a.characterRecord.Buffs.GetBuffIdsWithFlag(buffs.Flag(strings.ToLower(buffFlag))) {
		found = found || a.RemoveBuff(buffId)
	}

	return found
}

// Remove a buff silently
func (a ScriptActor) RemoveBuff(buffId int) bool {

	if !configs.GetGamePlayConfig().AllowItemBuffRemoval {
		buffList := a.characterRecord.GetBuffs(buffId)
		if len(buffList) > 0 {
			if buffList[0].PermaBuff {
				return false
			}
		}
	}

	return a.characterRecord.Buffs.RemoveBuff(buffId)

}

func (a ScriptActor) HasItemId(itemId int, excludeWorn ...bool) bool {
	for _, itm := range a.characterRecord.GetAllCarriedItems() {
		if itm.ItemId == itemId {
			return true
		}
	}
	if len(excludeWorn) == 0 || !excludeWorn[0] {
		for _, itm := range a.characterRecord.GetAllWornItems() {
			if itm.ItemId == itemId {
				return true
			}
		}
	}
	return false
}

func (a ScriptActor) GetBackpackItems() []ScriptItem {
	itms := make([]ScriptItem, 0, 5)
	for _, item := range a.characterRecord.GetAllCarriedItems() {
		itms = append(itms, newScriptItem(item))
	}
	return itms
}

func (a ScriptActor) HasSpell(spellId string) bool {
	return a.characterRecord.HasSpell(spellId)
}

func (a ScriptActor) LearnSpell(spellId string) bool {
	return a.characterRecord.LearnSpell(spellId)
}

// GiveMutation gives the player a specific mutation at level 1.
// Returns true if the mutation was newly granted, false if already owned or user is a mob.
func (a ScriptActor) GiveMutation(mutationId string) bool {
	if a.userRecord == nil {
		return false
	}
	if a.characterRecord.Mutations == nil {
		a.characterRecord.Mutations = make(map[string]int)
	}
	if _, exists := a.characterRecord.Mutations[mutationId]; !exists {
		a.characterRecord.Mutations[mutationId] = 1
		a.characterRecord.Validate()
		return true
	}
	return false
}

// GetMutationCount returns the number of mutations the character currently has.
func (a ScriptActor) GetMutationCount() int {
	if a.characterRecord.Mutations == nil {
		return 0
	}
	return len(a.characterRecord.Mutations)
}

// RollMutation randomly selects a mutation ID from the weighted pool that the player doesn't already own.
// Returns an empty string if no mutations are available (e.g., player owns all mutations).
func (a ScriptActor) RollMutation() string {
	if a.userRecord == nil {
		return ""
	}
	pool := mutations.GetWeightedPool(a.characterRecord.Mutations)
	if len(pool) == 0 {
		return ""
	}
	return mutations.RollAcquisition(pool)
}

func (a ScriptActor) IsAggro(actor ScriptActor) bool {
	return a.characterRecord.IsAggro(actor.UserId(), actor.InstanceId())
}

func (a ScriptActor) GetMobKills(mobId int) int {
	return a.characterRecord.KD.GetMobKills(mobId)
}

func (a ScriptActor) GetRaceKills(race string) int {

	raceKills := map[string]int{}

	for mid, kCt := range a.characterRecord.KD.Kills {
		if mobSpec := mobs.GetMobSpec(mobs.MobId(mid)); mobSpec != nil {
			if raceInfo := species.GetSpecies(mobSpec.Character.SpeciesId); raceInfo != nil {
				raceKills[raceInfo.Name] = raceKills[raceInfo.Name] + kCt
			}
		}
	}

	return raceKills[race]
}

func (a ScriptActor) SetHealth(amt int) {
	a.characterRecord.Health = amt
	if a.characterRecord.Health > a.characterRecord.HealthMax.Value {
		a.characterRecord.Health = a.characterRecord.HealthMax.Value
	}
}

func (a ScriptActor) GetHealth() int {
	return a.characterRecord.Health
}

func (a ScriptActor) GetHealthMax() int {
	return a.characterRecord.HealthMax.Value
}

func (a ScriptActor) GetHealthPct() float64 {
	return float64(a.characterRecord.Health) / float64(a.characterRecord.HealthMax.Value)
}

func (a ScriptActor) SetAdjective(adj string, addIt bool) {
	a.characterRecord.SetAdjective(adj, addIt)
}

func (a ScriptActor) IsHome() bool {
	if a.mobRecord != nil {
		return a.mobRecord.HomeRoomId == a.characterRecord.RoomId
	}
	return false
}

func (a ScriptActor) GetCharmCount() int {
	return len(a.characterRecord.GetCharmIds())
}

func (a ScriptActor) GetMaxCharmCount() int {
	return a.characterRecord.GetMaxCharmedCreatures()
}

func (a ScriptActor) GiveExtraLife() {
	c := configs.GetGamePlayConfig()
	a.characterRecord.ExtraLives += 1
	if a.characterRecord.ExtraLives > int(c.LivesMax) {
		a.characterRecord.ExtraLives = int(c.LivesMax)
	}
}

func (a ScriptActor) Uncurse() []*ScriptItem {

	retList := []*ScriptItem{}

	for _, itm := range a.characterRecord.Uncurse() {
		retList = append(retList, GetItem(itm))
	}

	return retList
}

func (a ScriptActor) GetPet() *pets.Pet {

	if a.characterRecord.Pet.Exists() {
		return &a.characterRecord.Pet
	}
	return nil
}

func (a ScriptActor) TimerSet(name string, period string) {
	a.characterRecord.TimerSet(name, period)
}

func (a ScriptActor) TimerExpired(name string) bool {
	return a.characterRecord.TimerExpired(name)
}

func (a ScriptActor) TimerExists(name string) bool {
	return a.characterRecord.TimerExists(name)
}

// ////////////////////////////////////////////////////////
//
// Functions only really useful for mobs
//
// ////////////////////////////////////////////////////////

// Returns true if a mob is charmed by/friendly to a player.
// If userId is ommitted, it will return true if the mob is charmed by any player.
func (a ScriptActor) IsCharmed(userId ...int) bool {
	if len(userId) < 1 {
		return a.characterRecord.IsCharmed()
	}
	return a.characterRecord.IsCharmed(userId[0])
}

func (a ScriptActor) GetCharmedUserId() int {
	return a.characterRecord.GetCharmedUserId()
}

// HasCharmedMobs returns true if this actor has any active charmed mobs.
func (a ScriptActor) HasCharmedMobs() bool {
	return len(a.characterRecord.CharmedMobs) > 0
}

func (a ScriptActor) CharmSet(userId int, charmRounds int, onRevertCommand ...string) {

	// If the player is in a party, add the mob to their party
	if a.mobInstanceId < 1 {
		return
	}

	if len(onRevertCommand) < 1 {
		onRevertCommand = append(onRevertCommand, ``)
	}

	// Anti-recursion: strip any companions this mob itself had before
	// charming it, so we never create companion chains.
	for _, subId := range a.characterRecord.GetCharmIds() {
		if subMob := mobs.GetInstance(subId); subMob != nil {
			subMob.Character.RemoveCharm()
			if subRoom := rooms.LoadRoom(subMob.Character.RoomId); subRoom != nil {
				subRoom.RemoveMob(subId)
			}
			mobs.DestroyInstance(subId)
		}
	}
	a.characterRecord.CharmedMobs = nil

	a.characterRecord.Charm(userId, charmRounds, onRevertCommand[0])

	if user := users.GetByUserId(userId); user != nil {
		user.Character.TrackCharmed(a.mobInstanceId, true)
	}

}

func (a ScriptActor) CharmRemove() {
	if a.characterRecord.Charmed == nil {
		return
	}
	charmUserId := a.characterRecord.RemoveCharm()

	if user := users.GetByUserId(charmUserId); user != nil {
		user.Character.TrackCharmed(a.mobInstanceId, false)
	}
}

func (a ScriptActor) CharmExpire() {
	a.characterRecord.Charmed.Expire()
}

func (a ScriptActor) getScript() string {
	if a.mobRecord != nil {
		return a.mobRecord.GetScript()
	}
	return ""
}

func (a ScriptActor) getScriptTag() string {
	if a.mobRecord != nil {
		return a.mobRecord.ScriptTag
	}
	return ""
}

func (a ScriptActor) ShorthandId() string {

	if a.userRecord != nil {

		return a.userRecord.ShorthandId()

	} else if a.mobRecord != nil {

		return a.mobRecord.ShorthandId()

	}

	return ``
}

func (a ScriptActor) GetLastInputRound() uint64 {
	if a.userRecord != nil {
		return a.userRecord.GetLastInputRound()
	}
	return 0
}

func (a ScriptActor) Pathing() bool {
	if a.mobRecord == nil {
		return false
	}
	return a.mobRecord.Path.Current() != nil || a.mobRecord.Path.Len() > 0
}

func (a ScriptActor) PathingAtWaypoint() bool {
	if a.mobRecord == nil {
		return false
	}
	pathStep := a.mobRecord.Path.Current()
	return pathStep != nil && pathStep.Waypoint()
}

// ////////////////////////////////////////////////////////
//
// # These functions get exported to the scripting engine
//
// ////////////////////////////////////////////////////////
func GetActor(userId int, mobInstanceId int) *ScriptActor {

	if userId > 0 {
		if user := users.GetByUserId(userId); user != nil {
			return &ScriptActor{
				userId:          userId,
				userRecord:      user,
				characterRecord: user.Character,
			}
		}
	} else if mobInstanceId > 0 {
		if mob := mobs.GetInstance(mobInstanceId); mob != nil {
			return &ScriptActor{
				mobInstanceId:   mobInstanceId,
				mobRecord:       mob,
				characterRecord: &mob.Character,
			}
		}
	}

	return nil
}

func GetUser(userId int) *ScriptActor {
	return GetActor(userId, 0)
}

func GetMob(mobInstanceId int) *ScriptActor {
	return GetActor(0, mobInstanceId)
}

func ActorNames(actorList []*ScriptActor) string {

	sBuilder := strings.Builder{}
	listSize := len(actorList)

	for i := 0; i < listSize; i++ {

		sBuilder.WriteString(actorList[i].GetCharacterName(true))

		if i < listSize-2 {
			sBuilder.WriteString(`, `)
		} else if i == listSize-2 {
			sBuilder.WriteString(`and `)
		}
	}

	return sBuilder.String()
}
