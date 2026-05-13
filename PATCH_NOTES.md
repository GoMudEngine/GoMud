# DOGMud Patch Notes

## 2026-05-12 — Phase 2 tactical: chunks 2.4 + 2.5 + 2.6

Three Phase-2 tactical chunks shipped on
`feature/mob-aliveness-1.3-crimes` in one day (14 / 41 done).
Branch carries chunks 1.1–2.6; merged to `development`, not yet
pushed to prod.

### 2.4 — Mob `consider` + threat-aware behaviors

Consolidated the `consider` math via the actor pattern (mirroring
chunk 2.1's `buy` consolidation) so players and mobs share the
same code path. Reframed from the original "covet a player's
gear" half (dropped — players don't drop gear) into reactive
lookout and opportunistic predator patterns.

- New shared function `actions.Consider(actor, target) → ConsiderResult`.
  Player wrapper collapses to ~15 lines (~830 lines deleted);
  `internal/mobcommands/consider.go` is the parallel mob wrapper
  (`MobActor.SendText` no-op so the math runs silently).
- New btree primitives: `target_power_ratio_above` /
  `target_power_ratio_below` (condition) and
  `target_weakest_mob_in_room` (action). Target resolution chain:
  `Event.UserId` → `Aggro.MobInstanceId` → `Aggro.UserId`
  (matches `actions.ResolveAggroTarget`).
- `mob.HatesMob` predicate gates predation — faction/pack
  awareness without coupling to the 1.2 substrate.
- Demo wiring: `lookout` archetype gains `player_enter` ambush
  branch (`target_power_ratio_above: 1.0` — outclass before
  ambushing); new `predator` archetype copies generic_fighter
  + adds a leading `mob_idle` predation branch
  (`ratio_below: 0.85`); 3 ironwind wolves (steppe 205, young
  206, scarred 223) flip to predator. Alpha wolf 215 retained
  `leader` archetype to preserve rally/warcry behavior; future
  `predator_leader` hybrid logged as follow-up.
- PowerScore audit deliverable: new "Power Scoring & Gear
  Contribution" section in `internal/combat/context.md`
  documenting how equipment flows through the existing
  `ValueAdj` / mitigation pipes (no math changes).

Spec at `docs/superpowers/specs/2026-05-12-mob-aliveness-2.4-mob-consider-design.md`,
plan at `docs/superpowers/plans/2026-05-12-mob-aliveness-2.4-mob-consider.md`.

### 2.5 — Mutations on mobs (body-plan gating + intrinsic mutations)

Generalized chunk 2.2a's incorporeal-only mutation support into
a full body-plan gating model. Species declare what body parts
they have; mutations declare what they require. Species can
additionally declare intrinsic mutations that stack additively
with acquired mutations at character init.

- Schema: `Species.BodyParts []string` + `IntrinsicMutations
  map[string]int` (yaml `body_parts:` / `intrinsic_mutations:`).
  Canonical seven-tag set: `arms, hands, legs, eyes, mouth,
  skin, tail`. `MutationSpec.RequiresArms bool` → `RequiresBodyParts
  []string`. Boot-time validation panics on unknown tags or
  unknown mutation ids in intrinsic_mutations.
- New: `Character.ApplyIntrinsicMutations(species)` helper —
  cap-aware additive merge at init time (`MutationMaxRank = 4`,
  chunk-2.2a convention). Called from mob spawn + player creation.
- Gating sites: random-roll pool (`GetWeightedPool` now takes
  `*species.Species`), curated `SpawnMutations` path on mob YAMLs
  (latent bug fix — was applying unconditionally), 4 mid-game
  acquisition sites in user-tick / btree quest action / quest
  engine bridge.
- Migration: all 35 existing species + 4 new elemental species
  (sand, storm, ice, smoke) — total 39 species YAMLs touched.
  17 mutation YAMLs gained `requires_body_parts:` declarations.
  Five `instance_planar_oasis` elemental mobs repointed: king
  kept on magma + new mob-YAML `spawnmutations: [large]` (top-
  level, NOT under a `mutations:` map — caught in smoke); queen
  moved to new ice species (dropped her chunk-2.2a
  `incorporeal:4` override since her crystal/water form is
  corporeal per description); prince moved to new smoke species.
- Cleanup: redundant `mutations: { incorporeal:4 }` overrides
  removed from 4 `summons` mobs (wraith/spectre/fire/air) —
  incorporeal is now intrinsic on the species.

Spec at `docs/superpowers/specs/2026-05-12-mob-aliveness-2.5-mutations-on-mobs-design.md`,
plan at `docs/superpowers/plans/2026-05-12-mob-aliveness-2.5-mutations-on-mobs.md`.

### 2.6 — Sunset legacy tactics engine

Reframed from the original "fix the Edrin priority race"
band-aid into the structural fix: deleted the legacy
`internal/mobai/` tactics engine entirely and migrated all 44
tactic-using mobs to the behavior tree (btree) system. The
Edrin priority-race bug is now structurally impossible (btree
selectors are inherently priority-ordered, no async reaction
queue racing `InitiateCast`).

- ~1,144 net lines of legacy code deleted. `internal/mobai/`
  directory removed entirely (10 files: tactics.go, reactor.go,
  actions.go, types.go, memory.go, triggers.go + tests). The
  `CombatMemory` substrate (grudge tracking across flee /
  re-engage) was preserved at `internal/mobs/combat_memory.go`
  — it was used outside the tactics engine.
- Zero new btree primitives — existing `mob_has_buff` + invert
  decorator covers the legacy `missing_buff:N` trigger.
- 5 existing archetypes (`generic_fighter`, `predator`,
  `leader`, `lookout`, `tank_taunter`) gained a shared
  `mob_hurt + mob_health_below:25 → flee` panic-flee branch as
  FIRST child. tank_taunter additionally gained a
  `mob_hurt + mob_health_below:20 → callforhelp` branch
  (absorbing the `tank` preset). ambusher gained a
  `mob_combat_round + target_is_casting → trip` branch
  (absorbing the `ambusher` preset's third rule).
  Post-smoke fix: panic-flee REMOVED from tank_taunter — flee
  preempted callforhelp at the threshold boundary; tanks
  semantically shouldn't flee.
- 1 new shared archetype `defensive_caster` absorbs 4 mobs
  from the `defensive_caster` and `caster_backline` presets
  (goblin_shaman 219, tunnel_shaman 74, bandit_caster 285,
  elemental_queen 321).
- 5 new per-boss archetypes preserve unique spell rotations:
  `boss_edrin`, `boss_sylara`, `boss_rhett`, `boss_soren`,
  `boss_chrysalis_phantom`.
- 44 mob YAML migrations strip `tactic_preset:`, `tactics:`,
  `reaction_delay:`, `tactical_discipline:`. The
  corresponding 4 `Mob` struct fields removed.
- Known follow-up: Edrin/Sylara's `conviction-ward` opening
  cast has no buff self-gate (conviction-ward is a shield
  spell with no `buff_id`). Bosses re-cast wastefully after
  the shield expires — behavior is not broken, just wasteful.
  Polish item for a future tuning pass.

Spec at `docs/superpowers/specs/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine-design.md`,
plan at `docs/superpowers/plans/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine.md`.

---

## 2026-05-09 — Phase 1 substrate complete (chunks 1.6 + 1.7)

Two more aliveness substrate chunks shipped on `feature/mob-aliveness-1.3-crimes`,
closing out **Phase 1** of the MOB_ALIVENESS_ROADMAP (7 / 40 done).
Branch carries chunks 1.1–1.7; not yet merged to development.

### 1.6 — NPC-to-NPC Relationships

Mob templates now declare a kinship/friendship/rivalry/lover/employer-
employee graph inline in their YAML. Engine builds an in-memory
graph at startup with auto-mirror — symmetric edges (family, friend,
rival, lover) reverse the same type, asymmetric (employer ↔ employee)
flip. Subtype is per-side flavor ("brother," "wife," "drinking-companion").
Permissive validation: bad edges warn-not-panic.

- Public API: `RelationsOf`, `RelationsOfType`, `KinOf`, `AlliesOf`,
  `RivalsOf`, `RelationsBetween`, `AreRelated`, `EmployerOf`,
  `EmployedBy`, `AllRelations`, plus mutation `Add` / `Remove` /
  `ChangeType` (in-memory v1; persistence overlay deferred).
- Admin command `relationship show / between / add / remove / list`
  + helpfile.
- Future consumers: 4.5 reactive goal seeding (revenge), 3.6 NPC↔NPC
  idle conversation.

Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.6-npc-relationships-design.md`,
plan at `docs/superpowers/plans/2026-05-09-mob-aliveness-1.6-npc-relationships.md`.

### 1.7 — World-Model Facts

Standing-fact registry plus per-NPC awareness store. The
`recentGossipEvents` TempData that the gossip pipeline used for
event dedup is now a persistent on-disk awareness file —
`heard_events` (bounded FIFO via `FactsHeardEventsMax`, default 32)
sit alongside `known_facts` in the same `facts.awareness/{mobId}.yaml`.
`buildGossipLine` was migrated; existing event-only output is
preserved, and known facts now mix into the gossip candidate pool
(70% events / 30% facts when both pools are populated).

- Three withdraw signals: manual `Withdraw`, time-based
  `expiry_round` + `PruneExpired` sweep, auto-withdraw via
  `withdraw_on_respawn_of` field on the fact (new RoomChange
  listener fires when the bound mob's instance enters a room).
- Lazy-filter on read for awareness × registry join — withdrawn /
  expired facts are skipped without active cleanup.
- Worldevents got a stable `Id uint64` field (atomic-counter at
  emit time) so awareness can reference events by id.
- Mob YAML extension: `knows_facts: [factId, ...]` for inline
  authoring; seeded into awareness at `mobs.LoadDataFiles`.
- Admin command `fact list / show / declare / withdraw / expire /
  prune-expired / awareness / teach / forget / forget-all` +
  helpfile.
- New `fact-default` gossip template family ("I heard {description}"
  / "Word is, {description}" / "They say {description}").

Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.7-world-facts-design.md`,
plan at `docs/superpowers/plans/2026-05-09-mob-aliveness-1.7-world-facts.md`.

### Backfilled `context.md` for chunks 1.1–1.6

Per a new aliveness-roadmap maintenance rule: every chunk that
creates a new `internal/<package>/` ships a `context.md` in the
established DOGMud style. Chunks 1.1–1.5 missed this; chunk 1.6's
plan included the backfill. Now present:
`internal/{opinions,factions,crimes,knowledge,bounties,relationships,facts}/context.md`.

### Path-doubling fix for chunks 1.4 / 1.5 / 1.7

Chunk 1.7's review caught that knowledge, bounties, and facts
packages all added `world/dogmud` to a `DataFiles` config that
already includes it — runtime data was landing at
`_datafiles/world/dogmud/world/dogmud/{knowledge,bounties.yaml,facts.yaml}`.
Same bug class that chunk 1.2 fixed for opinions/factions originally.
All three packages now use the unwrapped path. No data migration
required since chunks 1.x are feature-branch only and never reached
prod.

## 2026-05-06 — Crafter output routes to shop

Hotfix for a regression where crafter mobs' gear-grade crafts (iron
daggers, bucklers, leather armor — anything with combat stats)
landed in the mob's backpack instead of shop stock. Players saw
only the sub-component crafts (steel ingots, etc.) in the shop list.
Cause: the Priority-1 self-gear-upgrade selector flagged gear-grade
recipes as "upgrades" for the mob whenever an equipment slot was
empty, which is most slots on a shopkeeper crafter like Kerra. The
craft success path then routed to `mob.Character.StoreItem` rather
than `shopInv.AddStockAtRound`. Fix: all crafted output now lands
in shop stock regardless of craft selection priority. Priority-1
still triggers gear-grade crafts even when narrowly unprofitable
(so shops carry actual weapons/armor, not just intermediates).

## 2026-05-05 — Economy Scoring Refactor

Five-axis economy health scoring replacing the single weighted-fill score
on the admin dashboard. Spec at
`docs/superpowers/specs/2026-05-05-economy-scoring-refactor-design.md`,
plan at `docs/superpowers/plans/2026-05-05-economy-scoring-refactor.md`.

### What changed

- **Five score axes** — Stock, Throughput, Input Rate, Logistics Health,
  Shop Gold — each answering a different question. Old `PerShopScore` is
  retained as `StockScore` (just renamed in the dashboard).
- **Throughput score** measures Time-to-Refill (TtR): the player-facing
  "how long does iron ore stay out of stock" experience, weighted heavily
  toward common materials so crafting-grind viability drives the score.
- **Input rate score** measures items entering the supply per game-day
  (forager deliveries + restock cycles), per zone, against an
  auto-derived target.
- **Logistics health** for caravans and foragers is now a composite of
  cycle rate and cargo flow with hard multipliers for stuck (×0.4) and
  despawned (×0). Halix-despawning and Kessa-stuck failure modes now
  read near-zero on the dashboard instead of moderate.
- **Shop gold score** surfaces merchant liquidity: a merchant with no
  gold can't buy from players, which was previously invisible.
- **Per-rarity-tier restock cadence** replaces the single global ticker.
  Commons (tier 50) refill every 1 game-hour; rares (tier 10) every 5
  game-days as a slow backstop above forager/sale input.

### Data layer

- New persistent counters on `ShopInventory`: `SalesCount`, `BuysCount`,
  `RestockCount`, `StockEvents` (rolling 7-game-day window),
  `CurrentDepletion`. Drives TtR scoring.
- `LbsDelivered` cumulative counter on caravans and foragers. Drives
  logistics cargo-flow scoring.
- All zero-default; existing shop YAMLs and snapshots load cleanly with
  no migration step.

### Dashboard

- 5-card top row: Overall, Stock, Input, Throughput, Shop Gold.
- New Throughput table: per-shop TtR by tier band, per-window medians.
- New Input Rate table: per-zone rate with source breakdown
  (forager / restock) and tier mix.
- Logistics panels gain Multiplier and Lbs Delivered columns.
- Stock table relabeled and gold-score column added; noisy
  per-window stock-score-delta cells removed.

### Configuration

New `Balance` knobs (all overridable in `_datafiles/config.yaml`):
`RestockCadenceTier{50,40,30,20}Hours`, `RestockCadenceTier10Days`,
`TtRTargetTier{50,40,30}Hours`, `TtRTargetTier{20,10}Days`,
`TtRWindowGameDays`, `LogisticsStuckRounds`, `LogisticsStuckMultiplier`,
`ScoreWeight{Stock,Input,Throughput,ShopGold}`. Defaults set in code so
old config files continue to load.

### Smoke-test follow-ups

Same-day fixes that surfaced once the new dashboard had eyes on
zone-by-zone health:

- **Eager-spawn shop-bearing rooms at boot.** Shopkeeper mobs in zones
  without players were never instantiated — only the explicit
  `systemNPCAnchorRooms` list (foragers + caravan) got pre-spawned.
  Result: Stillwater, Sanctum Basin, Watchers Crossing showed flat
  zero on the input rate dashboard because no `MobIdle` ticks ever
  fired for their shopkeepers. `main.go` now scans every loaded room
  at boot and `Prepare()`s any with `HasShop()` spawninfo.
  `IsEssential()` extended so shop-bearing rooms aren't unloaded by
  `RoomMaintenance`.

- **Crafter consumption tracking.** Crafter mobs now use round-aware
  `RemoveStockAtRound` / `AddStockAtRound` so their consumption
  marks `CurrentDepletion` and refills push `StockEvent`s — the
  TtR throughput score sees crafter-caused depletions instead of
  only player buys. New `ConsumedByCrafterCount` counter on
  `ShopInventory` captures per-shop crafter outflow. `SaveShop` is
  called after successful crafts (and salvage) so freshly-crafted
  output items survive server restart.

- **Per-ingredient stock-reserve floor.** Crafter mobs no longer
  drain their own ingredient stock to a single unit. New
  `CrafterIngredientReservePct` config knob (default 0.25) keeps
  at least 25% of `MaxStock` of each ingredient available for
  player purchase. Per-ingredient check, hard floor of 1 for tiny
  `MaxStock` cases.

- **Unrated-item tier-50 fallback.** 145 of 213 weapons / armor /
  consumables predate the rarity_tier system and have no tier
  field — they were invisible to the new per-tier `RestockTier`
  ticker. `RestockTier` now defaults `RarityTier == 0` to 50
  (commons, hourly cadence). Tutorial NPCs (Adela's tutorial gear)
  restock again. Per-tier audit + tagging tracked as a follow-up.

- **Forager fixes.**
  - Halix marked `non_combatant: true` so hostile steppe mobs can't
    aggro. He was dying mid-cast-fold-recall to goblin/wolf hits;
    death calls `Suicide` which permanently destroys the instance.
  - Kessa & Tova got a direct-teleport recall replacing
    `cast fold-recall`. The fold-cast machinery only advances
    inside the combat loop, so foragers recalling outside combat
    set `CastingState` and froze forever. Direct teleport bypasses
    the cast system entirely.
  - `ForagerRestDurationRounds` exposed as a config knob (default
    40 rounds, down from a hardcoded 120) for cycle-pacing tuning.

- **Prewarm zone-key fix.** `prewarmThroughputFromPersistedFiles`
  for both forager and caravan keyed the cache by snake_case
  directory name while runtime callers use display-name zone via
  `mob.Zone`. Result: phantom duplicate cache entries and
  `SaveAllThroughputs: no cached entry` errors on every shutdown.
  Prewarm now keys by the YAML's `zone:` field (display form) so
  prewarmed and runtime entries share a key.

- **Validator fix for shared component tags.** `ValidateRecipeIngredientTags`
  picked one arbitrary item per `ComponentTag` for validation.
  Map iteration is randomized in Go, so when multiple items shared
  a tag (four bottles all tagged `bottle`), boot intermittently
  panicked if the chosen item lacked the recipe's discipline.
  Validator now checks "at least one item with this tag has the
  discipline" — eliminates the boot-roulette panic.

- **Stillwater bank.** Town tier-1 settlement now has a counting
  house bank with NPC clerk (mob 356, room 5100). Tutorial-style
  bank flavor matching the existing Thornwall bank pattern.

- **Multi-quantity sell** (`sell 5 iron-ore`, `sell all iron-ore`,
  `sell all.iron-ore`) mirroring the existing multi-buy parser.
  Bare `sell all` with no item rejected as too easy to fat-finger.

- **Bank storage stacking.** Stackable items (components, food,
  drink, potions, grenades, ammo) now consume one slot per stack
  in the player's bank instead of one slot per unit. Lazy on-load
  migration of legacy `Items []Item` into the new `Slots
  []StorageSlot` shape. Storage fees billed per slot. No flag day.

## 2026-05-04 — Vendor Polish Hotfix

Two post-merge fixes caught during smoke testing the same day.

### Bug fixes

- **Shopkeepers re-seeding at 500g instead of YAML values.** Three
  layers of bug:
  - `RegisterMobShop` hardcoded `startingGold = 500`, ignoring
    `mob.Character.Gold` from the YAML — Phase 7's bumps to 1000g
    specialist / 5000g general never flowed through.
  - `PrewarmShopForSpawnPlacement` (which runs at boot for every shop
    placement before real mobs spawn) built a synthetic Mob without
    forwarding `Gold`, so even after the first fix the prewarm path
    re-floored everything to 500g.
  - Persisted shop YAMLs from the buggy boot survived a server
    restart; only a fresh wipe of `_datafiles/world/dogmud/shops/`
    forces re-seed from the (now-correct) seeding code.

  Specialists now correctly seed at 1000g, generals at 5000g.

- **Vendors rejecting tagged items they should buy** (Maren refusing
  cattail cloak, Kerra refusing arena tower shield). `EvaluateBuyRules`
  was pricing items not in the vendor's stock list at the 5× scarcity
  ceiling (`current=0, restock=1` → ratio 0 → `PriceCeiling = 5.0`),
  pushing the buy offer above the gold-reserve floor and self-rejecting.
  Now: stocked items still get full dynamic scarcity pricing; unstocked
  items use flat `value × BuyRatio`. Opportunistic vendor buys work
  regardless of whether the shop normally stocks the item.

### Deploy step

Wipe the shop save directory on prod **before restarting** so the
gold reseeds from the new code path:

```bash
./tools/economy/wipe_shop_state.sh
```

Players' personal gold and inventories are untouched.

## 2026-05-04 — Vendor Types & Economy Polish

Big economy overhaul. Buy rule rewrite, per-vendor audit, tier-50/40
baseline restock layered onto caravan/forager flow, forager stuck-state
watchdog, dashboard rework with stock-score-delta + per-rarity-tier
throughput bars.

### Vendor system overhaul

- **Item-side `vendor_categories` tag.** Every salable item now carries
  one or more discipline tags (`alchemy`, `blacksmithing`, `cooking`,
  `enchanting`, `jewelcrafting`, `tailoring`). Cross-cutting mats like
  iron ingot are multi-tagged (`[blacksmithing, jewelcrafting,
  tailoring]`). Lore / flavor / removed-from-game items get
  `not_salable: true` instead of relying on value gymnastics.

- **New buy rule.** Single-rule tag-overlap replaces the old 5-rule
  chain (quest / craft-material recipe-walk / gear-upgrade / potion /
  general-fallback). Reject conditions in order: quest item, declining
  potion, untagged item, vendor's `craft_support` doesn't match any of
  the item's tags, vendor at MaxStock, or buying drops below gold
  reserve. Removes ~80 lines of rule-helper code.

- **No more gear-upgrade purchases.** Specialist shopkeepers no longer
  buy random equipment — they're non-combatants who never wore what
  they bought anyway. The behavior was vestigial.

- **Apothecary Ilsa now buys all alchemy mats.** Previously the
  recipe-walk gating rejected mats not in her specific recipe list.
  The new tag-overlap rule accepts every alchemy-tagged item.

### Per-vendor audit

- **6 NPCs reframed as questgivers / flavor mobs:**
  Korvath (52), Yenna (53), Sigrid (333), Haral (278), Whisper (273),
  Bram (348). Stripped `crafter` / `crafterskill` /
  `crafterrecipeids` / `crafterrestockmaterials` / `craft_support` /
  `shop` fields. Korvath + Yenna keep `non_combatant: true` for
  questline integrity. Bram drops the `noncombat_shopkeeper`
  archetype entirely.

- **Specialist shopkeeper gold bumped to 1000g** (was 200/300/500
  variable). 12 NPCs: Kerra, Voss, Thornwall food vendor, Tess, Vael,
  Maren, Brynn, Tov Brann, Brindle, Ilsa, Edda, Kess.

- **General store gold bumped to 5000g** (was 400/500). 4 NPCs:
  Adela, Brecca, Siv (fence), Wulf.

### Restock pacing

- **Tier-50 and tier-40 mats now refill at every shop** on the
  existing crafter tick. Caravan-served zones (Stillwater, Thornwall)
  used to fully suppress that path, leaving common mats reliant on
  caravan/forager throughput. Now baseline tier-50/40 supply layers
  on top, while rarer tiers (30/20/10) still flow through caravan +
  forager exclusively.

### Forager reliability

- **Stuck-state watchdog.** Foragers wedged in any state for more
  than `ForagerStuckThresholdRounds` (default 600) get force-reset
  to Recalling — they head home, dump satchel into the lockbox, and
  re-cycle. Logs a `Warn` on every reset. Should end the periodic
  Halix "(not active)" mystery.

- **Dashboard distinguishes despawned vs idle.** A forager whose
  mob isn't currently spawned shows `(despawned)`; a live mob with
  empty BTreeState shows `(idle, no state)` plus a structured Warn
  log. Adds `StuckRounds` field for at-a-glance stuck detection.

### Dashboard rework

- **Stock-score-delta replaces gold-delta** in the per-window
  columns (1h/6h/1d/3d/1w). Each shop's `StockScore` is
  `sum(Current) / sum(MaxStock)`; the column shows the change in
  percentage points between snapshots. Gold value still visible as a
  static column.

- **Tier-color bars replace bucket-color bars.** Five tier classes:
  50 (grey), 40 (green), 30 (blue), 20 (purple), 10 (gold). Applied
  to shop stock bars and to new caravan/forager throughput rows.

- **Per-rarity-tier throughput bars** for caravan + forager. Each
  delivery to a destination shop bumps a per-tier counter on the
  corresponding mob; dashboard renders the window-delta as a
  proportion-stacked tier bar. Caravan pickups don't count — only
  destination drop-offs.

- **Names work for un-spawned shopkeepers.** When the NPC isn't
  currently in the world, the dashboard now falls back to the mob
  template's name instead of showing `#<mobId>`.

### Persistence

- **Two new gitignored runtime directories:**
  `_datafiles/world/dogmud/foragers/<zone>/<mobId>.yaml` and
  `_datafiles/world/dogmud/caravans/<zone>/<mobId>.yaml`. Track
  cumulative `DeliveriesByTier` counters across reboots. Boot
  prewarm + graceful-shutdown save mirror the shops/ pattern.

- **`NotSalable bool` field on ItemSpec.** Replaces the brittle
  "Value <= 0" skip in the vendor validator with an explicit opt-out.
  38 lore / flavor / legacy items got `not_salable: true`.

### Manual deploy step

⚠️ **Before starting the new server on prod**, wipe the persisted
shop save state so shops re-seed fresh from the new mob templates:

```bash
./tools/economy/wipe_shop_state.sh
```

Or directly: `rm -rf _datafiles/world/dogmud/shops/`. Players' personal
gold and inventories are untouched — only NPC shop state (gold drift,
current stock levels, last-restock round) is reset.

### Migrations

- **Recipe discipline shuffle.** `master-lockpicks` moved from
  jewelcrafting → blacksmithing; `reinforced-disarm-kit` moved from
  blacksmithing → jewelcrafting. Players who learned either recipe
  under the OLD discipline get their NEW-discipline skill bumped to
  the recipe's minimum (20 for master-lockpicks, 15 for
  reinforced-disarm-kit) so they don't lose craft access. One-shot,
  gated by misc-data key.

### Validators

- **`items.ValidateVendorCategories`** — boot-time check that every
  salable item carries a valid `vendor_categories` value. Cold boot
  panics on offending items; reload logs structured Error.

- **`crafting.ValidateRecipeIngredientTags`** — ensures every recipe
  ingredient resolves to an item carrying the recipe's discipline.
  Catches typos like `item_tag: lake-mintt` at boot.

## 2026-05-02 — Forager + Caravan Followup

Five fixes in the now-shipped forager + caravan stack, plus a caravan-cadence
tweak.

### Bug fixes
- **Whisper off the caravan rotation.** Room 507 (The Listening Post) is
  Whisper's quest-only spot in the locked, trapped, phantom-guarded section
  of Thornwall — never a standard merchant. Previously the caravan tried to
  restock her on every Thornwall pass. She's now removed from
  `thornwallVendorRooms`.
- **System NPCs spawn at boot.** Caravan master (room 4042) and the three
  foragers (Tova/4123, Halix/3040, Kessa/4197) now have `room.Prepare(false)`
  fired against their anchor rooms during the boot data-file load. Previously
  these mobs only spawned the first time a player walked into the room — and
  forager anchors are wilderness, so they could go offline indefinitely. The
  `/admin/economy/` dashboard's "(not active)" forager rows are gone after a
  clean boot.
- **Foragers no longer deadlock at sanctuary.** Stage 3.4's carry-ratio
  rest-extension would park a forager forever if vendors were saturated and
  her satchel never drained. Foragers now dump satchel surplus into a new
  per-sanctuary lockbox container on Recall arrival, so the satchel always
  empties between cycles. The carry-ratio gate is retained as a backstop for
  the lockbox-full case.
- **Kessa actually delivers to the caravan.** The previous mechanism required
  Kessa and the caravan to coincide at North Road 4038 in the same dwell
  window, which never happened reliably. New mechanism: Kessa drops her
  fernway-bucket items into a persistent **shipping crate** at 4038 and
  heads home; the caravan drains the crate into its wagon on its next
  pass. No timing dependency. The flag-based `caravan_load` mechanism is
  deleted entirely — real items now move through the wagon (Stage 3.4
  vendor-restock path).

### Content
- **Sanctuary lockboxes** at the three forager anchor rooms (4123 Stillwater
  Temple, 3040 Ironwind Steppe, 4197 Forager's Camp). Difficulty-10 lock,
  500-item capacity, fresh combination each forager cycle (the lock's
  `RotationSeed` bumps on every dump, invalidating any cached keyring
  entries). Players who pick the lockbox get the forager's surplus
  materials but must redo the picklock minigame each cycle.
- **Roadside shipping crate** at North Road 4038 (`crates/4038-fernway_shipment.yaml`).
  Visible to players as a noun, but every interaction (`get`, `look in`,
  `put`, `picklock`, `lock`) returns flavor text — only the caravan and
  Kessa modify it via state-machine code.

### Tuning
- **`CaravanDepotDwellRounds: 720 → 360`.** Halved. Foragers now run from
  boot and never deadlock, so they dominate day-to-day throughput regardless
  of caravan cadence. Halving the depot dwell roughly doubles caravan
  visibility in each town — event-style deliveries beat once-per-day realism
  here.
- **New config knob `ForagerLockboxCapacity` (default 500).** Caps the
  per-forager sanctuary lockbox; if a player goes a long time without
  picking a forager's lockbox open, the box can saturate and the forager
  reverts to rest-extension behavior until space opens up.

### Engine
- **`gamelock.Lock.RotationSeed`** added. `SetLocked()` bumps it. When >0,
  it's mixed into the `util.GetLockSequence` hash so a re-locked container
  produces a new combination — invalidating any cached keyring entry.
  Default 0 keeps every existing lock's combination unchanged.
- **New package `internal/sealedcrate/`.** Player-untouchable, room-bound,
  capacity-bounded delivery container. Persists at
  `_datafiles/world/dogmud/crates/<roomid>-<label>.yaml`. Mutated only by
  forager + caravan tick functions; all player commands short-circuit.
- **`Room.SealedCrate`** field + `Room.AttachSealedCrate` + boot loader at
  `main.go` end of `loadAllDataFiles`.

## 2026-05-01 — Mob Aliveness Roadmap (planning)

**Note:** Planning doc only — no engine, content, or config changes.

- **`MOB_ALIVENESS_ROADMAP.md`** added at project root. Long-term plan
  for making NPCs feel alive: persistent disposition memory, factions,
  citizenship/justice, NPC schedules, motivations/goals, equipment-
  awareness, bounty hunting, and mob/player command parity.
- 39 chunks across 6 phases — Substrate, Tactical fill-in, Routine
  layer, Strategic layer, Cross-cutting features, Audit & polish — with
  a progress tracker at the top. Living doc; status updates as chunks
  ship. Each chunk gets its own spec/plan when picked up.
- Five MEMORY-tracked items absorbed into the roadmap: peacefulquest →
  faction system, Companion Phase 5 (mutations), tactics-cast
  preemption gap, PvM/MvP/PvP/MvM parity gaps, Stillwater town-flavor
  pass.

## 2026-05-01 — Economy Health Dashboard

**Note:** New `/admin/economy/` web dashboard for monitoring NPC
supply chain health. Backend-only release — no in-game commands or
content changes. Player-facing change is one config bump (idle
timeout).

### Dashboard
- **`/admin/economy/`** — five score cards (Economy / Shops / Caravans
  / Foragers / last snapshot), per-discipline rollup of shops grouped
  by `craft_support:` tag (blacksmithing, alchemy, tailoring, cooking,
  jewelcrafting, enchanting, general), per-shop detail with stock bars
  colored by supply bucket, caravan + forager tables with cargo bars
  in pounds. Auto-refresh 30s/60s/2m. Manual "Snapshot Now" button for
  ad-hoc before/after comparisons.
- **All tables sort alphabetically** (discipline name, shop name,
  caravan name, forager name) for predictable row order.
- **Scores:** 0-100 colored red <40 / yellow 40-70 / green >70.
  Per-shop score weights item fills by `RestockQty`; per-discipline
  score is the mean of shops in that bucket; caravan/forager scores
  count Thornwall→Thornwall and Resting→Resting cycles across the
  last 168 hourly snapshots with a stuck-penalty if a state has been
  held >5000 rounds. Overall economy score is weighted 0.6/0.2/0.2
  (shops/caravans/foragers) with renormalization for components with
  insufficient history.
- **Hourly snapshot ticker** writes to
  `_datafiles/economy/snapshots/{unix_ts}.yaml` (gitignored runtime
  state). Auto-snapshots pruned past 30 days; manual snapshots never
  pruned.
- **Delta columns** at 1h / 6h / 1d / 3d / 1w show gold deltas per
  shop and per discipline against the closest historical snapshot
  within ±50% tolerance.
- **Boot prewarm** populates the shop cache eagerly: every persisted
  shop YAML loads on startup, AND every shop-bearing mob's spawn
  placement (from room `spawninfo` blocks) is pre-registered without
  spawning the actual mob. Result: dashboard shows the full set of
  shops + foragers at boot, not just ones in zones a player has
  visited. Inactive forager profiles render as `(not active)` rows
  so the dashboard always shows all 3 (Tova / Halix / Kessa).

### Schema additions
- **`craft_support:` field** on every shop-bearing mob YAML (22
  files). One-of-7 valid values: `blacksmithing`, `alchemy`,
  `tailoring`, `cooking`, `jewelcrafting`, `enchanting`, `general`.
  Source of truth for the dashboard's discipline rollup.
- **Startup validator** (`shops.ValidateShopMobTags`) panics if any
  shop-bearing mob is missing or has an invalid `craft_support:` tag.
  Server refuses to boot until every shop is categorized. On
  `/reload` the validator logs a structured Error with remediation
  hint instead of panicking, so the running server stays up while
  you fix the listed mob YAMLs.
- **Persisted shop YAMLs auto-migrate** from the mob template's
  `craft_support:` value on next boot — no manual edits to the 7
  existing runtime files in `_datafiles/world/dogmud/shops/`.
- **Cargo metrics in pounds** — caravan + forager `cargo_weight` /
  `cargo_capacity` use real carry weight (5000 lbs for the wagon).
  Forager cargo capture also walks ComponentItems + PotionItems for
  foragers that equip a component bag or bandolier (Halix's spear
  case). Wagons unchanged — they don't equip.

### Config knobs (Balance section)
- `EconomySnapshotIntervalHours` (default 1)
- `EconomySnapshotRetentionDays` (default 30)
- `EconomyScoreWeightShop / Caravan / Forager` (defaults 0.6 / 0.2 / 0.2)

### Player QoL
- **`MaxIdleSeconds` 1800 → 18000** (30 min → 5 hours). Players being
  kicked after 30 min of idle was friction for roleplay sessions.
  10x bump. AfkSeconds and ZombieSeconds unchanged — only the hard
  kick.

### Runbook
- See `docs/economy/dashboard-runbook.md` for what each card means,
  how snapshots work, troubleshooting, and the process for adding a
  new vendor discipline.

## 2026-04-30 (evening) — Stage 3.4 Hardening + Pricing Pass (dev only)

**Note:** Smoke-test fixes and late-day polish on the Stage 3.4 economy stack. Promotes to `master` with the rest of the economy stack.

### Forager fixes
- **Halix anchor moved** from Thornwall Temple Interior (468) to Sheltered Ridge Alcove (3040) where Hermit Kael also camps. The original anchor put a Steppe forager in city center with forage range one zone away — round-trip walks blew through state-machine timeouts. The Steppe-side anchor matches Tova's and Kessa's pattern (anchor near forage range; walk into town only to deliver).
- **Forager Vella renamed to Tova** (mob 371, Stillwater Marsh). Disambiguates `look vella` from the long-running Mistress Vella Thorne (mob 355, Stillwater town).
- **Forager state machine no longer re-issues `fold-recall`** while already casting — was resetting cast progress every idle tick.

### Caravan hardening
- **Auto-reset watchdog at the top of every caravan tick.** If the caravan has been stuck in a single state longer than 5× the configured dwell (floor 300 rounds), state resets to `ThornwallDwell` with a `mudlog.Warn` entry. Recovers from orphaned-state corruption after restarts or unusual party deaths without admin intervention.
- **New admin command `caravan reset [<instanceId>]`** — manual reset for one caravan leader (numeric arg) or every caravan leader (no arg).
- **Party-aware hostile check** stops the caravan from abandoning members. Previously the leader's `hostilesInRoom` only checked the leader's room — a follower fighting alone in a different room got left behind. The new `partyHostilesNearby` walks every party member's room.
- **Shop persistence after caravan + forager visits.** Stock changes now persist to disk inside `VisitVendorsInRoom` (was in-memory only — a panic lost an in-flight cycle's deliveries).
- **Wagon equipment slots suppressed.** New `hide_equipment_slots: true` flag on the Mob struct hides the empty Equipment block in `look mob` for entities like the wagon that don't wear gear.
- **Boot panic fix:** wagon name shortened to match its YAML filename per the engine's `ConvertForFilename(name)` convention.

### Pricing + accessibility
- **Pricing pass on 26 mat YAMLs (Approach B)** — rarity-tier-aligned base values. The dynamic shop multiplier already does most rarity work via scarcity (0.25×–5.0× swing); base values now sit at band midpoints rather than encoding rarity twice. Bands: tier-50 = 1–3g (commodities), tier-40 = 5–25g (standard), tier-30 = 25–75g (regional), tier-20 = 80–500g (uncommon). Biggest corrections: Hive Fragment 500→25g (was tier-20-priced but tier-40-tagged), chrysalis_shard 6→80g, gold_wire 8→80g, mutation_catalyst 10→100g, ironbark_shaving 4→25g, raw_gem 5→25g. Stillwater pearl + Chrysalis Core unchanged at 400/500.
- **Starting player gold bumped 25 → 250.** With the new mat prices, a fresh character couldn't afford even one mid-tier craft attempt; 250g lets them try.

### Other small fixes
- **Companion spawn stamina:** previous spawn path set Health and Conviction to max but never set Stamina, so companions spawned at 0 SP and were immediately stamina-broken.
- **Steal gate ordering:** `skullduggery.steal` skill-rank check moved AFTER target validation. Stealing from a `player_attack_immune` mob now surfaces the immune rebuff first instead of the misleading "not advanced enough" rebuff.
- **Mob.Cast diagnostics:** surfaces `InitiateCast`'s silent early-exit reasons (AlreadyCasting / OnCooldown / InvalidSpell / NoTarget) at debug level. Caught by smoke test as a tactics-cast preemption gap (logged as followup).
- **Follow auto-timer removed.** The 10-min auto-expiry in `modules/follow` dropped follow with no in-fiction reason. Teleport drops, death drops, and explicit `follow stop` still apply.

### Developer docs
- `internal/items/context.md` gains three new sections (Rarity Tiers, Pricing Bands, Supply Pipeline) so the items package's developer doc reflects the post-3.4 economy.

## 2026-04-30 — Stage 3.4: Real Item Transfer (dev only)

**Note:** Final stage of the caravan/economy effort. Once this lands
on `development`, the entire economy stack (Stages 3.0b through 3.4)
promotes to `master` as a coherent update.

- The caravan now physically hauls items: a new wagon mob (374) with
  ~5000 carry capacity rides with the caravan party. `look wagon`
  shows the actual cargo. Two draft horses (Hob 375, Bran 376) pull
  it. All three are player_attack_immune.
- Wagon dies if the caravan is wiped at the bandit camp; cargo
  distributes to bandit inventories (round-robin, capped per bandit's
  carry capacity), with leftovers as wreckage corpse loot. Players
  who kill the bandits afterward get the cargo. Wagon corpse renders
  as "splintered wagon wreckage" with custom description.
- Vendor stock caps now derive from item `rarity_tier` × shopkeeper
  `stock_multiplier` (default 1.0). 51 mat YAMLs got rarity_tier set:
  15 tier-50 (common), 17 tier-40 (standard), 14 tier-30 (regional),
  5 tier-20 (uncommon — pearl, gold wire, chrysalis core/shard/
  catalyst). Tier 10 reserved for future ultra-rare content.
  Future big-city shops can set stock_multiplier > 1.0 for
  proportionally larger stock.
- Foragers now physically deliver items from their satchels to vendor
  inventories (no more abstract RestockBuckets). Items that don't fit
  stay in the satchel for next vendor / next cycle.
- New forager rest extension: when carry > 50% on return home,
  forager stays at sanctuary instead of cycling back out. Prevents
  futile loops in saturated economies — foragers wait at sanctuary
  until players consume from vendors and re-open delivery space.
- Caravan vendor stops are now BIDIRECTIONAL — caravan delivers
  items it brought AND picks up items the local vendors produce in
  abundance, hauling them across town. Pickup is gated by `Current
  >= MaxStock/2` so the caravan doesn't extract from a struggling
  vendor. Pays off the "wholesalers seeking arbitrage between
  regions" worldbuilding from the Stage 2 caravan.
- Chrysalis Core (40010) re-sourced: removed from Aberrant Chrysalis
  in Sanctum Basin tutorial. Now drops 10% from stone beetle queen
  (228) and 5% from windscour wyrm (229) in Ironwind Steppe.
- 6 new mob override fields: carry_capacity, health_max, stamina_max,
  corpse_name, corpse_description, stock_multiplier.
- New btree action `distribute_cargo_to_hostiles` for the wagon's
  death handler.
- New config knob `ForagerRestCarryThreshold` (default 0.5) for the
  rest extension.
- ItemSpec gains `rarity_tier` field. Mob struct gains 6 spawn-time
  override fields. Corpse rendering honors mob's CorpseName +
  CorpseDescription overrides.

## 2026-04-30 — Stage 3.1: Forager NPCs (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- Three new forager NPCs feed the supply pipeline that 3.0b wired up:
  - **Vella, the Marsh Forager** (mob 371) anchored to Stillwater
    Temple Interior (4123). Wanders Stillwater Marsh (rooms
    4177-4196), engages prey wildlife (marsh rats, dragonfly
    swarms), salvages corpses, delivers Stillwater + base + overlap
    mats to the 8 Stillwater vendors directly.
  - **Halix, the Steppe Forager** (mob 243) anchored to Thornwall
    Temple Interior (468). Walks the safe northern half of Ironwind
    Steppe, delivers base + overlap mats to the 9 Thornwall vendors.
    Statpool 225 (Ironwind is more dangerous than the marsh).
  - **Kessa, the Fernway Forager** (mob 366) anchored to a new
    Forager's Camp (room 4197, attached west of 4170 Tangled
    Bracken). Walks up to North Road 4038 to meet the caravan; the
    caravan distributes Fernway mats to both towns symmetrically.
- All three are `player_attack_immune: true` (rebuff like
  shopkeepers). They engage prey wildlife on a per-forager
  whitelist, drink a healing salve at HP < 75%, and cast fold-recall
  at HP < 50%. Each carries a thematic 1H weapon (gaff hook,
  hunting spear, hand axe) and a leather bandolier with healing
  salves.
- New behavior tree primitive `forager_step` drives the per-forager
  state machine (resting → traveling → foraging → delivering →
  recalling → loop). Three new conditions support it:
  `mob_can_safely_engage`, `mob_inventory_at_threshold`,
  `mob_hp_below_recall_threshold`.
- New `internal/economy` package mirrors the 3.0b mat-audit-matrix
  as a Go map. New `RestockBuckets([]string)` shop method gates
  vendor refills by supply bucket. Foragers and the caravan both
  use it; only slots whose item-id matches a carried bucket get
  topped up.
- **Caravan changes:**
  - Cycle slowed from ~900 to ~1620 rounds (~2 game days) by
    bumping `CaravanDepotDwellRounds` from 360 to 720. Foragers
    are now the day-to-day reliable supply; the caravan feels
    like a delivery-day event.
  - Two new substates inside each transit leg
    (`outbound_fernway_pickup`, `inbound_fernway_pickup`): the
    caravan dwells briefly at North Road 4038, detects the Fernway
    forager, and acquires the `fernway` bucket flag.
  - `caravan_load` MobState tracks which buckets the caravan
    currently carries. `VisitVendorsInRoom` consumes it, so a
    Stillwater-only caravan run won't restock Fernway slots.
- New room mutator `sanctuary` standardizes the "high-regen room"
  mechanic. Replaces the hardcoded `roomRegenMultiplier` switch in
  the auto-heal hook. `MutatorSpec` gains a `regenmultiplier float64`
  field; multipliers stack multiplicatively.
- Sanctuary mutator wired on:
  - Thornwall Temple Interior (468) — preserves prior 5x regen
  - Sanctum Basin tutorial zone (rooms 101-120) — preserves prior
    5x regen
  - Stillwater Temple of Stillwater (4123) — gains 5x regen for
    the first time, supports Vella's recall destination
  - Forager's Camp (4197) — gains 5x regen, becomes a known safe
    rest stop in Fernway South
- Three new low-tier 1H weapons: marsh gaff hook (10033), steppe
  hunting spear (10034), Fernway handaxe (10035).
- Six new balance config knobs gate forager behaviour:
  `FernwayPickupDwellRounds` (6), `ForagerForageDwellRounds` (8),
  `ForagerCarryThresholdPct` (0.75), `ForagerHPRecallThresholdPct`
  (0.50), `ForagerHealPotionThresholdPct` (0.75),
  `ForagerWaitTimeoutRounds` (150).
- The temple-regen hint generalizes to reference the sanctuary
  class — temples + camps + tutorial all read as one mechanic.
- `ForageCore` (Task 6 originally) consolidated to `internal/forager`
  package so both the player Forage command and the NPC forager
  routine share one yield-table source of truth.

## 2026-04-28 — Stage 3.0a: Stillwater Marsh Zone (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- New 20-room wetland zone west of Stillwater, themed as marsh
  giving way to upland steppe at the southern terminus. Connects
  from Mill Creek Footbridge (4133) via a new west exit; terminates
  at Far Bog Heart (4195, biome: plains) with a one-way view of
  the Dustwalk beyond.
- Five new wildlife mobs (366-370): river otter, marsh rat,
  dragonfly swarm, snapping turtle, bog adder. **Only the bog
  adder is hostile to players** AND it `hates: [rodent]` — it
  hunts the marsh-rats in adjacent rooms (mirror of 3.0c's
  wolf-hates-boar dynamic, but combined with the only-hostile-to-
  player role into one mob).
- The river otter is the **first non-badger consumer of the
  mustelid species** (24) added in Stage 3.0c — validates the
  species investment.
- All 6 existing Stillwater forage mats (lake-iron, marsh willow
  bark, lake mint, freshwater clam, skitter-shrimp shell,
  Stillwater black pearl) get fresh territory to spawn in. No
  new mats added.
- Stage 3.0a is the territory groundwork for Stage 3.1 forager
  NPCs — the marsh is now big enough for a Stillwater-anchored
  forager to wander, gather, and recall to depot when injured.
- Coord map gains 48 Stillwater catch-up rows (the doc was
  missing all of Stillwater) plus 20 new Stillwater Marsh rows.

## 2026-04-28 — Stage 3.0c: Fernway South Zone (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- New 20-room zone south of the existing Fernway, themed as deep
  forest tapering to the steppe edge. Connects from Fox Den (4156)
  via a new south exit; terminates at Steppe Edge (4175, biome:
  plains) with a one-way view of the Dustwalk beyond.
- New mustelid species (24) — fills a real gap in the species set
  (we had rodent and canine but nothing for badgers, weasels,
  otters). First consumer is the forest badger; future zones with
  otters or weasels reuse immediately.
- Six new wildlife mobs (360-365): wild hare, roe deer, honey bees,
  feral boar, timber wolf, forest badger. Only the badger is
  hostile to players — the rest are atmosphere or forage support.
  Wolf is `hostile: false` but `hates: [boar]` — emergent
  intra-zone hunt dynamic where the wolf may engage boars without
  threatening the player.
- The 6 existing Fernway forage mats (oak bark, shadowcap mushroom,
  wild hare meat, beeswax, blood-moss, pine pitch from 3.0b) gain
  fresh territory to spawn in. No new mats added.
- Stage 3.0c is the territory groundwork for Stage 3.1 forager
  NPCs — the forest is now big enough for a Fernway-based forager
  to wander, gather, and recall to depot when injured.

## 2026-04-28 — Stage 3.0d: NPC Fold-Recall (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- `fold-anchor` and `fold-recall` resolvers now accept `actions.Actor`
  rather than `*users.UserRecord`. Mobs can cast both spells via the
  existing tactics dispatcher and the new Go-hook switch in
  `resolveMobSpell`. Player behavior is unchanged.
- New mob YAML field `fold_anchor_room: <roomId>` pre-stamps a mob's
  fold-recall anchor at spawn. The runtime is then identical to a
  player who already cast `fold-anchor`.
- Old Edrin (mob 275) gets `fold-recall` as a panic spell at
  `health_below:30` priority above his existing flee — he recalls to
  the cluttered back room (4037) when injured. Useful smoke-test rig
  for the new pipeline.
- Caravan crew Ketil/Marta/Lars (mobs 357-359) get the same treatment
  with anchor at the Thornwall Market Square depot (465). Wipe
  insurance for the bandit camp ambush — if their HP drops they
  recall instead of dying, keeping the restock service running.
- Stage 3.0d does NOT add forager NPCs or logistic recall triggers
  (e.g., `inventory_full → cast fold-recall`). Those are Stage 3.1's
  job. Caravan recall is individual, not group-aware: each crew
  member recalls on their own panic threshold.

## 2026-04-28 — Stage 3.0e: Corpse Salvage (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- `salvage <corpse>` now works on room-resident corpses, not just
  inventory items. Animal-group mobs yield leather strip + sinew;
  humanoid-group mobs yield cloth strip + leather strip. Each material
  rolls independently against the salvage skill curve. Salvage kit
  required (sold by Fence Dealer Siv, 1g).
- The corpse is consumed on completion (mirrors tagged-item salvage
  behavior — the activity has cost regardless of roll outcome). If the
  activity is interrupted (combat, movement) the corpse stays untouched.
- Added **sinew** (40068), a tough animal-tendon mat sourced from
  corpse salvage on animals. Wired into 2 existing recipes: tailoring's
  Artisan's Satchel (heavy-duty seam binding) and blacksmithing's
  Lake-Iron Hook-Spear (haft lashing).
- 40002 leather strip and 40007 cloth strip reclassified in the audit
  matrix from "Defer to 3.0e" → "Mid-tier overlap (corpse-salvage
  sourced)". Source pipeline now decided. Vendor inventories continue
  to NOT stock these mats — corpse salvage is the v1 source.

## 2026-04-28 — Stage 3.0b: Material Region Split (dev only)

**Note:** This is a dev-only landing. The full economy stack (Stages
3.0b through 3.4) sits unmerged on the `development` branch and ships
to prod (`master`) as a coherent update once Stage 3.4 lands.

- Added 6 new Fernway forest materials: oak bark (40062), shadowcap
  mushroom (40063), wild hare meat (40064), beeswax (40065), blood-moss
  (40066), pine pitch (40067). Each consumed in 1-2 mid/high-tier
  recipes spanning at least 2 craft schools, giving forager-gathered
  Fernway mats real demand once Stage 3.1 ships. (Beeswax tailoring
  recipe wiring deferred to Stage 3.0e corpse salvage.)
- New audit matrix at `docs/economy/mat-audit-matrix.md` classifies
  all 67 raw materials into regional supply buckets (Stillwater,
  Thornwall, Fernway, base, mid-tier overlap, deferred-to-3.0e,
  quest/specialty). This is the durable artifact that subsequent
  stages (foragers, corpse salvage, real item transfer) consume.
- Reshaped vendor inventories across the 17 caravan-served vendors
  into mirrored same-craft pairs. Same-craft Stillwater + Thornwall
  vendors now stock the same mat slot lists, with regional pricing
  asymmetry reflecting the caravan markup (e.g., lake mint 10g at
  Stillwater Apothecary Ilsa, 15g at Thornwall Apothecary Voss).
  Cloth/leather/cord/sinew slots dropped pending Stage 3.0e (corpse
  salvage); the audit matrix flags them for 3.0e to wire properly.
- ~12 mid/high-tier recipes updated to wire demand for the new
  Fernway mats. No new recipes invented; existing recipe corpus
  expanded with one new ingredient slot each.

## 2026-04-27 — Stage 2: Thornwall ↔ Stillwater Caravan System

- Added the **Thornwall ↔ Stillwater caravan**: a three-NPC delivery
  crew (Ketil, Marta, Lars) that runs a continuous loop visiting every
  vendor in both towns and triggering restock on arrival. Cycle takes
  roughly **one in-game day** (~1 hour real time). The caravan rests at
  the Thornwall Market Square depot, departs for Stillwater, visits
  each Stillwater vendor in order, rests at Stillwater's North Square,
  returns to Thornwall, visits each Thornwall vendor, then loops.
- Vendor mobs in caravan-served zones (Stillwater, Thornwall City) **no
  longer auto-restock** on a per-mob timer — they restock only when the
  caravan visits. Vendors in non-served zones (Watchers Crossing,
  Sanctum Basin, etc.) keep the legacy auto-restock unchanged. Both the
  non-crafter merchant tick and the crafter material tick respect the
  served-zone gate.
- The caravan crew can be examined and talked to but **cannot be
  attacked by players** — same rebuff as a shopkeeper. Wired into
  attack/bash/grapple/kick/shoot/taunt/throw/trip and steal commands
  via a new `Mob.PlayerAttackImmune` flag. Caravan crew will fight
  bandits along the road and have been statted to win.
- **Bandit pack at the North Road camp** (lookout, fighter, caster,
  Soren) detuned by ~25–30% across the board so the road is challenging
  but passable for solo and small-group players. The pack also picks
  up `hates: caravan` so they engage the caravan when it passes through
  4052 — every cycle the brawl plays out, the bandits respawn for the
  next pass.
- **New `caravan_step` btree action** drives the cycle (`internal/caravan`
  package owns route data + state machine; `actions_caravan.go` wires
  it into the behavior tree).
- **New config knobs** (`Balance.CaravanServedZones`,
  `Balance.CaravanDepotDwellRounds`) so cadence and zone coverage are
  tunable live.
- **`lookfortrouble` mob command extended** to scan for hostile mobs by
  group hate (in addition to the existing player + species-hate scans).
  Bandits with `hates: [caravan]` aggro on caravan-group mobs in their
  room.

## 2026-04-25 (late evening) — Stillwater Zone + Two Quests + Town Flavor + Engine Polish

### New Zone — Stillwater (Zone 2.2)

- **47 new rooms** opening north of Ashwick via a 10-room interlude
  (the Fernway road). Town spine: gate, lakefront square, Pike &
  Lantern inn, Brindle's smithy, constabulary, north square. Lake
  quarter: docks, Crab-Trap Beach, the Reedy Foreshore, boat pier,
  cave mouth, and a 4-room cave dungeon ending at the Hollow Sump.
  West quarter: cooper's lane, healer's cottage, Ulla's parlor +
  her late husband's untouched workshop, cemetery, sluice pond,
  watermill, tailor's cottage, travelers' camp, the old chapel
  ruin, the wardstone circle, and a boat-builder's yard.
- **22 named NPCs.** Innkeeper Sigrid, weaver Edda, smith Brindle,
  apothecary Ilsa, pearl-carver Kess, storekeeper Wulf, fishmonger
  Tov Brann, dock master Arn, constable Drunn, temple priest Seren,
  miller Bram, old fisherman Hodder, old cottager Gyda, Ulla, the
  child Pip, the caravan crew (Ketil + Marta + Lars), and assorted
  others. Each carries dialogue with cross-references to other
  townsfolk; gossipers (Hodder, Gyda, the barmaid Neva) broadcast
  world-event news on the engine's gossip system.
- **All seven crafting stations on-site:** forge (Brindle's smithy),
  alchemy bench (healer's cottage), loom (tailor's), cooking fire
  (Pike & Lantern + bakehouse), jeweler bench (pearl-carver's
  garret), enchanting circle (wardstone), watermill grain.
- **Sethome anchor:** `set home stillwater` respawns at the Temple
  of Stillwater (room 4123).

### Two New Quests

- **Quest 19 — The Lake-Caves Bounty** (combat / bounty). Constable
  Drunn posts an escalating bounty on the cave creatures spilling
  into the shallows. Five steps with multiple completion paths:
  partial reward (150g) for clearing the shrimp + drowned hunters,
  full reward (500g) for bringing back a leviathan tooth from the
  sump dweller boss. Both Drunn and dock master Arn will accept
  the tooth. Dialogue acknowledges your choice.
- **Quest 20 — Ulla's Silence** (lore / investigation). Ulla
  finally asks someone to look through her late husband Elgar's
  things in the workshop above her parlor. The trail leads through
  six spiral-marked sites across town and the western ruins — a
  pre-Chrysalis breadcrumb that Elgar was researching when he went
  into the deep. Smith Brindle was supposed to descend with him
  and didn't, and has been finishing the spear Elgar ordered ever
  since. The kingfisher Vella buried at the cemetery is for
  whoever knows enough to look. Eleven steps, single zone, with a
  flag-tracked ending choice (whole truth vs partial). Players
  who completed Quest 19 with the full path receive an extra
  acknowledgment from Ulla.

### Crafting & Items

- **Forage extension.** New `water` biome added to the foraging
  system; swamp + water yields extended with five new materials:
  cattail-down (40055), marsh-willow bark (40056), lake mint
  (40057), freshwater clams (40058), lake-iron nodules (40059).
- **Four new craftable recipes:** Lake-Iron Hook-Spear
  (blacksmithing), Lake-Tonic of Steady Hand (alchemy → buff 82
  Steady Hand), Cattail-Down Cloak (tailoring), Stillwater Lake
  Chowder (cooking). Plus three quest-related craftables:
  Hunter-Eel Scale Vest (skullduggery affix), Stillwater Pearl
  Pendant, and the Drowned-Veil enchantment (back-slot, conviction
  reservoir).
- **New cave drops:** skitter-shrimp shell, drowned-hunter hide,
  Stillwater black pearl (boss-drop, 15% chance), leviathan tooth
  trophy (boss-drop, 100% chance — the proof for the bounty).

### Engine & Balance

- **Vitality progression rebalanced.** `StatProgressionMultipliers.
  vitality` bumped from 1.0 to 4.0. Vitality has no high-frequency
  caller (no skill primaries it), so its actual call count was
  ~4-5x lower than other 1.0-multiplier stats — players who
  weren't tank-styled (taking constant damage to trigger the
  regen-progression curve) saw vitality lag behind every other
  stat. The 4x multiplier brings effective progression rate in line.
- **Per-item drop chance.** New `Item.DropChance` field plus
  `ShouldDrop()` helper. Mob suicide refactored so equipped and
  carried items use the same drop-gating helper. Lets boss
  signature drops (Stillwater black pearl) ride a 15% roll
  instead of always dropping.
- **Skill-affix progression patch.** `Character.GetSkillLevel()`
  now includes StatMod contribution — fixes the orphaned skill-
  affix path so item statmods like `skullduggery: +7` actually
  count toward the player's effective skill level. Volcanic plate
  and similar instance-loot skill bonuses now work as intended.
- **Mapper cache reload.** New `mapcache` argument to admin
  `reload` command flushes the per-zone mapper cache without a
  server restart — useful after editing room mapsymbol/maplegend
  fields.
- **Quest engine SOP hardening** (sketch-quest skill). New
  required gates: player-POV walkthrough per step, trigger-
  mechanic ranking table (★★★★ ask-quest down to ☆ unguessable
  magic words), thousand-mudder test, narrator-overreach guard,
  and the `consume_item` requirement on every quest-engine
  item_give trigger to suppress give.go's behavior-tree
  fallthrough (which was firing the noncombat archetype's
  "declines politely" emote AFTER quest acceptance).

### Story / World

- **"What the Moons Keep" — V3 polish pass.** The novel went
  through a multi-round adversarial review pass: Section 1-3
  critical/HIGH severity fixes, Round 2 voice consistency fixes,
  Round 3 verbal-tic sweep, V3 aggregate review, V3 polish
  pre-DE pass. ~270 lines added, ~300 lines removed across the
  full ~735KB manuscript.
- **Stillwater carries a quiet unsolved mystery.** The pre-
  Chrysalis spiral motif appears at five sites the player can
  find, plus a sixth in Elgar's workshop. The Voss family quest
  reveals what Elgar was researching, but not the deeper question
  of WHAT the symbol meant or WHO carved them. Sealed for future
  content.

## 2026-04-25 — Player Rename + Account Delete + Name Collision Prevention

### Gameplay

- **Players can now rename their character.** Use `rename <newname>` to
  request a new name. Rename is cooldown-gated (default 7 days, configurable
  via `Balance.CharacterRenameCooldownHours`). You'll see a yes/no confirmation
  (default no) before the change takes effect.
- **Players can permanently delete their character and free the username.**
  Use `deletecharacter` for a two-stage confirmation: first yes/no (default no),
  then type your exact character name to confirm. The deletion is immediate
  and irreversible — your user file is removed and the name becomes available
  for a new character.
- **Companion naming now correctly prevents mob-template collisions.**
  When naming a companion or pet, the system now checks that the name doesn't
  match any mob template in the world — previously this check was missing,
  allowing companions to shadow core NPCs or monsters.

### Engine & Balance

- **Centralized name validation.** All player, companion, character, and pet
  name checks now flow through `users.ValidateActorName()`, ensuring consistent
  rules across the game.
- **Boot-time name collision audit.** On server startup, if any mob template
  name matches an existing player character, the server logs a warning. This
  helps catch and prevent collisions in production without blocking startup.
- **Admin command rename → renameitem.** The admin `rename` command for items
  moved to `renameitem` to free up the `rename` verb for player use.

## 2026-04-24 (late night) — World Mob Audit Complete + Engine Polish

### Gameplay

- **Every zone now uses the behavior archetype system.** North Road,
  Thornwall City, Marches Spur Road, Ashwick, Watchers Crossing,
  Thornwall Outskirts, Dustwalk Road, and the Labyrinth of Low Tunnels
  joined Sanctum Basin and Ironwind Steppe in the migration. Mobs in
  these zones now react consistently — fighters pile on, lookouts
  call for help, leaders rally their packs, shopkeepers and questgivers
  decline politely instead of fighting back.
- **Pack reactions across the world.** New routines connect mobs that
  belong together: bandit packs on Marches Spur Road, the smuggler
  ring beneath Thornwall City, and the chrysalis-touched mobs in the
  lower district. Hit one and the rest hear it.
- **Thornwall City has its own ambusher.** The chrysalis skulker
  joins the cave's pale lurker and blind stalker as hit-and-fade
  predators — strike from hidden, flee on hurt, slip into shadow,
  strike again.
- **Wilderness now has prey.** Hares, grouse, sparrows, squirrels,
  toads, mice, chickens, and similar small wildlife flee instead of
  fighting. They're still attackable for food and reagents — they
  just don't stand and die anymore.
- **Sylara of the steppe now speaks naturally.** Her dialogue was
  written with third-person stage directions ("Sylara inclines her
  head...") that the engine spoke aloud. Rewritten to first-person
  speech with stage directions moved into the bracketed narrator
  hints where they belong.

### Fixes

- **Hidden-tag no longer lingers mid-combat.** Three fixes layered:
  (1) `CancelCombatBuffs` now strips permabuff entries with the
  cancel-on-combat flag — previously the active buff was expired but
  Validate re-applied it from the permabuff list. (2) The `camo-skin`
  mutation switched from granting a permanent `hidden` flag to a
  proper `stealth_bonus` (matches `chameleon-skin`'s pattern) —
  mutations no longer leak the hidden tag into combat text.
  (3) The buff system suppresses start text when refreshing an
  already-active buff, so ambusher idle ticks don't spam "{mob}
  disappears into the shadows" every round.
- **Surprise strikes now show their dedicated text.** The btree's
  `actAttack` was always setting `DefaultAttack` aggro — even when
  the attacking mob was hidden. Mobs now properly promote to
  `SurpriseAttack` when striking from hidden, triggering the
  `*[SURPRISE ATTACK]*` prefix and the backstab crit bonus that
  ambushers were supposed to get all along.
- **Ambushers attack proactively when a player is in their room.**
  Previous design only fired on `player_enter`; if a player came
  back to a room where an ambusher had re-hid, the ambusher just
  sat there. Now they fire surprise strikes whenever they find
  themselves idle, hidden, and with a player present.

### Behind the scenes

- Dead startland and tutorial zones removed (players couldn't reach
  them; default `StartRoom: 113` lives in Sanctum Basin). Mob IDs
  1–5, 57, 58 freed for reuse.
- Effective archetype coverage: 100% of stock-combat mobs across
  every zone. The handful of skips (Old Edrin, Olen, Phantom, Sable,
  Pell, Dal, loot goblin) all have custom per-mob behavior trees
  that override archetype anyway.

## 2026-04-24 (late evening) — Ironwind Steppe Audit + Boss Behaviors

### Gameplay

- **Cave stalkers now ambush from the dark.** Pale lurkers and blind
  stalkers spawn hidden, open with a surprise strike when a player
  enters their room, then flee the moment they take damage and
  re-hide in an adjacent room. Maximum-nuisance hit-and-fade cycle.
- **Stone Beetle Queen calls her swarm.** Boss behavior: when wounded
  or when one of her brood is hurt, she calls for help — pulling
  cave beetles from adjacent rooms. Vitality bumped to match her
  tank role.
- **Windscour Wyrm goes two-phase.** Above 50% HP the wyrm fights
  its slow, devastating baseline rotation. Below 50% HP it rages —
  tail-sweep knockdown rotations on every round. Vitality bumped to
  support the pacing.
- **Prey animals flee when hit.** Hares, grouse, lizards, squirrels,
  toads, moths, tumble beetles, and dry creek crayfish now retreat
  to an adjacent room when attacked instead of standing and dying.
  They remain attackable for hunting.

### Behind the scenes

- Two new behavior archetypes: `ambusher` and `prey`.
- Custom per-mob btrees for the Stone Beetle Queen (228) and
  Windscour Wyrm (229).
- Ironwind Steppe now has 43/43 archetype coverage.
- No engine changes — all behaviors reuse existing primitives.

## 2026-04-24 (evening) — Sanctum Basin Mob Audit + Tutorial Content

### Gameplay

- **Sanctum Basin NPCs now offer tutorial guidance for newer gameplay
  systems.** Each of the nine non-combat NPCs covers a curated set of
  topics through their dialogue: ask Korvath about salvage or
  enchanting, ask Yenna about potion aging or the bandolier, ask Saris
  about spell discovery or manifestation, ask the Combat Trainer about
  rally/warcry or companions, ask Fen about tracking or packs, ask the
  Warden about respawn grace or aggro, ask the Scholar about mutations,
  ask the Chrysalis Priest about the Awakening, ask Merchant Adela about
  bartering or encumbrance.
- **Non-combatants now react when you try to attack them.** Trying to
  attack (or target with a harmful spell) an NPC who cannot be attacked
  now triggers an in-character emote from that NPC — a raised eyebrow
  from a questgiver, a step back from a shopkeeper. Rate-limited to one
  reaction per NPC per round so companion and party auto-assist cannot
  spam it.

### Behind the scenes

- Four new behavior archetypes: `noncombat_questgiver`,
  `noncombat_shopkeeper`, `noncombat_passive`, `combat_passive`. Every
  Sanctum Basin mob is now tagged with a `behavior_archetype` value.
  This is the first zone in a larger migration to the archetype system.
- New btree event `player_attack_rejected` fired from attack.go and
  from HarmSingle spell rejection in cast.go.
- All tutorial content is delivered via dialogue YAML `patterns`, which
  is deterministic and prod-safe (no LLM dependency).

## 2026-04-24 — Discovery Rate Stat Offset

### Gameplay

- **Spell and recipe discovery now scales with Perception + skill.**
  The decay that slows discovery as you learn more spells/recipes
  is now partially offset by your Perception stat and the relevant
  skill (Spellcasting for traditional spells, Manifestation for
  manifestation-school spells, or the specific crafting skill for
  each recipe). A newbie discovers at the current rate; a seasoned
  character with invested Per + skill discovers roughly 1.8× faster
  at 20 known — closing the late-game discovery drought without
  flooding new characters with learn-messages.
- **Offset mechanic:** Per contribution reaches 1.0 at Per=300,
  skill contribution reaches 1.0 at rank 100, combined via
  `1 - (1 - per)(1 - skill)` and capped at 0.8 (effective decay
  floor = 20% of base). Either Per or skill alone gives a partial
  benefit; the combination unlocks the full cap.
- **Mobs benefit too.** Caster mobs with high Per + Spellcasting
  will expand their spell repertoire faster than before — a
  battle-hardened mob learning from repeated casts.

### Config

- New `Balance` knobs: `DiscoveryPerceptionScale` (default 200),
  `DiscoverySkillScale` (default 100), `DiscoveryMaxDecayOffset`
  (default 0.8). Set `DiscoveryMaxDecayOffset: 0` to disable the
  offset mechanic entirely and revert to the prior flat-chance
  formula.

## 2026-04-22 (evening) — Pack Tactics Revamp + QOL Batch

### Gameplay

- **Priests and unrelated civilians no longer aggro you after fighting
  bandits.** The old group-hostility system flagged every mob sharing
  any group tag (including taxonomic ones like `humanoid`) as hostile
  when you hurt a bandit. Bandits and temple priests both have
  `humanoid`, so attacking bandits made Olen the priest swing at you
  on sight. Replaced with a routine-scoped pack-reaction system:
  packmates now have to share a specific `routine` string
  (`bandit_camp_guard`, `wolf_pack_ironwind`, etc.) to react to each
  other being attacked. Priests, merchants, guards, and wildlife
  unrelated to your fight stay peaceful.
- **Packs coordinate via behavior trees now.** A bandit fighter,
  caster, lookout, and leader in the same camp respond to one of
  them being attacked the way their role suggests. Fighters pile in.
  Casters shield the tank, then heal the most-wounded packmate, then
  engage. Leaders open with rally + warcry self-buffs, then engage.
  Lookouts yell for help, then engage.
- **Pack cries carry to adjacent rooms.** Mobs in neighboring rooms
  whose routine matches the caller's now move toward the commotion.
  Previously each room's mobs stayed oblivious to a fight next door.
- **Charmed wild creatures no longer snitch on their pack.** A
  charmed mob counts as your companion, not a packmate, so the
  pack doesn't react when you fight its former brothers.

### Fixes

- **`lookfortrouble` now respects the respawn grace buff.** Mobs
  scanning for targets when you arrive in a new room will skip a
  grace-protected player entirely instead of picking you as the
  "best" target and attacking through the grace check in the combat
  pipeline. Closes the Duard prod repro where mobs started on a
  respawning player inside the 3-round grace window.
- **Thornwall — Rift Chamber no longer overlaps Records Office.**
  The Rift Chamber was geographically beneath the Temple District
  but was authored with east-exit adjacency to the Records Office,
  putting both rooms at the same coordinate on the mapper. Rift
  Chamber is now reached by `down` from the Temple District.
- **North Road — Bandit Camp no longer overlaps the Inn's common
  room.** The Camp Approach intermediate room had been bent south
  to dodge one overlap and ended up dropped on top of another.
  Removed the Approach entirely; the bandit camp now hangs off the
  main road at a single exit.

### Quality of Life

- **`craft` listing shows the enchant target slot.** Enchanting
  recipes now print `(targets: weapon)`, `(targets: body armor)`,
  etc. so you know which slot an enchantment will land on before
  you spend materials.
- **Companion roster shows mutations.** `companions` now lists each
  mutation your companion has acquired underneath its name, stats,
  and status line. `companions <name>` shows the full set in the
  detail view.
- **Progression dashboard shows exact rank and grandmaster tier.**
  The player overview on the skill progression web page now shows
  the raw rank next to the tier description, and skills above 75
  display as "Grandmaster" instead of falling off the top of the
  previous tier chart.
- **`identify` is in the starter spellbook.** New characters can
  identify unidentified items without waiting for a scroll drop or
  shop purchase.

## 2026-04-22 — Combat Bug Fixes + Novel Canon Correction

### Fixes

- **Two-handed weapons no longer grant a spurious fist attack.** When
  you equip a 2H weapon, the pair-partner slot (offhand for the main
  hand, extra arm 2 for extra arm 1, extra arm 4 for extra arm 3) is
  cleared to an empty-item marker. The attack-collection code read
  that as "empty arm → generate fist" and produced a bonus unarmed
  swing from the arm physically occupied by the 2H weapon. Most
  visible on a 2H + extra-arm-shield loadout: the bogus fist then
  swung 3–4× per round via the normal swing-count formula, producing
  noticeable extra unarmed hits every round. That arm is now
  correctly treated as occupied.
- **Flee now actually flees.** The flee command sets aggro to a
  Flee-type state with no target so the combat loop can run the
  escape attempt on the next round. The round-start aggro validator
  was invalidating that no-target aggro (only SpellCast was in the
  allowlist), and the fallback then grabbed a new target before the
  flee attempt could run — silently losing you a round of attacks
  and keeping you in combat. Flee is now on the allowlist alongside
  SpellCast.

### Novel

- **Bloom-harvest canon consistency fix in "What the Moons Keep"
  (Chapters 18 and 22).** The scenes showing the captive woman on
  the pallet described her as "heavily mutated" with elongated
  limbs and frond-fingers — contradicting the book's rule that
  *hollow* means the absence of the Chrysalis change, and that
  Bloom is produced from hollow people by definition. Rewritten to
  show her unchanged but in visible reaction to something her
  captors introduced: puncture marks at both elbows and the hollow
  of the throat, skin flushed and faintly raised at each site,
  sweat despite the cold stone. Vane's spoken summary to Maren and
  her Ch 22 return-visit observations updated to match. Two parallel
  audit agents (Ch 1-17 and Ch 19-30) found no other instances of
  the same contradiction.

## 2026-04-21 — Tank Companions, Death Cleanup, and Two Instance-Save Fixes

### Gameplay

- **Tank companions (flesh golem, earth elemental, magma elemental)
  now actually hold aggro.** Previously their taunts missed ~70% of
  the time because their stat distribution and base charisma were
  tuned for general brawlers, not front-liners. Two changes stack:
  - A new "tank" stat archetype allocates 25% Charisma and 20%
    Vitality out of the stat pool (up from ~7% Charisma under the
    old "fighting" archetype).
  - Species base Charisma raised on all three: flesh golem 5→115
    (top-tier raised pet), earth elemental 5→70, magma elemental 5→80.
    These are imposing, otherworldly creatures; low-single-digit
    Charisma was a default that never got tuned.
- **Tank + generic fighter companion AI archetypes.** Flesh golem,
  earth elemental, and magma elemental run a tank routine:
  interrupt-on-cast, taunt-if-not-holding-aggro, bonus-damage kick
  when the target's prone or clinched, rally + warcry self-buffs,
  then a bash/grapple/trip knockdown cascade. Steppe spirit wolf and
  zombie run a simpler generic CC rotation. Tank archetype uses the
  new generic "bellows a thunderous challenge" taunt text instead
  of the wolf-themed howl.
- **Death no longer kills you twice.** Three long-standing respawn
  bugs fixed together:
  - **Poison, bleeding, and other conditions now clear on death.**
    Previously a poisoned player respawned at 5% HP still poisoned —
    the next DoT tick killed them again. Now the Conditions slice
    wipes alongside buffs.
  - **Inbound aggro clears on death.** Every mob in your combat room
    that was targeting you ends aggro when you die, and your
    companions' aggro clears too. Respawn arrives in a clean slate.
  - **3-round grace period post-respawn.** New `Respawn Grace` buff
    (id 81, NoAggroTarget flag) prevents mobs from acquiring aggro
    on you for 3 rounds after respawn. Configurable via
    `Death.RespawnGraceRounds` (default 3; 0 disables). PvP
    griefers: you can no longer chain-kill a respawning player.
- **Single combat hit no longer applies stacked penalties.** A
  pre-existing bug: the combat loop could queue the death-processing
  `suicide` command multiple times in the same round, applying two
  (or more) separate stat-decay + skill-rust rolls from one death
  event. A round-based dedupe flag (`Character.LastSuicideRound`) now
  guards against this.
- **Crafting blocks all 7 combat commands consistently.** Previously
  only rally and warcry checked `IsCrafting`; bash / trip / grapple /
  kick / taunt would let you swing your way out of a craft. All 7
  now universally reject with a friendly "focused on your work" message.
- **Tank companion rally/warcry don't burn cooldowns re-casting.**
  The behavior tree now skips rally/warcry when the buff is already
  active on the companion; they move on to other moves in the
  priority list instead.

### Fixes

- **Sable portal and other room-exit routes no longer break
  randomly.** An asymmetric bug in the room instance-save system:
  the save side correctly skipped structural fields (exits,
  description, nouns, etc.) via `instance:"skip"` struct tags, but
  the load side used raw `yaml.Unmarshal` which doesn't see the tag.
  Pre-fix corrupt files kept overwriting the template on every
  load. `LoadRoomInstance` now restores skip-tagged fields from a
  fresh template copy after the overlay unmarshal.
- **Summoned companions always start fresh.** The old
  `mobs.instances/summons/` file-based persistence was keyed by
  room, not by owner, and leaked progression across
  summon-dismiss-resummon cycles and across players. Removed the
  file layer entirely; companions now persist only via
  `CompanionInfo` on the owner's user YAML.

### Under the hood

- **`actions.CommandIsReady` is the single source of truth for
  combat-command gating.** New btree action `command_best_of` used
  by the archetypes queries CommandIsReady before issuing; a drift-
  detection test enforces that CommandIsReady and each Execute*
  agree on readiness. Signature flipped from `*mobs.Mob` → `Actor`.
- **New "tank" stat archetype** (internal/mobs/mobs.go) — 25% Cha,
  20% Vit, 15% each Str/Dex/Wil, 10% Per. Joins the existing
  "fighting" and "casting" archetypes.
- **`NoAggroTarget` buff flag + `characters.SetUserUntargetableCheck`
  callback.** Avoids the users↔characters import cycle; follows the
  same pattern as `rooms.SetCompanionTransport` and
  `rooms.SetBTreeStateEvictor`.
- **Tank-taunt aggro-pull now works for mob taunters.** Was
  previously gated on `attackerUserId > 0` (player-only); now also
  handles the mob-taunter path via `actor.GetMobInstanceId()`, so
  companion tanks' taunts actually switch the target's aggro.

## 2026-04-20 — Companion AI Overhaul (Two Archetypes + Follow + Regen)

### Gameplay

- **Summoned companions now fight intelligently.** Two new AI
  archetypes cover the five mage-crafted summons:
  - **Melee self-buffers** (vampire, fire elemental): maintain
    offensive + defensive self-buffs, then attack with bite /
    flavor moves between casts. A fresh vampire's first combat
    sequence casts conviction-surge, then iron-will, then
    conviction-ward across the first ~10 rounds.
  - **Pure casters** (wraith, spectre, air elemental): emergency-
    heal when HP drops below 40%, maintain defense, cast AoE
    damage when enemies are grouped, else single-target harm.
    Watch for heal, sparks, conviction-barrage, mind-spike,
    conviction-spike, and nerve-disruption depending on the mob
    and situation.
- **Air elemental reclassified as a caster.** Its stats (dex 20,
  perception 20, willpower 10) were always caster-shaped; this
  update gives it the spellbook and archetype to match.
- **Companions now follow their summoner through every movement
  path** — walking, recall, portal, fold-recall, sable, admin
  teleport. Mid-cast wind-ups abort cleanly to keep up (conviction
  spent during a cancelled cast is forfeit, same as a player
  self-interrupt). Aggro on a target no longer in the new room
  ends automatically.
- **Thornwall Temple (and other sanctuary rooms) now heal your
  companions too.** The 5x regen boost in the temple previously
  only applied to players; your pet sitting next to you at Olen's
  altar got normal regen. Now it matches yours. Same fix extends
  to the Sanctum Basin tutorial rooms and the testing arena.

### Under the hood

- **Behavior-tree archetypes are now a first-class concept.** Mob
  YAML gains a `behavior_archetype: <name>` field that resolves to a
  shared tree file at `behaviors/archetypes/<name>.yaml`. Resolution
  order: per-mob btree file wins, then archetype, then legacy. This
  unlocks future work where NPCs can switch archetypes at runtime
  (e.g., a caravan guard taking up banditry).
- **Spells carry `categories:` tags.** Free-form strings used by
  archetype AI to filter a mob's spellbook by purpose:
  `self_defense`, `self_offense`, `self_heal`, `harm_single`,
  `harm_multi`. Applied to 12 existing spells.
- **New btree action `cast_best_in_category`** picks the
  highest-scoring (`base_folds × cost`) spell in a category from the
  mob's spellbook, skipping already-active buffs, components,
  summons, and insufficient CP. Self-gates on the shared
  special-move cooldown, so mobs naturally alternate between cast
  rounds and attack rounds.
- **New event `mob_combat_round`** fires per mob combatant BEFORE
  legacy AI, so archetypes are authoritative for mobs that declare
  one. Legacy `preferredSpell` (shield-first priority) no longer
  preempts the archetype.
- **`multiple_enemies` btree condition is now perspective-aware.**
  A summoned caster no longer treats its summoner and fellow
  companions as "enemies" when deciding AoE vs single-target. Wild
  mobs (like bandit_leader) preserve original behavior.

### Latent engine fixes surfaced by this work

- **`applyMobSelfEffect` now handles `buff` effect_type.** Used to
  only handle heal and shield — buff-type spells (conviction-surge,
  iron-will) fell through silently when mobs self-cast them. Now
  they correctly apply, including per-buff tick snapshots.
- **Shield-active detection uses `ConditionShield`** instead of the
  equipment-layer `HasShield()` helper. The magical shield a spell
  applies is tracked as a condition, not as worn equipment.

## 2026-04-18 — Combat unification, target resolution, bleedout removal, lots of fixes

### Gameplay

- **Bleedout removed.** Health <= 0 = dead, for both players and mobs.
  No more "downed" two-tier rule, no PlayerDrop event, no
  CoupDeGraceRounds. One-shot kills and DoT kills are now possible.
  ~270 lines of bleedout-specific code removed.
- **Death respawn at 5% of max pools (was 100% since the shadow-realm
  removal).** Restores the "death run brake" that was unintentionally
  dropped during the JS Audit Phase 4c. Respawning weakened means you
  have to recover before your next attempt at whatever killed you.
  Configurable via new `Death.RespawnPoolFraction` knob; per-pool
  minimum of 1 so respawn doesn't immediately re-trigger the death
  check. Operators who want full restore can set 1.0.
- **PvP combat gains parity with PvE.** As part of the
  combat-quadrant unification (see below), PvP now correctly applies:
  - Adrenaline buff (low-HP stamina boost)
  - Return damage (thorns / spikes / spell deflection feedback)
  - Lifesteal (vampiric weapon enchant)
  - Moon-mod stat shifts on mutated combatants
  These were all missing from the legacy PvP-only handler.

### Combat & AI Fixes

- **MvM (mob vs mob) parity gaps closed:**
  - Defender mobs now receive `OnCritReceived` on crit hits (PvM/MvP/PvP
    already did).
  - Attacker mobs now fire `OnCriticalSuccess` / `OnCriticalFailure`
    callbacks on crit rolls (was only firing OnSkillUse).
  - Attacker mobs now emit room-visible stat-gain messages when
    `OnStatUse` returns true (was discarding the bool).
- **PvM defender `combat_start` AI signal preserved.** Previously
  emitted in a function that the unification was about to delete; now
  in the unified resolver, gated to PvM only. Reactive AI for first-
  round mob deaths no longer silently breaks.
- **Legacy MvP ConditionShield double-dip removed.** Player defenders
  with Minor Shield were getting magnitude/2 reduction applied on top
  of the magnitude already counted by the mitigation layer. Stage 11.4
  leftover from before mitigation was unified. Single application now.
- **Crit feedback on PvM/MvP no longer drops attacker text.** The
  return-damage room broadcast in PvM correctly excludes the
  attacking player from the third-person message.
- **Edrin engages after his revelation.** Was firing `combat_start`
  once and then sitting idle while you fought his elementals. Now has
  a `single_target` fallback tactic.
- **Behavior tree `hostile` param is now a real bool** (was string
  `"true"`). Backward compatible via `getBoolParam` helper that
  accepts both forms.
- **Knuckles only progress unarmed-combat.** Dual-wielding knuckles
  was incorrectly triggering weapon-combat progression alongside
  unarmed-combat. Extracted `isDualWieldingWeaponCombat` helper that
  checks at least one weapon routes to weapon-combat before granting
  its progression.
- **Dismiss is peaceful for crafted companions.** Mage-crafted
  companions (Summoned, Conjured, Raised) dissolve immediately
  instead of going hostile and lingering for 5 minutes. Charmed wild
  creatures keep the betrayal-turns-hostile behavior (thematically
  correct).

### Spell Fixes

- **Summon spells check their component before casting.** Previously
  summon-steppe-spirit / raise-* / conjure-* validated only their
  ComponentTag, missing SummonComponentId. The full cast animation
  ran and consumed conviction before failing at resolution with
  "You lack the required component." Now caught at cast init.
- **Fumbled spells no longer succeed.** Summon, charm, fold-anchor,
  fold-recall, and purge-affliction used to run their primary effect
  even on a fumbled cast (you took backfire damage AND got the
  summon). Now the fumble cleanly aborts the effect; component
  consumption stays unconditional (failed binding ate the catalyst).
  Covers 13 summon spells + charm + 3 Go hooks.
- **`ConditionRegen` heal-tick text** now emits per-tick "wounds knit
  closed" feedback while the regen is active.

### Commands

- **`rally` and `warcry` no longer slip through during crafting.**
  Both now check `IsCrafting()` and refuse with a thematic message,
  matching the `craft.go` re-entry pattern. (Broader audit of other
  active commands tracked as future work.)

### Refactor: Combat-quadrant unification

- **Four parallel `handle{P,M}vs{P,M}` combat handlers collapsed into
  a single `handleCombatRound(atk, def actions.Actor, ...)`** in
  `internal/hooks/NewRound_DoCombat_unified.go`. Eight named phase
  helpers (target resolve, wait round, attack roll, damage bonuses,
  crit + messaging, progression, behavior trigger, aggro + assist,
  round resolution). Routing strategy: `IsPlayer()` checks at leaf
  sites where divergence is required; no Quadrant enum.
- Future parity gaps are now structurally impossible — any new
  combat logic added to the unified handler applies to all four
  quadrants by default. Quadrant divergence requires explicit
  `IsPlayer()` gating + reason comment.
- New cross-package test harness:
  `behaviortree.SetMobTreeForTest`,
  `items.SeedAttackMessagesForTest`. Structural routing test drives
  all four quadrant pairs (UU / UM / MU / MM) through
  `handleCombatRound` end-to-end.

### Refactor: Target resolution

- **`actions.ResolveTargetActor(room, name, opts...) (Actor, error)`**
  consolidates the `room.FindByName + GetInstance + nil-check` chain
  that was reimplemented ~37 times across user/mob commands with
  subtle variations. Sentinel errors (`ErrTargetNotFound`,
  `ErrTargetVanished`) let callers give precise error messages.
- New `actions.NewUserActor` / `NewMobActor` /
  `NewUserActorInRoom` / `NewMobActorInRoom` constructors.
- Closes the latent-nil-crash class (e.g., `attack.go:27` was
  derefing a nil mob via `m.Character.Aggro` unguarded).
- Two minor UX wins fall out: `ask <player>` now errors cleanly
  ("You can't ask another player.") instead of silently
  fall-through; `party invite <mob>` errors cleanly ("You can only
  invite players to your party.") instead of "Something went wrong."

### Refactor: Rooms package

- **`AddTemporaryExit` now correctly enforces no-overwrite contract**
  while explicitly allowing the legitimate ephemeral-portal →
  ephemeral-portal overwrite case. Closes a long-standing failing
  test from the Stage 1.5 audit.
- **Instance cleanup chain consolidated in `CheckPortalTimers`** —
  TTL-triggered chain (boot players → Remove ephemeral rooms →
  EvictRoomBTreeState via callback → TryEphemeralCleanup) replaces
  the deleted CleanupEmptyInstances. Resolves the catch-22 where
  ephemeral rooms couldn't garbage-collect while their instance
  stayed registered.
- **`behaviortree.EvictRoomBTreeState` wired up via callback in
  `main.go`**, avoiding the rooms→behaviortree import cycle.

## 2026-04-17 — Code Cleanup Stage 1 complete (1.2a, 1.5, 1.6, 1.8)

**Stage 1 of the code cleanup roadmap is now complete (substages
1.1–1.8).** Four substages this day; see prior notes for 1.1 / 1.2b
/ 1.2c / 1.3 / 1.4 / 1.7.

### Stage 1.2a — Combat + Spell god-function refactor

- `handlePlayerVsMob` 286 → 39 lines.
- `handleMobVsPlayer` 236 → 82 lines.
- `applyMobEffect` 246 → 26 lines.
- New `internal/hooks/NewRound_DoCombat_resolution.go` holds combat
  phase helpers; spell case helpers inlined.
- Includes parity fix: PvM return-damage room broadcast now excludes
  the attacker (MvP already did).
- Removed dead "tame" EffectType (superseded by charm).

### Stage 1.5 — Error Handling Audit

- Audited code paths added after Phase 37.3a/b sweep.
- 3 Critical fixed: `spell_purgeaffliction` nil guard, two unsafe
  type assertions in `NewRound_BroadcastHints` and
  `RedrawPrompt_SendRedraw`.
- `mudlog.SetupLogger` panics → `log.Fatalf` (only intentional
  behavior change — the panic was uncatchable).
- Sable portal refund paths + admin dashboard nil-checks +
  behaviortree codebase all verified clean.

### Stage 1.6 — Test Coverage for New Systems

- 24 additive Go unit tests across 4 files: 6 room btree engine,
  7 Phase 4c conditions, 9 Phase 4c actions, 1 actSummonCompanion
  hostile, 1 give.go quest-engine vs btree handoff regression.
- Zero production code change.

### Stage 1.8 — Behavior Tree Engine Robustness

- **Panic-safe `DrainQueue`** via `safeExecuteDelayed` wrapper.
  Panics in delayed-action closures (typically caused by closures
  over destroyed mobs/rooms/users) are now recovered and logged
  at `mudlog.Error` instead of crashing the engine round tick.
- **`EvictRoomBTreeState(roomId)` API** with no-op-on-missing
  semantics. Wired up via callback in 2026-04-18's rooms-package
  pass.
- Negative-cache hot-reload assumption documented with
  `TODO(hot-reload)` marker. (No hot-reload exists today, so the
  cache is correct; comment for future-you when it's added.)

## 2026-04-16 — Code Cleanup 1.7: Performance Pass + Bug Fixes

### Performance (Stage 1.7)
- **Zone-activity lane split:** mobs in zones with zero players skip
  combat, progression, mutation acquisition, charm-state, and crafting
  every round. Idle mobs still tick cooldowns, buff/condition durations,
  charm duration, combat-memory expiry, and death checks so timers and
  DoTs keep working. Active-zone behavior is unchanged.
- **Registry mutexes:** `internal/mobs` and `internal/users` global maps
  now use `sync.RWMutex`. Closes a latent race between the HTTP admin
  dashboard and the main game loop.
- **PruneVisitors fast path:** empty-map early return on room cleanup.

### Bug Fixes
- **Companions no longer pack-scatter with wild mobs** — the prior
  partial fix only guarded the death-triggered flee. Per-round pack
  membership now skips charmed/summoned mobs entirely, so your pet
  elemental stays put when a wild pack alpha dies nearby.
- **Merchants can no longer be killed via group hostility** — when you
  attacked a combat mob that shares a faction group with a merchant
  (e.g., both in "townfolk"), the merchant was picking up aggro on
  their next room entry or behavior-tree tick. Non-combatants now
  ignore group hostility entirely, matching the direct-attack guard
  that was already in place.
- **Vitality progression farm via HP reservation closed** — regen-based
  stat progression used the full HealthMax as its denominator, but
  Chrysalis-enchantment HP reservation clamps current HP to a lower
  effective cap. Result: a player at "effective full" HP still
  counted as depleted and rolled vitality progression every 3 rounds
  forever. The regen calculation now subtracts reservation from the
  max, so at effective cap you hit the proper short-circuit.

## 2026-04-15 — Bugfixes & QOL (Hotfix 2)

### Bug Fixes
- **Companions can no longer be targeted by other players** — taunt,
  attack, target, and throw now block ALL companions, not just your
  own.
- **Mobs no longer sit downed for minutes** — dead mobs at 0 HP are
  now swept every combat round. Fixes dismissed companions and DOT
  kills lingering in downed state.
- **Buff text tokens fixed** — meditating, illumination, stunned,
  blinded, hidden now show character names instead of raw
  `{sourcename}` tokens.

### Security
- **Bcrypt password hashing** — ported from upstream GoMud. Replaces
  unsalted SHA256. Existing players migrate transparently on login.
  Plaintext and hash-of-hash bypasses removed. File permissions
  tightened (0777 to 0600).

## 2026-04-15 — Bugfixes & QOL

### Bug Fixes
- **Spell typos no longer waste cooldown** — casting at an invalid
  target or misspelling a spell name no longer triggers the special
  move cooldown timer.
- **Assess without corpse no longer wastes cooldown** — same fix
  for the assess command.
- **Companion autoassist toggle now works** — `companion <name>
  assist off` actually prevents the companion from joining combat.
  Fixed in all three engagement paths (player attacks, mob attacks
  player, PvP).
- **Tutorial NPCs are now non-combatant** — 8 Sanctum Basin quest
  NPCs (priest, trainer, korvath, yenna, fen, saris, warden,
  scholar) can no longer be attacked or stolen from.

### Balance
- Skill progression multipliers increased:
  weapon-combat and unarmed-combat +25%, spellcasting +25%,
  manifestation +25%, rhetoric +15%.

### Admin Tools
- **Player stats table** added to the progression dashboard's
  Player Overview tab (base+training values, color-coded).
- **File-based logging** — new `Logging` config section in
  config.yaml. Logs to both terminal and rotating file when enabled.

## 2026-04-15 — JS Audit Complete (Phases 4-5)

### Death System Simplified
- **Death no longer sends you to the Shadow Realm.** When you die,
  you respawn directly at your home location with full health,
  stamina, and conviction. Stat decay and skill rust penalties
  still apply.
- Type `sethome` to set your respawn location.
- Type `help death` for updated details.

### Mob AI: Behavior Trees
- **7 mob scripts migrated** to YAML-based behavior trees with
  perception-scaled reaction delays. Smarter mobs react faster.
- **Old Edrin upgraded** — multi-phase caster boss with elemental
  summons, reveal sequence, and tactical combat AI.
- **Chrysalis Phantom upgraded** — hit-and-run assassin with
  surprise strikes, flee-and-rehide loop, and target tracking.
- **Barmaid Dal** now heckles back at tavern patrons with 4
  randomized NPC-NPC interaction sequences.

### Room Behavior Trees (New System)
- **14 room scripts migrated** to room behavior trees — a new
  first-class system for room-level AI and event handling.
- **Sanctum Basin tutorial** fully converted — all 9 tutorial rooms
  now use room behavior trees for quest progression, command
  detection, and the Awakening Rite ceremony.
- Room behavior trees support command interception, timed NPC
  dialogue sequences, and room-scoped state.

### Spell & Buff Migration
- **Fold-anchor and fold-recall** moved to native Go hooks.
- **Purge-affliction** moved to native Go hook.
- **Chrysalis-aid removed** — vestigial resurrect spell (death
  system handles respawn). Pruned from spellbooks automatically.
- **Buff flavor text** (illumination, stunned, blinded, hidden,
  meditating) moved to YAML text fields.

### Sable Portal Vendor
- **Sable migrated** from JS to behavior tree with a new
  `open_instance_portal` action for creating instanced zones.

### JS Scripting Bridge Removed
- **The entire JavaScript scripting system has been removed.**
  The goja JS engine, all 152 JS files, and the Go/JS bridge
  code are gone. All game logic now runs natively in Go via
  behavior trees, Go hooks, quest engine triggers, and YAML.
- Admin `mob create` and `spell create` commands removed (content
  is hand-authored as YAML files).

### Bug Fixes
- Fixed quest NPC item delivery — quest engine triggers now
  properly grant rewards without behavior tree interference.
- Fixed inventory stacking by base item instead of enchant state.
- Fixed enchanting preservation of instanced zone affix bonuses.
- Fixed instance portal replacement for difficulty upgrades.
- Fixed deep copy of item slices in NewMobById (prevents shared
  state corruption between mob instances).
- Fixed raise spells accepting generic corpse targeting words.
- Fixed visual room broadcasts routing through darkness filter.

### Balance
- Mob reaction delay curve tuned — base 2.0s, max 2.0s (was 3.0s
  base, 4.0s max). Perception 100 yields ~1s delay.

## 2026-04-11 — JS Audit Phases 1-3

### Phase 3: Item Cleanup + Charm Migration
- **11 dead default item JS files deleted** — all shadowed by DOGMud
  items at the same ID or completely unused.
- **Herbalism recipe page** migrated to YAML `on_use_train_skill` field.
- **Charm spell ported to Go** — opposed roll with charisma+manifestation
  vs willpower+statpool, aggro penalties, companion registration. The
  last non-mob/room spell JS file eliminated.
- Net: 13 JS files deleted, ~380 lines removed.

### Phase 2: Companion Consolidation + Config-Driven Buff Ticks
- **13 companion spell JS files replaced** by one Go function with YAML
  config (summon_mob_id, summon_base_pool, etc.). Conjure, raise, and
  summon spells all use the same code path now.
- **~10 healing/DoT buff JS files replaced** with YAML tick config
  (tick_pool, tick_percent, tick_variance). Buff tick magnitude now
  scales with caster spellcasting skill for spell-cast buffs.
- **Chrysalis-construct spell deleted** (redundant, undiscovered).
- **Minor antidote** migrated via `start_remove_buffs` field.

## 2026-04-11 — JS Audit Phase 1: YAML Text Fields

### Code Cleanup: Spell & Buff Text Migration
- **60 JS files deleted** — flavor-only spell and buff scripts replaced
  by YAML text fields on the data definitions. No gameplay changes;
  same messages, same colors, now driven by data instead of scripts.
- **13 stub room scripts deleted** — empty JS files that did nothing.
- **20 complex spell/buff scripts slimmed** — flavor text extracted to
  YAML, logic (companion spawning, charm, teleport, healing ticks)
  remains in JS.
- **New `textutil` package** — centralized token substitution
  (`{source}`, `{target}`) and text dispatch for spell/buff messaging.
  Sets the stage for ANSI-aware line wrapping in a future pass.
- **Schema docs updated** — spell and buff schemas now document YAML
  text fields. JS files are optional for flavor-only spells/buffs.
- Net result: 185 files changed, ~60 fewer lines of code, 60 fewer
  files to maintain.

## 2026-04-10 — Instanced Zones: Arena, Planar Oasis, Randomized Loot

### Zone 2.1b: North Road — River Approach
- **10 new rooms** extending the North Road northward through river
  country toward Stillwater. Stone bridge, river ford, woodcutter's
  camp, travelers' rest, and the first glimpse of Stillwater on
  the horizon.
- **Woodcutter Hagen** — NPC at the camp with dialogue about
  Stillwater, the road, and bloodline agents seen heading north.
- **Lone Traveler** — NPC at the rest stop who foreshadows Maren's
  trail and describes Stillwater.
- **River rats** and **wild dogs** as ambient hostile wildlife.
- Milestone at Wide Bend reads "Stillwater — 2 leagues."

### Hot Fixes
- **Server crash fix**: `look` command on a mob with a stale instance
  reference no longer crashes the server. Added nil checks to
  consider, locate, and mob-look commands as well.
- **Taunt exploit fix**: can no longer taunt your own companions.
- **Crafting vendors** now know all recipes in their profession
  (Voss buys moonpetal/veilbloom, Kerra buys steel, etc.).
- **Taunt exploit fix**: can no longer taunt your own companions.

### New: Instanced Zones
- Pay the **Riftkeeper Sable** (Rift Chamber, east of Temple District)
  to open a private portal to a dangerous zone. More gold = tougher
  enemies = better loot. Party up before purchasing — only current
  party members can enter.
- Portals last 30 minutes. Instances persist until all players leave.
- **`help instances`**, **`help arena`**, **`help oasis`** for details.

### New: The Arena (Instance Zone)
- A linear gauntlet of pit fighters. Push through trash mobs,
  veterans, and the Arena Champion.
- **Death ends your run.** No recall. No re-entry.
- Enemies respawn in waves — how far can you get before they
  grind you down?
- Veterans and the Champion drop unique weapons and armor.

### New: The Planar Oasis (Instance Zone)
- A 4x4x4 wrapping cube of elemental terrain — 64 rooms where
  every direction wraps around. Navigation is the challenge.
- Elementals wander the maze. Two elite elementals and one
  elemental lord (king, queen, or prince) roam randomly.
- **Death allows re-entry.** Recall works. Guardians don't respawn
  — clear the cube methodically.
- Oasis gear is stronger than Arena gear.

### New: Randomized Loot System
- Tougher instance mobs spawn wearing randomly-generated equipment.
  The gear makes them harder to fight AND drops when they die.
- **Point budget system**: gold invested determines a bonus pool that
  is randomly distributed across damage, mitigation, stats, and skills.
- Items are prefixed by their dominant bonus: Keen (damage), Warding
  (mitigation), Empowered (stats), Masterwork (skills).
- Every item is unique — two runs at the same gold level produce
  different gear.
- Weapons favor damage bonuses, armor favors mitigation. A snowball
  effect creates focused items rather than thin spreads.

### New: Rift Chamber
- New room in Thornwall (east of Temple District) housing the
  Riftkeeper NPC and the rift archway.

### Balance
- Mob regen and companion scaling changes from 2026-04-09 are
  included (mob SP/CP regen 2%/tick, companion stat factor 150).

### Instance Framework
- Party-scoped access control — only authorized players can enter.
- Death policy per zone: ejected (arena) or rejoin (oasis).
- Recall blocking per zone (arena blocks, oasis allows).
- Difficulty scales linearly with gold (stat pools = gold * template
  multiplier).
- Instance cleanup when all players leave.
- Portal timer warnings at 5 minutes and 1 minute remaining.

---

## 2026-04-09 — QOL Batch, Grenades, Rhetoric Shouts

### New: Grenade System
- **Three throwable grenades** crafted via Alchemy:
  - **Flashbang** (Alchemy 35) — AoE stun + blind
  - **Firebomb** (Alchemy 25) — AoE physical damage
  - **Toxic Flask** (Alchemy 20) — AoE poison DoT
- New `throw` command: AoE opposed roll (Dex+Skullduggery vs
  Dex+Perception). Fumbles hit the thrower. Shares special move
  cooldown. Progresses Skullduggery and Dexterity.
- **Grenade aging**: grenades grow more potent over time, then
  decline and eventually spoil. Spoiled grenades in the bandolier
  are safely ejected to the ground. Spoiled grenades in your
  backpack have a chance to detonate when you check inventory!
- New material: **Putrid Residue** — salvaged from spoiled food.

### New: Rhetoric Shouts
- **Warcry** — AoE offense buff. Boosts physical damage for you,
  companions, and party members in the room. Scales with Rhetoric
  and Charisma (5-20%).
- **Rally** — AoE defense buff. Boosts dodge, parry, and block for
  all allies in the room. Same scaling curve.
- Both share the special move cooldown. Can be used before combat
  to pre-buff your group. Progress Rhetoric and Charisma.

### New: Tank Taunt
- Successful taunt now **forces the target to switch aggro** to the
  taunter. Essential for protecting companions and party members.
- Flesh Golem companion now taunts in combat — the "tank pet."
- New flavor text for aggro pulls.

### QOL Improvements
- **Sort** now moves potions AND grenades into the bandolier.
- **Sell** searches bandolier and component bag as fallbacks.
- **Auto-eject**: spoiled potions move to backpack, spoiled grenades
  drop safely to the ground.
- **Food spoiling**: crafted food now ages and can spoil. Spoiled
  food cannot be eaten but can be salvaged for putrid residue.
- **Inventory** label changed from "Potions:" to "Bandolier:".

### Balance
- **Mob regen**: stamina and conviction regen doubled (1% → 2% per
  tick) to match player rates. Reduces chip-away tactics on tough
  mobs and helps companion sustainability.
- **Companion stat scaling**: Charisma factor improved (divisor 500
  → 150). Companions get meaningfully larger stat pools.

### Bug Fixes
- Companions can no longer be ordered via `ask`. They respond with
  a blank stare instead. Companion help updated accordingly.
- Farmer and bloodline agent now wander the North Road as intended.
- Rhetoric help file now lists warcry and rally.

---

## 2026-04-08 — North Road Zone, Progression Balance, Bug Fixes

### New Zone: North Road — Southern Stretch
- **15 new rooms** west of Ashwick Crossroads: farmland road,
  crossroads village with tavern, Betta's farmstead, bandit camp.
- **Quest: The Caravan Guard** — deal with a bandit crew
  threatening the road. Kill their leader Soren and bring his
  iron pin back to the caravan master for a reward and a trade
  contact in Stillwater.
- **Quest: Betta's Silence** — discover a trace of a recent
  traveler in a farmstead barn. Betta asks you to keep quiet
  when a bloodline agent comes asking questions. Your choice
  has consequences.
- **12 new NPCs**: Corvin (whittler boy), Betta (taciturn farm
  wife), Haral (tavern-keeper with shop), Old Dessa (gossiper),
  Tam (loud farmhand), Caravan Master, ambient farmer, 4 bandits
  (coordinated group fight), roaming bloodline agent.
- **New loot**: Soren's Ironbound Buckler (mid-tier shield),
  bandit longsword, leather vest, lockpick set.
- Bandits fight as a coordinated unit (no pack scatter).
- Bandit lookout spawns hidden — high Perception detects on entry.

### Engine Improvements
- **pack_flee_immune** mob flag: mobs hold their ground when
  packmates die instead of scattering.
- **Dialogue root variants** now support grantsQuest, givesItem,
  and setsQuestFlag — quests can complete on NPC greeting.
- **Mob death quest notifications** moved to dedicated listener
  (was buried inside PackFlee handler).
- **2H weapon + shield equip bug fixed**: shields could be equipped
  in offhand alongside 2-handed weapons, applying invisible stats.
- **Level 4 mutation display**: now shows "extreme" instead of
  "unknown" for max-level mutations like Extra Arms.

### Difficulty-Scaled Skill Progression
- Spell and crafting skill progression now scales with difficulty.
  Harder spells and higher-tier recipes give proportionally more
  skill growth. Utility spells like Identify still progress skills,
  just slower than combat spells.
- Self-cast buff spells give reduced progression (50% by default).
- AoE spells cast in empty rooms no longer give progression.
- Spells no longer fire progression twice (was triggering at both
  cast start and spell resolution).
- Three new config knobs: `SpellDifficultyProgressionScale`,
  `CraftDifficultyProgressionScale`, `SelfCastProgressionMultiplier`.

### Spell Difficulty Pass
- All spells now have meaningful difficulty values (0-75 range).
  Previously most spells had difficulty 0.
- Removed Empathic Bond spell (redundant with Charm).
- `spells` command now shows Difficulty instead of Familiarity.
- Spell list sorted by category (utility → heals → buffs → damage
  → summon), then target scope, then difficulty.
- Neutral-type spells (conjure, raise, identify) now show "Self"
  instead of "Unknown" for target type.

### Bug Fixes
- **Companion corpse re-raise exploit**: dismissed companions can
  no longer be killed and re-raised via necromancy.
- **Condition duration display**: recasting a spell now correctly
  shows refreshed duration instead of stale "fading" text.
- **Multi-buy progression**: buying multiple items now triggers
  charisma and bartering progression for each item purchased,
  matching individual buy behavior.
- **Bartering skill**: bartering now actually progresses during
  buy and sell transactions (was never triggered before).

### Balance Tuning
- Vitality progression: crit-received base chance increased from
  5% to 25%. Regen progression base doubled (0.005 → 0.01).
- Weapon-combat and unarmed-combat skill progression multipliers
  increased ~20% (0.15 → 0.18).

---

## 2026-04-07 — Spell Deflection, Bank, Mutation Tuning, Mob AI

### Spell Deflection & Stoic Resolve
- **Spell Deflection**: high-Willpower characters have a chance to
  deflect incoming spells entirely (Willpower-based opposed roll).
  Mobs that deflect show attacker-facing messages.
- **Stoic Resolve**: high-Willpower characters have a chance to
  resist taunts and rhetoric attacks entirely.
- Both are percentage-based avoidance layers checked before damage
  resolution.

### Thornwall Bank
- New **bank room** in Thornwall with a bank clerk NPC.
- **Unlimited storage** with monthly per-item fees. Forfeiture
  warnings sent via inbox when fees are due.
- Room-level `StorageCapacity` field replaces hardcoded 20-item
  limit. Bank room has uncapped storage.
- Updated bank and storage help files.

### Mutation Discovery Tuning
- Mutations now **prefer deepening** existing mutations (70/30
  coin flip) over discovering new ones.
- **Rarity uplift**: higher-level characters are more likely to
  discover rare mutations (weighted pool scales with avg level).
- New config knob: `MutationDeepenChance` (default 0.70).

### Reactive Tactical AI
- Mobs can now have **tactical AI** that reacts to combat events
  in real time. Configured via YAML with triggers, actions, and
  discipline settings.
- **4 presets**: aggressive_melee, defensive_caster, ambusher, tank.
  Mobs can also define custom tactics that merge with presets.
- **19 mobs configured** across warren tunnels, Ironwind Steppe
  caves, bandits, named NPCs, and Thornwall enemies.
- Ambusher mobs (stalkers, lurkers, skulkers) flee after engaging,
  then re-hide for another surprise strike cycle.
- Caster mobs (shamans, Sylara) self-buff, heal, and interrupt.
- Tank mobs (beetle queen, sentries, Velk) call for help and
  protect allies.

### Combat Targeting
- Hostile mobs now **prefer player targets** over companions when
  choosing who to attack. Companions only get targeted when no
  eligible players are in the room.
- Mid-fight retargeting also prefers players over companions,
  keeping aggro on the player consistently.

### Cleanup
- Removed dead skills: `cast`, `ranged-combat`, `first-aid`.
- Player feedback files now persist across container restarts.

### Bug Fixes
- Fixed non-crafter merchant shops not restocking inventory.
- Fixed tutorial items having missing gold values (broke shop
  pricing).
- Fixed missing space in `assess` corpse essence message.
- Fixed mobs that die in round 1 never receiving their AI signal
  (combat_start now emits before the player's attack resolves).
- Fixed per-round triggers (health_below, etc.) never firing —
  added combat_round signal emission every round tick.
- Fixed ambusher preset trying to hide mid-combat instead of
  after fleeing.
- Restored backed-up tutorial room scripts.

---

## 2026-04-06 (evening) — Foraging, Salvage, Progression Dashboard

### Foraging
- **Iron ingots** now forageable in caves (common) and mountains
  (uncommon). Previously only available from merchants.

### Salvage
- **Spoiled/declining potions** can now be salvaged for binding
  paste (1-2 depending on salvage skill). Previously spoiled
  potions were useless.

### Economy
- Enchanter Vael's binding paste restock increased from 5 to 15.

### Admin
- **Progression Dashboard**: new admin tab with system-level health
  metrics for the use-based progression system.
  - Skill health scores (expected curve deviation, stall detection,
    clustering)
  - Population distribution charts (skill tiers + stat values)
  - Discovery health (spell/recipe flags: too_hidden, too_easy)
  - Player overview with tier badges and activity totals
  - Auto-refreshes every 30 seconds

---

## 2026-04-06 — Enchanting Rework, Multi-Arm Equip, Aggro Fix

### Enchanting System Rework
- Enchanting now targets **equipped items by slot**, not inventory.
  Use `craft <recipe>` for auto-targeting or `craft <recipe> weapon#2`
  / `craft <recipe> 2.ring` for specific slots.
- **18 enchantments** covering all equipment slots (was 10).
  New: Chitin Brace (wrist), Rootbind (belt), Rootwalker (feet),
  Chrysalis Bond (ring), Spore Mantle (shoulders), Thornguard
  (shield), Venomgrip (gloves), Shadowweave (back).
- **Mitigation coverage**: physical (body/wrist/feet), magical
  (shoulders/back), conviction (neck/ring).
- **Lifesteal**: Hungering Touch now heals on hit instead of
  flat damage bonus.
- **Thornguard**: shield enchant that deals return damage.
- **Two-handed weapons** get doubled enchantment effects and
  doubled reserve costs.
- All enchantments rebalanced to 5 tiers with standard reserve
  curve (1%/2%/4%/6%/8%).
- Existing enchanted items are automatically migrated on login.
- Help files updated for all 18 recipes.

### Multi-Arm Equipment Rework
- Arms are now grouped into **pairs**: (1+2), (3+4), (5+6).
  Two-handed weapons occupy a full pair. One-handed weapons and
  shields fill individual slots.
- Arm 1 is weapon-only; arms 2-6 hold weapons or shields.
  Maximum turtle build: 1 weapon + 5 shields.
- Equip syntax: `equip sword arm#3`, `equip shield 2.arm`.
  Two-handed weapons must target odd-numbered arms (1, 3, 5).
- Odd extra-arm counts (1 or 3) create a half-pair that holds
  one-handed items only.
- **Defense scores fixed**: parry and block now use the best
  rating across all equipped weapons/shields, not just main hand.
- **Gearup** (`wear all`) now fills extra arm slots, best items
  first.
- Inventory hides the partner slot when a 2H weapon occupies
  the pair (no more "Offhand: -nothing-").

### Companion Naming
- Use `companion <name> name <nickname>` to give your companion
  a personal name. Displays as "Nickname the Type" (e.g.,
  "Fred the Spirit Wolf") in room text, combat, and vitals.
- Names must be unique — no duplicates across companions or
  player characters. New characters also can't take a name
  that belongs to an active companion.
- Creature-type words (skeleton, wraith, elemental, etc.)
  added to the banned names list.
- Companion disambiguation now supports `2.earth` / `earth#2`
  syntax for targeting specific companions of the same type.
- Nicknames persist across logout/login.

### Skill Progression Overhaul
- **Ceiling fix**: combat skills (weapon-combat, unarmed-combat,
  ranged-combat) could never advance past ~apprentice due to an
  asymptotic ceiling in the progression formula. Now all skills
  can reach soft cap regardless of their progression multiplier.
- **Per-weapon progression**: each weapon in a multi-arm setup
  independently trains weapon-combat skill. Extra arm weapons
  now contribute to skill growth, not just the main hand.
- **Defender progression**: dodging trains unarmed-combat + dex,
  parrying trains weapon-combat + dex + str, blocking trains
  weapon-combat + str. Defense type is tracked per-round.
- **Manifestation multiplier**: bumped from 0.3 to 0.5, matching
  spellcasting progression rate.

### Unarmed Combat Rework
- **Both hands attack**: every empty hand is a fist. Bare-handed
  fighters get 2 fist attacks. With extra arms mutation, up to 6.
- **Mixed setups**: sword in one hand + empty offhand? The free
  hand still punches. Works with all arm slots.
- **Fist/claws weapons**: train unarmed-combat skill, not weapon.
- **No parry**: unarmed-style fighters (fists, claws, bare hands)
  can only dodge. You can't deflect a blade with your knuckles.
- **Speed bonus**: unarmed attack speed increased (1.4 → 1.8) to
  compensate for dodge-only defense.
- **No dual-wield penalty**: natural weapons fight penalty-free.

### Defense Crit Rework
- **Parry crit → RIPOSTE**: free counter-attack at half weapon
  damage. Replaces the old disarm-on-parry mechanic.
- **Dodge crit → SWEEP**: automatic trip attempt that ignores
  the special move cooldown. Can knock the attacker prone.
- **Block crit → SHIELD SLAM**: automatic bash attempt that
  ignores the special move cooldown. Can knock them down.
- All three use distinctive cyan-bold messages to stand out in
  the combat scroll.
- **Disarm reworked**: disarmed weapons now go to inventory
  instead of dropping on the ground (grapple disarm still exists).

### Balance
- Assess command now has a 6-round cooldown.
- Companion corpses can no longer be re-raised.
- Healing potion durations halved.
- Equipped items now encumber 50% less.
- Shop restock rate tripled (6 hours → 2 hours).

### Bug Fixes
- **Aggro cleanup on mob flee**: players no longer get stuck
  "in combat" after all enemies flee the room.
- **Staff combat messages**: removed all "two-handed" and
  "both hands" references. Staves can be equipped one-handed
  in extra arm slots.
- **Extra arm attack messages**: weapons in extra arms now show
  their own name in combat text instead of the main hand weapon.
- **Surprise attack arms 5-6**: surprise strike now swings all
  equipped weapons, including extra arms 3 and 4.
- **Merchant kill exploit**: non-combatant NPCs (merchants, quest
  givers) no longer flee from pack scatter, can't enter the combat
  loop, and can't be provoked into fighting.
- **Companion auto-assist aggro**: player now properly engages when
  their companion is attacked. Previously a dummy aggro with no
  target was set, leaving the player stuck swinging at nothing.
- **Target switch on dead targets**: switching targets when your
  current target is dead is now free — no skill roll, no round
  penalty.
- **Fold-recall clears combat**: casting fold-recall now ends combat
  before teleporting. New `EndCombat()` scripting API for spells.
- **Wooden shield price**: fixed inflated auto-value (was ~430g from
  legacy DamageReduction formula, now 8g).
- **Chrysalis cores**: Apothecary Voss and Alchemist Yenna now buy
  chrysalis cores.

## 2026-04-04 — Living Economy, Gear Upgrades, Spell & PvP Fixes

### Bug Fixes (Round 2)
- **Tail slot no longer shows** when the character lacks the tail
  mutation. Was caused by EnableAll() resetting the disabled state
  before the tail check ran.
- **Companions no longer despawn** from the idle boredom timer.
  Wolf spirits and other charmed companions now persist properly,
  fixing missing vitals bars in the web client.
- **Mobs targeting your companions now show red** in the look
  command, same as mobs targeting you directly.
- **Duplicate companion vitals fixed** — same-name companions
  (e.g., two skeletons) now show separate health bars.
- **Gossip quality improved** — NPCs now use different phrasing
  for local vs. distant events, and each gossiper tracks recently
  mentioned events to avoid repetition.

### Living Economy
- Merchants now track finite stock and gold. Prices rise when
  stock runs low and drop when overstocked.
- Crafter NPCs restock materials periodically (with flavor
  text) and craft items to sell — prioritizing self-gear
  upgrades, then profitable crafts, then salvage.
- Merchants will buy craft materials matching their trade,
  potions (unless they craft that potion themselves), and gear
  that upgrades their own equipment — including paired slots
  like rings and wrists. Specialists won't buy materials from
  other professions.
- Shopkeepers are now non-combatant — they cannot be attacked,
  stolen from, or targeted by harmful spells.
- Bartering skill now affects buy and sell prices at shops.

### Under the Hood
- Mobs now advance stats and skills from basic attacks, special
  moves, and spellcasting — same progression system as players.
- Combat commands (bash, kick, trip, grapple, cast) now handle
  skill progression in the shared action layer rather than
  separately for players and mobs.
- Mob howl and player taunt now share the same underlying
  conviction-damage mechanics. (Also fixed howl not applying
  the skill-weight multiplier to rhetoric.)
- Bite and hamstring are now shared actions, ready for future
  player species-gated abilities.

### Bug Fixes
- **Area harm spells no longer damage the caster's companions.**
  This was caused by the spell resolution step overwriting the
  companion-exclusion filter from cast initiation.
- Single-target harm spells now prevent targeting your own
  companion ("You can't target your own companion with a
  harmful spell.").
- Charmed mob casters no longer hit their owner or the owner's
  other companions with area spells.
- Casting an area spell with no valid targets now gives feedback
  ("Your spell erupts outward but finds no targets.") instead
  of silently consuming conviction.
- PvP is now properly blocked across all combat entry points
  (attack, bash, kick, trip, grapple, taunt, shoot, spells).
- Fixed enchanting craft command parsing for hyphenated recipe
  names (e.g. "craft honed-edge knuckles" no longer fails).
- Shop listing now shows correct finite stock and dynamic
  prices instead of infinite legacy quantities.

### Cleanup
- Removed deprecated mob commands: roar, throw, backstab.
- Renamed mob `alchemy` command to `selljunk` (converts
  inventory to gold — not related to player alchemy).

---

## 2026-04-03 — Manifestation, Companions, Necromancy, Elementals, New Zones

### New Content
- New hidden areas have been added to the world. Sharp-eyed
  adventurers may discover passages others have overlooked.
- A reclusive figure lives off the beaten path. Not everything
  is as it appears — tread carefully.
- Lockpicks and disarm kits now available from certain merchants.
- Crafters can forge superior tools at high skill levels.
- Powerful caster equipment can be found by those who earn it.

### New Mechanics
- **Defuse command** — disarm traps on locks before picking.
  Requires a disarm kit. Higher tier kits improve success.
- **Flee rework** — flee is now an opposed roll (Dex+skullduggery
  vs Dex+unarmed-combat). Rogues are better at escaping.
  Can't flee while grappled. Prone halves flee chance.
- **Fist weapons** — new weapon subtype using unarmed-combat skill.

### Quality of Life
- `idea` is now an alias for `suggest`.
- `disarm` is now an alias for `defuse`.
- `lockpick` and `pick` are aliases for `picklock`.
- Companions prevent sneaking — dismiss before stealth.
- Companion corpses cannot be raised by necromancy.

### Bug Fixes
- AOE harm spells no longer damage the caster or their companions.

### New System: Manifestation Skill
A new charisma-based skill governing summoning, conjuring, charming,
and raising undead companions. Manifestation spells use Charisma
instead of Willpower for fold rate and discovery.

### New System: Unified Companions
Pets, summoned creatures, conjured elementals, charmed mobs, and
raised undead all share a unified companion system. Summoned,
conjured, and raised companions persist across restarts. Charmed
companions are temporary — they don't survive server restarts.
All companions show in the vitals panel, respond to autoassist,
and can be buffed with help spells.

- `companion` / `companions` — view companion vitals and stats
- `dismiss` — release a companion (warning: full betrayal)
- `assess` — study a corpse for necromantic potential
- `{pet_hp}`, `{pet_sp}`, `{pet_cp}` prompt tokens

### Necromancy (6 undead types)
Raise undead from corpses. Stronger corpses support more powerful
types. Power scales 50/50 from caster stats and corpse strength.
- Skeleton, Zombie, Wraith (caster), Spectre (conviction caster),
  Vampire (life drain bite), Flesh Golem (absorbs corpses)

### Conjure Elementals (5 types)
Conjure elemental companions from nothing. Very high conviction
cost — conjuring a magma elemental drains nearly your entire pool.
- Water (tank), Earth (bash), Air (evasive), Fire (return damage),
  Magma (bash + return damage, skill gate 60)

### Charm Spell
Opposed roll (Charisma+manifestation vs Willpower+statpool%) to
convert hostile mobs into companions. Harder against targets in
combat. Duration-based with diminishing re-rolls — your hold
gradually weakens until it breaks or you reassert control.

### New Combat Mechanics
- **Return damage** — fire/magma elementals reflect melee damage.
  Also available via equipment and buffs (battlerager armor, etc.)
- **Natural bash** — earth/magma elementals bash without shields
  ("crushing slam" instead of "shield bash")
- **Grapple immunity** — wraiths, spectres, air and fire elementals
  can't be grappled or grapple others
- **Vampire bite** — life drain special attack

### Aggro Rework
Centralized aggro state management fixes multiple companion combat
bugs. Players now properly retarget when companions kill their
target, when targets flee, and when new threats appear.

### Bug Fixes
- Enchanting target search broken by multi-word recipe names
- Conditions display showed total duration instead of remaining
- Infinite gold exploit — merchants pay from own gold pool
- Companion duplication on browser refresh
- Summon species corrections (were using rodent stats)
- Pack flee excludes companions
- Stale aggro from companion kills
- Web client vitals panel resizes with companion rows

### Balance
- Melee skill progression 0.20 → 0.15 (auto-attack now works)
- Spell damage scale 1.6 → 1.2 (progression provides natural scaling)
- Merchant stats buffed (85-150 statpool, 50-300g gold)
- Corpse decay 1 hour → 4 hours (for necromancy)

## 2026-04-02 — Command Unification + Bug Fixes

### Command Unification (feature/command-unification)
Major architectural rework unifying player and mob command systems
through shared core logic. Both sides now call the same underlying
actions for all major game commands.

**Shared Actor System:**
- Actor interface in `internal/actions/` abstracts over players
  and mobs. Shared actions operate on either actor type.
- Atomic transfer primitives (TransferItem, TransferGold) with
  rollback prevent item duplication and loss.
- Registry audit at startup warns about unintentional command gaps.

**Unified Commands:** say, emote, drop, remove, equip, get, give,
go, bash, kick, trip, grapple, shoot, attack, cast, sneak, craft.

**Combat Parity:**
- Kick now selects stomp/knee variants for mobs (position-aware).
- Trip uses tailsweep for mobs with tail mutation.
- Shared combat helpers (target resolution, cooldowns, analytics).

**Progression Parity:**
- Mobs now advance stats and skills from combat, casting, crafting.
- Player auto-attack melee progression was broken (never fired) —
  now works correctly.
- Caster mobs discover new spells as spellcasting skill increases.
- Mob sneak uses opposed rolls instead of auto-succeeding.

**Mob Crafting:**
- Mobs can now craft items via the shared craft system.
- Crafting completion fires skill progression for mobs.

### Fixes
- **Hidden mob perma-stealth:** Root cause found — permabuff system
  re-added Hidden buff after every Validate(). Fixed with
  RemovePermaBuff + proper combat loop integration.
- **Hidden mob surprise attacks:** Mobs properly get [SURPRISE ATTACK]
  when ambushing from stealth. Hidden buff clears after first strike.
- **Duplicate "prepares to fight" message:** Suppressed when mob
  re-attacks the same target.
- **Sneak in combat:** Blocked for both players and mobs — sneaking
  mid-combat doesn't make sense and caused perma-hidden bug.
- **Conditions duration display:** Was showing total duration instead
  of remaining rounds (swapped return values).
- **Infinite gold exploit:** Merchants now pay from their own gold
  pool when buying items. Refuse if they can't afford it.
- **Defense hint:** Now points to `help defense` instead of a
  nonexistent `defense` command.

### Balance
- Melee skill progression reduced from 0.20 to 0.15 — the bump
  was compensating for broken auto-attack progression (now fixed).
- Spell damage scale reduced 25% (1.6 → 1.2) — progression now
  provides natural scaling.
- Merchants buffed: higher stats (85-150 statpool), gold reserves
  (50-300g), Siv armed with a dagger.

## 2026-04-02 — Bug Fixes & Polish

### Fixes
- **Enchantment idle bug:** Chrysalis enchantments (honed edge, etc.)
  no longer progress while idle. They now only tick during combat.
- **Web client side panels:** Map, Communications, and Vitals
  windows now resize and reposition dynamically when the browser
  window is resized (both horizontally and vertically). Vitals
  no longer gets cut off on smaller screens like laptops.
- **Small screen support:** Side panels are hidden entirely on
  very small screens (phones/small tablets under 768px) to keep
  the terminal usable.

### Content
- **help equipment:** New help file covering all equipment slots,
  back slot trade-off (cloaks vs backpacks), belt slot trade-off
  (belts vs bandoliers), and the component bag system.

## 2026-04-01 — Quest Engine

### New System: Quest Engine
A complete YAML-driven quest engine that replaces JavaScript
scripts for all quest logic. Quests are now defined entirely in
data files with declarative triggers and conditions.

- **9 event types:** room_enter, room_interact, item_gain,
  item_give, mob_death, skill_use, command, dialogue,
  quest_granted (for chaining steps automatically).
- **Trigger actions:** grant quest tokens, give/consume items,
  send text, NPC dialogue sequences, teleport, spawn mobs,
  teach spells, apply buffs, set quest flags.
- **Quest flags:** branching quests track which path the player
  chose. Flag-gated dialogue shows different content per path.
  Undeclared flags panic at startup to catch typos early.
- **hint command:** type `hint` for guidance on your current
  quest step. Hints give explicit directions and next actions.
- **Verbose quest debugging:** admins can enable per-player
  quest debug logging with `questdebug <player>`.

### All Quests Ported (1-16)
Every quest in the game now runs through the quest engine:
- **Quest 1 (Sanctum Trials)** — full tutorial with ceremony
  sequences, mutation grant, shopping/equip/combat/magic steps.
- **Quest 2 (Warren Compact)** — salve delivery to tunnel shaman.
  Mobs become peaceful after quest completion.
- **Quest 3 (Scholar's Collection)** — dual-item delivery with
  flag tracking for partial completion.
- **Quests 4-7** — item delivery and combat quests across
  Dustwalk Road, Watchers Crossing, and Thornwall Outskirts.
- **Quest 8 (Missing Person)** — investigation quest in Thornwall.
- **Quest 9 (Tithe Audit)** — ledger delivery to Priest Olen.
- **Quest 10 (Drowning Post)** — protection notice to Velk.
- **Quest 11 (Windwarden's Dilemma)** — opposed branching quest
  with quest flags. Choose Sylara or Rhett; the other dismisses
  you. Flag-gated followup quests (12 or 13).
- **Quests 12-13** — path-exclusive followup quests (Covenant
  vs Extraction) gated by Q11 branch flag.
- **Quest 14 (The Undertow)** — 6-step dungeon crawl with cellar
  gate, tally stick discovery, strongbox key/open interaction,
  and bribe ledger delivery. Full room_interact support.
- **Quest 15 (Peddler's Freight)** — crate delivery with combat
  or diplomacy paths.
- **Quest 16 (Herbalist's Shortage)** — dual-path herb delivery
  with bypass for players who explore first.
- **Quest 17 (Empty Cottage)** — converted to lore discovery
  (no longer a tracked quest).

### Bug Fixes
- **Quest re-grant prevention** — fixed 18 dialogue nodes across
  15 files where completed quests could be re-offered. Added
  runtime validation that warns if a quest-granting node is
  missing its end-token exclusion.
- **Quest hints improved** — all quests now give explicit
  step-by-step directions with cardinal directions and counts.
- **Dialogue hints** now display as narrator text, not NPC speech.
- **Branching quest dismissals** — wrong-path players get clear
  rejection instead of confusing keyword matches.
- **Shadow Realm combat trap** — fixed a bug where players could
  get stuck in the Shadow Realm with stale combat state after
  the warden-bandit alliance fight.
- **False skill-up messages** — skill progression messages no
  longer fire on critical failures or first mob kills when no
  real skill gain occurred.
- **Alchemy recipe cleanup** — removed legacy duplicate starter
  recipes that confused new players in the tutorial. Tutorial
  now uses healing salve instead of removed healing poultice.

### Balance
- **Moon phase effects doubled** — full/new moon bonuses and
  penalties are now more noticeable.

### Migration
- Players on removed quest steps are automatically reset to
  "start" on server startup. Quest 17 progress removed entirely.
- Quest 11 branch flags inferred from Q12/Q13 progress for
  existing players.
- Legacy healing poultice and stamina draught auto-converted to
  new alchemy equivalents.

---

## 2026-03-31 — Salvage System

### New Feature: Salvage
Break down crafted items and tagged salvageable items to recover
crafting materials with the new `salvage` command.

- **New skill: Salvage** — standalone Perception-based skill in
  the "scavenger" profession alongside Search. Recovery chance
  scales with skill via a sqrt curve (15% at novice, up to 85%
  at master). Each ingredient is rolled independently.
- **Recipe reverse-lookup** — any item produced by a crafting
  recipe can be salvaged at the matching station for free.
- **Salvage kit** — sold by Fence Dealer Siv in Thornwall's back
  alleys for 1g. Allows salvaging anywhere without a station.
  Consumed on each use.
- **Tagged items** — non-crafted items can be marked salvageable
  with `salvage_returns` on their item spec. Always requires a
  salvage kit.
- **Multi-round activity** — salvage duration scales with
  ingredient value (1-5 rounds). Interrupted by combat.
- **Item always consumed** — even if no materials are recovered,
  the item is destroyed.
- Type `help salvage` in-game for full details.

---

## 2026-03-31 — Bug Fixes & QoL

### Features
- **ASCII Charset Mode:** `set charset` toggles between UTF-8 and ASCII
  display. Legacy clients (zMUD etc.) that show garbled box-drawing
  characters can switch to clean ASCII mode. Persists across sessions.
- **Mutation Help Files:** All 40 mutations now have individual help
  pages (`help healing-gel`, `help extra-arms`, etc.).

### Bug Fixes
- **Skill progression messages fixed:** Critical hit "technique improves"
  messages were firing on every crit regardless of whether the skill
  actually advanced. Now only shows when a real gain occurs.
- **Harm spell exploit closed:** Casting harm spells with no target no
  longer grants free spellcasting progression.
- **Harmful buffs trigger aggro:** Spells like Nerve Disruption that
  apply debuffs now properly start combat, matching damage/dot/knockdown
  behavior.
- **Tutorial directions corrected:** Directions to the Training Yard
  now correctly say north-then-east (was "northeast").
- **Removed misleading combat-end message:** The generic "rage subsides"
  text no longer appears after every kill.

### Balance
- **Combat skill progression bumped:** Weapon-combat and unarmed-combat
  progression rate increased from 0.12 to 0.20. These skills were
  advancing too slowly relative to other skills.

---

## 2026-03-30 — Alchemy Rework (Phase 1-3)

### Alchemy Overhaul
- **Potion Aging:** Potions now age through five phases (Fresh →
  Fermented → Peak → Declining → Spoiled). Peak potions are 30% more
  potent. Spoiled potions cause nausea and triple toxicity.
- **Bottle Tiers:** Four bottle types control aging speed. Clay flask
  (ages 3x faster, cheap), glass vial (baseline), sealed phial (half
  speed, jewelcrafting), crystalline decanter (quarter speed, advanced
  jewelcrafting).
- **Toxicity System:** Every potion adds toxicity. Exceed your limit
  and your body rejects the brew. High toxicity causes stat penalties.
  Toxicity decays naturally over time.
- **Craft Skill Matters:** Higher alchemy skill at brew time means
  stronger, longer-lasting potions that age slower.
- **Skill-Based Detection:** Examining potions reveals aging info
  based on your alchemy skill. Novices can't tell fresh from spoiled.

### New Potions (21 recipes)
- **Pool Regen (7):** Healing salve, stamina tonic, conviction
  draught, warrior's brew, preacher's tincture, windrunner draught,
  elixir of renewal.
- **Combat/Utility (10):** Ironhide brew, mindshield elixir,
  veilguard tonic, stone stomach, cat's eye draught, swiftfoot
  essence, berserker elixir, silver tongue oil, battle trance,
  purging draught.
- **Progression (4):** Essence of growth, savant's infusion, mutagen
  brew, chrysalis catalyst. These accelerate character development
  but reserve portions of your resource pools.

### Potion Bandolier
- New belt-slot item that auto-routes potions and reduces their
  weight. Two tiers: leather (6 slots, 30% weight reduction) and
  reinforced (12 slots, 40% weight reduction). Craft via tailoring.

### New Materials
- **Moonpetal** — rare forage, night only.
- **Veilbloom Petal** — very rare forage on the steppe.
- **Serpent Venom Sac** — drops from river lurkers and blind stalkers.
- **Ironbark Shaving** — uncommon forest forage.
- Clay flask sold by Apothecary Voss.

### Consumption Rework
- Drinking a potion now checks toxicity before consuming. If you'd
  exceed your maximum, the potion is rejected.
- Aging phase affects potency: peak potions last 30% longer, declining
  potions are weaker, spoiled potions cause nausea + 3x toxicity.
- Craft skill at brew time scales potion duration (skill 20 = +20%).
- Bottle type is stamped on the potion at craft time, determining its
  aging speed for its entire lifecycle.

### Maker's Mark
- Skilled crafters (skill 30+) now leave their name on items they
  craft. Examine a crafted weapon, potion, or piece of armor to see
  "Made by {name}." Purely cosmetic — does not affect stacking.

### QoL
- Spoiled potions display as "(turned)" in inventory for alchemists.
- Potions in bandolier show in a dedicated "Potions:" section.
- Drink command pulls from bandolier first (oldest potion).
- Five new alchemy-related gameplay tips in the hints rotation.
- Old potions and recipe knowledge auto-migrate on login.

### Bug Fixes
- **Velk bribe ledger quest:** Fixed quest getting stuck at 83%.
  The dialogue was still asking for the ledger after it had been
  given. Players with the stuck quest should now be able to complete
  it by talking to Velk.
- **Sylara spirit fetish spell:** Fixed "You need a spirit fetish"
  error when the fetish was in the component bag. Spirit fetishes
  now stay in the regular backpack where the spell can find them.
- **Text wrapping:** Say, shout, whisper, emote, and party chat
  now wrap the full message (including speaker name) at 80 chars
  instead of wrapping text alone at 65 then prepending the name.
- **zMUD compatibility:** Fixed display flashing for legacy MUD
  clients that don't support GMCP. The server no longer sends GMCP
  data to clients that haven't completed the GMCP handshake.
- **Description wrapping:** Player and NPC descriptions no longer
  double-wrap with orphaned words. Descriptions are stored raw and
  wrapped once at display time. Existing player descriptions are
  auto-migrated on login.
- **Floor item stacking:** Identical items on the ground now display
  with (xN) count instead of separate lines.
- **Vendor room clutter:** Removed crafting materials baked into
  7 vendor/crafter room templates that respawned every restart.
- **Drop all:** No longer drops your gold. Use "drop N gold" to
  drop gold explicitly.

---

## 2026-03-30 — Mutations, Balance, Documentation & QoL

### New Mutations
- **Chameleon Skin** (rarity 7) — +30 stealth bonus, +10 dodge.
  Costs charisma and natural armor. Conflicts with thick-hide.
- **Tail** (rarity 8) — Adds Tail equipment slot, disables Legs
  slot. Reskins trip to tailsweep (better damage and knockdown).
  Three tail attachments: weighted cap, spiked band, bladed sheath.

### Stealth Improvements
- Characters emitting light have their sneak score halved.
- Moving while sneaking costs 50% more stamina.
- Hidden mobs now get surprise attack on their first strike.

### Spell Duration Scaling
- All spell durations now scale with fold count, spellcasting
  skill, and willpower via universal formula.
- Higher-fold spells naturally last longer. Investing in willpower
  and spellcasting extends everything.

### PowerScore Rework
- Skills are now a major factor (sqrt of total ranks × 25).
- All three resource pools count (HP + SP×0.5 + CP×0.5).
- Mutations contribute 20 points per level.
- KD ratio replaces raw kill count (kills/deaths × 10, cap 50).
- Magic/conviction offense normalized against physical.
- Defense weighted 3× more heavily.

### Defense Balance
- Dodge effectiveness 0.97→0.95, Parry 1.0→0.97, Block 1.02→1.05.
- New clinch defense penalties: dodge 0.80, parry 0.83, block 0.85.
- New grounded defense penalties: dodge 0.75, parry 0.77, block 0.80.
- Prone dodge/parry penalties 0.95→0.93.

### New Commands
- **afk** — Manual AFK toggle with optional message. Shows (AFK)
  next to your name in the room. Auto-clears on any input.
- **setdesc** — Set your own character description.

### Crafting
- Craft list now shows recipe completion tier per skill and overall.
- Subcomponent recipe thresholds lowered (steel ingot, chain links,
  chrysalis setting).

### Documentation
- Help files for all 39 spells, 47 recipes, and 4 combat skills.
- Completeness tests ensure new content always has help files.
- 15 new gameplay tips added to the hint rotation.

---

## 2026-03-29 — Combat, Stealth & Spell Balance

### Kick Rework
- **Kick** is now a powerful unarmed strike (damage doubled from 0.40 to
  0.80). Three automatic variants based on combat position:
  - **Kick** (standing target): 35% knockdown chance.
  - **Stomp** (prone target): 1.20x damage, bypasses half armor,
    extends prone duration. The payoff for knocking someone down.
  - **Knee** (grapple, in control): 1.0x damage, works in clinch/ground.
- `stomp` and `knee` are command aliases for `kick`.

### Opening Fights with Special Moves
- Kick, bash, trip, grapple, and taunt can now initiate combat by
  naming a target (e.g., `kick bandit`). No longer requires attacking
  first.

### Stealth System
- Players now detect hidden mobs when entering a room via opposed
  Perception+Search vs Dex+Skullduggery roll.
- Rogue NPCs added: Blind Stalker, Pale Lurker, Warren Scout, Tunnel
  Lookout, and Goblin Scout spawn hidden and ambush on entry.
- Thornwall Highwayman, Smuggler Runner, and Torvan Cresk use tactical
  combat stealth.

### Caster NPCs
- Elder Saris, Priest Olen, Geomancer Rhett, and Windwarden Sylara now
  have spellbooks and cast buff spells while idle. Attack them and
  they fight back with appropriate magic.

### Buff/Ward Spell Rework
- **Shield spells** now scale by spell magnitude. Conviction Ward is
  75% strength (quick/cheap). Chrysalis Cocoon is 125% strength and
  grants magical/conviction mitigation. Both last 2.5x longer.
- **Iron Will** now provides magical and conviction damage mitigation
  alongside the willpower boost. Lasts 50 rounds (was 8). Costs more.
- **Chrysalis Haste** costs more but lasts twice as long.
- **Vital Surge** regen lasts 3x longer.
- **Empathic Shroud** no longer cancels on entering combat.
- **Veil Sight** now grants see-hidden (was incorrectly giving light).
- **Skill Attunement** and **Mutation Catalyst** last 10x longer but
  cost 3x more conviction.
- All debuffs (Nerve Disruption, Mind Fog, Sensory Overload, Psychic
  Anchor) last 50% longer.

### New Commands
- **reply** — Whisper back to the last person who whispered to you.
- **rep/report** — Broadcast your vital bars to the room, party, or
  a specific player.
- **setdesc** — Set your own character description.

### Stat Progression
- Taking a critical hit now triggers stat progression: physical crits
  improve vitality, magical crits improve willpower, rhetoric crits
  improve charisma.

### Balance
- Taunt damage +50% (RhetoricDamageScale 2.0 → 3.0).
- Spell damage -14% (SpellDamageScale 1.85 → 1.6).
- Subcomponent recipe discovery thresholds lowered: Steel Ingot 10→4,
  Chain Links 15→7, Chrysalis Setting 15→7.

### Other
- Spells list now sorted by fold count (simplest first).
- Leaderboard expanded from 10 to 20 entries.
- 4 new tailoring recipes: Leather Backpack, Reinforced Travel Pack,
  Artisan's Satchel, Master's Component Case.
- Component bag capacities increased (20/40/75).
- Apothecary Voss now sells alchemy ingredients.

---

## 2026-03-29 — Equipment Slot Expansion + Component Bags

### New Equipment Slots
- **Wrist** (x2) — Bracelets and bracers now have their own slots
  instead of using the ring slot. Existing bracelet items have been
  updated.
- **Shoulders** — Pauldrons, mantles, and shoulder armor.
- **Back** — Cloaks for stats, or backpacks that reduce the effective
  weight of your carried items. Choose wisely.
- **Second Ring** — Two ring slots instead of one.
- **Component Bag** — A dedicated bag for crafting materials. Materials
  auto-sort into it on pickup. Use `sort` to migrate existing
  materials. Buy a starter pouch from Weaver Maren in Thornwall.

### Extra Arms Mutation — Levels 3-4
- The Extra Arms mutation can now reach levels 3 and 4, granting up
  to four additional arms (six total weapon slots).
- Each extra arm comes with an extra wrist slot for bracelets.
- Higher levels impose steeper charisma penalties and aggro, with
  diminishing dexterity returns.
- Combat hit penalties scale: +20 per arm beyond your offhand.

### Component Bag System
- Crafting materials with the `is_component` flag auto-route to your
  component bag on pickup and purchase.
- The `sort` command moves existing materials from your backpack into
  the bag.
- Crafting consumes from the component bag first, then your backpack.
- Component bags reduce the effective weight of their contents.

### Bug Fixes
- Extra arm weapons now correctly count toward carried weight.
- Bracelet items correctly equip to wrist slots instead of ring.
- Cloaks moved from neck slot to back slot (automatic migration
  on login for existing characters).

---

## 2026-03-29 — Combat, Spell & Crafting Fixes

### Spell Fixes
- **Sparks** now correctly hits all enemies in the room (was only
  hitting one target despite being an area spell).
- **Mend All** now actually heals (was missing effect type data).
- **Hemorrhagic Wave** rebalanced: folds 10→20, magnitude 300→225.
  Still powerful AoE but no longer trivializes encounters.
- **Healing spells can now target downed players**, enabling
  revive-style healing like Chrysalis Aid. Harm spells skip
  downed players.
- **Self-targeting blocked** for harm spells — Conviction Spike
  and Nerve Disruption can no longer be cast on yourself.
- **Player-targeted harm spells** now deal damage and trigger
  reciprocal aggro (previously did nothing).
- Pet damage messages no longer duplicate in same-room combat.

### Crafting Fixes
- **Skill level-up messages** no longer repeat on every successful
  craft. The "Your X skill reaches Y!" message only appears when
  your skill tier actually increases.
- **Recipe discovery** now filters by the skill you're currently
  crafting. No more learning enchanting recipes while blacksmithing.
- **Casting and sneaking blocked while crafting.** Moving to another
  room cancels the active craft.

### Other Fixes
- **Title command** no longer shows raw numbers. Mutation load and
  skill progress use descriptive labels (fledgling/seasoned/etc).
- **Apothecary Voss** now sells alchemy ingredients instead of
  enchanting binding paste.

---

## 2026-03-29 — Critical Fixes + Inventory Rework

### Critical Bug Fixes
- **Death loop fix**: Players can no longer get permanently stuck
  in the Shadow Realm with stale combat state. Root cause fixed
  (mobs could re-aggro dead players), plus safety net and escape
  hatch so the portal always works.
- **Spell scripts now work for players**: Fold Anchor, Chrysalis
  Aid, Summon Steppe Spirit, and other script-based spells were
  silently broken — onMagic/onCast/onWait callbacks never fired
  for player casts. All three hooks are now wired into the cast
  pipeline.
- **Fold Anchor split**: Now two spells — `fold-anchor` (set) and
  `fold-recall` (teleport back). Players who knew fold-anchor
  automatically receive fold-recall on login.
- **Quest spell rewards**: Quests can now teach spells on
  completion. The Warden's Covenant (quest 12) now properly
  grants Summon Steppe Spirit.
- **Fetish gating**: Windwarden Sylara no longer gives unlimited
  spirit fetishes. If you already have one, she refuses.

### Inventory Rework
- **Diku-style disambiguation**: Use `3.dagger` or `dagger#3` to
  target a specific item when you have duplicates. Use `all.item`
  with get/drop to affect all matching items (e.g., `drop all.potion`).
- **Inventory stacking**: Identical items now group together with a
  count, e.g., `iron ingot (x5)`. Items with different enchantments
  remain separate.
- **Equipped item targeting**: `look` and `identify` now search
  your backpack and equipment as a single pool. You can examine a
  wielded weapon without unequipping it — use `look 2.dagger` to
  reach the equipped one when a duplicate is in your pack.
- **Encumbrance display**: Carrying capacity has been rebalanced.
  The inventory command now shows a colored encumbrance tier
  (light / moderate / heavy / overburdened / crushed) instead of
  raw weight numbers. Add `{enc}` to your prompt to track it at
  a glance (`help set prompt`).
- **Multi-buy**: `buy 5 iron ingot` purchases multiple copies in
  one command. Stops early if you run out of gold or can't carry
  any more.
- **Enchanting targeting**: `craft <recipe> <item-name>` lets you
  choose which item to enchant. Works on equipped items too. Shows
  a numbered list when multiple targets match.
- **Look direction fix**: `look n` no longer matches inventory
  items when no north exit exists.

### Balance
- Carry capacity reduced ~78% (now Strength × 0.65, configurable).
  Being overweight costs more stamina to move and reduces combat
  swings.

## 2026-03-18 — Skullduggery Skill + Tutorial Fix

### New Skill: Skullduggery
The old `stealth` skill has been consolidated into **skullduggery**,
a full rogue toolkit with seven sub-commands:

- **sneak** — hide using opposed rolls (Dex+skill vs observers'
  Perception+search). Empty rooms auto-succeed. Party excluded.
- **steal** — take gold/items from NPCs or containers. Being hidden
  improves your chances. Replaces the old pickpocket command.
- **plant** — slip items onto NPCs or into containers unnoticed.
- **shadow** — tail a target between rooms while hidden (rank 2+).
  Room-entry detection checks on each transition.
- **surprise attack** — automatic extra crit hit when attacking from
  stealth. Swings all weapons with stacking hit penalties. Party
  auto-assist triggers coordinated ambushes.
- **picklock** — existing minigame, now gated behind skullduggery
  rank 1.
- **defuse** — trap disabling stub for future content (rank 3+).

### Stealth Detection Rework
- Hidden players are now rolled against when entering rooms
- New arrivals roll to spot hidden occupants in the room
- Party members excluded from all detection checks

### Stealth & Movement Improvements
- Hidden players skip room onEnter scripts (NPCs no longer greet
  you when they can't see you)
- Sneak buff now applies immediately (no tick delay before moving)
- Per-weapon surprise attack messaging shows each weapon's hit
  and damage individually

### Quality of Life
- MOTD now displays in a visible bordered box on login
- Skill-gated commands show "You aren't advanced enough at
  skullduggery for that" instead of "command not found"
- Updated help files for steal and plant with clear syntax and
  examples
- Added missing alchemy_bench station to Apothecary Lane (room 471)
- Added hidden buff (ID 9) to dogmud world buffs (was missing)

### Bug Fixes
- Tutorial casting step now accepts spell ID shortcuts and aliases
  (e.g., `conviction-spike echo` works, not just
  `cast conviction-spike echo`)
- Existing characters auto-migrate stealth skill to skullduggery
  on login

## 2026-03-17 — Bug Fixes, Hidden Object Discovery, Identify Spell

### Legacy Stat Scaling Fixes
- Map command perception thresholds rescaled for 100-baseline
  stats (was 25/50/75, now 110/135/175). New characters start
  at tier 1 zoom instead of getting max zoom immediately.

### New Spell: Identify
The old `inspect` command has been removed and replaced with
the **Identify** spell (Mental school).

- Cast `identify <item>` to reveal an item's properties using
  descriptive language (no raw numbers shown to players)
- Works on items in your backpack or currently equipped
- Costs 20 conviction, 3 folds to cast, 30-round cooldown
- Merchants still offer `appraise` as a non-magical alternative

### Appraise Rework
- Appraise now shows full item details (previously capped at
  tier 3). All output uses descriptive language instead of raw
  numbers.

## 2026-03-17 — Bug Fix Day + Hidden Object Discovery System

### Bug Fixes (9 issues from playtesting)
- Conviction regen bumped to 2% per tick (matches health/stamina)
- Removed legacy tame-on-kill messages (taming now uses spells)
- Fixed disarm crit triggering on unarmed/disabled-slot targets
- Fixed misspelled commands showing "can't do that in combat"
  instead of "command not recognized"
- Fixed 2H weapon + extra arms exploit (extra arm slots now
  cleared when equipping a two-handed weapon)
- Fixed fold-anchor recall failing due to type mismatch
- Fixed gossip system — NPCs now report mob kills and player
  mutations (event buffer was starving for events)
- Fixed bleedout test to match current rate (2 per tick)
- Added `{attack}` token for defense messages (resolves to
  "strike" when attacker is unarmed)

### New Feature: Hidden Object Discovery
Rooms can now contain hidden nouns and hidden containers that
players must actively discover using the search command.

- **Hidden nouns** — invisible until found via search. Once
  discovered, they appear in the room description and respond
  to `look <noun>` permanently for that character.
- **Hidden containers** — function like normal containers but
  are invisible until discovered. Locks still apply after
  discovery.
- Discoveries persist permanently per-character.

### Skill Consolidation: Search
The tracking and foraging skills have been merged into a single
**Search** skill governed by Perception.

- `search`, `track`, and `forage` all progress the Search skill
- All three commands now use gaussian dice rolls (Perception +
  Search skill bonus) instead of hard stat thresholds
- Forage difficulty varies by biome (farmland is easiest,
  cliffs are hardest)
- Existing players: Search rank = max(tracking, foraging).
  No progression is lost.

### Balance
- Extra-arms mutation restricted to species with arm slots
  (no more wolves with extra arms)
- Search skill progression only fires when there's something
  undiscovered to roll against (prevents AFK botting)

## 2026-03-14 — Zone Expansion, Spell Merge, Coordinate System

### New Zone: Marches Spur Road
A new 15-room zone connecting Watchers Crossing south to the Ashwick
Crossroads — the first road into the wider Windward Marches.

- **15 rooms** from scrubland through farmland to a waypoint inn and
  crossroads junction
- **The Broken Yoke Inn** — social hub with gossiper NPCs relaying
  world events
- **Peddler Malk** — road merchant and quest giver at a camp along
  the spur
- **Quest: The Peddler's Overdue Freight** — find a stolen freight
  crate. Solve it through combat (clear the bandit barn) or diplomacy
  (negotiate a toll with the bandit leader). Multiple breadcrumbs and
  elephant-path coverage.
- **Bandit Leader encounter** — non-hostile with a 5-round dialogue
  window before she attacks. Talk fast or fight.
- **Wildlife**: scrub coyotes, feral hogs, field sparrows, farm cats

### New Zone: Ashwick
Maren's home hamlet from the novel, 20 rooms east of the Ashwick
Crossroads. A quiet farming village with secrets beneath the surface.

- **20 rooms** — hamlet proper (central green, chapel, farmstead,
  ritual circle, well, Delia's cottage) plus forest outskirts
  leading into deep woods
- **Delia the herbalist** — quest giver and alchemy crafting station
- **Deacon Ferris** — lore NPC with quest-gated deeper dialogue
- **The Forager (Sev)** — a hollow person hiding in the woods,
  mirroring the novel's themes of identity and concealment
- **Quest: The Herbalist's Shortage** — someone is harvesting
  Delia's herbs. Negotiate with the forager or find an alternate
  source in a hidden Chrysalis-touched grove.
- **Quest: The Empty Cottage** — explore Maren's abandoned family
  home. Push a loose hearthstone to find a hidden letter mentioning
  "the hill" and "Voss in New Plymouth." Study a recipe page from
  the bedside table to advance your foraging skill.
- **Novel breadcrumbs** throughout — scorch mark on the ritual
  circle, inner orbit symbol at the well and chapel, the cottage's
  empty shelves and cold hearth. Layered discovery rewards
  attentive players without frontloading spoilers.
- **Wildlife**: timber wolves, briar hawks, forest foxes, village
  chickens, a cottage mouse

### Spell Changes
- **Fold Anchor + Fold Recall merged** into a single toggle spell.
  Cast once to set an anchor, cast again from elsewhere to teleport
  back. No more needing to learn two spells for one mechanic. Existing
  players with Fold Anchor gain recall automatically.
- New dedicated help template for Fold Anchor explaining both modes.

### Cartesian Coordinate System
- All 224 existing rooms now have hidden `coord` fields (x, y, z)
  for spatial validation
- Full coordinate map at `docs/coordinate_map.md`
- **3 geometry overlaps fixed** in Sanctum Basin and Ironwind Steppe
  where rooms occupied the same coordinate
- Cartesian consistency rules added to zone expansion standards

### Bug Fixes
- Fixed steppe rooms 3032/3082: replaced JS quest item scripts with
  native container-based spawns
- Removed extra mob spawn from goblin camp room 3070
- Deleted stale instance saves that were overriding template edits

### Infrastructure
- Zone expansion plan (ZONE_EXPANSION.md): 22 zones, ~600 rooms
  planned across the Windward Marches
- Geography aligned to novel canon (What the Moons Keep)
- AI player default host updated to dogmud.org

---

## 2026-03-05 — Ironwind Steppe Rebuild, Quests & Ecosystem AI

### Ironwind Steppe Zone Rebuild
The entire Ironwind Steppe zone was rebuilt from scratch on a clean
cardinal grid with proper reciprocal exits throughout.

- **Rebuilt entry area** (rooms 3000-3009) on a clean cardinal grid
- **Sagebrush Flats** expansion (3010-3015, 3018) with ambient wildlife
- **Northern wolf/hawk territory** (3019-3023) with predator encounters
- **Hollow and boar/viper area** (3024-3028) with burrowing wildlife
- **Ironwind Ridge column** and northern steppe edge (3029-3033)
- **Upper ridge** — nesting ledge to summit (3034-3038)
- **East ridge descent** — alcove to overlook (3039-3043)
- **Dry creek system** and ridge descent (3044-3048)
- **Creek basin depths** — undercut bank to boar wallows (3049-3053)
- **Lower creek basin** — junction to basin south end (3054-3058)
- **Basalt coulee system** east of creek basin (3059-3063)
- **Deep coulee goblin territory** (3064-3068)
- **Goblin camp interior** and coulee south exit (3069-3073)
- **Wolf Run** and eastern coulee edge (3074-3078)
- **Deep Wolf Run** and wolf ravine east column (3079-3083)
- **Lower wolf territory** (3084-3088)
- **Boar wallow column** and eastern grassland (3089-3093)
- **Mudflat/boar territory** and drinking pool (3094-3098)
- **Cave system** entrance through deepest chambers (3099-3114)

### Quests
- **Quest 12 audit** — Sylara now grants quest start directly,
  removed unnecessary Kael checkpoint that could brick progression
- **Quest 14: The Undertow** — new smuggler tunnel quest beneath
  the Drowning Post tavern in Thornwall City

### Ecosystem AI
- **Species-based pack behavior** — mobs now ally and flee based on
  shared species (SpeciesId) instead of broad group tags. Wolves pack
  with wolves, not with squirrels that happen to share a group tag.
- **Predator-prey combat** — `HatesMob()` rewritten to use species
  names. Added natural predator-prey hatred across the ecosystem:
  - Canines (wolves, foxes, dogs) hunt rodents and boars
  - Raptors (hawks, eagles) hunt rodents and serpents
  - Felines hunt rodents and insects
  - Serpents hunt rodents and insects
  - Arachnids (spiders, scorpions) hunt insects
  - Boars defensively attack canines
  - Trolls attack most wildlife species

### Balance
- Bumped player conviction regen from 1% to 1.5% per tick

### Bug Fixes
- Fixed broken ANSI tag in Old Citadel Plaza board noun
- Fixed scrubland dog species (was reptile, now canine)

---

## 2026-03-04 — Quest Fixes, Balance Tuning & Zone Repairs

### Quest Fixes
- **Velk/Elara quest** — made dialogue discoverable and unbrickable
- **Harn/Pell delivery quest (Quest 6)** — unbricked progression
- **Removed `requires` + `expiryPeriod` quest brick** from all
  dialogue files across the game. These combinations could silently
  brick quests when player memory expired.

### Balance Tuning
- Reduced `GlobalDamageMultiplier` from 1.75 to 1.25 for less swingy
  combat
- Potion buff improvements and helpfile additions
- Temple regen and hint system improvements
- Faster bleedout timer for downed players

### Zone Fixes
- Resolved 74 broken reciprocal exits across the Ironwind Steppe zone
- Fixed spatial inconsistency in Watchers Crossing river lurker loop
- Disconnected Ironwind Steppe from Thornwall temporarily until zone
  rebuild was complete

### Features
- **Player PvE death gossip** — tavern gossip system now broadcasts
  player deaths to the gossip channel (global, not just local)
- **setmotd admin command** — admins can now set the message of the
  day in-game

### Bug Fixes
- Fixed `FindRecipeByName` to prefer exact match, preventing wrong
  recipe selection when names overlap
- Removed Area field from status template to prevent zone name
  misalignment
- Aligned status template columns with consistent 12+13 char widths
- Added missing admin commands to help list with helpfiles

---

## 2026-03-03 — Launch Day Fixes

### Major Fixes
- **ANSI tag crash fix** — prevented nested ANSI tags in noun
  highlighting that caused client crashes. Root cause fix in
  `roomdetails.go` to skip nouns already inside ANSI tags.
- **Tinymap panic fix** — used `VisibleWidth()` instead of `len()`
  for tinymap padding, preventing panics from ANSI escape sequences
  in map rendering.
- **Instance save override fix** — added `instance:"skip"` tag to
  structural room fields (exits, nouns, signs) so instance saves
  can no longer silently override template data for these fields.

### Quest & Item Fixes
- Scholar quest now accepts totem and spore sac in either order
- Added `givesItem` field to dialogue engine for NPC item handoffs
- Fixed Watchers Crossing quest items using new `givesItem` system
- Replaced removed skulduggery quest reward
- Fixed `get all <container>` command support
- Fixed leaderboard stale stats and scholar `onGive` handler

### Combat & Mob Fixes
- Made mob commands darkness-aware (mobs no longer act normally in
  pitch-dark rooms)
- Enabled wolf vs boar predator/prey combat on the steppe
- Fixed web client auto-scroll behavior

### UI & Display Fixes
- Reorganized status template into logical sections
- Fixed per-player buy/equip tracking with purchase debug logging
- Renamed 'back corner' room noun to 'alcove' to fix room 472 crash
- Removed ANSI tags from descriptions where the word is also a noun
  key
- Removed long-range exits from Thornwall City templates
- Fixed Thornwall cardinality, Brecca shop inventory, copper ring
  naming
- Web portal "Who's Online" now uses Title instead of removed
  Profession field

### Documentation
- Added deployment troubleshooting guide for git sync and Docker
  cache issues
- Added compose file warnings and port conflict troubleshooting
