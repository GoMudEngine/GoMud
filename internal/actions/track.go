package actions

import (
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TrackOptions parameterizes the trail-read.
type TrackOptions struct {
	// TargetNoun: name of target to actively track. Empty = trail-scan
	// current room mode.
	TargetNoun string

	// TargetFrom (mob path): "aggro" | "event" | "soft_target" | "none".
	// Ignored if TargetNoun is set. When set, the action resolves the
	// target's name from the named context source and treats it as
	// TargetNoun.
	TargetFrom string

	// CancelTracking: "stop" / "clear" semantics — remove buff 86 + clear
	// tracking misc data without rolling. UserActor wrapper handles
	// the keyword check.
	CancelTracking bool
}

// TrackingInfo is the per-visitor record used by the descriptions/track
// template. Lifted verbatim from usercommands/skill.track.go.
type TrackingInfo struct {
	Name            string
	Type            string // "mob" or "user"
	Strength        string
	NumericStrength float64
	ExitName        string
}

// TrackResult is the structured outcome.
type TrackResult struct {
	// Trail-scan mode populates Visitors.
	Visitors []TrackingInfo

	// Active-track mode populates these.
	ActiveTargetUserId    int    // 0 when not user target
	ActiveTargetMobInstId int    // 0 when not mob target
	ActiveTargetName      string // for caller messaging
	DirectionExit         string // best exit toward target
	BuffApplied           bool   // true when buff 86 applied

	// Common.
	RollValue  float64 // roll.Value for tier-band introspection
	OnCooldown bool    // 1-round cooldown collision
	Reason     string  // human-readable reason on failure
}

// activeTrackingBuffId is the buff applied when active tracking starts.
// See _datafiles/world/dogmud/buffs/86-active_tracking.yaml.
const activeTrackingBuffId = 86

// Track runs the Perception+Search trail-read. With TargetNoun set (or
// resolved via TargetFrom), attempts active-tracking: locates the trail
// across adjacent rooms, applies buff 86, and stores tracking-user or
// tracking-mob misc data on the actor's Character. Without a target,
// reports the visitor log of the current room (tiered by roll).
func Track(actor Actor, opts TrackOptions) TrackResult {
	result := TrackResult{Visitors: []TrackingInfo{}}

	char := actor.GetCharacter()
	room := actor.GetRoom()
	if char == nil || room == nil {
		result.Reason = "no character or room"
		return result
	}

	// "stop"/"clear" path — pure cleanup, no roll.
	if opts.CancelTracking {
		char.SetMiscData("tracking-mob", nil)
		char.SetMiscData("tracking-user", nil)
		char.SetMiscData("tracking-display-count", nil)
		char.RemoveBuff(activeTrackingBuffId)
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem, `You stop tracking.`)
		}
		return result
	}

	// Resolve target source if TargetNoun absent.
	// TargetFrom resolution is handled by the btree wrapper before calling;
	// this action only consumes TargetNoun.
	targetNoun := opts.TargetNoun

	// Roll the Perception+Search score.
	searchScore := CalcSearchScore(char)
	roll := dice.RollStat(searchScore)
	result.RollValue = roll.Value

	// Skill progression on every fired roll.
	actor.OnSkillUse(string(skills.Search))

	// Roll < 125.0: no tracks visible at all.
	if roll.Value < 125.0 {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem, "You don't see any tracks.")
		}
		result.Reason = "roll below detection threshold"
		return result
	}

	// Trail-scan mode (no-arg).
	if targetNoun == "" {
		if !char.TryCooldown(skills.Search.String(), "1 round") {
			result.OnCooldown = true
			if actor.IsPlayer() {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf("You need to wait %d more rounds to use that skill again.",
						char.GetCooldown(skills.Search.String())))
			}
			return result
		}

		result.Visitors = readRoomTrail(room, roll.Value, actor.GetUserId(), actor.GetMobInstanceId())
		if actor.IsPlayer() {
			renderTrailToPlayer(actor, result.Visitors)
		}
		return result
	}

	// Active-track mode.
	if roll.Value < 175.0 {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem, "Your tracking skills aren't sharp enough right now.")
		}
		result.Reason = "active-track requires roll >= 175"
		return result
	}

	if !char.TryCooldown(skills.Search.String(), "1 round") {
		result.OnCooldown = true
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf("You need to wait %d more rounds to use that skill again.",
					char.GetCooldown(skills.Search.String())))
		}
		return result
	}

	// Find target in current room first (just reports "they are here").
	if targetUser := findUserInRoomByName(room, targetNoun, actor.GetUserId()); targetUser != nil {
		result.ActiveTargetUserId = targetUser.UserId
		result.ActiveTargetName = targetUser.Character.Name
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> is in the room with you!`, targetUser.Character.Name))
		}
		return result
	}
	if targetMob := findMobInRoomByName(room, targetNoun); targetMob != nil {
		result.ActiveTargetMobInstId = targetMob.InstanceId
		result.ActiveTargetName = targetMob.Character.Name
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is in the room with you!`, targetMob.Character.Name))
		}
		return result
	}

	// Search visitor log of current room for a trail matching targetNoun;
	// if found, apply buff 86 + store misc data + populate DirectionExit.
	if applied, miscKey, miscVal, dirExit, targetUserId, targetMobId, targetName :=
		lookupAdjacentTrail(room, targetNoun, actor.GetUserId()); applied {
		char.SetMiscData(miscKey, miscVal)
		char.SetMiscData("tracking-display-count", nil)
		actor.AddBuff(activeTrackingBuffId, "skill")
		result.BuffApplied = true
		result.ActiveTargetUserId = targetUserId
		result.ActiveTargetMobInstId = targetMobId
		result.ActiveTargetName = targetName
		result.DirectionExit = dirExit
		return result
	}

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem, "You don't see any tracks.")
	}
	result.Reason = "no trail found in adjacent rooms"
	return result
}

// readRoomTrail returns the visitor list for the current room, filtered
// by detection tier. roll < 135 returns at most one (strongest) visitor;
// roll >= 135 returns all; roll >= 175 also populates ExitName via
// adjacent-room scan.
func readRoomTrail(room *rooms.Room, rollValue float64, excludeUserId int, excludeMobInstId int) []TrackingInfo {
	out := []TrackingInfo{}
	currentMobs := room.GetMobs()
	currentUsers := room.GetPlayers()

	// Mob trails.
	for mId, timeLeft := range room.Visitors(rooms.VisitorMob) {
		if mId == excludeMobInstId {
			continue
		}
		skip := false
		for _, curId := range currentMobs {
			if mId == curId {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		m := mobs.GetInstance(mId)
		if m == nil {
			continue
		}
		info := TrackingInfo{
			Name:            m.Character.Name,
			Type:            "mob",
			Strength:        trailStrengthToString(timeLeft),
			NumericStrength: timeLeft,
		}
		if rollValue >= 175.0 {
			info.ExitName = findExitedTrack(room, mId, rooms.VisitorMob)
		}
		if rollValue < 135.0 {
			if len(out) == 0 {
				out = append(out, info)
			} else if out[0].NumericStrength < timeLeft {
				out[0] = info
			}
			continue
		}
		out = append(out, info)
	}

	// User trails.
	for uId, timeLeft := range room.Visitors(rooms.VisitorUser) {
		if uId == excludeUserId {
			continue
		}
		skip := false
		for _, curId := range currentUsers {
			if uId == curId {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		u := users.GetByUserId(uId)
		if u == nil {
			continue
		}
		info := TrackingInfo{
			Name:            u.Character.Name,
			Type:            "user",
			Strength:        trailStrengthToString(timeLeft),
			NumericStrength: timeLeft,
		}
		if rollValue >= 175.0 {
			info.ExitName = findExitedTrack(room, uId, rooms.VisitorUser)
		}
		if rollValue < 135.0 {
			if len(out) == 0 {
				out = append(out, info)
			} else if out[0].NumericStrength < timeLeft {
				out[0] = info
			}
			continue
		}
		out = append(out, info)
	}

	return out
}

// renderTrailToPlayer emits the descriptions/track template to the player.
// No-op for mob actors (caller gates on IsPlayer).
func renderTrailToPlayer(actor Actor, visitors []TrackingInfo) {
	if len(visitors) == 0 {
		actor.SendText(messaging.CategorySystem, "You don't see any tracks.")
		return
	}
	uid := actor.GetUserId()
	trackTxt, _ := templates.Process("descriptions/track", visitors, uid)
	actor.SendText(messaging.CategorySystem, trackTxt)
}

// findUserInRoomByName looks up a user in the room by prefix match.
// Returns nil if no match.
func findUserInRoomByName(room *rooms.Room, name string, excludeUserId int) *users.UserRecord {
	for _, pId := range room.GetPlayers() {
		if pId == excludeUserId {
			continue
		}
		u := users.GetByUserId(pId)
		if u == nil {
			continue
		}
		if strings.HasPrefix(strings.ToLower(u.Character.Name), strings.ToLower(name)) {
			return u
		}
	}
	return nil
}

// findMobInRoomByName looks up a mob in the room by prefix match.
func findMobInRoomByName(room *rooms.Room, name string) *mobs.Mob {
	for _, mId := range room.GetMobs() {
		m := mobs.GetInstance(mId)
		if m == nil {
			continue
		}
		if strings.HasPrefix(strings.ToLower(m.Character.Name), strings.ToLower(name)) {
			return m
		}
	}
	return nil
}

// lookupAdjacentTrail searches the current room's visitor log for a target
// matching targetNoun. Returns (applied=true) on first hit with the
// misc-data key/value to set, the exit direction toward the target, and
// target ids. Exit direction is found by scanning adjacent rooms for the
// strongest trail.
func lookupAdjacentTrail(room *rooms.Room, targetNoun string, excludeUserId int) (
	applied bool, miscKey string, miscVal interface{}, dirExit string, targetUserId int, targetMobId int, targetName string,
) {
	// First try users in current room's visitor log.
	allUserNames := []string{}
	for uId := range room.Visitors(rooms.VisitorUser) {
		if uId == excludeUserId {
			continue
		}
		if vu := users.GetByUserId(uId); vu != nil {
			allUserNames = append(allUserNames, vu.Character.Name)
		}
	}
	if len(allUserNames) > 0 {
		if match, closeMatch := util.FindMatchIn(targetNoun, allUserNames...); match != "" || closeMatch != "" {
			pick := match
			if pick == "" {
				pick = closeMatch
			}
			for uId := range room.Visitors(rooms.VisitorUser) {
				u := users.GetByUserId(uId)
				if u == nil || u.Character.Name != pick {
					continue
				}
				return true, "tracking-user", pick, findExitedTrack(room, uId, rooms.VisitorUser), u.UserId, 0, pick
			}
		}
	}

	// Then mobs in current room's visitor log.
	allMobNames := []string{}
	for mId := range room.Visitors(rooms.VisitorMob) {
		if vm := mobs.GetInstance(mId); vm != nil {
			allMobNames = append(allMobNames, vm.Character.Name)
		}
	}
	if len(allMobNames) > 0 {
		if match, closeMatch := util.FindMatchIn(targetNoun, allMobNames...); match != "" || closeMatch != "" {
			pick := match
			if pick == "" {
				pick = closeMatch
			}
			for mId := range room.Visitors(rooms.VisitorMob) {
				m := mobs.GetInstance(mId)
				if m == nil || m.Character.Name != pick {
					continue
				}
				return true, "tracking-mob", pick, findExitedTrack(room, mId, rooms.VisitorMob), 0, m.InstanceId, pick
			}
		}
	}

	return false, "", nil, "", 0, 0, ""
}

// trailStrengthToString converts a float64 trail strength to a human-readable
// tier name. Lifted from usercommands/skill.track.go.
func trailStrengthToString(trailStrength float64) string {
	strength := int(math.Round(trailStrength * 100))
	switch {
	case strength < 15:
		return "Dead"
	case strength < 50:
		return "Weak"
	case strength < 70:
		return "Good"
	case strength < 90:
		return "Warm"
	default:
		return "Hot"
	}
}

// findExitedTrack scans adjacent rooms for the strongest trail of the given
// target, returning the exit name that leads toward it. Lifted from
// usercommands/skill.track.go (renamed to avoid conflict with the existing
// package-level function of the same name).
func findExitedTrack(room *rooms.Room, targetId int, targetType rooms.VisitorType) string {
	var bestExit string
	var bestStrength float64

	for exitName, exitInfo := range room.Exits {
		if exitInfo.Secret {
			continue
		}
		testRoom := rooms.LoadRoom(exitInfo.RoomId)
		if testRoom == nil {
			continue
		}
		for vId, vStr := range testRoom.Visitors(targetType) {
			if vId != targetId {
				continue
			}
			if vStr < bestStrength {
				continue
			}
			bestExit = exitName
			bestStrength = vStr
		}
	}
	return bestExit
}
