package behaviortree

// actions_scout.go — btree action primitives for the scout suite:
//   try_scan, try_track, try_search, move_toward_tracked
//
// Each primitive resolves its target via ctx (Event.UserId →
// Aggro → SoftTarget chain) and delegates to actions.<Verb>.

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// actTryScan walks adjacent rooms via actions.Scan and, if hostile_only
// (default true), promotes the first hostile sighting to ctx.SoftTarget.
// Returns Success on any sighting (or hostile sighting when hostile_only);
// Failure otherwise.
func actTryScan(params map[string]any, ctx *EvalContext) Result {
	// hostile_only defaults to true — getBoolParam returns false when absent,
	// so we flip: treat absent as true by checking explicitly.
	hostileOnly := true
	if v, ok := params["hostile_only"]; ok {
		if b, ok := v.(bool); ok {
			hostileOnly = b
		} else if s, ok := v.(string); ok {
			hostileOnly = s != "false"
		}
	}

	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Scan(actor, actions.ScanOptions{HostileOnly: hostileOnly})

	if len(result.Sightings) == 0 {
		return Failure
	}

	// Walk sightings; promote first hostile (or any if !hostileOnly) to
	// SoftTarget.
	for _, s := range result.Sightings {
		for _, p := range s.Players {
			if hostileOnly && !isHostileToMob(mob, p.Id, true) {
				continue
			}
			ctx.SoftTarget = state.ActorRef{UserId: p.Id}
			return Success
		}
		for _, m := range s.Mobs {
			if hostileOnly && !isHostileToMob(mob, m.Id, false) {
				continue
			}
			ctx.SoftTarget = state.ActorRef{MobInstanceId: m.Id}
			return Success
		}
	}

	// No hostile match found. Non-hostile sightings still count as Success
	// when hostile_only is false.
	if !hostileOnly {
		return Success
	}
	return Failure
}

// actTryTrack resolves a target from the configured source and dispatches
// actions.Track. Sources: "aggro" (default), "event", "soft_target".
// Returns Success on trail hit or in-room target found; Failure on no
// target / no trail / on cooldown.
func actTryTrack(params map[string]any, ctx *EvalContext) Result {
	source := getStringParam(params, "target_from")
	if source == "" {
		source = "aggro"
	}

	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}

	targetName := resolveTrackTargetName(mob, ctx, source)
	if targetName == "" {
		return Failure
	}

	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Track(actor, actions.TrackOptions{TargetNoun: targetName})

	// Buff applied → trail found in adjacent rooms → success.
	// In-room hit also sets ActiveTarget* → success.
	if result.BuffApplied || result.ActiveTargetUserId != 0 || result.ActiveTargetMobInstId != 0 {
		// Seed SoftTarget for downstream consumers.
		if result.ActiveTargetUserId != 0 {
			ctx.SoftTarget = state.ActorRef{UserId: result.ActiveTargetUserId}
		} else if result.ActiveTargetMobInstId != 0 {
			ctx.SoftTarget = state.ActorRef{MobInstanceId: result.ActiveTargetMobInstId}
		}
		return Success
	}
	return Failure
}

// actTrySearch invokes actions.Search and promotes the first hidden
// hostile to ctx.SoftTarget. Tier 1 (exits, containers) and Tier 3
// (hidden nouns) hits are present in the result but ignored by the
// btree primitive — they're player-facing flavor only.
func actTrySearch(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Search(actor, actions.SearchOptions{})

	if result.OnCooldown {
		return Failure
	}

	// Promote first hostile-hidden to SoftTarget.
	for _, pId := range result.HiddenPlayersFound {
		if isHostileToMob(mob, pId, true) {
			ctx.SoftTarget = state.ActorRef{UserId: pId}
			return Success
		}
	}
	for _, mId := range result.HiddenMobsFound {
		if isHostileToMob(mob, mId, false) {
			ctx.SoftTarget = state.ActorRef{MobInstanceId: mId}
			return Success
		}
	}

	// Any non-hostile detection still counts as a sighting for the
	// archetype's purposes — but no SoftTarget seeded.
	if len(result.HiddenPlayersFound) > 0 || len(result.HiddenMobsFound) > 0 {
		return Success
	}
	return Failure
}

// actMoveTowardTracked reads the tracked target's misc data on the mob's
// character, locates the freshest adjacent-room trail toward that target,
// and dispatches a `go <direction>` command via mob.Command. Returns
// Failure if no buff 86 / no resolvable direction / no movement possible.
//
// Cleanup contract: when buff 86 is absent, clears tracking-mob /
// tracking-user / tracking-display-count misc data. Mirrors the
// roomdetails.go fix from Task 3 — misc-data state cannot outlive
// the buff.
func actMoveTowardTracked(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if !mob.Character.HasBuff(86) {
		// Buff expired or otherwise gone — clear stale misc data.
		mob.Character.SetMiscData("tracking-mob", nil)
		mob.Character.SetMiscData("tracking-user", nil)
		mob.Character.SetMiscData("tracking-display-count", nil)
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}

	// Resolve target name from misc data.
	var targetName string
	if v := mob.Character.GetMiscData("tracking-user"); v != nil {
		if s, ok := v.(string); ok {
			targetName = s
		}
	}
	if targetName == "" {
		if v := mob.Character.GetMiscData("tracking-mob"); v != nil {
			if s, ok := v.(string); ok {
				targetName = s
			}
		}
	}
	if targetName == "" {
		return Failure
	}

	// Find best exit direction via adjacent-room visitor trail scan.
	dir := findBestExitToward(room, targetName)
	if dir == "" {
		return Failure
	}

	// Dispatch `go <direction>` via the engine command pipeline.
	mob.Command("go " + dir)
	return Success
}

// resolveTrackTargetName returns the target's display name for the
// configured source. The action wrapper feeds this into
// actions.Track.TargetNoun (which does prefix-matching).
func resolveTrackTargetName(mob *mobs.Mob, ctx *EvalContext, source string) string {
	switch source {
	case "event":
		if ctx.Event.UserId > 0 {
			if u := users.GetByUserId(ctx.Event.UserId); u != nil {
				return u.Character.Name
			}
		}
	case "soft_target":
		if ctx.SoftTarget.IsPlayer() {
			if u := users.GetByUserId(ctx.SoftTarget.UserId); u != nil {
				return u.Character.Name
			}
		}
		if ctx.SoftTarget.IsMob() {
			if m := mobs.GetInstance(ctx.SoftTarget.MobInstanceId); m != nil {
				return m.Character.Name
			}
		}
	default: // "aggro"
		if mob.Character.Aggro != nil {
			if mob.Character.Aggro.UserId > 0 {
				if u := users.GetByUserId(mob.Character.Aggro.UserId); u != nil {
					return u.Character.Name
				}
			}
			if mob.Character.Aggro.MobInstanceId > 0 {
				if m := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); m != nil {
					return m.Character.Name
				}
			}
		}
	}
	return ""
}

// isHostileToMob returns true when the candidate target is hostile to the
// mob via mob.HatesMob (for mob-on-mob) or default-hostile (for
// mob-on-player). v1: any non-charmed player counts as hostile. Faction-
// rep substrate integration is logged as a followup (chunk 1.2 substrate
// consumer — see spec's Hostile determination section).
func isHostileToMob(mob *mobs.Mob, targetId int, targetIsPlayer bool) bool {
	if targetIsPlayer {
		// v1: treat any player as hostile. Charmed-pet and faction-rep
		// gating is deferred to the faction substrate followup.
		_ = targetId
		return true
	}
	if other := mobs.GetInstance(targetId); other != nil {
		return mob.HatesMob(other)
	}
	return false
}

// findBestExitToward returns the exit name whose adjacent room contains
// the strongest visitor trail for the named target. Scans adjacent rooms
// for both VisitorUser and VisitorMob trails, matching by name.
// Returns "" when no trail is found.
func findBestExitToward(room *rooms.Room, targetName string) string {
	var bestExit string
	var bestStrength float64

	for exitName, exitInfo := range room.Exits {
		if exitInfo.Secret {
			continue
		}
		adj := rooms.LoadRoom(exitInfo.RoomId)
		if adj == nil {
			continue
		}

		// Check user visitors in the adjacent room.
		for vId, vStr := range adj.Visitors(rooms.VisitorUser) {
			if u := users.GetByUserId(vId); u != nil {
				if !strings.EqualFold(u.Character.Name, targetName) {
					continue
				}
				if vStr > bestStrength {
					bestStrength = vStr
					bestExit = exitName
				}
			}
		}
		// Check mob visitors.
		for vId, vStr := range adj.Visitors(rooms.VisitorMob) {
			if m := mobs.GetInstance(vId); m != nil {
				if !strings.EqualFold(m.Character.Name, targetName) {
					continue
				}
				if vStr > bestStrength {
					bestStrength = vStr
					bestExit = exitName
				}
			}
		}
	}
	return bestExit
}
