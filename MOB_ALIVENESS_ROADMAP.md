# DOGMud — Mob Aliveness Roadmap

> Living document. Update chunk **Status** fields as work lands. Add new chunks
> as discovery happens. Lifespan: until mob behavior and mob/player parity are
> at a point we're satisfied with — likely many months.

## What this is

A long-term plan to make DOGMud's NPCs feel alive: remembering players,
forming opinions, holding grudges, pursuing goals, reacting to crimes, hunting
wanted targets, and closing the verb gap with players. The doc decomposes the
work into chunks and orders them so we build foundations before consumers and
vocabulary before planners.

This is a **roadmap, not a spec**. Each chunk gets a mini-brief — enough to
remember what it is and why we want it. Concrete specs and implementation
plans live in `docs/superpowers/specs/` and `docs/superpowers/plans/` and are
created chunk-by-chunk as we pick them up.

## The framing — three layers + a substrate

NPC behavior decomposes into three layers asking three different questions,
sitting on top of a state substrate they all consume.

| | Question | Current state in DOGMud |
|--|----------|-------------------------|
| **Strategic** | "What do I *want*?" | **Absent.** No goals, no drives. |
| **Routine** | "What do I do *regularly*?" | **Partial.** Foragers, caravans, idle wander, basic patrols. No schedules, no day/night. |
| **Tactical** | "What do I do *right now*?" | **Strong.** Behavior trees, archetypes, combat decisions. |

| | What | Current state |
|--|------|---------------|
| **Substrate** | Memory, disposition, faction, knowledge, world facts | **Thin.** Combat grudges + dialogue mood/visit-count only. |

Memory isn't a layer because it doesn't *do* anything — it's data the three
layers read from and write to. A guard's tactical "attack on sight" reaction
reads faction state. A merchant's strategic "save up for better stock" goal
reads inventory state. Same store, different consumers.

**Phase ordering principle:** build substrate before consumers, build
vocabulary before planners. You can't reason about "save up for better armor"
if the tactical layer can't compare two pieces of armor and the substrate
can't track gold-saving intent.

## How to read this doc

**Status values:**
- `Not started` — design pending
- `In progress` — actively being specced or built
- `Done` — shipped
- `Blocked` — waiting on dependency or external decision
- `Cancelled` — explicitly dropped

**Size scale (rough effort, not time):**
- `S` — small, contained, a few files
- `M` — moderate, multiple files, some integration work
- `L` — large, multiple subsystems touched, design choices required
- `XL` — very large; may itself need decomposition before specing

**Reading order:** phases run roughly in dependency order, but execution can
pull chunks forward across phases when dependencies are satisfied. Mid-Phase 1
is a fine time to grab a Phase 2 quick-win (e.g., 2.6 cast preemption) for
variety, as long as the dependency arrow allows.

**Suggested starting point:** **Chunk 1.1** — persistent NPC opinion store.
No dependencies, foundational, unlocks the highest number of follow-on
chunks.

---

## Progress tracker

Update **Status** here AND in the chunk's mini-brief as work moves. Both
should always agree.

| Chunk | Phase | Title | Size | Depends on | Status |
|-------|-------|-------|------|-----------|--------|
| 1.1 | Substrate | Persistent NPC opinion store | M | — | Done |
| 1.2 | Substrate | Faction system | L | 1.1 | Not started |
| 1.3 | Substrate | Crime/wanted state | M | 1.2 | Not started |
| 1.4 | Substrate | NPC knowledge model | M | 1.1 | Not started |
| 1.5 | Substrate | Bounty state | S | 1.2 | Not started |
| 1.6 | Substrate | NPC-to-NPC relationships | M | — | Not started |
| 1.7 | Substrate | World-model facts | M | 1.4 | Not started |
| 2.1 | Tactical | Mob `buy` command | M | — | Not started |
| 2.2 | Tactical | Item-comparison primitive | M | — | Not started |
| 2.3 | Tactical | Equip-if-better behavior | S | 2.2 | Not started |
| 2.4 | Tactical | Mob `appraise` / `assess` | S | 2.2 | Not started |
| 2.5 | Tactical | Mutations on mobs | L | — | Not started |
| 2.6 | Tactical | Tactics-cast preemption fix | S | — | Not started |
| 2.7 | Tactical | Mob skullduggery suite | M | — | Not started |
| 2.8 | Tactical | Mob scout / track / scan | S | — | Not started |
| 2.9 | Tactical | Mob `forage` as a command | S | — | Not started |
| 2.10 | Tactical | PvM/MvP/PvP/MvM parity audit | M | 2.1–2.9 | Not started |
| 3.1 | Routine | Game-time hook | S | — | Not started |
| 3.2 | Routine | NPC schedules | L | 3.1 | Not started |
| 3.3 | Routine | Sleeping / wake states | S | 3.1 | Not started |
| 3.4 | Routine | Waypoint patrols | M | — | Not started |
| 3.5 | Routine | Maintenance routines | M | 3.2 | Not started |
| 3.6 | Routine | NPC↔NPC idle conversation | M | 1.6 | Not started |
| 4.1 | Strategic | Goal representation | M | 1.1, 1.4 | Not started |
| 4.2 | Strategic | Goal selection | L | 4.1 | Not started |
| 4.3 | Strategic | Goal types catalog | M | 4.1 | Not started |
| 4.4 | Strategic | Strategic→tactical translation | L | 4.3, Phase 2 | Not started |
| 4.5 | Strategic | Reactive goal generation | M | 1.6, 4.1 | Not started |
| 4.6 | Strategic | Goal satisfaction & pruning | S | 4.1 | Not started |
| 5.1 | Cross-cut | Town justice | XL | 1.2, 1.3, 1.5, 3.4, Phase 4 | Not started |
| 5.2 | Cross-cut | Bounty hunting | L | 1.4, 1.5, 2.8, 4.4 | Not started |
| 5.3 | Cross-cut | Equipment-aware shopping | L | 2.1, 2.2, 2.3, 4.4 | Not started |
| 5.4 | Cross-cut | NPC market participation | M | 5.3 | Not started |
| 6.1 | Polish | Stillwater town-flavor pass | L | Phase 1, Phase 3 | Not started |
| 6.2 | Polish | Parity audit closeout | S | 6.1 | Not started |
| 6.3 | Polish | Per-zone tuning (1–2 zones) | M | 6.1 | Not started |
| 6.4 | Polish | Performance review (initial) | S | 6.3 | Not started |
| 6.5 | Polish | Content pass — broader rollout | XL | 6.3 | Not started |
| 6.6 | Polish | Performance re-review | S | 6.5 | Not started |

**Roll-up:** 1 / 39 done • 0 in progress • 38 not started.

---

## Phase 1 — Substrate

State primitives the rest of the layers read from and write to.

### 1.1 Persistent NPC opinion store
**Status:** Done (2026-05-06) • **Size:** M

- **Goal:** Per-NPC × per-player disposition score that persists across spawns, deaths, and server restarts.
- **In:** Storage schema, read/write API, decay rules, admin debug command, integration points for combat/dialogue/quest systems to mutate scores.
- **Out:** Player-facing visibility (deferred), per-faction roll-up (covered by 1.2).
- **Depends on:** —
- **Why:** Foundation. Without this, "the merchant remembers you cheated him last week" is impossible. Underlies most of Phase 4 and Phase 5.
- **Shipped:** `internal/opinions/` package with signed-scalar score [-100, +100], per-NPC YAML at `_datafiles/world/dogmud/opinions/{mobId}-{namesimple}.yaml`, lazy decay toward per-NPC default, public API (Get/Set/Bump/TierFor), admin command `opinion show/set/bump/reset`, helpfile, combat hookup on first-aggression in `attack`/`target`. Spec at `docs/superpowers/specs/2026-05-06-mob-aliveness-1.1-opinion-store-design.md`, plan at `docs/superpowers/plans/2026-05-06-mob-aliveness-1.1-opinion-store.md`.

### 1.2 Faction system
**Status:** Not started • **Size:** L

- **Goal:** Faction definitions, NPC membership, per-player reputation per faction.
- **In:** Faction YAML, NPC `faction` field, per-player rep store, rep-change API, faction-vs-faction relations (allies/enemies), admin inspection commands.
- **Out:** Faction-specific quests (content-pass time), full citizenship UI (future).
- **Depends on:** 1.1 (shares store backend)
- **Why:** Replaces the `peacefulquest` placeholder. Enables "Stillwater militia hates you because you killed one of theirs."

### 1.3 Crime/wanted state
**Status:** Not started • **Size:** M

- **Goal:** Per-player log of unresolved crimes (theft, assault, murder) keyed by zone or faction.
- **In:** Crime types, witness tracking, zone/faction scoping, expiry rules, query API, admin debug.
- **Out:** Guard reactions — that's 5.1.
- **Depends on:** 1.2
- **Why:** "I assaulted a Stillwater citizen — Stillwater knows it" requires this data. Town justice and bounty hunting both consume it.

### 1.4 NPC knowledge model
**Status:** Not started • **Size:** M

- **Goal:** What facts does this NPC know about player X — name learned, last-seen room, deeds witnessed, items seen carried.
- **In:** Knowledge schema, learn/forget API, perception-gated learning (NPCs only learn what they witness or are told), query API for tactical/strategic layers.
- **Out:** World-level facts (1.7).
- **Depends on:** 1.1
- **Why:** Lets an NPC say "I saw you with the bandit chief's coat — turn it in." Without this, NPCs are amnesiacs even if 1.1 tells them their feeling.

### 1.5 Bounty state
**Status:** Not started • **Size:** S

- **Goal:** Declared bounties (payer, target, reward, conditions, expiry) queryable by mobs and players.
- **In:** Bounty data structure, declaration API (faction-driven, quest-driven, NPC-driven), claim/resolution API, admin commands.
- **Out:** Bounty board UI for players (could be a follow-on player-facing affordance), bounty hunter behavior (5.2).
- **Depends on:** 1.2
- **Why:** Enables 5.2 bounty hunting, escalation in town justice, faction-driven contracts.

### 1.6 NPC-to-NPC relationships
**Status:** Not started • **Size:** M

- **Goal:** Kinship and friendship graph between NPCs (Voss is Lars's brother; Marta is the smith's wife).
- **In:** Relationship types (family, friend, rival, lover, employer/employee), per-NPC relationship list, query API, mutation API.
- **Out:** Relationship change as a player-facing mechanic (a romance system is way out of scope).
- **Depends on:** —
- **Why:** Killing one NPC seeds revenge goals in their kin. The world starts to feel woven, not flat.

### 1.7 World-model facts
**Status:** Not started • **Size:** M

- **Goal:** Zone-level or world-level facts NPCs can "know" (the bridge collapsed, the bandit camp moved, the king is dead).
- **In:** Fact schema, fact declaration API, NPC awareness-of-fact tracking (some know, some don't), propagation rules (gossip).
- **Out:** Dynamic fact generation from world events — start with author-declared facts.
- **Depends on:** 1.4
- **Why:** Makes the world feel like it has news, rumors, and shared context — not just isolated NPC bubbles.

---

## Phase 2 — Tactical fill-in

Verbs and behavior-tree gaps that the strategic layer will need to dispatch.
Build vocabulary before the planner.

### 2.1 Mob `buy` command
**Status:** Not started • **Size:** M

- **Goal:** Mobs can purchase from shops, including disambiguation, gold checks, carry capacity.
- **In:** Mobcommand `buy`, integration with existing shop pricing/stock, restocking interaction with NPC-buyer behavior.
- **Out:** Decision logic for *what* to buy — that lives in tactical/strategic.
- **Depends on:** —
- **Why:** Strategic-layer "save up for armor" is impossible without this verb.

### 2.2 Item-comparison primitive
**Status:** Not started • **Size:** M

- **Goal:** Callable function: "is item A an upgrade over item B for this mob?"
- **In:** Multi-axis comparison (damage, mitigation, weight, slot conflicts, archetype-fit), per-archetype weighting, returns a score so callers can rank a list.
- **Out:** Action — that's 2.3.
- **Depends on:** —
- **Why:** Underlies all "smart equipping" and "smart shopping." Without it, mobs can't tell good gear from bad.

### 2.3 Equip-if-better behavior
**Status:** Not started • **Size:** S

- **Goal:** Tactical behavior: on loot pickup or item-give, evaluate and equip if it beats the current slot occupant.
- **In:** Btree action, per-archetype configurable, emits emote when swapping.
- **Out:** —
- **Depends on:** 2.2
- **Why:** "I gave the bandit a steel sword and he's still using a club" → fixed.

### 2.4 Mob `appraise` / `assess`
**Status:** Not started • **Size:** S

- **Goal:** Mobs can assess players (combat capability, equipped gear quality) and items.
- **In:** Mobcommand wrappers around existing player commands, callable from btree.
- **Out:** —
- **Depends on:** 2.2 (for item assessment)
- **Why:** Lets NPCs decide to flee a strong player or covet a player's weapon.

### 2.5 Mutations on mobs
**Status:** Not started • **Size:** L

- **Goal:** Companion Phase 5 — mutations apply to mobs the way they do to players.
- **In:** Mob mutation slots, YAML schema, runtime application of mutation effects (extra arms, tail, etc.), combat integration, scaling.
- **Out:** Player-facing UI for mutations on companions/mobs (separate concern).
- **Depends on:** —
- **Why:** Closes a major parity gap. Mutated mobs are a content lever for novel encounters. (Absorbed from MEMORY.md — Companion Phase 5.)

### 2.6 Tactics-cast preemption fix
**Status:** Not started • **Size:** S

- **Goal:** Resolve the existing bug where higher-priority cast tactics lose `CastingState` to lower-priority casts that started in the same combat tick.
- **In:** Surface `AlreadyCasting` / `OnCooldown` exits in `mobcommands/cast.go`; reorder tactic resolution.
- **Out:** —
- **Depends on:** —
- **Why:** Smarter mobs are useless if their tactics get clobbered. (Absorbed from MEMORY.md.)

### 2.7 Mob skullduggery suite
**Status:** Not started • **Size:** M

- **Goal:** Mob versions of `steal`, `pickpocket`, `sneak`, `hide`, `plant` (and maybe `defuse`/`shadow`).
- **In:** Mobcommands for each, behavior tree integration, archetype tagging (only thieves/scouts use these).
- **Out:** —
- **Depends on:** —
- **Why:** Bandit NPCs that can't pickpocket aren't bandits. Major aliveness lever.

### 2.8 Mob scout / track / scan
**Status:** Not started • **Size:** S

- **Goal:** Information-gathering verbs available to mobs.
- **In:** Mobcommands for `track`, `scan`, `search`, `consider`.
- **Out:** —
- **Depends on:** —
- **Why:** Bounty hunters and patrols need to find things. Quest-NPC scouts feel passive without these.

### 2.9 Mob `forage` as a command
**Status:** Not started • **Size:** S

- **Goal:** Promote forage from routine-only behavior to a callable verb.
- **In:** Mobcommand wrapper around existing forage skill, btree integration.
- **Out:** —
- **Depends on:** —
- **Why:** Strategic NPCs that decide "I'm out of leather, let me forage" need the verb.

### 2.10 PvM/MvP/PvP/MvM parity audit
**Status:** Not started • **Size:** M

- **Goal:** Sweep remaining parity gaps after 2.1–2.9 land.
- **In:** Walk every player command, classify it (mob-equivalent / orthogonal / never-relevant), patch concrete gaps.
- **Out:** Forced parity for every player command — only what's relevant to mob behavior.
- **Depends on:** 2.1–2.9 (do the obvious ones first, then audit the long tail)
- **Why:** Catches verbs we didn't think to add. (Absorbed from MEMORY.md.)

---

## Phase 3 — Routine layer

Scheduled and repeating procedural behaviors. Adds time-of-day texture to the
world.

### 3.1 Game-time hook
**Status:** Not started • **Size:** S

- **Goal:** Expose time-of-day to behaviors (clock primitive — extend the existing time system if needed).
- **In:** Time tick, day/night flag, configurable day length, btree condition `time_of_day_is`.
- **Out:** Visible time-of-day UI for players (could come with content pass).
- **Depends on:** —
- **Why:** Without time, schedules are meaningless. Cheap foundation.

### 3.2 NPC schedules
**Status:** Not started • **Size:** L

- **Goal:** Timed routines: "smith works 9–5, home 5–8, tavern 8–11, sleep."
- **In:** Schedule YAML, schedule executor, behaviors for "go to room" and "perform activity at room."
- **Out:** Per-day variation (weekday/weekend/holiday) — start with single daily routine.
- **Depends on:** 3.1
- **Why:** A town that empties at night and fills in the morning feels a thousand percent more alive than a static town.

### 3.3 Sleeping / wake states
**Status:** Not started • **Size:** S

- **Goal:** NPCs visibly asleep at night; wakeable by sound, light, attack.
- **In:** Sleeping condition, room descriptions for sleeping NPCs, wake triggers, combat-on-sleeper consequences (more crime severity).
- **Out:** —
- **Depends on:** 3.1
- **Why:** A sleeping NPC is a tiny piece of fiction that compounds well — assassinations, theft, pickpocket-while-sleeping.

### 3.4 Waypoint patrols
**Status:** Not started • **Size:** M

- **Goal:** Looped multi-room routes with optional dwell times.
- **In:** Patrol route YAML, executor, interrupt-handling (combat aborts patrol; resume after).
- **Out:** Dynamic re-routing when paths blocked (start with hard-failure on blocked path).
- **Depends on:** —
- **Why:** Guards that actually walk a beat. Town justice (5.1) consumes this.

### 3.5 Maintenance routines
**Status:** Not started • **Size:** M

- **Goal:** Smith repairs gear, farmer tends crops, librarian shelves books — flavor activity tied to NPC role.
- **In:** Activity YAML, emote-driven flavor, optional integration with crafting (smith actually crafts inventory restock).
- **Out:** Activities producing real economic output (crafting restock can be a follow-on chunk).
- **Depends on:** 3.2 (maintenance often slots inside schedules)
- **Why:** Walking into the smithy and seeing the smith working tells the player the world isn't waiting on them.

### 3.6 NPC↔NPC idle conversation
**Status:** Not started • **Size:** M

- **Goal:** NPCs occasionally talk to each other (canned exchanges, mood-aware).
- **In:** Pair-conversation YAML (paired with 1.6 relationships), trigger logic (proximity + cooldown), mood-aware variants.
- **Out:** Player-overheard "spoken about you" gossip (later, ties to 1.4 knowledge spread).
- **Depends on:** 1.6
- **Why:** A guard and a baker chatting in the square is the highest-bang-for-buck aliveness signal.

---

## Phase 4 — Strategic layer

The new "what do I want?" engine. Builds on substrate state and tactical
verbs.

### 4.1 Goal representation
**Status:** Not started • **Size:** M

- **Goal:** Define what a goal is in code — type, target, satisfaction predicate, expiry, priority.
- **In:** Goal struct/interface, registration, persistence, debug command.
- **Out:** Goal selection logic (4.2).
- **Depends on:** 1.1, 1.4 (goals reference state)
- **Why:** Foundation for the strategic layer. Without this, "drives" stay vibes.

### 4.2 Goal selection
**Status:** Not started • **Size:** L

- **Goal:** NPC picks a current goal from a candidate set based on priority, context, and recent state.
- **In:** Selection function, hysteresis (don't goal-thrash), per-archetype weighting.
- **Out:** Multi-goal pursuit (start single-goal-at-a-time).
- **Depends on:** 4.1
- **Why:** The "pick what to want" engine. Without it, NPCs have goals but never act on them.

### 4.3 Goal types catalog
**Status:** Not started • **Size:** M

- **Goal:** Concrete goal types: survival, wealth, revenge, protection, social, mastery.
- **In:** YAML for each type, archetype-default goal sets, parameters per goal.
- **Out:** Player-authored custom goals (deferred).
- **Depends on:** 4.1
- **Why:** Without concrete types, the strategic layer is empty scaffolding.

### 4.4 Strategic→tactical translation
**Status:** Not started • **Size:** L

- **Goal:** Planner that turns a goal into routine/tactical actions (e.g., "save for armor" → walk to shop, check stock, sell loot, save gold, buy when affordable).
- **In:** Per-goal-type planners, fallback to btree, plan-failure recovery.
- **Out:** General-purpose planner (HTN, GOAP) — start with hand-authored per-goal planners.
- **Depends on:** 4.3, Phase 2 verbs
- **Why:** Bridges desire to action. The whole point of the strategic layer.

### 4.5 Reactive goal generation
**Status:** Not started • **Size:** M

- **Goal:** Events seed new goals (player kills NPC's friend → revenge goal seeded into the friend's NPCs).
- **In:** Event hooks (combat death, theft, faction insult), goal-seeding rules, goal deduplication.
- **Out:** —
- **Depends on:** 1.6, 4.1
- **Why:** Goals that *react* to player actions are what make the world feel responsive.

### 4.6 Goal satisfaction & pruning
**Status:** Not started • **Size:** S

- **Goal:** Cleanup — when is a goal done? Prune dead goals so NPCs don't accumulate ghost desires.
- **In:** Per-type satisfaction predicates, expiry, "abandoned" reasons.
- **Out:** —
- **Depends on:** 4.1
- **Why:** Strategic layer hygiene. Without it, performance degrades and NPCs get stuck on unreachable goals.

---

## Phase 5 — Cross-cutting features

Compose layers into player-facing systems. These wait until the layers exist
to compose.

### 5.1 Town justice
**Status:** Not started • **Size:** XL

- **Goal:** Replace `peacefulquest` placeholder with real citizenship + faction guards + crime detection + bounty workflow.
- **In:** Citizenship-by-faction, guard archetypes that react to crimes against citizens, escalation (warn → arrest → kill), bounty placement on offenders, redemption (pay fine, complete quest, etc.).
- **Out:** Per-zone *unique* justice (each zone uses the framework with config tweaks; one-off rules in content pass).
- **Depends on:** 1.2, 1.3, 1.5, 3.4 (patrols), Phase 4 (guards with goals)
- **Why:** The single biggest aliveness leap — the world reacts to player crimes meaningfully. (Absorbed from MEMORY.md — peacefulquest → faction system.)

### 5.2 Bounty hunting
**Status:** Not started • **Size:** L

- **Goal:** NPCs (and NPC archetypes) actively hunt declared-bounty targets — NPC bandits *or* wanted players.
- **In:** Bounty hunter archetype, hunt-goal seeded from 1.5 bounty state, tracking via 2.8, encounter behavior, optional contract acquisition (pick up bounties from boards).
- **Out:** Player-as-bounty-hunter system (player-facing flip side; could come later).
- **Depends on:** 1.4, 1.5, 2.8, 4.4
- **Why:** Bad actors can't safely hide. Wanted players get *chased*, not just yelled at by guards. NPC bad actors get hunted by their own world.

### 5.3 Equipment-aware shopping
**Status:** Not started • **Size:** L

- **Goal:** NPCs save gold and visit shops to buy upgrades, applying 2.2's comparison logic.
- **In:** "Upgrade my X slot" goal, gold-saving behavior, shop-route planning, archetype preferences.
- **Out:** NPCs commissioning custom-craft from player-crafters (maybe later).
- **Depends on:** 2.1, 2.2, 2.3, 4.4
- **Why:** A bandit who keeps the steel sword you dropped and shows up in better gear next time is far more memorable than one in starter rags.

### 5.4 NPC market participation
**Status:** Not started • **Size:** M

- **Goal:** NPCs sell looted/crafted goods through normal shop channels, contributing to the economy.
- **In:** Sell-trigger behavior, integration with shop pricing/stock, decay/clearance rules.
- **Out:** Player↔NPC barter beyond what shop UX already supports.
- **Depends on:** 5.3 (similar plumbing)
- **Why:** Living economy — shop stock reflects NPC activity, not just player drop-offs.

---

## Phase 6 — Audit & polish

Validate the framework against real content, then scale.

### 6.1 Stillwater town-flavor pass
**Status:** Not started • **Size:** L

- **Goal:** First zone benchmark — every Stillwater NPC gets relationships, schedule, knowledge, optional goals.
- **In:** 19 non-quest Stillwater NPCs (per MEMORY.md), idle dialogue, daily routines, mutual relationships.
- **Out:** New quests (content separate).
- **Depends on:** Phase 1, Phase 3, optionally Phase 4
- **Why:** Validates the framework against real content. Catches what's hard to author. (Absorbed from MEMORY.md — Stillwater town-flavor pass.)

### 6.2 Parity audit closeout
**Status:** Not started • **Size:** S

- **Goal:** Final sweep of parity gaps after Stillwater pass exposes what's still missing.
- **In:** Document remaining gaps, log next-tier ones to MEMORY for later.
- **Out:** —
- **Depends on:** 6.1
- **Why:** Captures what we learned from real content use.

### 6.3 Per-zone tuning (1–2 zones)
**Status:** Not started • **Size:** M

- **Goal:** Apply the framework to one or two more zones (e.g., Sanctum Basin, Dustwalk), tuning archetype defaults.
- **In:** Two zone passes, before-and-after notes on what worked.
- **Out:** Every zone — that's 6.5.
- **Depends on:** 6.1
- **Why:** Two zones reveal pattern. Three+ becomes process to delegate.

### 6.4 Performance review (initial)
**Status:** Not started • **Size:** S

- **Goal:** Measure substrate state size, persistence cost, and tick budget after the framework lands.
- **In:** Profile, log key metrics, document baseline.
- **Out:** —
- **Depends on:** 6.3
- **Why:** Can't optimize what we haven't measured. Catch regressions before content pass scales them up.

### 6.5 Content pass — broader rollout
**Status:** Not started • **Size:** XL

- **Goal:** Apply the framework to the rest of the game's zones and NPCs.
- **In:** Every remaining zone, schedule/relationship authoring, validation pass per zone.
- **Out:** —
- **Depends on:** 6.3
- **Why:** Scaling the formula across the world. The "and now actually populate it" step.

### 6.6 Performance re-review
**Status:** Not started • **Size:** S

- **Goal:** Re-profile after content pass — load profile changes once you have many active goal-driven NPCs across many zones.
- **In:** Re-run profiling, compare against 6.4 baseline, optimize hot paths if needed.
- **Out:** —
- **Depends on:** 6.5
- **Why:** Goal engines + persistent state + schedules can compound. Catch degradation while we still have headroom.

---

## Absorbed from MEMORY.md

These items were tracked in MEMORY.md before this roadmap existed. Each fits
the aliveness/parity goal and has been folded into a chunk above. They should
come off MEMORY.md's tracked-work tables when this roadmap is committed.

| MEMORY item | Absorbed as |
|-------------|-------------|
| peacefulquest → faction system | 1.2 + 5.1 |
| Companion Phase 5 (mutations + UI) | 2.5 (mutations); UI piece deferred to follow-on |
| Tactics-cast preemption gap | 2.6 |
| PvM/MvP/PvP/MvM parity gaps | 2.10 |
| Stillwater town-flavor pass | 6.1 |

Items deliberately *not* absorbed (they don't move the aliveness/parity needle): type-aware equip dispatch, recipe-aware craft dispatch, tutorial content refresh, active-command crafting audit, zone spawn pacing, lint modernization sweep, follow timer drop, steal gate ordering. Those stay in MEMORY.md.

## Future work / explicitly out of scope

Recorded so we don't forget, but not part of this roadmap:

- **LLM-driven dynamic dialogue.** The prod droplet is too small to host it well. Reconsider only if hosting changes.
- **Player notoriety as a worldwide mechanic** (rep visible to all NPCs everywhere). Interesting in DOGMud's "belief becomes truth" cosmology, but that's a separate design conversation. Tracked as a future-work memory entry.
- **Mob quest-giving as a parity goal.** NPCs already give quests to players; players giving quests to NPCs isn't a feature we want.
- **Player-as-bounty-hunter system.** The flip side of 5.2. Could be a later expansion once 5.2 ships and we see how players want to use the bounty data.
- **Player-facing UI for companion/mob mutations.** The companion-Phase-5 UI piece. Comes after 2.5 lands and we see what configuration surface is actually useful.
- **General-purpose planner (HTN/GOAP)** in the strategic layer. We start with hand-authored per-goal planners (4.4); generalize only if the hand-authored set sprawls.

## Maintenance

- **Updating Status:** Edit *both* the row in the **Progress tracker** table near the top *and* the `Status:` line on the chunk's mini-brief. Re-tally the roll-up line under the table when statuses change.
- **Adding chunks:** Append a row to the tracker table *and* a mini-brief to the relevant phase. If a chunk doesn't fit a phase, that's a signal something was missed in the framing — flag for design discussion.
- **Removing chunks:** Mark `Cancelled` rather than deleting, with a one-line reason. Helps future-you remember what was considered and why it was dropped.
- **Per-chunk specs and plans:** Each chunk gets its own `docs/superpowers/specs/YYYY-MM-DD-<chunk-id>-design.md` and corresponding plan when picked up.
- **MEMORY.md sync:** When a MEMORY-absorbed chunk ships, remove its MEMORY entry. When a brand-new chunk ships, add a note in COMPLETED.md.
