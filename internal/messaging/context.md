# internal/messaging

Centralized player-facing-text pipeline. Every `Room.SendText` /
`Room.SendTextVisual` / `UserRecord.SendText` / `Actor.SendText` call
in the engine flows through this package's pipeline before reaching
the recipient's connection.

## Pipeline Stages

1. **Compose** — caller produces `(Category, text)`.
2. **Style normalize** — sentence-start caps, a/an agreement,
   duplicate-word collapse, sentence-end punctuation, ANSI canon for
   names. Per-Category skip table in `normalize.go`.
3. **Sight gate** (visual channel only) — per-recipient: CanSeeClearly,
   CanSeeShapes, or skip-visual-deliver-audio. Consumes the chunk-6
   Perception FSM (see `internal/state/perception/`).
4. **Anonymize** (infrared-only path) — regex strips `username` /
   `mobname` / `petname` ANSI name tags, substitutes "a figure" + the
   `combat-anon` color alias.
5. **Apply category color tag** — `<ansi fg="<category-alias>">…</ansi>`.
6. **Wrap** at recipient's `UserRecord.LineWidth` (default 80, range
   40–240), ANSI-aware.
7. **Deliver** to the recipient's connection.

## Channels

| Channel  | Helper            | Sight-gated | Stages run            |
|----------|-------------------|-------------|-----------------------|
| Audio    | `SendText`        | no          | 1, 2, 5, 6, 7         |
| Visual   | `SendTextVisual`  | yes         | all 7                 |

Audio bypasses the sight gate and the anonymizer; visual runs the
full per-recipient pipeline.

## Public API

Types and constants:

- `Category` — enum of 58 text classes (combat hits, defense, grapple,
  submissions, specials, spells by school, social, system, environment,
  loot/equipment/buff/mutation/toxin).
- `Channel` — `ChannelAudio`, `ChannelVisual`.
- `SightDecision` — `SightFull`, `SightShapes`, `SightNone`.
- `RenderInput` — bundles Category, Text, Channel, SightDecision,
  LineWidth for one recipient's pipeline pass.
- `ProgressionKind` — `ProgSkill`, `ProgStat`.
- `TierChange` — `{From, To string}`; nil for non-tier advancement.
- `RoomVisibility` — minimal interface (`GetVisibility() int`)
  satisfied by `*rooms.Room`.
- `UserSender` — minimal interface (`SendText(cat, text)`) satisfied
  by `*users.UserRecord`.

Functions:

- `RenderForRecipient(in RenderInput) string` — entry point; runs the
  full pipeline for one recipient. Empty return = "don't deliver".
- `CanSeeClearly(observer *characters.Character, room RoomVisibility) bool`
- `CanSeeShapes(observer *characters.Character, room RoomVisibility) bool`
- `Normalize(cat Category, text string) string`
- `Anonymize(text string) string`
- `WrapAnsi(text string, maxWidth int) string`
- `FormatProgression(kind, name, tier) string`
- `SendProgression(user UserSender, kind, name, tier)`

## Import-direction discipline

`messaging` imports `internal/characters`, `internal/buffs`,
`internal/state/perception` directly (sight predicates need
Perception FSM state and the NightVision / InfraredVision buff
flags). Everything else — `rooms`, `users`, `mobs`, `combat`,
`hooks` etc. — is consumed via narrow interfaces (`RoomVisibility`,
`UserSender`) so the dependency arrow stays one-way:

- Many packages import `messaging` (combat, hooks, rooms, users,
  actions, behaviortree, questengine, modules, world.go, …).
- `messaging` imports characters/buffs/perception ONLY.
- Nothing in `characters` imports `messaging` (would close a cycle).
  This is why `messaging.SendProgression` is the orphan: legacy
  progression banners live in `internal/characters/` and can't call
  `messaging` directly. See `[[progression-banner-deferred]]` in
  MEMORY.md for the followup.

## Adding a new Category

1. Add a constant to the enum in `messaging.go`. Append at the end of
   its section.
2. Add the matching string in `Category.String()`.
3. Add the color alias in `_datafiles/world/dogmud/ansi-aliases.yaml`
   named `<category-name>` where `<category-name>` is the string the
   enum returns.
4. If the new Category needs style-normalization skips, edit the
   `normalize.go` skip table.

## See Also

- `docs/superpowers/specs/2026-05-19-messaging-framework-design.md` —
  full design spec.
- `internal/state/perception/context.md` — the FSM whose state the
  sight gate reads (shipped dormant in chunk 6; this chunk is the
  consumer).
- `_datafiles/world/dogmud/ansi-aliases.yaml` — color aliases.
