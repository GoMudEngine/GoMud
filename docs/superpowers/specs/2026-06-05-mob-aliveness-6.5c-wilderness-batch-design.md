# Mob Aliveness 6.5c — Wilderness Batch (light)

**Status:** Design approved — pending spec review → implementation plan
**Roadmap chunk:** 6.5c (wilderness/dungeon batch of the 6.5 content pass), Size S
**Date:** 2026-06-05
**Depends on:** 6.5a (factions — done; faction-tagging of wilderness mobs already shipped there)

## Goal

Apply a *light* aliveness pass to the wilderness/dungeon zones — pack/kin
relationships for the social mob groups and a handful of gossip-able facts for the
accessible sentient mobs. No schedules, no conversations, no behavior-archetype
changes (predator/forager/patrol archetypes are already set).

## Why this is small

Most wilderness zones are pure beasts or empty:
- a_dark_forest (0 mobs), the_fernway (0 mobs) — empty (rooms only).
- endless_trashheap (1: loot_goblin) — nothing to do.
- stillwater_marsh (6) / the_fernway_south (7) — beast scatter + a single forager NPC each.
- ironwind_steppe (44) — beast scatter + a wolf pack + a goblin tribe + sentient NPCs.
- labyrinth_of_low_tunnels (7) — the warren + cave beasts.

Faction tags were already applied world-wide in 6.5a. Beasts don't hold meaningful
relationships or gossip. So the genuine content is narrow: pack/kin bonds for the
three social groups + a few facts the sentient mobs can voice. This is data-only
(YAML), validated at boot.

## 1. Pack/kin relationships

Use `type: family` for pack/tribe/colony bonds — this feeds chunk-4.5 kin-revenge
(killing a member seeds revenge in surviving kin) and gives packs cohesion.
Author each edge ONCE (engine auto-mirrors symmetric same-type). Add a
`relationships:` block to the leader mob of each group:

**Ironwind wolf pack** — on `215-alpha_wolf.yaml`:
```yaml
relationships:
  - to: 205
    type: family
    subtype: pack
  - to: 206
    type: family
    subtype: offspring
  - to: 223
    type: family
    subtype: pack
```

**Ironwind goblin tribe** — on `219-goblin_shaman.yaml`:
```yaml
relationships:
  - to: 217
    type: family
    subtype: tribe
  - to: 218
    type: family
    subtype: tribe
  - to: 222
    type: family
    subtype: tribe
```

**Labyrinth warren** — on `75-warren_chieftain.yaml`:
```yaml
relationships:
  - to: 74
    type: friend
    subtype: council
  - to: 72
    type: family
    subtype: warren
  - to: 73
    type: family
    subtype: warren
```

## 2. Facts + gossipers

Four zone facts, appended to `_datafiles/world/dogmud/facts.yaml`. Gossiped only by
the *accessible* sentient mobs — the hostile packs/tribe never idle-gossip to
players, and the warren only surface theirs when at peace with the player (their
−25 default_rep means a low-rep player meets hostility, not chatter; a warm-rep
player — e.g. post Warren Compact — meets idle, gossiping warren). This rep-gating
falls out of the existing hostility/idle gossip gate; no new mechanism needed.

| fact id | description (public/observable) | gossipers (`knows_facts` + `gossiper` tag) |
|---|---|---|
| `ironwind-tribe-pressing` | The goblins of the steppe have grown bolder, pushing toward the trails and waterholes. | Halix 372, hermit_kael 240 |
| `ironwind-steppe-drying` | Water and game grow scarce on the steppe; every creature on it is pressed and hungry. | Halix 372, hermit_kael 240 |
| `warren-misjudged` | The surface folk fear the warren for no cause but their strangeness. | warren_chieftain 75, tunnel_shaman 74 |
| `fernway-wolves-ranging` | The timber wolves range closer to the forest paths than they used to. | Kessa 373 |

Tags: `ironwind-*` → `[ironwind_steppe, crisis]` (+ `bandits`/`ironwind_tribe`? use
`ironwind_tribe` tag on the tribe one); `warren-misjudged` → `[labyrinth, people]`;
`fernway-wolves-ranging` → `[the_fernway_south, wildlife]`.

`ironwind-tribe-pressing` cross-ties to the 6.5a `ironwind_tribe` faction;
`warren-misjudged` gives the 6.5a warren reframing an in-world voice;
`fernway-wolves-ranging` is a sibling to the Ashwick `ashwick-deep-woods-wolves`
fact from 6.5b (same regional wolf pressure).

## 3. Behavior tuning

None. The predator/forager/patrol/leader archetypes on these mobs are already set
(6.x). This batch adds no archetype or behavior changes.

## Validation

- **Boot test (hard gate):** the relationships loader panics on an unknown `to:`
  mob id; the facts loader + `knows_facts:` validation panics on unresolved fact
  ids. A clean boot past data-file load is the gate.
- **In-game smoke** (may be deferred to user): kill a wolf/goblin/warren member and
  confirm surviving kin react (4.5 kin-revenge); approach Halix/Kael/Kessa (and a
  warm-rep warren) and confirm the facts surface as gossip.

## Out of scope

- Empty zones (a_dark_forest, the_fernway) and pure-beast scatter — nothing to add.
- Schedules / conversations (wilderness has no townsfolk social hubs).
- Behavior-archetype changes.
- Roads/travel zones — that's 6.5d.
- Quest-boss mobs in ironwind (windwarden_sylara, geomancer_rhett) — leave as-is.

## Files touched (anticipated)

- Edit: `215-alpha_wolf.yaml`, `219-goblin_shaman.yaml`, `75-warren_chieftain.yaml`
  (relationships).
- Edit: `facts.yaml` (+4 facts).
- Edit: `372-halix.yaml`, `240-hermit_kael.yaml`, `373-kessa.yaml`,
  `75-warren_chieftain.yaml`, `74-tunnel_shaman.yaml` (`knows_facts:` + `gossiper`
  group tag).
- Edit: roadmap (6.5c status).
