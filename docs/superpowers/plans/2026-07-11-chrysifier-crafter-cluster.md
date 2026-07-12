# Chrysifier Crafter Cluster — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Chrysifier — a crafting-fed mutation cluster in the Generalist center — with four keystones (Provident Hands, Walking Chrysalis, Faithwrought, Homunculus apex) and the drift/config wiring behind them.

**Architecture:** Three passive keystones add new mutation effect-types/flags consumed at the forage, salvage, craft, and carry-capacity seams; a `chrysifier` cluster is added to `skillClusters` (crafting skills drift toward it). The Homunculus apex reuses the Wave-5 companion system + economy: a brood-style respawn tick spawns a stat-scaled "you-shaped boss" whose statpool = (crafting-skill sum) × scale, inheriting the player's non-crafting skills + a preset physical-mutation loadout, reserving heavy Conviction.

**Tech Stack:** Go. Packages: `internal/mutations`, `internal/configs`, `internal/actions` (forage/salvage/craft), `internal/crafting`, `internal/characters`, `internal/hooks`, `internal/mobs`, plus mutation/mob/help YAML.

**Spec:** `docs/superpowers/specs/2026-07-11-chrysifier-crafter-cluster-design.md`

**Per-node magnitudes are first-pass** (deferred to the Wave-6 playtest per the content-spec convention). Ship sensible values via config knobs.

---

## Verified context (codegraph/read-confirmed — do not re-discover)

- **`skillClusters`** (`internal/mutations/graph.go:51`) — `map[string][]string`, skill→clusters. `ClustersForSkill` reads it. Currently 7 combat/social skills; no crafting entries.
- **Mutation effect system** (`internal/mutations/mutations.go`) — `sumEffects(owned, type, target)` scales `Value × LevelMultiplier(level)`; add a `GetX` helper per new numeric effect-type. Flags via `GetMutationFlags`/`HasMutationFlag` (`type: flag`, Target=flag name). `DescribeEffect`/`flagPhrase` (`describe.go`) render them (needed or `TestDescribeEffect`/help tests fail). `GetCompanionReserveRank` pattern (companion reducer) is the model for a rank-reader.
- **Prereq validator** (`graph.go:23-38`) — **panics at boot if a `prerequisites` id doesn't exist.** Prereq YAML keys are **`id`/`min_level`** (`MutationPrereq`, graph.go:7-9).
- **Forage** — `actions.Forage(actor, ForageOptions)` (`internal/actions/forage.go:35`) computes `searchScore := CalcSearchScore(char)`, calls pure `forager.ForageCore(ForageAttempt{Biome,SearchScore,AtNight,Zone,Weather})` → `ForageResult{Found, ItemId}` (ONE item). Yield boost = raise SearchScore and/or roll a bonus item when the mutation is present.
- **Salvage** — `actions.Salvage(actor, opts)` (`internal/actions/salvage.go:53`) computes `chance := crafting.CalcSalvageChance(salvageSkill, min, max, softCap)` then `salvageCorpse`/`salvageItem` roll returns with `chance`. Yield boost = bump `chance` (cap ≤ 1.0) when the mutation is present.
- **Craft** — `actions.InitiateCraft(actor, recipeName)` (`internal/actions/craft.go:66`):
  - **Station gate:** `if recipe.Station != "" && room.Station != recipe.Station` (craft.go:104). Bypass when `portable-workshop` flag present.
  - **Quality stamp:** `newItem.CraftSkill = skillLevel` (craft.go:138, immediate path). Faithwrought raises the effective level. **NOTE:** the multi-round (async) completion path stamps `CraftSkill` elsewhere — grep `CraftSkill =` and patch both.
  - **Material consume:** `crafting.ConsumeIngredients(...)` (craft.go:135 immediate; async path mirrors). Material discount = a post-consume refund roll (simpler than editing ConsumeIngredients).
- **Carry capacity** — `Character.CarryCapacity()` = `Strength × Balance.CarryCapacityMultiplier` (`internal/characters/character.go`; grep `func (c *Character) CarryCapacity`). Add a `carry_capacity_multiplier` mutation term.
- **Companion spawn** (Wave-5 verified, `internal/hooks/companion_summon.go`): `mobs.NewMobByIdFresh(MobId, roomId, pool)` → `room.AddMob` → `mob.Character.Charm(userId,99999,"")` → `EndAggro` → `user.Character.TrackCharmed(id,true)` → build `CompanionInfo{…, ConvictionReserve: reserve}` → `AddCompanion` → `RecalculateStats`. `CalcCompanionReserve`/`CanAffordCompanion` (`companions.go`) exist. `MobDeath_CompanionCleanup` removes dead companions.
- **Config** — `Balance` struct in `config.balance.go`; defaults in `validateMobs`/`validateCombat` (`config.balance.mobs.go`/`.combat.go`), `ConfigInt`/`ConfigFloat` types, `<=0`/`<1` guards.

---

## File Structure

- `internal/mutations/graph.go` — add crafting skills → `chrysifier` in `skillClusters`.
- `internal/mutations/chrysifier.go` (+ `_test.go`) — effect-type reader helpers (`GetForageYieldMult`, `GetSalvageYieldBonus`, `GetCraftMaterialDiscount`, `GetCraftQualityBonus`, `GetCarryCapacityMultiplier`, `HasPortableWorkshop`, `HasHomunculus`).
- `internal/mutations/describe.go` — DescribeEffect + flagPhrase cases for the new types/flags.
- `internal/configs/config.balance.go` + `.mobs.go` — Chrysifier + Homunculus knobs.
- `internal/actions/forage.go`, `salvage.go` — yield hooks.
- `internal/actions/craft.go` (+ the async completion path) — station bypass, quality bonus, material-refund.
- `internal/characters/character.go` — carry-capacity term.
- `internal/hooks/chrysifier_homunculus.go` (+ `_test.go`) — the respawn tick + spawn/stat-scale/inherit helper.
- `_datafiles/world/dogmud/mobs/**` — a base Homunculus mob template.
- `_datafiles/world/dogmud/mutations/` — `provident-hands.yaml`, `walking-chrysalis.yaml`, `faithwrought.yaml`, `homunculus.yaml` (+ help templates).

---

### Task 1: Cluster drift wiring

**Files:** `internal/mutations/graph.go`; `internal/mutations/graph_test.go`

- [ ] **Step 1: Failing test** — assert `ClustersForSkill("blacksmithing")` etc. return `["chrysifier"]`:

```go
func TestSkillClusters_Chrysifier(t *testing.T) {
	for _, s := range []string{"blacksmithing", "alchemy", "tailoring", "cooking",
		"jewelcrafting", "enchanting", "salvage", "foraging"} {
		cls := ClustersForSkill(s)
		if len(cls) != 1 || cls[0] != "chrysifier" {
			t.Fatalf("%s -> %v, want [chrysifier]", s, cls)
		}
	}
}
```

- [ ] **Step 2: verify fails.** `go test ./internal/mutations/ -run TestSkillClusters_Chrysifier`

- [ ] **Step 3: Implement.** Add to the `skillClusters` map:

```go
	"blacksmithing": {"chrysifier"},
	"alchemy":       {"chrysifier"},
	"tailoring":     {"chrysifier"},
	"cooking":       {"chrysifier"},
	"jewelcrafting": {"chrysifier"},
	"enchanting":    {"chrysifier"},
	"salvage":       {"chrysifier"},
	"foraging":      {"chrysifier"},
```

> Drift-weighting note (spec §6): eight crafting skills feed one cluster. If drift proves too fast/slow vs. single-skill clusters, tune via the affinity signal later — out of scope here.

- [ ] **Step 4: verify passes. Step 5: commit** `feat(chrysifier): crafting skills drift toward the chrysifier cluster`.

---

### Task 2: Effect-type readers + descriptors

**Files:** `internal/mutations/chrysifier.go`, `_test.go`; `internal/mutations/describe.go`

- [ ] **Step 1: Failing test** for each reader (mirror `companion_reserve_test.go`): seed a mutation with each effect and assert the helper returns the summed/flag value. Cover `GetForageYieldMult`, `GetSalvageYieldBonus`, `GetCraftMaterialDiscount`, `GetCraftQualityBonus`, `GetCarryCapacityMultiplier` (all `sumEffects`-based floats), and `HasPortableWorkshop`, `HasHomunculus` (flag-based).

- [ ] **Step 2: verify fails.**

- [ ] **Step 3: Implement** (`chrysifier.go`) — each numeric reader is `sumEffects(owned, "<type>", "")`; flags via `HasMutationFlag`:

```go
func GetForageYieldMult(o map[string]int) float64      { return sumEffects(o, "forage_yield_multiplier", "") }
func GetSalvageYieldBonus(o map[string]int) float64    { return sumEffects(o, "salvage_yield_bonus", "") }
func GetCraftMaterialDiscount(o map[string]int) float64 { return sumEffects(o, "craft_material_discount", "") }
func GetCraftQualityBonus(o map[string]int) float64    { return sumEffects(o, "craft_quality_bonus", "") }
func GetCarryCapacityMultiplier(o map[string]int) float64 { return sumEffects(o, "carry_capacity_multiplier", "") }
func HasPortableWorkshop(o map[string]int) bool        { return HasMutationFlag(o, "portable-workshop") }
func HasHomunculus(o map[string]int) bool              { return HasMutationFlag(o, "homunculus") }
```

Add `DescribeEffect` cases for the five numeric types + `flagPhrase` cases for `portable-workshop` and `homunculus` (number-free, feel-based phrases).

- [ ] **Step 4: verify passes** (incl. `TestDescribeEffect`). **Step 5: commit** `feat(chrysifier): mutation effect-type readers + descriptors`.

---

### Task 3: Config knobs

**Files:** `internal/configs/config.balance.go` + `.mobs.go`; a `_test.go`

- [ ] Add + default-guard + test: `HomunculusCraftScale` (ConfigFloat, 4.0), `HomunculusConvictionReserve` (ConfigInt, 1000), `ChrysifierForageYieldBonusPct`/`ChrysifierSalvageYieldBonusPct`/`ChrysifierCraftDiscountPct`/`ChrysifierCraftQualityBonus`/`ChrysifierCarryBonusPct` (first-pass magnitudes if you prefer config-driven node values; otherwise bake into YAML effect values and skip these). Follow the Task-2 companion-knob pattern from the prior wave. **Commit** `feat(chrysifier): config knobs`.

---

### Task 4: Forage yield hook

**Files:** `internal/actions/forage.go`; `forage_test.go`

- [ ] **Step 1: Failing test** — with a `forage_yield_multiplier` mutation owned, the effective search score is boosted (assert via a seam or a small extracted helper `foragedSearchScore(char)` that applies the mult). Prefer extracting `foragedSearchScore(char) float64 = CalcSearchScore(char) * (1 + mutations.GetForageYieldMult(char.Mutations))` and testing that.

- [ ] **Step 2-3:** implement — in `Forage`, replace `searchScore := CalcSearchScore(char)` with the boosted helper; optionally roll a bonus second item when `GetForageYieldMult > 0` and the first find succeeds. Keep player/mob parity (mutations rarely on mobs → no-op).

- [ ] **Step 4-5:** verify + **commit** `feat(chrysifier): Provident Hands boosts foraging yield`.

---

### Task 5: Salvage yield hook

**Files:** `internal/actions/salvage.go`; `salvage_test.go`

- [ ] In `Salvage`, after `chance := crafting.CalcSalvageChance(...)`, apply `chance = min(1.0, chance * (1 + mutations.GetSalvageYieldBonus(char.Mutations)))`. Test that a salvage-bonus mutation raises the passed chance (extract the chance calc into a testable helper if needed). **Commit** `feat(chrysifier): Provident Hands boosts salvage yield`.

---

### Task 6: Craft hooks — station bypass, quality, material refund

**Files:** `internal/actions/craft.go` (+ the async completion path — grep `CraftSkill =` and `ConsumeIngredients` to find it); tests where a seam exists.

- [ ] **Step 1: Station bypass (Walking Chrysalis).** Change the gate:

```go
if recipe.Station != "" && room.Station != recipe.Station &&
	!mutations.HasPortableWorkshop(char.Mutations) {
	res.StationNeeded = strings.ReplaceAll(recipe.Station, "_", " ")
	res.WrongStation = true
	return res
}
```

- [ ] **Step 2: Craft quality (Faithwrought).** At each `newItem.CraftSkill = skillLevel` (immediate **and** async completion path), add the bonus:

```go
newItem.CraftSkill = skillLevel + int(math.Round(float64(skillLevel)*mutations.GetCraftQualityBonus(char.Mutations)))
```

(Or a flat bonus — pick per the effect's semantics; keep both paths consistent.)

- [ ] **Step 3: Material refund (Provident Hands).** After `ConsumeIngredients`, if `GetCraftMaterialDiscount > 0`, roll per consumed ingredient to return one to the pool. Simplest: a helper `refundIngredients(char, recipe, discount)` that re-adds a fraction of consumed items. Keep it contained; if the async path consumes on completion, apply there too.

- [ ] **Step 4: build + tests.** `go build ./... && go test ./internal/actions/ ./internal/crafting/`. Add `mutations` + `math` imports as needed.

- [ ] **Step 5: commit** `feat(chrysifier): Walking Chrysalis (portable workshop) + Faithwrought (quality) + material refund`.

---

### Task 7: Carry-capacity hook (Faithwrought)

**Files:** `internal/characters/character.go`; a `_test.go`

- [ ] In `CarryCapacity()`, fold in the mutation multiplier: `base * (1 + mutations.GetCarryCapacityMultiplier(c.Mutations))`. Test that a carry-mult mutation raises the returned capacity. **Commit** `feat(chrysifier): Faithwrought raises carry capacity`.

---

### Task 8: Author the three passive mutations + help

**Files:** `_datafiles/world/dogmud/mutations/{provident-hands,walking-chrysalis,faithwrought}.yaml` + `templates/help/*.template`

- [ ] Author each with `clusters: [chrysifier]`, `pole: ""`, rarity (entry ≈ 3, cores ≈ 6), prereq spine (Walking Chrysalis & Faithwrought each `prerequisites: [{id: provident-hands, min_level: 1}]`), and the effect blocks:
  - **provident-hands:** `forage_yield_multiplier`, `salvage_yield_bonus`, `craft_material_discount` pros.
  - **walking-chrysalis:** `flag: portable-workshop`.
  - **faithwrought:** `craft_quality_bonus` + `carry_capacity_multiplier` pros (lightly belief-flavored text).
- [ ] Help templates (80-col, number-free). Boot-smoke (prereqs resolve, mutations load). **Commit** `content(chrysifier): Provident Hands, Walking Chrysalis, Faithwrought`.

---

### Task 9: Homunculus base mob + spawn/scale/inherit helper

**Files:** `_datafiles/world/dogmud/mobs/**/homunculus.yaml` (base template); `internal/hooks/chrysifier_homunculus.go` (+ `_test.go`)

- [ ] **Base mob:** a generic Homunculus mob template (a blank-ish humanoid; stats will be overridden). Give it a name placeholder; the spawn code renames to `<player>'s Homunculus` (or the finalized name).

- [ ] **Spawn helper** `spawnHomunculus(user *users.UserRecord, room *rooms.Room)`:
  1. `craftSum := sum of the owner's crafting-skill levels` (blacksmithing/alchemy/tailoring/cooking/jewelcrafting/enchanting/salvage/foraging).
  2. `pool := int(craftSum * cfg.HomunculusCraftScale)`.
  3. `mob := mobs.NewMobByIdFresh(homunculusMobId, room.RoomId, pool)`; rename; `room.AddMob`.
  4. Inherit: copy the owner's **non-crafting** skills into the mob's skills; apply a **preset physical-cluster mutation loadout** (a fixed map, e.g. a Colossus/Ravener entry+core set — pick the exact set here).
  5. Reserve: `reserve := user.Character.CalcCompanionReserve(int(cfg.HomunculusConvictionReserve))`; `Charm(userId,99999,"")`; `EndAggro`; `TrackCharmed`; build `CompanionInfo{…, SourceType: CompanionSummoned, ConvictionReserve: reserve}`; `AddCompanion`; `RecalculateStats`.

- [ ] **Test** the pure parts: a `homunculusStatPool(craftSum, scale)` helper (= `round(craftSum*scale)`) and the non-crafting-skill filter. The spawn wiring itself is exercised at boot/smoke.

- [ ] **Commit** `feat(chrysifier): Homunculus spawn + craft-scaled statpool + inheritance`.

---

### Task 10: Homunculus respawn tick + apex mutation

**Files:** `internal/hooks/chrysifier_homunculus.go`; `internal/hooks/NewRound_UserRoundTick.go` (register the tick); `_datafiles/world/dogmud/mutations/homunculus.yaml` + help

- [ ] **Respawn tick** `tickHomunculus(user)` (per-round, called from the user round-tick): if the user `HasHomunculus` flag and has **no** live homunculus companion and the respawn cooldown has elapsed and they can afford the reserve → `spawnHomunculus`. Mirrors the superseded Wave-4c brood-respawn idea; death is handled by `MobDeath_CompanionCleanup`, so respawn is emergent. Use a cooldown key (`homunculus-respawn`, a few rounds).

- [ ] **Apex mutation** `homunculus.yaml`: `clusters: [chrysifier]`, `pole: ""`, rarity 8, `flag: homunculus`, `prerequisites: [{id: walking-chrysalis, min_level: 1}, {id: faithwrought, min_level: 1}]`. Help template (crafted-flavor; note it's your one big companion). Finalize the name (Homunculus vs "Mind of the Sculptor").

- [ ] **Build + boot smoke** — nuke instances, boot, confirm mutations load + prereq validator passes + a manual sanity that granting the apex spawns a companion (or leave to playtest). **Commit** `feat(chrysifier): Homunculus apex + respawn tick`.

---

### Task 11: Help/describe coverage, full suite, patch notes + boot

- [ ] `go test ./... -run 'TestHelpFileCompleteness_Mutations|TestDescribeEffect'` — add any missing template/case.
- [ ] `go test ./...` — exit 0.
- [ ] `PATCH_NOTES.md` — a dated player-facing entry (the maker's path: uncanny provisioning, craft-anywhere, faith-forged goods, and a crafted twin of yourself). No hard numbers.
- [ ] Nuke instances, full boot smoke (clean load, mapper errors=0, Server Ready, no panic).
- [ ] **Commit** `docs(chrysifier): patch notes` + any coverage fixes.

---

## Out of scope (follow-on)
- **Migration classification into Chrysifier** — lives in the parked prod-migration plan; this cluster unblocks it.
- **Per-rank magnitude curves** + the drift-weighting balance for eight-skills-one-cluster — Wave-6 playtest.
- **NPC mutations decision** (mobs still on the retired-41) — parked, see migration spec §8.
- **Final Homunculus name** + the exact preset physical-mutation loadout it carries — decided during Task 9/10 authoring.
