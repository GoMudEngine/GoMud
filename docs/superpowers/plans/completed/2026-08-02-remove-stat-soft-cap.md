# Remove the Stat Soft Cap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete the stat soft-cap compression entirely so `ValueAdj == Value` everywhere, and restore the resource-pool formulas to their intended design now that the compression which was silently shrinking them is gone.

**Architecture:** `stats.StatInfo.Recalculate()` currently compresses any value above `StatSoftCap` (150). Because `HealthMax`, `StaminaMax`, `ConvictionMax` and `ActionPointsMax` are also `StatInfo` and also call `Recalculate()`, that compression has been silently applying to the resource pools, which routinely sit in the hundreds. Both effects are unintended. This plan removes the compression from `Recalculate()` outright, then restores each pool to one primary stat at x3 plus one secondary at x1 — the intended design, which the coefficients had drifted from while being tuned against compressed output. The progression-rate soft cap (a *different* use of the same config knob) is kept and renamed.

**This is a deliberate power increase, not a neutral refactor.** See "Accepted balance consequences".

**Tech Stack:** Go 1.x, `gopkg.in/yaml.v2`, testify. No new dependencies.

---

## Background: why this is being removed

The compression came from upstream GoMud, which anchored at 100 with `sqrt(overage)*2` and gated it behind a threshold of 105. DOGMud moved the anchor to 150 and changed the exponent to 0.75 but left the threshold at 105, which put it below the anchor and made it inert — and removed the guard that had been suppressing the curve's amplifying region. That is the amplification bug documented in `docs/audits/` and parked on branch `fix/stat-softcap-amplification` (commit `cd874f0bb`). **That branch is superseded by this plan and must NOT be merged.** This plan applies to `master`.

### Measured impact (33 prod characters, `Megalomania` excluded as an admin character)

Stat compression affects almost nobody. Only eight organic stat values have ever passed 150:

| Character | Stat | Raw | Effective today | Change on removal |
|---|---|---|---|---|
| Duard | willpower | 195 | 185 | +10 |
| pruuk | willpower | 179 | 175 | +4 |
| Deios | willpower | 173 | 171 | +2 |
| Oriana | perception | 162 | 163 | −1 |
| meirok | perception | 152 | 153 | −1 |
| fyttyn | vitality | 411 | 280 | frozen to 280, see Task 5 |

Oriana and meirok currently read *higher* than raw — that is the amplification bug live in prod. Median prod stat is 105; p90 is 138.

Pool compression, by contrast, affects **every** character. A median character (all stats 105) has a true HP of 530 but plays with 322.

---

## Accepted balance consequences

Restoring the intended pool formulas raises every pool. Measured against the same prod characters:

| Character | HP now | HP after | SP now | SP after | CP now | CP after |
|---|---|---|---|---|---|---|
| median (105s) | 322 | 425 (+32%) | 322 | 425 (+32%) | 285 | 425 (+49%) |
| Duard | 360 | 511 (+42%) | 376 | 602 (+60%) | 344 | 533 (+55%) |
| meirok | 348 | 492 (+41%) | 358 | 504 (+41%) | 328 | 522 (+59%) |
| fyttyn (frozen) | 476 | 973 (+104%) | 471 | 1088 (+131%) | 379 | 668 (+76%) |

Three consequences follow, all accepted:

1. **Fights get longer.** Damage derives from stats (`CalcRawDamage(stat, ...)`), not from pools, so it does not scale with this change. Roughly, time-to-kill rises in proportion to the pool increase — on the order of a third longer for typical characters, more for high-vitality ones. This applies symmetrically: NPCs use the same formulas, so relative power is preserved and neither side gains an advantage. If pacing proves too slow after playtesting, the lever is the channel scales in the damage pipeline (`MeleeDamageScale` and friends), not a return to compression.
2. **Regen self-corrects.** All regen is percentage-of-max (`PlayerHealthRegenPct` and friends compute `floor(poolMax * pct)`), and heal spells and heal buffs work in fractions of max, so both scale with the larger pools automatically. No change needed.
3. **Two stats change meaning.** Strength no longer contributes to stamina, and conviction becomes charisma-primary rather than treating willpower and charisma equally. Existing characters built around those relationships shift accordingly — Duard, the highest-willpower character in prod, is the most affected.

---

## File Structure

| File | Responsibility | Change |
|---|---|---|
| `internal/stats/stats.go` | `StatInfo` and `Recalculate()` | Remove compression; drop `math` import |
| `internal/stats/softcap_test.go` | New | Pin `ValueAdj == Value` and pool non-compression |
| `internal/configs/config.balance.go` | Balance config struct | Delete 2 fields, rename 1, retune 5 |
| `internal/configs/config.balance.progression.go` | Validator | Delete 2 clamps, rename 1 |
| `internal/characters/progression.go` | Progression curve | Rename knob at 2 call sites |
| `_datafiles/config.yaml` | Live config | Delete 2 keys, rename 1, retune 5 |
| `_archive/prod-users/users/15.yaml` | fyttyn's save | One-time vitality freeze |
| `CLAUDE.md`, `internal/stats/context.md` | Docs | Rewrite soft-cap sections |

**`_datafiles/config.yaml` has `skip-worktree` set.** Before staging it: `git update-index --no-skip-worktree _datafiles/config.yaml`, stage, commit, then `git update-index --skip-worktree _datafiles/config.yaml`. A `git add` failure that mentions sparse-checkout is this flag.

---

## Task 1: Remove compression from `Recalculate()`

**Files:**
- Modify: `internal/stats/stats.go:40-58`
- Create: `internal/stats/softcap_test.go`

> `internal/stats` has no test file today. Adding one means Go builds a test binary for that package for the first time, which has previously triggered a Windows Defender false-positive quarantine (it hit `internal/relationships` in July). If `go test` fails with "the file contains a virus or potentially unwanted software", that is the environment, not the code — record it and let CI run the test.

- [ ] **Step 1: Write the failing test**

Create `internal/stats/softcap_test.go`:

```go
package stats

import "testing"

// Recalculate must be the identity: ValueAdj tracks Value with no
// compression. Before 2026-08-02 it compressed anything above 150, which
// silently applied to the resource pools too (they are StatInfo as well).
func TestRecalculateIsIdentity(t *testing.T) {
	for _, raw := range []int{0, 1, 85, 100, 105, 149, 150, 151, 165, 195, 280, 411, 530, 1000} {
		si := StatInfo{Base: raw}
		si.Recalculate()
		if si.Value != raw {
			t.Errorf("Base=%d: Value=%d, want %d", raw, si.Value, raw)
		}
		if si.ValueAdj != raw {
			t.Errorf("Base=%d: ValueAdj=%d, want %d (compression must be gone)", raw, si.ValueAdj, raw)
		}
	}
}

// Training and Mods must still sum into Value.
func TestRecalculateSumsComponents(t *testing.T) {
	si := StatInfo{Base: 85, Training: 326, Mods: 10}
	si.Recalculate()
	if si.Value != 421 || si.ValueAdj != 421 {
		t.Errorf("got Value=%d ValueAdj=%d, want 421/421", si.Value, si.ValueAdj)
	}
	if si.Racial != 85 {
		t.Errorf("Racial=%d, want 85", si.Racial)
	}
}

// A pool-sized value must pass through untouched. HealthMax/StaminaMax/
// ConvictionMax/ActionPointsMax are StatInfo and set Mods, so this is the
// path that was silently costing every character ~40% of every pool.
func TestPoolSizedValuesAreNotCompressed(t *testing.T) {
	si := StatInfo{Mods: 530}
	si.Recalculate()
	if si.ValueAdj != 530 {
		t.Errorf("pool Mods=530 -> ValueAdj=%d, want 530", si.ValueAdj)
	}
	ap := StatInfo{Mods: 200}
	ap.Recalculate()
	if ap.ValueAdj != 200 {
		t.Errorf("ActionPointsMax 200 -> ValueAdj=%d, want 200", ap.ValueAdj)
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/stats/ -run TestRecalculate -v`

Expected: FAIL. `TestRecalculateIsIdentity` reports `Base=151: ValueAdj=152` (the amplification) and `Base=530: ValueAdj=189`.

- [ ] **Step 3: Replace `Recalculate()`**

In `internal/stats/stats.go`, replace the whole function:

```go
func (si *StatInfo) Recalculate() {
	si.Racial = si.Base
	si.Value = si.Racial + si.Training + si.Mods
	si.ValueAdj = si.Value
}
```

`ValueAdj` is retained as a field so the ~189 call sites keep compiling. Collapsing it into `Value` is deliberately out of scope — see "Follow-up work".

- [ ] **Step 4: Drop the now-unused `math` import**

`math` was only used by the compression. Change the import block at the top of `internal/stats/stats.go` from:

```go
import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
)
```

to:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/configs"
)
```

If `go build` then reports `configs` as unused as well, remove that import too and delete the now-orphaned `b := configs.GetBalanceConfig()` line if any remains.

- [ ] **Step 5: Run the tests and the build**

Run: `go test ./internal/stats/ -v && go build ./...`
Expected: all three tests PASS, build succeeds with no output.

- [ ] **Step 6: Commit**

```bash
git add internal/stats/stats.go internal/stats/softcap_test.go
git commit -F - <<'EOF'
fix(stats): remove stat soft-cap compression entirely

Recalculate compressed any value above StatSoftCap. Because HealthMax,
StaminaMax, ConvictionMax and ActionPointsMax are also StatInfo and also
call Recalculate, that compression was silently applying to every resource
pool: a median character's true HP of 530 was being played as 322, and the
hardcoded ActionPointsMax of 200 was becoming 188. Neither effect was
intended.

ValueAdj is kept as a field so existing call sites compile; it is now
always equal to Value.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 2: Delete the dead config knobs and rename the surviving one

`StatSoftCap` is overloaded. It is the compression anchor (now gone) *and* the progression-rate soft cap plus anti-exploit floor in `internal/characters/progression.go`. The second meaning is the real brake on runaway stats and must survive — no prod character has organically exceeded 195 under it. Rename it so nobody mistakes it for a stat ceiling again.

**Files:**
- Modify: `internal/configs/config.balance.go:212,218`
- Modify: `internal/configs/config.balance.progression.go:10-11`
- Modify: `internal/characters/progression.go:167,174`
- Modify: `_datafiles/config.yaml:716-728`

- [ ] **Step 1: Delete `StatSoftCapMultiplier` and `StatSoftCapThreshold`, rename `StatSoftCap`**

In `internal/configs/config.balance.go`, delete the `StatSoftCapMultiplier` field entirely, delete the `StatSoftCapThreshold` field if still present, and rename `StatSoftCap`:

```go
	StatProgressionSoftCap ConfigInt `yaml:"StatProgressionSoftCap"` // Virtual rank where stat progression slows sharply (default 150). NOT a cap on stat values.
```

- [ ] **Step 2: Update the validator**

In `internal/configs/config.balance.progression.go`, delete the `StatSoftCapThreshold` and `StatSoftCapMultiplier` clamps if present, and rename the surviving clamp:

```go
	if b.StatProgressionSoftCap < 1 {
		b.StatProgressionSoftCap = 150
	}
```

- [ ] **Step 3: Update the two progression call sites**

In `internal/characters/progression.go`, line 167 and line 174, replace `b.StatSoftCap` with `b.StatProgressionSoftCap`:

```go
	if statVal := c.GetStatValue(statName); statVal > int(b.StatProgressionSoftCap) && statVal > virtualRank {
		virtualRank = statVal
	}
```

```go
	chance := CalculateProgressionChance(virtualRank, int(b.StatProgressionSoftCap)) * bonusMultiplier * mutStatMult * statMult * float64(b.StatProgressionRate)
```

- [ ] **Step 4: Build to confirm the compiler found every consumer**

Run: `go build ./...`
Expected: succeeds. If it reports `b.StatSoftCap undefined` anywhere else, rename those too and re-run — the compiler is the authoritative sweep here.

- [ ] **Step 5: Update `_datafiles/config.yaml`**

Replace the soft-cap block (around lines 716-728) with:

```yaml
  StatProgressionSoftCap: 150    # Virtual rank where stat progression slows sharply.
                                 # NOT a ceiling on stat values — stats are uncompressed.
```

Delete the `StatSoftCap:`, `StatSoftCapThreshold:` and `StatSoftCapMultiplier:` lines and any comment block describing the compression formula. Leaving an orphaned key is harmless (`yaml.Unmarshal` is non-strict, verified at `internal/configs/configs.go:115`), but leaving it invites someone to tune a knob that does nothing.

- [ ] **Step 6: Run the config tests**

Run: `go test ./internal/configs/ ./internal/characters/`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git update-index --no-skip-worktree _datafiles/config.yaml
git add internal/configs/config.balance.go internal/configs/config.balance.progression.go internal/characters/progression.go _datafiles/config.yaml
git commit -F - <<'EOF'
refactor(config): rename StatSoftCap to StatProgressionSoftCap, drop dead knobs

StatSoftCap meant two unrelated things: the compression anchor (deleted in
the previous commit) and the progression-rate soft cap plus anti-exploit
floor in CheckStatProgression. Only the second survives, so name it for
what it does. StatSoftCapMultiplier and StatSoftCapThreshold are deleted;
the threshold had been inert since the anchor moved from 100 to 150.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
git update-index --skip-worktree _datafiles/config.yaml
```

---

## Task 3: Pin the intended pool formulas

Write the test *first*, against the intended design, so Task 4 is verified rather than asserted. These tests fail until Task 4 lands — that is the point.

**Files:**
- Create: `internal/characters/poolmax_test.go`

- [ ] **Step 1: Write the failing pool-formula test**

Create `internal/characters/poolmax_test.go`:

```go
package characters

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// Go test binaries run with CWD set to their own package directory, so a
// test that reads the real config must chdir to the repo root first.
// Precedent: internal/web/auth_test.go.
func withRepoRoot(t *testing.T) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Join(cwd, "..", "..")
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir to repo root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	configs.ReloadConfig()
}

// poolsFor builds a character with the six stats set and returns its
// derived pool maxima.
func poolsFor(t *testing.T, str, dex, per, vit, wil, cha int) (int, int, int) {
	t.Helper()
	c := New()
	c.Stats.Strength.Base = str
	c.Stats.Dexterity.Base = dex
	c.Stats.Perception.Base = per
	c.Stats.Vitality.Base = vit
	c.Stats.Willpower.Base = wil
	c.Stats.Charisma.Base = cha
	c.RecalculateStats()
	return c.HealthMax.Value, c.StaminaMax.Value, c.ConvictionMax.Value
}

// TestPoolMaxFormula pins the intended pool design:
//
//	HealthMax     = 5 + Vitality*3 + Strength*1
//	StaminaMax    = 5 + Vitality*3 + Willpower*1
//	ConvictionMax = 5 + Charisma*3 + Willpower*1
//
// Expected values are exact — every term is integer arithmetic on raw
// stats, so there is no reason to accept a tolerance here.
func TestPoolMaxFormula(t *testing.T) {
	withRepoRoot(t)

	cases := []struct {
		name                   string
		str, dex, per          int
		vit, wil, cha          int
		wantHP, wantSP, wantCP int
	}{
		// 5 + vit*3 + str, 5 + vit*3 + wil, 5 + cha*3 + wil
		{"median", 105, 105, 105, 105, 105, 105, 425, 425, 425},
		{"Duard", 104, 87, 175, 134, 195, 111, 511, 602, 533},
		{"meirok", 136, 110, 152, 117, 148, 123, 492, 504, 522},
		{"fyttyn_frozen", 128, 123, 182, 280, 243, 140, 973, 1088, 668},
	}

	for _, tc := range cases {
		hp, sp, cp := poolsFor(t, tc.str, tc.dex, tc.per, tc.vit, tc.wil, tc.cha)
		if hp != tc.wantHP {
			t.Errorf("%s HealthMax = %d, want %d", tc.name, hp, tc.wantHP)
		}
		if sp != tc.wantSP {
			t.Errorf("%s StaminaMax = %d, want %d", tc.name, sp, tc.wantSP)
		}
		if cp != tc.wantCP {
			t.Errorf("%s ConvictionMax = %d, want %d", tc.name, cp, tc.wantCP)
		}
	}
}

// Strength must not contribute to stamina, and willpower must be the
// secondary (not equal) contributor to conviction. Both were true before
// 2026-08-02 and are corrected here, so pin them against regression.
func TestPoolMaxStatRoles(t *testing.T) {
	withRepoRoot(t)

	baseHP, baseSP, baseCP := poolsFor(t, 100, 100, 100, 100, 100, 100)

	// +10 Strength: HealthMax +10, StaminaMax unchanged.
	hp, sp, _ := poolsFor(t, 110, 100, 100, 100, 100, 100)
	if hp != baseHP+10 {
		t.Errorf("+10 Str: HealthMax %d, want %d", hp, baseHP+10)
	}
	if sp != baseSP {
		t.Errorf("+10 Str: StaminaMax %d, want %d (strength must not feed stamina)", sp, baseSP)
	}

	// +10 Charisma: ConvictionMax +30 (primary), HealthMax unchanged.
	hp2, _, cp := poolsFor(t, 100, 100, 100, 100, 100, 110)
	if cp != baseCP+30 {
		t.Errorf("+10 Cha: ConvictionMax %d, want %d (charisma is primary)", cp, baseCP+30)
	}
	if hp2 != baseHP {
		t.Errorf("+10 Cha: HealthMax %d, want %d", hp2, baseHP)
	}
}
```

`New()` is the package's `*Character` constructor (`internal/characters/character.go:328`); existing tests such as `bandolier_onepertype_test.go:20` use it the same way.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/characters/ -run TestPoolMax -v`

Expected: FAIL. With compression removed (Task 1) but the coefficients still at their drifted values, `median` HealthMax comes back as 530 (`5 + 105*1 + 105*4`) against the intended 425, and `TestPoolMaxStatRoles` fails because strength still feeds stamina and conviction still weights willpower equally with charisma.

- [ ] **Step 3: Commit the failing test**

```bash
git add internal/characters/poolmax_test.go
git commit -m "test(characters): pin the intended pool formulas

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Restore the intended pool formulas

The pool formulas drifted from their intended design while being tuned against compressed output. The target design is:

    HealthMax     = 5 + Vitality*3 + Strength*1
    StaminaMax    = 5 + Vitality*3 + Willpower*1
    ConvictionMax = 5 + Charisma*3 + Willpower*1

Each pool gets one primary stat at x3 and one secondary at x1. Two structural corrections fall out: **strength stops feeding stamina**, and **conviction becomes charisma-primary** instead of weighting willpower and charisma equally. The latter needs a code change, because the current formula applies a single coefficient to `(Wil + Cha)` and cannot express an asymmetric split.

This is **not** power-neutral — see "Accepted balance consequences" below. That is a deliberate decision.

**Files:**
- Modify: `internal/characters/validate.go:96-97`
- Modify: `internal/configs/config.balance.go:208`
- Modify: `internal/configs/config.balance.misc.go:36-55`
- Modify: `_datafiles/config.yaml:692-707`

- [ ] **Step 1: Split the conviction coefficient in the config struct**

In `internal/configs/config.balance.go`, replace the `ConvictionPerWilCha` field with two:

```go
	ConvictionPerCharisma  ConfigInt `yaml:"ConvictionPerCharisma"`  // Charisma multiplier toward ConvictionMax, primary (default 3)
	ConvictionPerWillpower ConfigInt `yaml:"ConvictionPerWillpower"` // Willpower multiplier toward ConvictionMax, secondary (default 1)
```

- [ ] **Step 2: Update the validator defaults**

In `internal/configs/config.balance.misc.go`, replace the `ConvictionPerWilCha` clamp (lines 54-55) with:

```go
	if b.ConvictionPerCharisma < 0 {
		b.ConvictionPerCharisma = 3
	}
	if b.ConvictionPerWillpower < 0 {
		b.ConvictionPerWillpower = 1
	}
```

In the same file, update the stale pool defaults so a missing config key produces the intended design rather than the drifted one:

```go
	if b.HealthPerVitality < 0 {
		b.HealthPerVitality = 3
	}
```

Check the neighbouring clamps in that block and set `HealthPerStrength` to 1, `StaminaPerVitality` to 3, `StaminaPerWillpower` to 1, and `StaminaPerStrength` to 0 wherever they appear. Leave the three `*Base` defaults at 5.

- [ ] **Step 3: Update the derivation in `RecalculateStats`**

In `internal/characters/validate.go`, replace the conviction line (currently lines 96-97):

```go
	c.ConvictionMax.Mods = int(rb.ConvictionBase) +
		c.Stats.Charisma.ValueAdj*int(rb.ConvictionPerCharisma) +
		c.Stats.Willpower.ValueAdj*int(rb.ConvictionPerWillpower)
```

Leave the `HealthMax.Mods` and `StaminaMax.Mods` lines structurally unchanged — their coefficients already exist and are set by config in the next step. `StaminaPerStrength: 0` is what removes strength's contribution to stamina; the term stays in the code and evaluates to zero.

- [ ] **Step 4: Build to confirm every consumer was found**

Run: `go build ./...`
Expected: succeeds. If `ConvictionPerWilCha` is reported undefined anywhere else, update those sites — the compiler is the authoritative sweep.

- [ ] **Step 5: Apply the coefficients in `_datafiles/config.yaml`**

Update the formula comment (around line 692) and the values:

```yaml
  #   HealthMax     = HealthBase + Vit x HealthPerVitality + Str x HealthPerStrength
  #   StaminaMax    = StaminaBase + Vit x StaminaPerVitality + Wil x StaminaPerWillpower
  #   ConvictionMax = ConvictionBase + Cha x ConvictionPerCharisma + Wil x ConvictionPerWillpower
  #
  # Restored 2026-08-02 to the intended design: each pool has one primary
  # stat (x3) and one secondary (x1). The previous values had drifted while
  # being tuned against compressed output — HealthMax is a StatInfo, so
  # Recalculate() was silently shrinking every pool by roughly 40%. Pools
  # are larger now; this is accepted, and NPCs use the same formulas.
  HealthBase: 5                  # Flat HP before stat contribution
  HealthPerVitality: 3           # + Vit x this toward HealthMax (primary)
  HealthPerStrength: 1           # + Str x this toward HealthMax (secondary)
  StaminaBase: 5                 # Flat stamina before stat contribution
  StaminaPerVitality: 3          # + Vit x this toward StaminaMax (primary)
  StaminaPerWillpower: 1         # + Wil x this toward StaminaMax (secondary)
  StaminaPerStrength: 0          # Strength does not contribute to stamina
  ConvictionBase: 5              # Flat conviction before stat contribution
  ConvictionPerCharisma: 3       # + Cha x this toward ConvictionMax (primary)
  ConvictionPerWillpower: 1      # + Wil x this toward ConvictionMax (secondary)
```

Delete the old `ConvictionPerWilCha:` line.

- [ ] **Step 6: Run the pool tests**

Run: `go test ./internal/characters/ -run "TestPoolMax" -v`
Expected: both `TestPoolMaxFormula` and `TestPoolMaxStatRoles` PASS.

- [ ] **Step 7: Run the full affected suite**

Run: `go test ./internal/characters/ ./internal/stats/ ./internal/configs/ ./internal/combat/`

Expected: PASS. Tests asserting specific HP/SP/CP numbers **will** fail — pools are intentionally larger now. Update each to the new expected value and list them in the commit message. Do not weaken an assertion to a range to make it pass.

- [ ] **Step 8: Commit**

```bash
git update-index --no-skip-worktree _datafiles/config.yaml
git add internal/characters/validate.go internal/configs/config.balance.go internal/configs/config.balance.misc.go _datafiles/config.yaml
git commit -F - <<'EOF'
balance: restore intended pool formulas, split conviction coefficient

Each pool now has one primary stat at x3 and one secondary at x1:

  HealthMax     = 5 + Vit*3 + Str*1
  StaminaMax    = 5 + Vit*3 + Wil*1
  ConvictionMax = 5 + Cha*3 + Wil*1

Two structural corrections: strength no longer contributes to stamina, and
conviction is charisma-primary rather than weighting willpower and charisma
equally. The latter needed a code change — ConvictionPerWilCha applied one
coefficient to (Wil+Cha) and could not express an asymmetric split, so it
is replaced by ConvictionPerCharisma and ConvictionPerWillpower.

Pools are materially larger than before, because the old coefficients had
been tuned against soft-cap-compressed output. This is accepted: NPCs use
the same formulas, so relative power holds. Fights will run longer, since
damage derives from stats rather than pools — see the plan's accepted
balance consequences.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
git update-index --skip-worktree _datafiles/config.yaml
```

---

## Task 5: Freeze fyttyn's exploited vitality

fyttyn reached vitality 411 through an exploit that has since been fixed, and has been *playing* at the compressed value of 280. Freezing raw vitality to 280 keeps their actual power identical across the switchover while removing the hidden 131 points.

`base: 85`, `training: 326`, total 411. Target 280 means `training: 195`.

**Files:**
- Modify: `_archive/prod-users/users/15.yaml:79-81`

- [ ] **Step 1: Back up the save**

```bash
cp _archive/prod-users/users/15.yaml _archive/prod-users/users/15.yaml.pre-softcap-removal
```

- [ ] **Step 2: Edit the vitality block**

In `_archive/prod-users/users/15.yaml`, change:

```yaml
    vitality:
      training: 326
      base: 85
```

to:

```yaml
    vitality:
      training: 195
      base: 85
```

- [ ] **Step 3: Verify the arithmetic**

Run: `grep -A 2 "^    vitality:" _archive/prod-users/users/15.yaml`
Expected: `training: 195` and `base: 85`, summing to 280 — the value fyttyn has effectively been playing with.

- [ ] **Step 4: Commit**

```bash
git add _archive/prod-users/users/15.yaml
git commit -m "balance(prod-users): freeze fyttyn vitality at its effective 280

Raw 411 came from a since-fixed exploit and was being compressed to 280 in
play. Freezing raw to the value actually in use keeps power identical
across the soft-cap removal instead of handing back 131 hidden points.
Original saved alongside as 15.yaml.pre-softcap-removal.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

> This edits the archived copy. The same edit must be applied to the live droplet save before or during deployment — the live file is authoritative and is not in this repo. Do not skip this.

---

## Task 6: Update the documentation

**Files:**
- Modify: `CLAUDE.md` (Stat & Progression System section)
- Modify: `internal/stats/context.md` (Stat Calculation System section)

- [ ] **Step 1: Update `CLAUDE.md`**

Replace the soft-cap bullet in the "Stat & Progression System" section with:

```markdown
- **There is no soft cap on stat values.** `ValueAdj == Value` always; stats
  are used raw. Compression was removed 2026-08-02 — it was inherited from
  upstream, hid ~10 points from three veteran characters, and (because
  `HealthMax`/`StaminaMax`/`ConvictionMax`/`ActionPointsMax` are also
  `StatInfo`) was silently shrinking every resource pool by roughly 40%.
- `StatProgressionSoftCap` (default 150) is the *virtual rank* where
  progression slows sharply, plus the anti-exploit floor in
  `CheckStatProgression`. It is not a ceiling on stat values. This is the
  real brake on runaway stats: no prod character has organically exceeded
  195 under it.
```

- [ ] **Step 2: Update `internal/stats/context.md`**

Replace the "Diminishing Returns" block with:

```markdown
### Stat Calculation

    Value = Base (Racial) + Training + Mods
    ValueAdj = Value

There is no compression. `ValueAdj` is retained only so existing call sites
compile; it is always equal to `Value`. Do not reintroduce a soft cap here
— `HealthMax`, `StaminaMax`, `ConvictionMax` and `ActionPointsMax` are also
`StatInfo` and call the same `Recalculate()`, so anything added here
silently applies to every resource pool as well. That was the 2026-08-02
bug. Pool sizing belongs in the `HealthPer*`/`StaminaPer*`/`ConvictionPer*`
coefficients in the balance config, where it is visible and tunable.
```

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md internal/stats/context.md
git commit -m "docs: record the stat soft-cap removal and its pool consequence

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Full verification

- [ ] **Step 1: Build and test the whole tree**

Run: `go build ./... && go test ./... 2>&1 | grep -v "^ok\|no test files"`

Expected: no failures. `internal/relationships` may fail with a Windows Defender virus-detection message on its test binary — that is a known environment issue, unrelated to this change. Record it; do not attempt to disable antivirus.

- [ ] **Step 2: Verify gofmt**

Run: `gofmt -l internal/stats internal/configs internal/characters`
Expected: no output.

- [ ] **Step 3: Boot test in an isolated worktree**

Do **not** start a server in the main working directory — the user runs the local server there and it must not be disturbed.

```bash
git worktree add ../DOGMud-boottest HEAD
cd ../DOGMud-boottest
go run . 2>&1 | tee /tmp/boot.log
```

Expected: startup reaches `mobs.LoadDataFiles() loadedCount=...`, `quests.LoadDataFiles() loadedCount=...` etc. with no panic. Stop the server, then:

```bash
cd - && git worktree remove ../DOGMud-boottest
```

- [ ] **Step 4: Confirm the parked branch is not merged**

Run: `git branch --merged master | grep softcap`
Expected: no output. `fix/stat-softcap-amplification` fixes a curve this plan deletes and must stay unmerged.

- [ ] **Step 5: In-game verification**

Removing compression changes combat feel at the pool level for every character, which no unit test can assess. Per the Content Playtest-Review Gate in `CLAUDE.md`, run an adversarial playtest before handing this to the user:

```
/playtest local bug-finder
```

Drive real combat at a range of stat levels and confirm fights do not feel materially longer or shorter than before. Report anything that reads as a pacing change.

- [ ] **Step 6: Update the patch notes**

Add a dated entry to `docs/PATCH_NOTES.md` describing the soft-cap removal, the pool recalibration, and the fyttyn freeze. Commit.

---

## Follow-up work (explicitly out of scope)

- **Collapse `ValueAdj` into `Value`.** Now that they are always equal, the field is redundant across ~189 call sites. Delete the field and let `go build` enumerate the consumers. Kept separate so that if something breaks it is obvious whether the cause was the balance change or a mechanical refactor.
- **`grappleScore` reads raw `.Value`** while all other combat reads `ValueAdj` (`internal/hooks/Position_GrappleTick.go:647-648`). This inconsistency becomes harmless once the two are equal, and disappears entirely with the collapse above.
- **Re-examine `StatProgressionRate` and `StatProgressionMultipliers`.** Prod use counts show characters with 40,000+ uses of a stat and near-zero gain, which suggests the progression curve may be tuned harder than intended. Independent of this plan.
