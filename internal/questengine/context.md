# Quest Engine Context

## Purpose

`internal/questengine` is the **evaluation half** of the quest system: it takes
game events, matches them against indexed triggers, evaluates conditions, and
runs actions. It does not parse quest files.

The split matters. Since the 5c-pre unification, `internal/quests` is the single
owner of the quest file parse and of every definition type. `questengine`
re-exports those types as aliases (`types.go`) purely so its API and tests stay
source-compatible — **the data shapes live in `quests`, the logic lives here.**
When you change a quest YAML field, you edit `internal/quests`; when you change
what a trigger *does*, you edit this package.

## Files

- **engine.go** — `Engine`, the trigger index, `Notify`, and the recursive
  evaluator.
- **types.go** — type aliases onto `internal/quests`, plus `EventDetails` and
  `NotifyResult` (which describe an engine *invocation*, so they stay here).
- **conditions.go** — `PlayerState` interface and `EvalConditions`.
- **actions.go** — `ActionContext` interface and `ExecuteAction`.
- **bridge.go** — `GameBridge`, the concrete implementation of `ActionContext`
  against real users/rooms/mobs. The biggest file in the package.
- **guards.go** — `EvalGuard`: depth limit, visit tracking, grant dedupe, trace.
- **loader.go** — `GetEngine` singleton, `LoadDataFiles`, `ValidateAllFlags`.
- **map_target.go** — `ResolveQuestTarget`, which feeds the map quest marker.
- **logging.go** — level-gated logging with per-player debug opt-in.
- **test_helpers.go** — `ResetEngineForTest() func()`.

## The Engine

```go
type Engine struct {
    quests       map[int]*QuestDef
    triggerIndex map[string][]*indexedTrigger
}

func NewEngine() *Engine
func GetEngine() *Engine                       // process singleton
func (e *Engine) RegisterQuest(q *QuestDef)
func (e *Engine) GetQuest(questId int) *QuestDef
func (e *Engine) AllQuests() []*QuestDef       // unordered
func (e *Engine) Notify(eventType string, details EventDetails, player PlayerState, ctx ActionContext) NotifyResult
```

`RegisterQuest` flattens every trigger into `triggerIndex` keyed by event type,
so `Notify` never scans quests it cannot match. Each trigger gets an
engine-assigned `indexedTrigger{def, questId, trigId}` — an explicit wrapper
that replaced an older pattern of mutating unexported fields *on* the shared
definition struct. Definitions are immutable shared data owned by `quests`;
keep it that way.

## Evaluation flow

`Notify` → `evaluate` (recursive), per matching trigger:

1. **Field match** (`matchTriggerFields`) — mob, room, item, skill, command,
   topic, questToken, noun, verb.
2. **Visit guard** — `MarkVisited(trigId)`; a trigger fires at most once per
   chain.
3. **Conditions** — `EvalConditions(t.Conditions, player)`.
4. **Actions** — `executeActions`, which returns the tokens granted.
5. **Chain** — for each granted token, recurse into `quest_granted` triggers.

That last step is what makes quest chains work: granting `10-end` can
immediately fire the trigger that grants `11-start`.

## `EvalGuard`

```go
func NewEvalGuard(maxDepth int) *EvalGuard
func (g *EvalGuard) PushDepth() bool      // false once maxDepth is passed
func (g *EvalGuard) PopDepth()
func (g *EvalGuard) DepthExceeded() bool
func (g *EvalGuard) MarkVisited(trigId string) bool   // false if already seen
func (g *EvalGuard) MarkGranted(token string) bool    // false if already granted
func (g *EvalGuard) AddToTrace(desc string)
func (g *EvalGuard) GetTrace() []string
```

One guard per `Notify` call. It is the only thing standing between a circular
quest chain and an infinite recursion, and its trace is what gets logged when
the depth limit trips.

Config: `Balance.QuestChainDepthLimit` (max recursion) and
`Balance.QuestPerformanceWarnMs` (slow-evaluation warning threshold).

## `GameBridge`

`NewGameBridge(user *users.UserRecord, roomId int) *GameBridge` implements both
the read side (`HasQuest`, `HasItem`, `GetQuestFlag`, `GetGold`, `GetRoomId`,
`GetUserId`, `HasOwnMasterwork`) and the write side — grant/consume/give,
gold, text, mob and item spawning, spell teaching, skill training, stat
increases, recipe learning, buffs, teleport, exit locking, quest flags, faction
rep, mutation grants, NPC dialogue queueing, and timed sequences.

This is the seam that keeps the evaluator testable: tests supply a fake
`ActionContext`/`PlayerState`, production supplies `GameBridge`.

## Gotchas

- **Zero and empty trigger fields are wildcards.** `matchTriggerFields` skips
  any filter where the value is `0` or `""`. A trigger with `mob: 0` matches
  *every* mob, not "no mob". Omitting a field widens the trigger.
- **A trigger's actions are applied as a unit.** Each runs under its own
  `recover()`, but a failure or panic **abandons the remaining actions of that
  trigger** (logged with how many were dropped). There is no rollback of what
  already ran, so a failure still leaves a partial effect — aborting only stops
  it compounding. Treat action errors as real bugs. A duplicate grant is a skip,
  not a failure, and does not abort.
- **Grant dedupe is per evaluation chain, not per player.** `MarkGranted` lives
  on the guard, which is created fresh in `Notify`. It stops a token being
  granted twice in one cascade; it does not stop re-granting across events.
  Re-grant prevention is a content concern — see the quest-token `questExcluded`
  SOP in `CLAUDE.md`.
- **`AllQuests` is sorted by quest id.** It iterates a map internally but sorts
  before returning, so audit and gate output is stable between runs.
- **`command_issued` and `command` are different events** — the first fires when
  a command is typed, the second only when it succeeded.
- **Ephemeral (instance) rooms match their TEMPLATE room id** in `room:`
  triggers, not their runtime id.
- **`ConsumeItem` is only honoured on `item_give`.** Setting it on any other
  event silently does nothing. Note also that `give.go` transfers the item to
  the mob *before* any handler runs, so consumption is a post-hoc correction —
  see the give.go gotcha in `CLAUDE.md`.

## Dependencies

`internal/quests` (all definition types), `configs`, `users`, `mobs`, `items`,
`rooms`, `messaging`, `mudlog`.

## Consumers

`internal/hooks` (event dispatch), `internal/usercommands`, `internal/dialogue`
(quest grants from dialogue nodes), and `internal/mapper` consumers that render
the quest marker via `ResolveQuestTarget`.
