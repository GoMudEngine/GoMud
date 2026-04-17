# Code Cleanup Stage 1 — Overview

Post-launch code quality pass. Modeled on Phase 37 from
`DEVELOPMENT_PLAN.md`. No behavior changes in most substages —
pure structural improvement, dead code removal, error handling
hardening, and targeted performance wins.

Each substage gets its own spec → plan → implementation cycle.

## Substages

| # | Stage | Effort | Risk | Status |
|---|-------|--------|------|--------|
| 1.1 | Behavior Tree Package Split | 3h | Low | Complete |
| 1.2a | God-Function Refactor — Combat + Spell | 5h | Medium | Not started |
| 1.2b | God-Function Refactor — Character + Admin | 5h | Low-Med | Complete |
| 1.2c | `character.go` File Split | 3h | Low | Not started |
| 1.3 | Config File Split | 2h | Low | Complete |
| 1.4 | Dead Code Sweep (Go code) | 3h | Low | Complete |
| 1.4b | Help Template Audit | 3h | Low | Complete |
| 1.5 | Error Handling Audit | 4h | Medium | Not started |
| 1.6 | Test Coverage for New Systems | 5h | Low | Not started |
| 1.7 | Performance Pass | 10h | Mixed | Complete |
| 1.8 | Behavior Tree Engine Robustness | 4h | Low | Not started |

**Total: ~47 hours**

**Recommended execution order:** 1.1 → 1.3 → 1.4 → 1.4b → 1.7 → **1.2b
→ 1.2c → 1.2a** → 1.5 → 1.8 → 1.6

---

## Stage 1.1: Behavior Tree Package Split

Split `internal/behaviortree/actions.go` (~29KB, 26+ actions) and
`conditions.go` (~10KB, 20 conditions) into themed files. Keep a thin
`registry.go` with the `init()` function that wires everything.

**Proposed split:**

`actions.go` →
- `actions_combat.go` — attack, flee, cast, add_buff, remove_buff
- `actions_dialogue.go` — say, respond, emote, send_user_text,
  send_room_text, mob_say, mob_emote
- `actions_room.go` — set_room_locked, spawn_item_in_room,
  add_temp_exit, move_player, intercept
- `actions_quest.go` — grant_quest, set_quest_flag, grant_mutation,
  give_gold, give_item, return_item, take_item, take_gold,
  give_item_multiple, set_misc_data
- `actions_mob.go` — spawn_mob, summon_companion, command, command_mob,
  move, open_instance_portal
- `actions_state.go` — set_state, increment_state, decrement_state,
  create_instance

`conditions.go` →
- `conditions_mob.go` — mob_in_combat, mob_health_below, mob_at_home,
  mob_has_buff, mob_in_room
- `conditions_player.go` — player_has_quest, player_missing_quest,
  player_has_item, player_has_gold, player_has_flag, player_has_spell,
  player_has_misc_data, players_in_room, multiple_enemies
- `conditions_room.go` — command_matches, command_rest_contains
- `conditions_state.go` — state_equals, state_greater_than,
  keyword_match, item_matches, time_of_day, round_mod, random_chance

**Keep:**
- `registry.go` — the `init()` function wiring everything. All
  registry maps (`actionRegistry`, `conditionRegistry`) live here.
- `types.go` — ActionFunc, ConditionFunc type definitions,
  ActionNode/ConditionNode, Evaluate methods
- `engine.go`, `helpers.go`, `loader.go`, `state.go`, `decorators.go`,
  `room_state.go` — unchanged

**Constraints:**
- Zero behavior change
- All exported names unchanged
- All tests continue to pass
- Each file has one clear theme
- `delayedActions` map stays in `actions_combat.go` or `types.go`
  (wherever it's cleanest)

## Stage 1.2: God-Function Refactor

Eight functions >200 lines after Phase 37. Break them up using the
same patterns as 37.1a/b/c — extract helpers, reduce nesting, make
each function do one thing.

Split into two substages by domain so each has a focused review surface:

### 1.2a — Combat + Spell (3 functions, ~5h, Medium risk)

Touches the hottest, most test-sensitive code paths. Scheduled after
1.2b and 1.2c ship so combat has time to settle before another
refactor.

1. `handlePlayerVsMob()` — `NewRound_DoCombat_helpers.go` (286 lines)
   — combat phase extraction
2. `handleMobVsPlayer()` — `NewRound_DoCombat_helpers.go` (236 lines)
   — combat phase extraction
3. `applyMobEffect()` — `spell_resolution.go` (246 lines) —
   channel-specific helpers

### 1.2b — Character + Admin (5 functions, ~5h, Low-Medium risk)

Function-level refactor in existing files. Characterization tests
written BEFORE refactoring the character functions (first tests ever
for these). Admin functions verified via manual smoke.

1. `Character.RecalculateStats()` — `character.go` (239 lines) — loop
   over stats array instead of 6 repeated blocks
2. `Character.Wear()` — `character.go` (~235 lines) — slot selection
   helpers
3. `Character.Validate()` — `character.go` (433 lines) — subsystem
   validators (skills, spells, equipment, stats)
4. `Room()` — `admin.room.go` (387 lines) — subcommand dispatcher
   (new file `admin.room.dispatcher.go`)
5. `room_Edit_Exits()` — `admin.room.go` (257 lines) — prompt state
   machine split (new file `admin.room.exits.go`)

### 1.2c — `character.go` File Split (~3h, Low risk)

`character.go` is ~3925 lines with many unrelated concerns
(init, description, keys, spells, migrations, charm, combat dice,
mitigation, etc.). Split into themed files following the pattern
already established in the package (`progression.go`, `cooldowns.go`,
`hand_slots.go`, `charminfo.go`, etc.). Proposed files TBD in spec.

Runs after 1.2b so that the refactored `RecalculateStats`, `Validate`,
and `Wear` + their new helpers can be moved as cohesive units into
their themed file(s).

**Constraints (apply to all three substages):**
- Zero behavior change
- `go build`, `go vet`, `go test` clean after each function
- Manual smoke test: combat, admin room edit, spell effects
- Each extracted helper should be <80 lines
- Clear bugs/vulnerabilities discovered during refactoring may be
  fixed inline with a clearly-scoped commit and comment; ambiguous
  cases pause for user review

## Stage 1.3: Config File Split

`config.balance.go` is ~56KB. Split by domain into smaller files,
all sharing the same `Balance` struct.

**Split:**
- `config.balance.combat.go` — mitigation caps, rolls, cooldowns,
  dodge/parry/block, crit, stamina costs
- `config.balance.progression.go` — skill/stat curves, use counts,
  progression multipliers, mutation rates
- `config.balance.spells.go` — fold calc, spell difficulty, spell
  damage scale, conviction costs
- `config.balance.mobs.go` — mob AI, mob regen, mob progression,
  mob stat cap, reaction delays
- `config.balance.shops.go` — shop pricing, restock, barter,
  crafter settings
- `config.balance.misc.go` — regen rates, toxicity, carry capacity,
  salvage, resource penalties
- `config.balance.go` — keeps the `Balance` struct definition and
  `Validate()`/`GetBalanceConfig()` accessor

**Constraints:**
- Zero behavior change
- All field names, YAML tags, and exported methods unchanged
- Move helper functions (like `GetStatProgressionMultiplier`) into
  the same file as their related fields

## Stage 1.4: Dead Code Sweep (Go code)

Post-JS-removal audit of Go code. What's unused now that the
scripting bridge is gone? This stage focuses on `.go` files only;
orphaned help templates and other data-file dead weight are handled
in Stage 1.4b.

**Investigate:**
- `util.Hash` — only used by the SHA256→bcrypt migration path. Once
  all users migrate, can we remove it? (Keep for now — migration
  not complete)
- Mob/room methods that were only called from JS bridge
- `_datafiles/world/empty/` — unused? Check `main.go` and config
- `_datafiles/world/default/` — only used by `ValidateWorldFiles()`
  as a folder-structure template. Can we delete the file contents
  and keep only empty directories?
- Orphaned help templates (old JS features)
- Any remaining `.bak` files anywhere
- Unused exports in `internal/mobs/`, `internal/rooms/`,
  `internal/spells/`, `internal/buffs/`

**Constraints:**
- Verify zero callers before removing anything (grep + test run)
- Delete files that have no references
- Document decisions in the spec's "KEPT" section

## Stage 1.4b: Help Template Audit

Audit `_datafiles/world/dogmud/templates/help/` and related
template directories for orphaned help files — templates that
reference commands, concepts, or systems that no longer exist.

**Process:**
- List every `.template` file in the help directory (~100 files)
- For each template, grep the codebase for references:
  - Is the filename referenced directly anywhere?
  - Is there a command that calls `templates.Process()` with this
    name?
  - Is there a keyword mapping in `internal/keywords/` that routes
    to this template?
- Flag templates with no references as candidates for removal
- Verify no hidden dynamic references (e.g., `{keyword}.template`
  patterns where the keyword is user input)

**Target categories:**
- Help files for removed commands (e.g., admin mob/spell create
  from Phase 5)
- Help files for JS-era concepts
- Duplicate / superseded help files
- Old `set <subcommand>` help files that no longer exist

**Constraints:**
- High caution — a missing help template is a player-visible gap
- Prefer to update stale help content over deleting it
- If a template could plausibly still be useful (e.g., a feature
  planned for later), leave it alone and document

## Stage 1.5: Error Handling Audit

Phase 37.3a/b predates ~500 commits of new code. Audit:

**Targets:**
- All 30+ behavior tree action functions — nil-check mob/user/room
  before dereferencing
- Room behavior tree entry points (TryRoomBehavior, EnsureRoomBTreeState)
- New Go spell hooks: fold-anchor, fold-recall, purge-affliction,
  meditating buff handler
- Sable portal vendor action (`open_instance_portal` — has refund
  paths, need to verify all error branches refund gold)
- Dashboard code (`admin.progression.go`, `buildPlayerOverview`)
- File logging setup (panics on directory creation)
- New hooks wired in Phase 4b/4c

**Pattern to look for:**
- `mobs.GetInstance()` without nil check
- `rooms.LoadRoom()` without nil check
- `users.GetByUserId()` without nil check
- Ignored error returns (`_, _ =` or missing error handling)
- Type assertions without `ok` check
- Goroutines without `defer recover()`

## Stage 1.6: Test Coverage for New Systems

**Add tests for:**
- Room behavior tree engine — `TryRoomBehavior`, state persistence,
  command interception via `ctx.Intercepted`
- New Phase 4c conditions: `command_matches`, `command_rest_contains`,
  `mob_in_room`
- New Phase 4c actions: `mob_say`, `mob_emote`, `grant_mutation`,
  `give_gold`, `send_user_text`, `send_room_text`, `intercept`,
  `remove_buff`, `move_player`
- Quest engine `item_give` trigger vs behavior tree `player_give`
  handler interaction (regression test for recent bugfix)
- Bcrypt migration path — SHA256 → bcrypt on login
- File logging config — LogToFile on/off, env var precedence
- `actSummonCompanion` with `hostile: "true"` — verify aggro + engage

**Constraints:**
- Additive only (no existing tests modified)
- Test files follow existing naming conventions
- Use existing test helpers where possible

## Stage 1.7: Performance Pass

Four targeted optimizations. See performance chart in brainstorming
notes for impact/risk analysis.

**Targets:**

1. **Combat round active-zones-only** (Very High impact)
   - Replace `mobs.GetAllMobInstanceIds()` in `handleMobCombat` with
     iteration only over mobs in rooms containing players
   - Background loop (slower cadence, e.g. every 5s) handles
     empty-zone mobs for regen/wandering
   - Risk: mobs in empty zones stop AI/regen if background loop bug
   - **4-6h, Medium risk**

2. **Global registry locks** (Medium impact)
   - Add `sync.RWMutex` to `mobs.mobInstances`, `users.userInstances`
   - Wrap all reads with RLock, writes with Lock
   - Prevents race conditions from Discord webhook, HTTP admin,
     file logger goroutines
   - **2-3h, Low risk**

3. **Room.GetMobs per-tick cache** (High impact)
   - Cache filtered mob lists per room per combat round
   - Invalidate on AddMob/RemoveMob
   - Behavior tree + combat frequently call GetMobs(FindAll) multiple
     times per event
   - **2-3h, Medium risk**

4. **PruneVisitors empty-map fast path** (Very Low impact)
   - Early return if `len(r.visitors) == 0`
   - **15min, Very Low risk**

**Constraints:**
- No behavior changes — same mobs process, same events fire
- Add benchmarks where practical to validate wins
- Manual smoke test: combat in populated zone, combat in empty
  wilderness zone, admin dashboard under load

## Stage 1.8: Behavior Tree Engine Robustness

Targeted audit of the btree engine after heavy use in 4b/4c.

**Investigate:**
- Delayed action queue: can queued actions fire on destroyed
  mobs/rooms? Add nil guards in QueueDelayed execution
- Negative cache (`noTree`/`noRoomTree`) invalidation — ever stale?
  Add cache bust on file hot-reload if we ever support it
- Room state memory leak — rooms destroyed (ephemeral zones) but
  their BehaviorState retained in `roomStates` map
- Tree parse errors on startup — do we surface them clearly or
  silently disable the mob?

**Constraints:**
- Each fix independently testable
- Add nil-safety without breaking existing tests
