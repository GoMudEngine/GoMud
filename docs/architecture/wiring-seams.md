# Wiring seams (import-cycle breakers)

**Last verified: 2026-07-20** against `main.go` and the referenced packages.

Several packages need to call into a package that sits *above* them in the
import graph. Importing it directly would create a cycle, which Go rejects. The
engine breaks these cycles the same way every time:

1. The lower package declares a package-level function variable (or an
   interface field), defaulting to nil / no-op.
2. It exposes a `Set…` function to install the real implementation.
3. `main.go`, at startup, calls that `Set…` with a function from the higher
   package.

`main.go` is the composition root: it imports everything, so it is the one place
allowed to know about both sides of a cycle.

This file exists because those seams are individually well-commented but
collectively invisible — you cannot find them by reading either package in
isolation, only by grepping `main.go` for `Set*`. If you are wondering "how does
`rooms` call into `hooks` without importing it", this is the answer. **When you
add or remove one, update this file.**

## The seams

Every row is a package-level indirection wired in `main.go`. "Breaks" is the
import that would otherwise cycle.

| Injected into | Setter | Wired to | Breaks |
|---|---|---|---|
| `rooms` | `SetCompanionTransport` | `hooks.CompanionTransportCallback` | `rooms → hooks` |
| `rooms` | `SetBTreeStateEvictor` | `behaviortree.EvictRoomBTreeState` | `rooms → behaviortree` |
| `behaviortree` | `SetCompanionSweep` | `hooks.CompanionSweepCallback` | `behaviortree → hooks` |
| `characters` | `SetUserUntargetableCheck` | closure over `users` | `characters → users` |
| `users` | `SetCanSeeInRoomCheck` | closure over `characters`/`rooms` | `users → rooms` |
| `goals` | `SetWeightsLookup` | closure over `mobs`/`behaviortree` | `goals → behaviortree` |
| `goals` | `SetArchetypeDefaultsLookup` | closure over `mobs` | `goals → behaviortree` |
| `goals` | `SetPlanStateClear` | `planners.ClearPlanState` | `goals → planners` |
| `mobs` | `SetScheduleWorldValidator` | closures over `rooms`/`mapper` | `mobs → rooms` |
| `mobs` | `SetPatrolWorldValidator` | closures over `rooms`/`mapper` | `mobs → rooms` |
| `conversations` | `SetConversationWorldValidator` | closures over `mobs`/`relationships` | `conversations → mobs` |
| `usercommands` | `SetCopyoverFunc` | `main.triggerCopyover` | `usercommands → main` |
| `connections` | `IssueWebSocketReconnectToken` (assigned var, not a setter) | closure in `main.go` | `connections → users`/`world` |

## The other pattern: a dedicated adapter package

Two of the cycles are broken not with a function var but with a small package
that sits *above* both sides and depends on each:

- **`internal/conversationadapter`** implements `conversations.MobConversant`
  over an `*mobs.Mob`. `mobs` imports `conversations` (for `Load()`), so
  `conversations` cannot import `mobs`; the adapter depends on both and is
  imported by the callers (`hooks`, `usercommands`) instead. This one is
  already documented in its own package comment and in `CLAUDE.md`.

Prefer this shape over a function var when the seam is a whole interface's worth
of behaviour rather than a single callback — it keeps the contract type-checked
at compile time instead of relying on `main.go` remembering to wire a nil var.

## Gotchas

- **A missed wiring fails at runtime, not compile time.** If `main.go` forgets a
  `Set…`, the var stays nil/no-op and the feature silently does nothing —
  exactly the class of bug that left `util.SetServerAddress` unwired and the
  admin IP readout stuck on "Unknown" (see the util dead-code cleanup,
  2026-07-20). There is no compiler check that every seam is wired; this file
  and the boot smoke test are the backstops.
- **Ordering matters for validators.** The `mobs`/`conversations` world
  validators must be set *before* the corresponding `LoadDataFiles()` call, or
  the load runs without them. `main.go`'s `loadAllDataFiles` ordering already
  accounts for this.

## Related

- `docs/TECH_DEBT_AUDIT_2026-07-20.md` §4.3 (the finding this documents).
- `CLAUDE.md` — "codegraph MCP" note on verifying struct/signature shapes before
  dispatching work that touches these seams.
