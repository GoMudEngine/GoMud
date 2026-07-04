# Pinnacle item primitives (Stage 1 engine)

Stage 1 built the engine primitives that legendary-BIS "pinnacle" items
draw on: data-driven procs, pool reservations, a sentient bandolier,
mutation drip, weapon hunger, item voices, an assembly-recipe
provenance gate, and a remort potion. Stage 1 shipped **no player-
facing content** — no items reference these fields yet. This page is
the content-author reference for Stage 2/3/4: every field below is
implemented and verified against the actual code (not just the design
spec), on branch `feature/pinnacle-stage1-engine-primitives`.

All fields live on `ItemSpec` (`internal/items/itemspec.go`) unless
noted otherwise.

## 1. Procs (`procs:`)

```yaml
procs:
  - trigger: on_hit           # on_hit | on_kill | on_block | on_grapple | on_spell_hit
    chance: 25                # 1-100, percent per trigger event
    cooldown_rounds: 0        # 0 = no cooldown
    effect: lifesteal         # lifesteal | steal_pool | aoe_stun | apply_condition
    params:
      ratio: 0.25
```

`Validate()` rejects unknown `trigger`/`effect` values and requires
`chance` in 1-100 — bad data panics at boot, not at first swing.

**Which equipment slot is consulted, per trigger**
(`procBearingItems` in `internal/hooks/item_procs.go`):

| Trigger        | Slot consulted |
|----------------|----------------|
| `on_hit`       | Weapon |
| `on_kill`      | Weapon |
| `on_spell_hit` | Weapon |
| `on_block`     | Offhand |
| `on_grapple`   | Body |

Only one item per trigger is ever consulted (the cost discipline is
1-2 spec lookups per swing) — a proc authored on the wrong slot for
its trigger simply never fires.

**Gate order** (`procGateOpen`): `GamePlay.ItemProcsEnabled` kill
switch → per-(item, proc-index) cooldown check → chance roll. The
cooldown is marked **only when the effect actually executed** (e.g. a
`lifesteal` proc that rolls on a 0-damage hit does not burn its
cooldown) — see `dispatchItemProcs`'s `executed` flag.

**`on_kill` fires once per player with damage attribution on the
kill**, not just the killing blow (`MobDeathItemProcs` iterates
`evt.PlayerDamage`). This is a deliberate party-friendly design
decision, not an oversight — it also resets every such player's
Blackrazor-style hunger anchor (`pinnacle_last_kill_round`).

### Per-effect `params`

- **`lifesteal`** — `{ratio: <fraction>}`. Heals the attacker
  `ratio * damage` (floored, minimum 1 if ratio*damage rounds to 0),
  clamped to `HealthMax` by `Character.Heal`.
- **`steal_pool`** — `{pool: 3, amount_pct: <fraction>}`. Only
  `pool: 3` (conviction) is wired; `1` (health) and `2` (stamina) are
  reserved but **unimplemented** (YAGNI until an item needs them —
  they silently no-op). `amount_pct` is a fraction of the **target's**
  pool max, capped by what the target actually has; drains the target
  and adds to the owner (clamped to the owner's max).
- **`aoe_stun`** — `{}` (no params consumed; `stun_rounds` is
  intentionally ignored — see below). Applies buff **84** (a fixed
  1-round stagger/stun) to every hostile mob in the owner's room.
  Non-combatants, `PlayerAttackImmune` mobs, and **any** charmed mob
  (not just the owner's own charm) are always skipped — sparing
  bystanders' companions, matching the `HarmArea` precedent. Mob
  owners (no `GetUserId()`) are a no-op — no Stage-2 mob wields one of
  these. `stun_rounds` is ignored by design: buff 84 is a fixed
  1-round stagger baked into its own YAML (`triggercount: 1`) and
  cannot be duration-scaled from proc params without hacking buff
  internals; tune an aoe_stun item's strength via `chance`/
  `cooldown_rounds` instead.
- **`apply_condition`** — `{condition: 1, duration: <rounds>,
  magnitude: <per-tick>}`. Only `condition: 1` (bleeding) is wired.
  `duration` defaults to 4 rounds, `magnitude` defaults to 2 per tick,
  when unset or < 1. Unknown condition ids no-op (cooldown not
  burned).

## 2. Pool reservations

```yaml
reserve_health_pct: 0.10       # [0, 1) — validated, panics outside range
reserve_stamina_pct: 0.0
reserve_conviction_pct: 0.0
```

While the item is equipped, `Character.GetPoolReservation(pool,
poolMax)` (`internal/characters/validate.go`) sums the reservation
from **every** equipped item and clamps the character's **current**
pool value down to `max - totalReservation` (the max itself is
untouched — this is a squeeze, not a debuff to the ceiling). A
Chrysalis-enchanted item's own reservation and its `reserve_*_pct`
field **stack** — both are summed, by design, even on the same item.
Consumed in `validate.go`'s post-equip pass, `NewRound_AutoHeal.go`
(regen targets the reserved-down max, not the full max), the `report`
command, and the player prompt.

## 3. Bandolier (`is_bandolier:` items)

```yaml
is_bandolier: true
bandolier_capacity: 6
preserves_contents: true     # contents never age while stored
ambient_potions: true        # slotted potions' buffs stay always-on
```

- **`preserves_contents`** freezes aging: each round,
  `tickPreserveContents` advances every slotted potion's
  `CraftedRound` by 1 in lockstep with the round counter, so the
  aging-elapsed calculation (`now - CraftedRound`) never grows.
- **`ambient_potions`** keeps every slotted potion's `BuffIds`
  continuously applied at **Peak potency (1.30x)**, `AddBuffScaled`,
  while the bandolier is worn and **attuned**. Drink-blocking of
  slotted potions is deferred to Stage 2 — Stage 1 does not stop you
  from drinking a slotted potion directly.
- **Attunement / content-fingerprint mechanism**: each tick builds a
  fingerprint of `beltItemId + sorted potion itemIds`
  (`bandolierFingerprint`). Any change — a slotted potion drunk,
  added, removed, or the belt itself swapped — flips the fingerprint,
  which immediately revokes all ambient buffs and stamps a cooldown
  running `Balance.BandolierAttuneRounds` (default 100) rounds into
  the future. Ambience only resumes once the fingerprint has been
  stable through that whole window. First-ever equip also counts as a
  fingerprint change (fp goes `"" → something`), so a freshly-equipped
  bandolier always pays the attunement cooldown once.

## 4. Mutation drip

```yaml
mutation_tick_interval: 50     # rounds between rolls; 0 = never
mutation_tick_chance: 10       # 1-100, required (validated) when interval > 0
mutation_rarity_floor: 5       # 0-10; minimum mutation rarity eligible, 0 = no floor
```

Every worn item (not slot-restricted like procs — the tick scans the
whole `GetAllWornItems()` set) with `mutation_tick_interval > 0` rolls
on interval-aligned rounds (`now % interval == 0`), then a percent
gate, then grants one mutation via
`Character.GrantRandomMutationRare(rarityFloor)`.

## 5. Hunger

```yaml
hunger_rounds: 100          # rounds without a kill before it starts feeding
hunger_drain_pct: 0.03      # [0, 1) fraction of HealthMax drained per hungry round
```

Gated on the **currently equipped weapon's** spec only
(`tickHunger` reads `c.Equipment.Weapon.GetSpec()`). The hunger clock
is **kill-anchored**: `pinnacle_hunger_anchor` starts at first tick
wielding the item, and any kill credit (`pinnacle_last_kill_round`,
stamped for every damage-attributed player on a kill, see §1) resets
it forward. Once overdue, drain escalates linearly from 1x up to a
hard cap of **3x** the base `hunger_drain_pct * HealthMax`, and
**never kills outright** — it floors the character at 1 HP. The drain
is applied directly to `Health`, **bypassing the damage hooks
entirely** (no sleep-wake, no aggro, no mitigation) — it's non-combat
attrition, not an attack. The feeding message is cooldown-gated
(reuses `Balance.SentientChatterCooldownRounds`) so an ignored hunger
debt doesn't spam a line every single overdue round. Swapping away
from a hunger weapon leaves its anchor stale but inert; re-wielding it
later re-bases the clock off the next kill/tick, with no separate
cleanup step.

## 6. Voices (`voice_id:`)

```yaml
voice_id: blackrazor
```

References `_datafiles/world/dogmud/itemvoices/<voiceid>.yaml`:

```yaml
voiceid: blackrazor
lines:
  on_equip: ["..."]
  on_unequip: ["..."]
  on_kill: ["..."]
  on_idle: ["..."]
  on_hunger_warning: ["..."]
  on_hunger_feeding: ["..."]
  on_taunt: ["..."]
  on_grudge: ["..."]
```

The `lines` map may use **exactly** these eight event keys
(`internal/itemvoices/itemvoices.go`'s `validVoiceEvents`) — any other
key panics at boot, and every key present must have a non-empty line
list. `itemvoices.LoadDataFiles()` also cross-validates every loaded
`ItemSpec.VoiceId` against the voice registry — a dangling `voice_id`
(referencing a voice file that doesn't exist) panics at boot, same as
`schedule_id`/`patrol_id`. `itemvoices.LoadDataFiles()` must run
**after** `items.LoadDataFiles()` for that cross-validation to see
every item. An empty `itemvoices/` directory is expected and correct
through the end of Stage 1 (`loadedCount=0` at boot is not an error).

**Pacing**: `pickVoiceEvent` picks `on_taunt` when the wearer is in
combat, `on_hunger_warning` when a hunger weapon has crossed 3/4 of its
hunger window, else `on_idle` — `on_equip`/`on_unequip`/`on_kill`/
`on_grudge` are authored slots for callers outside the tick (or future
wiring) to use via `emitVoiceLine` directly. Each eligible round rolls
once at `Balance.SentientChatterChancePct` (default 15) — only after
confirming a speakable line exists for the picked event, so quiet gear
never rolls. A successful line is paced by
`Balance.SentientChatterCooldownRounds` (default 20) between lines.
**Exactly one line per round, across all worn sentient items** — when
multiple voiced items are worn, the tick walks worn-slot order and the
first item with both a `voice_id` and a non-empty line pool for its
picked event wins the round; later sentient items never even roll
that round.

## 7. Recipes: `require_own_components`

```yaml
# on a RecipeSpec (internal/crafting)
require_own_components: true
```

When set, **every** ingredient that is itself a crafted component
(`ItemSpec.IsComponent == true`) must carry the assembling crafter's
`MakerName` — bulk (non-component) materials are exempt.
`CheckOwnComponents` is **strict-any-match**: if any tag-matching
component across the crafter's component bag + inventory is foreign,
the craft refuses, **even if the crafter also carries their own copy**
of that component — because `HasIngredients`/`ConsumeIngredients`
don't guarantee which matching item actually gets consumed, so the
engine can't safely assume the crafter's own copy would be the one
picked.

For this gate to be usable at all, component outputs must actually get
stamped: `ShouldStampMakerName` (skill 30+) now stamps `MakerName` on
a crafted output whenever `spec.Type != Object` **or**
`spec.IsComponent` — i.e., components stamp regardless of their
(conventionally `type: object`) declared Type.

## 8. Remort phial

Item id **40181** ("Phial of Second Birth") is **hardcoded** in
`internal/usercommands/drink.go` (`phialOfSecondBirthItemId`) — the
Stage 2 item YAML for this potion **must** use exactly this id, or the
special-case branch never fires. Drinking it:

1. `Character.ScourMutations(0)` — scours every acquired mutation back
   to species intrinsics, with **zero** reroll charges (contrast the
   unrelated Catalyst of Unmaking, which grants 3).
2. Immediately grants exactly one mutation via
   `GrantRandomMutationRare(5)` — rarity floor 5, hardcoded
   (`phialRarityFloor`).

## 9. Config knobs

| Knob | Location | Default | Purpose |
|------|----------|---------|---------|
| `GamePlay.PinnacleItemsEnabled` | config.gameplay.go | `true` | Master toggle for the whole per-round pinnacle tick (hunger/ambient/mutation-drip/voices); `pinnacleUserTick` early-returns entirely when off. |
| `GamePlay.ItemProcsEnabled` | config.gameplay.go | `true` | Kill switch for proc firing (`on_hit`/`on_block`/etc.); checked in `procGateOpen`. |
| `Balance.BandolierAttuneRounds` | config.balance.go | `100` | Re-attunement cooldown length after bandolier contents change. |
| `Balance.SentientChatterCooldownRounds` | config.balance.go | `20` | Minimum rounds between sentient item lines (also reused to pace the hunger-feeding message). |
| `Balance.SentientChatterChancePct` | config.balance.go | `15` | Percent chance per eligible round that a sentient item speaks. |

## 10. MiscData key registry (admin/debugging)

All keys live on the `Character`'s MiscData map and persist to player
YAML (numeric values round-trip through YAML as `int`/`int64`/
`float64` — readers tolerate all three).

| Key | Set by | Meaning |
|-----|--------|---------|
| `pinnacle_proc_cd_<itemId>_<procIdx>` | `markProcCooldown` | Round at which this item's Nth proc may fire again. |
| `pinnacle_last_kill_round` | `MobDeathItemProcs` | Last round this player got damage-attribution credit on a kill (drives hunger reset). |
| `pinnacle_hunger_anchor` | `tickHunger` | Round the current hunger weapon's clock is anchored to. |
| `pinnacle_hunger_msg_next_round` | `tickHunger` | Cooldown gate for the repeated feeding message. |
| `pinnacle_bandolier_attune_round` | `tickAmbientPotions` | Round at which ambient buffs may resume after a content change. |
| `pinnacle_bandolier_buffs` | `tickAmbientPotions` | The set of buff ids currently applied as ambience (so removal/rotation can be revoked cleanly). |
| `pinnacle_bandolier_fingerprint` | `tickAmbientPotions` | Last-seen `beltId:potionId,potionId,...` fingerprint, used to detect any content change. |
| `pinnacle_voice_next_round` | `tickVoices` | Cooldown gate between sentient item lines. |

Cooldown keys (`pinnacle_proc_cd_*` in particular) are **intentionally
never pruned** when an item is unequipped or lost — the key space is
bounded (per item id x proc index) and a stale cooldown is harmless
(it just means that exact item, if re-equipped, resumes mid-cooldown
rather than fresh). This is a deliberate simplicity trade-off, not an
oversight.

## Stage 2 shipped items

The nine legendary-BIS items that consume the Stage 1 primitives,
shipped on branch `feature/pinnacle-stage2-items`. All boot-verified
(`itemLoadedCount=386`, `itemvoices loadedCount=2`, buff 98 present,
`ValidateZoneConsistency errors=0 mode=panic`, 0 panics). Numbers are
starting values; combat/economy tuning is a later stage.

> **Folder gotcha — do NOT create a `pinnacle/` directory.**
> `ItemSpec.ItemFolder()` (`internal/items/itemspec.go`) buckets items
> **purely by ID range**: any `ItemId >= 40000` is loaded flat from
> `_datafiles/world/dogmud/items/materials-40000/` — no subtype
> subdirectory (unlike `armor-20000/<type>/`). Because these nine were
> allocated in the 40000 block, they live alongside crafting materials
> regardless of being weapons/armor/accessories. A file placed in any
> other directory (e.g. a hand-made `pinnacle-items/`) is never loaded.

| ID | Name | Slot / type | File (under `items/materials-40000/`) | Primitives used |
|----|------|-------------|----------------------------------------|-----------------|
| 40181 | Phial of Second Birth | consumable (potion) | `40181-phial_of_second_birth.yaml` | remort (`drink.go` hardcodes id 40181 → `ScourMutations` + rarity-floored grant) |
| 40182 | Vitalis Bandolier | belt | `40182-vitalis_bandolier.yaml` | `preserves_contents`, `ambient_potions`, `is_bandolier`/`bandolier_capacity` |
| 40183 | The Blackrazor | weapon (2H slashing) | `40183-the_blackrazor.yaml` | `reserve_health_pct`, `hunger_rounds`/`hunger_drain_pct`, `procs` (on_hit lifesteal), `voice_id: blackrazor` |
| 40184 | Wayfarer's Bottomless Pack | back | `40184-wayfarers_bottomless_pack.yaml` | `weight_reduction` (0.99) |
| 40185 | Aegis of Mockery | offhand (shield) | `40185-aegis_of_mockery.yaml` | `procs` (on_block aoe_stun), `taunt_pull`, `voice_id: aegis` |
| 40186 | Thornwall Harness | body | `40186-thornwall_harness.yaml` | `procs` (on_grapple apply_condition bleed) |
| 40187 | Seething Prism | neck | `40187-seething_prism.yaml` | `reserve_*_pct` (all three pools), `mutation_tick_interval`/`_chance`/`_rarity_floor` |
| 40188 | Zephyr Treads | feet | `40188-zephyr_treads.yaml` | `wornbuffids: [98]`, `staminamax` statmod |
| 40189 | Staff of the Hollow Choir | weapon (2H staff) | `40189-staff_of_the_hollow_choir.yaml` | `spell_damage_multiplier`, `procs` (on_spell_hit steal_pool), `casting`/`manifestation` statmods |

**Buff (worn):**

| ID | Name | File | Consumed by |
|----|------|------|-------------|
| 98 | Zephyr's Alacrity | `buffs/98-zephyrs_alacrity.yaml` | Zephyr Treads `wornbuffids` (permanent-haste-while-worn) |

**Sentient item voices** (`itemvoices/<voice_id>.yaml`):

| voice_id | File | Item | Character |
|----------|------|------|-----------|
| blackrazor | `itemvoices/blackrazor.yaml` | The Blackrazor (40183) | Ancient, vain, starving aristocrat |
| aegis | `itemvoices/aegis.yaml` | Aegis of Mockery (40185) | Period insult-comic |
