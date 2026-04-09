# Mutation Discovery Tuning — Design Spec

**Date**: 2026-04-07
**Status**: Approved

## Problem

Mutation discovery currently works as: acquire new mutations until at
max count, then deepen existing ones. This creates front-loaded
discovery where players collect a handful of mostly-common mutations
early on and only start deepening them much later. It also means
rarer mutations are purely luck-based regardless of how far along a
player's mutation journey is.

## Solution

Two changes to the discovery flow:

1. **Deepen-first bias**: When a player could either discover a new
   mutation or deepen an existing one, prefer deepening at a 70/30
   ratio. This makes early mutations grow stronger before the player
   collects a full set, creating a more gradual power curve.

2. **Rarity uplift**: When rolling a new mutation, players with
   higher average mutation levels see a gentle shift toward rarer
   results. This rewards progression — a player who has invested in
   deepening their mutations is more likely to discover interesting
   new ones when they do get a new discovery.

## Change 1: Deepen vs New Coin Flip

### Current Logic

```
if canAcquire (under max count):
    roll new mutation
else if canDeepen (any mutation below max level):
    deepen random mutation
```

### New Logic

```
if no mutations owned:
    acquire new (no choice)
else if canAcquire AND canDeepen:
    roll: 70% deepen, 30% new        ← NEW
else if canAcquire:
    acquire new (nothing to deepen)
else if canDeepen:
    deepen (at max count)
else:
    nothing (max count, all max level)
```

### Config

New balance knob:

| Key                    | Default | Description                                |
|------------------------|---------|--------------------------------------------|
| `MutationDeepenChance` | 0.70    | Probability of deepening over new discovery when both are possible |

Valid range: 0.0 (always new) to 1.0 (always deepen). Values outside
this range are clamped in validation.

### Edge Cases

- **No mutations owned**: Always acquire new. The coin flip only
  applies when there is something to deepen.
- **All mutations at max level**: Treated as "nothing to deepen" —
  acquire new if under max count, otherwise nothing happens.
- **At max count**: Always deepen if possible (same as today). The
  coin flip only matters when both options are available.

## Change 2: Rarity Uplift on New Acquisition

### Current Pool Weighting

Each unowned, non-conflicting mutation appears in the pool
`(11 - rarity)` times:

| Rarity | Weight |
|--------|--------|
| 1      | 10     |
| 5      | 6      |
| 8      | 3      |
| 10     | 1      |

### New Pool Weighting

Compute a rarity bonus from the player's average mutation level:

```
avgLevel = sum(all mutation levels) / count(mutations)
rarityBonus = floor(avgLevel) - 1
weight = max(1, 11 - rarity - rarityBonus)
```

If the player has no mutations, `rarityBonus` is 0 (normal weights).

### Effect by Average Level

**Average level 1 (bonus 0)** — no change, normal weights:

| Rarity | Weight |
|--------|--------|
| 1      | 10     |
| 5      | 6      |
| 10     | 1      |

**Average level 2 (bonus 1)** — gentle shift:

| Rarity | Weight |
|--------|--------|
| 1      | 9      |
| 5      | 5      |
| 10     | 1      |

**Average level 3 (bonus 2)** — noticeable shift:

| Rarity | Weight |
|--------|--------|
| 1      | 8      |
| 5      | 4      |
| 10     | 1      |

**Average level 4 (bonus 3)** — strong rare bias, commons still
possible:

| Rarity | Weight |
|--------|--------|
| 1      | 7      |
| 5      | 3      |
| 10     | 1      |

Commons are never eliminated — they just become proportionally less
likely relative to rares. Ultra-rare mutations (rarity 10) always
have weight 1 regardless of bonus due to the `max(1, ...)` floor.

### Integration

The rarity bonus is computed inside `GetWeightedPool()`. The function
already receives the player's `owned map[string]int` (mutation ID to
level), so the average level can be calculated from that map directly.

No new function signature needed — the behavior changes internally
based on the owned map contents.

## Scope Exclusions

- Mob mutation discovery is unchanged. Mobs use the same
  `RollAcquisition` but their pools are independent and they rarely
  reach high average levels.
- The quest-based `GiveMutation()` (direct grant) is unchanged — it
  bypasses the pool entirely.
- `RollDeepening()` (which mutation gets deepened) remains random
  among sub-max mutations. No weighting by rarity or level.
- The progress accumulation rate, load-based threshold scaling,
  conflict system, max count (5), and max level (4) are all
  unchanged.
- No changes to mutation effects, level multipliers, or visual
  descriptions.

## Files to Modify

- `internal/mutations/mutations.go` — update `GetWeightedPool()` to
  apply rarity bonus based on average mutation level
- `internal/hooks/NewRound_UserRoundTick.go` — restructure the
  acquire/deepen decision to use the coin flip
- `internal/hooks/NewRound_MobRoundTick.go` — same restructure for
  mob mutations (optional, could skip since mobs rarely hit this
  case)
- `internal/configs/config.balance.go` — add `MutationDeepenChance`
  field and validation
- `_datafiles/config.yaml` — add `MutationDeepenChance: 0.70`
