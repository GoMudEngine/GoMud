# Newbie Rework — Chunk 9: Polish (Pothole Coulee)

**Status:** APPROVED 2026-06-16 (design). Sub-spec of the parent
`2026-05-27-newbie-area-rework-design.md` (§ chunk table row 9).

**Worktree/branch:** `worktree-feature+newbie-area` (based on master
`9e1a0d00`). Same branch as all prior chunks. **NOT pushed** — merges
to master only at the C10 cutover. No new rooms in C9.

**Predecessors:** Spokes A–G (chunks 2–8) all COMPLETE + committed on
this branch. C9 is the second-to-last chunk; C10 is the cutover.

---

## 0. Scope & decisions (locked 2026-06-16)

The parent spec scopes chunk 9 as: *cross-spoke balance pass on rewards,
repeatable-quest tuning, hint-coverage audit, in-character no-numbers
audit, schedule integration for hub/outlying NPCs, NPC↔NPC conversations
seeded.* User chose **Full C9 including repeatables**. Decisions made
during brainstorming:

1. **Repeatable quests = native engine feature.** The `Quest` struct has
   NO repeatable support today (`internal/quests/quests.go:58`); once a
   player earns a quest's `-end` token it is done forever. Rather than a
   data-only bounty/dialogue loop, **build native repeatable-quest
   support in the engine** — consistent with the established
   reusable-feature pattern (`train_stat` / `recipe_info` / `item_info`).
2. **Schedules = hub-only day/night rhythm.** Schedule just the 8 hub
   NPCs (9100–9107). Spoke trainers stay put (they're quest anchors out
   in the wilds/mine/marsh; moving them risks "where did the questgiver
   go").
3. **Conversations = edges + type pools + a few pair overrides.** Add
   `relationships:` edges among the 8 hub NPCs, reuse the existing
   generic type pools, author 2–3 pair-override files for the most
   characterful pairs.
4. **Reward-balance sweep = consistency only.** Normalize gold/skill/stat
   reward outliers so they scale sensibly spoke-to-spoke. **Encounter
   difficulty stays deferred** to the post-build evening naive playtest
   (`feedback-defer-tuning-to-post-build-playtest`). C9 does NOT tune mob
   statpools.
5. **Repeatables offered off the existing capstone trainers** (no new
   NPCs): Garve (A), Ovell (B), Falv (C), Delk (D), Orrin (E), Sere (F),
   Bryn (G).

### ID allocations
- **Repeatable quests: 53–59** (one per spoke A→G). Quests currently top
  out at 52; verify with `python tools/id_inventory.py --type quests`
  before authoring (C4 lesson: run id_inventory first).
- **No new rooms, mobs, or items.** Repeatable quest "kill/forage/craft"
  targets reuse each spoke's existing foes/materials/recipes.
- **Schedules:** new files under
  `_datafiles/world/dogmud/schedules/pothole_coulee/<mobid>.yaml` for
  9100–9107.
- **Conversations:** new `relationships:` blocks on the 9100–9107 mob
  files + 2–3 new files under
  `_datafiles/world/dogmud/conversations/pairs/<lo>_<hi>.yaml`.
- **Dialogue:** repeatable offer-nodes added to EXISTING trainer dialogue
  files (no new dialogue files). Dialogue files are bare `<mobid>.yaml`
  (C8 lesson — NOT mobid-slug).

---

## 1. Phase 1 — Engine: repeatable quests + 7 repeatables

### 1a. Engine feature (reusable, world-wide)

Add to the `Quest` struct (`internal/quests/quests.go`):
- `Repeatable bool` — yaml key `repeatable` (tag-less → no-underscore;
  the loader is yaml.v2, see the QuestReward gotcha comment at
  quests.go:31).
- `CooldownRounds int` — yaml key `cooldownrounds` (or a properly tagged
  `yaml:"cooldown_rounds"` — pick the tagged form so it's snake_case and
  unambiguous; confirm against the loader at planning time).

**Completion behavior** (in the quest-update hook,
`internal/hooks/Quest_HandleQuestUpdate.go`, where `-end` is processed
and rewards are granted):
1. Grant rewards exactly as today (gold/item/skill/stat/recipe/item_info).
2. If `Repeatable`, after granting, **clear the quest's `QuestProgress`
   entry** so `HasQuest`/`IsQuestDone` go false and the quest can be
   re-taken. Use a new `Character.ClearQuestProgress(questId int)` method
   (mirrors the existing `ClearQuestFlag`); `QuestProgress` is
   `map[int]string` keyed by quest id (character.go:253) so this is a
   `delete`.
3. **Cooldown:** record the completion round and gate re-grant so the
   player cannot immediately re-take. Store the "available again at round
   N" timestamp (a quest flag or a MiscData key, e.g.
   `quest-<id>-cooldown-until`); the grant path (dialogue `grantsQuest`
   and/or quest `room_enter`/`command` triggers) refuses to re-start
   until the current round ≥ that value. Confirm the cleanest storage at
   planning time — prefer a mechanism that does NOT require new dialogue
   gating fields if one exists.

**Tests:** unit tests in the quests/hooks packages: (a) a repeatable
quest clears progress on `-end`; (b) rewards still grant on each
completion; (c) re-grant is refused inside the cooldown window and
allowed after it. Mirror `Quest_ItemGrants_test.go`.

**No-regression guard:** non-repeatable quests (all existing 1–52) MUST
behave identically — `Repeatable` defaults false, no progress clear.

### 1b. The 7 repeatable quests (53–59)

Each is a short single-or-two-step "practice loop" anchored on the
spoke's existing content. Reward = **modest gold + the use-based skill
progression the activity itself yields** (skill grows via `OnSkillUse`
just by doing it; the quest gold is deliberately small so the cooldown +
tiny payout prevent a gold farm). No new items/mobs.

| ID | Spoke | Trainer | Loop (reuses existing content) | Skill exercised |
|----|-------|---------|-------------------------------|-----------------|
| 53 | A Martial | Garve (9112) | Cull bandits in the wash (kill N of 9110/9111) | weapon/unarmed-combat |
| 54 | B Forge | Ovell (9117) | Forge an iron dagger at the forge (craft) | blacksmithing |
| 55 | C Alchemy | Falv (9129) | Brew & deliver a healing salve (craft + drink/deliver) | alchemy |
| 56 | D Wilderness | Delk (9137) | Bring back game (kill hare 9138 / pronghorn 9139 + forage) | search |
| 57 | E Folding | Orrin (9145) | Clear a grove foe by casting (kill 9147/9148 via cast) | spellcasting |
| 58 | F Lore | Sere (9160) | Recite/search the standing stones (search/help/social beat) | rhetoric |
| 59 | G Ranged | Bryn (9162) | Shoot down N canyon targets (shoot at the butt/foes) | ranged-combat |

Exact step counts and gold values to be fixed in the implementation plan;
keep gold per-completion in the single-digit-to-low-teens range (compare
to the cert quests' 15–50g one-time rewards — repeatables pay less).

**SOPs (all repeatable quests):**
- `repeatable: true` + a sensible `cooldownrounds`.
- Quest-granting dialogue node carries `quest` + `task` triggers (Quest
  NPC Dialogue SOP).
- `questExcluded` must NOT block the quest's own re-grant — because the
  engine clears progress on completion, the standard end-token exclusion
  pattern interacts with repeatables; verify the offer node re-fires
  after cooldown (this is the key thing the Phase-1 walkthrough must
  confirm).
- Reward block uses **no-underscore** keys (itemid/skillinfo/gold/
  playermessage) — the yaml.v2 gotcha (`reference-quest-reward-yaml-key-gotcha`).

---

## 2. Phase 2 — Living hub NPCs: schedules + conversations

### 2a. Schedules (hub-only, 9100–9107)

One schedule file per hub NPC at
`_datafiles/world/dogmud/schedules/pothole_coulee/<mobid>.yaml`, covering
all 24 hours (validators panic on gaps). Rhythm:
- **Day:** at-post / tend-stall / work — idle command pool fitting the
  role; `activity` set so `TickMobCraft` behaves sensibly per role.
- **Evening:** gather at a common room (the inn / Tally's, the natural
  hearth room) — this is where overheard conversations fire.
- **Night:** `activity: sleeping` in their home/quarters — visible sleep
  the curious player can observe (and wake). `ScheduleWakeGraceRounds`
  prevents immediate re-sleep after a wake.

Movement uses the existing `pathto` plumbing; every target room must be
reachable from the NPC's placement (validator-checked). Keep routines
short (3–4 segments) — texture, not a simulation. **Do NOT schedule any
quest-anchor whose absence would brick a hub quest step** — verify each
hub NPC's role against quests 30/31 before assigning movement; an NPC the
newbie must `ask` during onboarding should stay reachable or only move in
the evening/night after onboarding is realistically done.

### 2b. Conversations (edges + type pools + 2–3 overrides)

1. Add `relationships:` edges to the 9100–9107 mob files. Plausible
   edges (finalize at planning, keep role-agnostic per the conversation
   script semantics):
   - Innkeep Tally (9101) ↔ Cleric Hadwen (9100): friend.
   - A merchant ↔ a porter/helper: employer/employee.
   - Two trade NPCs of similar standing: friend or rival.
2. Rely on the EXISTING type pools (`conversations/types/`:
   employee/employer/family/friend/rival) for generic exchanges.
3. Author **2–3 pair-override files**
   (`conversations/pairs/<lo>_<hi>.yaml`, ids ascending) for the most
   characterful hub pairs. Role-agnostic scripts (engine randomizes which
   physical NPC plays "A").

Conversations only fire when both NPCs are fully idle (no combat/sleep/
patrol/cooldown) — the evening-gather segment is the window. Tune nothing
beyond seeding; the global chance knobs already exist.

---

## 3. Phase 3 — Audits + reward-balance sweep

### 3a. Reward-balance consistency sweep (quests 30–59)
Tabulate every reward (gold, skillinfo, stat_info, recipe_info,
item_info) across quests 30–59 and check **consistency**, not difficulty:
- Cert-quest gold scales sensibly inner→capstone within a spoke and
  across spokes (no spoke pays 5× another for equivalent effort).
- Capstone stat grants are uniform (the spokes grant +3 of a stat — keep
  parity).
- Repeatables all pay in the same small band.
Normalize outliers. **No mob statpool / encounter changes** — difficulty
is the evening playtest's job.

### 3b. Hint-coverage audit
Every triggerable thing is discoverable: every quest step has a `Hint`;
every `ask <npc> <keyword>` keyword appears in a hint / NPC text / room
desc / quest log; every interactable room noun is in its room body
(the manifest checker's noun-token rule already enforces room nouns —
extend the audit to dialogue keywords and quest-step hints). Fix any
undiscoverable trigger.

### 3c. No-hard-numbers audit
Sweep all C9-authored player-facing strings (repeatable quest text/hints,
schedule-driven emotes if any, conversation lines) for raw numbers; use
descriptive language (Player-Facing Messages SOP). The `status` stat
sheet is the only sanctioned numeric display.

---

## 4. Acceptance criteria (per phase, before its review gate)

Standard gate each phase:
- `go build ./...` exit 0; relevant `go test` packages pass (Phase 1:
  quests/hooks/characters).
- **Boot clean** past data-file loading: watch loadedCount deltas
  (quests 44→51 after Phase 1), **schedule validators** pass (Phase 2 —
  no coverage gaps / unreachable rooms / unresolved `schedule_id`),
  `ValidateAllFlags` ok, `ValidateZoneConsistency` errors=0 warnings=0
  (panic mode), Server Ready, 0 panics.
- `newbie_manifest_check.py` 0 FAIL; `coord_inventory.py` 0 collisions
  (unchanged — no new rooms).
- Scripted AI-port walkthrough where it adds signal: Phase 1 MUST confirm
  a repeatable quest completes, grants reward, clears progress, and
  **re-offers after the cooldown** (the novel behavior). Phase 2 SHOULD
  confirm a hub NPC sleeps at night and an overheard conversation fires.
- User review gate after each phase. Commit per-phase (exclude the dirty
  `config.yaml` dev settings + runtime YAMLs, as in C2–C8).

---

## 5. Known gotchas / risks (carried from prior chunks)

- **yaml.v2 reward-key gotcha** — repeatable quest reward blocks use
  no-underscore keys; the new `repeatable`/`cooldown_rounds` fields must
  match whatever tagging the loader expects (verify at planning).
- **Dialogue files are bare `<mobid>.yaml`** (C8 bug) — repeatable
  offer-nodes go INTO the existing trainer dialogue files; don't create
  slug-named ones.
- **Schedule validator panics** are load-time — the boot test is the only
  reliable check (CLAUDE.md schedule note).
- **Quest-anchor displacement** — scheduling a hub NPC that a hub quest
  needs reachable could brick onboarding; audit roles first.
- **Repeatable re-grant vs `questExcluded`** — the end-token exclusion
  pattern that prevents re-offer of normal quests must NOT permanently
  block a repeatable; the Phase-1 walkthrough is the gate for this.
- **No console popups** (`feedback-no-console-popups-windows`) — builds /
  boots / scripted walkthroughs run only in a user-OK'd window or by the
  user; safe windowless tools otherwise.
- **Phantom-path leak** — subagents have repeatedly written content to
  stray `.clone/` / `.claire/` trees; verify files land in the real
  `.claude` worktree after any dispatch.

---

## 6. After C9
C10 cutover (separate sub-spec): merge to master, corridor/connection
rooms to the wider world, `StartRoom` + `DeathRecoveryRoom` config,
sethome default re-point + **30-end migration for existing characters**
(veterans entering 5200 otherwise hit the movement gate), DELETE Sanctum
Basin, PATCH_NOTES. Plus the standing **PROD TODO: cherry-pick `697169bf`**
(the handleMobAIDecision nil-Aggro crash fix) to master independently.
