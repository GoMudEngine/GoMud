# Target Resolution Refactor — Design Spec

**Date:** 2026-04-18
**Status:** Draft / Awaiting approval
**Related memory:** `project_target_resolution_refactor.md`

## Background

Today's command code (in `internal/usercommands/` and
`internal/mobcommands/`) reimplements the same target-resolution
chain ~45 times:

```go
playerId, mobId := room.FindByName(targetName, ...flags)
if mobId > 0 {
    m := mobs.GetInstance(mobId)
    if m == nil { /* error */ }
    // ... mob-specific path ...
} else if playerId > 0 {
    p := users.GetByUserId(playerId)
    if p == nil { /* error */ }
    // ... user-specific path ...
} else {
    /* not-found error */
}
```

Variation across sites: error wording, defensive nil checks (some
present, some missing — `attack.go:27` does `m := GetInstance(mId);
m.Character.Aggro` with no guard), self-exclusion, FindFlag values,
follow-up gates (non-combatant, PvP eligibility, party membership).

The `actions.Actor` interface (`internal/actions/actor.go`) was
established in early April and battle-tested through today's
combat-quadrant unification — it's now the canonical polymorphic
attacker/defender abstraction. This refactor extends Actor's reach
from "combat-handler param" to "any user-facing target resolution"
so commands stop branching on user-vs-mob.

## Goals

1. Add `room.ResolveTargetActor(...) (actions.Actor, error)` (or
   sibling free function) that consolidates `FindByName + GetInstance/
   GetByUserId + nil-check` into one call.
2. Add exported constructors `actions.NewUserActor(*users.UserRecord)
   Actor` and `actions.NewMobActor(*mobs.Mob) Actor` so the helper
   (and any other code wanting to box an entity into an Actor) has a
   clean API.
3. Migrate all ~45 user-facing call sites in
   `internal/usercommands/` and `internal/mobcommands/` to use the
   helper. Commands stop branching on user-vs-mob at the resolution
   site; type assertions happen only at leaves where mob-only or
   user-only behavior genuinely diverges.
4. Close the latent-nil-crash class: every resolved target either
   comes back as a non-nil Actor or a typed error. No more
   `attack.go:27`-style unguarded dereferences.

## Non-Goals

- **Do not extend the Actor interface.** Leaf type assertions cover
  the small set of mob-only methods (`IsNonCombatant`, `Groups`,
  `Command`, `Character.Pet`) and user-only fields (`EventLog`,
  `PartyId`, `ScreenReader`, `HasRolePermission`). Same pattern the
  combat unification established. Widening Actor has a ripple cost
  across every implementor for marginal benefit.
- **Do not migrate downstream `mobs.GetInstance` lookups by known ID**
  (~170 sites). Those are not user-facing target resolution —
  they're "I already know the instance ID, hand me the pointer"
  lookups. They may benefit from a separate `mobs.SafeGetInstance`
  helper, but that's a different concern tracked in this spec's
  Out-of-Scope section.
- **Do not change FindByName semantics or signature.** It stays as
  `(playerId, mobInstanceId int)`. The new helper wraps it.
- **Do not change error message wording across the board.** Each
  command can override the not-found message; default is
  "You don't see them here." (the most common existing wording).

## Design

### The Helper

Free function on the `rooms` package, takes the room as receiver:

```go
// File: internal/rooms/target_resolution.go (new)

package rooms

import (
    "errors"
    "github.com/GoMudEngine/GoMud/internal/actions"
    "github.com/GoMudEngine/GoMud/internal/mobs"
    "github.com/GoMudEngine/GoMud/internal/users"
)

// ResolveTargetOptions configures target resolution behavior.
type ResolveTargetOptions struct {
    // FindFlags filters which entities are eligible (FindAll if empty).
    FindFlags []FindFlag
    // ExcludeUserId, when > 0, hides the named user from results
    // (used for self-exclusion in commands like consider).
    ExcludeUserId int
    // ExcludeMobInstanceId, when > 0, hides the named mob from results.
    ExcludeMobInstanceId int
}

// Sentinel errors for typed handling by callers.
var (
    ErrTargetNotFound       = errors.New("target not found")
    ErrTargetVanished       = errors.New("target vanished") // race: id resolved but pointer is nil
    ErrTargetSelfExcluded   = errors.New("target is self")  // matched but excluded
)

// ResolveTargetActor looks up a player or mob by name in this room
// and returns it wrapped in an actions.Actor. Returns an error if
// nothing matches, if the matched entity has vanished (stale ID),
// or if the only match was excluded.
//
// The returned Actor is concrete (*actions.UserActor or
// *actions.MobActor) — callers can type-assert when they need
// type-specific behavior (e.g., mob.IsNonCombatant(), user.EventLog).
//
// Caller pattern:
//   target, err := room.ResolveTargetActor(name)
//   if err != nil {
//       user.SendText(targetNotFoundMsg(err))  // caller controls wording
//       return true, nil
//   }
//   // ... use target uniformly ...
func (r *Room) ResolveTargetActor(name string, opts ...ResolveTargetOptions) (actions.Actor, error) {
    var o ResolveTargetOptions
    if len(opts) > 0 {
        o = opts[0]
    }

    flags := o.FindFlags
    if len(flags) == 0 {
        flags = []FindFlag{FindAll}
    }

    playerId, mobInstanceId := r.FindByName(name, flags...)

    // Apply exclusions.
    if o.ExcludeUserId > 0 && playerId == o.ExcludeUserId {
        playerId = 0
    }
    if o.ExcludeMobInstanceId > 0 && mobInstanceId == o.ExcludeMobInstanceId {
        mobInstanceId = 0
    }

    // Players take precedence over mobs in the same name (matches
    // legacy behavior — every existing call site checks playerId
    // first or mobId first based on context, but FindByName already
    // returns both so we follow the convention of "user wins").
    if playerId > 0 {
        u := users.GetByUserId(playerId)
        if u == nil {
            return nil, ErrTargetVanished
        }
        return actions.NewUserActor(u), nil
    }
    if mobInstanceId > 0 {
        m := mobs.GetInstance(mobInstanceId)
        if m == nil {
            return nil, ErrTargetVanished
        }
        return actions.NewMobActor(m), nil
    }
    return nil, ErrTargetNotFound
}
```

### Actor Constructors (new exports)

Today's code constructs actors inline as struct literals:
`&actions.UserActor{User: u, Room: r}`. Add factory functions so the
helper (and command code post-refactor) doesn't have to know the
internal field layout:

```go
// internal/actions/actor_user.go — add at top of file

// NewUserActor wraps a UserRecord in an Actor for polymorphic combat
// and target-resolution code paths. Room is resolved lazily from
// user.Character.RoomId on first GetRoom() call.
func NewUserActor(u *users.UserRecord) Actor {
    return &UserActor{User: u}
}

// internal/actions/actor_mob.go — add at top of file

// NewMobActor wraps a Mob in an Actor for polymorphic combat and
// target-resolution code paths.
func NewMobActor(m *mobs.Mob) Actor {
    return &MobActor{Mob: m}
}
```

Existing inline `&actions.UserActor{...}` / `&actions.MobActor{...}`
constructions in the codebase (we used them in
`handleCombatRound`'s call sites) get migrated to the constructors
in the same pass. Single grep confirms call site count.

### Migration Pattern Per Command

Before:
```go
playerId, mobId := room.FindByName(targetName)
if mobId > 0 {
    m := mobs.GetInstance(mobId)
    if m == nil {
        user.SendText("You don't see them here.")
        return true, nil
    }
    // mob path ...
} else if playerId > 0 {
    p := users.GetByUserId(playerId)
    // user path ...
} else {
    user.SendText("Nobody by that name here.")
    return true, nil
}
```

After:
```go
target, err := room.ResolveTargetActor(targetName)
if err != nil {
    user.SendText("You don't see them here.")
    return true, nil
}
// uniform path using target.GetCharacter(), target.SendText, etc.
// type-assert only where genuinely needed:
if !target.IsPlayer() {
    if mob := target.(*actions.MobActor).Mob; mob.IsNonCombatant() {
        user.SendText(...)
        return true, nil
    }
}
```

For commands where the user/mob paths diverge structurally beyond
trivially (e.g., `give` has different quest-engine + btree hooks for
mobs vs users; `ask` is mob-only):

- **Structurally divergent commands** keep the divergence but use
  Actor as the *parameter* to share the upfront resolution +
  validation. Internally they branch on `target.IsPlayer()` and
  pull out the concrete type via assertion.
- **Mob-only commands** (e.g., `ask`) use
  `ResolveTargetActor(name)` and immediately error on
  `target.IsPlayer()`: "You can't ask another player."
- **User-only commands** (e.g., `party invite`) symmetric: error
  if target is a mob.

## Migration Order

Per the scoping report, ~45 sites grouped:

1. **Combat targeting** (~9 sites): `bash`, `kick`, `grapple`,
   `taunt`, `trip`, `combat_shoot`. Plus `attack.go` where
   appropriate (most of attack.go's GetInstance calls are
   downstream-only). These benefit most from the latent-crash
   closure.
2. **Look-and-info** (~5 sites): `look`, `consider`, `show`,
   `skill.track`, `mobcommands/look`.
3. **Interaction** (~10 sites): `ask`, `give`, `talk`, `buy`,
   `party`, `mobcommands/give`, `mobcommands/sayto`,
   `mobcommands/show`, `mobcommands/befriend`.
4. **Admin/meta** (~10 sites): `admin.buff`, `admin.zap`,
   `admin.paz`, `admin.ai`, `admin.command`, `admin.deafen`,
   `admin.mute`, `admin.skillset`, `admin.locate`, `report`.
5. **Skills/specials** (~8 sites):
   `skill.skullduggery.steal`, `skill.skullduggery.plant`,
   `skill.skullduggery.shadow`, `target`, `cast`, plus mutation
   targeters (`mutation_blinding_flash`, etc.) where they target
   by name.
6. **Mutations/misc** (~3 sites): `mobcommands/aid`,
   `mobcommands/givequest`, `mobcommands/lookfortrouble`.

## Test Strategy

**New unit tests** in `internal/rooms/target_resolution_test.go`:

1. `TestResolveTargetActor_PlayerMatch` — name resolves to a player;
   returns `*actions.UserActor` wrapping correct user.
2. `TestResolveTargetActor_MobMatch` — name resolves to a mob;
   returns `*actions.MobActor` wrapping correct mob.
3. `TestResolveTargetActor_NotFound` — no match; returns
   `ErrTargetNotFound`.
4. `TestResolveTargetActor_StalePlayerId` — FindByName returns a
   player ID, but GetByUserId returns nil (simulate via deleting
   user mid-test); returns `ErrTargetVanished`.
5. `TestResolveTargetActor_StaleMobId` — same for mob race.
6. `TestResolveTargetActor_ExcludeSelf` — player matches but is in
   the ExcludeUserId option; returns `ErrTargetNotFound` (or
   `ErrTargetSelfExcluded` if no other matches; let the
   implementation decide which is more useful and document).
7. `TestResolveTargetActor_FindFlagFiltering` — verifies that
   passing `FindFighting` only returns combatants, etc.

**Existing tests:** `give_test.go:43` is the only test that
explicitly exercises the old `FindByName + GetInstance` chain. After
migration it should still pass (the function under test resolves
the mob the new way). Skim during execution; update if needed.

**Integration coverage:** all 45 migrated call sites are exercised
by manual smoke tests on the live server. Add a recommendation list
to the migration plan: hit one command from each category in the
smoke test.

## Risk Mitigations

| # | Risk | Mitigation |
|---|------|-----------|
| 1 | Silent nil-deref crashes (race: FindByName succeeds, GetInstance/GetByUserId returns nil) | Helper distinguishes `ErrTargetVanished` from `ErrTargetNotFound`; callers can give a more accurate message ("Your target has disappeared") if they care. Most won't. |
| 2 | Async closures store mob pointer (e.g., `ask.go` LLM callback at line 145) | Out of scope. Existing `if m != nil` guards handle this. Refactor doesn't change the closure pattern. |
| 3 | Player-vs-mob structural divergence in commands like `give` | Helper resolves uniformly; commands branch internally on `target.IsPlayer()` for divergent logic. No change to existing divergence; just shared resolution + nil safety. |
| 4 | Self-targeting bugs (commands forgetting to exclude self) | `ExcludeUserId` option is opt-in; commands that need it pass it. We do NOT auto-exclude — that would change behavior in commands that legitimately allow self-targeting (e.g., `consider self`). |
| 5 | Tests built around `room.FindByName` directly | Only `give_test.go` does this. Other tests are integration-level and route through commands; they auto-verify post-migration. |
| 6 | Performance: `ResolveTargetActor` adds an extra function call vs inline lookup | Negligible. Function-call overhead is ~ns; combat round runs at second cadence. Not measurable. |
| 7 | Mob-only / user-only methods accessed via type assertion | Pattern is established (Stage 2a/b combat work uses it). Each leaf assertion is a 2-line pattern: check `IsPlayer()` then assert. Document in commit message that this is the convention. |
| 8 | Refactor scope = many files = many opportunities for typo/regression | Migrate in commits-per-category (5-6 commits). Each commit independently testable + bisectable. Build/vet/test sweep after each commit. |

## Success Criteria

- `room.ResolveTargetActor` lives in `internal/rooms/target_resolution.go` with 7+ unit tests.
- `actions.NewUserActor` / `actions.NewMobActor` constructors exported and used by both the helper and pre-existing call sites (combat unification's inline literals migrated).
- All ~45 user-facing target-resolution call sites in
  `internal/usercommands/` and `internal/mobcommands/` use the helper.
- Inline `&actions.UserActor{...}` / `&actions.MobActor{...}` struct
  literals removed from non-package code (only the constructors and
  internal package code construct them directly).
- `attack.go:27` and any sibling unguarded `mobs.GetInstance(mId).<field>`
  patterns are gone — every dereference is preceded by either the
  helper's nil-safety or an explicit guard.
- `go test ./...` clean. `go build` and `go vet` clean.
- New feedback memory: `feedback_target_resolution_uses_actor.md`
  capturing the rule "user-facing target resolution goes through
  `room.ResolveTargetActor`; commands operate on `actions.Actor`,
  type-assert at the leaf for mob-only or user-only behavior."
- `MEMORY.md`: completion entry under `Completed (2026-04-18)`;
  retire `project_target_resolution_refactor.md` (move to a
  Resolved section in MEMORY.md or delete + reference the merge
  commit).

## Out of Scope (Tracked Elsewhere)

- **Downstream `mobs.GetInstance` lookups** (~170 sites by known ID,
  not name resolution). Could benefit from a
  `mobs.SafeGetInstance(id) (*Mob, bool)` helper to force callers to
  acknowledge nil. Logged as a follow-up project memory after this
  pass lands; not blocking.
- **`Actor` interface extension.** Even though every command's leaf
  assertion adds a tiny amount of boilerplate, widening the
  interface to add `IsNonCombatant()` etc. would propagate the cost
  across every Actor implementor and dilute the interface's combat
  focus. Defer indefinitely.
- **`FindByName` cleanup.** The room method's two return values
  (`playerId, mobInstanceId`) are now mostly an implementation
  detail of `ResolveTargetActor`. Direct callers may shrink
  significantly. Don't delete or change `FindByName` itself in this
  pass — just stop using it directly from commands.
- **Mobcommand parity** — mobcommands call mostly the same patterns
  as usercommands but for mob-initiated actions. They get migrated
  in this pass for symmetry, but a deeper "do mobs really need their
  own command package?" question is outside this refactor.

## Decision Log

- **2026-04-18**: Helper lives on `Room` as a method
  (`r.ResolveTargetActor`), not as a free function in `actions` or
  `mobs`. Rationale: room is the targeting context; this matches
  `r.FindByName` colocation; commands already have `room` in scope.
- **2026-04-18**: Use sentinel errors (`ErrTargetNotFound`,
  `ErrTargetVanished`, `ErrTargetSelfExcluded`) instead of returning
  `(actor, found bool)`. Rationale: typed errors let callers give
  better messages without leaking implementation details; matches Go
  stdlib idioms.
- **2026-04-18**: Don't extend Actor with mob-only methods. Use
  type assertions at leaves. Rationale: same decision as combat
  unification; keeps Actor focused on combat semantics; assertion
  pattern is already familiar to anyone reading the post-2a/b
  combat code.
- **2026-04-18**: Players take precedence over mobs when both
  match. Rationale: matches the implicit convention in existing
  call sites (most check `playerId > 0` first) and aligns with the
  intuition that named players are usually the intended target.
  Document this in the helper's docstring. **Known limitation:**
  this is a workaround, not a fix — nothing prevents name
  collisions in the first place. Tracked as future work in
  `project_name_collision_prevention.md` (player-creation guard,
  reserved-word list, and/or actor disambiguation syntax).
- **2026-04-18**: Migrate per-category (6 commits). Rationale:
  bisect-friendly; each commit has bounded blast radius; reviewer
  (and human smoke-tester) can verify category by category.
