package mobs

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/conversations"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"

	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	instanceCounter int = 0
	mobs                = map[int]*Mob{}
	allMobNames         = []string{}
	mobInstances        = map[int]*Mob{}
	mobsHatePlayers     = map[string]map[int]int{}
	mobNameCache        = map[MobId]string{}

	recentlyDied = map[int]int{}
)

type ItemTrade struct {
	AcceptedItemIds []int         `yaml:"accepteditemids,omitempty,flow"` // Must provide every item id in this list.
	AcceptedGold    int           `yaml:"acceptedgold,omitempty,flow"`    // Must provide at least this much gold.
	PrizeItemIds    []int         `yaml:"prizeitemids,omitempty,flow"`    // Will give these items in exchange.
	PrizeBuffIds    []int         `yaml:"prizebuffids,omitempty,flow"`    // Will give these buffs in exchange.
	PrizeRoomId     int           `yaml:"prizeroomid,omitempty,flow"`     // Will move player to this room in exchange.
	PrizeQuestIds   []string      `yaml:"prizequestids,omitempty,flow"`   // What quest id's will be awarded?
	PrizeGold       int           `yaml:"prizegold,omitempty,flow"`       // How much gold are they given?
	PrizeCommands   []string      `yaml:"prizecommands,omitempty,flow"`   // What commands will be executed?
	GivenItems      map[int][]int `yaml:"-"`                              // key = userId, value = Items given. Should only contain items from AcceptedItemIds
	GivenGold       map[int]int   `yaml:"-"`                              // key = userId, value = how much gold is given
}

type MobForHire struct {
	MobId    MobId
	Price    int
	Quantity int
}
type MobId int // Creating a custom type to help prevent confusion over MobId and MobInstanceId

type Mob struct {
	MobId           MobId
	Zone            string   `yaml:"zone,omitempty"`
	ItemDropChance  int      // chance in 100
	ActivityLevel   int      `yaml:"activitylevel,omitempty"` // 1-100%
	InstanceId      int      `yaml:"-"`
	HomeRoomId      int      `yaml:"-"`
	Hostile         bool     // whether they attack on sight
	LastIdleCommand uint8    `yaml:"-"` // Track what hte last used idlecommand was
	BoredomCounter  uint8    `yaml:"-"` // how many rounds have passed since this mob has seen a player
	Groups          []string // What group do they identify with? Helps with teamwork
	Hates           []string `yaml:"hates,omitempty"`        // What NPC groups or races do they hate and probably fight if encountered?
	IdleCommands    []string `yaml:"idlecommands,omitempty"` // Commands they may do while idle (not in combat)
	AngryCommands   []string // randomly chosen to queue when they are angry/entering combat.
	CombatCommands  []string `yaml:"combatcommands,omitempty"` // Commands they may do while in combat
	Character       characters.Character
	MaxWander       int      `yaml:"maxwander,omitempty"`       // Max rooms to wander from home
	WanderCount     int      `yaml:"-"`                         // How many times this mob has wandered
	PreventIdle     bool     `yaml:"-"`                         // Whether they can't possibly be idle
	ScriptTag       string   `yaml:"scripttag"`                 // Script for this mob: mobs/frostfang/scripts/{mobId}-{mobname}-{ScriptTag}.js
	QuestFlags      []string `yaml:"questflags,omitempty,flow"` // What quest flags are set on this mob?
	BuffIds         []int    `yaml:"buffids,omitempty"`         // Buff Id's this mob always has upon spawn
	tempDataStore   map[string]any
	conversationId  int              // Identifier of conversation currently involved in.
	Path            PathQueue        `yaml:"-"` // a pre-calculated path the mob is following.
	lastCommandTurn uint64           // The last turn a command was scheduled for
	playersAttacked map[int]struct{} // all players this mob has attacked at some point
}

func MobInstanceExists(instanceId int) bool {

	_, ok := mobInstances[instanceId]
	return ok
}

// Gets a copy of all mob info
func GetAllMobInfo() []Mob {
	ret := []Mob{}
	for _, m := range mobs {
		ret = append(ret, *m)
	}
	return ret
}

func GetAllMobNames() []string {
	return append([]string{}, allMobNames...)
}

func TrackRecentDeath(instanceId int) {
	recentlyDied[instanceId] = int(util.GetRoundCount())
}

func RecentlyDied(instanceId int) bool {

	if len(recentlyDied) > 30 {
		roundNow := int(util.GetRoundCount())
		for k, v := range recentlyDied {
			if roundNow-v > 15 {
				delete(recentlyDied, k)
			}
		}
	}

	_, ok := recentlyDied[instanceId]

	return ok
}

func MobIdByName(mobName string) MobId {

	match, partial := util.FindMatchIn(mobName, allMobNames...)
	if match == "" {
		match = partial
	}
	if match == "" {
		return 0
	}

	for _, m := range mobs {
		if m.Character.Name == match {
			return m.MobId
		}
	}

	for _, m := range mobs {
		if strings.HasPrefix(m.Character.Name, match) {
			return m.MobId
		}
	}

	for _, m := range mobs {
		if strings.Contains(m.Character.Name, match) {
			return m.MobId
		}
	}

	return 0
}

func NewMobById(mobId MobId, homeRoomId int, forceLevel ...int) *Mob {

	if m, ok := mobs[int(mobId)]; ok {

		instanceCounter++

		mob := *m // Make a copy of the mob

		mob.HomeRoomId = homeRoomId
		mob.Character.RoomId = homeRoomId
		mob.InstanceId = instanceCounter
		mob.Character.PlayerDamage = make(map[int]int)

		// Level related stuff
		if len(forceLevel) > 0 && forceLevel[0] > 0 {
			mob.Character.Level = forceLevel[0]
		}
		mob.Character.StatPoints = mob.Character.Level
		mob.Character.Level--
		mob.Character.Experience = mob.Character.XPTNL()
		mob.Character.Level++

		// Apply training for those stats
		mob.Character.AutoTrain()
		mob.Character.Health = mob.Character.HealthMax.Value
		mob.Character.Conviction = mob.Character.ConvictionMax.Value

		mob.Character.SetPermaBuffs(mob.BuffIds)

		mob.Character.Buffs = buffs.New()

		for idx, _ := range mob.Character.Items {
			mob.Character.Items[idx].Validate()
		}

		if mob.Character.Alignment == 0 {
			if raceInfo := races.GetRace(mob.Character.RaceId); raceInfo != nil {
				if raceInfo.DefaultAlignment != 0 {
					mob.Character.Alignment = raceInfo.DefaultAlignment
				}
			}
		}

		mob.Character.Equipment.Weapon.Validate()
		mob.Character.Equipment.Offhand.Validate()
		mob.Character.Equipment.Head.Validate()
		mob.Character.Equipment.Neck.Validate()
		mob.Character.Equipment.Body.Validate()
		mob.Character.Equipment.Belt.Validate()
		mob.Character.Equipment.Gloves.Validate()
		mob.Character.Equipment.Ring.Validate()
		mob.Character.Equipment.Legs.Validate()
		mob.Character.Equipment.Feet.Validate()

		mob.Validate()
		mob.Character.Validate(true)

		// Save the mob instance
		mobInstances[mob.InstanceId] = &mob

		return mobInstances[mob.InstanceId]
	}
	return nil
}

func GetMobSpec(mobId MobId) *Mob {
	if m, ok := mobs[int(mobId)]; ok {
		mob := *m // Make a copy of the mob
		return &mob
	}
	return nil
}

func GetInstance(instanceId int) *Mob {

	if m, ok := mobInstances[instanceId]; ok {
		return m
	}
	return nil
}

func GetAllMobInstanceIds() []int {

	ids := make([]int, 0)
	for id := range mobInstances {
		ids = append(ids, id)
	}
	return ids
}

func DestroyInstance(instanceId int) {

	delete(mobInstances, instanceId)
}

func (m *Mob) ShorthandId() string {
	return fmt.Sprintf(`#%d`, m.InstanceId)
}

func (m *Mob) AddBuff(buffId int, source string) {

	events.AddToQueue(events.Buff{
		MobInstanceId: m.InstanceId,
		BuffId:        buffId,
		Source:        source,
	})

}

func (m *Mob) PlayerAttacked(userId int) {
	if m.playersAttacked == nil {
		m.playersAttacked = map[int]struct{}{}
	}
	m.playersAttacked[userId] = struct{}{}
}

func (m *Mob) HasAttackedPlayer(userId int) bool {
	if m.playersAttacked == nil {
		return false
	}
	_, ok := m.playersAttacked[userId]
	return ok
}

func (m *Mob) InConversation() bool {
	return m.conversationId > 0
}

func (m *Mob) SetConversation(id int) {
	m.conversationId = id
}

func (m *Mob) Converse() {

	mobInst1, mobInst2, actions := conversations.GetNextActions(m.conversationId)

	var mob1 *Mob = nil
	var mob2 *Mob = nil

	if mobInst1 == int(m.InstanceId) {
		mob1 = m
		mob2 = GetInstance(mobInst2)
	} else {
		mob1 = GetInstance(mobInst1)
		mob2 = m
	}

	if mob1 == nil || mob2 == nil {
		conversations.Destroy(m.conversationId)
		if mob1 != nil {
			mob1.SetConversation(0)
		}
		if mob2 != nil {
			mob2.SetConversation(0)
		}
		return
	}

	for _, act := range actions {
		if len(act) >= 3 {

			target := act[0:3]
			cmd := act[3:]

			cmd = strings.ReplaceAll(cmd, ` #1 `, ` `+mob1.ShorthandId()+` `)
			cmd = strings.ReplaceAll(cmd, ` #2 `, ` `+mob2.ShorthandId()+` `)

			if target == `#1 ` {
				mob1.Command(cmd)
			} else {
				mob2.Command(cmd, 1)
			}
		}
	}

	if conversations.IsComplete(m.conversationId) {
		conversations.Destroy(m.conversationId)
		mob1.SetConversation(0)
		mob2.SetConversation(0)
		return
	}
}

// Cause the mob to basically wait and do nothing for x seconds
func (m *Mob) Sleep(seconds int) {
	m.Command(`noop`, float64(seconds))
}

func (m *Mob) Command(inputTxt string, waitSeconds ...float64) {

	readyTurn := util.GetTurnCount()
	turnDelay := uint64(0)

	// m.lastCommandTurn is used so that subsequent calls to Command()
	// are scheduled from this period forward.
	// If it's been long enough that the current turn has surpassed the lastCommandTurn, we failover to that.
	if readyTurn > m.lastCommandTurn {
		m.lastCommandTurn = readyTurn
	} else {
		readyTurn = m.lastCommandTurn
	}

	if len(waitSeconds) > 0 {
		turnDelay = uint64(float64(configs.GetTimingConfig().SecondsToTurns(1)) * waitSeconds[0])
	}

	for i, cmd := range strings.Split(inputTxt, `;`) {

		// Update lastCommandTurn to whenever this command is scheduled for
		m.lastCommandTurn = readyTurn + turnDelay + uint64(i)

		events.AddToQueue(events.Input{
			MobInstanceId: m.InstanceId,
			InputText:     cmd,
			ReadyTurn:     m.lastCommandTurn,
		})

	}

}

func (m *Mob) HasShop() bool {
	return len(m.Character.Shop) > 0
}

func (m *Mob) IsTameable() bool {
	if m.HasShop() {
		return false
	}
	if len(m.ScriptTag) > 0 {
		return false
	}
	if r := races.GetRace(m.Character.RaceId); r != nil {
		if !r.Tameable {
			return false
		}
	}
	return true
}

func (m *Mob) SetTempData(key string, value any) {

	if m.tempDataStore == nil {
		m.tempDataStore = make(map[string]any)
	}

	if value == nil {
		delete(m.tempDataStore, key)
		return
	}
	m.tempDataStore[key] = value
}

func (m *Mob) GetTempData(key string) any {

	if m.tempDataStore == nil {
		m.tempDataStore = make(map[string]any)
	}

	if value, ok := m.tempDataStore[key]; ok {
		return value
	}
	return nil
}

func (m *Mob) Despawns() bool {
	if m.HasShop() {
		return false
	}
	return true
}

func (m *Mob) GetSellPrice(item items.Item) int {

	if item.IsSpecial() {
		return 0
	}

	itemType := item.GetSpec().Type
	itemSubtype := item.GetSpec().Subtype
	value := 0
	likesType := false
	likesSubtype := false
	newAddition := true
	priceScale := 0.0

	currentSaleItems := m.Character.Shop.GetInstock()

	for _, stockItm := range currentSaleItems {
		if stockItm.ItemId == 0 {
			continue
		}

		if stockItm.ItemId == item.ItemId { // If it's in stock, we can set everyting and break out
			newAddition = false // already stocking this item
			likesType = true
			likesSubtype = true
			value = stockItm.Price
			// Scale down amount willing to pay based on how many there are already in stock
			priceScale = 1.0 - (float64(stockItm.Quantity) / 20)
			break
		}

		tmpItm := items.New(stockItm.ItemId)
		if tmpItm.ItemId == 0 {
			continue
		}

		if !likesType && tmpItm.GetSpec().Type == itemType {
			likesType = true
			priceScale += 0.5
		}

		if !likesSubtype && tmpItm.GetSpec().Subtype == itemSubtype {
			likesSubtype = true
			priceScale += 0.5
		}
	}

	// If this is a new addition, don't allow more than 20 varieites
	if newAddition && len(currentSaleItems) >= 20 {
		return 0
	}

	if value == 0 {
		value = item.GetSpec().Value
	}

	if priceScale < 0 {
		priceScale = 0
	} else if priceScale > 100 {
		priceScale = 100
	}

	priceScale *= .25 // Can never be more than 25% value of object

	return int(math.Ceil(float64(value) * priceScale))
}

func (r *Mob) HatesRace(raceName string) bool {
	raceName = strings.ToLower(raceName)
	for _, hateGroup := range r.Hates {
		if hateGroup == raceName {
			return true
		}
	}
	return false
}

func (r *Mob) HatesAlignment(otherAlignment int8) bool {

	// If either are neutral, no hatred
	if characters.AlignmentToString(r.Character.Alignment) == `neutral` || characters.AlignmentToString(otherAlignment) == `neutral` {
		return false
	}

	// If both on the good side, no hatred
	if r.Character.Alignment > 0 && otherAlignment > 0 {
		return false
	}

	// If both on the evil side, no hatred
	if r.Character.Alignment < 0 && otherAlignment < 0 {
		return false
	}

	delta := int(math.Abs(float64(r.Character.Alignment) - float64(otherAlignment)))

	return delta > characters.AlignmentAggroThreshold
}

func (r *Mob) HatesMob(m *Mob) bool {
	if r.MobId == m.MobId {
		return false // Can't hate exact same as self
	}

	mRace := races.GetRace(m.Character.RaceId)
	raceName := strings.ToLower(mRace.Name)
	for _, rGroup := range r.Groups {
		if rGroup == raceName {
			return true
		}
		for _, mGroup := range m.Groups {
			if rGroup == mGroup {
				return false // Can't hate groups its part of.
			}
		}
	}
	// Loop through groups it hates and if it finds a match, return true
	for _, groupName := range r.Hates {
		if groupName == `*` { // If * it hates all groups
			return true
		}
		for _, mGroup := range m.Groups {
			if groupName == mGroup {
				return true
			}
		}
	}
	return false
}

func (m *Mob) GetAngryCommand() string {

	// First check if the mob has a specific action
	if len(m.AngryCommands) > 0 {
		return m.AngryCommands[util.Rand(len(m.AngryCommands))]
	}

	// default to race based actions
	r := races.GetRace(m.Character.RaceId)
	actionCt := len(r.AngryCommands)
	if actionCt > 0 {
		return r.AngryCommands[util.Rand(actionCt)]
	}
	return ``
}

func (m *Mob) GetIdleCommand() string {

	// Always a 1 in 100 chance it will do nothing for an idle.
	// This is to prevent requiring Admins to assign an empy idlecommand to mob definitions
	// while still allowing "no idle command found" behavior to run.
	// Empty idle commands can still be defined in mobs, however.
	if util.Rand(100) == 0 {
		return ``
	}

	// First check if the mob has a specific action
	if len(m.IdleCommands) > 0 {
		return m.IdleCommands[util.Rand(len(m.IdleCommands))]
	}

	return ``
}

func (r *Mob) ConsidersAnAlly(m *Mob) bool {

	if m.MobId == r.MobId {
		return true // Auto ally with own kind
	}

	if len(m.Groups) == 0 && len(r.Groups) == 0 {
		return true // No allegiance on either side, consider an ally for now
	}

	// If they both belong to factions/groups, check for matches
	// Could conver tthis to a look up map.
	// Only a couple entries likely, so maybe not worth it.
	if len(r.Groups) > 0 {
		// Look for a group match
		for _, targetGroup := range r.Groups {
			for _, testGroup := range m.Groups {
				if testGroup == targetGroup {
					return true
				}
			}
		}
	}

	return false
}

func (r *Mob) Id() int {
	return int(r.MobId)
}

func (r *Mob) Validate() error {

	if r.ActivityLevel < 1 {
		r.ActivityLevel = 10
	} else if r.ActivityLevel > 100 {
		r.ActivityLevel = 100
	}

	r.Character.Validate()

	return nil
}

func (m *Mob) Filename() string {
	if name, ok := mobNameCache[m.MobId]; ok {
		return fmt.Sprintf("%d-%s.yaml", m.Id(), util.ConvertForFilename(name))
	}
	// Failover to character name
	filename := util.ConvertForFilename(m.Character.Name)
	return fmt.Sprintf("%d-%s.yaml", m.Id(), filename)
}

func (m *Mob) Filepath() string {
	zone := ZoneNameSanitize(m.Zone)
	return util.FilePath(zone, `/`, m.Filename())
}

func (r *Mob) Save() error {

	fileName := r.Filename()

	bytes, err := yaml.Marshal(r)
	if err != nil {
		return err
	}

	saveFilePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `mobs`, `/`, fmt.Sprintf("%s.yaml", fileName))

	err = os.WriteFile(saveFilePath, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (m *Mob) HasScript() bool {

	scriptPath := m.GetScriptPath()
	// Load the script into a string
	if _, err := os.Stat(scriptPath); err == nil {
		return true
	}

	return false
}

func (m *Mob) GetScript() string {

	scriptPath := m.GetScriptPath()
	// Load the script into a string
	if _, err := os.Stat(scriptPath); err == nil {
		if bytes, err := os.ReadFile(scriptPath); err == nil {
			return string(bytes)
		}
	}

	return ``
}

func (m *Mob) GetScriptPath() string {
	// Load any script for the room

	mobFilePath := m.Filename()

	newExt := `.js`
	if m.ScriptTag != `` {
		newExt = fmt.Sprintf(`-%s.js`, m.ScriptTag)
	}

	scriptFilePath := `scripts/` + strings.Replace(mobFilePath, `.yaml`, newExt, 1)
	fullScriptPath := strings.Replace(configs.GetFilePathsConfig().DataFiles.String()+`/mobs/`+m.Filepath(),
		mobFilePath,
		scriptFilePath,
		1)

	//mudlog.Info("SCRIPT PATH", "path", util.FilePath(fullScriptPath))
	return util.FilePath(fullScriptPath)
}

func ReduceHostility() {

	for groupName, group := range mobsHatePlayers {
		for userId, rounds := range group {
			rounds--
			if rounds < 1 {
				delete(mobsHatePlayers[groupName], userId)
			} else {
				mobsHatePlayers[groupName][userId] = rounds
			}
		}
		if len(mobsHatePlayers[groupName]) < 1 {
			delete(mobsHatePlayers, groupName)
		}
	}
}

func IsHostile(groupName string, userId int) bool {

	if _, ok := mobsHatePlayers[groupName]; !ok {
		return false
	}

	if _, ok := mobsHatePlayers[groupName][userId]; !ok {
		return false
	}

	return true
}

func MakeHostile(groupName string, userId int, rounds int) {

	if _, ok := mobsHatePlayers[groupName]; !ok {
		mobsHatePlayers[groupName] = make(map[int]int)
		mobsHatePlayers[groupName][userId] = rounds
		return
	}
	if mobsHatePlayers[groupName][userId] < rounds {
		mobsHatePlayers[groupName][userId] = rounds
	}
}

func ZoneNameSanitize(zone string) string {
	if zone == "" {
		return ""
	}
	// Convert spaces to underscores
	zone = strings.ReplaceAll(zone, " ", "_")
	// Lowercase it all, and add a slash at the end
	return strings.ToLower(zone)
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	tmpMobs, err := fileloader.LoadAllFlatFiles[int, *Mob](configs.GetFilePathsConfig().DataFiles.String() + `/mobs`)
	if err != nil {
		panic(err)
	}

	mobs = tmpMobs

	clear(mobNameCache)

	for _, mob := range mobs {
		mob.Character.CacheDescription()
		allMobNames = append(allMobNames, mob.Character.Name)
		// Keep track of all original names associated with a given mobId
		mobNameCache[mob.MobId] = mob.Character.Name
	}

	mudlog.Info("mobs.LoadDataFiles()", "loadedCount", len(mobs), "Time Taken", time.Since(start))

}
