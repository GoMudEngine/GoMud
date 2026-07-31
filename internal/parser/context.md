# Parser Context

## Purpose

`internal/parser` is the shared target-resolution seam for **composition-heavy**
commands — the ones that must split one line of input into more than one role
(item vs. container, mob vs. player). It exists so those commands stop
hand-rolling `strings.Fields` ladders, which is the class of bug that hid the
2026-07-08 corpse-loot failure.

**It only finds things. It does not decide whether you may have them.**
Ownership checks, hidden-container discovery, exploding-container guards, and
every other gate stay in the command.

## When *not* to use it

Most single-slot multi-word input already resolves through the existing fuzzy
matchers (`room.FindByName`, `items.FindMatchIn`, `room.FindNoun`), which match
the whole phrase. `attack bank clerk`, `get lake iron nodule`, and
`look hare paths` work without any parser plumbing. Reach for this package only
when a command must **split** input into multiple slots.

## Files

- **parser.go** — `Kind`, `Scope`, `Match`, tokenisation, and `Resolve`.
- **adapters.go** — one adapter per `Kind`; each tries to match a candidate
  string in the given scope.
- **helpers.go** — the split helpers and the typed convenience wrappers.

## Core types

```go
type Kind int          // noun, exit, container, corpse, floor item,
                       // inventory item, component, potion, mob, player, pet
type Scope struct { /* User, Room — the search context */ }
type Match struct { /* what was found, and as which Kind */ }
```

Resolution is **greedy longest-span**: `tokenize` splits the input and
`resolveWith` tries the longest candidate phrase first, shrinking until an
adapter accepts. That is what makes multi-word names work without quoting.

## Public API

```go
func Resolve(s Scope, input string, kinds ...Kind) (Match, bool)
func ResolveItem(s Scope, input string) (Match, bool)
func ResolveActor(s Scope, input string, kinds ...Kind) (Match, bool)

func SplitTrailingContainer(s Scope, input string) (itemPart string, cm Match, ok bool)
func SplitLeadingMatch(input string, matches func(candidate string) bool) (head, tail string, ok bool)
```

### `SplitTrailingContainer`

Splits `<item> [from] <container|corpse|pet>`. Room-scoped. `get.go` is the
reference caller. The `from` connective is optional — `get sword corpse` and
`get sword from corpse` both resolve.

### `SplitLeadingMatch`

Greedy longest-**leading**-span with a caller-injected validator, and
deliberately **scope-agnostic** — it takes a `func(candidate string) bool`
rather than a `Scope`. That is what lets admin commands resolve a globally-known
name in the first slot and a player name in the second (`knowledge` and
`opinion` use it).

## Gotchas

- **The parser never enforces permission.** If a command using it stops
  checking ownership, the parser will happily resolve the target anyway.
- **Author multi-word names with spaces, not hyphens.** Because the matchers
  handle multi-word input, room nouns, item names, and `component_tag`s should
  read naturally (`lake iron nodule`). Use a hyphen only where the term
  genuinely reads hyphenated to a player.
- **Mob-ident lookups must `ConvertForFilename()` the *input* too**, so a
  space-form query matches the underscore filename form. See
  `knowledgeResolveMobIdent` / `opinionResolveMobIdent` for the pattern.
- **Test player phrasing, not implementer syntax.** A checklist that only
  covers the `from` form ships `get all corpse` broken. Every new split needs a
  test per realistic player phrasing.

## Dependencies

`items`, `rooms`, `mobs`, `users`, `characters`.

## Consumers

`internal/usercommands` — currently `get`, `knowledge`, and `opinion`. Full
design and the divergences from it:
`docs/superpowers/specs/2026-07-08-unified-parser-seam-design.md`.
