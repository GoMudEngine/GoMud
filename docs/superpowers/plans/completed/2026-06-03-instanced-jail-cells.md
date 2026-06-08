# Instanced Jail Cells Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace chunk 5.1's shared static holding-cell rooms with per-prisoner single-room instanced jail cells (reusing the instance-zone machinery) that only the prisoner can enter, torn down on release/despawn/death and restored on login.

**Architecture:** A new instanceable `instance_jail_cell` zone (one template room, room id 5107). `internal/justice` (which already imports `internal/rooms`) creates a per-prisoner instance at arrest via new function seams, places the prisoner inside, and tears it down on release. `internal/rooms` gains a `CreateZoneInstanceWithOpts` variant with a `SuppressReturnPortal` flag (no-TTL falls out of an empty `portal_duration`). Despawn/death and login hooks keep the persisted `UntilRound` sentence clock authoritative across logout/restart.

**Tech Stack:** Go; YAML zone/room/help data; existing instance-zone machinery (`internal/rooms/instances.go`, `ephemeral.go`); justice seam-and-test pattern (`internal/justice/arrest.go`).

---

## Verified facts (do not re-derive)

- **Jail record** (`internal/justice/arrest.go:194`): `JailRecord{FineOriginal, DecayPerRound, UntilRound uint64, Faction string, CellRoom int}`. MiscData keys (arrest.go:28-35): `keyJailUntilRound="jail_until_round"`, `keyJailFineOriginal`, `keyJailDecayPerRound`, `keyJailFaction`, `keyJailCellRoom`, `keyJailCrimeIds`. `jailedBuffId=88`, `barracksRoomId=473`.
- **Seams** (arrest.go / justice.go): `cellRoomFn(faction) int` (justice.go:40, reads `FactionDefinition.HoldingCellRoom`), `releaseRoomFn(faction) int` (justice.go:49), `bNowFn() uint64` (round count), `aMoveFn = rooms.MoveToRoom`. Tests stub these.
- `ExecuteArrest(player *characters.Character, userId int, faction string, isMurder bool) bool` (arrest.go:240). `ResolveDetention(player *characters.Character, userId int) bool` (arrest.go:348). `JailInfo(player) (JailRecord, bool)` (arrest.go:205). `player.AddBuffScaled(buffId, float64(rounds))`, `player.RemoveBuff(id)`, `player.SetMiscData(key, val)`, `player.EndAggro()`.
- **Instance machinery** (`internal/rooms`): `CreateZoneInstance(zoneName string, goldPaid int, ownerUserId int, authorizedUsers []int, overworldRoomId int) (*ZoneInstance, error)` (instances.go:282). `ZoneInstance` has `EntryRoomId int`, `InstanceId int`, `AuthorizedUsers []int` + `IsAuthorized`. Return portal added at instances.go:~373-388 via `entryRoom.AddTemporaryExit("return portal", returnExit)`. `GetInstanceRegistry() *InstanceRegistry` (instances.go:258); `(*InstanceRegistry).FindByRoomId(roomId) *ZoneInstance` (85... :107); `.Remove(inst)` (:85). `TryEphemeralCleanup(ephemeralRoomId int) []int` (ephemeral.go:158). `CheckPortalTimers` SKIPS instances with empty `PortalDuration` (instances.go:162). The 2 existing `CreateZoneInstance` callers are `internal/behaviortree/actions_mob.go:252,350`.
- **Zone config** is YAML at `_datafiles/world/dogmud/rooms/<zone_folder>/zone-config.yaml` (`ZoneConfig` in `internal/rooms/zoneconfig.go`: `instanced`, `death_policy`, `portal_duration`, `entry_room`, `allow_recall`). Oasis example: name "Instance Planar Oasis", folder `instance_planar_oasis`.
- **Room description** = raw `r.Description` (rooms.go:1565), NO token substitution. Faction flavor is set by mutating the ephemeral clone's `Description` at arrest.
- **Hooks:** `internal/hooks/PlayerDespawn_TrackingCleanup.go` (despawn-cleanup pattern), `PlayerDespawn_HandleLeave.go`, `PlayerSpawn_HandleJoin.go` (login/enter-world).
- **Free room id:** global next-free rooms id is **5107** (per `python tools/id_inventory.py --type rooms`).
- **Docs:** `internal/justice/context.md`, `internal/rooms/context.md`; helpfiles `_datafiles/world/dogmud/templates/help/arrest.template`, `fine.template`, `payfine.template`.

## Pre-flight

Already on branch `feature/instanced-jail-cells` (spec committed). All work lands here.

---

## Task 1: Jail Cell template zone (data)

**Files:**
- Create: `_datafiles/world/dogmud/rooms/instance_jail_cell/zone-config.yaml`
- Create: `_datafiles/world/dogmud/rooms/instance_jail_cell/5107.yaml`

- [ ] **Step 1: Confirm room id 5107 is free**

Run: `python tools/id_inventory.py --type rooms`
Expected: "global next free (>= max+1): 5107" (or higher). If 5107 is now taken, use the reported next-free id and substitute it everywhere `5107` appears in this plan.

- [ ] **Step 2: Create `instance_jail_cell/zone-config.yaml`**

```yaml
name: Instance Jail Cell
instanced: true
death_policy: ejected
allow_recall: false
entry_room: 5107
```
(Note: `portal_duration` is intentionally omitted → empty → `CheckPortalTimers` skips it: no TTL eviction, no "portal collapsing" warnings. Lifetime is owned by release/despawn teardown.)

- [ ] **Step 3: Create `instance_jail_cell/5107.yaml`** (single cell room, NO exits)

```yaml
roomid: 5107
zone: Instance Jail Cell
title: A Holding Cell
description: >-
  Four close walls of cold, mortared stone press in around a single
  iron-strapped door with no handle on this side. A narrow pallet, a tin
  cup, and a barred slit too high to see through are the only furnishings.
  There is no way out but the law's mercy and the slow passage of time.
nouns:
  door: A heavy iron-strapped door, barred from the far side.
  pallet: A thin straw pallet against the wall.
```
(No `exits:` block — the prisoner cannot walk out. The return portal is also suppressed in Task 3's instance creation. Faction-specific flavor is appended to this description at arrest time in Task 3.)

- [ ] **Step 4: Build + boot-validate the zone loads**

Run: `go generate >/dev/null 2>&1 && go build ./...` (expect exit 0), then boot:
`timeout 45 go run . > /tmp/jail_boot.log 2>&1; grep -iE "panic|fatal|instance_jail_cell|Instance Jail Cell" /tmp/jail_boot.log; grep -i "Server Ready" /tmp/jail_boot.log`
Expected: no panic; `Server Ready` present. The zone loads as a normal room zone (instancing happens on demand at runtime; no load-time error from `instanced: true`).

- [ ] **Step 5: Commit**
```bash
git add _datafiles/world/dogmud/rooms/instance_jail_cell/
git commit -m "feat(justice): instance_jail_cell template zone (single cell room)"
```

---

## Task 2: `CreateZoneInstanceWithOpts` + `SuppressReturnPortal`

**Files:**
- Modify: `internal/rooms/instances.go`
- Test: `internal/rooms/instances_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/rooms/instances_test.go` (uses the existing test harness in that file — mirror an existing `CreateZoneInstance` test for setup, e.g. the one that creates an oasis/arena instance):

```go
func TestCreateZoneInstanceWithOpts_SuppressReturnPortal(t *testing.T) {
	// Default: return portal present.
	def, err := CreateZoneInstance("Instance Arena", 1, 9001, []int{9001}, 460)
	if err != nil {
		t.Fatalf("default create failed: %v", err)
	}
	defEntry := LoadRoom(def.EntryRoomId)
	if defEntry == nil || !defEntry.HasExit("return portal") {
		t.Fatalf("default instance should have a return portal exit")
	}
	GetInstanceRegistry().Remove(def)
	TryEphemeralCleanup(def.EntryRoomId)

	// SuppressReturnPortal: no return portal.
	supp, err := CreateZoneInstanceWithOpts("Instance Arena", 1, 9002, []int{9002}, 460,
		ZoneInstanceOpts{SuppressReturnPortal: true})
	if err != nil {
		t.Fatalf("opts create failed: %v", err)
	}
	suppEntry := LoadRoom(supp.EntryRoomId)
	if suppEntry == nil {
		t.Fatalf("entry room missing")
	}
	if suppEntry.HasExit("return portal") {
		t.Fatalf("SuppressReturnPortal instance must NOT have a return portal exit")
	}
	GetInstanceRegistry().Remove(supp)
	TryEphemeralCleanup(supp.EntryRoomId)
}
```
(If `Room` has no `HasExit(name string) bool` helper, use the equivalent check the file already uses for exits — e.g. inspect `suppEntry.Exits["return portal"]`/`GetTemporaryExits()`. Match the existing test style in the file.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run TestCreateZoneInstanceWithOpts_SuppressReturnPortal -v`
Expected: FAIL — `CreateZoneInstanceWithOpts` / `ZoneInstanceOpts` undefined (compile error → add minimal stub if needed to see a real assertion fail; see Step 3).

- [ ] **Step 3: Implement the options variant**

In `internal/rooms/instances.go`, add the options type and refactor:

```go
// ZoneInstanceOpts carries optional behavior toggles for instance creation.
// The zero value preserves the original CreateZoneInstance behavior.
type ZoneInstanceOpts struct {
	// SuppressReturnPortal skips adding the return-to-overworld exit in the
	// entry room. Used by confinement instances (jail cells) where a return
	// exit would be an escape hatch.
	SuppressReturnPortal bool
}
```

Rename the current `CreateZoneInstance` body to `CreateZoneInstanceWithOpts` by adding a trailing `opts ZoneInstanceOpts` parameter, and make `CreateZoneInstance` delegate:

```go
func CreateZoneInstance(zoneName string, goldPaid int, ownerUserId int, authorizedUsers []int, overworldRoomId int) (*ZoneInstance, error) {
	return CreateZoneInstanceWithOpts(zoneName, goldPaid, ownerUserId, authorizedUsers, overworldRoomId, ZoneInstanceOpts{})
}

func CreateZoneInstanceWithOpts(zoneName string, goldPaid int, ownerUserId int, authorizedUsers []int, overworldRoomId int, opts ZoneInstanceOpts) (*ZoneInstance, error) {
	// ... existing body unchanged UP TO the return-portal block ...
}
```

Gate the return-portal block (instances.go:~373-388) on the flag:

```go
	// 4. Add a return portal in the ephemeral entry room (unless suppressed).
	if !opts.SuppressReturnPortal {
		returnExit := exit.RoomExit{ /* ...existing fields... */ }
		entryRoom.AddTemporaryExit("return portal", returnExit)
	}
```
(Leave the 2 existing callers in `internal/behaviortree/actions_mob.go:252,350` untouched — they call `CreateZoneInstance`, which now delegates with zero opts.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run TestCreateZoneInstanceWithOpts_SuppressReturnPortal -v`
Expected: PASS. Then `go test ./internal/rooms/` — all pass.

- [ ] **Step 5: Commit**
```bash
git add internal/rooms/instances.go internal/rooms/instances_test.go
git commit -m "feat(rooms): CreateZoneInstanceWithOpts + SuppressReturnPortal"
```

---

## Task 3: `JailRecord.InstanceId` + arrest creates the cell instance

**Files:**
- Modify: `internal/justice/arrest.go`
- Test: `internal/justice/arrest_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/justice/arrest_test.go` (the file already stubs seams like `cellRoomFn`/`aMoveFn`; mirror that style). Stub the new seams:

```go
func TestExecuteArrest_UsesInstancedCell(t *testing.T) {
	// Arrange seams.
	origCell, origMove := cellRoomFn, aMoveFn
	origCreate, origDesc := aCreateCellFn, aSetCellDescFn
	defer func() { cellRoomFn, aMoveFn, aCreateCellFn, aSetCellDescFn = origCell, origMove, origCreate, origDesc }()

	cellRoomFn = func(string) int { return 473 } // static fallback exists
	var movedTo int
	aMoveFn = func(userId, roomId int) error { movedTo = roomId; return nil }
	var createdFor int
	aCreateCellFn = func(prisonerUserId, releaseRoomId int) (int, bool) {
		createdFor = prisonerUserId
		return 60001, true // pretend ephemeral entry room id
	}
	descSet := ""
	aSetCellDescFn = func(roomId int, desc string) { descSet = desc }

	ch := characters.New() // or the test helper the file already uses
	ok := ExecuteArrest(ch, 42, "thornwall_guards", false)

	if !ok {
		t.Fatalf("arrest should succeed")
	}
	if createdFor != 42 {
		t.Fatalf("expected cell instance created for prisoner 42, got %d", createdFor)
	}
	rec, jailed := JailInfo(ch)
	if !jailed || rec.InstanceId != 60001 {
		t.Fatalf("expected InstanceId 60001 stamped, got %+v", rec)
	}
	if movedTo != 60001 {
		t.Fatalf("expected prisoner moved to the instanced cell 60001, got %d", movedTo)
	}
	if descSet == "" {
		t.Fatalf("expected faction-flavored cell description to be set")
	}
}

func TestExecuteArrest_FallsBackToStaticCellOnInstanceFailure(t *testing.T) {
	origCell, origMove, origCreate := cellRoomFn, aMoveFn, aCreateCellFn
	defer func() { cellRoomFn, aMoveFn, aCreateCellFn = origCell, origMove, origCreate }()

	cellRoomFn = func(string) int { return 473 }
	var movedTo int
	aMoveFn = func(userId, roomId int) error { movedTo = roomId; return nil }
	aCreateCellFn = func(int, int) (int, bool) { return 0, false } // instance creation fails

	ch := characters.New()
	ok := ExecuteArrest(ch, 42, "thornwall_guards", false)
	if !ok {
		t.Fatalf("arrest should still succeed via static fallback")
	}
	rec, _ := JailInfo(ch)
	if rec.InstanceId != 0 || rec.CellRoom != 473 || movedTo != 473 {
		t.Fatalf("expected static-cell fallback (cell 473, InstanceId 0), got %+v movedTo=%d", rec, movedTo)
	}
}
```
(Use whatever `characters` test constructor `arrest_test.go` already uses to build a `*characters.Character`; `characters.New()` is a placeholder for that — match the file.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/justice/ -run TestExecuteArrest_Uses -v`
Expected: FAIL — `aCreateCellFn`/`aSetCellDescFn`/`rec.InstanceId` undefined.

- [ ] **Step 3: Implement**

In `internal/justice/arrest.go`:

(a) Add the MiscData key (in the const block at line 28):
```go
	keyJailInstanceId    = "jail_instance_id"
```

(b) Add `InstanceId int` to `JailRecord` (line 194) and read it in `JailInfo` (line 205):
```go
type JailRecord struct {
	FineOriginal  int
	DecayPerRound int
	UntilRound    uint64
	Faction       string
	CellRoom      int
	InstanceId    int
}
```
In `JailInfo`, after reading `cell`:
```go
	instId, _ := miscDataInt(player.MiscData, keyJailInstanceId)
	// ... add InstanceId: instId to the returned JailRecord{...}
```

(c) Add the new seams near the other release-path seams (after arrest.go:79):
```go
// jailCellZoneName is the instanceable zone cloned per prisoner.
const jailCellZoneName = "Instance Jail Cell"

// aCreateCellFn spins up a single-room jail-cell instance for the prisoner and
// returns the ephemeral entry room id. Returns (0,false) on failure so the
// caller can fall back to a static cell. Tests override.
var aCreateCellFn = func(prisonerUserId, releaseRoomId int) (int, bool) {
	inst, err := rooms.CreateZoneInstanceWithOpts(
		jailCellZoneName, 1, prisonerUserId, []int{prisonerUserId}, releaseRoomId,
		rooms.ZoneInstanceOpts{SuppressReturnPortal: true},
	)
	if err != nil || inst == nil {
		return 0, false
	}
	return inst.EntryRoomId, true
}

// aTeardownCellFn removes a jail-cell instance and frees its ephemeral chunk.
// Tests override.
var aTeardownCellFn = func(entryRoomId int) {
	reg := rooms.GetInstanceRegistry()
	if inst := reg.FindByRoomId(entryRoomId); inst != nil {
		reg.Remove(inst)
	}
	rooms.TryEphemeralCleanup(entryRoomId)
}

// aSetCellDescFn mutates an ephemeral cell room's description to weave in the
// arresting faction's flavor. Tests override.
var aSetCellDescFn = func(roomId int, desc string) {
	if r := rooms.LoadRoom(roomId); r != nil {
		r.Description = desc
	}
}
```

(d) In `ExecuteArrest`, replace the top cell resolution (arrest.go:241-244) with the instance-first logic:
```go
	// Resolve the cell: prefer a per-prisoner instanced cell, fall back to the
	// faction's static holding cell if instancing is unavailable.
	staticCell := cellRoomFn(faction)
	releaseRoom := releaseRoomFn(faction)
	if releaseRoom == 0 {
		releaseRoom = barracksRoomId
	}

	instanceId := 0
	cell := 0
	if entry, ok := aCreateCellFn(userId, releaseRoom); ok {
		instanceId = entry
		cell = entry
	} else if staticCell != 0 {
		cell = staticCell
	} else {
		return false // no cell available at all
	}
```
Then, after stamping the other jail keys (after arrest.go:269), stamp the instance id and weave the faction flavor:
```go
	player.SetMiscData(keyJailInstanceId, instanceId)
	if instanceId != 0 {
		factionName := faction
		if d := factions.GetDefinition(faction); d != nil && d.DisplayName != "" {
			factionName = d.DisplayName
		}
		aSetCellDescFn(instanceId, fmt.Sprintf(
			"Four close walls of cold, mortared stone press in around a single "+
				"iron-strapped door with no handle on this side. The seal of the %s "+
				"is stamped into the iron. There is no way out but the law's mercy "+
				"and the slow passage of time.", factionName))
	}
```
Keep the existing `keyJailCellRoom` stamp set to `cell` (so the static-path and instanced-path both record where the prisoner is), and the existing buff/EndAggro/`aMoveFn(userId, cell)` flow unchanged.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/justice/ -run TestExecuteArrest -v`
Expected: PASS (both new tests + any existing ExecuteArrest tests). Then `go test ./internal/justice/`.

- [ ] **Step 5: Commit**
```bash
git add internal/justice/arrest.go internal/justice/arrest_test.go
git commit -m "feat(justice): arrest creates per-prisoner instanced cell (static fallback)"
```

---

## Task 4: `ResolveDetention` tears down the instance

**Files:**
- Modify: `internal/justice/arrest.go`
- Test: `internal/justice/arrest_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestResolveDetention_TearsDownInstance(t *testing.T) {
	origMove, origTeardown := aMoveFn, aTeardownCellFn
	origResolve, origCrimes := aResolveCrimeFn, aCrimesForFactionFn
	defer func() {
		aMoveFn, aTeardownCellFn = origMove, origTeardown
		aResolveCrimeFn, aCrimesForFactionFn = origResolve, origCrimes
	}()
	aMoveFn = func(int, int) error { return nil }
	aCrimesForFactionFn = func(string, bool) []crimes.CrimeRecord { return nil }
	var torndown int
	aTeardownCellFn = func(entryRoomId int) { torndown = entryRoomId }

	ch := characters.New()
	ch.SetMiscData(keyJailUntilRound, uint64(100))
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailInstanceId, 60001)

	if !ResolveDetention(ch, 42) {
		t.Fatalf("resolve should succeed")
	}
	if torndown != 60001 {
		t.Fatalf("expected instance 60001 torn down, got %d", torndown)
	}
	if _, jailed := JailInfo(ch); jailed {
		t.Fatalf("jail record should be cleared")
	}
}

func TestResolveDetention_LegacyStaticCellNoTeardown(t *testing.T) {
	origMove, origTeardown := aMoveFn, aTeardownCellFn
	origCrimes := aCrimesForFactionFn
	defer func() { aMoveFn, aTeardownCellFn, aCrimesForFactionFn = origMove, origTeardown, origCrimes }()
	aMoveFn = func(int, int) error { return nil }
	aCrimesForFactionFn = func(string, bool) []crimes.CrimeRecord { return nil }
	teardownCalled := false
	aTeardownCellFn = func(int) { teardownCalled = true }

	ch := characters.New()
	ch.SetMiscData(keyJailUntilRound, uint64(100))
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailInstanceId, 0) // legacy static cell

	ResolveDetention(ch, 42)
	if teardownCalled {
		t.Fatalf("legacy static-cell release must NOT call instance teardown")
	}
}
```
(Match `crimes.CrimeRecord`/`AllForFaction` return type to the real signature; if the existing tests already stub `aCrimesForFactionFn`, copy their form exactly.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/justice/ -run TestResolveDetention_TearsDown -v`
Expected: FAIL — teardown not invoked (instance id ignored).

- [ ] **Step 3: Implement**

In `ResolveDetention` (arrest.go:348), after reading `faction` and before `ClearFactionRecord` (or right after it), add:
```go
	// Tear down the per-prisoner cell instance if this was an instanced cell.
	if instId, _ := miscDataInt(player.MiscData, keyJailInstanceId); instId != 0 {
		aTeardownCellFn(instId)
	}
```
And in the MiscData-clearing block (arrest.go:372-377), add:
```go
	player.SetMiscData(keyJailInstanceId, nil)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/justice/ -run TestResolveDetention -v` then `go test ./internal/justice/`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/justice/arrest.go internal/justice/arrest_test.go
git commit -m "feat(justice): ResolveDetention tears down instanced cell (dual-path)"
```

---

## Task 5: Despawn/death teardown + saved-room rewrite

**Files:**
- Modify: `internal/justice/arrest.go` (add exported helper)
- Test: `internal/justice/arrest_test.go`
- Create: `internal/hooks/PlayerDespawn_JailCleanup.go`
- Modify: `internal/hooks/hooks.go` (register the listener — follow how `PlayerDespawn_TrackingCleanup` is registered)

- [ ] **Step 1: Write the failing test for the justice helper**

```go
func TestHandleJailedDespawn_TearsDownKeepsRecordRewritesRoom(t *testing.T) {
	origTeardown, origCell := aTeardownCellFn, cellRoomFn
	defer func() { aTeardownCellFn, cellRoomFn = origTeardown, origCell }()
	var torndown int
	aTeardownCellFn = func(id int) { torndown = id }
	cellRoomFn = func(string) int { return 473 } // static fallback room

	ch := characters.New()
	ch.SetMiscData(keyJailUntilRound, uint64(9999))
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailInstanceId, 60001)

	HandleJailedDespawn(ch, 42)

	if torndown != 60001 {
		t.Fatalf("expected instance torn down, got %d", torndown)
	}
	// Jail record (sentence) preserved.
	if _, jailed := JailInfo(ch); !jailed {
		t.Fatalf("jail record must survive logout so the sentence persists")
	}
	// InstanceId cleared (the ephemeral room is gone).
	if instId, _ := miscDataInt(ch.MiscData, keyJailInstanceId); instId != 0 {
		t.Fatalf("InstanceId must be cleared on despawn, got %d", instId)
	}
	// Saved room rewritten to the static fallback (no dead ephemeral room).
	if ch.RoomId != 473 {
		t.Fatalf("expected saved RoomId rewritten to fallback 473, got %d", ch.RoomId)
	}
}
```
(Use the real "saved room" field/setter on `Character` — if it's not `RoomId`, use the correct one; grep `characters` for the room-id field used by the save/load path.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/justice/ -run TestHandleJailedDespawn -v`
Expected: FAIL — `HandleJailedDespawn` undefined.

- [ ] **Step 3: Implement the justice helper**

In `internal/justice/arrest.go`:
```go
// HandleJailedDespawn is called when a jailed player logs out / disconnects.
// The ephemeral cell instance can't survive the logout, so tear it down, but
// KEEP the jail record (the UntilRound sentence clock is absolute and persists
// with the character). Rewrite the saved room to the faction's static fallback
// cell so the character is never saved pointing at a now-dead ephemeral room;
// the login hook (RestoreJailOnLogin) re-instances or releases on return.
func HandleJailedDespawn(player *characters.Character, userId int) {
	if player == nil || player.MiscData == nil {
		return
	}
	if _, ok := miscDataRound(player.MiscData, keyJailUntilRound); !ok {
		return // not jailed
	}
	if instId, _ := miscDataInt(player.MiscData, keyJailInstanceId); instId != 0 {
		aTeardownCellFn(instId)
		player.SetMiscData(keyJailInstanceId, 0)
	}
	// Rewrite saved room so login never lands in a dead ephemeral room.
	faction, _ := miscDataString(player.MiscData, keyJailFaction)
	fallback := cellRoomFn(faction)
	if fallback == 0 {
		fallback = barracksRoomId
	}
	player.RoomId = fallback // use the real saved-room field/setter
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/justice/ -run TestHandleJailedDespawn -v`
Expected: PASS.

- [ ] **Step 5: Create the despawn hook**

Create `internal/hooks/PlayerDespawn_JailCleanup.go`, modeled on `PlayerDespawn_TrackingCleanup.go` (same event subscription + signature). On player-despawn, resolve the `*characters.Character` + userId from the event and call `justice.HandleJailedDespawn(char, userId)`. Register it in `internal/hooks/hooks.go` exactly as `PlayerDespawn_TrackingCleanup` is registered.

- [ ] **Step 6: Build + boot (hook wiring)**

Run: `go build ./...` (exit 0), then boot and confirm no panic / `Server Ready` (the hook registers cleanly).

- [ ] **Step 7: Commit**
```bash
git add internal/justice/arrest.go internal/justice/arrest_test.go internal/hooks/PlayerDespawn_JailCleanup.go internal/hooks/hooks.go
git commit -m "feat(justice): tear down cell + preserve sentence on jailed despawn"
```

---

## Task 6: Login restore (re-instance or release)

**Files:**
- Modify: `internal/justice/arrest.go` (add `RestoreJailOnLogin`)
- Test: `internal/justice/arrest_test.go`
- Modify: `internal/hooks/PlayerSpawn_HandleJoin.go` (call the restore on enter-world)

- [ ] **Step 1: Write the failing tests**

```go
func TestRestoreJailOnLogin_ReleasesWhenSentenceServedOffline(t *testing.T) {
	origNow, origMove, origCrimes := bNowFn, aMoveFn, aCrimesForFactionFn
	defer func() { bNowFn, aMoveFn, aCrimesForFactionFn = origNow, origMove, origCrimes }()
	bNowFn = func() uint64 { return 200 }            // now past sentence
	aMoveFn = func(int, int) error { return nil }
	aCrimesForFactionFn = func(string, bool) []crimes.CrimeRecord { return nil }

	ch := characters.New()
	ch.SetMiscData(keyJailUntilRound, uint64(100))   // already expired
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailInstanceId, 0)

	RestoreJailOnLogin(ch, 42)
	if _, jailed := JailInfo(ch); jailed {
		t.Fatalf("expired sentence should be released on login")
	}
}

func TestRestoreJailOnLogin_ReInstancesWhenStillServing(t *testing.T) {
	origNow, origMove, origCreate := bNowFn, aMoveFn, aCreateCellFn
	origRelease := releaseRoomFn
	defer func() { bNowFn, aMoveFn, aCreateCellFn, releaseRoomFn = origNow, origMove, origCreate, origRelease }()
	bNowFn = func() uint64 { return 50 }             // still serving (until 100)
	releaseRoomFn = func(string) int { return 473 }
	var movedTo int
	aMoveFn = func(_, roomId int) error { movedTo = roomId; return nil }
	aCreateCellFn = func(int, int) (int, bool) { return 60002, true }

	ch := characters.New()
	ch.SetMiscData(keyJailUntilRound, uint64(100))
	ch.SetMiscData(keyJailFaction, "thornwall_guards")
	ch.SetMiscData(keyJailInstanceId, 0)             // stale (instance gone on logout)

	RestoreJailOnLogin(ch, 42)
	rec, jailed := JailInfo(ch)
	if !jailed || rec.InstanceId != 60002 {
		t.Fatalf("expected fresh instance 60002, got %+v", rec)
	}
	if movedTo != 60002 {
		t.Fatalf("expected prisoner placed in fresh cell 60002, got %d", movedTo)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/justice/ -run TestRestoreJailOnLogin -v`
Expected: FAIL — `RestoreJailOnLogin` undefined.

- [ ] **Step 3: Implement**

In `internal/justice/arrest.go`:
```go
// RestoreJailOnLogin reconciles a returning player's jail state. The sentence
// clock (UntilRound) is absolute and persists across logout/restart; the
// ephemeral cell does not. On login: if the sentence elapsed while away, release
// the player; otherwise re-create a fresh cell instance, refresh the Jailed buff
// to the remaining rounds, and place the player inside. Safe to call for any
// logging-in player (no-op when not jailed).
func RestoreJailOnLogin(player *characters.Character, userId int) {
	if player == nil || player.MiscData == nil {
		return
	}
	until, ok := miscDataRound(player.MiscData, keyJailUntilRound)
	if !ok {
		return // not jailed
	}
	now := bNowFn()
	if until <= now {
		ResolveDetention(player, userId) // sentence served while away → release
		return
	}

	faction, _ := miscDataString(player.MiscData, keyJailFaction)
	releaseRoom := releaseRoomFn(faction)
	if releaseRoom == 0 {
		releaseRoom = barracksRoomId
	}

	entry, created := aCreateCellFn(userId, releaseRoom)
	if !created {
		// Fall back to the static cell if instancing is unavailable.
		if sc := cellRoomFn(faction); sc != 0 {
			entry = sc
		} else {
			entry = releaseRoom
		}
	} else {
		// Re-weave faction flavor onto the fresh ephemeral cell.
		factionName := faction
		if d := factions.GetDefinition(faction); d != nil && d.DisplayName != "" {
			factionName = d.DisplayName
		}
		aSetCellDescFn(entry, fmt.Sprintf(
			"Four close walls of cold, mortared stone press in around a single "+
				"iron-strapped door with no handle on this side. The seal of the %s "+
				"is stamped into the iron. There is no way out but the law's mercy "+
				"and the slow passage of time.", factionName))
	}

	instanceId := 0
	if created {
		instanceId = entry
	}
	player.SetMiscData(keyJailInstanceId, instanceId)
	player.SetMiscData(keyJailCellRoom, entry)

	// Refresh the Jailed buff to the remaining sentence.
	player.RemoveBuff(jailedBuffId)
	_ = player.AddBuffScaled(jailedBuffId, float64(until-now))

	_ = aMoveFn(userId, entry)
}
```

- [ ] **Step 4: Run to verify they pass**

Run: `go test ./internal/justice/ -run TestRestoreJailOnLogin -v` then `go test ./internal/justice/`
Expected: PASS.

- [ ] **Step 5: Wire the login hook**

In `internal/hooks/PlayerSpawn_HandleJoin.go`, after the player's character + userId are available on enter-world, call `justice.RestoreJailOnLogin(char, userId)`. (Place it after the character is fully loaded but before/at room placement, so the restore's `aMoveFn` placement is authoritative. Match the file's existing structure; if `PlayerSpawn_HandleJoin` isn't the right seam for "character fully loaded", use the same enter-world point the despawn counterpart pairs with.)

- [ ] **Step 6: Build + boot**

Run: `go build ./...` (exit 0), boot, confirm no panic / `Server Ready`.

- [ ] **Step 7: Commit**
```bash
git add internal/justice/arrest.go internal/justice/arrest_test.go internal/hooks/PlayerSpawn_HandleJoin.go
git commit -m "feat(justice): restore jail state on login (re-instance or release)"
```

---

## Task 7: context.md updates

**Files:**
- Modify: `internal/justice/context.md`
- Modify: `internal/rooms/context.md`

- [ ] **Step 1: Update `internal/justice/context.md`**

Add a subsection documenting instanced cells: arrest now creates a per-prisoner `Instance Jail Cell` instance via `aCreateCellFn` (`SuppressReturnPortal`, empty `portal_duration` → no TTL), stores `InstanceId` in the jail record, and falls back to the faction static `HoldingCellRoom` on failure. `ResolveDetention` tears the instance down (`aTeardownCellFn`). `HandleJailedDespawn` (logout) tears down but preserves the sentence + rewrites the saved room; `RestoreJailOnLogin` re-instances or releases based on the persisted `UntilRound`. Note the seam vars (`aCreateCellFn`/`aTeardownCellFn`/`aSetCellDescFn`) follow the existing test-override pattern.

- [ ] **Step 2: Update `internal/rooms/context.md`**

Document `CreateZoneInstanceWithOpts` + `ZoneInstanceOpts{SuppressReturnPortal}` (and that the legacy `CreateZoneInstance` delegates with zero opts), and that an empty `portal_duration` makes `CheckPortalTimers` skip the instance (no TTL/no warnings) — used by confinement instances like jail cells.

- [ ] **Step 3: Commit**
```bash
git add internal/justice/context.md internal/rooms/context.md
git commit -m "docs(context): document instanced jail cells + CreateZoneInstanceWithOpts"
```

---

## Task 8: Helpfile updates

**Files:**
- Modify: `_datafiles/world/dogmud/templates/help/arrest.template`
- Modify: `_datafiles/world/dogmud/templates/help/fine.template`
- Modify: `_datafiles/world/dogmud/templates/help/payfine.template`

- [ ] **Step 1: Read all three current templates** to match tone/format (80-col wrap, no raw numbers).

- [ ] **Step 2: Update `arrest.template`** to mention that an arrested prisoner is taken to a private holding cell they cannot leave until the fine is paid or the sentence is served, and that logging out does not end the sentence (time is served while away; you return to the cell — or walk free if the term elapsed). Keep it in-world voice, ≤80 cols, no numbers.

- [ ] **Step 3: Update `fine.template` and `payfine.template`** so they read correctly for the private-cell flow (you view your fine with `fine` and clear it with `payfine`; paying releases you from the cell). Only adjust wording that the cell change makes inaccurate; do not invent mechanics.

- [ ] **Step 4: Boot + verify help renders** (optional spot check): boot, and confirm no template-load panic for these three files.

- [ ] **Step 5: Commit**
```bash
git add _datafiles/world/dogmud/templates/help/arrest.template _datafiles/world/dogmud/templates/help/fine.template _datafiles/world/dogmud/templates/help/payfine.template
git commit -m "docs(help): jail/fine/payfine reflect private instanced cells"
```

---

## Task 9: Final integration boot + full test suite

**Files:** none (verification only)

- [ ] **Step 1: Full test suite**

Run: `go test ./...`
Expected: all packages pass.

- [ ] **Step 2: Clean-instance boot**

Run:
```
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/* 2>/dev/null
go generate >/dev/null 2>&1
timeout 45 go run . > /tmp/jail_boot.log 2>&1
grep -iE "panic|fatal error" /tmp/jail_boot.log
grep -iE "Instance Jail Cell|instance_jail_cell|Server Ready" /tmp/jail_boot.log
```
Expected: no panic; zone loaded; `Server Ready`.

- [ ] **Step 3: Commit (if any incidental fixups)** — otherwise nothing to commit.

---

## Manual smoke (deferred to user)

1. Commit a crime in a guarded town, get arrested → confirm you're alone in a cell with the arresting faction's flavor, no exits, recall blocked, and no guard follows you in.
2. `fine` to see the fine, `payfine` → released to the faction release room; (admin) confirm the ephemeral chunk was freed.
3. Get re-arrested, **log out mid-sentence**, log back in → you resume in a fresh private cell with the remaining sentence.
4. Get arrested, log out, wait past the sentence (or admin-advance rounds), log back in → you return **free** at the release room.
5. Confirm a second simultaneous prisoner gets their **own** separate cell (no shared room).

## Self-review notes

- **Spec coverage:** A→Task 1; B→Task 2; C→Task 3; D→Task 4; E→Task 5; G→Task 6; F (static retention)→preserved by the fallback in Tasks 3/5/6 (no static rooms deleted) + validation in Task 9; helpfiles/context.md (user request)→Tasks 7-8.
- **No placeholders:** all code/test bodies inline. Two spots say "match the file's existing X" (the `characters` test constructor; the exact saved-room field/setter; the `crimes.CrimeRecord` return type) — these are real-codebase lookups the implementer must confirm, not invented APIs; each names exactly what to confirm.
- **Type consistency:** `aCreateCellFn(prisonerUserId, releaseRoomId int) (int, bool)`, `aTeardownCellFn(entryRoomId int)`, `aSetCellDescFn(roomId int, desc string)`, `keyJailInstanceId`, `JailRecord.InstanceId`, `CreateZoneInstanceWithOpts(...)`, `ZoneInstanceOpts{SuppressReturnPortal}` used consistently across Tasks 2-6.
