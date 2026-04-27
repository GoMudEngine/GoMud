# NPC Party System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the existing `internal/parties` package to support NPC actors as first-class party members and leaders, add behavior tree primitives for group coordination, and refactor the bandit pack as the smoke-test consumer.

**Architecture:** Adapter pattern — introduce `actions.Actor`-based APIs as the new primary path; keep existing `UserId int`-based public APIs as thin compatibility shims that delegate to the actor APIs. Existing 20+ consumer files are unaffected. New btree primitives (6 actions, 5 conditions, 2 events) live in new files. Bandit pack migrates to use the new system as the validation case.

**Tech Stack:** Go 1.x, standard `testing` package. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-04-27-npc-party-system-design.md`

---

## Task 1: Scout — bandit camp room + existing bandit btree state

**Files:**
- Read-only scan: `_datafiles/world/dogmud/rooms/north_road/`, `_datafiles/world/dogmud/rooms/marches_spur_road/`
- Read-only scan: `_datafiles/world/dogmud/behaviors/north_road/{283,284,285,286}*.yaml` (if they exist)
- Read-only: `internal/hooks/MobDeath_PackFlee.go`

- [ ] **Step 1: Find the bandit camp room ID**

Run:
```bash
grep -rln "bandit camp\|drainage ditch\|soren" "C:/Users/Calabe Davis/workspace/DOGMud/_datafiles/world/dogmud/rooms/" | head -5
```

Read each candidate room file. Identify the room whose title or description matches "bandit camp" (the camp Soren's pack retreats to per quest 17). Record the room ID. Also identify the lookout's home room (where the lookout typically spawns; bandits respond to player_enter here).

- [ ] **Step 2: Inventory existing bandit behavior tree files**

Run:
```bash
ls "C:/Users/Calabe Davis/workspace/DOGMud/_datafiles/world/dogmud/behaviors/north_road/" 2>&1 | grep -E "28[3-6]"
ls "C:/Users/Calabe Davis/workspace/DOGMud/_datafiles/world/dogmud/behaviors/marches_spur_road/" 2>&1 | grep -E "25[3-4]"
```

Record which bandit mobs already have per-mob btree files vs which rely on the archetype + ad-hoc hooks (`MobDeath_PackFlee.go`).

- [ ] **Step 3: Read MobDeath_PackFlee.go**

Read the entire file. Note exactly what it does: which mob groups trigger pack flee, what conditions, what destinations. This becomes either deleted or migrated in Task 15.

- [ ] **Step 4: Record findings in plan-task-scratch.md (or a comment in this plan)**

Create a short scratchpad with: bandit camp room ID, lookout home room ID, list of existing bandit btree files, summary of MobDeath_PackFlee.go behavior. The implementation tasks below reference these.

---

## Task 2: ActorKey helper + tests

**Files:**
- Create: `internal/parties/actorkey.go`
- Create: `internal/parties/actorkey_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/parties/actorkey_test.go
package parties

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
)

// fakeActor is a minimal actions.Actor for tests that don't need the full
// users/mobs setup. It satisfies the IsPlayer / GetUserId / GetMobInstanceId
// methods used by ActorKey.
type fakeActor struct {
	userId        int
	mobInstanceId int
}

func (f *fakeActor) GetUserId() int                                  { return f.userId }
func (f *fakeActor) GetMobInstanceId() int                           { return f.mobInstanceId }
func (f *fakeActor) IsPlayer() bool                                  { return f.userId > 0 }
// other Actor methods left as panicking stubs; tests only use the three above.
func (f *fakeActor) GetCharacter() interface{}                       { panic("not used") }
func (f *fakeActor) GetRoom() interface{}                            { panic("not used") }
func (f *fakeActor) SendText(msg string)                             { panic("not used") }
func (f *fakeActor) SendRoomText(msg string, excludeSelf bool)       { panic("not used") }
func (f *fakeActor) SendRoomCommunication(msg string, exclSelf bool) { panic("not used") }
func (f *fakeActor) GetName() string                                 { return "fake" }
func (f *fakeActor) AddBuff(buffId int, source string)               { panic("not used") }
func (f *fakeActor) OnSkillUse(skillName string) bool                { return false }
func (f *fakeActor) OnStatUse(statName string) bool                  { return false }
func (f *fakeActor) OnCriticalSuccess(skillName string)              { panic("not used") }
func (f *fakeActor) OnCriticalFailure(skillName string)              { panic("not used") }

func TestActorKey_PlayerActor(t *testing.T) {
	a := &fakeActor{userId: 42}
	key := ActorKeyFor(a)
	if string(key) != "user:42" {
		t.Errorf("got %q, want %q", key, "user:42")
	}
}

func TestActorKey_MobActor(t *testing.T) {
	a := &fakeActor{mobInstanceId: 1234}
	key := ActorKeyFor(a)
	if string(key) != "mob:1234" {
		t.Errorf("got %q, want %q", key, "mob:1234")
	}
}

func TestActorKey_DistinctForUserAndMob(t *testing.T) {
	user := &fakeActor{userId: 1}
	mob := &fakeActor{mobInstanceId: 1}
	if ActorKeyFor(user) == ActorKeyFor(mob) {
		t.Error("user and mob with same numeric id should produce distinct keys")
	}
}
```

NOTE: the fake Actor's `GetCharacter` and `GetRoom` return `interface{}` — since we can't import the real concrete types from a test stub easily, this test doesn't exercise those methods. The actions.Actor interface has those return concrete types; the implementation will need to either embed a real test fixture or use a different test approach. **Implementation note:** if the simple fakeActor above doesn't compile because of return-type mismatches with the interface, replace with an embedded `actions.UserActor` or `actions.MobActor` stub from the existing test infrastructure (`internal/actions/actor_user.go`, `actor_mob.go`).

- [ ] **Step 2: Run test to verify it fails**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/parties/...
```
Expected: FAIL with "undefined: ActorKeyFor" or "undefined: ActorKey".

- [ ] **Step 3: Write the implementation**

```go
// internal/parties/actorkey.go
package parties

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
)

// ActorKey is an opaque map-key derived from an Actor. It encodes which
// concrete actor type the key represents (user or mob) and the underlying
// numeric ID, so a user with ID 42 and a mob instance with ID 42 produce
// distinct keys.
type ActorKey string

// ActorKeyFor returns the ActorKey for an Actor. Player actors yield
// "user:<userId>"; mob actors yield "mob:<mobInstanceId>".
func ActorKeyFor(a actions.Actor) ActorKey {
	if a == nil {
		return ""
	}
	if a.IsPlayer() {
		return ActorKey(fmt.Sprintf("user:%d", a.GetUserId()))
	}
	return ActorKey(fmt.Sprintf("mob:%d", a.GetMobInstanceId()))
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/parties/...
```
Expected: PASS.

- [ ] **Step 5: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 3: Refactor Party struct internals + add Actor-based slice + map

**Files:**
- Modify: `internal/parties/parties.go`

- [ ] **Step 1: Add Actor-based fields to Party (alongside existing UserId fields)**

Edit `internal/parties/parties.go`. Replace the Party struct definition with:

```go
type Party struct {
	// ── Legacy player-only fields (kept for backward compat) ──
	LeaderUserId  int
	UserIds       []int
	InviteUserIds []int
	AutoAttackers []int
	Position      map[int]string

	// ── New actor-based fields (Stage 1 NPC support) ──
	Leader        actions.Actor
	Members       []actions.Actor
	Invitees      []actions.Actor
	AutoAttackerActors []actions.Actor
	ActorPosition map[ActorKey]string

	// ── NPC party state ──
	HomeRoomId int  // 0 if none designated; for party_at_home_stand
	HelpRoomId int  // 0 if no active call; rally room when set
}
```

Add the `actions` import.

- [ ] **Step 2: Add a parallel actor-keyed registry alongside partyMap**

Below the existing `partyMap` declaration, add:

```go
var (
	// Existing player-keyed registry (kept as-is for backward compat)
	partyMap = map[int]*Party{}

	// New actor-keyed registry. Both player AND NPC parties live here;
	// player parties get DOUBLE-registered (in partyMap by UserId, here
	// by ActorKey) for the duration of Stage 1.
	actorPartyMap = map[ActorKey]*Party{}
)
```

- [ ] **Step 3: Build to verify no compile errors**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean (no consumers reference the new fields yet).

- [ ] **Step 4: Run existing tests to confirm zero behavioral change**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/parties/... ./internal/usercommands/... ./internal/hooks/...
```
Expected: PASS (same as before — we only added fields).

---

## Task 4: New Actor-based public APIs (NewByActor, GetByActor, GetByMember)

**Files:**
- Modify: `internal/parties/parties.go`
- Modify: `internal/parties/parties_test.go` (CREATE if not exists)

- [ ] **Step 1: Write the failing tests**

```go
// internal/parties/parties_test.go (create or extend)
package parties

import (
	"testing"
)

func TestNewByActor_CreatesParty(t *testing.T) {
	// Use the fakeActor from actorkey_test.go (same package).
	leader := &fakeActor{mobInstanceId: 100}
	p := NewByActor(leader)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	if p.Leader != leader {
		t.Error("Leader not set to provided actor")
	}
	if len(p.Members) != 1 || p.Members[0] != leader {
		t.Error("Leader not added to Members")
	}
}

func TestNewByActor_ReturnsNilIfActorAlreadyInParty(t *testing.T) {
	leader := &fakeActor{mobInstanceId: 200}
	first := NewByActor(leader)
	if first == nil {
		t.Fatal("first NewByActor unexpectedly nil")
	}
	second := NewByActor(leader)
	if second != nil {
		t.Error("second NewByActor for same actor should return nil")
	}
}

func TestGetByActor_ReturnsParty(t *testing.T) {
	leader := &fakeActor{mobInstanceId: 300}
	p := NewByActor(leader)
	got := GetByActor(leader)
	if got != p {
		t.Errorf("GetByActor returned wrong party")
	}
}

func TestGetByActor_NilForUnknown(t *testing.T) {
	a := &fakeActor{mobInstanceId: 9999}
	if GetByActor(a) != nil {
		t.Error("GetByActor should return nil for actor not in any party")
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/parties/...
```
Expected: FAIL with "undefined: NewByActor" / "undefined: GetByActor".

- [ ] **Step 3: Implement NewByActor and GetByActor**

In `internal/parties/parties.go`, add:

```go
// NewByActor creates a new party with the given actor as leader. Returns
// nil if the actor is already in another party. Player and NPC parties
// share the same registry (actorPartyMap).
func NewByActor(leader actions.Actor) *Party {
	key := ActorKeyFor(leader)
	if _, ok := actorPartyMap[key]; ok {
		return nil
	}
	p := &Party{
		Leader:        leader,
		Members:       []actions.Actor{leader},
		Invitees:      []actions.Actor{},
		AutoAttackerActors: []actions.Actor{},
		ActorPosition: map[ActorKey]string{},
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
func GetByActor(a actions.Actor) *Party {
	key := ActorKeyFor(a)
	if p, ok := actorPartyMap[key]; ok {
		return p
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/parties/...
```
Expected: PASS.

- [ ] **Step 5: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 5: AddActor / RemoveActor / Dissolve APIs

**Files:**
- Modify: `internal/parties/parties.go`
- Modify: `internal/parties/parties_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/parties/parties_test.go`:

```go
func TestAddActor_AddsToMembers(t *testing.T) {
	leader := &fakeActor{mobInstanceId: 400}
	p := NewByActor(leader)
	member := &fakeActor{mobInstanceId: 401}
	if !p.AddActor(member) {
		t.Error("AddActor returned false for new member")
	}
	if len(p.Members) != 2 {
		t.Errorf("Members len = %d, want 2", len(p.Members))
	}
	if GetByActor(member) != p {
		t.Error("member's GetByActor lookup did not return party")
	}
}

func TestAddActor_RejectsActorAlreadyInOtherParty(t *testing.T) {
	leaderA := &fakeActor{mobInstanceId: 410}
	leaderB := &fakeActor{mobInstanceId: 411}
	a := NewByActor(leaderA)
	NewByActor(leaderB)
	other := &fakeActor{mobInstanceId: 412}
	a.AddActor(other)
	// Try adding `other` to the second party — should fail.
	b := GetByActor(leaderB)
	if b.AddActor(other) {
		t.Error("AddActor should reject an actor already in another party")
	}
}

func TestRemoveActor_RemovesFromMembers(t *testing.T) {
	leader := &fakeActor{mobInstanceId: 420}
	member := &fakeActor{mobInstanceId: 421}
	p := NewByActor(leader)
	p.AddActor(member)
	if !p.RemoveActor(member) {
		t.Error("RemoveActor returned false for present member")
	}
	if len(p.Members) != 1 {
		t.Errorf("Members len after remove = %d, want 1", len(p.Members))
	}
	if GetByActor(member) != nil {
		t.Error("member should no longer be tracked in any party")
	}
}

func TestDissolve_RemovesAllMembersFromRegistry(t *testing.T) {
	leader := &fakeActor{mobInstanceId: 430}
	m1 := &fakeActor{mobInstanceId: 431}
	m2 := &fakeActor{mobInstanceId: 432}
	p := NewByActor(leader)
	p.AddActor(m1)
	p.AddActor(m2)
	p.Dissolve("test")
	if GetByActor(leader) != nil || GetByActor(m1) != nil || GetByActor(m2) != nil {
		t.Error("Dissolve did not remove all members from registry")
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/parties/...
```
Expected: FAIL with undefined methods.

- [ ] **Step 3: Implement the three methods**

In `internal/parties/parties.go`, add:

```go
// AddActor adds an actor to the party as a member. Returns false if the
// actor is already in another party.
func (p *Party) AddActor(a actions.Actor) bool {
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
func (p *Party) RemoveActor(a actions.Actor) bool {
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

// Dissolve removes all members, invitees, and auto-attackers from both
// registries. Fires the party_dissolved event for member awareness
// (the event type is added in Task 6; this call is wired up there).
func (p *Party) Dissolve(reason string) {
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
	// party_dissolved event fired in Task 6 — added here once the
	// event type exists.
	p.Members = nil
	p.Invitees = nil
	p.AutoAttackerActors = nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/parties/...
```
Expected: PASS.

- [ ] **Step 5: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 6: New events — party_help_requested + party_dissolved

**Files:**
- Modify: `internal/events/eventtypes.go`
- Modify: `internal/parties/parties.go` (wire Dissolve to fire event)

- [ ] **Step 1: Read the existing eventtypes.go to understand the pattern**

```bash
head -80 "C:/Users/Calabe Davis/workspace/DOGMud/internal/events/eventtypes.go"
```

Identify how existing events like `Input`, `Message` are declared as struct types.

- [ ] **Step 2: Add the two new event types**

In `internal/events/eventtypes.go`, add (location: with other event struct definitions):

```go
// PartyHelpRequested is fired when a party member calls for help via
// the party_call_help behavior tree action. Other party members'
// behavior trees can listen for this event to navigate to the
// rally room.
type PartyHelpRequested struct {
	PartyId       int           // simple incrementing ID; see parties package
	CallerActorId int           // user or mob instance ID; type from CallerIsPlayer
	CallerIsPlayer bool
	RallyRoomId   int
}

// Type satisfies the events.Event interface (returns a stable name).
func (PartyHelpRequested) Type() string { return "PartyHelpRequested" }

// PartyDissolved is fired when a party's leader dies or the party is
// explicitly disbanded. Member behavior trees can listen for this to
// react ("morale break" emote, panic flee, etc.) before reverting to
// solo behavior.
type PartyDissolved struct {
	PartyId  int
	Reason   string  // "leader_died" | "disbanded" | "all_dead"
	MemberActorIds []int
}

func (PartyDissolved) Type() string { return "PartyDissolved" }
```

NOTE on the `Type() string` method: read the existing events file to confirm the event-interface pattern. If existing events use a different conformance pattern (embedding a base type, registering with an init function, etc.), match that pattern instead.

- [ ] **Step 3: Wire Dissolve to fire the event**

In `internal/parties/parties.go`, modify Dissolve to fire `events.PartyDissolved`:

```go
import "github.com/GoMudEngine/GoMud/internal/events"

// (modify existing Dissolve to add this just before clearing Members)

func (p *Party) Dissolve(reason string) {
	memberIds := make([]int, 0, len(p.Members))
	for _, a := range p.Members {
		if a.IsPlayer() {
			memberIds = append(memberIds, a.GetUserId())
		} else {
			memberIds = append(memberIds, a.GetMobInstanceId())
		}
	}

	events.AddToQueue(events.PartyDissolved{
		PartyId:        p.id(),  // see Step 4 below for adding p.id()
		Reason:         reason,
		MemberActorIds: memberIds,
	})

	// existing cleanup logic...
	for _, a := range p.Members { /* existing */ }
	// ... etc
}
```

- [ ] **Step 4: Add a Party.id() helper**

The PartyDissolved event needs a stable ID. Add a global counter in `internal/parties/parties.go`:

```go
var nextPartyId int = 1

func (p *Party) id() int {
	if p.partyId == 0 {
		p.partyId = nextPartyId
		nextPartyId++
	}
	return p.partyId
}
```

Add `partyId int` to the Party struct (unexported field).

- [ ] **Step 5: Run tests**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/parties/... ./internal/events/...
```
Expected: PASS (existing dissolution test now also fires the event; existing test only checks the registry side which still works).

- [ ] **Step 6: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 7: BTree action — party_call_help

**Files:**
- Create: `internal/behaviortree/actions_party.go`
- Create: `internal/behaviortree/actions_party_test.go`

- [ ] **Step 1: Read existing action registration pattern**

Read one existing action file to understand the registration pattern:

```bash
head -50 "C:/Users/Calabe Davis/workspace/DOGMud/internal/behaviortree/actions_room.go"
```

Note how actions are registered, how they receive context, how they return Status.

- [ ] **Step 2: Write the failing test**

```go
// internal/behaviortree/actions_party_test.go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/parties"
)

func TestPartyCallHelp_SetsHelpRoomIdAndFiresEvent(t *testing.T) {
	// Setup: create an NPC party with a fake mob actor as leader and
	// another as member. Use the existing test infrastructure for
	// mobs/users (see actions_test.go's requireUser pattern, plus the
	// equivalent for mobs).

	// ... setup omitted; implementation will reference existing test
	// fixtures from actions_test.go and the mobs test seed data.

	// Fire the action via the registry.
	leader := /* ... */
	p := parties.NewByActor(leader)
	// ... add a member ...

	ctx := /* construct EventContext with Caller = a member of p,
	         RoomId = the caller's current room ... */

	action := LookupAction("party_call_help")
	if action == nil {
		t.Fatal("party_call_help not registered")
	}
	status := action(ctx)
	if status != Success {
		t.Errorf("got status %v, want Success", status)
	}
	if p.HelpRoomId != ctx.RoomId {
		t.Errorf("HelpRoomId not set; got %d want %d", p.HelpRoomId, ctx.RoomId)
	}

	// Verify the PartyHelpRequested event was queued.
	// (Use event-capture helpers from actions_test.go.)
}
```

NOTE: This test is sketched — the implementation will need to use the actual test seed mobs / users / actor adapters from the existing test infrastructure (`actions_test.go` + `mobs_test.go`). If concrete fixtures don't exist for an NPC-led party, the test may need to add them or use a stub Actor with `actions.MobActor` wrapping a real seeded mob.

- [ ] **Step 3: Run test to verify it fails**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: FAIL with `LookupAction("party_call_help") == nil` or compilation error.

- [ ] **Step 4: Implement the action**

```go
// internal/behaviortree/actions_party.go
package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/parties"
)

func init() {
	RegisterAction("party_call_help", actionPartyCallHelp)
}

// actionPartyCallHelp marks the calling actor's party as needing help
// at the caller's current room. Fires events.PartyHelpRequested for
// other party members to react to in their own behavior trees.
//
// The caller (Actor) is taken from ctx.Actor. If the caller isn't in
// a party, the action returns Failure.
func actionPartyCallHelp(ctx EventContext) Status {
	caller := ctx.Actor
	if caller == nil {
		return Failure
	}
	p := parties.GetByActor(caller)
	if p == nil {
		return Failure
	}
	p.HelpRoomId = ctx.RoomId

	callerId := caller.GetUserId()
	isPlayer := caller.IsPlayer()
	if !isPlayer {
		callerId = caller.GetMobInstanceId()
	}

	events.AddToQueue(events.PartyHelpRequested{
		PartyId:        p.PartyIdInternal(), // see step 5
		CallerActorId:  callerId,
		CallerIsPlayer: isPlayer,
		RallyRoomId:    ctx.RoomId,
	})
	return Success
}
```

NOTE: the action signature `func(ctx EventContext) Status` and the registry `RegisterAction` are placeholder names — match the actual existing pattern in `actions_room.go` / `actions.go`. Adjust as needed during implementation. Same for `Status` enum (`Success`/`Failure`/`Running`).

- [ ] **Step 5: Expose Party ID externally**

Add to `internal/parties/parties.go`:

```go
// PartyIdInternal returns the party's internal numeric ID. Used by
// behavior tree actions that emit events referring to the party.
func (p *Party) PartyIdInternal() int {
	return p.id()
}
```

- [ ] **Step 6: Run test to verify it passes**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: PASS.

- [ ] **Step 7: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 8: BTree action — party_respond_to_help

**Files:**
- Modify: `internal/behaviortree/actions_party.go`
- Modify: `internal/behaviortree/actions_party_test.go`

- [ ] **Step 1: Write the failing test**

Add to `actions_party_test.go`:

```go
func TestPartyRespondToHelp_MovesActorTowardRallyRoom(t *testing.T) {
	// Setup: create NPC party where leader is in room A and member
	// is in room B (different); set party.HelpRoomId = room A.
	// Fire party_respond_to_help on the member.
	// Assert: the member's room changes to a room one step closer
	// to room A (uses the engine's room navigation).

	// ... setup omitted; reference existing test fixtures.

	action := LookupAction("party_respond_to_help")
	if action == nil {
		t.Fatal("party_respond_to_help not registered")
	}
	status := action(ctx)
	if status != Success {
		t.Errorf("got status %v, want Success", status)
	}
	// Assert the actor is now in a room closer to HelpRoomId.
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: FAIL.

- [ ] **Step 3: Implement the action**

In `internal/behaviortree/actions_party.go`, add:

```go
func init() {
	// Existing init() entries plus:
	RegisterAction("party_respond_to_help", actionPartyRespondToHelp)
}

// actionPartyRespondToHelp navigates the caller toward the party's
// current HelpRoomId. If already in that room, returns Success without
// movement. If no party, no help-room set, or no path exists,
// returns Failure.
func actionPartyRespondToHelp(ctx EventContext) Status {
	caller := ctx.Actor
	if caller == nil {
		return Failure
	}
	p := parties.GetByActor(caller)
	if p == nil || p.HelpRoomId == 0 {
		return Failure
	}
	if ctx.RoomId == p.HelpRoomId {
		return Success
	}
	// Use the existing room-navigation helper. The exact API depends on
	// the codebase; one option is rooms.FindPath(currentRoomId, targetRoomId)
	// returning the next exit direction, then call ctx.Actor's move helper.
	// Implementation should match the pattern used by mob wandering /
	// the existing `move` action in actions_room.go.
	if !moveActorTowardRoom(caller, ctx.RoomId, p.HelpRoomId) {
		return Failure
	}
	return Success
}

// moveActorTowardRoom is a helper that finds the path from currentRoom
// to targetRoom and moves the actor one step along it. Returns false
// if no path exists.
//
// Implementation note: use whatever the existing engine pattern is for
// mob movement. Likely lives in a shared helper that the mob `move`
// action and `maxwander` use. If no such helper exists, it's a
// reasonable addition to behaviortree/actions_party.go (or its own
// shared file).
func moveActorTowardRoom(actor actions.Actor, currentRoomId, targetRoomId int) bool {
	// ... use rooms package's path-finding + actor's room-move method
	return false  // placeholder; implement during task
}
```

- [ ] **Step 4: Run test, verify pass**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: PASS.

- [ ] **Step 5: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 9: BTree action — party_follow_leader

**Files:**
- Modify: `internal/behaviortree/actions_party.go`
- Modify: `internal/behaviortree/actions_party_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestPartyFollowLeader_MovesTowardLeaderRoom(t *testing.T) {
	// Setup: NPC party where leader is in room A, member is in room B.
	// Fire party_follow_leader on the member.
	// Assert: member moves toward room A.

	// ...

	action := LookupAction("party_follow_leader")
	if action == nil {
		t.Fatal("party_follow_leader not registered")
	}
	status := action(ctx)
	if status != Success {
		t.Errorf("got %v, want Success", status)
	}
	// Assert member's room is now closer to leader's room.
}

func TestPartyFollowLeader_LeaderInSameRoom_NoOp(t *testing.T) {
	// Setup: leader and member in same room.
	// Fire party_follow_leader.
	// Assert: status Success, member still in same room.
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: FAIL.

- [ ] **Step 3: Implement**

```go
func init() {
	RegisterAction("party_follow_leader", actionPartyFollowLeader)
}

func actionPartyFollowLeader(ctx EventContext) Status {
	caller := ctx.Actor
	if caller == nil {
		return Failure
	}
	p := parties.GetByActor(caller)
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
	if !moveActorTowardRoom(caller, ctx.RoomId, leaderRoomId) {
		return Failure
	}
	return Success
}
```

- [ ] **Step 4: Run test, verify pass**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: PASS.

- [ ] **Step 5: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 10: BTree action — party_assist_target

**Files:**
- Modify: `internal/behaviortree/actions_party.go`
- Modify: `internal/behaviortree/actions_party_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestPartyAssistTarget_MatchesLeaderTarget(t *testing.T) {
	// Setup: NPC party. Leader is in combat targeting a player.
	// Member is in same room, no current target.
	// Fire party_assist_target on the member.
	// Assert: member's combat target is now the same as leader's.

	// ...

	action := LookupAction("party_assist_target")
	if action == nil {
		t.Fatal("party_assist_target not registered")
	}
	status := action(ctx)
	if status != Success {
		t.Errorf("got %v, want Success", status)
	}
	// Assert member.Aggro target == leader.Aggro target.
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: FAIL.

- [ ] **Step 3: Implement**

```go
func init() {
	RegisterAction("party_assist_target", actionPartyAssistTarget)
}

// actionPartyAssistTarget makes the calling actor target whatever the
// party leader is targeting. Returns Failure if the leader isn't in
// combat or the caller can't acquire the target (different room, etc.).
func actionPartyAssistTarget(ctx EventContext) Status {
	caller := ctx.Actor
	if caller == nil {
		return Failure
	}
	p := parties.GetByActor(caller)
	if p == nil || p.Leader == nil {
		return Failure
	}
	leaderChar := p.Leader.GetCharacter()
	if leaderChar == nil || leaderChar.Aggro == nil {
		return Failure
	}
	// Set caller's aggro to match leader's. Exact field/method depends
	// on the Character struct; see characters/character.go.
	callerChar := caller.GetCharacter()
	if callerChar == nil {
		return Failure
	}
	callerChar.Aggro = leaderChar.Aggro  // adjust to match actual API
	return Success
}
```

NOTE: the exact `Aggro` assignment depends on the `Character` struct's actual field. Match the pattern used by `assist.go` / existing combat helpers.

- [ ] **Step 4: Run test, verify pass**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: PASS.

- [ ] **Step 5: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 11: BTree action — party_flee_to_room

**Files:**
- Modify: `internal/behaviortree/actions_party.go`
- Modify: `internal/behaviortree/actions_party_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestPartyFleeToRoom_MovesEveryoneToward(t *testing.T) {
	// Setup: NPC party with 3 members in the same starting room.
	// Fire party_flee_to_room with target room = bandit camp room ID.
	// Assert: all 3 members move one step closer to camp.

	// ...

	action := LookupAction("party_flee_to_room")
	if action == nil {
		t.Fatal("party_flee_to_room not registered")
	}
	// EventContext needs to carry the target room ID as a parameter.
	ctx.Params = map[string]interface{}{"room_id": campRoomId}
	status := action(ctx)
	if status != Success {
		t.Errorf("got %v, want Success", status)
	}
	// Assert each member's room is closer to campRoomId.
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: FAIL.

- [ ] **Step 3: Implement**

```go
func init() {
	RegisterAction("party_flee_to_room", actionPartyFleeToRoom)
}

func actionPartyFleeToRoom(ctx EventContext) Status {
	caller := ctx.Actor
	if caller == nil {
		return Failure
	}
	p := parties.GetByActor(caller)
	if p == nil {
		return Failure
	}
	targetRoomId, ok := getActionParamInt(ctx, "room_id")
	if !ok || targetRoomId == 0 {
		return Failure
	}
	movedAny := false
	for _, member := range p.Members {
		memberRoom := member.GetRoom()
		if memberRoom == nil {
			continue
		}
		if memberRoom.RoomId == targetRoomId {
			continue
		}
		if moveActorTowardRoom(member, memberRoom.RoomId, targetRoomId) {
			movedAny = true
		}
	}
	if !movedAny {
		return Failure
	}
	return Success
}

// getActionParamInt is a helper to extract a typed int parameter from
// the EventContext. The exact signature/path depends on the existing
// behavior tree action parameter pattern (see actions_room.go for an
// existing precedent).
func getActionParamInt(ctx EventContext, name string) (int, bool) {
	if ctx.Params == nil {
		return 0, false
	}
	v, ok := ctx.Params[name].(int)
	return v, ok
}
```

- [ ] **Step 4: Run test, verify pass**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: PASS.

- [ ] **Step 5: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 12: BTree action — party_at_home_stand

**Files:**
- Modify: `internal/behaviortree/actions_party.go`
- Modify: `internal/behaviortree/actions_party_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestPartyAtHomeStand_SuppressesFleeInHomeRoom(t *testing.T) {
	// Setup: NPC party with HomeRoomId = X. Member is in room X.
	// Fire party_at_home_stand on the member.
	// Assert: status Success; member's "fleeing" state (if any
	// engine state tracks this) is cleared / suppressed.

	// ...

	action := LookupAction("party_at_home_stand")
	if action == nil {
		t.Fatal("party_at_home_stand not registered")
	}
	status := action(ctx)
	if status != Success {
		t.Errorf("got %v, want Success", status)
	}
}

func TestPartyAtHomeStand_FailureWhenNotAtHome(t *testing.T) {
	// Setup: NPC party with HomeRoomId = X. Member is in room Y (Y != X).
	// Fire party_at_home_stand.
	// Assert: status Failure (or a documented "not applicable" return).
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: FAIL.

- [ ] **Step 3: Implement**

```go
func init() {
	RegisterAction("party_at_home_stand", actionPartyAtHomeStand)
}

func actionPartyAtHomeStand(ctx EventContext) Status {
	caller := ctx.Actor
	if caller == nil {
		return Failure
	}
	p := parties.GetByActor(caller)
	if p == nil || p.HomeRoomId == 0 {
		return Failure
	}
	if ctx.RoomId != p.HomeRoomId {
		return Failure
	}
	// Mark the actor as "standing" — implementation depends on whether
	// the engine has a flee-suppression flag. If the caller has a
	// BehaviorState map (see behavior.md schema), set state[
	// "party_standing"] = "true". Member btrees check this to skip
	// flee branches.
	caller.GetCharacter().BehaviorState["party_standing"] = "true"  // adjust to actual API
	return Success
}
```

- [ ] **Step 4: Run test, verify pass**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: PASS.

- [ ] **Step 5: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 13: BTree conditions (5 in one task)

**Files:**
- Create: `internal/behaviortree/conditions_party.go`
- Create: `internal/behaviortree/conditions_party_test.go`

- [ ] **Step 1: Write tests for all 5 conditions**

```go
// internal/behaviortree/conditions_party_test.go
package behaviortree

import "testing"

func TestPartyMemberBelowPct_TrueWhenAnyMemberLow(t *testing.T) {
	// Setup: NPC party with member at 25% HP.
	// Condition params: pool="hp", percent=30.
	// Assert: condition returns true.
}

func TestPartyInCombat_TrueWhenAnyMemberInCombat(t *testing.T) {
	// Setup: NPC party with one member in combat.
	// Assert: party_in_combat condition true.
}

func TestPartyLeaderInCombat_TrueWhenLeaderInCombat(t *testing.T) {
	// Setup: NPC party with leader in combat.
	// Assert: party_leader_in_combat condition true.
}

func TestPartyInRoom_TrueWhenAllInSameRoom(t *testing.T) {
	// Setup: NPC party with all members in same room.
	// Assert: party_in_room condition true.
}

func TestPartyAtHome_TrueWhenAllAtHomeRoom(t *testing.T) {
	// Setup: NPC party with HomeRoomId = X, all members in room X.
	// Assert: party_at_home condition true.
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: FAIL with undefined conditions.

- [ ] **Step 3: Implement all 5 conditions**

```go
// internal/behaviortree/conditions_party.go
package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/parties"
)

func init() {
	RegisterCondition("party_member_below_pct", conditionPartyMemberBelowPct)
	RegisterCondition("party_in_combat", conditionPartyInCombat)
	RegisterCondition("party_leader_in_combat", conditionPartyLeaderInCombat)
	RegisterCondition("party_in_room", conditionPartyInRoom)
	RegisterCondition("party_at_home", conditionPartyAtHome)
}

func conditionPartyMemberBelowPct(ctx EventContext) bool {
	p := parties.GetByActor(ctx.Actor)
	if p == nil {
		return false
	}
	pool, _ := ctx.Params["pool"].(string)
	pct, _ := ctx.Params["percent"].(int)
	if pool == "" || pct <= 0 {
		return false
	}
	for _, m := range p.Members {
		c := m.GetCharacter()
		if c == nil {
			continue
		}
		var current, max int
		switch pool {
		case "hp":
			current, max = c.Health, c.HealthMax()
		case "sp":
			current, max = c.Stamina, c.StaminaMax()
		case "cp":
			current, max = c.Conviction, c.ConvictionMax()
		default:
			return false
		}
		if max > 0 && (current*100)/max < pct {
			return true
		}
	}
	return false
}

func conditionPartyInCombat(ctx EventContext) bool {
	p := parties.GetByActor(ctx.Actor)
	if p == nil {
		return false
	}
	for _, m := range p.Members {
		c := m.GetCharacter()
		if c != nil && c.Aggro != nil {
			return true
		}
	}
	return false
}

func conditionPartyLeaderInCombat(ctx EventContext) bool {
	p := parties.GetByActor(ctx.Actor)
	if p == nil || p.Leader == nil {
		return false
	}
	c := p.Leader.GetCharacter()
	return c != nil && c.Aggro != nil
}

func conditionPartyInRoom(ctx EventContext) bool {
	p := parties.GetByActor(ctx.Actor)
	if p == nil || len(p.Members) == 0 {
		return false
	}
	first := p.Members[0].GetRoom()
	if first == nil {
		return false
	}
	for _, m := range p.Members[1:] {
		r := m.GetRoom()
		if r == nil || r.RoomId != first.RoomId {
			return false
		}
	}
	return true
}

func conditionPartyAtHome(ctx EventContext) bool {
	p := parties.GetByActor(ctx.Actor)
	if p == nil || p.HomeRoomId == 0 {
		return false
	}
	for _, m := range p.Members {
		r := m.GetRoom()
		if r == nil || r.RoomId != p.HomeRoomId {
			return false
		}
	}
	return true
}
```

NOTE: `Character.HealthMax()`, `Character.Aggro` etc. are placeholders — match the actual `internal/characters/character.go` API.

- [ ] **Step 4: Run tests, verify pass**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./internal/behaviortree/...
```
Expected: PASS.

- [ ] **Step 5: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 14: Admin commands — party admin list-npc + show

**Files:**
- Modify: `internal/usercommands/party.go`
- Modify: `internal/parties/parties.go` (expose listing helper)

- [ ] **Step 1: Add a listing helper to parties package**

```go
// internal/parties/parties.go

// ListAllParties returns all parties currently in the registry.
// Used by admin commands and debug tooling.
func ListAllParties() []*Party {
	seen := map[*Party]bool{}
	out := []*Party{}
	for _, p := range actorPartyMap {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}
```

- [ ] **Step 2: Add admin subcommands to usercommands/party.go**

Read the existing `internal/usercommands/party.go` to understand the command-dispatch pattern. Add handling for:
- `party admin list-npc` — list all parties whose leader is an NPC (Leader != nil && !Leader.IsPlayer())
- `party admin show <party-id>` — show one party's full state (members, leader, home room, help room)

Show concrete code in the actual implementation pass; the pattern depends on the existing argv parsing in party.go.

- [ ] **Step 3: Manual smoke (optional — no automated test for admin commands)**

Boot the server, spawn an NPC party (via the bandit pack from Task 16, or a temp test), run `party admin list-npc`, verify the party appears.

- [ ] **Step 4: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 15: Migrate MobDeath_PackFlee.go to fire party_dissolved on leader death

**Files:**
- Modify: `internal/hooks/MobDeath_PackFlee.go`

- [ ] **Step 1: Read the current implementation**

Re-read `internal/hooks/MobDeath_PackFlee.go` (already done in Task 1, step 3). Identify the existing logic.

- [ ] **Step 2: Add party-dissolution path**

In the death handler, before (or instead of) the existing pack-flee logic, check whether the dying mob is the leader of an NPC party. If so, call `Dissolve("leader_died")`:

```go
import "github.com/GoMudEngine/GoMud/internal/parties"
import "github.com/GoMudEngine/GoMud/internal/actions"

// In the death handler:
mobActor := actions.MobActorFor(deadMob)  // adjust to actual API
if p := parties.GetByActor(mobActor); p != nil && p.Leader == mobActor {
	p.Dissolve("leader_died")
	return  // dissolution supersedes pack-flee
}

// Existing pack-flee logic remains for non-leader deaths and for
// non-party packs (if any).
```

- [ ] **Step 3: If existing pack-flee logic is fully redundant after the bandit migration in Task 16, delete it**

Make this decision during implementation — only delete if Task 16 fully replaces the pack-flee behavior with party-system primitives. If unsure, leave the existing code as a fallback for non-party packs.

- [ ] **Step 4: Build**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

- [ ] **Step 5: Test**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./...
```
Expected: PASS.

---

## Task 16: Wire bandit btrees to use new party primitives + lazy party formation

**Files:**
- Create or Modify: `_datafiles/world/dogmud/behaviors/north_road/283-bandit_lookout.yaml` (and 284, 285, 286)
- Create or Modify: `_datafiles/world/dogmud/behaviors/marches_spur_road/253-road_bandit.yaml`, `254-bandit_leader.yaml`

- [ ] **Step 1: Identify which bandit zones get the migration**

From Task 1's scout: list the bandit mob IDs and their zones (north_road has 283-286; marches_spur_road has 253-254). Decide which zone is the smoke-test target. Recommend: north_road (Soren's pack) first.

- [ ] **Step 2: Write the lookout's behavior tree**

Create `_datafiles/world/dogmud/behaviors/north_road/283-bandit_lookout.yaml`:

```yaml
# Bandit Lookout — Stage 1 NPC party smoke-test consumer
# Uses new party primitives: party_call_help, party_respond_to_help,
# party_assist_target, party_flee_to_room, party_at_home_stand.

tree:
  type: selector
  children:

    # ── PLAYER_ENTER: hostile player spotted, call for help ──
    - type: sequence
      event: player_enter
      children:
        - type: condition
          check: random_chance
          percent: 100   # always trigger; tighten later if needed
        - type: action
          do: emote
          text: "whistles sharply, eyes widening at the intruder"
        - type: action
          do: party_call_help

    # ── COMBAT: assist the leader's target ──
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: party_in_combat
        - type: action
          do: party_assist_target

    # ── HOME ROOM: at camp, hold ground (suppress flee) ──
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: party_at_home
        - type: action
          do: party_at_home_stand

    # ── OUT OF COMBAT, NOT AT HOME: respond to help if called ──
    - type: sequence
      event: mob_idle
      children:
        - type: action
          do: party_respond_to_help
```

Keep tree concise. Existing `mob_die` / hostile-by-default behavior comes from the archetype; this btree only adds the party-aware reactions.

- [ ] **Step 3: Write fighter (284) and caster (285) btrees**

Same template as lookout but WITHOUT the `player_enter` → `party_call_help` branch (only lookouts call help on player entry; fighters/casters respond when summoned). Files:
- `behaviors/north_road/284-bandit_fighter.yaml`
- `behaviors/north_road/285-bandit_caster.yaml`

- [ ] **Step 4: Write Soren's (286) leader btree**

Soren's tree adds the leader-specific retreat trigger:

```yaml
tree:
  type: selector
  children:

    # ── COMBAT: assist (drives own targeting; members assist him) ──
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: party_in_combat
        - type: action
          do: party_assist_target

    # ── LEADER FLEE TRIGGER: when group health is low, retreat to camp ──
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: party_in_combat
        - type: condition
          check: party_member_below_pct
          pool: hp
          percent: 30
        - type: action
          do: party_flee_to_room
          room_id: <CAMP_ROOM_ID_FROM_TASK_1>   # replace with actual ID

    # ── HOME ROOM: hold the camp ──
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: party_at_home
        - type: action
          do: party_at_home_stand
```

- [ ] **Step 5: Wire lazy party formation**

Add a one-time on-spawn party-creation step. This can be done in two ways; pick the simpler:

**Option a (recommended):** Add a new btree action `party_ensure_npc_party` that creates a party with the specified leader mob and adds the calling mob as a member. Lookout's `player_enter` branch calls this BEFORE `party_call_help`. Other bandits' `mob_idle` calls it on first tick.

OR

**Option b:** Add an init-handler in the mobs package that, on bandit-mob spawn, creates/joins the party (more centralized but requires mobs package changes).

For Stage 1, Option a is simpler and matches the btree-driven pattern. Implement `party_ensure_npc_party` as a new action with params `leader_mob_id` (Soren's mob ID) and `home_room_id` (camp room ID). The action checks whether the calling mob is already in a party; if not, finds the leader mob in the world, creates the party with that leader (using `parties.NewByActor` with the leader's MobActor), adds the calling mob with `AddActor`, and sets HomeRoomId.

Add this action to `internal/behaviortree/actions_party.go` with its own test.

- [ ] **Step 6: Update bandit btrees to call party_ensure_npc_party first**

Each bandit btree starts with a sequence that ensures party membership before any party-action runs:

```yaml
tree:
  type: selector
  children:
    # ── Always: ensure we're in the bandit party ──
    - type: action
      do: party_ensure_npc_party
      leader_mob_id: 286
      home_room_id: <CAMP_ROOM_ID>

    # ── (rest of existing tree) ──
```

The action returns Success unconditionally (party-ensure is idempotent), so this branch never blocks others.

- [ ] **Step 7: Build + test**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./... && go test ./...
```
Expected: clean + PASS.

---

## Task 17: Update behavior.md schema docs

**Files:**
- Modify: `docs/schemas/behavior.md`

- [ ] **Step 1: Add new actions to the action table**

In the appropriate section of `docs/schemas/behavior.md`, add documentation for:
- `party_call_help` (params: none; effect: marks party HelpRoomId, fires PartyHelpRequested event)
- `party_respond_to_help` (params: none; effect: navigates one step toward party.HelpRoomId)
- `party_follow_leader` (params: none; effect: navigates one step toward leader's room)
- `party_assist_target` (params: none; effect: matches caller's combat target to leader's)
- `party_flee_to_room` (params: room_id int; effect: all party members navigate one step toward target room)
- `party_at_home_stand` (params: none; effect: at HomeRoomId, sets standing flag to suppress flee branches)
- `party_ensure_npc_party` (params: leader_mob_id int, home_room_id int; effect: idempotent party creation/join for NPC parties)

- [ ] **Step 2: Add new conditions to the conditions table**

- `party_member_below_pct` (params: pool string "hp"|"sp"|"cp", percent int)
- `party_in_combat` (params: none)
- `party_leader_in_combat` (params: none)
- `party_in_room` (params: none)
- `party_at_home` (params: none)

- [ ] **Step 3: Add new events to the events table**

- `PartyHelpRequested` (fires from party_call_help; payload: PartyId, CallerActorId, CallerIsPlayer, RallyRoomId)
- `PartyDissolved` (fires from Party.Dissolve; payload: PartyId, Reason "leader_died"|"disbanded"|"all_dead", MemberActorIds)

- [ ] **Step 4: Build (no impact, but verify nothing accidentally got pulled in)**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

---

## Task 18: Verification — build, tests, server boot, smoke test

**Files:**
- None modified; verification only.

- [ ] **Step 1: Full build clean**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...
```
Expected: clean.

- [ ] **Step 2: Full test suite passes**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && go test ./...
```
Expected: PASS across all packages. Specifically check:
- `./internal/parties/...` — old + new tests
- `./internal/behaviortree/...` — new party action + condition tests
- `./internal/events/...`
- `./internal/usercommands/...` — existing party command tests still pass

- [ ] **Step 3: Local server boot per Pre-Push SOP (CLAUDE.md)**

Boot the server locally. Watch the startup log for:
- `parties.LoadDataFiles()` — if such a logging line exists; if not, just verify the package init doesn't panic
- `mobs.LoadDataFiles() loadedCount=...` — confirm bandit mobs load without panic
- No panic before the "server ready" line

- [ ] **Step 4: In-game smoke test (manual, the 9-step bandit sequence from spec)**

1. Walk to bandit lookout's room (recorded in Task 1).
2. Confirm the lookout emotes "whistles sharply" and party_call_help fires (visible: HelpRoomId set; PartyHelpRequested event in debug log if available).
3. Confirm fighter/caster (and Soren if not yet here) navigate toward the lookout's room over the next several rounds.
4. Engage in combat. Confirm bandits coordinate via party_assist_target (all attacking the same player).
5. Bring group HP below 30% on at least one member (let them hit you, or focus-fire one bandit).
6. Confirm Soren's `party_flee_to_room` fires; bandits start moving toward camp.
7. Walk to the camp room ahead of them; confirm they arrive.
8. Engage at the camp; confirm bandits no longer flee (party_at_home_stand suppresses retreat) — they fight to the death.
9. Kill Soren; confirm `party_dissolved` event fires; surviving bandits revert to solo behavior.

If any step fails, debug + fix; do not proceed to commit until smoke is green.

- [ ] **Step 5: Backward compat smoke test (manual)**

Form a player party (you + a test character or test NPC companion). Verify:
- `party invite` works
- `party accept` works
- `party position front/middle/back` works (targeting changes appropriately)
- `party leave` works (party transfers leadership or dissolves correctly)

- [ ] **Step 6: Commit (single end-of-plan commit; user can re-slice if desired)**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && git add . && git commit -m "$(cat <<'EOF'
feat(parties): Stage 1 NPC party system + bandit pack refactor

Extends internal/parties to support NPC actors as party members and
leaders via an actor-based API layer (NewByActor / GetByActor /
AddActor / RemoveActor / Dissolve). Existing UserId-based public API
preserved as compatibility shims; 20+ consumer files unchanged.

New behavior tree primitives:
- Actions: party_call_help, party_respond_to_help, party_follow_leader,
  party_assist_target, party_flee_to_room, party_at_home_stand,
  party_ensure_npc_party
- Conditions: party_member_below_pct, party_in_combat,
  party_leader_in_combat, party_in_room, party_at_home
- Events: PartyHelpRequested, PartyDissolved

Bandit pack (north_road 283-286) migrated to use the new party system
as the smoke-test consumer:
- Lookout calls help on player_enter
- Fighter / caster respond_to_help
- Soren triggers flee_to_room when group HP drops below 30%
- All bandits at-home_stand at the camp room (last-stand behavior)
- On Soren's death, PartyDissolved fires; survivors revert to solo

Stage 1 of multi-stage caravan effort. Stage 2 (basic caravan) consumes
these primitives.

Spec: docs/superpowers/specs/2026-04-27-npc-party-system-design.md
Plan: docs/superpowers/plans/2026-04-27-npc-party-system.md

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review Notes

**Spec coverage check:**
- Architecture overview → Tasks 2-6 (Actor abstraction, registry, events)
- Data model changes → Task 3 (struct refactor with parallel fields)
- New btree actions (6 of them) → Tasks 7-12
- New btree conditions (5 of them) → Task 13
- New events (2 of them) → Task 6
- Bandit migration → Tasks 15-16
- Edge cases → covered implicitly in implementation; specifically Dissolve handles leader death; HomeRoomId 0 means "no home" so at_home returns false; no-party callers return Failure
- Testing strategy → Tasks 2-13 each include tests; Task 18 wraps with full suite + manual smoke
- Out of scope → respected (no caravan / no cross-zone / no forager / no item transfer)
- Files affected → matches spec's files-affected list (parties.go, party.go user command, eventtypes.go, behaviortree/actions_party.go, conditions_party.go, MobDeath_PackFlee.go, behavior.md, bandit btree YAMLs)
- Admin commands → Task 14
- Documentation update → Task 17

**Placeholder scan:**
- Several "implementation note" comments flag specific places where the engineer needs to match existing API patterns (e.g., `Aggro` field name, `Character.HealthMax()` method, `EventContext.Params` shape). These are deliberate — they tell the engineer "match the existing pattern here" rather than guess. Not placeholder failures.
- No "TBD" / "TODO" / "fill in details" / "similar to Task N" patterns.

**Type consistency:**
- `actions.Actor` used consistently as the interface
- `ActorKey` used consistently as the map-key type
- `parties.GetByActor` / `parties.NewByActor` consistent across all task references
- `party_*` action and condition names used consistently across implementation, tests, behavior.md, and bandit btree YAMLs
