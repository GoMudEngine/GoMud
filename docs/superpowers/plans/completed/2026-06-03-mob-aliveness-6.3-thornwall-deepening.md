# Thornwall Deepening (6.3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply the mob-aliveness substrate (relationships, schedules, knowledge/facts, conversations) to Thornwall city as the second benchmark zone, plus a targeted gossip fix and before/after authoring notes.

**Architecture:** Pure data-file authoring against existing engine systems (mirrors chunk 6.1 Stillwater), boot-validated layer-by-layer, PLUS one small TDD'd Go fix to `buildGossipLine` so seeded facts gossip in zones with no recent world events. Quest-spoiler firewall: no quest-secret lore in any authored gossip/conversation/fact.

**Tech Stack:** YAML data under `_datafiles/world/dogmud/`; Go (`internal/hooks/MobIdle_HandleIdleMobs.go` gossip fix). Validators in `internal/mobs`, `internal/relationships`, `internal/facts`, `internal/conversations`, the schedule loader.

---

## Verified facts (do not re-derive)

- **Mob YAML fields:** `relationships:` = `[]{to:int, type:string, subtype:string?}`; `knows_facts:` = `[]string`; `schedule_id:` = string; gossiper = a `- gossiper` entry under `groups:`. Unknown relationship ids/types and unknown fact ids **warn, not panic**.
- **Relationship types:** family/friend/rival/lover (symmetric auto-mirror), employer↔employee (asymmetric auto-mirror). Author ONE side per edge.
- **Conversation type-pools** (friend, rival, employer, employee, family) all already exist. Pair overrides: `conversations/pairs/{lower}_{higher}.yaml` with `id`, `mob_a`, `mob_b`, `exchanges:` (`lines:` → `speaker: A|B`, `text:`). Speaker A/B is engine-randomized → every line MUST be swap-safe. Conversations only fire between two co-located idle NPCs.
- **Schedules:** file `_datafiles/world/dogmud/schedules/thornwall_city/<id>.yaml`, filename = id. Segments cover all 24h (no gaps/overlaps), `target_room` must exist, `mapper.GetPath` must succeed between consecutive segments — **these are PANICS**. `activity ∈ "" | craft | sleeping | patrol`. Existing thornwall schedules dir already present.
- **Anchor spawn rooms (confirmed):** Velk 94→473 (barracks), Voss 98→471, market merchant 102→465, food vendor 103→464, Tess 108→482, Maren 113→480.
- **Existing schedules (5):** `thornwall_tavern_keeper` (Marek 96), `thornwall_barmaid` (Dal 117), `thornwall_smith` (Kerra 97), `thornwall_temple_priest` (Olen 95), `thornwall_city_guard_dayshift` (city guard 106).
- **facts.yaml:** has `test-mayor` (id `test-mayor`, desc "The Thornwall mayor has resigned in disgrace.", significance 1, declared_round 2077194, tags [politics], status active). Repurpose it.
- **Gossip:** `buildGossipLine` (MobIdle_HandleIdleMobs.go:398) pulls recent world events; `renderFactGossip` (:509) fallback chain `fact-{id} → fact-{tag} → fact-default` — `fact-default` EXISTS and is non-empty, so any fact renders. **The gap:** when `len(evts)==0` (line ~416), it returns the generic `fallback` template and NEVER reaches the fact pool. `facts.KnownFactsOf(mobId) []facts.KnownFact` returns a mob's known facts; `renderFactGossip(kf facts.KnownFact) string`.
- **Authoring rule:** a fact only ever gossips if at least one **gossiper** NPC has it in `knows_facts`.

## Pre-flight

Already on branch `feature/mob-aliveness-6.3-thornwall-deepening` (spec committed). All work lands here. Mob files are under `_datafiles/world/dogmud/mobs/thornwall_city/`.

---

## Task 1: Relationship graph (Section A)

**Files (modify — add a `relationships:` block after each mob's `groups:` block):**
94, 95, 96, 98, 100, 108, 343-style? No — Thornwall ids:
- `94-guard_captain_velk.yaml`, `95-temple_priest_olen.yaml`, `96-tavern_keeper_marek.yaml`, `98-apothecary_voss.yaml`, `100-city_beggar.yaml`, `108-jeweler_tess.yaml`, `357-ketil.yaml`

Author ONE side per edge (reverse auto-mirrors). Read each file first; place `relationships:` at the same indent as `groups:`.

- [ ] **Step 1: Marek (96)** — add:
```yaml
relationships:
  - to: 117
    type: employer
    subtype: barmaid
  - to: 248
    type: employer
    subtype: cook
```

- [ ] **Step 2: Velk (94)** — add (the 94→102 "the-beat" edge backs the Velk/market-merchant conversation pair in Task 4):
```yaml
relationships:
  - to: 106
    type: employer
    subtype: captain
  - to: 102
    type: friend
    subtype: the-beat
```

- [ ] **Step 3: Ketil (357)** — add:
```yaml
relationships:
  - to: 359
    type: family
    subtype: son
  - to: 358
    type: employer
    subtype: guard
```

- [ ] **Step 4: Jeweler Tess (108)** — add (artisan colleagues):
```yaml
relationships:
  - to: 109
    type: friend
    subtype: colleague
```

- [ ] **Step 5: Apothecary Voss (98)** — add:
```yaml
relationships:
  - to: 109
    type: friend
    subtype: colleague
```

- [ ] **Step 6: Temple Priest Olen (95)** — add (the tithe audit):
```yaml
relationships:
  - to: 99
    type: friend
    subtype: the-audit
```

- [ ] **Step 7: City Beggar (100)** — add (street folk):
```yaml
relationships:
  - to: 101
    type: friend
    subtype: street
```

- [ ] **Step 8: Build + boot-validate relationships load**

Run: `go build ./...` (exit 0). Then boot and grep for relationship warnings:
`timeout 45 go run . > /tmp/t63.log 2>&1; grep -iE "relationship|panic" /tmp/t63.log | head; grep -i "Server Ready" /tmp/t63.log`
Expected: `Server Ready`, no panic, and NO "unknown id" relationship warnings naming 94/95/96/98/99/100/101/106/108/109/117/248/357/358/359 (a warning = a typo).

- [ ] **Step 9: Commit**
```bash
git add _datafiles/world/dogmud/mobs/thornwall_city/94-guard_captain_velk.yaml _datafiles/world/dogmud/mobs/thornwall_city/95-temple_priest_olen.yaml _datafiles/world/dogmud/mobs/thornwall_city/96-tavern_keeper_marek.yaml _datafiles/world/dogmud/mobs/thornwall_city/98-apothecary_voss.yaml _datafiles/world/dogmud/mobs/thornwall_city/100-city_beggar.yaml _datafiles/world/dogmud/mobs/thornwall_city/108-jeweler_tess.yaml _datafiles/world/dogmud/mobs/thornwall_city/357-ketil.yaml
git commit -m "content(thornwall): relationship graph (6.3 Section A)"
```

---

## Task 2: Facts + knowledge + gossiper expansion (Section C)

**Files:**
- Modify: `_datafiles/world/dogmud/facts.yaml` (rename test-mayor + append 4)
- Modify (add `knows_facts:`): mobs 94, 95(no—Olen not in fact list)… see Step 2 for exact ids.
- Modify (add `- gossiper` to `groups:`): 100, 101

- [ ] **Step 1: Repurpose `test-mayor` + append 4 facts in `facts.yaml`**

Change the existing `test-mayor` entry's `id` to `thornwall-mayor-disgraced` and add a `thornwall` tag (keep desc/significance/declared_round/status). Result:
```yaml
    - id: thornwall-mayor-disgraced
      description: The Thornwall mayor has resigned in disgrace.
      significance: 1
      declared_round: 2077194
      tags:
        - politics
        - thornwall
      status: active
```
Then append these 4 (siblings under `facts:`, `declared_round: 0` reads as long-standing):
```yaml
    - id: thornwall-road-bandits
      description: The road between Thornwall and Stillwater draws bandits; the caravans run guarded.
      significance: 1
      declared_round: 0
      tags:
        - thornwall
        - road
      status: active
    - id: thornwall-hard-times
      description: Taxes bite and folk grumble about paying for protection; trade is tight in the city.
      significance: 1
      declared_round: 0
      tags:
        - thornwall
        - hardship
      status: active
    - id: thornwall-steel-heritage
      description: Thornwall steel is an old guild craft, the technique passed hand to hand and never written down.
      significance: 1
      declared_round: 0
      tags:
        - thornwall
        - craft
      status: active
    - id: thornwall-caravan-trade
      description: A caravan runs the Thornwall-Stillwater road regularly, hauling lake-iron, pearls, and trade goods.
      significance: 1
      declared_round: 0
      tags:
        - thornwall
        - trade
      status: active
```

- [ ] **Step 2: Add `knows_facts:` to mobs** (place after `groups:`/`relationships:`; read each first). **At least one gossiper must know each fact** — the gossiper set is Fen 114, Gobb 115, Wrex 116, Beggar 100, Performer 101 (the last two become gossipers in Step 3). Author:

Pell (99):
```yaml
knows_facts:
  - thornwall-mayor-disgraced
```
Market Merchant (102):
```yaml
knows_facts:
  - thornwall-mayor-disgraced
  - thornwall-caravan-trade
  - thornwall-hard-times
```
Marek (96):
```yaml
knows_facts:
  - thornwall-mayor-disgraced
  - thornwall-hard-times
```
Ketil (357):
```yaml
knows_facts:
  - thornwall-road-bandits
  - thornwall-caravan-trade
```
Velk (94):
```yaml
knows_facts:
  - thornwall-road-bandits
  - thornwall-mayor-disgraced
```
Kerra (97):
```yaml
knows_facts:
  - thornwall-steel-heritage
```
Maren (113):
```yaml
knows_facts:
  - thornwall-steel-heritage
```
Old Gobb (115) — gossiper, give him the city-news spread:
```yaml
knows_facts:
  - thornwall-mayor-disgraced
  - thornwall-hard-times
  - thornwall-caravan-trade
```
Old Fen (114) — gossiper:
```yaml
knows_facts:
  - thornwall-mayor-disgraced
  - thornwall-steel-heritage
```
Beggar (100) — gossiper (hears everything):
```yaml
knows_facts:
  - thornwall-hard-times
  - thornwall-road-bandits
```
Performer (101) — gossiper:
```yaml
knows_facts:
  - thornwall-mayor-disgraced
  - thornwall-caravan-trade
```
(This guarantees every fact id is in ≥1 gossiper's list: mayor→Fen/Gobb/Performer; road→Beggar; hard-times→Gobb/Beggar; steel→Fen; caravan→Gobb/Performer.)

- [ ] **Step 3: Add `- gossiper` to groups** on `100-city_beggar.yaml` and `101-street_performer.yaml` (insert as a list item under `groups:`, after `- humanoid`; preserve `- thornwall_citizens`).

- [ ] **Step 4: Boot-validate**

Boot, grep `facts|unknown fact|panic`. Expected: no panic; no "unknown fact id" warnings (an id typo between Step 1 and Step 2 would warn). Confirm `facts.LoadFromMobs` runs.

- [ ] **Step 5: Commit**
```bash
git add _datafiles/world/dogmud/facts.yaml _datafiles/world/dogmud/mobs/thornwall_city/
git commit -m "content(thornwall): facts + knows_facts + gossiper expansion (6.3 Section C)"
```

---

## Task 3: Schedules (Section B), +6 anchors

**Files:**
- Create 6 schedules under `_datafiles/world/dogmud/schedules/thornwall_city/`: `thornwall_market_merchant.yaml`, `thornwall_food_vendor.yaml`, `thornwall_apothecary.yaml`, `thornwall_jeweler.yaml`, `thornwall_weaver.yaml`, `thornwall_guard_captain.yaml`.
- Modify 6 mob YAMLs to add `schedule_id:`.

Shop/stall anchors sleep IN PLACE (same room all night — trivially path-connected, no new rooms). Each segment has 3-4 distinct idlecommands (the 6.1 "widen idle pools" lesson — §E.2). Author verbatim:

- [ ] **Step 1: `thornwall_market_merchant.yaml`**
```yaml
id: thornwall_market_merchant
description: "Market Merchant: opens the stall, trades through the day, closes at dusk, sleeps by the stall."
segments:
  - start: 6
    end: 10
    target_room: 465
    activity: ""
    idlecommands:
      - emote lifts the shutters and sets out the day's wares.
      - emote chalks fresh prices onto the board.
      - say Fair prices, fair weight. Come and see.
      - emote weighs a coin against the light, checking it.
  - start: 10
    end: 18
    target_room: 465
    activity: ""
    idlecommands:
      - emote haggles cheerfully over a bolt of cloth.
      - say Everything's dearer this season. Don't blame the seller.
      - emote counts the till with a practiced thumb.
      - emote waves to a passing caravan hand.
  - start: 18
    end: 22
    target_room: 465
    activity: ""
    idlecommands:
      - emote packs the unsold goods into a locked chest.
      - say Come back tomorrow. Same stall, same fair price.
      - emote stretches a back stiff from standing.
  - start: 22
    end: 6
    target_room: 465
    activity: sleeping
    idlecommands:
      - emote dozes on a cot behind the stall, one eye on the strongbox.
```

- [ ] **Step 2: `thornwall_food_vendor.yaml`**
```yaml
id: thornwall_food_vendor
description: "Food Vendor: fires the braziers at dawn, sells hot food by day, banks the coals at night."
segments:
  - start: 6
    end: 10
    target_room: 464
    activity: craft
    idlecommands:
      - emote stokes the brazier until the coals glow.
      - emote ladles broth into a tasting cup and nods.
      - say Hot and fresh! Best on Main Street.
  - start: 10
    end: 18
    target_room: 464
    activity: craft
    idlecommands:
      - emote turns skewers over the coals with quick fingers.
      - say Mind the steam, friend. Worth the burn.
      - emote calls a regular by name and starts their usual.
      - emote sources a basket of clams from a caravan crate.
  - start: 18
    end: 22
    target_room: 464
    activity: ""
    idlecommands:
      - emote scrapes the griddle clean for the night.
      - say Sold out of the good stuff. Come earlier tomorrow.
  - start: 22
    end: 6
    target_room: 464
    activity: sleeping
    idlecommands:
      - emote sleeps near the banked, still-warm brazier.
```

- [ ] **Step 3: `thornwall_apothecary.yaml`**
```yaml
id: thornwall_apothecary
description: "Apothecary Voss: clinical day work at the lane, closes at dusk, sleeps among the jars."
segments:
  - start: 6
    end: 9
    target_room: 471
    activity: craft
    idlecommands:
      - emote measures a powder against a brass scale, exact to the grain.
      - emote labels a vial in a precise, cramped hand.
      - say Dosage is everything. A cure and a poison differ only by amount.
  - start: 9
    end: 17
    target_room: 471
    activity: craft
    idlecommands:
      - emote grinds a dark resin to fine dust without expression.
      - say State your symptoms plainly. I have no time for drama.
      - emote checks a chrysalis catalyst against the lamplight.
  - start: 17
    end: 21
    target_room: 471
    activity: ""
    idlecommands:
      - emote wipes the bench down with astringent spirits.
      - say Closed. Come back when you are actually ill.
  - start: 21
    end: 6
    target_room: 471
    activity: sleeping
    idlecommands:
      - emote sleeps stiffly amid the sharp smell of herbs.
```

- [ ] **Step 4: `thornwall_jeweler.yaml`**
```yaml
id: thornwall_jeweler
description: "Jeweler Tess: precise bench work by day, closes the workshop, sleeps at the bench."
segments:
  - start: 7
    end: 10
    target_room: 482
    activity: craft
    idlecommands:
      - emote sets a tiny stone into a setting with steady tweezers.
      - emote squints through a loupe at a flawed gem.
      - say Patience. The stone tells you where it wants to sit.
  - start: 10
    end: 18
    target_room: 482
    activity: craft
    idlecommands:
      - emote taps a setting closed with a jeweler's hammer.
      - emote blows dust from a finished chrysalis-set ring.
      - say Fine work can't be hurried, and I don't try.
  - start: 18
    end: 22
    target_room: 482
    activity: ""
    idlecommands:
      - emote locks the day's pieces into a velvet-lined case.
      - emote rubs cramped, precise fingers.
  - start: 22
    end: 7
    target_room: 482
    activity: sleeping
    idlecommands:
      - emote sleeps in a chair pulled close to the locked case.
```

- [ ] **Step 5: `thornwall_weaver.yaml`**
```yaml
id: thornwall_weaver
description: "Weaver Maren: loom work by day, closes the cottage, sleeps by the loom."
segments:
  - start: 6
    end: 10
    target_room: 480
    activity: craft
    idlecommands:
      - emote throws the shuttle in a steady, practiced rhythm.
      - emote checks the tension of the warp with dye-stained fingers.
      - say A good cloth is honest work, thread by thread.
  - start: 10
    end: 18
    target_room: 480
    activity: craft
    idlecommands:
      - emote treadles the loom without looking at her hands.
      - emote holds a finished bolt to the light, judging the weave.
      - say My niece down in Stillwater works the same trade, simpler patterns.
  - start: 18
    end: 22
    target_room: 480
    activity: ""
    idlecommands:
      - emote winds the day's thread back onto its spools.
      - emote sweeps lint and clipped threads from the floorboards.
  - start: 22
    end: 6
    target_room: 480
    activity: sleeping
    idlecommands:
      - emote sleeps to the faint creak of the cooling loom.
```

- [ ] **Step 6: `thornwall_guard_captain.yaml`** (the market-beat inspection co-locates Velk with the market merchant at 465)
```yaml
id: thornwall_guard_captain
description: "Guard Captain Velk: morning muster at the barracks, a midday market beat, command by day, sleeps at the barracks."
segments:
  - start: 6
    end: 10
    target_room: 473
    activity: ""
    idlecommands:
      - emote reviews the night's watch reports at the duty desk.
      - emote inspects a recruit's kit with a hard eye.
      - say A city this size, the watch is always stretched too thin.
  - start: 10
    end: 13
    target_room: 465
    activity: ""
    idlecommands:
      - emote walks the market with hands clasped behind his back.
      - emote nods to a stallholder and notes who is missing.
      - say Quiet's good. Quiet means folk are behaving.
  - start: 13
    end: 20
    target_room: 473
    activity: ""
    idlecommands:
      - emote signs a stack of patrol orders.
      - emote frowns over a map of the lower passages.
      - say Write it in the log or it never happened.
  - start: 20
    end: 6
    target_room: 473
    activity: sleeping
    idlecommands:
      - emote sleeps in the small room off the barracks armoury.
```

- [ ] **Step 7: Add `schedule_id:` to the 6 mob YAMLs** (near `groups:`; don't disturb relationships/knows_facts blocks):
  - 102: `schedule_id: thornwall_market_merchant`
  - 103: `schedule_id: thornwall_food_vendor`
  - 98: `schedule_id: thornwall_apothecary`
  - 108: `schedule_id: thornwall_jeweler`
  - 113: `schedule_id: thornwall_weaver`
  - 94: `schedule_id: thornwall_guard_captain`

- [ ] **Step 8: Build**

Run `go build ./...` (exit 0).

- [ ] **Step 9: Wipe instance saves (stale saves shadow schedule_id)**
```powershell
Remove-Item -Recurse -Force _datafiles/world/dogmud/mobs.instances/*, _datafiles/world/dogmud/rooms.instances/* -ErrorAction SilentlyContinue
```

- [ ] **Step 10: Boot-validate schedules (PANIC-prone)**

Boot; watch the full startup. Expected: **no panic** — none of `did not cover hour`, `target_room ... does not exist`, `GetPath` failure (notably Velk's 473↔465 route), `unresolved schedule_id`. Confirm `mobs.LoadSchedules() loadedCount=` increased by 6 (was 5 → expect 11). If a path panic names 473/465, recheck Velk's segments are town-reachable.

- [ ] **Step 11: Commit**
```bash
git add _datafiles/world/dogmud/schedules/thornwall_city/ _datafiles/world/dogmud/mobs/thornwall_city/
git commit -m "content(thornwall): 6 NPC daily schedules incl. market square life (6.3 Section B)"
```

---

## Task 4: Conversation pair overrides (Section D)

**Files (create — all swap-safe, co-location verified):**
- `_datafiles/world/dogmud/conversations/pairs/96_117.yaml` (Marek/Dal — both in tavern 472)
- `_datafiles/world/dogmud/conversations/pairs/114_115.yaml` (Fen/Gobb — both at Back Corner 484)
- `_datafiles/world/dogmud/conversations/pairs/94_102.yaml` (Velk/market merchant — co-locate at 465 during Velk's 10-13 beat + merchant's 10-18 trading)

(Marek/Brynn and Tess/Vael relationships stay substrate-only — they don't co-locate via schedules, so no pair file, per the 6.1 dead-pair lesson. Keep the existing `116_117`.)

- [ ] **Step 1: `96_117.yaml`** (Marek/Dal, tavern)
```yaml
id: marek_and_dal
mob_a: 96
mob_b: 117
exchanges:
  - lines:
      - speaker: A
        text: "Table four's been nursing one ale for an hour."
      - speaker: B
        text: "Then table four can keep nursing it. We're not a charity."
      - speaker: A
        text: "Ha. You'll run this place better than me one day."
  - lines:
      - speaker: A
        text: "Tax man came round again this morning."
      - speaker: B
        text: "Smile, pay, and water the cheap stuff. Same as always."
      - speaker: A
        text: "Aye. Same as always."
  - lines:
      - speaker: A
        text: "Brynn says we're low on the good flour."
      - speaker: B
        text: "Caravan's due. I'll flag Lars when he runs the orders."
  - lines:
      - speaker: A
        text: "Long night ahead?"
      - speaker: B
        text: "They always are. Pour me a small one for the feet."
```

- [ ] **Step 2: `114_115.yaml`** (Fen/Gobb, Back Corner)
```yaml
id: fen_and_gobb
mob_a: 114
mob_b: 115
exchanges:
  - lines:
      - speaker: A
        text: "You hear the mayor finally stepped down?"
      - speaker: B
        text: "Heard it twice from you already, old man."
      - speaker: A
        text: "Worth hearing thrice. Disgrace, they say."
  - lines:
      - speaker: A
        text: "My cousin's wife's brother works the docks now."
      - speaker: B
        text: "You've got a cousin's wife's brother for everything."
      - speaker: A
        text: "That's how I know everything."
  - lines:
      - speaker: A
        text: "Prices up again at the market."
      - speaker: B
        text: "Everything's up but our luck."
      - speaker: A
        text: "Drink's still the same. Small mercies."
  - lines:
      - speaker: A
        text: "Quiet in here tonight."
      - speaker: B
        text: "Quiet's fine. Loud means the watch is about."
```

- [ ] **Step 3: `94_102.yaml`** (Velk/market merchant, the beat). Swap-safe: neither line assumes who is guard vs vendor beyond what reads naturally either way; keep it neutral.
```yaml
id: velk_and_merchant
mob_a: 94
mob_b: 102
exchanges:
  - lines:
      - speaker: A
        text: "Quiet on the square today."
      - speaker: B
        text: "Quiet's good for nobody's purse but it's good for the nerves."
      - speaker: A
        text: "I'll take the nerves."
  - lines:
      - speaker: A
        text: "Any trouble I should know about?"
      - speaker: B
        text: "Pickpocket worked the east stalls last week. Gone now."
      - speaker: A
        text: "Tell me next time it's the same week."
  - lines:
      - speaker: A
        text: "Caravan's running guarded these days."
      - speaker: B
        text: "Road's not what it was. Bandits don't fear much."
      - speaker: A
        text: "They'll fear the gibbet if I catch them."
```

- [ ] **Step 4: Boot-validate**

Boot; grep `conversation|pair|panic`. Expected: no panic; the 3 new pairs load (alongside existing `116_117`). A warning that a pair references an unknown relationship means the Task 1 edge for that pair is missing — confirm relationship edges exist for 96/117, 114/115, 94/102 (94→106 is the Velk edge, but the 94_102 pair needs a 94↔102 RELATIONSHIP — see note below).

> **IMPORTANT co-location note:** conversations only fire between NPCs that have a relationship edge AND co-locate. The `94_102` pair's backing edge (Velk 94 → market merchant 102, `friend`/`the-beat`) was authored in Task 1 Step 2 — confirm it's present. The pair fires only when both are at room 465 (Velk's 10:00-13:00 beat overlapping the merchant's 10:00-18:00 trading).

- [ ] **Step 5: Commit**
```bash
git add _datafiles/world/dogmud/conversations/pairs/ _datafiles/world/dogmud/mobs/thornwall_city/94-guard_captain_velk.yaml
git commit -m "content(thornwall): conversation pairs incl. guard-captain market beat (6.3 Section D)"
```

---

## Task 5: Gossip fix — facts gossip in zones with no recent world events (Section E.1)

**Files:**
- Modify: `internal/hooks/MobIdle_HandleIdleMobs.go` (`buildGossipLine`)
- Test: `internal/hooks/MobIdle_HandleIdleMobs_test.go` (or the existing hooks test file that covers gossip — check `internal/hooks/hooks_test.go` / any `*gossip*_test.go`; put the test where the existing gossip tests live)

**Problem:** `buildGossipLine` returns the generic `fallback` template when there are no recent world events (`len(evts)==0`), BEFORE ever consulting the mob's known facts. So in a quiet zone (like Thornwall with few world events), seeded facts never gossip. Fix: in the no-events branch, try the mob's known facts first; only use the generic fallback if the mob has no known facts (or fact rendering returns empty).

- [ ] **Step 1: Write the failing test**

First read the existing gossip tests to learn how they construct a mob with a zone + seed known facts + stub `gossipTemplates`/`worldevents`. Then add a test asserting: a mob with a known fact and NO recent world events produces a fact-based gossip line (containing the fact's description), NOT the generic fallback. Sketch (adapt to the real test harness — the helper that seeds facts and the way `buildGossipLine` is invoked):
```go
func TestBuildGossipLine_NoEvents_UsesKnownFacts(t *testing.T) {
	// Arrange: a mob in a zone with NO recent world events, but with a known fact.
	// (Use the existing test helpers to: build the mob, set its Zone, seed a fact
	// into facts.KnownFactsOf via the facts test API, ensure GetRecentWorldEvents
	// returns empty for this zone, and load gossipTemplates including fact-default.)
	// Act:
	line := buildGossipLine(mob)
	// Assert: the line is the fact-based gossip, not the generic fallback.
	if !strings.Contains(line, "<the seeded fact description>") {
		t.Fatalf("expected fact-based gossip with no events, got %q", line)
	}
}
```
If seeding `facts.KnownFactsOf` in a test is hard, mirror exactly how the existing 1.7/gossip tests seed facts (e.g. `facts.RecordKnowsFact(mobId, factId, ...)` + declaring the fact via `facts.Declare`). Report the harness you used.

- [ ] **Step 2: Run; confirm RED**

Run the new test → FAIL (current code returns the generic fallback, so the line won't contain the fact description).

- [ ] **Step 3: Implement the fix**

In `buildGossipLine`, change the no-events branch (currently around lines 416-422):
```go
	if len(evts) == 0 {
		// No recent world events — try the mob's known facts before the generic
		// fallback, so seeded facts still gossip in quiet zones (6.3 §E.1).
		if fc := facts.KnownFactsOf(int(mob.MobId)); len(fc) > 0 {
			if line := renderFactGossip(fc[util.Rand(len(fc))]); line != "" {
				return line
			}
		}
		if fallbacks, ok := gossipTemplates["fallback"]; ok && len(fallbacks) > 0 {
			return fallbacks[util.Rand(len(fallbacks))]
		}
		return ""
	}
```

- [ ] **Step 4: Run; confirm GREEN**

Run the new test (PASS), then `go test ./internal/hooks/` (all pass), then `go build ./...`.

- [ ] **Step 5: Commit**
```bash
git add internal/hooks/MobIdle_HandleIdleMobs.go internal/hooks/MobIdle_HandleIdleMobs_test.go
git commit -m "fix(gossip): seeded facts gossip in zones with no recent world events (6.3 E.1)"
```

---

## Task 6: Final integration + context.md + roadmap + before/after notes

**Files:**
- Modify: `internal/hooks/context.md` (note the gossip no-events fact path) — only if that context.md documents gossip; otherwise skip.
- Modify: `MOB_ALIVENESS_ROADMAP.md` (mark 6.3 Done + roll-up)
- Create: `docs/superpowers/notes/2026-06-03-aliveness-before-after-stillwater-vs-thornwall.md` (the 6.3 deliverable)

- [ ] **Step 1: Clean instance saves + full boot**
```powershell
Remove-Item -Recurse -Force _datafiles/world/dogmud/mobs.instances/*, _datafiles/world/dogmud/rooms.instances/* -ErrorAction SilentlyContinue
```
Boot the full server. Expected: clean boot, no panics, `mobs.LoadSchedules() loadedCount=11`, normal `loadedCount` lines, `Server Ready`.

- [ ] **Step 2: Full test suite**

Run `go test ./...` → all pass.

- [ ] **Step 3: Document the gossip change** in `internal/hooks/context.md` IF it covers `buildGossipLine` (one line: no-events branch now tries known facts before the generic fallback). If the file doesn't mention gossip, skip this step.

- [ ] **Step 4: Write the before/after authoring notes** to `docs/superpowers/notes/2026-06-03-aliveness-before-after-stillwater-vs-thornwall.md`: what generalized cleanly from Stillwater (the layered-by-fit recipe; reusing type-pools; the swap-safe + co-location rules for pairs; sleep-in-place schedules avoiding new rooms), and what was harder in Thornwall (the quest-spoiler firewall; caravan mobs that travel so they're substrate-only; co-location harder in a bigger room graph so fewer pairs; the gossip-needs-a-gossiper-to-know-the-fact rule; the no-events gossip fix). 1 page, to script the 6.5 rollout.

- [ ] **Step 5: Update `MOB_ALIVENESS_ROADMAP.md`** — set the 6.3 tracker row Status to `Done`, set the 6.3 mini-brief `**Status:** Done (2026-06-03)` with a `**Shipped:**` paragraph summarizing the pass (relationships, +6 schedules incl. market-square life, 5 public facts + role-gated knowledge + 2 gossipers, 3 conversation pairs, the gossip no-events fact fix, before/after notes). Bump the roll-up to `38 / 42 done`.

- [ ] **Step 6: Commit**
```bash
git add MOB_ALIVENESS_ROADMAP.md docs/superpowers/notes/2026-06-03-aliveness-before-after-stillwater-vs-thornwall.md internal/hooks/context.md
git commit -m "docs(aliveness): mark 6.3 Thornwall deepening done + before/after notes"
```

---

## Manual smoke (deferred to user)

Wipe instance saves, boot, walk Thornwall across a day/night cycle and confirm:
- the 6 new anchors move to their segment rooms; the market square (464/465) lives by day and the vendors sleep at night;
- gossipers (Fen, Gobb, Wrex, Beggar, Performer) surface the 5 seeded facts — including the non-crisis ones (mayor, steel, caravan) now that the no-events gossip path is fixed;
- the conversation pairs fire: Marek/Dal in the tavern, Fen/Gobb at the Back Corner, and Velk/market-merchant on the square during Velk's ~10:00-13:00 beat;
- NO quest-secret lore leaks in any gossip/conversation;
- watch conversation cadence in the denser tavern/square rooms — if it's too chatty or too sparse, flag it (§E.3 knob tuning, `ConversationBaseChancePct`/cooldown, is a small post-smoke follow-up).

## Self-review notes

- **Spec coverage:** A→Task 1; B→Task 3; C→Task 2; D→Task 4; E.1 (gossip fact gap)→Task 5; E.2 (widen idle pools)→done inline in Task 3's ≥3-4 idlecommands/segment; E.3 (cadence)→deferred post-smoke (noted in manual smoke + Task 6 notes); F (validation + before/after notes)→Tasks' boot steps + Task 6.
- **No placeholders:** all relationship blocks, schedule files, facts, and conversation exchanges are inline; the gossip test is a sketch tied to "use the existing gossip-test harness" (a real lookup, since I can't see that test file's exact fact-seeding API from here — the implementer confirms it).
- **Type consistency:** field names (`relationships` to/type/subtype, `knows_facts`, `schedule_id`, schedule `target_room`/`activity`/`idlecommands`, conversation `mob_a`/`mob_b`/`exchanges`/`speaker`/`text`, fact id/description/significance/declared_round/tags/status) match the verified schemas and the 6.1 precedent.
- **Quest-spoiler firewall:** facts and conversations reference only public troubles (mayor, bandits, taxes/protection-grumble, steel, caravan); no Elara/tunnels/ledger/Torvan. Verified per authored line.
