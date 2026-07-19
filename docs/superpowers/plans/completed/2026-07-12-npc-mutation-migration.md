# NPC Mutation Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Nuke the ~34 legacy leftover mutations that leak into every acquisition/bloom/tutorial grant, and clean up all the mob/behavior/Go references they leave dangling, so the mutation graph the player migration ships against is exactly the designed set (9 clusters + 9 bridges + Chrysifier + ~10 Center enablers).

**Architecture:** Pure subtraction + reference-repointing. A guard test locks the "no legacy zero-cluster mutation" invariant first (TDD anchor); then delete the YAMLs/help/commands, remove the two Go consumers (`adrenaline-surge` combat block, active-ability dispatch), scrub dangling `conflicts:`, repoint mob `spawnmutations`, and re-base the mob archetype-shift feature off cluster/pole instead of the now-empty `archetype_pull` field. Nothing new is built; the engine is already graph-native.

**Tech Stack:** Go 1.x, YAML data files, `go test ./...`, local server boot smoke.

**Spec:** `docs/superpowers/specs/2026-07-12-npc-mutation-migration-design.md`

**Working conventions (from CLAUDE.md):**
- Boot smoke: `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*` then `go run .`, watch for `Server Ready` with **no** `panic:`/`ERROR: PANIC`.
- Commit style: conventional commits; footer `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- Do NOT push (arc-wide constraint: no push until the whole Chrysalis arc is done).

**The delete set (34 legacy mutations)** — every `_datafiles/world/dogmud/mutations/*.yaml` with **no `clusters:` field** whose id is NOT one of the 10 survivors below:

```
adrenaline-surge bioluminescence blinding-flash blinding-spit brazen-resolve
camo-skin clawed-hands cold-blooded elongated-limbs extra-legs fast-reflexes
hasted healing-gel heightened-senses incorporeal infrared-vision iron-constitution
keen-eyes large magical-resistance night-vision pacifism-aura pheromone-glands
photosynthetic-skin psychic-resistance rapid-metabolism regenerative-tissue
sixth-sense skilled small sonic-shout talented tough-skin toxic-bite
```

**The 10 survivors (KEEP, never touch)** — intended Center enablers:

```
hollow-bones prehensile-tail keen-senses rapid-healing thick-coat
tremorsense precognition spiracle-lungs winged-flight tail
```

---

### Task 1: Guard test — no legacy zero-cluster mutations

Locks the core invariant before any deletion, and prevents regressions (someone re-adding a zero-cluster mutation later).

**Files:**
- Create: `internal/mutations/legacy_pool_test.go`

- [ ] **Step 1: Write the failing test**

```go
package mutations

import "testing"

// intendedCenterEnablers is the closed set of zero-cluster ("Center" /
// always-eligible) mutations the graph is allowed to carry. Any OTHER
// zero-cluster mutation is a legacy leftover that leaks into every drift/
// bloom/tutorial grant (affinityFor returns MaxFloat64 for zero-cluster).
var intendedCenterEnablers = map[string]bool{
	"hollow-bones": true, "prehensile-tail": true, "keen-senses": true,
	"rapid-healing": true, "thick-coat": true, "tremorsense": true,
	"precognition": true, "spiracle-lungs": true, "winged-flight": true,
	"tail": true,
}

func TestNoLegacyZeroClusterMutations(t *testing.T) {
	for _, spec := range AllSpecs() {
		if len(spec.Clusters) == 0 && !intendedCenterEnablers[spec.MutationId] {
			t.Errorf("legacy zero-cluster mutation still present (must be deleted or cluster-tagged): %s", spec.MutationId)
		}
	}
}
```

- [ ] **Step 2: Run it and confirm it FAILS**

Run: `go test ./internal/mutations/ -run TestNoLegacyZeroClusterMutations -v`
Expected: FAIL, listing ~34 ids (adrenaline-surge, bioluminescence, …). This proves the leak exists.

- [ ] **Step 3: Commit the guard**

```bash
git add internal/mutations/legacy_pool_test.go
git commit -m "test(mutations): guard against legacy zero-cluster pool leak (currently failing)"
```

(The failing test is intentionally committed; Task 2 turns it green.)

---

### Task 2: Delete the 34 legacy mutation YAMLs + help templates

**Files:**
- Delete: `_datafiles/world/dogmud/mutations/<id>.yaml` for each id in the delete set
- Delete: `_datafiles/world/dogmud/templates/help/<id>.template` for each (if present)

- [ ] **Step 1: Delete the mutation YAMLs and their help templates**

```bash
cd "c:/Users/Calabe Davis/workspace/DOGMud"
for id in adrenaline-surge bioluminescence blinding-flash blinding-spit brazen-resolve \
  camo-skin clawed-hands cold-blooded elongated-limbs extra-legs fast-reflexes \
  hasted healing-gel heightened-senses incorporeal infrared-vision iron-constitution \
  keen-eyes large magical-resistance night-vision pacifism-aura pheromone-glands \
  photosynthetic-skin psychic-resistance rapid-metabolism regenerative-tissue \
  sixth-sense skilled small sonic-shout talented tough-skin toxic-bite; do
  rm -f "_datafiles/world/dogmud/mutations/$id.yaml"
  rm -f "_datafiles/world/dogmud/templates/help/$id.template"
done
```

- [ ] **Step 2: Confirm the guard test now PASSES**

Run: `go test ./internal/mutations/ -run TestNoLegacyZeroClusterMutations -v`
Expected: PASS (no zero-cluster leftovers besides the 10 survivors).

- [ ] **Step 3: Confirm the helpfile-completeness test still PASSES**

Run: `go test ./internal/devtools/ -run TestHelpFileCompleteness_Mutations -v`
Expected: PASS (it checks one template per *surviving* YAML; deleted pairs are gone together).

- [ ] **Step 4: Commit**

```bash
git add -A _datafiles/world/dogmud/mutations _datafiles/world/dogmud/templates/help
git commit -m "content(mutations): delete 34 legacy leftover mutations (pool leak fix)"
```

Note: the build will still compile and boot at this point — the Go actives and mob `spawnmutations` reference these ids as *strings*, which simply resolve to nothing until Tasks 3–7 clean them up. Each subsequent task keeps build+boot green.

---

### Task 3: Delete the 6 active-ability commands + dispatch + orphan buffs

The 6 legacy actives (`blinding-flash`, `blinding-spit`, `healing-gel`, `pacifism-aura`, `sonic-shout`, `toxic-bite`) are cooldown-dominated DOA. Remove their command code and dispatch registrations.

**Files:**
- Delete: `internal/usercommands/mutation_blinding_flash.go`, `mutation_blinding_spit.go`, `mutation_healing_gel.go`, `mutation_pacifism_aura.go`, `mutation_sonic_shout.go`, `mutation_toxic_bite.go`
- Delete: `internal/actions/mutation_blinding_flash.go`, `mutation_blinding_spit.go`, `mutation_healing_gel.go`, `mutation_pacifism_aura.go`, `mutation_sonic_shout.go`, `mutation_toxic_bite.go`
- Modify: `internal/behaviortree/actions_mutation.go` (remove 6 map entries)
- Modify: `internal/mobcommands/mobcommands.go` (remove 6 map entries)
- Possibly delete: orphaned buff YAMLs under `_datafiles/world/dogmud/buffs/`

- [ ] **Step 1: Identify any dedicated buffs the actives apply**

```bash
grep -rnE "AddBuff\(|buffs\.|BuffId|[0-9]+, *(false|true)" \
  internal/actions/mutation_blinding_flash.go internal/actions/mutation_blinding_spit.go \
  internal/actions/mutation_healing_gel.go internal/actions/mutation_pacifism_aura.go \
  internal/actions/mutation_sonic_shout.go internal/actions/mutation_toxic_bite.go \
  internal/usercommands/mutation_*.go
```
For each buff id found, check whether anything ELSE references it:
`grep -rn "AddBuff(<id>" internal/ _datafiles | grep -v mutation_<the active>`
If it is used only by the deleted active, delete `_datafiles/world/dogmud/buffs/<id>-*.yaml`. If shared, leave it.

- [ ] **Step 2: Delete the command source files**

```bash
rm -f internal/usercommands/mutation_blinding_flash.go internal/usercommands/mutation_blinding_spit.go \
      internal/usercommands/mutation_healing_gel.go internal/usercommands/mutation_pacifism_aura.go \
      internal/usercommands/mutation_sonic_shout.go internal/usercommands/mutation_toxic_bite.go
rm -f internal/actions/mutation_blinding_flash.go internal/actions/mutation_blinding_spit.go \
      internal/actions/mutation_healing_gel.go internal/actions/mutation_pacifism_aura.go \
      internal/actions/mutation_sonic_shout.go internal/actions/mutation_toxic_bite.go
```

- [ ] **Step 3: Remove the behaviortree dispatch entries**

In `internal/behaviortree/actions_mutation.go`, delete these 4 entries from the non-target map and 2 from the single-target map:

```go
// non-target map — DELETE these lines:
"blinding-flash": actions.TriggerBlindingFlash,
"healing-gel":    actions.TriggerHealingGel,
"pacifism-aura":  actions.TriggerPacifismAura,
"sonic-shout":    actions.TriggerSonicShout,
// single-target map — DELETE these lines:
"blinding-spit": actions.TriggerBlindingSpit,
"toxic-bite":    actions.TriggerToxicBite,
```

If either map becomes empty, keep the (now empty) map literal so the dispatch functions still compile; do not delete the surrounding `actTryMutationActive` / `actTryMutationActiveAtTarget` machinery (it is generic and may serve future actives).

- [ ] **Step 4: Remove the mobcommands registrations**

In `internal/mobcommands/mobcommands.go`, delete these 6 map entries and their now-unused command functions (`BlindingFlash`, `BlindingSpit`, `HealingGel`, `PacifismAura`, `SonicShout`, `ToxicBite`) wherever they are defined:

```go
"blinding-flash": {BlindingFlash, false},
"blinding-spit":  {BlindingSpit, false},
"healing-gel":    {HealingGel, false},
"pacifism-aura":  {PacifismAura, false},
"sonic-shout":    {SonicShout, false},
"toxic-bite":     {ToxicBite, false},
```

- [ ] **Step 5: Build and fix any remaining references**

Run: `go build ./... 2>&1 | head`
Expected: exit 0. If the build reports an undefined `Trigger*` / command symbol, grep for it (`grep -rn "TriggerBlindingFlash" internal/`) and remove that reference too. Repeat until clean.

- [ ] **Step 6: Full suite (delete any tests for the removed actives) + commit**

Run: `go test ./... 2>&1 | grep -E "FAIL|ok .*mobcommands|ok .*behaviortree"`
If a test file targets a deleted active (e.g. `internal/actions/mutation_toxic_bite_test.go`), delete it. Re-run until green.

```bash
git add -A internal/usercommands internal/actions internal/behaviortree internal/mobcommands _datafiles/world/dogmud/buffs
git commit -m "refactor(mutations): remove 6 cooldown-dominated active-ability mutation commands"
```

---

### Task 4: Remove the adrenaline-surge combat consumer

`adrenaline-surge` was the only `conditional_damage_low_hp` mutation. Deleting it orphans an id-named combat block.

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_unified.go:346-356` (delete the Adrenaline Surge block)
- Modify: `internal/mutations/mutations.go` (delete `IsAdrenalSurgeActive`)

- [ ] **Step 1: Delete the combat block**

In `internal/hooks/NewRound_DoCombat_unified.go`, remove this entire block:

```go
	// Adrenaline Surge (mutation).
	if mutations.IsAdrenalSurgeActive(atkChar.Mutations, atkChar.Health, atkChar.HealthMax.Value) {
		if surgeBonus := mutations.GetAdrenalSurgeBonus(atkChar.Mutations); surgeBonus > 0 {
			bonusDmg := int(math.Round(float64(res.DamageToTarget) * surgeBonus))
			if bonusDmg < 1 {
				bonusDmg = 1
			}
			defChar.Health -= bonusDmg
			res.DamageToTarget += bonusDmg
		}
	}
```

- [ ] **Step 2: Delete the id-named helper**

In `internal/mutations/mutations.go`, delete `IsAdrenalSurgeActive` (it hardcodes `"adrenaline-surge"`). Leave the generic `GetAdrenalSurgeBonus` (it is a `sumEffects` reader — inert, returns 0 with no owner) and the generic `conditional_damage_low_hp` `describe.go` case in place; they are harmless and cheap to keep.

- [ ] **Step 3: Build (fix `math` import if now unused)**

Run: `go build ./... 2>&1 | head`
Expected: exit 0. If `internal/hooks/NewRound_DoCombat_unified.go` now reports `"math" imported and not used`, confirm no other `math.` call remains in the file; if truly unused, remove the import — but it is used by the Conviction-Surge block just above, so it should stay.

- [ ] **Step 4: Full suite + commit**

Run: `go test ./internal/hooks/ ./internal/mutations/ 2>&1 | tail`
Expected: ok.

```bash
git add internal/hooks/NewRound_DoCombat_unified.go internal/mutations/mutations.go
git commit -m "refactor(combat): drop adrenaline-surge consumer (mutation deleted)"
```

---

### Task 5: Scrub dangling `conflicts:` references in surviving mutations

Deleted ids that appear in a surviving mutation's `conflicts:` list must be removed, or the graph carries references to nonexistent mutations.

**Files:**
- Modify: surviving `_datafiles/world/dogmud/mutations/*.yaml` with `conflicts:` entries pointing at the delete set

- [ ] **Step 1: Find every dangling conflict**

```bash
cd "c:/Users/Calabe Davis/workspace/DOGMud"
DELSET="adrenaline-surge|bioluminescence|blinding-flash|blinding-spit|brazen-resolve|camo-skin|clawed-hands|cold-blooded|elongated-limbs|extra-legs|fast-reflexes|hasted|healing-gel|heightened-senses|incorporeal|infrared-vision|iron-constitution|keen-eyes|large|magical-resistance|night-vision|pacifism-aura|pheromone-glands|photosynthetic-skin|psychic-resistance|rapid-metabolism|regenerative-tissue|sixth-sense|skilled|small|sonic-shout|talented|tough-skin|toxic-bite"
grep -rlE "^\s*-\s*($DELSET)\s*$" _datafiles/world/dogmud/mutations/*.yaml
grep -rnE "conflicts:.*($DELSET)" _datafiles/world/dogmud/mutations/*.yaml
```
Known hits to expect: `extra-arms` (→ `clawed-hands`, `elongated-limbs`), `chameleon-skin` (→ `bioluminescence`), `tail` (→ `small`).

- [ ] **Step 2: Remove each dangling conflict line**

For every file the grep surfaced, delete only the `- <deleted-id>` lines under `conflicts:` (keep conflicts that point at survivors). Example — `_datafiles/world/dogmud/mutations/extra-arms.yaml`:

```yaml
# BEFORE
conflicts:
  - clawed-hands
  - elongated-limbs
# AFTER: remove the block entirely if it becomes empty, or leave remaining survivor-conflicts.
```

`_datafiles/world/dogmud/mutations/chameleon-skin.yaml`: remove `- bioluminescence` (keep `- thick-hide`, a survivor).
`_datafiles/world/dogmud/mutations/tail.yaml`: remove `- small` (keep `- prehensile-tail`, a survivor).

- [ ] **Step 3: Re-grep to confirm zero dangling conflicts**

```bash
grep -rnE "conflicts:.*($DELSET)|^\s*-\s*($DELSET)\s*$" _datafiles/world/dogmud/mutations/*.yaml
```
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mutations
git commit -m "content(mutations): scrub conflicts pointing at deleted legacy mutations"
```

---

### Task 6: Repoint mob `spawnmutations` to surviving equivalents

38 mob templates carry `spawnmutations:`. Repoint each deleted id per the map; drop entries with no meaningful equivalent.

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/**/*.yaml` with `spawnmutations:` referencing deleted ids

**Repoint map:**

| Deleted id | Action |
|---|---|
| `large` | → `titan-growth` |
| `fast-reflexes` | → `keen-senses` |
| `keen-eyes` | → `keen-senses` |
| `iron-constitution` | → `titan-growth` |
| `tough-skin` | → `thick-hide` |
| `sixth-sense` | → `keen-senses` |
| `regenerative-tissue` | → `rapid-healing` |
| `camo-skin` | → `padded-soles` |
| `chameleon-skin` (now the Stalker apex) | → `padded-soles` |
| `psychic-resistance`, `magical-resistance`, `night-vision`, `cold-blooded`, `rapid-metabolism` | drop the id from the list |

- [ ] **Step 1: List every mob file needing a repoint**

```bash
cd "c:/Users/Calabe Davis/workspace/DOGMud"
grep -rlnE "spawnmutations:.*(large|fast-reflexes|keen-eyes|iron-constitution|tough-skin|sixth-sense|regenerative-tissue|camo-skin|chameleon-skin|psychic-resistance|magical-resistance|night-vision|cold-blooded|rapid-metabolism)" _datafiles/world/dogmud/mobs/
```

- [ ] **Step 2: Edit each file per the map**

For each file, edit the `spawnmutations: [...]` list: substitute per the map, or remove the id (and any resulting empty list → delete the whole `spawnmutations:` line). If a substitution would duplicate an id already in the list, just drop the deleted id. Example:

```yaml
# BEFORE
spawnmutations: [thick-hide, cold-blooded]
# AFTER (cold-blooded dropped)
spawnmutations: [thick-hide]

# BEFORE
spawnmutations: [regenerative-tissue, magical-resistance, psychic-resistance, thick-hide, brazen-resolve]
# AFTER (regenerative-tissue→rapid-healing; magical/psychic-resistance dropped; brazen-resolve dropped)
spawnmutations: [rapid-healing, thick-hide]

# BEFORE (highland stalker cat)
spawnmutations: [chameleon-skin]
# AFTER
spawnmutations: [padded-soles]
```
Note `brazen-resolve` is also a deleted id (drop it wherever it appears in a spawnmutations list).

- [ ] **Step 3: Confirm no deleted id remains in any spawnmutations**

```bash
grep -rnE "spawnmutations:.*(adrenaline-surge|bioluminescence|blinding-flash|blinding-spit|brazen-resolve|camo-skin|clawed-hands|cold-blooded|elongated-limbs|extra-legs|fast-reflexes|hasted|healing-gel|heightened-senses|incorporeal|infrared-vision|iron-constitution|keen-eyes|large|magical-resistance|night-vision|pacifism-aura|pheromone-glands|photosynthetic-skin|psychic-resistance|rapid-metabolism|regenerative-tissue|sixth-sense|skilled|small|sonic-shout|talented|tough-skin|toxic-bite)" _datafiles/world/dogmud/mobs/
```
Expected: no output.

- [ ] **Step 4: Boot smoke (mob templates load) + commit**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/npcmig_boot.log 2>&1 &
```
Wait for `Server Ready` (no `panic:`) in `/tmp/npcmig_boot.log`, then kill the server.

```bash
git add _datafiles/world/dogmud/mobs
git commit -m "content(mobs): repoint spawnmutations off deleted legacy mutations"
```

---

### Task 7: Re-base archetype-shift on clusters/pole

Preserve the "mob visibly re-archetypes as it mutates" behavior, but derive the pull from the new graph instead of the now-empty `archetype_pull` field.

**Files:**
- Modify: `internal/mutations/mutations.go` (remove `ArchetypePull` field)
- Modify: `internal/behaviortree/archetype_shift.go` (add cluster map; rewrite `strongestArchetypePull`; delete `validateArchetypePulls`/`ValidateArchetypePulls`)
- Modify: wherever `ValidateArchetypePulls()` is called at boot (remove the call)
- Modify: `internal/behaviortree/archetype_shift_test.go` (retarget to cluster-derived pull)

- [ ] **Step 1: Update the test first (TDD)**

Replace the pull-source assertions in `internal/behaviortree/archetype_shift_test.go`. The existing `TestValidateArchetypePullsCore` (which validated the YAML field) is deleted; add a cluster-map test:

```go
func TestArchetypeForSpec(t *testing.T) {
	cases := []struct {
		clusters []string
		want     string
	}{
		{[]string{"colossus"}, "tank_taunter"},
		{[]string{"ironhide"}, "tank_taunter"},
		{[]string{"ravener"}, "predator"},
		{[]string{"stalker"}, "ambusher"},
		{[]string{"ethereal"}, "pure_caster"},
		{[]string{"manifester"}, "defensive_caster"},
		{[]string{"zealot"}, "defensive_caster"},
		{[]string{"weaver"}, ""},   // hybrid → no pull
		{[]string{"trickster"}, ""},
		{[]string{"chrysifier"}, ""},
		{nil, ""}, // zero-cluster / Center → no pull
		{[]string{"ironhide", "zealot"}, "tank_taunter"}, // bridge: first mapping wins
	}
	for _, c := range cases {
		spec := &mutations.MutationSpec{Clusters: c.clusters}
		if got := archetypeForSpec(spec); got != c.want {
			t.Errorf("archetypeForSpec(%v) = %q, want %q", c.clusters, got, c.want)
		}
	}
}

func TestStrongestArchetypePull_ClusterDerived(t *testing.T) {
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"thick-hide":  {MutationId: "thick-hide", Rarity: 3, Clusters: []string{"ironhide"}},
		"ether-gland": {MutationId: "ether-gland", Rarity: 4, Clusters: []string{"ethereal"}},
		"keen-senses": {MutationId: "keen-senses", Rarity: 3}, // zero-cluster → no pull
	})
	defer cleanup()
	// Rarest owned mutation with a mapping wins: ether-gland (r4) > thick-hide (r3).
	got := strongestArchetypePull(map[string]int{"thick-hide": 1, "ether-gland": 1, "keen-senses": 1})
	if got != "pure_caster" {
		t.Fatalf("strongestArchetypePull = %q, want pure_caster", got)
	}
}
```

- [ ] **Step 2: Run the new test — confirm it FAILS to compile/pass**

Run: `go test ./internal/behaviortree/ -run "TestArchetypeForSpec|TestStrongestArchetypePull_ClusterDerived" -v`
Expected: FAIL (undefined `archetypeForSpec`; old `strongestArchetypePull` still reads `ArchetypePull`).

- [ ] **Step 3: Add the cluster→archetype map and `archetypeForSpec`**

In `internal/behaviortree/archetype_shift.go`, add near the other package vars:

```go
// clusterArchetype maps a graph cluster to the behavior archetype a mob
// drifts toward as it acquires that cluster's mutations. Clusters absent
// here (weaver, trickster, chrysifier) and zero-cluster/Center mutations
// produce no pull. TO-whitelist targets only.
var clusterArchetype = map[string]string{
	"colossus":   "tank_taunter",
	"ironhide":   "tank_taunter",
	"ravener":    "predator",
	"stalker":    "ambusher",
	"ethereal":   "pure_caster",
	"manifester": "defensive_caster",
	"zealot":     "defensive_caster",
}

// archetypeForSpec returns the archetype a mutation pulls toward, derived
// from its clusters (first mapped cluster wins — deterministic because a
// bridge's Clusters slice order is fixed by its YAML). "" = no pull.
func archetypeForSpec(spec *mutations.MutationSpec) string {
	if spec == nil {
		return ""
	}
	for _, cl := range spec.Clusters {
		if a, ok := clusterArchetype[cl]; ok {
			return a
		}
	}
	return ""
}
```

- [ ] **Step 4: Rewrite `strongestArchetypePull` to use it**

Replace the body of `strongestArchetypePull` in `internal/behaviortree/archetype_shift.go`:

```go
func strongestArchetypePull(owned map[string]int) string {
	bestKey, bestRarity, bestPull := "", -1, ""
	for key := range owned {
		spec := mutations.GetMutation(key)
		if spec == nil {
			continue
		}
		pull := archetypeForSpec(spec)
		if pull == "" {
			continue
		}
		if spec.Rarity > bestRarity || (spec.Rarity == bestRarity && key < bestKey) {
			bestKey, bestRarity, bestPull = key, spec.Rarity, pull
		}
	}
	return bestPull
}
```

- [ ] **Step 5: Delete the field-validator and its boot call**

- In `internal/behaviortree/archetype_shift.go`, delete `validateArchetypePulls` and `ValidateArchetypePulls` (and the now-unused `GetArchetypePath` **only if** nothing else references it — check with `grep -rn "GetArchetypePath" internal/`; keep it if used elsewhere).
- Remove the boot call. Find it: `grep -rn "ValidateArchetypePulls" internal/` and delete the call site (e.g. in the boot/loader sequence).

- [ ] **Step 6: Remove the `ArchetypePull` field from `MutationSpec`**

In `internal/mutations/mutations.go`, delete the field and its doc comment:

```go
	ArchetypePull string `yaml:"archetype_pull,omitempty"`
```

- [ ] **Step 7: Build and run the behaviortree tests**

Run: `go build ./... 2>&1 | head` → expect exit 0 (if anything still references `.ArchetypePull` or the deleted validator, remove it).
Run: `go test ./internal/behaviortree/ ./internal/mutations/ -v 2>&1 | tail -20` → expect ok, including the two new tests. Delete `TestValidateArchetypePullsCore` (it validated the removed field).

- [ ] **Step 8: Commit**

```bash
git add internal/behaviortree/archetype_shift.go internal/behaviortree/archetype_shift_test.go internal/mutations/mutations.go
git commit -m "feat(mobs): re-base archetype-shift on cluster/pole (drop archetype_pull field)"
```

---

### Task 8: Final verification sweep + boot + full suite

**Files:** none new — verification only.

- [ ] **Step 1: No hardcoded deleted-id remains in Go**

```bash
cd "c:/Users/Calabe Davis/workspace/DOGMud"
for id in adrenaline-surge bioluminescence blinding-flash blinding-spit brazen-resolve \
  camo-skin clawed-hands cold-blooded elongated-limbs extra-legs fast-reflexes hasted \
  healing-gel heightened-senses incorporeal infrared-vision iron-constitution keen-eyes \
  magical-resistance night-vision pacifism-aura pheromone-glands photosynthetic-skin \
  psychic-resistance rapid-metabolism regenerative-tissue sixth-sense skilled sonic-shout \
  talented tough-skin toxic-bite; do
  hits=$(grep -rn "\"$id\"" internal/ --include=*.go)
  [ -n "$hits" ] && echo "STILL REFERENCED: $id" && echo "$hits"
done
```
Expected: no output. (`large`/`small` intentionally omitted — they collide with the `species.Size` enum, which is unrelated. If they were in the list, verify each hit is the enum, not a mutation lookup.) Resolve any real hit.

- [ ] **Step 2: Tutorial + lore-gate still resolve**

Confirm quest-30's `cleric_hadwen` `grant_mutation` (no explicit id → draws from the pool) and room-5200's "has any mutation" gate still reference nothing deleted:
```bash
grep -rn "mutation" _datafiles/world/dogmud/behaviors/pothole_coulee/9100-cleric_hadwen.yaml _datafiles/world/dogmud/behaviors/rooms/pothole_coulee/5200.yaml _datafiles/world/dogmud/quests/30-the_awakening.yaml
```
Expected: no reference to a deleted id (only generic `grant_mutation` / `mutations` command mentions).

- [ ] **Step 3: Full test suite**

Run: `go test ./... 2>&1 | tail -5; go test ./... 2>&1 | grep -cE "^ok"`
Expected: exit 0, 87 packages ok (count may shift by ±1 if a test file was deleted — confirm no `FAIL`).

- [ ] **Step 4: Boot smoke (authoritative)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/npcmig_final_boot.log 2>&1 &
```
Wait for `Server Ready` with no `panic:`/`ERROR: PANIC` (this exercises the graph validator, helpfile completeness, mob loader, and the archetype re-base). Kill the server after.

- [ ] **Step 5: Update PATCH_NOTES + final commit**

Add a dated `PATCH_NOTES.md` entry describing the legacy-mutation nuke + archetype re-base (part of the 0.14.0 clean-break). Then:

```bash
git add PATCH_NOTES.md
git commit -m "docs(patch-notes): NPC mutation migration — legacy pool nuke + cluster-driven archetype shift"
```

---

## Notes for the executor

- **Order matters only for build-green:** Tasks 3, 4, and 7 touch Go; do them before the final `go test ./...` in Task 8. Tasks 2, 5, 6 are data-only. Within each task, keep build+boot green before committing.
- **Do NOT push.** Arc-wide constraint holds until the whole Chrysalis arc (including the player migration + balance pass) is done.
- **This substage does not run the player migration** — that is its own locked spec (`2026-07-11-mutation-migration-design.md`) and ships together in 0.14.0. This plan only removes the legacy content so the player migration lands on a clean graph.
