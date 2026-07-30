package mapper

import (
	"container/heap"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
	lru "github.com/hashicorp/golang-lru/v2"
)

var (
	ErrPathNotFound  = errors.New(`path not found`)
	ErrPathDestMatch = errors.New(`path destination is same as source`)
	pathCache, _     = lru.New[pathCacheKey, pathCacheValue](128)
)

type pathCacheKey struct {
	startRoomId int
	endRoomId   int
}

type pathCacheValue struct {
	steps []pathStep
	err   error
}

// pathStep is one move: take ExitName to arrive in RoomID.
type pathStep struct {
	exitName string
	roomId   int
	waypoint bool
}

func (p pathStep) ExitName() string {
	return p.exitName
}

func (p pathStep) RoomId() int {
	return p.roomId
}

func (p pathStep) Waypoint() bool {
	return p.waypoint
}

// internal struct to record how we reached each node
type prevInfo struct {
	prevRoom int    // the room we came from
	viaExit  string // the exit name we used
}

// A nodeRecord is what we store in the priority queue.
type nodeRecord struct {
	roomId int
	fScore float64
	index  int
}

type priorityQueue []*nodeRecord

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].fScore < pq[j].fScore }
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *priorityQueue) Push(x interface{}) {
	n := x.(*nodeRecord)
	n.index = len(*pq)
	*pq = append(*pq, n)
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := old[len(old)-1]
	old[len(old)-1] = nil // avoid mem leak
	n.index = -1
	*pq = old[:len(old)-1]
	return n
}
func (pq *priorityQueue) update(n *nodeRecord, newF float64) {
	n.fScore = newF
	heap.Fix(pq, n.index)
}

// heuristic is the 3D Manhattan distance.
// This should be redone - rooms can be one space away exit-wise, but 2+ spacex in grid space.
// Example: Frostfang city walls
func (r *mapper) heuristic(a, b int) float64 {
	ax, ay, az, _ := r.GetCoordinates(a)
	bx, by, bz, _ := r.GetCoordinates(b)
	return math.Abs(float64(ax-bx)) +
		math.Abs(float64(ay-by)) +
		math.Abs(float64(az-bz))
}

// FindPath returns the sequence of ExitName/RoomID steps from startRoom to goalRoom.
func (r *mapper) findPath(startRoom, goalRoom int) ([]pathStep, error) {

	if startRoom == goalRoom {
		return nil, ErrPathDestMatch
	}

	// sanity check
	if _, ok := r.crawledRooms[startRoom]; !ok {
		return nil, ErrRoomNotFound
	}
	if _, ok := r.crawledRooms[goalRoom]; !ok {
		return nil, ErrRoomNotFound
	}

	// cameFrom holds, for each room, how we got there.
	cameFrom := make(map[int]prevInfo, len(r.crawledRooms))

	// gScore: cost from start to here; fScore = gScore + heuristic
	gScore := make(map[int]float64, len(r.crawledRooms))
	fScore := make(map[int]float64, len(r.crawledRooms))
	for id := range r.crawledRooms {
		gScore[id] = math.Inf(1)
		fScore[id] = math.Inf(1)
	}
	gScore[startRoom] = 0
	fScore[startRoom] = r.heuristic(startRoom, goalRoom)

	// open set as a priority queue
	openSet := make(priorityQueue, 0, len(r.crawledRooms))
	heap.Init(&openSet)
	heap.Push(&openSet, &nodeRecord{roomId: startRoom, fScore: fScore[startRoom]})
	inOpen := map[int]*nodeRecord{startRoom: openSet[0]}

	for openSet.Len() > 0 {
		current := heap.Pop(&openSet).(*nodeRecord)
		delete(inOpen, current.roomId)

		// reached goal!
		if current.roomId == goalRoom {
			// reconstruct path
			var path []pathStep
			cur := goalRoom
			for cur != startRoom {
				info := cameFrom[cur]

				// record the exit name and the room we arrived in
				path = append(path, pathStep{exitName: info.viaExit, roomId: cur})
				cur = info.prevRoom
			}

			pathLen := len(path)

			if pathLen > 0 {
				// reverse
				for i := 0; i < pathLen/2; i++ {
					j := len(path) - 1 - i
					path[i], path[j] = path[j], path[i]
				}

				// Mark the final room as the waypoint
				path[pathLen-1].waypoint = true
			}
			return path, nil
		}

		// expand neighbors
		node := r.crawledRooms[current.roomId]
		for exitName, exitInfo := range node.Exits {
			neighbor := exitInfo.RoomId
			tentativeG := gScore[current.roomId] + 1 // uniform cost

			if tentativeG < gScore[neighbor] {
				// this is a better path to neighbor
				cameFrom[neighbor] = prevInfo{
					prevRoom: current.roomId,
					viaExit:  exitName,
				}
				gScore[neighbor] = tentativeG
				fScore[neighbor] = tentativeG + r.heuristic(neighbor, goalRoom)

				if nr, ok := inOpen[neighbor]; ok {
					openSet.update(nr, fScore[neighbor])
				} else {
					nr := &nodeRecord{roomId: neighbor, fScore: fScore[neighbor]}
					heap.Push(&openSet, nr)
					inOpen[neighbor] = nr
				}
			}
		}
	}

	return nil, ErrPathNotFound
}

func GetPath(startRoomId int, endRoomId ...int) ([]pathStep, error) {

	start := time.Now()
	defer func() {
		util.TrackTime(`mapper.GetPath()`, time.Since(start).Seconds())
	}()

	if len(endRoomId) == 0 {
		return []pathStep{}, fmt.Errorf("%d => %d (endRoom not found): %w", startRoomId, endRoomId, ErrPathNotFound)
	}

	startRoom := rooms.LoadRoom(startRoomId)
	if startRoom == nil {
		return []pathStep{}, fmt.Errorf("%d => %d (startRoom  not found): %w", startRoomId, endRoomId, ErrPathNotFound)
	}

	m := GetMapper(startRoom.RoomId)
	if m == nil {
		return []pathStep{}, fmt.Errorf("%d => %d (mapper not fond): %w", startRoomId, endRoomId, ErrPathNotFound)
	}

	cacheKey := pathCacheKey{}
	rNow := startRoomId
	finalPath := []pathStep{}
	for _, roomId := range endRoomId {

		if rNow == roomId { // Avoid repeating id's
			continue
		}

		cacheKey.startRoomId = rNow
		cacheKey.endRoomId = roomId

		if pCache, ok := pathCache.Get(cacheKey); ok {

			if pCache.err != nil {
				return pCache.steps, fmt.Errorf("%d => %d: %w", rNow, roomId, pCache.err)
			}

			finalPath = append(finalPath, pCache.steps...)
			rNow = roomId
			continue
		}

		if !m.HasRoom(roomId) {
			err := fmt.Errorf("%d => %d (room not in mapper): %w", rNow, roomId, ErrPathNotFound)
			pathCache.Add(cacheKey, pathCacheValue{steps: nil, err: err})
			return []pathStep{}, err
		}

		p, err := m.findPath(rNow, roomId)
		if err != nil {
			pathCache.Add(cacheKey, pathCacheValue{steps: p, err: err})
			return []pathStep{}, fmt.Errorf("%d => %d: %w", rNow, roomId, ErrPathNotFound)
		}

		// Add to LRU cache
		pathCache.Add(cacheKey, pathCacheValue{steps: p, err: nil})

		finalPath = append(finalPath, p...)
		rNow = roomId
	}

	// If final path is empty, they mapped to the same room they are in.
	// This can occur if endRoomId differs from startRoomId, but endRoomId was actually
	// an alias equal to startRoomId
	if len(finalPath) == 0 {
		return finalPath, ErrPathDestMatch
	}

	return finalPath, nil
}

// firstHop extracts the next room + exit direction from a computed path.
func firstHop(path []pathStep) (nextRoomId int, exitName string, found bool) {
	if len(path) == 0 {
		return 0, "", false
	}
	return path[0].roomId, path[0].exitName, true
}

// NextStep returns the next room to head toward on the way from fromRoomId to
// toRoomId, and the compass exit name to take. found is false when from == to
// or the target is genuinely unreachable.
//
// Same-zone journeys use the cached per-zone GetPath (as `pathto` does).
// Cross-zone journeys fall back to zone-level routing: head for the border of
// the neighbouring zone that leads toward the target. GetPath alone cannot do
// this — it resolves its mapper from the START room's zone, so it fails
// outright on a target elsewhere, which used to leave cross-zone quest markers
// with no arrow at all. See mapper.crosszone.go.
func NextStep(fromRoomId, toRoomId int) (nextRoomId int, exitName string, found bool) {
	if fromRoomId == toRoomId {
		return 0, "", false
	}
	if path, err := GetPath(fromRoomId, toRoomId); err == nil {
		return firstHop(path)
	}

	fromRoom := rooms.LoadRoom(fromRoomId)
	toRoom := rooms.LoadRoom(toRoomId)
	if fromRoom == nil || toRoom == nil || fromRoom.Zone == toRoom.Zone {
		return 0, "", false // same zone and still unreachable: nothing to add
	}
	return crossZoneHop(fromRoomId, fromRoom.Zone, toRoom.Zone)
}
