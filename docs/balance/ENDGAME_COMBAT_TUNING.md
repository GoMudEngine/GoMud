# Endgame Combat Tuning — the Meirok Unit + the Nodrop-Gear Method

**Author reference for building and calibrating endgame fights.** Adopted
2026-07-07. This is the standing method for every endgame encounter from here on.

## 1. The difficulty unit: the Meirok

Difficulty is measured in **Meiroks**. One **Meirok + companions** = one unit of
endgame party power.

The Meirok is the canonical geared endgame character (`prod_meirok.yaml`, prod
user 24; the harness test clones are quester4/5/6 = "Vael"/"Ryn"/"Doss"):

- **HP ~610–660**, six stats in the **93–115** band (just above the 100 human
  baseline; use-based progression, not levels).
- Skills: **weapon-combat ~69**, unarmed ~57, spellcasting ~51, rhetoric ~55,
  plus master-tier crafts.
- **Extra-arms L1** → triple drowned-claw (a mixed **physical + magical +
  conviction** attacker), **conviction-ward** shield, **rally / warcry /
  conviction-surge** self-buffs, **skill-attunement / mutation-catalyst**, and a
  **summoned companion** (Steppe Spirit Wolf) that adds substantial DPS.
- Masterwork elemental gear across every slot (mitigation + statmods).

**"1 Meirok + companions"** = one such character plus its summoned pet(s). A fight
is specified as targeting **N Meiroks** (1, 2, 3, …). N Meiroks ≈ N× the DPS, N×
the effective HP, and N companion pets — plus the coordination tools of a party
(taunt-swap, cross-heals).

## 2. The two-axis method (the SOP)

Tune an endgame mob on **two independent axes**:

| Axis | Field | Controls |
|------|-------|----------|
| **Threat** | `statpool` | damage dealt, accuracy, *and* the mob's own HP (via vitality) |
| **Durability** | equipped **`NeverDrops` mitigation gear** | effective HP (EHP) — how long it survives the party's DPS |

**Never use `base_pool`** to make a mob tanky: `base_pool` scales the mob's
**melee damage** alongside its pool, which turns support/adds into killers (the
#22-adds lesson). `NeverDrops` gear adds durability with **no loot pollution**
(the gear never drops) and **no damage side-effect**.

### The nodrop gear recipe

A mitigation item (mirror `items/materials-40000/40227-arc_welded_repair_housing.yaml`):

```yaml
itemid: <next-free, id_inventory.py>
name: <Title Case, article-less preferred>
namesimple: <one word>
description: >-
  <in-world flavor for the piece>
type: body        # or head / neck / shoulders / legs / etc.
subtype: wearable
physical_mitigation: 55     # ints, percentages
magical_mitigation: 45
conviction_mitigation: 40
never_drops: true
not_salable: true
weight: 0.0
value: 0
```

Equip it on the mob via `character.equipment`:

```yaml
character:
  equipment:
    body:
      itemid: <the gear id>
    neck:
      itemid: <a second piece, for more layers>
```

**Which channel to weight:** match the party's damage mix. A Meirok party is
**mostly physical** (claws + wolf bites) with some magical (spells) and conviction
(taunt), so lead with **`physical_mitigation`** and back it with magical/
conviction. Total mitigation per channel caps at **75%** (`PhysicalMitigationCap`
etc.) — a single 65-phys piece is fine; don't stack past ~70 total.

## 3. The baseline anchors (empirical, tune by interpolation)

These are the settled values from live harness calibration. Use them as the
reference — a new fight targeting N Meiroks starts near the matching row.

| Target | Encounter | `statpool` | Nodrop mitigation (phys/mag/conv) | Mechanic |
|--------|-----------|-----------|-----------------------------------|----------|
| **1 Meirok** | #20 Pass-Apex (9541) | **1100** | hide 40230: **62/52/52** | none (pure stat/gear) |
| **2 Meirok** | #21 Sentinel (9552) | **2400** | carapace 40231: **65/52/48** + warding-core 40232: **10/45/45** | 1 light hook — "Rouse the Wards" (summon 2 adds at 50% HP) |
| **3 Meirok** | #22 Core Guardian (9562) | **5000** (instanced; effectively scales with gold buy-in) | Hull Plating 40225: **55/45/–** + Core Matrix 40226 | full apparatus (drain / telegraphed discharge / interrupt / 3 adds) |

**The heuristic that falls out:** each added Meirok roughly **doubles** the
required `statpool` (1100 → 2400 → 5000 ≈ ×2.2, ×2.1), while **physical
mitigation stays in the ~55–65% band** across all tiers. So for a new fight:
start `statpool` at `~1100 × 2^(N−1)`, put phys mitigation around 60%, then
verify live.

**Mechanical-depth ramp** (depth scales with N, not just numbers): **1 Meirok =
no mechanic** (a pure stat/gear check), **2 Meirok = one light telegraphed hook**
(so it isn't a sponge — e.g. summon adds / a heavy telegraphed hit / a self-ward),
**3 Meirok = a full counterplay system** (interrupt-or-die, focus-fire adds,
mechanics the party must answer). Bigger numbers never substitute for counterplay
against a party that out-DPSes any stat-brute.

## 4. Success criterion — "tough but winnable"

The intended-size party (N Meiroks):

- **wins**, but ends **below ~30% HP** on at least one member, and
- **had to spend cooldowns / consumables** (shields, heals, taunt-swap, summons) —
  a *face-tanking* party (no reactive play) should **lose a member**.
- A party **one size smaller** (N−1) should **lose or be forced to disengage**.

Fail states: a **faceroll** (win at >70%, no cooldowns burned) is under-tuned; a
**wipe** by a competently-played intended-size party is over-tuned.

**Calibration note:** the AI harness plays sub-optimally (it reacts late), so a
harness *win that required a late emergency heal* ≈ a *comfortable win* for a
coordinated human party. Tune to land the harness slightly on the **hard** side,
and cross-check against DPS/EHP math rather than a single run (combat has RNG).

## 5. The test method (the manual N-bridge conductor rig)

This is what actually calibrated #20/#21/#22 — **not** the `ptorch`/scenario
system. One driver puppeteers **N** geared quester characters through **N**
`mudagent` file-bridges, paced on the `Playtest.Round` beacon. N = the target
Meirok count.

- **Harness:** `../gomud-playtest-harness/mudagent.exe`; server **AIPort 55555**;
  `Modules.playtest` on for the round beacon.
- **Chars:** users **10/11/12** = quester4/5/6 (= Vael/Ryn/Doss), `isai: true`,
  `role: admin`, geared Meirok clones. Password **`smoke123test`**.
- **Bridge:** `tail -n +1 -f cmd.txt | mudagent.exe --target localhost:55555
  --user quester4 --password smoke123test > events.jsonl`. Append one command per
  line to `cmd.txt`; read JSON events (`output` / `gmcp` / `beacon` / `status`)
  from `events.jsonl`. **Bare spell-names cast directly** (`conviction-ward`).
  `set charset` toggles — converge to ASCII (watch for the flip).
- **Per run:** edit YAML → **nuke `mobs.instances/*`** (overworld stat edits are
  shadowed otherwise) → `go build -o gomud_smoke.exe .` (NOT `go build ./...` — it
  doesn't produce the exe) → boot → connect N bridges → `teleport 201` (the Test
  Arena — no ambient spawns) → `party invite`/`party accept` → **buff, then
  `mob spawn <id>`** to isolate a fresh 100% target → `attack` on all bridges →
  pace on the beacon, react competently (heal/taunt) as a real party would →
  judge vs §4.
- Buffs are **cooldown-gated** (~1 special buff / few rounds) — you cannot instant-
  stack them, which is realistic; the char's auto-recast triggers maintain them
  once landed.

### Known quirks to account for
- **84%-HP-on-arrival spawn anomaly:** overworld bosses sometimes display ~84% HP
  when you arrive in their room. Use `mob spawn <id>` to get a clean 100% target
  for calibration.
- **Mobs auto-wield their loot weapons:** e.g. the Sentinel wields its own
  Ironhorn Warbow drop, adding damage. The tuned numbers already account for this.
- Helper: a small `evttail.py` that pretty-prints the last N events (needs
  `PYTHONIOENCODING=utf-8` on Windows) makes reading the bridge output sane.

## 6. Provenance

Method + anchors established during the #20/#21 arc-wide geared-party calibration
(branch `feature/endgame-combat-calibration-2021`, 2026-07-07). Spec + plan under
`docs/superpowers/{specs,plans}/2026-07-07-endgame-combat-calibration-2021*`. The
#22 Core Guardian row is from the 2026-07-06 Crash Site boss redesign.
