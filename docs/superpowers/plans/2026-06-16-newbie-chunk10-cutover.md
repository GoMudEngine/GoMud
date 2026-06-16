# Chunk 10 — Cutover Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Connect Pothole Coulee to the wider world (hike-out corridor to Ironwind), retire Sanctum Basin, repoint config/respawn, migrate existing characters past the hub gate, then merge the build branch to master so the new newbie experience is live locally.

**Architecture:** Mostly data + one small Go migration. All edits land on the worktree branch and are boot-verified there; then the branch merges `--no-ff` into master. Local-only — prod push is a later, separate step under the Pre-Push SOP.

**Tech Stack:** Go (one `Migrate*` method + test); YAML rooms/config; existing mapper consistency validator (`long` exits over empty cells are clean), `coord_inventory.py`, `cartcheck`, `newbie_manifest_check.py`.

---

## Context the engineer needs (verified 2026-06-16)

- **Branch:** `worktree-feature+newbie-area`, 66 commits ahead of master. Worktree dir: `c:\Users\Calabe Davis\workspace\DOGMud\.claude\worktrees\feature+newbie-area`. Main repo (on `master`): `c:\Users\Calabe Davis\workspace\DOGMud`.
- **Popups:** the user OK'd console popups for this session — go/git/python may run. (Otherwise defer console steps to a user window.)
- **Corridor anchors:** Ironwind **3074 "Eastern Coulee Edge"** at coord (18,−10,0); Pothole western surface anchor **5254 "Mine Mouth"** at (35,−1,0) (westmost z0 room; the Forge mine continues at z<0). Surface (z0) at x[19..34] is empty.
- **Long exits:** `internal/mapper/mapper.consistency.go` — `ExitLong` is fine; only a *warn* (`longcrossing`) if the straight span crosses an *occupied* cell. z0 spans over the badlands cross nothing (mine is z<0). Errors are collision / noreciprocal / deltamismatch only.
- **Next-free IDs:** rooms — Pothole next-free **5371**; Ironwind next-free **3116**. (Re-run `id_inventory.py --type rooms` before authoring.)
- **Config:** `_datafiles/config.yaml` — `StartRoom: 5200` (line ~1218, correct), `DeathRecoveryRoom: 75` (line ~1223 — 75 is the shadow_realm room; repoint to 5209).
- **Migration call-site:** `internal/users/users.go:492-494` (`MigratePairedSpells()`, `MigrateNeckToBack()`, `MigrateQuestSpells()` run on character load; `QuestProgress` is populated there — `MigrateQuestSpells` already reads it via `HasQuest`).
- **World Road 2001** (`rooms/world_road/2001.yaml`): exits `north → 103 (Sanctum Basin)` and `south → 400 (Dustwalk Road)`. Remove the north exit.
- **Sanctum Basin footprint** (delete): `rooms/sanctum_basin/` (101–123 + zone-config), `mobs/sanctum_basin/` (50–79,112), `dialogue/sanctum_basin/`, `behaviors/sanctum_basin/`, `behaviors/rooms/sanctum_basin/`, `shops/sanctum_basin/`, `quests/1-the_sanctum_trials.yaml`, `rooms.instances/sanctum_basin/`. Only external ref is World Road 2001 (closed in Task 1).

---

## File Structure
- Modify: `_datafiles/world/dogmud/rooms/world_road/2001.yaml` (remove north exit).
- Modify: `_datafiles/config.yaml` (DeathRecoveryRoom).
- Create: `internal/characters/migrations.go` (+method); `internal/characters/migrate_newbie_test.go` (test). Modify: `internal/users/users.go` (wire the call).
- Create: corridor rooms — `rooms/pothole_coulee/5371*.yaml` (Coulee Rim) + `rooms/ironwind_steppe/3116*.yaml` … (~8–10 total). Modify: `rooms/ironwind_steppe/3074.yaml` + `rooms/pothole_coulee/5254-mine_mouth.yaml` (add the connecting exits).
- Delete: the Sanctum Basin footprint.
- Modify: `PATCH_NOTES.md`.

---

## Task 1: Close the World Road 2001 → Sanctum exit (MUST precede deletion)

**Files:** Modify `_datafiles/world/dogmud/rooms/world_road/2001.yaml`.

- [ ] **Step 1: Read 2001.yaml**, confirm the exits block:
```yaml
exits:
  north:
    roomid: 103
    zone: Sanctum Basin
  south:
    roomid: 400
    zone: Dustwalk Road
```
- [ ] **Step 2: Remove the `north:` exit** (the whole 3-line block), leaving `south → 400`. If the room's `description`/prose promises a northward path to a sanctuary/basin, soften it (no dangling narrative). Keep the rest of the file intact.
- [ ] **Step 3: Verify** no other field in 2001 references room 103 / Sanctum. Report the final exits block.
- [ ] **Step 4: Commit**
```bash
git add _datafiles/world/dogmud/rooms/world_road/2001.yaml
git commit -m "chore(world): drop World Road 2001 north exit to retired Sanctum Basin"
```

---

## Task 2: Repoint DeathRecoveryRoom 75 → 5209

**Files:** Modify `_datafiles/config.yaml`.

- [ ] **Step 1:** Read the `DeathRecoveryRoom:` line (~1223). Confirm it reads `DeathRecoveryRoom: 75`.
- [ ] **Step 2:** Change to `DeathRecoveryRoom: 5209` (the Mending Hut, matching the respawn-home default).
- [ ] **Step 3: Commit**
```bash
git add _datafiles/config.yaml
git commit -m "config(newbie): DeathRecoveryRoom -> 5209 (Mending Hut)"
```
(Note: config.yaml also has uncommitted local dev edits — stage ONLY the DeathRecoveryRoom change with `git add -p` if other hunks are present.)

---

## Task 3: Veteran migration `MigrateNewbieAwakening`

**Files:** Modify `internal/characters/migrations.go`; create `internal/characters/migrate_newbie_test.go`; modify `internal/users/users.go`.

- [ ] **Step 1: Write the failing test** — create `internal/characters/migrate_newbie_test.go`:
```go
package characters

import "testing"

func TestMigrateNewbieAwakening_Veteran(t *testing.T) {
	c := &Character{QuestProgress: map[int]string{8: "end"}} // old-world progress, no q30
	c.MigrateNewbieAwakening()
	if c.QuestProgress[30] != "end" {
		t.Errorf("veteran: want 30=end, got %q", c.QuestProgress[30])
	}
}

func TestMigrateNewbieAwakening_FreshChar(t *testing.T) {
	c := &Character{} // brand-new: empty progress
	c.MigrateNewbieAwakening()
	if _, ok := c.QuestProgress[30]; ok {
		t.Errorf("fresh char: 30 should be unset, got %q", c.QuestProgress[30])
	}
}

func TestMigrateNewbieAwakening_MidRite(t *testing.T) {
	c := &Character{QuestProgress: map[int]string{30: "start"}} // doing the rite
	c.MigrateNewbieAwakening()
	if c.QuestProgress[30] != "start" {
		t.Errorf("mid-rite: must not overwrite, got %q", c.QuestProgress[30])
	}
}

func TestMigrateNewbieAwakening_RunsOnce(t *testing.T) {
	c := &Character{QuestProgress: map[int]string{8: "end"}}
	c.MigrateNewbieAwakening()
	delete(c.QuestProgress, 30)       // simulate a later quest-30 reset
	c.MigrateNewbieAwakening()        // must NOT re-grant (guard set)
	if _, ok := c.QuestProgress[30]; ok {
		t.Errorf("run-once: should not re-grant after guard set")
	}
}
```

- [ ] **Step 2: Run, expect FAIL**
Run: `go test ./internal/characters/ -run TestMigrateNewbieAwakening -v`
Expected: FAIL — `c.MigrateNewbieAwakening undefined`.

- [ ] **Step 3: Implement** — append to `internal/characters/migrations.go`:
```go
// MigrateNewbieAwakening grants 30-end to pre-Pothole veteran characters
// so the hub movement gate (room 5200, gated on 30-end) never traps them.
// Run-once via MiscData. Guard: grant only when the character already has
// SOME quest progress but no quest-30 entry — a brand-new character has
// empty progress (or 30:start from the hub auto-grant) and is skipped, so
// it still does the Awakening rite. Errs toward NOT granting (a stray
// veteran is merely told to do the rite by the gate; a skipped rite would
// be worse).
func (c *Character) MigrateNewbieAwakening() {
	const migrationKey = "migration-newbie-awakening-done"
	if c.GetMiscData(migrationKey) != nil {
		return
	}
	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}
	if _, hasQ30 := c.QuestProgress[30]; !hasQ30 && len(c.QuestProgress) > 0 {
		c.QuestProgress[30] = "end"
	}
	c.SetMiscData(migrationKey, "1")
}
```

- [ ] **Step 4: Run, expect PASS**
Run: `go test ./internal/characters/ -run TestMigrateNewbieAwakening -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Wire the call** — in `internal/users/users.go`, after line 494 (`...MigrateQuestSpells()`), add:
```go
	loadedUser.Character.MigrateNewbieAwakening()
```

- [ ] **Step 6: Build + test**
Run: `go build ./... && go test ./internal/characters/ ./internal/users/`
Expected: build 0; PASS.

- [ ] **Step 7: Commit**
```bash
git add internal/characters/migrations.go internal/characters/migrate_newbie_test.go internal/users/users.go
git commit -m "feat(characters): MigrateNewbieAwakening grants 30-end to pre-Pothole veterans"
```

---

## Task 4: Hike-out corridor (~8–10 rooms, Ironwind 3074 ⟷ Pothole Mine Mouth 5254)

**Files:** Create the Coulee Rim (`rooms/pothole_coulee/5371-coulee_rim.yaml`) + ~8–9 corridor rooms (`rooms/ironwind_steppe/3116-*.yaml` …). Modify `rooms/ironwind_steppe/3074.yaml` and `rooms/pothole_coulee/5254-mine_mouth.yaml` to add the connecting exits.

**Route (the road OUT — badlands/steppe, real-world difficulty, NOT sanctuary):**
- **East end (Pothole):** `5254 Mine Mouth` (35,−1,0) gains a **west** exit to **Coulee Rim 5371** (place at (34,−1,0), zone Pothole Coulee). The Rim is the last coulee room — prose makes the way out to the steppe discoverable.
- **Corridor body (Ironwind Steppe zone, IDs 3116+):** ~8–9 rooms winding from the Rim down/across the empty badlands to Ironwind. They occupy z0 at x∈[19..33], y∈[−10..−1] (extend below the reserved y[-6..6] band toward 3074's y=−10 — region is empty, allowed per coord-budget §6). Connect with **normal** exits where cells are adjacent and **long** exits to span empty badlands (clean — nothing at z0 to cross). Winding, evocative of a long hike.
- **West end (Ironwind):** the westmost corridor room connects to **3074 Eastern Coulee Edge** (18,−10,0) — add reciprocal exits (3074 gains an **east** exit into the corridor; corridor gains a **west**/return exit to 3074). The zone boundary (Pothole↔Ironwind) is the Rim↔corridor edge — use cross-zone exits (`zone:` on the exit).

**Authoring rules:** noun-token rule, no digits in prose, 80-col wrap, no-hard-numbers, distinct badlands flavor per room. Title Case room/zone names. Bare-id discipline. Reciprocal exits. Verify files land in the real worktree (no `.clone`/`.claire`).

- [ ] **Step 1:** Run `python tools/id_inventory.py --type rooms` — confirm Pothole next-free 5371 and Ironwind next-free 3116. Read `3074.yaml` and `5254-mine_mouth.yaml` for their exact current exits + coords.
- [ ] **Step 2:** Author the Coulee Rim (5371) + the corridor rooms (3116+), assigning coords along the route above. Keep a running coord list to avoid self-collisions. Add the connecting exits to 3074 and 5254.
- [ ] **Step 3:** Extend `newbie_manifest_check.py` with the corridor rooms (a CORRIDOR section) if it has a room-manifest gate; otherwise skip.
- [ ] **Step 4: Verify (console)** — `python tools/coord_inventory.py` (0 collisions) and a boot with `MapConsistencyEnforce: panic` (it already is) to run `ValidateZoneConsistency` for Ironwind Steppe + Pothole Coulee: expect `errors=0` (warnings for long-over-empty should be 0 since z0 badlands is empty; any `longcrossing` warn means a span crossed an occupied cell — reroute). Iterate until clean.
- [ ] **Step 5: Commit**
```bash
git add _datafiles/world/dogmud/rooms/pothole_coulee/5371-*.yaml \
        _datafiles/world/dogmud/rooms/ironwind_steppe/31*.yaml \
        _datafiles/world/dogmud/rooms/ironwind_steppe/3074.yaml \
        _datafiles/world/dogmud/rooms/pothole_coulee/5254-mine_mouth.yaml \
        tools/newbie_manifest_check.py
git commit -m "content(newbie): hike-out corridor Pothole Coulee <-> Ironwind Steppe east rim"
```

---

## Task 5: Delete Sanctum Basin (AFTER Task 1)

**Files:** Delete the footprint.

- [ ] **Step 1: Confirm Task 1 landed** (World Road 2001 has no north→103 exit). Then delete:
```bash
cd "<worktree>"
git rm -r _datafiles/world/dogmud/rooms/sanctum_basin/
git rm -r _datafiles/world/dogmud/mobs/sanctum_basin/
git rm -r _datafiles/world/dogmud/dialogue/sanctum_basin/
git rm -r _datafiles/world/dogmud/behaviors/sanctum_basin/ 2>/dev/null || true
git rm -r _datafiles/world/dogmud/behaviors/rooms/sanctum_basin/ 2>/dev/null || true
git rm -r _datafiles/world/dogmud/shops/sanctum_basin/
git rm _datafiles/world/dogmud/quests/1-the_sanctum_trials.yaml
rm -rf _datafiles/world/dogmud/rooms.instances/sanctum_basin/   # instance saves (gitignored)
```
- [ ] **Step 2: Detection pass** — grep the tree for any surviving reference to the deleted ranges as exit targets / spawns / zone fields:
```bash
grep -rnE "zone: *Sanctum Basin|roomid: *(1[01][0-9]|12[0-3])\b" _datafiles/world/dogmud/ | grep -v "sanctum_basin/"
grep -rn "sanctum_basin" internal/ _datafiles/ | grep -v "_archive"
```
Expected: only benign flavor-text mentions (dustwalk_road/400,405; thornwall_outskirts/445; watchers_crossing/427; hints.yaml) — leave those. Any functional exit/spawn ref → fix before proceeding. (Optionally fix the orphaned `labyrinth_of_low_tunnels/300` zone field.)
- [ ] **Step 3: Commit**
```bash
git add -A _datafiles/world/dogmud/
git commit -m "content(newbie): retire Sanctum Basin (rooms/mobs/dialogue/shop/quest1 deleted)"
```

---

## Task 6: PATCH_NOTES, branch boot verification, merge to master

- [ ] **Step 1: PATCH_NOTES** — add a dated 2026-06-16 entry: newbie-area cutover (Pothole Coulee live; hike-out corridor to Ironwind; Sanctum Basin retired; StartRoom 5200 / DeathRecoveryRoom 5209; veteran 30-end migration; C9 polish: repeatables + living hub). Commit.
- [ ] **Step 2: Branch boot test (console)**
```bash
cd "<worktree>"
go build ./... && go test ./internal/characters/ ./internal/users/
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go build -o GoMud_c10.exe . && ./GoMud_c10.exe > _c10boot.log 2>&1   # boot, watch, kill
```
Expected, no panics: `quests.LoadDataFiles` count = **50** (was 51; quest 1 gone), rooms count up by the corridor count, mobs count down by the Sanctum mobs (18), `ValidateAllFlags` ok, `ValidateZoneConsistency errors=0 warnings=0` (incl. corridor + the removed 2001 exit), Server Ready. Grep the log for `panic|FATAL|sanctum|unreachable|did not end`.
- [ ] **Step 3: Live walk (AI port, smoketester)** — fresh char spawns 5200; walk hub → Coulee Rim → corridor → Ironwind 3074 and back (two-way reciprocal); confirm a veteran-style char (QuestProgress nonempty, no q30) loads with 30-end; `DeathRecoveryRoom` resolves to 5209. Kill server, clean `GoMud_c10.exe` + logs.
- [ ] **Step 4: Merge to master**
```bash
cd "c:/Users/Calabe Davis/workspace/DOGMud"   # main repo, on master
git merge --no-ff worktree-feature+newbie-area -m "Merge newbie-area rework (Pothole Coulee) — chunks 1-10 cutover"
```
- [ ] **Step 5: Boot master** — same instance-nuke + boot + `ValidateZoneConsistency 0/0` + 0-panic check, from the main repo on master. Confirm a fresh char spawns into Pothole 5200 and the world is reachable. Hand to the user for extensive testing.

---

## Self-review notes (author)

- **Spec coverage:** Implements C10 sub-spec §1 (corridor) §2 (2001) §3 (config) §4 (migration) §5 (deletion) §6 (PATCH_NOTES/verify/merge). Sequencing (§0) is enforced by task order: Task 1 (close 2001) precedes Task 5 (delete); Task 6 (merge) is last.
- **Migration guard refined** vs the spec's RoomId idea to a load-order-independent `len(QuestProgress)>0 && no q30` — documented in the method comment + covered by 4 tests; errs toward not-granting (safe).
- **Corridor feasibility:** `long` exits over empty z0 badlands are clean (verified in mapper.consistency.go); the ~8–10 room target is reachable by spanning empty cells, with `cartcheck` as the gate. Exact coords/IDs are placed under coord_inventory + cartcheck (the per-chunk pattern).
- **Irreversibility:** deletion uses `git rm` (recoverable via git history); the merge is `--no-ff` (one revert point). Local-only — no prod push here.
- **Type/name consistency:** `MigrateNewbieAwakening` used consistently in Tasks 3 (define + test + wire). DeathRecoveryRoom 5209, StartRoom 5200, corridor anchors 3074/5254/5371 consistent across tasks.
