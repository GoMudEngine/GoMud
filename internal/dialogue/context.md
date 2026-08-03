# Dialogue Context

## Purpose

`internal/dialogue` is the NPC conversation engine: it loads a mob's dialogue
YAML, matches what the player said against keyword patterns or a stateful
conversation tree, applies whatever quest effects the matched node carries, and
returns the text to speak plus a hint line.

It deliberately does **not** import `characters` or `users`. Everything it needs
to know about the player arrives through `PlayerState`, a struct of callbacks.
That keeps the package testable and keeps the import graph acyclic.

Two conversation styles coexist in one file:

- **Patterns** — stateless keyword → response rules. Good for ambient topics.
- **Tree** — stateful nodes with triggers, unlock requirements, and per-player
  memory. Good for quest lines.

## Files

- **types.go** — every YAML shape plus `PlayerState`.
- **engine.go** — `Match`, `MatchWithFallbackInfo`, `TreeAdvance`, `Greet`, and
  the shared gate/effect helpers.
- **loader.go** — `Load(mobId, zone)`, zone-name sanitising, exclusion warnings.
- **save.go** — `SaveDialogueFile`, `CreateNewDialogueFile`,
  `DeleteDialogueFile` (the admin web builder writes through these).
- **memory.go** — per-(mob instance, player) conversation memory.
- **mood.go** — per-mob-instance mood state.
- **greetings.go** — ambient arrival greetings and greet-once tracking.
- **validate.go** — `ValidateDialogueFile` with injectable validators.
- **validation.go** — `CollectFlagReferences` for quest-flag cross-checking.
- **walk.go** — `WalkAllFiles`, used by audits and boot-time validation.

## Data model

```go
type DialogueFile struct {
    MobId       int
    Zone        string
    DefaultMood string
    Greetings   []Greeting
    Patterns    []Pattern
    Tree        *Tree          // optional
    Memory      MemoryConfig
}
```

`Pattern`, `TreeNode`, and `QuestGreeting` each carry the **same ten gating
fields**: `QuestRequired`, `QuestExcluded`, `GrantsQuest`, `RequiresItem`,
`GivesItem`, `QuestFlagRequired`, `QuestFlagExcluded`, `SetsQuestFlag`,
`BumpsRep`, `GivesGold`, `MasterworkRequired`. Two shared helpers do the work
for all three:

```go
func checkQuestGate(questRequired, questExcluded []string, requiresItem int,
    flagRequired, flagExcluded map[string]string, masterworkRequired int, ps *PlayerState) bool
func applyQuestEffects(grantsQuest string, requiresItem, givesItem int,
    flagSet *QuestFlagSet, bumpsRep []RepBump, givesGold int, ps *PlayerState) bool
```

`applyQuestEffects` attempts `GiveItem` **before every other effect** and
returns false — applying nothing else — when delivery fails (2026-08-03
soft-lock fix): no quest grant, no `requiresItem` removal, no flags/rep/
gold, and `TreeAdvance` skips `UpdateMemory` so the node re-fires once
the player makes room. `PlayerState.GiveItem` is `func(int) bool`; the
production callback in `usercommands.buildPlayerState` returns
`StoreItem`'s verdict and returns false (with an error log) for an
invalid item id. A nil `GiveItem` counts as delivered (skip-checks
contract).

Moods: `friendly`, `neutral`, `hostile`, `afraid`, `grateful`. A pattern or
greeting can restrict itself to a subset via `moods:`, and matching a node can
shift the mood via `moodChange:`.

## Public API

```go
func Load(mobId int, zone string) *DialogueFile
func WalkAllFiles(fn func(mobId int, zone string, df *DialogueFile))

func Match(df *DialogueFile, mobInstanceId int, topic string, ps *PlayerState) (text, hints string, ok bool)
func MatchWithFallbackInfo(df *DialogueFile, mobInstanceId int, topic string, ps *PlayerState) (text, hints string, ok, wasFallback bool)
func TreeAdvance(df *DialogueFile, mobInstanceId, userId int, topic string, ps *PlayerState) (text, hints, nodeId string, ok bool)
func Greet(df *DialogueFile, mobInstanceId, userId int, ps *PlayerState) (text, hints string, ok bool)
func PickGreeting(gs []Greeting, currentMood Mood) (string, bool)

func GetMood(mobInstanceId int, defaultMood string) Mood
func SetMood(mobInstanceId int, mood Mood)
func ShiftMood(mobInstanceId int, target, defaultMood string)

func GetMemory(mobInstanceId, userId int) *PlayerMemory
func UpdateMemory(mobInstanceId, userId int, nodeId string, unlocks []string, topic string)
func IsExpired(mem *PlayerMemory, expiryPeriod string) bool
func ResetMemory(mobInstanceId, userId int)
func HasGreeted(mobInstanceId, userId int) bool
func MarkGreeted(mobInstanceId, userId int)

func ValidateDialogueFile(df DialogueFile, v DialogueValidators) (errs, warns []string)
func CollectFlagReferences(df *DialogueFile) (refs, sets []FlagRef)

func SaveDialogueFile(df DialogueFile) error
func CreateNewDialogueFile(mobId int, zone string) error
func DeleteDialogueFile(mobId int, zone string) error
```

## Per-player memory

```go
type PlayerMemory struct {
    VisitCount      int
    LastVisitRound  uint64
    UnlockedNodes   map[string]bool
    CurrentRootSeen bool
    Greeted         bool
    RecentTopics    []string   // capped at 5
}
```

Keyed by `mobInstanceId<<32 | uint32(userId)` in a package-level map. **This is
in-process only** — never written to disk, so every restart resets who has been
greeted and which nodes are unlocked.

The map is swept so it cannot grow for the life of the process:

```go
func SweepMemories() int              // drops entries idle > memorySweepIdleRounds
func ForgetMobInstance(mobId int) int // drops every player's memory of one instance
```

`hooks.SweepDialogueMemory` runs the sweep every 500 turns. Call
`ForgetMobInstance` when a mob despawns — instance ids are reused, and a new
mob must not inherit a stranger's conversation history.

## Gotchas

- **A `nil` `*PlayerState` disables all gating.** Accepted for backward
  compatibility with mob-to-mob and other non-user contexts; every requirement,
  exclusion, flag check and masterwork gate is skipped. Never pass nil from a
  production path — `buildPlayerState` always returns a populated one.
- **Every callback field is optional and individually nil-checked.** A missing
  *interrogative* callback fails the gate **closed** (no `HasQuest` means a
  `questRequired` node is hidden, because absent information is not proof the
  player qualifies). `questExcluded` is the deliberate exception: without
  `HasQuest` we cannot confirm the excluding token, so the node stays visible
  rather than silently vanishing.
- **`TreeAdvance` substring-matches triggers in file order.** Put quest **grant
  nodes first** under `tree.nodes`, or an earlier generic node will swallow the
  trigger word.
- **Memory is per *instance*, not per template.** Respawn the mob and the
  player's unlocked nodes are gone. Prefer `questRequired` over `requires` for
  anything that must survive — `requires` reads `UnlockedNodes`, which is
  volatile.
- **`expiryPeriod` should almost always be empty.** The only legitimate use is
  a quest where urgency is the design intent; otherwise it bricks conversations.
- **The filename must be `<mobId>.yaml`.** A mismatch does not error — it
  silently mutes the NPC.
- **A bare scalar where a list is expected nils the entire file.** Writing
  `questRequired: "X"` instead of `["X"]` makes yaml.v2 fail the whole
  document, and because dialogue loads lazily the boot test will not catch it.
- **Colons and semicolons in `text:`/`hints:` break the YAML or the renderer.**
  Use `>` block scalars or an em-dash.
- **Voice rules are enforced by review, not by code:** `text` is the NPC
  speaking (first person); `hints` is narrator text addressed to the player
  (never a third-person self-reference).
- **Every `grantsQuest` node needs the quest's end token in `questExcluded`**,
  or a player who finished the quest gets re-offered it. The loader warns at
  runtime when this is missing.

## Dependencies

`gametime` (memory expiry), `util` (round count), `mudlog`, plus the YAML
loader. Quest effects are applied through `PlayerState` callbacks, so there is
no compile-time dependency on the quest engine.

## Consumers

`internal/usercommands` (`talk`, `ask`), `internal/mobs`, `internal/hooks`,
`internal/web` (the admin dialogue editor), and the boot-time validation pass.
