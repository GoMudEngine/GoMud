# Instanced Zones Framework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the core instanced zones framework — ephemeral zone cloning with party-scoped portals, access control, death policies, AFK ejection, and timer lifecycle.

**Architecture:** Extends the existing ephemeral room system (`internal/rooms/ephemeral.go`) with an instance registry that tracks active instances, their access lists, and metadata. A gatekeeper check in `MoveToRoom` enforces access control. Zone configs get new fields for instance behavior. NPC vendor scripts handle the gold-to-portal flow.

**Tech Stack:** Go, YAML data files, JS mob scripts, testify for tests

**Spec:** `docs/superpowers/specs/2026-04-09-instanced-zones-framework-design.md`

---

### Task 1: Zone Config Extensions

**Files:**
- Modify: `internal/rooms/zoneconfig.go`
- Test: `internal/rooms/zoneconfig_test.go` (new)

- [ ] **Step 1: Write test for new zone config fields**

Create `internal/rooms/zoneconfig_test.go`:

```go
package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZoneConfig_InstanceDefaults(t *testing.T) {
	zc := NewZoneConfig("test-zone")
	assert.False(t, zc.Instanced)
	assert.Equal(t, "rejoin", zc.DeathPolicy)
	assert.Equal(t, "", zc.PortalDuration)
	assert.Equal(t, 0, zc.EntryRoom)
	assert.True(t, zc.AllowRecall)
}
```

- [ ] **Step 2: Run test — should fail**

Run: `go test ./internal/rooms/ -run TestZoneConfig_InstanceDefaults -v`

- [ ] **Step 3: Add instance fields to ZoneConfig**

In `internal/rooms/zoneconfig.go`, add to the `ZoneConfig` struct:

```go
	Instanced      bool   `yaml:"instanced,omitempty"`       // true = this zone is an instance template
	DeathPolicy    string `yaml:"death_policy,omitempty"`    // "rejoin" (default) or "ejected"
	PortalDuration string `yaml:"portal_duration,omitempty"` // e.g. "30m", "1h"
	EntryRoom      int    `yaml:"entry_room,omitempty"`      // room ID where portal drops players
	AllowRecall    bool   `yaml:"allow_recall,omitempty"`    // whether recall works inside (default true)
```

Update `NewZoneConfig` to set defaults:

```go
func NewZoneConfig(zName string) *ZoneConfig {
	return &ZoneConfig{
		Name:        zName,
		RoomIds:     map[int]struct{}{},
		DeathPolicy: "rejoin",
		AllowRecall: true,
	}
}
```

Update `Validate()` to default `DeathPolicy` if empty:

```go
if z.DeathPolicy == "" {
	z.DeathPolicy = "rejoin"
}
if z.Instanced && z.PortalDuration == "" {
	z.PortalDuration = "30m"
}
```

- [ ] **Step 4: Run test — should pass**

Run: `go test ./internal/rooms/ -run TestZoneConfig_InstanceDefaults -v`

- [ ] **Step 5: Compile and commit**

Run: `go build ./...`

```bash
git add internal/rooms/zoneconfig.go internal/rooms/zoneconfig_test.go
git commit -m "feat(instances): add instance fields to ZoneConfig"
```

---

### Task 2: Instance Registry

**Files:**
- Create: `internal/rooms/instances.go`
- Test: `internal/rooms/instances_test.go` (new)

- [ ] **Step 1: Write tests for instance registry**

Create `internal/rooms/instances_test.go`:

```go
package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceRegistry_CreateAndLookup(t *testing.T) {
	reg := NewInstanceRegistry()

	inst := &ZoneInstance{
		InstanceId:      1,
		TemplateZone:    "test-arena",
		GoldPaid:        500,
		AuthorizedUsers: []int{10, 20, 30},
		OwnerUserId:     10,
		CreatedRound:    1000,
		PortalDuration:  "30m",
		DeathPolicy:     "rejoin",
		AllowRecall:     true,
		OverworldRoomId: 100,
		EntryRoomId:     1000001,
		RoomIdMap:       map[int]int{1: 1000001, 2: 1000002},
	}

	reg.Add(inst)

	// Lookup by room ID
	found := reg.FindByRoomId(1000001)
	assert.NotNil(t, found)
	assert.Equal(t, "test-arena", found.TemplateZone)
	assert.Equal(t, 500, found.GoldPaid)

	// Lookup second room in same instance
	found2 := reg.FindByRoomId(1000002)
	assert.NotNil(t, found2)
	assert.Equal(t, found, found2)

	// Unknown room returns nil
	assert.Nil(t, reg.FindByRoomId(999))

	// Auth check
	assert.True(t, found.IsAuthorized(10))
	assert.True(t, found.IsAuthorized(20))
	assert.False(t, found.IsAuthorized(99))

	// Remove
	reg.Remove(inst)
	assert.Nil(t, reg.FindByRoomId(1000001))
}

func TestInstanceRegistry_RevokeAccess(t *testing.T) {
	inst := &ZoneInstance{
		AuthorizedUsers: []int{10, 20, 30},
		DeathPolicy:     "ejected",
	}
	assert.True(t, inst.IsAuthorized(20))
	inst.RevokeAccess(20)
	assert.False(t, inst.IsAuthorized(20))
}
```

- [ ] **Step 2: Run tests — should fail**

Run: `go test ./internal/rooms/ -run TestInstanceRegistry -v`

- [ ] **Step 3: Implement instance registry**

Create `internal/rooms/instances.go`:

```go
package rooms

import "sync"

// ZoneInstance tracks an active instanced zone.
type ZoneInstance struct {
	InstanceId      int            // ephemeral chunk ID
	TemplateZone    string         // zone name that was cloned
	GoldPaid        int            // gold amount (for scaling system)
	AuthorizedUsers []int          // userId snapshot at creation
	OwnerUserId     int            // who paid
	CreatedRound    uint64         // for timer tracking
	PortalDuration  string         // from zone config
	DeathPolicy     string         // "rejoin" or "ejected"
	AllowRecall     bool           // from zone config
	OverworldRoomId int            // room where portal was created
	EntryRoomId     int            // ephemeral entry room ID
	RoomIdMap       map[int]int    // original → ephemeral room ID mapping
}

// IsAuthorized checks if a user is on the access list.
func (z *ZoneInstance) IsAuthorized(userId int) bool {
	for _, id := range z.AuthorizedUsers {
		if id == userId {
			return true
		}
	}
	return false
}

// RevokeAccess removes a user from the authorized list (used by
// "ejected" death policy).
func (z *ZoneInstance) RevokeAccess(userId int) {
	for i, id := range z.AuthorizedUsers {
		if id == userId {
			z.AuthorizedUsers = append(z.AuthorizedUsers[:i],
				z.AuthorizedUsers[i+1:]...)
			return
		}
	}
}

// InstanceRegistry tracks all active zone instances. Thread-safe.
type InstanceRegistry struct {
	mu        sync.RWMutex
	instances []*ZoneInstance
	roomIndex map[int]*ZoneInstance // ephemeral roomId → instance
}

func NewInstanceRegistry() *InstanceRegistry {
	return &InstanceRegistry{
		roomIndex: make(map[int]*ZoneInstance),
	}
}

// Add registers a new instance.
func (r *InstanceRegistry) Add(inst *ZoneInstance) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instances = append(r.instances, inst)
	for _, ephId := range inst.RoomIdMap {
		r.roomIndex[ephId] = inst
	}
}

// Remove deregisters an instance.
func (r *InstanceRegistry) Remove(inst *ZoneInstance) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, existing := range r.instances {
		if existing == inst {
			r.instances = append(r.instances[:i], r.instances[i+1:]...)
			break
		}
	}
	for _, ephId := range inst.RoomIdMap {
		delete(r.roomIndex, ephId)
	}
}

// FindByRoomId returns the instance that owns a given ephemeral room.
func (r *InstanceRegistry) FindByRoomId(roomId int) *ZoneInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.roomIndex[roomId]
}

// All returns a snapshot of all active instances.
func (r *InstanceRegistry) All() []*ZoneInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ZoneInstance, len(r.instances))
	copy(result, r.instances)
	return result
}
```

Add a package-level singleton (at the bottom of the file):

```go
var instanceRegistry = NewInstanceRegistry()

// GetInstanceRegistry returns the global instance registry.
func GetInstanceRegistry() *InstanceRegistry {
	return instanceRegistry
}
```

- [ ] **Step 4: Run tests — should pass**

Run: `go test ./internal/rooms/ -run TestInstanceRegistry -v`

- [ ] **Step 5: Compile and commit**

Run: `go build ./...`

```bash
git add internal/rooms/instances.go internal/rooms/instances_test.go
git commit -m "feat(instances): add ZoneInstance struct and registry"
```

---

### Task 3: Instance Creation Function

**Files:**
- Modify: `internal/rooms/instances.go` (add `CreateZoneInstance`)
- Modify: `internal/rooms/instances_test.go`

- [ ] **Step 1: Write test for CreateZoneInstance**

Add to `internal/rooms/instances_test.go`:

```go
func TestCreateZoneInstance(t *testing.T) {
	// This test requires zone templates to be loaded.
	// If that's not feasible in unit tests, this becomes
	// an integration test verified manually.
	// At minimum, test the function signature exists.
	_ = CreateZoneInstance // ensure it compiles
}
```

Note: Full integration testing of `CreateZoneInstance` requires loaded zone data. The implementer should verify manually that the function works by creating a test instance zone. Write a struct-level compile check at minimum.

- [ ] **Step 2: Implement CreateZoneInstance**

Add to `internal/rooms/instances.go`:

```go
import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CreateZoneInstance clones a zone template into ephemeral rooms,
// creates a return portal in the entry room, and registers the
// instance in the global registry.
//
// Parameters:
//   - zoneName: the template zone to clone
//   - goldPaid: gold amount (stored for scaling system)
//   - ownerUserId: who paid for the instance
//   - authorizedUsers: userId snapshot (owner + party members)
//   - overworldRoomId: room where the entry portal should be created
//
// Returns the ZoneInstance and an error if creation fails.
func CreateZoneInstance(
	zoneName string,
	goldPaid int,
	ownerUserId int,
	authorizedUsers []int,
	overworldRoomId int,
) (*ZoneInstance, error) {

	// Look up zone config
	zCfg := GetZoneConfig(zoneName)
	if zCfg == nil {
		return nil, fmt.Errorf("zone %q not found", zoneName)
	}
	if !zCfg.Instanced {
		return nil, fmt.Errorf("zone %q is not an instance template", zoneName)
	}

	// Clone the zone into ephemeral rooms
	roomIdMap, err := CreateEphemeralZone(zoneName)
	if err != nil {
		return nil, fmt.Errorf("failed to create ephemeral zone: %w", err)
	}

	// Find the ephemeral entry room
	entryRoomId, ok := roomIdMap[zCfg.EntryRoom]
	if !ok {
		return nil, fmt.Errorf("entry room %d not found in zone %q", zCfg.EntryRoom, zoneName)
	}

	// Compute chunk ID from any ephemeral room ID for cleanup tracking
	var firstEphId int
	for _, ephId := range roomIdMap {
		firstEphId = ephId
		break
	}

	inst := &ZoneInstance{
		InstanceId:      firstEphId, // any ephemeral ID works for chunk lookup
		TemplateZone:    zoneName,
		GoldPaid:        goldPaid,
		AuthorizedUsers: authorizedUsers,
		OwnerUserId:     ownerUserId,
		CreatedRound:    util.GetRoundCount(),
		PortalDuration:  zCfg.PortalDuration,
		DeathPolicy:     zCfg.DeathPolicy,
		AllowRecall:     zCfg.AllowRecall,
		OverworldRoomId: overworldRoomId,
		EntryRoomId:     entryRoomId,
		RoomIdMap:       roomIdMap,
	}

	// Create a return portal in the ephemeral entry room
	entryRoom := LoadRoom(entryRoomId)
	if entryRoom != nil {
		entryRoom.AddTemporaryExit("return portal", exit.TemporaryRoomExit{
			RoomId:  overworldRoomId,
			Title:   "return portal",
			UserId:  0, // system-created, never expires via prune
			Expires: "999h", // effectively permanent — cleaned up when instance is destroyed
		})
	}

	// Store instance metadata on each ephemeral room for scripting access
	for _, ephId := range roomIdMap {
		if r := LoadRoom(ephId); r != nil {
			r.SetTempData("instance_id", firstEphId)
			r.SetTempData("allow_recall", zCfg.AllowRecall)
			r.SetTempData("death_policy", zCfg.DeathPolicy)
			r.SetTempData("gold_paid", goldPaid)
		}
	}

	instanceRegistry.Add(inst)

	return inst, nil
}
```

Note: The implementer MUST verify that `Room.SetTempData` exists. Search for `SetTempData` or `tempDataStore` in `rooms.go`. If the method doesn't exist, use the `LongTermDataStore` map or add a simple setter. The key requirement is that room-level metadata is accessible from scripts.

- [ ] **Step 3: Compile and commit**

Run: `go build ./...`

```bash
git add internal/rooms/instances.go internal/rooms/instances_test.go
git commit -m "feat(instances): add CreateZoneInstance function"
```

---

### Task 4: Access Control Gatekeeper in MoveToRoom

**Files:**
- Modify: `internal/rooms/roommanager.go` (add instance access check in `MoveToRoom`)
- Test: manual verification

- [ ] **Step 1: Add access check to MoveToRoom**

In `internal/rooms/roommanager.go`, inside `MoveToRoom()`, AFTER the room is loaded (after `newRoom := LoadRoom(toRoomId)` at ~line 286) but BEFORE preparing the room, add:

```go
	// Instance access control: block unauthorized entry to instanced zones
	if IsEphemeralRoomId(toRoomId) {
		if inst := instanceRegistry.FindByRoomId(toRoomId); inst != nil {
			if !inst.IsAuthorized(userId) {
				return fmt.Errorf("instance access denied")
			}
		}
	}
```

This blocks ALL paths into an instanced room — normal movement, teleport, scripted MoveRoom — since they all flow through `MoveToRoom`.

- [ ] **Step 2: Add user-facing message for blocked entry**

The error returned by `MoveToRoom` needs to surface to the player. Check how `go.go` (the movement command) handles `MoveToRoom` errors. If it swallows them, the access denial message needs to be sent directly. The implementer should check the call sites of `MoveToRoom` and ensure the rejection message reaches the player:

`"The portal's energy pushes you back. It wasn't opened for you."`

This may require sending the message before returning the error, or handling the specific error in the `go.go` command handler.

- [ ] **Step 3: Compile and commit**

Run: `go build ./...`

```bash
git add internal/rooms/roommanager.go
git commit -m "feat(instances): add access control gatekeeper in MoveToRoom"
```

---

### Task 5: Recall Blocking

**Files:**
- Modify: `_datafiles/world/dogmud/spells/fold-recall.js`

- [ ] **Step 1: Add instance recall check**

In `_datafiles/world/dogmud/spells/fold-recall.js`, in the `onCast` function, add a check before the existing anchor validation:

```javascript
// Check if current room blocks recall (instanced zone)
var currentRoomId = sourceActor.GetRoomId();
var currentRoom = GetRoom(currentRoomId);
if (currentRoom) {
    var allowRecall = currentRoom.GetTempData('allow_recall');
    if (allowRecall === false) {
        SendUserMessage(sourceActor.UserId(),
            'Something about this place prevents you from recalling.');
        return false;
    }
}
```

Note: Check what method name is used for reading temp data from a room in the scripting API. It might be `GetTempData`, `GetPermData`, or similar. The implementer must verify by reading `internal/scripting/room_func.go` for the correct method name. If temp data isn't exposed to scripts, use perm data instead (set during instance creation).

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/spells/fold-recall.js
git commit -m "feat(instances): block recall in no-recall instanced zones"
```

---

### Task 6: Death Policy Handling

**Files:**
- Modify: `internal/hooks/NewRound_AutoHeal.go` (or wherever death/respawn is handled)
- Modify: `internal/rooms/instances.go` (if needed)

- [ ] **Step 1: Find the death/respawn code path**

Search for where a player dies and is moved to the death recovery room. This is likely in `NewRound_AutoHeal.go` (the bleedout section around line 67 where `user.Command("suicide")` is called) or in the `suicide` command handler. The implementer must trace the death flow:

1. Health drops to <= -10 in AutoHeal → `user.Command("suicide")`
2. Suicide command handles respawn → `MoveToRoom(userId, deathRecoveryRoom)`

- [ ] **Step 2: Add death policy check**

After the player dies and before (or after) they are moved to the death recovery room, check if they were in an instance:

```go
// Check if player died in an instanced zone
if IsEphemeralRoomId(user.Character.RoomId) {
	if inst := GetInstanceRegistry().FindByRoomId(user.Character.RoomId); inst != nil {
		if inst.DeathPolicy == "ejected" {
			inst.RevokeAccess(user.UserId)
			user.SendText(`<ansi fg="red">You have been expelled from the instance. There is no return.</ansi>`)
		}
	}
}
```

The `RevokeAccess` call removes the player from the authorized list, so even if the portal is still open, the MoveToRoom gatekeeper will block re-entry.

For `rejoin` policy: no special handling needed. The player respawns at death recovery, walks back to the overworld portal room, and re-enters through the still-open portal. The MoveToRoom gatekeeper allows it because they're still authorized.

- [ ] **Step 3: Compile and commit**

Run: `go build ./...`

```bash
git add internal/hooks/NewRound_AutoHeal.go
git commit -m "feat(instances): handle death policy (rejoin/ejected)"
```

---

### Task 7: AFK Ejection from Instances

**Files:**
- Modify: `internal/hooks/NewRound_InactivePlayers.go`

- [ ] **Step 1: Add instance AFK check**

In `internal/hooks/NewRound_InactivePlayers.go`, in the loop that checks idle players, add a check for players in instanced zones. Before the existing max-idle kick, add an instance-specific AFK check:

```go
// Check if player is AFK in an instanced zone — eject them
if IsEphemeralRoomId(user.Character.RoomId) {
	afkRounds := uint64(configs.SecondsToRounds(int(configs.GetNetworkConfig().AfkSeconds)))
	if afkRounds > 0 && roundNow-user.GetLastInputRound() >= afkRounds {
		if inst := GetInstanceRegistry().FindByRoomId(user.Character.RoomId); inst != nil {
			user.SendText(`<ansi fg="yellow">You've been idle too long. The unstable magic of this place expels you.</ansi>`)
			MoveToRoom(user.UserId, inst.OverworldRoomId)
			continue
		}
	}
}
```

Note: The implementer must read `NewRound_InactivePlayers.go` to find the exact loop structure and variable names. The round count and user iteration pattern need to match what's already there. The `rooms` package import may need to be added (check for circular dependency — if `hooks` can't import `rooms`, the registry lookup needs to be exposed differently).

- [ ] **Step 2: Compile and commit**

Run: `go build ./...`

```bash
git add internal/hooks/NewRound_InactivePlayers.go
git commit -m "feat(instances): eject AFK players from instances"
```

---

### Task 8: Instance Cleanup on Empty

**Files:**
- Modify: `internal/rooms/ephemeral.go` or `internal/rooms/instances.go`
- Modify: `internal/rooms/roommanager.go` (maintenance loop)

- [ ] **Step 1: Hook instance cleanup into ephemeral maintenance**

The existing `EphemeralRoomMaintenance()` already cleans up empty ephemeral chunks. We need to also clean up the instance registry entry when a chunk is destroyed.

Option A: Override or wrap `TryEphemeralCleanup` to also remove the registry entry.
Option B: In the maintenance loop that calls `EphemeralRoomMaintenance`, also scan the instance registry for instances whose rooms are all empty.

The implementer should choose the approach that fits cleanest. The key requirement: when the last player leaves an instance and the ephemeral rooms are cleaned up, the corresponding `ZoneInstance` must be removed from the registry.

Add a function to `instances.go`:

```go
// CleanupEmptyInstances removes instance registry entries for
// instances that have been cleaned up (ephemeral rooms destroyed).
func (r *InstanceRegistry) CleanupEmptyInstances() {
	r.mu.Lock()
	defer r.mu.Unlock()

	remaining := make([]*ZoneInstance, 0, len(r.instances))
	for _, inst := range r.instances {
		// Check if any room in the instance still exists in memory
		alive := false
		for _, ephId := range inst.RoomIdMap {
			if LoadRoom(ephId) != nil {
				alive = true
				break
			}
		}
		if alive {
			remaining = append(remaining, inst)
		} else {
			// Clean up room index entries
			for _, ephId := range inst.RoomIdMap {
				delete(r.roomIndex, ephId)
			}
		}
	}
	r.instances = remaining
}
```

Call this from the room maintenance loop (wherever `EphemeralRoomMaintenance` is called).

- [ ] **Step 2: Also remove the overworld portal when instance is cleaned up**

When `CleanupEmptyInstances` removes an instance, it should also remove the temporary exit from the overworld room:

```go
if !alive {
	// Remove the entry portal from the overworld room
	if owRoom := LoadRoom(inst.OverworldRoomId); owRoom != nil {
		owRoom.RemoveTemporaryExit(exit.TemporaryRoomExit{
			RoomId: inst.EntryRoomId,
			Title:  "instance portal",
			UserId: 0,
		})
	}
}
```

Note: The `RemoveTemporaryExit` matches on `UserId + Title + RoomId`. The implementer must ensure the Title matches what was set during portal creation. Consider using a consistent portal title constant.

- [ ] **Step 3: Compile and commit**

Run: `go build ./...`

```bash
git add internal/rooms/instances.go internal/rooms/roommanager.go
git commit -m "feat(instances): clean up registry and portals when instances empty"
```

---

### Task 9: Portal Timer Warnings

**Files:**
- Modify: `internal/rooms/instances.go` (add warning check function)
- Modify: maintenance loop (call the warning function each round)

- [ ] **Step 1: Add timer warning function**

Add to `internal/rooms/instances.go`:

```go
// CheckPortalTimers sends warnings to players inside instances
// when the portal timer is running low. Called each maintenance cycle.
func (r *InstanceRegistry) CheckPortalTimers() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	currentRound := util.GetRoundCount()

	for _, inst := range r.instances {
		g := gametime.GetDate(inst.CreatedRound)
		expiryRound := g.AddPeriod(inst.PortalDuration)

		if currentRound >= expiryRound {
			continue // already expired, cleanup handles this
		}

		remainingRounds := expiryRound - currentRound
		fiveMinRounds := uint64(configs.SecondsToRounds(300))
		oneMinRounds := uint64(configs.SecondsToRounds(60))

		// Broadcast to all players in the instance
		var msg string
		if remainingRounds == fiveMinRounds {
			msg = `<ansi fg="yellow">The portal flickers — it won't hold much longer.</ansi>`
		} else if remainingRounds == oneMinRounds {
			msg = `<ansi fg="red">The portal is barely a shimmer now. Leave soon or find your own way out.</ansi>`
		} else if remainingRounds == 0 {
			msg = `<ansi fg="red">The entry portal collapses. The return portal in the entry chamber still glows steadily.</ansi>`
		}

		if msg != "" {
			for _, ephId := range inst.RoomIdMap {
				if room := LoadRoom(ephId); room != nil {
					room.SendText(msg)
				}
			}
		}
	}
}
```

Note: The implementer must import `gametime` and `configs` packages. Check for the exact function to convert seconds to rounds — it might be `configs.SecondsToRounds` or a method on a config struct. Verify by searching the codebase.

- [ ] **Step 2: Wire into maintenance loop**

Add `GetInstanceRegistry().CheckPortalTimers()` call alongside the existing `EphemeralRoomMaintenance()` call in the room maintenance ticker.

- [ ] **Step 3: Compile and commit**

Run: `go build ./...`

```bash
git add internal/rooms/instances.go internal/rooms/roommanager.go
git commit -m "feat(instances): add portal timer warnings at 5min and 1min"
```

---

### Task 10: Riftkeeper NPC, Room, and Vendor Script

**Files:**
- Create: `_datafiles/world/dogmud/rooms/thornwall_city/<roomid>.yaml` (new Rift Chamber room in Thornwall)
- Create: `_datafiles/world/dogmud/mobs/thornwall_city/<mobid>-riftkeeper_sable.yaml` (new mob)
- Create: `_datafiles/world/dogmud/mobs/thornwall_city/<mobid>-riftkeeper_sable.js` (NPC script)
- Create: `_datafiles/world/dogmud/templates/help/instances.template`
- Modify: `_datafiles/world/dogmud/keywords.yaml` (add instances to help)
- Modify: An existing Thornwall room to add an exit to the new Rift Chamber

Note: The mob ID, room ID, and exit placement need to be determined by the implementer. Check the next available mob/room IDs. The Rift Chamber should connect to an existing Thornwall room (e.g., near the market or main square) via a standard cardinal exit. The room description should convey arcane energy and a dormant stone archway.

- [ ] **Step 1: Create the vendor mob YAML**

First create the **Rift Chamber room** in Thornwall. Use the next available room ID. Connect it to an existing Thornwall room via a cardinal exit (and add the reverse exit on that room). The room should describe a chamber with a dormant stone archway etched with shifting runes.

Then create the **vendor mob**. The mob should be `non_combatant: true`, `charm_immune: true`, spawned in the new Rift Chamber. Use the next available mob ID. The mob needs an `onAsk` script handler.

Example mob YAML (adjust ID):

```yaml
mobid: <next_id>
zone: Thornwall City
non_combatant: true
charm_immune: true
groups:
  - humanoid
activitylevel: 5
idlecommands:
  - 'emote studies a swirling orb of energy, muttering calculations'
  - ''
  - 'emote adjusts the runes carved into a stone archway'
  - ''
character:
  name: riftkeeper Sable
  description: >-
    A gaunt figure in robes that shimmer with barely-contained
    energy. The air around them crackles with static. A stone
    archway behind them hums with dormant power, its surface
    etched with geometric patterns that seem to shift when you
    look away.
  speciesid: 1
  level: 1
  gold: 0
```

- [ ] **Step 2: Create the vendor script**

The script handles `ask <npc> about <zone>` to list available instances, and `ask <npc> <zone> <gold_amount>` to purchase a portal.

```javascript
function onAsk(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (!user) return false;

    var text = (eventDetails.askText || '').toLowerCase().trim();

    // List available zones
    if (text === 'portal' || text === 'portals' || text === 'zones'
        || text === 'instance' || text === 'instances') {
        mob.Command('say I can open rifts to dangerous places, for a price.');
        mob.Command('say Currently I can reach: the Arena and the Planar Oasis.');
        mob.Command('say Tell me the zone and how much gold you are willing to invest.');
        mob.Command('say For example: ask sable arena 500');
        SendUserMessage(user.UserId(),
            '<ansi fg="181">  [You could ask about specific zones, or name a zone and gold amount.]</ansi>');
        return true;
    }

    // Parse "<zone> <gold>" format
    var parts = text.split(' ');
    if (parts.length >= 2) {
        var zoneName = parts[0];
        var goldAmount = parseInt(parts[1], 10);

        if (isNaN(goldAmount) || goldAmount <= 0) {
            mob.Command('say How much gold are you offering? Name a zone and an amount.');
            return true;
        }

        // Validate zone exists and is instanced
        // NOTE: The implementer needs to map player-facing zone names
        // to actual zone template names. For now, hardcode "arena".
        var zoneMap = {
            'arena': 'Instance Arena',
            'oasis': 'Instance Planar Oasis'
        };

        var templateZone = zoneMap[zoneName];
        if (!templateZone) {
            mob.Command('say I do not know of such a place.');
            return true;
        }

        // Check minimum gold (100g minimum for any instance)
        if (goldAmount < 100) {
            mob.Command('say That barely covers the cost of the runes. I need at least 100 gold.');
            return true;
        }

        // Check player has enough gold
        if (user.GetGold() < goldAmount) {
            mob.Command('say You do not have that much gold.');
            return true;
        }

        // Take gold
        user.AddGold(-goldAmount);

        // Get party members for access list
        // NOTE: The implementer must check if there's a scripting API
        // to get party member IDs. If not, this needs a new scripting
        // function exposed. For now, just authorize the buyer.
        // The Go-side CreateZoneInstance handles the actual creation.

        // Signal the Go side to create the instance.
        // This likely needs a new scripting API function:
        //   CreateInstance(zoneName, goldPaid, userId, roomId)
        // The implementer must add this to the scripting API.
        var success = CreateInstance(templateZone, goldAmount,
            user.UserId(), room.GetRoomId());

        if (success) {
            mob.Command('say The rift is open. You and your companions may enter.');
            mob.Command('say It will hold for a time. Do not tarry.');
            SendRoomMessage(room.GetRoomId(),
                'The stone archway flares with energy. A shimmering portal appears.',
                0);
        } else {
            // Refund on failure
            user.AddGold(goldAmount);
            mob.Command('say Something went wrong. The planes resist me. Your gold is returned.');
        }

        return true;
    }

    return false; // fall through to normal dialogue
}
```

**CRITICAL:** The `CreateInstance` function does not exist in the scripting API yet. The implementer MUST add a new function to `internal/scripting/` that:
1. Calls `rooms.CreateZoneInstance(...)` with the correct parameters
2. Gets the party member list from `parties.Get(userId)`
3. Creates the portal in the current room via `room.AddTemporaryExit`
4. Returns success/failure to the script

This is the glue between the JS script and the Go-side instance creation.

- [ ] **Step 3: Create help file**

Create `_datafiles/world/dogmud/templates/help/instances.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">instances</ansi>

Instanced zones are dangerous, private areas you can enter
by paying a Riftkeeper NPC. The more gold you invest, the
tougher the enemies — and the better the potential loot.

<ansi fg="yellow">How It Works:</ansi>

  1. Find a Riftkeeper NPC in town.
  2. Ask about available zones.
  3. Name a zone and how much gold to invest.
  4. A portal opens for you and your current party.

<ansi fg="yellow">Important Rules:</ansi>

  Party up BEFORE purchasing! Only party members at the
  time of purchase can enter the portal. Members added
  later cannot get in.

  The portal stays open for a limited time. When it closes,
  you can still leave through the return portal inside.

  Your companions follow you in automatically.

<ansi fg="yellow">Death:</ansi>

  Some zones allow you to re-enter after dying. Others
  expel you permanently on death. The Riftkeeper will
  tell you which rules apply.

<ansi fg="yellow">Storage:</ansi>

  Loot drops normally from enemies inside. Anything left
  on the ground when the instance closes is lost forever.
  Pick up your loot!

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help party</ansi>, <ansi fg="command">help combat</ansi>
```

- [ ] **Step 4: Add to keywords.yaml**

Add `instances` under the appropriate section in `_datafiles/world/dogmud/keywords.yaml` (likely `general:` or a new `instances:` subsection).

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/mobs/thornwall/ \
  _datafiles/world/dogmud/templates/help/instances.template \
  _datafiles/world/dogmud/keywords.yaml
git commit -m "feat(instances): add Riftkeeper vendor NPC, help file"
```

---

### Task 11: Scripting API — CreateInstance Bridge

**Files:**
- Modify: `internal/scripting/` (add CreateInstance global function)

- [ ] **Step 1: Add CreateInstance to scripting API**

The implementer needs to find where global scripting functions are registered (search for `AddGlobalFunction` or similar in `internal/scripting/`). Add a new function that:

1. Takes `zoneName string, goldPaid int, userId int, roomId int`
2. Gets the user's party members via `parties.Get(userId)`
3. Builds the authorized user list (owner + party members)
4. Calls `rooms.CreateZoneInstance(...)`
5. Creates the entry portal via `room.AddTemporaryExit`
6. Returns `true` on success, `false` on failure

```go
func scriptCreateInstance(zoneName string, goldPaid int, userId int, roomId int) bool {
	// Build authorized user list
	authorizedUsers := []int{userId}
	if p := parties.Get(userId); p != nil {
		for _, memberId := range p.UserIds {
			if memberId != userId {
				authorizedUsers = append(authorizedUsers, memberId)
			}
		}
	}

	inst, err := rooms.CreateZoneInstance(zoneName, goldPaid, userId, authorizedUsers, roomId)
	if err != nil {
		mudlog.Error("CreateInstance", "error", err)
		return false
	}

	// Create entry portal in the overworld room
	owRoom := rooms.LoadRoom(roomId)
	if owRoom != nil {
		owRoom.AddTemporaryExit("instance portal", exit.TemporaryRoomExit{
			RoomId:  inst.EntryRoomId,
			Title:   "instance portal",
			UserId:  0, // system portal, not player-owned
			Expires: inst.PortalDuration,
		})
	}

	return true
}
```

Register this function so it's callable from JS as `CreateInstance(...)`.

- [ ] **Step 2: Compile and commit**

Run: `go build ./...`

```bash
git add internal/scripting/
git commit -m "feat(instances): add CreateInstance scripting API bridge"
```

---

### Task 12: Test Instance Zone Templates — Arena + Planar Oasis

**Files:**
- Create: `_datafiles/world/dogmud/rooms/instance_arena/zone-config.yaml`
- Create: `_datafiles/world/dogmud/rooms/instance_arena/*.yaml` (2-3 stub rooms)
- Create: `_datafiles/world/dogmud/rooms/instance_planar_oasis/zone-config.yaml`
- Create: `_datafiles/world/dogmud/rooms/instance_planar_oasis/*.yaml` (2-3 stub rooms)

These are minimal stubs for testing the framework pipeline — not final
content. Just enough rooms to verify portal creation, entry, mob spawns,
death policy, and cleanup.

- [ ] **Step 1: Create arena zone — ejected death policy, no recall**

Arena zone config:
```yaml
name: Instance Arena
instanced: true
death_policy: ejected
portal_duration: "30m"
entry_room: <first_room_id>
allow_recall: false
```

Create 2-3 rooms:
1. **Arena Entrance** — safe room, return portal spawns here. Short
   description about a blood-stained fighting pit ahead.
2. **Arena Floor** — one combat room with a mob spawn (use an existing
   hostile mob like a bandit or dustwalk creature). This verifies
   mob spawning, combat, and death ejection.

- [ ] **Step 2: Create planar oasis zone — rejoin death policy, recall allowed**

Planar oasis zone config:
```yaml
name: Instance Planar Oasis
instanced: true
death_policy: rejoin
portal_duration: "30m"
entry_room: <first_room_id>
allow_recall: true
```

Create 2-3 rooms:
1. **Oasis Threshold** — safe entry room with return portal.
2. **Shimmering Sands** — one combat room with elemental mob spawns.

This gives you two zones with different death policies and recall
settings for testing both paths.

Use next available room IDs. Folder names: `instance_arena` and
`instance_planar_oasis` (matching `ConvertForFilename` output).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/rooms/instance_arena/
git commit -m "feat(instances): add test arena zone template"
```

---

### Task 13: Integration Test and Verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v 2>&1 | tail -30`

- [ ] **Step 2: Verify clean build**

Run: `go build ./...`

- [ ] **Step 3: Manual integration test**

Start the server and test the full flow:

**Arena (ejected, no recall):**
1. Go to the Rift Chamber in Thornwall
2. `ask sable arena 200` — should create portal, list authorized players
3. Walk through portal — should enter Arena Entrance
4. Walk to Arena Floor — mobs should spawn
5. Try `cast fold-recall` — should be blocked
6. Die — should respawn at death recovery, NOT able to re-enter
7. Have someone unauthorized try the portal — should be rejected

**Planar Oasis (rejoin, recall allowed):**
8. `ask sable oasis 300` — should create a second portal
9. Enter, fight, die — should be able to re-enter through portal
10. Try `cast fold-recall` — should work
11. Re-enter through portal — should still work

**Cleanup:**
12. Leave instance, wait — should see timer warnings
13. After all players leave — instance should clean up
14. Log in as different character, try someone else's portal — blocked

- [ ] **Step 4: Verify all help files**

`help instances`, `help portal` (if created)

- [ ] **Step 5: Final commit if fixups needed**

```bash
git add -A
git commit -m "chore: integration fixups for instanced zones framework"
```
