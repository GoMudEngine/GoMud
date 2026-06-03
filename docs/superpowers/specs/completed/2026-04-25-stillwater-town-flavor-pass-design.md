# Stillwater Town-Flavor Pass — Design

**Date:** 2026-04-25
**Status:** Approved (brainstorming complete, ready for implementation plan)

## Goal

Give the 17 existing Stillwater NPCs without dialogue + 1 new caravan-son NPC enough conversational depth and idle texture to make the town feel inhabited rather than populated. Bake the caravan-on-layover atmosphere into the existing caravan crew (Ketil, Marta) plus the new son (Lars) so that when the caravan-system spec lands later, the foreshadowing is already in place.

## Out of scope

- Any actual caravan movement, scheduling, or restock-trigger mechanics (deferred to a separate caravan-system spec)
- New mobs other than Lars (mob 356)
- New items
- Room edits beyond Lars's spawninfo
- Combat balance tuning on caravan crew (system spec will handle)
- Gossip cadence config tuning (logged for future polish)
- Any quest hooks or new quest content
- Foreshadowing of any zone 2.3+ content

## Scope and file inventory

**New files (19 total: 1 mob + 18 dialogue):**

- `mobs/stillwater/356-lars_ketilson.yaml` — new caravan-son NPC (1)
- `dialogue/stillwater/356.yaml` — Lars's dialogue (1)
- `dialogue/stillwater/{17 mob ids}.yaml` — one new dialogue file per existing NPC missing one (17): Sigrid 333, Neva 334, Tov Brann 336, Ilsa 338, Edda 339, Kess 340, Wulf 341, Hodder 343, Finn 345, Luc 346, Bram 348, Gyda 349, Pip 350, Ketil 351, Marta 352, Fenwick 353, Oswin 354

**Modified files (~9):**

- 3 mob YAMLs get richer idlecommands: `351-caravan_driver_ketil.yaml`, `352-caravan_guard_marta.yaml`, `356-lars_ketilson.yaml` (the new file with caravan-layover idle baked in from creation)
- 1 mob YAML gets `groups: gossiper` addition: `334-barmaid_neva.yaml`
- 1 room YAML gets spawninfo for Lars: `rooms/stillwater/4102.yaml`
- 6 existing dialogue files get small cross-reference extensions (~5-10 new lines each, no rewrites): `335.yaml` Drunn, `342.yaml` Arn, `337.yaml` Brindle, `344.yaml` Seren, `355.yaml` Vella, `347.yaml` Ulla

## Implementation order

1. **Direct work (Claude main):**
   - Create Lars mob YAML + dialogue file
   - Add Lars spawninfo to 4102
   - Write 3 HEAVY-tier dialogue files (Sigrid 333, Ketil 351, Lars 356)
   - Expand idlecommands on Ketil + Marta with caravan-layover lines
   - Add `groups: gossiper` to Neva
   - Apply 6 cross-quest extensions to existing dialogue files
2. **Subagent dispatch:** 9 MEDIUM dialogues + 6 LIGHT dialogues with templates and per-NPC topic spine
3. Build check; spot-verify subagent output for voice consistency; restart guidance

## Lars Ketilson — full character

- **Name:** Lars Ketilson; mob name `lars`; introduces himself with patronymic when asked.
- **Age:** mid-teens, mock-slouching but actually attentive.
- **Role:** apprentice on the Stillwater↔Thornwall route; the one his father makes do all the in-town deliveries and errand-running on layover. Wants the next step up the ladder; doesn't whine about it, deflects with humor.
- **Personality:** jokey/loose, quick to make a small joke at his father's expense (out of his father's hearing), respectful when it counts. Picks up gossip the way teens do — by being underestimated and overhearing.

**Mob YAML profile (key fields):**

```yaml
mobid: 356
zone: Stillwater
behavior_archetype: noncombat_passive
statpool: 70
hostile: false
charm_immune: true
maxwander: 3        # runs errands around town center
groups:
  - humanoid
  - traveler
archetype: ""
character:
  name: lars
  speciesid: 1
  gold: 12
  stats:
    dexterity: { training: 10 }
    perception: { training: 12 }
    charisma:   { training: 6  }
```

**Spawn:** 4102 Lakefront Square (with Ketil + Marta on layover), `cooldown: 600 rounds`. With `maxwander: 3` he occasionally drifts to nearby rooms (4101 South Approach, 4103 Pike & Lantern interior, 4108 Crier's Step) — visually reinforces the errand-running role without him being missing for long.

## Caravan-layover framing (Ketil, Marta, Lars)

The three caravan members read as "camped in town between runs." Their idle/dialogue all reflect that they:

- Just came in from Thornwall (1-3 days ago, vague — "we got in Tuesday" type)
- Are leaving again soon ("morning after next, weather holding")
- Are road-tired but not road-traumatized — bandits got handled
- Have road stories they tell on each other
- Talk about their counterparts ("the smith's wife in Thornwall sends her best") and the road conditions ("ditch washed out at the second crossing")

**Key constraint: no caravan SYSTEM mechanics in any line.** All references are atmospheric. The future system spec must be able to plug in real timing/triggers without these dialogues feeling stale or contradicted.

## Dialogue depth tiers

### HEAVY (3 NPCs, ~90-110 lines each)

True social hubs / new characters with arcs.

**Sigrid (333)** — innkeeper, town nerve center.

- Patterns (~7): greetings (mood-aware) · ale/wine/menu · room/lodging/bed · caravan/Ketil/Lars/Marta · bandits/road news · catch-all
- Tree root variants:
  - Post 20-end (any flag): "Ulla came in for a drink last week. First time in years. That's your doing."
  - Post 19-end (full): "Drunn told me you put the deep-water thing down. Drinks are on the house tonight."
  - Default: "Sit anywhere. Common room's quiet this hour."
- Tree nodes (info, no quest grants):
  - `lake_decline` — catch is shorter this year, but Tov says nets are mending now
  - `caravan_news` — Ketil's crew is on layover, brought letters and goods from Thornwall
  - `voss_delicate` — Ulla and Vella, kept brief and respectful; available only post 20-end
  - `town_history` — Pike & Lantern was Sigrid's mother's place
  - `gossip_drunn` — the constable is a fixture, slightly understaffed but capable
  - `lodging_rates` — describes the upstairs loft and its cost (no actual mechanic)

**Ketil (351)** — caravan driver, anchors caravan canon.

- Patterns (~6): greetings (road-weary register) · route/Thornwall/road · wagons/horses/hub · weather/schedule · caravan crew · catch-all
- Tree root variants:
  - Post 20-end: "I knew Voss in passing — Maren's uncle, ran a few things north for him. Glad it's settled."
  - Default: "Welcome. Mind the wagon ruts."
- Tree nodes:
  - `route_thornwall` — the run, schedule (vague), why the caravan is necessary
  - `road_bandits` — there used to be a serious problem on the North Road; quieter now
  - `lars_kid` — paternal grumble about Lars
  - `marta_pro` — quiet respect for Marta
  - `maren_letter` — sometimes carries letters between Maren in Thornwall and Ulla here
  - `shopkeepers_know_him` — knows what each shopkeeper orders without checking the slate (sets up future delivery mechanic implicitly)

**Lars (356)** — already detailed above.

- Same template structure as the other two heavy-tier files — patterns, tree root variants, info nodes for: route stories, his father, the slate-list of orders, what Thornwall's like, the joke about being the kid who runs notes, what he wants to be doing.

### MEDIUM (9 NPCs, ~50 lines each)

**Structure:** 5 patterns (greetings mood-aware + 3 topic + catch-all) · tree.root with default + 1 cross-quest variant · 2-3 info nodes · `memory.expiryPeriod: ""`

**Per-NPC topic spine:**

| NPC | Mob | Trade topic | Personal/lore topic | Cross-reference target |
|-----|-----|-------------|---------------------|------------------------|
| Tov Brann | 336 | clams/fish/the catch | the lake's lean year | Drunn (bounty), Hodder (mended nets) |
| Ilsa | 338 | tinctures/willow-bark | which fishermen she's patched up | Vella (her senior counterpart at the cottage) |
| Edda | 339 | cattail/weaving | not being Maren-of-Thornwall | Maren (Thornwall), Ulla (loom-sharing) |
| Kess | 340 | pearls/silverwork | the divers who used to bring stillwater pearls | Vella (medicinal pearls), Tov (fishermen who dive) |
| Wulf | 341 | salvage kits/lanterns | knows everyone's order | Ketil (caravan stock arrivals) |
| Bram | 348 | wheels/grain pickup | the seasonal flood pattern | Drunn (mill-bridge maintenance), Edda (cattail harvest) |
| Hodder | 343 | net-mending/forty years | names old fishermen no longer alive | Old Voss (delicately — knew Elgar from the docks), Tov (apprentice generation) |
| Gyda | 349 | porch-sitting | town in her grandmother's day | Sigrid (Pike & Lantern history), Vella (deliveries to elders) |
| Fenwick | 353 | the road | news from Sanctum Basin/Thornwall/elsewhere | Maren (Thornwall waystop), Ketil (road conditions exchange) |

### LIGHT (6 NPCs, ~20 lines each)

**Structure:** 3 patterns (greeting + 1 character topic + catch-all) · tree.root with single default text · NO tree.nodes · `memory.expiryPeriod: ""`

**Per-NPC topic:**

| NPC | Mob | One personality beat |
|-----|-----|----------------------|
| Neva | 334 | barmaid quips, gossiper-tagged so engine carries her ambient news |
| Finn | 345 | temple acolyte, devoted, mentions Seren respectfully |
| Luc | 346 | young fisherman, ambitious, watches Hodder for technique |
| Pip | 350 | the kid, wants to be a fisherman like his da |
| Marta | 352 | caravan guard, terse, oils her sword, "the road's been quieter than last summer" |
| Oswin | 354 | beggar, ex-fisherman, heard a thing about the boat that sank in '08 |

## Gossiper expansion

Currently 2 gossipers (Hodder 343, Gyda 349). Adding 1: **Neva (334)** at the inn.

Total: 3 gossipers — Hodder (Net Yard 4117), Gyda (Cooper's Lane 4132), Neva (Pike & Lantern 4103). Coverage spans dock, residential, and inn — three distinct social contexts.

Future: if gossip cadence proves too chatty after the world-event system has more feeders, EITHER bump `GossipIntervalRounds` upward in `config.yaml` OR add 2-3 more gossipers to thin per-mob frequency.

## Cross-quest extensions

Each of the 6 already-written dialogue files gets a small addition (1 new node or pattern, ~5-10 lines) — NOT a rewrite.

| File | Mob | Addition |
|------|-----|----------|
| `335.yaml` | Drunn | New pattern on `caravan/ketil/road/north`: brief acknowledgment that the caravan crew is in town, the road's been quieter since Soren's lot was cleared, Drunn checks in on Ketil when they're through |
| `342.yaml` | Arn | New pattern on `caravan/ketil`: Arn and Ketil exchange weather and lake-condition reports; Arn's docks load some Thornwall freight onto the wagons |
| `337.yaml` | Brindle | New node `caravan_orders` (post-quest 19/20 friendly mood): Brindle takes lake-iron stock from Ketil's crew, says Lars is a good kid even if he writes orders down wrong half the time |
| `344.yaml` | Seren | New pattern on `caravan/road`: priest Seren says little but mentions Ketil keeps a small donation envelope for the temple each run |
| `355.yaml` | Vella | Post-quest 20 only: a single new line acknowledging the kingfisher is on Ulla's mantel now, Vella visited her last week |
| `347.yaml` | Ulla | Post-quest 20 only: a single new line — "I went to the Pike & Lantern for the first time in a long while. Sigrid kept the old seat for me." |

## Lore beats to plant

These thread through multiple NPCs (subagent brief spells out which NPCs mention which beats):

1. **Lake decline (background)** — catch is shorter than it was, but bounty resolution + caves cleared means nets are mending. Carriers: Tov, Hodder, Sigrid, Bram, Pip.
2. **Post-Voss town aftermath** — Ulla left the cottage. Sigrid kept her seat. Vella visits her now. Carriers: Sigrid (heavy), Vella (extension), Gyda (porch gossip), Edda (loom solidarity).
3. **Maren-in-Thornwall thread** — Maren is canon (mob 113); writes occasionally; Elgar's niece. Carriers: Edda, Ketil (carries letters), Sigrid (a room kept for her), Ulla (existing).
4. **Caravan foreshadowing without commitment** — caravan brings goods; caravan gets ambushed sometimes; Marta saved the wagons twice. Carriers: Ketil (heavy), Lars (heavy), Marta (light), Wulf, Brindle (extension).
5. **Pre-Chrysalis mystery left mysterious** — spiral is older than the order; nobody mentions it post-quest 20. Sealed for future content. **NO carriers** — players who finished quest 20 own the mystery; no NPCs dilute it with speculation.

## Cross-reference web

```
                       ┌─────── Sigrid ───────┐
                       │ (inn, town hub)     │
        ┌──────────────┼──────────────────────┤
        │              │                      │
      Drunn          Hodder                  Tov ────── Drunn (bounty)
      (caravan)      (lake old-timer)         │
        │              │                      Kess ──── Vella (pearls)
        │              │                                  │
       Ketil ─── Lars  Gyda ─── Sigrid                  Ilsa ── Vella
       (caravan)│      (porch)  (mom's history)          │
                Marta                                    Bram ──── Drunn
                │                                              (mill bridge)
                │
                Wulf ─── Ketil (caravan stock)

       Edda ── Maren (Thornwall) ── Ketil (letters) ── Ulla
                │                                       │
                Ulla (loom)                            Vella

       Fenwick ── Maren / Ketil (road news)

       Pip ── Hodder (kid watches fishermen)
       Luc ── Hodder (young vs old fisherman)
       Finn ── Seren
       Neva ── Sigrid (gossiper at the same inn)
       Oswin ── square-fixture, no specific tie
```

Density check: every MEDIUM/HEAVY NPC has at least one outbound reference. LIGHT NPCs reference at least one MEDIUM/HEAVY (no isolated NPCs).

## Subagent dispatch brief — outline

The subagent will receive a single message containing:

1. **Goal & scope:** create 15 dialogue YAML files (9 MEDIUM + 6 LIGHT) at `_datafiles/world/dogmud/dialogue/stillwater/{mobid}.yaml`
2. **Templates:** literal YAML templates for MEDIUM and LIGHT (above) — voice fills in, structure is fixed
3. **Per-NPC brief:** the topic-spine table, plus existing mob descriptions to read for voice
4. **Voice canon:** lakefolk Norse register (Ulla, Vella, Maren are reference points); post-Voss aftermath baseline; caravan-on-layover atmosphere is fresh news everyone's heard
5. **Hard constraints (lessons memory + SOP):**
   - NPC text first-person, hints narrator-perspective
   - NO quest hooks (no `grantsQuest`, no quest tokens) — flavor only
   - NO `requires`, NO `expiryPeriod`
   - NO narrator overreach (no mind-reading dead/absent characters; no inventing details not in room/mob descriptions)
   - Catch-all `keywords: [""]` at end of every patterns list
   - Trigger discoverability: every keyword sourced from existing in-game text
6. **Verification:** `go build ./...` clean at end; report any ambiguity or canon contradictions found while reading source

## Verification plan

1. Restart server with all new files in place
2. For each of the 17 existing NPCs + Lars: walk to spawn room, `look <npc>`, then test 2-3 of their topic keywords
3. Confirm Lars spawns at 4102 and occasionally appears at neighboring rooms via maxwander
4. Confirm Neva broadcasts gossip lines after sufficient idle time
5. Confirm caravan-layover idlecommands fire on Ketil, Marta, Lars
6. Spot-test cross-quest extensions: `ask drunn caravan` (post-extension), `ask brindle caravan_orders` (post-quest 19/20), etc.
7. Walk through Stillwater post-quest 20 and confirm the "Ulla emerged" beat lands across Sigrid + Vella + Gyda + Edda
