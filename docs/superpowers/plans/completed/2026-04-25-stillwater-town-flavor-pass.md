# Stillwater Town-Flavor Pass — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give Stillwater's 17 existing dialogue-less NPCs + 1 new caravan-son NPC enough conversational depth and idle texture to make the town feel inhabited, with caravan-on-layover atmosphere baked in for future caravan-system spec.

**Architecture:** Content-only pass — YAML data files. New mob (Lars 356), new dialogue files for 18 NPCs, idlecommand expansions on caravan crew, gossiper group flag on Neva, 6 small cross-quest extensions to existing dialogue files. No engine code changes. Three depth tiers (HEAVY/MEDIUM/LIGHT) with the HEAVY tier written by hand and MEDIUM/LIGHT delegated to a subagent with templates and per-NPC topic spines. Verification is `go build ./...` clean (no unit tests for content YAML).

**Tech Stack:** YAML data files only. GoMud engine (no engine changes). Existing dialogue/mob/room schemas in `docs/schemas/`.

**Spec:** `docs/superpowers/specs/2026-04-25-stillwater-town-flavor-pass-design.md`

---

## Task 1: Create Lars Ketilson mob YAML

**Files:**
- Create: `_datafiles/world/dogmud/mobs/stillwater/356-lars_ketilson.yaml`

- [ ] **Step 1: Verify mob ID 356 is clear**

Run: `grep -rl "^mobid: 356" _datafiles/world/dogmud/mobs/`
Expected: no matches.

- [ ] **Step 2: Write the mob YAML**

```yaml
mobid: 356
zone: Stillwater
behavior_archetype: noncombat_passive
statpool: 70
itemdropchance: 0
hostile: false
charm_immune: true
non_combatant: true
maxwander: 3
groups:
  - humanoid
  - traveler
idlecommands:
  - 'emote checks his slate-list of orders and grimaces faintly'
  - ''
  - 'emote tries to read his father''s handwriting on the slate, fails politely, holds it sideways'
  - ''
  - 'say I am supposed to be running these to the smith. Five minutes more.'
  - ''
  - 'emote tucks the slate under one arm and leans on the well, watching the square'
  - ''
  - 'say They make me run the deliveries because I am the youngest. I am told this is character-building.'
  - ''
  - 'emote pats the side of one of the wagons absently as he passes it'
  - ''
activitylevel: 14
character:
  name: lars
  description: |
    A wiry teen with a sun-darkened face, a too-large oilskin jacket
    hand-me-down from his father, and the alert quiet of someone who
    has spent the last three days holding the reins on a wagon that
    Did Not Want To Be Held. Lars Ketilson is the Stillwater--Thornwall
    caravan's youngest hand -- the one his father makes do all the
    in-town deliveries and errand-running on layover, the one who
    carries the slate-list of which shopkeeper gets which crate. He
    would rather be on a guard's pay grade than running notes around
    town, but he does not say that out loud and he does not sulk.
    Mostly he deflects with a small joke and gets the job done.
  speciesid: 1
  gold: 12
  stats:
    dexterity:
      training: 10
    perception:
      training: 12
    charisma:
      training: 6
```

- [ ] **Step 3: Verify file loads — build check**

Run: `go build ./...`
Expected: clean.

---

## Task 2: Add Lars's spawninfo to room 4102

**Files:**
- Modify: `_datafiles/world/dogmud/rooms/stillwater/4102.yaml` (append to existing `spawninfo` block)

- [ ] **Step 1: Read 4102's current spawninfo**

Read the existing `spawninfo` block at the bottom of `_datafiles/world/dogmud/rooms/stillwater/4102.yaml`. It currently lists mobs 336 (fishmonger), 351 (caravan driver), 352 (caravan guard), 354 (beggar). Verify the format.

- [ ] **Step 2: Append Lars's spawn entry**

Add at the end of the `spawninfo` list:

```yaml
- mobid: 356
  cooldown: 600 rounds
```

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean. Restart will spawn Lars in 4102 alongside the existing caravan crew.

---

## Task 3: Write Sigrid's HEAVY-tier dialogue (mob 333)

**Files:**
- Create: `_datafiles/world/dogmud/dialogue/stillwater/333.yaml`

- [ ] **Step 1: Read mob 333 description for voice**

Read `_datafiles/world/dogmud/mobs/stillwater/333-innkeeper_sigrid.yaml` to recall Sigrid's existing description and tone.

- [ ] **Step 2: Write the dialogue file**

Write to `_datafiles/world/dogmud/dialogue/stillwater/333.yaml`. Structure:

- `mobid: 333`, `zone: Stillwater`, `defaultMood: friendly`
- `patterns` (~7 entries):
  - greetings (mood-aware: neutral/friendly + grateful variants)
  - `ale/wine/drink/menu` — what she sells, no specific prices
  - `room/lodging/bed/upstairs` — rentable rooms, the loft, lake view
  - `caravan/ketil/lars/marta` — the caravan crew is on layover, brought letters and goods from Thornwall
  - `bandits/road/north` — quieter than it was; Drunn does her credit
  - `lake/fish/catch` — short year but the nets are mending
  - catch-all `keywords: [""]`
- `tree.root`:
  - default: "Sit anywhere. Common room is quiet this hour."
  - hints: "You could ask Sigrid about the inn, the caravan crew, news from Thornwall, or the town."
  - variants:
    - `questRequired: ["20-end"]` → "Ulla came in for a drink last week. First time in years. That is your doing." (acknowledges Voss family resolution)
    - `questRequired: ["19-end"]` + `questFlagRequired: {"19-completion": "full"}` → "Drunn told me you put the deep-water thing down. Drinks are on the house tonight."
- `tree.nodes`:
  - `lake_decline` — triggers `["lake", "fish", "catch", "decline"]` — "Catch is shorter than it was, but Tov down at the stall says the nets are mending now. Drunn's bounty did the trick." (cross-ref Tov + Drunn)
  - `caravan_news` — triggers `["caravan", "ketil", "lars", "marta", "thornwall", "letters"]` — describes Ketil's crew on layover, letters from Maren, packages for the shopkeepers (cross-ref Ketil/Lars/Marta + Maren)
  - `voss_delicate` — triggers `["voss", "ulla", "vella", "elgar"]`, `questRequired: ["20-end"]` — brief, respectful: "I kept Ulla's chair by the window. She came in last week with Vella. Neither of them said much. They did not need to." (cross-ref Ulla, Vella)
  - `town_history` — triggers `["history", "mother", "inn", "pike", "lantern"]` — "My mother had this place before me. She would barely recognise the common room now -- I rebuilt the bar after the '12 fire." (small personal lore)
  - `gossip_drunn` — triggers `["drunn", "constable", "guard"]` — "Drunn is a fixture. Slightly understaffed, slightly too many years in, but he handles what comes."
  - `lodging_rates` — triggers `["loft", "rent", "lodging", "rate", "stay"]` — describes the loft (no actual mechanic, just flavor)
- `memory: {expiryPeriod: ""}`

**Voice notes:** warm and welcoming default; carries weight of seeing the town through hard years; first-person ("I", "my"); hints in narrator voice; no narrator overreach (never invent details about other characters' inner lives).

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean.

---

## Task 4: Write Ketil's HEAVY-tier dialogue (mob 351)

**Files:**
- Create: `_datafiles/world/dogmud/dialogue/stillwater/351.yaml`

- [ ] **Step 1: Read mob 351 description for voice**

Read `_datafiles/world/dogmud/mobs/stillwater/351-caravan_driver_ketil.yaml`.

- [ ] **Step 2: Write the dialogue file**

Structure:

- `mobid: 351`, `zone: Stillwater`, `defaultMood: neutral`
- `patterns` (~6 entries):
  - greetings (road-weary register: "Welcome. Mind the wagon ruts.")
  - `route/road/thornwall/north` — the run takes 3 days each way, ditch washed out at the second crossing
  - `wagon/horses/hub/wheel` — wagon needs repair, Brindle takes a look each layover
  - `weather/schedule/leaving/depart` — "Morning after next, weather holding."
  - `lars/son/marta/crew` — about the caravan crew (gentle paternal grumble about Lars)
  - catch-all
- `tree.root`:
  - default: "Welcome. Mind the wagon ruts."
  - hints: "You could ask Ketil about the caravan route, his crew, or news from Thornwall."
  - variants:
    - `questRequired: ["20-end"]` → "I knew Voss in passing -- Maren's uncle, ran a few small things north for him before he went into the deep. Glad it is settled."
- `tree.nodes`:
  - `route_thornwall` — triggers `["route", "road", "thornwall", "north", "trip", "run"]` — describes the route in atmospheric terms only (DOES NOT specify cadence/timing — "we run when we can" is the most specific allowed). Foreshadows the future caravan-system mechanic without committing.
  - `road_bandits` — triggers `["bandits", "raid", "raiders", "soren"]` — "There used to be a serious problem on the North Road. Quieter now -- Drunn's lot, and a traveler or two who took the bounty seriously, cleared the worst of them. Never gone, though."
  - `lars_kid` — triggers `["lars", "son", "boy", "kid"]` — "He will thank me when he is running his own wagon, which will be when I am too old to read the manifests. He thinks I make him run errands to spite him. I do not. I make him run errands because someone has to and his legs are younger." (cross-ref Lars)
  - `marta_pro` — triggers `["marta", "guard", "fighter"]` — "Marta has saved the caravan twice. I overpay her on purpose." (cross-ref Marta)
  - `maren_letter` — triggers `["maren", "letter", "letters", "thornwall"]` — "Sometimes I carry letters between Maren in Thornwall and her aunt Ulla here. I do not open them. The seal is the same wax both ways -- they made it as girls together." (cross-ref Maren + Ulla)
  - `shopkeepers_know_him` — triggers `["shopkeepers", "orders", "deliveries", "stock", "brindle", "ilsa", "edda"]` — "I know what each shopkeeper orders without checking the slate. Brindle takes lake-iron stock. Ilsa takes the small ground-glass bottles. Edda takes thread-spool by the dozen. Lars writes the slate down anyway, in case I die between runs and someone has to take over." (cross-ref Brindle/Ilsa/Edda — sets up future delivery mechanic)
- `memory: {expiryPeriod: ""}`

**Voice notes:** weary-but-content, pragmatic, dry humor; first-person; hints narrator-perspective; no specific timing/cadence on the caravan run (atmospheric only).

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean.

---

## Task 5: Write Lars's HEAVY-tier dialogue (mob 356)

**Files:**
- Create: `_datafiles/world/dogmud/dialogue/stillwater/356.yaml`

- [ ] **Step 1: Reference Lars's mob description (from Task 1) for voice**

Voice: jokey/loose teen, slouching but attentive, deflects road stress with humor, picks up gossip by being underestimated.

- [ ] **Step 2: Write the dialogue file**

Structure:

- `mobid: 356`, `zone: Stillwater`, `defaultMood: friendly`
- `patterns` (~6 entries):
  - greetings (jokey: "Welcome to my office, which is this corner of the square. Mind the slate.")
  - `who/yourself/lars/name` — introduces self with patronymic when asked: "Lars Ketilson. The kid in the caravan crew. I do all the deliveries because I have working knees."
  - `route/road/thornwall/run` — what Thornwall is like from a teen's perspective: "Bigger. Louder. Better food but worse beer. The market goes on for blocks."
  - `father/dad/ketil` — affectionate-resigned: "He could send Marta to do the deliveries. He does not. The man has a sense of humour about who has working knees."
  - `slate/orders/deliveries` — describes the slate-list and his routine
  - catch-all
- `tree.root`:
  - default: "Welcome to my office, which is this corner of the square. What can I help with?"
  - hints: "You could ask Lars about the caravan, his father, the run from Thornwall, or what he is doing here."
- `tree.nodes`:
  - `caravan_run` — triggers `["caravan", "run", "trip", "route"]` — "Three days each way, four if the ditch at the second crossing is full. We came in Tuesday. Leaving morning after next, weather holding. I get the layover off, except for the deliveries. Which is most of the layover." (joke at his own expense)
  - `thornwall` — triggers `["thornwall", "city", "maren"]` — "Bigger than Stillwater. Maren the weaver's place is on the way through; we drop in if we have a letter for her aunt here. She always has bread." (cross-ref Maren)
  - `father_grumble` — triggers `["father", "dad", "ketil"]` — "He runs a tight wagon. I respect that. I respect it less when I am running a crate of bottles to the apothecary in the rain. But I respect it." (cross-ref Ketil)
  - `marta_respect` — triggers `["marta", "guard"]` — "I have seen her do things with that sword that I will be telling stories about when I am old. She does not talk much. I think she likes me. I cannot prove it." (cross-ref Marta)
  - `wants_promotion` — triggers `["promotion", "job", "guard", "fighting", "want"]` — "I want a guard's pay. I am not going to pretend otherwise. My father says I have to grow into it. I have been growing for three years. He says it does not happen on a schedule. I think he just enjoys the deliveries being done." (deflective humor)
  - `slate_orders` — triggers `["slate", "orders", "list", "deliveries"]` — "Brindle takes lake-iron. Ilsa takes the small bottles. Edda takes thread-spool. Wulf takes everything else. Sigrid pretends she is not on the list because she has a pride thing about it. She is on the list." (cross-ref multiple shopkeepers; foreshadows future delivery mechanic)
- `memory: {expiryPeriod: ""}`

**Voice notes:** teen-jokey but not whining; quick deflection; first-person; respects his elders even when grumbling; never breaks the layover atmosphere with caravan-system mechanics.

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean.

---

## Task 6: Expand Ketil's idlecommands with caravan-layover lines

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/stillwater/351-caravan_driver_ketil.yaml`

- [ ] **Step 1: Read existing idlecommands**

Read `_datafiles/world/dogmud/mobs/stillwater/351-caravan_driver_ketil.yaml` to find the current `idlecommands` block.

- [ ] **Step 2: Append caravan-layover lines**

Add to the existing `idlecommands` list (after the existing entries, before `activitylevel`):

```yaml
  - 'say Ditch at the second crossing washed out again. Going to take longer next run.'
  - ''
  - 'emote checks the manifest slate against the wagon contents and ticks something off'
  - ''
  - 'say Lars. Did you take the bottles to Ilsa yet. -- Yes. -- Then why are they here. -- Oh.'
  - ''
  - 'emote stretches his back with an audible crack and grimaces'
  - ''
```

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean.

---

## Task 7: Expand Marta's idlecommands with caravan-layover lines

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/stillwater/352-caravan_guard_marta.yaml`

- [ ] **Step 1: Read existing idlecommands**

Read `_datafiles/world/dogmud/mobs/stillwater/352-caravan_guard_marta.yaml`.

- [ ] **Step 2: Append caravan-layover lines**

Add to the existing `idlecommands` list:

```yaml
  - 'emote draws her sword half-out of the scabbard, examines the edge, oils it with a small rag'
  - ''
  - 'emote tightens the strap on her shield and tests its swing-weight'
  - ''
  - 'say Road has been quieter than last summer. Make of that what you will.'
  - ''
  - 'emote leans against a wagon and watches the square with the steady patience of someone whose attention rarely lapses'
  - ''
```

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean.

---

## Task 8: Add gossiper group to Neva

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/stillwater/334-barmaid_neva.yaml`

- [ ] **Step 1: Read Neva's mob YAML**

Read `_datafiles/world/dogmud/mobs/stillwater/334-barmaid_neva.yaml` to find the `groups` block.

- [ ] **Step 2: Add `gossiper` to her groups list**

Modify the existing `groups` block to add `gossiper`:

```yaml
groups:
  - humanoid
  - merchant
  - gossiper
```

(Or whatever her existing groups are — the change is to ADD `- gossiper` as a new entry; do not remove existing ones.)

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean. After server restart, Neva will broadcast gossip lines from `gossip_templates.yaml` per the engine's Stage 42.5 system.

---

## Task 9: Cross-quest extension — Drunn (335)

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/stillwater/335.yaml` (existing; add a new pattern entry)

- [ ] **Step 1: Read Drunn's existing patterns block**

Find the `patterns` list in `_datafiles/world/dogmud/dialogue/stillwater/335.yaml`. Identify a good insertion point (after his existing topic patterns, before the catch-all `keywords: [""]`).

- [ ] **Step 2: Insert the caravan pattern**

Add this pattern entry before the catch-all:

```yaml
  - keywords: ["caravan", "ketil", "road", "north", "lars"]
    responses:
      - "Ketil's lot is in town a couple days every fortnight or so. The road has been quieter since the bounty cleared the worst of Soren's lot -- I check in on the crew when they're through, just so they know who runs the patrols. The kid Lars is a fixture by the well most layovers."
```

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean.

---

## Task 10: Cross-quest extension — Arn (342)

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/stillwater/342.yaml`

- [ ] **Step 1: Read Arn's existing patterns block**

- [ ] **Step 2: Insert caravan pattern before catch-all**

```yaml
  - keywords: ["caravan", "ketil", "wagon", "freight"]
    responses:
      - "Ketil and I trade weather and lake-condition reports each layover. The docks load some Thornwall-bound freight onto his wagons -- smoked fish, tanned hides, the small things that travel well. Marta keeps a courteous distance from my paperwork. We get along."
```

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean.

---

## Task 11: Cross-quest extension — Brindle (337)

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/stillwater/337.yaml`

- [ ] **Step 1: Read Brindle's existing dialogue tree**

Find the `tree.nodes` list. Identify a good insertion point (after his existing nodes).

- [ ] **Step 2: Insert a `caravan_orders` node**

```yaml
    - id: caravan_orders
      triggers: ["caravan", "ketil", "lars", "stock", "orders"]
      questRequired: ["19-end"]
      text: "Ketil's crew brings my lake-iron stock from the smelters in
        Thornwall. Lars is a good kid even if he writes the orders down
        wrong half the time. I let him think I am annoyed about it
        because his father expects me to. I am not."
```

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean.

---

## Task 12: Cross-quest extension — Seren (344)

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/stillwater/344.yaml`

- [ ] **Step 1: Read Seren's existing patterns**

- [ ] **Step 2: Insert caravan pattern before catch-all**

```yaml
  - keywords: ["caravan", "ketil", "donation", "alms"]
    responses:
      - "Ketil keeps a small donation envelope for the temple each run. I do not pry about what is in it. The amount is consistent enough that I suspect his wife in Thornwall set the rate, not him."
```

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean.

---

## Task 13: Cross-quest extension — Vella (355)

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/stillwater/355.yaml`

- [ ] **Step 1: Read Vella's existing tree.root variants**

Find the existing post-quest 20 root variant (`questRequired: ["20-end"]`).

- [ ] **Step 2: Add a single line to that variant's text**

Modify the existing post-quest 20 root variant to APPEND a final sentence:

```yaml
      - questRequired: ["20-end"]
        text: "Sit. The kettle is on. The town is quieter for what you
          did, and so am I. I have not slept that well in twelve years.
          The kingfisher sits on Ulla's mantel now. I went to see it
          last week."
        hints: "You could ask Vella about her tinctures or the town."
```

(Replace the existing post-20 variant's text with this expanded version.)

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean.

---

## Task 14: Cross-quest extension — Ulla (347)

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/stillwater/347.yaml`

- [ ] **Step 1: Read Ulla's existing tree.root post-20 variants**

Find the partial-truth and whole-truth post-20 root variants.

- [ ] **Step 2: Append a single sentence to each**

Add to the partial-truth variant text: " I went to the Pike & Lantern for the first time in a long while. Sigrid kept the old seat for me."

Add to the whole-truth variant text: " I went to the Pike & Lantern for the first time in a long while. Sigrid kept the old seat for me. Vella came with me."

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: clean.

---

## Task 15: Direct-work commit checkpoint

- [ ] **Step 1: Confirm all direct work is in place**

Files touched in Tasks 1-14:
- 1 new mob YAML (Lars 356)
- 3 new HEAVY dialogue YAMLs (Sigrid 333, Ketil 351, Lars 356)
- Room 4102 spawninfo addition
- 2 mob idlecommand expansions (Ketil 351, Marta 352)
- Neva (334) gossiper addition
- 6 cross-quest extensions to existing dialogue files

Run: `git status`
Expected: shows all the above as new/modified.

- [ ] **Step 2: Build check before commit**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Commit the direct work**

```bash
git add _datafiles/world/dogmud/mobs/stillwater/356-lars_ketilson.yaml \
        _datafiles/world/dogmud/mobs/stillwater/351-caravan_driver_ketil.yaml \
        _datafiles/world/dogmud/mobs/stillwater/352-caravan_guard_marta.yaml \
        _datafiles/world/dogmud/mobs/stillwater/334-barmaid_neva.yaml \
        _datafiles/world/dogmud/rooms/stillwater/4102.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/333.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/351.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/356.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/335.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/342.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/337.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/344.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/355.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/347.yaml
git commit -m "$(cat <<'EOF'
feat(stillwater): town-flavor pass — Lars + HEAVY-tier dialogues + cross-quest extensions

Adds Lars Ketilson (mob 356) as the caravan master's son on layover at
Lakefront Square. Writes HEAVY-tier dialogue files for Sigrid,
Ketil, and Lars (town hub + caravan canon anchors). Expands Ketil and
Marta's idlecommands with caravan-layover atmosphere lines. Tags Neva
with gossiper group. Adds small caravan/post-quest cross-references to
the 6 existing dialogue files (Drunn, Arn, Brindle, Seren, Vella, Ulla).

MEDIUM and LIGHT tier dialogues for the remaining 15 NPCs follow in
the next commit, dispatched to a subagent.

Spec: docs/superpowers/specs/2026-04-25-stillwater-town-flavor-pass-design.md

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 16: Dispatch subagent for MEDIUM and LIGHT tier dialogues

**Files (subagent will create 15):**

MEDIUM (9): `_datafiles/world/dogmud/dialogue/stillwater/{336,338,339,340,341,343,348,349,353}.yaml`
LIGHT (6): `_datafiles/world/dogmud/dialogue/stillwater/{334,345,346,350,352,354}.yaml`

- [ ] **Step 1: Verify all 15 mob YAMLs exist (defensive)**

Run: `ls _datafiles/world/dogmud/mobs/stillwater/{336,338,339,340,341,343,348,349,353,334,345,346,350,352,354}*.yaml`
Expected: 15 files listed.

- [ ] **Step 2: Dispatch subagent with the brief**

Dispatch a `general-purpose` subagent (model: sonnet) with this complete brief:

```
You are creating 15 dialogue YAML files for Stillwater NPCs as part of
a town-flavor pass for DOGMud. This is mechanical content writing
following tight templates — no design decisions to make, just voice
and structure execution.

GOAL: Create 15 files at _datafiles/world/dogmud/dialogue/stillwater/{mobid}.yaml

PER-NPC SCOPE (MEDIUM tier — 9 files, ~50 lines each):

| Mob | NPC | Trade topic keywords | Personal/lore topic | Cross-references |
|-----|-----|---------------------|---------------------|------------------|
| 336 | Tov Brann (fishmonger) | clams, fish, catch, stall | the lake's lean year | Drunn (bounty), Hodder (mended nets) |
| 338 | Ilsa (apothecary) | tinctures, willow, bark, remedy | which fishermen she's patched up | Vella (her senior counterpart) |
| 339 | Edda (weaver) | cattail, weaving, loom, wool | not being Maren-of-Thornwall | Maren (Thornwall), Ulla (loom) |
| 340 | Kess (pearl-carver) | pearls, silver, wire, work | the divers who used to bring stillwater pearls | Vella (medicinal pearls), Tov (fishermen who dive) |
| 341 | Wulf (storekeeper) | salvage, lantern, oil, kit | knows everyone's order | Ketil (caravan stock arrivals) |
| 343 | Hodder (old fisherman, gossiper) | nets, mending, forty | names old fishermen no longer alive | Old Voss (delicately), Tov (apprentice generation) |
| 348 | Bram (miller) | wheel, grain, mill, flood | the seasonal flood pattern | Drunn (mill-bridge), Edda (cattail harvest) |
| 349 | Gyda (old cottager, gossiper) | porch, days, town, old | town in her grandmother's day | Sigrid (Pike & Lantern history), Vella (deliveries to elders) |
| 353 | Fenwick (pilgrim) | road, news, travel, sanctum | news from elsewhere | Maren (Thornwall waystop), Ketil (road conditions) |

PER-NPC SCOPE (LIGHT tier — 6 files, ~20 lines each):

| Mob | NPC | One personality beat |
|-----|-----|----------------------|
| 334 | Neva (barmaid, gossiper) | barmaid quips; engine carries her ambient news |
| 345 | Finn (temple acolyte) | devoted, mentions Seren respectfully |
| 346 | Luc (young fisherman) | ambitious, watches Hodder for technique |
| 350 | Pip (the kid) | wants to be a fisherman like his da |
| 352 | Marta (caravan guard) | terse, oils her sword, "the road's been quieter than last summer" |
| 354 | Oswin (beggar) | ex-fisherman, heard a thing about the boat that sank in '08 |

TEMPLATES:

MEDIUM template structure (per file, ~50 lines):
- mobid, zone, defaultMood (pick: neutral, friendly, cautious — match the mob's existing description tone)
- patterns: 5 entries — greetings (mood-aware) + 3 topic patterns from the table + catch-all `keywords: [""]`
- tree.root: default text + hints + 1 cross-quest variant if relevant (e.g., post-19-end or post-20-end acknowledgment)
- tree.nodes: 2-3 info nodes drawing on the trade/lore/cross-reference topics
- memory: {expiryPeriod: ""}

LIGHT template structure (per file, ~20 lines):
- mobid, zone, defaultMood
- patterns: 3 entries — greeting + 1 character-beat topic + catch-all
- tree.root: single default text with hints; NO variants, NO nodes
- memory: {expiryPeriod: ""}

VOICE CANON (apply across all files):
- Lakefolk Norse register — Ulla, Vella, Maren, Drunn, Arn are reference points. Read their existing dialogue files (dialogue/stillwater/{347,355,335,342}.yaml) for tone calibration.
- Post-Voss aftermath baseline — the town has always known something was off about Elgar's death. Pre-quest characters can carry that quiet weight without naming it directly.
- Caravan-on-layover atmosphere — Ketil's crew is in town; this is fresh news everyone's heard. Shopkeepers may reference recent stock arrivals.
- First-person NPC text. Hints in narrator voice ("You could ask X about Y").
- No 3rd-person self-references in hints (never "Ask about why she left" — write "You could ask why she left").

HARD CONSTRAINTS:
- NO quest hooks (no `grantsQuest`, no quest tokens granted from these files)
- NO `requires:` (use `questRequired` if quest gating is needed)
- NO `expiryPeriod` set (always empty string)
- NO narrator overreach: never attribute internal motives to absent/dead characters; never invent details not in existing room or mob descriptions; stick to what the player can directly observe + standard narrator-tier guidance hints
- Catch-all `keywords: [""]` MUST be the LAST pattern entry in every patterns list
- Trigger discoverability: every keyword must be sourced from existing in-game text (mob descriptions, room descriptions, item names, the topic table above). Don't use clever keywords the player can't derive.

LORE BEATS TO THREAD:
1. Lake decline — Tov, Hodder, Bram should reference. Catch was short last year, mending now after Drunn's bounty.
2. Post-Voss aftermath — Gyda (porch gossip about Ulla emerging) and Edda (loom solidarity, references Maren in Thornwall) should reference, gated by `questRequired: ["20-end"]` only.
3. Maren-in-Thornwall thread — Edda and Fenwick should reference Maren (mob 113 in Thornwall, weaver, Elgar's niece, writes letters).
4. Caravan foreshadowing without commitment — Wulf and Hodder may mention Ketil's crew; NEVER specify cadence (never "every 5 days" or similar — atmospheric only). Future caravan-system spec will add real timing.
5. Pre-Chrysalis spiral mystery — DO NOT reference. Sealed for future content; players who finished quest 20 own the mystery.

PROCESS:
1. Read each mob's YAML (mobs/stillwater/{mobid}-*.yaml) to absorb voice
2. Read 2-3 existing dialogue files in dialogue/stillwater/ for structural reference
3. Write each file in turn
4. After all 15 files written, run `go build ./...` from the repo root
5. Report any ambiguity, canon contradictions, or trigger-discoverability concerns you encountered

DELIVERABLE:
- 15 dialogue YAML files at the paths listed above
- Build clean confirmed
- A brief report listing any decisions you made (especially around voice, defaultMood per NPC, and any cross-references that didn't fit naturally)
```

- [ ] **Step 3: Wait for subagent completion and review the report**

The subagent should report when done with build status and any decisions notes. Expected runtime: 5-10 minutes.

- [ ] **Step 4: Spot-check 3 files for voice consistency**

Read 3 files (mix of MEDIUM and LIGHT, e.g., 339 Edda + 343 Hodder + 350 Pip). Verify:
- First-person NPC text, narrator-perspective hints
- Catch-all is last in patterns
- No `requires`, no `expiryPeriod` set
- Cross-references match the table above
- Voice fits the lakefolk Norse register

If voice drifts on any file, ask the subagent to revise that specific file (don't restart the whole batch).

- [ ] **Step 5: Final build check**

Run: `go build ./...`
Expected: clean.

---

## Task 17: Subagent-work commit

- [ ] **Step 1: Confirm all subagent files exist**

Run: `ls _datafiles/world/dogmud/dialogue/stillwater/{336,338,339,340,341,343,348,349,353,334,345,346,350,352,354}.yaml`
Expected: 15 files listed.

- [ ] **Step 2: Final build check**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Commit the subagent work**

```bash
git add _datafiles/world/dogmud/dialogue/stillwater/336.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/338.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/339.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/340.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/341.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/343.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/348.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/349.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/353.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/334.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/345.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/346.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/350.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/352.yaml \
        _datafiles/world/dogmud/dialogue/stillwater/354.yaml
git commit -m "$(cat <<'EOF'
feat(stillwater): town-flavor pass — MEDIUM and LIGHT tier dialogues (15 NPCs)

Completes the Stillwater town-flavor pass with dialogue files for the
remaining 15 NPCs: 9 MEDIUM tier (Tov Brann, Ilsa, Edda, Kess, Wulf,
Hodder, Bram, Gyda, Fenwick) and 6 LIGHT tier (Neva, Finn, Luc, Pip,
Marta, Oswin).

Each file follows the layered tier template per the spec — patterns
+ tree variants + info nodes for MEDIUM, patterns + tree.root only
for LIGHT. Cross-reference web seeded per the spec's topic spine
table. Lore beats threaded: lake decline (Tov/Hodder/Bram), post-Voss
aftermath (Gyda/Edda gated on 20-end), Maren-in-Thornwall thread
(Edda/Fenwick), caravan foreshadowing without commitment
(Wulf/Hodder).

Spec: docs/superpowers/specs/2026-04-25-stillwater-town-flavor-pass-design.md
Plan: docs/superpowers/plans/2026-04-25-stillwater-town-flavor-pass.md

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 18: Verification + handoff

- [ ] **Step 1: Final repo state check**

Run: `git log --oneline -3`
Expected: two new commits (HEAVY-tier + cross-quest extensions, then MEDIUM/LIGHT tier).

- [ ] **Step 2: Final build check**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Print verification guidance for the user**

Output to user:

> Town-flavor pass shipped. 19 new files, 9 modified. Build clean.
>
> **In-game smoke (after restart):**
> 1. Walk to Lakefront Square (4102) — confirm Lars (mob 356) spawned alongside Ketil/Marta/Tov/Oswin/caravan.
> 2. `look lars` → confirm description renders. `ask lars caravan`, `ask lars father`, `ask lars promotion` → confirm dialogue.
> 3. Walk to Pike & Lantern (4103) — `ask sigrid caravan`, `ask sigrid lake`, `ask sigrid drunn`. If you have 20-end, confirm the Voss-aftermath greeting fires.
> 4. `ask ketil route`, `ask ketil lars`, `ask ketil maren`, `ask ketil shopkeepers`. If you have 20-end, confirm the Voss greeting variant.
> 5. Sit in Pike & Lantern for ~150 rounds — Neva (334) should periodically broadcast gossip lines from the world-event templates.
> 6. Spot-check 3-4 of the MEDIUM/LIGHT tier NPCs from the subagent batch — `look <npc>` + `ask <npc> <topic>` per their topic spine.
> 7. After completing quest 20, return to Sigrid + Vella + Edda + Gyda — confirm the post-Voss aftermath beats land across the four NPCs.
>
> **Caravan system spec is the natural next step** — the layover dialogue is in place; now we can design the actual movement/combat/restock mechanic. Run `/sketch-quest` adjacent or invoke brainstorming directly.

---

## Self-Review Notes

**Spec coverage:** Each spec section maps to tasks:
- Lars's full character → Task 1 (mob YAML), Task 5 (dialogue)
- Caravan-layover framing → Tasks 6, 7 (idlecommand expansions) + the layover beats baked into HEAVY dialogues (Tasks 4, 5)
- HEAVY tier → Tasks 3, 4, 5
- MEDIUM and LIGHT tiers → Task 16 (subagent dispatch)
- Gossiper expansion (Neva) → Task 8
- Cross-quest extensions → Tasks 9-14
- Lore beats → embedded in HEAVY dialogues + subagent brief
- Cross-reference web → embedded in HEAVY dialogues + subagent brief
- Verification plan → Task 18

**Placeholder scan:** No TBDs or unspecified content. All step-3 build checks have explicit expected output. All file content is shown inline (Tasks 1-14) or specified via the table-driven subagent brief (Task 16).

**Type/structure consistency:** YAML field names (`mobid`, `zone`, `defaultMood`, `patterns`, `tree`, `nodes`, `triggers`, `text`, `hints`, `questRequired`, `questFlagRequired`, `memory.expiryPeriod`, `idlecommands`, `groups`, `spawninfo`, `cooldown`) used consistently across all tasks per the dialogue/mob/room schemas.
