# Game-wide naming casing convention + hardening — design

**Date:** 2026-06-09
**Status:** Approved (brainstorming) — pending spec review → writing-plans
**Scope note:** This is sub-project 1 of two independent immersion-polish efforts.
Sub-project 2 (non-human mob attack types / messaging) is specced separately.

## Problem

Player-visible names and titles have inconsistent capitalization, which reads
as goofy/sloppy and breaks immersion. Concrete examples:

- Player titles like **"human Ascendant master warrior"** — the mutation tier
  (`Ascendant`) is capitalized while skill tier (`master`) and archetype
  (`warrior`) are lowercase, and race (`human`) is lowercase.
- Mob/NPC names like **"temple priest Olen"** vs **"Temple Priest Olen"** —
  mob names are stored lowercase and blanket-Title-Cased only on the `mobname`
  display path (`titleCase()` in `internal/characters/formattedname.go`), so
  casing differs across surfaces (telnet vs GMCP/web, names embedded in prose,
  items, rooms).

Root causes (from code investigation 2026-06-09):

- `skills.GetTitle()` (`internal/skills/skills.go:247`) joins three pieces with
  mismatched stored casing: `GetMutationTier` returns Capitalized words,
  `GetSkillTier` and `GetStatArchetype` return lowercase words.
- `internal/characters/formattedname.go` has a local `titleCase()` that
  capitalizes **every** word and is applied only to `mobname`-typed names — not
  items, rooms, spells, buffs, or every surface.
- There is no single canonical casing function and no guardrail, so casing
  drifts as content/code is added.

## Decisions (from brainstorming)

1. **Convention: smart Title Case, everywhere.** Capitalize names/titles in
   title case. "Temple Priest Olen", "City Guard", "Iron Sword", "Ascendant
   Master Warrior". Chosen over the prose-natural proper-noun rule specifically
   because Title Case is *consistent by construction* and trivially enforceable
   — directly serving the "fix it throughout + harden against reappearing" goal.
2. **Smart, not naive:** short connecting words (articles, coordinating
   conjunctions, short prepositions) are lowercased when they are neither the
   first nor last word. "captain of the guard" → "Captain of the Guard".
3. **Scope: everything + prose/template audit.** All entity-name surfaces
   (player titles, mob/NPC names, item names, room titles, spell/buff names)
   across telnet AND GMCP/web, PLUS an audit of room descriptions, combat-message
   templates, and dialogue prose for hand-typed names / stray casing.
4. **Hardening: load-time validator that PANICS** on a non-canonical stored
   name (mirrors DOGMud's schedule/quest-flag/buff startup validators).

## Architecture

### 1. Canonical formatter — `internal/casing`

A new leaf package with one exported function:

```go
// Title returns s in smart title case: each word's first rune is upper-cased
// except minor words (articles/conjunctions/short prepositions) that are
// neither the first nor last word. Existing internal capitals are preserved
// (so "McGregor", "Olen", acronyms survive). Idempotent.
func Title(s string) string
```

- **Minor-word set** (lowercased when interior): `a an and as at but by for from
  in nor of on or the to with`. Tunable in one place; documented.
- **Always capitalize** the first and last word regardless of the set.
- **Preserve internal capitals:** transform only the first rune of each word;
  never down-case the remainder (`ToUpper(w[:1]) + w[1:]`). This keeps `McGregor`,
  `Olen`, and intentional mixed-case intact and makes the function idempotent.
- Word boundary = whitespace. Hyphens inside a single token are left to the
  token (we do not split `iron-touched`; only whitespace-separated words are
  treated as separate words). Punctuation-leading tokens (e.g. a quote) cap the
  first letter.
- **Pure, no dependencies** → unit-testable in isolation and importable
  anywhere without import cycles. This is the single source of truth; the sweep
  tool, the load validator, and runtime title assembly all call it.

`internal/characters/formattedname.go`'s local `titleCase()` is **removed**.
Because static names are stored canonical (§2), the mob-name display path shows
the **stored value as-is** — no per-render title-casing. `casing.Title` is the
shared transform used by the sweep, the load validator, and any **runtime-
assembled** display string (player titles in §3; and if affixed/enchanted item
names are assembled at display time, that assembly passes through `casing.Title`
too). This avoids a per-render transform on static names while still guaranteeing
assembled strings are canonical.

### 2. Canonical storage + one-time sweep

Entity **display names** are stored in canonical Title-Case form:

- **Targets:** `name`/display-name fields in mob, item, room (title), spell,
  and buff YAML across `_datafiles/world/dogmud/`.
- A one-time sweep runs `casing.Title` over each display-name field and rewrites
  the YAML. Idempotent, so re-running is safe.
- **Display of static names = as-stored** (already canonical). The removed
  `formattedname.titleCase` is not replaced with a per-render transform for
  static names; storage is the source of truth, the validator keeps it clean.

**Critical boundary — names only, not keys.** The convention applies ONLY to
player-visible name strings. It MUST NOT touch:

- lookup/targeting keys, noun tags, `namesimple`/keyword fields, item
  `component_tag`s, dialogue trigger keywords,
- filenames / `ConvertForFilename` output, YAML ids, zone folder names,
- anything used for parsing/matching.

Those stay lowercase/unchanged so targeting (`attack temple priest`, `get iron
sword`), filename derivation, and parsing are unaffected.

### 3. Player titles + race

- `skills.GetTitle` keeps its component words lowercase internally (lowercase
  the `GetMutationTier` constants so the source is uniform), assembles, and
  returns `casing.Title(joined)` → "Ascendant Master Warrior".
- Race: Title-Case where displayed (`characters.Species()` consumers / GMCP
  `Char.Info.Race`) → "Human". Identify the surface(s) that concatenated race +
  title (the "human Ascendant master…" string) and render both through the
  convention → "Human Ascendant Master Warrior" (or race in its own field/column
  where the layout already separates them — keep existing layout, fix casing).

### 4. Display routing

- Every name-bearing display surface uses the canonical value: telnet
  (look, who, room mob/item lists, combat lines, inventory, score/status),
  and GMCP/web (`Char.Info`, `Char.Inventory`, room/mob payloads).
- Names injected into prose/combat templates via tokens (`{source}`,
  `{target}`, `{itemname}`, mob/room name tokens) automatically carry the
  canonical (stored) value — no per-template work for token-injected names.

### 5. Load-time validator (the guardrail)

At startup, after each loader runs, validate every entity **display name**:
`name == casing.Title(name)`. On mismatch → **panic** with the offending
file/id, the bad value, and the expected canonical form. Runs for mobs, items,
rooms, spells, buffs.

- The sweep runs first, so a clean tree boots fine.
- Any future author who types a non-canonical name trips the panic immediately;
  the pre-push boot-test SOP catches it before prod.
- Validates display names only — never keys/tags/filenames (see boundary above).

### 6. Prose / template audit

Sweep for **hand-typed** entity names / stray casing that bypasses the
formatter:

- room descriptions, combat-message template literals, dialogue `text`/`hints`.
- Fix literals that name a specific entity to match the canonical form.
- **Not in scope:** title-casing prose *sentences*. Body prose keeps normal
  sentence capitalization; we only correct entity-name literals within it.

## Testing

- **TDD `casing.Title`:** first/last word always capitalized; interior minor
  words lowercased; internal capitals preserved (`McGregor`, `Olen`);
  idempotence (`Title(Title(x)) == Title(x)`); single word; empty string;
  leading punctuation; multi-space; already-canonical input unchanged.
- **Validator tests:** canonical names pass; a non-canonical name panics with a
  helpful message.
- **Sweep idempotence test** (running the sweep twice is a no-op).
- **Boot test** post-sweep: validator green, no panics.
- Manual spot-check: look / who / combat / inventory / GMCP+web show consistent
  casing; targeting still works (`attack temple priest`, `get iron sword`).

## Rollout order

1. Land `internal/casing` + tests (TDD).
2. Repoint `formattedname` + `skills.GetTitle` + race display to `casing.Title`.
3. Run the data sweep (entity display names) — review the diff.
4. Add the load-time validator (panic).
5. Prose/template literal audit + fixes.
6. Boot test; spot-check displays; push per SOP.

## Out of scope

- **Non-human mob attack types / messaging** — the second immersion problem;
  separate spec + plan.
- Proper-noun/sentence-case conventions (rejected in favor of Title Case).
- Title-casing prose sentences (only entity-name literals are corrected).
- Renaming lookup keys, tags, filenames, or ids.

## Risks / watch-items

- **Over-reach into keys:** the biggest risk is the sweep or validator touching
  a lookup/parse field. Mitigation: explicit allowlist of *display-name* fields
  per loader; never touch keys/tags/filenames; targeting smoke-test.
- **Smart-title minor-word edge cases** (a name that is *only* minor words, or a
  proper name that happens to be a minor word) — first/last-word-always-cap rule
  covers the common cases; unit tests pin the rest.
- **Stored mixed-case proper names** (`McGregor`) — preserved by transforming
  only the leading rune; covered by tests.
- Player-title race-concatenation surface must be located during implementation
  (investigation flagged it but did not pin the exact template/command).
