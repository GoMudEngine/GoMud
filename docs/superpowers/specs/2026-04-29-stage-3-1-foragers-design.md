# Forager NPCs — Design (Stage 3.1 of Caravan/Economy Effort)

**Date:** 2026-04-29
**Status:** Approved (brainstorming complete, ready for implementation plan)

## Goal

Add three forager NPCs — one per region — that gather raw materials in
their home territory, salvage corpses they encounter, and feed the
supply pipeline that 3.0b wired up. Two foragers (Marsh, Steppe) deliver
directly to their nearest town's vendors; the third (Fernway) hands off
to the caravan at a roadside meeting point, and the caravan distributes
Fernway mats to both towns symmetrically. Slow the caravan cadence so
foragers carry the day-to-day load and the caravan feels like a
delivery-day event. Standardize the existing hardcoded "high regen
room" mechanic into a reusable `sanctuary` room mutator so foragers
have explicit recall destinations and players gain a known safe rest
in Fernway South.

## Multi-stage context

This spec is **Stage 3.1** of the multi-stage caravan/economy effort.

1. ✅ **Stage 1 — NPC parties** — shipped 2026-04-27. Spec: `docs/superpowers/specs/2026-04-27-npc-party-system-design.md`.
2. ✅ **Stage 2 — Basic caravan** — shipped 2026-04-27. Spec: `docs/superpowers/specs/2026-04-27-caravan-system-design.md`.
3. ✅ **Stage 3.0a — Stillwater Marsh zone** — shipped 2026-04-28.
4. ✅ **Stage 3.0b — Mat region split** — shipped 2026-04-28. Spec: `docs/superpowers/specs/2026-04-28-mat-region-split-design.md`. Audit matrix: `docs/economy/mat-audit-matrix.md`.
5. ✅ **Stage 3.0c — Fernway South zone** — shipped 2026-04-28.
6. ✅ **Stage 3.0d — NPC fold-recall** — shipped 2026-04-28. Spec: `docs/superpowers/specs/2026-04-28-npc-fold-recall-design.md`.
7. ✅ **Stage 3.0e — Corpse salvage** — shipped 2026-04-28. Spec: `docs/superpowers/specs/2026-04-28-corpse-salvage-design.md`.
8. **Stage 3.1 — Forager NPCs** (THIS SPEC).
9. ⏳ **Stage 3.4 — Real item transfer** — caravan + foragers move actual items between inventories instead of triggering bucket-aware top-off.

Per user direction, nothing ships to prod (`master`) until the entire
economy stack (Stages 3.0b through 3.4) lands on `development`.

## Worldbuilding

The world has three foraging regions, each with a distinct material
identity (per the Stage 3.0b audit matrix):

| Region | Theme | Forager destination |
|---|---|---|
| **Stillwater** | Lake / marsh / fishing / fine craft | Stillwater town vendors directly |
| **Thornwall** | Chrysalis / enchanting / refined metals (in-shop crafted) + base/overlap mats from the steppes | Thornwall town vendors directly |
| **The Fernway** | Forest / herbal / wild-game / alchemy | Caravan picks up at a roadside meeting point and distributes to both towns |

**Why three foragers, not two:** Stillwater-unique mats are forager-fed
to Stillwater vendors, and Thornwall-unique mats are in-shop crafted at
Thornwall vendors. But base mats and mid-tier overlap mats are stocked
at vendors of both towns, and they need a supply pipeline too. The
Steppe forager (Ironwind Steppe is east of Thornwall, biome `land`)
fills the role of Thornwall's daily-supply pipeline for non-unique mats
— mirroring the Stillwater Marsh forager's role for Stillwater. This
gives Thornwall a forager-fed pipeline parallel to Stillwater's, instead
of relying entirely on in-shop crafting + caravan deliveries from
elsewhere.

**Why Fernway funnels through the caravan:** Fernway has no town
vendors of its own. The Fernway forager is the only producer of the
forest/herbal mat bucket, and both towns demand them (alchemy, cooking,
enchanting, blacksmithing). The forager walks UP from Fernway South to
the road and meets the caravan as it passes — once each direction per
caravan cycle. The caravan's pickup at Fernway is added as a brief
substate in each transit leg.

**Why foragers personally don't haul cargo through the bandit camp:**
Stage 1+2 established the bandit pack at North Road 4052 as the reason
caravans need armed escorts. Foragers are individually fragile (statpool
150-225), and the meeting point at North Road 4038 is on the
Thornwall-side of the bandit camp during inbound transit and the
Stillwater-side during outbound transit. The caravan is the only safe
party-strong way Fernway mats reach the towns; the forager only walks
the safe stretch from Fernway 4153 (Western Trailhead) to the meeting
point at 4038 and back.

## Architecture overview

**Three forager NPCs:**

| Forager | Territory | Sanctuary | Statpool | Supplies via |
|---|---|---|---|---|
| **Marsh Forager** (placeholder name: Vella) | Stillwater Marsh (rooms 4177–4196) | Stillwater Temple Interior (4123) | 150 | direct visits to 8 Stillwater vendors |
| **Steppe Forager** (placeholder name: Halix) | safe northern half of Ironwind Steppe (3000s) | Thornwall Temple Interior (468) | **225** | direct visits to 8 Thornwall vendors |
| **Fernway Forager** (placeholder name: Kessa) | Fernway South (rooms 4157–4176) | new "Forager's Camp" room (TBD location, attached off the main spine) | 150 | caravan handoff at North Road 4038 |

The Steppe forager's higher statpool reflects Ironwind being a more
dangerous zone than the marsh or southern fernway — wolf packs,
raptors, vipers, and goblins live there. The forager's territory is
restricted to the safer northern half (no goblins, no apex predators),
but stronger stats give an additional buffer.

**Existing systems leveraged:**

| System | Use |
|---|---|
| `internal/parties/` (Stage 1) | Not used — foragers are solo. |
| `internal/caravan/` (Stage 2) | Extended: route gains a `fernway_pickup` substate inside each transit leg. |
| `internal/hooks/spell_foldrecall.go` (Stage 3.0d) | Foragers cast `foldrecall` to escape combat or return home full. Each forager YAML sets `fold_anchor_room` to its sanctuary. |
| `internal/usercommands/skill.forage.go` | Core forage logic extracted to a shared function callable from both player and NPC paths. |
| Stage 3.0e corpse salvage | Foragers salvage corpses they encounter (their own kills, wildlife-on-wildlife kills, anyone's). |
| Existing `non_combatant: true` / `player_attack_immune: true` flags (Stage 2) | Foragers are `player_attack_immune: true` — same rebuff path as shopkeepers and caravan crew. They are NOT `non_combatant` because they engage prey wildlife. |
| Existing room mutator system | Standardized into a `sanctuary` mutator with a `regen_multiplier` field. |

**New systems:**

| Name | Purpose |
|---|---|
| `internal/forager/` package | Forager state machine, territory data, prey whitelist, vendor visit lists. |
| `internal/economy/` package | Item-id → bucket map derived from `docs/economy/mat-audit-matrix.md`. Backs `RestockBuckets`. |
| `RestockBuckets([]string)` shop method | Bucket-aware refill; only tops up entries whose item is in the given bucket list. |
| `forager_step` btree action + supporting conditions | Mirror of `caravan_step`. Reads `MobState["forager_state"]` and dispatches. |
| `sanctuary` room mutator | Standardizes the existing hardcoded `roomRegenMultiplier` switch. |

## Forager state machine

State stored in `MobState["forager_state"]` (mirroring the caravan's
`caravan_state` pattern), driven by a new `forager_step` btree action.

```
        ┌────────────────────────┐
        │  resting               │  ← spawn here
        │  (at sanctuary; ~120   │
        │   rounds OR until HP   │
        │   /SP/CP all full)     │
        └────────────┬───────────┘
                     ↓
        ┌────────────────────────┐
        │  traveling_to_territory│  pathto territory entry room
        └────────────┬───────────┘
                     ↓
        ┌────────────────────────┐
        │  foraging              │  wander territory; every
        │  (until inventory      │  6-10 rounds, run shared
        │   carry-cap ≥ 75%      │  forage core; if corpse
        │   OR fatigue timer ≥   │  in room, salvage it
        │   480 rounds)          │
        └────────────┬───────────┘
                     ↓ inventory full / fatigue / HP < 50%
        ┌────────────────────────┐
        │  traveling_to_dropoff  │
        │  (Marsh: → Stillwater  │
        │   Steppe: → Thornwall  │
        │   Fernway: → north_rd  │
        │   4038 meeting pt.)    │
        └────────────┬───────────┘
                     ↓
        ┌────────────────────────┐
        │  delivering            │  visit each vendor in
        │  (Marsh + Steppe: hit  │  hardcoded order; trigger
        │   each town vendor in  │  RestockBuckets() at each.
        │   sequence; Fernway:   │  Fernway: idle at 4038 up
        │   wait for caravan up  │  to N rounds; if caravan
        │   to wait_timeout)     │  arrives, set caravan flag,
        │                        │  exit.
        └────────────┬───────────┘
                     ↓
        ┌────────────────────────┐
        │  recalling             │  cast fold-recall (3-5 rd
        │                        │  cast time per Stage 3.0d)
        └────────────┬───────────┘
                     ↓ arrived at sanctuary
                (loops to resting)
```

**HP < 50% emergency branch:** any state can short-circuit to
`recalling` if HP drops below threshold. Cast time of fold-recall is the
survivability window — with 50% HP buffer + 150/225 statpool + cautious
wildlife targeting, this is a real save not a theoretical one.

**Healing salve auto-drink:** when HP < 75% and not yet recalling,
forager drinks a healing salve from their bandolier (3 salves total).
This adds a soft buffer between "took a hit" and "must recall."

**Visible behavior in `foraging` state:**

- Forager moves to a random adjacent territory room every 8-12 rounds.
- Every 6-10 rounds, fires the shared forage core (extracted from
  `skill.forage.go`). On success, an item is added to the forager's
  inventory AND a flavor message hits the room: *"Kessa stoops over a
  moss patch and tucks a handful into her satchel."* Forager-name +
  biome-flavored verb tables in the dialogue YAMLs.
- If a corpse is in the room (from wildlife killing wildlife, the
  forager's own kills, or anyone else's), forager runs the existing
  Stage 3.0e salvage flow, adding cloth/leather/sinew to inventory.
  Visible: *"Kessa kneels by the carcass and cuts strips of hide from
  it."*

**Visible behavior in `delivering` state (Marsh + Steppe):**

- Forager walks vendor-room to vendor-room in fixed order. At each,
  fires `RestockBuckets()` and emits a flavor message: *"Kessa lays a
  satchel of mats on the counter. Smith Brindle nods his thanks."*

**Visible behavior in `delivering` state (Fernway):**

- Forager idles at North Road 4038 (the meeting point). Fires
  occasional ambient lines: *"The forager scans the road, satchel at
  her feet."*
- When the caravan arrives at 4038 (caravan adds a brief stop here on
  transit), handoff fires. Both publish flavor lines: *"Kessa hands the
  satchel to Marta; the caravan rolls on."* Caravan acquires the
  `"fernway"` bucket flag in its `caravan_load` slice.
- If wait timeout expires without caravan arrival
  (`ForagerWaitTimeoutRounds`, default 150), forager logs the miss,
  recalls home with the satchel — the load doesn't get delivered this
  cycle.

**Initial spawn:** at sanctuary (Stillwater forager at 4123, Steppe at
468, Fernway at the new camp room). State defaults to `resting` on
first btree tick.

## Combat behavior + targeting

In `foraging` state, the forager's btree includes a **prey-only
engagement** sub-routine (new btree condition
`mob_can_safely_engage`):

```
mob_can_safely_engage:
  - opponent's effective stat-sum ≤ self_stat_sum × 0.6
  - opponent's species in PreyWhitelist (configurable per forager)
  - forager HP ≥ 75%
```

If all three pass, forager engages with their themed weapon. Otherwise,
ignore (continue wandering) — and if the opponent is hostile-attacking
the forager, immediately branch to `recalling` regardless of HP.

**Per-forager prey whitelists** (drawn from existing zone wildlife):

| Forager | Prey whitelist | Predators (always flee) |
|---|---|---|
| Marsh | marsh rat (367), dragonfly swarm (368) | river otter (366), snapping turtle (369), bog adder (370) |
| Steppe | steppe rat (200), dust crow (201), dust hare (213), ground squirrel (234), sage grouse (214), tumble beetle (231) | wolves, raptors, vipers, goblins, all southern-half wildlife |
| Fernway | wild hare (360), honey bees (362) | feral boar (363), timber wolf (364), forest badger (365), roe deer (361) |

**Themed weapons** (low-tier, statted to match):

| Forager | Weapon (new item) | Notes |
|---|---|---|
| Marsh | gaff hook (puncture, 1H) | Lake-fishing utility weapon |
| Steppe | hunting spear (puncture, 1H) | Plus a steppe-knife as a non-combat skinning utility |
| Fernway | hand axe (slashing, 1H) | Forest/woodcutter |

No ranged weapons — ranged isn't a real combat system in DOGMud yet,
and we don't introduce one in this stage.

**Bandolier of healing salves:** each forager carries a bandolier with
3x healing salve (item 30036). Auto-consumed when HP < 75% and not yet
recalling.

## Fold-recall integration

Stage 3.0d's `internal/hooks/spell_foldrecall.go` already supports NPC
casting. Each forager's mob YAML sets `fold_anchor_room` to its
sanctuary:

| Forager | Anchor room |
|---|---|
| Marsh | Stillwater Temple Interior (4123) |
| Steppe | Thornwall Temple Interior (468) |
| Fernway | new Forager's Camp room |

The forager's btree casts `foldrecall` when `forager_state == recalling`
is set (either by the normal end-of-delivery flow or by the
HP-emergency short-circuit). Cast time is the standard 3-5 rounds. NPC
cast is interruptible by death but not by damage (per Stage 3.0d's
spec).

## Player-attack rebuff

Foragers set `player_attack_immune: true` (the Stage 2 caravan flag).
This blocks `attack`, `bash`, `kick`, `grapple`, `taunt`, `trip`,
`shoot`, `throw`, and `steal` against them, with the standard
shopkeeper-style rebuff message ("You can't attack <name>."). Foragers
remain mob-attackable (wildlife can still threaten them — that's the
intended danger), and they engage prey wildlife normally
(player_attack_immune does NOT block their own attacks).

## RestockBuckets — the bucket-aware shop method

New method on `internal/shops/shopinventory.go`:

```go
// RestockBuckets is like Restock(), but only refills slots whose
// item_id is in the given buckets. Used by foragers (always one
// bucket per call) and the caravan (multiple buckets per call,
// based on what it last picked up).
func (si *ShopInventory) RestockBuckets(buckets []string) bool {
    for _, entry := range si.Entries {
        bucket := economy.BucketFor(entry.ItemId)
        if !slices.Contains(buckets, bucket) { continue }
        // existing top-up logic
    }
    return refilledAny
}
```

Backing data: `internal/economy/buckets.go` — a Go-side map derived
from `docs/economy/mat-audit-matrix.md`:

```go
package economy

var ItemBucket = map[int]string{
    40001: "base", 40003: "base", 40006: "base", // ... 13 base
    40046: "fernway", 40049: "fernway", 40062: "fernway", // ... 8 fernway
    40051: "stillwater", 40053: "stillwater",  // ... 6 stillwater
    40010: "thornwall", 40018: "thornwall",    // ... 13 thornwall
    40004: "overlap", 40005: "overlap",        // ... 11 overlap
}
```

A unit test asserts every classified mat in the audit matrix has a
bucket entry, and every bucket entry has a corresponding ItemSpec. The
audit matrix doc gets a "kept in sync with `internal/economy/buckets.go`"
marker comment.

### Forager → bucket mapping

| Forager | Buckets they fill |
|---|---|
| Marsh | `["stillwater", "base", "overlap"]` |
| Steppe | `["base", "overlap"]` (Thornwall-unique stays in-shop crafted by Vael/Kerra/Tess) |
| Fernway | `["fernway"]` (caravan delivers to both towns) |
| Caravan northbound (Stillwater → Thornwall) | `["stillwater", "fernway"]` if Fernway pickup happened this leg, else `["stillwater"]` |
| Caravan southbound (Thornwall → Stillwater) | `["thornwall", "fernway"]` if Fernway pickup happened, else `["thornwall"]` |

In-shop crafters (Vael 105, Kerra 97, Tess 108) keep their existing
self-restock behavior — they still produce Thornwall-unique mats
locally; that pipeline is untouched.

## Caravan route extension

Current Stage 2 route states:

```
thornwall_dwell → outbound_transit → stillwater_route → stillwater_dwell
   → inbound_transit → thornwall_route → (loop)
```

**Stage 3.1 inserts a `fernway_pickup` substate into both transit
legs.** The transit leg is no longer a single pathto; it's a two-hop
pathto:

```
outbound_transit:
  ├─ pathto north_road 4038 (Fernway meeting point)
  ├─ dwell at 4038 for FernwayPickupDwellRounds (default 6 rounds)
  │  └─ on entry: if Fernway forager is in this room, fire handoff:
  │       caravan picks up "fernway" bucket flag; emit handoff messages
  │       to the room
  ├─ pathto Stillwater depot
  └─ → stillwater_route

inbound_transit:
  ├─ pathto north_road 4038
  ├─ same dwell + handoff check
  ├─ pathto Thornwall depot
  └─ → thornwall_route
```

Caravan's MobState gains a `caravan_load` field (string slice). Set to
`["stillwater"]` after the stillwater_route, `["thornwall"]` after the
thornwall_route. If Fernway handoff fires on the next transit leg,
append `"fernway"`. Cleared at the next vendor route's start, after
each vendor visit.

`VisitVendorsInRoom()` is updated to call `RestockBuckets(caravan_load)`
instead of the bucket-blind `Restock()`.

### Cadence retune

| Knob | Stage 2 | Stage 3.1 | Effect |
|---|---|---|---|
| `CaravanDepotDwellRounds` | 360 | 720 | doubles depot dwells |
| `FernwayPickupDwellRounds` | — | 6 | new |
| `ForagerForageDwellRounds` | — | 8 | new (rounds between forage attempts) |
| `ForagerCarryThresholdPct` | — | 75 | new (head-home trigger) |
| `ForagerHPRecallThresholdPct` | — | 50 | new (emergency recall trigger) |
| `ForagerHealPotionThresholdPct` | — | 75 | new (drink salve trigger) |
| `ForagerWaitTimeoutRounds` | — | 150 | new (Fernway-only; how long forager waits at 4038) |

Total caravan cycle becomes ~1620 rounds = ~2 game days. Forager-fed
local pipelines refresh on each forager cycle (~300-400 rounds), making
them clearly the day-to-day reliable supply; the caravan becomes a
delivery-day event.

## Sanctuary mutator standardization

Replace the hardcoded switch in `internal/hooks/NewRound_AutoHeal.go:375`
with a room-mutator-driven lookup.

**New mutator** `_datafiles/world/dogmud/mutators/sanctuary.yaml`:

```yaml
mutatorid: sanctuary
regenmultiplier: 5.0
descriptionmodifier:
  behavior: append
  text: A peace that is older than the stones themselves settles
        on you here — wounds close more easily, breath comes more
        deeply.
  colorpattern: pearl
```

Mutator schema gains `regen_multiplier float64` (defaults 1.0). The
auto-heal hook reads any sanctuary-class mutator on the room and
multiplies regen accordingly.

**`roomRegenMultiplier(room *rooms.Room)` becomes:**

```go
func roomRegenMultiplier(room *rooms.Room) float64 {
    mult := 1.0
    for _, m := range room.GetMutatorsResolved() {
        if m.RegenMultiplier > 0 {
            mult *= m.RegenMultiplier   // multiplicative if multiple
        }
    }
    return mult
}
```

**Rooms gaining the `sanctuary` mutator:**

| Room | Currently | After 3.1 |
|---|---|---|
| Thornwall Temple Interior (468) | hardcoded 5.0× | `sanctuary` mutator |
| Sanctum Basin tutorial zone (rooms 101–120) | hardcoded 5.0× | `sanctuary` mutator on each (or zone-config-level if supported) |
| Stillwater Temple Interior (4123) | none | `sanctuary` 5.0× |
| Fernway Forager's Camp (new room) | none | `sanctuary` 5.0× |

**Testing arena entrance (200):** out of scope per user — testing arena
is unused code. The hardcoded 10× line is dropped along with the rest
of the switch. If anyone revives the arena, they add the mutator
explicitly.

The Old Chapel Ruin (4144) does NOT get the mutator — it's a ruin, not
a functioning temple; flavor doesn't fit.

## Hint update

`_datafiles/world/dogmud/hints.yaml:107-110` currently reads:

> Healing up slow? Try hanging out in the Sanctum Basin or the Temple
> in Thornwall. Some rooms are great places to rest.

Updated to:

> Healing up slow? Some rooms are sanctuaries — temples, certain camps,
> the Sanctum Basin tutorial — and regenerate health, stamina, and
> conviction much faster than ordinary rooms. Look for a peaceful
> description.

A broader hints audit (general cleanup pass — many hints have drifted
out of sync with current systems) is logged separately as a
low-priority followup; not done in 3.1.

## The new Fernway Forager's Camp room

A new room added to `the_fernway_south/` zone, in a plausible spot off
the main spine. From the existing room layout, room 4163 (central spine)
or 4172 (southern terminus area) would be natural anchors. Plan task 0
surveys exits and picks the actual room id + parent room. The camp gets:

- Title: "Forager's Camp"
- Description: a small clearing with a lean-to, a banked firepit, drying
  racks for hides
- `sanctuary` mutator
- Spawninfo: the Fernway forager mob

## Edge cases

| Scenario | Behavior |
|---|---|
| Forager killed by wildlife mid-territory | Standard mob-death handling: corpse drops carried inventory. Respawn at sanctuary after standard mob respawn timer (~5-10 min real). State defaults to `resting`. The carried mats are simply lost from the supply pipeline — temporary scarcity hit. With 150/225 statpool + 50% recall + healing-salve buffer + cautious targeting, this is a rare event. |
| Forager dies during fold-recall cast | Same as above. Cast is cancelled implicitly. |
| Player follows forager into vendor cluster | Forager ignores the player. Standard mob behavior. Player can watch the deliveries. |
| Player tries `attack` / `steal` etc. on a forager | `player_attack_immune` rebuff message ("You can't attack <name>."). No aggro. |
| Player lures wildlife into a forager | Forager's prey-only engagement + 50% HP recall + healing salve buffer make this very unlikely to succeed. If it does, forager dies and corpse drops inventory; player gets some mats. Single-cycle scarcity hit. Not a sustainable grief vector since direct attacks are blocked and indirect kills require setup AND wildlife AND timing AND the forager not recalling first. |
| Caravan wiped by bandits | Stage 2's existing recovery: caravan respawns at Thornwall depot, state resets, cycle resumes. No Fernway pickup until next cycle. Fernway forager either waits at 4038 until timeout (150 rounds) and recalls home with satchel, or — if already returned home — the next cycle picks up where it left off. |
| Fernway forager arrives at 4038, caravan never comes | After `ForagerWaitTimeoutRounds`, forager recalls home with the satchel intact. Mats get a second chance next cycle. Eventually if the caravan stays dead, the forager's inventory hits carry-cap and additional foraging is paused; this is the intended "supply drying up" pressure. |
| Two foragers spawn into the same sanctuary at startup | Each forager starts at its own sanctuary; no collision in practice. |
| Forager in `delivering` state when the server restarts | Respawn at sanctuary. State resets to `resting`. Carried mats lost. Brief pipeline interruption — same as caravan restart. |
| Tutorial player enters Sanctum Basin (101–120) | Sanctuary mutator gives them 5× regen, same as today's hardcoded behavior. No regression. |
| Multiple sanctuary mutators on the same room (future-proofing) | Stack multiplicatively per `RegenMultiplier > 0` field. No room currently has more than one. |
| Player enters the new Fernway Forager's Camp | Sanctuary mutator gives them 5× regen too — positive externality. Camp becomes a known safe rest stop in Fernway South. |

## Testing strategy

### Unit tests

**`internal/forager/`** (new package):
- `state_test.go` — every state transition returns the right next state given preconditions (carry-cap reached, HP threshold, fatigue timer, caravan-arrival flag)
- `prey_whitelist_test.go` — `mob_can_safely_engage` returns the expected verdict for the matrix of (forager, opponent species, opponent stats, forager HP)
- `vendor_route_test.go` — vendor visit order is stable, no duplicates, all referenced mob IDs exist on disk

**`internal/economy/`** (new package):
- `buckets_test.go` — every audit-matrix item has a bucket entry; every bucket entry resolves to a valid ItemSpec; no orphans either way

**`internal/shops/shopinventory_test.go`** (extended):
- `RestockBuckets(["stillwater"])` only refills Stillwater-bucket slots
- Empty bucket list = no-op
- Multiple buckets = union behavior
- Unknown bucket = silently skipped

**`internal/behaviortree/actions_forager_test.go`** (new):
- `forager_step` reads/writes `MobState["forager_state"]` correctly per state
- HP-threshold short-circuit fires regardless of current state
- Inventory threshold computed correctly against carry capacity

**`internal/behaviortree/actions_caravan_test.go`** (extended):
- New `fernway_pickup` substate fires handoff when forager present, no-ops cleanly when forager absent
- `caravan_load` is appended/cleared correctly across the state cycle

**`internal/hooks/NewRound_AutoHeal_test.go`** (extended):
- Sanctuary mutator on a room yields the expected multiplier
- Multiple sanctuary-class mutators stack multiplicatively
- Existing room 468 / 101–120 behavior preserved through the migration

### In-game smoke test

1. Boot. Foragers each at their sanctuary in `resting`.
2. Wait. Marsh forager wakes up, walks to Stillwater Marsh, begins foraging. Watch the flavor messages.
3. Watch a forager engage prey (a marsh rat in the same room). Forager wins; corpse drops; forager salvages it; loot shows in their inventory.
4. Watch the satchel fill. Once carry-cap ≥ 75%, forager exits territory toward Stillwater town.
5. Forager arrives at Smith Brindle. Flavor message fires. `inventory smithbrindle` shows previously-empty Stillwater-bucket slots are now filled. Iterate through 8 vendors.
6. After last vendor, forager casts foldrecall. Arrives at Stillwater Temple Interior (4123). Verify the room's display modifier shows the sanctuary append-text. Verify HP/SP/CP regen 5×.
7. Repeat 1-6 for Steppe forager (Thornwall vendors) and Fernway forager (waits at 4038).
8. **Caravan-Fernway handoff:** wait for the caravan to enter outbound_transit. Watch it stop at 4038 with the Fernway forager present. Handoff messages fire; caravan acquires `caravan_load: ["fernway"]`. At Stillwater vendors, Fernway-bucket slots fill. On inbound, repeat for Thornwall.
9. **Caravan-Fernway miss:** wait until Fernway forager hasn't yet reached 4038. Caravan passes through 4038, finds no forager, fires no handoff, continues. Stillwater vendor visit fills only Stillwater bucket. Fernway slots stay empty until next cycle.
10. **Forager-attack rebuff:** `attack <forager_name>` → rebuff message for all three foragers.
11. **Forager-emergency recall:** drag a predator into a forager's room. Forager HP drops below 50%, casts foldrecall, ends up at sanctuary alive.
12. **Sanctuary regen verification:** stand a low-HP player in the Stillwater Temple. Watch their HP regen rate match the Thornwall temple's behavior.

## Files affected

**New Go packages / files:**

| Action | File | Purpose |
|---|---|---|
| CREATE | `internal/forager/state.go` | State enum + transitions |
| CREATE | `internal/forager/territory.go` | Per-forager territory + prey whitelist data |
| CREATE | `internal/forager/state_test.go` | |
| CREATE | `internal/forager/territory_test.go` | |
| CREATE | `internal/economy/buckets.go` | Item-id → bucket map |
| CREATE | `internal/economy/buckets_test.go` | Audit-matrix invariants |
| CREATE | `internal/behaviortree/actions_forager.go` | `forager_step` btree action |
| CREATE | `internal/behaviortree/conditions_forager.go` | `mob_can_safely_engage`, `mob_inventory_at_threshold`, `mob_hp_below_recall_threshold` |
| CREATE | `internal/behaviortree/actions_forager_test.go` | |

**Existing Go files modified:**

| Action | File | Purpose |
|---|---|---|
| MODIFY | `internal/usercommands/skill.forage.go` | Extract core into a function reusable by NPCs |
| MODIFY | `internal/shops/shopinventory.go` | Add `RestockBuckets([]string)` |
| MODIFY | `internal/caravan/state.go` | Add `fernway_pickup` substate to both transit legs |
| MODIFY | `internal/caravan/visit.go` | Switch to `RestockBuckets(caravan_load)` |
| MODIFY | `internal/behaviortree/actions_caravan.go` | Wire fernway_pickup state |
| MODIFY | `internal/hooks/NewRound_AutoHeal.go` | Replace hardcoded `roomRegenMultiplier` switch with mutator-driven lookup |
| MODIFY | `internal/rooms/mutators.go` (or wherever Mutator schema lives) | Add `RegenMultiplier float64` field |
| MODIFY | `internal/configs/config.balance.go` | Add 6 new forager config knobs + bump `CaravanDepotDwellRounds` default 360→720 |
| MODIFY | `_datafiles/config.yaml` | Defaults for new knobs |

**New data files:**

| Action | File | Purpose |
|---|---|---|
| CREATE | `_datafiles/world/dogmud/mutators/sanctuary.yaml` | Sanctuary regen mutator |
| CREATE | `_datafiles/world/dogmud/mobs/stillwater_marsh/371-marsh_forager.yaml` | Marsh forager (Vella) |
| CREATE | `_datafiles/world/dogmud/behaviors/stillwater_marsh/371-marsh_forager.yaml` | Marsh forager btree |
| CREATE | `_datafiles/world/dogmud/dialogue/stillwater_marsh/371-marsh_forager.yaml` | Marsh forager dialogue + flavor |
| CREATE | `_datafiles/world/dogmud/mobs/the_fernway_south/372-fernway_forager.yaml` | Fernway forager (Kessa) |
| CREATE | `_datafiles/world/dogmud/behaviors/the_fernway_south/372-fernway_forager.yaml` | |
| CREATE | `_datafiles/world/dogmud/dialogue/the_fernway_south/372-fernway_forager.yaml` | |
| CREATE | `_datafiles/world/dogmud/mobs/ironwind_steppe/243-steppe_forager.yaml` | Steppe forager (Halix) |
| CREATE | `_datafiles/world/dogmud/behaviors/ironwind_steppe/243-steppe_forager.yaml` | |
| CREATE | `_datafiles/world/dogmud/dialogue/ironwind_steppe/243-steppe_forager.yaml` | |
| CREATE | `_datafiles/world/dogmud/items/weapons-10000/(N)-marsh_gaff_hook.yaml` | Marsh weapon |
| CREATE | `_datafiles/world/dogmud/items/weapons-10000/(N)-steppe_hunting_spear.yaml` | Steppe weapon |
| CREATE | `_datafiles/world/dogmud/items/weapons-10000/(N)-fernway_handaxe.yaml` | Fernway weapon |
| CREATE | `_datafiles/world/dogmud/rooms/the_fernway_south/(N).yaml` | Forager's Camp room (sanctuary mutator + forager spawn) |

**Existing data files modified:**

| Action | File | Purpose |
|---|---|---|
| MODIFY | `_datafiles/world/dogmud/rooms/thornwall_city/468.yaml` | Add `sanctuary` mutator |
| MODIFY | `_datafiles/world/dogmud/rooms/sanctum_basin/101.yaml` … `120.yaml` | Add `sanctuary` mutator (or zone-config-level if supported) |
| MODIFY | `_datafiles/world/dogmud/rooms/stillwater/4123.yaml` | Add `sanctuary` mutator |
| MODIFY | a chosen Fernway South room's exits | Add the new Forager's Camp as an off-spine adjacent |
| MODIFY | `_datafiles/world/dogmud/hints.yaml` | Generalize the temple-regen hint |
| MODIFY | `docs/economy/mat-audit-matrix.md` | Add "kept in sync with `internal/economy/buckets.go`" marker |
| MODIFY | `docs/schemas/mob.md` | Document `fold_anchor_room` is now used by foragers |
| MODIFY | `docs/schemas/room.md` (or mutator.md) | Document `regen_multiplier` mutator field |
| MODIFY | `docs/schemas/behavior.md` | Document `forager_step`, new conditions |
| MODIFY | `PATCH_NOTES.md` | Stage 3.1 dev-only entry |

## Out of scope (explicitly)

- **Stage 3.4 real item transfer.** Foragers use real-items-in-satchel, but vendor delivery is still a `RestockBuckets()` call, not actual item transfer. Caravan-to-vendor delivery is also still bucket-flag-driven. Stage 3.4 will retrofit both to physical item transfer; the forager's existing inventory makes that retrofit minor.
- **Forager-related quests.** The user has these in mind for a 3.5+ pass. Stage 3.1 ships purely the supply-pipeline NPCs.
- **General hints-audit cleanup.** Logged separately as a low-priority followup.
- **Forager dialogue beyond light flavor.** Each forager gets ~6-10 hint patterns and a couple of root nodes (greeting, weather, quest-hookable "what brings you here"). No full dialogue trees, no flag-gated branches.
- **Bandit detune at 4052.** Already handled in Stage 2; left as-is.
- **Cross-zone routine clock standardization.** The caravan + foragers run on independent btree-tick cycles; no global synchronization clock. Loose timing is intentional.
- **Multiple foragers per territory** (apprentices, helper NPCs).
- **Forager-driven shop pricing** (negotiated rates, regional discounts). Pricing keeps existing scarcity-multiplier behavior — no manual edits.
- **Steppe forager southern-half exposure.** Forager's territory is the safe northern half of Ironwind Steppe; the goblin-haunted south is for players, not foragers.
- **Old Chapel Ruin (4144)** as a sanctuary. It's a ruin — flavor doesn't fit; skip.
- **Testing arena migration.** Arena is unused; the existing hardcoded 10× line is dropped without replacement.
- **Ranged weapons** for any forager. No ranged combat system in DOGMud yet; not introduced here.

## Open implementation questions (for the plan stage)

- Exact Fernway Forager's Camp parent room (4163 or 4172 or other; survey + pick during plan task 0).
- Exact mob IDs (next free in each zone's range — likely 371 for stillwater_marsh, 372 for the_fernway_south, 243 for ironwind_steppe — verified against filesystem at plan time).
- Forager names + visual descriptions (3 NPCs; placeholder names Vella / Halix / Kessa; user can rename in plan review).
- Steppe forager prey whitelist final tuning. Initial list above; some entries may be aggressive enough to actually threaten the forager — quick stat audit during plan.
- Exact territory beat for the Steppe forager (northern half — which rooms specifically). Plan task 0 surveys.
- Sanctum Basin tutorial zone: per-room mutator vs. zone-config-level mutator. Depends on whether zone-config supports mutators today (probably yes — verify).
- Vendor visit order: Marsh forager visits 8 Stillwater vendors; final order likely follows physical adjacency in the town to feel like a milk route. Plan task picks order.
- Item IDs for the 3 new forager weapons (next free range in `items/weapons-10000/` — survey at plan time).

## Verification

**Phase 1 — unit + boot:**
- `go test ./internal/forager/... ./internal/economy/... ./internal/shops/... ./internal/caravan/... ./internal/behaviortree/... ./internal/hooks/...`
- `go test ./...` overall (no regressions)
- Server boot clean (per Pre-Push SOP)

**Phase 2 — bucket invariants:**
- `internal/economy/buckets_test.go` asserts every audit-matrix-classified item has a bucket entry, and vice-versa. Catches drift between docs and code.

**Phase 3 — in-game smoke** (12-step sequence above)

**Phase 4 — backward compat:**
- Thornwall Temple regen behavior preserved (5×, players + companions both)
- Sanctum Basin tutorial regen preserved (5×)
- Stage 2 caravan still does its thing on non-Fernway-pickup ticks
- Existing player `forage` command unchanged
- Non-served zones (Sanctum Basin, Watchers Crossing, etc.) unchanged
- Existing player-party features unchanged
- Existing shopkeeper attack-rebuff unchanged
