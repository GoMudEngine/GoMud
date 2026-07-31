# Enchantments Context

## Purpose

`internal/enchantments` owns authored enchantment definitions and the act of
stamping one onto an item. An enchantment is a **tiered** stat-mod package plus
a name adjective: applying it renames the item and merges the tier's stat mods
into it.

## Files

- **enchantments.go** — `EnchantmentDef`, `TierDef`, the registry, apply/strip.
- **test_helpers.go** — `SeedEnchantmentsForTest`.

## Types

```go
type TierDef struct        { /* per-tier adjective, stat mods, reserve pct */ }
type EnchantmentDef struct { /* EnchantId, type, tiers */ }
```

## API

```go
func LoadEnchantmentFiles()
func GetEnchantment(id string) *EnchantmentDef
func GetAll() map[string]*EnchantmentDef

func (e *EnchantmentDef) Id() string
func (e *EnchantmentDef) Filepath() string
func (e *EnchantmentDef) Validate() error

func ApplyTier(item *items.Item, def *EnchantmentDef, tier int)
func StripEnchantment(item *items.Item)
func GetTierReservePct(enchantType string, tier int, hands ...int) float64
func SeedEnchantmentsForTest(defs map[string]*EnchantmentDef) func()
```

`GetTierReservePct` takes a variadic `hands` because a two-handed weapon
reserves a different share of the material budget than a one-hander for the
same tier.

## Apply / strip symmetry

`ApplyTier` records enough on the item (enchant type and tier) for
`StripEnchantment` to reverse it — removing the adjective and the merged stat
mods. `copyStatMods` deep-copies rather than aliasing, so two items enchanted
from the same definition do not share a mutable stat-mod map.

## Gotchas

- **`ApplyTier` mutates the item in place** and does not validate the tier
  index against the definition. An out-of-range tier is the caller's bug.
- **Strip relies on the adjective being recognisable** (`isEnchantAdjective`).
  Renaming an item by hand after enchanting it can leave the adjective
  unstrippable.
- **Stat mods are merged, not replaced.** Applying two enchantments stacks
  them; the system assumes one at a time.
- **`GetEnchantment` returns nil for an unknown id.** Check it — a typo in a
  recipe otherwise panics at the apply call.

## Dependencies

`items`, `statmods`, `configs`, `mudlog`, plus the fileloader.

## Consumers

`internal/crafting` (the enchanting recipes), `internal/items`, and the
identify/appraise display paths.
