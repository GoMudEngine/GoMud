# Quest Port Batch 2 — Item Delivery Quests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove legacy alchemy items (healing poultice 30010, stamina draught 30011), then port Quests 2, 7, and 15 to the quest engine with `item_give` triggers replacing JS `onGive` scripts.

**Architecture:** Legacy alchemy cleanup first (delete items/recipes/help files, update shop listings, add player migration). Then port each quest: collapse dead steps to `[start, end]`, add `item_give` triggers with `npc_say` actions to quest YAML, update dialogue references, disable JS scripts, add `hint:` fields with cardinal directions.

**Tech Stack:** YAML data files, Go (migration only), existing quest engine `item_give`/`npc_say`/`quest_granted` infrastructure.

---

### Task 1: Legacy Alchemy Cleanup — Remove Items, Recipes, Help Files

**Files:**
- Delete: `_datafiles/world/dogmud/items/consumables-30000/30010-healing_poultice.yaml`
- Delete: `_datafiles/world/dogmud/items/consumables-30000/30011-stamina_draught.yaml`
- Delete: `_datafiles/world/dogmud/recipes/alchemy/healing-poultice.yaml`
- Delete: `_datafiles/world/dogmud/recipes/alchemy/stamina-draught.yaml`
- Delete: `_datafiles/world/dogmud/recipes/alchemy/greater-healing-poultice.yaml`
- Delete: `_datafiles/world/dogmud/items/consumables-30000/30031-greater_healing_poultice.yaml`
- Delete: `_datafiles/world/dogmud/templates/help/healing-poultice.template`
- Delete: `_datafiles/world/dogmud/templates/help/stamina-draught.template`
- Delete: `_datafiles/world/dogmud/templates/help/greater-healing-poultice.template`
- Modify: `_datafiles/world/dogmud/mobs/sanctum_basin/53-alchemist_yenna.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/98-apothecary_voss.yaml`

- [ ] **Step 1: Delete legacy item definitions**

```bash
rm _datafiles/world/dogmud/items/consumables-30000/30010-healing_poultice.yaml
rm _datafiles/world/dogmud/items/consumables-30000/30011-stamina_draught.yaml
rm _datafiles/world/dogmud/items/consumables-30000/30031-greater_healing_poultice.yaml
```

- [ ] **Step 2: Delete legacy recipes**

```bash
rm _datafiles/world/dogmud/recipes/alchemy/healing-poultice.yaml
rm _datafiles/world/dogmud/recipes/alchemy/stamina-draught.yaml
rm _datafiles/world/dogmud/recipes/alchemy/greater-healing-poultice.yaml
```

- [ ] **Step 3: Delete legacy help templates**

```bash
rm _datafiles/world/dogmud/templates/help/healing-poultice.template
rm _datafiles/world/dogmud/templates/help/stamina-draught.template
rm _datafiles/world/dogmud/templates/help/greater-healing-poultice.template
```

- [ ] **Step 4: Update Alchemist Yenna's shop**

Read `_datafiles/world/dogmud/mobs/sanctum_basin/53-alchemist_yenna.yaml`. Find the shop entry with `itemid: 30010` and replace it with `itemid: 30036` (healing salve). The shop section is near the end of the file.

- [ ] **Step 5: Update Apothecary Voss's knownrecipes**

Read `_datafiles/world/dogmud/mobs/thornwall_city/98-apothecary_voss.yaml`. Find the `knownrecipes:` list. Replace `healing-poultice` with `healing-salve` and `stamina-draught` with `stamina-tonic`.

- [ ] **Step 6: Check for mob instance saves that override templates**

Check for instance saves for mobs 53 and 98. If they exist, delete them so the server loads fresh templates.

```bash
ls _datafiles/world/dogmud/mobs.instances/sanctum_basin/53-*
ls _datafiles/world/dogmud/mobs.instances/thornwall_city/98-*
```

Delete any found.

- [ ] **Step 7: Build and verify**

Run: `go build ./...`
Expected: Clean build. The server should start without panicking (item/recipe files are loaded dynamically — missing files just mean those items no longer exist).

- [ ] **Step 8: Commit**

```bash
git add -A _datafiles/world/dogmud/items/ _datafiles/world/dogmud/recipes/ _datafiles/world/dogmud/templates/help/ _datafiles/world/dogmud/mobs/sanctum_basin/53-alchemist_yenna.yaml _datafiles/world/dogmud/mobs/thornwall_city/98-apothecary_voss.yaml
git commit -m "chore: remove legacy healing poultice and stamina draught

Deleted items 30010, 30011, 30031 and their recipes and help files.
Updated Yenna's shop (30010 → 30036) and Voss's knownrecipes
(healing-poultice → healing-salve, stamina-draught → stamina-tonic).
These were replaced by healing salve and stamina tonic in the alchemy
rework but never fully cleaned up."
```

---

### Task 2: Player Migration — Convert Legacy Potions

**Files:**
- Modify: `internal/characters/character.go`
- Modify: `internal/users/users.go`

- [ ] **Step 1: Add MigrateLegacyPotions method**

In `internal/characters/character.go`, add after the existing `MigrateQuestFlags` method:

```go
// MigrateLegacyPotions replaces removed alchemy items and recipes
// with their current equivalents.
// 30010 (healing poultice) → 30036 (healing salve)
// 30011 (stamina draught)  → 30037 (stamina tonic)
func (c *Character) MigrateLegacyPotions() {
	// Replace items in backpack
	for i := range c.Items {
		switch c.Items[i].ItemId {
		case 30010:
			c.Items[i].ItemId = 30036
		case 30011:
			c.Items[i].ItemId = 30037
		case 30031:
			c.Items[i].ItemId = 30036
		}
	}

	// Replace items in component bag
	for i := range c.ComponentItems {
		switch c.ComponentItems[i].ItemId {
		case 30010:
			c.ComponentItems[i].ItemId = 30036
		case 30011:
			c.ComponentItems[i].ItemId = 30037
		case 30031:
			c.ComponentItems[i].ItemId = 30036
		}
	}

	// Replace items in potion bandolier
	for i := range c.PotionItems {
		switch c.PotionItems[i].ItemId {
		case 30010:
			c.PotionItems[i].ItemId = 30036
		case 30011:
			c.PotionItems[i].ItemId = 30037
		case 30031:
			c.PotionItems[i].ItemId = 30036
		}
	}

	// Replace recipe knowledge
	if c.KnownRecipes != nil {
		if _, ok := c.KnownRecipes["healing-poultice"]; ok {
			delete(c.KnownRecipes, "healing-poultice")
			c.KnownRecipes["healing-salve"] = 1
		}
		if _, ok := c.KnownRecipes["stamina-draught"]; ok {
			delete(c.KnownRecipes, "stamina-draught")
			c.KnownRecipes["stamina-tonic"] = 1
		}
		delete(c.KnownRecipes, "greater-healing-poultice")
	}
}
```

- [ ] **Step 2: Add migration call**

In `internal/users/users.go`, find the migration chain (after `MigrateQuestFlags`). Add:

```go
loadedUser.Character.MigrateLegacyPotions()
```

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 4: Commit**

```bash
git add internal/characters/character.go internal/users/users.go
git commit -m "feat: migration converts legacy potions to new equivalents

MigrateLegacyPotions replaces healing poultice (30010) → healing
salve (30036), stamina draught (30011) → stamina tonic (30037),
and greater healing poultice (30031) → healing salve in all
inventory slots and recipe knowledge."
```

---

### Task 3: Port Quest 2 — The Warren Compact

**Files:**
- Modify: `_datafiles/world/dogmud/quests/2-the_warren_compact.yaml`
- Modify: `_datafiles/world/dogmud/dialogue/labyrinth_of_low_tunnels/74.yaml`
- Modify: `_datafiles/world/dogmud/dialogue/labyrinth_of_low_tunnels/75.yaml`
- Disable: `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/scripts/74-tunnel_shaman.js`

- [ ] **Step 1: Update Quest 2 YAML — collapse steps, add hints and trigger**

Replace the full content of `_datafiles/world/dogmud/quests/2-the_warren_compact.yaml`:

```yaml
questid: 2
name: The Warren Compact
description: The tunnel shaman has asked you to prove your good
  intent by bringing a healing salve to the warren.
secret: false

steps:
  - id: start
    description: "The Tunnel Shaman has asked you to bring a healing
      salve as proof of good intent. Obtain one and bring it to the
      shaman in the Low Junction."
    hint: "Bring a healing salve to the Tunnel Shaman. From the
      Labyrinth entrance, go down 1 to the Low Junction. Salves can
      be bought from Alchemist Yenna in Sanctum Basin or crafted
      with the healing-salve recipe."
  - id: end
    description: "You have earned a fragile peace with the warren."

rewards:
  playermessage: "The chieftain inclines its heavy head -- the
    closest thing to a bow its mutated frame allows. You are not
    welcome here. But you are tolerated. That is more than any
    surface-dweller has been given in living memory."
  roommessage: "A low, guttural exchange passes between the shaman
    and the chieftain. Something has shifted."
  gold: 15
  skillinfo: "rhetoric:1"

triggers:
  # Item delivery via give command
  - event: item_give
    mob: 74
    item: 30036
    conditions:
      has: [2-start]
      missing: [2-end]
    actions:
      - grant: 2-end
      - npc_say:
          mob: 74
          lines:
            - {delay: 1, text: "These are good. Strong. Our sick
                will benefit."}
            - {delay: 3, text: "You have done what I asked. The
                Eldest will hear of this. Go deeper -- the warriors
                will not stop you now."}
```

- [ ] **Step 2: Update Shaman dialogue (mob 74) — salve references + step collapse**

Read `_datafiles/world/dogmud/dialogue/labyrinth_of_low_tunnels/74.yaml`. Make these changes:

1. In patterns, replace all "healing poultice" text with "healing salve" and "poultice" with "salve" in the help/quest/task pattern responses.

2. In root variants: change `questRequired: ["2-start"]` / `questExcluded: ["2-poultices"]` to `questExcluded: ["2-end"]`. Update "poultice" text to "salve".

3. Change root variant `questRequired: ["2-poultices"]` / `questExcluded: ["2-end"]` to `questRequired: ["2-end"]` — this becomes the post-completion variant (or remove it since the `2-end` variant already exists).

4. In `prove_intent` node: change "healing poultice" to "healing salve" in text and hints. Change trigger words to include "salve". Keep `grantsQuest: "2-start"`.

5. In `poultice_delivery` node: change id to `salve_delivery`. Change triggers to include "salve" instead of "poultice". Change `requiresItem: 30010` to `requiresItem: 30036`. Change `grantsQuest: "2-poultices"` to `grantsQuest: "2-end"`. Update text to use "salve". Update `questExcluded` to `["2-end"]`.

- [ ] **Step 3: Update Chieftain dialogue (mob 75) — remove quest grant**

Read `_datafiles/world/dogmud/dialogue/labyrinth_of_low_tunnels/75.yaml`. Make these changes:

1. In root variant: change `questRequired: ["2-poultices"]` to `questRequired: ["2-end"]` (since the `poultices` step no longer exists). The text about "brought medicine" stays — it's post-completion flavor now.

2. In `audience` node: remove `grantsQuest: "2-end"` (the Shaman delivery already grants it). Change `questRequired: ["2-poultices"]` to `questRequired: ["2-end"]`. Keep the text as post-completion dialogue. Remove `questExcluded: ["2-end"]` since this is now the completed state.

- [ ] **Step 4: Disable JS script**

```bash
mv _datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/scripts/74-tunnel_shaman.js \
   _datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/scripts/74-tunnel_shaman.js.bak
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`

- [ ] **Step 6: Commit**

```bash
git add -A _datafiles/world/dogmud/quests/2-the_warren_compact.yaml \
  _datafiles/world/dogmud/dialogue/labyrinth_of_low_tunnels/ \
  _datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/scripts/
git commit -m "feat: port Quest 2 (Warren Compact) — salve delivery via item_give trigger

Collapsed steps to [start, end]. Replaced healing poultice with
healing salve. Added item_give trigger with npc_say on Shaman.
Disabled JS onGive script. Added hint with directions."
```

---

### Task 4: Port Quest 7 — The Fallow Field

**Files:**
- Modify: `_datafiles/world/dogmud/quests/7-the_fallow_field.yaml`
- Verify: `_datafiles/world/dogmud/dialogue/thornwall_outskirts/89.yaml`
- Disable: `_datafiles/world/dogmud/mobs/thornwall_outskirts/scripts/89-farmer_dorn.js`

- [ ] **Step 1: Update Quest 7 YAML — collapse steps, add hints and trigger**

Replace the full content of `_datafiles/world/dogmud/quests/7-the_fallow_field.yaml`:

```yaml
questid: 7
name: The Fallow Field
description: Farmer Dorn's neighbour was evicted by the Thornwall
  Tax Authority, and the abandoned field is breeding pests that
  threaten the remaining farms. Kill the pests and bring Dorn a
  sample.
secret: false

steps:
  - id: start
    description: "Farmer Dorn has asked you to clear the crop pests
      from the south fields and bring back a sample."
    hint: "Kill crop pests and collect a sample for Farmer Dorn.
      From Dorn's farm, go east 1 to the fallow field, then south
      1 to the pest fields. Kill pests until one drops a sample,
      then bring it back to Dorn."
  - id: end
    description: "You reported back to Dorn. The pest problem is
      under control for now."

rewards:
  playermessage: "Dorn examines the pest sample with a grim
    expression, then sets it down carefully. 'Bigger than they
    should be. The fallow field is feeding them -- all that
    untended root growth, it is a paradise for vermin.' He looks
    toward Thornwall's walls. 'Gareth was a good farmer. The tax
    took everything. And now the rest of us pay for it in
    different coin.' He reaches into a pouch. 'This is not much,
    but it is honest.'"
  roommessage: "Dorn nods slowly, the weight of the situation
    evident in his weathered face."
  gold: 8
  itemid: 23

triggers:
  # Item delivery via give command
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
            - {delay: 1, text: "Too big. This is not natural
                size."}
            - {delay: 3, text: "Something in the fallow soil is
                feeding them. As long as that field sits empty,
                this will keep happening."}
            - {delay: 5, text: "You have done what you can. The
                rest is politics."}
```

- [ ] **Step 2: Verify Dorn dialogue SOP compliance**

Read `_datafiles/world/dogmud/dialogue/thornwall_outskirts/89.yaml`. Verify:
- `help_quest` node has "quest" and "task" in triggers ✅
- `sample_return` node has `questRequired: ["7-start"]` and `questExcluded: ["7-end"]` ✅
- `sample_return` has `requiresItem: 24` and `grantsQuest: "7-end"` ✅
- Root variants correctly gate on `7-start` / `7-end` ✅
- All text is NPC speech (first person) ✅
- Hints use narrator voice ✅

The existing dialogue already skips the dead intermediate steps (grants `7-end` directly from `7-start`). No dialogue changes needed unless SOP violations are found.

- [ ] **Step 3: Verify pest sample item**

Read `_datafiles/world/dogmud/items/other-0/24-pest_sample.yaml`. Confirm `is_component` is not set or is false.

- [ ] **Step 4: Disable JS script**

```bash
mv _datafiles/world/dogmud/mobs/thornwall_outskirts/scripts/89-farmer_dorn.js \
   _datafiles/world/dogmud/mobs/thornwall_outskirts/scripts/89-farmer_dorn.js.bak
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`

- [ ] **Step 6: Commit**

```bash
git add -A _datafiles/world/dogmud/quests/7-the_fallow_field.yaml \
  _datafiles/world/dogmud/mobs/thornwall_outskirts/scripts/
git commit -m "feat: port Quest 7 (Fallow Field) — pest sample delivery via item_give trigger

Collapsed steps to [start, end]. Added item_give trigger with
npc_say on Dorn. Disabled JS onGive script. Added hint with
directions to pest fields."
```

---

### Task 5: Port Quest 15 — The Peddler's Overdue Freight

**Files:**
- Modify: `_datafiles/world/dogmud/quests/15-the_peddlers_overdue_freight.yaml`
- Verify: `_datafiles/world/dogmud/dialogue/marches_spur_road/250.yaml`
- Disable: `_datafiles/world/dogmud/mobs/marches_spur_road/scripts/250-peddler_malk.js`

- [ ] **Step 1: Update Quest 15 YAML — add hints and triggers**

Replace the full content of `_datafiles/world/dogmud/quests/15-the_peddlers_overdue_freight.yaml`:

```yaml
questid: 15
name: The Peddler's Overdue Freight
description: A road peddler on the southern spur has been waiting
  on a freight shipment that never arrived. Bandits on the road
  may have intercepted it.
secret: false

steps:
  - id: start
    description: "Peddler Malk has been waiting on a freight
      shipment for days. He suspects bandits intercepted it. Find
      the missing freight crate."
    hint: "Find Malk's missing freight crate. From his camp, go
      east 1 to the Waypoint Shrine, then south along the road.
      The bandits have been seen near a ruined barn west of the
      farmland stretch."
  - id: end
    description: "You recovered Malk's freight crate and returned
      it to him."

rewards:
  playermessage: "Malk examines the crate, running his elongated
    fingers over the forced lock. 'This is the one. Everything
    accounted for, near enough.' He reaches into the cart and
    produces a small purse. 'Road rate from me, any time you
    need supplies. Better than road rate, in fact. I do not
    forget a good turn.'"
  roommessage: "Malk examines the recovered freight crate with
    evident relief, already tallying his inventory."
  gold: 15

triggers:
  # Normal completion: player has quest, delivers crate
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
            - {delay: 1, text: "That is the one! The iron straps,
                the stencil marks -- that is my consignment."}
            - {delay: 3, text: "I owe you for this. Road prices
                from me, any time. Better than road prices."}

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
            - {delay: 1, text: "My freight! Where did you -- never
                mind, I can see the lock has been forced. Bandits."}
            - {delay: 3, text: "You have done me a considerable
                favor. Road prices from me, any time."}

  # Chain: after pre-discovery grants 15-start, auto-complete
  - event: quest_granted
    quest_token: "15-start"
    conditions:
      missing: [15-end]
    actions:
      - grant: 15-end
```

- [ ] **Step 2: Verify Malk dialogue SOP compliance**

Read `_datafiles/world/dogmud/dialogue/marches_spur_road/250.yaml`. Verify:
- `quest_start` node has "quest" and "task" in triggers ✅
- `quest_complete_dialogue` has `requiresItem: 40039` and `grantsQuest: "15-end"` ✅
- `quest_active` shows "still waiting" when quest is active but not complete ✅
- `quest_done` handles post-completion ✅
- `lost_item` provides recovery hint ✅
- All text is NPC speech (first person) ✅

No dialogue changes expected.

- [ ] **Step 3: Verify freight crate item**

Read `_datafiles/world/dogmud/items/materials-40000/40039-freight_crate.yaml`. Confirm `is_component` is not set or is false.

- [ ] **Step 4: Disable JS script**

```bash
mv _datafiles/world/dogmud/mobs/marches_spur_road/scripts/250-peddler_malk.js \
   _datafiles/world/dogmud/mobs/marches_spur_road/scripts/250-peddler_malk.js.bak
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`

- [ ] **Step 6: Commit**

```bash
git add -A _datafiles/world/dogmud/quests/15-the_peddlers_overdue_freight.yaml \
  _datafiles/world/dogmud/mobs/marches_spur_road/scripts/
git commit -m "feat: port Quest 15 (Peddler's Freight) — crate delivery via item_give trigger

Added item_give triggers with npc_say on Malk. Pre-discovery edge
case: giving crate before talking to Malk grants both 15-start and
15-end via quest_granted chain. Disabled JS onGive script. Added
hint with directions to ruined barn."
```

---

### Task 6: Manual Test — All Three Quests

- [ ] **Step 1: Test Quest 2**

1. Restart server
2. New character (or use questtoken to clear Q2)
3. Travel to Tunnel Shaman (Labyrinth entrance → down 1)
4. `ask shaman quest` → should grant 2-start, mention healing salve
5. `hint` → should show directions to get a salve
6. Obtain a healing salve (buy from Yenna or craft)
7. `give salve shaman` → should see npc_say response + quest complete banner + 15 gold + rhetoric:1
8. `ask shaman quest` → should NOT re-grant

- [ ] **Step 2: Test Quest 7**

1. Travel to Farmer Dorn (Thornwall Outskirts, room 442)
2. `ask dorn quest` → should grant 7-start
3. `hint` → should show directions (east 1, south 1 to pest fields)
4. Go east, south to pest fields → kill crop pests → collect pest sample
5. Return to Dorn → `give sample dorn` → npc_say + complete + 8 gold + Thornwall Pass
6. Also test: `ask dorn sample` with sample in backpack → same result

- [ ] **Step 3: Test Quest 15**

1. Travel to Peddler Malk (Marches Spur Road, room 4003)
2. `ask malk quest` → should grant 15-start
3. `hint` → should show directions to ruined barn
4. Find freight crate at ruined barn → pick it up
5. Return to Malk → `give crate malk` → npc_say + complete + 15 gold

- [ ] **Step 4: Test Quest 15 pre-discovery edge case**

1. New character or clear Q15 progress
2. Go directly to ruined barn WITHOUT talking to Malk
3. Pick up freight crate
4. Go to Malk → `give crate malk` → should grant BOTH 15-start and 15-end

- [ ] **Step 5: Verify legacy potion migration**

1. Log in with an existing character who had healing poultice recipe
2. Check `recipes` command → should show healing-salve, not healing-poultice
3. If they had poultice items, check inventory → should be salves now
