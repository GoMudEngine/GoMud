# Mob Aliveness 4.5 — Reactive Goal Generation Design

**Date:** 2026-05-27
**Status:** Approved (design)
**Roadmap position:** Phase 4 (strategic layer). After 4.5: 27/42.
**Depends on:** 1.1 (opinions), 1.6 (relationships), 4.1 (goal substrate), 4.2 (selection), 4.3 (catalog — Predicates + ContextScore), 4.4 (planners).
**Next chunks:** 4.6 Goal satisfaction & pruning → Phase 5 cross-cutting features.

---

## Summary

NPCs now react to world events — player kills an NPC's friend and that NPC gets a `revenge-mob` goal; player gifts an item and the receiver's opinion bumps up; player helps a mob in a fight against another mob and the helped mob warms to the player. Three "forced" hooks land too: the faction-kill counter that makes `revenge-faction` actually work, the faction-rep counter that makes `befriend-faction` move, and the materials-missing → wealth-item seed that lets `craft-item` planners gracefully fall back when they can't proceed.

**Architecture:** new `internal/seeders/` subpackage (mirrors 4.3 catalog + 4.4 planners pattern). One file per rule. Each file's `init()` registers via `Register(ruleFn, eventTypes...)`. A package-level dispatcher maps event types to subscribed rules; `main.go` adds one `events.AddListener(eventType, seeders.Dispatch)` per event type seeders care about. Each rule is a Go function that inspects the event payload, decides whether to act, and either calls `goals.Add(...)` (goal seeders), bumps `mob.Character.MiscData` (counter writers), or calls `opinions.Bump(...)` (opinion shifters). One architectural exception: the `craft_materials_to_wealth_item` rule is invoked directly from the craft-item planner's Failure branch rather than via event subscription (no clean "planner failed because materials missing" world event).

**Scope (L):** 10 rules total — 3 forced + 7 reactive. Sized up from M during brainstorming to ship all categories deep enough to feel observable.

---

## Goals

- Ship the three forced hooks 4.4 planners depend on: faction-kill counter, faction-rep counter, materials-missing → wealth-item seed.
- Ship seven additional reactive rules that make the world feel responsive to player actions: friend-killed → revenge, witness-of-theft → revenge, aggressive-action → revenge, gift → opinion boost, quest-completion → opinion boost, combat-assist → opinion boost, mastery-milestone → priority bump.
- Centralize all reactive logic in one `internal/seeders/` subpackage — single place to find / test / extend.
- Stateless dispatcher with panic-recovery around each rule invocation (one bad rule cannot cascade-fail the event handler or other rules).
- All goal seeds flow through `goals.Add` so 4.3's `DedupKey` mechanism automatically collapses repeat seedings against the same target.
- Counter writes use the same `mob.Character.MiscData` key convention as 4.3's predicates that read them (`faction_kills_inflicted:<faction>`, `faction_rep_built_with:<faction>`).

## Non-goals

- **Cross-type goal conflict mechanism** (deferred from 4.3, still deferred). Seeders pre-check obvious contradictions (don't seed revenge if befriend already exists for the same target) on a per-rule basis; the engine doesn't enforce.
- **Goal satisfaction / pruning sweep** — that's 4.6. 4.5 seeds freely; 4.6 cleans up.
- **Bespoke per-content reactive rules** — generic rules only. Quest-specific reactions ("completing quest 47 boosts opinion with three specific NPCs by 20 each") belong in quest YAML, not seeders.
- **Content-side trigger lists** (insult word lists, hate-speech detection, etc.) — out of engine scope.
- **Reactive-seed visualization admin command** — rules log to mudlog at debug level when they fire; no `seeders log` command in 4.5.
- **Per-archetype seed gating** — no opt-out mechanism. Goal selector's ContextScore filters irrelevant-to-archetype goals naturally; 4.6 prunes permanent never-progressed strays.
- **Hostility / faction-aware nuance** in friend-killed propagation. 4.5 walks `relationships.RelationsOf` straightforwardly — edge cases ("the friendship was secretly broken") belong in richer 1.6 extensions.
- **Player-side opinion display** — admin can already inspect via `opinion show <mob>`; players never see opinion numbers (per project rule "no hard numbers in player-facing text").
- **`insult` opinion-drop rule** — explicitly dropped during brainstorming. `taunt` (existing combat verb using rhetoric skill) already covers "deliberate social aggression with consequences"; a separate `insult` would mostly be duplicate behavior with worse mechanical hooks.

---

## 1. Architecture & data flow

```
internal/goals/                    (4.1 substrate + 4.2 selection)
internal/goals/catalog/            (4.3 — Predicates + ContextScore + DedupKey)
internal/planners/                 (4.4 — per-goal-type planners)
internal/seeders/                  (NEW — 4.5)
    seeders.go                       framework: RuleFn type + Register + Dispatch
    state.go                         shared helpers (cooldown markers, factions walk, etc.)
    context.md                       package overview per project convention
    faction_kill_counter.go          rule 1
    faction_rep_counter.go           rule 2
    craft_materials_to_wealth_item.go rule 3 (planner-invoked, not event-subscribed)
    friend_killed_to_revenge.go      rule 4
    witness_of_theft_to_revenge.go   rule 5
    aggressive_action_to_revenge.go  rule 6
    gift_to_opinion_boost.go         rule 7
    quest_completion_to_opinion_boost.go rule 8
    combat_assist_to_opinion_boost.go rule 9
    mastery_milestone_to_priority_bump.go rule 10
    <one _test.go per rule alongside>
```

**Per-event data flow** (for any rule subscribed to event type X):

1. Game code fires `events.AddToQueue(events.X{...})`.
2. The events package's listener dispatch fires every registered `(X-type, listener)` pair.
3. `seeders.Dispatch(event)` is registered as one such listener (one entry per type, wired in main.go boot).
4. `Dispatch` looks up `registry[event.Type()]` → slice of `RuleFn`.
5. Each rule invoked under panic recovery. Rule inspects event payload, applies its own logic (faction filter, relationship walk, witness scan, etc.), and produces zero or more effects.
6. Effects flow through normal substrate APIs: `goals.Add`, `mob.Character.SetMiscData`, `opinions.Bump`. Each effect is observable through the existing systems (4.2 Recompute fires on goal mutations; predicates read counters next tick; admin commands inspect opinions).

**Architectural exception — rule 3 (`craft_materials_to_wealth_item`)** is the one rule NOT triggered by a world event. The trigger is "the craft-item planner's Failure branch determined that materials are missing." There's no world-event hook for that — it's an internal planner state. So rule 3 exposes a public function `seeders.SeedMaterialsForRecipe(mob, recipeId)` that the craft-item planner calls directly from `internal/planners/craft_item.go`'s Failure branch. The rule's logic still lives in `internal/seeders/` for discoverability; only the trigger path differs.

**Key invariant:** the seeders package owns the *rule logic*. Triggers are either event-driven (registered with the dispatcher) or planner-driven (exported function called from the planner). In both cases the rule lives in seeders/.

---

## 2. API surface

```go
package seeders

// RuleFn is invoked once per event the rule is registered for. The rule
// is responsible for inspecting the event payload, deciding whether to
// act, and applying its effects via the normal substrate APIs
// (goals.Add, mob.Character.SetMiscData, opinions.Bump).
//
// Rules MUST be safe to call concurrently — though in practice they are
// dispatched serially from the event listener, defensive thread-safety
// is required because the event subsystem may evolve.
type RuleFn func(event events.Event)

// Register subscribes a rule to one or more event type names. Called
// from each per-rule file's init() function. ruleName is used for
// panic-recovery log lines + future admin tooling.
func Register(ruleName string, fn RuleFn, types ...string)

// Dispatch is the package-level event listener wired by main.go for
// every event type seeders care about. Looks up rules for the event's
// type, invokes each under panic recovery.
func Dispatch(event events.Event) events.ListenerReturn

// SeedMaterialsForRecipe is rule 3's public trigger. Called directly
// from internal/planners/craft_item.go when the planner determines
// materials are missing. Walks the recipe ingredients, checks each
// against mob inventory, seeds a wealth-item goal for each missing
// ingredient tag. Skips ingredients that already have a wealth-item
// goal targeting them (dedup).
//
// Architectural exception — see §1.
func SeedMaterialsForRecipe(mob *mobs.Mob, recipeId string)

// OnCombatAssist is rule 9's optional direct trigger if no clean
// "Aggro changed" event exists in the events package. Called from
// attack.go / taunt.go action handlers when a player engages a mob.
// (If a clean event hook exists, rule 9 subscribes the normal way and
// this function is unused.)
func OnCombatAssist(playerUserId int, attackerMob *mobs.Mob)
```

Internal helpers in `state.go`:

```go
// applyCooldown returns true and writes a fresh cooldown timestamp
// if the cooldown is not active. Returns false (no write) if active.
// Used by rules with per-pair throttling (gift, combat-assist, etc.).
//
// Cooldowns live on the BENEFICIARY mob's MiscData under
// "seed_cooldown:<rule_name>:<key>" where key is a per-rule identifier
// (e.g., "<userId>" for gift, "<attackerInstanceId>" for combat-assist).
func applyCooldown(mob *mobs.Mob, ruleName, key string, windowRounds uint64) bool

// seedRevengeGoalIfAbsent is a shared helper for the multiple revenge-
// seeding rules. Checks whether a revenge-mob goal targeting the same
// (kind, id) already exists on the mob; if not, calls goals.Add. Returns
// the added Goal or nil if skipped (dedup or error).
func seedRevengeGoalIfAbsent(mob *mobs.Mob, targetKind string, targetId, priority int) *goals.Goal
```

Main.go boot wiring (in addition to the 4.2/4.3/4.4 callback registrations):

```go
// Wire seeders.Dispatch to every event type the seeders package cares
// about. The list is hand-maintained — implementers add a line here
// when a new rule subscribes to a new event type. Chunk 4.5.
events.AddListener(events.MobDeath{}, seeders.Dispatch)
events.AddListener(events.Communication{}, seeders.Dispatch)
// ...others added per per-rule verification at impl time...
```

Plus a regular import of `internal/seeders` so the per-rule `init()`s fire.

---

## 3. The 10 seeder rules

### 3.1 `faction_kill_counter` — forced by 4.4

- **Trigger:** `events.MobDeath`
- **Filter:** killer must be a mob (player-as-killer doesn't write to mob counters). Reads `events.MobDeath.KillerMobInstanceId` (verify exact field at impl time).
- **Effect:** for each faction the victim belongs to (via `factions.FactionsForMob(victim)`), bump `mob.Character.MiscData["faction_kills_inflicted:<faction>"]` on the killer by 1 (uses an int read + write helper that tolerates int / int64).
- **Why:** 4.3 `revenge-faction` Predicate reads this counter; nothing else writes it.

### 3.2 `faction_rep_counter` — forced by 4.4

- **Trigger:** `events.Communication` (specifically positive interactions — give, successful trade, dialogue with positive subtype). Implementer verifies actual event-type and subtype filters at impl time.
- **Filter:** giver/initiator must be a mob whose `factions.FactionsForMob(giver)` returns at least one faction. Recipient must be a mob (to receive the rep bump).
- **Effect:** for each faction the giver belongs to, bump `mob.Character.MiscData["faction_rep_built_with:<faction>"]` on the recipient by 1.
- **Caveat:** if no clean Communication subtype distinguishes positive vs neutral interactions, implementer may need to either (a) add a subtype field to `events.Communication`, (b) subscribe to a more specific event (`Quest` completion may already exist), or (c) document the gap and stub. The catalog's `befriend-faction` predicate already documents that the counter ships in 4.5.
- **Why:** 4.3 `befriend-faction` Predicate reads this counter.

### 3.3 `craft_materials_to_wealth_item` — forced by 4.4, planner-invoked

- **Trigger:** `internal/planners/craft_item.go` calls `seeders.SeedMaterialsForRecipe(mob, recipeId)` from its Failure branch when materials are determined missing.
- **Filter:** none beyond the planner's own gating (recipe exists, recipe known, skill sufficient — materials missing is the only thing failing).
- **Effect:** walk `recipe.Ingredients`. For each ingredient tag the mob doesn't have: call `goals.Add(mobId, name, &Goal{Type: "wealth-item", Priority: 60, Params: map[string]any{"item_tag": tag}})`. The catalog's `wealth-item` `DedupKey` skips duplicate seedings naturally.
- **Why:** 4.4's craft-item planner expects this so its Failure-on-missing-materials becomes a productive fallback (mob switches to acquiring the missing ingredient rather than spinning).

### 3.4 `friend_killed_to_revenge`

- **Trigger:** `events.MobDeath`
- **Filter:** victim has at least one relationship edge of friendly subtype (`friend`, `family`, `lover`, etc. — exact set verified against 1.6 catalog at impl time). Killer must be resolvable to a target (mob or player).
- **Effect:** walk `relationships.RelationsOf(victim)`. For each edge of friendly type, find the related NPC via the other side of the edge; call `seedRevengeGoalIfAbsent(friend, killerKind, killerId, 85)`.
- **Priority:** 85 (above survival's 80 — grief outweighs baseline self-preservation).
- **Dedup:** revenge-mob's DedupKey naturally collapses repeat seedings against the same killer. If the killer rampages and kills 5 mutual friends in one zone, the surviving NPCs end up with one revenge goal each, not five.

### 3.5 `witness_of_theft_to_revenge`

- **Trigger:** `events.ItemOwnership` with a "stolen" subtype, OR a new `Theft` event if `ItemOwnership` doesn't subtype that way. Implementer verifies.
- **Filter:** thief must be a player (mob-vs-mob theft doesn't seed mob revenge in 4.5). Victim must be a mob (player-victim revenge isn't relevant since players manage their own grudges).
- **Effect:**
  - On victim: `seedRevengeGoalIfAbsent(victim, "player", thiefId, 90)` (priority 90 — very high; you were directly wronged).
  - On other mobs in the same room: `seedRevengeGoalIfAbsent(witness, "player", thiefId, 60)` for each (witnesses care less than the victim).
- **Implementation note:** room-mob walk via `rooms.LoadRoom(victim.RoomId).MobInstances` (or similar — verify).

### 3.6 `aggressive_action_to_revenge`

- **Trigger:** the combat-engagement event (verify — likely the moment `Character.SetAggro` is called on a mob with a player attacker). If no clean event exists, action handlers (`attack.go`, etc.) can call `seeders.OnAggressiveAction(player, attackedMob)` directly.
- **Filter:** attacker is a player; attacked NPC isn't already auto-hostile to all players (otherwise revenge is redundant noise).
- **Effect:**
  - On attacked NPC: `seedRevengeGoalIfAbsent(npc, "player", attackerId, 75)`.
  - On other non-hostile mobs in the same room: `seedRevengeGoalIfAbsent(witness, "player", attackerId, 50)`.
- **Why:** triggers BEFORE death. NPC that survives a hit-and-run remembers and can later act on the grudge.

### 3.7 `gift_to_opinion_boost`

- **Trigger:** `events.Communication` with `give` action subtype, OR a dedicated give-item event (verify via codegraph — `give.go` action probably fires something).
- **Filter:** giver is a player; receiver is a mob; gifted item has non-zero `ItemSpec.Value`.
- **Effect:** `opinions.Bump(receiverMobId, giverUserId, +N)` where N scales with item value:
  - value 1-49 → +1
  - value 50-199 → +3
  - value 200-999 → +5
  - value 1000+ → +8 (capped — even diamond-class gifts don't trivialize befriending)
- **Cooldown:** `applyCooldown(receiver, "gift_opinion_boost", strconv.Itoa(giverUserId), 100)` — once per 100 rounds per giver per receiver. Prevents spam-gifting a single NPC from instant max-friending.

### 3.8 `quest_completion_to_opinion_boost`

- **Trigger:** `events.Quest` with a completion subtype (verify exact shape). The event likely carries the questId and completingUserId.
- **Filter:** quest has a designated giver mob (most do; verify the quest data model). If the quest YAML declares a `complete_opinion_bump:` field, use that as N; otherwise default to +10.
- **Effect:** `opinions.Bump(questGiverMobId, completingUserId, +N)`.
- **No cooldown** — completing a quest is itself rate-limited by the quest's own gating; can't be spammed.

### 3.9 `combat_assist_to_opinion_boost`

- **Trigger:** player attack / taunt action targeting a mob whose `Aggro` is currently set to another non-player mob. If no clean event hook exists, action handlers (`attack.go`, `taunt.go`) call `seeders.OnCombatAssist(playerUserId, attackerMob)` directly.
- **Filter:**
  - Helped mob (`attacker.Character.Aggro.MobInstanceId` resolved) must exist and must NOT itself be hostile to the player (`factions.IsPeacefulToward(beneficiary, player)` check — prevents fake credit when the player attacks one of two hostile mobs that happened to be fighting each other).
- **Effect:** `opinions.Bump(beneficiaryMobId, playerUserId, +N)` where N is 3 (baseline). Could scale with damage dealt to attacker in a future tuning pass.
- **Multi-target:** if attacker was engaged with multiple mobs (rare — most attackers have a single Aggro target, but the future Combat Phase system may track multiple), bump each one's opinion independently. Each gets full N.
- **Cooldown:** `applyCooldown(beneficiary, "combat_assist", strconv.Itoa(attacker.InstanceId), 50)` — once per 50 rounds per beneficiary-per-attacker pair. Long fights with many player interventions still produce meaningful opinion gain, just not runaway stacks.

### 3.10 `mastery_milestone_to_priority_bump`

- **Trigger:** `events.SkillUsed` if the event exists AND fires on rank changes. If verification shows it doesn't fire that way, ship the rule with the registration commented out + a clear deferral note ("requires events.SkillUsed to fire on rank-up events"). The rest of 4.5 ships regardless.
- **Filter:** new rank is a multiple of 10 (10, 20, 30, ...) AND the new rank is below the soft cap of 50.
- **Effect:** seed `mastery-skill{skill_name: <skill>, target_rank: <next_milestone>, priority: 40}` into the mob. DedupKey on `skill_name` ensures only one mastery-skill per skill exists.
- **Why:** self-prompting NPCs autonomously aim at their next training milestone. Borderline-useful but small effort if the event hook exists.

---

## 4. Cross-rule patterns

### 4.1 Dedup

All goal-seeders route through `goals.Add` which invokes the catalog's `DedupKey` func. Repeat seedings against the same target collapse naturally. Cross-type conflicts (e.g., befriend + revenge on same target) — 4.3 deferred the conflict mechanism; seeders should pre-check obviously-contradictory pairs:

- `friend_killed_to_revenge`: skip if a befriend goal already targets the killer (the NPC has decided to forgive; don't override).
- `gift_to_opinion_boost` + revenge: gift still bumps opinion regardless; revenge goal isn't affected. The opinion bump might eventually cross threshold and obsolete the revenge goal naturally — that's 4.6's pruning to handle.
- `combat_assist_to_opinion_boost`: filtered upstream by the IsPeacefulToward check; no further dedup needed.

### 4.2 Archetype gating

None in 4.5. Bosses and noncombat passives may receive seeded goals (e.g., a peaceful merchant whose friend was killed gets a `revenge-mob` goal). The goal selector's ContextScore filters them naturally — a noncombat_passive with no combat skills scores `revenge-mob`'s effective value very low; the goal exists but never fires planner action. 4.6's prune sweep can clean up never-progressed goals.

### 4.3 Cooldown convention

Rules using per-pair cooldowns (gift, combat-assist) all share the `applyCooldown` helper in `state.go`. Cooldown markers live on the beneficiary mob's MiscData under `seed_cooldown:<rule_name>:<key>`. They are NOT prefixed with `plan:` so 4.4's `ClearPlanState` does NOT wipe them on goal switch — the cooldown is independent of strategic-layer state.

### 4.4 Logging

Each rule logs at mudlog Debug level when it fires, with structured fields: `rule_name`, `event_type`, `mob_id`, `effect` (goal-seeded / counter-bumped / opinion-shifted), `value` (for opinion + counter cases). Single line per firing; quiet in normal operation, traceable in admin debug.

### 4.5 Panic recovery

`Dispatch` wraps each rule call in `defer/recover` (mirrors 4.2 `invokeContextScore`, 4.3 `invokeDedupKey`, 4.4 `invokePlannerSafely` patterns):

```go
defer func() {
    if r := recover(); r != nil {
        mudlog.Warn("seeders.rule panic",
            "rule", ruleName, "event_type", event.Type(),
            "panic", fmt.Sprintf("%v", r))
    }
}()
```

One bad rule cannot crash the event handler or prevent other rules' invocation.

---

## 5. Edge cases

| # | Case | Behavior |
|---|---|---|
| 1 | Killer mob has no factions (`factions.FactionsForMob` returns empty) | `faction_kill_counter` rule no-ops. Counter not bumped. Faction-based revenge-faction never satisfies if the only victim faction has no killer-faction overlap. Correct. |
| 2 | Victim has no relationship edges (lone NPC) | `friend_killed_to_revenge` walks an empty list. No revenge goals seeded. Correct — nobody mourns. |
| 3 | Mob B is killed; mob A had a friendly edge to B but A was killed first earlier | `friend_killed_to_revenge` queries `mobs.GetInstance` per friend to seed; if A's instance is gone (dead), the seed silently skips. Goal not seeded into a dead mob. Correct. |
| 4 | Player gifts the same item-id 100 times in quick succession to the same NPC | First gift bumps opinion + sets cooldown. Subsequent gifts skip (cooldown active). After 100 rounds, next gift bumps again. Spam-proof. |
| 5 | Combat-assist: mob A's Aggro changes mid-fight (re-targets) between when player attacks and when the rule fires | Rule reads `attacker.Aggro` at fire-time; reflects current state. If A re-targeted to a different mob, the new target gets the credit. Correct (player helped whoever A is currently attacking). |
| 6 | Quest completion event fires but the quest's giver mob is no longer alive | Rule skips with no error — `opinions.Bump` on a nonexistent mob id is a no-op or logs a warning depending on the opinions package's contract. Verify at impl time. |
| 7 | Two seeders subscribe to the same event type; rule 1 panics; rule 2 also subscribed | `Dispatch` invokes rules in registration order under independent panic-recovery. Rule 1 panics → logged → rule 2 still invoked. Correct. |
| 8 | Materials-missing seed fires for a recipe whose Ingredients include both a tag the mob has AND a tag it doesn't | Rule iterates all ingredients; for each missing one, seeds a wealth-item. The catalog's DedupKey collapses repeat seedings. Mob ends up with one wealth-item per missing tag. Correct. |
| 9 | `events.SkillUsed` doesn't fire on rank-up events as the rule assumes | Rule 10 registration is commented out OR the event subscription is no-op. Other 9 rules unaffected. Deferral noted at impl time. |
| 10 | Player attacks a mob that has Aggro on a player (not a mob) | `combat_assist_to_opinion_boost` filter: `attacker.Aggro.UserId > 0` means the attacker was engaged with a player, not a mob. Rule skips (no mob beneficiary to credit). Correct — player vs player combat isn't an assist scenario for mob NPCs. |
| 11 | A friend mob's revenge goal already exists targeting the killer (e.g., the mob is on a revenge tear) and another of their friends gets killed | `seedRevengeGoalIfAbsent` finds the existing goal via DedupKey; skips. Revenge goal stays at its current priority — doesn't escalate. Future enhancement could bump priority on repeat-offense, but 4.5 keeps it simple. |
| 12 | `MobDeath` event fires for a mob whose template no longer loads (admin removed mid-session) | `factions.FactionsForMob` returns empty; `relationships.RelationsOf` returns empty. All rules no-op gracefully. |
| 13 | The `internal/seeders` package never gets imported (someone removes the import from main.go) | `init()` funcs never fire; `Register` never called; `Dispatch` never invoked. Events fire; no seeders react. Game still works — just no reactive aliveness. (Same failure mode as removing the planners blank-import in 4.4.) |

---

## 6. Testing strategy & rollout

### 6.1 Per-rule unit tests

`internal/seeders/<rule>_test.go` — one per rule. Each covers:

- Registration check: rule appears in dispatcher for its event type after package init.
- Happy path: event with matching payload → expected effect (goal added / counter bumped / opinion shifted).
- Each filter branch: wrong faction, hostile beneficiary, missing relationship edge, etc. → no effect.
- Dedup / cooldown round-trip: re-fire the same scenario, observe second invocation skipping.

Estimated ~4-6 tests per rule × 10 rules = ~50 tests.

### 6.2 Framework tests

`internal/seeders/seeders_test.go`:

- `Register` + `Dispatch` round-trip — rule registered for type X fires only on type-X events.
- Multiple rules subscribed to the same event type all fire.
- Panic in one rule doesn't prevent other rules' invocation.
- `Dispatch` with no registered rules for the event type is a graceful no-op.

`internal/seeders/state_test.go`:

- `applyCooldown` returns true + writes on first call.
- `applyCooldown` returns false within window.
- `applyCooldown` returns true again after window elapses.

### 6.3 Cross-package integration tests

A few end-to-end checks:

- `friend_killed_to_revenge`: seed a relationship edge via the test API (chunk 1.6), fire a `MobDeath` event, verify a `revenge-mob` goal exists on the friend after dispatch settles.
- `faction_kill_counter`: fire `MobDeath` 5 times against members of faction X, verify the killer's `faction_kills_inflicted:X` MiscData = 5. Then verify the `revenge-faction` predicate satisfies at target=5.
- `craft_materials_to_wealth_item`: construct a mob with a craft-item goal whose recipe is missing ingredient X. Invoke `seeders.SeedMaterialsForRecipe`. Verify a `wealth-item` goal with `item_tag=X` now exists on the mob.

### 6.4 Boot smoke (per pre-push SOP)

- Server boots clean with all 10 rules registered (or rule 10 cleanly deferred if SkillUsed event doesn't fire that way).
- Per-event listeners hooked up in main.go without panic.
- Engineered MobDeath: admin-kills a test mob whose template has a friend edge to another test mob. Verify a new `revenge-mob` goal seeded on the friend via `goal current <friend>`.

### 6.5 Out of scope for 4.5 tests

- 4.6's prune sweep (separate chunk).
- Bespoke per-quest reactive content (content authoring, not engine).
- Long-tail event types not currently in the 10-rule catalog.

### 6.6 Rollout — feature branch decomposition

Single chunk on `feature/aliveness-4.5-reactive-goal-generation`. Suggested task ordering:

1. **Framework**: `RuleFn`, `Register`, `Dispatch` with panic recovery + framework tests.
2. **`state.go`** helpers: `applyCooldown`, `seedRevengeGoalIfAbsent` + tests.
3. **main.go boot wiring**: `events.AddListener` calls per event type seeders care about + regular import of `internal/seeders`.
4. **Rule 1**: `faction_kill_counter` (foundational — unblocks revenge-faction observability).
5. **Rule 2**: `faction_rep_counter` (foundational — unblocks befriend-faction observability).
6. **Rule 3**: `craft_materials_to_wealth_item` (with planner integration — modifies `internal/planners/craft_item.go`'s Failure branch to call `seeders.SeedMaterialsForRecipe`).
7. **Rule 4**: `friend_killed_to_revenge` (the roadmap headliner).
8. **Rule 5**: `witness_of_theft_to_revenge`.
9. **Rule 6**: `aggressive_action_to_revenge` (with possible action-handler integration if no clean event).
10. **Rule 7**: `gift_to_opinion_boost` (with cooldown).
11. **Rule 8**: `quest_completion_to_opinion_boost`.
12. **Rule 9**: `combat_assist_to_opinion_boost` (with cooldown + action-handler integration if no clean event).
13. **Rule 10**: `mastery_milestone_to_priority_bump` (with "skip if event doesn't exist" caveat).
14. **`internal/seeders/context.md`** per project convention.
15. **Smoke + roadmap rollup (26/42 → 27/42) + PATCH_NOTES entry**.

15 tasks. Slightly larger than the rough estimate from brainstorming (14) because the state.go helpers warrant their own task. Per-rule tasks are simpler than 4.4's planners — each is one Go file with `init()` + per-rule logic + ~4 tests.

**Push to prod is safe** — seeders fire on events but their effects (goal seeds, counter writes, opinion bumps) all flow through existing 4.2/4.3/4.4 substrate. Observable change: NPCs visibly react to player actions.

### 6.7 Roadmap rollup after 4.5 ships

**27/42.** Next: 4.6 Goal satisfaction & pruning (sweep loop that detects satisfied / permanent-stuck / expired goals and removes them). 4.6 is sized S — the last small chunk in Phase 4 before Phase 5 cross-cutting features.

---

## 7. File touch list

**New:**
- `internal/seeders/` subpackage:
  - `context.md` — package overview per convention.
  - `seeders.go` — `RuleFn`, `Register`, `Dispatch`, panic-recovery wrapper.
  - `state.go` — `applyCooldown`, `seedRevengeGoalIfAbsent`, shared helpers.
  - 10 rule files (`faction_kill_counter.go`, `faction_rep_counter.go`, etc.) + test files alongside.

**Modified:**
- `main.go` — `events.AddListener(eventType, seeders.Dispatch)` for each subscribed event type; regular import of `internal/seeders`.
- `internal/planners/craft_item.go` — Failure branch calls `seeders.SeedMaterialsForRecipe(mob, recipeId)` when materials are determined missing. (Rule 3's planner-integration touch point.)
- `internal/actions/attack.go` and/or `internal/actions/taunt.go` — call `seeders.OnCombatAssist(...)` and/or `seeders.OnAggressiveAction(...)` if no clean events.AddListener path exists for rules 6 + 9. (Implementer verifies at impl time; may not be needed if event subscriptions cover everything.)
- `MOB_ALIVENESS_ROADMAP.md` — flip 4.5 to Done, bump rollup to 27/42, size M → L per brainstorming.
- `PATCH_NOTES.md` — chunk 4.5 entry.

**Not touched in 4.5:** existing `internal/hooks/` (no new hook files — seeders are their own package), 4.3 catalog, 4.4 planners (apart from the one craft_item planner Failure-branch hook), 4.2 selection, 4.1 substrate.
