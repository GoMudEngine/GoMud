# Rogue Lair Zone + Mechanics — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Chrysalis Undercity rogue zone (10 rooms), implement flee rework and defuse command, create lockpick/disarm kit items and recipes, update Siv's shop, and populate with chrysalis-themed mobs and loot.

**Architecture:** Global mechanics first (flee rework, defuse command), then items/recipes, then zone content (rooms, mobs, equipment). Zone extends Thornwall City at z:-3 below smuggler tunnels.

**Tech Stack:** Go, YAML data files, JS scripts (boss mob behavior)

---

## Task Ordering

1. Flee rework (Go — global mechanic)
2. Defuse command (Go — global mechanic)
3. Lockpick + disarm kit items (YAML)
4. Tool crafting recipes (YAML)
5. Siv + Whisper shop stock (YAML)
6. Zone equipment items (YAML — 6 items)
7. Zone rooms (YAML — 10 rooms)
8. Zone mobs (YAML + JS — 4 mobs)
9. Wire up room 487 entry + spawninfo
10. Help files + hints
11. Final verification

---

### Task 1: Flee Rework

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go`

- [ ] **Step 1: Read the existing `handlePlayerFlee` function**

Read `internal/hooks/NewRound_DoCombat_helpers.go` and find
`handlePlayerFlee`. The current formula at line ~409 is:

```go
chanceIn100 := int(float64(user.Dex) / (float64(user.Dex) + float64(mob.Dex)) * 70)
chanceIn100 += 30
```

Replace with an opposed roll using the shared `dice.OpposedRollStat`:

```go
// Flee: fleeer's Dex + Skullduggery vs blocker's Dex + Unarmed
fleeScore := user.Character.Stats.Dexterity.ValueAdj +
    user.Character.GetSkillLevel(skills.Skullduggery) * 25
blockScore := mob.Character.Stats.Dexterity.ValueAdj +
    mob.Character.GetCombatSkillLevel() * 25
success, _, _, _ := dice.OpposedRollStat(fleeScore, blockScore)
if !success {
    blockedByMob = mob.Character.Name
    break
}
```

Apply the same formula to the player-blocking-player check below
it (lines ~424-441).

Also read `internal/mobcommands/flee.go` — mob flee is instant
(no opposed roll). Update it to also use the opposed roll against
the mob's target. If the target is a player, roll flee vs the
player's Dex + unarmed. If the target is a mob, roll flee vs
that mob's Dex + unarmed.

- [ ] **Step 2: Add skills import if needed**

The hooks file needs `skills.Skullduggery` — check if already
imported, add if not.

- [ ] **Step 3: Verify build + test**

Run: `go build ./...`
Run: `go test ./internal/hooks/ -count=1`

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: flee rework — opposed roll Dex+skullduggery vs Dex+unarmed"
```

---

### Task 2: Defuse Command

**Files:**
- Modify: `internal/usercommands/skill.skullduggery.defuse.go`
- Modify: `internal/usercommands/usercommands.go` (if not registered)

- [ ] **Step 1: Read the existing defuse stub**

Read `internal/usercommands/skill.skullduggery.defuse.go`. It
currently returns "You don't detect any traps here." Wire it up.

- [ ] **Step 2: Implement defuse logic**

The defuse command should:
1. Check player has a disarm kit in inventory (search by
   `component_tag: disarm_kit` or item type)
2. Find the exit or container the player specifies (`defuse north`
   or `defuse chest`)
3. Check if the exit/container has `trapbuffids` set
4. If no trap: "You don't detect any traps here."
5. Opposed roll: `Perception + skullduggery × 25` vs trap
   difficulty. If the player's disarm kit has a stat mod
   (`defuse_bonus`), add it to the roll.
6. Consume the disarm kit regardless of outcome
7. Success: remove `trapbuffids` from the exit/container, message
   "You carefully disarm the trap."
8. Failure: trigger the trap (apply all trapbuffids to player),
   message "The trap triggers as you fumble the mechanism!"
9. Fire `OnSkillUse("skullduggery")` for progression

Read `internal/usercommands/picklock.go` to understand how exits
and containers with locks/traps are accessed. The trap data lives
on the lock structure.

Read how `trapbuffids` works — on failed lockpick, the buffs are
applied. Defuse should remove them BEFORE lockpicking, so the
player can pick safely after defusing.

NOTE: Traps re-arm when the lock relocks (`relockinterval`). The
defuse only disarms the current trap instance. This is handled
automatically since the lock data reloads on relock.

- [ ] **Step 3: Register command if needed**

Check `internal/usercommands/usercommands.go` for `defuse`
registration. It should already be there since the stub exists.
If the `AllowedInCombat` flag is wrong, fix it (defuse should
NOT be allowed in combat).

- [ ] **Step 4: Verify build + commit**

```bash
git commit -m "feat: defuse command — opposed roll to disarm traps, consumes kit"
```

---

### Task 3: Lockpick + Disarm Kit Items

**Files:**
- Create: `_datafiles/world/dogmud/items/other-0/` (3 lockpick items + 3 kit items)

- [ ] **Step 1: Determine item IDs**

Check the highest item ID in the `other-0/` range. Use the next
available IDs.

Read an existing tool item for format reference (e.g., the
salvage kit, item 32).

- [ ] **Step 2: Create 3 lockpick items**

All lockpicks need `type: lockpicks` (this is the type the
picklock command checks for).

Iron Lockpicks: uses 3, value 5
Steel Lockpicks: uses 8, value 15
Master Lockpicks: uses 20, value 50

Each needs: itemid, name, description (80-char wrap), type,
value, uses.

- [ ] **Step 3: Create 3 disarm kit items**

Basic Disarm Kit: value 30, single-use
Reinforced Disarm Kit: value 45, `defuse_bonus: 15` stat mod
Precision Disarm Kit: value 80, `defuse_bonus: 30` stat mod

Use `is_component: false` so they stay in backpack, not pouch.
The stat mod `defuse_bonus` is read by the defuse command.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: lockpick + disarm kit items (3 tiers each)"
```

---

### Task 4: Tool Crafting Recipes

**Files:**
- Create: 4 recipe YAMLs in `_datafiles/world/dogmud/recipes/`

- [ ] **Step 1: Create Steel Lockpicks recipe**

Blacksmithing skill 12. Station: forge.
Ingredients: materials that cost > 5g total. Include at least
one ingredient that isn't trivially available (e.g., a specific
component_tag that requires foraging or mob drops).
Output: steel lockpicks item ID.

- [ ] **Step 2: Create Master Lockpicks recipe**

Jewelcrafting skill 20. Station: jeweler's bench.
Include a rare ingredient (mob drop or zone-specific forage).
Output: master lockpicks item ID.

- [ ] **Step 3: Create Reinforced Disarm Kit recipe**

Blacksmithing skill 15. Station: forge.
Ingredients cost > 30g.
Output: reinforced disarm kit item ID.

- [ ] **Step 4: Create Precision Disarm Kit recipe**

Jewelcrafting skill 22. Station: jeweler's bench.
Include a rare ingredient.
Output: precision disarm kit item ID.

- [ ] **Step 5: Commit**

```bash
git commit -m "feat: crafting recipes for steel/master lockpicks + reinforced/precision kits"
```

---

### Task 5: Siv + Whisper Shop Stock

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/104-fence_dealer_siv.yaml`

- [ ] **Step 1: Add iron lockpicks + basic disarm kit to Siv**

Read Siv's mob file. Add to her `shop:` list:
- Iron Lockpicks (item ID from Task 3), price 5
- Basic Disarm Kit (item ID from Task 3), price 30

- [ ] **Step 2: Create Whisper NPC (room 507)**

Create mob YAML for Whisper the fence NPC. This mob will be
created in Task 8 alongside other zone mobs. For now, note
the shop stock:
- Iron Lockpicks, price 5
- Basic Disarm Kit, price 30
- Steel Lockpicks, price 20 (slight markup over craft cost)
- Chrysalis Resin Vial (poison consumable if we add one)

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: Siv sells lockpicks + disarm kits"
```

---

### Task 6: Zone Equipment Items (6 items)

**Files:**
- Create 6 item YAMLs in appropriate item folders

- [ ] **Step 1: Determine item IDs + folders**

Check highest item IDs in weapon and armor ranges.
Weapons go in `items/weapons-10000/`.
Armor goes in `items/armor-20000/<slot>/`.

- [ ] **Step 2: Create Chrysalis Knuckles (fist weapon)**

type: weapon, subtype: fist. Unarmed-combat scaling.
Poison on hit (add a buff via `buffids` on the weapon, or
use `WornBuffIds` for an on-hit effect — check how existing
poison weapons work).
Description: chrysalis-themed, 80-char wrap.

- [ ] **Step 3: Create Chitin Spaulders (shoulders)**

type: shoulders, subtype: wearable. Physical mitigation.
Str stat mod bonus. Chrysalis-themed description.

- [ ] **Step 4: Create Phantom's Wraps (gloves)**

type: gloves, subtype: wearable. Dex bonus, magical mitigation.

- [ ] **Step 5: Create Resin-Laced Bracers (wrist)**

type: wrist, subtype: wearable. Skullduggery stat mod bonus.

- [ ] **Step 6: Create Silkstep Boots (feet)**

type: feet, subtype: wearable. Dex bonus. Movement stamina
reduction if possible (check if there's a stat mod for this).

- [ ] **Step 7: Create Phantom's Cowl (head)**

type: head, subtype: wearable. Per + Dex bonus. Sneak-themed.

- [ ] **Step 8: Commit**

```bash
git commit -m "feat: 6 chrysalis-themed equipment items for rogue lair"
```

---

### Task 7: Zone Rooms (10 rooms)

**Files:**
- Create 10 room YAMLs in `_datafiles/world/dogmud/rooms/thornwall_city/`

Use `/new-room` skill or create manually following the room schema.
Each room needs: roomid, zone, title, description (80-char wrap),
biome: cave, coordinates from the spec, exits, and any locks/traps.

- [ ] **Step 1: Create rooms 500-504**

Room 500: Sealed Drain Grate (2,-2,-3). Exit up→487, north→503, down→501.
Room 501: Resin-Slicked Shaft (2,-3,-3). Exits up→500, south→502. Trap on entry.
Room 502: Warded Corridor (2,-4,-3). Exits north→501, east→504, south→505. Alarm trap.
Room 503: Fungal Nook (2,-1,-3). Exit south→500. Locked chest with spaulders.
Room 504: The Crawl (3,-4,-3). Exits west→502, south→505.

Each room needs rich descriptions with noun lookups (bric-a-brac).
Add `containers` for locked chests in rooms 503, 506, 509.
Add `lock` and `trapbuffids` to appropriate exits/containers.

- [ ] **Step 2: Create rooms 505-509**

Room 505: Chrysalis Den (2,-5,-3). Exits north→502, west→506, east→507, south→508.
Room 506: Resin Armory (1,-5,-3). Exit east→505. Locked (difficulty 22).
Room 507: The Listening Post (3,-5,-3). Exit west→505.
Room 508: Chitin Throne (2,-6,-3). Exits north→505, south→509.
Room 509: The Stash (2,-7,-3). Exit north→508. Locked (difficulty 30) + multi-trapped.

- [ ] **Step 3: Add noun descriptions**

Each room needs 3-5 nouns with `look` descriptions for flavor.
Chrysalis bric-a-brac, stolen goods, crude tools, scratched
symbols, etc. See spec for per-room noun lists.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: 10 rogue lair rooms with locks, traps, and flavor nouns"
```

---

### Task 8: Zone Mobs (4 mobs)

**Files:**
- Create 4 mob YAMLs in `_datafiles/world/dogmud/mobs/thornwall_city/`
- Possibly create JS script for boss behavior

- [ ] **Step 1: Determine mob IDs**

Check highest mob ID. Use next available range.

- [ ] **Step 2: Create Chrysalis Skulker (×2-3)**

Statpool: 100-120. Archetype: fighting. Species: human (1).
Skills: skullduggery 12, weapon-combat 8, unarmed-combat 6.
High Dex training (30+), high Per training (20+).
buffids: [9] (spawn hidden).
Groups: [humanoid, smuggler].
Hostile: true.
idlecommands: sneak + flavor emotes.
combatcommands: flee + attack emotes (NO sneak in combat).
activitylevel: 60.
Items: poisoned dagger (existing item or create one).
itemdropchance: 10.

- [ ] **Step 3: Create Resin Hound**

Statpool: 70. Archetype: fighting. Species: canine (2).
High Per training (15+), high Dex training (10+).
Groups: [predatory].
Hostile: true.
Only spawns via alarm trap (spawninfo in room 502 with
quest flag or script trigger — or just always present).
activitylevel: 50.

- [ ] **Step 4: Create Chrysalis Phantom (boss)**

Statpool: 300. Archetype: fighting. Species: human (1).
Skills: skullduggery 20, unarmed-combat 15, weapon-combat 10.
Dex training: 60+. Per training: 40+. Str training: 30+.
buffids: [9] (spawn hidden).
charm_immune: true.
Groups: [humanoid, smuggler].
Hostile: true.
Equipment: chrysalis knuckles (dual wield — weapon + offhand).
combatcommands: flee + emotes about phasing/shimmering.
idlecommands: sneak + predatory emotes.
activitylevel: 70.
itemdropchance: 20 (for knuckles), items list includes cowl.

- [ ] **Step 5: Create Whisper (fence NPC)**

Statpool: 80. Not hostile. charm_immune: true.
Groups: [humanoid, merchant].
Shop stock: iron lockpicks, basic disarm kit, steel lockpicks.
High perception training (15+).
idlecommands: whispered emotes, examining goods.

- [ ] **Step 6: Commit**

```bash
git commit -m "feat: 4 rogue lair mobs — skulkers, resin hound, phantom boss, whisper fence"
```

---

### Task 9: Wire Up Entry + Spawninfo

**Files:**
- Modify: `_datafiles/world/dogmud/rooms/thornwall_city/487.yaml`
- Modify: rooms 502, 503, 505, 506, 508, 509 (add spawninfo)

- [ ] **Step 1: Add hidden locked exit to room 487**

Read room 487 (Collapsed Passage). Add a `down` exit to room 500
with a lock (difficulty 15) and trap (minor poison buffid).
The exit should be hidden (requires `search` to discover).

Check how hidden exits work — there may be a `hidden: true` field
or it may use the noun/discovery system.

- [ ] **Step 2: Add spawninfo to zone rooms**

Room 502: resin hound spawninfo (or script-triggered).
Room 505: 2-3 chrysalis skulker spawns, respawnrate 10 minutes.
Room 504: 1 skulker spawn, respawnrate 10 minutes.
Room 503: chitin spaulders in container, respawnrate 30 minutes.
Room 506: chrysalis knuckles + phantom's wraps in container,
          respawnrate 30 minutes.
Room 508: chrysalis phantom spawn, respawnrate 30 minutes.
Room 509: resin-laced bracers + silkstep boots in container,
          respawnrate 45 minutes.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: wire rogue lair entry from room 487 + all spawninfo"
```

---

### Task 10: Help Files + Hints

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/defuse.template`
- Modify: `_datafiles/world/dogmud/templates/help/skullduggery.template`
- Modify: `_datafiles/world/dogmud/hints.yaml`
- Modify: `_datafiles/world/dogmud/keywords.yaml`

- [ ] **Step 1: Create defuse help file**

Cover: what defuse does, requires disarm kit, opposed roll,
higher tier kits improve chances, skill progression.

- [ ] **Step 2: Update skullduggery help with defuse**

Add defuse to the skullduggery skill help file's command list.

- [ ] **Step 3: Add rogue zone hints**

Add 2-3 hints:
- "Rumor has it the smuggler tunnels go deeper than anyone lets on.
  A sharp eye might spot what others miss."
- "Disarm kits can save your life in trapped passages. Fence Dealer
  Siv in Thornwall sells basic kits."
- "Higher skullduggery makes you harder to catch when fleeing.
  Rogues always have an escape plan."

- [ ] **Step 4: Add keyword aliases**

```yaml
  defuse:           [disarm, disarm trap, defuse trap]
  lockpick:         [pick, pick lock, lockpicking]
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat: defuse help file, skullduggery update, rogue zone hints"
```

---

### Task 11: Final Verification

- [ ] **Step 1: Full build + tests**

Run: `go build ./...`
Run: `go test ./... -count=1 -timeout 300s`

- [ ] **Step 2: Manual smoke test**

Walk through the entire zone:
- Search room 487 → find hidden grate
- Picklock the grate → enter zone
- Navigate trapped corridor → defuse or trigger traps
- Fight skulkers (they flee + re-sneak + surprise strike)
- Pick locked armory → get chrysalis knuckles + gloves
- Talk to Whisper → buy lockpicks
- Fight the Phantom boss (test flee rework, boss difficulty)
- Pick the stash → get bracers + boots
- Test flee with skullduggery bonus
- Buy lockpicks from Siv

- [ ] **Step 3: Update patch notes + MOTD**

- [ ] **Step 4: Commit any fixups + push**
