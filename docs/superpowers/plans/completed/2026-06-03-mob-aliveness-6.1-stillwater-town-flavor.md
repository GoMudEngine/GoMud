# Stillwater Town-Flavor Pass (6.1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply the mob-aliveness substrate (relationships, schedules, knowledge/facts, NPC↔NPC conversations) to the Stillwater zone as the first-zone benchmark.

**Architecture:** Pure data-file authoring against existing engine systems — no Go changes. Author layer-by-layer in dependency order (relationships → facts/knowledge → schedules → conversations), boot-validating after each layer because the engine's startup validators panic on bad data. The "test" for each task is a clean server boot with no panics and no relevant warnings.

**Tech Stack:** YAML data files under `_datafiles/world/dogmud/`. Go server (`go build` + boot). Validators live in `internal/mobs`, `internal/relationships`, `internal/facts`, `internal/conversations`, and the schedule loader.

---

## Verified facts (do not re-derive)

- **Mob YAML fields:** `relationships:` is `[]{to:int, type:string, subtype:string(optional)}`; `knows_facts:` is `[]string` of fact ids; `schedule_id:` is a string. (`internal/mobs/mobs.go:75-79,168-169`)
- **Relationship types:** `family`, `friend`, `rival`, `lover` (symmetric, auto-mirrored), `employer`↔`employee` (asymmetric, auto-mirrored). Author **one side per edge**; the engine creates the reverse. Unknown ids/types **warn, not panic**.
- **Facts** (`facts.yaml`): each entry `{id, description, significance(int), declared_round(uint64), tags:[], status: active}`. Unknown `knows_facts` ids warn, not panic.
- **Schedules:** file at `_datafiles/world/dogmud/schedules/stillwater/<id>.yaml`, filename = id. Segments must cover all 24h with no gaps/overlaps; `target_room` must exist; `mapper.GetPath` must succeed between consecutive segments (incl. wrap-around). **These are PANICS.** `activity` ∈ `"" | craft | sleeping | patrol`.
- **Conversation type pools:** `_datafiles/world/dogmud/conversations/types/<type>.yaml` with `id` = relationship type, `description`, `exchanges:`, optional `subtypes:`. Pair overrides: `pairs/<lowerId>_<higherId>.yaml` with `id`, `mob_a`, `mob_b`, `exchanges:`. Speaker labels are role-agnostic `A`/`B` (engine randomizes which physical NPC is A). Conversation resolves the **initiator's** edge type, and the initiator is random — so for employment pairs **both** `employer` and `employee` pools must exist.
- **Verified room connectivity (reciprocal, unlocked):** 4103↔4104 (up/down), 4103↔4106 (north/south), 4116↔4113 (north/south), 4136↔4137 (east/west).
- **Anchor spawn rooms:** Sigrid 333→4103, Neva 334→4103, Brindle 337→4106, Seren 344→4123, Arn 342→4116, Ilsa 338→4125, Bram 348→4135, Vella 355→4136.
- **Maren exists:** Thornwall weaver, mob id **113**.
- **Gossiper tag:** add `- gossiper` under `groups:`.

## Pre-flight: branch

Already on branch `feature/mob-aliveness-6.1-stillwater-town-flavor` (spec + folder-cleanup committed). All work below lands on this branch.

---

## Task 1: Relationship graph (Section A)

**Files (modify — add a `relationships:` block to each, placed after the `groups:` block):**
- `_datafiles/world/dogmud/mobs/stillwater/333-innkeeper_sigrid.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/335-constable_drunn.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/337-smith_brindle.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/338-apothecary_ilsa.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/343-old_fisherman_hodder.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/344-temple_priest_seren.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/347-ulla.yaml`
- `_datafiles/world/dogmud/mobs/stillwater/349-old_cottager_gyda.yaml`

Edges authored **one side each** (reverse auto-mirrors). Insert exactly these blocks:

- [ ] **Step 1: Sigrid (333)** — add:
```yaml
relationships:
  - to: 334
    type: employer
    subtype: barmaid
  - to: 341
    type: rival
    subtype: petty
```

- [ ] **Step 2: Drunn (335)** — add:
```yaml
relationships:
  - to: 333
    type: friend
    subtype: regular
```

- [ ] **Step 3: Brindle (337)** — add:
```yaml
relationships:
  - to: 343
    type: friend
```

- [ ] **Step 4: Ilsa (338)** — add:
```yaml
relationships:
  - to: 355
    type: friend
    subtype: colleague
```

- [ ] **Step 5: Hodder (343)** — add (Hodder is the mentor):
```yaml
relationships:
  - to: 336
    type: friend
    subtype: mentor
  - to: 346
    type: friend
    subtype: mentor
```

- [ ] **Step 6: Seren (344)** — add:
```yaml
relationships:
  - to: 345
    type: employer
    subtype: acolyte
```

- [ ] **Step 7: Ulla (347)** — add (cross-zone niece edge to Maren 113 confirmed):
```yaml
relationships:
  - to: 355
    type: family
    subtype: sister-in-law
  - to: 113
    type: family
    subtype: niece
```

- [ ] **Step 8: Gyda (349)** — add:
```yaml
relationships:
  - to: 355
    type: friend
    subtype: neighbor
```

- [ ] **Step 9: Build**

Run: `go build ./...`
Expected: exit 0, no output.

- [ ] **Step 10: Boot and check relationship load**

Run (PowerShell): start the server, watch startup log, then stop it. Use the project's run pattern (e.g. `go run . 2>&1 | Select-String -Pattern "relationship|Relationship|panic|mobs.LoadDataFiles"`).
Expected: `mobs.LoadDataFiles() loadedCount=...` line present; **no panic**; no `relationships:` warnings naming ids 333/335/337/338/343/344/347/349/113/334/336/341/345/346/355 (a warning about an unknown id means a typo). The cross-zone edge to 113 must NOT warn (Maren exists).

- [ ] **Step 11: Commit**
```bash
git add _datafiles/world/dogmud/mobs/stillwater/333-innkeeper_sigrid.yaml \
        _datafiles/world/dogmud/mobs/stillwater/335-constable_drunn.yaml \
        _datafiles/world/dogmud/mobs/stillwater/337-smith_brindle.yaml \
        _datafiles/world/dogmud/mobs/stillwater/338-apothecary_ilsa.yaml \
        _datafiles/world/dogmud/mobs/stillwater/343-old_fisherman_hodder.yaml \
        _datafiles/world/dogmud/mobs/stillwater/344-temple_priest_seren.yaml \
        _datafiles/world/dogmud/mobs/stillwater/347-ulla.yaml \
        _datafiles/world/dogmud/mobs/stillwater/349-old_cottager_gyda.yaml
git commit -m "content(stillwater): relationship graph (6.1 Section A)"
```

---

## Task 2: Facts seed + knowledge + gossiper expansion (Section C)

**Files:**
- Modify: `_datafiles/world/dogmud/facts.yaml` (append 5 facts)
- Modify (add `knows_facts:`): mobs 336, 337, 340, 342, 343, 344, 346, 347, 355, 335
- Modify (add `- gossiper` to `groups:`): 353, 354

- [ ] **Step 1: Append 5 standing facts to `facts.yaml`**

Add under the existing `facts:` list (sibling entries to `test-mayor`). `declared_round: 0` reads as long-standing:
```yaml
    - id: stillwater-lake-decline
      description: Boats return half-empty from the lake; nets come up shredded by something out of the caves.
      significance: 2
      declared_round: 0
      tags:
        - stillwater
        - crisis
      status: active
    - id: stillwater-voss-death
      description: Elgar Voss was lost to the lake twelve years ago; his body was never recovered.
      significance: 1
      declared_round: 0
      tags:
        - stillwater
        - history
      status: active
    - id: stillwater-spiral-motif
      description: A spiral older than the Chrysalis order is carved on the temple, the garden marker, the old chapel ruin, and the wardstones.
      significance: 1
      declared_round: 0
      tags:
        - stillwater
        - lore
      status: active
    - id: stillwater-cave-creatures
      description: Drowned hunters and skitter-shrimp from the deep caves have pushed into the shallows.
      significance: 2
      declared_round: 0
      tags:
        - stillwater
        - crisis
      status: active
    - id: stillwater-pearl-divers-gone
      description: The black-pearl divers stopped coming years ago; folk say the water has gone strange.
      significance: 1
      declared_round: 0
      tags:
        - stillwater
        - history
      status: active
```

- [ ] **Step 2: Add `knows_facts:` to each NPC** (place after `groups:`/`relationships:`). Author exactly:

Arn (342):
```yaml
knows_facts:
  - stillwater-lake-decline
  - stillwater-cave-creatures
```
Hodder (343):
```yaml
knows_facts:
  - stillwater-lake-decline
  - stillwater-cave-creatures
  - stillwater-voss-death
  - stillwater-pearl-divers-gone
```
Tov (336):
```yaml
knows_facts:
  - stillwater-lake-decline
  - stillwater-cave-creatures
```
Luc (346):
```yaml
knows_facts:
  - stillwater-lake-decline
  - stillwater-cave-creatures
```
Drunn (335):
```yaml
knows_facts:
  - stillwater-lake-decline
  - stillwater-cave-creatures
```
Ulla (347):
```yaml
knows_facts:
  - stillwater-voss-death
```
Vella (355):
```yaml
knows_facts:
  - stillwater-voss-death
  - stillwater-spiral-motif
```
Brindle (337):
```yaml
knows_facts:
  - stillwater-voss-death
```
Seren (344):
```yaml
knows_facts:
  - stillwater-voss-death
  - stillwater-spiral-motif
```
Kess (340):
```yaml
knows_facts:
  - stillwater-pearl-divers-gone
```

- [ ] **Step 3: Add `- gossiper` to groups** on Fenwick (353) and Oswin (354). Example for 354 — change:
```yaml
groups:
  - humanoid
  - stillwater_citizens
```
to:
```yaml
groups:
  - humanoid
  - gossiper
  - stillwater_citizens
```
(Do the same on 353.)

- [ ] **Step 4: Boot and check facts load**

Run the server, grep startup for `facts|panic`.
Expected: no panic; `knows_facts` for the 5 seeded ids produce **no** "unknown fact id" warnings (a warning here means an id typo between Step 1 and Step 2). Confirm `facts.LoadFromMobs` runs.

- [ ] **Step 5: Commit**
```bash
git add _datafiles/world/dogmud/facts.yaml _datafiles/world/dogmud/mobs/stillwater/
git commit -m "content(stillwater): seed facts + knows_facts + gossiper expansion (6.1 Section C)"
```

---

## Task 3: Schedules (Section B) — 8 anchors

**Files:**
- Create 8 schedules under `_datafiles/world/dogmud/schedules/stillwater/`:
  `sigrid.yaml`, `neva.yaml`, `brindle.yaml`, `seren.yaml`, `arn.yaml`, `ilsa.yaml`, `bram.yaml`, `vella.yaml`
- Modify each of the 8 anchor mob YAMLs to add `schedule_id: <id>`.

- [ ] **Step 1: Create `schedules/stillwater/sigrid.yaml`**
```yaml
id: sigrid
description: "Innkeeper Sigrid: bar prep at dawn, service through the day, evening rush, sleeps in the loft."
segments:
  - start: 6
    end: 10
    target_room: 4103
    activity: ""
    idlecommands:
      - emote wipes down the bar with a grey rag.
      - emote sets the morning's loaves out under a cloth.
      - say Early yet. Kettle's on if you want it.
  - start: 10
    end: 18
    target_room: 4103
    activity: ""
    idlecommands:
      - emote draws a measure of ale without looking at the tap.
      - emote keeps half an eye on the door.
      - say Sit where you like. It's quiet enough.
  - start: 18
    end: 23
    target_room: 4103
    activity: ""
    idlecommands:
      - emote leans on the bar and listens more than she talks.
      - say Mind your tab, now.
      - emote nods to a regular by the hearth.
  - start: 23
    end: 6
    target_room: 4104
    activity: sleeping
    idlecommands:
      - emote breathes slow and even in the dark of the loft.
```

- [ ] **Step 2: Create `schedules/stillwater/neva.yaml`**
```yaml
id: neva
description: "Barmaid Neva: opens the room, serves through day and evening, sleeps in the loft."
segments:
  - start: 7
    end: 11
    target_room: 4103
    activity: ""
    idlecommands:
      - emote stacks clean tankards behind the bar.
      - emote sweeps last night's rushes toward the door.
      - say Morning. Sigrid's in the back.
  - start: 11
    end: 18
    target_room: 4103
    activity: ""
    idlecommands:
      - emote carries plates two to a hand.
      - emote pauses to look out the window at the lake.
      - say Catch of the day's the catch of the week, truth be told.
  - start: 18
    end: 23
    target_room: 4103
    activity: ""
    idlecommands:
      - emote weaves between the crowded tables.
      - say Last orders soon, lads.
      - emote wipes a spill before it spreads.
  - start: 23
    end: 7
    target_room: 4104
    activity: sleeping
    idlecommands:
      - emote sleeps curled near the loft window.
```

- [ ] **Step 3: Create `schedules/stillwater/brindle.yaml`**
```yaml
id: brindle
description: "Smith Brindle: wakes at the forge, works lake-iron by day, drinks quietly in the evening, sleeps by the cooling anvil."
segments:
  - start: 6
    end: 9
    target_room: 4106
    activity: ""
    idlecommands:
      - emote rakes the banked coals back to life.
      - emote tests the bellows with a slow pull.
      - say Fire first. Everything else after.
  - start: 9
    end: 18
    target_room: 4106
    activity: craft
    idlecommands:
      - emote hammers a glowing bar in steady, unhurried strokes.
      - emote quenches the work with a hiss and a cloud of steam.
      - emote glances at the half-finished hook-spear and says nothing.
  - start: 18
    end: 22
    target_room: 4103
    activity: ""
    idlecommands:
      - emote nurses a single tankard in the corner.
      - say Long enough day. No need to make it longer.
      - emote flexes a soot-stained hand.
  - start: 22
    end: 6
    target_room: 4106
    activity: sleeping
    idlecommands:
      - emote sleeps on the cot beside the cooling anvil.
```

- [ ] **Step 4: Create `schedules/stillwater/seren.yaml`**
```yaml
id: seren
description: "Temple Priest Seren: morning service, tends the altar by day, evening service, sleeps in the temple cell."
segments:
  - start: 5
    end: 9
    target_room: 4123
    activity: ""
    idlecommands:
      - emote lights the morning incense at the altar.
      - emote kneels a while in the grey early light.
      - say The water hears, whether or not it answers.
  - start: 9
    end: 18
    target_room: 4123
    activity: ""
    idlecommands:
      - emote tends the candles and trims their wicks.
      - emote traces the old spiral worn into the pillar, thoughtful.
      - say Confession's open, if anyone's carrying weight.
  - start: 18
    end: 21
    target_room: 4123
    activity: ""
    idlecommands:
      - emote leads the evening rite in a low, even voice.
      - emote marks the absent places in the pews without comment.
  - start: 21
    end: 5
    target_room: 4123
    activity: sleeping
    idlecommands:
      - emote rests in the narrow cell behind the altar.
```

- [ ] **Step 5: Create `schedules/stillwater/arn.yaml`**
```yaml
id: arn
description: "Dock Master Arn: opens the docks, works the ledger and disputes by day, walks the promenade at dusk, sleeps at the dockhouse."
segments:
  - start: 6
    end: 9
    target_room: 4116
    activity: ""
    idlecommands:
      - emote unlocks the gear shed and counts the drying racks.
      - emote scans the water for the early boats.
      - say Three out, two back so far. We'll see.
  - start: 9
    end: 17
    target_room: 4116
    activity: ""
    idlecommands:
      - emote runs a thumb down the boat ledger.
      - emote settles a quarrel over a berth with a flat word.
      - say Write it down or it didn't happen. That's the rule.
  - start: 17
    end: 20
    target_room: 4113
    activity: ""
    idlecommands:
      - emote walks the promenade with his hands behind his back.
      - emote watches the light go long across the lake.
      - say Quieter than it used to be out here.
  - start: 20
    end: 6
    target_room: 4116
    activity: sleeping
    idlecommands:
      - emote sleeps in the cramped room above the dock office.
```

- [ ] **Step 6: Create `schedules/stillwater/ilsa.yaml`**
```yaml
id: ilsa
description: "Apothecary Ilsa: bottles tinctures in the morning, restocks by afternoon, closes at dusk, sleeps in the alcove."
segments:
  - start: 6
    end: 9
    target_room: 4125
    activity: craft
    idlecommands:
      - emote decants a cooled tincture into small clay bottles.
      - emote labels each stopper in a cramped, careful hand.
      - say Morning's the time for the delicate work.
  - start: 9
    end: 15
    target_room: 4125
    activity: craft
    idlecommands:
      - emote grinds dried marsh-willow to a green dust.
      - emote frowns at a near-empty jar of lake-mint.
      - say Short on half of what I need since the caves turned.
  - start: 15
    end: 19
    target_room: 4125
    activity: ""
    idlecommands:
      - emote wipes down the worktable and stoppers the jars.
      - say Come back tomorrow if it's not urgent.
  - start: 19
    end: 6
    target_room: 4125
    activity: sleeping
    idlecommands:
      - emote sleeps among the faint smell of crushed herbs.
```

- [ ] **Step 7: Create `schedules/stillwater/bram.yaml`**
```yaml
id: bram
description: "Miller Bram: starts the wheel at dawn, grinds grain by day, banks the wheel at dusk, sleeps at the mill."
segments:
  - start: 5
    end: 9
    target_room: 4135
    activity: ""
    idlecommands:
      - emote opens the sluice and the wheel groans into motion.
      - emote brushes old flour from the hopper.
      - say Water's good and high today. Good grinding.
  - start: 9
    end: 17
    target_room: 4135
    activity: ""
    idlecommands:
      - emote feeds grain into the turning stones.
      - emote hefts a full flour sack onto the stack with a grunt.
      - say Bakehouse'll want this within the hour.
  - start: 17
    end: 20
    target_room: 4135
    activity: ""
    idlecommands:
      - emote closes the sluice and the wheel slows to a stop.
      - emote sweeps the floury floor by lantern light.
  - start: 20
    end: 5
    target_room: 4135
    activity: sleeping
    idlecommands:
      - emote sleeps to the drip of the stilled wheel.
```

- [ ] **Step 8: Create `schedules/stillwater/vella.yaml`**
```yaml
id: vella
description: "Mistress Vella Thorne: stillroom work by day, sees patients, visits Ulla in the evening, sleeps at the cottage."
segments:
  - start: 6
    end: 10
    target_room: 4136
    activity: craft
    idlecommands:
      - emote sets the still to a slow simmer.
      - emote checks the patient bench is clean and ready.
      - say Earliest hours are for the work that can't wait.
  - start: 10
    end: 16
    target_room: 4136
    activity: craft
    idlecommands:
      - emote strains a dark infusion through clean cloth.
      - emote dresses a fisherman's gashed arm without fuss.
      - say Hold still. I've stitched worse than you.
  - start: 16
    end: 20
    target_room: 4137
    activity: ""
    idlecommands:
      - emote sits with Ulla in the quiet parlor.
      - emote pours two cups and says little.
      - say Some evenings the sitting is the medicine.
  - start: 20
    end: 6
    target_room: 4136
    activity: sleeping
    idlecommands:
      - emote sleeps by the banked hearth of the cottage.
```

- [ ] **Step 9: Add `schedule_id:` to the 8 anchor mob YAMLs** (place near `groups:`):
  - 333: `schedule_id: sigrid`
  - 334: `schedule_id: neva`
  - 337: `schedule_id: brindle`
  - 344: `schedule_id: seren`
  - 342: `schedule_id: arn`
  - 338: `schedule_id: ilsa`
  - 348: `schedule_id: bram`
  - 355: `schedule_id: vella`

- [ ] **Step 10: Build**

Run: `go build ./...`
Expected: exit 0.

- [ ] **Step 11: Wipe instance saves (stale saves shadow schedule_id)**

Run (PowerShell):
```powershell
Remove-Item -Recurse -Force _datafiles/world/dogmud/mobs.instances/*, _datafiles/world/dogmud/rooms.instances/* -ErrorAction SilentlyContinue
```

- [ ] **Step 12: Boot and check schedules load (PANIC-prone)**

Run the server and watch the full startup.
Expected: **no panic.** Specifically none of: `schedule ... did not cover hour`, `target_room ... does not exist`, `GetPath` failure between segments, `unresolved schedule_id`. Confirm a clean `mobs.LoadDataFiles() loadedCount=...`. If a path panic names a room pair, recheck that schedule's `target_room`s against the verified-connectivity list.

- [ ] **Step 13: Commit**
```bash
git add _datafiles/world/dogmud/schedules/stillwater/ _datafiles/world/dogmud/mobs/stillwater/
git commit -m "content(stillwater): 8 NPC daily schedules (6.1 Section B)"
```

---

## Task 4: Conversation type pools (Section D.1)

**Files (create):**
- `_datafiles/world/dogmud/conversations/types/employer.yaml`
- `_datafiles/world/dogmud/conversations/types/employee.yaml`
- `_datafiles/world/dogmud/conversations/types/family.yaml`

(Role-agnostic A/B. Both employer and employee pools are required — see Verified facts.)

- [ ] **Step 1: Create `types/employer.yaml`**
```yaml
id: employer
description: |
  Generic banter between someone and the person who works for or
  under them — work talk, small instructions, dry approval.
exchanges:
  - lines:
      - speaker: A
        text: "Slow morning?"
      - speaker: B
        text: "Slow enough. I'll find work if you'd rather."
      - speaker: A
        text: "It'll find you. It always does."
  - lines:
      - speaker: A
        text: "You did well yesterday."
      - speaker: B
        text: "Didn't feel like it."
      - speaker: A
        text: "That's usually when it counts."
  - lines:
      - speaker: A
        text: "Take your break when the room thins out."
      - speaker: B
        text: "And if it doesn't thin out?"
      - speaker: A
        text: "Then we're both lucky and both tired."
  - lines:
      - speaker: A
        text: "Anything I should know about?"
      - speaker: B
        text: "Nothing that won't keep till morning."
  - lines:
      - speaker: A
        text: "You're not paid to worry. That's my part."
      - speaker: B
        text: "Somebody has to help you with it."
      - speaker: A
        text: "Hmph. Fair."
  - lines:
      - speaker: A
        text: "Long as the work's honest, the day's honest."
      - speaker: B
        text: "You say that every morning."
      - speaker: A
        text: "And it's true every morning."
```

- [ ] **Step 2: Create `types/employee.yaml`**
```yaml
id: employee
description: |
  Generic banter from the perspective of someone who works for
  another — deference, gentle complaint, shared routine.
exchanges:
  - lines:
      - speaker: A
        text: "Where do you want this?"
      - speaker: B
        text: "Wherever it'll be out from underfoot."
      - speaker: A
        text: "That narrows it down."
  - lines:
      - speaker: A
        text: "Same as yesterday?"
      - speaker: B
        text: "Same as every day. That's the trade."
  - lines:
      - speaker: A
        text: "My back's telling me it's near closing."
      - speaker: B
        text: "Your back's an hour fast, as always."
  - lines:
      - speaker: A
        text: "You ever think of doing something else?"
      - speaker: B
        text: "Every winter. Then spring comes."
      - speaker: A
        text: "Aye. Then spring comes."
  - lines:
      - speaker: A
        text: "I'll lock up if you want to get on."
      - speaker: B
        text: "Good of you. I'll not argue."
  - lines:
      - speaker: A
        text: "Quiet today."
      - speaker: B
        text: "Don't say it too loud. You'll wake the custom."
```

- [ ] **Step 3: Create `types/family.yaml`**
```yaml
id: family
description: |
  Generic banter between kin — easy familiarity, old griefs left
  half-spoken, care expressed sideways. Suitable for any related pair.
exchanges:
  - lines:
      - speaker: A
        text: "You eating enough?"
      - speaker: B
        text: "You ask me that every time."
      - speaker: A
        text: "And every time you dodge it."
  - lines:
      - speaker: A
        text: "It's good to sit a while."
      - speaker: B
        text: "It is. We don't, often enough."
  - lines:
      - speaker: A
        text: "I thought of him today."
      - speaker: B
        text: "I think of him most days."
      - speaker: A
        text: "Aye. I know you do."
  - lines:
      - speaker: A
        text: "You'd tell me if something was wrong."
      - speaker: B
        text: "Would I?"
      - speaker: A
        text: "...You'd better."
  - lines:
      - speaker: A
        text: "The years go quick now."
      - speaker: B
        text: "Quicker than they ought."
  - lines:
      - speaker: A
        text: "Come by more. The door's always open."
      - speaker: B
        text: "I will. I mean it this time."
```

- [ ] **Step 4: Boot and check conversation pools load**

Run the server, grep startup for `conversation|panic`.
Expected: no panic; pools for `employer`, `employee`, `family` load alongside the existing `friend`/`rival`. (Loader validates pool id ∈ known relationship types — these three are valid.)

- [ ] **Step 5: Commit**
```bash
git add _datafiles/world/dogmud/conversations/types/
git commit -m "content(conversations): employer/employee/family type pools (6.1 Section D.1)"
```

---

## Task 5: Conversation pair overrides (Section D.2)

**Files (create):**
- `_datafiles/world/dogmud/conversations/pairs/333_334.yaml` (Sigrid/Neva)
- `_datafiles/world/dogmud/conversations/pairs/336_343.yaml` (Tov/Hodder)
- `_datafiles/world/dogmud/conversations/pairs/338_355.yaml` (Ilsa/Vella)
- `_datafiles/world/dogmud/conversations/pairs/347_355.yaml` (Ulla/Vella)

Filename = `<lowerId>_<higherId>.yaml`. Role-agnostic A/B.

- [ ] **Step 1: Create `pairs/333_334.yaml`**
```yaml
id: sigrid_and_neva
mob_a: 333
mob_b: 334
exchanges:
  - lines:
      - speaker: A
        text: "Wulf send that ale round yet?"
      - speaker: B
        text: "Says tomorrow. Said tomorrow yesterday too."
      - speaker: A
        text: "That man and his tomorrows."
  - lines:
      - speaker: A
        text: "Hodder was in early, mending by the window."
      - speaker: B
        text: "He always knows the water before the boats do."
      - speaker: A
        text: "Forty years'll do that."
  - lines:
      - speaker: A
        text: "Keep the corner table clear tonight."
      - speaker: B
        text: "For the constable?"
      - speaker: A
        text: "He likes to see the door. Let him."
  - lines:
      - speaker: A
        text: "Fewer boats out than last spring."
      - speaker: B
        text: "Fewer every spring, seems like."
      - speaker: A
        text: "Pour 'em strong, then. It's that kind of season."
```

- [ ] **Step 2: Create `pairs/336_343.yaml`**
```yaml
id: tov_and_hodder
mob_a: 336
mob_b: 343
exchanges:
  - lines:
      - speaker: A
        text: "Your knots still hold better than mine."
      - speaker: B
        text: "Taught you the same knots, boy."
      - speaker: A
        text: "Taught. Past tense. I had a good teacher."
  - lines:
      - speaker: A
        text: "Catch was poor again."
      - speaker: B
        text: "It's not the catch. It's what's eating the catch."
      - speaker: A
        text: "Don't let Arn hear you. He'll want it in the ledger."
  - lines:
      - speaker: A
        text: "You remember when the deep shelves were full?"
      - speaker: B
        text: "I remember men who fished them. Most are gone now."
      - speaker: A
        text: "..."
      - speaker: B
        text: "Mend your nets, lad. That much we can still do."
  - lines:
      - speaker: A
        text: "Cold coming off the water today."
      - speaker: B
        text: "Wrong kind of cold. You'll learn the difference."
```

- [ ] **Step 3: Create `pairs/338_355.yaml`**
```yaml
id: ilsa_and_vella
mob_a: 338
mob_b: 355
exchanges:
  - lines:
      - speaker: A
        text: "I'm out of lake-mint again."
      - speaker: B
        text: "Try pond-willow at half the dose. It'll hold."
      - speaker: A
        text: "Crude, but it'll hold. You're right."
  - lines:
      - speaker: A
        text: "Sent the Aldous boy to you. Beyond my stitching."
      - speaker: B
        text: "You stitched fine. He just bleeds like his father."
      - speaker: A
        text: "Runs in the blood, then. Of course it does."
  - lines:
      - speaker: A
        text: "Different methods, you and I."
      - speaker: B
        text: "Same patients, though. That's what matters."
  - lines:
      - speaker: A
        text: "How's Ulla, truly?"
      - speaker: B
        text: "Some days the chair by the window is enough. Some days it isn't."
      - speaker: A
        text: "You're good to sit with her."
```

- [ ] **Step 4: Create `pairs/347_355.yaml`** (the grief pair — gentle, never names Elgar outright)
```yaml
id: ulla_and_vella
mob_a: 347
mob_b: 355
exchanges:
  - lines:
      - speaker: A
        text: "You didn't have to come tonight."
      - speaker: B
        text: "I know. I came anyway."
  - lines:
      - speaker: A
        text: "Maren wrote. She asks after you."
      - speaker: B
        text: "She has her mother's hand. Tell her I'm well."
      - speaker: A
        text: "I'll tell her you're stubborn. She'll know it's the same thing."
  - lines:
      - speaker: A
        text: "Twelve years, this autumn."
      - speaker: B
        text: "I know what autumn it is."
      - speaker: A
        text: "...I know you do."
  - lines:
      - speaker: A
        text: "The workshop's still locked."
      - speaker: B
        text: "Leave it locked, if it's easier."
      - speaker: A
        text: "Nothing about it is easier. But I'll leave it."
```

- [ ] **Step 5: Boot and check pair overrides load**

Run the server, grep startup for `conversation|pair|panic`.
Expected: no panic; the 4 pair files load (alongside existing `116_117`). A warning about a pair referencing an unknown relationship would indicate the Task 1 edge for that pair is missing — confirm the relationship edges from Task 1 exist for 333/334, 336/343, 338/355, 347/355.

- [ ] **Step 6: Commit**
```bash
git add _datafiles/world/dogmud/conversations/pairs/
git commit -m "content(conversations): Stillwater pair overrides (6.1 Section D.2)"
```

---

## Task 6: Final integration boot + docs

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md` (mark 6.1 Done in tracker + mini-brief; bump roll-up)
- Modify: `PATCH_NOTES.md` (dated entry)

- [ ] **Step 1: Clean instance saves + full boot**
```powershell
Remove-Item -Recurse -Force _datafiles/world/dogmud/mobs.instances/*, _datafiles/world/dogmud/rooms.instances/* -ErrorAction SilentlyContinue
```
Then boot the full server.
Expected: clean boot, no panics, normal `loadedCount` lines for mobs/quests/etc.

- [ ] **Step 2: Run the test suite**

Run: `go test ./...`
Expected: all packages pass (no Go was changed; this guards against accidental edits).

- [ ] **Step 3: Update `MOB_ALIVENESS_ROADMAP.md`**

In the progress-tracker table, change the 6.1 row Status from `Not started` to `Done`. In the 6.1 mini-brief, set `**Status:** Done (2026-06-03)` and append a `**Shipped:**` paragraph summarizing: relationship graph (~10 edges across 12 NPCs incl. cross-zone Ulla→Maren), 8 schedules, 5 seeded facts + role-gated knows_facts + 2 gossiper additions (Fenwick/Oswin), 3 new conversation type-pools (employer/employee/family) + 4 pair overrides, zero new rooms. Bump the roll-up count (`36 / 42 done`). Note manual in-game smoke deferred to user.

- [ ] **Step 4: Update `PATCH_NOTES.md`** with a dated 2026-06-03 entry describing the Stillwater town-flavor pass in player-facing terms (NPCs now keep daily routines, talk to each other, and remember the town's troubles).

- [ ] **Step 5: Commit**
```bash
git add MOB_ALIVENESS_ROADMAP.md PATCH_NOTES.md
git commit -m "docs(aliveness): mark 6.1 Stillwater town-flavor done + patch notes"
```

---

## Manual smoke (deferred to user)

Per the chunk-2.x precedent, in-game validation is the user's. Suggested walk:
1. Wipe instance saves, boot, log in near Lakefront Square.
2. Use admin time-set to step across a day; confirm the 8 anchors move to their segment `target_room`s and that sleepers show as asleep at night.
3. Idle in the tavern (4103) and the healer's cottage/parlor (4136/4137) to catch NPC↔NPC conversations; confirm the Ulla/Vella parlor exchange fires in the 16:00–20:00 window.
4. `relationship between 347 355`, `relationship show 343`, `fact list`, `fact awareness 343` (admin) to confirm substrate state.
5. Watch for gossipers (Neva/Hodder/Gyda/Fenwick/Oswin) surfacing the seeded facts in idle gossip.

## Self-review notes

- **Spec coverage:** Section A → Task 1; Section B → Task 3; Section C → Task 2; Section D.1 → Task 4; Section D.2 → Task 5; Section E (no new rooms) → honored throughout (all schedule targets are existing, connectivity-verified rooms); validation/smoke → Tasks' boot steps + Task 6 + deferred manual smoke.
- **No placeholders:** every file's full content is inline.
- **Type consistency:** field names (`to`/`type`/`subtype`, `knows_facts`, `schedule_id`, `target_room`/`activity`, `mob_a`/`mob_b`/`exchanges`/`speaker`/`text`, fact `id`/`description`/`significance`/`declared_round`/`tags`/`status`) match the verified struct tags throughout.
