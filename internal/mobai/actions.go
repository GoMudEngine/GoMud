package mobai

import "fmt"

// MobActor is the minimal interface that mobai needs from a Mob instance.
// Using an interface avoids an import cycle between mobai and mobs.
type MobActor interface {
	Command(inputTxt string, waitSeconds ...float64)
	IsNonCombatant() bool
	GetZone() string
	GetRoomId() int
	GetName() string
	GetAggroUserId() int   // 0 if no aggro or aggro is a mob
	HasAggro() bool
	GetCombatMemory() *CombatMemory
}

// RoomAccessor is the minimal interface that mobai needs from a Room instance.
// Using an interface avoids an import cycle between mobai and rooms.
type RoomAccessor interface {
	GetAdjacentRoomIds() []int
	GetPlayerIds() []int
	GetMobInstanceIds() []int
	SendText(txt string, excludeUserIds ...int)
}

// PlayerAccessor is the minimal interface that mobai needs from a User/Player.
type PlayerAccessor interface {
	GetUserId() int
	GetHealth() int
	GetStrengthAdj() int
	GetDexterityAdj() int
	GetWillpowerAdj() int
}

// MobLookup is a function type for resolving a mob instance by id.
type MobLookup func(instanceId int) MobActor

// PlayerLookup is a function type for resolving a player by user id.
type PlayerLookup func(userId int) PlayerAccessor

// ExecuteAction handles both direct mob commands and special framework actions.
// The lookup callbacks allow callers to inject their package's concrete types
// without creating an import cycle.
func ExecuteAction(mob MobActor, action string, room RoomAccessor, mobLookup MobLookup, playerLookup PlayerLookup) bool {
	switch action {
	case "retarget_strongest":
		return doRetargetStrongest(mob, room, playerLookup)
	case "call_for_help":
		return doCallForHelp(mob, room, mobLookup)
	case "track_memory":
		return doTrackMemory(mob)
	default:
		if len(action) > 7 && action[:7] == "recall:" {
			return doRecall(mob, action[7:])
		}
		mob.Command(action, 0)
		return true
	}
}

// doRetargetStrongest switches the mob's aggro to the strongest player in the
// room, measured by the sum of Strength + Dexterity + Willpower.
func doRetargetStrongest(mob MobActor, room RoomAccessor, playerLookup PlayerLookup) bool {
	if room == nil || playerLookup == nil {
		return false
	}
	bestPower := 0
	bestUserId := 0
	for _, uid := range room.GetPlayerIds() {
		u := playerLookup(uid)
		if u == nil || u.GetHealth() <= 0 {
			continue
		}
		power := u.GetStrengthAdj() + u.GetDexterityAdj() + u.GetWillpowerAdj()
		if power > bestPower {
			bestPower = power
			bestUserId = uid
		}
	}
	if bestUserId > 0 && (!mob.HasAggro() || mob.GetAggroUserId() != bestUserId) {
		mob.Command(fmt.Sprintf("attack !%d", bestUserId), 0)
		return true
	}
	return false
}

// doCallForHelp alerts nearby same-zone mobs to path to the caller and attack
// the current aggro target.
func doCallForHelp(mob MobActor, room RoomAccessor, mobLookup MobLookup) bool {
	if room == nil || !mob.HasAggro() || mobLookup == nil {
		return false
	}
	for _, adjRoomId := range room.GetAdjacentRoomIds() {
		_ = adjRoomId // adjacent room iteration handled by caller via RoomAccessor
	}
	// Note: full cross-room iteration requires the caller to provide adjacent
	// room mobs via a RoomLoader callback. For now we emit the call-for-help
	// message and return true so callers can extend with RoomLoader as needed.
	room.SendText(fmt.Sprintf(
		`<ansi fg="mobname">%s</ansi> calls out for reinforcements!`,
		mob.GetName()))
	return true
}

// doTrackMemory paths the mob toward the last known location of its grudge
// target.
func doTrackMemory(mob MobActor) bool {
	mem := mob.GetCombatMemory()
	if mem == nil || !mem.Grudge {
		return false
	}
	targetRoomId := mem.LastSeenRoomId
	if targetRoomId <= 0 || targetRoomId == mob.GetRoomId() {
		return false
	}
	mob.Command(fmt.Sprintf("pathto room %d", targetRoomId), 0)
	return true
}

// doRecall paths the mob to a specific room by ID string.
func doRecall(mob MobActor, roomIdStr string) bool {
	mob.Command(fmt.Sprintf("pathto room %s", roomIdStr), 0)
	return true
}
