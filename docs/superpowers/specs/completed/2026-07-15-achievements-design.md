# Achievements / accolades

**Date:** 2026-07-15
**Status:** Design (approved decisions; pending spec review)
**Roadmap:** `docs/PATH_TO_1.0.md` §3 Retention/stickiness — first of two (guilds is the other).
**Depends on:** the existing `modules/leaderboards/` (for the points board) and rich existing
character state (`KDStats`, stats, skills, gold, `VisitedRooms`, quest tokens, mutations).

---

## 1. Problem / goal

New players need **goals + bragging rights**; the game has leaderboards but no per-player
milestones. Add an achievements system: YAML-authored milestone definitions, unlocked by
evaluating each player's state, shown via an `achievements` command + web page, with an
achievement-points leaderboard.

---

## 2. Approved design decisions

1. **YAML-authored definitions** with a **fixed trigger-type vocabulary** (a handler per
   type; new achievements of an existing type = pure YAML).
2. **Poll-based evaluation** (mirrors the leaderboards `NewRound` pass) — every trigger is
   derivable from current character state, so no per-event wiring or new counters.
3. **Per-character storage** as a new `Character.Achievements map[string]uint64` (id → round
   unlocked).
4. **Private unlock announcement** only (no global broadcast). **ANSI-styled text, no emoji**
   (the client has an ASCII-conversion gap for glyphs — feel-test 2026-07-13).
5. **Author a ~15–25 starter set** across five categories in this build.
6. **Achievement-points leaderboard** added to the existing `modules/leaderboards/`.

---

## 3. Architecture

Split content/logic (internal, testable, boot-validated) from runtime glue (a thin module):

- **`internal/achievements/`** — the `Definition`/`Trigger` types, the **loader**
  (`LoadDataFiles()`, world data, validate + **panic** on bad defs at boot — matches the
  quests/mobs loader convention), the registry (`All()`, `Get(id)`), the pure evaluator
  `Evaluate(trigger, *characters.Character) bool`, and points helpers.
- **`modules/achievements/`** — a plugin (like leaderboards): the `NewRound` **poll**, the
  `achievements` **command**, the **web page**, and the unlock+announce+save glue.
- **`modules/leaderboards/`** — add an achievement-**points** board.
- **`_datafiles/world/dogmud/achievements/*.yaml`** — the authored definitions.
- **`internal/characters/character.go`** — the `Achievements` field + `HasAchievement` /
  `GrantAchievement` helpers.

Boot: wire `achievements.LoadDataFiles()` into the central data-load sequence alongside
`quests.LoadDataFiles()` (so a bad def panics at startup, caught by the pre-push boot test).

---

## 4. Definition schema — `_datafiles/world/dogmud/achievements/<id>.yaml`

```yaml
id: first-blood                 # unique kebab id; also the filename base
name: "First Blood"             # display name
description: "Defeat your first foe."
category: combat                # combat | exploration | wealth | progression | quests
points: 5                       # leaderboard/points weight
trigger:
  type: mob_kills               # fixed vocabulary (below)
  threshold: 1                  # for count/value types
  # stat: strength              # for stat_reached
  # skill: rhetoric             # for skill_reached
  # token: "10-end"             # for quest_completed
```

### Trigger vocabulary (all state-derivable → poll evaluates them)

| `type`               | Params      | Unlocks when (evaluated against the live `Character`)            |
|----------------------|-------------|-----------------------------------------------------------------|
| `mob_kills`          | threshold   | `KD.TotalKills >= threshold`                                     |
| `pvp_kills`          | threshold   | `KD.TotalPvpKills >= threshold`                                  |
| `deaths`             | threshold   | `KD.TotalDeaths >= threshold` (survivor accolades)              |
| `stat_reached`       | stat, threshold | the named primary stat's adjusted value `>= threshold`; `stat: any` unlocks when the player's HIGHEST primary stat clears it |
| `skill_reached`      | skill, threshold | `GetSkillLevel(skill) >= threshold`; `skill: any` unlocks when the player's HIGHEST skill clears it |
| `gold_total`         | threshold   | `Gold + Bank >= threshold`                                      |
| `rooms_explored`     | threshold   | total visited rooms across zones `>= threshold`                |
| `quest_completed`    | token       | `HasQuest(token)` (a specific quest's end token)               |
| `quests_completed`   | threshold   | count of completed quests (`-end` tokens) `>= threshold`        |
| `mutation_count`     | threshold   | `len(Mutations) >= threshold`                                  |
| `item_rarity`        | threshold   | owns any **equipment** item (weapon/armor/wearable; NOT components/materials/potions) with `spec.RarityTier >= threshold`, scanning backpack + equipped + bank storage. Captures "acquired a pinnacle item" — the legendary BIS gear sits at rarity 82–90; the equipment filter excludes the high-rarity crafting reagents so a raw component doesn't count. |
| `achievement_points` | threshold   | the player's current total earned points `>= threshold` (meta/tiered) |

**Loader validation (panic at boot):** unknown `type`; duplicate `id`; missing required
param for the type (e.g. `stat_reached` without `stat`, `quest_completed` without `token`);
unknown `stat`/`skill` name (the literal `any` is allowed); `category` not in the allowed
set; `points < 0`. Filename base must equal `id`.

---

## 5. Evaluation + unlock (the module poll)

`Evaluate(trigger, *Character) bool` is a pure switch over `type` — the testable core.

The module's `NewRound` handler (gated to a modest interval like the leaderboard `Update()`
cadence; config `AchievementPollRounds`, default e.g. 10) does:

```
for each ONLINE, non-admin, non-AI user:
    earnedPoints = sum(points of the user's already-unlocked achievements)
    for each definition NOT yet in user.Character.Achievements:
        if Evaluate(def.Trigger, user.Character):   // achievement_points uses earnedPoints
            user.Character.GrantAchievement(def.Id, currentRound)
            announce privately (ANSI, no emoji):
              <ansi fg="yellow-bold">*** Achievement unlocked: NAME ***</ansi>
              <ansi fg="white">DESCRIPTION  (+POINTS points)</ansi>
        (save the user once if anything was granted)
```

- **`achievement_points`** meta triggers evaluate against `earnedPoints` computed at the
  start of the pass, so a points-tier accolade unlocks on the poll *after* the one that
  pushed the player over the line (a one-interval delay — acceptable).
- Admin/AI exclusion mirrors the leaderboard's filter.
- Poll only touches online players; an offline player's new achievements evaluate on their
  next poll after login (fine).

---

## 6. Storage + Character helpers — `internal/characters/character.go`

```go
	Achievements map[string]uint64 `yaml:"achievements,omitempty"` // achievement id -> round unlocked
```

Helpers: `HasAchievement(id string) bool`, `GrantAchievement(id string, round uint64)`
(lazy-inits the map). Points are computed by the `achievements` package (needs the defs), not
on the Character.

---

## 7. `achievements` command — `internal/usercommands/achievements.go`

`achievements` (no arg) shows the caller's accolades grouped by category:
- **Earned**: name + points (and optionally "when"); a running total (count + points).
- **Locked**: name + a one-line progress hint derived from the trigger
  (e.g. `mob_kills 42/100`, `stat_reached strength 138/150`, `quest_completed — not yet`).

`achievements <category>` filters to one category. All player-facing; **no raw internal
values beyond the achievement's own thresholds/progress** (thresholds are the point of an
achievement, so showing "42/100" is intended, like the status sheet exception).

Registered `` `achievements`: {Achievements, true, true, false} `` (read-only, always allowed).

---

## 8. Web page + leaderboard tie-in

- **Web page** (`modules/achievements/`): mirror the leaderboards module's
  `plug.Web.WebPage(...)` + `webAchievementData` — a page listing all achievements by
  category with earned/locked state for the viewing user (or a global showcase). Client HTML
  `achievements.html`. Scope to parity with the leaderboards page; details in the plan.
- **Leaderboard**: add `LB_Achievements` (value = a character's total achievement points) to
  `modules/leaderboards/` — the `Update()` pass already iterates users; compute points via
  `internal/achievements`. Appears alongside the Power board.

---

## 9. Starter content (~15–25, this build)

A rounded set so new players have goals immediately. Sketch (final names/thresholds tuned):

- **Combat:** First Blood (`mob_kills` 1), Cutting Teeth (`mob_kills` 100), Veteran
  (`mob_kills` 1000), Duelist (`pvp_kills` 10), Hard to Kill (`deaths` 10).
- **Exploration:** Wanderer (`rooms_explored` 25), Pathfinder (100), Cartographer (250).
- **Wealth:** Coin Purse (`gold_total` 1000), Well-Off (10000), Magnate (100000).
- **Progression:** Honed (`stat_reached` any 130), Formidable (`stat_reached` any 160),
  Skilled (`skill_reached` any 25), Twice-Touched (`mutation_count` 2), Chimeric
  (`mutation_count` 5), **The Pinnacle (`item_rarity` 82 — acquire a legendary/pinnacle
  piece of gear; high points, ~40, it's a big deal)**.
- **Quests:** Errand Runner (`quests_completed` 1), Adventurer (`quests_completed` 5),
  Hero of the Realm (`quests_completed` 15).
- **Meta:** Decorated (`achievement_points` 50), Legend (`achievement_points` 150).

(Provisional thresholds/points — tuned in playtest; the point is a coherent ladder.)

---

## 10. Testing

- **Evaluator (the core):** table-test `Evaluate` for every trigger type — met vs unmet vs
  boundary (== threshold unlocks), with a constructed `Character` (KD, stats, skills, gold,
  VisitedRooms, quest tokens, mutations). `achievement_points` tested against a supplied
  earned-points value.
- **Loader validation:** a bad def per rule (unknown type, missing param, bad stat, dup id,
  bad category) → the loader/validator reports/panics; a good set loads.
- **Character helpers:** `HasAchievement` / `GrantAchievement` (lazy-init, idempotent).
- **Points helper:** total points over a set of earned ids.
- **Poll grant path:** given a user whose state meets N locked achievements, one poll grants
  exactly those, sets the map, and doesn't re-grant on a second poll.
- Full suite green + **boot clean** (the starter YAML loads without panic — the loader
  validates every rule).

---

## 11. Out of scope / deferred

- **Event-driven instant unlock** (poll is v1; event hooks for immediate "unlocked!" feedback
  later).
- **Global/broadcast announcements** (private only in v1).
- **Hidden/secret achievements, tiers/chains, rewards** (gold/item on unlock) — future.
- **`/new-achievement` content command** — nice follow-up now that the schema exists.
- **Guilds** (the other §3 item — separate arc).

---

## 12. Files touched

- `internal/achievements/achievements.go` (+ `loader.go`, `evaluate.go`) — types, loader,
  registry, `Evaluate`, points helpers.
- `internal/achievements/*_test.go` — evaluator + loader tests.
- `internal/characters/character.go` — `Achievements` field + helpers (+ test).
- `internal/usercommands/achievements.go` (+ help template + registration) — the command.
- `modules/achievements/achievements.go` — plugin: poll, web page, unlock glue.
- `modules/leaderboards/leaderboards.go` — the `LB_Achievements` points board.
- boot data-load wiring — `achievements.LoadDataFiles()`.
- `_datafiles/world/dogmud/achievements/*.yaml` — starter set.
- `_datafiles/html/.../achievements.html` — web page (parity with leaderboards.html).
- `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md` (mark achievements done).
