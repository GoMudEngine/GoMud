# NPC Mutation Kits Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the world's monstrous/magical creatures a coherent identity in the mutation graph — a curated cluster kit per appropriate species (`intrinsic_mutations`) plus bespoke overlays on ~40–60 curated bosses (`spawnmutations`) — as the NPC companion to the player migration.

**Architecture:** Data-only. Edit species YAMLs to carry `intrinsic_mutations` kits and boss mob YAMLs to carry `spawnmutations` overlays; the engine already merges both and boot-validates the ids. A new filesystem-walk coherence test (in `internal/devtools`, mirroring the legacy-pool guard) locks the invariants: every kit id is a live mutation, and each kitted species is *anchored* to its declared primary cluster. Ranks are provisional per the spec (§6) and retuned later in the 6e balance pass.

**Tech Stack:** YAML data files, Go `go test ./...`, local server boot smoke.

**Spec:** `docs/superpowers/specs/2026-07-12-npc-mutation-kits-design.md`

**Conventions (CLAUDE.md):**
- Boot smoke: `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*` then `go run .`; wait for `Server Ready` with no `panic:`. (In this env, wait via a `ping -n`-spaced poll, not a busy loop; kill with `taskkill //F //IM go.exe`.)
- Commit footer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`. **Do NOT push** (arc-wide constraint).

---

## Reference A — species → kit (from spec §4)

Each block below is the **exact `intrinsic_mutations` map** to author. Ranks are
PROVISIONAL (common keystone 1, elite keystone 1–2, apex 1). Where a species
already has intrinsics (e.g. `tail: 1`), **merge** — keep existing entries, add
the new ones.

```
# Undead
32-wraith          : ether-gland: 2, second-sight: 1, kinetic-backlash: 1, discorporation: 1
33-spectre         : ether-gland: 2, second-sight: 1, discorporation: 1
0-ghostly_spirit   : ether-gland: 1
30-skeleton        : (keep hollow-bones: 1 as-is — no change)
31-zombie          : regrowth: 1
34-vampire         : padded-soles: 1, compound-eyes: 1, venom-glands: 1
35-flesh_golem     : dense-muscles: 1, titan-growth: 1

# Elementals
36-water_elemental : regrowth: 1
37-earth_elemental : dense-muscles: 1, titan-growth: 1, thick-hide: 1
38-air_elemental   : ether-gland: 1, second-sight: 1
39-fire_elemental  : ether-gland: 1, kinetic-backlash: 1
40-magma_elemental : thick-hide: 1, chitin-plating: 1
41-sand_elemental  : padded-soles: 1, veiling-musk: 1
42-storm_elemental : ether-gland: 1, reflect-skin-voltaic: 1
43-ice_elemental   : thick-hide: 1, reflect-skin-frostbite: 1
44-smoke_elemental : second-sight: 1, veiling-musk: 1

# Overtly-magical / monstrous
23-aberration      : evil-eye: 1, corvid-brain: 1
99-ascended        : commanding-presence: 1, zealous-conviction: 1, radiant-avatar: 1
16-slime           : regrowth: 1
15-fungal_colony   : sticky-secretion: 1, dissonance-organ: 1
14-carnivorous_plant : grasping-tendrils: 1
20-orb             : ether-gland: 1
4-troll            : thick-hide: 1, regrowth: 1

# Mundane beasts — light 1–2 touch (merge with existing tail:1 etc.)
2-canine           : rending-claws: 1
11-feline          : padded-soles: 1
3-bear             : dense-muscles: 1
6-boar             : dense-muscles: 1
9-raptor           : raptor-legs: 1
7-deer             : keen-senses: 1
8-serpent          : venom-glands: 1
17-arachnid        : silk-glands: 1
18-worm            : (keep tremorsense: 1 as-is — no change)
22-bat             : keen-senses: 1
24-mustelid        : padded-soles: 1
12-insectoid       : compound-eyes: 1
10-rodent          : (keep tail: 1 — no change)
13-fish            : (no kit — no change)

# Bare (no kit — do not touch): 1-human, 5-goblin, 19-dummy, 21-reptile*
# (*reptile keeps its existing tail:1 only; no cluster kit.)
```

## Reference B — cluster → apex + supporting keystone (for boss overlays, §5)

Boss overlay (`spawnmutations`) = **its species' cluster apex + one supporting
keystone**. Signature/endgame bosses may add a second entry or splash.

```
colossus   : colossus-form   (+ titan-growth)
ironhide   : living-carapace (+ regrowth)
ravener    : apex-predator   (+ rending-claws)
stalker    : chameleon-skin  (+ compound-eyes)
ethereal   : discorporation  (+ kinetic-backlash)
manifester : brood-mother    (+ hive-mind)
zealot     : radiant-avatar  (+ commanding-presence)
weaver     : paralytic-field (+ grasping-tendrils)
trickster  : translucent-body(+ corvid-brain)
```

---

### Task 1: Kit-coherence test (TDD anchor)

Locks the two invariants and fails until Phase 1 kits are authored.

**Files:**
- Create: `internal/devtools/npc_kits_test.go`

- [ ] **Step 1: Write the test**

```go
package devtools

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// speciesPrimaryCluster is the executable encoding of spec §4: each kitted,
// cluster-anchored species and the body/belief cluster its kit must anchor to.
// Center-anchored species (skeleton→hollow-bones) and bare species are omitted —
// they are not anchoring-checked. Keep in sync with the design doc.
var speciesPrimaryCluster = map[string]string{
	"32-wraith": "ethereal", "33-spectre": "ethereal", "0-ghostly_spirit": "ethereal",
	"31-zombie": "ironhide", "34-vampire": "stalker", "35-flesh_golem": "colossus",
	"36-water_elemental": "ironhide", "37-earth_elemental": "colossus",
	"38-air_elemental": "ethereal", "39-fire_elemental": "ethereal",
	"40-magma_elemental": "ironhide", "41-sand_elemental": "stalker",
	"42-storm_elemental": "ethereal", "43-ice_elemental": "ironhide",
	"44-smoke_elemental": "ethereal", "23-aberration": "trickster",
	"99-ascended": "zealot", "16-slime": "ironhide", "15-fungal_colony": "weaver",
	"14-carnivorous_plant": "weaver", "20-orb": "ethereal", "4-troll": "ironhide",
}

var intrinsicLineRe = regexp.MustCompile(`^\s+([a-z0-9-]+):\s*\d+\s*$`)
var clustersLineRe = regexp.MustCompile(`^clusters:\s*\[(.*)\]`)

// mutationClusters reads the clusters tag list from a mutation YAML ("" -> none).
func mutationClusters(t *testing.T, root, id string) ([]string, bool) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root, "mutations", id+".yaml"))
	if err != nil {
		return nil, false // mutation does not exist
	}
	for _, ln := range strings.Split(string(body), "\n") {
		if m := clustersLineRe.FindStringSubmatch(strings.TrimSpace(ln)); m != nil {
			var out []string
			for _, c := range strings.Split(m[1], ",") {
				if c = strings.TrimSpace(c); c != "" {
					out = append(out, c)
				}
			}
			return out, true
		}
	}
	return nil, true // exists, zero-cluster (Center)
}

// speciesIntrinsicIDs returns the intrinsic mutation ids declared in a species file.
func speciesIntrinsicIDs(t *testing.T, path string) []string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var ids []string
	in := false
	for _, ln := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(ln, "intrinsic_mutations:") {
			in = true
			continue
		}
		if in {
			if m := intrinsicLineRe.FindStringSubmatch(ln); m != nil {
				ids = append(ids, m[1])
			} else if strings.TrimSpace(ln) != "" && !strings.HasPrefix(ln, " ") {
				in = false
			}
		}
	}
	return ids
}

// TestNPCKits_IDsLiveAndAnchored asserts every species intrinsic + mob
// spawnmutation id is a live mutation, and every cluster-anchored species (per
// spec §4) has a kit with at least one member in its declared primary cluster.
func TestNPCKits_IDsLiveAndAnchored(t *testing.T) {
	root := dataRoot(t)

	// (1) every intrinsic + spawnmutation id must resolve to a live mutation YAML.
	check := func(id, where string) {
		if _, ok := mutationClusters(t, root, id); !ok {
			t.Errorf("%s references non-existent mutation %q", where, id)
		}
	}
	speciesDir := filepath.Join(root, "species")
	speciesFiles, _ := filepath.Glob(filepath.Join(speciesDir, "*.yaml"))
	for _, f := range speciesFiles {
		for _, id := range speciesIntrinsicIDs(t, f) {
			check(id, "species "+filepath.Base(f))
		}
	}

	// (2) each cluster-anchored species has a kit anchored to its primary cluster.
	for sp, cluster := range speciesPrimaryCluster {
		ids := speciesIntrinsicIDs(t, filepath.Join(speciesDir, sp+".yaml"))
		if len(ids) == 0 {
			t.Errorf("species %s: expected a cluster kit, has no intrinsic_mutations", sp)
			continue
		}
		anchored := false
		for _, id := range ids {
			cls, _ := mutationClusters(t, root, id)
			for _, c := range cls {
				if c == cluster {
					anchored = true
				}
			}
		}
		if !anchored {
			t.Errorf("species %s: kit %v not anchored to primary cluster %q", sp, ids, cluster)
		}
	}
}
```

- [ ] **Step 2: Run it and confirm it FAILS**

Run: `go test ./internal/devtools/ -run TestNPCKits_IDsLiveAndAnchored -v`
Expected: FAIL — most anchored species (wraith, spectre, …) have no
`intrinsic_mutations` yet, so "expected a cluster kit" fires for each. (The
id-liveness half passes because current data has no dead ids.)

- [ ] **Step 3: Commit the failing guard**

```bash
git add internal/devtools/npc_kits_test.go
git commit -m "test(species): NPC-kit coherence guard — ids-live + cluster-anchored (currently failing)"
```

---

### Task 2: Phase 1 — undead species kits

**Files (Modify):** `_datafiles/world/dogmud/species/{0-ghostly_spirit,31-zombie,34-vampire,35-flesh_golem,32-wraith,33-spectre}.yaml`

- [ ] **Step 1: Add/merge each undead kit** (per Reference A)

For a species with **no** `intrinsic_mutations` block yet (e.g. wraith), insert
the block just after the last top-of-file scalar and before `buffids:`/`stats:`.
Example — `32-wraith.yaml` (insert after `tameable: false`):

```yaml
intrinsic_mutations:
  ether-gland: 2
  second-sight: 1
  kinetic-backlash: 1
  discorporation: 1
```

`33-spectre.yaml`:
```yaml
intrinsic_mutations:
  ether-gland: 2
  second-sight: 1
  discorporation: 1
```

`0-ghostly_spirit.yaml`:
```yaml
intrinsic_mutations:
  ether-gland: 1
```

`31-zombie.yaml`:
```yaml
intrinsic_mutations:
  regrowth: 1
```

`34-vampire.yaml` (it had `regenerative-tissue`/`night-vision` dropped — add the Stalker kit):
```yaml
intrinsic_mutations:
  padded-soles: 1
  compound-eyes: 1
  venom-glands: 1
```

`35-flesh_golem.yaml` (replace the single repointed `titan-growth: 1` with the full base):
```yaml
intrinsic_mutations:
  dense-muscles: 1
  titan-growth: 1
```

Leave `30-skeleton.yaml` unchanged (keep `hollow-bones: 1`).

- [ ] **Step 2: Boot smoke**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/kits_boot.log 2>&1 &
```
Wait for `Server Ready` (no `panic:`) in `/tmp/kits_boot.log` (the intrinsic-id
validator runs here). Kill the server.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/species
git commit -m "content(species): undead cluster kits (wraith/spectre Ethereal, vampire Stalker, golem Colossus, ...)"
```

---

### Task 3: Phase 1 — elemental species kits

**Files (Modify):** `_datafiles/world/dogmud/species/{36-water,37-earth,38-air,39-fire,40-magma,41-sand,42-storm,43-ice,44-smoke}_elemental.yaml`

- [ ] **Step 1: Add/merge each elemental kit** (per Reference A). Each elemental
currently has an empty (dropped) or single intrinsic — set the block to exactly:

```yaml
# 36-water_elemental.yaml
intrinsic_mutations:
  regrowth: 1
# 37-earth_elemental.yaml
intrinsic_mutations:
  dense-muscles: 1
  titan-growth: 1
  thick-hide: 1
# 38-air_elemental.yaml
intrinsic_mutations:
  ether-gland: 1
  second-sight: 1
# 39-fire_elemental.yaml
intrinsic_mutations:
  ether-gland: 1
  kinetic-backlash: 1
# 40-magma_elemental.yaml
intrinsic_mutations:
  thick-hide: 1
  chitin-plating: 1
# 41-sand_elemental.yaml
intrinsic_mutations:
  padded-soles: 1
  veiling-musk: 1
# 42-storm_elemental.yaml
intrinsic_mutations:
  ether-gland: 1
  reflect-skin-voltaic: 1
# 43-ice_elemental.yaml
intrinsic_mutations:
  thick-hide: 1
  reflect-skin-frostbite: 1
# 44-smoke_elemental.yaml
intrinsic_mutations:
  second-sight: 1
  veiling-musk: 1
```

Note: `reflect-skin-voltaic`/`reflect-skin-frostbite` carry a `moon_flavor` tag,
but that only gates the *acquisition pool* — assigning them directly as an
intrinsic is unaffected (like `spawnmutations`). They are the single thematic
splash allowed by spec §3.1 alongside the Ethereal/Ironhide anchor.

- [ ] **Step 2: Boot smoke** (as Task 2 Step 2).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/species
git commit -m "content(species): elemental cluster kits (Ethereal/Ironhide anchors + elemental reflect riders)"
```

---

### Task 4: Phase 1 — magical/monstrous species kits

**Files (Modify):** `_datafiles/world/dogmud/species/{23-aberration,99-ascended,16-slime,15-fungal_colony,14-carnivorous_plant,20-orb,4-troll}.yaml`

- [ ] **Step 1: Add/merge each kit** (per Reference A):

```yaml
# 23-aberration.yaml
intrinsic_mutations:
  evil-eye: 1
  corvid-brain: 1
# 99-ascended.yaml
intrinsic_mutations:
  commanding-presence: 1
  zealous-conviction: 1
  radiant-avatar: 1
# 16-slime.yaml
intrinsic_mutations:
  regrowth: 1
# 15-fungal_colony.yaml
intrinsic_mutations:
  sticky-secretion: 1
  dissonance-organ: 1
# 14-carnivorous_plant.yaml
intrinsic_mutations:
  grasping-tendrils: 1
# 20-orb.yaml
intrinsic_mutations:
  ether-gland: 1
# 4-troll.yaml
intrinsic_mutations:
  thick-hide: 1
  regrowth: 1
```

- [ ] **Step 2: Boot smoke** (as before).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/species
git commit -m "content(species): magical/monstrous cluster kits (aberration Trickster, ascended Zealot, plant Weaver, troll Ironhide, ...)"
```

---

### Task 5: Phase 1 — mundane beast light touch + coherence green

**Files (Modify):** `_datafiles/world/dogmud/species/{2-canine,11-feline,3-bear,6-boar,9-raptor,7-deer,8-serpent,17-arachnid,22-bat,24-mustelid,12-insectoid}.yaml`

- [ ] **Step 1: Merge the 1-mutation touch into each beast** (keep any existing
`tail: 1`). Add under the existing (or new) `intrinsic_mutations:`:

```
2-canine     -> rending-claws: 1     11-feline    -> padded-soles: 1
3-bear       -> dense-muscles: 1     6-boar       -> dense-muscles: 1
9-raptor     -> raptor-legs: 1       7-deer       -> keen-senses: 1
8-serpent    -> venom-glands: 1      17-arachnid  -> silk-glands: 1
22-bat       -> keen-senses: 1       24-mustelid  -> padded-soles: 1
12-insectoid -> compound-eyes: 1
```
Do NOT touch: `10-rodent` (keep `tail`), `18-worm` (keep `tremorsense`),
`13-fish`, `21-reptile`, `1-human`, `5-goblin`, `19-dummy`, `30-skeleton`.

- [ ] **Step 2: Run the coherence test — now PASSES**

Run: `go test ./internal/devtools/ -run TestNPCKits_IDsLiveAndAnchored -v`
Expected: PASS (every anchored species now has an anchored kit; all ids live).

- [ ] **Step 3: Boot smoke + commit**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/kits_boot.log 2>&1 &   # wait for Server Ready, then kill
git add _datafiles/world/dogmud/species
git commit -m "content(species): mundane-beast flavor mutations; NPC-kit coherence test green"
```

---

### Task 6: Phase 2 — boss bespoke overlays

Author `spawnmutations` overlays on the curated bosses. The overlay ADDS to the
species base (both merge onto the creature).

**Files (Modify):** boss mob YAMLs under `_datafiles/world/dogmud/mobs/**` (list built in Step 1).

- [ ] **Step 1: Build the candidate boss list**

```bash
cd "c:/Users/Calabe Davis/workspace/DOGMud"
# (a) the existing 33 spawnmutations mobs:
grep -rln "spawnmutations:" _datafiles/world/dogmud/mobs/ | sort
# (b) proper-noun / unique mobs (name not starting with "a "/"an "/"the " lowercase article),
#     focusing on apex/alpha/sentinel/keeper/prime/guardian/king/queen/lord/matriarch:
grep -rilnE "^name:.*(apex|alpha|sentinel|keeper|warden|prime|guardian|king|queen|lord|matriarch|chieftain|the )" _datafiles/world/dogmud/mobs/ | sort
```
Cross-reference with the endgame/quest bosses named in the spec §5.2
(`docs/ENDGAME_COMBAT_TUNING.md` for #20/#21 + Meirok; Confluence #17 Q73/Q74;
Q34 captain). Assemble the final ~40–60 set. `log`-style note in the commit which
mobs were included and which named mobs were intentionally left on species-kit
only (no silent truncation).

- [ ] **Step 2: For each boss, derive and author its overlay**

Rule: look up the boss's `species:` (or infer from its lore/name) → its primary
cluster → apply **Reference B** (`<cluster> apex + supporting keystone`). Add:

```yaml
spawnmutations: [<cluster-apex>, <supporting-keystone>]
```
If the mob already has `spawnmutations` (one of the 33), replace its (repointed)
single entry with the boss kit. Worked examples:

```yaml
# the_foldweaver (Weaver boss):        spawnmutations: [paralytic-field, grasping-tendrils]
# warden_prime / the_core_guardian (construct/Colossus): spawnmutations: [colossus-form, titan-growth]
# the_pass_apex / the_reach_alpha (beast/Ravener):        spawnmutations: [apex-predator, rending-claws]
# a_highland_stalker_cat / cliff_stalker (Stalker):       spawnmutations: [chameleon-skin, compound-eyes]
# elemental_king (elemental, signature — splash):         spawnmutations: [discorporation, colossus-form]
# stone_beetle_queen (arachnid/Weaver or Ironhide):       spawnmutations: [living-carapace, chitin-plating]
```
Signature/endgame bosses (Meirok, #20/#21) get a heavier kit per
`ENDGAME_COMBAT_TUNING.md` — their cluster apex + 2 supporting keystones, or a
two-cluster splash where their lore warrants it.

- [ ] **Step 3: Verify no deleted-legacy id crept in + boot**

```bash
grep -rnE "spawnmutations:.*(incorporeal|large|small|tough-skin|keen-eyes|fast-reflexes|iron-constitution|regenerative-tissue|camo-skin|night-vision|cold-blooded|magical-resistance|psychic-resistance|sixth-sense|heightened-senses|brazen-resolve|adrenaline-surge|bioluminescence|hasted|clawed-hands|elongated-limbs|extra-legs|healing-gel|blinding-flash|blinding-spit|pacifism-aura|sonic-shout|toxic-bite|infrared-vision|photosynthetic-skin|rapid-metabolism|pheromone-glands|skilled|talented)\b" _datafiles/world/dogmud/mobs/
```
Expected: no output (this grep is the spawn-id regression guard). Then boot smoke
(`Server Ready`, no panic — the engine validates every merged mutation id at
load). Re-run `go test ./internal/devtools/ -run TestNPCKits_IDsLiveAndAnchored`
→ still PASS (species side unchanged by this task).

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs
git commit -m "content(mobs): boss bespoke mutation-kit overlays (~N curated bosses)"
```

---

### Task 7: Phase 3 — final verification + PATCH_NOTES

**Files:** `PATCH_NOTES.md`

- [ ] **Step 1: Full suite + final boot**

```bash
go test ./... 2>&1 | tail -3; go test ./... 2>&1 | grep -cE "^ok"   # expect 87
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/kits_final_boot.log 2>&1 &   # expect Server Ready, no panic; then kill
```

- [ ] **Step 2: Regression sweep — no deleted-legacy id in any intrinsic OR spawn**

```bash
grep -rnwE "(incorporeal|large|small|tough-skin|keen-eyes|fast-reflexes|iron-constitution|regenerative-tissue|camo-skin|night-vision|cold-blooded|magical-resistance|psychic-resistance|sixth-sense|heightened-senses|brazen-resolve)" _datafiles/world/dogmud/species/ _datafiles/world/dogmud/mobs/ | grep -viE "description:|#|size:"
```
Expected: no structured (non-prose) hit.

- [ ] **Step 3: Add PATCH_NOTES entry + commit**

Add a dated entry describing the creature identity overhaul (players will notice
tougher, thematically-coherent monsters). Then:

```bash
git add PATCH_NOTES.md
git commit -m "docs(patch-notes): NPC mutation kits — species identities + boss overlays"
```

---

## Notes for the executor

- **Ranks are PROVISIONAL** (spec §6). Author exactly the ranks in Reference A;
  do NOT invent higher ranks — the 6e balance pass retunes them world-wide.
- **Merge, never clobber** existing intrinsic entries (`tail`, `hollow-bones`,
  `tremorsense`) — those are structural/Center traits.
- **Species fields stay** (`grapple_immune`, `lifedrain`, `buffids`, `body_parts`)
  — the kit is additive graph identity, not a replacement.
- **Do NOT push.** Arc-wide constraint until the whole Chrysalis arc (incl. the
  player migration + 6e balance) is done.
- The **player migration** is a separate spec; its doc deliverables (mutation
  helpfiles, `help dogmud`, README) are recorded there, not here.
