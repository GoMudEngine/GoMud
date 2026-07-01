# Mutation-Scour (Moon-Crash Remort) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the mutation-scour "remort" mechanic — a consumable that strips all of a character's acquired (Chrysalis) mutations, after which the Chrysalis returns stronger (biased toward rarer mutations) as they re-acquire.

**Architecture:** A new `Character.ScourMutations()` clears the acquired-mutation map and re-applies species intrinsics, then grants N "reroll charges" (`Character.MutationRerollBonus`). While charges remain, the mutation-acquisition path (`BloomSeedNewMutation` + the normal use-based grant) squares the Rarity weighting (biasing hard toward rare mutations) and consumes a charge per new mutation. Delivered by a new potion (modeled on the Bloom Wafer) gated to the Crash Site.

**Tech Stack:** Go (internal/characters, internal/mutations), `go test`; a YAML item + buff. This is Plan A of the #22 sequence (then: the Crash Site Interior zone plan, which wires this in as a drop reward + the data-only suppression aura).

**Reference spec:** `docs/superpowers/specs/2026-07-01-crash-site-interior-design.md` §6.

**Verified APIs (do not re-derive):**
- `Character.Mutations map[string]int` (mutationId→level), `character.go:258`.
- `Character.MutationProgress float64`, `character.go:259`.
- `(*Character) BloomSeedNewMutation(rng *rand.Rand) (string,int)` — rarity-weighted new-mutation grant; weight `w := spec.Rarity` (`bloom_mutation.go:99`).
- `(*Character) ApplyIntrinsicMutations(sp *species.Species)` — re-adds species-baseline mutations, cap-aware (`intrinsic.go:21`).
- `(*Character) GrantRandomMutation() string` (`character.go:791`).
- `species.GetSpecies(c.SpeciesId)`; `mutations.GetAll()`, `spec.Rarity`.
- Potion/consume pattern: `internal/usercommands/drink.go` (Bloom Wafer item 40108; buffs applied via `AddBuffScaled`).

---

## Task 1: `MutationRerollBonus` field + `ScourMutations()`

**Files:**
- Modify: `internal/characters/character.go` (add field near `Mutations`/`MutationProgress` ~line 258)
- Create: `internal/characters/mutation_scour.go`
- Test: `internal/characters/mutation_scour_test.go`

- [ ] **Step 1: Add the field.** In the `Character` struct, after `MutationProgress`:
```go
	MutationRerollBonus     int                            `yaml:"mutationrerollbonus,omitempty"` // post-scour charges: while >0, re-acquired mutations bias hard toward rare, one charge consumed per new mutation
```

- [ ] **Step 2: Write the failing test** (`mutation_scour_test.go`):
```go
package characters

import "testing"

func TestScourMutations_ClearsAcquiredKeepsIntrinsicsGrantsCharges(t *testing.T) {
	c := New()
	c.SpeciesId = 1 // human (no intrinsics expected)
	c.Mutations = map[string]int{"keen-eyes": 2, "large": 1}
	c.MutationProgress = 0.9

	c.ScourMutations(3)

	if len(c.Mutations) != 0 {
		t.Fatalf("expected acquired mutations cleared, got %v", c.Mutations)
	}
	if c.MutationRerollBonus != 3 {
		t.Fatalf("expected 3 reroll charges, got %d", c.MutationRerollBonus)
	}
	if c.MutationProgress != 0 {
		t.Fatalf("expected MutationProgress reset to 0, got %v", c.MutationProgress)
	}
}
```

- [ ] **Step 3: Run it — expect FAIL** (`ScourMutations` undefined):
Run: `go test ./internal/characters/ -run TestScourMutations_ClearsAcquired -v`
Expected: FAIL (undefined: ScourMutations)

- [ ] **Step 4: Implement** (`mutation_scour.go`):
```go
package characters

import "github.com/GoMudEngine/GoMud/internal/species"

// ScourMutations strips ALL of the character's mutations (the moon-crash
// "remort"), re-applies species intrinsics, resets mutation progress, and
// grants `charges` reroll charges. While charges remain, re-acquired
// mutations bias hard toward rare ones (see BloomSeedNewMutation).
func (c *Character) ScourMutations(charges int) {
	c.Mutations = map[string]int{}
	if sp := species.GetSpecies(c.SpeciesId); sp != nil {
		c.ApplyIntrinsicMutations(sp)
	}
	c.MutationProgress = 0
	if charges < 0 {
		charges = 0
	}
	c.MutationRerollBonus = charges
}
```

- [ ] **Step 5: Run — expect PASS.** `go test ./internal/characters/ -run TestScourMutations_ClearsAcquired -v` → PASS.
- [ ] **Step 6: Commit.** `git add internal/characters/character.go internal/characters/mutation_scour.go internal/characters/mutation_scour_test.go && git commit -m "feat(mutations): ScourMutations + MutationRerollBonus field (moon-crash remort)"`

## Task 2: Bias re-acquisition toward rare while charges remain

**Files:**
- Modify: `internal/characters/bloom_mutation.go` (the `w := spec.Rarity` weighting, ~line 99)
- Test: `internal/characters/mutation_scour_test.go`

- [ ] **Step 1: Write the failing test** — with charges, the rare pool is favored and a charge is consumed on a successful seed:
```go
func TestBloomSeedNewMutation_RerollBonusFavorsRareAndConsumesCharge(t *testing.T) {
	c := New()
	c.SpeciesId = 1
	c.Mutations = map[string]int{}
	c.MutationRerollBonus = 2
	id, lvl := c.BloomSeedNewMutation(nil)
	if id == "" || lvl == 0 {
		t.Fatalf("expected a mutation to be seeded")
	}
	if c.MutationRerollBonus != 1 {
		t.Fatalf("expected one reroll charge consumed, got %d", c.MutationRerollBonus)
	}
}
```
(Statistical rare-bias is asserted in Step 4's follow-up test; this test locks the charge-consume contract.)

- [ ] **Step 2: Run — expect FAIL** (charge not consumed): `go test ./internal/characters/ -run TestBloomSeedNewMutation_RerollBonus -v` → FAIL.

- [ ] **Step 3: Implement** — in `BloomSeedNewMutation`, change the weight line and consume a charge on a successful grant. Replace:
```go
		w := spec.Rarity
		if w < 1 {
			w = 1
		}
```
with:
```go
		w := spec.Rarity
		if w < 1 {
			w = 1
		}
		if c.MutationRerollBonus > 0 {
			w = w * w // post-scour: square the rarity weight, biasing hard toward rare
		}
```
Then, just before the function returns a successful `(id, 1)` grant (after the mutation is added to `c.Mutations`), add:
```go
	if c.MutationRerollBonus > 0 {
		c.MutationRerollBonus--
	}
```
(Place the decrement only on the success path — not on the `return "", 0` no-candidate path.)

- [ ] **Step 4: Add a statistical-bias test** proving charges shift the distribution toward rare:
```go
func TestRerollBonus_ShiftsDistributionTowardRare(t *testing.T) {
	rareCountWith, rareCountWithout := 0, 0
	for i := 0; i < 400; i++ {
		c := New(); c.SpeciesId = 1; c.Mutations = map[string]int{}; c.MutationRerollBonus = 1
		if id, _ := c.BloomSeedNewMutation(nil); id != "" {
			if spec := getMutationSpecForTest(id); spec != nil && spec.Rarity >= 6 { rareCountWith++ }
		}
		c2 := New(); c2.SpeciesId = 1; c2.Mutations = map[string]int{}
		if id, _ := c2.BloomSeedNewMutation(nil); id != "" {
			if spec := getMutationSpecForTest(id); spec != nil && spec.Rarity >= 6 { rareCountWithout++ }
		}
	}
	if rareCountWith <= rareCountWithout {
		t.Fatalf("reroll bonus should favor rare mutations: with=%d without=%d", rareCountWith, rareCountWithout)
	}
}
```
Add a small test helper in the test file: `func getMutationSpecForTest(id string) *mutations.MutationSpec { return mutations.GetMutation(id) }` (import `internal/mutations`). (If the global-rand path is not deterministic enough at 400 samples, raise the iteration count; the squared weighting makes the shift large.)

- [ ] **Step 5: Run — expect PASS** (both tests). Also run the existing Bloom tests to confirm no regression: `go test ./internal/characters/ -run "Bloom|Reroll|Scour" -v`.
- [ ] **Step 6: Commit.** `git commit -am "feat(mutations): post-scour reroll bonus biases re-acquisition toward rare"`

## Task 3: The scour potion item + the drink effect

**Files:**
- Create: the potion item YAML under `_datafiles/world/dogmud/items/consumables-30000/` (or `materials`/`potions` per convention — confirm the drinkable path; Bloom Wafer is `40108` under materials — match wherever drinkable potions live). Run `python tools/id_inventory.py --type items` for the next free id.
- Modify: `internal/usercommands/drink.go` (add the scour effect branch)
- Test: `internal/usercommands/` (a drink-scour test if the harness supports it; else in-game verify)

- [ ] **Step 1: Confirm the drinkable-effect hook.** Read `internal/usercommands/drink.go` — find how special potions map to effects (Bloom Wafer 40108 → the Bloom dose). Identify the itemid→effect dispatch. (Bloom uses a hook keyed on the item; the scour uses the same pattern.)

- [ ] **Step 2: Author the potion YAML** — an "Unmaking Draught" (or similar): a drinkable whose effect is the scour. Model on the Bloom Wafer's item structure. Include a strong warning in the description (drinking it strips all your Chrysalis-change). `type` per the drinkable convention. **Gate:** the effect only fires when the drinker is inside the ship (has the suppression flag / is in the Crash Site zone) — the drink handler checks this and refuses elsewhere with a message ("Nothing happens — out here the Chrysalis holds too fast to shed."). (The suppression flag/zone check is provided by the zone plan; for now gate on zone name == the crash-site zone, or a `truth`/ship flag — the zone plan finalizes the exact gate.)

- [ ] **Step 3: Wire the drink effect** — in `drink.go`, add a branch for the scour potion's itemid that: verifies the ship-gate, then calls `user.Character.ScourMutations(N)` (N = the reroll charges, e.g. 3 — a config knob `Balance.MutationScourCharges`), sends a dramatic message (the change sloughs away), and (per the "return stronger" design) leaves the charges to bias re-acquisition as the player plays on after leaving. Consume the potion.

- [ ] **Step 4: Config knob** — add `MutationScourCharges` (default 3) to the Balance config (`internal/configs/config.balance*.go`) so the reroll strength is tunable.

- [ ] **Step 5: Build + boot-verify** — `go build`, boot, confirm the item loads and (via the mudagent harness or a unit test) drinking it inside the ship scours mutations + grants charges, and drinking it elsewhere is refused. `go test ./internal/characters/ ./internal/usercommands/`.

- [ ] **Step 6: Commit.** `git commit -am "feat(mutations): scour potion (moon-crash remort) + ship gate + config knob"`

## Task 4: Persistence + safety

- [ ] **Step 1:** Confirm `MutationRerollBonus` persists (yaml tag added in Task 1) — save/reload a character in a test or in-game, verify the charge count survives.
- [ ] **Step 2:** Guard against edge cases with a test: scouring with 0 owned mutations still grants charges + resets progress; scouring twice re-sets charges (does not stack unboundedly — `ScourMutations` sets, not adds). Confirm the existing `TestScourMutations` covers the empty case; add a double-scour test if not.
- [ ] **Step 3:** Run the full `internal/characters` + `internal/usercommands` + `internal/mutations` suites green. Commit any fixes.

---

## Self-review notes
- **Spec coverage:** the mutation-scour reward (spec §6 "THE MUTATION-SCOUR") — ScourMutations (T1) + return-stronger reroll bias (T2) + the potion delivery gated to the ship (T3) + tunable charges (T3.4) + persistence (T4). ✓ The "chamber" alternative delivery (a room_interact instead of a potion) is deferred to the zone plan (it wires the same `ScourMutations` call from a room interaction).
- **Deferred to the zone plan (Plan B):** the exact ship-gate (suppression flag / zone check), the chamber delivery, the potion's drop placement + rarity, and balance of N.
- **Type consistency:** `ScourMutations(charges int)`, `MutationRerollBonus int`, weight-squaring in `BloomSeedNewMutation` — consistent across tasks.
- **Risk:** the statistical bias test (T2.4) depends on the rarity spread of the mutation pool; if flaky, raise iterations — the squared weight makes the effect large, so a clear separation should hold.
