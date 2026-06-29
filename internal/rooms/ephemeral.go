package rooms

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const (
	ephemeralChunksLimit = 100     // The maximum number of ephemeral chunks that can be created
	ephemeralChunkSize   = 250     // The maximum quantity of ephemeral room's that can be copied/created in a given chunk.
	roomIdMin32Bit       = 1000000 // 1,000,000 - Safe for 32-bit systems (was 1,000,000,000 which overflows when multiplied by 1000)
)

var (
	ephemeralRoomIdMinimum = roomIdMin32Bit                // 1,000,000 is assuming 32 bit. the init() function may override this value to 1,000,000,000 on 64-bit systems.
	ephemeralRoomChunks    = [ephemeralChunksLimit][]int{} // map of ranges to actual rooms. If empty, slot is available.
	originalRoomIdLookups  = map[int]int{}                 // a map of ephemeralId's to their original RoomId's, for special purposes
	// errors
	errNoRoomIdsProvided   = errors.New(`no RoomId's were provided`)
	errRoomNotFound        = errors.New(`the requested RoomId wasn't found`)
	errEphemeralChunkLimit = fmt.Errorf(`the ephemeral chunk limit of %d has been reached.`, ephemeralChunksLimit)
	errEphemeralRoomLimit  = fmt.Errorf(`the ephemeral room request limit of %d is exceeded.`, ephemeralChunkSize)
	errNonUniqueRoomId     = errors.New(`a RoomId has been provided more than once. they must all be unique`)
)

func GetChunkCount() int {
	result := 0
	for i := 0; i < ephemeralChunksLimit; i++ {
		if len(ephemeralRoomChunks[i]) > 0 {
			result++
		}
	}
	return result
}

// Looks for any ephemeralRoomId's that exits for the given roomId.
// Returns a slice containing all found ephemeralIds
func FindEphemeralRoomIds(roomId int) []int {

	allEphemeralRoomIds := []int{}
	for ephemeralRoomId, originalRoomId := range originalRoomIdLookups {
		if originalRoomId == roomId {
			allEphemeralRoomIds = append(allEphemeralRoomIds, ephemeralRoomId)
		}
	}

	return allEphemeralRoomIds
}

// accepts RoomId's as arguments, and creates ephemeral copies of them, returning the new ID's of the copies.
func CreateEphemeralRoomIds(roomIds ...int) (map[int]int, error) {

	ephemeralRooms := map[int]int{}

	if len(roomIds) == 0 {
		return ephemeralRooms, errNoRoomIdsProvided
	}

	if len(roomIds) > ephemeralChunkSize {
		return ephemeralRooms, errEphemeralRoomLimit
	}

	// Make sure that all values in the roomIds slice are unique.
	roomIdReplacements := map[int]int{} // original=>ephemeral replacements
	for _, roomId := range roomIds {
		if _, ok := roomIdReplacements[roomId]; ok {
			return ephemeralRooms, errNonUniqueRoomId
		}
		roomIdReplacements[roomId] = 0
	}

	// First reserve the chunk
	chunkId := -1
	for i := 0; i < ephemeralChunksLimit; i++ {
		if len(ephemeralRoomChunks[i]) == 0 {
			chunkId = i
			break
		}
	}

	ephemeralRoomIds := []int{}
	for idx, roomId := range roomIds {
		// Load only data from the template

		if roomId == 0 {
			continue
		}

		room := LoadRoomTemplate(roomId)
		if room == nil {
			continue
		}

		room.RoomId = ephemeralRoomIdMinimum + (chunkId * ephemeralChunkSize) + idx

		// Save the original room ID in case we need it at some point
		originalRoomIdLookups[room.RoomId] = roomId

		// Temporarily track what the original room has been copied to.
		roomIdReplacements[roomId] = room.RoomId

		addRoomToMemory(room)

		ephemeralRooms[roomId] = room.RoomId
		ephemeralRoomIds = append(ephemeralRoomIds, room.RoomId)
	}

	// Replace references to original RoomId's with new Ephemeral ones
	for _, roomId := range ephemeralRoomIds {
		room := LoadRoom(roomId)
		if room == nil {
			continue
		}

		for exitName, exitInfo := range room.Exits {
			if replacementRoomId, ok := roomIdReplacements[exitInfo.RoomId]; ok {
				exitInfo.RoomId = replacementRoomId
				room.Exits[exitName] = exitInfo
			}
		}

	}

	ephemeralRoomChunks[chunkId] = ephemeralRoomIds

	mudlog.Info("CreateEphemeral...()",
		"created", len(ephemeralRoomIds),
		"chunkId", chunkId,
		"Ephemeral RoomIds", fmt.Sprintf("%d - %d", ephemeralRoomIds[0], ephemeralRoomIds[len(ephemeralRoomIds)-1]),
		"Chunks Remaining", GetChunkCount())

	return ephemeralRooms, nil
}

// accepts RoomId's as arguments, and creates ephemeral copies of them, returning the new ID's of the copies.
func CreateEphemeralZone(zoneName string) (map[int]int, error) {

	zone := roomManager.zones[zoneName]
	if zone == nil {
		return map[int]int{}, fmt.Errorf("CreateEphemeralZone: zone %q not found", zoneName)
	}

	roomIds := make([]int, 0, len(zone.RoomIds))
	for roomId := range zone.RoomIds {
		roomIds = append(roomIds, roomId)
	}

	// Fallback: if the zone's rooms aren't registered in memory yet — e.g. a
	// force-created instance whose template room was never visited (the jail
	// cell is created at arrest, not entered via a portal) — clone the
	// configured entry room directly. CreateEphemeralRoomIds -> LoadRoomTemplate
	// reads from the file cache, so no prior visit is required. Correct and
	// complete for single-room zones; mirrors the oasis entry-room clone.
	if len(roomIds) == 0 && zone.EntryRoom != 0 {
		roomIds = append(roomIds, zone.EntryRoom)
	}

	return CreateEphemeralRoomIds(roomIds...)
}

func IsEphemeralRoomId(roomId int) bool {
	return roomId >= ephemeralRoomIdMinimum
}

func TryEphemeralCleanup(ephemeralRoomId int) []int {

	chunkId := int(math.Floor(float64(ephemeralRoomId-ephemeralRoomIdMinimum) / ephemeralChunkSize))

	for _, ephemeralRoomId := range ephemeralRoomChunks[chunkId] {

		room := LoadRoom(ephemeralRoomId)
		if room == nil {
			continue
		}

		if len(room.players) > 0 {
			return []int{}
		}

		// Don't clean up rooms belonging to an active instance —
		// instance cleanup is handled by the TTL chain in CheckPortalTimers
		// (which calls Remove(inst) before calling TryEphemeralCleanup).
		if instanceRegistry.FindByRoomId(ephemeralRoomId) != nil {
			return []int{}
		}
	}

	deletedMin := 0
	deletedMax := 0

	deletedRoomIds := make([]int, len(ephemeralRoomChunks[chunkId]))

	for i, ephemeralRoomId := range ephemeralRoomChunks[chunkId] {

		deletedRoomIds[i] = ephemeralRoomId

		if deletedMin == 0 || ephemeralRoomId < deletedMin {
			deletedMin = ephemeralRoomId
		}
		if deletedMax == 0 || ephemeralRoomId > deletedMax {
			deletedMax = ephemeralRoomId
		}

		room := LoadRoom(ephemeralRoomId)
		if room == nil {
			continue
		}

		delete(originalRoomIdLookups, room.RoomId)
		removeRoomFromMemory(room)
	}

	ephemeralRoomChunks[chunkId] = []int{}

	mudlog.Info("TryEphemeralCleanup", "deleted", len(deletedRoomIds), "chunkId", chunkId, "RoomIds", fmt.Sprintf("%d - %d", deletedMin, deletedMax), "Chunks Remaining", GetChunkCount())

	return deletedRoomIds
}

// All this does is unload chunks with no players in them.
func EphemeralRoomMaintenance() []int {
	start := time.Now()
	defer func() {
		util.TrackTime(`EphemeralRoomMaintenance()`, time.Since(start).Seconds())
	}()

	// If no lookups are stored, then there can't be anything in the chunks (unless we messed up)
	if len(originalRoomIdLookups) == 0 {
		return []int{}
	}

	for i := 0; i < ephemeralChunksLimit; i++ {
		if len(ephemeralRoomChunks[i]) > 0 {
			return TryEphemeralCleanup(ephemeralRoomChunks[i][0])
		}
	}
	return []int{}
}

func GetOriginalRoom(roomId int) int {
	if roomId < ephemeralRoomIdMinimum {
		return roomId
	}
	return originalRoomIdLookups[roomId]
}

// OriginalRoomId returns the template room id an ephemeral room was copied
// from, and true, if roomId is an ephemeral instance. Returns (roomId, false)
// for non-ephemeral rooms.
//
// NOTE: originalRoomIdLookups is accessed lock-free throughout this package
// (CreateEphemeralRoomIds, TryEphemeralCleanup, FindEphemeralRoomIds, and
// GetOriginalRoom all operate without a mutex). This function matches that
// convention. Callers that write to the map (CreateEphemeralRoomIds,
// TryEphemeralCleanup) run from the main game loop on a single goroutine, so
// there is no concurrent write hazard today; if that changes a read lock
// should be added here and in all other readers.
func OriginalRoomId(roomId int) (int, bool) {
	if orig, ok := originalRoomIdLookups[roomId]; ok {
		return orig, true
	}
	return roomId, false
}

func init() {
	if math.MaxInt > ephemeralRoomIdMinimum*1000 {
		ephemeralRoomIdMinimum = ephemeralRoomIdMinimum * 1000 // 1,000,000 => 1,000,000,000 on 64-bit systems
	}
}
