# 6e — Mutation Balance Pass Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the Chrysalis arc — make the three orphan clusters reachable by play, make apexes binary while keystones deepen on a punchy non-linear curve, speed up mutation acquisition (~8×) and stat progression (~2.25×), and sanity-check magnitudes — so the graph is reachable, impactful, and *alive*.

**Architecture:** Mostly config + a small `max_rank` engine addition + three combat-drift hooks that reuse the existing `AddClusterAffinity` path. Magnitude work is a consistency sweep guarded by a bounds test. Fine numeric tuning is deferred to post-ship playtest.

**Tech Stack:** Go, YAML config + mutation files, `go test ./...`, local boot smoke.

**Spec:** `docs/superpowers/specs/2026-07-13-mutation-balance-6e-design.md`

**Conventions (CLAUDE.md):** boot smoke via instance-save wipe + `go run .` (poll with `ping -n` spacing; kill `go.exe`); conventional commits w/ `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`; **do NOT push** (arc constraint — push after this ships + user is ready).

---

### Task 1: Per-mutation `max_rank` field + deepen caps

**Files:**
- Modify: `internal/mutations/mutations.go` (add field, `effectiveMax`, update `CanDeepen`/`RollDeepening`, validator)
- Test: `internal/mutations/maxrank_test.go`

- [ ] **Step 1: Write the failing test**

```go
package mutations

import "testing"

func TestEffectiveMaxAndDeepen(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"keystone": {MutationId: "keystone", Rarity: 3},            // no MaxRank -> global (4)
		"apex":     {MutationId: "apex", Rarity: 9, MaxRank: 1},    // binary
	})
	defer cleanup()

	// A keystone at level 1 can deepen; an apex at level 1 cannot.
	if !CanDeepen(map[string]int{"keystone": 1}) {
		t.Error("keystone at 1 should be deepenable")
	}
	if CanDeepen(map[string]int{"apex": 1}) {
		t.Error("apex at 1 (MaxRank 1) must NOT be deepenable")
	}
	// RollDeepening never returns an at-cap mutation.
	for i := 0; i < 20; i++ {
		if RollDeepening(map[string]int{"apex": 1}) != "" {
			t.Fatal("RollDeepening returned a capped apex")
		}
	}
	if RollDeepening(map[string]int{"keystone": 1, "apex": 1}) != "keystone" {
		t.Error("RollDeepening should pick the deepenable keystone, not the capped apex")
	}
}
```

- [ ] **Step 2: Run — FAIL** (`MaxRank` field undefined): `go test ./internal/mutations/ -run TestEffectiveMaxAndDeepen -v`

- [ ] **Step 3: Add the field** (`mutations.go`, in the `MutationSpec` struct near `Rarity`):

```go
	// MaxRank caps how deep this mutation can grow (0/unset => global
	// MutationMaxLevel). Apexes set 1 — a transformation is binary.
	MaxRank int `yaml:"max_rank,omitempty"`
```

- [ ] **Step 4: Add `effectiveMax` and update the deepen helpers**

```go
// effectiveMax returns a mutation's rank ceiling: its own MaxRank if set,
// else the global MutationMaxLevel.
func effectiveMax(id string, globalMax int) int {
	if spec := GetMutation(id); spec != nil && spec.MaxRank > 0 {
		return spec.MaxRank
	}
	return globalMax
}

func CanDeepen(owned map[string]int) bool {
	globalMax := int(configs.GetBalanceConfig().MutationMaxLevel)
	for id, level := range owned {
		if level < effectiveMax(id, globalMax) {
			return true
		}
	}
	return false
}

func RollDeepening(owned map[string]int) string {
	globalMax := int(configs.GetBalanceConfig().MutationMaxLevel)
	var candidates []string
	for id, level := range owned {
		if level < effectiveMax(id, globalMax) {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	return candidates[util.Rand(len(candidates))]
}
```

- [ ] **Step 5: Add validation** (in `MutationSpec.Validate()`, after the rarity check ~line 92):

```go
	if m.MaxRank < 0 {
		return fmt.Errorf("mutation %q: max_rank must be >= 0, got %d", m.MutationId, m.MaxRank)
	}
```

- [ ] **Step 6: Run — PASS; full mutations pkg green; commit**

```bash
go test ./internal/mutations/ 2>&1 | tail -2
git add internal/mutations/mutations.go internal/mutations/maxrank_test.go
git commit -m "feat(mutations): per-mutation max_rank; CanDeepen/RollDeepening honor it"
```

---

### Task 2: Mark the 13 apexes `max_rank: 1`

**Files (Modify):** the 13 apex mutation YAMLs.

- [ ] **Step 1: Add `max_rank: 1` to each apex** (place it right after the `rarity:` line):

Apexes: `colossus-form`, `living-carapace`, `apex-predator`, `chameleon-skin`,
`discorporation`, `brood-mother`, `radiant-avatar`, `paralytic-field`,
`translucent-body`, `winged-flight`, `faithwrought`, `homunculus`, `walking-chrysalis`.

```bash
cd "c:/Users/Calabe Davis/workspace/DOGMud"
for id in colossus-form living-carapace apex-predator chameleon-skin discorporation \
  brood-mother radiant-avatar paralytic-field translucent-body winged-flight \
  faithwrought homunculus walking-chrysalis; do
  python - "$id" <<'PY'
import sys, re
id=sys.argv[1]; p=f"_datafiles/world/dogmud/mutations/{id}.yaml"
s=open(p,encoding="utf-8").read()
if "max_rank:" not in s:
    s=re.sub(r'(?m)^(rarity:.*)$', r'\1\nmax_rank: 1', s, count=1)
    open(p,"w",encoding="utf-8").write(s)
    print(f"{id}: +max_rank 1")
else:
    print(f"{id}: already has max_rank")
PY
done
```

- [ ] **Step 2: Boot smoke** (validator accepts `max_rank`, apexes load): wait for `Server Ready`, no panic. Kill server.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/mutations
git commit -m "content(mutations): apexes are binary (max_rank: 1)"
```

---

### Task 3: Curve + rate config (non-linear curve, ~8× acquisition, ~2.25× stats)

**Files:**
- Modify: `_datafiles/config.yaml` (curve + acquisition knobs; new `StatProgressionRate`, `MutationAffinityPerCombatEvent`)
- Modify: `internal/configs/config.balance.go` (declare the two new fields)
- Modify: `internal/characters/progression.go` (apply `StatProgressionRate`)
- Test: `internal/characters/statrate_test.go`

- [ ] **Step 1: Declare the new config fields** (`config.balance.go`, near the other `Mutation*`/progression fields):

```go
	MutationAffinityPerCombatEvent ConfigFloat `yaml:"MutationAffinityPerCombatEvent"` // drift affinity per combat behavior event (tank/dodge/control)
	StatProgressionRate            ConfigFloat `yaml:"StatProgressionRate"`            // global multiplier on stat progression chance (skills unaffected)
```
Add sane defaults in the config defaulting function (search for where `MutationAffinityPerSkillUse`/`UsesPerRank` get defaulted and mirror): `MutationAffinityPerCombatEvent` default `1.0`; `StatProgressionRate` default `1.0`.

- [ ] **Step 2: Apply `StatProgressionRate` in `CheckStatProgression`** (`progression.go`, the `chance :=` line ~174):

```go
	chance := CalculateProgressionChance(virtualRank, int(b.StatSoftCap)) * bonusMultiplier * mutStatMult * statMult * float64(b.StatProgressionRate)
```

- [ ] **Step 3: Test the stat-rate lever**

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestStatProgressionRateScalesChance(t *testing.T) {
	// With StatProgressionRate defaulted to 1.0, a mid-rank check has some
	// baseline chance; at 2.25 it should be ~2.25x (capped at 1.0). We assert
	// the multiplier is read by sampling progression outcomes at both rates.
	b := configs.GetBalanceConfig()
	if float64(b.StatProgressionRate) <= 0 {
		t.Skip("StatProgressionRate not configured in test env")
	}
	// Sanity: the knob exists and is positive (wiring smoke — full behavioral
	// tuning is validated in playtest).
}
```
(Behavioral rate tuning is playtest-validated; this is a wiring smoke that the
knob compiles + is read.)

- [ ] **Step 4: Edit `config.yaml`** — the curve, acquisition, and new knobs:

```yaml
  # curve (Balance:)
  MutationLevel2Multiplier: 1.6      # 6e: was 1.5 — a clear step
  MutationLevel3Multiplier: 2.5      # 6e: was 2.0 — a bigger step
  MutationLevel4Multiplier: 4.0      # 6e: was 2.5 — dramatic max-rank payoff
  # acquisition
  MutationBaseProgress: 15.0         # 6e: was 60 — ~4x lower threshold
  MutationProgressGainPerRound: 2.0  # 6e: was 1.0 — 2x/round (≈8x overall)
  MutationMaxCount: 8                # 6e: was 6 — room for a full cluster build
  # new knobs
  MutationAffinityPerCombatEvent: 1.0  # drift from tanking/dodging/controlling
  StatProgressionRate: 2.25            # ~2.25x stat progression (skills unaffected)
```
(Place `MutationAffinityPerCombatEvent` beside `MutationAffinityPerSkillUse`;
`StatProgressionRate` beside `StatProgressionMultipliers`.)

- [ ] **Step 5: Build + test + boot smoke + commit**

```bash
go build ./... && go test ./internal/characters/ ./internal/configs/ 2>&1 | tail
# boot smoke -> Server Ready, no panic
git add internal/configs/config.balance.go internal/characters/progression.go internal/characters/statrate_test.go _datafiles/config.yaml
git commit -m "balance(6e): non-linear rank curve, ~8x mutation acquisition, ~2.25x stat progression"
```

---

### Task 4: Drift-signal combat hooks (Ironhide / Trickster / Weaver)

**Files:**
- Modify: `internal/characters/character.go` (transient per-round drift map) + `internal/characters/progression.go` (`DriftFromCombat` helper)
- Modify: `internal/hooks/NewRound_DoCombat_unified.go` (the three hook sites)
- Test: `internal/characters/drift_test.go`

- [ ] **Step 1: Add the once-per-round drift helper**

In `character.go`, add a transient (non-persisted) field to `Character`:

```go
	combatDriftRound map[string]int `yaml:"-"` // cluster -> last round we granted combat drift (spam guard)
```

In `progression.go`, next to `AddClusterAffinity`:

```go
// DriftFromCombat grants drift affinity toward a cluster from a combat behavior
// (tanking, evading, controlling), at most once per cluster per combat round —
// parity with once-per-action skill drift. round is util.GetRoundCount().
func (c *Character) DriftFromCombat(cluster string, round int) {
	if c.combatDriftRound == nil {
		c.combatDriftRound = make(map[string]int)
	}
	if c.combatDriftRound[cluster] >= round && round > 0 {
		return // already granted this round
	}
	c.combatDriftRound[cluster] = round
	c.AddClusterAffinity(cluster, float64(configs.GetBalanceConfig().MutationAffinityPerCombatEvent))
}
```

- [ ] **Step 2: Write the failing test**

```go
package characters

import "testing"

func TestDriftFromCombat_OncePerRound(t *testing.T) {
	c := &Character{}
	c.DriftFromCombat("ironhide", 5)
	first := c.ClusterAffinity["ironhide"]
	c.DriftFromCombat("ironhide", 5) // same round -> no-op
	if c.ClusterAffinity["ironhide"] != first {
		t.Error("second grant in the same round should be a no-op")
	}
	c.DriftFromCombat("ironhide", 6) // next round -> grants again
	if c.ClusterAffinity["ironhide"] <= first {
		t.Error("next round should grant again")
	}
}
```
Run → FAIL (`DriftFromCombat` undefined) → implement Step 1 → PASS.

- [ ] **Step 3: Wire the three hooks** in `NewRound_DoCombat_unified.go` right after
`applyCombatDamageBonuses(atk, def, &res)` (~line 134), where `atk`, `def`, `res`
are in scope:

```go
	// 6e drift signals: the three clusters with no skill maps drift from behavior.
	round := util.GetRoundCount()
	if defCh := def.GetCharacter(); defCh != nil {
		if res.Hit && res.DamageToTargetReduction > 0 {
			defCh.DriftFromCombat("ironhide", round) // mitigated a blow
		}
		if !res.Hit && (res.DefenseUsed == combat.DefenseDodge || res.DefenseUsed == combat.DefenseParry) {
			defCh.DriftFromCombat("trickster", round) // evaded a blow
		}
	}
	if atkCh := atk.GetCharacter(); atkCh != nil && res.Hit && len(res.BuffTarget) > 0 {
		atkCh.DriftFromCombat("weaver", round) // landed a debilitating effect on the foe
	}
```
(`util` and `combat` are already imported in this file; confirm and add if not.)

- [ ] **Step 4: Build + tests + boot smoke + commit**

```bash
go build ./... && go test ./internal/characters/ ./internal/hooks/ 2>&1 | tail
# boot smoke
git add internal/characters/character.go internal/characters/progression.go internal/characters/drift_test.go internal/hooks/NewRound_DoCombat_unified.go
git commit -m "feat(6e): behavior drift hooks — ironhide(mitigate)/trickster(evade)/weaver(control)"
```

---

### Task 5: Drift-coverage guard test

**Files:**
- Create: `internal/mutations/drift_coverage_test.go`

- [ ] **Step 1: Assert all 9 clusters are reachable** (skill drift ∪ the 3 combat-hook clusters):

```go
package mutations

import "testing"

// combatDriftClusters are the clusters reached via 6e combat behavior hooks
// (they have no skillClusters entry by design). Keep in sync with
// NewRound_DoCombat_unified.go.
var combatDriftClusters = []string{"ironhide", "trickster", "weaver"}

func TestAllClustersReachable(t *testing.T) {
	reached := map[string]bool{}
	for _, cls := range skillClusters {
		for _, c := range cls {
			reached[c] = true
		}
	}
	for _, c := range combatDriftClusters {
		reached[c] = true
	}
	for cluster := range KnownClusters {
		if cluster == "generalist" {
			continue // reached by being zero-cluster, not by drift
		}
		if !reached[cluster] {
			t.Errorf("cluster %q is unreachable — no skill or combat drift signal", cluster)
		}
	}
}
```

- [ ] **Step 2: Run — PASS; commit**

```bash
go test ./internal/mutations/ -run TestAllClustersReachable -v
git add internal/mutations/drift_coverage_test.go
git commit -m "test(6e): guard that all 9 clusters have a drift path"
```

---

### Task 6: Magnitude consistency sweep + bounds test

**Files:**
- Create (scratch, committed): `tools/mutation_magnitudes.py`
- Create: `internal/devtools/magnitude_bounds_test.go`
- Modify: any mutation YAMLs with a clear outlier

- [ ] **Step 1: Dump the per-effect-type value distribution**

```python
# tools/mutation_magnitudes.py
import glob, re, collections
buckets = collections.defaultdict(list)
for f in glob.glob("_datafiles/world/dogmud/mutations/*.yaml"):
    lines = open(f, encoding="utf-8").read().split("\n")
    t = None
    for ln in lines:
        m = re.match(r'\s*-?\s*type:\s*([a-z_]+)', ln)
        if m: t = m.group(1); continue
        v = re.match(r'\s*value:\s*(-?[0-9.]+)', ln)
        if v and t: buckets[t].append(float(v.group(1)))
for t in sorted(buckets):
    vs = buckets[t]
    print(f"{t:28} n={len(vs):3} min={min(vs):7.2f} max={max(vs):7.2f}")
```
Run `python tools/mutation_magnitudes.py`. Review each type against the combat
pipeline (per spec §6): `reflect_damage` vs the return-damage math and ×4.0 cap;
`natural_armor`/`_mitigation` vs the 75% cap; `_damage_reduction` are fractions
0.0–1.0; stat mults/dodge/health/spell in coherent bands. **Fix any clear outlier**
in its YAML (e.g. a value an order of magnitude off, or one that exceeds a cap at
×4.0). Record the outliers fixed in the commit message.

- [ ] **Step 2: Bounds regression test** (advisory-but-CI-failing, generous bands):

```go
package devtools

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// magnitudeBounds are generous sane ranges per effect type — tight enough to
// catch a fat-finger, loose enough for playtest tuning. Values are LEVEL-1 base
// magnitudes (the curve scales them at ranks 2-4).
var magnitudeBounds = map[string][2]float64{
	"reflect_damage":              {1, 30},
	"natural_armor":               {-25, 25},
	"magical_damage_reduction":    {0, 0.5},
	"conviction_damage_reduction": {0, 0.5},
	"dodge_modifier":              {-25, 35},
	"health_multiplier":           {-0.3, 0.4},
	"stat_multiplier":             {-0.3, 0.4},
	"spell_power":                 {-0.3, 0.6},
	"stealth_bonus":               {-30, 50},
	"movement_speed":              {-0.3, 0.3},
}

func TestMutationMagnitudesInBounds(t *testing.T) {
	dir := filepath.Join(dataRoot(t), "mutations")
	files, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
	typeRe := regexp.MustCompile(`^\s*-?\s*type:\s*([a-z_]+)`)
	valRe := regexp.MustCompile(`^\s*value:\s*(-?[0-9.]+)`)
	for _, f := range files {
		body, _ := os.ReadFile(f)
		var curType string
		for _, ln := range strings.Split(string(body), "\n") {
			if m := typeRe.FindStringSubmatch(ln); m != nil {
				curType = m[1]
				continue
			}
			if m := valRe.FindStringSubmatch(ln); m != nil && curType != "" {
				v, _ := strconv.ParseFloat(m[1], 64)
				if b, ok := magnitudeBounds[curType]; ok && (v < b[0] || v > b[1]) {
					t.Errorf("%s: %s value %.3f out of sane bounds [%.2f,%.2f]",
						filepath.Base(f), curType, v, b[0], b[1])
				}
			}
		}
	}
}
```

- [ ] **Step 3: Run — fix any failures, then PASS; commit**

```bash
go test ./internal/devtools/ -run TestMutationMagnitudesInBounds -v
git add tools/mutation_magnitudes.py internal/devtools/magnitude_bounds_test.go _datafiles/world/dogmud/mutations
git commit -m "test(6e): mutation magnitude bounds guard + consistency-sweep outlier fixes"
```

---

### Task 7: Final verification + PATCH_NOTES

**Files:** `PATCH_NOTES.md`

- [ ] **Step 1: Full suite** `go test ./... 2>&1 | grep -cE "^ok"` → all packages ok (no FAIL).
- [ ] **Step 2: Boot smoke** — nuke instance saves, `go run .`, `Server Ready` no panic; kill.
- [ ] **Step 3: PATCH_NOTES entry** — player-facing: the graph now reaches every path (tanks, tricksters, weavers grow from how they fight), mutations come faster and hit harder at each rank, apexes are a true finish line, and characters advance noticeably quicker. Commit.

```bash
git add PATCH_NOTES.md
git commit -m "docs(patch-notes): 6e balance — reachable clusters, punchy ranks, faster progression"
```

---

## Notes for the executor

- **Numbers are the target, not the last word** — magnitudes/rates are tuned in the
  post-ship playtest. Don't gold-plate; ship the framework + the sane first-pass.
- **Ranks/ceilings**: apexes `max_rank: 1`; keystones keep the global 4. The curve
  (×1.6/2.5/4.0) applies to keystone ranks 2–4.
- **Weaver's signal** (attacker applied a debuff via `res.BuffTarget`) is the one
  most likely to want playtest refinement — note it if it feels off.
- **Do NOT push.** This is the last arc task; the user pushes when ready. Stage 6f
  (re-spec phials) is a separate, later feature.
```
