# Unified Parser Seam — Design Spec

**Date:** 2026-07-08
**Status:** Approved for staging (each stage gets its own implementation plan)
**Author:** Calabe Davis + Claude

## Problem

DOGMud's command parsers each hand-roll their own target-resolution ladder.
`get.go` tokenizes, strips `from`/`stash`/`ground`, then tries bag → bandolier →
room-container → pet → corpse → floor → noun in ~150 lines of bespoke glue —
which is exactly where the 2026-07-08 corpse-loot bug hid (the `get all corpse`
and `get <item> corpse` syntaxes were never wired into the ladder). `look.go`
has its own ladder (player → mob → exit → corpse → item → noun). `give` grew a
bespoke greedy splitter (`splitGiveArgs`/`giveObjectResolves`/`giveTargetResolves`);
`cast` and `craft` (`FindRecipeByName`) each grew their own greedy longest-match
matcher for the single-word-names work.

So the matching *primitives* are fine and plentiful — `util.SplitButRespectQuotes`
(quote-aware tokenizer), `util.FindMatchIn` (exact→prefix→substring fuzzy match),
`util.GetMatchNumber` (`2.dagger` / `all.item` disambiguation),
`util.StripPrepositions`, and per-type resolvers (`room.FindByName`,
`room.FindOnFloor`, `room.FindContainerByName`, `room.FindCorpseIndex`,
`room.FindNoun`, `spells.ResolveSpell`, `crafting.FindRecipeByName`). What's
missing is **one place that composes them.** Each command re-implements the
composition, and each copy is a place the composition can be wrong or incomplete.

Two consequences:

1. **Player-facing:** natural multi-word input fails inconsistently. `look hare
   paths`, `get lake iron nodule`, `attack bank clerk` may or may not resolve
   depending on which command's bespoke ladder you hit, and content leans on a
   hyphenated-noun crutch (`hare-paths`) to dodge the gap — a database key
   leaking into prose.
2. **Maintenance:** every new command re-derives the ladder, and bugs like the
   corpse-loot gap recur because there's no single tested seam.

## Goals

- **C (primary):** one shared, unit-tested resolution seam that every
  target-taking command routes through, so the per-command ladder — and its bug
  surface — stops being re-invented.
- **A (falls out of C):** natural multi-word input resolves uniformly across
  commands (`look hare paths`, `get lake iron nodule`, `attack bank clerk`),
  with both the multi-word and existing hyphenated forms working.
- **B (fallout):** admin commands stop requiring numeric template IDs for
  multi-word mobs/players (`knowledge show bank clerk <player>`).

## Non-Goals

- **No content sweep / de-hyphenation.** Existing hyphenated nouns, item names,
  `component_tag`s, and prose stay exactly as authored; both forms resolve
  forever. (De-hyphenating the world is a separate, much larger content project.)
- **No new interactive disambiguation UX.** The resolver preserves today's
  deterministic pick (exact→prefix→substring, first/priority match, `N.item`
  selection). A "did you mean…?" prompt is out of scope; the design leaves a
  hook for it but does not build it.
- **No grammar DSL.** We are not building a general command-grammar parser
  (that was the rejected Approach 2). Two-slot commands split on the connective
  preposition and call the resolver per side.
- **No behavior changes** beyond "multi-word now resolves where it previously
  failed." Every migration is refactor-only for existing inputs.

## Divergences Discovered During Implementation

Recorded as stages land, so the spec stays honest about what we learned.

### After Stage 0 (foundation) + reading the live commands — 2026-07-08

**The "A" (multi-word) gap is far narrower than this spec assumed.** Most
multi-word input *already resolves* via the existing fuzzy matchers, because
single-target commands pass the whole `rest` to a substring/prefix matcher:

- `attack bank clerk` already works — `actions.FindAttackTarget` →
  `room.FindByName` → `util.FindMatchIn("bank clerk", ["Bank Clerk#1"])`
  (prefix match, `rooms.go:1819`).
- `get lake iron nodule` already works — the floor path passes the whole `rest`
  to `FindOnFloor` → `FindMatchIn` (substring match on the item name).
- `look hare paths` already works — `FindNoun` gained multi-word aliasing +
  `SplitButRespectQuotes` since the memory behind this spec was written.

**What actually breaks is ladder *composition*, not multi-word matching** — the
cases where the parser must *split* input into two roles: `get <item>
<container>` / `get <item> <corpse>` (which trailing span is the container — the
exact site of the 2026-07-08 corpse-loot bug), and two-slot commands (`give
<item> to <mob>`, admin `<mob> <player>`).

**Consequence — the project is re-scoped to composition-only** (user decision,
2026-07-08):

- **In scope (composition-heavy):** `get` (container/corpse split), `give`
  (item→actor split), admin two-slot lookups (`knowledge`/`opinion`/`crime`).
  The seam's value here is **C** (kill the duplicated ladders + the composition
  bugs they hide), not A.
- **Out of scope now (YAGNI):** migrating pure single-token-fuzzy commands
  (`attack`, `consider`, `target`, single-item `get`/`drop`, `look <noun>`) —
  they already work; migrating them is a pure refactor with regression risk and
  ~zero user benefit. The revised staging below drops those.

**Carried refinements from Stage 0:** `KindBackpackItem`+`KindEquippedItem` →
single `KindInventoryItem` (`character.FindItem` returns the combined pool);
`lootFromContainer`'s pet-inventory branch was deferred and is completed in the
`get` migration.

### Revised staging (supersedes the table in "Staged Decomposition")

| Stage | Scope | Status |
|-------|-------|--------|
| 0 — Foundation | `internal/parser` package | ✅ done (master) |
| 1 — `get` composition | Route `get`'s container/corpse detection through `SplitTrailingContainer`; retire that ladder; gates stay in the command. | ✅ done (master) — also fixed a latent multi-word-container bug |
| 2 — Admin two-slot | multi-word mob lookup via a scope-agnostic `SplitLeadingMatch` helper. | ✅ done (master) — `knowledge` + `opinion` fixed (greedy split + `ConvertForFilename`-normalized ident). `crime`/`faction` verified NOT affected (faction-slug + single-token player, no mob names). |
| 3 — Convergence | Retire dead bespoke matchers; document the un-hyphenated authoring convention. | ✅ done (master) — "retire dead matchers" was a **no-op** (verified: `splitGiveArgs`/`FindRecipeByName`/`ResolveSpell` are all still live & working; `go vet` flags nothing dead). Delivered the authoring-convention doc in `CLAUDE.md` ("Command Parsing & Multi-Word Input"). |

**Project status (2026-07-08): COMPLETE.** Stage 0 (foundation) + Stage 1
(`get`, which fixed a real player-facing multi-word-container bug) + Stage 2
(admin knowledge/opinion) + Stage 3 (docs) are on `master`. Everything else the
original spec proposed was verified to already work and was dropped. Net result:
a reusable `internal/parser` seam + two genuine bug fixes, with zero churn on
already-working commands.

The original Stages 2 (item/inventory), 3 (`give`), and 4 (nouns) are **dropped** —
those commands already handle multi-word via existing matchers (verified).

### After Stage 1 (`get`) + verifying the other candidates — 2026-07-08

More stages proved unnecessary once verified against live code:

- **`get all <container>` already works** — the `all` branch resolves the
  container by last word via `FindContainerByName`'s fuzzy match, and has no
  item-span to mis-strip (that was the non-`all` bug). Verified by test.
- **`give` two-slot already works** — `give.go` line 1 `StripPrepositions` drops
  `to`, then `splitGiveArgs` does a greedy-from-right split validating both the
  object and recipient resolve (`give dagger to smith rusk` works). Migrating it
  would be a pure refactor with regression risk and ~zero benefit — dropped.
- **`look <noun>` / single-item / actor commands already work** (Stage-0
  divergence) — dropped.

**Only the admin two-slot commands are genuinely broken:** `knowledge` /
`opinion` / `crime` / `faction` use `strings.Fields` + positional `args[0]` /
`args[1]`, so `knowledge show bank clerk <player>` reads mob="bank",
player="clerk". Confirmed by reading the code.

**Divergence in the fix approach:** the parser's adapters are **room-scoped**
(`FindByName` searches mobs present in the room), but admin resolves mobs by
**template name** globally (`knowledgeResolveMobIdent` → `AllMobTemplates`). So
admin cannot reuse the room adapters. Instead it reuses the composition
*pattern* via a new **scope-agnostic** helper `parser.SplitLeadingMatch(input,
matches func(string) bool)` — greedy longest-leading-span with a caller-injected
validator. This keeps the seam useful for global-scoped commands without forcing
a room `Scope` on them. Value is admin/dev ergonomics (B) — the numeric-mobId
workaround exists today, so this is a low-priority polish, done because the gap
is real and the fix is small.

## Design

### New package `internal/parser`

A small package exposing one primitive plus a couple of composed helpers, all
pure and unit-testable against a `Scope` (the search context).

```go
type Kind int
const (
    KindMob Kind = iota
    KindPlayer
    KindPet
    KindFloorItem
    KindBackpackItem
    KindEquippedItem
    KindComponentItem   // component bag
    KindPotionItem      // bandolier
    KindRoomContainer
    KindCorpse
    KindNoun
    KindExit
)

// Scope is what to search. Room is required; User is required for
// inventory/equipped/component/potion kinds and ownership-sensitive lookups.
type Scope struct {
    User *users.UserRecord
    Room *rooms.Room
}

// Match is the typed result. Only the fields relevant to Kind are populated.
type Match struct {
    Kind          Kind
    Name          string      // canonical resolved name (for messaging)
    Item          items.Item  // item-ish kinds
    MobInstanceId int         // KindMob
    UserId        int         // KindPlayer / KindPet
    ContainerName string      // KindRoomContainer
    CorpseIdx     int         // KindCorpse
    Leftover      string      // unconsumed tokens (e.g. an item-target after a recipe name)
    Ambiguous     []Match     // >1 candidate at the winning span (for future disambiguation UX)
}

// Resolve is the primitive: greedy longest-span multi-word match across the
// requested kinds, in the caller's priority order. Returns the best Match and
// whether anything resolved.
func Resolve(s Scope, input string, kinds ...Kind) (Match, bool)
```

### Kind adapters

Each `Kind` maps to a thin adapter that wraps the *existing* resolver — we are
not rewriting matching logic, only giving it one front door:

| Kind | Adapter wraps |
|------|---------------|
| KindMob / KindPlayer | `room.FindByName` (+ users lookup) |
| KindPet | `room.FindByPetName` |
| KindFloorItem | `room.FindOnFloor` |
| KindBackpackItem / KindEquippedItem | `user.Character.FindItem` / equipment scan |
| KindComponentItem / KindPotionItem | `items.FindMatchIn` over `ComponentItems` / `PotionItems` |
| KindRoomContainer | `room.FindContainerByName` (+ hidden-discovery gate) |
| KindCorpse | `room.FindCorpseIndex` |
| KindNoun | `room.FindNoun` |
| KindExit | `room.Exits` lookup |

### Resolution algorithm (`Resolve`)

1. Tokenize `input` with `SplitButRespectQuotes` (so `get "lake iron nodule"`
   and `get lake iron nodule` both work).
2. **Greedy longest span:** for span length L from `len(tokens)` down to 1, take
   the first L tokens as the candidate string; for each requested kind in the
   caller's order, call its adapter; the first hit at the longest span wins. This
   makes `bank clerk` (2-word mob) beat `bank` (1-word noun), and within a span
   length the caller's kind order breaks ties deterministically.
3. Disambiguation prefixes (`2.dagger`, `all.item`, `dagger#2`) are handled
   inside the adapters (they call `FindMatchIn`, which calls `GetMatchNumber`),
   so that behavior is inherited unchanged.
4. Populate `Match.Ambiguous` with any co-matches at the winning span; default
   behavior returns the highest-priority single match.

### Composed helpers (keep commands thin)

Built on the primitive so commands don't re-derive the common shapes:

- **`ResolveItem(scope, input) (Match, bool)`** — the full get/drop/look-item
  ladder, lifted *once*: detect and strip a `from` keyword; resolve an optional
  trailing **container / corpse / pet** span (greedy); if found, resolve the item
  from inside it; otherwise resolve the item from floor / backpack / equipped. The
  returned `Match` carries the item plus its source (`ContainerName` / `CorpseIdx`
  / floor / backpack) so the command can still run its own gates. This is
  precisely the `get.go` ladder, now shared and tested.
- **`ResolveActor(scope, input, kinds…) (Match, bool)`** — mob / player / pet for
  `attack`, `consider`, `target`, and give-recipient resolution.

**Two-slot commands** (`give <item> to <mob>`, admin `<mob> <player>`) split on
the connective preposition first, then call `ResolveItem`/`ResolveActor` per
side. No grammar engine; the split is a one-line helper
(`splitOnConnective(input, "to")`), replacing `give`'s bespoke `splitGiveArgs`.

### Migration contract (the safety rail)

Every command migration is **refactor-only**: identical messages, identical
resolution results, identical disambiguation for all existing inputs. The one
intended *additive* change is that multi-word input now resolves where it
previously failed. Enforced per command by:

1. Characterize current behavior with tests (the golden behavior).
2. Swap the bespoke ladder for a `Resolve`/`ResolveItem`/`ResolveActor` call.
3. Old tests stay green; new multi-word tests pass.

## Staged Decomposition

Each stage is independently shippable, leaves the game fully working (additive +
behavior-preserving), and gets **its own implementation plan**. Stage 0 is a
prerequisite for all others; Stage 1 precedes 2–5; Stage 6 is last.

| Stage | Scope | Rationale |
|-------|-------|-----------|
| **0 — Foundation** | `internal/parser`: `Kind`/`Scope`/`Match`/`Resolve` + kind adapters + `ResolveItem`/`ResolveActor` + `splitOnConnective`. Full unit tests. No command migrated. | Pure, fully tested seam shipped before any behavior change. |
| **1 — Reference: `get`** | Migrate `get.go`'s ladder onto `ResolveItem`. Characterization tests first. | Messiest ladder and where the corpse-loot bug lived; proves the pattern end-to-end. |
| **2 — Item/inventory commands** | `drop`, `look` (item path), `equip`, `remove`, `use`, `eat`, `drink`. | All share `ResolveItem` / inventory resolution; mechanical once Stage 1 sets the shape. May split if large. |
| **3 — Actor commands** | `attack`, `consider`, `target`, `give` (two-slot). Retire `give`'s `splitGiveArgs`. | Actor resolution + connective split, distinct from the item path. |
| **4 — Nouns / `look` anything** | Route `look <noun>` and multi-word prose through the resolver; retire `FindNoun`'s per-call alias map in favor of the shared greedy matcher. | The headline A win (`look hare paths`), self-contained. |
| **5 — Admin (B fallout)** | `knowledge` / `opinion` / `crime` / faction multi-word mob+player lookup via two-slot split + `ResolveActor`. | No player-facing risk; delivers B. |
| **6 — Convergence** | Optionally route `cast` / `craft` through the primitive, retire the now-dead bespoke matchers, document the un-hyphenated authoring convention (scope-B habit). | Cleanup only after everything's proven. |

## Testing Strategy

- **Stage 0:** package unit tests for the primitive (greedy span, kind priority,
  quote handling, `N.item` inheritance) and each kind adapter against a seeded
  `Scope`.
- **Each migration stage:** per-command characterization/regression tests written
  *before* the swap (behavior-preservation contract), plus new multi-word
  resolution tests (the A win).
- **Regression backbone:** the existing `TestGet_CorpseLoot`,
  `TestGive_MultiWordRecipientName`, and the bandolier / container-word subtests
  in `usercommands_test.go` must stay green through every stage.

## Risks & Mitigations

- **Behavior drift during migration** → the migration contract + characterization
  tests before each swap; stages are small and independently reviewed.
- **Kind-priority differences between commands** → priority is caller-supplied
  per call, not global, so each command keeps its own precedence (e.g. `get`
  prefers containers, `look` prefers actors).
- **Ownership/gate logic living in commands** (corpse loot-mode gates, hidden-
  container discovery, exploding-item guards) → these stay in the command layer;
  the resolver only *finds* the target, it does not apply command-specific gates.
  `ResolveItem` returns the container/corpse handle; the command still runs its
  ownership/mode checks. This keeps the seam free of command policy.
- **Scope creep into a grammar engine** → explicitly a non-goal; two-slot split
  is the only structural concession.

## Out-of-Scope Follow-ups (noted, not built here)

- Active de-hyphenation of existing content (scope C from the brainstorm).
- Interactive "did you mean…?" disambiguation UX (the `Match.Ambiguous` hook is
  left in place for it).
