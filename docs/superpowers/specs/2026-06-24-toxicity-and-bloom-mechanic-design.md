# Toxicity Overhaul + Bloom Mechanic (design)

**Status:** approved 2026-06-24 (brainstorming). The first of the capital
**quests-phase** enrichment work: the **Bloom mechanic** (the deferred
[[project-bloom-mechanic]]) — the foundation the **Bloom Trail questline** (next
spec) sits on. A pre-design audit found the existing **toxicity system is wired but
effectively inert**, so this spec covers BOTH: **Part A — make toxicity real +
visible + dangerous** (the foundation), then **Part B — the Bloom mechanic** layered
on it.

> **Push policy:** New Plymouth is built + prod-ready but the push is HELD by the
> user pending enrichment. This is enrichment; it merges to master, push stays held
> until the user calls it.

## 0. Scope (locked with user 2026-06-24)

IN (one combined spec):
- **Part A — Toxicity overhaul:** player visibility (status + prompt token + GMCP/web
  vitals + threshold messages); wire the 2 dead penalties (Perception, Dexterity);
  an acute high-toxicity consequence ("shortened life" harm); accumulation/decay
  tuning; `AddToxicity` clamp fix + consistent use.
- **Part B — Bloom mechanic:** a Bloom Wafer consumable; the Communion high
  (all-pools surge); the Crash; **addiction** (`BloomAddiction`); **withdrawal**;
  **mutation acceleration** (deepen strongest/most-used mutation); the toxicity cost
  (folds into Part A); and an **escape path** (endure / Ysolde detox).

OUT (deferred / separate):
- **The Bloom Trail questline** (Deren, the captive rescue, Marn's supply chain,
  detox-as-reward) — the IMMEDIATE next spec→plan→build; it *uses* this mechanic.
- A pure economy money-sink mechanic; legendary BIS crafts ([[project-legendary-bis-craft-items]]).
- Multiple Bloom grades (v1 = one grade; a purer Old-Quarter grade is a later knob).
- Reworking the spoiled-potion nausea (buff 75) — independent; left as-is.

**Locked design decisions (2026-06-24 brainstorming):**
1. The high = **all-pools "communion" surge** (#3) + immunities; the cost is a
   prolonged crash + toxicity + addiction + faster mutation progression.
2. **Shortened life folds into toxicity** (no separate stat — sustained high toxicity
   does real harm).
3. Mutation acceleration **leans to the strongest/most-used mutation** (occasional new
   random one); mutations are inherently pro+con so deepening cuts both ways.
4. Addiction is **brutal but escapable** (endure withdrawal or Ysolde detox; mutations
   gained are permanent).

---

## PART A — Toxicity overhaul (the foundation)

### A1. Audit findings being fixed (2026-06-24)
The current system (`Character.Toxicity float64`; `GetToxicityMax` = BaseMax 100 +
Vit/5 ≈ 120; `GetToxicityPenalties` → (regen, Per, Dex) at 50/75/90%; decay 1.0/round
in `NewRound_AutoHeal.go`; incremented in `drink.go` by `itemSpec.Toxicity`) is inert
because:
- **No player visibility** — absent from `status`, the prompt, GMCP/web vitals; no
  threshold warning message exists anywhere.
- **2 of 3 penalties are dead** — `NewRound_AutoHeal.go:114` does `toxRegenMult, _, _`,
  discarding the Perception & Dexterity multipliers. Only regen is applied.
- **The live penalty is imperceptible** (regen −10%..−40%).
- **Thresholds barely reachable** — decay (1/round) outpaces normal potion use
  (8–22 tox each); you'd chain ~8 potions to hit 50%.
- **No acute consequence** — no damage/sickness/gating; `drink.go` adds raw, even past
  max.

### A2. Visibility (make it perceptible)
1. **`status` command:** add a Toxicity line to the stat sheet (the deliberate
   mechanical-display exception to "no hard numbers" — show a descriptive tier AND,
   consistent with the existing pools, it may show a meter). Use descriptive tiers:
   *clear / queasy / sick / poisoned / critical* mapped to the <50/50/75/90/≥100%
   bands.
2. **Prompt token:** add a `{tox}` prompt token (descriptive/colored tier, like the
   `{enc}` encumbrance token) so players can surface it in their prompt.
3. **GMCP / web vitals:** add toxicity (current + max, or tier) to the GMCP `Char`
   vitals payload (`modules/gmcp/gmcp.Char.go`) so the web client dashboard shows it
   alongside HP/SP/CP. (The user plays via the web client — this is the key gap.)
4. **Threshold-crossing messages** (player-facing, descriptive — NO raw numbers):
   on crossing UP into a band, emit a sensory line, e.g. 50% "A faint nausea settles
   in and will not quite lift."; 75% "Your hands have developed a fine tremor and your
   sight swims at the edges."; 90% "Your whole body is in revolt — sweat, shakes, the
   taste of metal." On crossing DOWN, a relief line. Fire from the regen tick where
   toxicity changes; gate on band-change so it isn't spammy.

### A3. Wire the dead penalties (make it bite)
- Apply the **Perception** multiplier from `GetToxicityPenalties()` to Perception-based
  rolls (perception/search detection, the `shoot`/aimed-shot Perception path) and the
  **Dexterity** multiplier to Dex-based rolls (dodge, movement cost, hit) — at the
  appropriate roll sites. **The plan pins the exact application points via codegraph**
  (mirror how other transient multipliers are applied; prefer a single choke-point per
  stat where feasible — e.g. an effective-Perception / effective-Dexterity accessor —
  over scattering the multiply). Keep the regen application as-is.
- Stop discarding the values at `NewRound_AutoHeal.go:114` (it can still read regenMult
  there; Per/Dex apply at their own sites).

### A4. Acute consequence near max (the "shortened life")
- At the top band (≥90% and especially at/near max), add an **acute escalating harm**:
  a **toxicity-sickness effect** that does percentage-of-max HP damage per regen tick
  (small but real — the body poisoning itself) and/or applies a debuff. Scale with how
  far over the band you are. This is the danger that makes chronic Bloom (and potion
  abuse) genuinely costly — canon's "shortens the life." **Never instant-death from a
  single dose**, but sustained max-toxicity can kill if ignored (routes through the
  existing non-combat death path, like other DoTs).
- **Tuning:** all magnitudes are config knobs (new `ToxicitySicknessDamagePct`,
  band thresholds already exist) — ship sane defaults, tune after playtest (deferred-
  tuning policy).

### A5. Accumulation / decay tuning + `AddToxicity` fix
- **Decay:** make decay meaningful vs accumulation — options the plan will choose a
  default for: lower `ToxicityDecayPerTick`, and/or **slow decay at high toxicity**
  (the body clears the last of it slowly — a curve, not flat). Goal: sustained use
  *accumulates*; a single potion still clears in reasonable time. Config-driven.
- **`AddToxicity`:** change from "reject if over max" to **clamp to max** (and allow
  exceeding into the acute-harm band per A4 design — i.e., max is the *penalty* ceiling
  but toxicity can ride at max, not a hard wall that silently no-ops). Route
  `drink.go` (and Bloom) through `AddToxicity` consistently instead of raw `+=`.

### A6. Part A definition of done
- Toxicity visible in `status` + `{tox}` prompt token + GMCP/web vitals.
- Threshold-crossing messages fire (up + down), band-gated, descriptive.
- Perception & Dexterity penalties actually apply (verified by a unit test on the
  effective-stat accessors, and observably in play).
- High toxicity does acute, escalating, survivable-but-dangerous harm.
- Decay/accumulation tuned so sustained use matters; `AddToxicity` clamps + is used
  consistently. Boot + `go test ./...` green.

---

## PART B — the Bloom mechanic (layered on Part A)

### B1. The Bloom Wafer (item)
- A consumable **Bloom Wafer** (new item id, 40108+; type object/consumable on the
  eat/drink path — VERIFY the consumable type used by potions vs a new "drug" handling;
  reuse the `drink`/`eat` command path). Thin, iridescent, near-odorless (canon).
  `not_salable` is likely correct (contraband — not a general vendor good; it enters
  the world via the Bloom Trail quest / specific NPCs, not shops). One grade for v1.
- A high `Toxicity` value on the item (Bloom spikes toxicity hard — folds into A4).

### B2. The Communion (the high) — buff (90–95 block)
On consume → apply **Bloom Communion** (new buff, id in 90–95): for a meaningful
duration (~the canon "4-hour" feel — e.g. ~30–40 rounds, config knob):
- **Surge all three pools** — boost current HP/SP/CP (overfill or a large heal) + a
  strong regen multiplier across pools (reuse the regen-multiplier buff statmods).
- **Immunity to fear/pain debuffs** while active (a buff flag).
- A **minor all-stat lift** (small, tempting, not dominant).
- It *feels invincible* — the panic-button / "one more for this fight" lure. Player
  message on consume: a euphoric "communion" line (descriptive, no numbers).

### B3. The Crash — buff (90–95 block)
When Communion **expires** → apply **Bloom Crash** (new buff): a **prolonged** debuff
(longer than the high — e.g. 2–3× duration, config): pool penalties + regen penalty +
a stat sag (negative statmods). Plus, applied **at dose time** (not crash time):
- **Toxicity spike** (B1's item toxicity, via `AddToxicity`).
- **Addiction +1** (B4).
- **Mutation acceleration roll** (B5).
Crash severity scales with **addiction level** (B4) — deeper addicts crash harder.

### B4. Addiction (`Character.BloomAddiction`)
- New persisted field `BloomAddiction int` (yaml `bloom_addiction,omitempty`) on
  Character, default 0.
- **+1 per dose** (or scaled — config). **Decays slowly with abstinence** (e.g. −1 per
  N rounds of no Bloom — much slower than it builds; config knob).
- Governs: crash severity, **withdrawal** onset speed + severity, craving message
  frequency. Higher addiction = the trap tightens.
- Surfaced descriptively (status line when > 0: "You feel the pull of the Bloom." →
  escalating). Not a raw number in normal display (a tier).

### B5. Mutation acceleration
On each dose, roll (high chance, config `BloomMutationAdvanceChance`) to **advance the
character's strongest/most-used mutation** by one level (cap-aware — `Mutations` is
`map[string]int`; pick the **highest-level** entry, ties broken deterministically; if
the player has NO mutations, seed/advance a default — canon's bark-skin, if a bark-skin
mutation exists or is added). A **smaller** chance to grant a **new random** mutation
instead. Reuse the existing mutation-progression rails (cf. buff 73 mutagen-brew / 74
chrysalis-catalyst, which already advance mutations) — **the plan pins the exact
cap-aware advance call** (likely incrementing `Mutations[id]` within the slot/cap
validators in `internal/characters/validate.go:validateMutationSlots` /
`mutations` pkg). Mutations carry pro+con, so deepening is the built-in gamble. A
descriptive message on advance ("Something under your skin shifts and settles
differently.").

### B6. Withdrawal
- While `BloomAddiction > 0` and **no Bloom for `BloomWithdrawalOnsetRounds`**, apply
  an escalating **Bloom Withdrawal** buff (90–95 block): worsening pool/regen/stat
  penalties + frequent craving messages ("Your skin crawls; the Bloom is all you can
  think of."). Severity scales with addiction level + time since last dose.
- **Dosing resets** withdrawal (the trap). This — not the high — is why deep addicts
  keep using.
- Withdrawal stacks with the toxicity penalties (Part A) for a genuinely rough state.

### B7. The toxicity cost (folds into Part A)
Bloom's high item-toxicity (B1) + repeated dosing keeps toxicity in the dangerous
bands → Part A's acute harm (A4) IS the "shortened life." No separate Bloom-specific
life stat. A chronic Bloom user lives at high toxicity, taking ongoing harm, with the
mutation load deepening — the slow burn.

### B8. Escape (brutal but escapable)
- **Endure:** stop dosing. Toxicity decays (A5), withdrawal eventually subsides (B6),
  `BloomAddiction` slowly drops (B4). Hard because withdrawal hurts — but free.
- **Detox via Ysolde (mob 9323, Common back-alley healer — canon treats addicts):** a
  **detox service/cure** that accelerates the kick — clears toxicity faster and/or
  steps down `BloomAddiction` (a multi-step or paid/quest-gated service; the Bloom
  Trail quest can make a full cure its reward). The plan defines the exact mechanism
  (a dialogue-triggered effect / a consumable cure item Ysolde provides).
- **Mutations are permanent** — you are what Bloom made you. (Mutation *reversal* is
  out of scope; a future "cure mutation" is a separate idea.)

### B9. Part B definition of done
- Bloom Wafer consumable exists; consuming it applies Communion → (on expiry) Crash;
  toxicity spikes, addiction +1, mutation-advance rolls — all observable.
- Addiction persists, decays slowly; withdrawal onsets on abstinence and resets on
  dosing; both scale with addiction level.
- Mutation acceleration advances the strongest mutation cap-aware (unit-tested).
- Toxicity (Part A) provides the life-cost; chronic use is genuinely dangerous.
- Ysolde detox path works (reduces addiction/toxicity).
- All magnitudes are config knobs with sane defaults. Boot + `go test ./...` green;
  a `/playtest` smoke (dose → high → crash → withdrawal → detox) verified.

---

## Architecture & isolation

- **Toxicity** logic stays in `internal/characters/resources.go` (+ the regen tick in
  `internal/hooks/NewRound_AutoHeal.go`); visibility touches `status.go`, the prompt
  package, `modules/gmcp/gmcp.Char.go`. Effective-stat accessors centralize the Per/Dex
  application.
- **Bloom** is mostly **data** (the wafer item + the 90–95 buffs with statmods/flags)
  + a thin code layer: the consume hook (dose → communion + toxicity + addiction++ +
  mutation roll), a crash-on-expiry hook (buff-expiry event), the addiction field +
  decay/withdrawal tick (a NewRound hook), and the Ysolde detox effect. Reuse buff
  statmods/flags and the mutation-progression rails so the new Go surface is small and
  testable.
- **Config:** a Bloom/Toxicity block in `Balance` — every magnitude a knob
  (deferred-tuning policy: ship defaults, tune after playtest).

## Testing
- Unit tests: toxicity Per/Dex effective-stat accessors; band-crossing message gating;
  acute-harm damage; `AddToxicity` clamp; `BloomAddiction` increment/decay; mutation
  cap-aware advance (strongest pick + cap respect); withdrawal onset/reset.
- Boot + `/playtest` smoke for the full loop.

## Honored conventions
No hard numbers in player-facing text (descriptive tiers; `status` is the deliberate
mechanical exception) · config knobs for all balance · buffs in the reserved 90–95
block · reuse toxicity/mutation/buff rails · `go test ./...` + boot green before merge ·
push stays HELD.
