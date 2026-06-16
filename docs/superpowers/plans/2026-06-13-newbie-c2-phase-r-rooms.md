# Newbie Chunk 2 (Spoke A — Martial) — Phase R Plan (Rooms + Nouns)

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development.
> Design of record: `docs/superpowers/specs/2026-06-13-newbie-chunk2-spoke-a-martial-subspec.md` (§3).

**Goal:** Author the 17 Spoke A rooms (5227–5243) — a dry training canyon
climbing east from the existing hub stub 5220 up to a ruined watchtower —
with nouns, coords, biomes, and the sanctuary boundary; wire 5220's east
exit in; pass all audits; produce a walkthrough artifact. STOP at the
Phase R review gate. NO mobs, items, dialogue, or quests (later phases).

**Working dir (ALL tasks):** `C:/Users/Calabe Davis/workspace/DOGMud/.claude/worktrees/feature+newbie-area`. Never `git add -A`.

---

## Verified facts (do not re-derive)

- **Room YAML shape** — read an existing Pothole room as the model:
  `_datafiles/world/dogmud/rooms/pothole_coulee/5220.yaml` (roomid, zone
  `Pothole Coulee`, title, `description: >` folded block, biome, coord
  {x,y,z}, optional `mutators: [- mutatorid: sanctuary]`, exits map,
  nouns map, optional idlemessages). Filename = `<roomid>.yaml`.
- **5220 already exists** (Dry Coulee Mouth, sanctuary, (48,0,0)) and its
  `east` exit is currently absent/dead — Task 1 adds `east → 5227`.
  Touch NOTHING else in 5220.
- **Biomes available:** shore, cliffs, mountains, fort, cave, city, etc.
  `fort` and `cave` are `indoor: true` (weather won't render inside) —
  use `fort` for tower-interior rooms. Spoke palette: **shore** (lower
  canyon floor, continues 5220), **cliffs** (the climbing wash + tower
  approach), **fort** (tower interior: stair + watch room).
- **Sanctuary boundary:** inner-ring drill yard rooms 5227–5231 carry
  `mutators: [- mutatorid: sanctuary]` (safe training; also the 5× regen).
  5232 onward (middle + outer) have **NO** sanctuary — real combat. 5231
  is "The Last Safe Step": its prose says crossing east leaves safety.
- **Coordinates are ENGINE-IGNORED** (authoring discipline only);
  `tools/coord_inventory.py` enforces cross-zone uniqueness. Engine
  mapper convention: **north = y−1, south = y+1, east = x+1, up = z+1.**
  Spoke A reserve: x 49–59, y −6…+6, z 0–1 (clear of hub x42–48 and
  corridor x19–29). Use the exact coords in the manifest below.
- **Exits must be reciprocal** (cartcheck panics otherwise) except where a
  one-way is intentional (`oneway: true`). Every exit here is two-way.
- **Vertical exits** (5240↔5241, 5241↔5242) use `up`/`down` and the
  z-coordinate (z+1 per "up"). The mapper renders these as ▲/▼ ticks.
- **Noun discoverability (HARD rule, playtest-enforced):** every feature
  named in room prose that a player might `look <noun>` at MUST be a noun
  key. The ch.1 playtest caught a missing `water` noun the manifest
  checker had not flagged — so for EACH room, every concrete noun in the
  description gets a noun entry (3–5 per room typical). The manifest
  checker (Task 6) asserts each noun token appears in its own description.
- **YAML colon trap:** a `: ` inside a plain-scalar noun value parses as a
  mapping and panics at boot. Use em-dash idiom (` -- `) instead of `: `
  inside noun/description prose, or quote the scalar.
- **No hard numbers** in any player-facing prose; **≤80-char source lines**
  for descriptions (folded `>` blocks); scablands voice, mutation-is-normal.
- **5243 (Tower Top)** gets a stubbed lateral-connector exit comment ONLY
  (no actual exit yet) — wired when an adjacent spoke is built. Document
  it in the room as a comment; do not add a dangling exit.

## Room manifest (authoritative — from sub-spec §3)

| Id | Title | Biome | Coord (x,y,z) | Sanctuary | Exits |
|---|---|---|---|---|---|
| 5227 | The Drill Yard | shore | 49,0,0 | yes | W→5220, E→5228, N→5229 |
| 5228 | Weapon Rack Lean-to | shore | 50,0,0 | yes | W→5227 |
| 5229 | Sparring Circle | shore | 49,-1,0 | yes | S→5227, E→5230 |
| 5230 | Yard Overlook | cliffs | 50,-1,0 | yes | W→5229, E→5231 |
| 5231 | The Last Safe Step | cliffs | 51,-1,0 | yes | W→5230, E→5232 |
| 5232 | Lower Wash | cliffs | 52,-1,0 | NO | W→5231, E→5233, S→5234 |
| 5233 | Gravel Bend | cliffs | 53,-1,0 | NO | W→5232, E→5236 |
| 5234 | Squatter's Hollow | cliffs | 52,0,0 | NO | N→5232, E→5235 |
| 5235 | Cracked Cistern | cliffs | 53,0,0 | NO | W→5234, N→5236 |
| 5236 | Upper Wash | cliffs | 54,-1,0 | NO | W→5233, S→5235, E→5237 |
| 5237 | Tower Approach | cliffs | 55,-1,0 | NO | W→5236, E→5238 |
| 5238 | Broken Gate | cliffs | 56,-1,0 | NO | W→5237, E→5239, N→5240 |
| 5239 | Collapsed Barracks | fort | 57,-1,0 | NO | W→5238 |
| 5240 | Tower Base | fort | 56,-2,0 | NO | S→5238, U→5241 |
| 5241 | Tower Stair | fort | 56,-2,1 | NO | D→5240, U→5242 |
| 5242 | The Watch Room | fort | 56,-2,2 | NO | D→5241, E→5243 |
| 5243 | Tower Top | cliffs | 57,-2,2 | NO | W→5242 (+stub comment) |

Exit-reciprocity check (every pair appears on both rooms): 5220↔5227,
5227↔5228, 5227↔5229, 5229↔5230, 5230↔5231, 5231↔5232, 5232↔5233,
5232↔5234, 5234↔5235, 5235↔5236, 5233↔5236, 5236↔5237, 5237↔5238,
5238↔5239, 5238↔5240, 5240↔5241, 5241↔5242, 5242↔5243. (5236 has W,S,E =
3 exits; 5232 has W,E,S = 3; 5238 has W,E,N = 3 — all ≤5, fine.)

---

### Task 1: Inner-ring drill yard (5227–5231) + wire 5220
**Files:** create `rooms/pothole_coulee/5227.yaml`–`5231.yaml`; modify
`5220.yaml` (add `east: {roomid: 5227}` ONLY).

- [ ] Author the 5 inner rooms per the manifest. ALL carry the sanctuary
  mutator. Voice/flavor: a drillmaster's training ground at the dry
  coulee mouth — 5227 the yard with a beaten practice post and a
  straw-and-leather dummy (foreshadows the dummy mob), 5228 a lean-to of
  racked practice weapons, 5229 a scuffed sparring circle, 5230 an
  overlook with a sightline EAST up the wash to the distant ruined tower
  (boss foreshadow — name it in prose + noun), 5231 "The Last Safe Step"
  whose prose explicitly marks the edge of safety (east = real danger).
  Nouns: post, dummy, rack, circle, tower (overlook), etc. — every
  concrete feature look-able.
- [ ] Add `east → 5227` to 5220; verify 5220 otherwise unchanged.
- [ ] Commit: `git commit -m "content(newbie-c2): Spoke A inner ring (drill yard) + hub stub wiring"`

### Task 2: Middle-ring wash (5232–5237)
**Files:** create `5232.yaml`–`5237.yaml`.

- [ ] Author the 6 middle rooms (NO sanctuary). Flavor: a contested
  gravel wash held by bandit squatters — broken gear, cold fire-pits, a
  cracked cistern, a hollow where a held-up caravan guard shelters
  (5234, foreshadows the quest NPC). 5237 "Tower Approach" pivots the
  climb toward the outer ring. Real-danger tone without hard numbers.
  Nouns throughout.
- [ ] Commit: `git commit -m "content(newbie-c2): Spoke A middle ring (the wash)"`

### Task 3: Outer-ring watchtower (5238–5243)
**Files:** create `5238.yaml`–`5243.yaml`.

- [ ] Author the 6 outer rooms. 5238 Broken Gate (cliffs), 5239
  Collapsed Barracks (fort, indoor), 5240 Tower Base (fort) with the
  stairs up, 5241 Tower Stair (fort, z=1), 5242 The Watch Room (fort,
  z=2) — the captain's room, tense and commanding, sightlines over the
  whole canyon, 5243 Tower Top (cliffs, z=2) the post-boss vista. 5243
  carries a YAML comment marking the reserved lateral-connector exit to a
  future adjacent spoke (NO actual dangling exit). Vertical exits use
  up/down + z. Nouns throughout.
- [ ] Commit: `git commit -m "content(newbie-c2): Spoke A outer ring (the watchtower)"`

### Task 4: Coordinate + cartcheck audit
- [ ] `python tools/coord_inventory.py` → **0 collisions** (the new rooms
  must not collide with any existing coord world-wide). Fix any clash by
  nudging within the x49–59 reserve.
- [ ] Boot the server (wipe instances first: `rm -rf
  _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`;
  build `go build -o /tmp/dogmud-c2r.exe .`; launch hidden via powershell
  `Start-Process -WindowStyle Hidden -RedirectStandardError`; wait ~45s;
  confirm `rooms.LoadDataFiles` count rose by 17 and ZERO panics). Then on
  the AI port run `cartcheck pothole_coulee` as smoketester → clean (no
  collisions, no non-reciprocal exits, no stray wrap exits).
- [ ] No commit (audit only) unless a coord fix was needed.

### Task 5: Manifest-checker extension
**Files:** modify `tools/newbie_manifest_check.py`.

- [ ] Add a Spoke A room block: assert all 17 files exist with correct
  roomid/zone/title/biome; assert the sanctuary mutator is present on
  5227–5231 and ABSENT on 5232–5243; assert exit reciprocity for the
  pairs listed above; assert every noun key appears as a token in its own
  room's description (the discoverability rule). Run → ALL PASS (ch.1
  rooms + Spoke A rooms).
- [ ] Commit: `git commit -m "test(newbie-c2): manifest-check Spoke A rooms"`

### Task 6: Walkthrough artifact + STOP
**Files:** create `docs/superpowers/specs/newbie-c2-phase-r-walkthrough.txt`.

- [ ] Boot; AI port as smoketester; `teleport` to each of the 17 rooms in
  id order; `look` (room) + `look <each noun>` for every noun key; ANSI-
  strip → the artifact with per-room headers and a header block (date,
  build, method). Confirm the sanctuary boundary reads correctly in prose
  at 5231→5232. Kill server; re-wipe instances; no strays.
- [ ] Commit: `git commit -m "content(newbie-c2): Spoke A Phase R walkthrough"`
- [ ] **STOP — Phase R review gate.** Do NOT start Phase M (mobs/items).

---

## Self-review notes
- Sub-spec §3 fully covered (17 rooms, rings, sanctuary boundary, 5220
  wiring, tower verticality, 5243 connector stub).
- Phase R is rooms+nouns ONLY. Mobs (drillmaster, dummy, bandits,
  captain), items, dialogue, quests, and the default-home re-point are
  Phases M and D.
- Noun discoverability is the ch.1 playtest's hard-won lesson — the
  manifest checker enforces it so a naive player never hits "look at
  what???" on a feature the prose named.
