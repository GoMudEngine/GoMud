# Phase 2: Companion Consolidation + Config-Driven Buff Ticks

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace 13 near-identical companion spell JS files with one parameterized Go function, and replace ~10 healing/DoT buff JS files with YAML config fields and a Go tick handler.

**Architecture:** New YAML fields on SpellData (summon params) and BuffSpec (tick config). One `ResolveCompanionSummon()` function handles all companion spawning. Buff tick amounts are snapshot at application time (with stat scaling for spells, flat for potions) and applied by a Go handler each tick.

**Tech Stack:** Go, YAML, existing combat/damage_pipeline and scripting packages

**Spec:** `docs/superpowers/specs/2026-04-11-phase2-companion-bufftick-design.md`

---

### Task 1: Add Summon Fields to SpellData

**Files:**
- Modify: `internal/spells/spells.go`

- [ ] **Step 1: Add 6 summon fields to SpellData struct**

In `internal/spells/spells.go`, after the `QuestRequired` field (before the text fields comment), add:

```go
	// Companion summoning fields — replaces JS onMagic for summon spells
	SummonMobId          int    `yaml:"summon_mob_id,omitempty"`
	SummonBasePool       int    `yaml:"summon_base_pool,omitempty"`
	SummonScalingDivisor int    `yaml:"summon_scaling_divisor,omitempty"`
	SummonComponentId    int    `yaml:"summon_component_id,omitempty"`
	SummonRequiresCorpse bool   `yaml:"summon_requires_corpse,omitempty"`
	SummonMinCorpsePool  int    `yaml:"summon_min_corpse_pool,omitempty"`
```

- [ ] **Step 2: Add validation in Validate()**

After the existing difficulty clamping and text token validation, add:

```go
	// Validate summon fields
	if s.SummonMobId > 0 && s.SummonBasePool == 0 {
		mudlog.Warn("Spell.Validate", "spellId", s.SpellId, "warning", "summon_mob_id set but summon_base_pool is 0")
	}
	if s.SummonRequiresCorpse && s.SummonMinCorpsePool == 0 {
		mudlog.Warn("Spell.Validate", "spellId", s.SpellId, "warning", "summon_requires_corpse set but summon_min_corpse_pool is 0")
	}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/spells/...`

- [ ] **Step 4: Commit**

```bash
git add internal/spells/spells.go
git commit -m "feat: add companion summon YAML fields to SpellData

Six fields: summon_mob_id, summon_base_pool, summon_scaling_divisor,
summon_component_id, summon_requires_corpse, summon_min_corpse_pool.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Add Tick and Cure Fields to BuffSpec + TickAmount to Buff

**Files:**
- Modify: `internal/buffs/buffspec.go`
- Modify: `internal/buffs/buffs.go`

- [ ] **Step 1: Add 5 fields to BuffSpec struct**

In `internal/buffs/buffspec.go`, after the text fields block, add:

```go
	// Config-driven tick fields — replaces JS onTrigger for heal/DoT buffs
	TickPool         string `yaml:"tick_pool,omitempty"`          // "health", "stamina", "conviction"
	TickPercent      float64 `yaml:"tick_percent,omitempty"`      // Base % of max pool. Positive=heal, negative=damage
	TickVariance     float64 `yaml:"tick_variance,omitempty"`     // Random variance added to percent
	TickMin          int    `yaml:"tick_min,omitempty"`           // Minimum absolute tick amount (default 1)
	StartRemoveBuffs []int  `yaml:"start_remove_buffs,omitempty"` // Buff IDs to remove when this buff starts
```

- [ ] **Step 2: Add validation in BuffSpec.Validate()**

After the existing text token validation, add:

```go
	// Validate tick fields
	if b.TickPool != "" {
		switch b.TickPool {
		case "health", "stamina", "conviction":
			// valid
		default:
			return fmt.Errorf("buffId %d (%s) has invalid tick_pool %q (must be health/stamina/conviction)", b.BuffId, b.Name, b.TickPool)
		}
		if b.TickPercent == 0 {
			mudlog.Warn("Buff.Validate", "buffId", b.BuffId, "warning", "tick_pool set but tick_percent is 0")
		}
	}
```

- [ ] **Step 3: Add TickAmount to Buff struct**

In `internal/buffs/buffs.go`, after the `TriggersLeft` field in the `Buff` struct, add:

```go
	TickAmount int `yaml:"tickamount,omitempty"` // Snapshot: computed at application time, applied each trigger
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/buffs/...`

- [ ] **Step 5: Commit**

```bash
git add internal/buffs/buffspec.go internal/buffs/buffs.go
git commit -m "feat: add tick config and cure fields to BuffSpec, TickAmount to Buff

BuffSpec: tick_pool, tick_percent, tick_variance, tick_min, start_remove_buffs.
Buff instance: TickAmount for snapshot-based tick application.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Write ResolveCompanionSummon Function

**Files:**
- Create: `internal/hooks/companion_summon.go`

This is the core function replacing 13 JS onMagic scripts. It lives in the hooks package alongside spell_resolution.go.

- [ ] **Step 1: Create companion_summon.go**

Create `internal/hooks/companion_summon.go`. The function needs to:

1. Check companion cap via `caster.Character.GetMaxCompanions()` and `len(caster.Character.Companions)`
2. If `spellData.SummonComponentId > 0`: iterate `caster.Character.GetBackpackItems()`, find matching ItemId, call `caster.Character.RemoveItem(item)` to consume it. Error if not found.
3. If `spellData.SummonRequiresCorpse`: get room via `rooms.LoadRoom(caster.Character.RoomId)`, iterate `room.Corpses`, find valid corpse (not player corpse: `c.UserId == 0`, not companion: `!c.WasCompanion`, pool >= `spellData.SummonMinCorpsePool`). If `spellRest` is non-empty, match against corpse name (case-insensitive contains). Call `room.RemoveCorpse(corpse)`. Error if no valid corpse.
4. Compute scaled pool: `scale := 1.0 + float64(charisma)/float64(divisor) + float64(manifestationSkill)*0.02`. If divisor is 0, default to 500. `pool := int(math.Round(float64(spellData.SummonBasePool) * scale))`. If corpse was consumed, average: `pool = (pool + corpsePool) / 2`.
5. Spawn: `room.SpawnMob(spellData.SummonMobId)` if base_pool is 0, otherwise look up how `SpawnMobScaled` works in the scripting layer (it may be on the room object or a separate function). Set charm: find the spawned mob instance, call `mob.Character.Charm.Set(caster.UserId, 99999)` or equivalent. Register companion via `caster.Character.AddCompanion(...)`.

**Important:** Read the existing JS files (`raise-skeleton.js`, `conjure-water.js`, `summon-hive-swarm.js`) to understand the exact API calls and their Go equivalents. Read `internal/scripting/actor_func.go` to find how `CharmSet`, `AddCompanion`, `SpawnMobScaled` are bridged — the Go-side functions they call are what you need.

Read `internal/rooms/rooms.go` for `SpawnMob`/`SpawnMobScaled` signatures and `Corpse` struct definition.

Read `internal/characters/companions.go` for `CompanionInfo` struct and `AddCompanion` signature.

Error messages should be descriptive (no raw numbers), sent only to caster via `user.SendText()`.

Function signature:
```go
func resolveCompanionSummon(user *users.UserRecord, spellData *spells.SpellData, spellRest string, room *rooms.Room) bool
```

Returns true if summon succeeded, false if it failed (error already sent to user).

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/hooks/...`

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/companion_summon.go
git commit -m "feat: add ResolveCompanionSummon for YAML-driven summoning

Handles companion cap, component consumption, corpse consumption,
stat scaling, mob spawning, charm, and companion registration.
Replaces 13 JS onMagic scripts with one parameterized Go function.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Wire Companion Summon into Spell Resolution

**Files:**
- Modify: `internal/hooks/spell_resolution.go`

- [ ] **Step 1: Add summon resolution to the spell effect pipeline**

In `spell_resolution.go`, find the `resolveSpell` function. After the YAML magic text is sent and BEFORE the `onMagic` JS call (around line 170), add a check:

```go
	// Resolve companion summon (if configured)
	if spellData.SummonMobId > 0 {
		resolveCompanionSummon(user, spellData, cs.SpellRest, room)
	}
```

The JS `onMagic` call still runs after this — but for migrated spells, the JS file won't exist, so it's a no-op. For non-migrated spells (during the transition), both could theoretically run, but we'll delete the JS files immediately after adding the YAML fields.

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/hooks/...`

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/spell_resolution.go
git commit -m "feat: wire companion summon resolution into spell pipeline

Calls resolveCompanionSummon when summon_mob_id > 0, before JS onMagic.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Write Tick Snapshot Computation

**Files:**
- Create: `internal/buffs/tick.go`
- Create: `internal/buffs/tick_test.go`

- [ ] **Step 1: Write tests for snapshot computation**

Create `internal/buffs/tick_test.go`:

```go
package buffs

import (
	"math"
	"testing"
)

func TestComputeTickAmount_HealNoScaling(t *testing.T) {
	// Potion path: 5% of 200 max HP = 10
	amount := ComputeTickAmount(200, 0.05, 0, 1, 1.0)
	if amount != 10 {
		t.Errorf("expected 10, got %d", amount)
	}
}

func TestComputeTickAmount_DamageNoScaling(t *testing.T) {
	// DoT: -8% of 200 max HP = -16
	amount := ComputeTickAmount(200, -0.08, 0, 3, 1.0)
	if amount != -16 {
		t.Errorf("expected -16, got %d", amount)
	}
}

func TestComputeTickAmount_MinFloor(t *testing.T) {
	// 5% of 10 max HP = 0.5, should floor to min of 1
	amount := ComputeTickAmount(10, 0.05, 0, 1, 1.0)
	if amount != 1 {
		t.Errorf("expected 1, got %d", amount)
	}
}

func TestComputeTickAmount_DamageMinFloor(t *testing.T) {
	// -5% of 10 = -0.5, should floor to -3 (min=3 for damage)
	amount := ComputeTickAmount(10, -0.05, 0, 3, 1.0)
	if amount != -3 {
		t.Errorf("expected -3, got %d", amount)
	}
}

func TestComputeTickAmount_WithSpellScaling(t *testing.T) {
	// Spell: 5% of 200 = 10, scaled by 2.0x = 20
	amount := ComputeTickAmount(200, 0.05, 0, 1, 2.0)
	if amount != 20 {
		t.Errorf("expected 20, got %d", amount)
	}
}

func TestComputeTickAmount_ZeroPercent(t *testing.T) {
	amount := ComputeTickAmount(200, 0, 0, 1, 1.0)
	if amount != 0 {
		t.Errorf("expected 0, got %d", amount)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v ./internal/buffs/... -run TestComputeTickAmount`
Expected: compilation failure

- [ ] **Step 3: Write the implementation**

Create `internal/buffs/tick.go`:

```go
package buffs

import (
	"math"
	"math/rand"
)

// ComputeTickAmount calculates the per-tick heal/damage amount.
// maxPool: target's max HP/SP/CP for the relevant pool.
// percent: base percentage (positive=heal, negative=damage).
// variance: random variance added to percent (0=none).
// minAmount: minimum absolute value (1 if not specified).
// scalingMult: spell skill multiplier (1.0 for potions/non-spell).
// Returns the signed tick amount (positive=heal, negative=damage).
func ComputeTickAmount(maxPool int, percent float64, variance float64, minAmount int, scalingMult float64) int {
	if percent == 0 {
		return 0
	}
	if minAmount < 1 {
		minAmount = 1
	}

	effectivePercent := percent
	if variance > 0 {
		effectivePercent += rand.Float64() * variance
	}

	base := float64(maxPool) * effectivePercent
	scaled := base * scalingMult
	amount := int(math.Round(math.Abs(scaled)))

	if amount < minAmount {
		amount = minAmount
	}

	if percent < 0 {
		return -amount
	}
	return amount
}
```

- [ ] **Step 4: Run tests**

Run: `go test -v ./internal/buffs/... -run TestComputeTickAmount`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/buffs/tick.go internal/buffs/tick_test.go
git commit -m "feat: add ComputeTickAmount for snapshot-based buff ticks

Computes heal/damage amount from percentage of max pool, with
optional variance and spell skill scaling multiplier.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Wire Tick Snapshot into Spell Resolution (Spell Path)

**Files:**
- Modify: `internal/hooks/spell_resolution.go`

When a spell applies buffs via `buff_ids`, compute the tick snapshot for each buff that has `tick_pool` configured. This is the spell path — stat scaling applies.

- [ ] **Step 1: Add snapshot computation after buff application**

Find the two buff application sites in `spell_resolution.go`:

**Mob target** (around line 364-366): After `mob.AddBuff(buffId, "spell")`, add snapshot logic.

**Player target** (around line 617-619): After `target.AddBuff(buffId, "spell")`, add snapshot logic.

At each site, after the `AddBuff` call, insert:

```go
		// Compute tick snapshot for config-driven buffs
		if buffSpec := buffs.GetBuffSpec(buffId); buffSpec != nil && buffSpec.TickPool != "" {
			skillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
			scalingMult := combat.SkillMultiplier(skillLevel)
			// Apply weapon spell damage multiplier if equipped
			if user.Character.Equipment.Weapon.ItemId > 0 {
				if sdm := items.GetItemSpec(user.Character.Equipment.Weapon.ItemId); sdm != nil && sdm.SpellDamageMultiplier > 0 {
					scalingMult *= sdm.SpellDamageMultiplier
				}
			}
			var maxPool int
			switch buffSpec.TickPool {
			case "health":
				maxPool = targetChar.HealthMax.Value
			case "stamina":
				maxPool = targetChar.StaminaMax.Value
			case "conviction":
				maxPool = targetChar.ConvictionMax.Value
			}
			tickAmt := buffs.ComputeTickAmount(maxPool, buffSpec.TickPercent, buffSpec.TickVariance, buffSpec.TickMin, scalingMult)
			// Store on the buff instance
			targetChar.Buffs.SetTickAmount(buffId, tickAmt)
		}
```

Note: `targetChar` is `mob.Character` for the mob path and `target.Character` for the player path. Adjust variable names to match what's in scope.

You'll need to add a `SetTickAmount` method to the `Buffs` type — see Step 2.

- [ ] **Step 2: Add SetTickAmount to Buffs**

In `internal/buffs/buffs.go`, add a method to set the tick amount on a recently-added buff:

```go
// SetTickAmount sets the TickAmount on the most recently added buff with
// the given buffId. Called right after AddBuff to set the snapshot.
func (b *Buffs) SetTickAmount(buffId int, amount int) {
	// Search backwards since we just added it
	for i := len(*b) - 1; i >= 0; i-- {
		if (*b)[i].BuffId == buffId {
			(*b)[i].TickAmount = amount
			return
		}
	}
}
```

Check what the `Buffs` type actually is (likely `[]Buff` or a wrapper). Read `buffs.go` to confirm and adjust the receiver type.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/hooks/... ./internal/buffs/...`

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/spell_resolution.go internal/buffs/buffs.go
git commit -m "feat: compute tick snapshot on spell buff application

When a spell applies a buff with tick_pool configured, computes
TickAmount using caster's spellcasting skill and weapon multiplier.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Wire Tick Snapshot into Potion Path

**Files:**
- Modify: `internal/usercommands/drink.go`

Potions apply buffs without stat scaling. After `AddBuff` or `AddBuffScaled`, compute the tick snapshot with `scalingMult = 1.0`.

- [ ] **Step 1: Add snapshot after potion buff application**

In `internal/usercommands/drink.go`, find where buffs are applied (around lines 143-150). After each `AddBuff`/`AddBuffScaled` call, add:

```go
	// Compute tick snapshot for config-driven buffs (no stat scaling for potions)
	if buffSpec := buffs.GetBuffSpec(buffId); buffSpec != nil && buffSpec.TickPool != "" {
		var maxPool int
		switch buffSpec.TickPool {
		case "health":
			maxPool = user.Character.HealthMax.Value
		case "stamina":
			maxPool = user.Character.StaminaMax.Value
		case "conviction":
			maxPool = user.Character.ConvictionMax.Value
		}
		tickAmt := buffs.ComputeTickAmount(maxPool, buffSpec.TickPercent, buffSpec.TickVariance, buffSpec.TickMin, 1.0)
		user.Character.Buffs.SetTickAmount(buffId, tickAmt)
	}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/usercommands/...`

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/drink.go
git commit -m "feat: compute tick snapshot on potion buff application

Potions use scalingMult=1.0 (no stat scaling). Snapshot stored on
the buff instance for the Go tick handler.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Write Auto-Tick Handler

**Files:**
- Modify: `internal/hooks/NewRound_UserRoundTick.go`

- [ ] **Step 1: Add auto-tick before JS onTrigger**

In `NewRound_UserRoundTick.go`, inside the triggered buffs loop, AFTER the YAML trigger text is sent but BEFORE `scripting.TryBuffScriptEvent("onTrigger", ...)`, insert:

```go
		// Apply config-driven tick amount (if set)
		if buff.TickAmount != 0 {
			tickBuffSpec := buffs.GetBuffSpec(buff.BuffId)
			if tickBuffSpec != nil {
				switch tickBuffSpec.TickPool {
				case "health":
					user.Character.AddHealth(buff.TickAmount)
				case "stamina":
					user.Character.AddStamina(buff.TickAmount)
				case "conviction":
					user.Character.AddConviction(buff.TickAmount)
				}
			}
		}
```

Check that `AddHealth`, `AddStamina`, `AddConviction` exist on Character and accept signed ints (negative for damage). Read the Character methods to confirm signatures.

- [ ] **Step 2: Also handle mob buff ticks**

Check if there's a mob equivalent of the buff trigger loop (mobs may have their own tick handler). If mobs can have tick buffs (DoTs applied by players), add the same logic there. Search for `TryBuffScriptEvent("onTrigger"` in mob tick handlers.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/hooks/...`

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/NewRound_UserRoundTick.go
git commit -m "feat: add config-driven auto-tick handler for buff ticks

Applies TickAmount to the correct pool each trigger tick, before
calling JS onTrigger. Replaces JS heal/DoT calculation logic.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Wire start_remove_buffs Handler

**Files:**
- Modify: `internal/hooks/Buff_ApplyBuffs.go`

- [ ] **Step 1: Add buff removal on start**

In `Buff_ApplyBuffs.go`, after the YAML start text is sent and before the `onStart` script call, insert:

```go
		// Remove buffs listed in start_remove_buffs (cure effects)
		buffSpec := buffs.GetBuffSpec(evt.BuffId)
		if buffSpec != nil && len(buffSpec.StartRemoveBuffs) > 0 {
			for _, removeId := range buffSpec.StartRemoveBuffs {
				targetChar.Buffs.Remove(removeId)
			}
		}
```

Check what the actual method is to remove a buff by ID — it may be `RemoveBuff`, `Remove`, or similar. Read `buffs.go` to find it.

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/hooks/...`

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/Buff_ApplyBuffs.go
git commit -m "feat: add start_remove_buffs handler for cure-type buffs

Removes listed buff IDs when buff starts. Replaces minor antidote's
JS onStart logic.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Delete Chrysalis Construct

**Files:**
- Delete: `_datafiles/world/dogmud/spells/chrysalis-construct.yaml`
- Delete: `_datafiles/world/dogmud/spells/chrysalis-construct.js`
- Potentially delete mob 110 and item 40010 if nothing else references them

- [ ] **Step 1: Check for references to mob 110 and item 40010**

Search the entire codebase for references to mob ID 110 and item ID 40010 outside of chrysalis-construct files. If they're only referenced by the spell, delete them too.

```bash
grep -r "110" _datafiles/world/dogmud/mobs/ --include="*.yaml" -l
grep -r "40010" _datafiles/ --include="*.yaml" -l
```

- [ ] **Step 2: Delete the spell files**

```bash
git rm _datafiles/world/dogmud/spells/chrysalis-construct.yaml
git rm _datafiles/world/dogmud/spells/chrysalis-construct.js
```

Delete mob and item files too if they're orphaned.

- [ ] **Step 3: Commit**

```bash
git commit -m "chore: delete chrysalis-construct spell (redundant with raise-golem)

No players had discovered this spell on prod. Removes spell YAML/JS
and orphaned mob/item definitions if applicable.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 11: Migrate 13 Companion Spell YAMLs + Delete JS

**Files:**
- Modify: 13 spell YAML files (add summon fields)
- Delete: 13 spell JS files

- [ ] **Step 1: Add summon fields to each spell YAML**

For each spell, add the summon configuration fields. The exact values:

| Spell | summon_mob_id | summon_base_pool | summon_scaling_divisor | summon_component_id | summon_requires_corpse | summon_min_corpse_pool |
|-------|--------------|-----------------|----------------------|--------------------|-----------------------|----------------------|
| raise-skeleton | 300 | 60 | 500 | 0 | true | 30 |
| raise-zombie | 301 | 80 | 500 | 0 | true | 60 |
| raise-wraith | 302 | 70 | 500 | 0 | true | 120 |
| raise-spectre | 303 | 90 | 500 | 0 | true | 200 |
| raise-vampire | 304 | 100 | 500 | 0 | true | 300 |
| raise-golem | 305 | 120 | 500 | 0 | true | 500 |
| conjure-water | 310 | 80 | 500 | 0 | false | 0 |
| conjure-earth | 311 | 90 | 500 | 0 | false | 0 |
| conjure-air | 312 | 70 | 500 | 0 | false | 0 |
| conjure-fire | 313 | 85 | 500 | 0 | false | 0 |
| conjure-magma | 314 | 130 | 500 | 0 | false | 0 |
| summon-hive-swarm | 111 | 18 | 200 | 40011 | false | 0 |
| summon-steppe-spirit | 243 | 120 | 200 | 40031 | false | 0 |

Read each YAML file, add the fields at the end (before text fields).

- [ ] **Step 2: Delete all 13 JS files**

```bash
cd _datafiles/world/dogmud/spells
git rm raise-skeleton.js raise-zombie.js raise-wraith.js raise-spectre.js \
      raise-vampire.js raise-golem.js \
      conjure-water.js conjure-earth.js conjure-air.js conjure-fire.js \
      conjure-magma.js summon-hive-swarm.js summon-steppe-spirit.js
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: migrate 13 companion spells to YAML summon fields

All companion spell JS files deleted. Summon parameters now in YAML,
resolved by Go ResolveCompanionSummon function.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 12: Migrate Buff YAMLs + Delete JS

**Files:**
- Modify: ~10 buff YAML files (add tick/cure fields)
- Delete: ~10 buff JS files (or strip onTrigger)

- [ ] **Step 1: Add tick fields to DOGMud buffs**

| Buff | tick_pool | tick_percent | tick_variance | tick_min | start_remove_buffs |
|------|-----------|-------------|---------------|---------|-------------------|
| 32-vital_surge | health | 0.05 | 0 | 1 | |
| 33-chrysalis_regeneration | health | 0.08 | 0 | 1 | |
| 39-venom | health | -0.08 | 0.04 | 3 | |
| 40-spore_toxin | health | -0.05 | 0.03 | 2 | |
| 78-toxic_cloud | health | -0.06 | 0.04 | 2 | |
| 47-minor_antidote | health | 0.05 | 0 | 1 | [39, 40] |

Read each YAML, add the fields.

- [ ] **Step 2: Add tick fields to default-world buffs**

| Buff | tick_pool | tick_percent | tick_min |
|------|-----------|-------------|---------|
| 5-minor_potion_healing | health | 0.04 | 1 |
| 6-stamina_draught | stamina | 0.08 | 1 |
| 7-conviction_draught | conviction | 0.08 | 1 |
| 50-greater_healing | health | 0.12 | 1 |

These are in `_datafiles/world/default/buffs/` (or `dogmud/buffs/` — check which).

- [ ] **Step 3: Delete JS files**

For buffs where the JS file now has NO remaining functions (onTrigger was the last one), delete the entire file. For buffs where onStart/onEnd were already stripped in Phase 1 and onTrigger is now replaced, the file should be empty — delete it.

For buff 47 (minor_antidote): if the JS only had onStart logic (remove poison + heal), and that's now handled by `start_remove_buffs` + `tick_pool`, delete the JS file entirely.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/buffs/*.yaml _datafiles/world/default/buffs/*.yaml
git rm [all deleted JS files]
git commit -m "feat: migrate ~10 healing/DoT buffs to YAML tick config

Tick calculations now config-driven. Stat scaling applied for spell
buffs, flat for potions. Minor antidote uses start_remove_buffs.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 13: Update Schema Docs

**Files:**
- Modify: `docs/schemas/spell.md`
- Modify: `docs/schemas/buff.md`

- [ ] **Step 1: Add summon fields to spell schema**

In `docs/schemas/spell.md`, add the 6 summon fields to the field reference table with descriptions. Add an example of a summon spell YAML showing all fields.

- [ ] **Step 2: Add tick/cure fields to buff schema**

In `docs/schemas/buff.md`, add the 5 new fields to the field reference table. Add examples of a healing buff and a DoT buff with tick config. Document `start_remove_buffs` with the antidote example.

- [ ] **Step 3: Commit**

```bash
git add docs/schemas/spell.md docs/schemas/buff.md
git commit -m "docs: add summon and tick config fields to spell/buff schemas

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```
