package scripting

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/dop251/goja"
	"github.com/mattn/go-runewidth"
)

func setRoomFunctions(vm *goja.Runtime) {
	vm.Set(`GetRoom`, GetRoom)
	vm.Set(`GetMap`, GetMap)
	vm.Set(`CreateInstancesFromRoomIds`, CreateInstancesFromRoomIds)
	vm.Set(`CreateInstancesFromZone`, CreateInstancesFromZone)
}

type ScriptRoom struct {
	roomId     int
	roomRecord *rooms.Room
}

func (r ScriptRoom) RoomId() int {
	return r.roomId
}

func (r ScriptRoom) RoomIdSource() int {
	return rooms.GetOriginalRoom(r.roomId)
}

func (r ScriptRoom) SetTempData(key string, value any) {
	r.roomRecord.SetTempData(key, value)
}

func (r ScriptRoom) GetTempData(key string) any {
	return r.roomRecord.GetTempData(key)
}

func (r ScriptRoom) SetPermData(key string, value any) {
	r.roomRecord.SetLongTermData(key, value)
}

func (r ScriptRoom) GetPermData(key string) any {
	return r.roomRecord.GetLongTermData(key)
}

func (r ScriptRoom) GetItems() []ScriptItem {
	itms := make([]ScriptItem, 0, 5)
	for _, item := range r.roomRecord.GetAllFloorItems(false) {
		itms = append(itms, newScriptItem(item))
	}
	return itms
}

func (r ScriptRoom) DestroyItem(itm ScriptItem) {
	r.roomRecord.RemoveItem(*itm.itemRecord, false)
}

func (r ScriptRoom) SpawnItem(itemId int, inStash bool) {
	i := items.New(itemId)
	if i.ItemId != 0 {
		r.roomRecord.AddItem(i, inStash)
	}
}

// Optionally can provide a MobId to look for
func (r ScriptRoom) GetMobs(mobId ...int) []*ScriptActor {
	actorList := []*ScriptActor{}

	targetMobId := 0
	if len(mobId) > 0 {
		targetMobId = mobId[0]
	}

	for _, mobInstanceId := range r.roomRecord.GetMobs() {
		a := GetActor(0, mobInstanceId)
		if a == nil {
			continue
		}
		if targetMobId == 0 || a.MobTypeId() == targetMobId {
			actorList = append(actorList, a)
		}
	}
	return actorList
}

// Get the first mob of the MobId type provided.
func (r ScriptRoom) GetMob(mobId int, createIfMissing ...bool) *ScriptActor {

	for _, mobInstanceId := range r.roomRecord.GetMobs() {
		a := GetActor(0, mobInstanceId)
		if a == nil {
			continue
		}

		if a.MobTypeId() == mobId {
			return a
		}
	}

	if len(createIfMissing) > 0 && createIfMissing[0] {
		return r.SpawnMob(mobId)
	}

	return nil
}

func (r ScriptRoom) GetPlayers() []*ScriptActor {
	actorList := []*ScriptActor{}
	for _, userId := range r.roomRecord.GetPlayers() {
		a := GetActor(userId, 0)
		if a == nil {
			continue
		}
		actorList = append(actorList, a)
	}
	return actorList
}

func (r ScriptRoom) GetAllActors() []*ScriptActor {
	actorList := []*ScriptActor{}
	for _, mobInstanceId := range r.roomRecord.GetMobs() {
		actorList = append(actorList, GetActor(0, mobInstanceId))
	}
	for _, userId := range r.roomRecord.GetPlayers() {
		actorList = append(actorList, GetActor(userId, 0))
	}
	return actorList
}

func (r ScriptRoom) GetContainers() []string {
	keys := []string{}
	for key, _ := range r.roomRecord.Containers {
		keys = append(keys, key)
	}
	return keys
}

func (r ScriptRoom) GetExits() []map[string]any {

	exits := []map[string]any{}

	seed := string(configs.GetServerConfig().Seed)
	for exitName, exitInfo := range r.roomRecord.Exits {

		exitMap := map[string]any{
			"Name":      exitName,
			"RoomId":    exitInfo.RoomId,
			"Secret":    exitInfo.Secret,
			"Lock":      false,
			"temporary": false,
		}

		if exitInfo.HasLock() {
			lockId := fmt.Sprintf(`%d-%s`, r.roomId, exitName)
			exitMap["Lock"] = map[string]any{
				"LockId":     lockId,
				"Difficulty": exitInfo.Lock.Difficulty,
				"Sequence":   util.GetLockSequence(lockId, int(exitInfo.Lock.Difficulty), seed),
			}
		} else {
			exitMap["Lock"] = nil
		}

		exits = append(exits, exitMap)
	}

	for _, exitInfo := range r.roomRecord.ExitsTemp {
		exitMap := map[string]any{
			"Name":      exitInfo.Title,
			"RoomId":    exitInfo.RoomId,
			"Secret":    false,
			"Lock":      false,
			"temporary": true,
		}
		exits = append(exits, exitMap)
	}

	return exits
}

func (r ScriptRoom) IsLocked(exitName string) bool {

	if exitInfo, ok := r.roomRecord.GetExitInfo(exitName); ok {
		if !exitInfo.HasLock() {
			return false
		}
		return exitInfo.Lock.IsLocked()
	}

	return false
}

func (r ScriptRoom) SetLocked(exitName string, lockIt bool) {

	if exitInfo, ok := r.roomRecord.GetExitInfo(exitName); ok {

		if exitInfo.HasLock() {
			r.roomRecord.SetExitLock(exitName, lockIt)
		}

	}
}

// Returns a list of userIds found to have the questId
// if userIdParty is specified, will only check users in the party of the user.
func (r ScriptRoom) HasQuest(questId string, partyUserId ...int) []int {

	hasQuestUsers := []int{}

	// Only check the user and their party?
	if len(partyUserId) > 0 && partyUserId[0] > 0 {

		if party := parties.Get(partyUserId[0]); party != nil {
			for _, userId := range party.GetMembers() {
				if user := users.GetByUserId(userId); user != nil {
					if user.Character.HasQuest(questId) {
						hasQuestUsers = append(hasQuestUsers, userId)
					}
				}
			}

			return hasQuestUsers
		}

		// No party, so just check the user
		if user := users.GetByUserId(partyUserId[0]); user != nil {
			if user.Character.HasQuest(questId) {
				hasQuestUsers = append(hasQuestUsers, user.UserId)
			}
		}

		return hasQuestUsers
	}

	// Just check all players
	for _, userId := range r.roomRecord.GetPlayers() {
		if user := users.GetByUserId(userId); user != nil {
			if user.Character.HasQuest(questId) {
				hasQuestUsers = append(hasQuestUsers, userId)
			}
		}
	}

	return hasQuestUsers
}

// Returns a list of userIds found to NOT have the questId
// if userIdParty is specified, will only check users in the party of the user.
func (r ScriptRoom) MissingQuest(questId string, partyUserId ...int) []int {

	missingQuestUsers := []int{}

	// Only check the user and their party?
	if len(partyUserId) > 0 && partyUserId[0] > 0 {

		if party := parties.Get(partyUserId[0]); party != nil {
			// Check all party members
			for _, userId := range party.GetMembers() {
				if user := users.GetByUserId(userId); user != nil {
					if !user.Character.HasQuest(questId) {
						missingQuestUsers = append(missingQuestUsers, userId)
					}
				}
			}

			return missingQuestUsers
		}

		// No party, so just check the user
		if user := users.GetByUserId(partyUserId[0]); user != nil {
			if !user.Character.HasQuest(questId) {
				missingQuestUsers = append(missingQuestUsers, user.UserId)
			}
		}
		return missingQuestUsers

	}

	// Just check all players
	for _, userId := range r.roomRecord.GetPlayers() {
		if user := users.GetByUserId(userId); user != nil {
			if !user.Character.HasQuest(questId) {
				missingQuestUsers = append(missingQuestUsers, userId)
			}
		}
	}

	return missingQuestUsers
}

func (r ScriptRoom) SpawnMob(mobId int) *ScriptActor {

	if mob := mobs.NewMobById(mobs.MobId(mobId), r.roomId); mob != nil {

		r.roomRecord.AddMob(mob.InstanceId)

		return GetMob(mob.InstanceId)
	}

	return nil
}

// SpawnMobScaled spawns a mob and overrides its stat pool with the given
// value. Use CalcCompanionStatPool (characters package) to compute the
// scaled value before calling this. Returns nil if the mob could not be
// created.
func (r ScriptRoom) SpawnMobScaled(mobId int, statPool int) *ScriptActor {
	if mob := mobs.NewMobById(mobs.MobId(mobId), r.roomId, statPool); mob != nil {
		r.roomRecord.AddMob(mob.InstanceId)
		return GetMob(mob.InstanceId)
	}
	return nil
}

func (r ScriptRoom) SendText(msg string, excludeIds ...int) {

	msg = roomTextWrap.Wrap(msg)

	r.roomRecord.SendText(msg, excludeIds...)
}

func (r ScriptRoom) SendTextToExits(msg string, isQuiet bool, excludeUserIds ...int) {

	msg = roomTextWrap.Wrap(msg)

	r.roomRecord.SendTextToExits(msg, isQuiet, excludeUserIds...)
}

func (r ScriptRoom) RepeatSpawnItem(itemId int, roundFrequency int, containerName ...string) bool {
	return r.roomRecord.RepeatSpawnItem(itemId, roundFrequency, containerName...)
}

func (r ScriptRoom) AddTemporaryExit(exitNameSimple string, exitNameFancy string, exitRoomId int, expiresTimeString string) bool {

	if exitNameFancy[0:1] == `:` {
		exitNameFancy = colorpatterns.ApplyColorPattern(exitNameSimple, exitNameFancy[1:])
	}

	tmpExit := exit.TemporaryRoomExit{
		RoomId:  exitRoomId,
		Title:   exitNameFancy,
		UserId:  0,
		Expires: expiresTimeString,
	}

	// Spawn a portal in the room that leads to the portal location
	return r.roomRecord.AddTemporaryExit(exitNameSimple, tmpExit)
}

func (r ScriptRoom) RemoveTemporaryExit(exitNameSimple string, exitNameFancy string, exitRoomId int) bool {
	tmpExit := exit.TemporaryRoomExit{
		RoomId: exitRoomId,
		Title:  exitNameFancy,
		UserId: 0,
	}

	// Spawn a portal in the room that leads to the portal location
	return r.roomRecord.RemoveTemporaryExit(tmpExit)
}

func (r ScriptRoom) HasMutator(mutName string) bool {
	return r.roomRecord.Mutators.Has(mutName)
}

func (r ScriptRoom) AddMutator(mutName string) {
	r.roomRecord.Mutators.Add(mutName)
}

func (r ScriptRoom) RemoveMutator(mutName string) {
	r.roomRecord.Mutators.Remove(mutName)

	if zoneConfig := rooms.GetZoneConfig(r.roomRecord.Zone); zoneConfig != nil {
		zoneConfig.Mutators.Remove(mutName)
	}
}

func (r ScriptRoom) IsEphemeral() bool {
	return rooms.IsEphemeralRoomId(r.roomRecord.RoomId)
}

// ////////////////////////////////////////////////////////
//
// # These functions get exported to the scripting engine
//
// ////////////////////////////////////////////////////////
func CreateInstancesFromRoomIds(roomList []int) map[int]int {
	ret, _ := rooms.CreateEphemeralRoomIds(roomList...)
	return ret
}

func CreateInstancesFromZone(zoneName string) map[int]int {
	ret, _ := rooms.CreateEphemeralZone(zoneName)
	return ret
}

func GetRoom(roomId int) *ScriptRoom {
	if room := rooms.LoadRoom(roomId); room != nil {
		return &ScriptRoom{roomId, room}
	}
	return nil
}

// mapRoomId    - Room the map is centered on
// mapSize      - wide or normal
// mapHeight	- Height of the map
// mapWidth     - Width of the map
// mapName 		- The title of the map
// showSecrets  - Include secret exits/rooms?
// mapMarkers   - A list of strings representing custom map markers:
//
//	[roomId],[symbol],[legend text]
//	1,×,Here
func GetMap(mapRoomId int, zoomLevel int, mapHeight int, mapWidth int, mapName string, showSecrets bool, mapMarkers ...string) string {
	// mapRoomId    - Room the map is centered on
	// mapSize      - wide or normal
	// mapHeight	- Height of the map
	// mapWidth     - Width of the map
	// mapName 		- The title of the map
	// showSecrets  - Include secret exits/rooms?
	// mapMarkers   - A list of strings representing custom map markers:
	//                [roomId],[symbol],[legend text]
	//                1,×,Here

	room := rooms.LoadRoom(mapRoomId)
	if room == nil {
		return ""
	}

	rMapper := mapper.GetMapper(room.RoomId)
	if rMapper == nil {
		mudlog.Error("Map", "error", "Could not find mapper for roomId:"+strconv.Itoa(room.RoomId))
		return "Could not find mapper for roomId:" + strconv.Itoa(room.RoomId)
	}

	c := mapper.Config{
		ZoomLevel: zoomLevel,
		Width:     mapWidth,
		Height:    mapHeight,
	}

	if showSecrets {
		c.UserId = -1
	}

	if len(mapMarkers) > 0 {
		for _, overrideString := range mapMarkers {
			parts := strings.Split(overrideString, `,`)
			if len(parts) == 3 {
				roomId, _ := strconv.Atoi(parts[0])
				symbol := parts[1]
				legend := parts[2]

				if roomId > 0 && len(symbol) > 0 && len(legend) > 0 {
					c.OverrideSymbol(roomId, []rune(symbol)[0], legend)
				}
			}
		}
	}

	mapOutput := rMapper.GetLimitedMap(mapRoomId, c)

	legend := mapOutput.GetLegend(keywords.GetAllLegendAliases(room.Zone))

	displayLines := []string{}
	for i, line := range mapOutput.Render {
		displayLines = append(displayLines, string(line))
		for sym, txtLegend := range legend {
			txtLc := strings.ToLower(txtLegend)
			displayLines[i] = strings.Replace(displayLines[i], string(sym), fmt.Sprintf(`<ansi fg="map-room"><ansi fg="map-%s" bg="mapbg-%s">%c</ansi></ansi>`, txtLc, txtLc, sym), -1)
		}
	}

	mapData := map[string]any{
		"Title":        mapName,
		"DisplayLines": displayLines,
		"Height":       len(displayLines),
		"Width":        runewidth.StringWidth(string(displayLines[0])),
		"Legend":       legend,
		"LegendWidth":  runewidth.StringWidth(string(displayLines[0])),
		"LeftBorder": map[string]any{
			"Top":    ".-=~=-.",
			"Mid":    []string{"( _ __)", "(__  _)"},
			"Bottom": "`-._.-'",
		},
		"MidBorder": map[string]any{
			"Top":    "-._.-=",
			"Bottom": "-._.-=",
		},
		"RightBorder": map[string]any{
			"Top":    ".-=~=-.",
			"Mid":    []string{"( _ __)", "(__  _)"},
			"Bottom": "`-._.-'",
		},
	}

	mapTxt, err := templates.Process("maps/map", mapData)
	if err != nil {
		mudlog.Error("Map", "error", err.Error())
		return err.Error()
	}

	return mapTxt
}
