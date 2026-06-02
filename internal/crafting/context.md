# Crafting Package Context

## Overview

The `internal/crafting` package implements the data-driven recipe and
crafting framework (Stage 13.1) and the corpse-salvage lookup table
(cooking supply chain, chunk 5.4). It is a pure-logic package: it
owns no persistent state and emits no side effects — callers in
`internal/usercommands/` and `internal/hooks/` drive the actual item
transfers and player messaging.

Two distinct systems live here:

1. **Recipe crafting** — players and NPC crafters combine tagged
   ingredients at optional stations to produce output items.
2. **Corpse salvage** — salvaging a mob corpse yields materials
   determined by the mob's group membership (rodent → wild-hare-meat;
   animal → raw-meat; humanoid → cloth/leather).

The corpse-salvage system feeds the cooking supply chain: foragers
salvage animal/rodent corpses, the yielded meat flows into forager
lockboxes, and `BackfillVendorFromChests` (in `internal/forager/`)
drains those chests into cook-vendor stock.

## Key Components

### Core Files

- **crafting.go** — `RecipeSpec` struct + `fileloader.Loadable`
  implementation; `LoadRecipes`, `GetRecipe`, `GetRecipesForSkill`,
  ingredient/output access helpers.
- **validation.go** — `ValidateRecipe` checks ingredient tags against
  the item registry; called at load time and by integration tests.
- **salvage.go** — pure math helpers: `CalcSalvageChance` (sqrt-curve
  per-ingredient recovery probability), `CalcSalvageRounds` (duration
  from ingredient gold value), `RollSalvageReturns` (per-unit
  Bernoulli roll over an ingredient list).
- **corpse_salvage.go** — static `corpseSalvageTable` + public
  `LookupCorpseSalvage(groups []string) []items.SalvageReturn`.
- **crafting_test.go** — recipe-load + GetRecipe unit tests.
- **salvage_test.go** — CalcSalvageChance / CalcSalvageRounds /
  RollSalvageReturns unit tests.
- **corpse_salvage_test.go** — LookupCorpseSalvage unit tests
  covering each table entry + no-match + first-entry-wins ordering.
- **validation_test.go** — ValidateRecipe unit tests.
- **integration_crafting_test.go** — end-to-end recipe-load
  integration test (loads YAML from disk).
- **test_main_test.go** — test-binary setup (data-file path
  initialization).

## Key Functions

### Recipe Crafting

- **`LoadRecipes(dir string) error`** — walks `dir` recursively,
  loads all `*.yaml` recipe files, validates each, populates the
  in-process registry. Called from `main.go` at startup.
- **`GetRecipe(id string) (*RecipeSpec, bool)`** — look up a recipe
  by its string id.
- **`GetRecipesForSkill(skill string) []*RecipeSpec`** — return all
  recipes belonging to a skill tag (e.g., `"cooking"`), sorted by
  name for stable UI ordering.

### Salvage Math

- **`CalcSalvageChance(skill, minChance, maxChance, softCap)`** —
  returns a probability in `[minChance, maxChance]` via
  `min + (max - min) × sqrt(clamp(skill,1,softCap) / softCap)`.
  Config knobs: `SalvageMinChance` (0.15), `SalvageMaxChance`
  (0.85), `SalvageSoftCap` (50).
- **`CalcSalvageRounds(totalGoldValue, goldPerRound, maxRounds)`** —
  duration = `max(1, min(maxRounds, goldValue / goldPerRound))`.
- **`RollSalvageReturns(ingredients, chance)`** — per-unit Bernoulli
  roll; returns only recovered items (non-zero quantity).

### Corpse Salvage

- **`LookupCorpseSalvage(groups []string) []items.SalvageReturn`** —
  scans `corpseSalvageTable` in declaration order; returns the
  `Returns` slice of the first entry whose `Group` string appears in
  the mob's groups list. Returns `nil` when no entry matches (e.g.,
  elementals, chrysalis mobs).

  Table order (specific before broad):

  | Group | Yields |
  |-------|--------|
  | `rodent` | wild-hare-meat ×1, leather-strip ×1 |
  | `animal` | raw-meat ×1, leather-strip ×2, sinew ×1 |
  | `humanoid` | cloth-strip ×2, leather-strip ×1 |

  To add a new mob category, prepend its entry before `animal` if it
  should take priority over the generic-animal row, or append after
  `humanoid` for unmatchable fallbacks. The `rodent` entry sits before
  `animal` so small-game mobs with both groups match the narrow row.

## Global State

- **`recipeRegistry map[string]*RecipeSpec`** — in-process map keyed
  by `RecipeSpec.RecipeId`; populated at startup by `LoadRecipes`.
  Read-only after load; no mutex needed.
- **`corpseSalvageTable []corpseSalvageEntry`** — package-level
  slice; declared statically in `corpse_salvage.go`, never mutated
  at runtime.

## Data Structure Design

### RecipeSpec (YAML)

```yaml
id: grilled-meat
name: Grilled Meat
skill: cooking
skill_minimum: 0
station: ""          # "" = no station required
time_rounds: 2
ingredients:
  - item_tag: raw-meat
    quantity: 1
output:
  item_id: 40060
  quantity: 1
success_message: "You grill the meat over the fire."
failure_message: ""
```

Optional enchanting fields: `target_type`, `enchant_type`.

### corpseSalvageEntry (Go struct, not YAML)

```go
type corpseSalvageEntry struct {
    Group   string
    Returns []items.SalvageReturn  // {ItemTag string, Quantity int}
}
```

## Integration Notes

**Consumers of recipe crafting:**
- `internal/usercommands/craft.go` — player `craft` command; calls
  `GetRecipe`, resolves ingredients from backpack + equipped items,
  drives the multi-round activity, calls `RollSalvageReturns` for
  salvage recipes.
- `internal/hooks/TickMobCraft*.go` — NPC crafter tick; calls
  `GetRecipesForSkill` to pick recipes, manufactures items into the
  NPC's shop stock.

**Consumers of corpse salvage:**
- `internal/usercommands/salvage.go` — player `salvage <corpse>`
  command; calls `LookupCorpseSalvage(mob.Groups)`, then
  `RollSalvageReturns`.
- `internal/actions/salvage.go` — shared `actions.Salvage` called by
  both player and mob salvage paths; same lookup chain.

**Upstream dependencies:**
- `internal/items` — `SalvageReturn`, `ItemSpec`, `component_tag`
  resolution.
- `internal/fileloader` — generic YAML walker used by `LoadRecipes`.
- `internal/configs` — balance knobs read by callers (not by this
  package directly).

## Testing Notes

- All pure-math functions (`CalcSalvageChance`, `CalcSalvageRounds`,
  `RollSalvageReturns`) are covered by table-driven unit tests in
  `salvage_test.go`.
- `LookupCorpseSalvage` tests in `corpse_salvage_test.go` cover each
  table row, the no-match case, nil/empty input, and the
  first-entry-wins ordering guarantee (a mob with both `animal` and
  `humanoid` groups gets the `animal` row; a mob with both `rodent`
  and `animal` groups gets the `rodent` row).
- `integration_crafting_test.go` loads YAML from disk — requires
  the test binary to be run from the repo root or with the data path
  initialized by `test_main_test.go`.
- Adding a new `corpseSalvageTable` entry requires a matching test
  case in `corpse_salvage_test.go`.
