# Town Justice 5.1 — Stillwater Rollout + Quest-Rep Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the shipped Thornwall town-justice framework to the town of Stillwater, move the holding-cell registry from hardcoded Go to faction data, and wire faction-reputation rewards onto seven civic quests across both towns.

**Architecture:** Mirror the Thornwall justice pattern for Stillwater. One infrastructure refactor (cell room id moves from a hardcoded `jailCellFor` map onto the faction definition YAML, read through an injectable seam) plus data: two new factions, a combat-capable constable, a new cell room, citizen faction tags, and quest `bump_rep` actions. No new justice *mechanics*.

**Tech Stack:** Go (packages `internal/justice`, `internal/factions`), YAML data files under `_datafiles/world/dogmud/`, Go `testing`.

**Spec:** `docs/superpowers/specs/2026-05-30-town-justice-5.1-stillwater-rollout-design.md`

**Conventions for every task:**
- Build check: `go build ./...`
- Boot smoke (per CLAUDE.md SOP): wipe instance saves first —
  `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`
  then run the server and confirm it loads past data files without panic
  (watch for `mobs.LoadDataFiles() loadedCount=...`,
  `factions.LoadAllDefinitions loadedCount=...`, `quests.LoadDataFiles() loadedCount=...`).
- Every task ends booting cleanly. Commit after each task.

---

## Task 1: HoldingCellRoom field + faction-data cell lookup

Replaces the hardcoded `jailCellFor` map with a `holding_cell_room:` field on the
faction definition, read through an injectable seam (matching the existing
`alliesFn`/`repTierFn`/`aDecayFn` pattern in this package).

**Files:**
- Modify: `internal/factions/types.go:9-16` (add field)
- Modify: `internal/justice/justice.go:27-38` (add `cellRoomFn` read-seam)
- Modify: `internal/justice/arrest.go:37-40` (delete `jailCellFor`), `arrest.go:243-248` (read from seam)
- Modify: `_datafiles/world/dogmud/factions/thornwall_guards.yaml` (migrate the one entry)
- Test: `internal/justice/arrest_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/justice/arrest_test.go`:

```go
func TestExecuteArrest_NoCellForFaction_NoOps(t *testing.T) {
	origCell := cellRoomFn
	origMove := aMoveFn
	t.Cleanup(func() { cellRoomFn = origCell; aMoveFn = origMove })

	moved := false
	aMoveFn = func(int, int, ...bool) error { moved = true; return nil }
	cellRoomFn = func(faction string) int { return 0 } // faction has no jail

	ch := characters.New()
	ok := ExecuteArrest(ch, 1, "stillwater_citizens", false)

	if ok {
		t.Fatalf("ExecuteArrest should return false when faction has no cell")
	}
	if moved {
		t.Fatalf("player must not be moved when faction has no cell")
	}
}

func TestExecuteArrest_UsesFactionCellRoom(t *testing.T) {
	origCell := cellRoomFn
	origMove := aMoveFn
	t.Cleanup(func() { cellRoomFn = origCell; aMoveFn = origMove })

	var movedTo int
	aMoveFn = func(userId int, toRoomId int, isSpawn ...bool) error { movedTo = toRoomId; return nil }
	cellRoomFn = func(faction string) int { return 5106 }

	ch := characters.New()
	ok := ExecuteArrest(ch, 1, "stillwater_guards", false)

	if !ok {
		t.Fatalf("ExecuteArrest should succeed when faction has a cell")
	}
	if movedTo != 5106 {
		t.Fatalf("player hauled to %d, want 5106", movedTo)
	}
}
```

(If `characters.New()` is not the constructor used elsewhere in this test file,
match the existing helper — grep the file for how other tests build a
`*characters.Character`.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/justice/ -run TestExecuteArrest_ -v`
Expected: FAIL — `cellRoomFn` undefined (compile error).

- [ ] **Step 3: Add the field to the faction Definition**

In `internal/factions/types.go`, add the field to the struct:

```go
type Definition struct {
	FactionId   string   `yaml:"faction_id"`
	DisplayName string   `yaml:"display_name"`
	Description string   `yaml:"description"`
	DefaultRep  int      `yaml:"default_rep"`
	Allies      []string `yaml:"allies"`
	Enemies     []string `yaml:"enemies"`
	// HoldingCellRoom is the room id a guard of this faction hauls arrestees
	// to. 0 / omitted means this faction has no jail (correct for citizen and
	// non-guard factions). Read by internal/justice.
	HoldingCellRoom int `yaml:"holding_cell_room"`
}
```

- [ ] **Step 4: Add the `cellRoomFn` read-seam**

In `internal/justice/justice.go`, inside the `var (` read-seams block
(after `alliesFn`, ~line 38), add:

```go
	// cellRoomFn returns the holding-cell room id for a faction (0 = none).
	cellRoomFn = func(faction string) int {
		d := factions.GetDefinition(faction)
		if d == nil {
			return 0
		}
		return d.HoldingCellRoom
	}
```

- [ ] **Step 5: Delete the hardcoded map and read from the seam**

In `internal/justice/arrest.go`, delete the map (lines 37-40):

```go
// DELETE:
var jailCellFor = map[string]int{
	"thornwall_guards": 5105,
}
```

Replace the lookup at the top of `ExecuteArrest` (lines 244-247):

```go
	cell := cellRoomFn(faction)
	if cell == 0 {
		return false
	}
```

- [ ] **Step 6: Migrate the Thornwall entry into faction YAML**

Append to `_datafiles/world/dogmud/factions/thornwall_guards.yaml`:

```yaml
holding_cell_room: 5105
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `go test ./internal/justice/ ./internal/factions/ -v`
Expected: PASS (all existing justice tests still pass; the two new ones pass).

- [ ] **Step 8: Build + boot smoke**

Run: `go build ./...` then the boot smoke (wipe instances, start server).
Expected: clean boot; `factions.LoadAllDefinitions` logs the existing factions;
no panic. Thornwall arrests still target 5105.

- [ ] **Step 9: Commit**

```bash
git add internal/factions/types.go internal/justice/justice.go internal/justice/arrest.go internal/justice/arrest_test.go _datafiles/world/dogmud/factions/thornwall_guards.yaml
git commit -m "feat(justice): holding-cell room moves from hardcoded map to faction data"
```

---

## Task 2: Pick the arresting faction by cell ownership

Today `enforce.go` uses `guardFactions[0]` blindly. With multi-faction guards
(Drunn will carry `stillwater_guards` + `stillwater_citizens`), the arrest must
use the first faction that actually owns a cell.

**Files:**
- Modify: `internal/justice/enforce.go:177-182`
- Test: `internal/justice/enforce_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

Add to `internal/justice/enforce_test.go`:

```go
func TestFirstFactionWithCell(t *testing.T) {
	origCell := cellRoomFn
	t.Cleanup(func() { cellRoomFn = origCell })

	cellRoomFn = func(faction string) int {
		if faction == "stillwater_guards" {
			return 5106
		}
		return 0 // citizens / others: no cell
	}

	got := firstFactionWithCell([]string{"stillwater_citizens", "stillwater_guards"})
	if got != "stillwater_guards" {
		t.Fatalf("firstFactionWithCell = %q, want stillwater_guards", got)
	}

	if firstFactionWithCell([]string{"stillwater_citizens"}) != "" {
		t.Fatalf("firstFactionWithCell should return \"\" when no faction owns a cell")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/justice/ -run TestFirstFactionWithCell -v`
Expected: FAIL — `firstFactionWithCell` undefined.

- [ ] **Step 3: Add the helper**

In `internal/justice/enforce.go` (near the other unexported helpers), add:

```go
// firstFactionWithCell returns the first faction in order that owns a
// holding cell, or "" if none do. Used to pick the arresting faction when a
// guard belongs to several factions (e.g. guards + citizens).
func firstFactionWithCell(guardFactions []string) string {
	for _, f := range guardFactions {
		if cellRoomFn(f) != 0 {
			return f
		}
	}
	return ""
}
```

- [ ] **Step 4: Use it in the arrest-haul branch**

In `internal/justice/enforce.go`, replace the `arrestOutcomeHaul` branch body
(lines 177-182):

```go
			case arrestOutcomeHaul:
				faction := firstFactionWithCell(guardFactions)
				if faction == "" {
					// No arresting faction owns a cell — cannot haul; leave
					// the pending stamp so the declaration line isn't lost.
					break
				}
				executeArrestFn(user.Character, uid, faction, false)
				mob.Character.SetMiscData(pendingKey, nil)
				acts = append(acts, EnforceAction{uid, SeverityArrest, true})
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/justice/ -v`
Expected: PASS.

- [ ] **Step 6: Build + commit**

```bash
go build ./...
git add internal/justice/enforce.go internal/justice/enforce_test.go
git commit -m "feat(justice): arrest picks first guard-faction that owns a cell"
```

---

## Task 3: Boot-time holding-cell validation

Mirror the `SetScheduleWorldValidator` / `SetPatrolWorldValidator` DI pattern in
`main.go`: after rooms load, validate every faction's `HoldingCellRoom` resolves
to a real room. Panic on a dangling reference (data-file SOP).

**Files:**
- Modify: `internal/factions/registry.go` (add `ValidateHoldingCells`)
- Modify: `main.go:1201` area (call after `rooms.LoadDataFiles()`)
- Test: `internal/factions/registry_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/factions/registry_test.go`:

```go
func TestValidateHoldingCells(t *testing.T) {
	clearRegistryForTest()
	definitionsMu.Lock()
	definitions = map[string]*Definition{
		"good_guards": {FactionId: "good_guards", HoldingCellRoom: 100},
		"no_cell":     {FactionId: "no_cell", HoldingCellRoom: 0},
	}
	definitionsMu.Unlock()

	// Room 100 exists -> no panic.
	ValidateHoldingCells(func(roomId int) bool { return roomId == 100 })

	// Room 100 does NOT exist -> panic.
	defer func() {
		if recover() == nil {
			t.Fatalf("ValidateHoldingCells must panic on a dangling cell room")
		}
	}()
	ValidateHoldingCells(func(roomId int) bool { return false })
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/factions/ -run TestValidateHoldingCells -v`
Expected: FAIL — `ValidateHoldingCells` undefined.

- [ ] **Step 3: Implement `ValidateHoldingCells`**

In `internal/factions/registry.go` (after `AllDefinitions`), add:

```go
// ValidateHoldingCells panics if any loaded faction declares a HoldingCellRoom
// that roomExists reports as missing. Cells of 0 (no jail) are skipped. Called
// from main.go after rooms load (DI breaks the factions<-rooms import edge).
func ValidateHoldingCells(roomExists func(roomId int) bool) {
	definitionsMu.RLock()
	defer definitionsMu.RUnlock()
	for _, def := range definitions {
		if def.HoldingCellRoom == 0 {
			continue
		}
		if !roomExists(def.HoldingCellRoom) {
			panic(fmt.Sprintf("factions: faction %q holding_cell_room %d does not exist",
				def.FactionId, def.HoldingCellRoom))
		}
	}
}
```

(`definitionsMu` and `definitions` already exist in the package;
`clearRegistryForTest` already exists at `registry.go:122`. Confirm a
`sync.RWMutex` — if it is a plain `Mutex`, use `Lock`/`Unlock` here.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/factions/ -run TestValidateHoldingCells -v`
Expected: PASS.

- [ ] **Step 5: Wire it into boot**

In `main.go`, immediately after `rooms.LoadDataFiles()` (line 1201) and before
the schedule validator block, add:

```go
	factions.ValidateHoldingCells(func(roomId int) bool {
		return rooms.LoadRoom(roomId) != nil
	})
```

Confirm `factions` is imported in `main.go` (it is used elsewhere in the boot
sequence — add the import if the build complains).

- [ ] **Step 6: Build + boot smoke**

Run: `go build ./...` then boot smoke.
Expected: clean boot — `thornwall_guards`'s 5105 resolves, no panic.

- [ ] **Step 7: Commit**

```bash
git add internal/factions/registry.go internal/factions/registry_test.go main.go
git commit -m "feat(factions): boot-time validation of holding_cell_room references"
```

---

## Task 4: Stillwater holding cell room 5106

New cell room `down` from the Constabulary (4110), mirroring Thornwall 5105.
Created before the Stillwater factions (Task 5) so boot validation passes.

**Files:**
- Create: `_datafiles/world/dogmud/rooms/stillwater/5106.yaml`
- Modify: `_datafiles/world/dogmud/rooms/stillwater/4110.yaml` (add `down` exit)

- [ ] **Step 1: Reconfirm the room id is free**

Run: `python tools/id_inventory.py --type rooms`
Expected: global next-free `>= 5106`. If 5106 is taken, use the reported next
free id and substitute it everywhere below AND in Task 5's `holding_cell_room`.

- [ ] **Step 2: Create the cell room**

Create `_datafiles/world/dogmud/rooms/stillwater/5106.yaml`:

```yaml
roomid: 5106
zone: Stillwater
title: Holding Cell
description: >
  A single stone cell behind the constabulary's iron bars, barely wide
  enough to pace. The walls are mortared lake-stone gone green at the
  joints, and the only light comes from a narrow
  <ansi fg="itemname">window</ansi> slit set high near the ceiling, too
  small for anything but a wrist. A low wooden
  <ansi fg="itemname">bench</ansi> runs along one wall, a fitted slop
  <ansi fg="itemname">bucket</ansi> in the corner, and clean straw scattered
  on the packed-earth floor. The barred door is the only way out, and it
  opens only from the office above.
biome: city
allow_recall: false
coord:
  x: -17
  y: 5
  z: -1
exits:
  up:
    roomid: 4110
nouns:
  window: A slit of a window high in the stone, no wider than a hand and
    crossed by a single iron bar. It lets in a grey wedge of daylight and
    the smell of the lake, and nothing else.
  bench: A low bench of grey, water-stained planks bolted to the wall. The
    wood is worn smooth in the middle where the town's drunks and brawlers
    have slept off their nights.
  bucket: A wooden slop bucket with a fitted lid in the corner. Someone
    scours it out between occupants; it is, all things considered, not the
    worst part of being here.
```

- [ ] **Step 3: Add the reciprocal `down` exit on 4110**

In `_datafiles/world/dogmud/rooms/stillwater/4110.yaml`, under `exits:`
(currently only `west: {roomid: 4109}`), add:

```yaml
  down:
    roomid: 5106
```

- [ ] **Step 4: Boot smoke**

Wipe instances, boot. Expected: room 5106 loads; `look` then `down` from 4110
reaches the cell; `up` returns to 4110. No coordinate-collision warnings for
(x −17, y 5, z −1).

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/rooms/stillwater/5106.yaml _datafiles/world/dogmud/rooms/stillwater/4110.yaml
git commit -m "content(justice): Stillwater holding cell (5106) down from the constabulary"
```

---

## Task 5: Stillwater faction definitions

Two factions mirroring the Thornwall guards/citizens pair. References cell 5106
(created in Task 4) and each other (allies), so boot validation + ally
validation both pass.

**Files:**
- Create: `_datafiles/world/dogmud/factions/stillwater_guards.yaml`
- Create: `_datafiles/world/dogmud/factions/stillwater_citizens.yaml`

- [ ] **Step 1: Create `stillwater_guards.yaml`**

```yaml
faction_id: stillwater_guards
display_name: "Stillwater Constabulary"
description: |
  The lone constable's office of Stillwater -- a quiet lake town that keeps
  order with one tough lawman rather than a garrison.
default_rep: 0
allies: [stillwater_citizens]
enemies: []
holding_cell_room: 5106
```

- [ ] **Step 2: Create `stillwater_citizens.yaml`**

```yaml
faction_id: stillwater_citizens
display_name: "Stillwater Townsfolk"
description: |
  The fishers, traders, and craftspeople of Stillwater. They keep the lake
  town alive and look out for one another.
default_rep: 0
allies: [stillwater_guards]
enemies: []
```

- [ ] **Step 3: Boot smoke**

Wipe instances, boot. Expected: `factions.LoadAllDefinitions` logs
`loadedCount` increased by 2; ally validation passes (each names the other);
`ValidateHoldingCells` passes (5106 exists). No panic.

- [ ] **Step 4: Verify in-game (admin)**

`faction list` shows both new factions; `faction show stillwater_guards`
reflects default_rep 0.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/factions/stillwater_guards.yaml _datafiles/world/dogmud/factions/stillwater_citizens.yaml
git commit -m "content(justice): stillwater_guards + stillwater_citizens factions"
```

---

## Task 6: Constable Drunn → combat-capable lone enforcer

Flip Drunn (335) from a non-combat questgiver to the `guard_captain` hybrid
(questgiver + combat, like Velk), bump his stats to tank tier, and add the
faction groups. He already carries the `guard` group, so `isGuardMob` already
ticks `RunGuardEnforcement` for him — adding the faction flips enforcement on.

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/stillwater/335-constable_drunn.yaml`

- [ ] **Step 1: Apply the combat-profile + faction edits**

Edit `335-constable_drunn.yaml`:

- Line 3: `archetype: fighting` → `archetype: tank`
- Line 4: `behavior_archetype: noncombat_questgiver` → `behavior_archetype: guard_captain`
- Line 5: `statpool: 80` → `statpool: 240`
- Lines 10-12 `groups:` — add the two faction groups after `guard`:

```yaml
groups:
  - humanoid
  - guard
  - stillwater_guards
  - stillwater_citizens
```

- Replace the `stats:` block (lines 40-48) with a tank distribution (mirrors
  Velk's training seeds; a melee constable leads on strength/vitality):

```yaml
  stats:
    strength:
      training: 10
    vitality:
      training: 8
    perception:
      training: 8
    willpower:
      training: 6
```

Leave everything else (name, description, equipment club 10010 + body 20011,
`hostile: false`, `charm_immune: true`, idlecommands, his quest-19 dialogue)
unchanged.

- [ ] **Step 2: Boot smoke**

Wipe instances, boot. Expected: Drunn loads with `guard_captain` archetype
(no archetype-resolution panic), statpool 240, faction groups present.

- [ ] **Step 3: Verify enforcement wiring (admin / in-game)**

`relationship`/`faction show stillwater_guards` is unaffected; confirm Drunn
spawns in 4110. A wanted player (see Task-9 smoke) standing in 4110 draws
Drunn's warn/arrest on his next tick — full validation in Task 10 in-game smoke.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/stillwater/335-constable_drunn.yaml
git commit -m "content(justice): Constable Drunn -> combat-capable guard_captain + stillwater factions"
```

---

## Task 7: Citizen faction tagging

Tag the Stillwater townsfolk, the foragers, and the human caravan crew with the
appropriate citizen faction(s). Citizens are crime *witnesses* only —
`isGuardMob` gates enforcement to the `guard` group, so none of these attack
wanted players.

**EXCLUDE** the three lake-cave monsters (330 skitter_shrimp_swarm, 331
drowned_hunter, 332 sump_dweller) — they are quest-19 enemies, not townsfolk.
**EXCLUDE** Drunn (335, handled in Task 6) and the caravan animals (375 Hob,
376 Bran).

**Files (Stillwater townsfolk — add `- stillwater_citizens` to each one's `groups:`):**
```
333-innkeeper_sigrid  334-barmaid_neva  336-fishmonger_tov_brann
337-smith_brindle  338-apothecary_ilsa  339-weaver_edda  340-pearl_carver_kess
341-storekeeper_wulf  342-dock_master_arn  343-old_fisherman_hodder
344-temple_priest_seren  345-temple_acolyte_finn  346-young_fisherman_luc
347-ulla  348-miller_bram  349-old_cottager_gyda  350-child_pip
353-traveling_pilgrim_fenwick  354-beggar_oswin  355-mistress_vella_thorne
356-counting_house_clerk
```
(all under `_datafiles/world/dogmud/mobs/stillwater/`)

**Files (foragers):**
- `_datafiles/world/dogmud/mobs/stillwater_marsh/371-tova.yaml` → add `stillwater_citizens`
- `_datafiles/world/dogmud/mobs/the_fernway_south/373-kessa.yaml` → add `stillwater_citizens`
- `_datafiles/world/dogmud/mobs/ironwind_steppe/372-halix.yaml` → add `thornwall_citizens`

**Files (caravan humans — add BOTH `thornwall_citizens` + `stillwater_citizens`):**
- `_datafiles/world/dogmud/mobs/thornwall_city/357-ketil.yaml`
- `_datafiles/world/dogmud/mobs/thornwall_city/358-marta.yaml`
- `_datafiles/world/dogmud/mobs/thornwall_city/359-lars.yaml`

- [ ] **Step 1: Tag the 21 Stillwater townsfolk**

For each townsfolk file listed above, append `- stillwater_citizens` to its
existing `groups:` list. Example for `333-innkeeper_sigrid.yaml` (current
`groups: [humanoid, innkeeper, gossiper]`):

```yaml
groups:
  - humanoid
  - innkeeper
  - gossiper
  - stillwater_citizens
```

Apply the same single-line addition to all 21 files. Do **not** add the `guard`
group — these are witnesses, not enforcers.

- [ ] **Step 2: Tag the foragers**

In `371-tova.yaml` and `373-kessa.yaml`, append `- stillwater_citizens` to
`groups:`. In `372-halix.yaml`, append `- thornwall_citizens` to `groups:`.

- [ ] **Step 3: Tag the caravan humans (dual citizenship)**

In `357-ketil.yaml`, `358-marta.yaml`, `359-lars.yaml`, append both to the
existing `groups: [caravan, merchant_train, ...]`:

```yaml
  - thornwall_citizens
  - stillwater_citizens
```

Leave 375-hob and 376-bran (the `animal` draft beasts) untouched.

- [ ] **Step 4: Verify the tagging**

Run:
```bash
grep -L stillwater_citizens _datafiles/world/dogmud/mobs/stillwater/{333,334,336,337,338,339,340,341,342,343,344,345,346,347,348,349,350,353,354,355,356}-*.yaml
```
Expected: no output (every townsfolk file now has the tag).

Run:
```bash
grep -l stillwater_citizens _datafiles/world/dogmud/mobs/stillwater/{330,331,332}-*.yaml
```
Expected: no output (the three monsters are NOT tagged).

- [ ] **Step 5: Boot smoke**

Wipe instances, boot. Expected: all retagged mobs load without panic; mob
loadedCount unchanged.

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/mobs/stillwater/*.yaml _datafiles/world/dogmud/mobs/stillwater_marsh/371-tova.yaml _datafiles/world/dogmud/mobs/the_fernway_south/373-kessa.yaml _datafiles/world/dogmud/mobs/ironwind_steppe/372-halix.yaml _datafiles/world/dogmud/mobs/thornwall_city/357-ketil.yaml _datafiles/world/dogmud/mobs/thornwall_city/358-marta.yaml _datafiles/world/dogmud/mobs/thornwall_city/359-lars.yaml
git commit -m "content(justice): citizen faction tags for Stillwater townsfolk, foragers, caravan crew"
```

---

## Task 8: Quest-rep wiring (+15 on seven civic quests)

Add a `bump_rep` action to each quest's terminal completion node, beside the
existing `grant: <id>-end`, matching the quest-2 pattern
(`2-the_warren_compact.yaml:37-39`).

**Files:**
- `_datafiles/world/dogmud/quests/7-the_fallow_field.yaml` → `thornwall_citizens`
- `_datafiles/world/dogmud/quests/8-the_city_watchs_missing_person.yaml` → `thornwall_guards`
- `_datafiles/world/dogmud/quests/9-the_temples_tithe_audit.yaml` → `thornwall_citizens`
- `_datafiles/world/dogmud/quests/10-the_drowning_posts_debt.yaml` → `thornwall_citizens`
- `_datafiles/world/dogmud/quests/14-the_undertow.yaml` → `thornwall_citizens`
- `_datafiles/world/dogmud/quests/19-the_lake_caves_bounty.yaml` → `stillwater_guards`
- `_datafiles/world/dogmud/quests/20-ullas_silence.yaml` → `stillwater_citizens`

- [ ] **Step 1: Locate each completion node**

For each quest file, find the terminal `actions:` list that contains
`grant: <questId>-end` (the node that completes the quest). Grep to confirm:
```bash
grep -n "grant:" _datafiles/world/dogmud/quests/{7,8,9,10,14,19,20}-*.yaml
```

- [ ] **Step 2: Add the `bump_rep` action to each**

In each quest's completion `actions:` list, add a line directly after the
`grant: <id>-end` line. Use the faction from the table above and `delta: 15`.
Example for `8-the_city_watchs_missing_person.yaml`:

```yaml
    actions:
      - grant: 8-end
      - bump_rep: {faction: thornwall_guards, delta: 15}
```

And for `20-ullas_silence.yaml`:

```yaml
    actions:
      - grant: 20-end
      - bump_rep: {faction: stillwater_citizens, delta: 15}
```

Apply the matching faction for the other five quests. Match the existing
indentation of each file exactly.

- [ ] **Step 3: Boot smoke**

Wipe instances, boot. Expected: `quests.LoadDataFiles() loadedCount=...` logs
the same count as before with no parse error; no panic about unknown faction
(all four referenced factions exist).

- [ ] **Step 4: Verify**

```bash
grep -A1 "grant: \(7\|8\|9\|10\|14\|19\|20\)-end" _datafiles/world/dogmud/quests/{7,8,9,10,14,19,20}-*.yaml
```
Expected: each shows the matching `bump_rep` line.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/quests/7-the_fallow_field.yaml _datafiles/world/dogmud/quests/8-the_city_watchs_missing_person.yaml _datafiles/world/dogmud/quests/9-the_temples_tithe_audit.yaml _datafiles/world/dogmud/quests/10-the_drowning_posts_debt.yaml _datafiles/world/dogmud/quests/14-the_undertow.yaml _datafiles/world/dogmud/quests/19-the_lake_caves_bounty.yaml _datafiles/world/dogmud/quests/20-ullas_silence.yaml
git commit -m "content(justice): +15 faction rep on seven civic quests (quest-rep audit)"
```

---

## Task 9: Warn-stamp in-tick sweep

Prune stale `justice_warned_*` MiscData on guards. Mob instance saves are not
deployed to prod (wiped per SOP), so a guard-save hook would not fire there —
sweep in-tick at the top of `RunGuardEnforcement`, which already runs per-round
for guard mobs. The arrest-pending stamp self-cleans on haul and is left alone.

**Files:**
- Modify: `internal/justice/enforce.go` (add sweep helper + call it)
- Test: `internal/justice/enforce_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/justice/enforce_test.go`:

```go
func TestPruneStaleWarnStamps(t *testing.T) {
	md := map[string]any{
		"justice_warned_1":         uint64(100), // stale (now-100 > staleAfter)
		"justice_warned_2":         uint64(950), // fresh
		"justice_arrest_pending_3": uint64(100), // must be left alone
		"some_other_key":           uint64(100), // must be left alone
	}

	pruneStaleWarnStamps(md, 1000, 200) // now=1000, staleAfter=200 rounds

	if _, ok := md["justice_warned_1"]; ok {
		t.Fatalf("stale justice_warned_1 should have been pruned")
	}
	if _, ok := md["justice_warned_2"]; !ok {
		t.Fatalf("fresh justice_warned_2 must be kept")
	}
	if _, ok := md["justice_arrest_pending_3"]; !ok {
		t.Fatalf("arrest-pending stamp must not be touched")
	}
	if _, ok := md["some_other_key"]; !ok {
		t.Fatalf("unrelated key must not be touched")
	}
}
```

(If the mob's MiscData type is not `map[string]any`, match the actual type —
grep `mob.Character.MiscData` declaration. The helper must accept that type.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/justice/ -run TestPruneStaleWarnStamps -v`
Expected: FAIL — `pruneStaleWarnStamps` undefined.

- [ ] **Step 3: Implement the helper**

In `internal/justice/enforce.go`, add (reuse the existing `miscDataRound`
helper this file already uses to read round-stamps):

```go
// pruneStaleWarnStamps deletes justice_warned_* entries older than staleAfter
// rounds. Cold-rep warn stamps are never revisited once a player's rep
// recovers (the Warn branch stops running), so they would otherwise leak.
// Leaves justice_arrest_pending_* (self-cleaning on haul) and all other keys.
func pruneStaleWarnStamps(md map[string]any, now, staleAfter uint64) {
	for key := range md {
		if !strings.HasPrefix(key, "justice_warned_") {
			continue
		}
		stamped, ok := miscDataRound(md, key)
		if !ok {
			continue
		}
		if now >= stamped && now-stamped > staleAfter {
			delete(md, key)
		}
	}
}
```

Add `"strings"` to the imports if not already present.

- [ ] **Step 4: Call it from `RunGuardEnforcement`**

Near the top of `RunGuardEnforcement` (after the `guardFactions` empty check,
before the player loop), add:

```go
	pruneStaleWarnStamps(mob.Character.MiscData, nowRound, warnStampStaleAfter())
```

Add a small config-backed staleness helper alongside the other `*Rounds()`
helpers in this file (reuse the crime lookback window as the threshold):

```go
func warnStampStaleAfter() uint64 {
	v := configs.GetBalanceConfig().JusticeCrimeLookbackRounds
	if v < 1 {
		return 1000
	}
	return uint64(v)
}
```

(Confirm `configs` is already imported in `enforce.go`; if not, add it. If a
`nowRound` variable is not already in scope at that point in
`RunGuardEnforcement`, it is the function's `nowRound` parameter.)

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/justice/ -v`
Expected: PASS.

- [ ] **Step 6: Build + boot smoke + commit**

```bash
go build ./...
git add internal/justice/enforce.go internal/justice/enforce_test.go
git commit -m "feat(justice): sweep stale justice_warned_* stamps in-tick (5.1 followup #2)"
```

---

## Task 10: Docs, memory, and final boot smoke

**Files:**
- Modify: `internal/justice/context.md`
- Modify: `internal/factions/context.md`
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- Modify: `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_town_justice_5_1_followups.md`

- [ ] **Step 1: Update `internal/justice/context.md`**

Add a note: the holding-cell room id moved from the hardcoded `jailCellFor`
map to the faction definition's `holding_cell_room:` field, read via the
`cellRoomFn` seam; the arrest fork now picks the first guard-faction that owns
a cell (`firstFactionWithCell`); stale `justice_warned_*` stamps are swept
in-tick.

- [ ] **Step 2: Update `internal/factions/context.md`**

Document the new `holding_cell_room:` field (optional; 0 = no jail; only guard
factions set it) and its boot validation via `ValidateHoldingCells` (DI'd from
main.go after rooms load).

- [ ] **Step 3: Update the roadmap mini-brief**

In `MOB_ALIVENESS_ROADMAP.md`, update the 5.1 entry: Stillwater rollout slice
shipped — data-driven cell registry, `stillwater_guards`/`stillwater_citizens`
factions, combat-capable Drunn, cell 5106, citizen tagging, quest-rep audit
(+15 on 7 quests), warn-stamp sweep. Note guards/citizens factions now exist
for two towns.

- [ ] **Step 4: Resolve the followup memory**

In `project_town_justice_5_1_followups.md`, mark followup #1 (Drunn faction)
and followup #2 (warn-stamp pruning) as RESOLVED with this slice, dated
2026-05-30. Leave the `guardSay` dark-room note (belongs to the messaging
framework chunk).

- [ ] **Step 5: Full regression + final boot smoke**

```bash
go build ./...
go test ./...
```
Then wipe instances and boot. Expected: all tests pass; clean boot past data
load; both Stillwater factions, cell 5106, Drunn's new profile, all tagged
mobs, and all seven edited quests load without panic.

- [ ] **Step 6: Commit**

```bash
git add internal/justice/context.md internal/factions/context.md MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(justice): context + roadmap for Stillwater justice rollout"
```

(The memory file lives outside the repo; it is updated but not committed here.)

---

## In-game smoke test plan (deferred to user, per chunk precedent)

1. **Stillwater arrest path:** As a wanted player (unresolved crime or open
   bounty against `stillwater_guards`), enter 4110 with Drunn present →
   warn → arrest declaration → after grace, hauled `down` to cell 5106 →
   `pay fine` or serve the sentence → release clears crime + withdraws bounty
   + resets rep. Confirm the cell blocks walk/flee (Jailed buff) and recall
   (`allow_recall:false`).
2. **Citizen witnessing (no guard present):** Commit a crime in a Stillwater
   room with only a tagged civilian present → `crime show` records it against
   `stillwater_citizens` with the civilian as witness → Drunn reacts on his
   next pass through the room.
3. **Wilderness witnessing:** Commit a crime in the marsh near Tova → recorded
   against `stillwater_citizens`; near Halix in Ironwind → recorded against
   `thornwall_citizens`.
4. **Quest rep:** Complete quest 8 (Velk) → `faction show thornwall_guards`
   shows +15; complete quest 20 (Ulla) → `faction show stillwater_citizens`
   shows +15.
5. **Warn-stamp sweep:** After a guard warns a cold-rep player and that
   player's rep recovers, confirm the guard's `justice_warned_*` stamp is gone
   after the staleness window (admin MiscData inspection).
