# 6e — Mutation Balance Pass + Progression Rate Design

**Date:** 2026-07-13
**Status:** Design approved (drift-signals, ceilings, magnitude-depth, rates locked)
**Closes:** the Chrysalis mutation-graph arc. After this ships + boots clean, the
whole batch (~180 commits) is pushable.

---

## 1. Purpose

The mutation graph is built and every player/creature is migrated onto it — but
three things keep it from feeling *good*: three clusters are unreachable by normal
play, apexes scale as if they were keystones, and both mutation and stat
progression are far too slow to experience. This pass fixes reachability and the
rank framework, sanity-checks magnitudes, and — per playtest feedback — makes
gaining and deepening mutations (and gaining stats) feel impactful and *happen*.

Guiding rule (user + §7): **gaining or deepening a mutation must feel like a real
upgrade or playstyle shift — never a decimal tweak.** Each rank is a noticeable
step; max rank is a dramatic jump.

## 2. Drift-signal combat hooks (reachability)

`skillClusters` currently gives a drift signal to 6 of 9 clusters. **Ironhide,
Weaver, and Trickster have none** — no skill cleanly maps to tank / control /
illusion — so they are only reachable via phials (unbuilt). Fix: drift from what
a player *does* in combat, mirroring the existing skill-drift path
(`Character.AddClusterAffinity(cluster, amount)` — see `progression.go:243`).

Add these hooks in the combat pipeline, each granting affinity to a character
(players are the reachability concern; hooks are character-general):

| Behavior | Where | Grants |
|---|---|---|
| A blow is **mitigated/absorbed** (armor/natural-armor/mitigation reduces incoming damage) | damage pipeline, mitigation step | `+ironhide` |
| A blow is **dodged or parried** | defense resolution (best-of-all defense) | `+trickster` |
| You **land a debuff/root/snare/control** (on_hit_buff, aura_enemy_debuff, reflect-rider) | the effect-application sites | `+weaver` |

- Gain per event = new config knob **`MutationAffinityPerCombatEvent`** (default
  `1.0`, matching `MutationAffinityPerSkillUse`).
- Guard against per-round spam: at most one grant of each type per combat round
  per character (combat fires many sub-events; one affinity tick per round per
  behavior keeps parity with once-per-action skill drift).
- Affinity still decays (`MutationAffinityDecay`) — sustained tanking/evading/
  controlling is what builds the path, exactly like sustained skill use.

## 3. Per-mutation rank ceilings

New optional field on `MutationSpec`:

```go
MaxRank int `yaml:"max_rank,omitempty"` // 0/unset => global MutationMaxLevel (4)
```

- **Apexes → `max_rank: 1`** (a transformation is binary — you *are* a colossus;
  there is no "more transformed"). The 13 apexes: `colossus-form`,
  `living-carapace`, `apex-predator`, `chameleon-skin`, `discorporation`,
  `brood-mother`, `radiant-avatar`, `paralytic-field`, `translucent-body`,
  `winged-flight`, plus the Chrysifier apexes `faithwrought`, `homunculus`,
  `walking-chrysalis`.
- **Keystones** keep the default (deepen 1→4).
- Consumers honor the per-mutation cap instead of the global one:
  - `CanDeepen` (`mutations.go:359`) — deepenable only if `level < effectiveMax(spec)`.
  - `RollDeepening` (`mutations.go:370`) — only eligible below its own cap.
  - `effectiveMax(spec) = spec.MaxRank if >0 else MutationMaxLevel`.
- Boot validator: `max_rank` must be in `[1, MutationMaxLevel]`.

## 4. Non-linear deepening curve

Deepening must *escalate the effect*, not nudge a decimal. Replace the flat
`MutationLevelNMultiplier` values with an **accelerating** curve and a big
max-rank payoff (applies to keystones ranks 2–4; rank 1 = ×1.0):

```yaml
MutationLevel2Multiplier: 1.6   # was 1.5 — a clear step
MutationLevel3Multiplier: 2.5   # was 2.0 — a bigger step
MutationLevel4Multiplier: 4.0   # was 2.5 — a dramatic payoff for deep investment
```

The `LevelMultiplier` code is unchanged (it already reads these three knobs); only
the values change. Flag-type effects still don't scale (a flag is binary).

## 5. Acquisition + progression rates

Playtest showed acquisition is broken (Meirok: ~2 mutations in dozens of hours).
Make the graph *live*.

### 5.1 Mutation acquisition — ~8× faster
```yaml
MutationBaseProgress: 15.0        # was 60 — ~4x lower threshold to first mutation
MutationProgressGainPerRound: 2.0 # was 1.0 — 2x progress/round  (combined ≈ 8x)
MutationProgressScale: 1.5        # unchanged — later mutations still cost more, 8x sooner
MutationMaxCount: 8               # was 6 — room for a full cluster spine + apex + a splash
```
Affinity is not the bottleneck (it accumulates fast and a keystone's rarity gate
is low); the threshold + per-round gain are. This lands in the user's 5–10× target.

### 5.2 Stat progression — ~2.25× faster (all six stats, skills untouched)
New global knob multiplied into `CheckStatProgression` (`progression.go:174`):

```go
// progression.go, in CheckStatProgression's chance calc:
chance := CalculateProgressionChance(...) * bonusMultiplier * mutStatMult * statMult * float64(b.StatProgressionRate)
```
```yaml
StatProgressionRate: 2.25   # new; default 1.0. Uniform across all stats; skills unaffected.
```
Per-stat `StatProgressionMultipliers` keep their carefully-tuned *relative* values;
this knob scales the whole set.

## 6. Magnitude consistency + sanity pass

Not a full numeric audit (fine-tuning defers to playtest) — a **consistency sweep**
to catch outliers before play:

- **Method:** a script/report dumps every mutation's effects grouped by type, with
  the value distribution per type. Review each band against the combat pipeline:
  - `reflect_damage` (return-damage %) — sane vs. the `returnPct` math and the new
    ×4.0 rank-4 (a rank-4 reflect must not exceed ~full-return).
  - `natural_armor` / `_mitigation` — vs. the 75% mitigation caps.
  - `_damage_reduction` (magical/conviction) — must be **fractions 0.0–1.0** (a
    known gotcha) and stay sane at ×4.0.
  - `stat_multiplier` / `stat_flat` / `dodge_modifier` / `health_multiplier` /
    `spell_power` — coherent bands; no accidental order-of-magnitude outlier.
  - Buff magnitudes (100–116) — DoT %, debuff stat drops reasonable at their
    trigger counts.
- **Fix** clear outliers only; document the review.
- **Regression guard:** a `internal/devtools` **bounds test** asserting each
  effect type stays within a documented sane range (advisory bounds generous
  enough to allow playtest tuning, tight enough to catch a fat-finger). Fails CI
  on an out-of-band value.
- **Explicitly deferred:** per-mutation, play-validated fine numbers → the
  post-ship playtest.

## 7. Testing

- **Unit:** per-mutation `max_rank` (apex can't deepen past 1; keystone can reach
  4); the `effectiveMax` helper; combat-hook affinity grants (mitigated blow →
  +ironhide, dodge → +trickster, debuff-landed → +weaver; once-per-round cap).
- **Devtools:** the magnitude-bounds test (§6); a drift-coverage test asserting
  all 9 clusters are now reachable (skillClusters ∪ combat-hook clusters == the 9).
- **Boot smoke:** the `max_rank` validator + curve knobs load clean.
- **Full suite** `go test ./...` → all packages ok.
- **Manual (post-ship):** playtest to fine-tune magnitudes + confirm the rates
  feel right (the numbers here are the *target*, not the final word).

## 8. Config knobs changed/added (summary)

| Knob | From | To | Why |
|---|---|---|---|
| `MutationLevel2Multiplier` | 1.5 | 1.6 | non-linear curve |
| `MutationLevel3Multiplier` | 2.0 | 2.5 | non-linear curve |
| `MutationLevel4Multiplier` | 2.5 | 4.0 | big max-rank payoff |
| `MutationBaseProgress` | 60 | 15 | ~8× acquisition |
| `MutationProgressGainPerRound` | 1.0 | 2.0 | ~8× acquisition |
| `MutationMaxCount` | 6 | 8 | full cluster build |
| `MutationAffinityPerCombatEvent` | — | 1.0 (new) | drift hooks |
| `StatProgressionRate` | — | 2.25 (new) | ~2.25× stat rate |

## 9. Out of scope

- Play-validated final magnitudes (post-ship playtest).
- Directional re-spec phials (separate feature).
- Skill-progression rate changes (stats only, per the request).
- New mutations / clusters (graph is complete).
