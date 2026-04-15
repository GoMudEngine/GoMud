# Spell & Rhetoric Avoidance Layers — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add defensive avoidance checks for spells (Spell Deflection) and rhetoric attacks (Stoic Resolve) that halve damage on success, fully negate on crit, and grant defender skill progression.

**Architecture:** Two new helper functions in `internal/hooks/combat_shared_helpers.go` — `trySpellDeflection` and `tryStoicResolve` — called from the existing spell/taunt damage paths. Config knobs in `BalanceConfig`. No new files needed; this integrates into the existing damage pipeline.

**Tech Stack:** Go, `dice.OpposedRollStat`, existing progression infrastructure.

---

### Task 1: Add Config Knobs

**Files:**
- Modify: `internal/configs/config.balance.go:90-97` (add fields + defaults)
- Modify: `_datafiles/config.yaml` (add entries)

- [ ] **Step 1: Add fields to BalanceConfig struct**

In `internal/configs/config.balance.go`, add after the `ConvictionMitigationCap` field (line 97):

```go
SpellAvoidanceDamageMultiplier    ConfigFloat `yaml:"SpellAvoidanceDamageMultiplier"`    // Damage multiplier on successful spell deflection (default 0.50)
RhetoricAvoidanceDamageMultiplier ConfigFloat `yaml:"RhetoricAvoidanceDamageMultiplier"` // Damage multiplier on successful stoic resolve (default 0.50)
```

- [ ] **Step 2: Add default validation**

In the `Validate()` method of `config.balance.go`, add after the `ConvictionMitigationCap` validation block (around line 469):

```go
if b.SpellAvoidanceDamageMultiplier <= 0 || b.SpellAvoidanceDamageMultiplier > 1.0 {
	b.SpellAvoidanceDamageMultiplier = 0.50
}
if b.RhetoricAvoidanceDamageMultiplier <= 0 || b.RhetoricAvoidanceDamageMultiplier > 1.0 {
	b.RhetoricAvoidanceDamageMultiplier = 0.50
}
```

- [ ] **Step 3: Add config entries to config.yaml**

In `_datafiles/config.yaml`, add under the Balance section near the mitigation caps:

```yaml
  SpellAvoidanceDamageMultiplier: 0.50     # Damage multiplier on successful spell deflection (half damage)
  RhetoricAvoidanceDamageMultiplier: 0.50  # Damage multiplier on successful stoic resolve (half damage)
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.balance.go _datafiles/config.yaml
git commit -m "feat: add config knobs for spell/rhetoric avoidance multipliers"
```

---

### Task 2: Implement trySpellDeflection Helper

**Files:**
- Modify: `internal/hooks/combat_shared_helpers.go` (add function)
- Test: `internal/hooks/hooks_test.go` (add test)

- [ ] **Step 1: Write the test**

Add to `internal/hooks/hooks_test.go`:

```go
func TestTrySpellDeflection_ReturnsMultiplier(t *testing.T) {
	// trySpellDeflection should return a damage multiplier between 0.0 and 1.0
	// We can't control dice rolls, so we test the function signature and
	// that it always returns a value in [0.0, 1.0].
	attacker := characters.New()
	attacker.Stats.Willpower.ValueAdj = 100
	attacker.Skills["spellcasting"] = 10

	defender := characters.New()
	defender.Stats.Perception.ValueAdj = 100
	defender.Skills["spellcasting"] = 10

	// Run many times to exercise both success and failure paths
	for i := 0; i < 100; i++ {
		mult := trySpellDeflection(attacker, defender, 0)
		assert.GreaterOrEqual(t, mult, 0.0, "multiplier must be >= 0")
		assert.LessOrEqual(t, mult, 1.0, "multiplier must be <= 1")
	}
}

func TestTrySpellDeflection_HighPerceptionAdvantage(t *testing.T) {
	// Defender with massively higher perception+spellcasting should deflect
	// more often than not
	attacker := characters.New()
	attacker.Stats.Willpower.ValueAdj = 50
	attacker.Skills["spellcasting"] = 1

	defender := characters.New()
	defender.Stats.Perception.ValueAdj = 200
	defender.Skills["spellcasting"] = 40

	deflections := 0
	trials := 500
	for i := 0; i < trials; i++ {
		mult := trySpellDeflection(attacker, defender, 0)
		if mult < 1.0 {
			deflections++
		}
	}
	// With a huge stat advantage, defender should deflect majority of the time
	assert.Greater(t, deflections, trials/2,
		"high-perception defender should deflect more than half the time")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hooks/... -run TestTrySpellDeflection -v`
Expected: FAIL — `trySpellDeflection` undefined.

- [ ] **Step 3: Implement trySpellDeflection**

Add to `internal/hooks/combat_shared_helpers.go`, after the `calcSpellDamageForCharacter` function (after line 92):

```go
// trySpellDeflection rolls a defensive avoidance check for the target of a
// damage spell. Returns a damage multiplier: 1.0 (no deflection), the
// configured avoidance multiplier (partial deflection), or 0.0 (crit = full
// negation).
//
// Roll: Defender Perception + Spellcasting vs Attacker Willpower + Spellcasting.
//
// The defender always receives spellcasting and perception skill/stat
// progression from the attempt regardless of outcome.
//
// defenderUserId should be 0 for mobs.
func trySpellDeflection(attacker *characters.Character, defender *characters.Character, defenderUserId int) float64 {
	cfg := configs.GetBalanceConfig()
	skillWeight := float64(cfg.SkillWeight)

	// Attacker score: Willpower + weighted Spellcasting
	atkSpellcasting := float64(attacker.GetSkillLevel(skills.Spellcasting)) * skillWeight
	attackScore := float64(attacker.Stats.Willpower.ValueAdj) + atkSpellcasting

	// Defender score: Perception + weighted Spellcasting
	defSpellcasting := float64(defender.GetSkillLevel(skills.Spellcasting)) * skillWeight
	defenseScore := float64(defender.Stats.Perception.ValueAdj) + defSpellcasting

	success, _, _, defRoll := dice.OpposedRollStat(attackScore, defenseScore)

	// Defender always gets progression from being targeted by a spell
	defender.OnSkillUse(string(skills.Spellcasting), defenderUserId)
	defender.OnStatUse("perception", defenderUserId)

	if !success {
		// Defender won the roll
		if defRoll.ZScore >= 2.0 {
			// Crit deflection: full negation
			return 0.0
		}
		// Normal deflection: half damage
		return float64(cfg.SpellAvoidanceDamageMultiplier)
	}

	// Attacker won — no deflection
	return 1.0
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/hooks/... -run TestTrySpellDeflection -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/combat_shared_helpers.go internal/hooks/hooks_test.go
git commit -m "feat: implement trySpellDeflection avoidance helper"
```

---

### Task 3: Implement tryStoicResolve Helper

**Files:**
- Modify: `internal/hooks/combat_shared_helpers.go` (add function)
- Test: `internal/hooks/hooks_test.go` (add test)

- [ ] **Step 1: Write the test**

Add to `internal/hooks/hooks_test.go`:

```go
func TestTryStoicResolve_ReturnsMultiplier(t *testing.T) {
	attacker := characters.New()
	attacker.Stats.Charisma.ValueAdj = 100
	attacker.Skills["rhetoric"] = 10

	defender := characters.New()
	defender.Stats.Willpower.ValueAdj = 100
	defender.Skills["rhetoric"] = 10

	for i := 0; i < 100; i++ {
		mult := tryStoicResolve(attacker, defender, 0)
		assert.GreaterOrEqual(t, mult, 0.0, "multiplier must be >= 0")
		assert.LessOrEqual(t, mult, 1.0, "multiplier must be <= 1")
	}
}

func TestTryStoicResolve_HighWillpowerAdvantage(t *testing.T) {
	attacker := characters.New()
	attacker.Stats.Charisma.ValueAdj = 50
	attacker.Skills["rhetoric"] = 1

	defender := characters.New()
	defender.Stats.Willpower.ValueAdj = 200
	defender.Skills["rhetoric"] = 40

	resolves := 0
	trials := 500
	for i := 0; i < trials; i++ {
		mult := tryStoicResolve(attacker, defender, 0)
		if mult < 1.0 {
			resolves++
		}
	}
	assert.Greater(t, resolves, trials/2,
		"high-willpower defender should resolve more than half the time")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hooks/... -run TestTryStoicResolve -v`
Expected: FAIL — `tryStoicResolve` undefined.

- [ ] **Step 3: Implement tryStoicResolve**

Add to `internal/hooks/combat_shared_helpers.go`, after `trySpellDeflection`:

```go
// tryStoicResolve rolls a defensive avoidance check for the target of a
// hostile rhetoric attack (taunt, demoralize, etc.). Returns a damage
// multiplier: 1.0 (no resolve), the configured avoidance multiplier (partial),
// or 0.0 (crit = full negation).
//
// Roll: Defender Willpower + Rhetoric vs Attacker Charisma + Rhetoric.
//
// The defender always receives rhetoric and willpower skill/stat progression
// from the attempt regardless of outcome.
//
// defenderUserId should be 0 for mobs.
func tryStoicResolve(attacker *characters.Character, defender *characters.Character, defenderUserId int) float64 {
	cfg := configs.GetBalanceConfig()
	skillWeight := float64(cfg.SkillWeight)

	// Attacker score: Charisma + weighted Rhetoric
	atkRhetoric := float64(attacker.GetSkillLevel(skills.Rhetoric)) * skillWeight
	attackScore := float64(attacker.Stats.Charisma.ValueAdj) + atkRhetoric

	// Defender score: Willpower + weighted Rhetoric
	defRhetoric := float64(defender.GetSkillLevel(skills.Rhetoric)) * skillWeight
	defenseScore := float64(defender.Stats.Willpower.ValueAdj) + defRhetoric

	success, _, _, defRoll := dice.OpposedRollStat(attackScore, defenseScore)

	// Defender always gets progression from being targeted
	defender.OnSkillUse(string(skills.Rhetoric), defenderUserId)
	defender.OnStatUse("willpower", defenderUserId)

	if !success {
		if defRoll.ZScore >= 2.0 {
			return 0.0
		}
		return float64(cfg.RhetoricAvoidanceDamageMultiplier)
	}

	return 1.0
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/hooks/... -run TestTryStoicResolve -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/combat_shared_helpers.go internal/hooks/hooks_test.go
git commit -m "feat: implement tryStoicResolve avoidance helper"
```

---

### Task 4: Integrate Spell Deflection into Player-Cast-on-Mob Path

**Files:**
- Modify: `internal/hooks/spell_resolution.go:187-366` (`applyMobEffect` function)

The avoidance check fires for damage-dealing effect types (`"damage"` and `"knockdown"`) but NOT for `"dot"`, `"buff"`, `"tame"`, or other non-damage effects.

- [ ] **Step 1: Add spell deflection to the "damage" case in applyMobEffect**

In `internal/hooks/spell_resolution.go`, in the `applyMobEffect` function, find the `case "damage":` block (around line 205). After the `calcSpellDamageForCharacter` call and before `mob.Character.Health -= dmg`, insert the avoidance check.

Replace the existing `case "damage":` block:

```go
	case "damage":
		var casterChar *characters.Character
		if user != nil {
			casterChar = user.Character
		}
		dmg := calcSpellDamageForCharacter(spellData, casterChar, &mob.Character, magnitude, isCrit)

		// Spell Deflection: mob defender attempts to partially deflect
		if !isCrit && casterChar != nil {
			deflectMult := trySpellDeflection(casterChar, &mob.Character, 0)
			if deflectMult < 1.0 {
				dmg = int(math.Round(float64(dmg) * deflectMult))
				if dmg < 1 && deflectMult > 0 {
					dmg = 1
				}
			}
		}

		dmgDealt = dmg
		mob.Character.Health -= dmg
```

Note: When `isCrit` is true, we skip the deflection check — spell crits already bypass mitigation and should also bypass avoidance. The `deflectMult > 0` guard on the min-1 ensures a crit deflection (0.0) results in zero damage.

The aggro setup and messaging code below the health subtraction remains unchanged.

- [ ] **Step 2: Add spell deflection to the "knockdown" case in applyMobEffect**

Same pattern for the `case "knockdown":` block (around line 263). After the `calcSpellDamageForCharacter` call and before `mob.Character.Health -= dmg`:

```go
	case "knockdown":
		var casterChar2 *characters.Character
		if user != nil {
			casterChar2 = user.Character
		}
		dmg := calcSpellDamageForCharacter(spellData, casterChar2, &mob.Character, magnitude, isCrit)

		// Spell Deflection: mob defender attempts to partially deflect
		if !isCrit && casterChar2 != nil {
			deflectMult := trySpellDeflection(casterChar2, &mob.Character, 0)
			if deflectMult < 1.0 {
				dmg = int(math.Round(float64(dmg) * deflectMult))
				if dmg < 1 && deflectMult > 0 {
					dmg = 1
				}
			}
		}

		dmgDealt = dmg
		mob.Character.Health -= dmg
```

The rest of the knockdown block (prone, aggro, messaging) remains unchanged.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/spell_resolution.go
git commit -m "feat: integrate spell deflection into player-cast-on-mob path"
```

---

### Task 5: Integrate Spell Deflection into Player-Cast-on-Player Path

**Files:**
- Modify: `internal/hooks/spell_resolution.go:417-533` (`applyPlayerEffect` function)

- [ ] **Step 1: Add spell deflection to the "damage" case in applyPlayerEffect**

In `applyPlayerEffect`, find the `case "damage":` block (around line 426). After `calcSpellDamageForCharacter` and before `target.Character.Health -= dmg`:

```go
	case "damage":
		dmg := calcSpellDamageForCharacter(spellData, user.Character, target.Character, magnitude, isCrit)

		// Spell Deflection: player defender attempts to partially deflect
		deflected := false
		critDeflect := false
		if !isCrit {
			deflectMult := trySpellDeflection(user.Character, target.Character, target.UserId)
			if deflectMult < 1.0 {
				deflected = true
				if deflectMult == 0.0 {
					critDeflect = true
				}
				dmg = int(math.Round(float64(dmg) * deflectMult))
				if dmg < 1 && deflectMult > 0 {
					dmg = 1
				}
			}
		}

		target.Character.Health -= dmg

		if critDeflect {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green">You read <ansi fg="username">%s</ansi>'s spell perfectly and unravel it before it reaches you!</ansi>`,
				user.Character.Name))
			user.SendText(fmt.Sprintf(
				`<ansi fg="yellow"><ansi fg="username">%s</ansi> completely unravels your spell!</ansi>`,
				target.Character.Name))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi> unravels <ansi fg="username">%s</ansi>'s spell completely!`,
				target.Character.Name, user.Character.Name), user.UserId, target.UserId)
			return
		}

		dmgDesc := combat.GetDamageDescription(dmg, target.Character.HealthMax.Value)

		if deflected {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green">You partially deflect <ansi fg="username">%s</ansi>'s <ansi fg="cyan-bold">%s</ansi>! (<ansi fg="damage">%s</ansi>)</ansi>`,
				user.Character.Name, spellData.Name, dmgDesc))
			user.SendText(fmt.Sprintf(
				`<ansi fg="yellow"><ansi fg="username">%s</ansi> partially deflects your <ansi fg="cyan-bold">%s</ansi>! (<ansi fg="damage">%s</ansi>)</ansi>`,
				target.Character.Name, spellData.Name, dmgDesc))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi> partially deflects <ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi>!`,
				target.Character.Name, user.Character.Name, spellData.Name), user.UserId, target.UserId)
		} else {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> strikes <ansi fg="username">%s</ansi>! (<ansi fg="damage">%s</ansi>)%s</ansi>`,
				spellData.Name, target.Character.Name, dmgDesc, critTag))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> strikes <ansi fg="username">%s</ansi>!`,
				user.Character.Name, spellData.Name, target.Character.Name), user.UserId, target.UserId)
			target.SendText(fmt.Sprintf(
				`<ansi fg="red"><ansi fg="username">%s</ansi>'s <ansi fg="cyan-bold">%s</ansi> strikes you! (<ansi fg="damage">%s</ansi>)</ansi>`,
				user.Character.Name, spellData.Name, dmgDesc))
		}
```

Note: On a crit deflection (zero damage), we return early after the crit deflection messages — no need to show damage descriptions for zero damage.

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/spell_resolution.go
git commit -m "feat: integrate spell deflection into player-vs-player spell path"
```

---

### Task 6: Integrate Spell Deflection into Mob-Cast-on-Player Path

**Files:**
- Modify: `internal/hooks/spell_resolution.go:716+` (`resolveMobSpellAgainstPlayer` function)

- [ ] **Step 1: Add spell deflection to the "damage" case**

In `resolveMobSpellAgainstPlayer`, find the `case "damage":` block (around line 746). After `calcSpellDamageForCharacter` and before `target.Character.Health -= dmg`:

```go
	case "damage":
		dmg := calcSpellDamageForCharacter(spellData, &caster.Character, target.Character, magnitude, isCrit)

		// Spell Deflection: player defender attempts to partially deflect
		deflected := false
		critDeflect := false
		if !isCrit {
			deflectMult := trySpellDeflection(&caster.Character, target.Character, target.UserId)
			if deflectMult < 1.0 {
				deflected = true
				if deflectMult == 0.0 {
					critDeflect = true
				}
				dmg = int(math.Round(float64(dmg) * deflectMult))
				if dmg < 1 && deflectMult > 0 {
					dmg = 1
				}
			}
		}

		mobSpellDmg = dmg
		target.Character.Health -= dmg

		if critDeflect {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green">You read <ansi fg="mobname">%s</ansi>'s spell perfectly and unravel it before it reaches you!</ansi>`,
				caster.Character.Name))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi> unravels <ansi fg="mobname">%s</ansi>'s spell completely!`,
				target.Character.Name, caster.Character.Name), target.UserId)
		} else if deflected {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green">You partially deflect <ansi fg="mobname">%s</ansi>'s <ansi fg="cyan-bold">%s</ansi>! (<ansi fg="damage">%s</ansi>)</ansi>`,
				caster.Character.Name, spellData.Name,
				combat.GetDamageDescription(dmg, target.Character.HealthMax.Value)))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi> partially deflects <ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi>!`,
				target.Character.Name, caster.Character.Name, spellData.Name), target.UserId)
		} else {
			target.SendText(fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> strikes you! (<ansi fg="damage">%s</ansi>)%s`,
				caster.Character.Name, spellData.Name,
				combat.GetDamageDescription(dmg, target.Character.HealthMax.Value), critTag))
			room.SendText(fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> strikes <ansi fg="username">%s</ansi>!`,
				caster.Character.Name, spellData.Name, target.Character.Name), target.UserId)
		}
```

Replace the existing damage case messaging. The aggro setup that follows remains unchanged.

- [ ] **Step 2: Add spell deflection to the "knockdown" case**

Same pattern for the `case "knockdown":` block in `resolveMobSpellAgainstPlayer` (around line 779). After `calcSpellDamageForCharacter` and before `target.Character.Health -= dmg`:

```go
	case "knockdown":
		dmg := calcSpellDamageForCharacter(spellData, &caster.Character, target.Character, magnitude, isCrit)

		// Spell Deflection (damage only — knockdown still applies)
		if !isCrit {
			deflectMult := trySpellDeflection(&caster.Character, target.Character, target.UserId)
			if deflectMult < 1.0 {
				dmg = int(math.Round(float64(dmg) * deflectMult))
				if dmg < 1 && deflectMult > 0 {
					dmg = 1
				}
			}
		}

		mobSpellDmg = dmg
		target.Character.Health -= dmg
```

The knockdown position change and messaging below remain unchanged.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/spell_resolution.go
git commit -m "feat: integrate spell deflection into mob-cast-on-player path"
```

---

### Task 7: Integrate Spell Deflection into Mob-Cast-on-Mob Path

**Files:**
- Modify: `internal/hooks/spell_resolution.go:697-714` (`resolveMobSpellAgainstMob` function)

Mob-on-mob uses `applyMobEffect` which was already patched in Task 4. No additional work needed — `applyMobEffect` handles both player-cast-on-mob and mob-cast-on-mob via the `user` nil check.

- [ ] **Step 1: Verify mob-on-mob path**

Confirm that `resolveMobSpellAgainstMob` calls `applyMobEffect(nil, target, ...)` (line 713). The `nil` user means `casterChar` will be nil in the `applyMobEffect` damage case, and the deflection check is gated by `casterChar != nil`.

We need to fix this — mob-on-mob should also allow deflection. The caster character is available as `&caster.Character`. Update the `applyMobEffect` function signature to accept an explicit caster character pointer.

Actually, looking at the code more carefully: `applyMobEffect` already computes `casterChar` from `user`, but `resolveMobSpellAgainstMob` passes `nil` for user. Rather than refactoring the signature, we can pass the caster character directly into the deflection check by adding a `casterChar` parameter to `applyMobEffect`.

Modify `applyMobEffect` signature from:

```go
func applyMobEffect(user *users.UserRecord, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, magnitude int, isCrit bool) int {
```

To:

```go
func applyMobEffect(user *users.UserRecord, casterChar *characters.Character, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, magnitude int, isCrit bool) int {
```

Then remove the internal `casterChar` / `casterChar2` derivation from `user`, and use the parameter directly.

Update all callers:
- `resolveAgainstMob` (line 182): change `applyMobEffect(user, mob, room, ...)` to `applyMobEffect(user, user.Character, mob, room, ...)`
- `resolveMobSpellAgainstMob` (line 713): change `applyMobEffect(nil, target, room, ...)` to `applyMobEffect(nil, &caster.Character, target, room, ...)`

And in the damage/knockdown cases, remove the `var casterChar *characters.Character` / `if user != nil` blocks — just use the parameter.

- [ ] **Step 2: Update applyMobEffect and callers**

In `internal/hooks/spell_resolution.go`:

Update the function signature and body as described above. The deflection check in the damage case becomes:

```go
	case "damage":
		dmg := calcSpellDamageForCharacter(spellData, casterChar, &mob.Character, magnitude, isCrit)

		// Spell Deflection: mob defender attempts to partially deflect
		if !isCrit && casterChar != nil {
			deflectMult := trySpellDeflection(casterChar, &mob.Character, 0)
			if deflectMult < 1.0 {
				dmg = int(math.Round(float64(dmg) * deflectMult))
				if dmg < 1 && deflectMult > 0 {
					dmg = 1
				}
			}
		}

		dmgDealt = dmg
		mob.Character.Health -= dmg
```

And similarly for knockdown.

- [ ] **Step 3: Verify build and tests**

Run: `go build ./... && go test ./internal/hooks/... -v -count=1 2>&1 | tail -30`
Expected: Clean build, all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/spell_resolution.go
git commit -m "feat: refactor applyMobEffect to accept caster char, enable mob-on-mob deflection"
```

---

### Task 8: Integrate Stoic Resolve into Taunt Resolution

**Files:**
- Modify: `internal/actions/combat_taunt.go:136-201` (inside `ExecuteTaunt`)

- [ ] **Step 1: Add TauntResult fields for stoic resolve**

In `internal/actions/combat_taunt.go`, add to the `TauntResult` struct after the `Damage` field:

```go
// Deflected reports whether the target partially deflected the conviction damage.
Deflected bool

// CritDeflected reports whether the target fully negated the conviction damage.
CritDeflected bool
```

- [ ] **Step 2: Add stoic resolve check in the hit path**

In `ExecuteTaunt`, inside the `if attackSuccess {` block (line 136), after the damage calculation and before `target.Char.Conviction -= dmg` (line 168), add:

```go
		// Stoic Resolve: defender attempts to partially resist
		deflected := false
		critDeflect := false
		if !isCrit {
			defenderUserId := 0
			if target.UserId > 0 {
				defenderUserId = target.UserId
			}
			resolveMult := tryStoicResolve(char, target.Char, defenderUserId)
			if resolveMult < 1.0 {
				deflected = true
				if resolveMult == 0.0 {
					critDeflect = true
				}
				dmg = int(math.Round(float64(dmg) * resolveMult))
				if dmg < 1 && resolveMult > 0 {
					dmg = 1
				}
			}
		}
```

Note: `tryStoicResolve` is in package `hooks`, but `ExecuteTaunt` is in package `actions`. We need to either move `tryStoicResolve` to a shared location or expose it. The cleanest approach: add `tryStoicResolve` as a function in `internal/combat/avoidance.go` instead of `hooks`, so both `hooks` and `actions` can import it. This requires moving both helper functions.

**Actually**, looking at the code more carefully, the taunt resolution is in `internal/actions/combat_taunt.go` while the spell deflection is in `internal/hooks/combat_shared_helpers.go`. To avoid a circular import, we should put both avoidance helpers in `internal/combat/avoidance.go`.

**This changes the approach for Tasks 2-3.** The helpers should be in `internal/combat/` not `internal/hooks/`. Let me restructure:

Create `internal/combat/avoidance.go` with both `TrySpellDeflection` and `TryStoicResolve` (exported, since they're called from both `hooks` and `actions` packages).

- [ ] **Step 3: Create internal/combat/avoidance.go**

Create a new file `internal/combat/avoidance.go`:

```go
package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// TrySpellDeflection rolls a defensive avoidance check for the target of a
// damage spell. Returns a damage multiplier: 1.0 (no deflection), the
// configured avoidance multiplier (partial deflection), or 0.0 (crit = full
// negation).
//
// Roll: Defender Perception + Spellcasting vs Attacker Willpower + Spellcasting.
//
// The defender always receives spellcasting and perception progression from
// the attempt regardless of outcome.
func TrySpellDeflection(attacker *characters.Character, defender *characters.Character, defenderUserId int) float64 {
	cfg := configs.GetBalanceConfig()
	skillWeight := float64(cfg.SkillWeight)

	atkSpellcasting := float64(attacker.GetSkillLevel(skills.Spellcasting)) * skillWeight
	attackScore := float64(attacker.Stats.Willpower.ValueAdj) + atkSpellcasting

	defSpellcasting := float64(defender.GetSkillLevel(skills.Spellcasting)) * skillWeight
	defenseScore := float64(defender.Stats.Perception.ValueAdj) + defSpellcasting

	success, _, _, defRoll := dice.OpposedRollStat(attackScore, defenseScore)

	defender.OnSkillUse(string(skills.Spellcasting), defenderUserId)
	defender.OnStatUse("perception", defenderUserId)

	if !success {
		if defRoll.ZScore >= 2.0 {
			return 0.0
		}
		return float64(cfg.SpellAvoidanceDamageMultiplier)
	}

	return 1.0
}

// TryStoicResolve rolls a defensive avoidance check for the target of a
// hostile rhetoric attack. Returns a damage multiplier: 1.0 (no resolve),
// the configured avoidance multiplier (partial), or 0.0 (crit = full
// negation).
//
// Roll: Defender Willpower + Rhetoric vs Attacker Charisma + Rhetoric.
//
// The defender always receives rhetoric and willpower progression from the
// attempt regardless of outcome.
func TryStoicResolve(attacker *characters.Character, defender *characters.Character, defenderUserId int) float64 {
	cfg := configs.GetBalanceConfig()
	skillWeight := float64(cfg.SkillWeight)

	atkRhetoric := float64(attacker.GetSkillLevel(skills.Rhetoric)) * skillWeight
	attackScore := float64(attacker.Stats.Charisma.ValueAdj) + atkRhetoric

	defRhetoric := float64(defender.GetSkillLevel(skills.Rhetoric)) * skillWeight
	defenseScore := float64(defender.Stats.Willpower.ValueAdj) + defRhetoric

	success, _, _, defRoll := dice.OpposedRollStat(attackScore, defenseScore)

	defender.OnSkillUse(string(skills.Rhetoric), defenderUserId)
	defender.OnStatUse("willpower", defenderUserId)

	if !success {
		if defRoll.ZScore >= 2.0 {
			return 0.0
		}
		return float64(cfg.RhetoricAvoidanceDamageMultiplier)
	}

	return 1.0
}
```

- [ ] **Step 4: Update spell deflection calls in hooks**

In `internal/hooks/spell_resolution.go` and `internal/hooks/combat_shared_helpers.go`, replace all calls to `trySpellDeflection(...)` with `combat.TrySpellDeflection(...)`. Remove the local `trySpellDeflection` function from `combat_shared_helpers.go` if it was added in Task 2.

- [ ] **Step 5: Add stoic resolve to ExecuteTaunt**

In `internal/actions/combat_taunt.go`, in the `if attackSuccess {` block, after the damage calculation (after line 162 — `dmg = int(math.Round(dmgRoll.Value))`):

```go
		// Stoic Resolve: defender attempts to partially resist
		deflected := false
		critDeflect := false
		if !isCrit {
			defenderUserId := 0
			if target.UserId > 0 {
				defenderUserId = target.UserId
			}
			resolveMult := combat.TryStoicResolve(char, target.Char, defenderUserId)
			if resolveMult < 1.0 {
				deflected = true
				if resolveMult == 0.0 {
					critDeflect = true
				}
				dmg = int(math.Round(float64(dmg) * resolveMult))
				if dmg < 1 && resolveMult > 0 {
					dmg = 1
				}
			}
		}
```

And set the result fields:

```go
		return TauntResult{
			Target:        target,
			Executed:      true,
			Hit:           true,
			Crit:          isCrit,
			Damage:        dmg,
			DmgDesc:       dmgDesc,
			Deflected:     deflected,
			CritDeflected: critDeflect,
		}
```

- [ ] **Step 6: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 7: Commit**

```bash
git add internal/combat/avoidance.go internal/actions/combat_taunt.go internal/hooks/spell_resolution.go internal/hooks/combat_shared_helpers.go
git commit -m "feat: move avoidance helpers to combat package, integrate stoic resolve into taunt"
```

---

### Task 9: Add Stoic Resolve Messages to Taunt Callers

**Files:**
- Modify: `internal/usercommands/taunt.go` (player taunt messaging)
- Modify: `internal/mobcommands/taunt.go` or equivalent (mob taunt messaging)

The `ExecuteTaunt` result now carries `Deflected` and `CritDeflected` flags. Callers need to show appropriate messages.

- [ ] **Step 1: Find the player taunt message handler**

Read `internal/usercommands/taunt.go` to find where `TauntResult.Hit` is checked and messages are sent. Add deflection/resolve messaging there.

- [ ] **Step 2: Add stoic resolve messages to player taunt**

After the existing hit message block in the player taunt handler, check the new flags:

```go
if result.CritDeflected {
	// Target fully resisted — show resolve messages
	if result.Target.UserId > 0 {
		if target := users.GetByUserId(result.Target.UserId); target != nil {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green">The words wash over you harmlessly — you are unmoved.</ansi>`))
		}
	}
	user.SendText(fmt.Sprintf(
		`<ansi fg="yellow">Your words have no effect — your target is completely unmoved!</ansi>`))
} else if result.Deflected {
	if result.Target.UserId > 0 {
		if target := users.GetByUserId(result.Target.UserId); target != nil {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green">You steel yourself against the barrage of words.</ansi>`))
		}
	}
	user.SendText(fmt.Sprintf(
		`<ansi fg="yellow">Your words fail to fully penetrate your target's resolve!</ansi>`))
}
```

The exact integration depends on the existing message structure — read the file first and integrate appropriately.

- [ ] **Step 3: Add stoic resolve messages to mob taunt**

Similar treatment for the mob taunt handler (`internal/mobcommands/taunt.go` or `howl.go`). When a mob taunts a player and the player deflects, show the resolve messages to the player.

- [ ] **Step 4: Verify build and run tests**

Run: `go build ./... && go test ./internal/... -count=1 2>&1 | tail -20`
Expected: Clean build, all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/taunt.go internal/mobcommands/taunt.go
git commit -m "feat: add stoic resolve messaging to taunt callers"
```

---

### Task 10: Write Tests for Avoidance Helpers

**Files:**
- Create: `internal/combat/avoidance_test.go`

- [ ] **Step 1: Write tests**

Create `internal/combat/avoidance_test.go`:

```go
package combat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/stretchr/testify/assert"
)

func TestTrySpellDeflection_ReturnsValidMultiplier(t *testing.T) {
	attacker := characters.New()
	attacker.Stats.Willpower.ValueAdj = 100
	attacker.Skills["spellcasting"] = 10

	defender := characters.New()
	defender.Stats.Perception.ValueAdj = 100
	defender.Skills["spellcasting"] = 10

	for i := 0; i < 200; i++ {
		mult := combat.TrySpellDeflection(attacker, defender, 0)
		assert.GreaterOrEqual(t, mult, 0.0)
		assert.LessOrEqual(t, mult, 1.0)
	}
}

func TestTrySpellDeflection_HighDefenderAdvantage(t *testing.T) {
	attacker := characters.New()
	attacker.Stats.Willpower.ValueAdj = 50
	attacker.Skills["spellcasting"] = 1

	defender := characters.New()
	defender.Stats.Perception.ValueAdj = 200
	defender.Skills["spellcasting"] = 40

	deflections := 0
	trials := 500
	for i := 0; i < trials; i++ {
		if combat.TrySpellDeflection(attacker, defender, 0) < 1.0 {
			deflections++
		}
	}
	assert.Greater(t, deflections, trials/2)
}

func TestTryStoicResolve_ReturnsValidMultiplier(t *testing.T) {
	attacker := characters.New()
	attacker.Stats.Charisma.ValueAdj = 100
	attacker.Skills["rhetoric"] = 10

	defender := characters.New()
	defender.Stats.Willpower.ValueAdj = 100
	defender.Skills["rhetoric"] = 10

	for i := 0; i < 200; i++ {
		mult := combat.TryStoicResolve(attacker, defender, 0)
		assert.GreaterOrEqual(t, mult, 0.0)
		assert.LessOrEqual(t, mult, 1.0)
	}
}

func TestTryStoicResolve_HighDefenderAdvantage(t *testing.T) {
	attacker := characters.New()
	attacker.Stats.Charisma.ValueAdj = 50
	attacker.Skills["rhetoric"] = 1

	defender := characters.New()
	defender.Stats.Willpower.ValueAdj = 200
	defender.Skills["rhetoric"] = 40

	resolves := 0
	trials := 500
	for i := 0; i < trials; i++ {
		if combat.TryStoicResolve(attacker, defender, 0) < 1.0 {
			resolves++
		}
	}
	assert.Greater(t, resolves, trials/2)
}

func TestTrySpellDeflection_LowDefenderRarelyDeflects(t *testing.T) {
	attacker := characters.New()
	attacker.Stats.Willpower.ValueAdj = 200
	attacker.Skills["spellcasting"] = 40

	defender := characters.New()
	defender.Stats.Perception.ValueAdj = 50
	defender.Skills["spellcasting"] = 1

	deflections := 0
	trials := 500
	for i := 0; i < trials; i++ {
		if combat.TrySpellDeflection(attacker, defender, 0) < 1.0 {
			deflections++
		}
	}
	assert.Less(t, deflections, trials/2)
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/combat/... -run TestTry -v`
Expected: All PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/combat/avoidance_test.go
git commit -m "test: add avoidance helper tests for spell deflection and stoic resolve"
```

---

### Task 11: Final Integration Test and Cleanup

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1 2>&1 | tail -30`
Expected: All tests pass.

- [ ] **Step 2: Clean up any leftover local helpers**

If `trySpellDeflection` or `tryStoicResolve` (lowercase, unexported) still exist in `internal/hooks/combat_shared_helpers.go` from earlier tasks, remove them — the exported versions in `internal/combat/avoidance.go` are the canonical ones.

- [ ] **Step 3: Verify no duplicate progression triggers**

Check that `OnSkillUse` for spellcasting/rhetoric is only called once per avoidance attempt — in the avoidance helper, not also in the caller. The existing `actor.OnSkillUse(string(skills.Rhetoric))` in `ExecuteTaunt` (lines 119, 185, 205) is for the **attacker's** progression and should remain. The **defender's** progression is handled inside `TryStoicResolve`. These are different characters, so both are correct.

- [ ] **Step 4: Commit final cleanup**

```bash
git add -A
git commit -m "chore: final cleanup for spell/rhetoric avoidance integration"
```
