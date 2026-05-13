# Real Item Transfer — Design (Stage 3.4 of Caravan/Economy Effort)

**Date:** 2026-04-30
**Status:** Approved (brainstorming complete, ready for implementation plan)

## Goal

Replace the bucket-flag-driven `RestockBuckets` mechanism with real item
transfer between forager satchels, the caravan wagon, and vendor
inventories. Add an actual wagon mob (and a draft horse pair) to the
caravan party — `look wagon` shows the cargo, the wagon dies if the
caravan is wiped, and bandits loot the cargo when they win the brawl.
Decouple vendor stock caps from per-vendor configuration via a
rarity-tier system on items + a stock-multiplier on shopkeepers. Pay
off the "wholesalers seeking arbitrage between regions" worldbuilding
that the Stage 2 caravan spec gestured at.

## Multi-stage context

This spec is **Stage 3.4** of the multi-stage caravan/economy effort.

1. ✅ **Stage 1 — NPC parties** — shipped 2026-04-27.
2. ✅ **Stage 2 — Basic caravan** — shipped 2026-04-27.
3. ✅ **Stage 3.0a — Stillwater Marsh zone** — shipped 2026-04-28.
4. ✅ **Stage 3.0b — Mat region split** — shipped 2026-04-28.
5. ✅ **Stage 3.0c — Fernway South zone** — shipped 2026-04-28.
6. ✅ **Stage 3.0d — NPC fold-recall** — shipped 2026-04-28.
7. ✅ **Stage 3.0e — Corpse salvage** — shipped 2026-04-28.
8. ✅ **Stage 3.1 — Forager NPCs** — shipped 2026-04-30.
9. **Stage 3.4 — Real item transfer** (THIS SPEC).

Per user direction: nothing ships to prod (`master`) until the entire
economy stack (Stages 3.0b through 3.4) lands on `development`.
Stage 3.4 is the last stage; once it merges to development, the full
stack promotes to master as a coherent economy update.

## Worldbuilding

The caravan crew are wholesalers seeking arbitrage between regions.
Stage 2 established the narrative — Stage 3.4 makes it real. The
caravan literally:

1. **Picks up surplus** at each vendor it stops at (the goods that
   region produces in abundance — Stillwater pickup gets lake-iron,
   marsh willow, freshwater clam from Stillwater vendors)
2. **Hauls it across the road** to the other town
3. **Delivers it** to vendors there who stock matching items but
   produce them locally less abundantly

A wagon (with a draft horse team to pull it) provides the visible
cargo. Players see the wagon, can `look wagon` to see what's currently
loaded, and if the bandit pack at North Road 4052 wipes the caravan,
the bandits actually loot the cargo — making the bandit kills meaningful
in a way they weren't before.

The flow is symmetric per cycle: outbound delivers (Thornwall + Fernway
goods to Stillwater vendors) AND picks up (Stillwater goods for return);
inbound delivers (Stillwater + Fernway goods to Thornwall vendors) AND
picks up (Thornwall goods for next outbound). Each vendor stop is a
two-pass interaction.

## Architecture overview

**New components:**

| Layer | What | Why |
|---|---|---|
| Engine — `ItemSpec` | `RarityTier int` field (50/40/30/20/10) | Decouples MaxStock from per-vendor YAML; centralizes rarity. |
| Engine — `Mob` | `StockMultiplier float64` field (default 1.0) | Per-shop scale for tier-derived MaxStock. Future big-city shops set > 1.0. |
| Engine — `Mob` | `CarryCapacityOverride float64` field | Wagon needs ~5000; default Strength-derived calc gives ~6. |
| Engine — `Mob` | `HealthMaxOverride int` field | Wagon needs ~1500 to survive bandit raids. |
| Engine — `Mob` | `StaminaMaxOverride int` field | Wagon needs effectively-infinite SP so it never blocks movement. |
| Engine — `Mob` | `CorpseName string` field | Wagon corpse renders as "splintered wagon wreckage", not "<name> corpse". |
| Engine — `Mob` | `CorpseDescription string` field | Wreckage description differs from live wagon description. |
| Engine — `shops` | `EffectiveMaxStock(itemId, mob) int` helper | RarityTier × StockMultiplier; falls back to per-entry override if explicitly set. |
| Engine — `caravan` | `VisitVendorsInRoom(roomId, wagon, deliveryBuckets, pickupBuckets)` rewrite | Replace bucket-flag restock with real bidirectional item transfer. |
| Engine — `forager` | `tickForagerDeliveringTown` rewrite | Forager → vendor real item transfer. |
| Engine — `forager` | `tickForagerResting` extension | Stay-home-if-satchel-full rule (steady-state stability). |
| Btree action | `distribute_cargo_to_hostiles` | Wagon `mob_death` handler; round-robin distributes items to hostile mobs in room. |

**Existing systems leveraged unchanged:**

- Stage 1 `internal/parties` — `party_follow_leader` handles wagon + horse movement; `party_ensure_npc_party` adds the new members
- Stage 2 caravan state machine — outbound/inbound/dwell phases unchanged
- Stage 3.0d NPC fold-recall — caravan crew still uses it for emergency escape
- Stage 3.0e corpse salvage — still works on horse corpses (`groups: [animal]`)
- Stage 3.1 forager state machine — extended (rest extension), other paths unchanged
- `Character.StoreItem` / `RemoveItem` — used for all transfers

**New content:**

| Action | File | Purpose |
|---|---|---|
| CREATE | `_datafiles/world/dogmud/mobs/thornwall_city/374-caravan_wagon.yaml` | Wagon mob |
| CREATE | `_datafiles/world/dogmud/behaviors/thornwall_city/374-caravan_wagon.yaml` | Wagon btree (passive follower + death handler) |
| CREATE | `_datafiles/world/dogmud/mobs/thornwall_city/375-pell.yaml` | Draft horse 1 (dappled grey) |
| CREATE | `_datafiles/world/dogmud/behaviors/thornwall_city/375-hob.yaml` | Hob btree |
| CREATE | `_datafiles/world/dogmud/mobs/thornwall_city/376-brick.yaml` | Draft horse 2 (bay) |
| CREATE | `_datafiles/world/dogmud/behaviors/thornwall_city/376-bran.yaml` | Bran btree |
| MODIFY | `_datafiles/world/dogmud/mobs/thornwall_city/357-ketil.yaml` | Update party leader to expect 5 followers (Marta, Lars, wagon, Hob, Bran) |
| MODIFY | `_datafiles/world/dogmud/behaviors/thornwall_city/357.yaml` | `party_ensure_npc_party` member list expanded |
| MODIFY | `_datafiles/world/dogmud/rooms/thornwall_city/465.yaml` | spawninfo gets wagon + 2 horses added |
| MODIFY | All ~50 mat YAMLs in `items/materials-40000/` | Set `rarity_tier` per the agreed mapping |
| MODIFY | 17 caravan-served vendor mob YAMLs | Set `stock_multiplier: 1.0` (or omit — same default); drop explicit `max_stock` from each StockEntry |
| MODIFY | `_datafiles/world/dogmud/mobs/sanctum_basin/69-aberrant_chrysalis.yaml` | Remove Chrysalis Core (40010) from drops |
| MODIFY | `_datafiles/world/dogmud/mobs/ironwind_steppe/228-stone_beetle_queen.yaml` | Add Chrysalis Core 10% drop |
| MODIFY | `_datafiles/world/dogmud/mobs/ironwind_steppe/229-windscour_wyrm.yaml` | Add Chrysalis Core 5% drop |

## Wagon mob

**ID:** 374 (next free above the Stage 3.1 foragers, verified globally unique).

**YAML shape:**

```yaml
mobid: 374
zone: Thornwall City
archetype: ""              # not a fighter, not a caster — passive object
statpool: 100              # mostly cosmetic; pools are overridden below
itemdropchance: 100        # cargo always drops on death (not random)
groups:
  - caravan
  - merchant_train
hostile: false
non_combatant: false       # MOB-attackable (bandits damage it) but...
player_attack_immune: true # ...players can't attack
maxwander: -1
activitylevel: 0           # no idle commands
charm_immune: true

# Engine override fields (new in Stage 3.4)
carry_capacity: 5000       # default ~6 from Strength × 0.65
health_max: 1500           # ~5x crew HP — survives bandit raids; dies on a real wipe
stamina_max: 9999          # effectively infinite; never a movement bottleneck
corpse_name: splintered wagon wreckage
corpse_description: |
  Shattered timbers and twisted iron bands lie heaped where
  the supply wagon once stood, the canvas roof torn loose
  and trampled into the dirt. The driver's bench has split
  clean in two; the iron lantern is bent beyond use.
  Scattered among the wreckage, broken crates and split
  sacks lie half-emptied — though the bandits did their
  best to take what they could carry.

character:
  name: a sturdy oak-and-iron supply wagon
  description: |
    A hardwood freight wagon, broad-bedded and shoulder-high,
    its frame banded with cold-forged iron. The bed is roofed
    in tarred canvas stretched tight over hoop-frames, the
    canvas weatherbeaten and patched in three places along
    the seams. A pair of leather-padded yokes at the front
    rig the wagon to its draft team. The wagon's right rear
    wheel has been reset twice — the new spokes are paler
    than the rest. A small iron lantern hangs at the
    driver's bench, unlit by day.
  speciesid: 1
  level: 1
  gold: 0
  stats:
    vitality: {training: 60}
    strength: {training: 10}
    dexterity: {training: 10}
    perception: {training: 5}
    willpower: {training: 5}
    charisma: {training: 10}
```

**Wagon behavior tree:**

```yaml
tree:
  type: sequence
  children:
    - type: action
      do: party_ensure_npc_party
      leader_mob_id: 357
      home_room_id: 465
    - type: selector
      children:
        - type: sequence
          event: mob_death
          children:
            - type: action
              do: distribute_cargo_to_hostiles
        - type: sequence
          event: mob_idle
          children:
            - type: action
              do: party_follow_leader
```

## Draft horses

**IDs:** Hob 375, Bran 376. Same Thornwall zone (the caravan is
crewed and stationed there). Identical statline, different cosmetic
descriptions.

**Hob mob YAML (Bran mirrors with adjusted description):**

```yaml
mobid: 375
zone: Thornwall City
archetype: fighting
behavior_archetype: ""
statpool: 110
itemdropchance: 0
groups:
  - caravan
  - merchant_train
  - animal                 # corpse-salvage-eligible per Stage 3.0e
hostile: false
non_combatant: false       # bandits can attack
player_attack_immune: true # players can't
maxwander: -1
activitylevel: 5           # very rare idle emote
charm_immune: true
character:
  name: Hob, a dappled-grey draft horse
  description: |
    A solid, broad-shouldered draft horse with a patient
    eye, her coat dappled grey going to white at the
    muzzle. Her hooves are heavy with iron shoes, recently
    reset. The harness across her chest is good leather,
    oiled and supple. A small brass bell hangs from her
    bridle — silent at rest, soft at a walk.
  stats:
    strength: {training: 35}
    dexterity: {training: 15}
    vitality: {training: 30}
    perception: {training: 15}
    willpower: {training: 10}
    charisma: {training: 5}
  skills:
    weapon-combat: 5      # token kick/bite when bandits actually engage them
```

**Horse behavior tree (both same):**

```yaml
tree:
  type: selector
  children:
    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: attack
    - type: sequence
      event: mob_idle
      children:
        - type: action
          do: party_follow_leader
```

The horses don't need carry-capacity overrides — they're not carrying
anything. They don't need HP overrides either; their statpool 110 with
fighting archetype produces ~115 HP, sensible for a draft horse.

## Caravan party expansion

Ketil's btree's `party_ensure_npc_party` member list expands from
`[358, 359]` to `[358, 359, 374, 375, 376]`. The party becomes 6
members: 1 leader + 5 followers.

Thornwall depot room 465's `spawninfo` gains:

```yaml
- mobid: 374        # caravan wagon
- mobid: 375        # Hob, draft horse
- mobid: 376        # Bran, draft horse
```

## Engine override fields (mob struct extensions)

```go
// internal/mobs/mobs.go
type Mob struct {
    // ... existing fields
    CarryCapacityOverride float64 `yaml:"carry_capacity,omitempty"`
    HealthMaxOverride     int     `yaml:"health_max,omitempty"`
    StaminaMaxOverride    int     `yaml:"stamina_max,omitempty"`
    CorpseName            string  `yaml:"corpse_name,omitempty"`
    CorpseDescription     string  `yaml:"corpse_description,omitempty"`
    StockMultiplier       float64 `yaml:"stock_multiplier,omitempty"`
}
```

Override application happens at mob spawn time, AFTER the standard
stat-derived calculation. Each override is gated by `> 0` (or `!= ""`
for strings) so unset values fall through to defaults.

```go
// internal/characters/character.go (sketch)
func (c *Character) applyMobOverrides(m *mobs.Mob) {
    if m.CarryCapacityOverride > 0 {
        c.carryCapacity = m.CarryCapacityOverride
    }
    if m.HealthMaxOverride > 0 {
        c.HealthMax.Value = m.HealthMaxOverride
        c.Health = c.HealthMax.Value
    }
    if m.StaminaMaxOverride > 0 {
        c.StaminaMax.Value = m.StaminaMaxOverride
        c.Stamina = c.StaminaMax.Value
    }
    // corpse fields read at corpse-render time, not stamped onto Character
}
```

Corpse rendering paths read `mob.CorpseName` / `mob.CorpseDescription`
when they exist, fall back to the existing "<Name> corpse" pattern
otherwise. This affects:

- `internal/rooms/rooms.go:229` — corpse-decay message
- `internal/usercommands/look.go` — `look corpse` text
- Any other corpse-rendering site that calls `<Character.Name>`

## RarityTier system

**`ItemSpec.RarityTier int`** — new field on item YAML schema. Set
on every mat YAML during the audit task. Values: 50/40/30/20/10
(higher = more common).

**`Mob.StockMultiplier float64`** — new field on mob YAML schema.
Default 1.0. Set on each shopkeeper mob to scale its overall capacity.

**`shops.EffectiveMaxStock(itemId int, mob *mobs.Mob) int`** — new
helper:

```go
func EffectiveMaxStock(itemId int, mob *mobs.Mob) int {
    spec := items.GetItemSpec(itemId)
    if spec == nil || spec.RarityTier <= 0 {
        return 0
    }
    mult := mob.StockMultiplier
    if mult <= 0 {
        mult = 1.0
    }
    return int(float64(spec.RarityTier) * mult)
}
```

**Loader integration:** when `ShopInventory` is loaded for a vendor,
each `StockEntry`'s `MaxStock` is set to `EffectiveMaxStock(...)` if
not already specified. Existing `Restock()` and the new pickup pass
both consume `entry.MaxStock` as before — no API churn.

### Tier mapping

| Tier | Cap | Mats | Count |
|---|---|---|---|
| **50 — Common** | 50 | All Base bucket (13): 40001 iron ingot, 40003 wooden plank, 40006 glass vial, 40012 thread spool, 40013 bone needle, 40014 raw meat, 40015 wild vegetables, 40016 water flask, 40017 salt pouch, 40019 chain link, 40043 clay flask, 40044 sealed phial, 40045 crystalline decanter; PLUS 40021 copper wire, 40028 binding paste | 15 |
| **40 — Standard** | 40 | All mid-tier overlap (11): 40002 leather strip, 40004 healer's root, 40005 bitter thistle, 40007 cloth strip, 40008 spore sac, 40009 dustwalk herb, 40020 coal dust, 40047 veilbloom petal, 40048 serpent venom sac, 40050 putrid residue, 40068 sinew; PLUS 40011 Hive Fragment, 40018 steel ingot, 40022 silver wire, 40024 polished stone, 40026 gem dust, 40030 chrysalis setting | 17 |
| **30 — Regional** | 30 | Stillwater non-pearl (5): 40051 skitter-shrimp shell, 40056 marsh willow bark, 40057 lake mint, 40058 freshwater clam, 40059 lake-iron nodule. All Fernway (8): 40046 moonpetal, 40049 ironbark shaving, 40062 oak bark, 40063 shadowcap mushroom, 40064 wild hare meat, 40065 beeswax, 40066 blood-moss, 40067 pine pitch. PLUS 40025 raw gem | 14 |
| **20 — Uncommon** | 20 | 40053 Stillwater black pearl, 40010 Chrysalis Core, 40027 chrysalis shard, 40029 mutation catalyst, 40023 gold wire | 5 |
| **10 — Ultra-rare** | 10 | RESERVED | 0 |

Total classified: 51 mats. Quest items (40031–40042, 40054, 40060,
40061 — 15 items) and defer-to-3.0e items (40052, 40055) get NO
RarityTier; they are not vendor stock.

## Bidirectional vendor visit

```go
// internal/caravan/visit.go (rewrite)
func VisitVendorsInRoom(
    roomId int,
    wagon *mobs.Mob,
    deliveryBuckets []string,
    pickupBuckets []string,
) (delivered, pickedUp []ItemMove) {

    room := rooms.LoadRoom(roomId)
    if room == nil {
        return nil, nil
    }
    for _, instId := range room.GetMobs(rooms.FindAll) {
        vendor := mobs.GetInstance(instId)
        if vendor == nil || !vendor.HasShop() {
            continue
        }
        shop := shops.GetShopInventory(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId)
        if shop == nil {
            continue
        }

        // DELIVER pass: wagon → vendor
        for i := len(wagon.Character.Items) - 1; i >= 0; i-- {
            item := wagon.Character.Items[i]
            bucket := economy.BucketFor(item.ItemId)
            if bucket == "" || !slices.Contains(deliveryBuckets, bucket) {
                continue
            }
            entry := shop.GetStock(item.ItemId)
            if entry == nil || entry.Current >= entry.MaxStock {
                continue
            }
            wagon.Character.RemoveItem(item)
            entry.Current++
            delivered = append(delivered, ItemMove{
                Vendor: vendor.Character.Name,
                Item:   item.DisplayName(),
            })
        }

        // PICKUP pass: vendor → wagon
        for _, entry := range shop.Stock {
            bucket := economy.BucketFor(entry.ItemId)
            if bucket == "" || !slices.Contains(pickupBuckets, bucket) {
                continue
            }
            if entry.Current < entry.MaxStock/2 {
                continue
            }
            qty := entry.RestockQty
            if qty > entry.Current {
                qty = entry.Current
            }
            for j := 0; j < qty; j++ {
                item := items.New(entry.ItemId)
                if !item.IsValid() {
                    break
                }
                if !wagon.Character.StoreItem(item) {
                    break // wagon at carry cap
                }
                entry.Current--
                pickedUp = append(pickedUp, ItemMove{
                    Vendor: vendor.Character.Name,
                    Item:   item.DisplayName(),
                })
            }
        }
    }
    return delivered, pickedUp
}
```

**Bucket assignment per route phase** (set in `actions_caravan.go`):

| State | `deliveryBuckets` (wagon → vendor) | `pickupBuckets` (vendor → wagon) |
|---|---|---|
| `stillwater_route` | `["thornwall", "fernway"]` | `["stillwater"]` |
| `thornwall_route` | `["stillwater", "fernway"]` | `["thornwall"]` |

**Flavor messages** (room broadcast on a successful transfer):

- Both delivery + pickup at vendor: *"Marta hands [vendor] a small purse, takes a crate of [item] in trade."*
- Delivery only: *"Marta unloads a bundle of [item] for [vendor]."*
- Pickup only: *"Marta hands [vendor] a small purse and takes a crate of [item] for the road."*
- Vendor full + nothing matched: no message — caravan moves on quietly.

## Forager → vendor real-item delivery

```go
// internal/behaviortree/actions_forager.go
// tickForagerDeliveringTown rewrite

func tickForagerDeliveringTown(
    p *forager.ForagerProfile,
    mob *mobs.Mob,
    ctx *EvalContext,
) Result {
    idx := getIntFromState(ctx.MobState, keyVisitIndex)
    if idx >= len(p.VendorRooms) {
        transitionForager(ctx.MobState, forager.StateRecalling)
        return Success
    }
    target := p.VendorRooms[idx]
    if ctx.RoomId != target {
        mob.Command(fmt.Sprintf("pathto %d", target))
        return Success
    }
    // Real-item delivery
    deliverForagerSatchel(p, mob, ctx, target)
    ctx.MobState.Set(keyVisitIndex, strconv.Itoa(idx+1))
    return Success
}

func deliverForagerSatchel(
    p *forager.ForagerProfile,
    mob *mobs.Mob,
    ctx *EvalContext,
    roomId int,
) {
    room := rooms.LoadRoom(roomId)
    if room == nil { return }
    for _, instId := range room.GetMobs(rooms.FindAll) {
        vendor := mobs.GetInstance(instId)
        if vendor == nil || !vendor.HasShop() { continue }
        shop := shops.GetShopInventory(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId)
        if shop == nil { continue }
        for i := len(mob.Character.Items) - 1; i >= 0; i-- {
            item := mob.Character.Items[i]
            bucket := economy.BucketFor(item.ItemId)
            if bucket == "" || !slices.Contains(p.Buckets, bucket) {
                continue
            }
            entry := shop.GetStock(item.ItemId)
            if entry == nil || entry.Current >= entry.MaxStock {
                continue
            }
            mob.Character.RemoveItem(item)
            entry.Current++
            room.SendText(fmt.Sprintf(
                `<ansi fg="mobname">%s</ansi> hands a %s to <ansi fg="mobname">%s</ansi>.`,
                p.Name, item.DisplayName(), vendor.Character.Name,
            ))
        }
    }
}
```

Items that don't fit (vendor at cap, or wrong bucket for that vendor)
stay in the forager's satchel for the next vendor on the route. After
all vendors visited, the forager transitions to `recalling` carrying
whatever remained.

## Forager rest extension

```go
// internal/behaviortree/actions_forager.go
// tickForagerResting extension

func tickForagerResting(
    p *forager.ForagerProfile,
    mob *mobs.Mob,
    ctx *EvalContext,
) Result {
    if ctx.RoomId != p.SanctuaryRoom {
        mob.Command(fmt.Sprintf("pathto %d", p.SanctuaryRoom))
        return Success
    }
    startedStr := ctx.MobState.GetString(keyStateStartedRound)
    started, _ := strconv.ParseUint(startedStr, 10, 64)
    dwellElapsed := util.GetRoundCount() >= started+restingDuration
    if !dwellElapsed {
        return Failure // resting — let legacy idle fire flavor
    }
    if mob.Character.Health < mob.Character.HealthMax.Value {
        return Failure
    }
    // NEW: stay home if satchel still over half-full — vendors didn't
    // absorb much last cycle, so foraging more would just overflow.
    // Narratively: "Vella sits at the temple looking content; the
    // merchants don't need more right now."
    if carryRatio(mob) > 0.5 {
        return Failure
    }
    // All clear — go work.
    transitionForager(ctx.MobState, forager.StateTravelingToTerritory)
    return Success
}
```

The 50% threshold is a config knob: `Balance.ForagerRestCarryThreshold`
(default 0.5).

## Wagon death — distribute cargo to hostiles

**New btree action: `distribute_cargo_to_hostiles`**

```go
// internal/behaviortree/actions_caravan.go (or a new actions_wagon.go)

func actDistributeCargoToHostiles(params map[string]any, ctx *EvalContext) Result {
    wagon := mobs.GetInstance(ctx.InstanceId)
    if wagon == nil {
        return Failure
    }
    room := rooms.LoadRoom(ctx.RoomId)
    if room == nil {
        return Failure
    }
    // Find hostile mobs in the room (those whose Hates intersects
    // the wagon's Groups — i.e., bandits with caravan/merchant_train hate)
    var hostiles []*mobs.Mob
    for _, instId := range room.GetMobs(rooms.FindAll) {
        if instId == wagon.InstanceId { continue }
        m := mobs.GetInstance(instId)
        if m == nil { continue }
        if m.HatesAnyGroup(wagon.Groups) {
            hostiles = append(hostiles, m)
        }
    }
    if len(hostiles) == 0 {
        return Failure // fall through to standard mob-death corpse drop
    }
    // Round-robin distribution, capped by each hostile's carry capacity
    h := 0
    for i := len(wagon.Character.Items) - 1; i >= 0; i-- {
        item := wagon.Character.Items[i]
        // Try up to len(hostiles) targets before giving up on this item
        placed := false
        for tries := 0; tries < len(hostiles); tries++ {
            target := hostiles[h]
            h = (h + 1) % len(hostiles)
            if target.Character.StoreItem(item) {
                wagon.Character.RemoveItem(item)
                placed = true
                break
            }
        }
        if !placed {
            // All hostiles at cap; remaining items drop as wagon-corpse pile
            break
        }
    }
    return Success
}
```

Items that don't fit anywhere drop in the standard wagon-corpse path —
players can recover them from the wreckage.

## Caravan death recovery

Standard Stage 2 path: Ketil's death dissolves the party; all members
respawn at depot. Stage 3.4 changes:

- Wagon respawns with empty inventory (per standard mob-respawn — no
  state preservation across deaths)
- Horses respawn with their token inventories (none) and full HP/SP
- First post-respawn `party_ensure_npc_party` rebuilds the 6-mob party

The cargo loss is intentional. Players who arrive after the brawl find
the wreckage with leftover cargo + bandits with looted items in their
inventories.

## Chrysalis Core re-source

**Today:** Chrysalis Core (40010) drops from Aberrant Chrysalis (mob 69)
in Sanctum Basin tutorial. This is unbalanced — a tutorial-tier mob
dropping a tier-20 specialty mat. Vael (109) and Voss (98) stock the
item but do NOT craft it.

**Stage 3.4 change:**

| Mob | ID | Change |
|---|---|---|
| Aberrant Chrysalis | 69 | REMOVE Chrysalis Core (40010) from drops |
| Stone beetle queen | 228 | ADD Chrysalis Core 10% drop (thematic — chrysalis transformation imagery) |
| Windscour wyrm | 229 | ADD Chrysalis Core 5% drop (apex-tier rare source) |

Rationale: Chrysalis Core becomes a real Ironwind Steppe drop, gating
Vael's chrysalis production behind real exploration. Future content
stages can add cave-forage in northern Ironwind for a non-combat
acquisition path.

## Edge cases

| Scenario | Behavior |
|---|---|
| Wagon full when caravan tries to pick up at vendor | Pickup pass simply skips items that can't fit. Vendor retains stock. With CarryCapacity 5000 this should never happen in practice. |
| Vendor at MaxStock for an item the caravan tries to deliver | Skip that item. Stays in wagon for next vendor / next cycle. |
| All vendors at MaxStock for everything in wagon | Wagon retains everything. Returns to depot full. Next cycle re-tries. With players consuming, vendors reopen space. |
| Forager's satchel hit cap mid-foraging | `tickForagerForaging` already transitions to `traveling_to_dropoff` at 75% carry. New rest extension keeps her home post-delivery if cap remains > 50%. |
| Wagon dies in 4052 brawl with no bandits in room (DoT, fled-bandit case) | `distribute_cargo_to_hostiles` finds zero hostiles; falls through to standard mob-death corpse-drop. All items recoverable from wreckage corpse. |
| Wagon dies, bandits at full carry capacity | Round-robin distribution stops when first bandit hits cap; remaining items drop as wagon-corpse pile. Players get most loot from bandits, scraps from wreckage. |
| Caravan-respawn after wipe | Standard spawninfo respawn at room 465. Fresh wagon (mob 374) with empty inventory. Same for horses. Caravan party re-forms via `party_ensure_npc_party` on Ketil's first idle tick. |
| Caravan reaches Stillwater depot with no Fernway pickup this cycle | Wagon contains only items from previous Thornwall pickup pass. Stillwater visit delivers Thornwall-bucket items, picks up Stillwater-bucket items. Standard flow. |
| Forager death | Forager respawns at sanctuary with empty satchel (per existing mob-death handling). Items the forager was carrying drop at the death site, available to whoever finds the corpse. |
| Two foragers visit same vendor town | Vella → Stillwater vendors only. Halix → Thornwall only. Kessa → caravan-only (never enters towns). No collision. |
| In-shop crafter (Vael, Voss, Kerra, Tess) restock tick | Untouched. Existing `TickMobShopRestock` for non-caravan-served zones still runs; for served zones, the existing skip remains. Crafters in served zones tick their CRAFTED slot's stock locally as before. |
| `look wagon` in transit | Shows wagon description + current Items (whatever's loaded mid-cycle). Players can see flow in real time. |
| Player picks up item from wagon-wreckage corpse | Standard corpse-loot flow. No special case. |

## Testing strategy

### Unit tests

| Package | Test | Verifies |
|---|---|---|
| `internal/items` | `TestItemSpec_RarityTier_YAMLRoundtrip` | Field parses correctly, defaults to 0 for legacy items |
| `internal/shops` | `TestEffectiveMaxStock_TierTimesMultiplier` | 50 × 1.0 = 50; 30 × 5.0 = 150; tier-0 fallback |
| `internal/shops` | `TestEffectiveMaxStock_LegacyOverride` | Per-StockEntry MaxStock still wins if explicitly set |
| `internal/caravan` | `TestVisitVendorsInRoom_DeliveryOnly` | Wagon → vendor when bucket matches; vendor at cap = no transfer |
| `internal/caravan` | `TestVisitVendorsInRoom_PickupOnly` | Vendor → wagon when ≥ MaxStock/2; below floor = no pickup |
| `internal/caravan` | `TestVisitVendorsInRoom_BothPasses` | Mixed delivery + pickup at one vendor stop |
| `internal/caravan` | `TestVisitVendorsInRoom_WagonAtCarryCap` | Pickup pass skips items that exceed wagon weight |
| `internal/behaviortree` | `TestForagerStep_RestExtensionStaysHomeWhenSatchelFull` | Vella with carry > 50% stays in resting |
| `internal/behaviortree` | `TestForagerStep_DeliveringTownTransfersRealItems` | Vella's items move to vendor inventories at each stop |
| `internal/behaviortree` | `TestDistributeCargoToHostiles_RoundRobin` | Wagon items distribute across multiple hostile mobs |
| `internal/behaviortree` | `TestDistributeCargoToHostiles_NoHostiles_FallsThrough` | Empty hostile list = standard corpse drop (returns Failure) |
| `internal/behaviortree` | `TestDistributeCargoToHostiles_BanditAtCarryCap` | Stops distributing when target hits cap; remaining drops as corpse |
| `internal/mobs` | `TestMobOverrides_CarryCapacity` | `carry_capacity` field overrides Strength-derived calc |
| `internal/mobs` | `TestMobOverrides_HealthMax` | `health_max` field overrides Vitality-derived calc |
| `internal/mobs` | `TestMobOverrides_StaminaMax` | `stamina_max` field overrides default calc |
| `internal/rooms` | `TestCorpseRendering_OverrideName` | When `corpse_name` set, room display uses it instead of "<Name> corpse" |

### In-game smoke test

1. Boot. Caravan party at Thornwall depot 465 includes 6 mobs: Ketil, Marta, Lars, wagon, Hob, Bran.
2. `look wagon` shows the wagon's description AND `Items: (empty)` initially.
3. Halix delivers his satchel directly to Thornwall vendors. Confirm Kerra's iron ingot stock increases. Confirm flavor message fires.
4. Caravan departs Thornwall. Wagon empty until Fernway pickup.
5. At room 4038, if Kessa is present: handoff. `look wagon` now shows the fernway items.
6. At first Stillwater vendor (e.g., Smith Brindle 4106): caravan delivers fernway items + picks up stillwater-bucket items. Watch for delivery + pickup flavor messages. `look wagon` should show items shifting.
7. Caravan reaches Stillwater depot. Vella delivers her satchel separately to Stillwater vendors during dwell.
8. Caravan starts inbound. At each Thornwall vendor: deliver stillwater + fernway items, pick up thornwall-bucket items.
9. Caravan returns to Thornwall depot. Cycle complete. Verify Brindle has fewer lake-iron than start of cycle, Kerra has more.
10. **Brawl test:** lure caravan into combat at 4052 (or admin-force a wipe). Verify wagon's items distribute to bandit inventories. Kill bandits. Verify items recoverable from bandit corpses. Verify wagon corpse renders as "splintered wagon wreckage" with leftover items as a corpse pile.
11. **Capacity smoke:** force the caravan to do several full cycles with no players consuming. After ~5 cycles, verify wagon has accumulated items but is well below 5000 weight cap. Verify Vella's rest extension fired (find her at sanctuary with full satchel).
12. **Chrysalis Core:** kill Aberrant Chrysalis in Sanctum Basin tutorial. Confirm NO Chrysalis Core drops. Kill stone beetle queen ~10x. Confirm Chrysalis Core appears at roughly 10% rate. Kill windscour wyrm ~20x. Confirm ~5% rate.
13. **Tier mapping smoke:** `inventory smith brindle` shows MaxStock values matching the tier table (e.g., iron ingot 50, lake-iron 30).

### Verification phases

- **Phase 1 — unit tests:** `go test ./...` green
- **Phase 2 — server boot clean:** all data files load (~50 mat YAMLs gain RarityTier, 17 vendor YAMLs lose explicit MaxStock, 3 new caravan mobs load, behavior trees parse)
- **Phase 3 — smoke test (13-step sequence above)**
- **Phase 4 — backward compat:**
  - Stage 1 NPC parties: still work (caravan party just got bigger)
  - Stage 2 caravan state machine: unchanged
  - Stage 3.0d fold-recall: unchanged (caravan crew still uses)
  - Stage 3.0e corpse salvage: works on horse corpses (animal group)
  - Stage 3.1 forager state machine: extended (rest rule), other paths unchanged
  - Non-caravan-served vendors (Sanctum Basin, Watchers Crossing, Ashwick): unchanged

## Out of scope (explicitly)

- **Per-stock-entry MaxStock overrides for caravan-served vendors** — the `EffectiveMaxStock` fallback supports legacy overrides if needed, but no caravan-served vendor uses this. Tier-derived is uniform.
- **Real gold flow** — caravan paying vendors / vendors paying caravan is flavor emote only. No actual gold movement. Spec'd value-balance work deferred to a future stage if ever needed.
- **Player-rideable wagon / passenger system** — Vehicle abstraction explicitly deferred per Q1 brainstorming. If/when DOGMud needs piloted vehicles, that becomes its own design.
- **Cave forage / additional Chrysalis Core sources** beyond beetle queen + wyrm — future content stages.
- **Halix going through The Fernway** like Vella does — Halix-specific worldbuilding stays simpler in scope.
- **Vendor-to-vendor direct trade** (i.e., shopkeepers trading with each other when caravan absent) — caravan is the only cross-town flow.
- **Caravan-related quests** (escort, ambush, etc.) — possible future content.
- **Player-attackable wagon / horse griefing prevention beyond `player_attack_immune`** — existing flag covers it.
- **Tier-10 ultra-rare mat content** — reserved for future expansion. No current items meet this threshold.
- **MaxStock audit for non-caravan-served zones** — Sanctum Basin, Dustwalk, etc. retain current values. Touching them invites unrelated breakage.

## Open implementation questions (for the plan stage)

- Exact corpse-rendering integration points for `CorpseName` / `CorpseDescription` overrides — survey the codebase for every "<Character.Name> corpse" string format and adapt each one. Plan task 0 surveys.
- Whether `EffectiveMaxStock` should also be used for non-caravan-served vendors going forward (could simplify the system long-term) — out of scope for 3.4 but worth a follow-up note.
- Bandit `HatesAnyGroup` resolution — confirm the existing API on Mob; the Stage 2 spec referenced this method. Plan task confirms.
- Final names for the draft horses (Hob + Bran are placeholders).
- `ForagerRestCarryThreshold` default value — 0.5 is the spec recommendation; tunable via config.

## Files affected

| Action | File | Purpose |
|---|---|---|
| MODIFY | `internal/mobs/mobs.go` | Add 6 override fields to Mob struct |
| MODIFY | `internal/characters/character.go` | Apply HP/SP/CarryCap overrides at spawn |
| MODIFY | `internal/items/itemspec.go` | Add `RarityTier int` field |
| CREATE | `internal/shops/effective_max_stock.go` | `EffectiveMaxStock` helper |
| MODIFY | `internal/shops/shopinventory.go` | Loader integration of EffectiveMaxStock |
| MODIFY | `internal/caravan/visit.go` | Rewrite VisitVendorsInRoom for bidirectional real-item flow |
| MODIFY | `internal/behaviortree/actions_caravan.go` | Pass wagon + delivery/pickup buckets to VisitVendorsInRoom |
| CREATE | `internal/behaviortree/actions_wagon.go` | `distribute_cargo_to_hostiles` action |
| MODIFY | `internal/behaviortree/actions_forager.go` | Rewrite tickForagerDeliveringTown + extend tickForagerResting |
| MODIFY | `internal/configs/config.balance.go` | Add `ForagerRestCarryThreshold` |
| MODIFY | `_datafiles/config.yaml` | Default for new knob |
| MODIFY | `internal/rooms/rooms.go` (~line 229) | Use mob.CorpseName when set |
| MODIFY | `internal/usercommands/look.go` | Use mob.CorpseDescription when looking at corpse |
| CREATE | `_datafiles/world/dogmud/mobs/thornwall_city/374-caravan_wagon.yaml` | Wagon mob |
| CREATE | `_datafiles/world/dogmud/behaviors/thornwall_city/374-caravan_wagon.yaml` | Wagon btree |
| CREATE | `_datafiles/world/dogmud/mobs/thornwall_city/375-pell.yaml` | Hob horse |
| CREATE | `_datafiles/world/dogmud/behaviors/thornwall_city/375-hob.yaml` | Hob btree |
| CREATE | `_datafiles/world/dogmud/mobs/thornwall_city/376-brick.yaml` | Bran horse |
| CREATE | `_datafiles/world/dogmud/behaviors/thornwall_city/376-bran.yaml` | Bran btree |
| MODIFY | `_datafiles/world/dogmud/behaviors/thornwall_city/357-ketil.yaml` | Update party_ensure_npc_party member list |
| MODIFY | `_datafiles/world/dogmud/rooms/thornwall_city/465.yaml` | Add wagon + horse spawninfo |
| MODIFY | ~50 mat YAMLs in `_datafiles/world/dogmud/items/materials-40000/` | Set `rarity_tier` |
| MODIFY | 17 caravan-served vendor mob YAMLs | Drop explicit max_stock from each StockEntry |
| MODIFY | `_datafiles/world/dogmud/mobs/sanctum_basin/69-aberrant_chrysalis.yaml` | Remove Chrysalis Core (40010) drop |
| MODIFY | `_datafiles/world/dogmud/mobs/ironwind_steppe/228-stone_beetle_queen.yaml` | Add Chrysalis Core 10% drop |
| MODIFY | `_datafiles/world/dogmud/mobs/ironwind_steppe/229-windscour_wyrm.yaml` | Add Chrysalis Core 5% drop |
| MODIFY | `docs/economy/mat-audit-matrix.md` | Add rarity-tier column to the matrix |
| MODIFY | `docs/schemas/mob.md` | Document 6 new override fields |
| MODIFY | `docs/schemas/item.md` | Document `rarity_tier` |
| MODIFY | `docs/schemas/behavior.md` | Document `distribute_cargo_to_hostiles` action |
| MODIFY | `PATCH_NOTES.md` | Stage 3.4 entry — and the prod-ready entry that promotes the entire economy stack |
