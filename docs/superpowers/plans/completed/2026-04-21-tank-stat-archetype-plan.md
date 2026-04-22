# Tank Stat Archetype + Rhetoric Bump — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make tank companion mobs (golem/earth/magma) reliably land their signature taunt by giving them a stat distribution weighted toward Charisma + Vitality, plus a modest Rhetoric skill bump.

**Architecture:** One new `case "tank"` in `internal/mobs/mobs.go`'s archetype switch, three mob YAML edits (archetype flip + `skills:` block), one unit test for the new distribution. Small, focused refactor — no engine-level API changes.

**Tech Stack:** Go 1.21+, existing `util.Rand` + mob stat-roll logic, YAML content.

**Related spec:** `docs/superpowers/specs/2026-04-21-tank-stat-archetype-design.md`
**Branch:** `feature/tank-stat-archetype` (created; spec committed as `ad2f562b`).

---

## File Structure

**Modified files:**
- `internal/mobs/mobs.go` — add `case "tank"` to the archetype switch.
- `internal/mobs/mobs_test.go` — append distribution test.
- `_datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml` — flip archetype + add skills.
- `_datafiles/world/dogmud/mobs/summons/311-earth_elemental.yaml` — same.
- `_datafiles/world/dogmud/mobs/summons/314-magma_elemental.yaml` — same.

---

## Task 1: Add `tank` archetype case + unit test

**Files:**
- Modify: `internal/mobs/mobs.go` (archetype switch around line 402)
- Modify: `internal/mobs/mobs_test.go` (append test)

### Step 1: Write the failing test

Append to `internal/mobs/mobs_test.go` (near the end, or alongside existing archetype tests if present):

```go
// ─── Tank archetype distribution ────────────────────────────────────────────

// TestNewMobById_TankArchetypeDistributesStats verifies the new tank
// archetype allocates ~25% Cha and ~20% Vit out of the stat pool,
// with ~15% each Str/Dex/Wil and ~10% Per. Large pool (1000) shrinks
// random variance; assertions use generous slack bands.
func TestNewMobById_TankArchetypeDistributesStats(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Register a minimal tank mob template (reuse mobId 1's
	// character scaffold, override Archetype + StatPool).
	mobsMu.Lock()
	mobs[1].Archetype = "tank"
	mobs[1].StatPool = 1000
	mobsMu.Unlock()

	mob := NewMobById(1, 100)
	if mob == nil {
		t.Fatal("NewMobById returned nil")
	}

	total := mob.Character.Stats.Strength.Training +
		mob.Character.Stats.Dexterity.Training +
		mob.Character.Stats.Vitality.Training +
		mob.Character.Stats.Perception.Training +
		mob.Character.Stats.Willpower.Training +
		mob.Character.Stats.Charisma.Training

	assert.Equal(t, 1000, total, "tank stat pool must sum to 1000 with no leakage")

	// Slack bands: expected ±25% of target share (1000-unit pool).
	assert.GreaterOrEqual(t, mob.Character.Stats.Charisma.Training, 200,
		"Cha training should be ≥200 (~25%% target)")
	assert.GreaterOrEqual(t, mob.Character.Stats.Vitality.Training, 150,
		"Vit training should be ≥150 (~20%% target)")
	assert.LessOrEqual(t, mob.Character.Stats.Perception.Training, 150,
		"Per training should be ≤150 (~10%% target)")
}
```

**Helper note:** `seedRegistry()` exists in `mobs_test.go` (used by other tests). The test mutates `mobs[1].Archetype` and `mobs[1].StatPool` directly because `NewMobById` reads those fields from the registry. The surrounding cleanup restores the original.

### Step 2: Run the test to verify it fails

Run: `go test -run 'TestNewMobById_TankArchetypeDistributesStats' ./internal/mobs/`
Expected: the test compiles and runs, but the assertions fail because the `default` branch of the switch (uniform random) doesn't meet the tank-specific bands. Current output will show ~`Cha training ~166`, ~`Vit training ~166` — uniform ~16.7% per stat.

### Step 3: Add the `tank` archetype case

In `internal/mobs/mobs.go`, find the archetype switch (around line 402). Current shape:

```go
switch mob.Archetype {
case "fighting":
    // 80% physical (Str/Dex/Vit), 20% mental (Per/Wil/Cha)
    if util.Rand(100) < 80 {
        statIdx = util.Rand(3) // 0=Str, 1=Dex, 2=Vit
    } else {
        statIdx = 3 + util.Rand(3) // 3=Per, 4=Wil, 5=Cha
    }
case "casting":
    // 20% physical (Str/Dex/Vit), 80% mental (Per/Wil/Cha)
    if util.Rand(100) < 20 {
        statIdx = util.Rand(3)
    } else {
        statIdx = 3 + util.Rand(3)
    }
default:
    // Even distribution across all 6 stats
    statIdx = util.Rand(6)
}
```

Insert a new `case "tank":` between `casting` and `default`:

```go
case "casting":
    // 20% physical (Str/Dex/Vit), 80% mental (Per/Wil/Cha)
    if util.Rand(100) < 20 {
        statIdx = util.Rand(3)
    } else {
        statIdx = 3 + util.Rand(3)
    }
case "tank":
    // Tank/taunter: 25% Cha (taunt), 20% Vit (HP buffer),
    // 15% each Str/Dex/Wil, 10% Per.
    r := util.Rand(100)
    switch {
    case r < 25:
        statIdx = 5 // Charisma
    case r < 45:
        statIdx = 2 // Vitality
    case r < 60:
        statIdx = 0 // Strength
    case r < 75:
        statIdx = 1 // Dexterity
    case r < 90:
        statIdx = 4 // Willpower
    default:
        statIdx = 3 // Perception
    }
default:
    // Even distribution across all 6 stats
    statIdx = util.Rand(6)
}
```

### Step 4: Run the test to verify it passes

Run: `go test -run 'TestNewMobById_TankArchetypeDistributesStats' ./internal/mobs/`
Expected: PASS.

### Step 5: Run the full mobs package test suite

Run: `go test ./internal/mobs/`
Expected: clean. Existing `fighting` / `casting` tests (if any) untouched.

### Step 6: Full project build + test

Run: `go build ./... && go test ./...`
Expected: clean.

### Step 7: Commit

```bash
git add internal/mobs/mobs.go internal/mobs/mobs_test.go
git commit -m "$(cat <<'EOF'
feat(mobs): "tank" stat-distribution archetype

Allocates stat pool: 25% Cha, 20% Vit, 15% each Str/Dex/Wil,
10% Per. Designed for tank_taunter mobs where the signature
taunt (conviction attack) needs enough charisma to win the
opposed roll against a typical-willpower target.

Surfaced during T7 smoke of the tank_taunter behavior-tree
archetype — the three wired mobs were using "fighting" (80/20
physical/mental) and landing taunts <30% of the time.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Update the three tank mob YAMLs

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml`
- Modify: `_datafiles/world/dogmud/mobs/summons/311-earth_elemental.yaml`
- Modify: `_datafiles/world/dogmud/mobs/summons/314-magma_elemental.yaml`

Two edits per file: flip `archetype:` and add a `skills:` block with `rhetoric: 10`.

### Step 1: Update flesh golem (305)

In `_datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml`:

**(a) Flip the archetype line** at line 3. Change:
```yaml
archetype: fighting
```
to:
```yaml
archetype: tank
```

**(b) Add `skills:` block** under the `character:` block, positioned ABOVE the existing `stats:` block (matches the `304-vampire.yaml` convention). Example — insert right before `  stats:`:

```yaml
  skills:
    rhetoric: 10
  stats:
    strength:
      training: 25
    vitality:
      training: 25
    dexterity:
      training: -10
```

### Step 2: Update earth elemental (311)

Same two edits in `_datafiles/world/dogmud/mobs/summons/311-earth_elemental.yaml`:

**(a)** Flip `archetype: fighting` → `archetype: tank`.
**(b)** Insert a `skills:` block with `rhetoric: 10` immediately above the existing `stats:` block.

### Step 3: Update magma elemental (314)

Same two edits in `_datafiles/world/dogmud/mobs/summons/314-magma_elemental.yaml`:

**(a)** Flip `archetype: fighting` → `archetype: tank`.
**(b)** Insert a `skills:` block with `rhetoric: 10` immediately above the existing `stats:` block.

### Step 4: Verify

Run: `grep -n "^archetype:\|^  skills:\|    rhetoric:" _datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml _datafiles/world/dogmud/mobs/summons/311-earth_elemental.yaml _datafiles/world/dogmud/mobs/summons/314-magma_elemental.yaml`

Expected: 9 lines — 3 `archetype: tank` lines, 3 `skills:` block headers, 3 `rhetoric: 10` entries.

Run: `go build ./... && go test ./...`
Expected: clean. (YAML-only change; unit tests not affected.)

### Step 5: Commit

```bash
git add _datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml _datafiles/world/dogmud/mobs/summons/311-earth_elemental.yaml _datafiles/world/dogmud/mobs/summons/314-magma_elemental.yaml
git commit -m "$(cat <<'EOF'
feat(mobs): wire tank archetype + rhetoric 10 to golem/earth/magma

Switch archetype from "fighting" to "tank" on the three tank
companion templates. Add rhetoric: 10 to each so tank mobs land
taunts reliably (~65-75% success against typical-willpower
targets) without being un-missable.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Manual smoke test

Over to the user. Rebuild and verify.

- [ ] **Smoke 1: Stats on summon — flesh golem.** Summon flesh golem. `consider` or `status` it. Confirm Charisma is highest stat, Vitality second-highest. If not, re-check YAML.
- [ ] **Smoke 2: Stats on summon — earth elemental.** Same check. Charisma highest, Vitality second.
- [ ] **Smoke 3: Stats on summon — magma elemental.** Same check.
- [ ] **Smoke 4: Taunt success rate.** Engage a tank mob (e.g., magma elemental) against a single-target enemy. Count taunt outcomes across ~10 cooldown cycles. Landing rate should be ≥60-70%.
- [ ] **Smoke 5: Existing fighting/casting mobs unchanged.** Summon a non-tank mob that uses `fighting` (e.g., 301 zombie) or `casting` (e.g., 302 wraith). Stats should look normal — no overflow, no weird distribution.

Report pass/fail per scenario.

---

## Task 4: Finalize + merge

- [ ] **Step 1: Full build + test**

Run: `go build ./... && go test ./...`
Expected: clean.

- [ ] **Step 2: Update MEMORY.md**

In `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`:

(a) Remove the "Tank taunt viability" entry from `## Bugs to Fix`. Optionally add a status note to `project_tank_taunt_viability.md` ("STATUS: Landed 2026-04-21").

(b) Add to `## Completed (2026-04-21)`:

```markdown
- **Tank stat archetype + rhetoric bump** — new `tank` stat-distribution case in `internal/mobs/mobs.go` alongside `fighting` / `casting` — allocates 25% Cha / 20% Vit / 15% each Str/Dex/Wil / 10% Per. Three tank companion templates (305 flesh_golem, 311 earth_elemental, 314 magma_elemental) flipped from `archetype: fighting` to `archetype: tank` plus `skills: { rhetoric: 10 }` added per mob. Closes the "tank_taunter archetype fires taunt but it never lands" issue surfaced during T7 smoke of the archetype landing (Cha was ~7% of stat pool under `fighting`; now ~25%). Expected taunt success ~65-75% against typical players. 2 code commits + spec + plan. Branch: `feature/tank-stat-archetype`. Design in `docs/superpowers/specs/2026-04-21-tank-stat-archetype-design.md`, plan in `docs/superpowers/plans/2026-04-21-tank-stat-archetype-plan.md`.
```

- [ ] **Step 3: Prompt user about merge**

Per `github_guide.md`: feature branches merge into `development` with `--no-ff`. Ask user before merging; do NOT merge autonomously.

After confirmation:

```bash
git checkout development
git merge --no-ff feature/tank-stat-archetype -m "..."
git branch -d feature/tank-stat-archetype
```

Commit message summarizes the refactor + references spec.

---

## Self-Review

**Spec coverage:**
- §"Change 1 — New tank stat archetype" → Task 1 (switch case + unit test). ✓
- §"Change 2 — Wire three tank mobs to archetype: tank" → Task 2 Step (a). ✓
- §"Change 3 — Rhetoric skill bump" → Task 2 Step (b). ✓
- §"Testing / Unit tests" → Task 1 Step 1's `TestNewMobById_TankArchetypeDistributesStats`. ✓
- §"Testing / Smoke tests" → Task 3. ✓

**Placeholder scan:**
- No TBD / TODO / "similar to".
- Every code block has concrete content.

**Type consistency:**
- Stat index mapping `{0=Str, 1=Dex, 2=Vit, 3=Per, 4=Wil, 5=Cha}` consistent between Task 1's new switch case and the unit test assertions.
- `util.Rand(100)` usage matches existing archetype pattern.
- Buff IDs unchanged from this spec's scope (uses characters/mobs machinery only).

No issues.
