# Mob Aliveness 6.5d — Roads Batch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Light, data-only roads pass — 2 road-danger gossip facts voiced by the travelling folk + 2 relationships (Marches waystation pair, caravan master→crew), and confirm the 3.7 caravan still resolves. Closes the 6.5 content pass.

**Architecture:** Pure YAML. Two mobs gain `relationships:`; `facts.yaml` gains 2 facts; nine road-folk gain `knows_facts:` + (where missing) a `gossiper` group tag. No schedules/conversations/behavior/patrol changes. Boot-time loaders validate refs and **panic on bad ids** — the boot test is the verification gate (no Go unit tests; matches the codebase).

**Tech Stack:** YAML data files; relationships + facts loaders.

**Spec:** `docs/superpowers/specs/2026-06-05-mob-aliveness-6.5d-roads-batch-design.md`

---

## Reference: schemas (verified this session)

Relationships (author each edge ONCE; engine auto-mirrors symmetric same-type and employer↔employee):
```yaml
relationships:
  - to: <otherMobId>
    type: friend   # friend|rival|family|lover|employer|employee
    subtype: <flavor>
```
Facts (append under `facts:` in `_datafiles/world/dogmud/facts.yaml`, 4-space item indent):
```yaml
    - id: <slug>
      description: <one sentence>
      significance: 1
      declared_round: 0
      tags:
        - <tag>
      status: active
```
Mob: `knows_facts: [<id>, ...]` (top-level); `gossiper` in `groups:` makes it gossip. NOTE: `251-innkeeper_thessa` already has `gossiper` — only add `knows_facts` there; the others need `gossiper` appended.

## Reference: mob ids
innkeeper_thessa 251, peddler_malk 250, traveler_bren 252 (`marches_spur_road/`); caravan_master 281, lone_traveler 329, woodcutter_hagen 328, corvin 276, farmer 282 (`north_road/`); road_warden_tessara 83 (`dustwalk_road/`). Caravan crew: ketil 357, marta 358, lars 359 (`thornwall_city/`).

---

## Task 1: Relationships

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/marches_spur_road/251-innkeeper_thessa.yaml`
- Modify: `_datafiles/world/dogmud/mobs/north_road/281-caravan_master.yaml`

- [ ] **Step 1: `251-innkeeper_thessa.yaml`** — add after `groups:`:
```yaml
relationships:
  - to: 250
    type: friend
    subtype: waystation
```

- [ ] **Step 2: `281-caravan_master.yaml`** — add after `groups:`:
```yaml
relationships:
  - to: 357
    type: employer
    subtype: caravan
  - to: 358
    type: employer
    subtype: caravan
  - to: 359
    type: employer
    subtype: caravan
```

- [ ] **Step 3: Build + commit**
```bash
go build ./...
git add _datafiles/world/dogmud/mobs/marches_spur_road/251-innkeeper_thessa.yaml _datafiles/world/dogmud/mobs/north_road/281-caravan_master.yaml
git commit -m "feat(6.5d): road relationships (Marches waystation pair, caravan master->crew)"
```

---

## Task 2: Facts + gossipers

**Files:**
- Modify: `_datafiles/world/dogmud/facts.yaml`
- Modify: the 9 gossiper mobs (83, 329, 328, 252, 276, 282 for peril; 281, 250, 251 for caravans-guarded)

- [ ] **Step 1: Append 2 facts to `facts.yaml`** (under `facts:`, 4-space item indent):
```yaml
    - id: roads-bandit-peril
      description: The trade roads have grown dangerous; bandits ambush the lonely stretches, so folk travel in numbers.
      significance: 2
      declared_round: 0
      tags:
        - roads
        - bandits
        - crisis
      status: active
    - id: roads-caravans-guarded
      description: The caravans cross under hired guard now, running the gauntlet between the towns.
      significance: 1
      declared_round: 0
      tags:
        - roads
        - trade
      status: active
```

- [ ] **Step 2: `roads-bandit-peril` gossipers** — for each of these 6 mobs, append `- gossiper` to `groups:` (preserve existing entries) and add `knows_facts` with `roads-bandit-peril`:
  - `dustwalk_road/83-road_warden_tessara.yaml`
  - `north_road/329-lone_traveler.yaml`
  - `north_road/328-woodcutter_hagen.yaml`
  - `marches_spur_road/252-traveler_bren.yaml`
  - `north_road/276-corvin.yaml`
  - `north_road/282-farmer.yaml`
  ```yaml
  knows_facts:
    - roads-bandit-peril
  ```

- [ ] **Step 3: `roads-caravans-guarded` gossipers**
  - `north_road/281-caravan_master.yaml` — append `- gossiper` to `groups:`; add `knows_facts: [roads-caravans-guarded]`. (It already has a `relationships:` block from Task 1 — leave it.)
  - `marches_spur_road/250-peddler_malk.yaml` — append `- gossiper`; add `knows_facts: [roads-caravans-guarded]`.
  - `marches_spur_road/251-innkeeper_thessa.yaml` — **already has `gossiper`** (do NOT duplicate); just add `knows_facts: [roads-caravans-guarded]`. (It also has the Task 1 `relationships:` block — leave it.)
  Use block form for each:
  ```yaml
  knows_facts:
    - roads-caravans-guarded
  ```

- [ ] **Step 4: Build + commit**
```bash
go build ./...
git add _datafiles/world/dogmud/facts.yaml _datafiles/world/dogmud/mobs/dustwalk_road/ _datafiles/world/dogmud/mobs/marches_spur_road/ _datafiles/world/dogmud/mobs/north_road/
git commit -m "feat(6.5d): 2 road-danger facts + knows_facts + gossiper tags"
```

---

## Task 3: Boot-validate + smoke (incl. caravan resolves)

- [ ] **Step 1: YAML + build**
Run: `go build ./...` and a YAML parse check:
`python -c "import yaml,glob; [yaml.safe_load(open(f,encoding='utf-8')) for f in glob.glob('_datafiles/world/dogmud/mobs/north_road/*.yaml')+glob.glob('_datafiles/world/dogmud/mobs/marches_spur_road/*.yaml')+glob.glob('_datafiles/world/dogmud/mobs/dustwalk_road/*.yaml')+['_datafiles/world/dogmud/facts.yaml']]; print('parse ok')"`
Expected: build clean; parse ok.

- [ ] **Step 2: Wipe instances + boot**
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Boot the server. **Confirm no panic** from the relationships loader (unknown `to:` — esp. caravan crew 357/358/359) or facts loader (`knows_facts` ref). Confirm "Server Ready". Also scan the boot log for any caravan/patrol resolution error (the 3.7 north-road caravan loads via the caravan system) — there should be none; if one appears, log it as a pre-existing 3.7 issue (out of 6.5d scope), don't fix here.

- [ ] **Step 3: In-game smoke** (admin `smoketester`; MAY be deferred to user per precedent)
- Approach a road-folk gossiper (lone_traveler / woodcutter / traveler_bren) and confirm `roads-bandit-peril` surfaces; approach the caravan_master / peddler / innkeeper for `roads-caravans-guarded`.
- `relationship between 281 357` → expect `employer`/`employee` (auto-mirrored); `relationship between 251 250` → `friend`.
- Confirm the caravan still runs the north road.

No commit (verification only).

---

## Task 4: Roadmap update (close the 6.5 content pass)

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Mark 6.5d + the 6.5 parent done**
Add a `6.5d` tracker row (Status `Done (2026-06-05)`); change the `6.5` parent row to `Done (2026-06-05)` (all four sub-batches shipped). Add a 6.5d mini-brief with a `**Shipped:**` bullet (2 road-danger facts + gossiper tags on 9 road-folk, 2 relationships [Marches waystation pair, caravan master→crew], caravan resolves confirmed). Re-tally the roll-up (6.5d done + 6.5 parent done).

- [ ] **Step 2: Commit**
```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(roadmap): mark 6.5d roads batch + 6.5 content pass Done"
```

---

## Self-review notes

**Spec coverage:**
- Spec §1 relationships → Task 1 (2 mobs; thessa→malk friend, caravan_master→crew employer).
- Spec §2 facts + gossipers → Task 2 (2 facts; 9 gossipers, thessa's pre-existing gossiper tag noted).
- Spec §3 caravan/route verification → Task 3 Step 2 (boot-log caravan-resolution check; no new patrols).
- Spec validation → Task 3; roadmap (close 6.5) → Task 4.

**Placeholder check:** both relationship blocks + both facts given in full; exact ids/paths; the gossiper list is explicit per fact. Only soft step is the deferrable in-game smoke.

**Consistency check:** relationship `to:` ids (250, 357, 358, 359) are valid mobs (357/358/359 are the 6.5a-tagged caravan crew). Both `knows_facts` ids match the facts declared in Task 2 Step 1. caravan_master 281 and innkeeper_thessa 251 are each edited in both Task 1 (relationships) and Task 2 (gossiper/knows_facts) — non-conflicting additions; thessa's gossiper tag is pre-existing so only knows_facts is added there. Road fact ids are distinct from the 6.5b/6.5c facts.

**TDD note:** content validated by the boot-time loaders (panic on bad refs), not Go unit tests — consistent with the codebase. Task 3 is the gate.
