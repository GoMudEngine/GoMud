# Player Mutation Migration (0.14.0) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate every existing player off the retired-41 mutations onto the new cluster graph — classify each account's play history into a cluster, wipe the old mutations, and grant a modest, regrowable cluster seed — shipping alongside the doc updates that teach the new system.

**Architecture:** A pure-Go classifier (`ClassifyPlayer`) reads a player's skills/companions/spellbook/willpower-training and returns a cluster; a versioned `0.14.0.go` migration (mirroring the proven `0.13.0.go` framework) globs the user saves, backs them up, runs dry-run-first, then classifies → wipes retired mutations → grants the cluster seed → version-stamps for idempotency. Ranks are provisional (coupled to the 6e balance pass). Developed and verified against **copies** of the 34 real prod accounts placed in the local users dir, then nuked.

**Tech Stack:** Go, YAML player saves, the `internal/migration` versioned framework, `go test ./...`, local boot smoke.

**Spec:** `docs/superpowers/specs/2026-07-11-mutation-migration-design.md` (decisions §1–7 locked; thresholds/ranks §8 fit here).

**Conventions (CLAUDE.md):** boot smoke via instance-save wipe + `go run .` (poll with `ping -n` spacing, kill `go.exe`); conventional commits w/ `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`; **do NOT push** (arc constraint). **The 34 prod files live in `_archive/prod-users/users/` — gitignored, NEVER commit them or their copies.**

**Player-save shape** (under `character:`): `skills: {name: level}`, `mutations: {id: rank}`, `spellbook: {spellid: rank}`, `companions: [ ... ]`, `stats.willpower.training: N`, `role: "admin"|""`.

---

## Reference A — cluster seed grants (spec §5 tiers, PROVISIONAL ranks)

| Classification | Seed granted into `character.mutations` |
|---|---|
| **Manifester** (meirok-class) | `symbiotic-bond: 1`, `hive-mind: 1` *(keystones, just short of brood-mother apex)* |
| **Ethereal** (Deios-class) | `ether-gland: 1`, `second-sight: 1` |
| **Ravener** (Saphira-class) | `rending-claws: 1`, `raptor-legs: 1` |
| **Colossus** | `dense-muscles: 1`, `titan-growth: 1` |
| **Stalker** | `padded-soles: 1`, `compound-eyes: 1` |
| **Zealot** | `commanding-presence: 1`, `zealous-conviction: 1` |
| **Chrysifier** (Duard-class crafter) | `provident-hands: 1`, `walking-chrysalis: 1` *(crafter keystones; apex still earned)* |
| **Generalist** (floor: crafters below Duard-class, test/newbie/low) | `keen-senses: 1` *(one Center entry keystone, rank 1)* |
| **admin** (Megalomania) | ALL cluster entry keystones (all-access, §6) — see Task 5 |

> Ranks are PROVISIONAL and retuned in 6e. Verify each id is a live mutation before use (all above are live post-nuke).

---

### Task 1: Signal extraction + threshold fitting

Produce a signals table for all 34 prod accounts and lock thresholds that reproduce spec §4.

**Files:**
- Create (scratch, NOT committed): `tools/analyze_players.py`

- [ ] **Step 1: Dump each account's classification signals**

Create `tools/analyze_players.py`:

```python
import glob, yaml, os
COMBAT = {"weapon-combat":"colossus","unarmed-combat":"ravener","ranged-combat":"stalker",
          "skullduggery":"stalker","rhetoric":"zealot"}
CRAFT = ["alchemy","blacksmithing","tailoring","cooking","jewelcrafting","enchanting","salvage"]
for f in sorted(glob.glob("_archive/prod-users/users/*.yaml")):
    d = yaml.safe_load(open(f, encoding="utf-8"))
    ch = d.get("character", d)
    sk = ch.get("skills", {}) or {}
    comps = ch.get("companions", []) or []
    spellbook = ch.get("spellbook", {}) or {}
    wil = (((ch.get("stats") or {}).get("willpower") or {}).get("training")) or 0
    role = ch.get("role","")
    topcombat = max(((v,COMBAT[k]) for k,v in sk.items() if k in COMBAT), default=(0,""))
    craftsum = sum(sk.get(c,0) for c in CRAFT)
    print(f"{ch.get('name','?'):16} role={role:6} comps={len(comps)} manif={sk.get('manifestation',0):3} "
          f"spellbook={len(spellbook):2} wilTrain={wil:3} topcombat={topcombat[0]:3}->{topcombat[1]:9} craftsum={craftsum}")
```
Run: `python tools/analyze_players.py`

- [ ] **Step 2: Fit thresholds to reproduce §4**

From the output, confirm these provisional thresholds land every account per spec §4
(meirok→Manifester, Deios→Ethereal, Saphira→Ravener, fyttyn/Duard/Oriana/pruuk→Chrysifier,
Megalomania→admin, the rest→Generalist). Record the final constants (adjust only if a
named account misclassifies):

```
CompanionManifesterMin   = 2      # >=2 companions -> Manifester
ManifestationSoloMin     = 25     # OR (>=1 companion AND manifestation >= this)
CasterSpellbookMin       = 5      # spellbook entries
CasterWillpowerTrainMin  = 50     # willpower training (Deios=88 clears)
CombatStandoutMin        = 25     # top combat skill (Saphira unarmed 27 clears)
CrafterSumMin            = 150    # summed crafting skill (multiple near-maxed)
```

- [ ] **Step 3: Commit the analysis tool** (thresholds are embedded in Task 2's code)

```bash
git add tools/analyze_players.py
git commit -m "tools: prod player signal-extraction for mutation-migration classification"
```

---

### Task 2: The `ClassifyPlayer` classifier + tests

**Files:**
- Create: `internal/migration/classify.go`
- Test: `internal/migration/classify_test.go`

- [ ] **Step 1: Write the failing tests** (the §4 anchor profiles)

```go
package migration

import "testing"

func TestClassifyPlayer(t *testing.T) {
	cases := []struct {
		name  string
		in    PlayerSignals
		want  string
	}{
		{"meirok-manifester", PlayerSignals{Companions: 2, Manifestation: 48}, "manifester"},
		{"solo-companion-manifester", PlayerSignals{Companions: 1, Manifestation: 40}, "manifester"},
		{"deios-ethereal", PlayerSignals{SpellbookDepth: 9, WillpowerTrain: 88}, "ethereal"},
		{"saphira-ravener", PlayerSignals{TopCombat: 27, TopCombatCluster: "ravener"}, "ravener"},
		{"crafter-chrysifier", PlayerSignals{CraftSum: 402, Companions: 1, TopCombat: 100, TopCombatCluster: "stalker"}, "chrysifier"},
		{"admin-allaccess", PlayerSignals{Role: "admin"}, "admin"},
		{"newbie-generalist", PlayerSignals{}, "generalist"},
		{"dabbler-generalist", PlayerSignals{TopCombat: 10, TopCombatCluster: "colossus"}, "generalist"},
	}
	for _, c := range cases {
		if got := ClassifyPlayer(c.in); got != c.want {
			t.Errorf("%s: ClassifyPlayer = %q, want %q", c.name, got, c.want)
		}
	}
}
```
NOTE the crafter case: skullduggery-100 is incidental (a crafter's search/skulldug), so
the crafter branch must be checked *before* the combat-standout branch fires on it — but
only when `CraftSum` clears and the combat skill is a *crafting-adjacent* one
(stalker via skullduggery). Encoded below.

- [ ] **Step 2: Run — confirm FAIL** (`ClassifyPlayer`/`PlayerSignals` undefined)

Run: `go test ./internal/migration/ -run TestClassifyPlayer -v` → FAIL (undefined).

- [ ] **Step 3: Implement `classify.go`**

```go
package migration

// PlayerSignals are the classification inputs extracted from a player save.
type PlayerSignals struct {
	Role             string
	Companions       int
	Manifestation    int
	SpellbookDepth   int
	WillpowerTrain   int
	TopCombat        int
	TopCombatCluster string // colossus|ravener|stalker|zealot
	CraftSum         int
}

// Thresholds fit against the 34 real prod accounts (Task 1) to reproduce spec §4.
const (
	companionManifesterMin  = 2
	manifestationSoloMin    = 25
	casterSpellbookMin      = 5
	casterWillpowerTrainMin = 50
	combatStandoutMin       = 25
	crafterSumMin           = 150
)

// ClassifyPlayer slots a player into a cluster per spec §3. Priority order:
// admin > Manifester > Ethereal > crafter-guard > combat-standout > Generalist.
func ClassifyPlayer(s PlayerSignals) string {
	if s.Role == "admin" {
		return "admin"
	}
	// 1. Companions -> Manifester.
	if s.Companions >= companionManifesterMin ||
		(s.Companions >= 1 && s.Manifestation >= manifestationSoloMin) {
		return "manifester"
	}
	// 2. Deep spellbook + mental training -> Ethereal.
	if s.SpellbookDepth >= casterSpellbookMin && s.WillpowerTrain >= casterWillpowerTrainMin {
		return "ethereal"
	}
	// 4-before-3 guard: a heavy crafter whose only "combat" standout is the
	// crafting-adjacent Stalker line (skullduggery/search) is a maker, not a rogue.
	crafter := s.CraftSum >= crafterSumMin
	combatStandout := s.TopCombat >= combatStandoutMin && s.TopCombatCluster != ""
	if crafter && (!combatStandout || s.TopCombatCluster == "stalker") {
		return "chrysifier"
	}
	// 3. Genuine combat standout.
	if combatStandout {
		return s.TopCombatCluster
	}
	// 5. Else Generalist.
	return "generalist"
}
```

- [ ] **Step 4: Run — confirm PASS**; **Step 5: Commit**

```bash
go test ./internal/migration/ -run TestClassifyPlayer -v   # PASS
git add internal/migration/classify.go internal/migration/classify_test.go
git commit -m "feat(migration): ClassifyPlayer — history -> cluster (thresholds fit to prod §4)"
```

---

### Task 3: Signal extraction from a save + seed-grant map

**Files:**
- Create: `internal/migration/grant.go`
- Test: `internal/migration/grant_test.go`

- [ ] **Step 1: Write the failing test**

```go
package migration

import "testing"

func TestSeedForCluster(t *testing.T) {
	if got := SeedForCluster("ravener"); got["rending-claws"] != 1 || got["raptor-legs"] != 1 {
		t.Errorf("ravener seed = %v", got)
	}
	if got := SeedForCluster("generalist"); len(got) != 1 || got["keen-senses"] != 1 {
		t.Errorf("generalist seed = %v", got)
	}
	if got := SeedForCluster("admin"); len(got) < 9 { // all-access
		t.Errorf("admin seed too small: %d", len(got))
	}
}
```

- [ ] **Step 2: Run — FAIL. Step 3: Implement `grant.go`**

```go
package migration

// SeedForCluster returns the provisional mutation seed granted for a
// classification (spec §5 tiers; ranks retuned in 6e). Empty for unknown.
func SeedForCluster(cluster string) map[string]int {
	switch cluster {
	case "manifester":
		return map[string]int{"symbiotic-bond": 1, "hive-mind": 1}
	case "ethereal":
		return map[string]int{"ether-gland": 1, "second-sight": 1}
	case "ravener":
		return map[string]int{"rending-claws": 1, "raptor-legs": 1}
	case "colossus":
		return map[string]int{"dense-muscles": 1, "titan-growth": 1}
	case "stalker":
		return map[string]int{"padded-soles": 1, "compound-eyes": 1}
	case "zealot":
		return map[string]int{"commanding-presence": 1, "zealous-conviction": 1}
	case "chrysifier":
		return map[string]int{"provident-hands": 1, "walking-chrysalis": 1}
	case "generalist":
		return map[string]int{"keen-senses": 1}
	case "admin":
		// All-access: one entry keystone per cluster so the admin can test everything.
		return map[string]int{
			"dense-muscles": 1, "thick-hide": 1, "rending-claws": 1, "padded-soles": 1,
			"ether-gland": 1, "symbiotic-bond": 1, "commanding-presence": 1,
			"sticky-secretion": 1, "quicksilver-nerves": 1, "provident-hands": 1, "keen-senses": 1,
		}
	}
	return nil
}
```

- [ ] **Step 4: Run — PASS. Step 5: Commit**

```bash
git add internal/migration/grant.go internal/migration/grant_test.go
git commit -m "feat(migration): cluster seed-grant map (provisional ranks, admin all-access)"
```

---

### Task 4: The `0.14.0.go` migration (backup + dry-run + apply)

**Files:**
- Create: `internal/migration/0.14.0.go`
- Modify: `internal/migration/migration.go` (add the 0.14.0 block)

- [ ] **Step 1: Write `0.14.0.go`** (mirrors `0.13.0.go`'s glob/parse/write loop)

```go
package migration

import (
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// retiredMutations is the retired-41 set wiped from every player (matches the
// data-side nuke in the NPC migration). Any id here is removed from a save.
var retiredMutations = map[string]bool{
	"adrenaline-surge": true, "bioluminescence": true, "blinding-flash": true,
	"blinding-spit": true, "brazen-resolve": true, "camo-skin": true, "clawed-hands": true,
	"cold-blooded": true, "elongated-limbs": true, "extra-legs": true, "fast-reflexes": true,
	"hasted": true, "healing-gel": true, "heightened-senses": true, "incorporeal": true,
	"infrared-vision": true, "iron-constitution": true, "keen-eyes": true, "large": true,
	"magical-resistance": true, "night-vision": true, "pacifism-aura": true,
	"pheromone-glands": true, "photosynthetic-skin": true, "psychic-resistance": true,
	"rapid-metabolism": true, "regenerative-tissue": true, "sixth-sense": true,
	"skilled": true, "small": true, "sonic-shout": true, "talented": true,
	"tough-skin": true, "toxic-bite": true,
}

// migrate_ReclassifyPlayerMutations wipes retired mutations and grants a cluster
// seed per the player's play history. DryRun logs intended changes with no writes.
// Idempotent via the `mutation_migrated` marker under character.
func migrate_ReclassifyPlayerMutations(dryRun bool) error {
	c := configs.GetConfig()
	usersGlob := filepath.Join(string(c.FilePaths.DataFiles), "users", "*.yaml")
	matches, err := filepath.Glob(usersGlob)
	if err != nil {
		return err
	}
	mode := "APPLY"
	if dryRun {
		mode = "DRY-RUN"
	}
	mudlog.Info("Migration 0.14.0", "message", "Reclassifying player mutations onto the cluster graph", "mode", mode)

	for _, path := range matches {
		if filepath.Base(path) == "users.idx" {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc map[string]interface{}
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			mudlog.Warn("Migration 0.14.0", "file", filepath.Base(path), "error", err)
			continue
		}
		ch, _ := doc["character"].(map[interface{}]interface{})
		if ch == nil {
			continue
		}
		if ch["mutation_migrated"] == true { // idempotency
			continue
		}
		sig := extractSignals(ch)
		cluster := ClassifyPlayer(sig)
		seed := SeedForCluster(cluster)

		newMuts := map[string]int{}
		for id, rank := range seed {
			newMuts[id] = rank
		}
		mudlog.Info("Migration 0.14.0", "user", filepath.Base(path), "class", cluster, "seed", len(newMuts))
		if dryRun {
			continue
		}
		ch["mutations"] = newMuts
		ch["mutationprogress"] = 0.0
		ch["mutation_migrated"] = true
		out, err := yaml.Marshal(doc)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, out, 0644); err != nil {
			return err
		}
	}
	return nil
}
```

Add `extractSignals` in the same file (reads the `map[interface{}]interface{}` YAML shape):

```go
var combatClusterOf = map[string]string{
	"weapon-combat": "colossus", "unarmed-combat": "ravener",
	"ranged-combat": "stalker", "skullduggery": "stalker", "rhetoric": "zealot",
}
var craftSkills = []string{"alchemy", "blacksmithing", "tailoring", "cooking", "jewelcrafting", "enchanting", "salvage"}

func extractSignals(ch map[interface{}]interface{}) PlayerSignals {
	s := PlayerSignals{}
	if r, ok := ch["role"].(string); ok {
		s.Role = r
	}
	skills := toIntMap(ch["skills"])
	s.Manifestation = skills["manifestation"]
	for skill, cluster := range combatClusterOf {
		if lv := skills[skill]; lv > s.TopCombat {
			s.TopCombat, s.TopCombatCluster = lv, cluster
		}
	}
	for _, cs := range craftSkills {
		s.CraftSum += skills[cs]
	}
	s.SpellbookDepth = len(toIntMap(ch["spellbook"]))
	if comps, ok := ch["companions"].([]interface{}); ok {
		s.Companions = len(comps)
	}
	if stats, ok := ch["stats"].(map[interface{}]interface{}); ok {
		if wil, ok := stats["willpower"].(map[interface{}]interface{}); ok {
			if t, ok := wil["training"].(int); ok {
				s.WillpowerTrain = t
			}
		}
	}
	return s
}

// toIntMap coerces a YAML map[interface{}]interface{} of {string: int} to Go.
func toIntMap(v interface{}) map[string]int {
	out := map[string]int{}
	m, ok := v.(map[interface{}]interface{})
	if !ok {
		return out
	}
	for k, val := range m {
		ks, _ := k.(string)
		if iv, ok := val.(int); ok {
			out[ks] = iv
		}
	}
	return out
}
```

- [ ] **Step 2: Wire it into `migration.go`** — add after the 0.13.0 block in `doAllMigrations`:

```go
	if lastConfigVersion.IsOlderThan(version.New(0, 14, 0)) {
		// Player mutation migration: reclassify onto the cluster graph.
		// Backup is handled by the framework's backup.go before doAllMigrations.
		if err := migrate_ReclassifyPlayerMutations(false); err != nil {
			return err
		}
	}
```

- [ ] **Step 3: Build + confirm the framework backs up users before running**

Run: `go build ./... ` → exit 0. Read `internal/migration/backup.go` and confirm
`users/` is tarred before `doAllMigrations`; if not, add a backup call at the top
of the 0.14.0 block. (Spec §7 safety — non-negotiable.)

- [ ] **Step 4: Commit**

```bash
git add internal/migration/0.14.0.go internal/migration/migration.go
git commit -m "feat(migration): 0.14.0 player mutation reclassification (wipe retired, grant cluster seed, idempotent)"
```

---

### Task 5: Acquisition-rate retune

**Files:**
- Modify: `_datafiles/config.yaml` (the `Balance.Mutation*` knobs)

- [ ] **Step 1: Review + set the knobs for the new graph**

The graph is deeper than the retired-41 flat pool, so first-mutation cost and the
per-cluster affinity gate want a light retune. Set (provisional, retuned in 6e):

```yaml
# _datafiles/config.yaml (Balance:)
MutationBaseProgress: 60        # was 50 — slightly slower first bloom on the richer graph
MutationMaxCount: 6             # was 5 — the graph supports a deeper build
MutationAffinityPerRarity: 1.0  # unchanged — the rarity gate is already tuned per content §7
```
Leave `MutationProgressScale`, `MutationProgressGainPerRound`, `MobMutationRate` as-is.

- [ ] **Step 2: Boot smoke + commit**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/pm_boot.log 2>&1 &   # wait Server Ready, no panic; kill
git add _datafiles/config.yaml
git commit -m "balance(mutations): light player acquisition-rate retune for the cluster graph (provisional)"
```

---

### Task 6: Mutation helpfiles + `help mutations` framing

**Files:**
- Modify: `_datafiles/world/dogmud/templates/help/mutations.template` (the overview)
- Verify: per-mutation templates already exist (helpfile-completeness test)

- [ ] **Step 1: Rewrite `help mutations`** to teach the new system

Open `_datafiles/world/dogmud/templates/help/mutations.template` and replace the
body with an explanation of: the cluster ring (9 clusters + poles), the Generalist
center, drift/affinity (mutations grow toward the skills you use), prereq spines +
apexes, and the Chrysifier crafter path. Wrap at 80 cols; no hard numbers. (Full
prose in the commit — write it against the live cluster list.)

- [ ] **Step 2: Confirm helpfile-completeness still green + commit**

```bash
go test ./internal/devtools/ -run TestHelpFileCompleteness_Mutations   # PASS
git add _datafiles/world/dogmud/templates/help/mutations.template
git commit -m "docs(help): reframe 'help mutations' around the cluster graph"
```

---

### Task 7: `help dogmud` + repo README

**Files:**
- Modify: `_datafiles/world/dogmud/templates/help/dogmud.template` (verify exact name via `ls`)
- Modify: `README.md`

- [ ] **Step 1: Refresh `help dogmud`** to headline the mutation-graph identity system as a defining feature (a paragraph pointing players at `help mutations`).

- [ ] **Step 2: Update `README.md`** project overview — describe the cluster/bridge/Center mutation graph + use-based progression as a defining DOGMud feature (replace/augment any stale mutation description).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/dogmud.template README.md
git commit -m "docs: help dogmud + README reflect the mutation-graph redesign"
```

---

### Task 8: Verify against copied prod players, then nuke

The migration is DEVELOPED and VERIFIED against copies of real accounts, per the
user's testing strategy. **Copies live in the local `users/` dir; they are
gitignored there and MUST be deleted after.**

**Files:** temporary copies in `_datafiles/world/dogmud/users/` (never committed).

- [ ] **Step 1: Copy representative accounts into the local users dir**

Pick a heavy veteran (meirok), the heaviest crafter (Duard), a caster (Deios), a
martial (Saphira), one early/light player, and confirm a freshie path. Identify
their prod filenames from Task 1's output, then:

```bash
cd "c:/Users/Calabe Davis/workspace/DOGMud"
mkdir -p _datafiles/world/dogmud/users
# copy the chosen prod files (example ids — use the real ones from Task 1):
for id in <meirok> <duard> <deios> <saphira> <early>; do
  cp "_archive/prod-users/users/$id.yaml" "_datafiles/world/dogmud/users/mtest_$id.yaml"
done
ls _datafiles/world/dogmud/users/mtest_*.yaml
```

- [ ] **Step 2: DRY-RUN first — verify classifications match §4 with zero writes**

Temporarily flip the 0.14.0 call to `migrate_ReclassifyPlayerMutations(true)` (dry-run),
force the migration by lowering the stored config version, boot, and read the log:

```bash
grep "Migration 0.14.0" /tmp/pm_boot.log   # class per user
```
Confirm meirok→manifester, Duard→chrysifier, Deios→ethereal, Saphira→ravener,
early→generalist. If any is wrong, fix the classifier/thresholds (Task 2) and repeat.
Then restore the call to `migrate_ReclassifyPlayerMutations(false)`.

- [ ] **Step 3: APPLY — run for real against the copies, inspect results**

Boot with writes enabled; then inspect a migrated copy:

```bash
grep -A6 "  mutations:" _datafiles/world/dogmud/users/mtest_<meirok>.yaml
grep "mutation_migrated" _datafiles/world/dogmud/users/mtest_<meirok>.yaml
```
Confirm: retired mutations gone, cluster seed present, `mutation_migrated: true`,
`mutationprogress: 0`. Also confirm re-running the migration skips them (idempotent).

- [ ] **Step 4: Freshie check** — create a brand-new character via the normal
new-player flow (or an empty save), run the migration, confirm it classifies
Generalist and grants `keen-senses: 1` with no errors.

- [ ] **Step 5: NUKE the test copies**

```bash
rm -f _datafiles/world/dogmud/users/mtest_*.yaml
ls _datafiles/world/dogmud/users/ | grep mtest   # expect nothing
```

- [ ] **Step 6: Full suite + boot + commit (code/docs only — NO user saves)**

```bash
go test ./... 2>&1 | grep -cE "^ok"   # 87
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/pm_final_boot.log 2>&1 &   # Server Ready, no panic; kill
git status --porcelain _datafiles/world/dogmud/users/   # MUST be empty — no user saves staged
# PATCH_NOTES entry for the player migration:
git add PATCH_NOTES.md
git commit -m "docs(patch-notes): player mutation migration (0.14.0) — reclassify onto the cluster graph"
```

---

## Notes for the executor

- **Never commit user saves** (`_datafiles/world/dogmud/users/`) or the prod archive
  (`_archive/prod-users/`). Verify with `git status` before every commit.
- **Ranks are PROVISIONAL** (spec §5/§8) — retuned in 6e with the NPC kits.
- **Dry-run before apply** is non-negotiable (spec §7) — it's how we verify against §4.
- **Do NOT push.** Arc constraint holds until 6e is also done.
- **Re-spec phials** (spec's "directional re-spec" idea) are a *separate* feature —
  out of scope here; the seeds are regrowable through play. Note for a follow-on.
- The **actual prod migration** (running 0.14.0 against the live droplet saves) is the
  user's deploy step, not part of this dev work — this plan makes it correct + verified.
```
