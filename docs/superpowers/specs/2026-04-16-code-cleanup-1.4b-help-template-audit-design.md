# Code Cleanup 1.4b: Help Template Audit — Design Spec

## Goal

Close the gap in help discoverability by (1) adding a
`TestHelpFileCompleteness_Commands` test that catches missing help
templates for registered commands, and (2) indexing all 32 commands
that are registered but missing from `keywords.yaml`'s help menu.

Additive improvement — no deletions, no behavior change to existing
gameplay.

## Scope

**In scope:**
- Add test `TestHelpFileCompleteness_Commands` in
  `internal/devtools/helpfile_completeness_test.go`
- Extend the test to verify admin commands too
- Add missing command entries to `_datafiles/world/dogmud/keywords.yaml`
  under appropriate categories
- Create stub help templates for any command that has no template at
  all (only if the command's purpose is clear enough to write a
  one-sentence description — otherwise flag as TODO)

**Out of scope:**
- Deleting any help templates
- Rewriting or updating stale template content
- Creating deep help content for complex commands (stub is fine)
- Auditing spell/recipe/mutation/skill completeness (existing tests
  cover those)

## Current state

The existing `internal/devtools/helpfile_completeness_test.go` has
tests for:
- Spells — every `spells/*.yaml` has `help/<name>.template`
- Recipes — every `recipes/*.yaml` has `help/<name>.template`
- Mutations — every `mutations/*.yaml` has `help/<name>.template`
- Skills — a hardcoded list of required skills has help files

**Gap:** No test covers commands registered in
`internal/usercommands/usercommands.go` or
`internal/mobcommands/mobcommands.go`.

The audit surfaced 32 commands registered in `usercommands.go` that
are NOT listed in `keywords.yaml`. Many have help templates but are
not discoverable via the categorized `help` menu. Players calling
`help <command>` directly works; browsing `help` to find them doesn't.

## Architecture

### Test design

Add a new test function in `helpfile_completeness_test.go`:

```go
// TestHelpFileCompleteness_Commands ensures every registered user
// command has a matching help template (either help/<cmd>.template
// for regular commands or admincommands/help/command.<cmd>.template
// for admin-only commands).
func TestHelpFileCompleteness_Commands(t *testing.T) {
    root := dataRoot(t)
    // ... check every command from usercommands.userCommands map ...
}
```

### How the test finds commands

The command registry is `userCommands map[string]CommandAccess` in
`internal/usercommands/usercommands.go`. The `CommandAccess` struct
has an `AdminOnly` bool field (check exact field name during
implementation).

The test imports `usercommands` and reads the registry directly. For
each `(name, info)` pair:
- If `info.AdminOnly` → look for help at
  `admincommands/help/command.<name>.template`
- Otherwise → look for help at `help/<name>.template`

Commands may share a help file with their alias (e.g., `companions`
is an alias for `companion`). The test will check a small allowlist
of aliases that share a help template:

```go
var commandAliases = map[string]string{
    "companions": "companion",
    // ... any others found during implementation ...
}
```

If a command name has an alias entry, the test checks the alias's
help file instead. If no alias, it checks the command's own name.

### What "missing" means

A command is missing help if:
1. No file at `help/<name>.template` (for user commands) or
2. No file at `admincommands/help/command.<name>.template` (for admin
   commands), and
3. No alias mapping exists

The test reports ALL missing commands in a single failure message so
they can be fixed in one pass. Example error output:

```
commands missing help files (15):
  attack (user) — expected help/attack.template
  go (user) — expected help/go.template
  dismiss (user) — expected help/dismiss.template
  ...
```

### Circular import consideration

`internal/usercommands` may already import `internal/devtools` for
unrelated reasons (or vice versa). If importing `usercommands` from
`devtools` creates a cycle, the test could instead read command
names from a manually maintained list in the test file, or use
reflection over the package. Prefer direct import if possible; fall
back to a maintained list as a last resort.

**Verification during implementation:** If the naive
`import "github.com/.../internal/usercommands"` works, use it.
Otherwise, document the cycle and use an alternative approach.

### Keywords.yaml update

For each command that has a help template but isn't in keywords.yaml,
add it to an appropriate category. Categories in keywords.yaml
include: configuration, character, communication, shops, quests,
combat, movement, magic, crafting, etc.

**Assignment rules:**
- Commands that modify character state (attack, flee, cast) → combat
- Commands that send messages → communication
- Movement commands (go, recall, flee) → movement or combat
- Admin commands → NOT in keywords.yaml (they have their own help
  system via `admincommands/help/`)
- Non-obvious commands → flag and ask

**Preserve alphabetical order** within each category if the existing
file uses that convention.

### Stub templates

For commands with zero help content, create a minimal stub:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">{command}</ansi>

{One-sentence description of what the command does.}

<ansi fg="yellow">Usage:</ansi>

  <ansi fg="command">{command}</ansi>              {Usage example}

{If there's an obvious "See also" link, add it.}
<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help related</ansi>
```

Only create stubs for commands where the purpose is obvious from the
command name and registration context. For ambiguous commands, don't
create a stub — instead flag them in the commit message for a
follow-up documentation task.

## Constraints

- **Zero behavior change.** Test is additive. Keyword entries don't
  change gameplay.
- **Test must pass after all fixes.** Running
  `go test ./internal/devtools/...` should report 0 failures at the
  end.
- **Don't delete any existing help files**, even ones that look
  orphaned.
- **Stub content quality.** If you can't write a one-sentence
  accurate description for a command, don't stub it. Mark as TODO in
  a committed notes file (or skip-list in the test).
- **Admin commands** use a different template path:
  `admincommands/help/command.<name>.template`, not
  `help/<name>.template`.

## Testing

After each task:
```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud
go test ./internal/devtools/... -run TestHelpFileCompleteness
```

After full completion:
```bash
go test ./...
```

Manually verify in-game (optional):
- Log in as an admin
- Type `help` — verify all categories render
- Type `help <command>` for a previously-missing command and verify
  the template loads

## Risk Assessment

**Very low.** The test is an additive diagnostic. The keyword
entries are a data-only change that affects help rendering, not
command behavior. Stub templates are text-only additions.

Risks:
- Import cycle between `usercommands` and `devtools`. Mitigation:
  fall back to maintained list in test.
- Mis-classifying a command into the wrong keyword category. Low
  impact — players can still find it via `help <cmd>` directly.

## Execution Plan

### Task 1: Add `TestHelpFileCompleteness_Commands`

- Add the test to `internal/devtools/helpfile_completeness_test.go`
- Handle both user commands and admin commands
- Handle alias allowlist
- Run test — it will fail with a list of missing entries
- Commit the test in its failing state (tests documenting
  deficiencies are fine — every team member now sees the gap)

### Task 2: Fix the failures

- For each failing command, determine: does a template exist at a
  different path (indicating alias or path mismatch)? Or is it truly
  missing?
- If template exists at a different path → add alias entry in test
  OR rename file
- If template missing and command is obvious → create stub
- If template missing and command is complex → note in a followup
  TODO, add to test's skip list with a reason
- Add each indexed command to `keywords.yaml` under appropriate
  category
- Run test — it must pass
- Commit

### Task 3: Final verification

- Test passes
- Full test suite passes
- Manual check: `help` menu in-game shows reasonable categorization
- Report total templates created, keyword entries added, and any
  remaining TODO commands

## Success criteria

- `TestHelpFileCompleteness_Commands` passes
- All 32 (or however many) unindexed commands now appear in
  `keywords.yaml` under appropriate categories
- No existing template files deleted
- No behavior change — gameplay unaffected
