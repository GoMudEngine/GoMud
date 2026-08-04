# Spawn-List Editor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Author a room's spawn list (mobs, items, gold, per-spawn overrides, respawn rate) from the `/build` Rooms inspector, and strip the 36 inert `cooldown:` lines that have never done anything.

**Architecture:** No new GMCP verb. `buildRoomDetail` carries the spawn list out, `roomUpdateReq` carries it back in, and validation lives in a pure function behind the existing `buildDeps` seam so it unit-tests against the fake world.

**Tech Stack:** Go, testify `assert` in `internal/rooms`, plain `testing` in `modules/gmcp`, vanilla JS in `builder.js`/`build.html`.

**Spec:** `docs/superpowers/specs/2026-07-25-spawn-list-editor-design.md`

---

## Verified reference

**`rooms.SpawnInfo`** (`internal/rooms/spawninfo.go`) — editable fields:
`MobId`, `Container`, `ItemId`, `Gold`, `Message`, `Name`, `ForceHostile`,
`MaxWander`, `IdleCommands`, `ScriptTag`, `QuestFlags`, `BuffIds`, `StatPool`,
`StatPoolMod`, `RespawnRate`. `InstanceId` and `DespawnedRound` are
`yaml:"-"` runtime tracking — **not editable, must survive a save**.

**Existing seams to extend, not replace:**

```go
type roomUpdateReq struct   // gmcp.Build.go:62  — add Spawns
type buildRoomDetail struct // gmcp.Build.go:140 — add Spawns
func buildRoomUpdate(d buildDeps, req roomUpdateReq) BuildResult // applies fields then d.save
type buildDeps struct       // gmcp.Build.go:167 — add mobExists/itemExists/buffExists
```

**Validators available:**

```go
mobs.GetMobSpec(mobId mobs.MobId) *mobs.Mob   // mobs.go:688  — nil = unknown
items.GetItemSpec(itemId int) *items.ItemSpec // itemspec.go:667 — nil = unknown
buffs.GetBuffSpec(buffId int) *buffs.BuffSpec // buffspec.go:165 — nil = unknown
rooms.Room.Containers map[string]Container    // rooms.go:94
```

**Respawn-rate validation — round-trip, do NOT reimplement the parser.**
`gametime.GameDate.AddPeriod(periodStr) uint64` (`gametime.go:277`) **never
errors**: it returns the caller's own round number for both an empty string
and unparseable garbage. So a typo'd respawn rate means *respawn immediately*,
silently. The parser matches units on 3-character prefixes (`yea`, `mon`,
`wee`, `day`/`dai`, `hou`, …) with `real`/`irl`/`game` modifiers — duplicating
that list would drift. Validate by calling it:

```go
// non-empty period that does not advance the round number = not understood
if period != "" && gd.AddPeriod(period) <= gd.RoundNumber { reject }
```

**Default respawn rate is `15 real minutes`** when unset
(`internal/rooms/rooms.go:2591`).

**Spawn lists are NOT shadowed by instance saves** — `SpawnInfo` is tagged
`instance:"skip"` (`rooms.go:103`) and `restoreSkipTaggedFields`
(`save_and_load.go:137`) copies it back from the template after the overlay.
No instance wipe is needed to see an edit take effect.

---

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/rooms/spawninfo_validate.go` **(create)** | `ValidateSpawnEntry` — pure policy |
| `internal/rooms/spawninfo_validate_test.go` **(create)** | Its tests |
| `modules/gmcp/gmcp.Build.go` **(modify)** | `Spawns` on detail + update req, existence deps, apply-on-save |
| `modules/gmcp/gmcp.Build_test.go` **(modify)** | Handler tests |
| `tools/strip_spawn_cooldown.py` **(create)** | One-off migration |
| `boot_smoke_test.go` **(modify)** | Delete the baseline entry |
| `_datafiles/html/public/static/js/builder.js` **(modify)** | Spawns section |
| `CLAUDE.md` **(modify)** | Correct the stale instance-save claim |

---

### Task 1: Spawn entry validation (pure policy)

**Files:** Create `internal/rooms/spawninfo_validate.go`, `internal/rooms/spawninfo_validate_test.go`

- [ ] **Step 1: Write the failing test**

```go
package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSpawnEntry_KindIsExclusive(t *testing.T) {
	known := spawnValidators{
		mobExists:  func(int) bool { return true },
		itemExists: func(int) bool { return true },
		buffExists: func(int) bool { return true },
		periodOK:   func(string) bool { return true },
		containers: map[string]struct{}{"chest": {}},
	}

	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{MobId: 1}, known))
	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{ItemId: 1}, known))
	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{Gold: 5}, known))

	// Exactly one kind.
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{}, known), "an entry must spawn something")
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, ItemId: 1}, known))
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, Gold: 5}, known))
}

func TestValidateSpawnEntry_ContainerRules(t *testing.T) {
	known := spawnValidators{
		mobExists:  func(int) bool { return true },
		itemExists: func(int) bool { return true },
		buffExists: func(int) bool { return true },
		periodOK:   func(string) bool { return true },
		containers: map[string]struct{}{"chest": {}},
	}

	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{ItemId: 1, Container: "chest"}, known))
	// A mob cannot spawn into a container.
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, Container: "chest"}, known))
	// The container must exist in THIS room.
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{ItemId: 1, Container: "barrel"}, known))
}

func TestValidateSpawnEntry_UnknownReferences(t *testing.T) {
	none := spawnValidators{
		mobExists:  func(int) bool { return false },
		itemExists: func(int) bool { return false },
		buffExists: func(int) bool { return false },
		periodOK:   func(string) bool { return true },
	}
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 999}, none))
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{ItemId: 999}, none))

	badBuff := spawnValidators{
		mobExists:  func(int) bool { return true },
		itemExists: func(int) bool { return true },
		buffExists: func(int) bool { return false },
		periodOK:   func(string) bool { return true },
	}
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, BuffIds: []int{404}}, badBuff))
}

// An unparseable respawn rate does not error at runtime — AddPeriod returns
// the caller's own round number, so the mob respawns IMMEDIATELY. Catch it
// here, where an author can still see it.
func TestValidateSpawnEntry_RespawnRateMustParse(t *testing.T) {
	v := spawnValidators{
		mobExists:  func(int) bool { return true },
		itemExists: func(int) bool { return true },
		buffExists: func(int) bool { return true },
		periodOK:   func(p string) bool { return p == "5 real minutes" },
	}
	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, RespawnRate: "5 real minutes"}, v))
	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, RespawnRate: ""}, v), "empty means the 15-minute default")
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, RespawnRate: "banana"}, v))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run TestValidateSpawnEntry -v`
Expected: FAIL — `undefined: spawnValidators`

- [ ] **Step 3: Write minimal implementation**

Create `internal/rooms/spawninfo_validate.go`:

```go
package rooms

import "fmt"

// spawnValidators injects the existence checks so the spawn policy is
// testable without a loaded world.
type spawnValidators struct {
	mobExists  func(id int) bool
	itemExists func(id int) bool
	buffExists func(id int) bool
	periodOK   func(period string) bool
	containers map[string]struct{} // container nouns present in THIS room
}

// ValidateSpawnEntry enforces the spawn-entry rules an author can get wrong.
//
// A spawn entry spawns exactly ONE of: a mob, an item, or gold. The YAML
// shape allows all three at once, which behaves unpredictably; reject it at
// save so a contradictory entry cannot reach a room file.
func ValidateSpawnEntry(s SpawnInfo, v spawnValidators) error {
	kinds := 0
	if s.MobId != 0 {
		kinds++
	}
	if s.ItemId != 0 {
		kinds++
	}
	if s.Gold != 0 {
		kinds++
	}
	if kinds == 0 {
		return fmt.Errorf("a spawn entry must spawn a mob, an item, or gold")
	}
	if kinds > 1 {
		return fmt.Errorf("a spawn entry must spawn exactly one of mob / item / gold")
	}

	if s.MobId != 0 && !v.mobExists(s.MobId) {
		return fmt.Errorf("mob %d does not exist", s.MobId)
	}
	if s.ItemId != 0 && !v.itemExists(s.ItemId) {
		return fmt.Errorf("item %d does not exist", s.ItemId)
	}

	if s.Container != "" {
		if s.MobId != 0 {
			return fmt.Errorf("a mob cannot spawn into a container")
		}
		if _, ok := v.containers[s.Container]; !ok {
			return fmt.Errorf("this room has no container named %q", s.Container)
		}
	}

	for _, b := range s.BuffIds {
		if !v.buffExists(b) {
			return fmt.Errorf("buff %d does not exist", b)
		}
	}

	// An unparseable period is worse than an error at runtime: AddPeriod
	// returns the current round, so the spawn returns immediately and nobody
	// notices until the world feels wrong.
	if s.RespawnRate != "" && !v.periodOK(s.RespawnRate) {
		return fmt.Errorf("respawn rate %q is not a period the engine understands (e.g. \"5 real minutes\")", s.RespawnRate)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run TestValidateSpawnEntry -v`
Expected: PASS (all four)

- [ ] **Step 5: Commit**

```bash
git add internal/rooms/spawninfo_validate.go internal/rooms/spawninfo_validate_test.go
git commit -m "feat(rooms): spawn entry validation policy"
```

---

### Task 2: Production wiring of the validators

**Files:** Modify `internal/rooms/spawninfo_validate.go`, `internal/rooms/spawninfo_validate_test.go`

- [ ] **Step 1: Write the failing test**

```go
// The period check must round-trip through the real parser rather than
// duplicating its 3-char-prefix unit vocabulary, which would drift.
func TestRealPeriodOK(t *testing.T) {
	assert.True(t, RealPeriodOK("5 real minutes"))
	assert.True(t, RealPeriodOK("600 rounds"))
	assert.True(t, RealPeriodOK("2 real hours"))
	assert.False(t, RealPeriodOK("banana"))
	assert.False(t, RealPeriodOK("soon"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run TestRealPeriodOK -v`
Expected: FAIL — `undefined: RealPeriodOK`

- [ ] **Step 3: Write minimal implementation**

Append to `internal/rooms/spawninfo_validate.go`:

```go
// RealPeriodOK reports whether the engine's period parser understands a
// string, by asking it. gametime.AddPeriod never errors — it returns the
// caller's own round number for anything it cannot read — so "does not
// advance" IS the failure signal. Deliberately not a reimplementation of the
// parser's unit vocabulary, which would drift from it.
func RealPeriodOK(period string) bool {
	if period == "" {
		return true // empty is legal; it means the default
	}
	gd := gametime.GetDate(1)
	return gd.AddPeriod(period) > gd.RoundNumber
}

// ValidateSpawnEntryLive is the production wiring, checked against the real
// registries and the given room's containers.
func ValidateSpawnEntryLive(s SpawnInfo, containers map[string]Container) error {
	set := map[string]struct{}{}
	for name := range containers {
		set[name] = struct{}{}
	}
	return ValidateSpawnEntry(s, spawnValidators{
		mobExists:  func(id int) bool { return mobs.GetMobSpec(mobs.MobId(id)) != nil },
		itemExists: func(id int) bool { return items.GetItemSpec(id) != nil },
		buffExists: func(id int) bool { return buffs.GetBuffSpec(id) != nil },
		periodOK:   RealPeriodOK,
		containers: set,
	})
}
```

Imports needed: `github.com/GoMudEngine/GoMud/internal/gametime`,
`.../internal/mobs`, `.../internal/items`, `.../internal/buffs`.
`internal/rooms` already imports `mobs` (roomdetails.go) and `items`. Verify
`gametime` and `buffs` do not create an import cycle before adding them — if
either does, move `ValidateSpawnEntryLive` into `modules/gmcp` (which imports
everything freely) and leave only the pure policy in `rooms`. Report which
you did.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run TestRealPeriodOK -v && go build ./...`
Expected: PASS, build clean

- [ ] **Step 5: Commit**

```bash
git add internal/rooms/spawninfo_validate.go internal/rooms/spawninfo_validate_test.go
git commit -m "feat(rooms): live spawn validators incl. round-trip period check"
```

---

### Task 3: Carry spawns out on `Build.Room.Get`

**Files:** Modify `modules/gmcp/gmcp.Build.go`, `modules/gmcp/gmcp.Build_test.go`

- [ ] **Step 1: Write the failing test**

Append to `modules/gmcp/gmcp.Build_test.go` (it already has a fakeWorld fake with .rooms map[int]*rooms.Room, .saved []rooms.Room, and a .deps() buildDeps method — reuse it, do not add a second fake):

```go
func TestBuildRoomGet_CarriesSpawnList(t *testing.T) {
	w := newFakeWorld()
	r := &rooms.Room{RoomId: 100, Title: "T", Description: "D"}
	r.SpawnInfo = []rooms.SpawnInfo{
		{MobId: 336, RespawnRate: "5 real minutes", Message: "A guard arrives."},
		{ItemId: 40001, Container: "chest"},
	}
	w.rooms[100] = r

	d, ok := buildRoomGet(w.deps(), 100)
	if !ok {
		t.Fatal("expected room detail")
	}
	if len(d.Spawns) != 2 {
		t.Fatalf("expected 2 spawns, got %d", len(d.Spawns))
	}
	if d.Spawns[0].MobId != 336 || d.Spawns[0].RespawnRate != "5 real minutes" {
		t.Errorf("mob spawn not mapped: %+v", d.Spawns[0])
	}
	if d.Spawns[1].ItemId != 40001 || d.Spawns[1].Container != "chest" {
		t.Errorf("item spawn not mapped: %+v", d.Spawns[1])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./modules/gmcp/ -run TestBuildRoomGet_CarriesSpawnList -v`
Expected: FAIL — `d.Spawns` undefined

- [ ] **Step 3: Write minimal implementation**

Add to `buildRoomDetail` (`gmcp.Build.go:140`):

```go
	Spawns []rooms.SpawnInfo `json:"spawns"`
```

and populate it in `buildRoomGet` alongside the other fields:

```go
	detail.Spawns = r.SpawnInfo
```

`rooms.SpawnInfo` already carries json-friendly exported fields and its two
runtime fields are `yaml:"-"`; they serialise harmlessly and are ignored on
the way back in (Task 4 preserves them from the template rather than trusting
the client).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./modules/gmcp/ -run TestBuildRoomGet_CarriesSpawnList -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add modules/gmcp/gmcp.Build.go modules/gmcp/gmcp.Build_test.go
git commit -m "feat(build): Build.Room.Get carries the room's spawn list"
```

---

### Task 4: Save spawns on `Build.Room.Update`

**Files:** Modify `modules/gmcp/gmcp.Build.go`, `modules/gmcp/gmcp.Build_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestBuildRoomUpdate_SavesSpawnsAndValidates(t *testing.T) {
	w := newFakeWorld()
	r := &rooms.Room{RoomId: 100, Title: "T", Description: "D"}
	r.Containers = map[string]rooms.Container{"chest": {}}
	w.rooms[100] = r

	base := func() roomUpdateReq {
		return roomUpdateReq{RoomId: 100, Title: "T", Description: "D"}
	}

	// Valid list saves and preserves order.
	req := base()
	req.Spawns = []rooms.SpawnInfo{{MobId: 1}, {ItemId: 2, Container: "chest"}}
	res := buildRoomUpdate(w.deps(), req)
	if !res.Ok {
		t.Fatalf("valid spawn list should save: %+v", res)
	}
	saved := w.saved[len(w.saved)-1]
	if len(saved.SpawnInfo) != 2 || saved.SpawnInfo[0].MobId != 1 || saved.SpawnInfo[1].ItemId != 2 {
		t.Errorf("spawn list not saved in order: %+v", saved.SpawnInfo)
	}

	// A contradictory entry is refused and nothing is saved.
	before := len(w.saved)
	bad := base()
	bad.Spawns = []rooms.SpawnInfo{{MobId: 1, ItemId: 2}}
	if res := buildRoomUpdate(w.deps(), bad); res.Ok {
		t.Error("an entry spawning both a mob and an item must be rejected")
	}
	if len(w.saved) != before {
		t.Error("nothing may be saved when a spawn entry is invalid")
	}

	// A container that is not in this room is refused.
	bad2 := base()
	bad2.Spawns = []rooms.SpawnInfo{{ItemId: 2, Container: "barrel"}}
	if res := buildRoomUpdate(w.deps(), bad2); res.Ok {
		t.Error("a container absent from the room must be rejected")
	}
}

// InstanceId/DespawnedRound are runtime tracking the client never sees
// meaningfully. A save must not let a client blank them for entries that are
// unchanged, or a live mob stops being tracked as spawned.
func TestBuildRoomUpdate_PreservesSpawnRuntimeTracking(t *testing.T) {
	w := newFakeWorld()
	r := &rooms.Room{RoomId: 100, Title: "T", Description: "D"}
	r.SpawnInfo = []rooms.SpawnInfo{{MobId: 1, InstanceId: 4242, DespawnedRound: 99}}
	w.rooms[100] = r

	req := roomUpdateReq{RoomId: 100, Title: "T", Description: "D",
		Spawns: []rooms.SpawnInfo{{MobId: 1}}} // client sends no tracking

	if res := buildRoomUpdate(w.deps(), req); !res.Ok {
		t.Fatalf("update should succeed: %+v", res)
	}
	saved := w.saved[len(w.saved)-1]
	if saved.SpawnInfo[0].InstanceId != 4242 || saved.SpawnInfo[0].DespawnedRound != 99 {
		t.Errorf("runtime tracking must be preserved from the template, got %+v", saved.SpawnInfo[0])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./modules/gmcp/ -run TestBuildRoomUpdate_ -v`
Expected: FAIL — `req.Spawns` undefined

- [ ] **Step 3: Write minimal implementation**

Add to `roomUpdateReq` (`gmcp.Build.go:62`):

```go
	Spawns []rooms.SpawnInfo `json:"spawns"`
```

In `buildRoomUpdate`, after the room loads and before `d.save(*r)`:

```go
	// Validate every entry before touching the room, so a bad entry leaves the
	// existing list intact rather than half-applied.
	for i, s := range req.Spawns {
		if err := rooms.ValidateSpawnEntryLive(s, r.Containers); err != nil {
			return buildErr("spawn %d: %s", i+1, err.Error())
		}
	}
	// InstanceId/DespawnedRound are runtime tracking the client does not own.
	// Carry them across from the existing entry with the same target so a live
	// mob keeps being tracked as spawned.
	req.Spawns = preserveSpawnTracking(r.SpawnInfo, req.Spawns)
	r.SpawnInfo = req.Spawns
```

And the helper:

```go
// preserveSpawnTracking copies runtime tracking (InstanceId, DespawnedRound)
// from the existing spawn list onto the incoming one, matching on the spawn
// target. Without it, saving a room from the editor would blank the tracking
// for every currently-spawned mob, and the engine would spawn duplicates.
func preserveSpawnTracking(old, incoming []rooms.SpawnInfo) []rooms.SpawnInfo {
	type key struct{ mob, item, gold int }
	prev := map[key]rooms.SpawnInfo{}
	for _, s := range old {
		prev[key{s.MobId, s.ItemId, s.Gold}] = s
	}
	for i, s := range incoming {
		if p, ok := prev[key{s.MobId, s.ItemId, s.Gold}]; ok {
			incoming[i].InstanceId = p.InstanceId
			incoming[i].DespawnedRound = p.DespawnedRound
		}
	}
	return incoming
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./modules/gmcp/ -run TestBuildRoomUpdate_ -v`
Expected: PASS (both)

- [ ] **Step 5: Run the whole package**

Run: `go test ./modules/gmcp/ && go build ./...`
Expected: PASS, build clean

- [ ] **Step 6: Commit**

```bash
git add modules/gmcp/gmcp.Build.go modules/gmcp/gmcp.Build_test.go
git commit -m "feat(build): save spawn lists via Build.Room.Update"
```

---

### Task 5: Strip the 36 inert `cooldown:` lines

**Files:** Create `tools/strip_spawn_cooldown.py`; modify 36 room YAMLs and `boot_smoke_test.go`

- [ ] **Step 1: Write the migration script**

Create `tools/strip_spawn_cooldown.py`:

```python
"""Strip the inert `cooldown:` key from room spawninfo blocks.

`cooldown` is not a field on rooms.SpawnInfo (the real field is
`respawnrate`), so these 36 authored lines have never done anything. They are
DELETED rather than converted: the values are in rounds while every live
respawnrate is in real minutes, and converting verbatim would make 36
starter-area spawn points 1.3-11x slower — a gameplay change nobody asked for.
The authored values are recorded in the spec for a future pacing pass.

Line-level edit, not a YAML round-trip: re-marshalling reflows every
description block in the file.
"""
import glob, io, re, sys

changed, removed = 0, 0
for path in glob.glob('_datafiles/world/dogmud/rooms/*/[0-9]*.yaml'):
    src = io.open(path, encoding='utf-8', newline='').read()
    eol = '\r\n' if '\r\n' in src else '\n'
    lines = src.split(eol)
    keep = [ln for ln in lines if not re.match(r'^  cooldown:', ln)]
    if len(keep) != len(lines):
        removed += len(lines) - len(keep)
        changed += 1
        io.open(path, 'w', encoding='utf-8', newline='').write(eol.join(keep))
print('files changed: %d   cooldown lines removed: %d' % (changed, removed))
```

- [ ] **Step 2: Confirm the target count BEFORE running**

Run: `grep -rc "^  cooldown:" _datafiles/world/dogmud/rooms/*/*.yaml | awk -F: '{s+=$2} END {print s}'`
Expected: `36`

If it is not 36, STOP and report — the world has changed since the spec.

- [ ] **Step 3: Run the migration**

Run: `python tools/strip_spawn_cooldown.py`
Expected: `files changed: 36   cooldown lines removed: 36`
(36 files each carrying one line; if a file carried two the counts differ and that is fine.)

- [ ] **Step 4: Verify ONLY those lines changed**

Run: `git diff --stat` — expect ~36 files, 36 deletions, **0 insertions**.
Run: `git diff -U0 | grep "^+" | grep -v "^+++"` — expect **no output**. Any
added line means the script reflowed something; revert and fix.

- [ ] **Step 5: Delete the drift-gate baseline entry**

In `boot_smoke_test.go`, delete the line:

```go
	"cooldown|rooms.SpawnInfo":               true,
```

and remove `cooldown` from the explanatory comment block above
`knownSilentlyIgnoredKeys` (it is listed alongside `zone|exit.RoomExit`; leave
that one).

- [ ] **Step 6: Prove the gate now sees fewer ignored keys**

Run: `DOGMUD_BOOT_SMOKE=1 go test . -run TestSmoke_NoNewSilentlyIgnoredYAMLKeys -v`
Expected: PASS, with the distinct-key count **lower than 27** and `new: 0`.

- [ ] **Step 7: Commit**

```bash
git add tools/strip_spawn_cooldown.py boot_smoke_test.go _datafiles/world/dogmud/rooms
git commit -m "fix(world): strip 36 inert spawninfo cooldown keys"
```

---

### Task 6: Spawns section in the Rooms inspector

**Files:** Modify `_datafiles/html/public/static/js/builder.js`

Read `builder.js`'s inspector rendering and `mobs.js`'s Advanced-drawer
pattern before writing. Follow both.

- [ ] **Step 1: Render the list**

A **Spawns** section in the room inspector showing each entry as a collapsed
row summarising it ("mob 336 · every 5 real minutes"), expandable to a form.
An **+ Add spawn** control appends a new entry.

- [ ] **Step 2: Kind selector**

Each entry starts with a kind choice — mob / item / gold — showing only that
kind's target control. The server rejects an entry with more than one set
(Task 1), so the UI must make the contradiction unreachable rather than
merely discouraged.

- [ ] **Step 3: Common fields**

Target picker (mob or item, from `Build.Mob.List` / `Build.Item.List` — never
a free-text id), gold amount, container (a select of the room's own container
nouns, empty = floor), respawn rate (text, hint: *"blank = 15 real minutes"*),
message.

- [ ] **Step 4: Advanced drawer**

Collapsed by default, holding `name`, `forcehostile`, `maxwander`,
`idlecommands` (list), `scripttag`, `questflags` (list), `buffids` (list),
`statpool`, `statpoolmod`.

- [ ] **Step 5: Remove and reorder**

Per-entry remove; up/down reorder preserving list order on save. No
confirmation and no blocker scan — a spawn entry has no downstream
references, unlike item/mob/zone deletion.

- [ ] **Step 6: Save integration**

Spawns ride the existing room Save button in the existing
`Build.Room.Update` payload. No separate save.

- [ ] **Step 7: Syntax check**

Run: `node --check _datafiles/html/public/static/js/builder.js`
Expected: no output

- [ ] **Step 8: Commit**

```bash
git add _datafiles/html/public/static/js/builder.js
git commit -m "feat(build): spawn-list editing in the Rooms inspector"
```

---

### Task 7: Correct the stale CLAUDE.md claim

**Files:** Modify `CLAUDE.md`

- [ ] **Step 1: Fix the instance-save SOP**

In the "Instance Saves & Smoke-Test SOP" section, the list of things stale
instance saves shadow includes "room spawn lists". That is false:
`Room.SpawnInfo` is tagged `instance:"skip"` (`rooms.go:103`) and
`restoreSkipTaggedFields` (`save_and_load.go:137`) copies it back from the
template after the overlay unmarshal.

Remove "room spawn lists" from that list and add a short note that fields
tagged `instance:"skip"` — spawn lists among them — are restored from the
template and are NOT shadowed. Leave the rest of the SOP intact; it is
accurate for the fields that genuinely are shadowed.

- [ ] **Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: correct stale instance-save claim about spawn lists"
```

---

### Task 8: Full verification gate

- [ ] **Step 1: Full suite**

Run: `go test ./...`
Expected: no failures.

- [ ] **Step 2: Format and vet**

Run: `gofmt -l internal modules && go vet ./...`
Expected: no output from either.

- [ ] **Step 3: Boot under the prod setting**

Confirm `_datafiles/config.yaml` has `MapConsistencyEnforce: panic` (that file
carries git **skip-worktree**, so local edits never show in `git status`).

Run: `go run .` — expect
`mapper.ValidateZoneConsistency errors=0 warnings=0 mode="panic"`, no
`ERROR: PANIC`, and `rooms.LoadDataFiles()` loading the same room count as
before the migration. Ctrl-C once the HTTP server starts.

- [ ] **Step 4: Drift gate**

Run: `DOGMUD_BOOT_SMOKE=1 go test . -run TestSmoke_NoNewSilentlyIgnoredYAMLKeys -v`
Expected: PASS, `new: 0`, distinct count **below 27**.

- [ ] **Step 5: PATCH_NOTES.md**

Add a dated staff-facing entry in the house voice (prose, no identifiers, no
raw numbers) covering spawn-list authoring and the respawn-timing cleanup.

- [ ] **Step 6: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs(patch-notes): spawn-list editor"
```

- [ ] **Step 7: Browser gate — hand to the user**

Ask the user to drive `/build` → Rooms → a room with spawns: add a mob spawn,
set a respawn rate, open Advanced, remove an entry, save, and confirm the mob
actually appears in game. Spawn lists are not shadowed by instance saves, so
no wipe is needed for the change to take effect.

---

## Out of scope

Ground items / `Room.Stash` (loot SOP), world-wide spawn search, respawn-rate
tuning (the authored round values are recorded in the spec), and sub-project 5
(dialogue / behavior trees / quests), which remains deferred.
