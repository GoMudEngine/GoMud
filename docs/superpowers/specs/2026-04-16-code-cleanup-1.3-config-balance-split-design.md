# Code Cleanup 1.3: Config Balance File Split — Design Spec

## Goal

Split `internal/configs/config.balance.go` (949 lines) into 6 domain
files by extracting the monolithic `Validate()` method into themed
validator functions. The `Balance` struct itself stays unchanged in
a single file.

Pure structural change — zero behavior impact.

## Scope

**In scope:**
- Extract `Validate()` method logic into 6 domain files
- Each domain file contains one `validate<Domain>()` method on
  `*Balance`
- `Validate()` becomes a thin dispatcher

**Out of scope:**
- Any change to `Balance` struct fields, order, or YAML tags
- Any change to default values
- Any change to YAML config format
- Struct splitting (Go doesn't allow) or sub-struct nesting
- Changes to `GetBalanceConfig()` accessor

## Architecture

### Why not split the struct?

Go doesn't allow splitting a struct definition across files. Nesting
sub-structs (`Balance.Combat`, `Balance.Progression`, etc.) would
break the YAML config format — every user's config.yaml would need
restructuring. Pure validation split preserves the format.

### File layout

**`config.balance.go` (main file, ~340 lines)**
- `Balance` struct definition with all fields (unchanged)
- `Validate()` — thin dispatcher
- `GetBalanceConfig()` accessor (unchanged)
- `GetStatProgressionMultiplier()` helper
- `GetSkillProgressionMultiplier()` helper

Validate dispatcher:

```go
func (b *Balance) Validate() {
    b.validateCombat()
    b.validateProgression()
    b.validateSpells()
    b.validateMobs()
    b.validateShops()
    b.validateMisc()
}
```

### Domain files

Each file contains one `validate<Domain>()` method on `*Balance`
that sets defaults for its domain's fields. Extracted verbatim from
the current `Validate()` method body.

**`config.balance.combat.go`** — `validateCombat()`
- RollSpread
- Defense costs (dodge/parry/block stamina)
- Defense effectiveness (multipliers, caps)
- Prone & grapple (penalties, multipliers)
- Special moves (cooldowns, damage percentages)
- Skullduggery (steal, shadow cooldowns)
- Darkness combat penalty
- Combat messages (ConsistentAttackMessages)
- Damage channels & scales (physical/magical/conviction mitigation,
  melee/spell/rhetoric damage scales)
- Resource depletion penalties (curve, per-pool max)
- Toxicity (base max, decay, vitality scale, penalties)

**`config.balance.progression.go`** — `validateProgression()`
- Skill soft cap, progression chance params
- Base progression chance, decay below/above cap
- Stat soft cap, threshold, multiplier
- Per-stat progression multipliers
- Per-skill progression multipliers
- Uses per rank, progression weights
- Mutation progress scale, max count/level, deepen chance,
  level multipliers, regen progression
- Regen-based stat progression base/curve

**`config.balance.spells.go`** — `validateSpells()`
- Spell conviction/health cost multipliers
- Self-cast progression multiplier
- Spell concentration base, initiation base/skill/willpower
- Spell folds skill factor
- Spell damage scale, spell attack skill factor
- Spell avoidance damage multiplier
- Spell difficulty progression scale
- Spell discovery base chance, decay rate
- Spell proficiency casts per point
- Enchant (max tier, removal penalty, tier-up base chance,
  tier uses base/scale)

**`config.balance.mobs.go`** — `validateMobs()`
- Mob AI enabled, combat memory duration
- Mob reaction delay min/max
- Mob behavior tree reaction base/perception scale
- Mob health/stamina/conviction regen pct
- Mob damage multiplier
- Mob progression (enabled, rate), mob stat cap, skill cap,
  instance max age
- Mob mutation (enabled, rate)
- Pack scaling (enabled, max size, survival rounds, scatter,
  max bonus, bonus training pts)
- Pack roaming (enabled)
- Gossip interval rounds
- Moon phase stat mod max
- Manifestation stat scale (charisma factor, skill factor)

**`config.balance.shops.go`** — `validateShops()`
- Shop price floor/ceiling, buy ratio, abundance threshold,
  material reserve, gold reserve ratio
- Barter max discount/bonus
- Storage fee per item
- Crafter enabled, material restock rate, rare threshold
- Crafting success chances (base/min/max, skill bonus per level)
- Craft difficulty progression scale
- Recipe discovery base chance, decay rate

**`config.balance.misc.go`** — `validateMisc()`
- Player health/stamina/conviction regen pct
- Regen progression base/curve
- Health/stamina/conviction base, per-stat scaling
- Starting health, health penalty max, stamina/conviction penalty max
- Character creation (stat roll mean/std-dev/min/max)
- Salvage (min/max chance, soft cap, gold per round, max rounds)
- Quest engine (chain depth limit, log level, performance warn ms)
- World event buffer size
- Carry capacity multiplier, min defense chance, min attack hit chance
- Movement stamina (base/max cost)
- Stand min stamina, stand stamina cost
- Global damage multiplier, haste swing multiplier
- Skill multiplier base/max
- Third-party grapple penalty
- Unarmed stats (base damage, variance, strength/skill divisors,
  speed multiplier, attack stamina cost)
- Bash/kick/stomp/knee/trip (damage %, knockdown chance)
- Surprise attack penalties (offhand, extra arms)
- Enchant removal penalty rounds (if not placed in spells)
- Coup de grace rounds
- Clinch penalties (block, dodge, parry)
- Grounded penalties (block, dodge, parry, damage)
- Loot budget scalar
- Instance stat pool cap

### Method signatures

All domain validators follow the same signature:

```go
func (b *Balance) validateCombat() {
    if b.RollSpread <= 0 {
        b.RollSpread = 0.15
    }
    // ... rest of combat defaults ...
}
```

Package-private (lowercase) since only called from `Validate()`
within the same package.

### Field mapping source of truth

For each field, the correct domain is determined by its section
comment in the current `Validate()` method. During extraction:

1. Read each `// ── SECTION NAME ──` comment block
2. Copy all `if b.X` default-setting logic from that section
3. Place in the matching domain file

Any field without a clear domain goes to `config.balance.misc.go`.

## Constraints

- **Zero behavior change.** Every default value, every field name,
  every YAML tag unchanged.
- **Struct stays in one place** — `config.balance.go` keeps full
  `Balance` definition.
- **All fields validated somewhere.** After extraction, the sum of
  `validateCombat()` + `validateProgression()` + ... must cover
  every field currently validated in `Validate()`.
- **Order within domain preserved.** Keep the existing
  `if b.X <= 0 { b.X = default }` order within each domain so diffs
  are easy to review.
- **Public API unchanged.** `Validate()`, `GetBalanceConfig()`,
  `GetStatProgressionMultiplier()`, `GetSkillProgressionMultiplier()`
  all keep the same signatures.

## Testing

After each domain extraction:

```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go build ./...
go test ./internal/configs/...
```

After all 6 domains extracted:

- Start the server locally — it must load config without errors
- Compare a `configs.GetBalanceConfig()` snapshot before and after
  (all fields should have identical values, assuming empty config
  overrides)
- Verify no orphaned `if b.X` defaults left in the main `Validate()`

## Risk Assessment

**Low risk.** This is a validation-logic extraction, not a struct
change. The struct is unchanged so field access patterns across
the codebase are unaffected.

The only way to break something: fail to extract a default-setting
`if b.X <= 0 { ... }` block and leave a field with a zero default.
Mitigated by:
- Line-by-line diff review of `Validate()` before/after
- Checking that the sum of lines in all new `validate<Domain>()`
  methods approximately equals the original `Validate()` line count
  (minus ~5 dispatcher lines)

## Execution Order

Each task extracts one domain. Order picks lower-risk domains first:

1. Extract `validateMisc()` → `config.balance.misc.go` (largest,
   most heterogeneous — do first while the old Validate() still has
   the clearest section comments as guideposts)
2. Extract `validateShops()` → `config.balance.shops.go`
3. Extract `validateSpells()` → `config.balance.spells.go`
4. Extract `validateMobs()` → `config.balance.mobs.go`
5. Extract `validateProgression()` → `config.balance.progression.go`
6. Extract `validateCombat()` → `config.balance.combat.go`
7. After all 6 extracted, `Validate()` should be ~8 lines (the
   dispatcher). Final verification: full build, test, manual server
   start.

Alternative order: do the smallest domain first to get a feel for
the extraction pattern, then scale up. Acceptable either way.
