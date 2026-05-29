# Planar Oasis: All Bosses + Elemental Princess + Larger Grid — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every Planar Oasis boss spawn each run (king/queen/prince + a new water Elemental Princess), grow the cube grid 4×4×4 → 5×5×5, and give the princess two affix-scaled drops (unarmed claws + a neck torc).

**Architecture:** Three new data files (princess mob, claws weapon, neck armor) plus a small edit to the instance cube generator (`internal/rooms/cubegen.go`). "Scaling" comes for free from the existing instance affix engine (`internal/items/affixgen.go`) — the new items are plain base templates listed in the princess's `loot_pool`. The claws route to unarmed-combat purely via `subtype: claws` (no engine change).

**Tech Stack:** Go, YAML data files, testify.

**Spec:** `docs/superpowers/specs/2026-05-29-oasis-all-bosses-princess-design.md`

**IDs (verified free):** princess mob **377**, claws **10036**, neck torc **20077**.

---

## File Structure

| File | Responsibility |
|------|----------------|
| `_datafiles/world/dogmud/items/weapons-10000/10036-drowned_claws.yaml` | New unarmed claws base item (affix-scaled in instances) |
| `_datafiles/world/dogmud/items/armor-20000/neck/20077-tidal_torc.yaml` | New neck base item (affix-scaled in instances) |
| `_datafiles/world/dogmud/mobs/instance_planar_oasis/377-elemental_princess.yaml` | New 4th boss; drops the two items above |
| `internal/rooms/cubegen.go` | Grid size 4→5, 5th layer title, all bosses spawn each in own room |
| `internal/rooms/cubegen_test.go` | Update size-dependent tests; add all-bosses integration test |

Data files first (the mob's `loot_pool` references the items; the generator references the mob), then the generator + tests.

---

### Task 1: Claws weapon base item (10036)

**Files:**
- Create: `_datafiles/world/dogmud/items/weapons-10000/10036-drowned_claws.yaml`

- [ ] **Step 1: Create the item file**

```yaml
itemid: 10036
name: drowned claws
namesimple: claws
vendor_categories:
- blacksmithing
type: weapon
subtype: claws
hands: 1
damage_multiplier: 0.45
speedmultiplier: 1.20
damage:
  basedamage: 4
  variance: 2
parryrating: 2
staminacost: 3
grapplemodifier: 0.5
weight: 0.8
value: 70
description: >-
  Four curved talons of solidified oasis water, clear as glass and
  cold to the touch. They flex like living tendrils between strikes,
  then snap rigid at the moment of impact. Water beads and runs along
  the edges without ever dripping free.
```

Note: `subtype: claws` is what makes `characters.CombatSkillTagForItem` resolve attacks to **unarmed-combat** for any wielder (mob or player). Do not change it.

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/items/weapons-10000/10036-drowned_claws.yaml
git commit -m "feat(oasis): add drowned claws (unarmed boss-drop weapon, 10036)"
```

---

### Task 2: Neck torc base item (20077)

**Files:**
- Create: `_datafiles/world/dogmud/items/armor-20000/neck/20077-tidal_torc.yaml`

- [ ] **Step 1: Create the item file**

```yaml
itemid: 20077
name: tidal torc
namesimple: torc
vendor_categories:
- jewelcrafting
type: neck
subtype: wearable
magical_mitigation: 6
physical_mitigation: 3
weight: 1.0
value: 85
description: >-
  A circlet of ever-flowing oasis water held in a perfect ring by
  some quiet planar will. It turns and ripples against the wearer's
  throat without spilling, and the air around it carries the cool,
  mineral smell of deep water.
```

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/items/armor-20000/neck/20077-tidal_torc.yaml
git commit -m "feat(oasis): add tidal torc (neck boss-drop, 20077)"
```

---

### Task 3: Elemental Princess boss mob (377)

**Files:**
- Create: `_datafiles/world/dogmud/mobs/instance_planar_oasis/377-elemental_princess.yaml`

- [ ] **Step 1: Create the mob file**

```yaml
mobid: 377
zone: Instance Planar Oasis
archetype: fighting
behavior_archetype: leader
routine: planar_oasis_pack
statpool: 4
hostile: true
groups:
  - elemental
activitylevel: 30
combatcommands:
  - 'emote slips sideways like a current and rakes with liquid talons'
  - ''
  - 'emote coils a whip of water around its prey before striking'
  - 'trip'
  - 'emote dissolves to spray and reforms at your flank, claws first'
  - ''
character:
  name: elemental princess
  description: >-
    A lithe figure of clear running water held in the shape of a
    woman, her edges never quite still. She circles rather than
    charges, watching with the patient attention of something that
    has drowned patient things before. Where her hands should be,
    four curved talons of hardened water catch the twilight. The
    air around her smells of cold deep water.
  speciesid: 36
  level: 1
  gold: 0
  stats:
    dexterity:
      training: 12
    perception:
      training: 8
  skills:
    unarmed-combat: 5
itemdropchance: 75
loot_pool:
  - 10036
  - 20077
```

Note: species 36 (water elemental) already has all equipment slots enabled (2026-05-29 companion-gear fix), so she equips the claws + torc. Filename `377-elemental_princess.yaml` matches `ConvertForFilename("elemental princess")`.

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/mobs/instance_planar_oasis/377-elemental_princess.yaml
git commit -m "feat(oasis): add elemental princess boss (water, claws, 377)"
```

---

### Task 4: Cube generator — grid 5×5×5 + all bosses spawn

**Files:**
- Modify: `internal/rooms/cubegen.go`
- Test: `internal/rooms/cubegen_test.go`

- [ ] **Step 1: Add the failing all-bosses integration test**

Append to `internal/rooms/cubegen_test.go`. Add `"github.com/GoMudEngine/GoMud/internal/exit"` and `"github.com/stretchr/testify/require"` to the imports.

```go
func TestGenerateOasisCube_AllBossesDistinctRooms(t *testing.T) {
	// Seed a roomManager containing only the threshold room (5003).
	threshold := &Room{
		RoomId:    5003,
		Zone:      "Instance Planar Oasis",
		Exits:     make(map[string]exit.RoomExit),
		ExitsTemp: make(map[string]exit.TemporaryRoomExit),
	}
	cleanup := SeedRoomsForTest(
		map[int]*Room{5003: threshold},
		map[string]*ZoneConfig{},
	)
	defer cleanup()
	// Reset ephemeral state so the test is isolated and re-runnable.
	defer func() {
		for i := range ephemeralRoomChunks {
			ephemeralRoomChunks[i] = nil
		}
		originalRoomIdLookups = map[int]int{}
	}()

	roomIds, _, err := GenerateOasisCube(5003, "Instance Planar Oasis", 500, 1, true, "rejoin")
	require.NoError(t, err)
	assert.Len(t, roomIds, 125, "5x5x5 cube should have 125 rooms")

	bossIds := []int{320, 321, 322, 377}
	count := map[int]int{}
	roomOf := map[int]int{}
	for _, rid := range roomIds {
		r := LoadRoom(rid)
		require.NotNil(t, r)
		for _, si := range r.SpawnInfo {
			for _, bid := range bossIds {
				if si.MobId == bid {
					count[bid]++
					roomOf[bid] = rid
				}
			}
		}
	}

	for _, bid := range bossIds {
		assert.Equal(t, 1, count[bid], "boss %d should spawn exactly once", bid)
	}

	seen := map[int]bool{}
	for _, bid := range bossIds {
		assert.False(t, seen[roomOf[bid]], "bosses must be in distinct rooms")
		seen[roomOf[bid]] = true
	}
}
```

- [ ] **Step 2: Run it to confirm it fails**

Run: `go test ./internal/rooms/ -run TestGenerateOasisCube_AllBossesDistinctRooms -v`
Expected: FAIL — only 1 boss present (current code places one random boss) and the cube has 64 rooms, not 125.

- [ ] **Step 3: Change the grid size and titles**

In `internal/rooms/cubegen.go`, change the size constant:

```go
const (
	cubeSize  = 5
	cubeTotal = cubeSize * cubeSize * cubeSize // 125
)
```

Add a 5th layer title for z=4:

```go
var cubeTitles = []string{"Elemental Depths", "Lower Wastes", "Upper Wastes", "Elemental Heights", "Elemental Apex"}
```

- [ ] **Step 4: Add the princess to the boss list**

```go
// Boss mob IDs.
var cubeBossMobs = []int{320, 321, 322, 377}
```

- [ ] **Step 5: Place every boss, each in its own room**

Replace the single-boss placement block (currently using `pickUniqueIndices(1, ...)` and `cubeBossMobs[util.Rand(...)]`) with:

```go
	// Place every boss, each in its own room (distinct from tough rooms).
	bossIndices := pickUniqueIndices(len(cubeBossMobs), cubeTotal, toughIndices)
	for i, idx := range bossIndices {
		rooms[idx].SpawnInfo = []SpawnInfo{
			{
				MobId:        cubeBossMobs[i],
				StatPool:     4,
				RespawnRate:  "2 real hours",
				ForceHostile: true,
				MaxWander:    5,
				IdleCommands: wanderIdle,
			},
		}
	}
```

- [ ] **Step 6: Update the size-dependent existing tests**

In `internal/rooms/cubegen_test.go`, update the assertions that hardcoded the 4×4×4 layout:

`TestCubeIndex`:
```go
func TestCubeIndex(t *testing.T) {
	assert.Equal(t, 0, cubeIndex(0, 0, 0))
	assert.Equal(t, 124, cubeIndex(4, 4, 4)) // max for 5x5x5
	// x*25 + y*5 + z
	assert.Equal(t, 25, cubeIndex(1, 0, 0))
	assert.Equal(t, 5, cubeIndex(0, 1, 0))
	assert.Equal(t, 1, cubeIndex(0, 0, 1))
}
```

`TestCubeTitle` (z=4 now has a title; z=5 is out of range):
```go
func TestCubeTitle(t *testing.T) {
	assert.Equal(t, "Elemental Depths", cubeTitle(0))
	assert.Equal(t, "Lower Wastes", cubeTitle(1))
	assert.Equal(t, "Upper Wastes", cubeTitle(2))
	assert.Equal(t, "Elemental Heights", cubeTitle(3))
	assert.Equal(t, "Elemental Apex", cubeTitle(4))
	assert.Equal(t, "Planar Wastes", cubeTitle(5))  // out of range
	assert.Equal(t, "Planar Wastes", cubeTitle(-1)) // out of range
}
```

`TestCubeWrappingExits` (corner is now (4,4,4); south of (0,0,0) wraps to (0,4,0)):
```go
func TestCubeWrappingExits(t *testing.T) {
	// Room at (0,0,0): north -> (0,1,0), south wraps -> (0,4,0)
	northIdx := cubeIndex(0, wrapCoord(0+1, cubeSize), 0)
	assert.Equal(t, cubeIndex(0, 1, 0), northIdx)

	southIdx := cubeIndex(0, wrapCoord(0-1, cubeSize), 0)
	assert.Equal(t, cubeIndex(0, 4, 0), southIdx) // wraps

	// Room at (4,4,4): east wraps -> (0,4,4), up wraps -> (4,4,0)
	eastIdx := cubeIndex(wrapCoord(4+1, cubeSize), 4, 4)
	assert.Equal(t, cubeIndex(0, 4, 4), eastIdx)

	upIdx := cubeIndex(4, 4, wrapCoord(4+1, cubeSize))
	assert.Equal(t, cubeIndex(4, 4, 0), upIdx)
}
```

`TestCubeEntryRoomIndex` (comment only — assertion already derives from cubeSize):
```go
func TestCubeEntryRoomIndex(t *testing.T) {
	// Entry at (2,2,0)
	entryIdx := cubeIndex(2, 2, 0)
	assert.Equal(t, 2*cubeSize*cubeSize+2*cubeSize+0, entryIdx) // 2*25+2*5+0 = 60
}
```

(`TestWrapCoord` passes the size explicitly as `4` and is size-agnostic — leave it unchanged.)

- [ ] **Step 7: Run the full rooms test suite**

Run: `go test ./internal/rooms/`
Expected: PASS (all updated tests + the new all-bosses test).

- [ ] **Step 8: Commit**

```bash
git add internal/rooms/cubegen.go internal/rooms/cubegen_test.go
git commit -m "feat(oasis): spawn all bosses + grow cube to 5x5x5 (125 rooms)"
```

---

### Task 5: Verification

**Files:** none (verification only)

- [ ] **Step 1: ID collision check**

Run: `python tools/id_inventory.py`
Expected: no duplicate-ID warnings; 377 / 10036 / 20077 present, no collisions.

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: no output (success).

- [ ] **Step 3: Boot smoke (per SOP — wipe instance saves first)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/oasis_boot.log 2>&1 &
```
Poll the log until "Server Ready" or a panic, then stop the server.
Expected: `species.LoadDataFiles()`, `itemspec.LoadDataFiles()`, `mobs.LoadDataFiles()` all load with no panic and no `did not end in Filepath()` error (validates the new princess mob + claws + torc YAMLs and the `subtype: claws` weapon). Confirm `Server Ready`.

- [ ] **Step 4: Stop the server**

```bash
taskkill //IM "GoMud.exe" //F
```

---

## Notes for the implementer

- **No new scaling mechanic.** The items are plain base templates; the instance affix engine scales them on spawn from gold paid. Do not add stat affixes to the base YAMLs.
- **Difficulty:** four bosses across 125 rooms is intended (clear-once zone). No retuning of existing mobs is in scope.
- **Claws skill routing** is entirely driven by `subtype: claws`; verified against `characters.CombatSkillTagForItem`. No Go change for that.
