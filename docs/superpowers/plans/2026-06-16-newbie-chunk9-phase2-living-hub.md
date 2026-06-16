# Chunk 9 Phase 2 — Living Hub (Schedules + Conversations) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Pothole Coulee hub feel alive — give the 8 hub NPCs day/night schedules and seed NPC↔NPC conversations among the evening crowd — WITHOUT moving any NPC the newbie must reach for onboarding (quests 30/31).

**Architecture:** Pure DATA. The schedule system, relationship registry, and conversation system all already exist in the engine — Phase 2 authors YAML only. Schedules attach via `schedule_id:` on the mob and live at `schedules/pothole_coulee/<id>.yaml`; the loader panics at boot on coverage gaps / unreachable target rooms / unresolved ids. Relationship edges are `relationships:` blocks on mob specs; conversations require an edge between the pair and draw from existing type pools plus optional pair-override files. No Go code, no tests beyond the load-time validators + a live spot-check.

**Tech Stack:** YAML data files; engine schedule validator (`internal/mobs/schedule_loader.go`), relationship registry (`internal/relationships/`), conversation loader (`internal/conversations/loader.go`).

---

## Context the engineer needs (verified 2026-06-16)

### The 8 hub NPCs and their home rooms
| Mob | Name | Home room | Onboarding role |
|-----|------|-----------|-----------------|
| 9100 | Cleric Hadwen | 5200 The Awakening Pool | **CRITICAL** — grants the rite (quest 30); must be askable 24/7 |
| 9101 | Innkeep Tally | 5205 The Drowned Lantern (the inn) | none; the inn is the evening-gather venue |
| 9102 | Sala the Mender | 5209 The Mending Hut | death-recovery room; players respawn here (Sala need not be awake) |
| 9103 | Ledger Keeper Croup | 5208 Strongbox House (bank) | none |
| 9104 | Trader Onna | 5207 Coulee Provisions (shop) | **CRITICAL** — quest 31 buy step; must be buyable 24/7 |
| 9105 | Granny Wicker | 5210 Wickerwork Cottage | none |
| 9106 | Crier Toke | 5203 Hub Square | delivers ambient nudges; low risk |
| 9107 | Warden Esk | 5215 The Threshold House (the arch) | arch works via room behavior regardless; low risk |

### Schedule design (the onboarding-safety rule)
- **Flavor-only (static, always responsive — NO movement, NO sleeping-away):** Hadwen (9100) and Onna (9104). Every segment uses their home room as `target_room` and `activity: ""` — only the `idlecommands` vary by time of day (dawn / day / evening / night). This keeps the rite and the shop available at any in-game hour.
- **Sleeps at home (no inter-room movement):** Sala (9102) — day/evening idle at 5209, night `activity: sleeping` at 5209.
- **Full movement (day at post → evening gather at the inn 5205 → night sleep at home):** Tally (9101, home IS 5205 so she "gathers" in place + sleeps there), Croup (9103), Wicker (9105), Toke (9106), Esk (9107).
- **Evening gather room = 5205.** Croup/Wicker/Toke/Esk path to 5205 for the evening segment; Tally is already there. That convergence is the conversation venue.

### Schedule schema essentials (docs/schemas/schedule.md)
- Required: `id` (== filename via ConvertForFilename), `description`, `segments` covering all 24 hours with no gaps/overlaps.
- Segment: `start` (0-23 incl), `end` (1-24 excl), `target_room` (must exist), `activity` (`""`|`craft`|`sleeping`|`patrol`), `idlecommands` (NPC-voice say/emote pool).
- **Validator panics** on: bad hours, empty segment, coverage gap/overlap, missing `target_room`, `mapper.GetPath` failure between consecutive segments (incl. the wrap-around night→dawn transition), unresolved `schedule_id`.
- Attach with `schedule_id: <id>` on the mob YAML.

### Relationship edge format (on the mob YAML, e.g. stillwater/333)
```yaml
relationships:
  - to: <mobid>
    type: friend            # friend | rival | employer | employee | family
    subtype: <optional>
```
A conversation requires an edge between the pair (registry). Author **reciprocal** edges (each NPC lists the other).

### Conversation pair-override format (conversations/pairs/<lo>_<hi>.yaml)
```yaml
id: <arbitrary>
mob_a: <lower mobid>
mob_b: <higher mobid>      # mob_a < mob_b required
exchanges:
  - lines:
      - speaker: A          # initiator-role
        text: "..."         # role-agnostic, ~<=80 chars, NPC voice
      - speaker: B          # partner-role
        text: "..."
```
Type pools `friend.yaml` etc. already exist — edges alone produce conversations; pair overrides add bespoke lines. **Role-agnostic** scripts only (engine randomizes who plays A/B).

### Conversation network (the 5 evening gatherers, all friend-type, reciprocal)
- 9101 Tally ↔ 9103 Croup
- 9101 Tally ↔ 9106 Toke
- 9103 Croup ↔ 9105 Wicker
- 9105 Wicker ↔ 9106 Toke
- 9106 Toke ↔ 9107 Esk

(Hadwen/Onna are static-alone and Sala sleeps at her hut, so they have no conversation partners — intentionally excluded.)

### 3 pair-override files (the most characterful pairs; all have edges above)
- `pairs/9101_9106.yaml` — Tally (innkeep) & Toke (crier): the two town gossips.
- `pairs/9103_9105.yaml` — Croup (ledger-keeper) & Wicker (granny): dry old-folk talk.
- `pairs/9106_9107.yaml` — Toke (crier) & Esk (warden): watching the comings and goings.

---

## File Structure
- **Create** 8 schedule files: `_datafiles/world/dogmud/schedules/pothole_coulee/pothole_<npc>.yaml`.
- **Modify** 8 hub mob files: add `schedule_id:` (all 8) and `relationships:` (the 5 gatherers only).
- **Create** 3 conversation pair files: `_datafiles/world/dogmud/conversations/pairs/{9101_9106,9103_9105,9106_9107}.yaml`.

---

## Task 1: Schedules (8 files + schedule_id on 8 mobs)

**Files:**
- Create: `_datafiles/world/dogmud/schedules/pothole_coulee/pothole_{hadwen,tally,sala,croup,onna,wicker,toke,esk}.yaml`
- Modify: the 8 mob files `_datafiles/world/dogmud/mobs/pothole_coulee/910{0..7}-*.yaml` — add one `schedule_id:` line each.

### Schedule id ↔ mob ↔ shape
| schedule id | mob | shape |
|-------------|-----|-------|
| pothole_hadwen | 9100 | STATIC at 5200, activity "" all segments, idle varies by time (no sleep) |
| pothole_onna | 9104 | STATIC at 5207, activity "" all segments, idle varies (no sleep) |
| pothole_sala | 9102 | 5209 day/evening (""), 5209 night (sleeping) |
| pothole_tally | 9101 | 5205 all day (tends bar); night sleeping at 5205 |
| pothole_croup | 9103 | 5208 day (""), 5205 evening gather (""), 5208 night sleeping |
| pothole_wicker | 9105 | 5210 day (""), 5205 evening gather (""), 5210 night sleeping |
| pothole_toke | 9106 | 5203 day (crier, ""), 5205 evening gather (""), 5203 night sleeping |
| pothole_esk | 9107 | 5215 day (""), 5205 evening gather (""), 5215 night sleeping |

- [ ] **Step 1: Write the STATIC exemplar — `pothole_hadwen.yaml`**

Path: `_datafiles/world/dogmud/schedules/pothole_coulee/pothole_hadwen.yaml`

```yaml
id: pothole_hadwen
description: "Cleric Hadwen keeps the Awakening Pool around the clock -- dawn devotions, day tending, evening hush, a night vigil. Never leaves; the rite is always available."
segments:
  - start: 5
    end: 9
    target_room: 5200
    activity: ""
    idlecommands:
      - emote kneels at the water's edge and murmurs the dawn devotion.
      - emote skims a fallen leaf from the still surface.
      - say The pool keeps its own hours. So do I.
  - start: 9
    end: 18
    target_room: 5200
    activity: ""
    idlecommands:
      - emote tends the lanterns ringing the pool with unhurried care.
      - emote watches the water as if reading something in it.
      - say When you are ready to be Opened, come to the water.
  - start: 18
    end: 22
    target_room: 5200
    activity: ""
    idlecommands:
      - emote lowers the wick on the nearest lantern to an evening glow.
      - say The coulee quiets at dusk. The water does not.
  - start: 22
    end: 5
    target_room: 5200
    activity: ""
    idlecommands:
      - emote keeps a slow vigil beside the dark water, eyes half-closed.
      - emote sits motionless, the pool mirroring the night sky.
```

Note: Hadwen's night segment is `activity: ""` (a *vigil*, NOT sleeping) so `ask hadwen begin` works at any hour. Onna's `pothole_onna.yaml` follows the identical static shape at room 5207 with shopkeeper-voice idle (tending the stall by day, "the stall stays open late in a frontier town" at night) — also no sleep, so buying works 24/7.

- [ ] **Step 2: Write the MOVER exemplar — `pothole_toke.yaml`**

Path: `_datafiles/world/dogmud/schedules/pothole_coulee/pothole_toke.yaml`

```yaml
id: pothole_toke
description: "Crier Toke works Hub Square by day, joins the evening crowd at the Drowned Lantern, and sleeps off the square at night."
segments:
  - start: 6
    end: 18
    target_room: 5203
    activity: ""
    idlecommands:
      - emote plants himself in the middle of the square and clears his throat.
      - say Fresh from the coulee -- mind the wash road after dark!
      - emote nods to a passerby and files the face away for later.
  - start: 18
    end: 23
    target_room: 5205
    activity: ""
    idlecommands:
      - emote settles onto a bench at the Lantern with a satisfied grunt.
      - say A crier's throat earns its rest by lamplight.
      - emote trades the day's news with whoever will listen.
  - start: 23
    end: 6
    target_room: 5203
    activity: sleeping
    idlecommands:
      - emote dozes on the bench at the edge of the square, hat over his eyes.
```

Reachability: 5203 → 5205 (evening) → 5203 (night) must each `mapper.GetPath` — they are connected hub rooms; the boot validator confirms. If any path fails, route the mover through an adjacent hub room is NOT possible (schedule has no waypoints) — instead the boot error names the unreachable pair; fix by choosing a reachable `target_room`.

- [ ] **Step 3: Write the remaining 6 schedules**

Author `pothole_onna` (static @5207), `pothole_sala` (5209 day/evening, 5209 night sleeping), `pothole_tally` (5205 all day; 22-6 sleeping @5205), `pothole_croup` (5208 day / 5205 evening / 5208 night sleeping), `pothole_wicker` (5210 day / 5205 evening / 5210 night sleeping), `pothole_esk` (5215 day / 5205 evening / 5215 night sleeping). Each:
- covers all 24 hours, no gaps/overlaps (a typical split: 6-18 day, 18-22/23 evening, 22/23-6 night);
- NPC-voice `idlecommands` (2-4 lines per segment), distinct per NPC, no hard numbers;
- movers gather at 5205 for the evening segment;
- night segment `activity: sleeping` for the 5 movers + Sala; Hadwen + Onna never sleep.

- [ ] **Step 4: Add `schedule_id:` to the 8 mob files**

For each mob 9100-9107, add one top-level line (place it near the other top-level fields like `zone:`):
- 9100 → `schedule_id: pothole_hadwen`
- 9101 → `schedule_id: pothole_tally`
- 9102 → `schedule_id: pothole_sala`
- 9103 → `schedule_id: pothole_croup`
- 9104 → `schedule_id: pothole_onna`
- 9105 → `schedule_id: pothole_wicker`
- 9106 → `schedule_id: pothole_toke`
- 9107 → `schedule_id: pothole_esk`

Read each mob file first; add the line; do not disturb existing fields.

- [ ] **Step 5: Self-review (reading)**

Confirm: every schedule covers 0-23 with no gap/overlap; every `target_room` is a real hub room id from the table; Hadwen + Onna have NO `sleeping` segment; the 5 movers + Sala have exactly one night `sleeping` segment; each `schedule_id` filename matches its `id` via ConvertForFilename; all 16 files are under `.claude\worktrees\feature+newbie-area` (NOT `.clone`/`.claire`). Do NOT run go/git.

---

## Task 2: Conversations (edges on 5 mobs + 3 pair files)

**Files:**
- Modify: mob files 9101, 9103, 9105, 9106, 9107 — add a `relationships:` block to each.
- Create: `_datafiles/world/dogmud/conversations/pairs/9101_9106.yaml`, `9103_9105.yaml`, `9106_9107.yaml`.

- [ ] **Step 1: Add reciprocal relationship edges**

Add a `relationships:` block (top-level, like stillwater/333) to each of these mob files. If a mob already has a `relationships:` block, append to it.

- 9101 (Tally):
```yaml
relationships:
  - to: 9103
    type: friend
  - to: 9106
    type: friend
```
- 9103 (Croup):
```yaml
relationships:
  - to: 9101
    type: friend
  - to: 9105
    type: friend
```
- 9105 (Wicker):
```yaml
relationships:
  - to: 9103
    type: friend
  - to: 9106
    type: friend
```
- 9106 (Toke):
```yaml
relationships:
  - to: 9101
    type: friend
  - to: 9105
    type: friend
  - to: 9107
    type: friend
```
- 9107 (Esk):
```yaml
relationships:
  - to: 9106
    type: friend
```

- [ ] **Step 2: Write pair override `9101_9106.yaml` (Tally & Toke)**

Path: `_datafiles/world/dogmud/conversations/pairs/9101_9106.yaml`

```yaml
id: tally_and_toke
mob_a: 9101
mob_b: 9106
exchanges:
  - lines:
      - speaker: A
        text: "Any news worth pouring a drink over?"
      - speaker: B
        text: "Wash road's quiet for once. I'll take quiet."
      - speaker: A
        text: "Quiet's good for the coulee. Bad for your trade."
  - lines:
      - speaker: A
        text: "Another new face came through the pool today."
      - speaker: B
        text: "Saw them. Green as spring reed. They always are."
      - speaker: A
        text: "We were too, once. Pour 'em a small one on me."
  - lines:
      - speaker: A
        text: "You shout all day. Doesn't your throat give out?"
      - speaker: B
        text: "That's what your back room and your ale are for."
```

- [ ] **Step 3: Write pair overrides `9103_9105.yaml` and `9106_9107.yaml`**

`9103_9105.yaml` (id `croup_and_wicker`, mob_a 9103, mob_b 9105) — dry ledger-keeper + granny exchanges (3 short exchanges, role-agnostic). `9106_9107.yaml` (id `toke_and_esk`, mob_a 9106, mob_b 9107) — crier + warden watching the road (3 short exchanges). Same shape as Step 2; ~<=80 chars/line; role-agnostic (no baked names/gender).

- [ ] **Step 4: Self-review (reading)**

Confirm: every edge is reciprocal (each NPC lists the other); pair files have `mob_a < mob_b`; filenames are `<lower>_<higher>.yaml`; every pair file's pair has a relationship edge (Tally-Toke, Croup-Wicker, Toke-Esk all do); lines are role-agnostic NPC voice. Files under the correct worktree path.

---

## Task 3: Verification gate + commit

(Console work — popups OK'd this session.)

- [ ] **Step 1: Build + nuke instances + boot**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud/.claude/worktrees/feature+newbie-area"
go build -o GoMud_c9p2.exe . && rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
./GoMud_c9p2.exe > _c9p2boot.log 2>&1   # run, watch, then kill
```
Expected, NO panics: schedule loader loads 8 pothole schedules; conversation loader loads the 3 pair files; relationship registry builds the edges; `ValidateZoneConsistency 0/0`; `Server Ready`. The schedule validator panics on coverage gaps / unreachable target rooms / unresolved `schedule_id` — those are the #1 things to watch. The conversation loader warns/panics on a pair file whose relationship edge is missing.

- [ ] **Step 2: Inspect the boot log**

```bash
grep -iE "schedule|conversation|relationship|panic|FATAL|unreachable|GetPath|coverage" _c9p2boot.log | grep -v progression | head -40
```
Expected: schedules + conversations loaded, 0 panic, no "unreachable"/"coverage gap"/"GetPath" errors.

- [ ] **Step 3: Live spot-check (AI port, optional but recommended)**

With the server up, drive the smoketester (admin): `mob schedule <instId>` on a hub NPC to confirm the executor resolves the right segment; visit the inn (5205) in the evening hours to confirm gatherers converge and a conversation fires. If the in-game clock makes "evening" hard to hit, the boot validation (Step 2) is the authoritative gate; note the live conversation check as deferred to the evening playtest.

- [ ] **Step 4: Clean up + commit**

```bash
taskkill //F //IM GoMud_c9p2.exe 2>/dev/null; rm -f GoMud_c9p2.exe _c9p2boot.log
git add _datafiles/world/dogmud/schedules/pothole_coulee/ \
        _datafiles/world/dogmud/mobs/pothole_coulee/9100-*.yaml \
        _datafiles/world/dogmud/mobs/pothole_coulee/9101-*.yaml \
        _datafiles/world/dogmud/mobs/pothole_coulee/9102-*.yaml \
        _datafiles/world/dogmud/mobs/pothole_coulee/9103-*.yaml \
        _datafiles/world/dogmud/mobs/pothole_coulee/9104-*.yaml \
        _datafiles/world/dogmud/mobs/pothole_coulee/9105-*.yaml \
        _datafiles/world/dogmud/mobs/pothole_coulee/9106-*.yaml \
        _datafiles/world/dogmud/mobs/pothole_coulee/9107-*.yaml \
        _datafiles/world/dogmud/conversations/pairs/9101_9106.yaml \
        _datafiles/world/dogmud/conversations/pairs/9103_9105.yaml \
        _datafiles/world/dogmud/conversations/pairs/9106_9107.yaml
git commit -m "content(newbie): hub NPC schedules + NPC-NPC conversations (C9 Phase 2)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```
Exclude the dirty config.yaml + runtime YAMLs (use the explicit paths above only).

---

## Self-review notes (author)

- **Spec coverage:** Implements C9 sub-spec §2a (hub-only schedules: day → evening-gather → night-sleep) and §2b (relationship edges + type pools + 2-3 pair overrides) in full.
- **Onboarding-safety (the spec's explicit guard):** Hadwen (rite) and Onna (shop) are flavor-only/static and never sleep, so quests 30/31 cannot brick at any in-game hour; Sala sleeps only at her own respawn hut; the movers' absence touches no required onboarding step. This narrows the user's "all 8 move" intent for two NPCs — a deliberate, spec-mandated trade-off, flagged for the review gate.
- **No engine work** — schedules/relationships/conversations all pre-exist; the only validation is the load-time validators + boot. Stale instance saves shadow `schedule_id`, so the verification MUST nuke instances first (Instance-Save SOP).
- **Type/name consistency:** schedule `id` == filename (pothole_<npc>) across the file, the table, and the `schedule_id:` on the mob; pair filenames `<lower>_<higher>` match `mob_a`/`mob_b`; relationship edges are reciprocal for all 5 conversation participants.
