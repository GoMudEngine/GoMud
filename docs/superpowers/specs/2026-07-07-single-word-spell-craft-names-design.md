# Single-Word Invocation for Spells & Recipes (aliases + parser)

**Date:** 2026-07-07
**Status:** Design approved (brainstorm), pending plan.
**Scope:** Let players invoke every spell and craft recipe by a short single word
(e.g. `cast ward` instead of `cast conviction-ward`) via curated **aliases**, and
upgrade the `cast` parser to also accept full **multi-word** names — all with zero
ripple to existing references (no rename, no save migration).

## Motivation

Spell and recipe invocation tokens are hyphenated (`conviction-ward`,
`kinetic-shove`, `anti-corrosion-quench`) because the parser treats the invocation
as a single whitespace-delimited token. Hyphenated names are annoying to type and
leak a database-key look into play. The player wants short, natural invocation.

A true *rename* of the canonical ids is high-risk: the hyphenated id is referenced
by player-save spellbook keys, existing player aliases, dialogue `grantsSpell`,
quest rewards, mob spellbooks, help filenames, and the YAML filenames — a rename
needs a player-save migration and touches all of it. The **alias** approach
delivers the same "type a short word" win with **zero ripple**.

## Current state (verified 2026-07-07)

- **Spells:** `internal/spells/spells.go` — `SpellData` struct has
  `SpellId string` (yaml `spellid`, the invocation token + filename base) and
  `Name string` (display). 59 spells in `_datafiles/world/dogmud/spells/`,
  filename `{spellid}.yaml`. Lookup: `GetSpell(spellId)` (exact, :141),
  `FindSpellByName(spellName)` (:148, fallback). Player saves store
  `spellbook: {conviction-ward: <uses>}` keyed by spellid.
- **Recipes:** `_datafiles/world/dogmud/recipes/<discipline>/<id>.yaml` — `id:`
  (invocation token) + `name:` (display). 126 recipes. Lookup via
  `crafting.FindRecipeByName`.
- **`cast` parser** (`internal/usercommands/skill.cast.go`): `parts :=
  strings.SplitN(rest, " ", 2)` → `spellName = parts[0]` (FIRST token only),
  `targetName = parts[1]`. This is why spell names must be single-token
  (hyphenated). Resolves via `GetSpell` then `FindSpellByName`.
- **`craft` parser** (`internal/usercommands/craft.go:32-45`): ALREADY does
  greedy multi-word matching — tries `FindRecipeByName(rest)`, then progressively
  shortens the candidate (recipe-name vs trailing item-target). So the recipe
  side's parser mostly works already; the gap is only that recipe names are
  hyphenated (and no short alias).

## Design

### 1. Alias field
- Add `Aliases []string` (yaml `aliases,omitempty`) to `SpellData`. First entry
  is the primary short form; a spell may list a few (e.g. `[regen, cr]`).
- Add the equivalent `aliases []string` to the recipe struct/YAML.
- Canonical `spellid` / recipe `id` are **unchanged** — aliases are additive.

### 2. Curated single-word aliases (the content)
- Author a **collision-free single word for all 59 spells + 126 recipes**
  (e.g. `conviction-ward`→`ward`, `chrysalis-glow`→`glow`, `kinetic-shove`→`shove`,
  `conjure-fire`→`fire`, `anti-corrosion-quench`→`quench`).
- **Spells and recipes are SEPARATE namespaces** (`cast` vs `craft`), so a word may
  appear in both (e.g. a `heal` spell and a `heal` potion recipe are fine). Within
  each namespace, aliases must be unique AND must not collide with any canonical
  `spellid`/recipe `id` in that namespace.
- Generated for review; the player approves the alias list before build-out.

### 3. Resolution
- Extend spell lookup so `cast <x>` resolves `x` against, in order: exact
  `spellid` → alias → full display name (lowercased, space-normalized). Add an
  index `spellsByAlias` built at load (like `spellsById`), and have
  `FindSpellByName` also match the display name with spaces.
- Extend `crafting.FindRecipeByName` the same way (match `id`, aliases, and the
  multi-word display name).
- Result — all three forms work for every spell/recipe:
  `cast ward` (alias) · `cast conviction-ward` (canonical, back-compat) ·
  `cast conviction ward` (multi-word display).

### 4. Cast parser upgrade (greedy longest-match)
- Replace `skill.cast.go`'s first-token split with the same progressive-shortening
  strategy `craft.go` already uses: try the whole `rest` as a spell name; if no
  match, drop the last word and retry, until a spell resolves — the leftover
  trailing words are the target. So `cast conviction ward on goblin` → spell
  `conviction-ward`, target `on goblin` (existing target resolution already
  tolerates a leading `on`/`at`; verify).
- Keep the fast path: a single-token input still resolves in one lookup.

### 5. Validator (startup panic)
- At spell load, build the alias index and **panic on any duplicate** within the
  spell namespace (alias == another alias, or alias == some spellid). Mirror the
  quest-flag `ValidateAllFlags` pattern. Same for recipes at recipe load.
- This is the pre-push boot-test safety net — collisions/typos fail fast.

### 6. Discoverability
- Show the alias in the `spells` list (e.g. `Conviction Ward (ward)`) and in
  `help <spell>` / the per-spell help so players learn the short form.
- The `cast` / `craft` "not found" error suggests the closest alias if there is a
  near-match.

### 7. Zero ripple (the whole point)
- Canonical `spellid` / recipe `id`, filenames, player-save spellbook keys,
  existing player aliases (`cw: conviction-ward`), dialogue `grantsSpell`, quest
  reward grants, mob spellbooks, and help filenames are all **untouched** — they
  keep working exactly as today. No data migration.

## Build order

1. Engine: `Aliases` field on `SpellData` + recipe struct; the alias index +
   resolution in `GetSpell`/`FindSpellByName` + `FindRecipeByName`; the load-time
   uniqueness validators. (TDD.)
2. Engine: the `cast` greedy-longest-match parser upgrade. (TDD + in-game verify.)
3. Content: curate + add the 59 spell aliases (review the list), then the 126
   recipe aliases. Boot-validate (the new validator).
4. Discoverability: `spells` list + `help` show the alias; not-found suggestion.
5. Verify: full suite + boot + in-game (`cast ward`, `cast conviction ward`,
   `cast conviction-ward` all cast; `craft <alias>`; a deliberate duplicate alias
   panics the boot).

## Out of scope
- Renaming any canonical `spellid` / recipe `id` (aliases only).
- Changing display `name:` fields (immersive names stay).
- Player-save migration (not needed — canonical keys unchanged).
- Aliases for anything other than spells + craft recipes.

## Open items for the plan
- The exact alias field name (`aliases`) + whether recipes reuse the same yaml key.
- The precise not-found "did you mean" suggestion (Levenshtein vs prefix) — or
  skip it for v1 (nice-to-have).
- Whether `cast`'s target parser needs any change to strip a leading `on`/`at`
  (verify current behavior; likely already handled).
- The curated alias list itself (generated in the plan/build; player-reviewed).
