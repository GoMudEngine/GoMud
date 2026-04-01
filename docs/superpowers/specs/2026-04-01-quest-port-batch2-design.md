# Quest Port Batch 2 — Item Delivery Quests Design Spec

## Overview

Port Quests 2, 7, and 15 to the quest engine. These are item delivery
quests where players obtain an item and give it to an NPC. The JS
`onGive` scripts that currently handle delivery are replaced by quest
engine `item_give` triggers with `npc_say` actions.

## Pre-requisite: Legacy Alchemy Item Cleanup

The healing poultice (item 30010) and stamina draught (item 30011)
were replaced by healing salve (30036) and stamina tonic (30037) in
the alchemy rework but never fully removed. Quest 2 references the
poultice — must switch to salve before porting.

**Items to remove:**
- `items/consumables-30000/30010-healing_poultice.yaml`
- `items/consumables-30000/30011-stamina_draught.yaml`

**Recipes to remove:**
- `recipes/alchemy/healing-poultice.yaml`
- `recipes/alchemy/stamina-draught.yaml`
- `recipes/alchemy/greater-healing-poultice.yaml`

**Help templates to remove:**
- `templates/help/healing-poultice.template`
- `templates/help/stamina-draught.template`
- `templates/help/greater-healing-poultice.template`

**Shop listings to update:**
- Alchemist Yenna (mob 53): replace item 30010 with 30036 in shop
- Apothecary Voss (mob 98): replace healing-poultice and
  stamina-draught in knownrecipes with healing-salve and stamina-tonic

**Player migration (`MigrateLegacyPotions`):**
- Convert any 30010 items in inventory/equipment to 30036
- Convert any 30011 items in inventory/equipment to 30037
- Convert `healing-poultice` recipe knowledge to `healing-salve`
- Convert `stamina-draught` recipe knowledge to `stamina-tonic`
- Remove `greater-healing-poultice` recipe knowledge

**Quest 2 dialogue update:**
- All references to "healing poultice" in Shaman's dialogue →
  "healing salve"
- `requiresItem: 30010` → `requiresItem: 30036`
- `item_give` trigger item filter: 30010 → 30036

## Quests

### Quest 2: The Warren Compact

**NPCs:** Tunnel Shaman (mob 74, room 301), Warren Chieftain (mob 75,
room 317)

**Current steps:** start → poultices → audience → end
**New steps:** `[start, end]` — collapse dead steps. The salve
delivery and chieftain audience are one continuous flow handled by
dialogue and triggers.

**Flow:**
1. Player asks Shaman about proving peace → dialogue node `prove_intent`
   grants `2-start`
2. Player obtains healing salve (item 30036) from Alchemist Yenna or
   crafts it
3. Player gives salve to Shaman → quest engine `item_give` trigger
   fires, grants `2-end` with `npc_say` response
4. Alternatively: player uses `ask shaman salve` → dialogue node
   `salve_delivery` fires (has `requiresItem: 30010`,
   `grantsQuest: "2-end"`)

**Item delivery trigger (replaces JS):**
```yaml
- event: item_give
  mob: 74
  item: 30010
  conditions:
    has: [2-start]
    missing: [2-end]
  actions:
    - grant: 2-end
    - npc_say:
        mob: 74
        lines:
          - {delay: 1, text: "These are good. Strong. Our sick will
              benefit."}
          - {delay: 3, text: "You have done what I asked. The Eldest
              will hear of this."}
```

**Dialogue changes:**
- `poultice_delivery` node: rename to `salve_delivery`, change
  `requiresItem: 30010` to `requiresItem: 30036`, change
  `grantsQuest: "2-poultices"` to `grantsQuest: "2-end"`. Update
  trigger words from "poultice" to "salve".
- Root variant for `2-poultices` → change to `2-start` (since
  `poultices` step no longer exists). Update text to match.
  Change "poultice" references to "salve".
- Root variant for `2-end` stays as-is.
- Chieftain (mob 75) `audience` node: change `grantsQuest: "2-end"`
  to remove it — the Shaman delivery already grants `2-end`. Keep
  the dialogue text as flavor for players who visit after completing.

**Hints:**
- `start`: "Obtain a healing salve and bring it to the Tunnel
  Shaman. From the Labyrinth entrance, go down 1 to the Low Junction.
  Poultices can be bought from Alchemist Yenna in Sanctum Basin or
  crafted with the recipe."
- `end`: (completed, no hint needed)

**JS disabled:** Rename `74-tunnel_shaman.js` → `.js.bak`

### Quest 7: The Fallow Field

**NPC:** Farmer Dorn (mob 89, room 442)

**Current steps:** start → pests → evidence → end
**New steps:** `[start, end]` — collapse dead `pests` and `evidence`

**Flow:**
1. Player asks Dorn about pests → dialogue node `help_quest` grants
   `7-start`
2. Player goes east 1 to fallow field (443), south 1 to pest fields
   (444), kills crop pests
3. Crop pest (mob 91) drops pest sample (item 24)
4. Player gives sample to Dorn → quest engine `item_give` trigger
   fires, grants `7-end` with `npc_say`
5. Alternatively: `ask dorn sample` → dialogue node `sample_return`
   (has `requiresItem: 24`, `grantsQuest: "7-end"`)

**Item delivery trigger (replaces JS):**
```yaml
- event: item_give
  mob: 89
  item: 24
  conditions:
    has: [7-start]
    missing: [7-end]
  actions:
    - grant: 7-end
    - npc_say:
        mob: 89
        lines:
          - {delay: 1, text: "Too big. This is not natural size."}
          - {delay: 3, text: "Something in the fallow soil is feeding
              them. As long as that field sits empty, this will keep
              happening."}
          - {delay: 5, text: "You have done what you can. The rest is
              politics."}
```

**Dialogue changes:**
- `sample_return` node: change `questRequired: ["7-start"]` (already
  correct), change `questExcluded` to `["7-end"]` (already correct).
  Change `grantsQuest` from `"7-end"` — actually this is already
  correct, it skips the dead intermediate steps.
- Root variant for `7-start`: already correct ("pest fields are south")
- Remove root variant references to dead steps if any exist.

**Hints:**
- `start`: "Kill crop pests and collect a sample for Farmer Dorn. From
  Dorn's farm, go east 1 to the fallow field, then south 1 to the pest
  fields. Kill pests until one drops a sample, then bring it back."
- `end`: (completed)

**JS disabled:** Rename `89-farmer_dorn.js` → `.js.bak`

### Quest 15: The Peddler's Overdue Freight

**NPC:** Peddler Malk (mob 250, room 4003)

**Steps:** `[start, end]` (unchanged)

**Flow:**
1. Player asks Malk about quest → dialogue node `quest_start` grants
   `15-start`
2. Player heads south and west to ruined barn (room 4007), finds
   freight crate (item 40039) in the crates container
3. Player gives crate to Malk → quest engine `item_give` trigger
   fires, grants `15-end` with `npc_say`
4. Alternatively: `ask malk crate` → dialogue node
   `quest_complete_dialogue` (has `requiresItem: 40039`)
5. **Edge case:** If player finds crate before talking to Malk and
   gives it, a separate `item_give` trigger with `missing: [15-start]`
   grants `15-start` first, then chains via `quest_granted` to grant
   `15-end`

**Item delivery triggers (replace JS):**
```yaml
# Normal completion
- event: item_give
  mob: 250
  item: 40039
  conditions:
    has: [15-start]
    missing: [15-end]
  actions:
    - grant: 15-end
    - npc_say:
        mob: 250
        lines:
          - {delay: 1, text: "That is the one! The iron straps, the
              stencil marks -- that is my consignment."}
          - {delay: 3, text: "I owe you for this. Road prices from me,
              any time. Better than road prices."}

# Pre-discovery: player found crate before talking to Malk
- event: item_give
  mob: 250
  item: 40039
  conditions:
    missing: [15-start]
  actions:
    - grant: 15-start
    - npc_say:
        mob: 250
        lines:
          - {delay: 1, text: "My freight! Where did you -- never mind,
              I can see the lock has been forced. Bandits."}
          - {delay: 3, text: "You have done me a considerable favor.
              Road prices from me, any time."}

# Chain: after pre-discovery grants 15-start, auto-complete
- event: quest_granted
  quest_token: "15-start"
  conditions:
    missing: [15-end]
  actions:
    - grant: 15-end
```

**Hints:**
- `start`: "Find Malk's missing freight crate. From his camp, go east
  1 to the Waypoint Shrine, then south along the road. The bandits
  have been seen near a ruined barn west of the farmland stretch."
- `end`: (completed)

**JS disabled:** Rename `250-peddler_malk.js` → `.js.bak`

## SOP Compliance Checklist

For each quest:
- [ ] `ask <npc> quest/task` triggers exist on all quest-granting nodes
- [ ] `questExcluded` prevents double-completion on all `grantsQuest` nodes
- [ ] All `text` fields are pure NPC speech (first person)
- [ ] All `hints` fields are narrator voice (third person, bracketed)
- [ ] Quest step `hint:` fields give explicit cardinal directions
- [ ] Trigger words in hints are discoverable from NPC text or prior hints
- [ ] Quest items do NOT have `is_component: true`
- [ ] `item_give` triggers handle the `give <item> <npc>` elephant path
- [ ] Dialogue `requiresItem` handles the `ask <npc> <topic>` path
- [ ] Both paths grant the same quest token
- [ ] Players who return to quest giver before completing get a "not yet"
  response (root variant with `questRequired` + `questExcluded`)
- [ ] All text wraps at 78-80 chars per line

## Files Changed

| Action | File | Purpose |
|--------|------|---------|
| MODIFY | `quests/2-the_warren_compact.yaml` | Collapse steps, add hints, add triggers |
| MODIFY | `quests/7-the_fallow_field.yaml` | Collapse steps, add hints, add triggers |
| MODIFY | `quests/15-the_peddlers_overdue_freight.yaml` | Add hints, add triggers |
| MODIFY | `dialogue/labyrinth_of_low_tunnels/74.yaml` | Fix grantsQuest refs for collapsed steps |
| MODIFY | `dialogue/labyrinth_of_low_tunnels/75.yaml` | Remove quest grant from audience node |
| VERIFY | `dialogue/thornwall_outskirts/89.yaml` | SOP audit only, likely no changes |
| VERIFY | `dialogue/marches_spur_road/250.yaml` | SOP audit only, likely no changes |
| DISABLE | `mobs/.../scripts/74-tunnel_shaman.js` | Rename to .js.bak |
| DISABLE | `mobs/.../scripts/89-farmer_dorn.js` | Rename to .js.bak |
| DISABLE | `mobs/.../scripts/250-peddler_malk.js` | Rename to .js.bak |

## Testing

For each quest:
1. New character → ask NPC about quest → confirm start banner
2. `hint` → confirm directions are correct and specific
3. Obtain quest item → `give <item> <npc>` → confirm `npc_say` fires
   and quest completes
4. `hint` → confirm completed state
5. Restart server → verify quest state persists
6. Test `ask <npc> quest` after completion → confirm "already done"

Quest 15 extra: find crate before talking to Malk → give crate →
confirm both start and end fire.
