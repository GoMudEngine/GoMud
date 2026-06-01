package rooms

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	pkgerrors "github.com/pkg/errors"
)

// companionTransport is set at startup by main.go to wire the
// hooks.TransportCompanions helper into MoveToRoom without creating
// an import cycle (rooms → hooks would cycle).
var companionTransport func(userId, oldRoomId, newRoomId int)

// SetCompanionTransport registers the companion-follow hook that fires
// after every user move. Called from main.go at startup.
func SetCompanionTransport(fn func(userId, oldRoomId, newRoomId int)) {
	companionTransport = fn
}

var (
	roomManager = &RoomManager{
		rooms:             make(map[int]*Room),
		zones:             make(map[string]*ZoneConfig),
		roomsWithUsers:    make(map[int]int),
		roomsWithMobs:     make(map[int]int),
		roomIdToFileCache: make(map[int]string),
	}
)

const (
	StartRoomIdAlias = 0
)

type RoomManager struct {
	rooms             map[int]*Room
	zones             map[string]*ZoneConfig // a map of zone name to room id
	roomsWithUsers    map[int]int            // key is roomId to # players
	roomsWithMobs     map[int]int            // key is roomId to # mobs
	roomIdToFileCache map[int]string         // key is room id, value is the file path
}

// Deletes any knowledge of a room in memory.
// Loading this room after the fact will trigger full re-loading and caching of room data.
func ClearRoomCache(roomId int) error {

	room := roomManager.rooms[roomId]
	if room == nil {
		return fmt.Errorf(`room %d not found in cache`, roomId)
	}

	if zoneData, ok := roomManager.zones[room.Zone]; ok {
		delete(zoneData.RoomIds, roomId)
		roomManager.zones[room.Zone] = zoneData
	}

	delete(roomManager.rooms, roomId)
	delete(roomManager.roomsWithUsers, roomId)
	delete(roomManager.roomsWithMobs, roomId)
	delete(roomManager.roomIdToFileCache, roomId)

	return nil
}

func (r *RoomManager) GetFilePath(roomId int) string {

	if cachedPath, ok := roomManager.roomIdToFileCache[roomId]; ok {
		return cachedPath
	}

	filename := searchForRoomFile(roomId)

	if filename == `` {
		return filename
	}

	roomManager.roomIdToFileCache[roomId] = filename

	return filename
}

// Find a file for a roomId and cache the file location.
func searchForRoomFile(roomId int) string {

	searchFileName := filepath.FromSlash(fmt.Sprintf(`/%d.yaml`, roomId))

	walkPath := filepath.FromSlash(configs.GetFilePathsConfig().DataFiles.String() + `/rooms`)

	foundFilePath := ``
	filepath.Walk(walkPath, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if strings.HasSuffix(path, searchFileName) {
			foundFilePath = path
			return errors.New(`found`)
		}

		return nil
	})

	return strings.TrimPrefix(foundFilePath, walkPath)
}

type ZoneInfo struct {
	RootRoomId      int
	DefaultBiome    string // city, swamp etc. see biomes.go
	HasZoneMutators bool   // does it have any zone mutators assigned?
	RoomIds         map[int]struct{}
}

func GetNextRoomId() int {
	return int(configs.GetServerConfig().NextRoomId)
}

func SetNextRoomId(nextRoomId int) {
	configs.SetVal(`Server.NextRoomId`, strconv.Itoa(nextRoomId))
}

func GetAllRoomIds() []int {

	var roomIds []int = make([]int, len(roomManager.roomIdToFileCache))
	i := 0
	for roomId, _ := range roomManager.roomIdToFileCache {
		roomIds[i] = roomId
		i++
	}

	return roomIds
}

func GetZonesWithMutators() ([]string, []int) {

	zNames := []string{}
	rootRoomIds := []int{}

	for zName, zInfo := range roomManager.zones {
		if len(zInfo.Mutators) > 0 {
			zNames = append(zNames, zName)
			rootRoomIds = append(rootRoomIds, zInfo.RoomId)
		}
	}
	return zNames, rootRoomIds
}

func RoomMaintenance() []int {
	start := time.Now()
	defer func() {
		util.TrackTime(`RoomMaintenance()`, time.Since(start).Seconds())
	}()

	c := configs.GetMemoryConfig()

	roundCount := util.GetRoundCount()
	// Get the current round count
	unloadRoundThreshold := roundCount - uint64(c.RoomUnloadRounds)
	unloadRooms := make([]*Room, 0)

	allowedUnloadCt := len(roomManager.rooms) - int(c.RoomUnloadThreshold)
	if allowedUnloadCt < 0 {
		allowedUnloadCt = 0
	}

	for _, room := range roomManager.rooms {

		room.PruneVisitors()

		// Notify that room that something happened to the sign?
		if prunedSigns := room.PruneSigns(); len(prunedSigns) > 0 {

			if roomPlayers := room.GetPlayers(); len(roomPlayers) > 0 {
				for _, userId := range roomPlayers {
					for _, sign := range prunedSigns {
						if sign.VisibleUserId == 0 {
							if u := users.GetByUserId(userId); u != nil {
								u.SendText(messaging.CategoryRoomDescription, "A sign crumbles to dust.\n")
							}
						} else if sign.VisibleUserId == userId {
							if u := users.GetByUserId(userId); u != nil {
								u.SendText(messaging.CategorySystem, "The rune you had enscribed here has faded away.\n")
							}
						}
					}
				}
			}
		}

		// Notify the room that the temp exits disappeared?
		if prunedExits := room.PruneTemporaryExits(); len(prunedExits) > 0 {

			if roomPlayers := room.GetPlayers(); len(roomPlayers) > 0 {
				for _, exit := range prunedExits {
					for _, userId := range roomPlayers {
						if u := users.GetByUserId(userId); u != nil {
							u.SendText(messaging.CategoryRoomExit, fmt.Sprintf("The %s vanishes.\n", exit.Title))
						}
					}
				}
			}
		}

		// Consider unloading rooms from memory?
		if allowedUnloadCt > 0 && !room.IsEphemeral() {
			if room.lastVisited < unloadRoundThreshold && !roomHasEssentialMob(room) {
				unloadRooms = append(unloadRooms, room)
				allowedUnloadCt--
			}
		}

	}

	removedRoomIds := make([]int, len(unloadRooms))
	if len(unloadRooms) > 0 {
		for i, room := range unloadRooms {
			removeRoomFromMemory(room)
			removedRoomIds[i] = room.RoomId
		}
	}

	return removedRoomIds
}

func GetAllZoneNames() []string {

	var zoneNames []string = make([]string, len(roomManager.zones))
	i := 0
	for zoneName, _ := range roomManager.zones {
		zoneNames[i] = zoneName
		i++
	}

	return zoneNames
}

func GetAllZoneRoomsIds(zoneName string) []int {

	if zoneInfo, ok := roomManager.zones[zoneName]; ok {
		result := make([]int, len(zoneInfo.RoomIds))
		idx := 0
		for roomId, _ := range zoneInfo.RoomIds {
			result[idx] = roomId
			idx++
		}
		return result
	}

	return []int{}
}

func MoveToRoom(userId int, toRoomId int, isSpawn ...bool) error {

	user := users.GetByUserId(userId)

	currentRoom := LoadRoom(user.Character.RoomId)

	cfg := configs.GetSpecialRoomsConfig()

	// If they are being moved to the death recovery room
	// Put them in their own instance of it.
	deathRecoveryRoomId := int(cfg.DeathRecoveryRoom)
	if toRoomId == deathRecoveryRoomId {
		if newRooms, err := CreateEphemeralRoomIds(deathRecoveryRoomId); err == nil {
			toRoomId = newRooms[deathRecoveryRoomId]
		}
	}

	if toRoomId == StartRoomIdAlias {

		// If "StartRoom" is set for MiscData on the char, use that.
		if charStartRoomId := user.Character.GetMiscData(`StartRoom`); charStartRoomId != nil {
			if rId, ok := charStartRoomId.(int); ok {
				toRoomId = rId
			}
		}

		// If still StartRoomIdAlias, use config value
		if toRoomId == StartRoomIdAlias && cfg.StartRoom != 0 {
			toRoomId = int(cfg.StartRoom)
		}

		// If toRomoId is zero after all this, default to 1
		if toRoomId == 0 {
			toRoomId = 1
		}
	}

	newRoom := LoadRoom(toRoomId)
	if newRoom == nil {
		return fmt.Errorf(`room %d not found`, toRoomId)
	}

	// Instance access control: block unauthorized entry to instanced zones
	if IsEphemeralRoomId(toRoomId) {
		if inst := instanceRegistry.FindByRoomId(toRoomId); inst != nil {
			if !inst.IsAuthorized(userId) {
				if user != nil {
					user.SendText(messaging.CategoryError, `<ansi fg="red">The portal's energy pushes you back. It wasn't opened for you.</ansi>`)
				}
				return fmt.Errorf("instance access denied")
			}
		}
	}

	// r.prepare locks, so do it before the upcoming lock
	if len(newRoom.players) == 0 {
		newRoom.Prepare(true)
	}

	fromRoomId := user.Character.RoomId
	if currentRoom != nil {
		currentRoom.MarkVisited(userId, VisitorUser, 1)
		if len, _ := currentRoom.RemovePlayer(userId); len < 1 {
			delete(roomManager.roomsWithUsers, currentRoom.RoomId)
		}
	}

	newRoom.MarkVisited(userId, VisitorUser)

	//
	// Apply any mutators from the zone or room
	// This will only add mutators that the player
	// doesn't already have.
	//
	for mut := range newRoom.ActiveMutators {
		spec := mut.GetSpec()
		if len(spec.PlayerBuffIds) == 0 {
			continue
		}
		for _, buffId := range spec.PlayerBuffIds {
			if !user.Character.HasBuff(buffId) {
				user.AddBuff(buffId, `area`)
			}
		}
	}
	//
	// Done adding mutator buffs
	//

	user.Character.RoomId = newRoom.RoomId
	user.Character.Zone = newRoom.Zone
	user.Character.RememberRoom(newRoom.RoomId) // Mark this room as remembered.

	playerCt := newRoom.AddPlayer(userId)
	roomManager.roomsWithUsers[newRoom.RoomId] = playerCt

	events.AddToQueue(events.RoomChange{
		UserId:     userId,
		FromRoomId: fromRoomId,
		ToRoomId:   newRoom.RoomId,
		Unseen:     user.Character.IsHidden(),
	})

	// Companion follow: move every live companion of this user into the
	// new room. Aborts in-progress casts. Registered callback — may be
	// nil during early startup or in tests that don't wire it.
	if companionTransport != nil && fromRoomId != newRoom.RoomId {
		companionTransport(userId, fromRoomId, newRoom.RoomId)
	}

	return nil
}

// skipRecentlyVisited means ignore rooms with recent visitors
// minimumItemCt is the minimum items in the room to care about it
func GetRoomWithMostItems(skipRecentlyVisited bool, minimumItemCt int, minimumGoldCt int) (roomId int, itemCt int) {

	lgConfig := configs.GetLootGoblinConfig()
	goblinZone := ``
	if goblinRoomId := int(lgConfig.RoomId); goblinRoomId != 0 {
		if goblinRoom := LoadRoom(int(lgConfig.RoomId)); goblinRoom != nil {
			goblinZone = goblinRoom.Zone
		}
	}

	topItemRoomId, topItemCt := 0, 0
	topGoldRoomId, topGoldCt := 0, 0

	for cRoomId, cRoom := range roomManager.rooms {
		// Don't include goblin trash zone items
		if cRoom.Zone == goblinZone {
			continue
		}

		iCt := len(cRoom.Items)

		if iCt < minimumItemCt && cRoom.Gold < minimumGoldCt {
			continue
		}

		if iCt > topItemCt {
			if skipRecentlyVisited && cRoom.HasRecentVisitors() {
				continue
			}
			topItemRoomId = cRoomId
			topItemCt = iCt
		}

		if cRoom.Gold > topGoldCt {
			if skipRecentlyVisited && cRoom.HasRecentVisitors() {
				continue
			}
			topGoldRoomId = cRoomId
			topGoldCt = cRoom.Gold
		}
	}

	if topItemRoomId == 0 && topGoldCt > 0 {
		return topGoldRoomId, topGoldCt
	}

	return topItemRoomId, topItemCt
}

func GetRoomsWithPlayers() []int {

	deleteKeys := []int{}
	roomsWithPlayers := []int{}

	for roomId, _ := range roomManager.roomsWithUsers {
		roomsWithPlayers = append(roomsWithPlayers, roomId)
	}

	for i := len(roomsWithPlayers) - 1; i >= 0; i-- {
		roomId := roomsWithPlayers[i]
		if r := LoadRoom(roomId); r != nil {
			if len(r.players) < 1 {
				roomsWithPlayers = append(roomsWithPlayers[:i], roomsWithPlayers[i+1:]...)
				deleteKeys = append(deleteKeys, roomId)
				continue
			}
		}
	}

	if len(deleteKeys) > 0 {

		for _, roomId := range deleteKeys {
			delete(roomManager.roomsWithUsers, roomId)
		}

	}

	return roomsWithPlayers
}

func GetRoomsWithMobs() []int {

	var roomsWithMobs []int = make([]int, len(roomManager.roomsWithMobs))
	i := 0
	for roomId, _ := range roomManager.roomsWithMobs {
		roomsWithMobs[i] = roomId
		i++
	}

	return roomsWithMobs
}

// Saves a room to disk and unloads it from memory
func removeRoomFromMemory(r *Room) {

	room, ok := roomManager.rooms[r.RoomId]

	if !ok {
		return
	}

	if len(room.players) > 0 {
		return
	}

	// Don't unload rooms that contain essential living-economy mobs (foragers,
	// caravan crew). Their BTree state lives in the in-memory mob instance —
	// destroying them resets the state machine and breaks the cycle they're
	// supposed to be running. Memory cost is small: typically <20 rooms pinned
	// across the world at any moment.
	if roomHasEssentialMob(room) {
		return
	}

	keptMobs := room.mobs[:0:0]
	for _, mobInstanceId := range room.mobs {
		// Only destroy mobs actually in this room. A scheduled/patrolling mob
		// may be listed here but have already moved on (its Character.RoomId
		// points elsewhere); its schedule/patrol executor owns it in its
		// current room, so destroying it here would orphan-kill it. Drop the
		// stale listing without destroying the instance.
		if m := mobs.GetInstance(mobInstanceId); m != nil && m.Character.RoomId != room.RoomId {
			continue
		}
		mobs.DestroyInstance(mobInstanceId)
		keptMobs = append(keptMobs, mobInstanceId)
	}
	room.mobs = keptMobs

	for _, spawnDetails := range room.SpawnInfo {
		if spawnDetails.InstanceId > 0 {

			if m := mobs.GetInstance(spawnDetails.InstanceId); m != nil {
				if m.Character.RoomId == room.RoomId {
					mobs.DestroyInstance(spawnDetails.InstanceId)
				}
			}

		}
	}

	SaveRoomInstance(*room)

	delete(roomManager.rooms, r.RoomId)
}

func getRoomFromMemory(roomId int) *Room {
	return roomManager.rooms[roomId]
}

// Loads a room from disk and stores in memory
func addRoomToMemory(room *Room, forceOverWrite ...bool) error {

	if len(forceOverWrite) > 0 && forceOverWrite[0] {
		ClearRoomCache(room.RoomId)
	}

	if _, ok := roomManager.rooms[room.RoomId]; ok {
		return fmt.Errorf(`room %d is already stored in memory`, room.RoomId)
	}

	// Automatically set the last visitor to now (reset the unload timer)
	room.lastVisited = util.GetRoundCount()

	// Save to room cache lookup
	roomManager.rooms[room.RoomId] = room

	// Save filepath to cache
	if _, ok := roomManager.roomIdToFileCache[room.RoomId]; !ok {
		roomManager.roomIdToFileCache[room.RoomId] = room.Filepath()
	}

	// Track whatever the last room id created is so we know what to number the next one.
	if room.RoomId < ephemeralRoomIdMinimum && room.RoomId >= GetNextRoomId() {
		SetNextRoomId(room.RoomId + 1)
	}

	//
	zoneInfo, ok := roomManager.zones[room.Zone]
	if !ok {
		zoneInfo = &ZoneConfig{
			Name:    room.Zone,
			RoomId:  room.RoomId,
			RoomIds: make(map[int]struct{}),
		}
	}

	// Populate the room present lookup in the zone info
	zoneInfo.RoomIds[room.RoomId] = struct{}{}

	roomManager.zones[room.Zone] = zoneInfo

	return nil
}

func GetZoneRoot(zone string) (int, error) {

	if zoneInfo, ok := roomManager.zones[zone]; ok {
		return zoneInfo.RoomId, nil
	}

	return 0, fmt.Errorf("zone %s does not exist.", zone)
}

func GetZoneConfig(zone string) *ZoneConfig {
	return roomManager.zones[zone]
}

func IsRoomLoaded(roomId int) bool {
	_, ok := roomManager.rooms[roomId]
	return ok
}

func ZoneStats(zone string) (rootRoomId int, totalRooms int, err error) {

	if zoneInfo, ok := roomManager.zones[zone]; ok {
		return zoneInfo.RoomId, len(zoneInfo.RoomIds), nil
	}

	return 0, 0, fmt.Errorf("zone %s does not exist.", zone)
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

func ZoneToFolder(zone string) string {
	zone = ZoneNameSanitize(zone)
	// Lowercase it all, and add a slash at the end
	return zone + "/"
}

func ValidateZoneName(zone string) error {
	if zone == "" {
		return nil
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9_ ]+$`).MatchString(zone) {
		return errors.New("allowable characters in zone name are letters, numbers, spaces, and underscores")
	}

	return nil
}

func FindZoneName(zone string) string {

	if _, ok := roomManager.zones[zone]; ok {
		return zone
	}

	for zoneName, _ := range roomManager.zones {
		if strings.Contains(strings.ToLower(zoneName), strings.ToLower(zone)) {
			return zoneName
		}
	}

	return ""
}

func GetZoneBiome(zone string) string {

	if z, ok := roomManager.zones[zone]; ok {
		return z.DefaultBiome
	}

	return ``
}

func MoveToZone(roomId int, newZoneName string) error {

	tplRoom := LoadRoomTemplate(roomId)

	if tplRoom == nil {
		return errors.New("room doesn't exist")
	}

	oldZoneName := tplRoom.Zone
	oldZoneInfo, ok := roomManager.zones[oldZoneName]
	if !ok {
		return errors.New("old zone doesn't exist")
	}
	oldFilePath := fmt.Sprintf("%s/rooms/%s", configs.GetFilePathsConfig().DataFiles.String(), tplRoom.Filepath())
	oldInstanceFilePath := fmt.Sprintf("%s/rooms.instances/%s", configs.GetFilePathsConfig().DataFiles.String(), tplRoom.Filepath())

	newZoneInfo, ok := roomManager.zones[newZoneName]
	if !ok {
		return errors.New("new zone doesn't exist")
	}

	if oldZoneInfo.RoomId == roomId {
		return errors.New("can't move the root room of a zone")
	}

	tplRoom.Zone = newZoneName
	newFilePath := fmt.Sprintf("%s/rooms/%s", configs.GetFilePathsConfig().DataFiles.String(), tplRoom.Filepath())
	newInstanceFilePath := fmt.Sprintf("%s/rooms.instances/%s", configs.GetFilePathsConfig().DataFiles.String(), tplRoom.Filepath())

	if err := os.Rename(oldFilePath, newFilePath); err != nil {
		return err
	}

	os.Rename(oldInstanceFilePath, newInstanceFilePath)

	delete(oldZoneInfo.RoomIds, roomId)
	roomManager.zones[oldZoneName] = oldZoneInfo

	newZoneInfo.RoomIds[roomId] = struct{}{}
	roomManager.zones[newZoneName] = newZoneInfo

	SaveRoomTemplate(*tplRoom)

	return nil
}

// #build zone The Arctic
// Build a zone, popualtes with an empty boring room
func CreateZone(zoneName string) (roomId int, err error) {

	zoneName = strings.TrimSpace(zoneName)

	if len(zoneName) < 2 {
		return 0, errors.New("zone name must be at least 2 characters")
	}

	if zoneInfo, ok := roomManager.zones[zoneName]; ok {
		return zoneInfo.RoomId, errors.New("zone already exists")
	}

	zoneInfo := NewZoneConfig(zoneName)

	zoneFolder := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), "/", "rooms", "/", ZoneToFolder(zoneName))
	if err := os.Mkdir(zoneFolder, 0755); err != nil {
		return 0, err
	}

	if err := fileloader.SaveFlatFile[*ZoneConfig](zoneFolder, zoneInfo); err != nil {
		return 0, err
	}

	roomManager.zones[zoneName] = zoneInfo

	instanceZoneFolder := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), "/", "rooms.instances", "/", ZoneToFolder(zoneName))
	if err := os.Mkdir(instanceZoneFolder, 0755); err != nil {
		return 0, err
	}

	newRoom := NewRoom(zoneName)

	if err := newRoom.Validate(); err != nil {
		return 0, err
	}

	addRoomToMemory(newRoom)

	// save to the flat file
	SaveRoomTemplate(*newRoom)

	// write room to the folder under the new ID
	return newRoom.RoomId, nil
}

// #build room north
// Build a room to a specific direction, and connect it by exit name
// You still need to visit that room and connect it the opposite way
func BuildRoom(fromRoomId int, exitName string, mapDirection ...string) (room *Room, err error) {

	exitName = strings.TrimSpace(exitName)
	exitMapDirection := exitName

	if len(mapDirection) > 0 {
		exitMapDirection = mapDirection[0]
	}

	fromRoom := LoadRoomTemplate(fromRoomId)
	if fromRoom == nil {
		return nil, fmt.Errorf(`room %d not found`, fromRoomId)
	}

	if _, ok := fromRoom.Exits[exitName]; ok {
		return nil, fmt.Errorf(`this room already has a %s exit`, exitName)
	}

	newRoom := NewRoom(fromRoom.Zone)
	if err := newRoom.Validate(); err != nil {
		return nil, fmt.Errorf("BuildRoom(%d, %s, %s): %w", fromRoomId, exitName, exitMapDirection, err)
	}

	newRoom.Title = fromRoom.Title
	newRoom.Description = fromRoom.Description
	newRoom.MapSymbol = fromRoom.MapSymbol
	newRoom.MapLegend = fromRoom.MapLegend
	newRoom.Biome = fromRoom.Biome

	if len(fromRoom.IdleMessages) > 0 {
		//newRoom.IdleMessages = fromRoom.IdleMessages
	}

	mudlog.Info("Connecting room", "fromRoom", fromRoom.RoomId, "newRoom", newRoom.RoomId, "exitName", exitName)

	// connect the old room to the new room
	newExit := exit.RoomExit{RoomId: newRoom.RoomId, Secret: false}
	if exitMapDirection != exitName {
		newExit.MapDirection = exitMapDirection
	}
	fromRoom.Exits[exitName] = newExit

	// Add the new room to memory.
	addRoomToMemory(newRoom)

	// Update the memory for the source room
	addRoomToMemory(fromRoom, true)

	SaveRoomTemplate(*fromRoom)
	SaveRoomTemplate(*newRoom)

	return newRoom, nil
}

// #build exit north 1337
// Build an exit in the current room that links to room by id
// You still need to visit that room and connect it the opposite way
func ConnectRoom(fromRoomId int, toRoomId int, exitName string, mapDirection ...string) error {

	// exitname will be "north"
	exitName = strings.TrimSpace(exitName)
	exitMapDirection := exitName
	// Return direction will be "north" or "north-x2"
	if len(mapDirection) > 0 {
		exitMapDirection = mapDirection[0]
	}

	fromRoom := LoadRoomTemplate(fromRoomId)
	if fromRoom == nil {
		return fmt.Errorf(`room %d not found`, fromRoomId)
	}

	toRoom := LoadRoomTemplate(toRoomId)
	if toRoom == nil {
		return fmt.Errorf(`room %d not found`, toRoomId)
	}

	// connect the old room to the new room
	newExit := exit.RoomExit{RoomId: toRoom.RoomId, Secret: false}
	if exitMapDirection != exitName {
		newExit.MapDirection = exitMapDirection
	}
	fromRoom.Exits[exitName] = newExit

	SaveRoomTemplate(*fromRoom)
	roomManager.rooms[fromRoom.RoomId] = fromRoom

	return nil
}

func GetRoomCount(zoneName string) int {

	zoneInfo, ok := roomManager.zones[zoneName]
	if !ok {
		return 0
	}

	return len(zoneInfo.RoomIds)
}

func LoadDataFiles() {

	if len(roomManager.zones) > 0 {
		mudlog.Info("rooms.LoadDataFiles()", "msg", "skipping reload of room files, rooms shouldn't be hot reloaded from flatfiles.")
		return
	}

	if err := loadAllRoomZones(); err != nil {
		panic(pkgerrors.Wrap(err, `filepath: rooms`))
	}

}

// roomHasEssentialMob returns true if any mob currently in the room is
// essential (drives a living-economy system such as foragers or caravan crew).
// Used by RoomMaintenance and removeRoomFromMemory to pin rooms containing
// these mobs so their in-memory BTree state is never destroyed by the
// idle-unload cycle.
func roomHasEssentialMob(r *Room) bool {
	for _, mobInstanceId := range r.mobs {
		if m := mobs.GetInstance(mobInstanceId); m != nil && m.IsEssential() {
			return true
		}
	}
	return false
}
