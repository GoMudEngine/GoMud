package parties

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// nextPartyId is a monotonically increasing counter used to assign
// unique numeric IDs to Party instances for event payloads.
var nextPartyId int = 1

// partyActor is the local interface for actor-based Party operations.
// It extends actorIdentity (IsPlayer/GetUserId/GetMobInstanceId) with the
// wider set of methods Party operations will need. Using a local interface
// avoids the import cycle with internal/actions, which already imports
// internal/parties. Any actions.Actor satisfies partyActor via Go's
// structural typing.
type partyActor interface {
	actorIdentity // IsPlayer, GetUserId, GetMobInstanceId

	// GetCharacter returns the underlying character data.
	GetCharacter() *characters.Character

	// GetRoom returns the room the actor currently occupies.
	GetRoom() *rooms.Room

	// GetName returns the display name of the actor.
	GetName() string
}

type Party struct {
	// ── Legacy player-only fields (kept for backward compat) ──
	LeaderUserId  int
	UserIds       []int
	InviteUserIds []int
	AutoAttackers []int
	Position      map[int]string

	// ── New actor-based fields (Stage 1 NPC support) ──
	Leader             partyActor
	Members            []partyActor
	Invitees           []partyActor
	AutoAttackerActors []partyActor
	ActorPosition      map[ActorKey]string

	// ── NPC party state ──
	HomeRoomId int // 0 if none designated; for party_at_home_stand
	HelpRoomId int // 0 if no active call; rally room when set

	// ── Internal bookkeeping ──
	partyId int // lazy-assigned numeric ID; see PartyIdInternal()
}

var (
	// Existing player-keyed registry (kept as-is for backward compat).
	// Key is leader user id, value is party.
	partyMap = map[int]*Party{}

	// New actor-keyed registry. Both player AND NPC parties live here;
	// player parties get DOUBLE-registered (in partyMap by UserId, here
	// by ActorKey) for the duration of Stage 1.
	actorPartyMap = map[ActorKey]*Party{}
)

// NewByActor creates a new party with the given actor as leader. Returns
// nil if the actor is already in another party. Player and NPC parties
// share the same registry (actorPartyMap).
func NewByActor(leader partyActor) *Party {
	key := ActorKeyFor(leader)
	if _, ok := actorPartyMap[key]; ok {
		return nil
	}
	p := &Party{
		Leader:             leader,
		Members:            []partyActor{leader},
		Invitees:           []partyActor{},
		AutoAttackerActors: []partyActor{},
		ActorPosition:      map[ActorKey]string{},
	}
	actorPartyMap[key] = p
	// For player leaders, also populate the legacy UserId-keyed registry
	// so existing code paths continue to find the party.
	if leader.IsPlayer() {
		uid := leader.GetUserId()
		p.LeaderUserId = uid
		p.UserIds = []int{uid}
		p.InviteUserIds = []int{}
		p.AutoAttackers = []int{}
		p.Position = map[int]string{}
		partyMap[uid] = p
	}
	return p
}

// GetByActor returns the party containing the given actor (as leader,
// member, or invitee), or nil if not found.
func GetByActor(a partyActor) *Party {
	key := ActorKeyFor(a)
	if p, ok := actorPartyMap[key]; ok {
		return p
	}
	return nil
}

func New(userId int) *Party {
	if _, ok := partyMap[userId]; ok {
		return nil
	}
	p := &Party{
		LeaderUserId:  userId,
		UserIds:       []int{userId},
		InviteUserIds: []int{},
		AutoAttackers: []int{},
		Position:      map[int]string{},
	}
	partyMap[userId] = p
	return p
}

func Get(userId int) *Party {
	if party, ok := partyMap[userId]; ok {
		return party
	}
	return nil
}

func (p *Party) ChanceToBeTargetted(userId int) int {

	rank := p.GetRank(userId)
	if rank == `front` {
		return 2
	}

	if rank == `back` {
		return 0
	}
	// middle rank
	return 1
}

func (p *Party) GetRank(userId int) string {
	if val, ok := p.Position[userId]; ok {
		return val
	}
	return `middle`
}

func (p *Party) SetRank(userId int, rank string) {
	if rank == `front` || rank == `back` {
		p.Position[userId] = rank
		return
	}

	delete(p.Position, userId)
}

func (p *Party) IsLeader(userId int) bool {
	return p.LeaderUserId == userId
}

func (p *Party) New(userId int) *Party {
	if party, ok := partyMap[userId]; ok {
		return party
	}
	return nil
}

func (p *Party) SetAutoAttack(userId int, on bool) bool {

	if on {
		for _, id := range p.AutoAttackers {
			if id == userId {
				return true
			}
		}
		p.AutoAttackers = append(p.AutoAttackers, userId)
		return false
	}

	for i, id := range p.AutoAttackers {
		if id == userId {
			p.AutoAttackers = append(p.AutoAttackers[:i], p.AutoAttackers[i+1:]...)
			return true
		}
	}
	return false
}

func (p *Party) GetAutoAttackUserIds() []int {
	return append([]int{}, p.AutoAttackers...)
}

func (p *Party) Leave(userId int) bool {
	if p.IsLeader(userId) {
		if len(p.UserIds) == 1 {
			p.Disband()
			return true
		}

		for _, id := range p.UserIds {
			if id != userId {
				p.LeaderUserId = id
				break
			}
		}
	}

	for i, id := range p.UserIds {
		if id == userId {
			p.UserIds = append(p.UserIds[:i], p.UserIds[i+1:]...)
			break
		}
	}

	delete(partyMap, userId)

	return true
}

func (p *Party) IsMember(userId int) bool {
	for _, id := range p.UserIds {
		if id == userId {
			return true
		}
	}
	return false
}

func (p *Party) Invited(userId int) bool {
	for _, id := range p.InviteUserIds {
		if id == userId {
			return true
		}
	}
	return false
}

func (p *Party) InvitePlayer(userId int) bool {
	if _, ok := partyMap[userId]; ok {
		return false
	}
	p.InviteUserIds = append(p.InviteUserIds, userId)
	partyMap[userId] = p

	return true
}

func (p *Party) AcceptInvite(userId int) bool {
	if !p.Invited(userId) {
		return false
	}

	p.UserIds = append(p.UserIds, userId)

	for idx, uid := range p.InviteUserIds {
		if uid == userId {
			p.InviteUserIds = append(p.InviteUserIds[:idx], p.InviteUserIds[idx+1:]...)
			break
		}
	}
	return true
}

func (p *Party) DeclineInvite(userId int) bool {
	if !p.Invited(userId) {
		return false
	}

	for idx, uid := range p.InviteUserIds {
		if uid == userId {
			p.InviteUserIds = append(p.InviteUserIds[:idx], p.InviteUserIds[idx+1:]...)
			break
		}
	}

	delete(partyMap, userId)

	return true
}

func (p *Party) Disband() {
	for _, userId := range p.UserIds {
		delete(partyMap, userId)
	}
	for _, userId := range p.InviteUserIds {
		delete(partyMap, userId)
	}
}

func (p *Party) GetMembers() []int {
	return append([]int{}, p.UserIds...)
}

func (p *Party) GetInvited() []int {
	return append([]int{}, p.InviteUserIds...)
}

// AddActor adds an actor to the party as a member. Returns false if the
// actor is already in another party.
func (p *Party) AddActor(a partyActor) bool {
	key := ActorKeyFor(a)
	if _, ok := actorPartyMap[key]; ok {
		return false
	}
	p.Members = append(p.Members, a)
	actorPartyMap[key] = p
	if a.IsPlayer() {
		uid := a.GetUserId()
		p.UserIds = append(p.UserIds, uid)
		partyMap[uid] = p
	}
	return true
}

// RemoveActor removes an actor from the party (members, invitees, or
// auto-attackers). Returns true if the actor was found and removed.
func (p *Party) RemoveActor(a partyActor) bool {
	key := ActorKeyFor(a)
	if _, ok := actorPartyMap[key]; !ok {
		return false
	}
	delete(actorPartyMap, key)
	for i, m := range p.Members {
		if ActorKeyFor(m) == key {
			p.Members = append(p.Members[:i], p.Members[i+1:]...)
			break
		}
	}
	for i, m := range p.Invitees {
		if ActorKeyFor(m) == key {
			p.Invitees = append(p.Invitees[:i], p.Invitees[i+1:]...)
			break
		}
	}
	for i, m := range p.AutoAttackerActors {
		if ActorKeyFor(m) == key {
			p.AutoAttackerActors = append(p.AutoAttackerActors[:i], p.AutoAttackerActors[i+1:]...)
			break
		}
	}
	delete(p.ActorPosition, key)
	if a.IsPlayer() {
		uid := a.GetUserId()
		delete(partyMap, uid)
		for i, id := range p.UserIds {
			if id == uid {
				p.UserIds = append(p.UserIds[:i], p.UserIds[i+1:]...)
				break
			}
		}
	}
	return true
}

// PartyIdInternal returns the party's internal numeric ID. The ID is
// lazy-assigned on first call using a package-level counter. Used by
// behavior tree actions and events that need a stable party reference.
func (p *Party) PartyIdInternal() int {
	if p.partyId == 0 {
		p.partyId = nextPartyId
		nextPartyId++
	}
	return p.partyId
}

// Dissolve removes all members, invitees, and auto-attackers from both
// registries and fires a PartyDissolved event so that member behavior
// trees can react before reverting to solo behavior.
func (p *Party) Dissolve(reason string) {
	// Collect member actor IDs before clearing slices.
	memberIds := make([]int, 0, len(p.Members))
	for _, a := range p.Members {
		if a.IsPlayer() {
			memberIds = append(memberIds, a.GetUserId())
		} else {
			memberIds = append(memberIds, a.GetMobInstanceId())
		}
	}

	events.AddToQueue(events.PartyDissolved{
		PartyId:        p.PartyIdInternal(),
		Reason:         reason,
		MemberActorIds: memberIds,
	})

	for _, a := range p.Members {
		delete(actorPartyMap, ActorKeyFor(a))
		if a.IsPlayer() {
			delete(partyMap, a.GetUserId())
		}
	}
	for _, a := range p.Invitees {
		delete(actorPartyMap, ActorKeyFor(a))
		if a.IsPlayer() {
			delete(partyMap, a.GetUserId())
		}
	}
	p.Members = nil
	p.Invitees = nil
	p.AutoAttackerActors = nil
}
