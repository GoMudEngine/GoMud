# Unified Parser Seam — Stage 2 (Admin two-slot) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix multi-word mob lookup in the admin commands (`knowledge`,
`opinion`, `crime`, `faction`), which today mis-parse `knowledge show bank clerk
<player>` as mob="bank" / player="clerk" (positional `strings.Fields`). Add one
scope-agnostic `parser.SplitLeadingMatch` helper and route each command's mob
resolution through it.

**Architecture:** The parser's existing adapters are room-scoped, but admin
resolves mobs by **template name** globally. So this stage adds a scope-agnostic
building block — `SplitLeadingMatch(input, matches)` = the longest leading token
span for which the caller's validator returns true — and each admin command
passes its own `…ResolveMobIdent` as the validator. No room `Scope` involved.

**Tech Stack:** Go, `testify`, `internal/parser`, the admin command files in
`internal/usercommands/`.

**Spec:** `docs/superpowers/specs/2026-07-08-unified-parser-seam-design.md`
(see "After Stage 1" divergence — this is the ONLY genuinely-broken remaining
command family; value is admin/dev ergonomics, B-tier).

---

## Task 1: Add `parser.SplitLeadingMatch`

**Files:**
- Modify: `internal/parser/helpers.go`
- Test: `internal/parser/helpers_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/parser/helpers_test.go`:

```go
func TestSplitLeadingMatch(t *testing.T) {
	// Validator: "bank clerk" and "bank" are valid; nothing longer is.
	matches := func(c string) bool { return c == "bank clerk" || c == "bank" }

	// Longest valid leading span wins.
	head, tail, ok := SplitLeadingMatch("bank clerk smoketester", matches)
	require.True(t, ok)
	assert.Equal(t, "bank clerk", head)
	assert.Equal(t, "smoketester", tail)

	// Single-token head + trailing value.
	head, tail, ok = SplitLeadingMatch("bank smoketester extra", matches)
	require.True(t, ok)
	assert.Equal(t, "bank", head)
	assert.Equal(t, "smoketester extra", tail)

	// No leading span matches.
	_, _, ok = SplitLeadingMatch("nobody here", matches)
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/parser/ -run TestSplitLeadingMatch -v`
Expected: FAIL — `undefined: SplitLeadingMatch`.

- [ ] **Step 3: Implement**

Append to `internal/parser/helpers.go`:

```go
// SplitLeadingMatch finds the longest leading token span of input for which
// matches() returns true, and returns that span (head) plus the remaining tail.
// It is scope-agnostic — the caller injects the validator — so it serves
// global-scoped commands (e.g. admin "<mob-template-name> <player> [value]")
// that the room-scoped adapters don't fit. ok=false when no leading span
// matches.
func SplitLeadingMatch(input string, matches func(candidate string) bool) (head, tail string, ok bool) {
	tokens := strings.Fields(input)
	for headLen := len(tokens); headLen >= 1; headLen-- {
		candidate := strings.Join(tokens[:headLen], " ")
		if matches(candidate) {
			return candidate, strings.Join(tokens[headLen:], " "), true
		}
	}
	return "", "", false
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/parser/ -run TestSplitLeadingMatch -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/parser/helpers.go internal/parser/helpers_test.go
git commit -m "feat(parser): scope-agnostic SplitLeadingMatch for global two-slot commands"
```

---

## Task 2: Fix `knowledge`'s mob resolution (reference migration)

Route the three `knowledge` sub-functions (`knowledgeShow`, `knowledgeFrequented`,
`knowledgeForget`) through `SplitLeadingMatch`, so the mob name can be multi-word
and the remaining tokens become the player + optional value.

**Files:**
- Modify: `internal/usercommands/admin.knowledge.go`
- Test: `internal/usercommands/admin.knowledge_test.go` (create)

- [ ] **Step 1: Write the failing test**

`internal/usercommands/admin.knowledge_test.go`:

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit-level: the greedy mob-prefix split that the knowledge command relies on
// must consume a multi-word mob name and leave the player as the tail.
func TestKnowledge_MultiWordMobSplit(t *testing.T) {
	// Simulate the resolver: "bank clerk" is a real mob template; "bank" is not.
	resolves := func(c string) bool { return c == "bank clerk" }

	head, tail, ok := parser.SplitLeadingMatch("bank clerk smoketester", resolves)
	require.True(t, ok)
	assert.Equal(t, "bank clerk", head, "the full multi-word mob name must be the head")
	assert.Equal(t, "smoketester", tail, "the player name must be the tail")
}
```

(Note: a full end-to-end `Knowledge(...)` test would require seeding a mob
template named "Bank Clerk" via `mobs.SeedMobsForTest` with matching
`Character.Name`; the unit test above locks the split contract the command uses.
If you seed a two-word-named mob, add an end-to-end assertion that
`Knowledge("show bank clerk", user, room, 0)` reports the mob, not "Unknown mob:
bank".)

- [ ] **Step 2: Run to verify it passes at the helper level, fails end-to-end today**

Run: `go test ./internal/usercommands/ -run TestKnowledge_MultiWordMobSplit -v`
Expected: PASS (this validates the helper contract). The BUG being fixed is that
`admin.knowledge.go` doesn't yet USE this split — Step 3 wires it in.

- [ ] **Step 3: Wire `SplitLeadingMatch` into the three sub-functions**

In `internal/usercommands/admin.knowledge.go`, add the import
`"github.com/GoMudEngine/GoMud/internal/parser"`, then replace the
`mobId, ok := knowledgeResolveMobIdent(args[0])` opener in each of
`knowledgeShow`, `knowledgeFrequented`, and `knowledgeForget` with a greedy split
over the *joined* args:

```go
	// Greedy multi-word mob name: consume the longest leading span that resolves
	// to a mob template; the remaining tokens are the player + optional value.
	head, tail, ok := parser.SplitLeadingMatch(strings.Join(args, " "), func(c string) bool {
		_, found := knowledgeResolveMobIdent(c)
		return found
	})
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", strings.Join(args, " ")))
		return true, nil
	}
	mobId, _ := knowledgeResolveMobIdent(head)
	rest := strings.Fields(tail) // [player, value...]
```

Then, in each function, replace the subsequent positional references:
- `args[1]` (player name) → `rest[0]`
- `args[2]` (value/fact/topK) → `rest[1]`
- the `len(args) == 1` "list all for this mob" check → `len(rest) == 0`
- the `len(args) < 2` / `len(args) < 3` guards → the equivalent on `len(rest)`

Preserve every existing message and code path; only the source of `mobId` and the
player/value indices change. `knowledgeResolveSubject` (the `mob <id>` subject
shorthand) still receives `rest` in place of the old `args[1:]`.

- [ ] **Step 4: Run the knowledge + parser suites**

Run: `go test ./internal/usercommands/ -run 'TestKnowledge' -v && go test ./internal/parser/`
Expected: PASS. If you added the end-to-end mob-seeded assertion, it now reports
the multi-word mob instead of "Unknown mob: bank".

- [ ] **Step 5: Build + commit**

```bash
go build ./...
git add internal/usercommands/admin.knowledge.go internal/usercommands/admin.knowledge_test.go
git commit -m "fix(admin): knowledge resolves multi-word mob names via parser.SplitLeadingMatch"
```

---

## Task 3: Sweep `opinion`, `crime`, `faction` with the same pattern

Each of these has its own `…ResolveMobIdent` (e.g. `opinionResolveMobIdent`) and
the same `strings.Fields` + positional-`args` bug. Apply the identical
`SplitLeadingMatch` transformation.

**Files:**
- Modify: `internal/usercommands/admin.opinion.go`,
  `internal/usercommands/admin.crime.go`, `internal/usercommands/admin.faction.go`

- [ ] **Step 1: Confirm each command's mob-ident resolver name**

Run: `grep -nE "ResolveMobIdent|strings.Fields|args\[0\]|args\[1\]" internal/usercommands/admin.opinion.go internal/usercommands/admin.crime.go internal/usercommands/admin.faction.go`
Note the resolver function name per file (e.g. `opinionResolveMobIdent`). Some
sub-commands may take only a mob (no player) — those still benefit from
multi-word mob resolution but have empty `tail`.

- [ ] **Step 2: Apply the transformation per command**

For each sub-function that currently does `mobId, ok := xResolveMobIdent(args[0])`,
apply the same block as Task 2 Step 3, substituting the file's own resolver:

```go
	head, tail, ok := parser.SplitLeadingMatch(strings.Join(args, " "), func(c string) bool {
		_, found := opinionResolveMobIdent(c) // <- per-file resolver
		return found
	})
	if !ok { /* existing "unknown mob" message */ }
	mobId, _ := opinionResolveMobIdent(head)
	rest := strings.Fields(tail)
```

Re-index the downstream `args[1]`/`args[2]` to `rest[0]`/`rest[1]` and the
length guards accordingly, preserving all messages. Add the `parser` import to
each file. (Note: `opinionResolveMobIdent` returns `(id, name, ok)` — adapt the
validator to `func(c string) bool { _, _, found := opinionResolveMobIdent(c); return found }`.)

- [ ] **Step 3: Run the admin suites + build**

Run: `go test ./internal/usercommands/ -run 'TestOpinion|TestCrime|TestFaction' && go build ./...`
Expected: PASS, build OK. (`admin.opinion_test.go` and `admin.crime_test.go`
already exist — they must stay green.)

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/admin.opinion.go internal/usercommands/admin.crime.go internal/usercommands/admin.faction.go
git commit -m "fix(admin): opinion/crime/faction resolve multi-word mob names via SplitLeadingMatch"
```

---

## Definition of Done (Stage 2)

- `parser.SplitLeadingMatch` exists and is unit-tested.
- `knowledge`/`opinion`/`crime`/`faction` resolve multi-word mob template names;
  `knowledge show bank clerk <player>` reports the mob instead of "Unknown mob:
  bank".
- The existing admin tests (`admin.opinion_test.go`, `admin.crime_test.go`) and
  the full `internal/usercommands` + `internal/parser` suites are green.
- `go build ./...` clean.

## Divergences From Spec (this stage)

- **Scope-agnostic helper, not a room adapter.** Admin resolves mobs by template
  name globally, so it can't use the room-scoped `Resolve`/adapters. It uses the
  injected-validator `SplitLeadingMatch` instead — the composition *pattern*
  reused without the room `Scope`.
- **`give` / `get all` NOT migrated** — verified already working; migrating them
  would be pure refactors with regression risk (see spec divergence notes).
- After this stage the only remaining item is optional **Stage 3 convergence**
  (retire dead bespoke matchers, doc the authoring convention) — low priority.
