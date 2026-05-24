# Mob Aliveness 2.3 — Equip-If-Better Behavior

> **Phase 2 tactical (third chunk).** Two paths: rewrite the
> existing `gearup` mobcommand's gold-value heuristic to use
> `itemvalue.IsUpgrade` from chunk 2.2 (the give-equip "push"
> path), and add an idle-tick floor-loot scan (the new "pull"
> path). Gate both by archetype, species, and incorporeal state.
> Charmed-status additionally gates pull.

## Goal

Make NPCs visibly smart about gear: when a player gives them an
upgrade they wield/wear it, and when they walk into a room with
useful loot on the floor they pick it up. The chunk brief
captures the user-facing failure mode: "I gave the bandit a steel
sword and he's still using a club" — fixed. Plus the converse:
"I dropped a sword on the floor and the bandit didn't pick it up
even though it's better than what he's holding" — also fixed.

The chunk's job is **two thin consumer-site additions plus a
small shared gating layer**. Chunk 2.2 already shipped the
`itemvalue.IsUpgrade` primitive; chunk 2.2a's Incorporeal
mutation already gates ethereal mobs via itemvalue scoring. This
chunk is the behavior that consumes those primitives.

## Architectural musts

The chunk brief lists "btree action, per-archetype configurable,
emits emote when swapping" as in-scope. Brainstorming refined
the framing:

1. **Push path = rewrite existing `gearup` mobcommand**, not a
   new btree action. The give → fallback → `gearup` chain is
   already wired in `give.go:237`. Today `gearup` uses
   `item.Value` (gold) as the upgrade heuristic — wrong proxy.
   Swap that for `itemvalue.IsUpgrade(char, profile, candidate)`.
   No new btree action; the per-archetype "configurability"
   from the brief is satisfied by `itemvalue.ProfileFor`'s
   weight profiles (chunk 2.2), which encode per-archetype
   gear preferences uniformly.

2. **Pull path = idle-tick floor-loot scan**, integrated into
   the existing `MobIdle_HandleIdleMobs.go` per-mob loop. A
   single new function `EquipBestFloorItem(mob, room) bool`
   scans `room.Items`, scores each via `ItemValueDelta`,
   equips the best positive-scoring upgrade. Combat-state
   gated (mob with `Aggro != nil` skips the scan).

3. **Two gate helpers, both in `internal/itemvalue/`:**
   - `CanEquipFromGive(mob)`: skips animal species (species
     with Weapon in `DisabledSlots`), non-combat archetypes
     (`noncombat_*`, `prey`, `combat_passive`).
   - `CanScanFloorLoot(mob)`: `CanEquipFromGive` plus
     `!mob.Character.IsCharmed()`. Companions and mercs accept
     pushes from their owner but skip pull (owner has dibs).

4. **Frequency: every idle tick**, no cooldown. The hot path
   exits in ~5 operations for non-eligible mobs or empty
   rooms. The expensive part (`ItemValueDelta` per floor item)
   runs only when an eligible mob has floor items to consider
   — rare in practice. Cooldown can be a tuning knob later if
   playtesting shows mobs hoover up player loot too quickly.

5. **Displaced item disposition: preserved from existing
   `gearup`.** Wild mobs keep displaced items in their
   backpack (loot when killed). Charmed mobs drop displaced
   items to the floor (owner can reclaim). This is the
   pre-existing convention; the rewrite preserves it. Pull
   path doesn't need separate handling — charmed mobs can't
   reach the pull path (CanScanFloorLoot returns false).

6. **Incorporeal handling falls out automatically.** Chunk
   2.2a's Incorporeal mutation soft-scales gear contributions
   via `ItemValueDelta`. A rank-4 incorporeal mob (wraith,
   spectre, fire/air elemental, elemental queen) sees all
   gear score 0 → `IsUpgrade` returns false → no swap. No
   special skip path in 2.3 for ethereal mobs.

7. **Emote semantics: distinct phrasing for push vs pull.**
   Push uses the existing "puts on" / "wields" phrasing from
   `equip` mobcommand (preserved). Pull uses "picks up
   <item> and dons it" / "picks up <item> and wields it" to
   signal the loot-pickup origin.

## Architecture & module layout

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/itemvalue/equip_eligibility.go` | NEW | `CanEquipFromGive(mob) bool`, `CanScanFloorLoot(mob) bool`, `EquipBestFloorItem(mob, room) bool` |
| `internal/itemvalue/equip_eligibility_test.go` | NEW | Gate tests + scan tests |
| `internal/mobcommands/gearup.go` | MODIFY | Replace gold-value heuristic with `itemvalue.IsUpgrade`; add `CanEquipFromGive` gate |
| `internal/mobcommands/gearup_test.go` | NEW or expand | Gearup behavior tests |
| `internal/hooks/MobIdle_HandleIdleMobs.go` | MODIFY | Add `itemvalue.EquipBestFloorItem(mob, room)` call alongside existing per-mob idle behaviors |
| `internal/itemvalue/context.md` | MODIFY | Document new helpers + push/pull behaviors |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Mark 2.3 Done, roll-up 11/41 |

The helpers live in `internal/itemvalue/` to colocate with the
item-value primitives they gate. `itemvalue` already imports
`characters`, `items`, `mutations`; adding `mobs` and `species`
is a small expansion that keeps equip-related logic in one
package.

## Public API

```go
package itemvalue

// CanEquipFromGive returns true if this mob is the kind of
// creature that can rationally equip gear from a player give.
// Returns false for:
//   - Animal species (Species.DisabledSlots contains "Weapon")
//   - Non-combat archetypes (noncombat_*, prey, combat_passive)
// Note: incorporeal-rank-4 mobs are NOT explicitly skipped here;
// their IsUpgrade naturally returns false because gear scores 0.
func CanEquipFromGive(mob *mobs.Mob) bool

// CanScanFloorLoot returns true if this mob should scan floor
// loot for upgrades on idle ticks. Equals CanEquipFromGive plus
// !mob.Character.IsCharmed(). Companions and mercs accept
// pushes from their owner but skip pull (owner has dibs).
func CanScanFloorLoot(mob *mobs.Mob) bool

// EquipBestFloorItem scans items on the floor of mob's room,
// scores each via ItemValueDelta, and equips the best positive-
// scoring upgrade if any. Returns true if a swap occurred.
//
// No-op (returns false) if any of:
//   - !CanScanFloorLoot(mob)
//   - mob is in combat (Aggro != nil)
//   - room is nil or has no floor items
//   - no floor item scores as an upgrade for this mob+profile
//
// On successful pickup, emits a room broadcast distinct from
// give-equip phrasing ("picks up X and dons it" / "wields it").
// Displaced items go to the mob's backpack per the actions.EquipItem
// default behavior (charmed mobs don't reach this path).
func EquipBestFloorItem(mob *mobs.Mob, room *rooms.Room) bool
```

## Gate helpers

`CanEquipFromGive(mob)` implementation:

```go
func CanEquipFromGive(mob *mobs.Mob) bool {
    if mob == nil {
        return false
    }

    // Animal-species gate: species with Weapon slot disabled.
    if speciesInfo := species.GetSpecies(mob.Character.SpeciesId); speciesInfo != nil {
        for _, slot := range speciesInfo.DisabledSlots {
            if slot == "Weapon" {
                return false
            }
        }
    }

    // Non-combat archetypes silently skip equip-if-better.
    switch mob.BehaviorArchetype {
    case "noncombat_passive", "noncombat_questgiver",
        "noncombat_shopkeeper", "prey", "combat_passive":
        return false
    }

    return true
}
```

`CanScanFloorLoot(mob)` implementation:

```go
func CanScanFloorLoot(mob *mobs.Mob) bool {
    if !CanEquipFromGive(mob) {
        return false
    }
    return !mob.Character.IsCharmed()
}
```

**Notes on the gates:**

- The animal check tests for `"Weapon"` specifically. A species
  with `DisabledSlots: [Weapon, Body, ...]` (typical animal —
  wolf, bear, rat) is filtered. A species with only specific
  slots disabled (e.g., centaur disables Legs but keeps
  Weapon/Body) is NOT filtered — they can still equip what
  their species allows.
- The non-combat archetypes use string-match against
  `BehaviorArchetype`. The exhaustive list is checked at this
  one point in code; if new non-combat archetypes are added
  later they need to be added here too. Acceptable maintenance
  surface for v1.
- `IsCharmed()` is a `*characters.Character` method that
  returns true for any mob currently charmed to a player —
  both companions (via charm spell or similar) and mercs (via
  future paid-hire flow). One signal covers both.

## Push path: gearup rewrite

The existing `give.go:237` fallback chain remains unchanged:

```go
// In give.go, after a player gives an item to a mob:
if behaviortree.TryMobBehavior(...) {
    return true, nil
}
m.Command(`emote considers the X for a moment.`)
m.Command(`gearup !<itemid>`)
```

The new `gearup` body in `internal/mobcommands/gearup.go`:

```go
func Gearup(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
    // PermaGear buff blocks any equip change (boss equipment lockdown).
    if mob.Character.HasBuffFlag(buffs.PermaGear) {
        mob.Command(`emote struggles with their gear for a while, then gives up.`)
        return true, nil
    }

    // Animal species, non-combat archetypes, prey, etc. silently skip.
    if !itemvalue.CanEquipFromGive(mob) {
        return true, nil
    }

    profile := itemvalue.ProfileFor(mob.Archetype, mob.BehaviorArchetype)
    actor := &actions.MobActor{Mob: mob, Room: room}

    if rest != "" {
        // Specific-item case: "gearup !12345" or "gearup sword name"
        candidate, found := mob.Character.FindInBackpack(rest)
        if !found {
            return true, nil
        }
        if !itemvalue.IsUpgrade(&mob.Character, profile, candidate) {
            return true, nil
        }
        equipAndDisplay(actor, candidate, mob)
        return true, nil
    }

    // Bare "gearup": scan backpack, equip each item that's an
    // upgrade. Iteration order doesn't matter — equipping
    // changes the baseline so subsequent IsUpgrade calls
    // reflect the new loadout.
    itemsList := mob.Character.GetAllBackpackItems()
    for _, itm := range itemsList {
        if !itemvalue.IsUpgrade(&mob.Character, profile, itm) {
            continue
        }
        equipAndDisplay(actor, itm, mob)
    }
    return true, nil
}

// equipAndDisplay invokes actions.EquipItem, emits the room
// broadcast text, and (for charmed mobs) drops displaced items
// to the floor so the owner can reclaim them. Wild mobs keep
// displaced items in their backpack (the actions.EquipItem
// default).
func equipAndDisplay(actor *actions.MobActor, itm items.Item, mob *mobs.Mob) {
    isCharmed := mob.Character.IsCharmed()
    oldEquipped := mob.Character.Equipment.GetAllItems()

    result := actions.EquipItem(actor, itm.Name())
    if !result.Equipped {
        return
    }

    spec := result.Item.GetSpec()
    if spec.Subtype == items.Wearable {
        actor.Room.SendTextVisual(fmt.Sprintf(
            `<ansi fg="mobname">%s</ansi> puts on <ansi fg="item">%s</ansi>.`,
            mob.Character.Name, result.Item.DisplayName()))
    } else {
        actor.Room.SendTextVisual(fmt.Sprintf(
            `<ansi fg="mobname">%s</ansi> wields <ansi fg="item">%s</ansi>.`,
            mob.Character.Name, result.Item.DisplayName()))
    }

    // Charmed-mob convention: drop displaced items to the floor.
    if isCharmed {
        newEquipped := mob.Character.Equipment.GetAllItems()
        for _, oldItm := range oldEquipped {
            if oldItm.ItemId < 1 {
                continue
            }
            stillEquipped := false
            for _, newItm := range newEquipped {
                if oldItm.ItemId == newItm.ItemId {
                    stillEquipped = true
                    break
                }
            }
            if !stillEquipped {
                mob.Command(fmt.Sprintf(`drop !%d`, oldItm.ItemId))
            }
        }
    }
}
```

**Key behavior changes vs old gearup:**

| Aspect | Old | New |
|---|---|---|
| Upgrade heuristic | `item.Value > equipped.Value` (gold value) | `itemvalue.IsUpgrade(char, profile, candidate)` |
| Animal / non-combat gate | None | Skip via `CanEquipFromGive` |
| Iteration sort | Sort by gold value desc | None — IsUpgrade does the work per-item |
| Slot / 2H awareness | None | Inherited from `ItemValueDelta` (handles slot conflicts + 2H displacement) |

**Emote behavior preserved:**
- "puts on" for wearable
- "wields" for weapon
- Charmed-drop convention for displaced items
- PermaGear buff blocks equip attempts

## Pull path: idle floor-loot scan

The pull path is new — no existing equivalent. A wild non-charmed
combat mob, on idle tick, scans its room's floor for items that
score as upgrades and equips the best one.

**Function body** (`internal/itemvalue/equip_eligibility.go`):

```go
func EquipBestFloorItem(mob *mobs.Mob, room *rooms.Room) bool {
    if !CanScanFloorLoot(mob) {
        return false
    }
    if mob.Character.Aggro != nil {
        return false // busy fighting
    }
    if room == nil || len(room.Items) == 0 {
        return false // nothing on floor
    }

    profile := ProfileFor(mob.Archetype, mob.BehaviorArchetype)

    // Find the floor item with the highest positive delta score.
    var bestItem items.Item
    bestScore := 0.0
    for _, floorItem := range room.Items {
        delta := ItemValueDelta(&mob.Character, profile, floorItem)
        if delta.Score > bestScore {
            bestScore = delta.Score
            bestItem = floorItem
        }
    }
    if bestItem.ItemId == 0 {
        return false // nothing was an upgrade
    }

    // Remove from floor and into backpack so EquipItem can find it.
    room.RemoveItem(bestItem, false)
    mob.Character.StoreItem(bestItem)

    actor := &actions.MobActor{Mob: mob, Room: room}
    result := actions.EquipItem(actor, bestItem.Name())
    if !result.Equipped {
        // Edge case: ItemValueDelta thought slot was compatible
        // but EquipItem refused (rare — e.g., mutation just
        // changed mid-iteration). Item is still in backpack;
        // mob effectively "picked it up" without equipping.
        return false
    }

    // Room broadcast: distinct from give-equip ("picks up X
    // and dons it") to signal the loot-pickup origin.
    spec := result.Item.GetSpec()
    if spec.Subtype == items.Wearable {
        room.SendTextVisual(fmt.Sprintf(
            `<ansi fg="mobname">%s</ansi> picks up <ansi fg="item">%s</ansi> and dons it.`,
            mob.Character.Name, result.Item.DisplayName()))
    } else {
        room.SendTextVisual(fmt.Sprintf(
            `<ansi fg="mobname">%s</ansi> picks up <ansi fg="item">%s</ansi> and wields it.`,
            mob.Character.Name, result.Item.DisplayName()))
    }

    return true
}
```

**Idle-tick wiring** (`internal/hooks/MobIdle_HandleIdleMobs.go`):

Inside the existing per-mob idle iteration, after other idle
behaviors (wander, gossip, emote), add:

```go
// Floor-loot scan: wild non-charmed combat mobs pick up gear
// upgrades they find lying around.
itemvalue.EquipBestFloorItem(mob, room)
```

The exact insertion point depends on the current ordering of
behaviors in the idle handler. Place it AFTER combat-related
checks (so a mob entering combat that round doesn't try to
loot first) but BEFORE wander (so a mob about to leave the
room first checks for upgrades in this room).

**Cost analysis (sanity check the "every idle tick" decision):**

For each idle tick × each mob, the hot path:
- `CanScanFloorLoot(mob)`: ~3 reads (species, archetype, charmed) — typically false-skip for most NPCs (charmed/non-combat/animal coverage is wide)
- `mob.Character.Aggro != nil`: one pointer check
- `room == nil || len(room.Items) == 0`: two reads

Common-case exit: ~5 operations. Cheap enough to run unconditionally on every tick.

When eligible-mob meets a room with floor items:
- One `ItemValueDelta` call per floor item
- Most rooms have 0-3 floor items
- Realistic worst case: ~30 multiplications per scan

Negligible. The "every idle tick" choice doesn't need a
cooldown for performance reasons. If anti-fun emerges
(mobs grabbing loot before players can pick it up), a
cooldown can be added as a tuning knob.

## Testing

Per the chunks 2.1/2.2/2.2a pattern, tests use synthetic Mob/
Character values inline. Fixture-dependent integration cases
SKIP cleanly with documented reasons; full coverage relies on
Task-12 smoke.

| Test file | Cases |
|---|---|
| `internal/itemvalue/equip_eligibility_test.go` | `CanEquipFromGive`: animal species (mock Species with DisabledSlots: [Weapon]) → false; non-combat archetype → false; `prey` → false; `combat_passive` → false; regular bruiser (no disabled slots, generic_fighter) → true. `CanScanFloorLoot`: above + charmed → false; non-charmed bruiser → true. |
| `internal/itemvalue/equip_eligibility_test.go` | `EquipBestFloorItem`: nil room → false; empty floor → false; non-eligible mob → false; in-combat mob (Aggro != nil) → false. Real swap requires fixture loading; documented and skipped. |
| `internal/mobcommands/gearup_test.go` | PermaGear → no-op + emote (existing behavior preserved); non-eligible mob → silent no-op (new behavior); specific-item that's not an upgrade → no-op (new behavior, replacing the gold-value comparison); rest of the cases require fixture-loaded items + IsUpgrade machinery, deferred to smoke. |

## Smoke test

After unit tests pass:

1. `go build ./...` clean
2. `go test ./...` no FAILs
3. Boot the server, watch for clean data load
4. Spot-check via admin:
   - **Push (positive):** spawn a bandit, give it a stronger weapon than its current → wields the new one and emotes
   - **Push (negative):** give the bandit an inferior weapon → keeps in backpack but doesn't wield
   - **Push (animal):** give a sword to a wolf → no-op
   - **Push (non-combat):** give a dagger to a shopkeeper → no-op
   - **Push (incorporeal):** give a sword to a wraith → no-op (ItemValueDelta returns 0 score from gear-effectiveness scaling)
   - **Pull (positive):** drop a stronger-than-equipped weapon in a bandit's room → next idle tick, bandit picks up and wields, emits "picks up X and wields it"
   - **Pull (charmed):** drop loot in a room where your companion is → companion ignores it (CanScanFloorLoot returns false)
   - **Pull (in combat):** drop loot during active combat → mob doesn't grab it mid-fight

## Out of scope / deferred

- **Hysteresis / minimum upgrade margin.** Strict `Score > 0`
  threshold. If thrashing emerges, add a `MinUpgradeDelta`
  knob to `WeightProfile` in a tuning pass. Captured as a
  MEMORY follow-on.
- **Per-archetype opt-outs beyond animal + non-combat.** E.g.,
  a `leader` mob choosing not to grab floor loot mid-patrol
  for flavor reasons. v1 ships the basic gates; archetype-
  specific opt-outs are content-pass concerns.
- **Floor-loot cooldown.** Every idle tick for v1. Tuning
  knob if anti-fun emerges.
- **Scan radius beyond current room.** Mobs don't peek into
  adjacent rooms; cross-room awareness is a different design
  conversation (potential Phase 3 routine-layer chunk).
- **Auto-equip on mob spawn.** Mobs spawn with their YAML-
  defined equipment; we don't re-evaluate at spawn. The first
  idle-tick after spawn would naturally trigger floor-loot
  pickup if there's an upgrade lying around.
- **Player-side equip-if-better.** Purely mob-side; players
  manage their own equipment manually.
- **Equipment-upgrade event broadcasts.** The room broadcast
  text is the only signal — no separate `MobUpgraded` event
  for other systems to consume.
- **Auto-sell-junk integration.** Wild mobs accumulate
  displaced items in their backpack. We don't add an auto-
  sell path. Falls under chunks 5.3 (equipment-aware
  shopping) / 5.4 (NPC market participation).
- **Per-mutation gating beyond chunk 2.2a.** The Incorporeal
  rank-4 case is handled via itemvalue scoring. Other
  mutations (e.g., a hypothetical "Berserker" that should
  refuse to swap weapons mid-combat) are not specially
  gated. Add later as needed.

## Roadmap touchpoints

This chunk:

- Closes chunk **2.3** on `MOB_ALIVENESS_ROADMAP.md`. Roll-up
  moves from 10/41 → 11/41.
- Consumes chunks 2.2 (item-comparison primitive) and 2.2a
  (Incorporeal mutation) — both shipped earlier in the
  aliveness effort.
- Unblocks future chunks that depend on "mobs intelligently
  equip gear": chunk 5.3 (equipment-aware shopping) will reuse
  `CanEquipFromGive` / `CanScanFloorLoot` for the shopping
  decision flow.
