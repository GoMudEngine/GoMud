# Newbie Rework — Chunk 10: Cutover (Pothole Coulee → live on master)

**Status:** APPROVED 2026-06-16 (design). Sub-spec of the parent
`2026-05-27-newbie-area-rework-design.md` (chunk table row 10) and the
corridor reservation in `newbie-area-coord-budget.md` §6.

**Worktree/branch:** `worktree-feature+newbie-area` (66 commits ahead of
master). This is the FINAL chunk: the build branch merges to master here
and only here.

**Goal:** Connect Pothole Coulee to the wider world, retire Sanctum
Basin, repoint config/respawn, migrate existing characters, and merge to
master so the new newbie experience is live locally for extensive
testing.

**Local-only:** this cutover lands on master **locally** for testing.
The prod push (origin/master) is a SEPARATE later step that must follow
the Pre-Push SOP (PATCH_NOTES, `Logging.LogToFile: false`, boot test).
Until then `config.yaml` dev settings (LogToFile true, MapConsistency
panic) stay as-is for local work.

---

## 0. Sequencing (safety-critical)

Do ALL edits ON THE BRANCH, boot-test on the branch, THEN merge to
master. Master never sees a broken intermediate state.

1. Build the hike-out corridor (§1) + Coulee Rim trailhead.
2. Repoint/remove the orphaned World Road 2001 exit (§2).
3. Config: `DeathRecoveryRoom 75 → 5209` (§3).
4. Add the `MigrateNewbieAwakening` character migration (§4).
5. Delete Sanctum Basin (§5) — only AFTER §2 closes its last external ref.
6. PATCH_NOTES (§6).
7. **Boot-test the branch clean** (corridor loads, Sanctum gone, no
   dangling-exit / undeclared-step panics, `ValidateZoneConsistency`
   0/0, new char spawns 5200, can walk hub→rim→corridor→Ironwind and
   back).
8. Commit on the branch.
9. In the main repo (on master): `git merge --no-ff worktree-feature+newbie-area`.
10. Boot master, confirm clean, hand to user for testing.

---

## 1. Hike-out corridor (~8–10 rooms)

A winding badlands/steppe trail — the "road OUT," real-world difficulty,
NOT sanctuary. Connects Ironwind Steppe's east rim to Pothole's western
boundary, threading the reserved corridor band.

- **West anchor:** Ironwind Steppe room **3074 "Eastern Coulee Edge"**
  (x=18). Add an east exit from 3074 into the corridor.
- **Corridor body:** new rooms in the reserved band **x[19..29], y[-6..6],
  z0** (verified empty). Biome steppe/badlands. Zone **Ironwind Steppe**
  (the road descends from the coulee into the steppe — reuses Ironwind's
  zone-config/biome, no new zone). IDs from `id_inventory --type rooms`
  next-free (≈3075+; verify).
- **East anchor / zone transition:** a new **"Coulee Rim"** surface
  trailhead, zone **Pothole Coulee** (the last coulee room before the
  wilds; the Pothole↔Ironwind zone boundary is crossed here), at the
  zone's western edge. It connects WEST to the corridor and EAST back
  into the existing Pothole graph so a player can walk hub → rim →
  corridor → Ironwind.
- **Hub-side approach:** the western Pothole surface between the hub
  (x45) and the rim (~x30) is currently empty (the Forge spoke occupies
  that x-band underground). The ~8–10 room budget INCLUDES a short
  surface approach from the hub's west side out to the rim if cardinal
  adjacency requires it. **Exact per-room coords, IDs, the zone-boundary
  room, and the hub-side attach room are finalized in the plan via
  `coord_inventory.py` (0 collisions) + `cartcheck` (0/0).** If the chosen
  Ironwind rim room or approach pushes outside y[-6..6], extend the
  reservation FIRST and re-run the emptiness check (coord-budget §6 rule).
- **Two-way, cartesian-clean, all-cardinal** where possible; a `long`
  exit is acceptable for one connector if geometry demands it (cartcheck
  allows long connectors). Noun-token rule + no-digits + 80-col wrap +
  no-hard-numbers apply to all new room prose.
- **Signage:** the rim room and the hub-west approach should make the
  road out discoverable (prose naming the way to the steppe / the wider
  world), and the corridor's Ironwind end should read as arrival in the
  steppe.

## 2. Fix the orphaned world exit

World Road room **2001** (`rooms/world_road/2001.yaml`) has
`north → roomid 103, zone Sanctum Basin` — the old Sanctum entrance.
Pothole now connects via Ironwind, not World Road. **Remove** the north
exit from 2001 (and any prose that promises a northward path). Confirm
2001 has no other Sanctum dependency. This MUST land before §5 (it is the
only external reference into the deleted zone).

## 3. Config

`_datafiles/config.yaml`:
- `StartRoom: 5200` — already correct, no change.
- `DeathRecoveryRoom: 75 → 5209` (75 is a stale Sanctum-era room; align
  with the Mending Hut + the respawn-home default already at 5209).

## 4. Veteran migration — `MigrateNewbieAwakening`

Existing characters who never saw Pothole would be trapped by the room
5200 movement gate (which intercepts movement for players missing
`30-end`). Add a one-time, run-once migration mirroring the existing
`MigrateQuestSpells` pattern in `internal/characters/migrations.go`:

```go
// MigrateNewbieAwakening grants 30-end to pre-Pothole veterans so the
// hub movement gate (room 5200) never traps them. Run-once via MiscData.
// It must NOT fire on brand-new characters (who should do the rite): a
// fresh char is created into the Pothole range, so skip when RoomId is
// in [5200,5499] or the char already has any quest-30 progress.
func (c *Character) MigrateNewbieAwakening() {
    const migrationKey = "migration-newbie-awakening-done"
    if c.GetMiscData(migrationKey) != nil {
        return
    }
    if c.QuestProgress == nil {
        c.QuestProgress = make(map[int]string)
    }
    _, hasQ30 := c.QuestProgress[30]
    inPothole := c.RoomId >= 5200 && c.RoomId <= 5499
    if !hasQ30 && !inPothole {
        c.QuestProgress[30] = "end"   // force the completed state
    }
    c.SetMiscData(migrationKey, "1")
}
```

- Wire it where the other `Migrate*` methods are invoked on character
  load (find the call site of `MigrateQuestSpells`/`MigratePairedSpells`
  and add this alongside).
- Directly setting `QuestProgress[30] = "end"` (not `GiveQuestToken`) is
  required: quest 30 is multi-step, so `GiveQuestToken("30-end")` would
  be refused by the token-ordering logic from a no-progress state.
- Unit test (characters pkg): (a) a veteran (RoomId outside Pothole, no
  q30) gets `30-end`; (b) a fresh char (RoomId 5200) does NOT; (c) a char
  already mid-q30 is untouched; (d) run-once guard holds.
- **Prod-correctness, not a local-test blocker:** local testing works via
  fresh chars (do the rite) + the smoketester (already `30-end`).

## 5. Delete Sanctum Basin

After §2, remove the mapped footprint (no external refs remain):

```
rooms/sanctum_basin/            (101–123 + zone-config)
mobs/sanctum_basin/             (50–79, 112)
dialogue/sanctum_basin/         (50–79 dialogue)
behaviors/sanctum_basin/        (mob behaviors 65, 69)
behaviors/rooms/sanctum_basin/  (room behaviors)
shops/sanctum_basin/            (63-room108)
quests/1-the_sanctum_trials.yaml
rooms.instances/sanctum_basin/  (123 instance save)
```

- Quest 1 deletion approved (user 2026-06-16): existing saves holding
  quest-1 progress are harmless (the loader simply won't find quest 1).
- Flavor-text mentions of "Sanctum Basin" in other zones
  (dustwalk_road/400,405; thornwall_outskirts/445; watchers_crossing/427;
  hints.yaml) are non-breaking — leave. Optionally tidy the orphaned
  `labyrinth_of_low_tunnels/300` zone field (it reads "Sanctum Basin" but
  is unreachable) — low priority.
- After deletion, grep the tree once more for the Sanctum room/mob ID
  ranges (101–123 / 50–79) as exit targets / spawn refs to confirm zero
  dangling references (detection pass).

## 6. PATCH_NOTES + verification

- `PATCH_NOTES.md`: dated 2026-06-16 entry — newbie-area cutover (Pothole
  Coulee live; hike-out corridor to Ironwind; Sanctum Basin retired;
  StartRoom/DeathRecovery repointed; veteran 30-end migration; C9 polish
  summary).
- **Branch boot test (acceptance):** `go build ./...` 0; `go test`
  (characters for the migration); nuke instances; boot —
  `quests.LoadDataFiles` count drops by 1 (quest 1 gone), rooms count
  rises by the corridor count, `ValidateAllFlags` ok,
  `ValidateZoneConsistency errors=0 warnings=0` (panic mode, incl. the
  new corridor + the removed 2001 exit), Server Ready, 0 panics.
- `coord_inventory.py` 0 collisions (corridor rooms added). `cartcheck`
  for Ironwind Steppe + Pothole Coulee clean. `newbie_manifest_check.py`
  extended with the corridor rooms (0 FAIL).
- **Live walk:** fresh char spawns 5200; completes hub funnel; walks hub →
  Coulee Rim → corridor → Ironwind 3074 and back (two-way); a veteran-
  style char (RoomId outside Pothole, no q30) loads with `30-end` granted;
  death → wakes at 5209.
- Merge `--no-ff` to master; boot master clean.

---

## 7. Risks / gotchas

- **Delete order:** §2 (close 2001) BEFORE §5 (delete Sanctum), else a
  dangling exit panics the boot.
- **Corridor geometry:** the empty western-Pothole surface means the
  hub→rim approach may need a couple of surface rooms; keep all within
  the reserved/empty bands and re-run coord checks. A `long` exit is the
  escape hatch if one cardinal hop is impossible.
- **Migration false-positive:** the RoomId-in-Pothole + has-q30 guard is
  what protects brand-new chars from skipping the rite — both conditions
  matter; verify the new-player flow sets RoomId=5200 BEFORE load-time
  migrations run (check at plan time).
- **Instance saves:** nuke `mobs.instances/*` + `rooms.instances/*` before
  every boot test (stale saves shadow the new corridor + deleted zone).
- **Merge hygiene:** the merge brings the branch's committed `config.yaml`
  (StartRoom 5200, DeathRecovery 5209). The working-tree dev edits
  (LogToFile true) are local; do NOT let prod settings leak — but this is
  a LOCAL master, prod push is later under the SOP.
- **No console popups** unless the user has OK'd a popup window (they have
  for this session); otherwise defer build/boot/merge to a user window.

## 8. After C10
Newbie rework COMPLETE on master (local). Remaining: (a) prod push under
the Pre-Push SOP (PATCH_NOTES, LogToFile false, boot test) — bundles the
whole rework + the C9 work + still-parked web/weather work; (b) standing
PROD TODO: cherry-pick `697169bf` (archer nil-Aggro crash fix); (c) the
evening naive playtest for whole-area difficulty tuning.
