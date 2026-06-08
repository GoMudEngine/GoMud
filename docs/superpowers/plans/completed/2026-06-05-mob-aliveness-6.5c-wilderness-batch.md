# Mob Aliveness 6.5c — Wilderness Batch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A light, data-only aliveness pass on the wilderness zones — `family` pack/kin relationships for the three social mob groups (ironwind wolf pack, ironwind goblin tribe, labyrinth warren) and four gossip facts voiced by the accessible sentient mobs.

**Architecture:** Pure YAML content. Three leader mobs gain `relationships:` blocks (engine auto-mirrors); `facts.yaml` gains 4 facts; five sentient mobs gain `knows_facts:` + a `gossiper` group tag. No schedules, conversations, or behavior changes. The relationships + facts loaders validate refs and **panic at boot** on bad ids — the boot test is the verification gate (no Go unit tests; matches the codebase).

**Tech Stack:** YAML data files; the relationships + facts loaders.

**Spec:** `docs/superpowers/specs/2026-06-05-mob-aliveness-6.5c-wilderness-batch-design.md`

---

## Reference: schemas (verified this session)

**Relationships** (on a mob; author each edge ONCE — engine auto-mirrors symmetric same-type):
```yaml
relationships:
  - to: <otherMobId>
    type: family   # family|friend|rival|lover|employer|employee
    subtype: <flavor>
```

**Facts** (append under the top-level `facts:` list in `_datafiles/world/dogmud/facts.yaml`; existing entries use 4-space item indentation):
```yaml
    - id: <slug>
      description: <one sentence>
      significance: 1
      declared_round: 0
      tags:
        - <tag>
      status: active
```

**Mob fact-awareness:** `knows_facts: [<id>, ...]` (top-level list); a mob that gossips facts to players carries `gossiper` in its `groups:` list (append to the existing list).

## Reference: mob ids

Wolf pack: alpha_wolf 215, steppe_wolf 205, young_wolf 206, scarred_wolf 223 (all `_datafiles/world/dogmud/mobs/ironwind_steppe/`).
Goblin tribe: goblin_shaman 219, goblin_scout 217, goblin_scrapper 218, goblin_sentry 222 (ironwind_steppe).
Warren: warren_chieftain 75, tunnel_shaman 74, warren_scout 72, warren_warrior 73 (`_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/`).
Gossipers: halix 372 (ironwind_steppe), hermit_kael 240 (ironwind_steppe), kessa 373 (the_fernway_south), warren_chieftain 75, tunnel_shaman 74.

---

## Task 1: Pack/kin relationships

Add a `relationships:` block (after `groups:`) to the 3 leader mobs. Author each edge once; the engine mirrors to the members. Read each file first; preserve all fields.

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/215-alpha_wolf.yaml`
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/219-goblin_shaman.yaml`
- Modify: `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/75-warren_chieftain.yaml`

- [ ] **Step 1: `215-alpha_wolf.yaml`**
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

- [ ] **Step 2: `219-goblin_shaman.yaml`**
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

- [ ] **Step 3: `75-warren_chieftain.yaml`**
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

- [ ] **Step 4: Build + commit**
```bash
go build ./...
git add _datafiles/world/dogmud/mobs/ironwind_steppe/215-alpha_wolf.yaml _datafiles/world/dogmud/mobs/ironwind_steppe/219-goblin_shaman.yaml _datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/75-warren_chieftain.yaml
git commit -m "feat(6.5c): pack/kin relationships (wolf pack, goblin tribe, warren)"
```

---

## Task 2: Facts + gossipers

**Files:**
- Modify: `_datafiles/world/dogmud/facts.yaml`
- Modify: `372-halix.yaml`, `240-hermit_kael.yaml` (ironwind_steppe), `373-kessa.yaml` (the_fernway_south), `75-warren_chieftain.yaml`, `74-tunnel_shaman.yaml` (labyrinth)

- [ ] **Step 1: Append 4 facts to `facts.yaml`** (under the `facts:` list, 4-space item indentation matching existing entries):
```yaml
    - id: ironwind-tribe-pressing
      description: The goblins of the steppe have grown bolder, pushing toward the trails and the waterholes.
      significance: 2
      declared_round: 0
      tags:
        - ironwind_steppe
        - ironwind_tribe
        - crisis
      status: active
    - id: ironwind-steppe-drying
      description: Water and game grow scarce on the steppe; every creature on it is pressed and hungry.
      significance: 2
      declared_round: 0
      tags:
        - ironwind_steppe
        - crisis
      status: active
    - id: warren-misjudged
      description: The surface folk fear the warren for no cause but their strangeness.
      significance: 1
      declared_round: 0
      tags:
        - labyrinth
        - people
      status: active
    - id: fernway-wolves-ranging
      description: The timber wolves range closer to the forest paths than they once did.
      significance: 1
      declared_round: 0
      tags:
        - the_fernway_south
        - wildlife
      status: active
```

- [ ] **Step 2: Halix + Hermit Kael (ironwind gossipers)**
For `372-halix.yaml` and `240-hermit_kael.yaml`: append `- gossiper` to the `groups:` list (preserve existing entries), and add:
```yaml
knows_facts:
  - ironwind-tribe-pressing
  - ironwind-steppe-drying
```

- [ ] **Step 3: Kessa (fernway gossiper)**
For `373-kessa.yaml`: append `- gossiper` to `groups:`, and add:
```yaml
knows_facts:
  - fernway-wolves-ranging
```

- [ ] **Step 4: Warren leaders (warren gossipers)**
For `75-warren_chieftain.yaml` and `74-tunnel_shaman.yaml`: append `- gossiper` to `groups:`, and add:
```yaml
knows_facts:
  - warren-misjudged
```
(75-warren_chieftain already gained a `relationships:` block in Task 1 — that stays; this just adds the gossiper tag + knows_facts.)

- [ ] **Step 5: Build + commit**
```bash
go build ./...
git add _datafiles/world/dogmud/facts.yaml _datafiles/world/dogmud/mobs/ironwind_steppe/ _datafiles/world/dogmud/mobs/the_fernway_south/ _datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/
git commit -m "feat(6.5c): 4 wilderness facts + knows_facts + gossiper tags"
```

---

## Task 3: Boot-validate + smoke

- [ ] **Step 1: YAML + build + test**
Run: `go build ./... && go test ./internal/relationships/ ./internal/facts/ ./internal/mobs/`
Expected: clean / pass. Optionally `python -c "import yaml,glob; [yaml.safe_load(open(f,encoding='utf-8')) for f in glob.glob('_datafiles/world/dogmud/mobs/ironwind_steppe/*.yaml')+glob.glob('_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/*.yaml')+['_datafiles/world/dogmud/facts.yaml']]; print('parse ok')"` to confirm all touched YAML parses.

- [ ] **Step 2: Wipe instances + boot**
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Boot the server. **Confirm no panic** from the relationships loader (unknown `to:` mob) or the facts loader (`knows_facts:` referencing an undeclared fact). Confirm "Server Ready". Watch the `LoadFromMobs`/relationships + `facts.Load` lines load without panic.

- [ ] **Step 3: In-game smoke** (admin `smoketester`; MAY be deferred to user per precedent — note which ran)
- Kill a wolf-pack member (e.g. steppe_wolf 205) and confirm a surviving pack member (alpha 215 / scarred 223) reacts per 4.5 kin-revenge; same spot-check for a goblin and a warren member.
- Approach Halix / Hermit Kael / Kessa and confirm the facts surface as gossip; with warm warren rep, confirm the warren leaders voice `warren-misjudged`.

No commit (verification only).

---

## Task 4: Roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Add/mark the 6.5c row**
Add a `6.5c` row to the Progress tracker (Status `Done (2026-06-05)`), and a 6.5c mini-brief under Phase 6 with a `**Shipped:**` bullet: pack/kin relationships for wolf pack / goblin tribe / warren, 4 wilderness facts + gossiper tags on Halix/Kael/Kessa/warren leaders, boot-validated. Update the 6.5 parent row's status note (6.5a+6.5b+6.5c done; 6.5d roads remains) and re-tally the roll-up.

- [ ] **Step 2: Commit**
```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(roadmap): mark 6.5c wilderness batch Done"
```

---

## Self-review notes

**Spec coverage:**
- Spec §1 pack/kin relationships → Task 1 (3 leader mobs, family/friend edges, auto-mirror).
- Spec §2 facts + gossipers → Task 2 (4 facts; gossiper + knows_facts on the 5 accessible sentient mobs; warren rep-gating falls out of the hostility/idle gossip gate — no code).
- Spec §3 behavior tuning (none) → no task, by design.
- Spec validation → Task 3 (boot panic-gate + smoke).
- Roadmap → Task 4.

**Placeholder check:** all relationship blocks + all 4 facts given in full; exact mob ids/paths; the only soft step is the in-game smoke (deferrable per precedent).

**Consistency check:** every relationship `to:` id (205,206,223,217,218,222,74,72,73) and every `knows_facts` id matches a declared fact (Task 2 Step 1). warren_chieftain 75 is edited in both Task 1 (relationships) and Task 2 (gossiper+knows_facts) — non-conflicting additions to the same file. Fact ids are settlement/zone-scoped and distinct from the 6.5b facts.

**TDD note:** content validated by the boot-time loaders (panic on bad refs), not Go unit tests — consistent with the codebase. Task 3 is the gate.
