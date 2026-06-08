# Mob Aliveness 6.5b — Towns Batch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the two micro-settlements Ashwick and Watcher's Crossing to life — relationships, medium-depth schedules, bespoke + type-pool conversations, and fact/gossip — mirroring the 6.1/6.3 framework at small scale.

**Architecture:** Pure YAML content. Mob templates gain `relationships:`, `schedule_id:`, `knows_facts:`, and `gossiper` group tags; new schedule files steer NPCs between real settlement rooms; new conversation-pair files add signature exchanges; new `facts.yaml` entries seed gossip. The schedule/conversation/facts loaders validate refs and **panic at boot** on coverage gaps, unreachable rooms, or unresolved ids — so the boot test is the verification gate (no Go unit tests; matches the codebase).

**Tech Stack:** YAML data files; the schedules, conversations, facts loaders.

**Spec:** `docs/superpowers/specs/2026-06-05-mob-aliveness-6.5b-towns-batch-design.md`

---

## Reference: verified schemas

**Schedule** (`_datafiles/world/dogmud/schedules/<zone>/<id>.yaml`) — segments MUST cover all 24h (wrap allowed, e.g. 22→6); every `target_room` must exist + be reachable. `activity` ∈ `craft` | `sleeping` | `patrol` | `""` (empty = ordinary idle):
```yaml
id: <id>
description: "..."
segments:
  - start: 6
    end: 18
    target_room: <roomId>
    activity: ""
    idlecommands:
      - emote ...
      - say ...
  - start: 18
    end: 6
    target_room: <roomId>
    activity: sleeping
    idlecommands:
      - emote sleeps ...
```
Mob references it via a top-level `schedule_id: <id>` field.

**Relationships** (on a mob template; engine auto-mirrors symmetric same-type and employer↔employee, so author each edge ONCE):
```yaml
relationships:
  - to: <otherMobId>
    type: friend   # friend|rival|family|lover|employer|employee
    subtype: <flavor string>
```

**Facts** (`_datafiles/world/dogmud/facts.yaml`, append under `facts:`):
```yaml
    - id: <slug>
      description: <one sentence, public-knowledge level>
      significance: 1   # 1 minor, 2 notable
      declared_round: 0
      tags:
        - <zone>
        - <category>
      status: active
```
Mobs that know a fact list it under `knows_facts: [<id>, ...]`; mobs that gossip facts to players carry `gossiper` in their `groups:` list.

**Conversation pair** (`_datafiles/world/dogmud/conversations/pairs/<loMobId>_<hiMobId>.yaml`; speaker A = initiator role, B = partner role — role-agnostic & swap-safe):
```yaml
id: <name>
mob_a: <mobId>
mob_b: <mobId>
exchanges:
  - lines:
      - speaker: A
        text: "..."
      - speaker: B
        text: "..."
```

## Reference: NPC ids + rooms

Ashwick: Delia 259 (cottage 4027 / garden 4028), Deacon Ferris 260 (chapel 4022 / ritual circle 4021), Farmer Hesta 261 (farmstead 4020 / south fields 4025), The Forager 262 (herb clearing 4032 / camp 4031). Social hub: Central Green 4017.
Watcher's Crossing: Innkeeper Tolva 84 (inn 423), Merchant Brecca 85 (trading post 424), Toll Collector Harn 86 (tollhouse 421), Traveling Merchant 88 (inn 423).

---

## Task 1: Relationships

Author each edge once (auto-mirror handles the reverse). Add a `relationships:` block to these 4 mobs (the other 4 receive their edges via mirroring).

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/ashwick/259-delia.yaml`
- Modify: `_datafiles/world/dogmud/mobs/ashwick/260-deacon_ferris.yaml`
- Modify: `_datafiles/world/dogmud/mobs/watchers_crossing/84-innkeeper_tolva.yaml`
- Modify: `_datafiles/world/dogmud/mobs/watchers_crossing/85-merchant_brecca.yaml`

- [ ] **Step 1: Add to `259-delia.yaml`** (top-level, e.g. after `groups:`):
```yaml
relationships:
  - to: 261
    type: friend
    subtype: neighbor
  - to: 260
    type: friend
    subtype: village
  - to: 262
    type: employer
    subtype: herb-work
```

- [ ] **Step 2: Add to `260-deacon_ferris.yaml`:**
```yaml
relationships:
  - to: 261
    type: friend
    subtype: congregation
```

- [ ] **Step 3: Add to `84-innkeeper_tolva.yaml`:**
```yaml
relationships:
  - to: 85
    type: friend
    subtype: crossing
  - to: 86
    type: friend
    subtype: crossing
```

- [ ] **Step 4: Add to `85-merchant_brecca.yaml`:**
```yaml
relationships:
  - to: 86
    type: rival
    subtype: toll
  - to: 88
    type: rival
    subtype: trade
```

- [ ] **Step 5: Build + commit**
```bash
go build ./...
git add _datafiles/world/dogmud/mobs/ashwick/259-delia.yaml _datafiles/world/dogmud/mobs/ashwick/260-deacon_ferris.yaml _datafiles/world/dogmud/mobs/watchers_crossing/84-innkeeper_tolva.yaml _datafiles/world/dogmud/mobs/watchers_crossing/85-merchant_brecca.yaml
git commit -m "feat(6.5b): relationship edges for Ashwick + Watcher's Crossing NPCs"
```

---

## Task 2: Facts + knows_facts + gossiper tags

**Files:**
- Modify: `_datafiles/world/dogmud/facts.yaml`
- Modify: the 7 gossiper NPC mob YAMLs (259, 260, 261 ashwick; 84, 85, 86, 88 watchers). The Forager (262) is private — NOT a gossiper, NOT a knower.

- [ ] **Step 1: Append 4 facts to `facts.yaml`** (under the `facts:` list):
```yaml
    - id: ashwick-deep-woods-wolves
      description: Wolves have been prowling out of the Deep Woods, taking livestock from the outlying farms.
      significance: 2
      declared_round: 0
      tags:
        - ashwick
        - crisis
      status: active
    - id: ashwick-newcomer-forager
      description: Delia has taken in a wary newcomer, a skilled herb-gatherer no one knows the past of.
      significance: 1
      declared_round: 0
      tags:
        - ashwick
        - people
      status: active
    - id: watchers-road-bandits
      description: Bandits have grown bold on the trade roads; travelers and caravans cross with one hand on their purse.
      significance: 2
      declared_round: 0
      tags:
        - watchers_crossing
        - crisis
        - bandits
      status: active
    - id: watchers-toll-grumbling
      description: Folk grumble over the bridge toll, though the collector only does as he is bid.
      significance: 1
      declared_round: 0
      tags:
        - watchers_crossing
        - people
      status: active
```

- [ ] **Step 2: Add `knows_facts:` + `gossiper` group to the Ashwick gossipers**
For `259-delia.yaml`, `260-deacon_ferris.yaml`, `261-farmer_hesta.yaml`: append `- gossiper` to their `groups:` list (preserve existing entries incl. `ashwick_villagers`), and add:
```yaml
knows_facts:
  - ashwick-deep-woods-wolves
  - ashwick-newcomer-forager
```

- [ ] **Step 3: Add `knows_facts:` + `gossiper` to the Watcher's gossipers**
For `84-innkeeper_tolva.yaml`, `85-merchant_brecca.yaml`, `86-toll_collector_harn.yaml`, `88-traveling_merchant.yaml`: append `- gossiper` to `groups:` (preserve existing). knows_facts:
- Tolva (84), Brecca (85): both facts:
```yaml
knows_facts:
  - watchers-road-bandits
  - watchers-toll-grumbling
```
- Harn (86), Traveling Merchant (88): bandits fact only:
```yaml
knows_facts:
  - watchers-road-bandits
```

- [ ] **Step 4: Build + commit**
```bash
go build ./...
git add _datafiles/world/dogmud/facts.yaml _datafiles/world/dogmud/mobs/ashwick/ _datafiles/world/dogmud/mobs/watchers_crossing/
git commit -m "feat(6.5b): 4 settlement facts + knows_facts + gossiper tags"
```

---

## Task 3: Ashwick schedules

Author 4 schedule files + add `schedule_id:` to each Ashwick NPC. Each NPC's idlecommands must be authored IN that NPC's established voice (read the mob's `description:` and existing `idlecommands:` first) — 3-4 lines per segment, mixing `emote` and `say`. The segment hours/rooms/activity below are exact; the example idlecommands show the voice — expand each segment to 3-4 lines.

Voice notes: Delia = brisk herbalist, plant-stained, too much to do. Deacon Ferris = kind, careful, resonant voice. Farmer Hesta = broad, steady, unimpressed, heard-it-all. The Forager = gaunt, wary, guarded, quick precise hands.

**Files:**
- Create: `_datafiles/world/dogmud/schedules/ashwick/ashwick_delia.yaml`
- Create: `_datafiles/world/dogmud/schedules/ashwick/ashwick_ferris.yaml`
- Create: `_datafiles/world/dogmud/schedules/ashwick/ashwick_hesta.yaml`
- Create: `_datafiles/world/dogmud/schedules/ashwick/ashwick_forager.yaml`
- Modify: the 4 ashwick NPC mob YAMLs (add `schedule_id:`)

- [ ] **Step 1: `ashwick_delia.yaml`** (cottage→garden→green→sleep; the 13–17 garden block co-locates with the Forager's delivery beat):
```yaml
id: ashwick_delia
description: "Delia: morning prep in the cottage, afternoon in the garden, evening on the green, sleeps among the drying herbs."
segments:
  - start: 6
    end: 13
    target_room: 4027
    activity: craft
    idlecommands:
      - emote sorts dried bundles by smell, brisk and sure.
      - say No two seasons grow the same. You learn the plant, not the page.
  - start: 13
    end: 17
    target_room: 4028
    activity: craft
    idlecommands:
      - emote pinches back a seed-head and tucks it into her apron.
      - say Mind where you step — half of this bed isn't in any book.
  - start: 17
    end: 21
    target_room: 4017
    activity: ""
    idlecommands:
      - emote rolls a stiff shoulder, watching the light go.
  - start: 21
    end: 6
    target_room: 4027
    activity: sleeping
    idlecommands:
      - emote sleeps amid the green smell of drying herbs.
```

- [ ] **Step 2: `ashwick_ferris.yaml`** (chapel→ritual circle→chapel→green→sleep at chapel):
```yaml
id: ashwick_ferris
description: "Deacon Ferris: tends the chapel, keeps the ritual circle, joins the green at dusk, sleeps by the altar."
segments:
  - start: 6
    end: 12
    target_room: 4022
    activity: ""
    idlecommands:
      - emote straightens the worn cloth on the little altar.
      - say Come in out of the wind. The door is always open here.
  - start: 12
    end: 14
    target_room: 4021
    activity: ""
    idlecommands:
      - emote walks the old ring slowly, naming each stone under his breath.
  - start: 14
    end: 17
    target_room: 4022
    activity: ""
    idlecommands:
      - emote trims a guttering candle with two careful fingers.
  - start: 17
    end: 21
    target_room: 4017
    activity: ""
    idlecommands:
      - emote nods a quiet greeting to each soul on the green.
  - start: 21
    end: 6
    target_room: 4022
    activity: sleeping
    idlecommands:
      - emote sleeps on a narrow cot beside the altar.
```

- [ ] **Step 3: `ashwick_hesta.yaml`** (farmstead→south fields→green→sleep):
```yaml
id: ashwick_hesta
description: "Farmer Hesta: dawn chores at the farmstead, the day in the south fields, the green at dusk, early to bed."
segments:
  - start: 6
    end: 12
    target_room: 4020
    activity: ""
    idlecommands:
      - emote hauls a feed sack onto one shoulder without slowing.
      - say Work doesn't care how you feel about it. Get on.
  - start: 12
    end: 17
    target_room: 4025
    activity: ""
    idlecommands:
      - emote leans on a hoe and squints down the furrows.
      - say Twenty years this dirt and me. It still surprises me. Rarely good.
  - start: 17
    end: 21
    target_room: 4017
    activity: ""
    idlecommands:
      - emote settles onto the green with the groan of an honest day done.
  - start: 21
    end: 6
    target_room: 4020
    activity: sleeping
    idlecommands:
      - emote sleeps hard and early, boots by the door.
```

- [ ] **Step 4: `ashwick_forager.yaml`** (herb clearing→Delia's garden delivery→camp→sleep; skips the green — stays wary):
```yaml
id: ashwick_forager
description: "The Forager: gathers in the herb clearing, brings the day's take to Delia's garden, keeps to the camp, wary always."
segments:
  - start: 6
    end: 13
    target_room: 4032
    activity: ""
    idlecommands:
      - emote moves through the clearing without bending a blade out of place.
      - emote freezes at a far sound, then keeps gathering.
  - start: 13
    end: 16
    target_room: 4028
    activity: ""
    idlecommands:
      - emote lays a sorted bundle at the garden's edge for Delia.
      - say Found the bitterroot. Three days east. I didn't stay.
  - start: 16
    end: 22
    target_room: 4031
    activity: ""
    idlecommands:
      - emote sits where the camp's back is to stone and the front to the path.
  - start: 22
    end: 6
    target_room: 4031
    activity: sleeping
    idlecommands:
      - emote sleeps lightly, one boot still on.
```

- [ ] **Step 5: Add `schedule_id:` to the 4 mobs**
- `259-delia.yaml` → `schedule_id: ashwick_delia`
- `260-deacon_ferris.yaml` → `schedule_id: ashwick_ferris`
- `261-farmer_hesta.yaml` → `schedule_id: ashwick_hesta`
- `262-the_forager.yaml` → `schedule_id: ashwick_forager`

- [ ] **Step 6: Build + commit**
```bash
go build ./...
git add _datafiles/world/dogmud/schedules/ashwick/ _datafiles/world/dogmud/mobs/ashwick/
git commit -m "feat(6.5b): Ashwick NPC schedules (work/social/home-sleep)"
```

---

## Task 4: Watcher's Crossing schedules

3 schedule files (Tolva, Brecca, Harn) + `schedule_id:`. The Traveling Merchant (88) gets NO schedule (transient — stays at the Inn). Brecca + Harn share the Inn 18–22 (enables the Harn↔Brecca conversation). Author idlecommands to voice: Tolva = warm-but-watchful innkeeper, never idle, runs the inn for decades. Brecca = sharp, measures people, military-precise post. Harn = heavyset, deliberate, meticulous ledger, apologetic about the unpopular toll.

**Files:**
- Create: `_datafiles/world/dogmud/schedules/watchers_crossing/watchers_tolva.yaml`
- Create: `_datafiles/world/dogmud/schedules/watchers_crossing/watchers_brecca.yaml`
- Create: `_datafiles/world/dogmud/schedules/watchers_crossing/watchers_harn.yaml`
- Modify: `84-innkeeper_tolva.yaml`, `85-merchant_brecca.yaml`, `86-toll_collector_harn.yaml` (add `schedule_id:`)

- [ ] **Step 1: `watchers_tolva.yaml`** (the inn all day — the social anchor; sleeps at the inn):
```yaml
id: watchers_tolva
description: "Innkeeper Tolva: runs the common room from dawn to late, misses nothing, sleeps in the back."
segments:
  - start: 6
    end: 22
    target_room: 423
    activity: ""
    idlecommands:
      - emote wipes down the bar without once looking at it.
      - say Room and a meal? Sit anywhere you can see the door.
      - emote marks a tally on the slate, watchful of the whole room.
  - start: 22
    end: 6
    target_room: 423
    activity: sleeping
    idlecommands:
      - emote sleeps in the back room, one ear to the common room still.
```

- [ ] **Step 2: `watchers_brecca.yaml`** (trading post by day, inn in the evening, sleeps at the post):
```yaml
id: watchers_brecca
description: "Merchant Brecca: the trading post by day, the inn at dusk, quarters behind the post."
segments:
  - start: 6
    end: 18
    target_room: 424
    activity: ""
    idlecommands:
      - emote tags and shelves a crate, every item in its place.
      - say Everything's priced. Haggle if you like; I priced in the haggling.
      - emote measures a customer with one glance, the goods with the next.
  - start: 18
    end: 22
    target_room: 423
    activity: ""
    idlecommands:
      - emote takes a corner table at the inn, ledger still under her arm.
  - start: 22
    end: 6
    target_room: 424
    activity: sleeping
    idlecommands:
      - emote sleeps in the quarters behind the post, door barred.
```

- [ ] **Step 3: `watchers_harn.yaml`** (tollhouse by day, inn in the evening, sleeps at the tollhouse):
```yaml
id: watchers_harn
description: "Toll Collector Harn: mans the tollbooth, unwinds at the inn, sleeps by his ledger."
segments:
  - start: 6
    end: 18
    target_room: 421
    activity: ""
    idlecommands:
      - emote records a payment in a heavy ledger, slow and exact.
      - say A copper for the bridge. I don't set the rate, friend, I only keep it.
      - emote offers an apologetic shrug to a grumbling traveler.
  - start: 18
    end: 22
    target_room: 423
    activity: ""
    idlecommands:
      - emote nurses one cup in the corner, glad to be off the booth.
  - start: 22
    end: 6
    target_room: 421
    activity: sleeping
    idlecommands:
      - emote sleeps on a cot by the tollhouse ledger.
```

- [ ] **Step 4: Add `schedule_id:` to the 3 mobs**
- `84-innkeeper_tolva.yaml` → `schedule_id: watchers_tolva`
- `85-merchant_brecca.yaml` → `schedule_id: watchers_brecca`
- `86-toll_collector_harn.yaml` → `schedule_id: watchers_harn`

- [ ] **Step 5: Build + commit**
```bash
go build ./...
git add _datafiles/world/dogmud/schedules/watchers_crossing/ _datafiles/world/dogmud/mobs/watchers_crossing/
git commit -m "feat(6.5b): Watcher's Crossing NPC schedules"
```

---

## Task 5: Bespoke conversation pairs

Two signature pairs. Role-agnostic & swap-safe (don't bake which physical mob is A vs B — the engine randomizes). 3-4 short exchanges each.

**Files:**
- Create: `_datafiles/world/dogmud/conversations/pairs/259_262.yaml` (Delia ↔ The Forager)
- Create: `_datafiles/world/dogmud/conversations/pairs/85_86.yaml` (Brecca ↔ Harn)

- [ ] **Step 1: `259_262.yaml`** (mentor warmth; the "newcomer I took in" beat, public-level):
```yaml
id: delia_and_forager
mob_a: 259
mob_b: 262
exchanges:
  - lines:
      - speaker: A
        text: "You can put your back to the wall here, you know. Nobody's coming."
      - speaker: B
        text: "Habit. It's kept me breathing."
      - speaker: A
        text: "Then keep it. But eat something too."
  - lines:
      - speaker: A
        text: "Your hands know plants that don't grow in any garden I've seen."
      - speaker: B
        text: "Someone taught me. Before."
      - speaker: A
        text: "Before's your business. The bitterroot's mine. Fair trade."
  - lines:
      - speaker: A
        text: "Folk ask where you came from."
      - speaker: B
        text: "Let them ask."
      - speaker: A
        text: "I tell them you're mine to vouch for. That ends it."
  - lines:
      - speaker: A
        text: "Stay through winter. The clearing's no place to freeze."
      - speaker: B
        text: "...I'll think on it."
      - speaker: A
        text: "Think warm. There's a cot."
```

- [ ] **Step 2: `85_86.yaml`** (the toll friction; Harn apologetic, Brecca sharp):
```yaml
id: brecca_and_harn
mob_a: 85
mob_b: 86
exchanges:
  - lines:
      - speaker: A
        text: "Half my buyers turn back at your booth, Harn."
      - speaker: B
        text: "I only collect it. I didn't set it."
      - speaker: A
        text: "Tell that to my empty shelves."
  - lines:
      - speaker: A
        text: "Waive it for my regulars. Just the regulars."
      - speaker: B
        text: "If I waived it for everyone's regulars, there'd be no bridge."
      - speaker: A
        text: "There's a thought."
  - lines:
      - speaker: A
        text: "You wrote me down twice last market-day."
      - speaker: B
        text: "I did not. I never write a man down twice."
      - speaker: A
        text: "Check the ledger, then. I'll wait. I've got nothing but time, thanks to your queue."
  - lines:
      - speaker: A
        text: "Buy you a cup, toll-man. No hard feelings."
      - speaker: B
        text: "...that's decent of you, Brecca."
      - speaker: A
        text: "It's on credit. I'll add it to your bridge."
```

- [ ] **Step 3: Commit**
```bash
git add _datafiles/world/dogmud/conversations/pairs/259_262.yaml _datafiles/world/dogmud/conversations/pairs/85_86.yaml
git commit -m "feat(6.5b): bespoke conversation pairs (Delia/Forager, Brecca/Harn)"
```

---

## Task 6: Boot-validate + smoke

- [ ] **Step 1: Build + full test**
Run: `go build ./... && go test ./...`
Expected: build clean; all packages pass.

- [ ] **Step 2: Wipe instances + boot**
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Boot the server. **Confirm no panic** from: the schedule validator (coverage gaps / unreachable `target_room` / unknown `schedule_id`), the relationships loader (unknown `to:` mob), the facts loader (unknown `knows_facts:` id), or the conversation loader (unknown mob_a/mob_b). Confirm "Server Ready".
Expected: clean boot past data-file load. Watch for `schedules.LoadDataFiles`/`conversations`/`facts` load lines without panic.

- [ ] **Step 3: In-game smoke** (admin `smoketester`; MAY be deferred to user per chunk 2.8/2.9 precedent — note which steps ran)
- Walk Ashwick across day hours: confirm Delia/Ferris/Hesta gather at Central Green (4017) in the evening; the Forager delivers to Delia's Garden (4028) mid-afternoon and stays away from the green; all sleep at night in their home rooms.
- Walk Watcher's: Brecca + Harn both at the Inn (423) in the evening; watch for the Brecca↔Harn exchange; Tolva anchored at the inn.
- `ask`/observe gossip: confirm the 4 facts surface from the gossiper NPCs.

No commit (verification only).

---

## Task 7: Roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Add/mark the 6.5b row**
Add a `6.5b` row to the Progress tracker (Status `Done (2026-06-05)`) if not present, or mark it Done. Add a 6.5b mini-brief under Phase 6 with a `**Shipped:**` bullet: relationships on 8 NPCs, 7 schedules, 4 facts + gossiper tags, 2 bespoke conversation pairs, boot-validated. Re-tally the roll-up.

- [ ] **Step 2: Commit**
```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(roadmap): mark 6.5b towns batch Done"
```

---

## Self-review notes

**Spec coverage:**
- Spec §1 Relationships → Task 1 (each edge authored once; auto-mirror).
- Spec §2 Schedules → Tasks 3 (Ashwick) + 4 (Watcher's); medium depth, real rooms, social co-location for conversations, Traveling Merchant left schedule-less.
- Spec §3 Conversations → Task 5 (bespoke 259_262 + 85_86; type pools need no files).
- Spec §4 Facts → Task 2 (4 facts + knows_facts + gossiper; Forager excluded as private).
- Spec Validation → Task 6 (boot panic-gate + smoke).
- Roadmap maintenance → Task 7.

**Placeholder check:** schedules give exact hours/rooms/activity with concrete in-voice idlecommands (implementer expands each segment to 3-4 lines in the same voice); conversation scripts + facts are given in full; no TBDs.

**Consistency check:** co-location verified — Delia(4028 @13–17) ∩ Forager(4028 @13–16); Brecca(423 @18–22) ∩ Harn(423 @18–22) ∩ Tolva(423). All schedule segments cover 24h (each NPC's segments chain start→end with a wrapping sleep segment back to 6). Conversation pair filenames use lower_higher mobid (259_262, 85_86). Fact ids referenced in knows_facts match those declared in Task 2. Gossiper set excludes the Forager (262) per spec.

**TDD note:** content validated by the boot-time loaders (panic on bad refs / schedule coverage gaps), not Go unit tests — consistent with the codebase. Task 6 is the gate.
