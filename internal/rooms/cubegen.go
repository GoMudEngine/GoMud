package rooms

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const (
	cubeSize = 5
	cubeTotal = cubeSize * cubeSize * cubeSize // 125
)

// Trash elemental mob IDs.
var cubeTrashMobs = []int{310, 311, 312, 313, 314}

// Tough mob IDs.
var cubeToughMobs = []int{318, 319}

// Boss mob IDs.
var cubeBossMobs = []int{320, 321, 322}

var cubeDescriptions = []string{
	"Crystalline sand stretches in every direction, each grain a tiny prism. The air hums with planar energy.",
	"Scorched earth crunches underfoot. Heat radiates from cracks in the ground where molten rock glows faintly below.",
	"Frozen dunes of pale blue ice rise in gentle waves. The cold is sharp and immediate, cutting through armor.",
	"Wind-torn flats where grit stings exposed skin. Dust devils spiral lazily in the distance.",
	"A forest of stone pillars rises from the sand, each one thrumming with a low vibration felt more than heard.",
	"The ground here is smooth obsidian, reflecting a sky that cannot decide between dawn and dusk.",
	"Pools of dark water dot the sand, each perfectly still. Something moves beneath the surface of the nearest one.",
	"The air crackles with static. Tiny arcs of lightning jump between grains of sand with each footstep.",
	"Massive crystals jut from the ground at odd angles, casting prismatic light in all directions.",
	"The sand gives way to packed clay veined with glowing mineral deposits. The warmth here is almost pleasant.",
}

var cubeTitles = []string{"Elemental Depths", "Lower Wastes", "Middle Wastes", "Upper Wastes", "Elemental Heights"}

// wrapCoord wraps a coordinate into [0, size).
func wrapCoord(v, size int) int {
	return ((v % size) + size) % size
}

// cubeIndex converts (x, y, z) to a flat index: x*25 + y*5 + z.
func cubeIndex(x, y, z int) int {
	return x*cubeSize*cubeSize + y*cubeSize + z
}

// cubeTitle returns the title for a given z-level.
func cubeTitle(z int) string {
	if z >= 0 && z < len(cubeTitles) {
		return cubeTitles[z]
	}
	return "Planar Wastes"
}

// GenerateOasisCube creates a 5x5x5 cube of ephemeral rooms with wrapping
// exits. Returns the slice of all cube room IDs and the entry cube room ID
// at position (2,2,0) which connects south to entryRoomId.
func GenerateOasisCube(
	entryRoomId int,
	zoneName string,
	goldPaid int,
	instanceId int,
	allowRecall bool,
	deathPolicy string,
) (cubeRoomIds []int, entryCubeRoomId int, err error) {

	// 1. Reserve a chunk of ephemeral IDs.
	chunkId := -1
	for i := 0; i < ephemeralChunksLimit; i++ {
		if len(ephemeralRoomChunks[i]) == 0 {
			chunkId = i
			break
		}
	}
	if chunkId < 0 {
		return nil, 0, errEphemeralChunkLimit
	}

	baseId := ephemeralRoomIdMinimum + (chunkId * ephemeralChunkSize)

	// 2. Create 125 rooms and compute their IDs.
	roomIds := make([]int, cubeTotal)
	rooms := make([]*Room, cubeTotal)

	for x := 0; x < cubeSize; x++ {
		for y := 0; y < cubeSize; y++ {
			for z := 0; z < cubeSize; z++ {
				idx := cubeIndex(x, y, z)
				roomId := baseId + idx
				roomIds[idx] = roomId

				room := &Room{
					RoomId:      roomId,
					Zone:        zoneName,
					Title:       cubeTitle(z),
					Description: cubeDescriptions[util.Rand(len(cubeDescriptions))],
					Exits:       make(map[string]exit.RoomExit),
					ExitsTemp:   make(map[string]exit.TemporaryRoomExit),
				}
				rooms[idx] = room
			}
		}
	}

	// 3. Wire wrapping exits on every room.
	for x := 0; x < cubeSize; x++ {
		for y := 0; y < cubeSize; y++ {
			for z := 0; z < cubeSize; z++ {
				idx := cubeIndex(x, y, z)
				room := rooms[idx]

				room.Exits["north"] = exit.RoomExit{RoomId: roomIds[cubeIndex(x, wrapCoord(y+1, cubeSize), z)]}
				room.Exits["south"] = exit.RoomExit{RoomId: roomIds[cubeIndex(x, wrapCoord(y-1, cubeSize), z)]}
				room.Exits["east"] = exit.RoomExit{RoomId: roomIds[cubeIndex(wrapCoord(x+1, cubeSize), y, z)]}
				room.Exits["west"] = exit.RoomExit{RoomId: roomIds[cubeIndex(wrapCoord(x-1, cubeSize), y, z)]}
				room.Exits["up"] = exit.RoomExit{RoomId: roomIds[cubeIndex(x, y, wrapCoord(z+1, cubeSize))]}
				room.Exits["down"] = exit.RoomExit{RoomId: roomIds[cubeIndex(x, y, wrapCoord(z-1, cubeSize))]}
			}
		}
	}

	// 4. Override room (2,2,0) south exit to point to the entry room.
	entryIdx := cubeIndex(2, 2, 0)
	entryCubeRoomId = roomIds[entryIdx]
	entryRoom := rooms[entryIdx]
	entryRoom.Exits["south"] = exit.RoomExit{RoomId: entryRoomId}

	// 5. Assign mob spawns: 1 trash elemental per room. Force hostile
	//    since the base templates are companion mobs (not hostile).
	//    All mobs wander through the cube.
	wanderIdle := []string{"wander", "", "wander", ""}
	for i := 0; i < cubeTotal; i++ {
		rooms[i].SpawnInfo = []SpawnInfo{
			{
				MobId:        cubeTrashMobs[util.Rand(len(cubeTrashMobs))],
				StatPool:     1,
				RespawnRate:  "2 real hours",
				ForceHostile: true,
				MaxWander:    5,
				IdleCommands: wanderIdle,
			},
		}
	}

	// Pick 2 random rooms for tough mobs.
	toughIndices := pickUniqueIndices(2, cubeTotal, nil)
	for _, idx := range toughIndices {
		rooms[idx].SpawnInfo = []SpawnInfo{
			{
				MobId:        cubeToughMobs[util.Rand(len(cubeToughMobs))],
				StatPool:     2,
				RespawnRate:  "2 real hours",
				ForceHostile: true,
				MaxWander:    5,
				IdleCommands: wanderIdle,
			},
		}
	}

	// Pick 1 random room for a boss (different from tough rooms).
	bossIndices := pickUniqueIndices(1, cubeTotal, toughIndices)
	for _, idx := range bossIndices {
		rooms[idx].SpawnInfo = []SpawnInfo{
			{
				MobId:        cubeBossMobs[util.Rand(len(cubeBossMobs))],
				StatPool:     4,
				RespawnRate:  "2 real hours",
				ForceHostile: true,
				MaxWander:    5,
				IdleCommands: wanderIdle,
			},
		}
	}

	// 6. Add all rooms to memory and stamp temp data.
	for i := 0; i < cubeTotal; i++ {
		if err := addRoomToMemory(rooms[i]); err != nil {
			return nil, 0, fmt.Errorf("GenerateOasisCube: addRoomToMemory room %d: %w", rooms[i].RoomId, err)
		}
		rooms[i].SetTempData("instance_id", instanceId)
		rooms[i].SetTempData("allow_recall", allowRecall)
		rooms[i].SetTempData("death_policy", deathPolicy)
		rooms[i].SetTempData("gold_paid", goldPaid)
	}

	// 7. Register the chunk so cleanup works.
	ephemeralRoomChunks[chunkId] = roomIds

	// 8. Add a north exit on the entry room pointing into the cube.
	thresholdRoom := LoadRoom(entryRoomId)
	if thresholdRoom != nil {
		thresholdRoom.AddTemporaryExit("north", exit.TemporaryRoomExit{
			RoomId:       entryCubeRoomId,
			Title:        "north",
			UserId:       0,
			SpawnedRound: util.GetRoundCount(),
			Expires:      "999 real hours",
		})
	}

	mudlog.Info("GenerateOasisCube",
		"created", cubeTotal,
		"chunkId", chunkId,
		"baseId", baseId,
		"entryCubeRoomId", entryCubeRoomId,
	)

	cubeRoomIds = roomIds
	return cubeRoomIds, entryCubeRoomId, nil
}

// pickUniqueIndices selects count random unique indices in [0, max),
// excluding any indices in the exclude slice.
func pickUniqueIndices(count, max int, exclude []int) []int {
	excludeSet := make(map[int]bool, len(exclude))
	for _, v := range exclude {
		excludeSet[v] = true
	}

	selected := make(map[int]bool, count)
	result := make([]int, 0, count)

	for len(result) < count {
		idx := util.Rand(max)
		if excludeSet[idx] || selected[idx] {
			continue
		}
		selected[idx] = true
		result = append(result, idx)
	}
	return result
}
