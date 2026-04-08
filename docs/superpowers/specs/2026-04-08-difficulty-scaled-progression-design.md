# Difficulty-Scaled Skill Progression

**Date:** 2026-04-08
**Status:** Draft
**Problem:** Players exploit low-difficulty spells (identify, glow) and cheap
recipes (iron daggers) to grind skill progression risk-free. Higher-difficulty
actions should reward proportionally more progression.

## Design

### Core Principle

Skill progression chance scales with the difficulty of the action performed.
Easy actions still progress skills, just slower. Hard actions reward more.
Discovery systems (spell discovery, recipe discovery) are **not affected** —
they remain separate rolls with their own decay curves.

---

### 1. Spellcasting Progression Changes

#### 1a. Remove Double Progression

Currently every spell fires `OnSkillUse` twice: once at cast initiation
(`actions/cast.go:294`) and once at resolution
(`NewRound_DoCombat_helpers.go:282`). Remove the **initiation trigger**.
Progression fires only when the spell completes and resolves.

Discovery already fires only at resolution, so this change has no impact
on spell/manifestation discovery.

**Files:** `internal/actions/cast.go` — remove `OnSkillUse` call at line 294.

#### 1b. Difficulty Bonus Multiplier

Pass the spell's `difficulty` field into `OnSkillUse` as a bonus multiplier.

**Formula:**
```
bonusMultiplier = 1.0 + difficulty * SpellDifficultyProgressionScale
```

**Config knob:** `SpellDifficultyProgressionScale` (default `0.01`, so
difficulty/100). At default:

| Spell           | Difficulty | Multiplier |
|-----------------|-----------|------------|
| Identify        | 0         | 1.0x       |
| Chrysalis Glow  | 0         | 1.0x       |
| Mass Mend       | 10        | 1.1x       |
| Sparks          | 50        | 1.5x       |
| Conviction Spike| 75        | 1.75x      |

#### 1c. Self-Target Penalty

When a HelpSingle spell's only target is the caster, apply a penalty
multiplier.

**Config knob:** `SelfCastProgressionMultiplier` (default `0.5`).

Stacks multiplicatively with the difficulty multiplier:
self-cast chrysalis-haste (difficulty 0) = 1.0 * 0.5 = 0.5x.

#### 1d. Empty-Room AoE Guard

HarmArea and HarmMulti spells that resolve with zero valid targets skip
progression entirely. Check at resolution time: if the spell script
reports no targets hit, do not call `OnSkillUse`.

**Files:** `internal/hooks/NewRound_DoCombat_helpers.go` — add target
count check before the resolution-time `OnSkillUse` call.
`internal/hooks/spell_resolution.go` — ensure resolved target count is
available in the return/result.

#### 1e. OnSkillUse Signature Change

Add an optional `bonusMultiplier` parameter to `OnSkillUse` (or add a
new method `OnSkillUseScaled`). The multiplier flows into
`CheckSkillProgression`'s existing `bonusMultiplier` parameter.

**Preferred approach:** Add `OnSkillUseScaled(skillName, userId, bonus)`
and have `OnSkillUse` call it with `1.0`. This avoids changing every
existing callsite.

---

### 2. Crafting Progression Changes

#### 2a. Skill-Minimum Bonus Multiplier

Pass the recipe's `skill_minimum` field into progression as a bonus
multiplier on craft completion.

**Formula:**
```
bonusMultiplier = 1.0 + skill_minimum * CraftDifficultyProgressionScale
```

**Config knob:** `CraftDifficultyProgressionScale` (default `0.02`, so
skill_minimum/50). At default:

| Recipe          | skill_minimum | Multiplier |
|-----------------|--------------|------------|
| Iron Dagger     | 0            | 1.0x       |
| Clarity Tonic   | 15           | 1.3x       |
| Mid-tier        | 25           | 1.5x       |
| High-tier       | 40           | 1.8x       |
| Endgame         | 50           | 2.0x       |

#### 2b. Salvage — No Change

Salvage uses its own dedicated skill with a 2.0x progression multiplier
and consumes an item each time. Material cost is a natural throttle.
No difficulty scaling needed.

**Files:**
- `internal/hooks/NewRound_UserRoundTick.go` — craft completion path
  (line ~342), pass `recipe.SkillMinimum` into scaled progression.
- `internal/hooks/NewRound_MobRoundTick.go` — mob craft completion
  (line ~354), same change.
- `internal/mobcommands/craft.go` — mob immediate-complete (line ~51).

---

### 3. Config Summary

All new config knobs live in the `Balance` section of `config.yaml`:

| Knob | Default | Description |
|------|---------|-------------|
| `SpellDifficultyProgressionScale` | 0.01 | Per-point difficulty bonus for spell skill progression |
| `CraftDifficultyProgressionScale` | 0.02 | Per-point skill_minimum bonus for craft skill progression |
| `SelfCastProgressionMultiplier` | 0.5 | Multiplier when spell's only target is self |

---

### 4. What This Does NOT Change

- **Spell/recipe discovery** — separate rolls, unaffected.
- **Stat progression via OnStatUse** — unaffected (stats progress from
  the auto-stat-track inside OnSkillUse, which still fires).
- **Bartering/buy/sell progression** — separate system, unaffected.
- **Salvage progression** — no change.
- **Mob progression** — mobs use the same OnSkillUse path, so they
  benefit from difficulty scaling too. This is intentional: mob casters
  using harder spells should also progress faster.

---

### 5. Files to Modify

| File | Change |
|------|--------|
| `internal/characters/progression.go` | Add `OnSkillUseScaled(skill, userId, bonus)` method |
| `internal/actions/cast.go` | Remove initiation-time `OnSkillUse` (line 294) |
| `internal/hooks/NewRound_DoCombat_helpers.go` | Pass spell difficulty into `OnSkillUseScaled` at resolution; add empty-room AoE guard; self-cast multiplier |
| `internal/hooks/spell_resolution.go` | Ensure target count is returned for AoE guard |
| `internal/hooks/NewRound_UserRoundTick.go` | Pass `recipe.SkillMinimum` into `OnSkillUseScaled` at craft completion |
| `internal/hooks/NewRound_MobRoundTick.go` | Same for mob crafting |
| `internal/mobcommands/craft.go` | Same for mob immediate-complete |
| `internal/configs/config.balance.go` | Add three new config knobs |
| `_datafiles/config.yaml` | Add defaults + comments for new knobs |
