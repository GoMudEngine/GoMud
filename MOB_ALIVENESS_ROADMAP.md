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
| 1.2 | Substrate | Faction system | L | 1.1 | Done |
| 1.3 | Substrate | Crime/wanted state | M | 1.2 | Done |
| 1.4 | Substrate | NPC knowledge model | M | 1.1 | Done |
| 1.5 | Substrate | Bounty state | S | 1.2 | Done |
| 1.6 | Substrate | NPC-to-NPC relationships | M | — | Done |
| 1.7 | Substrate | World-model facts | M | 1.4 | Done |
| 2.1 | Tactical | Mob `buy` command | M | — | Done |
| 2.2 | Tactical | Item-comparison primitive | M | — | Done |
| 2.2a | Tactical | Incorporeal mutation | M | — | Done |
| 2.3 | Tactical | Equip-if-better behavior | S | 2.2 | Done |
| 2.4 | Tactical | Mob `consider` + threat-aware behaviors | S | 2.2 | Done |
| 2.5 | Tactical | Mutations on mobs | L | — | Done |
| 2.6 | Tactical | Sunset legacy tactics engine | L | — | Done |
| 2.7 | Tactical | Mob skullduggery suite | M | — | Done |
| 2.8 | Tactical | Mob scout / track / scan | M | — | Done |
| 2.9 | Tactical | Mob `forage` as a command | M | — | Done |
| 2.10 | Tactical | PvM/MvP/PvP/MvM parity audit | M | 2.1–2.9 | Done |
| 3.1 | Routine | Game-time hook | S | — | Done |
| 3.2 | Routine | NPC schedules | L | 3.1 | Done |
| 3.3 | Routine | Sleeping / wake states | M | 3.1 | Done |
| 3.4 | Routine | Waypoint patrols | M | — | Done |
| 3.5 | Routine | Maintenance routines | M | 3.2 | Deferred |
| 3.6 | Routine | NPC↔NPC idle conversation | M | 1.6 | Done |
| 3.7 | Routine | Inter-zone patrols + caravan unification | L | 3.4 | Done |
| 3.8 | Routine | One-shot sub-patrols (caravan runner + forager delivery) | M | 3.7 | Done |
| 4.1 | Strategic | Goal representation | M | 1.1, 1.4 | Done |
| 4.2 | Strategic | Goal selection | L | 4.1 | Done |
| 4.3 | Strategic | Goal types catalog | L | 4.1 | Done |
| 4.4 | Strategic | Strategic→tactical translation | XL | 4.3, Phase 2 | Done |
| 4.5 | Strategic | Reactive goal generation | L | 1.6, 4.1 | Done |
| 4.6 | Strategic | Goal satisfaction & pruning | S | 4.1 | Done |
| 5.1 | Cross-cut | Town justice | XL | 1.2, 1.3, 1.5, 3.4, Phase 4 | Done |
| 5.2 | Cross-cut | Bounty hunting | L | 1.4, 1.5, 2.8, 4.4 | Done |
| 5.3 | Cross-cut | Equipment-aware shopping | L | 2.1, 2.2, 2.3, 4.4 | Done |
| 5.4 | Cross-cut | NPC market participation | M | 5.3 | Done (2026-06-02) |
| 6.1 | Polish | Stillwater town-flavor pass | L | Phase 1, Phase 3 | Done |
| 6.2 | Polish | Parity audit closeout | S | 6.1 | Done |
| 6.3 | Polish | Per-zone tuning (Thornwall deepening) | M | 6.1 | Done (2026-06-03) |
| 6.4 | Polish | Performance review (initial) | S | 6.3 | Done (2026-06-05) |
| 6.5 | Polish | Content pass — broader rollout | XL | 6.3 | Not started |
| 6.5a | Polish | Faction definitions content pass | M | 1.2, 1.3 | Done (2026-06-05) |
| 6.6 | Polish | Performance re-review | S | 6.5 | Not started |

**Roll-up:** 40 / 42 done • 0 in progress • 2 not started (+1 deferred: 3.5).

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
**Status:** Done (2026-05-06) • **Size:** L

- **Goal:** Faction definitions, NPC membership, per-player reputation per faction.
- **In:** Faction YAML, NPC `faction` field, per-player rep store, rep-change API, faction-vs-faction relations (allies/enemies), admin inspection commands.
- **Out:** Faction-specific quests (content-pass time), full citizenship UI (future).
- **Depends on:** 1.1 (shares store backend)
- **Why:** Replaces the `peacefulquest` placeholder. Enables "Stillwater militia hates you because you killed one of theirs."
- **Shipped:** `internal/factions/` package with signed-scalar rep [-100, +100], no decay, per-faction default rep. Definition YAMLs at `_datafiles/world/dogmud/factions/{slug}.yaml` (committed); rep state at `_datafiles/world/dogmud/factions.rep/{slug}.yaml` (gitignored). Public API (GetRep/SetRep/BumpRep/TierFor + FactionsForMob/IsPeacefulToward), reuses `opinions.Tier` for banding. New quest engine `bump_rep` action. Combat hookup `MobDeath_FactionRep` bumps killer + same-room party members. Admin command `faction list/show/set/bump/reset` + helpfile. Two authored factions: warren (default -25, enemies thornwall_guards) and thornwall_guards (default 0, narrow guards-only — broader thornwall_citizens deferred to chunk 1.3 alongside crime/wanted state). `peacefulquest` field deleted from Mob struct after Migration 0.13.0 seeded warren rep at +30 for live players holding the legacy `2-end` quest token. Quest 2 (The Warren Compact) now fires `bump_rep: warren +30` on completion. Smoke-boot verified: migration ran cleanly, faction definitions load without panic on ally/enemy validation, rep file persists. **Manual in-game smoke test (smoketester walking through quest 2 end-to-end) deferred to user verification per the spec's smoke-test section.** Fixed a latent path bug from chunk 1.1 — `opinions.opinionsBaseDir` and `factions.*BaseDir` were treating `DataFiles` as if it didn't already include `/world/dogmud`, which it does. Roadmap chunk 6.5a added for the broader faction content pass. Spec at `docs/superpowers/specs/2026-05-06-mob-aliveness-1.2-faction-system-design.md`, plan at `docs/superpowers/plans/2026-05-06-mob-aliveness-1.2-faction-system.md`.

### 1.3 Crime/wanted state
**Status:** Done (2026-05-06) • **Size:** M

- **Goal:** Per-player log of unresolved crimes (theft, assault, murder) keyed by zone or faction.
- **In:** Crime types, witness tracking, zone/faction scoping, expiry rules, query API, admin debug.
- **Out:** Guard reactions — that's 5.1.
- **Depends on:** 1.2
- **Why:** "I assaulted a Stillwater citizen — Stillwater knows it" requires this data. Town justice and bounty hunting both consume it.
- **Shipped:** `internal/crimes/` package storing per-faction murder/assault/theft logs at `_datafiles/world/dogmud/factions.crimes/{slug}.yaml` (gitignored). Witness-required model: faction-aligned mob in the room identifies the perpetrator; lone acts record `perp: unknown` with no rep impact. Public API: Record, Resolve, FindRecentAssault, AllForFaction, AllForPlayer, PruneStale, WitnessesInRoom, IdentifiedPerp, UpgradeAssaultToMurder. Crime-aware combat hookups: `MobDeath_FactionRep` rewritten to consult crime log + apply per-kind rep deltas (CrimeRepDeltaMurder -25, Assault -10, Theft -5); first-aggression in `attack`/`target` records assault crime alongside chunk 1.2 opinion bump; failed-steal records theft crime. Each fight yields ONE crime (assault upgrades in-place to murder on death). 365-day game-time stale safety net via PruneStale (Balance.CrimeStaleAfterRounds = 7,884,000). Authored `thornwall_citizens` faction (deferred from 1.2) — 20 named civilians + 3 guards via multi-faction membership. New admin command `crime list/show/resolve/prune-stale` + helpfile. Bonus fix during T7: caught a real `loadOrLazyInit` race condition (concurrent goroutines on uncached faction could lose records — saw 491/500 before fix); fixed via double-check-lock pattern. Spec at `docs/superpowers/specs/2026-05-06-mob-aliveness-1.3-crime-wanted-design.md`, plan at `docs/superpowers/plans/2026-05-06-mob-aliveness-1.3-crime-wanted.md`.

### 1.4 NPC knowledge model
**Status:** Done (2026-05-09) • **Size:** M

- **Goal:** What facts does this NPC know about player X — name learned, last-seen room, deeds witnessed, items seen carried.
- **In:** Knowledge schema, learn/forget API, perception-gated learning (NPCs only learn what they witness or are told), query API for tactical/strategic layers.
- **Out:** World-level facts (1.7).
- **Depends on:** 1.1
- **Why:** Lets an NPC say "I saw you with the bandit chief's coat — turn it in." Without this, NPCs are amnesiacs even if 1.1 tells them their feeling.
- **Shipped:** `internal/knowledge/` package storing per-observer-NPC YAML at `_datafiles/world/dogmud/knowledge/{mobId}-{namesimple}.yaml` (gitignored). Polymorphic subject (`{type, id}` for player or mob template), source/confidence tier on every record, per-fact decay rules, NPC-on-NPC supported. v1 fact types: identity (HasMet + NameLearned), location (LastSeen + bounded observation log capped at `KnowledgeObservationLogMax = 32`), routine (FrequentedRooms top-K query), deeds-witnessed (crime row IDs, lazy-filtered against 1.3 on read via WitnessedCrimes). Auto-write triggers v1: forager/caravan room change (new hook listener `MobRoomChange_KnowledgeObservers` wraps `knowledge.RecordRoutineObservers`; archetype discriminators `forager.IsForagerMob` and `caravan.IsCaravanMob` added) and 1.3 crime witnessing (three call-site additions in attack.go, MobDeath_FactionRep.go, skullduggery steal). Explicit `Forget` / `ForgetFact` API for amnesia consumers. New admin command `knowledge show/forget/frequented` + helpfile. No cross-substrate cascade in v1 (documented as a deferred decision pending amnesia spell consumer). Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.4-knowledge-model-design.md`, plan at `docs/superpowers/plans/2026-05-09-mob-aliveness-1.4-knowledge-model.md`.

### 1.5 Bounty state
**Status:** Done (2026-05-09) • **Size:** S

- **Goal:** Declared bounties (payer, target, reward, conditions, expiry) queryable by mobs and players.
- **In:** Bounty data structure, declaration API (faction-driven, quest-driven, NPC-driven), claim/resolution API, admin commands.
- **Out:** Bounty board UI for players (could be a follow-on player-facing affordance), bounty hunter behavior (5.2).
- **Depends on:** 1.2
- **Why:** Enables 5.2 bounty hunting, escalation in town justice, faction-driven contracts.
- **Shipped:** `internal/bounties/` package storing per-bounty registry at `_datafiles/world/dogmud/bounties.yaml` (gitignored). Polymorphic target via `knowledge.Subject` (player or mob template); three issuer types (faction, quest, npc). Reward auto-computes from target statpool — `gold = floor(statpool × BountyGoldDefaultMultiplier)` (default 0.5, floor 50) and `rep = max(1, floor(statpool / 100))`, both stored on the row at declaration with declarer override available via `DeclareOpts`. Auto-claim hook `MobDeath_BountyClaim` fires on mob death — highest-damager wins (companion damage already rolls up via `combat.go`'s charmed-userId path), gold transferred to character, faction rep bumped when issuer is a faction. Quest engine `declare_bounty` action wires the substrate into quest content. Single `bounty` command with role-gated subcommands: list/show available to all players (filter by mob/player/<faction-slug>), declare/withdraw/prune-expired admin-only. Admin helpfile + player helpfile. Two physical bounty boards as flavor nouns (Thornwall Guard Barracks 473, Stillwater Constabulary 4110) — discovery via `look bounty board`; data flow via the universal command. Withdraw + expiry semantics; non-open rows preserved for audit. Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.5-bounty-state-design.md`, plan at `docs/superpowers/plans/2026-05-09-mob-aliveness-1.5-bounty-state.md`.

### 1.6 NPC-to-NPC relationships
**Status:** Done (2026-05-09) • **Size:** M

- **Goal:** Kinship and friendship graph between NPCs (Voss is Lars's brother; Marta is the smith's wife).
- **In:** Relationship types (family, friend, rival, lover, employer/employee), per-NPC relationship list, query API, mutation API.
- **Out:** Relationship change as a player-facing mechanic (a romance system is way out of scope).
- **Depends on:** —
- **Why:** Killing one NPC seeds revenge goals in their kin. The world starts to feel woven, not flat.
- **Shipped:** `internal/relationships/` package storing the in-memory mob-to-mob relationship graph. Source of truth: each mob template's YAML gains an optional `relationships:` field with `to`, `type`, `subtype`. Six types (family, friend, rival, lover, employer, employee); engine auto-mirrors symmetric (same-type reverse) and asymmetric (employer ↔ employee) at load time. Subtype is per-side flavor. Permissive validation — unknown ids, self-edges, unknown types, conflicts all warn-not-panic. Public API: `RelationsOf`, `RelationsOfType`, `KinOf`, `AlliesOf`, `RivalsOf`, `RelationsBetween`, `AreRelated`, `EmployerOf`, `EmployedBy`, `AllRelations`, plus mutation `Add`/`Remove`/`ChangeType` (in-memory only v1; persistence overlay deferred). Loader hook in `mobs.LoadDataFiles` flattens mob templates into `LoadFromMobs(edges, validateMobId)` post-load. Admin command `relationship show/between/add/remove/list` + helpfile. **Backfilled `context.md` for chunks 1.1–1.5** plus authored fresh one for 1.6, per the new aliveness roadmap maintenance rule. Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.6-npc-relationships-design.md`, plan at `docs/superpowers/plans/2026-05-09-mob-aliveness-1.6-npc-relationships.md`.

### 1.7 World-model facts
**Status:** Done (2026-05-09) • **Size:** M

- **Goal:** Zone-level or world-level facts NPCs can "know" (the bridge collapsed, the bandit camp moved, the king is dead).
- **In:** Fact schema, fact declaration API, NPC awareness-of-fact tracking (some know, some don't), propagation rules (gossip).
- **Out:** Dynamic fact generation from world events — start with author-declared facts.
- **Depends on:** 1.4
- **Why:** Makes the world feel like it has news, rumors, and shared context — not just isolated NPC bubbles.
- **Shipped:** `internal/facts/` package storing the standing-fact registry at `_datafiles/world/dogmud/facts.yaml` (committed; empty seed) and per-NPC awareness at `_datafiles/world/dogmud/facts.awareness/{mobId}-{namesimple}.yaml` (gitignored). Awareness store is unified — it holds BOTH heard-event ids (bounded FIFO via `FactsHeardEventsMax`, default 32; replaces the in-memory `recentGossipEvents` TempData) AND known-fact ids (persistent). Three withdraw signals: manual `Withdraw`, time-based `expiry_round` + `PruneExpired` sweep, auto via `withdraw_on_respawn_of` field with new `MobRoomChange_FactsAutoWithdraw` listener. Lazy-filter on read for awareness × registry join. Public API: Declare, Withdraw, Expire, PruneExpired, WithdrawAllBoundTo, GetFact, AllActiveFacts, AllFactsByTag, AllRows, RecordHeardEvent, HeardEvent, RecordKnowsFact, KnowsFact, KnownFactsOf, ForgetFact, ForgetAll, AllForObserver, LoadFromMobs. Mob YAML extension: `knows_facts: [factId, ...]` for inline authoring (chunk 1.6 pattern); seeded into awareness at `mobs.LoadDataFiles` post-load. Worldevents gained `Id uint64` field, atomic-counter assigned at `EmitWorldEvent` time. `buildGossipLine` migrated from `recentGossipEvents` TempData to facts substrate; gossip candidate pool extended with known facts (new `fact-default` template family, 70/30 event/fact split when both pools non-empty). Admin command `fact list/show/declare/withdraw/expire/prune-expired/awareness/teach/forget/forget-all` + helpfile. New package context.md authored. Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.7-world-facts-design.md`, plan at `docs/superpowers/plans/2026-05-09-mob-aliveness-1.7-world-facts.md`.

**Phase 1 substrate complete.** All seven Phase 1 chunks shipped: opinions (1.1), factions (1.2), crimes (1.3), knowledge (1.4), bounties (1.5), relationships (1.6), facts (1.7).

---

## Phase 2 — Tactical fill-in

Verbs and behavior-tree gaps that the strategic layer will need to dispatch.
Build vocabulary before the planner.

### 2.1 Mob `buy` command
**Status:** Done (2026-05-11) • **Size:** M

- **Goal:** Mobs can purchase from shops, including disambiguation, gold checks, carry capacity.
- **In:** Mobcommand `buy`, integration with existing shop pricing/stock, restocking interaction with NPC-buyer behavior.
- **Out:** Decision logic for *what* to buy — that lives in tactical/strategic.
- **Depends on:** —
- **Why:** Strategic-layer "save up for armor" is impossible without this verb.
- **Shipped:** Consolidated `actions.Buy(buyer Actor, opts BuyOptions) BuyResult` lifted from `internal/usercommands/buy.go` into the shared `actions` package. Player wrapper at `internal/usercommands/buy.go` collapses to ~22 lines (~830 lines deleted); new mob wrapper at `internal/mobcommands/buy.go` (~20 lines) registered in the `mobCommands` map. Both shop backends supported symmetrically — legacy `Character.Shop` (with `Restock()` on access + the `+1` merchant-gold cheat preserved) and `ShopInventory` (dynamic pricing, persistence, bartering discount up to 15% at skill 50). Sale types limited to items + buffs; merc and pet paths dropped entirely (no current shop YAML sells either; `executePurchaseMerc` / `executePurchasePet` deleted). New pre-side-effect carry-capacity gate in `validatePurchase` closes a pre-existing player-side gap: `char.GetCarriedWeight() + newItem.GetSpec().Weight > char.CarryCapacity()` blocks the purchase before destock or gold deduction. Quest-engine `command:buy` notification gated by `buyer.IsPlayer()`. Mob bartering progression falls out naturally via the symmetric `OnSkillUse("bartering")` call. Quantity (`buy N <item>`) and `from <merchant>` syntax work on both wrappers. `EffectiveRestock` exported from actions package because `internal/usercommands/list.go` shared the helper. Unit tests cover empty-request, no-merchant, encumbrance-gate-pre-side-effect, catalog merc/pet filtering, quantity parsing; full purchase-flow integration testing deferred to the broader aliveness-effort manual smoke. Smoke verified at build level: `go build` clean, all 47 packages pass tests, server boots cleanly past data-file load with 225 mobs / 248 items / 21 quests. Spec at `docs/superpowers/specs/2026-05-11-mob-aliveness-2.1-mob-buy-command-design.md`, plan at `docs/superpowers/plans/2026-05-11-mob-aliveness-2.1-mob-buy-command.md`.

### 2.2 Item-comparison primitive
**Status:** Done (2026-05-11) • **Size:** M

- **Goal:** Callable function: "is item A an upgrade over item B for this mob?"
- **In:** Multi-axis comparison (damage, mitigation, weight, slot conflicts, archetype-fit), per-archetype weighting, returns a score so callers can rank a list.
- **Out:** Action — that's 2.3.
- **Depends on:** —
- **Why:** Underlies all "smart equipping" and "smart shopping." Without it, mobs can't tell good gear from bad.

- **Shipped:** New `internal/itemvalue/` package with two-tier
  API. Pure `ItemValue(spec, profile) float64` for catalog
  ranking (used by chunk 5.3 equipment-aware shopping). Mob-
  aware `ItemValueDelta(char, profile, candidate) SwapDelta`
  for swap decisions with smart slot selection (rings pick the
  weaker occupant; 1H weapons compare Weapon vs Offhand
  placements; 2H weapons displace both Weapon and Offhand).
  Symmetric bonus application (`DualWieldBonus`, `ShieldBonus`,
  `TwoHandedBonus`); `DualWieldBonus` conditional on the pre-
  swap main hand holding a 1H weapon (no synergy without a
  partner). Encumbrance tier penalty applied per-tier crossed
  (thresholds 0.25/0.50/0.75/1.00 matching userrecord prompt
  rendering). Six named weight profiles (`PhysicalBruiser`,
  `PhysicalTank`, `Stealth`, `MagicalPure`, `MagicalSupport`,
  `Neutral`) derived via `ProfileFor(stat, behavior)` —
  `BehaviorArchetype` primary, `Archetype` fallback. New
  `IsUpgrade(char, profile, candidate) bool` convenience
  wrapper. Deleted v0 helpers `items.ItemPower` and
  `items.IsUpgrade`; migrated sole caller `mobs/crafter.go`
  to the new API (~200 lines net deleted). Skill mods on item
  instances flagged as out of scope (instance-zone loot
  affixes carry +skill mods that the spec.StatMods view
  doesn't currently surface). Spec at
  `docs/superpowers/specs/2026-05-11-mob-aliveness-2.2-item-comparison-primitive-design.md`,
  plan at
  `docs/superpowers/plans/2026-05-11-mob-aliveness-2.2-item-comparison-primitive.md`.

### 2.2a Incorporeal mutation
**Status:** Done (2026-05-11) • **Size:** M

- **Goal:** Model ethereal beings (wraiths, spectres, fire and
  air elementals, elemental queen) as a new rarest mutation
  (`incorporeal`) with four ranks scaling gear effectiveness
  loss + physical defense bonus + stat shifts.
- **In:** Mutation YAML, two new effect types
  (`gear_effectiveness_loss`, `physical_defense_bonus`), five
  consumer-site integrations (stat aggregation, three
  mitigation getters, weapon damage, spell damage, defense
  resolution, itemvalue scoring), mob YAML tagging on five
  templates, helpfile + per-mutation help template +
  context.md updates.
- **Out:** Per-rank tuning beyond starting values, player
  acquisition trigger beyond rarity weighting, earth/water
  elemental tagging.
- **Depends on:** —
- **Why:** Chunk 2.3 (equip-if-better) needs a gate to skip
  ethereal mobs/players. Soft-scaling via itemvalue scoring
  is cleaner than a hardcoded skip path. Also unblocks future
  "incorporeal player" progression goals.
- **Shipped:** New `_datafiles/world/dogmud/mutations/incorporeal.yaml`
  with rarity 10 + conflict list (seven body-dependent
  mutations). New `GetGearEffectivenessLoss`,
  `GearEffectivenessMultiplier`, `GetPhysicalDefenseBonus`
  helpers in `internal/mutations/mutations.go` —
  `gear_effectiveness_loss` uses raw level multiplication
  (linear 0.25/0.50/0.75/1.00 across ranks), the carve-out
  documented in `internal/mutations/context.md`. Five
  integration sites: `character.go` (StatMod scales Equipment
  portion + three Get*Mitigation methods separate gear from
  non-gear with the slot list completed during this chunk
  — previously missing Shoulders, Back, Wrist1/2, ExtraWrist1-4,
  Ring2, ExtraArm3-4, ComponentBag), `combat_helpers.go`
  (buildWeaponSetup applies multiplier to weaponDmgMult;
  best-of-all defense resolution adds physical_defense_bonus
  for physical-channel attacks), `calculations.go` (spell
  damage gear contributions scaled at 4 sites including buff
  tick-pool scaling), `itemvalue/delta.go` (ItemValueDelta
  applies multiplier to candidate + displaced totals). Five
  mob templates tagged with `mutations: { incorporeal: 4 }`
  (wraith, spectre, fire elemental, air elemental, elemental
  queen). Helpfile + dedicated per-mutation help template +
  three context.md files updated. Spec at
  `docs/superpowers/specs/2026-05-11-mob-aliveness-2.2a-incorporeal-mutation-design.md`,
  plan at
  `docs/superpowers/plans/2026-05-11-mob-aliveness-2.2a-incorporeal-mutation.md`.

### 2.3 Equip-if-better behavior
**Status:** Done (2026-05-11) • **Size:** S

- **Goal:** Tactical behavior: on loot pickup or item-give, evaluate and equip if it beats the current slot occupant.
- **In:** Btree action, per-archetype configurable, emits emote when swapping.
- **Out:** —
- **Depends on:** 2.2
- **Why:** "I gave the bandit a steel sword and he's still using a club" → fixed.
- **Shipped:** Two gate helpers in `internal/itemvalue/equip_eligibility.go`:
  `CanEquipFromGive` (skips animal species via `Species.DisabledSlots`
  + non-combat archetypes) and `CanScanFloorLoot` (above + charmed-
  status). New `EquipBestFloorItem(mob, room) bool` lives in
  `internal/hooks/mob_equip_best_floor_item.go` (not in `itemvalue`
  — would close an import cycle through rooms→mobs→itemvalue) and
  is wired into `internal/hooks/MobIdle_HandleIdleMobs.go`. Existing
  `internal/mobcommands/gearup.go` rewritten to use `itemvalue.IsUpgrade`
  instead of the gold-value heuristic; PermaGear / charmed-drop /
  emote phrasing all preserved (charmed-drop logic now applied to
  both the specific-item and bare-gearup paths). Push and pull
  broadcast emotes are distinct ("puts on" / "wields" for push;
  "picks up X and dons it" / "wields it" for pull). Incorporeal
  mobs (chunk 2.2a) skip naturally via gear-effectiveness scoring
  — no special path. Per-archetype configurability is satisfied by
  chunk 2.2's WeightProfile system (no new knobs). Spec at
  `docs/superpowers/specs/2026-05-11-mob-aliveness-2.3-equip-if-better-design.md`,
  plan at
  `docs/superpowers/plans/2026-05-11-mob-aliveness-2.3-equip-if-better.md`.

### 2.4 Mob `consider` + threat-aware behaviors
**Status:** Done (2026-05-12) • **Size:** S

- **Goal:** Mobs size up combat threats the same way players do; reactive (lookout-only-ambushes-weaker-players) and opportunistic (predator-engages-weaker-prey) behaviors unlocked.
- **In:** Actor-pattern consolidation of `consider` into `actions.Consider(actor, target) ConsiderResult` shared by player + mob wrappers (`internal/usercommands/consider.go` thinned, `internal/mobcommands/consider.go` added). Two btree primitives: `target_power_ratio_above`/`_below` condition (in new `conditions_combat.go`) and `target_weakest_mob_in_room` action (added to `actions_combat.go`). Demo wiring: lookout archetype gains `player_enter`→ambush-if-stronger branch; new `predator` archetype copies generic_fighter and adds a leading `mob_idle` predation branch; three ironwind wolf YAMLs (steppe/young/scarred) flip to predator (alpha kept as `leader` to preserve its rally/warcry pack-leader behavior — a `predator_leader` hybrid is logged as future work).
- **Out:** Player gear-coveting (players don't drop gear so no use case); `appraise` mobcommand (player command is obsoleted by identify spell); `combat.PowerScore` math changes (audit confirmed gear is already reflected through `ValueAdj`/`Get*Mitigation` pipes; the audit deliverable is a documentation section in `internal/combat/context.md`).
- **Depends on:** 2.2 (item-comparison primitive contributed conceptually but PowerScore-based assessment uses existing combat infrastructure).
- **Why:** Reactive lookouts that don't suicide-ambush strong players. Opportunistic predators that go after weaker prey. Foundation for chunk 2.6 (tactics-cast preemption — power-ratio gating offensive vs. defensive cast selection) and 5.2 (bounty hunting — bounty hunters need to assess wanted targets).
- **Shipped:** `internal/actions/consider.go` — `Consider(actor, target) ConsiderResult` with prediction text emission via `actor.SendText` (MobActor no-op preserves silent compute path). Player + mob wrappers each ~15 lines. Btree primitives in `conditions_combat.go` and `actions_combat.go` (new function alongside existing entries). Target resolution chain: `Event.UserId` → `Aggro.MobInstanceId` → `Aggro.UserId` (matches `actions.ResolveAggroTarget` convention). `mob.HatesMob(other)` predicate gates predation — covers faction/pack-awareness without coupling to 1.2 substrate. Lookout `player_enter` branch with `target_power_ratio_above: 1.0` ambush gate. Predator archetype `ratio_below: 0.85` predation ceiling. PowerScore audit section added to `internal/combat/context.md`. Spec at `docs/superpowers/specs/2026-05-12-mob-aliveness-2.4-mob-consider-design.md`, plan at `docs/superpowers/plans/2026-05-12-mob-aliveness-2.4-mob-consider.md`.

### 2.5 Mutations on mobs
**Status:** Done (2026-05-12) • **Size:** L

- **Goal:** Companion Phase 5 — mutations apply to mobs the way they do to players.
- **In:** Mob mutation slots, YAML schema, runtime application of mutation effects (extra arms, tail, etc.), combat integration, scaling.
- **Out:** Player-facing UI for mutations on companions/mobs (separate concern).
- **Depends on:** —
- **Why:** Closes a major parity gap. Mutated mobs are a content lever for novel encounters. (Absorbed from MEMORY.md — Companion Phase 5.)
- **Shipped:** Body-plan gating model — `Species.BodyParts []string` from a canonical seven-tag set (`arms, hands, legs, eyes, mouth, skin, tail`); `MutationSpec.RequiresBodyParts []string` replaces the old `RequiresArms bool`. Three gating sites updated: random-roll pool (`GetWeightedPool` signature changed to take species), curated `SpawnMutations` path (latent bug fix — was applying unconditionally), and mid-game mutation grants (5 call sites across user round tick, behavior tree action, quest engine bridge, login, and mob spawn). `Character.ApplyIntrinsicMutations(species)` merges species intrinsics additively into the character's mutation map at init time, cap-aware via `MutationMaxRank = 4`. Migration covered all 35 existing species (skip dummy 19, orb 20) + 4 new elemental species (sand 41, storm 42, ice 43, smoke 44). 17 mutation YAMLs gained `requires_body_parts:` declarations. 5 mob YAMLs in `instance_planar_oasis/` repointed: king kept on magma + added `mutations: { large: 1 }` override, queen moved to new ice species (dropping her chunk-2.2a `incorporeal: 4` override since her crystal/water form is corporeal), prince moved to new smoke species. Redundant `mutations: { incorporeal: 4 }` overrides on 4 summons mobs (wraith, spectre, fire, air) cleaned up — incorporeal is now intrinsic on the species. Boot-time validation panics on unknown body-part tags or unknown mutation ids in intrinsic_mutations. Helpfiles updated to document body-plan gating in player-facing terms (mutations.template, species.template, 17 per-mutation templates each got a "Requires:" line). Spec at `docs/superpowers/specs/2026-05-12-mob-aliveness-2.5-mutations-on-mobs-design.md`, plan at `docs/superpowers/plans/2026-05-12-mob-aliveness-2.5-mutations-on-mobs.md`.

### 2.6 Sunset legacy tactics engine
**Status:** Done (2026-05-12) • **Size:** L

- **Goal:** Delete the legacy `internal/mobai/` tactics engine and migrate all 44 tactic-using mobs to the behavior tree (btree) system.
- **In:** Reframed from the original "fix the Edrin priority race" band-aid into the structural fix. Btree now the single mob-behavior substrate. Five existing archetypes gain a shared panic-flee branch (generic_fighter, predator, leader, lookout, tank_taunter). tank_taunter additionally gets a call_for_help branch (absorbing the `tank` preset); ambusher gets a target_casting→trip branch (absorbing the `ambusher` preset). One new `defensive_caster` archetype absorbs 4 mobs from the old `defensive_caster` and `caster_backline` presets. Five per-boss archetypes for named encounter mobs (Edrin, Sylara, Rhett, Soren, Chrysalis Phantom) preserve their unique spell rotations.
- **Out:** Boss encounter tuning (faithful translation only); generic-mob inline-tactic preservation beyond what the augmented archetypes cover (acceptable loss).
- **Depends on:** —
- **Why:** Eliminated the dual-system architectural smell. The original Edrin priority-race bug became structurally impossible (btree selectors are inherently priority-ordered, no async reaction queue racing `InitiateCast`). ~1,144 net lines of legacy code deleted.
- **Shipped:** Zero new btree primitives (mob_has_buff + invert decorator covers missing_buff). 6 new archetypes (defensive_caster + 5 boss). 5 archetype augmentations. 44 mob YAML migrations (24 preset-only + 5 named bosses + 9 generic inline-tactics). Engine deletion: `internal/mobai/` directory entirely removed (10 files including tactics.go, reactor.go, actions.go, types.go, memory.go, triggers.go and tests). `CombatMemory` substrate migrated to `internal/mobs/combat_memory.go` (it was used outside the tactics engine for grudge tracking). Mob struct fields `Tactics`, `TacticPreset`, `ReactionDelay`, `TacticalDiscipline` removed. Hook callers in `internal/hooks/` cleaned (MobAI_Reactor.go deleted entirely). `internal/mobs/context.md` + `internal/behaviortree/context.md` updated. `project_tactics_cast_preemption.md` MEMORY entry deleted. **Known follow-up:** Edrin/Sylara's conviction-ward opening cast lacks a self-gate because conviction-ward is a shield spell (no buff_id). Bosses re-cast wastefully after shield expires; behavior is not broken, just wasteful. Polish item for a future tuning pass. Spec at `docs/superpowers/specs/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine-design.md`, plan at `docs/superpowers/plans/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine.md`.

### 2.7 Mob skullduggery suite
**Status:** Done (2026-05-15) • **Size:** M

- **Goal:** Mob versions of `steal`, `pickpocket`, `sneak`, `hide`, `plant` (and maybe `defuse`/`shadow`).
- **In:** Mobcommands for each, behavior tree integration, archetype tagging (only thieves/scouts use these).
- **Out:** —
- **Depends on:** —
- **Why:** Bandit NPCs that can't pickpocket aren't bandits. Major aliveness lever.
- **Shipped:** All 5 skullduggery verbs lifted into `internal/actions/` via the actor pattern (Sneak/Steal/Plant/Defuse/Shadow). Player wrappers thinned; mob wrappers added; btree primitives (`try_steal`, `try_sneak`, `try_plant`, `try_shadow`, `try_defuse`) + state-query conditions (`mob_is_hidden`, `target_is_hidden`, `target_has_gold`) + new picker (`target_random_player_in_room`) added. New `thief` archetype with steal-and-flee loop (panic-flee, power-overmatch combat, self-defense, steal+flee, re-stealth). Thornwall highwayman flipped to thief archetype. Mob-on-player detection roll (search vs sneak) on Steal mirror to the existing player-on-mob path; gold transfer happens regardless of detection, only the victim's awareness is detection-gated. Legacy `stealth_detection.go` shim + pickpocket test placeholder sunset (triple-removal in go.go left intact — load-bearing for permabuffs, see followup memory). **In-game smoke validated 2026-05-15** after the combat-state-machines side quest (chunk 0) landed: Thornwall highwayman attempts hide → steals 73 gold from smoketester → flees, never enters combat. Originally (2026-05-13) the highwayman picked up a sword, hid, then opened with grapple — root cause was `target_random_player_in_room` calling `SetAggro`, which is the in-combat flag. The chunk-0 `EvalContext.SoftTarget` slot structurally prevents that bug class. Spec at `docs/superpowers/specs/2026-05-13-mob-aliveness-2.7-skullduggery-suite-design.md`, plan at `docs/superpowers/plans/2026-05-13-mob-aliveness-2.7-skullduggery-suite.md`, smoke report at `tools/testing/reports/2026-05-15-local-feature-tester-chunk-0-combat-state-machines.md`.

### 2.8 Mob scout / track / scan
**Status:** Done (2026-05-22) • **Size:** M (originally scoped S; expanded during plan-writing)

- **Goal:** Information-gathering verbs available to mobs.
- **In:** Mobcommands for `track`, `scan`, `search`, `consider`.
- **Out:** —
- **Depends on:** —
- **Why:** Bounty hunters and patrols need to find things. Quest-NPC scouts feel passive without these.
- **Shipped:** Three actions lifted into `internal/actions/` (`Scan`, `Track`, `Search`). Four btree action primitives (`try_scan`, `try_track`, `try_search`, `move_toward_tracked`) + two conditions (`room_has_hidden_entity`, `mob_is_tracking`). New `scout` archetype + flip on goblin_scout (217). Single-branch grafts onto `lookout` (scan-before-ambush), `thief` (search-before-steal), `leader` (track-on-aggro-lost). **Bundled bug fix #1:** authored buff 86 (Active Tracking, 25-round duration) replacing buff 26 (Conviction Surge) misuse in skill.track.go. Migrated 4 AddBuff + 6 RemoveBuff call sites (one extra discovered in skill.track.go's `track stop/clear` handler). Fixed the "tracking forever" bug by adding a `HasBuff(86)` outer gate at the roomdetails.go renderer that clears misc data on buff absence. **Bundled bug fix #2:** authored buff 87 (Shadowing, 25-round duration). Shadow now applies buff 87 on success and the auto-follow consumer in go.go gates on buff presence, preventing phantom shadows from dragging players to dead/logged-off targets. **Universal escape gates:** new hooks `MobDeath_TrackingCleanup` and `PlayerDespawn_TrackingCleanup` clear tracking/shadow misc data + buffs 86/87 on any character pointing to the dying mob / leaving user. **Smoke testing deferred to user** (13-scenario in-game validation, see plan section "Smoke test plan"). Spec at `docs/superpowers/specs/2026-05-22-mob-aliveness-2.8-scout-track-scan-design.md`, plan at `docs/superpowers/plans/2026-05-22-mob-aliveness-2.8-scout-track-scan.md`.

### 2.9 Mob `forage` as a command
**Status:** Done (2026-05-22) • **Size:** M (originally scoped S; expanded during brainstorming to include salvage parallel + forager archetype migration + state-machine refactor)

- **Goal:** Promote forage from routine-only behavior to a callable verb.
- **In:** Mobcommand wrapper around existing forage skill, btree integration.
- **Out:** —
- **Depends on:** —
- **Why:** Strategic NPCs that decide "I'm out of leather, let me forage" need the verb.
- **Shipped:** Two actions lifted into `internal/actions/` (`Forage`, `Salvage` single-tick core). Three btree primitives (`try_forage`, `try_salvage`, `wander_territory`) + one condition (`forager_state_is_foraging`). Hybrid forager state-machine refactor: Foraging-state per-tick loop dissolved into YAML, multi-state daily cycle preserved in Go via `forager_step`. New shared `forager` archetype replaces three per-mob behavior YAMLs for Tova (371), Halix (372), Kessa (373). Player per-tick salvage resolve in `hooks/NewRound_UserRoundTick.go` refactored to call `actions.Salvage`. Includes follow-up fix to restore the "corpse no longer here" player message that the lift initially dropped. Spec at `docs/superpowers/specs/2026-05-22-mob-aliveness-2.9-mob-forage-salvage-design.md`, plan at `docs/superpowers/plans/2026-05-22-mob-aliveness-2.9-mob-forage-salvage.md`. **In-game smoke testing deferred to user.**

### 2.10 PvM/MvP/PvP/MvM parity audit
**Status:** Done (2026-05-23) • **Size:** M

- **Goal:** Sweep remaining parity gaps after 2.1–2.9 land.
- **In:** Walk every player command, classify it (mob-equivalent / orthogonal / never-relevant), patch concrete gaps.
- **Out:** Forced parity for every player command — only what's relevant to mob behavior.
- **Depends on:** 2.1–2.9 (do the obvious ones first, then audit the long tail)
- **Why:** Catches verbs we didn't think to add. (Absorbed from MEMORY.md.)
- **Shipped:** **Audit deliverable:** 119 player commands + 63 mob commands classified against a 6-bucket parity scheme (Equivalent / Orthogonal / Never-relevant / Gap: patch inline / Gap: delete divergent verb / Gap: defer). Player-side: 44 Equivalent, 28 Orthogonal, 37 Never-relevant, 0 Gap-patch-inline (the 6 mutation_* gaps handled via Stage B), 0 Gap-delete, 4 Gap-defer (lock/picklock/throw/unlock — surprise_attack reclassified to Equivalent after triage). Mob-side: 44 Equivalent, 18 Orthogonal, 0 Never-relevant, 0 Gap-patch-inline, 1 Gap-delete (selljunk), 0 Gap-defer. Tables embedded in the design spec at `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit-design.md`. **Mutation_* actions-lift (closes Companion Phase 5 mob half):** 6 new `actions.TriggerXxx` functions: blinding_flash (AoE), blinding_spit (single-target), healing_gel (self, out-of-combat), pacifism_aura (AoE de-aggro), sonic_shout (AoE damage+prone+self-deafen), toxic_bite (single-target damage+poison). Shared `actions/mutation_helpers.go` with `mutationPreamble(actor, key, combatRequired, staminaCost)` running the 4 gate checks (mutation presence, in-combat, cooldown, stamina) used by all 6. Player wrappers collapsed (~480 LoC deleted from `internal/usercommands/`); new mob wrappers added (~120 LoC) registered in `mobcommands.go`. New btree action `try_mutation_active` accepting `key` (single) or `keys` (ordered list); rejects nodes with neither at first-call time with logged error + Failure. Single-target mutations (`blinding-spit`, `toxic-bite`) fail-out with "no-target" via this action — separate primitive with target resolution is a deferred followup. `MutationOpts` / `MutationResult` types live in `mutation_blinding_flash.go` (defined first as the worked example). **selljunk deletion:** Phantom verb with zero callers — deleted from `mobcommands/selljunk.go`, registration in `mobcommands.go`, test in `mobcommands_test.go`, and divergences entry in `actions/divergences.go`. Case study for the new "delete divergent verb" verdict from the audit. **Pre-existing bugs surfaced + preserved verbatim (followups logged):** sonic-shout damage = `int(Wil × 0.08)` raw arithmetic, not `combat.CalcRawDamage` ([[project_mutation_damage_pipeline_bypass]]); toxic-bite damage = `int(Str × 0.06)` + poison magnitude = `int(Vit × 0.04)` — same pipeline bypass; sonic-shout "stun" is actually `TransitionToProne` + self-deafen (`ConditionBlinded` 3 rounds), NOT `ConditionStunned` (which doesn't exist); charge.go reimplements trip math instead of delegating to actions.ExecuteTrip ([[project_charge_trip_math_duplication]]). **Pre-existing bug FIXED during the lift:** mutation_healing_gel was computing `int(Vitality × 0.15)` — a stat-derived flat value violating the CLAUDE.md no-flat-heal rule. Corrected to `floor(HealthMax × 25 / 100)` (25% of pool). Balance change worth flagging at prod-push time — [[project_healing_gel_balance_change]]. **Deferred-gap triage outcomes (user-triaged 2026-05-23):** surprise_attack reclassified to Equivalent (mobs already have parity via `mobcommands/attack.go:64` + Awareness_Cascades); picklock wontfix (intentional design misalignment; minigame is player-only); lock + unlock bundled into the forager locked-chest workflow chunk ([[project_forager_locked_chest_workflow]]); throw deferred to future ranged weapon system ([[project_throwable_mobs_ranged_dependency]]). Review doc retained at `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-deferred-gaps-review.md`. **Additional followup logged:** runtime-evolved mutations don't auto-flow into btree dispatch ([[project_mutation_active_runtime_evolution_btree]]). **Closes:** Companion Phase 5 (player-facing UI was already wontfix per `feedback_companion_autonomy`; mutation actives on mobs now shipped). **Manual in-game smoke testing deferred to user** (per the chunk 2.9 precedent). Design spec: `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit-design.md`. Plan: `docs/superpowers/plans/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit.md`. Deferred-gap review: `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-deferred-gaps-review.md`.
- **Followup chunk shipped 2026-05-23** (branch `feature/mob-aliveness-2.10-followups`, will merge as one commit): 5 followups bundled — charge.go trip-math dedup, surprise-attack unification (added per-weapon burst on mob side for true parity), new `try_any_active_mutation` btree action (rarity-descending dispatch), mutation damage pipeline routing for sonic-shout (Conviction channel) and toxic-bite bite damage (Physical channel), forager locked-chest workflow (new Tova dwelling room 4198 north of Spring Pool, lock/unlock mob verbs, `try_store_excess` btree primitive, `StateStoring` forager state). Spec at `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-followups-design.md`; plan at `docs/superpowers/plans/2026-05-23-mob-aliveness-2.10-followups.md`. **Vendor backfill from chests left as critical followup** ([[project_vendor_backfill_from_forager_chests]]).

**Phase 2 tactical fill-in complete.** All ten Phase 2 chunks shipped (2.1 buy, 2.2 item-comparison, 2.2a incorporeal mutation, 2.3 equip-if-better, 2.4 consider, 2.5 mutations, 2.6 sunset legacy tactics, 2.7 skullduggery, 2.8 scout/track/scan, 2.9 forage, 2.10 parity audit). Mob tactical vocabulary substantially closes the gap with player verbs; remaining gaps are tracked in [[project_pvm_mvp_parity_gaps]] and the chunk-2.10 deferred-gaps review doc.

---

## Phase 3 — Routine layer

Scheduled and repeating procedural behaviors. Adds time-of-day texture to the
world.

### 3.1 Game-time hook
**Status:** Done (2026-05-23) • **Size:** S

- **Goal:** Expose time-of-day to behaviors (clock primitive — extend the existing time system if needed).
- **In:** Time tick, day/night flag, configurable day length, btree condition `time_of_day_is`.
- **Out:** Visible time-of-day UI for players (could come with content pass).
- **Depends on:** —
- **Why:** Without time, schedules are meaningless. Cheap foundation.
- **Shipped:** Extended the existing `time_of_day` btree condition (`internal/behaviortree/conditions_state.go:64`) with a `range:` parameter for hour-precise time gating that chunk 3.2 schedules will use. Range format `"<start>-<end>"` with `[start, end)` semantics, wraps midnight when `start > end` (e.g., `"22-6"` for nightwatch). When both `period:` and `range:` are set, `range:` wins. Empty range (`"5-5"`) always Failure + warning; full-day (`"0-24"`) always Success + warning; malformed strings log an error once and return Failure. `sync.Map` dedup prevents log spam. Most of the roadmap requirements were already in place: `util.GetRoundCount()` provides the time tick, `gametime.IsNight()` + `GameDate.Night` provide the day/night flag, `configs.GetTimingConfig().RoundsPerDay` provides configurable day length, and `modules/time/time.go` already gives players a `time` command. The only new code was the `range:` parser + wrap-around comparator + 20 unit tests. Spec at `docs/superpowers/specs/2026-05-23-mob-aliveness-3.1-game-time-hook-design.md`, plan at `docs/superpowers/plans/2026-05-23-mob-aliveness-3.1-game-time-hook.md`.

### 3.2 NPC schedules
**Status:** Done • **Size:** L

- **Goal:** Timed routines: "smith works 9–5, home 5–8, tavern 8–11, sleep."
- **In:** Schedule YAML, schedule executor, behaviors for "go to room" and "perform activity at room."
- **Out:** Per-day variation (weekday/weekend/holiday) — start with single daily routine.
- **Depends on:** 3.1
- **Why:** A town that empties at night and fills in the morning feels a thousand percent more alive than a static town.
- **Shipped:** Schedule loader + 24h-coverage validator + pathfinding sanity in `internal/mobs/schedule.go` and `internal/mobs/schedule_loader.go`. Go-side executor in `internal/hooks/NewRound_IdleMobs_schedule.go` steers scheduled mobs via existing `pathto` plumbing, swaps per-segment `IdleCommands`, falls back to home after `ScheduleMaxPathRetries` failures. Spawn override in `newMobByIdInternal` places scheduled mobs at the current segment's target room. `TickMobCraft` respects per-segment `activity: craft` so Blacksmith Kerra only forges at the forge. New `mob_at_target_room` btree condition. New `mob schedule <instId>` admin inspector. Three Thornwall pilots: Blacksmith Kerra, Tavern Keeper Marek, Temple Priest Olen, each with a new above-shop home room. Spec at `docs/superpowers/specs/completed/2026-05-25-mob-aliveness-3.2-npc-schedules-design.md`, plan at `docs/superpowers/plans/completed/2026-05-25-mob-aliveness-3.2-npc-schedules.md`.

### 3.3 Sleeping / wake states
**Status:** Done (2026-05-25) • **Size:** M

- **Goal:** NPCs visibly asleep at night; wakeable by sound, light, attack.
- **In:** Sleeping condition, room descriptions for sleeping NPCs, wake triggers, combat-on-sleeper consequences (more crime severity).
- **Out:** —
- **Depends on:** 3.1
- **Why:** A sleeping NPC is a tiny piece of fiction that compounds well — assassinations, theft, pickpocket-while-sleeping.
- **Shipped:** Sleeping is a queryable state-flag (`buffs.Sleeping`) applied via
  `actions.Sleep(actor)` from both player `sleep` and mob `sleep` commands, and
  from the schedule executor's `activity: sleeping` segment hook. Sleepers gain
  5× HP/SP/CP regen (`SleepRegenMultiplier`, default 5.0). Attackers in the first
  round against a sleeper auto-crit via a start-of-round victim snapshot +
  `forceCrit bool` on the damage pipeline. Wake triggers: damage (new
  `cancel-on-damage` flag), failed steal, shout-in-room, light source on room
  entry (via the existing `EmitsLight` buff flag), `stand` command. Schedule
  executor honors a grace cooldown (`ScheduleWakeGraceRounds`, default 50) after
  a forced wake. Room render appends `(asleep)` to occupant names. Three
  Thornwall pilots retrofit. Spec at
  `docs/superpowers/specs/completed/2026-05-25-mob-aliveness-3.3-sleeping-wake-states-design.md`,
  plan at
  `docs/superpowers/plans/completed/2026-05-25-mob-aliveness-3.3-sleeping-wake-states.md`.
- **3.3 leaves available for 5.1:** Town Justice may wish to scale faction
  response by victim state at crime time. The data is queryable live
  (`victim.Character.HasBuffFlag(buffs.Sleeping)`) at the moment
  `crimes.Record(...)` is called; no Crime-schema change is required up front.

### 3.4 Waypoint patrols
**Status:** Done • **Size:** M

- **Goal:** Looped multi-room routes with optional dwell times.
- **In:** Patrol route YAML, executor, interrupt-handling (combat aborts patrol; resume after).
- **Out:** Dynamic re-routing when paths blocked (start with hard-failure on blocked path).
- **Depends on:** —
- **Why:** Guards that actually walk a beat. Town justice (5.1) consumes this.
- **Shipped:** Patrol primitive — multi-room routes with strict +
  yo-yo loop shapes, per-waypoint dwell, combat interrupt with
  resume-to-same-waypoint, retry-then-pathto-home fallback (reuses
  chunk 3.2 `ScheduleMaxPathRetries`). Two integration paths:
  standalone (`patrol_id` on mob) and composed (`activity: patrol`
  segment via chunk 3.2 schedules). New
  `internal/mobs/patrol.go` + `patrol_loader.go`, new
  `internal/hooks/NewRound_IdleMobs_patrol.go`. Schedule schema
  gains `target_room`-optional for patrol segments and a
  `patrol_id` field; spawn override falls back to the patrol's
  first waypoint when a patrol segment has no explicit target.
  Admin `mob schedule <instId>` inspector extended to render
  patrol state. Pilot: Thornwall city guard (mob 106) with a
  6-22 patrol of the market beat + 22-6 sleep at a new guard
  barracks room (5104, above existing constabulary 473). Spec
  at `docs/superpowers/specs/completed/2026-05-25-mob-aliveness-3.4-waypoint-patrols-design.md`,
  plan at
  `docs/superpowers/plans/completed/2026-05-25-mob-aliveness-3.4-waypoint-patrols.md`.

### 3.5 Maintenance routines
**Status:** Deferred • **Size:** M

- **Goal:** Smith repairs gear, farmer tends crops, librarian shelves books — flavor activity tied to NPC role.
- **In:** Activity YAML, emote-driven flavor, optional integration with crafting (smith actually crafts inventory restock).
- **Out:** Activities producing real economic output (crafting restock can be a follow-on chunk).
- **Depends on:** 3.2 (maintenance often slots inside schedules)
- **Why:** Walking into the smithy and seeing the smith working tells the player the world isn't waiting on them.
- **Builds on:** Chunk 3.2's per-segment `activity:` field. New maintenance verbs (`tend_crops`, `shelve_books`, etc.) will be dispatched when a segment declares them.
- **Deferred (2026-05-25):** Chunk 3.2's per-segment `idlecommands:` field already delivers the canonical "see the smith working" experience for Kerra (and will for future similarly-authored NPCs). The reusable activity-library angle has no consumers yet — we only have one smith in one zone. Re-evaluate this chunk when content authoring hits real duplication pain across multiple smiths/farmers/librarians in multiple zones. The concrete "crafter ticks but item doesn't appear in shop list" complaint that surfaced during this triage is being tracked separately (see follow-up bug fix).

### 3.6 NPC↔NPC idle conversation
**Status:** Done • **Size:** M

- **Goal:** NPCs occasionally talk to each other (canned exchanges, mood-aware).
- **In:** Pair-conversation YAML (paired with 1.6 relationships), trigger logic (proximity + cooldown), mood-aware variants.
- **Out:** Player-overheard "spoken about you" gossip (later, ties to 1.4 knowledge spread).
- **Depends on:** 1.6
- **Why:** A guard and a baker chatting in the square is the highest-bang-for-buck aliveness signal.
- **Shipped:** New `internal/conversations/` package (replaces
  legacy upstream system retired in T1.5) with type pools
  (`types/<relationship-type>.yaml`) and per-pair overrides
  (`pairs/<lower>_<higher>.yaml`). Loader with filename↔id
  check, speaker validation, mob existence + relationship edge
  cross-check via DI. Picker draws uniformly from
  `type_pool ∪ matching_subtype ∪ pair_override`. State machine
  runs in `NewRound_IdleMobs`: per-tick trigger
  (`ConversationBaseChancePct`, default 1%) + player-arrival
  boost (`ConversationPlayerArrivalBoostPct`, default 25%) in
  `go.go`. Line-per-round pacing via shared
  `conversation_line_idx` MiscData; deterministic speaker
  alternation. Graceful abort on partner moves / sleeps /
  combats / player dialogue. Cooldown
  (`ConversationCooldownRounds`, default 50) on both NPCs
  after completion. `MobConversant` interface +
  `internal/conversationadapter/` keep the import graph
  acyclic. Pilot: Thornwall tavern back-room — Dal + Fen +
  Gobb + Wrex friend edges (6 pairs) + optional rival edge
  Fen↔Wrex + friend pool (11 exchanges + 2 fond-subtype) +
  optional rival pool (5 exchanges) + optional Dal↔Wrex pair
  override (3 exchanges, role-agnostic). NPC↔NPC opinion store
  and "spoken about you" gossip explicitly deferred (see
  spec). Spec at
  `docs/superpowers/specs/completed/2026-05-25-mob-aliveness-3.6-npc-conversations-design.md`,
  plan at
  `docs/superpowers/plans/completed/2026-05-25-mob-aliveness-3.6-npc-conversations.md`.

### 3.7 Inter-zone patrols + caravan unification
**Status:** Not started • **Size:** L

- **Goal:** Extend chunk 3.4 patrols to cross zone boundaries, and migrate caravan movement code onto the shared patrol layer (caravans become a specialized yo-yo patrol with cargo + vendor semantics layered on top).
- **In:** Cross-zone waypoint references in patrol YAML, patrol-executor handling of zone-boundary pathing, caravan movement code refactored to call the patrol executor (caravan-specific concerns — cargo, vendor visits, gold exchange — stay in `internal/caravan/`).
- **Out:** Multi-stop caravan optimization (pathfinding "best order" of stops), seasonal route variation.
- **Depends on:** 3.4 (patrols)
- **Why:** Caravans currently maintain their own movement logic, parallel to mob wandering and (now) patrols. Unifying onto patrol primitives reduces drift, lets caravan routes be authored as standard patrol files, and surfaces inter-zone routing as a first-class engine feature for both caravans and future zone-spanning NPCs (traveling merchants, pilgrim NPCs, etc.). Decision deferred from chunk 3.4 to focus that chunk on the single-zone patrol primitive.
- **3.4 satisfied:** Single-zone patrol primitive shipped in
  3.4. 3.7 lifts the single-zone restriction and migrates
  caravan movement onto the shared layer.

### 3.8 One-shot sub-patrols (caravan runner + forager delivery)
**Status:** Done (2026-05-26) • **Size:** M

- **Goal:** Two related routine-layer refinements that share
  the post-3.7 patrol substrate:
  1. **Caravan runner-delivery flavor pass.** The 3.7
     migration preserves the existing "wagon visits every
     vendor room" pattern as a faithful refactor — but
     semantically the wagon shouldn't be dragged into an
     alchemy shop. Resurrect the original design intent
     (lost in the first caravan implementation) where
     Ketil's son acts as a runner, walking supplies from the
     parked wagon at the depot out to each vendor.
  2. **Explore moving foragers onto the patrol layer.**
     Foragers currently use a custom state-machine for
     wander-a-territory movement and are still getting hung
     up on prod (stranding, edge-case pathto failures). The
     patrol layer's retry-then-home-fallback + standardized
     interrupt handling may offer a more robust substrate.
     "Wander a territory" isn't a straight-line patrol, so
     this is exploration first — design a probabilistic or
     territory-aware patrol variant, then decide whether to
     port foragers.
- **In:**
  - Runner mob (Ketil's son) authored + slotted into caravan
    crew; vendor-stop waypoints in the caravan patrol
    collapse to just the depot, with restock dispatched via
    a depot-arrival listener that walks the runner through
    the local vendor circuit.
  - Forager-territory pattern investigation: prototype a
    `loop_shape: random` or `wander_territory` patrol mode,
    A/B against the current forager state machine in
    smoke testing, decide whether the engine extension is
    worth the migration.
  - If foragers migrate: per-forager territory definition as
    patrol YAML, deletion of the current forager wander
    state machine in `internal/forager/` (parallel to the
    caravan reduction in 3.7).
- **Out:** Multi-runner per caravan (one runner is enough for
  the proof-of-concept). Generalized "delegate vendor visits
  to a runner crew member" abstraction beyond caravans
  (defer until a second consumer materializes).
- **Depends on:** 3.7 (cross-zone patrols + caravan unification).
- **Why:** 3.7 ships engine unification but accepts the
  wagon-in-shops semantic compromise as a faithful refactor.
  3.8 closes the flavor loop and explores whether the same
  unification path makes sense for foragers, which have been
  a recurring source of prod stability issues since chunk
  2.9. Bundling the two reduces context-switching cost —
  both share the same patrol-substrate mental model.
- **Memory references:** `caravan-runner-delivery-flavor`,
  `forager_fatigue_cadence` (chunk 2.9 follow-up bug
  tracking).
- **Shipped:** Reframed during brainstorming from "two
  related refinements" into a single unifying engine
  primitive — a one-shot sub-patrol invoked from within a
  larger NPC routine. New `loop_shape: oneshot` patrol mode
  emits `events.PatrolCompleted` on final dwell and clears
  itself via `mobs.ClearOneshotPatrol`. Runtime assignment
  via `mobs.StartOneshotPatrol(mob, patrolId)`.

  Caravan consumer: Lars (mob 359, Ketil's son, previously
  tagged as a guard but canonically the runner) becomes the
  depot-to-vendor runner. Cargo transfers wagon→Lars at
  depot arrival (outbound buckets only); Lars walks the
  town's vendor circuit; on `PatrolCompleted`, residual
  cargo returns to wagon. Caravan main route shrinks from
  22 waypoints to 4 (depots + Fernway pickups). Stillwater
  dwell bumps 20→180 to fit Lars's circuit. Hob and Bran
  identified as horses (not guards); Marta is the only
  proper guard. Lars's strength training bumped 18→60 for
  carry capacity. New `bucketsForRunnerPatrol` keyed by
  patrol id (not waypoint idx). Synthesizer evolution keeps
  dashboard JSON schema byte-identical — `*Route` shows
  while Lars is mid-circuit, `*Dwell` while resting.

  Forager consumer: Marsh (Tova) and Steppe (Halix) Delivering
  phase collapses from internal vendor-room loop to
  oneshot sub-patrol. `tickForagerDeliveringTown` reduces to
  ~3 branches: 5.4 sanctuary-fallback safety, no-op if patrol
  already active, otherwise start patrol. Wander+forage phase
  stays in `internal/forager/` unchanged. Fernway forager
  (Kessa) keeps her existing single-stop sealed-crate handoff.

  Refactor: `npcVisitVendorsInRoom` moved from
  `internal/behaviortree/actions_forager.go` to
  `internal/forager/vendor_sell.go` (renamed to exported
  `SellToVendor`) — required to break the import cycle the
  forager listener would have introduced.

  Forward-looking note: attacking the caravan crew or wagon
  will carry severe consequences once Town Justice (chunk
  5.1) lands. No 1.3 crime hooks wired in 3.8; flagged in
  PATCH_NOTES and spec.

  17-task plan executed via subagent-driven development.
  Spec at `docs/superpowers/specs/2026-05-26-mob-aliveness-3.8-oneshot-subpatrols-design.md`,
  plan at `docs/superpowers/plans/2026-05-26-mob-aliveness-3.8-oneshot-subpatrols.md`.

---

## Phase 4 — Strategic layer

The new "what do I want?" engine. Builds on substrate state and tactical
verbs.

### 4.1 Goal representation
**Status:** Done • **Size:** M

- **Goal:** Define what a goal is in code — type, target, satisfaction predicate, expiry, priority.
- **In:** Goal struct/interface, registration, persistence, debug command.
- **Out:** Goal selection logic (4.2).
- **Depends on:** 1.1, 1.4 (goals reference state)
- **Why:** Foundation for the strategic layer. Without this, "drives" stay vibes.
- **Shipped:** `internal/goals/` package — `Goal` struct + YAML round-trip, per-NPC `MobGoals` store with atomic-write
  persistence to `_datafiles/world/dogmud/goals/{mobId}-{namesimple}.yaml` (gitignored), type-metadata
  registry with symmetry check at init, `Add/Remove/Clear/GoalsOf/IsSatisfied/IsExpired` store API,
  conflict-resolution (highest-priority wins on duplicate target+type), concurrent-safe cache.
  Admin command `goal list/show/add/remove/clear` for live inspection and testing. No change to
  observable NPC behavior — foundation only. Spec at
  `docs/superpowers/specs/2026-05-26-mob-aliveness-4.1-goal-representation-design.md`, plan at
  `docs/superpowers/plans/2026-05-26-mob-aliveness-4.1-goal-representation.md`.

### 4.2 Goal selection
**Status:** Done • **Size:** L

- **Goal:** NPC picks a current goal from a candidate set based on priority, context, and recent state.
- **In:** Selection function, hysteresis (don't goal-thrash), per-archetype weighting.
- **Out:** Multi-goal pursuit (start single-goal-at-a-time).
- **Depends on:** 4.1
- **Why:** The "pick what to want" engine. Without it, NPCs have goals but never act on them.
- **Shipped:** Pure `Select` function in `internal/goals/select.go` over the 4.1 substrate; per-archetype
  `goal_weights:` map (parsed by `behaviortree.LoadArchetypeYAMLFromFile`); optional per-type `ContextScore`
  hook on `GoalTypeMeta` (registry empty until 4.3; panic-recovered); two-gate hysteresis (margin +
  min-hold) with config knobs `GoalSelectSwitchMargin`/`GoalSelectMinHoldRounds`/`GoalSelectTickEnabled`;
  `Recompute` orchestrator persists `current_goal_id` / `current_since_round` / `last_switch_round` to
  the `MobGoals` YAML on switch; per-round tick hook `tickMobRecomputeGoals` (idle lane, cheap-paths on
  empty goal list); eager `Recompute` on `Add`/`Remove`/`Clear`; `goals.SetWeightsLookup` boot-wired in
  `main.go` to bridge → `behaviortree.Engine.GetArchetypeGoalWeights` without an import cycle; admin
  subcommands `goal current` / `goal scores` + helpfile updates; structured `goals.switch` debug log
  line per selection switch. No btree integration yet (4.4's job). Spec at
  `docs/superpowers/specs/2026-05-27-mob-aliveness-4.2-goal-selection-design.md`, plan at
  `docs/superpowers/plans/2026-05-27-mob-aliveness-4.2-goal-selection.md`.

### 4.3 Goal types catalog
**Status:** Done • **Size:** L (upsized from M during brainstorming — all 6 categories shipped deep)

- **Goal:** Concrete goal types: survival, wealth, revenge, protection, social, mastery, exploration.
- **In:** YAML for each type, archetype-default goal sets, parameters per goal.
- **Out:** Player-authored custom goals (deferred).
- **Depends on:** 4.1
- **Why:** Without concrete types, the strategic layer is empty scaffolding.
- **Shipped:** 13 narrow goal types in `internal/goals/catalog/`: `survival`, `wealth-gold`,
  `wealth-item`, `craft-item`, `revenge-mob`, `revenge-faction`, `protection-mob`, `protection-faction`,
  `befriend`, `befriend-faction`, `mastery-skill`, `mastery-equip`, `visit-zone`. Each has its own
  Predicate + ContextScore + optional DedupKey + ParamSchema, registered via `init()`. Engine deltas:
  declarative `ParamSchema` validation at `Add` time; `AllowMultiple` + `DedupKey` on `GoalTypeMeta`
  for multi-instance types; archetype lazy-seed sentinel (`SeededFromArchetype` on `MobGoals`) so
  defaults seed once per template and survive admin Clear. `behaviortree.LoadArchetypeYAMLFromFile`
  parses a new top-level `default_goals:` block; `goals.SetArchetypeDefaultsLookup` boot-wired in
  `main.go` to bridge without an import cycle. New `Mob.VisitedZones` instance state + room-change
  hook in `rooms.AddMob` to feed `visit-zone`. Sparse archetype defaults: `survival` seeded on
  every combat-capable archetype + generic `wealth-gold` for thief / shopkeeper (16 archetypes).
  `gossip` goal type intentionally dropped during brainstorming — existing MobIdle gossiper system
  (`buildGossipLine` in `NewRound_HandleIdleMobs.go`) stays as-is; goal-driven directed-gossip
  belongs in a future gossip-system refinement chunk. Cross-type conflict mechanism deferred.
  Closed `internal/goals/context.md` documentation gap from 4.1/4.2; new
  `internal/goals/catalog/context.md`. Spec at
  `docs/superpowers/specs/2026-05-27-mob-aliveness-4.3-goal-types-catalog-design.md`, plan at
  `docs/superpowers/plans/2026-05-27-mob-aliveness-4.3-goal-types-catalog.md`.

### 4.4 Strategic→tactical translation
**Status:** Done • **Size:** XL (upsized from L during brainstorming — all 13 planners shipped deep)

- **Goal:** Planner that turns a goal into routine/tactical actions (e.g., "save for armor" → walk to shop, check stock, sell loot, save gold, buy when affordable).
- **In:** Per-goal-type planners, fallback to btree, plan-failure recovery.
- **Out:** General-purpose planner (HTN, GOAP) — start with hand-authored per-goal planners.
- **Depends on:** 4.3, Phase 2 verbs
- **Why:** Bridges desire to action. The whole point of the strategic layer.
- **Shipped:** 13 deep planners in `internal/planners/`: `survival`, `wealth-gold`, `wealth-item`,
  `craft-item`, `revenge-mob`, `revenge-faction`, `protection-mob`, `protection-faction`,
  `befriend`, `befriend-faction`, `mastery-skill`, `mastery-equip`, `visit-zone`. Each is a
  stateless Go function (`PlanFn`) called per-tick when a goal of its type is current; intermediate
  progress lives in `mob.Character.MiscData` under a `plan:<goal_type>:` key prefix wiped on goal
  switch via `SetPlanStateClear` callback registered in `main.go`. New `try_goal_planner` btree
  action (`internal/behaviortree/actions_goal.go`) dispatches per `goals.CurrentGoalOf`; inserted
  into all 18 non-boss archetype trees at author-chosen positions. 15 supporting helpers
  (`shop-in-zone selling/buying`, faction member filters, hostile finder, crafting-station
  finder, zone-adjacency BFS cache, gift picker, random-exit picker, social-emote rotation,
  recipe-by-skill picker, MiscData read/write helpers). Skill training table maps the canonical
  `skills.SkillTag` values to TrainingContext kinds for the mastery-skill planner. Reactive seeding
  hooks for `craft-item` (materials missing → seed wealth-item), `revenge-faction` /
  `befriend-faction` (counter writes) deferred to 4.5. Permanent-stuck-goal pruning deferred to 4.6.
  Spec at `docs/superpowers/specs/2026-05-27-mob-aliveness-4.4-strategic-tactical-translation-design.md`,
  plan at `docs/superpowers/plans/2026-05-27-mob-aliveness-4.4-strategic-tactical-translation.md`.

### 4.5 Reactive goal generation
**Status:** Done • **Size:** L (upsized from M during brainstorming — Option B all-events architecture)

- **Goal:** Events seed new goals (player kills NPC's friend → revenge goal seeded into the friend's NPCs).
- **In:** Event hooks (combat death, theft, faction insult), goal-seeding rules, goal deduplication.
- **Out:** —
- **Depends on:** 1.6, 4.1
- **Why:** Goals that *react* to player actions are what make the world feel responsive.
- **Shipped:** 10 rules in `internal/seeders/` package. Live: rule 1 (faction_kill_counter, MobDeath),
  rule 3 (craft_materials_to_wealth_item, planner-invoked), rule 4 (friend_killed_to_revenge,
  MobDeath + relationships walk), rule 5 (witness_of_theft_to_revenge, steal-action invoked),
  rule 6 (aggressive_action_to_revenge, new PlayerAttackedMob event), rule 7 (gift_to_opinion_boost,
  new GiftAccepted event with value-tiered bumps + cooldown), rule 9 (combat_assist_to_opinion_boost,
  shares PlayerAttackedMob with rule 6 — multi-consumer payoff of Option B). Stubbed for follow-up:
  rule 2 (faction_rep_counter — no clean positive-interaction event signal), rule 8
  (quest_completion_to_opinion_boost — no quest-giver field on quest YAML), rule 10
  (mastery_milestone_to_priority_bump — events.SkillUsed is player-only with no NewRank).
  Three new event types added to events package: `PlayerAttackedMob`, `GiftOffered`, `GiftAccepted`.
  Centralized dispatcher with panic recovery; main.go wires `events.RegisterListener` per type.
  Cross-package follow-up logged: witness-of-theft archetype split (some mobs report crime instead
  of personal revenge) — wait on 5.1 Town Justice. Spec at
  `docs/superpowers/specs/2026-05-27-mob-aliveness-4.5-reactive-goal-generation-design.md`,
  plan at `docs/superpowers/plans/2026-05-27-mob-aliveness-4.5-reactive-goal-generation.md`.

### 4.6 Goal satisfaction & pruning
**Status:** Done (2026-05-29) • **Size:** S

Shipped: throttled per-mob `goals.Prune` sweep retiring satisfied
(`IsSatisfied`) and expired (`IsExpired`) goals, plus dormancy-based
abandonment (per-goal `DormantSinceRound`; abandoned once context score
has been ~0 for `GoalAbandonDormantRounds`). Runs from the goals tick on a
staggered cadence (`GoalPruneIntervalRounds`). Spec at
`docs/superpowers/specs/2026-05-29-mob-aliveness-4.6-goal-pruning-design.md`,
plan at `docs/superpowers/plans/2026-05-29-mob-aliveness-4.6-goal-pruning.md`.

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
**Status:** Done (2026-06-01) — 5.1a–c + Stillwater rollout shipped; 5.1d
closed as "done as far as we are taking it" • **Size:** XL

Decomposition (each its own spec→plan→build): **5.1a** wanted-verdict + guard
warn→attack (DONE — `internal/justice`, per-round guard tick; spec/plan at
`docs/superpowers/{specs,plans}/2026-05-29-town-justice-5.1a-guard-enforcement*`).
**5.1b** crime→auto-bounty trigger (DONE). **5.1c** arrest mechanic (DONE).
**5.1d** redemption (pay-fine/serve/quest) — **not built; closed 2026-06-01**
per user decision that the current justice loop (warn → attack → arrest →
detain, plus death-pays-the-debt via the 5.2 bounty-hunter `ClearFactionRecord`
path) is sufficient. Death and serving a sentence already clear the faction
record, so a dedicated pay-fine/quest redemption flow is deferred indefinitely;
reopen as a fresh chunk if redemption UX is later wanted. The chunk-5.1-followup
warn-stamp pruning already landed in the Stillwater rollout.

**Stillwater rollout shipped 2026-05-30:** data-driven holding-cell registry
(`HoldingCellRoom` field on faction definition YAML, boot-validated via
`factions.ValidateHoldingCells`; replaces hardcoded `jailCellFor` map);
arresting faction selected by `firstFactionWithCell`; new holding cell room
5106 (down from Stillwater constabulary 4110); new `stillwater_guards`
(holding_cell_room 5106, ally stillwater_citizens) + `stillwater_citizens`
factions — guards+citizens now exist for **two** towns (Thornwall +
Stillwater); Constable Drunn (mob 335) flipped to combat-capable
`guard_captain` archetype with tank stats; 21 Stillwater townsfolk + 2
foragers (Tova/Kessa) tagged `stillwater_citizens`; 3 caravan humans
(Ketil/Marta/Lars) tagged both citizen factions; 7 civic quests grant +15
rep via new `RepFaction`/`RepAmount` quest reward fields; stale
`justice_warned_*` stamp pruning in per-guard-tick sweep (5.1 followup #2).

- **Goal:** Replace `peacefulquest` placeholder with real citizenship + faction guards + crime detection + bounty workflow.
- **In:** Citizenship-by-faction, guard archetypes that react to crimes against citizens, escalation (warn → arrest → kill), bounty placement on offenders, redemption (pay fine, complete quest, etc.).
- **Out:** Per-zone *unique* justice (each zone uses the framework with config tweaks; one-off rules in content pass).
- **Depends on:** 1.2, 1.3, 1.5, 3.4 (patrols), Phase 4 (guards with goals)
- **Why:** The single biggest aliveness leap — the world reacts to player crimes meaningfully. (Absorbed from MEMORY.md — peacefulquest → faction system.)

### 5.2 Bounty hunting
**Status:** Done (2026-05-30) • **Size:** L

- **Goal:** NPCs (and NPC archetypes) actively hunt declared-bounty targets — NPC bandits *or* wanted players.
- **In:** Bounty hunter archetype, hunt-goal seeded from 1.5 bounty state, tracking via 2.8, encounter behavior, optional contract acquisition (pick up bounties from boards).
- **Out:** Player-as-bounty-hunter system (player-facing flip side; could come later).
- **Depends on:** 1.4, 1.5, 2.8, 4.4
- **Why:** Bad actors can't safely hide. Wanted players get *chased*, not just yelled at by guards. NPC bad actors get hunted by their own world.
- **Shipped:** Two halves. **Half A (NPC hunts wanted player):** a new
  `internal/bountyhunter` package runs a per-round dispatch sweep
  (`RunDispatchSweep`) that spawns ONE scaled, affix-geared hunter per
  player whose single open faction bounty meets or exceeds
  `BountyHunterGoldThreshold`; the hunter is placed at the issuer
  faction's `release_room`, carries the faction group (so the
  `PlayerDeath_BountyResolve` `killGuard` path credits the kill), and
  has a param-less `hunt_bounty_target` goal added so `try_goal_planner`
  fires. Per-hunter target lives in instance `MiscData`
  (`bh_target_user_id`) — not in goal params — because the goals store is
  template-keyed and cannot hold per-instance divergent state. New
  `hunt_bounty_target` goal type + planner: pathto → attack; jailed target
  → planner hold (never enters a cell); offline target → hold. On death,
  `justice.ClearFactionRecord` clears the record — death pays the debt
  identically to serving a sentence. Redispatch cooldown
  (`BountyHunterRedispatchCooldown`) enforces a reprieve window after a
  hunter is killed. **Not PvP** — only NPC hunters pursue wanted players.
  **Half B (player-claimable standing bounties):** an authored
  `bounties.standing.yaml` seed file (3 notable hostile NPCs) is loaded
  idempotently at boot via `bounties.SeedStanding`; players claim by
  killing the target and receive gold + faction rep via the existing
  `MobDeath_BountyClaim` hook. Package context at
  `internal/bountyhunter/context.md`. Spec at
  `docs/superpowers/specs/2026-05-30-mob-aliveness-5.2-bounty-hunting-design.md`,
  plan at
  `docs/superpowers/plans/2026-05-30-mob-aliveness-5.2-bounty-hunting.md`.
- **Deferred follow-ups:**
  - **Disguise-kit evasion** — a skullduggery-skill item that hides the
    player's identity from the hunter's pursuit check (the planner could
    query a `hidden_identity` MiscData key set by the item).
  - **Criminal NPCs hunted (NPC-vs-NPC)** — wanted mobs hunted by
    faction-aligned hunters. Requires mob-target bounty dispatch, a
    second dispatch branch in `RunDispatchSweep`, and planner support for
    mob targets. No current content need; land when a faction-criminal NPC
    archetype is authored.
  - **Instance-keyed goal state** — 5.2 sidesteps the template-keyed
    goal store via instance MiscData. If more instance-divergent
    goal-param use cases accumulate, invest in per-instance goal state in
    `internal/goals`.

### 5.3 Equipment-aware shopping
**Status:** Done (2026-06-01) • **Size:** L

- **Goal:** NPCs save gold and visit shops to buy upgrades, applying 2.2's comparison logic.
- **In:** "Upgrade my X slot" goal, gold-saving behavior, shop-route planning, archetype preferences.
- **Out:** NPCs commissioning custom-craft from player-crafters (maybe later).
- **Depends on:** 2.1, 2.2, 2.3, 4.4
- **Why:** A bandit who keeps the steel sword you dropped and shows up in better gear next time is far more memorable than one in starter rags.
- **Shipped:** Survey-worst-slot upgrade drive. New planner-local evaluator
  `scanZoneUpgrades` (`internal/planners/shop_upgrade.go`) scores every in-stock
  item across the mob's zone shops via `itemvalue.ItemValueDelta` — the highest
  positive delta naturally targets whichever slot benefits most, so no per-slot
  authoring — prices each with the buyer-side dynamic price `shops.CalcSellPrice`,
  and returns the best affordable positive-delta candidate (tie-break cheaper).
  New perpetual `upgrade-gear` goal type (`internal/goals/catalog/upgrade_gear.go`):
  `Predicate` always false, `ContextScore` a non-zero floor (1.0, so 4.6 dormancy
  never abandons it) rising to 2.5 when idle + has spendable gold or sellable
  loot; optional `reserve` param; cheap and self-contained (no shop scan in
  scoring — the planner owns stock decisions). New `upgrade-gear` planner
  (`internal/planners/upgrade_gear.go`): pending-equip one-shot → `gearup`;
  affordable upgrade → `buy <name>` at shop or `pathto`; unaffordable upgrade →
  composes the existing wealth-gold sell loop to save up; nothing in stock →
  idle. **Known gap (final-review):** the save-up `sell all` branch is a no-op
  for mobs — there is no `sell` mob command / `actions.Sell` yet, so it emits a
  confused emote (pre-existing in wealth-gold; [[project_mob_sell_command_missing]]).
  So in practice mobs buy upgrades from gold they ALREADY carry; the
  sell-to-fund path lands when the `sell` verb is lifted. Buy target rescanned
  each tick (only the save-up sell vendor is sticky; the buy branch clears that
  sticky to avoid stale vendors across cycles). The evaluator only considers
  items `gearup` can actually wear (Weapon/Wearable) so a mis-tagged shop item
  can't trigger a buy-but-can't-equip re-buy loop; the planner also idles for
  mobs that can't equip-from-give (non-combat / animal species). Two
  balance knobs `MobUpgradeGoldReserve` (50) + `MobUpgradeMinDelta` (1.0, may
  retune after live smoke). Seeded as a low-priority `default_goals` entry on
  `thief` + `guard_captain` (no btree edits — `try_goal_planner` already present
  in every non-boss tree). **Out:** cross-respawn persistence of bought gear
  (instance saves are wiped in prod/smoke, so upgrades persist only for the
  instance's lifetime — a remembered-loadout system is a separate design);
  adjacent-zone shopping (same-zone only); the legacy `mastery-equip`
  planner left untouched (now partially redundant; candidate for later
  deprecation). Follow-up logged: instance-zone items need to be sellable with
  affix-scaled value (separate chunk). **Unit tests** cover the pure helper +
  nil/empty/branch shapes; the live scoring/pricing path is validated by an
  in-game smoke deferred to the user (per the 2.8/2.9/2.10 precedent). Boot
  smoke clean (`Server Ready`, no panic). Spec at
  `docs/superpowers/specs/2026-06-01-mob-aliveness-5.3-equipment-aware-shopping-design.md`,
  plan at
  `docs/superpowers/plans/2026-06-01-mob-aliveness-5.3-equipment-aware-shopping.md`.

### 5.4 NPC market participation
**Status:** Done (2026-06-02) • **Size:** M

- **Goal:** NPCs sell looted/crafted goods through normal shop channels, contributing to the economy.
- **In:** Sell-trigger behavior, integration with shop pricing/stock, decay/clearance rules.
- **Out:** Player↔NPC barter beyond what shop UX already supports.
- **Depends on:** 5.3 (similar plumbing)
- **Why:** Living economy — shop stock reflects NPC activity, not just player drop-offs.
- **Shipped:** Three integrated components: (1) **`actions.Sell` verb lift** —
  shared seller entry point for players and mobs. Mob sales credit the seller's
  gold via normal merchant pricing and buy-rules but do NOT drain shop/merchant
  gold (`seller.IsPlayer()` gate), fixing the broken wealth-gold and
  upgrade-gear goal planners that previously had no sell path. `SellAllSellable`
  mode sweeps a mob's full backpack in one call. (2) **Time-based overstock
  decay** (`TickOverstockDecay` / `TickOverstockDecayWith` in
  `internal/shops/`) — unsold stock above the `RestockQty` baseline drains by
  `ShopOverstockDecayQty` (default 1) per entry per `ShopOverstockDecayRounds`
  (default 21600) rounds of inactivity (grace period measured from
  `StockEntry.LastGrewRound`). Crafting materials (`is_component`) are always
  excluded. NPC-dumped entries (RestockQty 0) drain fully to zero. (3)
  **Forager-chest aggregate backfill** (`BackfillVendorFromChests` in
  `internal/forager/`) — on restock ticks, pulls items from forager sanctuary
  lockboxes into the neediest vendor stock gaps (largest `MaxStock - Current`
  first), free of charge. Chest rooms are tracked via a self-populating
  `zone → lockbox-room` index (`chest_index.go`; populated at `StateStoring`
  time, not from the static profiles registry). Chest-full back-pressure:
  when a forager's lockbox exceeds `ChestBackpressureResumePct × ForagerLockboxCapacity`
  items, the forager stays resting until the backfill drains it to ≤ that
  fraction (the chest "overflow cache" now actually drains).
  **Deferred:** proactive surplus-offload goal (combat-looter/crafter
  sell-surplus goal + anti-thrash + donation bins) — dropped because mob
  sellable surplus is too thin today (mobs never inherit player gear); revisit
  when mob corpse-salvage yields a richer surplus stream. Spec at
  `docs/superpowers/specs/2026-06-02-mob-aliveness-5.4-npc-market-participation-design.md`,
  plan at
  `docs/superpowers/plans/2026-06-02-mob-aliveness-5.4-npc-market-participation.md`.

**Store-restock follow-ups:** The COOKING and ENCHANTING halves of the
store-restock fix (`project_store_restock_considered_fix`) are now
addressed:
- **Cooking (chunk 5.4):** cooks are crafter mobs fed by
  `crafterrestockmaterials` (salt, water, healer's root) plus
  forager-salvaged meat flowing through the chest-backfill pipe.
- **Enchanting (chunk 5.4, 2026-06-02):** alchemy vendors' unsold
  potions decay via `TickOverstockDecay`; the `[]DecayedUnit` return
  is converted to enchanting mats by `crafting.EnchantSalvageYield`
  (tiered by recipe `skill_minimum`, bands 1-4) and held in a global
  in-memory reserve (`shops.AddToReserve`). Enchanters with
  `craft_support: "enchanting"` draw from the reserve on their idle
  tick via `shops.SelectStockTransfers`, neediest stock-gap first.
  Player spoiled-potion salvage uses the same `EnchantSalvageYield`
  logic.

Two remaining open follow-ups:
- **(a) Fernway caravan `deliveries_by_tier` is empty** — the caravan
  YAML has an empty delivery-tier block; no cooking goods reach Fernway
  vendors from the caravan route.
- **(b) General-store restock** — broad restocking for non-cooking,
  non-enchanting merchant types remains open.

---

## Phase 6 — Audit & polish

Validate the framework against real content, then scale.

### 6.1 Stillwater town-flavor pass
**Status:** Done (2026-06-03) • **Size:** L

- **Goal:** First zone benchmark — every Stillwater NPC gets relationships, schedule, knowledge, optional goals.
- **In:** 19 non-quest Stillwater NPCs (per MEMORY.md), idle dialogue, daily routines, mutual relationships.
- **Out:** New quests (content separate).
- **Depends on:** Phase 1, Phase 3, optionally Phase 4
- **Why:** Validates the framework against real content. Catches what's hard to author. (Absorbed from MEMORY.md — Stillwater town-flavor pass.)
- **Shipped:** "Layered-by-fit" benchmark — each substrate layer applied where it
  earns its keep rather than uniformly. **Relationships (1.6):** ~10 edges across
  12 NPCs authored one-side-per-edge (engine auto-mirrors) — Voss-family kinship
  (Ulla↔Vella sister-in-law, plus cross-zone Ulla→Maren niece to Thornwall mob
  113), employment (Sigrid→Neva, Seren→Finn), Hodder's mentorships (Tov, Luc),
  healer/neighbor friendships (Ilsa/Vella, Gyda/Vella), Brindle/Hodder, and a
  petty Sigrid/Wulf rivalry. **Schedules (3.2):** 9 full 24h daily routines
  (the 8 anchors Sigrid, Neva, Brindle, Seren, Arn, Ilsa, Bram, Vella, plus
  Hodder added in review so the Tov/Hodder and Ilsa/Vella pairs co-locate)
  routing between existing
  rooms only — **zero new rooms** (rest/sleep targets reuse work rooms or the
  lodging loft; all routes pre-verified path-connected). `LoadSchedules` 5 to 14.
  **Knowledge/facts (1.4/1.7):** 5 standing facts seeded into `facts.yaml`
  (lake-decline, voss-death, spiral-motif, cave-creatures, pearl-divers-gone) with
  role-gated `knows_facts:` on 10 NPCs (NOT universal — the knowledge-model point),
  plus 2 new gossipers (Fenwick, Oswin) so the gossip pipeline spreads them.
  **Conversations (3.6):** 3 new generic type-pools authored where none existed
  (`employer`, `employee`, `family` — friend/rival already shipped) + 4 Stillwater
  pair overrides (Sigrid/Neva, Tov/Hodder, Ilsa/Vella, and the gentle Ulla/Vella
  grief pair that fires when Vella's schedule routes her to Ulla's parlor in the
  evening). `conversations.Load` pools 2→5, pairs 1→5. **Benchmark lessons:** the
  conversation engine randomizes speaker A/B, so all exchange lines must be
  swap-safe — the voice review caught several mentor/employer lines that hard-coded
  a speaker and they were rewritten role-agnostic; this is the chief authorability
  gotcha for 6.5's broader rollout. Faction membership was already wired
  (`groups: stillwater_citizens`/`stillwater_guards`); strategic goals (Phase 4)
  left out of scope. No engine changes — pure data. Boot-validated clean after each
  layer (no panics, `go test ./...` green). **Manual in-game smoke deferred to
  user.** Spec at
  `docs/superpowers/specs/2026-06-03-mob-aliveness-6.1-stillwater-town-flavor-design.md`,
  plan at
  `docs/superpowers/plans/2026-06-03-mob-aliveness-6.1-stillwater-town-flavor.md`.

### 6.2 Parity audit closeout
**Status:** Done (2026-06-03) • **Size:** S

- **Goal:** Final sweep of parity gaps after Stillwater pass exposes what's still missing.
- **In:** Document remaining gaps, log next-tier ones to MEMORY for later.
- **Out:** —
- **Depends on:** 6.1
- **Why:** Captures what we learned from real content use.
- **Shipped:** Scoped (per user) to the `CommandParity` boot-audit warnings,
  which had crept back up from the intended single `throw` warning. Three
  aliveness-introduced verbs were firing un-allowlisted: `goal` (Phase 4 admin
  mob-goal inspector), `fine` and `payfine` (5.1 jailed-player justice
  interactions). None warrant a mob equivalent — all three added to
  `userOnlyCommands` in `internal/actions/divergences.go` (`goal`=admin,
  `fine`/`payfine`=player-mechanic). `throw` deliberately left un-listed (with
  an explanatory comment) so it stays the single standing parity warning until
  the ranged-weapon system lands. Boot-verified: only `throw` warns. The broader
  combat-quadrant parity list is already structurally closed (four-handler split
  unified into `handleCombatRound`, 2026-04-18). The 6.1 authorability lessons
  (conversation A/B swap-safety, dead-pair co-location) are captured in the 6.1
  record + smoke followups rather than re-audited here.

### 6.3 Per-zone tuning (Thornwall deepening)
**Status:** Done (2026-06-03) • **Size:** M

- **Goal:** Apply the 6.1 framework to a second zone — Thornwall City. (Sanctum
  Basin was dropped: it's being fully replaced by the newbie-area rework.)
- **Shipped:** Relationship edges on 7 mobs; 5 new facts (renamed
  test-mayor→thornwall-mayor-disgraced) + knows_facts on 11 mobs + gossiper
  group on 2; 6 new schedules (market merchant/food vendor/apothecary/jeweler/
  weaver/guard captain) wired onto 6 mobs; 3 conversation pairs (marek_and_dal,
  fen_and_gobb, velk_and_merchant), all swap-safe. Flavor = public troubles
  only, no quest spoilers. Plus a gossip fix in MobIdle: the no-room-events
  branch now tries known facts before the generic idle fallback.
- **Out:** Every zone — that's 6.5. Sanctum Basin — newbie rework.
- **Depends on:** 6.1
- **Why:** Two zones reveal pattern. Three+ becomes process to delegate.

### 6.4 Performance review (initial)
**Status:** Done (2026-06-05) • **Size:** S

- **Goal:** Measure substrate state size, persistence cost, and tick budget after the framework lands.
- **In:** Profile, log key metrics, document baseline.
- **Out:** —
- **Depends on:** 6.3
- **Why:** Can't optimize what we haven't measured. Catch regressions before content pass scales them up.
- **Shipped:** Extended the existing always-on `util.TrackTime` / `util.AddMemoryReporter` infra (no new framework, no toggle — decision recorded in the baseline doc). 8 substrate memory reporters (opinions, factions, crimes, knowledge, bounties [nil-safe], facts [nil-safe], relationships, goals) surface store size + count in `server stats`; 5 tick sub-timers broken out of the lumped roll-ups (`IdleMobs::{schedule,patrol,conversation}`, `MobIdle::goalplanner`, `Enforcement`). Captured a re-runnable idle (Ct=524) + under-load (Ct=536, 1 player) baseline at `docs/perf/aliveness-perf-baseline.md`: aliveness substrate ~7 KB of a 2.1 MB non-Go total, every tick seam sub-microsecond avg — comfortable headroom before the 6.5 content pass. Conversations are the largest aliveness seam; the only large timer high (~67 ms `events.ProcessEvents()`) is an autosave/GC outlier, not aliveness. Spec at `docs/superpowers/specs/2026-06-05-mob-aliveness-6.4-performance-review-design.md`, plan at `docs/superpowers/plans/2026-06-05-mob-aliveness-6.4-performance-review.md`. **Bonus:** found a pre-existing concurrent-map-write crash in `templates.Process` (login path) during capture — fixed separately on `fix/templates-configcache-race`. Decomposed the follow-on 6.5 content pass + specced 6.5a faction definitions during the same session.

### 6.5 Content pass — broader rollout
**Status:** Not started • **Size:** XL

- **Goal:** Apply the framework to the rest of the game's zones and NPCs.
- **In:** Every remaining zone, schedule/relationship authoring, validation pass per zone.
- **Out:** —
- **Depends on:** 6.3
- **Why:** Scaling the formula across the world. The "and now actually populate it" step.

### 6.5a Faction definitions content pass
**Status:** Done (2026-06-05) • **Size:** M

- **Goal:** Author the rest of the world's factions on top of the
  1.2/1.3 substrate.
- **Shipped:** 8 new factions (13 total) — `bandits` (-35), `ironwind_tribe`
  (-25), `dustwalk_caravans` (+10), `road_wardens`, `shopkeepers` (the
  "Merchants' Concord"), `ashwick_villagers`, `watchers_crossing`, and
  `bloodline_agents` (neutral placeholder for the future-major Bloodline force;
  lone member `north_road/287` tagged). Wired a **unified law-bloc clique** (9
  civilization factions mutually allied, linking the previously-islanded
  Thornwall/Stillwater blocs) vs a loose **outlaw cluster** (bandits ↔
  ironwind_tribe, enemies of all 9). **Corrected warren** — removed the legacy
  `warren ↔ thornwall_guards` enemy edge; the Warren are insular/discriminated-
  against (negative default_rep) not outlaws. Member-tagged bounded factions
  (bandits ×9, ironwind goblins ×4, caravan crew ×4 sentient, ashwick ×4,
  watchers ×4 incl. 2 dual shopkeepers, road warden, bloodline) + scan-driven
  `shopkeepers` on all 15 non-sanctum shop-owners (dual-membership with their
  town citizenry); dropped the tag from caravan props (wagon/draft horses).
  Boot-validated (loader panics on bad ally/enemy refs — clean; `faction list`
  confirms the full graph in-game). Followups: `278-haral` (tavern) should join
  shopkeepers once it gets a `shop:` block in the roads batch; combat-rep in-game
  smoke deferred to user. Spec at
  `docs/superpowers/specs/2026-06-05-mob-aliveness-6.5a-faction-definitions-design.md`,
  plan at `docs/superpowers/plans/2026-06-05-mob-aliveness-6.5a-faction-definitions.md`.
- **In:** YAML faction definitions for bandits, warden, ironwind
  shaman, Sanctum Basin guards, Dustwalk caravans, Stillwater
  militia & citizens, etc. Tag remaining faction-relevant mobs
  with their `groups: [<faction_id>]`. Define ally/enemy graphs
  across the full set. Surface any schema gaps the substrate
  didn't anticipate.
- **Out:** Per-faction quests (own content chunk).
- **Depends on:** 1.2, 1.3
- **Why:** 1.2 ships substrate + warren + thornwall_guards. 1.3
  adds thornwall_citizens + alliance-aware guard logic. Bulk
  authoring the rest now would risk schema churn — better to
  validate the substrate against two reference factions, then
  bulk-author once the pattern is settled.

### 6.6 Performance re-review
**Status:** Not started • **Size:** S

- **Goal:** Re-profile after content pass — load profile changes once you have many active goal-driven NPCs across many zones.
- **In:** Re-run profiling, compare against 6.4 baseline, optimize hot paths if needed.
- **Out:** —
- **Depends on:** 6.5
- **Why:** Goal engines + persistent state + schedules can compound. Catch degradation while we still have headroom.

---

## Parallel arc — Combat state machines + messaging framework

A multi-chunk infrastructure arc that ran in parallel with Phase 1
substrate work on the `feature/mob-aliveness-1.3-crimes` branch. Not
part of any aliveness phase, but tracked here because it shipped on
the same branch and unblocks downstream aliveness work (mob position
narration, dormant-mob hibernation, sight-gated infrared rendering,
color-coded combat text, etc.).

| Sub-chunk | Title | Date | Status |
|-----------|-------|------|--------|
| 0  | Combat state-machine framework (chunk 0 substrate) | 2026-05-13 | Done |
| 1  | Awareness state machine | 2026-05-14 | Done |
| 2  | Combat-phase state machine | 2026-05-15 | Done |
| 3  | Life state machine | 2026-05-15 | Done |
| 4a | Position FSM scaffold (dormant) | 2026-05-16 | Done |
| 4b | Position writers + readers cutover | 2026-05-16 | Done |
| 4c | Weapon-reach utility + position rebalance | 2026-05-16 | Done |
| 4d | Submissions + outside-damage drift | 2026-05-17 | Done |
| 4e | Third-party interactions | 2026-05-18 | Done |
| 4f | Position balance + spell disruption | 2026-05-19 | Done |
| 5  | Presence state machine (AFK + mob hibernation) | 2026-05-19 | Done |
| 6  | Perception state machine (dormant) | 2026-05-19 | Done |
| 7  | Centralized messaging framework | 2026-05-20 | Done |

**Chunk 7 — Centralized messaging framework (2026-05-20):**
Successor to chunk 6's dormant Perception FSM. Built
`internal/messaging/` (Category enum, 7-stage pipeline,
sight predicates, infrared anonymizer, ANSI-aware wrapper,
categorized SendText API). Migrated ~2300 callsites across
combat/, hooks/, mobcommands/, usercommands/, rooms/, actions/,
behaviortree/, questengine/, modules/, world.go. Deleted all
legacy SendText shims and duplicate helpers (canSeeInRoom × 2,
sendRoomTextDarknessAware, wrapText, *.SendTextLegacy, etc.).
Fixed the long-standing companion-name leak (pet names no longer
visible to blind/dark-room observers). Spec at
`docs/superpowers/specs/2026-05-19-messaging-framework-design.md`.

Arc complete: chunks 0-7 all shipped on the same feature branch.
Mob-aliveness Phase 1 substrate work fully resumes from here.

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
- **`context.md` is required per chunk.** Every chunk that creates a new `internal/<package>/` directory MUST ship a `context.md` documenting the package, in the established DOGMud style. Chunks that meaningfully modify an existing package (extend its API surface, add new files, reshape its data model) MUST update the existing `context.md` to match. Style references — copy the section structure from one of these:
  - `internal/badinputtracker/context.md` (~170 lines) — small, single-responsibility package
  - `internal/clans/context.md` (~190 lines) — medium, multi-file package
  - `internal/buffs/context.md` (~700 lines) — large, deeply-integrated package
  Required sections: Overview, Key Components (file map), Key Functions (signatures + behavior), Global State (if any), Data Structure Design (schemas + YAML shapes), Integration Notes (which packages consume / are consumed by), and Testing Notes. Aliveness chunks 1.1–1.5 missed this; they're being backfilled in chunk 1.6's plan.
