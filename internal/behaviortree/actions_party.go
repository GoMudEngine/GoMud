package behaviortree

// actions_party.go — party-coordination behavior tree actions for NPC parties.
//
// Each action looks up the caller's party via
// parties.GetByMobInstanceId(ctx.InstanceId). Actions return Failure if the
// caller isn't in a party (or other preconditions aren't met) so behavior
// tree selectors fall through to other branches gracefully.
//
// Movement: mob.Command("pathto <roomId>") is the standard engine path —
// the pathto mob command calls mapper.GetPath + mob.Path.SetPath, and the
// mob's walk loop follows the path over subsequent ticks. This is consistent
// with how MobIdle_HandleIdleMobs.go and go.go do mob navigation.
//
// Aggro mirroring: mob.Character.SetAggro copies both UserId and
// MobInstanceId from the leader's Aggro struct, keeping combat target parity.

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {
	actionRegistry["party_call_help"] = actPartyCallHelp
	actionRegistry["party_ensure_npc_party"] = actPartyEnsureNpcParty
	actionRegistry["party_respond_to_help"] = actPartyRespondToHelp
	actionRegistry["party_follow_leader"] = actPartyFollowLeader
	actionRegistry["party_assist_target"] = actPartyAssistTarget
	actionRegistry["party_flee_to_room"] = actPartyFleeToRoom
	actionRegistry["party_at_home_stand"] = actPartyAtHomeStand
}

// actPartyEnsureNpcParty creates or joins an NPC party on demand. Idempotent:
// if the calling mob is already in a party the action returns Success without
// any state change. This is intended to be the first node in every bandit-pack
// behavior tree so the party is guaranteed to exist before any subsequent
// party_* actions run.
//
// Params:
//
//	leader_mob_id (int) — mob template ID of the designated leader
//	home_room_id  (int) — room ID to use as the party's HomeRoomId
//
// Behavior:
//  1. Already in a real party → Success (no-op).
//  2. Already in an interim party AND real leader has now spawned →
//     dissolve interim, fall through to join real party.
//  3. Find the leader mob instance by scanning all loaded instances for a
//     matching MobId. If not found (not yet spawned), the calling mob becomes
//     the interim leader and creates the party itself, marked with
//     IntendedLeaderMobId so a later ensure call can consolidate.
//  4. If the leader instance exists, get-or-create its party and absorb any
//     interim parties whose IntendedLeaderMobId matches.
func actPartyEnsureNpcParty(params map[string]any, ctx *EvalContext) Result {
	leaderMobId := getIntParam(params, "leader_mob_id")
	homeRoomId := getIntParam(params, "home_room_id")
	if leaderMobId == 0 || homeRoomId == 0 {
		return Failure
	}

	// 1/2. Already in a party — usually a no-op, but check for the
	// interim-consolidation case.
	if existing := parties.GetByMobInstanceId(ctx.InstanceId); existing != nil {
		// Real party (no intended-leader marker, OR caller IS the real leader): no-op.
		if existing.IntendedLeaderMobId == 0 {
			return Success
		}
		// Interim party. Has the real leader spawned yet?
		realLeader := findMobInstanceByTemplateId(existing.IntendedLeaderMobId)
		if realLeader == nil {
			return Success // still waiting for the real leader to load
		}
		// Real leader is here — let consolidation fire below by falling through
		// to the real-leader branch. AbsorbInterimParties will pull us in.
	}

	// Find the calling mob's instance.
	callerMob := mobs.GetInstance(ctx.InstanceId)
	if callerMob == nil {
		return Failure
	}
	callerActor := &npcActor{mob: callerMob}

	// 3. Find the leader mob instance — scan all loaded instances.
	leaderMob := findMobInstanceByTemplateId(leaderMobId)
	if leaderMob == nil {
		// Leader not yet spawned; caller becomes interim leader.
		p := parties.NewByActor(callerActor)
		if p == nil {
			// Caller is already in a party (probably their own interim from a prior tick).
			return Success
		}
		p.HomeRoomId = homeRoomId
		p.IntendedLeaderMobId = leaderMobId
		return Success
	}

	leaderActor := &npcActor{mob: leaderMob}

	// 4. Get-or-create the party keyed on the leader.
	p := parties.GetByActor(leaderActor)
	if p == nil {
		p = parties.NewByActor(leaderActor)
		if p == nil {
			return Failure
		}
		p.HomeRoomId = homeRoomId
	}

	// Absorb any interim parties whose IntendedLeaderMobId matches — this
	// reconciles the startup-race case where followers self-formed before
	// the real leader's mob instance loaded.
	p.AbsorbInterimParties(leaderMobId)

	// If the caller IS the leader, NewByActor already added them.
	if callerMob.InstanceId == leaderMob.InstanceId {
		return Success
	}

	// Add the caller as a member (idempotent for already-present actors).
	if parties.GetByMobInstanceId(callerMob.InstanceId) != p {
		p.AddActor(callerActor)
	}
	return Success
}

// npcActor is a minimal partyActor adapter built directly inside
// actions_party.go to avoid importing actions (which imports parties,
// creating a cycle). It satisfies the partyActor interface via structural
// typing — the same interface that actions.MobActor satisfies.
type npcActor struct {
	mob *mobs.Mob
}

func (a *npcActor) IsPlayer() bool                             { return false }
func (a *npcActor) GetUserId() int                             { return 0 }
func (a *npcActor) GetMobInstanceId() int                      { return a.mob.InstanceId }
func (a *npcActor) GetCharacter() *characters.Character        { return &a.mob.Character }
func (a *npcActor) GetRoom() *rooms.Room                       { return rooms.LoadRoom(a.mob.Character.RoomId) }
func (a *npcActor) GetName() string                            { return a.mob.Character.Name }

// findMobInstanceByTemplateId scans all loaded mob instances and returns the
// first one whose MobId matches the given template ID. Returns nil if no
// matching instance is loaded.
func findMobInstanceByTemplateId(templateMobId int) *mobs.Mob {
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m != nil && int(m.MobId) == templateMobId {
			return m
		}
	}
	return nil
}

// actPartyCallHelp marks the caller's party as needing help in the caller's
// current room, then fires a PartyHelpRequested event so other party members'
// behavior trees can react and navigate toward the rally room.
func actPartyCallHelp(params map[string]any, ctx *EvalContext) Result {
	p := parties.GetByMobInstanceId(ctx.InstanceId)
	if p == nil {
		return Failure
	}
	p.HelpRoomId = ctx.RoomId
	p.HelpCallerInstanceId = ctx.InstanceId
	events.AddToQueue(events.PartyHelpRequested{
		PartyId:        p.PartyIdInternal(),
		CallerActorId:  ctx.InstanceId,
		CallerIsPlayer: false,
		RallyRoomId:    ctx.RoomId,
	})
	return Success
}

// actPartyRespondToHelp navigates the caller one step toward their party's
// HelpRoomId (set by party_call_help). On arrival at the rally room it
// engages a hostile-target player in the room synchronously by setting
// Aggro — putting the responder into combat THIS tick, before the engine's
// next-round wander-home check (which would otherwise immediately path
// MaxWander=0 mobs back to their home room).
//
// Returns Success when movement was initiated or aggro was set; Failure
// when no party / no help room / no engageable target.
func actPartyRespondToHelp(params map[string]any, ctx *EvalContext) Result {
	p := parties.GetByMobInstanceId(ctx.InstanceId)
	if p == nil || p.HelpRoomId == 0 {
		return Failure
	}
	if ctx.RoomId == p.HelpRoomId {
		// Arrived. Engage a hostile-target player in the room so the mob
		// enters combat and isn't pulled home by the wander check next round.
		if engageHostilePlayerInRoom(ctx.InstanceId, ctx.RoomId) {
			return Success
		}
		// No player to engage in the rally room — let the selector fall
		// through. If HelpRoomId is still set the caller will eventually
		// die or otherwise resolve, and the death handler clears the call.
		return Failure
	}
	if !moveMobTowardRoom(ctx.InstanceId, p.HelpRoomId) {
		return Failure
	}
	return Success
}

// engageHostilePlayerInRoom finds a player in the given room and sets the
// caller mob's Aggro to that player, but only if the mob is hostile and
// not already in combat. Synchronous (not queued via mob.Command) so the
// mob's Aggro is set before the engine's next-round wander check fires.
//
// Returns true if Aggro was set or the mob was already in combat.
func engageHostilePlayerInRoom(mobInstanceId int, roomId int) bool {
	mob := mobs.GetInstance(mobInstanceId)
	if mob == nil {
		return false
	}
	if mob.Character.IsInCombat() {
		return true // already engaged
	}
	if !mob.AutoAggro {
		return false
	}
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return false
	}
	for _, uid := range room.GetPlayers() {
		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}
		// Honor faction-rep peace gate — same as mobcommands.LookForTrouble.
		if factions.IsPeacefulToward(mob, u.UserId) {
			continue
		}
		mob.Character.SetAggro(uid, 0, characters.DefaultAttack)
		// SetAggro silently no-ops on grace-protected players; verify it stuck.
		if mob.Character.IsInCombat() {
			return true
		}
	}
	return false
}

// actPartyFollowLeader navigates the caller one step toward the leader's
// current room. Returns Success if already there or if movement was
// initiated; Failure if no party / no leader / no path.
//
// Chunk 3.8: when the caller has an active PatrolId (e.g., Lars running
// his runner-circuit oneshot patrol while Ketil dwells at the depot),
// the patrol executor is the source of truth for movement — return
// Success without queueing a pathto-leader command that would
// overwrite the patrol's pathto-vendor command in the same round.
func actPartyFollowLeader(params map[string]any, ctx *EvalContext) Result {
	self := mobs.GetInstance(ctx.InstanceId)
	if self != nil && self.PatrolId != "" {
		return Success // patrol layer owns movement; party-follow defers
	}
	p := parties.GetByMobInstanceId(ctx.InstanceId)
	if p == nil || p.Leader == nil {
		return Failure
	}
	leaderRoom := p.Leader.GetRoom()
	if leaderRoom == nil {
		return Failure
	}
	leaderRoomId := leaderRoom.RoomId
	if ctx.RoomId == leaderRoomId {
		return Success
	}
	if !moveMobTowardRoom(ctx.InstanceId, leaderRoomId) {
		return Failure
	}
	return Success
}

// actPartyAssistTarget makes the caller target whatever the leader is
// targeting. Returns Failure if leader isn't in combat or caller isn't
// in a party.
func actPartyAssistTarget(params map[string]any, ctx *EvalContext) Result {
	p := parties.GetByMobInstanceId(ctx.InstanceId)
	if p == nil || p.Leader == nil {
		return Failure
	}
	leaderChar := p.Leader.GetCharacter()
	if leaderChar == nil || !leaderChar.IsInCombat() {
		return Failure
	}
	self := mobs.GetInstance(ctx.InstanceId)
	if self == nil {
		return Failure
	}
	self.Character.SetAggro(
		leaderChar.CurrentCombatTarget().UserId,
		leaderChar.CurrentCombatTarget().MobInstanceId,
		characters.DefaultAttack,
	)
	return Success
}

// actPartyFleeToRoom moves all mob members of the caller's party one step
// toward the target room. Typically triggered by the leader when retreating.
// Param: room_id (int) — the destination room.
func actPartyFleeToRoom(params map[string]any, ctx *EvalContext) Result {
	p := parties.GetByMobInstanceId(ctx.InstanceId)
	if p == nil {
		return Failure
	}
	targetRoomId := getIntParam(params, "room_id")
	if targetRoomId == 0 {
		return Failure
	}
	movedAny := false
	for _, member := range p.Members {
		// Only mob members can be moved via btree pathfinding; player
		// members move themselves via player commands.
		mobInstId := member.GetMobInstanceId()
		if mobInstId == 0 {
			continue
		}
		memberRoom := member.GetRoom()
		if memberRoom == nil {
			continue
		}
		if memberRoom.RoomId == targetRoomId {
			continue
		}
		if moveMobTowardRoom(mobInstId, targetRoomId) {
			movedAny = true
		}
	}
	if !movedAny {
		return Failure
	}
	return Success
}

// actPartyAtHomeStand sets a behavior-state flag on the caller to suppress
// flee branches. Only fires when the caller is at the party's HomeRoomId.
// Other branches can gate on state key "party_standing" == "true".
func actPartyAtHomeStand(params map[string]any, ctx *EvalContext) Result {
	p := parties.GetByMobInstanceId(ctx.InstanceId)
	if p == nil || p.HomeRoomId == 0 {
		return Failure
	}
	if ctx.RoomId != p.HomeRoomId {
		return Failure
	}
	if ctx.MobState != nil {
		ctx.MobState.Set("party_standing", "true")
	}
	return Success
}

// moveMobTowardRoom issues a "pathto <roomId>" command to the given mob
// instance, which causes the engine's pathfinder (mobcommands/pathto.go)
// to compute a route via mapper.GetPath and store it in mob.Path. The
// mob's walk loop then follows the path over subsequent ticks.
//
// Returns true if the mob was found and the command was issued. Returns
// false if the mob instance does not exist.
//
// NOTE: this returns true as long as the command was issued — it does NOT
// guarantee the path exists. If no path is found, pathto marks the mob as
// "lost" (same as normal pathto home failure handling). Callers that need
// path-guarantee should check HelpRoomId / LeaderRoom reachability separately.
func moveMobTowardRoom(mobInstanceId int, targetRoomId int) bool {
	mob := mobs.GetInstance(mobInstanceId)
	if mob == nil {
		return false
	}
	mob.Command(fmt.Sprintf("pathto %d", targetRoomId))
	return true
}
