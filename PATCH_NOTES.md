# DOGMud Patch Notes

## 2026-04-20 — Pure Caster Archetype

### Gameplay

- **Wraiths, spectres, and air elementals are now proper mages.**
  Each maintains defensive buffs, emergency-heals when HP drops below
  40%, picks AoE damage when enemies are grouped, and single-target
  damage otherwise. Watch for `heal`, `sparks`, `conviction-barrage`,
  `mind-spike`, `conviction-spike`, and `nerve-disruption` depending
  on the mob and situation.
- **Air elemental is now a caster** (was a melee specialist in the
  previous Phase 4 release). Its stats were always caster-shaped —
  dex 20, perception 20, willpower 10 — and this update gives it the
  spellbook to match. Vampire and fire elemental stay on melee.

### Under the hood

- **New archetype `pure_caster`** sits alongside `melee_self_buff`.
  Both share the same framework — only the tree YAML and mob
  spellbook tags differ. Decision order: emergency heal → maintain
  defense → AoE if multiple enemies → single-target harm → legacy
  fallthrough.
- **`multiple_enemies` btree condition is now perspective-aware.**
  Previously it counted `players + charmed mobs` regardless of the
  calling mob's perspective, so a summoned caster with a fellow
  companion in the room saw "multiple enemies" when fighting a
  single wild mob. Now, from a charmed mob's POV, the summoner and
  fellow same-owner companions are excluded. Wild mobs (like
  bandit_leader) preserve original behavior for regression safety.
- **Three new spell categories** — `self_heal`, `harm_single`,
  `harm_multi` — tag the spells archetypes filter for. Applied to
  8 existing spells (heal, mind-spike, conviction-spike,
  nerve-disruption, sparks, conviction-barrage, hemorrhagic-wave,
  hemorrhagic-burst).

## 2026-04-20 — Companion AI Phase 4 (Melee Self-Buff Archetype)

### Gameplay

- **Vampires, air elementals, and fire elementals now maintain
  self-buffs intelligently during combat.** Each picks the
  highest-value buff it knows from its spellbook, skips buffs
  already active, respects the shared cast cooldown, and falls
  back to normal attacks when buffs are covered. A fresh vampire's
  first combat sequence casts conviction-surge for its offensive
  boost, then iron-will, then conviction-ward across the first
  ~10 rounds — then attacks with bite and flavor emotes.
- **Companions now follow their summoner through every movement
  path** — walking, recall, portal, fold-recall, sable, admin
  teleport. Mid-cast wind-ups are aborted to follow (conviction
  already spent is forfeit, same as a player self-interrupt).
  Aggro on a target that isn't in the new room ends automatically.

### Under the hood

- **Behavior-tree archetypes are now a first-class concept.** Mob
  YAML gains an optional `behavior_archetype: <name>` field that
  resolves to a shared tree file at
  `_datafiles/world/dogmud/behaviors/archetypes/<name>.yaml`.
  Resolution order: per-mob btree file wins, then archetype, then
  legacy. This unlocks future work where NPCs can switch archetypes
  at runtime (e.g., a caravan guard taking up banditry).
- **Spells gain an optional `categories:` field.** Free-form string
  list used by archetype AI to filter spellbooks by purpose. Today:
  `self_defense` (iron-will, conviction-ward, conviction-armor) and
  `self_offense` (conviction-surge).
- **New behavior-tree action `cast_best_in_category`** picks the
  highest-scoring (`base_folds × cost`) spell in a category from the
  mob's spellbook, skipping already-active buffs, components,
  summons, and insufficient CP. Self-gates on the shared
  special-move cooldown, so mobs naturally alternate between cast
  rounds and attack rounds.
- **New event `mob_combat_round`** fires per mob combatant BEFORE the
  legacy AI decision, so behavior-tree archetypes are authoritative
  for mobs that declare one. Legacy `preferredSpell` (shield-first
  priority) no longer preempts the archetype.

### Latent-bug fixes surfaced by Phase 4

These were pre-existing engine gaps that no prior code path exercised;
Phase 4's archetype is the first thing that asks a mob to self-cast
non-shield buffs, and the first to track the magical shield a
mob-cast ward applies.

- **`applyMobSelfEffect` now handles `buff` effect_type.** Used to only
  handle `heal` and `shield` — buff-type spells (conviction-surge,
  iron-will) fell through silently when mobs self-cast them. They
  now correctly apply, including per-buff tick snapshots.
- **Shield-active detection uses `ConditionShield`** instead of the
  equipment-layer `HasShield()` helper. The magical shield a spell
  applies via `AddCondition(ConditionShield, ...)` is what the
  archetype needs to check against for "already active."

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
