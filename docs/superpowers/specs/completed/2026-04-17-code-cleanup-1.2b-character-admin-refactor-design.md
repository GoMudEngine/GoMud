# Code Cleanup 1.2b: Character + Admin God-Function Refactor — Design Spec

## Goal

Decompose 5 oversized functions in `internal/characters/character.go`
and `internal/usercommands/admin.room.go` into focused helpers, backed
by characterization tests for the three character-package functions so
the refactor is provably semantics-preserving.

Paired substage 1.2a (combat + spell god-functions) is deferred until
after 1.2c ships, giving combat a week to settle before another round
of refactoring.

## Scope

**In scope:**

1. **Character batch** — write characterization tests, then refactor.
   Extracted helpers stay inline in `character.go` (1.2c will split
   `character.go` into themed files as a follow-up).
   - `Character.RecalculateStats()` (239 lines) — collapse 6 repeated
     per-stat blocks into loops.
   - `Character.Wear()` (~235 lines) — extract slot-selection
     helpers.
   - `Character.Validate()` (433 lines) — extract 4 subsystem
     validators (skills, spells, equipment, stats).
2. **Admin batch** — manual smoke test + build/vet.  Extracted
   helpers move to **new files** adjacent to `admin.room.go`.
   - `Room()` (387 lines) — subcommand dispatcher → new
     `admin.room.dispatcher.go`.
   - `room_Edit_Exits()` (257 lines) — prompt state machine → new
     `admin.room.exits.go`.

**Out of scope:**

- `handlePlayerVsMob`, `handleMobVsPlayer`, `applyMobEffect` — these
  are Stage **1.2a**, scheduled after 1.2c.
- `character.go` file split into themed files — Stage **1.2c**,
  runs immediately after 1.2b so refactored character functions can
  be moved as cohesive units with their new helpers.
- Any function under 200 lines.
- Any behavior change, except clear-bug fixes during refactoring
  (see scope-creep policy below).

## Decisions Locked During Brainstorming

- **Domain split over one mega-substage.** Stage 1.2 as originally
  scoped was ~10 hours across 8 functions in 4 subsystems. Splitting
  into 1.2a/b/c gives each substage a focused review surface and
  lets combat code rest between 1.7 and 1.2a.
- **Characterization tests before refactoring the 3 character
  functions.** These functions have zero existing test coverage
  today. Writing tests first captures current behavior as the
  invariant, making "zero behavior change" a claim we can actually
  verify instead of hope for.
- **Helpers inline for character batch, new file per admin
  function.** Character helpers are decomposition of a single
  parent's logic — they don't form their own theme. Admin helpers
  (one-per-subcommand, prompt steps) cluster cleanly into themed
  files and shrink `admin.room.go`'s 1429 lines.
- **Character batch runs before admin batch.** Clean seam between
  test-backed work and smoke-tested work. Admin refactor can't
  regress character tests because it doesn't touch that package.
- **Scope-creep policy: fix clear bugs in a separate prior commit;
  pause on ambiguous cases.** Clear bugs (nil deref, dropped error,
  obvious typo) get fixed in a preceding `fix:` commit so the
  refactor diff remains semantics-preserving. Ambiguous cases stop
  and ask — intentional → add a code comment explaining why, not a
  bug → case-by-case fix-or-TODO decision.

## Architecture

**Execution order — 7 commits on `feature/stage-1.2b-character-admin-refactor`:**

| # | Commit | Description |
|---|--------|-------------|
| 1 | `test(characters): characterization tests for Validate/Wear/RecalculateStats` | Capture current behavior as tests. Must pass on unrefactored code before moving on. |
| 2 | `refactor(characters): extract RecalculateStats per-stat loops` | Collapse the six repeated per-stat blocks into loops. Helpers inline in `character.go`. |
| 3 | `refactor(characters): extract Wear slot-selection helpers` | Pull slot-matching, multi-arm routing, and swap-out logic into private helpers. Inline in `character.go`. |
| 4 | `refactor(characters): extract Validate subsystem validators` | Four subsystem validators each <80 lines. `Validate()` becomes a short dispatcher. Inline helpers. |
| 5 | `refactor(usercommands): split admin room subcommand dispatcher` | Move `Room()` dispatcher into new `admin.room.dispatcher.go`. One helper per subcommand. |
| 6 | `refactor(usercommands): split admin room exits prompt machine` | Move `room_Edit_Exits()` + its prompt helpers to new `admin.room.exits.go`. |
| 7 | `docs: mark code cleanup 1.2b complete` | Flip status in `code_cleanup_stage_1_overview.md`. |

**Why this order:** Tests first so every character refactor has a
safety net. `RecalculateStats` before the harder `Validate` so the
mechanical loop-collapse is a warm-up. Character batch finishes
before admin batch — clean package seam. Admin commits are
independent of each other; internal order is not load-bearing.

**Commit cadence rule:** after each commit,
`go build && go vet ./... && go test ./...` must be clean. No
`--no-verify`. If a hook fails, fix and create a new commit (per
repo convention).

## Characterization Test Plan

New test file: `internal/characters/godfunc_refactor_test.go`.
The name makes it clear these tests exist for the refactor; they
can be absorbed into sibling `*_test.go` files post-1.2c.

### `RecalculateStats` — 6 tests

1. **Base-stat hydration from species** — fresh `Character` with
   zero base stats, species set; verify each stat's `Base` is
   populated from species data; verify rolled stats (non-zero base)
   are NOT overwritten.
2. **Equipment stat mods** — equip items with stat_flat mods, call
   `RecalculateStats`, verify `.Mods` fields pick up the right
   sums.
3. **Mutation flat + multiplier** — apply stat_flat and
   stat_multiplier mutations, verify both apply (flat before
   `Recalculate`, multiplier after).
4. **Pool max derivation** — stats set deterministically, verify
   `HealthMax`, `StaminaMax`, `ConvictionMax`, `ActionPointsMax`
   match the config-driven formulas. Lock in floors: `HealthMax≥1`,
   `StaminaMax≥0`, `ConvictionMax≥0`, `ActionPointsMax≥50`.
5. **Pool reservation clamping** — equip a chrysalis-enchanted
   item; verify `Health/Stamina/Conviction` are clamped to
   `max - reservation` (with floors).
6. **CharacterStatsChanged event emission** — `userId != 0`,
   change a stat between two calls, verify event queued; no change
   → no event.

### `Wear` — 5 tests

1. **Happy path, empty slot** — wear body armor into empty Body
   slot; verify equipped, no displaced items, `newItemWorn=true`,
   empty `failureReason`.
2. **Slot occupied, swap** — Body slot has an item; wear another;
   verify old item returned in `returnItems`.
3. **Two-handed weapon displaces offhand** — equip shield in
   offhand, wear a 2H weapon; verify both the existing weapon (if
   any) AND the offhand are returned.
4. **Multi-arm routing** — with Extra Arms mutation level 2, wear
   a fourth weapon; verify it routes to arm 3/4 slot, not the
   occupied main hand.
5. **Wrong item type** — try to wear a potion; verify
   `newItemWorn=false`, failureReason populated, no state change.

### `Validate` — 7 tests

1. **Empty/zero character is corrected** — fresh `Character{}`;
   verify mandatory fields set to defaults without error.
2. **Skill map contains all expected skills** — partial skill map
   is filled in; unknown/obsolete skill keys stripped.
3. **Equipped item that no longer exists** — equipment slot points
   to item with no ItemSpec; verify it's cleared / returned to
   backpack (whichever current behavior is). **May surface latent
   bug** — pause per scope-creep policy if current behavior is
   unclear.
4. **Spell map consistency** — character knows a spellId that
   no longer exists in spell definitions; verify it's pruned.
5. **Buff expiry cleanup** — character has expired buffs; verify
   they're removed. **May surface latent bug** — same pause policy.
6. **PermaBuff recalc** — `Validate(true)`; verify perma buffs are
   reapplied to stats.
7. **Returns nil when character is valid** — no-op case:
   already-valid character → no error, no state change, no events
   queued.

Each test uses a real `Character` (no mocking), deterministic
setup, exact equality checks. Slow-ish for unit tests but fast
enough — not a runtime hot path.

## Refactor Patterns per Function

### `RecalculateStats` (239 → ~80 lines)

No extracted helper functions — just collapse repetition. Build
a slice of per-stat entries:

```go
type statEntry struct {
    ptr     *stats.Stat
    modName string  // statmods.Strength, etc.
    mutKey  string  // "strength", "dexterity", etc.
    before  int     // captured for change detection
}
```

Three linear passes:
1. Species-base hydration (conditional on `Base == 0`).
2. Mods + mutation-flat assignment, then `Recalculate()`.
3. Mutation-multiplier application to `ValueAdj`.

Post-stat block (pools, floors, enchant withdrawal, event emit)
stays as-is.

### `Wear` (~235 → <80 lines parent)

Private helpers (inline in `character.go`):
- `selectWearSlots(item) ([]Slot, error)` — determines target
  slots, handles 2H weapons, multi-arm mutation, shields-in-arms.
- `swapOutOccupiedSlots(slots) []items.Item` — collects items
  currently in target slots into a return list.
- `applyItemToSlots(item, slots)` — writes item into slot(s),
  updates paired-slot metadata.

Parent flow: validate → select → swap out → apply → return.

### `Validate` (433 → <60 lines parent)

Four private helpers, each <80 lines:
- `validateSkillMap() bool`
- `validateSpellMap() bool`
- `validateEquipment() bool`
- `validateStatsAndPools() bool`

Each returns `changed`. Parent calls each, ORs results,
conditionally runs `RecalculateStats` + perma-buff recalc,
returns error.

### `Room()` (387 lines → `admin.room.dispatcher.go`)

Current shape: large `switch` on subcommand keyword. Pattern:
one unexported helper per subcommand, named `adminRoom_<Sub>()`:

- `adminRoom_Info`, `adminRoom_Set`, `adminRoom_Spawn`,
  `adminRoom_Despawn`, `adminRoom_AddDetail`,
  `adminRoom_RemoveDetail`, `adminRoom_EditContainers`,
  `adminRoom_EditExits`, `adminRoom_EditMutators`, etc.

Exact list derived from reading the current switch. All helpers
move to new `admin.room.dispatcher.go`. `Room()` stays in
`admin.room.go` as a thin dispatcher (~40 lines).

### `room_Edit_Exits()` (257 lines → `admin.room.exits.go`)

Multi-step prompt state machine. Extract each prompt step +
its handler:
- `promptExitDirection` / `handleExitDirectionInput`
- `promptExitDestination` / `handleExitDestinationInput`
- `promptExitLock` / `handleExitLockInput`
- Any additional steps discovered while reading the function.

Move `room_Edit_Exits()` + helpers into
`admin.room.exits.go`. Check whether `editLockAndTrap` is called
only from here — if shared with other functions, leave it in
`admin.room.go`.

## Verification & Rollout

**Per-commit verification:**
```bash
go build ./...
go vet ./...
go test ./internal/characters/...     # after commits 1–4
go test ./...                          # after commits 5–6
```

Any failure = fix before committing.

**After each character-function refactor (commits 2, 3, 4):**
- Characterization tests from commit 1 must still pass unchanged.
- Diff review: confirm the change looks purely structural.

**After the admin batch (commits 5, 6):**
- Boot a local server.
- Smoke test: `room info`, `room set name`, `room spawn mob`,
  `room edit exits` (walk through at least one exit creation/edit),
  `room edit containers`, `room edit mutators`.

**Pre-push:**
- All commits pass per-commit verification.
- Manual smoke of admin commands complete.
- `PATCH_NOTES.md` — no entry (internal-only, zero player impact).
- `code_cleanup_stage_1_overview.md` — 1.2b flipped to Complete.

**Branch strategy:**
- Branch: `feature/stage-1.2b-character-admin-refactor` off
  `development`.
- Merge `--no-ff` into `development`.
- No urgent prod push — can wait until 1.2c also ships.

**Rollback:**
- Each commit independently revertable. Breakage in commit 4 → revert
  just that commit without losing 2/3.
- Commit 1 (tests) is pure value regardless — keep even if the rest
  of the branch is abandoned.

## Risk Register

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Characterization tests capture a latent bug and lock us to shipping it | Medium | Scope-creep policy — clear bugs get a prior `fix:` commit; ambiguous cases pause |
| `Validate` refactor changes validation order subtly | Medium | Tests 1, 3, 5 exercise multi-subsystem interactions explicitly |
| Admin `Room` dispatcher refactor misses a subcommand branch | Low | Manual smoke walks each branch |
| `editLockAndTrap` turns out to be called from multiple files | Low | Grep before moving; leave in `admin.room.go` if shared |
| Test setup requires a live `Equipment` / `ItemSpec` fixture more complex than expected | Low-Med | Build minimal real fixtures; defer truly hard cases to post-1.2b integration tests |

## Success Criteria

- 5 functions reduced from 239–433 lines each to <80 lines
  (parent) with helpers <80 lines each.
- 18 new characterization tests pass on current code AND on
  refactored code.
- `go build`, `go vet`, `go test ./...` clean after every commit.
- Manual admin smoke of all `room` subcommands + full exit
  edit flow succeeds.
- No player-visible behavior change.
