package mobs

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// Pack roaming coordinates alpha-follow movement so grouped mobs
// (sharing a group tag) move together under an elected leader.

// PackRoamingEnabled returns whether the pack roaming system is active.
func PackRoamingEnabled() bool {
	return bool(configs.GetBalanceConfig().PackRoamingEnabled)
}

// SelectPackAlpha picks the mob with the highest StatPool from the given
// instance IDs. Ties are broken by lowest InstanceId. Returns 0 if fewer
// than 2 members.
func SelectPackAlpha(memberInstIds []int) int {
	if len(memberInstIds) < 2 {
		return 0
	}

	bestId := 0
	bestPool := -1

	for _, instId := range memberInstIds {
		mob := GetInstance(instId)
		if mob == nil {
			continue
		}
		if mob.StatPool > bestPool || (mob.StatPool == bestPool && (bestId == 0 || instId < bestId)) {
			bestPool = mob.StatPool
			bestId = instId
		}
	}

	return bestId
}

// GetPackMembersInRoom returns instance IDs of mobs in the given room
// that share at least one group tag with the given mob.
func GetPackMembersInRoom(mob *Mob, roomMobIds []int) []int {
	if len(mob.Groups) == 0 {
		return nil
	}

	groupSet := make(map[string]bool, len(mob.Groups))
	for _, g := range mob.Groups {
		groupSet[g] = true
	}

	var members []int
	for _, instId := range roomMobIds {
		other := GetInstance(instId)
		if other == nil {
			continue
		}
		for _, g := range other.Groups {
			if groupSet[g] {
				members = append(members, instId)
				break
			}
		}
	}
	return members
}

// TickPackRoaming runs once per round. For each room with 2+ same-group
// mobs, it elects an alpha and assigns PackAlphaId on followers. It also
// decrements ScatterRounds on scattered mobs.
// Caller must check PackRoamingEnabled() before calling.
func TickPackRoaming() {
	maxSize := int(configs.GetBalanceConfig().PackMaxSize)
	tickPackRoamingWithMaxSize(maxSize)
}

// tickPackRoamingWithMaxSize is the internal implementation, separated
// for testability (avoids config dependency in unit tests).
func tickPackRoamingWithMaxSize(maxSize int) {
	// First pass: decrement scatter rounds and clear stale pack state
	allIds := GetAllMobInstanceIds()
	for _, instId := range allIds {
		mob := GetInstance(instId)
		if mob == nil {
			continue
		}

		// Decrement scatter rounds
		if mob.ScatterRounds > 0 {
			mob.ScatterRounds--
		}

		// Clear stale alpha references — if our alpha no longer exists
		// or is in a different room, clear pack state
		if mob.PackAlphaId > 0 {
			alpha := GetInstance(mob.PackAlphaId)
			if alpha == nil || alpha.Character.RoomId != mob.Character.RoomId {
				mob.PackAlphaId = 0
				mob.IsPackAlpha = false
			}
		}
	}

	// Second pass: build room → group → members map
	type roomGroup struct {
		roomId   int
		groupTag string
	}
	rgMembers := map[roomGroup][]int{}

	for _, instId := range allIds {
		mob := GetInstance(instId)
		if mob == nil || len(mob.Groups) == 0 {
			continue
		}
		// Skip scattered mobs — they're not ready to form packs yet
		if mob.ScatterRounds > 0 {
			continue
		}
		for _, g := range mob.Groups {
			key := roomGroup{roomId: mob.Character.RoomId, groupTag: g}
			rgMembers[key] = append(rgMembers[key], instId)
		}
	}

	// Track which mobs have already been assigned this tick
	assigned := map[int]bool{}

	// Third pass: elect alphas per room-group
	for _, members := range rgMembers {
		if len(members) < 2 {
			for _, instId := range members {
				mob := GetInstance(instId)
				if mob != nil {
					mob.IsPackAlpha = false
					mob.PackAlphaId = 0
				}
			}
			continue
		}

		alphaId := SelectPackAlpha(members)
		if alphaId == 0 {
			continue
		}

		// Sort members so alpha assignment is deterministic
		sort.Ints(members)

		alphaMob := GetInstance(alphaId)
		if alphaMob != nil && !assigned[alphaId] {
			alphaMob.IsPackAlpha = true
			alphaMob.PackAlphaId = 0 // Alpha doesn't follow anyone
			assigned[alphaId] = true
		}

		followerCount := 0
		for _, instId := range members {
			if instId == alphaId {
				continue
			}
			mob := GetInstance(instId)
			if mob == nil {
				continue
			}

			// Enforce max pack size
			if maxSize > 0 && followerCount >= maxSize {
				mob.PackAlphaId = 0
				mob.IsPackAlpha = false
				continue
			}

			mob.PackAlphaId = alphaId
			mob.IsPackAlpha = false
			assigned[instId] = true
			followerCount++
		}
	}

	// Final pass: clear pack state on any mob that wasn't assigned this tick
	for _, instId := range allIds {
		if assigned[instId] {
			continue
		}
		mob := GetInstance(instId)
		if mob == nil {
			continue
		}
		mob.IsPackAlpha = false
		mob.PackAlphaId = 0
	}
}

// HandleAlphaDeath clears pack state on surviving members and sets
// ScatterRounds so they skip wandering briefly after their leader dies.
// Caller must check PackRoamingEnabled() before calling.
func HandleAlphaDeath(deadMob *Mob, roomMobIds []int) {
	if deadMob == nil || len(deadMob.Groups) == 0 {
		return
	}

	scatterRounds := int(configs.GetBalanceConfig().PackScatterRounds)
	handleAlphaDeathWithScatter(deadMob, roomMobIds, scatterRounds)
}

// handleAlphaDeathWithScatter is the internal implementation.
func handleAlphaDeathWithScatter(deadMob *Mob, roomMobIds []int, scatterRounds int) {
	if deadMob == nil || len(deadMob.Groups) == 0 {
		return
	}
	groupSet := make(map[string]bool, len(deadMob.Groups))
	for _, g := range deadMob.Groups {
		groupSet[g] = true
	}

	for _, instId := range roomMobIds {
		mob := GetInstance(instId)
		if mob == nil || mob.InstanceId == deadMob.InstanceId {
			continue
		}

		hasOverlap := false
		for _, g := range mob.Groups {
			if groupSet[g] {
				hasOverlap = true
				break
			}
		}
		if !hasOverlap {
			continue
		}

		mob.PackAlphaId = 0
		mob.IsPackAlpha = false
		if scatterRounds > 0 {
			mob.ScatterRounds = scatterRounds
		}
	}
}

// MovePackFollowers moves all followers of the given alpha mob through
// the specified exit. Called after the alpha successfully moves.
// Caller must check PackRoamingEnabled() before calling.
func MovePackFollowers(alphaMob *Mob, exitName string, oldRoomMobIds []int) {
	if alphaMob == nil {
		return
	}

	for _, instId := range oldRoomMobIds {
		mob := GetInstance(instId)
		if mob == nil || mob.InstanceId == alphaMob.InstanceId {
			continue
		}

		if mob.PackAlphaId != alphaMob.InstanceId {
			continue
		}

		// Check MaxWander limit
		if mob.MaxWander > -1 && mob.WanderCount > mob.MaxWander {
			mob.PackAlphaId = 0
			continue
		}

		mob.WanderCount++
		mob.Command("go " + exitName)
	}
}
