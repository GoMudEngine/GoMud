# Mob Aliveness 4.4 — Strategic → Tactical Translation Design

**Date:** 2026-05-27
**Status:** Approved (design)
**Roadmap position:** Phase 4 (strategic layer). After 4.4: 26/42.
**Depends on:** 4.1 (goal substrate), 4.2 (goal selection), 4.3 (goal types catalog), Phase 2 (tactical verbs).
**Next chunks:** 4.5 Reactive goal generation → 4.6 Goal satisfaction & pruning.

---

## Summary

Bridges desire to action. NPCs now finally *act* on the goals 4.2 selects from the 4.3 catalog — combat-capable mobs flee when low on HP, thieves wander to vendors and sell loot when seeded with `wealth-gold`, named NPCs path to and pursue revenge / protection / befriend targets, foragers visit unvisited zones, crafters seek stations to produce known recipes.

**Architecture:** a new `internal/planners/` package registers one `PlanFn` per goal type (mirrors 4.3's catalog pattern). Planners are pure Go functions called every mob round tick when a goal of their type is current. They return a single command to execute this tick plus a btree status. Stateful planners write intermediate progress to `mob.Character.MiscData` under a `plan:<goal_type>:` key prefix; the prefix lets us cleanly wipe state on goal switch.

A new btree action `try_goal_planner` dispatches per `goals.CurrentGoalOf` lookup. Authors place it explicitly in each archetype's tree at the priority they want — typically after reflex reactions (flee-if-alpha-dead, pack-cohesion) and before idle/wander. Boot wiring registers a `planners.ClearPlanState` callback into `goals.Recompute` so plan state gets wiped on goal switch.

**Scope (XL):** 13 deep planners + framework + per-archetype btree edits. All goal types ship with observable behavior. Permanent-stuck-goal detection deferred to 4.6.

---

## Goals

- Ship a deep planner for every type in the 4.3 catalog so NPCs visibly pursue their current goal.
- Keep planners pure functions: `Plan(mob, goal) PlanResult` — stateless from the framework's perspective; intermediate progress lives in `mob.MiscData` under a `plan:<goal_type>:` prefix.
- Integrate via one new btree action `try_goal_planner` per the existing action-library pattern (`try_combat`, `try_forage`, etc.). Authors place it explicitly in archetype trees at the priority they want.
- Plan state cleanup: `goals.Recompute` invokes a registered `ClearPlanState` callback on every goal switch — wipes `plan:`-prefixed MiscData keys.
- Plan-failure recovery is minimal: `StatusFailure` from a planner lets the btree fall through to its next action. No automatic goal demotion; permanent stuck-detection waits for 4.6.
- Stay decoupled: `internal/planners/` imports `goals`, `mobs`, `items`, `rooms`, `factions`, `shops`, `crafting` — but is itself imported only by `behaviortree` (for the action) and `main` (for the boot wiring). The `goals → planners` bridge goes through a registered callback to avoid a cycle.

## Non-goals

- General-purpose planner (HTN / GOAP) — explicitly out per roadmap.
- New tactical verbs — planners use only the existing Phase 2 verb set.
- Reactive goal generation (4.5) — `craft-item`'s "fall through to Failure when materials missing" assumes 4.5 will eventually seed a `wealth-item` for the missing tag, but 4.4 ships only the Failure path.
- Goal satisfaction / pruning sweep (4.6) — `protection-mob` after target's death, permanently-stuck plans, expired goals → all 4.6's job.
- Plan visualization admin command — inspecting MiscData via existing admin commands is the workaround. A future `goal plan <mob>` cmd is doable but out of 4.4 scope.
- Schedule / patrol integration beyond ordering — schedules and patrols still own their time-slots; goals fire when those are idle. Tighter coupling (e.g., goals influencing schedule choices) is Phase 5+.
- Companion / pet planning — companions still use existing follow/fight rules.
- Cross-type conflict mechanism — deferred from 4.3; revisit if 4.4 testing surfaces real issues.
- Goal cooldowns / patience timers — only `befriend` has a built-in cooldown (don't spam emotes). No global patience system.

---

## 1. Architecture & data flow

```
internal/goals/                    (4.1 substrate + 4.2 selection)
internal/goals/catalog/            (4.3 — Predicate + ContextScore + DedupKey)
internal/planners/                 (NEW — 4.4)
    planners.go                      package doc + RegisterPlanner / LookupPlanner / PlanResult types
    state.go                         ClearPlanState + plan-key prefix conventions
    helpers.go                       shared subsystem-adapter helpers (shop finder, faction filter, zone graph, etc.)
    skill_training_table.go          per-skill heuristic dispatch table (consumed by mastery-skill)
    survival.go                      one file per planner, init() registers
    wealth_gold.go
    wealth_item.go
    craft_item.go
    revenge_mob.go
    revenge_faction.go
    protection_mob.go
    protection_faction.go
    befriend.go
    befriend_faction.go
    mastery_skill.go
    mastery_equip.go
    visit_zone.go
    context.md                       package overview (per project convention)
internal/behaviortree/actions_goal.go  NEW — try_goal_planner btree action
```

**Per-tick data flow** (for one mob whose current goal type has a registered planner):

1. Behavior tree fires.
2. Tree reaches `try_goal_planner` node at its author-chosen position.
3. Action calls `goals.CurrentGoalOf(int(mob.MobId), namesimple)` → `*Goal` or nil.
4. If nil → `StatusFailure`, btree falls through.
5. Else calls `planners.LookupPlanner(goal.Type)` → `PlanFn` or nil.
6. If nil → `StatusFailure`.
7. Else invokes planner under panic recovery (mirrors 4.2's ContextScore wrapper). Returns `PlanResult{Command, Status}`.
8. If `result.Command != ""` → `mob.Command(result.Command)`.
9. Returns `result.Status` to the btree.

**Per-goal-switch data flow:**

1. `goals.Recompute` detects a switch (4.2 path unchanged).
2. After persisting the new current goal + emitting the `goals.switch` log line, Recompute invokes the registered `ClearPlanState` callback.
3. Callback walks `mob.Character.MiscData` and deletes keys matching `plan:*`.
4. Next tick the new goal's planner starts fresh.

**Key invariant:** planners are stateless from the framework's perspective. They MAY read/write `mob.MiscData` under their `plan:<goal_type>:` prefix, but that state is never durable across goal switches — Recompute clears it.

---

## 2. API surface

```go
package planners

// PlanFn is the per-tick planner. Stateless from the framework's
// perspective — for multi-step plans, write intermediate progress to
// mob.Character.MiscData under the convention "plan:<goal_type>:<key>".
// State is wiped automatically on goal switch (via ClearPlanState
// callback registered in main.go).
type PlanFn func(mob *mobs.Mob, goal *goals.Goal) PlanResult

// PlanResult is what a planner returns each tick.
type PlanResult struct {
    // Command to execute this tick (empty string = no action; btree falls
    // through to the next node). Executed via mob.Command(cmd) — reuses
    // the existing mob command path (same as schedule idle commands).
    Command string

    // Status propagated as the try_goal_planner btree action's result.
    //   StatusRunning = action consumed this tick; planner has more to do.
    //   StatusSuccess = goal advanced meaningfully (let predicate settle).
    //   StatusFailure = planner can't make progress; btree falls through.
    Status BTreeStatus
}

// BTreeStatus mirrors the behavior-tree status enum (re-exported to
// avoid forcing planners to import internal/behaviortree, which would
// create a cycle).
type BTreeStatus int
const (
    StatusFailure BTreeStatus = iota
    StatusSuccess
    StatusRunning
)

// Register a planner for a goal type. Called from each per-type
// planner file's init() function. Late registrations overwrite earlier
// ones (last-write-wins; useful for test override).
func RegisterPlanner(goalType string, fn PlanFn)

// LookupPlanner returns the registered planner for a goal type, or nil
// if none. Called by the try_goal_planner btree action.
func LookupPlanner(goalType string) PlanFn

// ClearPlanState wipes all "plan:" prefixed keys from mob.MiscData.
// Wired into goals.Recompute via a SetPlanStateClear callback registered
// in main.go (avoids goals→planners import cycle).
func ClearPlanState(mob *mobs.Mob)

// PlanKeyPrefix is "plan:". Exported so test helpers + admin tooling can
// recognize planner-owned MiscData keys.
const PlanKeyPrefix = "plan:"
```

Goals package extension (mirrors 4.2's SetWeightsLookup / 4.3's SetArchetypeDefaultsLookup patterns):

```go
package goals

// PlanStateClearFn wipes any planner-owned intermediate state on a mob.
// Registered once at boot from main.go; invoked by Recompute on every
// goal switch. nil = no cleanup (safe default for tests).
type PlanStateClearFn func(mob *mobs.Mob)

func SetPlanStateClear(fn PlanStateClearFn)
```

Behavior-tree action (in `internal/behaviortree/actions_goal.go`):

```go
package behaviortree

// actGoalPlanner is the btree action registered as "try_goal_planner".
// Dispatches to planners.LookupPlanner per the mob's current goal.
func actGoalPlanner(mob *mobs.Mob, args map[string]any) Status {
    // 1. Lookup current goal via 4.2 accessor.
    // 2. Lookup planner via 4.4 registry.
    // 3. Invoke under panic recovery.
    // 4. Execute returned command + propagate status.
}
```

Main.go boot wiring (in addition to 4.2's SetWeightsLookup + 4.3's SetArchetypeDefaultsLookup):

```go
goals.SetPlanStateClear(planners.ClearPlanState)
```

Plus a blank import to fire all 13 planner `init()` registrations:

```go
import _ "github.com/GoMudEngine/GoMud/internal/planners"
```

---

## 3. The 13 planners

Per-planner sketch. **Read state** lists subsystems the planner inspects. **Output command** is the literal mob command string. **MiscData keys** lists `plan:<goal_type>:`-prefixed keys this planner uses for intermediate state. **Status** describes the exit per branch.

### 3.1 survival

- **Purpose:** Flee combat or recover HP until safe.
- **Read state:** `mob.Character.Health`, `mob.Character.HealthMax`, `mob.Character.Aggro`, mob inventory (for healing potions).
- **MiscData keys:** none (purely reactive).
- **Branches:**
  - `Aggro != nil` → `flee <random_exit_name>`, `StatusRunning`. If no exits, fall through to next.
  - `hp_pct < safe_threshold` AND has a healing potion in inventory → `drink <potion>`, `StatusRunning`.
  - `hp_pct < safe_threshold` AND no potion → `rest`, `StatusRunning`.
  - `hp_pct >= safe_threshold AND aggro == nil` → empty command, `StatusSuccess` (predicate fires next tick).

### 3.2 wealth-gold

- **Purpose:** Accumulate gold by selling loot at vendor shops.
- **Read state:** `mob.Character.Gold`, `mob.Character.Items` (filter to sellable: any item with non-zero `ItemSpec.Value` AND not in the goal's exclusion set), shop locations in zone via `shops.InZone(mob.Character.Zone)`.
- **MiscData keys:** `plan:wealth-gold:target_shop_room` (int — sticky chosen shop until reached or invalidated).
- **Branches:**
  - `Gold >= target` → empty command, `StatusSuccess`.
  - Has sellable items + at a vendor room → `sell all`, `StatusRunning`. (Defer to `sell` command's own handling; multi-item sell loops over inventory.)
  - Has sellable items + not at a vendor → resolve `target_shop_room` (from MiscData or pick nearest shop in zone via `findShopInZoneBuying`). Write it back. Emit `pathto <room>`, `StatusRunning`.
  - No sellable items + no vendor activity needed → `wander`, `StatusRunning` (chance of loot).
  - No vendors in zone → empty command, `StatusFailure` (btree falls through).

### 3.3 wealth-item

- **Purpose:** Acquire a specific item by tag or id.
- **Read state:** inventory + equipment (via 4.3 catalog's `mobHasItem`), shop stock for the target item in zone, `mob.Character.Gold`.
- **MiscData keys:** `plan:wealth-item:target_shop_room` (int).
- **Branches:**
  - Item present (anywhere on mob) → empty command, `StatusSuccess`.
  - Shop in zone sells item + sufficient gold + at that shop → `buy <item>`, `StatusRunning`.
  - Shop in zone sells item + sufficient gold + not at shop → resolve `target_shop_room` via `findShopInZoneSelling`. Emit `pathto <room>`, `StatusRunning`.
  - Shop in zone sells item + insufficient gold → empty command, `StatusFailure` (4.5 will seed a coexisting `wealth-gold` reactively; for now drop through).
  - No shop sells it → `forage` if forager-capable archetype, else `wander`, `StatusRunning`.

### 3.4 craft-item

- **Purpose:** Produce the recipe's output item.
- **Read state:** `crafting.GetRecipe(recipe_id)`, `mob.Character.KnownRecipes`, `mob.Character.Skills`, `mob.Character.Items`, `mob.Character.ComponentItems`, station rooms in zone.
- **MiscData keys:** `plan:craft-item:target_station_room` (int).
- **Branches:**
  - Recipe unknown → empty command, `StatusFailure` (catalog already filters via ContextScore=0; planner is defensive).
  - Skill below `recipe.SkillMinimum` → empty command, `StatusFailure` (mastery-skill coexistence handles training; planner drops out).
  - Materials missing → empty command, `StatusFailure` (4.5 will reactively seed a `wealth-item` for the missing material tag).
  - Materials on hand + recipe's `Station` is empty OR mob in same room as a matching station → `craft <recipe_id>`, `StatusRunning`.
  - Materials on hand + station needed + not at station → resolve `target_station_room` via `findCraftingStationInZone(mob, recipe.Station)`. Emit `pathto <room>`, `StatusRunning`.
  - Recipe output present in inventory → empty command, `StatusSuccess`.

### 3.5 revenge-mob

- **Purpose:** Kill or witness death of the named target.
- **Read state:** target lookup (`mobs.GetAllMobInstanceIds` filter for mob kind; `users.GetByUserId` for player kind), `mob.Character.RoomId`, `mob.Character.Zone`, target's current room.
- **MiscData keys:** none (target room re-resolved each tick; if target moves, planner adapts immediately).
- **Branches:**
  - Target dead (kind=mob: no instance found; kind=player: nil or `Health <= 0`) → empty command, `StatusSuccess` (catalog predicate satisfies next tick; goal removed by 4.6).
  - Target in same room → `attack <target_name_or_id>`, `StatusRunning`.
  - Target in zone (resolved room ≠ mob's room but same zone) → `pathto <target_room>`, `StatusRunning`.
  - Target in different zone → empty command, `StatusFailure` (no cross-zone pursuit in 4.4).
  - Target not resolvable (e.g., player offline) → empty command, `StatusFailure`.

### 3.6 revenge-faction

- **Purpose:** Inflict kills against faction members until counter reaches target.
- **Read state:** faction member instances in zone (via `factions.FactionsForMob` filter, reused from 4.3 catalog's `factionMembersInZone`), `mob.MiscData["faction_kills_inflicted:<faction>"]` (the 4.5-written counter — for 4.4 planner just reads to confirm not satisfied; counter check is already done by catalog Predicate).
- **MiscData keys:** none.
- **Branches:**
  - Hostile faction member in same room → `attack <member>`, `StatusRunning`.
  - Hostile faction member in zone → `pathto <member_room>`, `StatusRunning`.
  - No member in zone → `wander`, `StatusRunning` (search).
  - Counter has reached target (defensive — Predicate normally fires first) → empty command, `StatusSuccess`.

### 3.7 protection-mob

- **Purpose:** Defend the named ally; intervene when they're attacked.
- **Read state:** target lookup (same as revenge-mob), target's room, target's `Aggro` state, the target's attacker (`target.Aggro.MobInstanceId` / `target.Aggro.UserId`).
- **MiscData keys:** none.
- **Branches:**
  - Target dead → empty command, `StatusFailure` (4.6 prunes after timeout).
  - Target in combat in same room → `attack <attacker>`, `StatusRunning`.
  - Target in combat in zone (different room) → `pathto <target_room>`, `StatusRunning`.
  - Target safe in same room → empty command, `StatusSuccess` (we're at our charge; nothing to do).
  - Target safe in zone (different room) → `pathto <target_room>`, `StatusRunning` (close the distance; ready for next attack).
  - Target in different zone → empty command, `StatusFailure`.

### 3.8 protection-faction

- **Purpose:** Defend faction members in current zone from hostile mobs.
- **Read state:** faction members in zone, hostiles in zone (per 4.3 catalog's `factionMemberInCombatInZone` + `hostileMobsInZone`).
- **MiscData keys:** none.
- **Branches:**
  - Faction member in combat in same room → `attack <attacker>`, `StatusRunning`.
  - Faction member in combat elsewhere in zone → `pathto <member_room>`, `StatusRunning`.
  - Hostile mob in zone (no member-in-combat yet) → `pathto <hostile_room>`, then `attack <hostile>` on arrival, `StatusRunning`.
  - Zone calm → empty command, `StatusSuccess` (job is done for now; let predicate-never-satisfies keep the goal alive).
  - No faction members in zone → empty command, `StatusFailure`.

### 3.9 befriend

- **Purpose:** Raise opinion with a specific target above threshold via positive social interactions.
- **Read state:** target lookup (`opinions.Get(int(mob.MobId), target_id)` for player kind; mob→mob returns 0 per 4.3 catalog limitation), target's room, current round count, suitable gift items in mob's inventory.
- **MiscData keys:** `plan:befriend:cooldown_round` (uint64 — earliest round on which to fire next interaction; prevents emote-spam every tick).
- **Branches:**
  - Opinion >= threshold → empty command, `StatusSuccess`.
  - Cooldown active (`now < cooldown_round`) → empty command, `StatusRunning` (waiting it out; don't fall through to other behaviors).
  - Target in same room → pick one of: `say hello, <target_name>` / `emote bows to <target>` / `give <gift_item> <target>` (rotated per call). Write `cooldown_round = now + BefriendInteractionCooldown` (config knob, default 30). `StatusRunning`.
  - Target in same zone → `pathto <target_room>`, `StatusRunning`.
  - Target out of zone → empty command, `StatusFailure`.

### 3.10 befriend-faction

- **Purpose:** Raise rep with a faction via positive interactions with its members in zone.
- **Read state:** faction members in zone (reused from `factionMembersInZone`), the focused member (from MiscData), current round.
- **MiscData keys:** `plan:befriend-faction:focus_mob_instance_id` (int — sticks to one member at a time to avoid scattering attention; faction members are mobs per the 4.3 catalog's `factionMembersInZone` walk), `plan:befriend-faction:cooldown_round` (uint64).
- **Branches:**
  - No members in zone → empty command, `StatusFailure`.
  - No focus or stale focus (focus member no longer in zone) → pick a member, write to MiscData, fall through to next branch.
  - Cooldown active → empty command, `StatusRunning`.
  - Focus member in same room → positive interaction (an emote from the social rotation, applied near the member). Set cooldown. `StatusRunning`.
  - Focus member in zone (different room) → `pathto <member_room>`, `StatusRunning`.
- **Limitation:** the actual rep counter (`faction_rep_built_with:<faction>` in mob MiscData) is WRITTEN by chunk 4.5's reactive hooks (e.g., "successful trade with a member" → bump). 4.4's planner just gets the mob in proximity to faction members; observable rep movement waits for 4.5. Documented in the catalog's `befriend_faction.go` file-level comment too.

### 3.11 mastery-skill

- **Purpose:** Train a named skill toward target rank.
- **Read state:** `mob.Character.GetSkillLevel(skill_name)`, the skill's training-context kind (from `skill_training_table.go`), available training opportunities in zone.
- **MiscData keys:** none.
- **Skill training table** (data-driven dispatch — small Go map literal):
  ```
  "weapon-combat" / "unarmed-combat" / "ranged-combat" → "combat" → wander/attack
  "spellcasting"                                       → "combat" → wander/attack
  "rhetoric"                                           → "social" → emote/say
  "skullduggery"                                       → "skullduggery" → wander (steal/lockpick opportunities)
  "smithing" / "tanning" / "tailoring" / "alchemy"
   / "fletching" / "cooking" / "salvage"               → "crafting" → station + craft
  "foraging"                                           → "foraging" → forage
  ```
- **Branches:**
  - Rank >= target → empty command, `StatusSuccess`.
  - Training context "combat" + auto-aggro mob in current room → `attack <mob>`, `StatusRunning`.
  - Training context "combat" + no fight available → `wander`, `StatusRunning`.
  - Training context "crafting" + at a station with a known recipe of this skill → `craft <some_recipe_for_this_skill>` (pick lowest-skill_minimum known recipe). `StatusRunning`.
  - Training context "crafting" + not at station → `pathto <station_room>`, `StatusRunning`.
  - Training context "foraging" → `forage`, `StatusRunning`.
  - Training context "social" → `emote <random_social_emote>` if anyone in room else `wander`, `StatusRunning`.
  - Training context "skullduggery" → `wander`, `StatusRunning` (no autonomous theft attempts in 4.4 — too easy to misfire; needs PvE design later).
  - Unknown skill name in table → empty command, `StatusFailure`.

### 3.12 mastery-equip

- **Purpose:** Upgrade a specific equipment slot to rarity tier ≥ target.
- **Read state:** mob's equipment in the named slot (via 4.3 catalog's `mobSlotItem`), shop stock for slot-appropriate items in zone, `mob.Character.Gold`.
- **MiscData keys:** `plan:mastery-equip:target_shop_room` (int).
- **Branches:**
  - Current slot meets tier → empty command, `StatusSuccess`.
  - Shop in zone has slot-appropriate item ≥ target tier + sufficient gold + at that shop → `buy <appropriate_item>` then on success `wear <item>` (or `wield` for weapons) — two-tick sequence: emit `buy` this tick (`StatusRunning`), next tick the inventory check sees the new item and emits `wear`/`wield`.
  - Same conditions but not at shop → resolve `target_shop_room` via `findShopInZoneSelling` (filtered to slot-appropriate items). Emit `pathto <room>`, `StatusRunning`.
  - Shop exists but insufficient gold → empty command, `StatusFailure` (4.5 might seed a wealth-gold reactively; for now drop through).
  - No shop in zone has slot-appropriate items at tier → empty command, `StatusFailure`.

### 3.13 visit-zone

- **Purpose:** Visit a named zone for the first time.
- **Read state:** `mob.VisitedZones`, `mob.Character.Zone`, zone-graph adjacency (via `zoneAdjacentTo` helper — lazy-computed from inter-zone room exits, cached).
- **MiscData keys:** `plan:visit-zone:next_hop_zone` (string — intermediate zone if multi-hop traversal needed).
- **Branches:**
  - Already visited target zone (predicate satisfied) → empty command, `StatusSuccess`.
  - In target zone → empty command, `StatusSuccess` (room-change hook from 4.3 sets `VisitedZones[target_zone] = true` next tick).
  - Adjacent to target zone → find the exit room in mob's current zone that leads into target zone; emit `pathto <exit_room>`, `StatusRunning`. (Walking into the exit naturally transitions the mob — the existing room-change pipeline handles it.)
  - Multi-hop required → resolve `next_hop_zone` via a simple "any unvisited adjacent zone in the direction of target" heuristic (4.4 ships the heuristic; full BFS over zone graph deferred). Emit `pathto <exit_room_toward_next_hop>`, `StatusRunning`.
  - No known path → empty command, `StatusFailure`.

---

## 4. Supporting infrastructure

Helpers in `internal/planners/helpers.go` (introduced as needed; not all 13 planners use all helpers):

### 4.1 `findShopInZoneSelling(mob, tagOrItemId) (roomId int, ok bool)`

Walks `shops.InZone(mob.Character.Zone)` (or the equivalent existing API — verify via codegraph during implementation), filters to shops carrying the target item (by tag or id), returns the first match's room id. Cached per-tick is overkill; called once per planner invocation.

### 4.2 `findShopInZoneBuying(mob, item *items.Item) (roomId int, ok bool)`

Used by `wealth-gold`. Filters shops to those that BUY (i.e., have gold reserves) AND are interested in the item type — most shops buy most things, so this collapses to "first shop with gold > 0". Verify shop API for the right filter.

### 4.3 `findFactionMemberInZone(mob, factionId, mustBeInCombat bool) (target *mobs.Mob, ok bool)`

Extends 4.3 catalog's `factionMembersInZone` (currently in `revenge_faction.go`). Walks `mobs.GetAllMobInstanceIds`, filters by same zone + `factions.FactionsForMob` membership, optionally filters by `target.Character.Aggro != nil`. Returns the first match. **Note:** if the catalog version was extracted to `internal/goals/catalog/helpers.go` during 4.3, the planner version may need to re-export or duplicate — they live in different packages.

### 4.4 `findHostileInZone(mob) (target *mobs.Mob, ok bool)`

Walks zone mob instances; returns first with `AutoAggro == true`. Used by `protection-faction`.

### 4.5 `findCraftingStationInZone(mob, stationName string) (roomId int, ok bool)`

Walks `rooms` in mob's zone, filters by station tag (rooms have a `Station` or similar field — verify via codegraph). Returns the first matching room's id.

### 4.6 `zoneAdjacentTo(zoneA string) []string` (cached)

Lazy-computed from inter-zone room exits. On first call for a zone, walks rooms in that zone, looks at each exit's destination room, collects the unique destination zones. Result cached in a package-level `map[string][]string`. Cache invalidation: not needed at runtime (room graph is static after boot).

### 4.7 `pickGiftItemFromInventory(mob, target) *items.Item`

Heuristic: highest-value non-equipped non-quest item. Quest items defined by `ItemSpec.QuestToken != ""` (verify field). Returns nil if no suitable item. Used by `befriend`.

### 4.8 `pickRandomExit(mob) string`

Returns a random exit direction name from mob's current room. Used by `survival`'s flee branch.

### 4.9 `pickSocialEmote() string`

Returns a random emote from a small hand-authored list: `nods`, `bows`, `smiles`, `waves`, `grins`. Used by `mastery-skill`'s social branch + befriend's interaction rotation.

### 4.10 `pickKnownRecipeForSkill(mob, skillName) string`

Walks `mob.Character.KnownRecipes`, filters to recipes whose `Skill == skillName`, returns the one with the lowest `SkillMinimum` (least risky for training). Used by `mastery-skill` and `craft-item` when picking what to craft.

---

## 5. Btree integration: per-archetype edit guidance

The `try_goal_planner` action gets inserted at an author-chosen position in each archetype's tree. **Per-archetype placement decisions are part of the implementation tasks** — implementer reviews each tree and picks the position. General guidance:

### 5.1 Combat archetypes

`ambusher`, `combat_passive`, `defensive_caster`, `generic_fighter`, `melee_self_buff`, `predator`, `pure_caster`, `scout`, `support_caster`, `tank_taunter`, `leader`, `prey`.

- Insert AFTER reflex reactions (flee-if-alpha-dead, pack-cohesion, panic-flee).
- Insert BEFORE wander/idle.
- Insert BEFORE existing combat-engagement actions ONLY when the planner's goal-driven combat is expected to dominate (revenge-mob, protection-mob) — typically place it AFTER the basic `try_combat` so default reactive combat still works.

### 5.2 Forager archetype

- Insert BEFORE the forage selector loop — a survival or wealth-item goal should preempt routine forage.
- Insert AFTER any patrol step (`try_patrol`) — patrol movements are the routine layer; goals fire when patrol is idle.

### 5.3 Noncombatant archetypes

`noncombat_shopkeeper`, `noncombat_passive`, `noncombat_questgiver`.

- Insert near the top of the tree. These archetypes are mostly idle today; goal-driven behavior IS their primary action.
- Shopkeeper specifically: insert AFTER any schedule step (shopkeepers may have schedules for "open hours" vs "closed hours").

### 5.4 Patrol-driven archetypes

`lookout`, possibly others that grow patrols later.

- Insert AFTER patrol step. Patrol is their routine; goals fire when patrol is paused (combat interrupt, dwell waypoint).

### 5.5 Boss archetypes

`boss_chrysalis_phantom`, `boss_edrin`, `boss_rhett`, `boss_soren`, `boss_sylara`.

- **Skip entirely.** Bosses have no seeded defaults (4.3 §5.1). Inserting `try_goal_planner` would always return Failure (no current goal). Harmless but pointless. If a future boss-specific encounter wants goal-driven phasing, the boss archetype's tree can be edited at that time.

---

## 6. Plan-state cleanup

`goals.Recompute` already persists the new current goal + emits the `goals.switch` debug log on switch. 4.4 extends it with one more side effect: invoke the registered `ClearPlanState` callback.

### 6.1 The cleanup function

```go
// internal/planners/state.go
const PlanKeyPrefix = "plan:"

func ClearPlanState(mob *mobs.Mob) {
    if mob == nil || mob.Character.MiscData == nil {
        return
    }
    for k := range mob.Character.MiscData {
        if strings.HasPrefix(k, PlanKeyPrefix) {
            delete(mob.Character.MiscData, k)
        }
    }
}
```

### 6.2 Recompute extension

In `internal/goals/store.go`'s `Recompute`, after the existing log emission:

```go
// Existing log line ...
mudlog.Debug("goals.switch", ...)

// chunk 4.4: wipe any planner intermediate state so the new goal's
// planner starts clean. Best-effort; nil callback (tests, ungated boot)
// is fine.
if planStateClear != nil {
    planStateClear(mob)
}
```

Where `planStateClear PlanStateClearFn` is the registered callback set via `SetPlanStateClear`. Lives in `internal/goals/lookup.go` alongside `weightsLookup` and `archetypeDefaultsLookup`.

### 6.3 Cleanup correctness invariants

- ClearPlanState runs AFTER persistence and log emission so any failure inside it doesn't compromise the goal switch itself.
- Per-mob — only the mob whose Recompute switched gets its plan state wiped. Other mobs' plan state untouched.
- Non-plan MiscData keys (e.g., `faction_kills_inflicted:`, `conversation_line_idx`, schedule keys) are left alone — the prefix filter is exact.
- If the callback panics: wrap in a panic-recovery `defer/recover` (mirrors 4.2's `invokeContextScore` pattern). Logs a warning + returns; the goal switch already happened so no rollback needed.

---

## 7. Edge cases

| # | Case | Behavior |
|---|---|---|
| 1 | Planner panics | Wrapped in panic recovery inside `try_goal_planner` action (mirrors 4.2's ContextScore pattern). Logs `planners.plan panic` with goal type + mob id. Returns `StatusFailure` for that tick. Mob's btree falls through; no crash. |
| 2 | `mob.Command(cmd)` itself fails (malformed command, target gone) | Existing command path's error handling fires (typically silent log + no-op). Planner's `StatusRunning` is still propagated to the btree; next tick the planner re-evaluates and either picks a different action or returns Failure. |
| 3 | Two mobs of the same template tick same round (4.2 shared-template limitation) | Both Recompute calls invoke their planner with the same `*Goal` pointer. Planners read mob-specific state (HP, room, inventory), so per-instance behavior diverges naturally. The `plan:*` MiscData IS per-instance (lives on `mob.Character.MiscData`, not on the template), so plan state stays per-mob. |
| 4 | Recompute fires twice quickly (Add + tick in same round) | First Recompute either switches (clears plan state) or doesn't. Second Recompute sees the post-first state; if no switch happened, plan state is preserved. Idempotent. |
| 5 | Planner returns command that no-ops (e.g., `attack` with no target name when target left between tick start and command resolution) | Same as case 2 — command path silently no-ops. Next tick the planner re-evaluates. |
| 6 | Mob goal switches in the middle of a multi-tick `pathto` (e.g., during a 5-room path the goal switches to survival because HP dropped) | Recompute's switch fires → ClearPlanState wipes `plan:wealth-gold:target_shop_room`. New `survival` planner activates on the next tick — first action is `flee` since mob is in combat. Pathto is naturally abandoned (mob's `pathto` queue is a separate mechanism; verify whether it persists across goal switches — if so, may need an additional cleanup step or accept that pathto-queue clears on the next pathto invocation). |
| 7 | Planner registered for a type that has no current implementation (registration races init order) | Go init order between packages is deterministic per import-graph; `_ "internal/planners"` blank-import fires after `goals` package is initialized. Within the planners package, init order between files is undefined but every file calls `RegisterPlanner` which is idempotent at the map level. No race. |
| 8 | Admin manually adds a goal with no registered planner (4.3 catalog has the type but 4.4 forgot a planner) | `try_goal_planner` looks up via `LookupPlanner`, finds nil, returns `StatusFailure`. Btree falls through. No crash. Admin can debug via `goal current <mob>` to see the goal is selected but unplanned. |
| 9 | Planner produces output for a mob that's in a combat-interrupted state where the command would be rejected (e.g., trying to `craft` while in combat) | Same as case 2 — command path's existing guards (combat-blocks-crafting etc.) silently reject. Planner re-tries next tick; eventually conditions change or planner exits to Failure. |
| 10 | `mob.Character.MiscData` is nil when ClearPlanState fires | Defensive nil check at top of ClearPlanState; no-op return. (Existing Character constructor lazy-inits MiscData on first write, so this is rare but possible for freshly-spawned mobs whose first goal switch precedes any MiscData write.) |
| 11 | A planner's `pathto` is to a room that becomes unreachable mid-walk (e.g., a door closes) | The existing `pathto` mechanism handles this (drops the queue with a log). Next planner tick re-resolves `target_*_room` from MiscData or recomputes from scratch. Plan adapts. |
| 12 | Mob enters target zone mid-plan for visit-zone (e.g., follows another mob through an exit) | The room-change hook (4.3) sets `VisitedZones[zone] = true`. Next planner tick the predicate satisfies (returns true from `mobHasVisited(target_zone)`); planner returns `StatusSuccess`. ClearPlanState fires on the next goal-switch (when 4.2 picks a different goal). |
| 13 | All 13 planners simultaneously needed by different mobs in the same tick | Each `try_goal_planner` invocation is independent; no shared state across mob instances except the registry (read-only after init). Performance: ~200 mobs × pure-Go planner func ≈ negligible per round at MUD scale. |
| 14 | Planner writes to MiscData but mob YAML serializer doesn't know about the `plan:` keys | MiscData is `map[string]any` — serializer handles any keys via reflection / yaml.v3's interface handling. Plan keys persist via `mobs.instances/<zone>/<id>.yaml` along with the rest of MiscData. (Verify the serializer doesn't skip unknown keys — likely fine since the field is `omitempty` on the map level, not per-key.) On server restart, the keys load back into MiscData; next planner tick uses them naturally. If a goal switch happened between save and restart that should have cleared them — accept this small staleness window (the planner's defensive checks for "is this state still valid?" already cover it). |

---

## 8. Testing strategy & rollout

### 8.1 Per-planner unit tests

`internal/planners/<type>_test.go` — one file per planner. Each covers:

- Registration: `LookupPlanner(type)` returns non-nil after package init.
- Happy path: one fixture per terminal branch (e.g., for survival: in-combat-flee, low-hp-drink-potion, low-hp-rest, recovered-success).
- MiscData round-trip for stateful planners (sticky shop, befriend cooldown, etc.) — write key, re-invoke planner, observe the read.
- Failure paths: target out of zone, no shop available, no faction members in zone, etc. — planner returns `StatusFailure` with empty command.

Estimated 8-15 tests per planner × 13 planners = ~150 tests.

### 8.2 Framework tests

`internal/planners/planners_test.go`:

- `LookupPlanner` returns nil for unregistered types.
- `RegisterPlanner` overwrites on duplicate (last-write-wins for test override).

`internal/planners/state_test.go`:

- `ClearPlanState` wipes `plan:*` keys.
- `ClearPlanState` leaves other MiscData keys untouched (e.g., `faction_kills_inflicted:bandits`, `conversation_line_idx`).
- `ClearPlanState` is nil-safe on `mob == nil` and `MiscData == nil`.

### 8.3 Btree action test

`internal/behaviortree/actions_goal_test.go`:

- `try_goal_planner` with no current goal → `StatusFailure`.
- `try_goal_planner` with current goal but no registered planner → `StatusFailure`.
- `try_goal_planner` invokes the planner, propagates `Status`, executes `Command` via mock mob command spy.
- Planner panic → recovered, `StatusFailure`, warn log fired.

### 8.4 Recompute integration test

`internal/goals/store_test.go` (extend):

- Switch fires registered `ClearPlanState` callback.
- No callback registered → switch still works, no error.
- Multiple consecutive switches: each fires the callback once.

### 8.5 Boot smoke (per pre-push SOP)

- Server boots cleanly with all 13 planners registered via blank import.
- `try_goal_planner` action inserted into 18 archetype YAMLs (per §5); each archetype YAML parses cleanly.
- Engineered smoke #1 — survival: damage a test mob in a zone, observe `flee` command in log when HP drops below threshold.
- Engineered smoke #2 — revenge-mob: admin-add `revenge-mob target_kind=player target_id=<admin>` to a mob in the same zone, observe mob path-to admin's room and attack.
- Engineered smoke #3 — wealth-gold: admin-add `wealth-gold target=200` to a thief in a zone with a shop, observe sell-loop firing over several rounds (sell items, gold rises, predicate satisfies).

### 8.6 Out of scope for 4.4 tests

- Reactive seed integration (4.5 will test the wealth-gold-after-buy-failure seed).
- Bulk-stuck-goal pruning sweep (4.6).
- Plan visualization admin command (deferred).

### 8.7 Rollout — feature branch decomposition

Single chunk on `feature/aliveness-4.4-strategic-tactical-translation`. Suggested task ordering for the plan:

1. **Framework**: `PlanFn`, `PlanResult`, `BTreeStatus`, `RegisterPlanner` / `LookupPlanner` registry. Unit tests.
2. **`ClearPlanState` + plan-key prefix convention.** Unit tests.
3. **Goals-side callback registration**: `SetPlanStateClear`, `planStateClear` invocation in `Recompute`. Mirror the 4.2/4.3 lookup pattern.
4. **main.go boot wiring**: register `goals.SetPlanStateClear(planners.ClearPlanState)` + add blank import of `internal/planners`.
5. **Btree action**: `actGoalPlanner` + register `try_goal_planner` action name. Panic-recovered planner invocation. Unit tests.
6. **Supporting helpers** (`helpers.go`): `findShopInZoneSelling/Buying`, `findFactionMemberInZone`, `findHostileInZone`, `findCraftingStationInZone`, `zoneAdjacentTo`, `pickGiftItemFromInventory`, `pickRandomExit`, `pickSocialEmote`, `pickKnownRecipeForSkill`. Each with minimal unit test.
7. **Skill training table** (`skill_training_table.go`): static map literal + lookup helper. Unit test.
8-20. **13 planner tasks** — one per type (survival, wealth-gold, wealth-item, craft-item, revenge-mob, revenge-faction, protection-mob, protection-faction, befriend, befriend-faction, mastery-skill, mastery-equip, visit-zone). Each task: planner Go file + per-branch unit tests + `init()` registration.
21. **Per-archetype YAML edits** — 16-18 YAMLs get `try_goal_planner` inserted at author-chosen position per §5. One commit grouping. Boot smoke after.
22. **`internal/planners/context.md`** (new package docs per project convention).
23. **Smoke checklist + roadmap rollup (25/42 → 26/42) + PATCH_NOTES entry.**

23 tasks total. Comparable shape to 4.3 (23 tasks for L). Per-planner tasks are larger than 4.3's per-type tasks (planners have more branches + state) but still ship one per task.

**Push to prod is safe** — planners run, NPCs visibly pursue goals, but no schema change. The MiscData additions are additive (existing files load fine; new keys added on-demand). Per-archetype YAML edits are additive. Roll-back option: revert the per-archetype edits to disable `try_goal_planner` per archetype while leaving the planner code in place.

### 8.8 Roadmap rollup after 4.4 ships

**26/42.** Next: 4.5 Reactive goal generation (event hooks seed new goals from world state, including the kill-counter that feeds revenge-faction and the materials-missing → wealth-item seeding referenced in craft-item) → 4.6 Goal satisfaction & pruning (sweep loop that calls per-type Predicates in bulk + detects permanent-stuck goals).

---

## File touch list

**New:**
- `internal/planners/` subpackage:
  - `context.md` — package overview (per project convention).
  - `planners.go` — `PlanFn`, `PlanResult`, `BTreeStatus`, registry + lookup.
  - `state.go` — `ClearPlanState`, `PlanKeyPrefix`.
  - `helpers.go` — shared subsystem adapters (~10 functions).
  - `skill_training_table.go` — per-skill heuristic dispatch.
  - 13 planner files (`survival.go`, `wealth_gold.go`, etc.) + 13 test files.
- `internal/behaviortree/actions_goal.go` — `actGoalPlanner` + `try_goal_planner` action registration.
- `internal/behaviortree/actions_goal_test.go`.
- `internal/planners/planners_test.go`, `state_test.go`, `helpers_test.go`.

**Modified:**
- `internal/goals/lookup.go` — add `PlanStateClearFn` type + `SetPlanStateClear` + internal `planStateClear` var.
- `internal/goals/store.go` — `Recompute` invokes `planStateClear` callback on switch (under panic recovery, after persistence + log).
- `internal/goals/store_test.go` — extend with callback-invocation tests.
- `main.go` — register `goals.SetPlanStateClear(planners.ClearPlanState)` + add blank import of `internal/planners`.
- `_datafiles/world/dogmud/behaviors/archetypes/*.yaml` — insert `try_goal_planner` into 16-18 archetype trees per §5 guidance. (Boss archetypes skipped.)
- `MOB_ALIVENESS_ROADMAP.md` — flip 4.4 to Done, bump rollup to 26/42, upsize L → XL.
- `PATCH_NOTES.md` — chunk 4.4 entry.

**Not touched in 4.4:** `internal/goals/select.go` (4.2 unchanged), `internal/goals/catalog/` (4.3 catalog unchanged); behavior-tree actions other than the new goal action; schedules; patrols; conversations; the existing gossip system.
