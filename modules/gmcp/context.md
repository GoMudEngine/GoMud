# GMCP Module Context

## Overview

The `modules/gmcp` package implements the GMCP (Generic Mud Communication
Protocol) sub-negotiation layer that pushes structured JSON payloads from
the server to GMCP-aware clients (web client, Mudlet, etc.). Each
`gmcp.<Name>.go` file is a self-contained module: it registers event
listeners in its `init()` function and emits `GMCPOut` events to deliver
payloads to connected clients. All modules follow the same
listener/payload pattern; this document covers the `Char.Automation`
module (Phase 1–4) and the `Char.Vitals` additions from Phase 3.

## Char.Automation Module (`gmcp.Automation.go`)

### Purpose

Keeps the web-client automation panel in sync with the player's
server-side macro and alias data. On any change the module pushes a
fresh `Char.Automation` payload over GMCP so the panel reflects the
current state without a full page reload.

### Outbound Payload

Module name (as sent over GMCP): **`Char.Automation`**

```json
{
  "macros":    [{ "key": "=1", "commands": "wave;say hi" }, ...],
  "aliases":   [{ "name": "ms", "command": "cast mind-spike" }, ...],
  "ticks":     [{ "id": "abc123", "name": "My Timer", "commands": "forage;rest",
                  "intervalSec": 30, "enabled": true }, ...],
  "triggers":  [{ "id": "def456", "name": "Auto-heal", "pattern": "* hits you*",
                  "commands": "cast heal", "enabled": true,
                  "queueMode": "back",
                  "condition": {
                    "sourceKind": "pool_pct",
                    "sourceKey":  "hp",
                    "op":         "below",
                    "values":     ["40"]
                  },
                  "thenCommands": "cast heal",
                  "elseCommands": "" }, ...]
}
```

All arrays are sorted by key/name/id for stable ordering. Fields:

| Field                       | Type     | Description                                        |
|-----------------------------|----------|----------------------------------------------------|
| `macros[].key`              | string   | Macro slot identifier (e.g. `"=1"`)               |
| `macros[].commands`         | string   | Semicolon-delimited command string                 |
| `aliases[].name`            | string   | Alias name (e.g. `"ms"`)                          |
| `aliases[].command`         | string   | Expanded command string                            |
| `ticks[].id`                | string   | Unique identifier for the tick                     |
| `ticks[].name`              | string   | Human-readable label shown in the panel            |
| `ticks[].commands`          | string   | Semicolon-delimited command string                 |
| `ticks[].intervalSec`       | int      | Fire interval in seconds (minimum 1)               |
| `ticks[].enabled`           | bool     | Whether the tick is currently active               |
| `triggers[].id`             | string   | Unique identifier for the trigger                  |
| `triggers[].name`           | string   | Human-readable label shown in the panel            |
| `triggers[].pattern`        | string   | Wildcard pattern; `*` captures into `$1`, `$2`, … |
| `triggers[].commands`       | string   | Commands run when pattern matches (no condition)   |
| `triggers[].enabled`        | bool     | Whether the trigger is currently active            |
| `triggers[].condition`      | object?  | Optional condition; `null` when not set            |
| `triggers[].thenCommands`   | string   | Commands run when condition is true (may be empty) |
| `triggers[].elseCommands`   | string   | Commands run when condition is false (may be empty)|
| `triggers[].queueMode`      | string   | `""` (fire immediately, default), `"back"` (add to end of action queue), or `"front"` (add to front — priority) |

**`triggers[].condition` shape** (when present):

| Field        | Type     | Values / notes                                          |
|--------------|----------|---------------------------------------------------------|
| `sourceKind` | string   | `"pool_pct"` / `"conditions"` / `"capture"` / `"target"` / `"cooldown"` |
| `sourceKey`  | string   | Disambiguator: pool name (`"hp"`, `"sp"`, `"cp"`), capture index (`"$1"`), ability name, or empty for others |
| `op`         | string   | Operator string (see table below)                       |
| `values`     | []string | Operand(s); single-element for most sources, multi for target list |

Operators by `sourceKind`:

| `sourceKind`  | Valid `op` values                    | `values` element(s)             |
|---------------|--------------------------------------|---------------------------------|
| `pool_pct`    | `"below"`, `"above"`, `"equals"`    | `["40"]` (percent, 0–100)      |
| `conditions`  | `"includes"`, `"excludes"`          | `["poisoned"]`                  |
| `capture`     | `"equals"`, `"contains"`            | `["some text"]`                 |
| `target`      | `"is_one_of"`, `"is_not_one_of"`    | `["troll","orc","goblin"]`      |
| `cooldown`    | `"ready"`, `"not_ready"`            | `["cast heal"]`                 |

When a condition is present, `commands` is ignored; `thenCommands` and
`elseCommands` drive behaviour. An empty string for either branch means
"do nothing" for that outcome.

**Available-pool-% note:** pool percentages for the `pool_pct` source
are computed against the player's usable (unreserved) pool, not the
raw total. The client derives this from `Char.Vitals` fields:
`hp_reserved`, `stamina_reserved`, `conviction_reserved` (see
`Char.Vitals` section below).

### Push Triggers

The module registers two event listeners in `init()`:

| Event                        | When it fires                                         |
|------------------------------|-------------------------------------------------------|
| `events.PlayerSpawn{}`       | Player logs in — sends the full current state         |
| `events.AutomationChanged{}` | Any macro, alias, tick, or trigger is changed/removed |

Both listeners call `sendAutomation(userId)`, which:
1. Looks up the `UserRecord` for the given `UserId`.
2. Skips silently if GMCP is not enabled for that connection
   (`isGMCPEnabled(connectionId)`).
3. Calls `buildAutomationPayload(...)` to assemble a sorted, stable
   payload including macros, aliases, ticks, and triggers.
4. Enqueues a `GMCPOut{UserId, Module: "Char.Automation", Payload: ...}`
   event for delivery.

### Event Emitters

`events.AutomationChanged{UserId}` is emitted by:

- `internal/usercommands/set.go` — `cmdSetMacro`: after any `set =#`
  or `set =# command` that adds, updates, or clears a macro slot.
- `internal/usercommands/alias.go` — `Alias`: after any
  `alias name=value` or `alias name=` that creates, updates, or
  removes a custom alias.

### Inbound GMCP

Ticks and triggers are managed exclusively through the web panel via
inbound GMCP messages — there is no typed command for either. The web
client sends binary GMCP frames to `gmcp.go`'s `HandleIAC` switch:

| Inbound message          | `kind` gate  | Action                                          |
|--------------------------|--------------|-------------------------------------------------|
| `Char.Automation.Set`    | `"tick"`     | Create or update a tick on the `UserRecord`     |
| `Char.Automation.Remove` | `"tick"`     | Delete a tick by `id` from the `UserRecord`     |
| `Char.Automation.Set`    | `"trigger"`  | Create or update a trigger on the `UserRecord`  |
| `Char.Automation.Remove` | `"trigger"`  | Delete a trigger by `id` from the `UserRecord`  |

`Char.Automation.Set` payload shape for a **tick**:
```json
{ "kind": "tick", "id": "abc123", "name": "My Timer",
  "commands": "forage;rest", "intervalSec": 30, "enabled": true }
```

`Char.Automation.Set` payload shape for a **trigger**:
```json
{ "kind": "trigger", "id": "def456", "name": "Auto-heal",
  "pattern": "* hits you*", "commands": "cast heal", "enabled": true,
  "queueMode": "front",
  "condition": { "sourceKind": "pool_pct", "sourceKey": "hp",
                 "op": "below", "values": ["40"] },
  "thenCommands": "cast heal", "elseCommands": "" }
```

`Char.Automation.Remove` payload shape (same for tick and trigger):
```json
{ "kind": "trigger", "id": "def456" }
```

After processing, either handler emits `events.AutomationChanged{UserId}`
to re-push the full `Char.Automation` payload to the client. Unknown
`kind` values are silently ignored.

See `docs/superpowers/specs/2026-06-07-web-client-automation-panel-design.md`
for the full Phase 2–3 design.

## Char.Automation — Action Queue (Phase 4)

### Purpose

Phase 4 added a client-side FIFO action queue so that multiple triggers
firing at once (e.g. several buffs expiring simultaneously) do not stomp
each other. Instead of sending all commands immediately, each trigger with
a non-empty `queueMode` pushes its resolved command onto the queue, which
drains one entry per shared ability cooldown.

### `queueMode` field

The `queueMode` field is included on every trigger in both the outbound
`Char.Automation` payload and the inbound `Char.Automation.Set` message.

| Value      | Behaviour                                                         |
|------------|-------------------------------------------------------------------|
| `""`       | Fire immediately (default, backward-compatible)                   |
| `"back"`   | Append to the end of the action queue                             |
| `"front"`  | Prepend to the front of the queue (priority); multiple concurrent |
|            | `"front"` entries preserve their arrival order (priority FIFO)   |

### Client-side queue behaviour

The queue is implemented entirely in the web client (not on the server).
Key rules:

- **Shared cooldown gate.** Kick, trip, bash, grapple, taunt, rally,
  warcry, and every spell share one cooldown tracked as
  `Commands.State.cooldowns["special-move"]`. The queue drains one entry
  only when that cooldown is free.
- **Dedup.** A trigger whose resolved command is already present in the
  queue will not add a second entry.
- **Cap.** The queue holds at most 10 entries.
- **Ephemeral.** The queue is cleared on: the Clear button in the panel,
  `Commands.State.mode == "downed"` (death), and any page reload or
  reconnect. It is never persisted to the server.
- **No auto-retry.** If a queued command fails (e.g. concentration lost),
  the queue does not re-add it. Authors should write a companion trigger
  that matches the failure message and re-queues the command.

### Panel UI

Triggers that are currently in the queue are highlighted and floated to
the top of the Triggers tab. A badge on each shows its position (1 =
next to fire).

## Char.Vitals — Reserved-Pool Fields (Phase 3)

### Purpose

The `Char.Vitals` payload was extended in Phase 3 to expose the
**reserved** portion of each resource pool. The web client uses these
values to compute a player's _usable_ pool for trigger condition
evaluation: pool-percent conditions compare against the unreserved
fraction, not the raw total.

### Additional Fields

These three fields were appended to the existing `Char.Vitals` payload
(`gmcp.Char.go`) in Phase 3:

| Field               | Type | Description                                        |
|---------------------|------|----------------------------------------------------|
| `hp_reserved`       | int  | HP currently set aside and unavailable for use     |
| `stamina_reserved`  | int  | Stamina currently set aside and unavailable         |
| `conviction_reserved` | int | Conviction currently set aside and unavailable    |

### Client Usage

The client trigger engine reads these fields from each incoming
`Char.Vitals` event and caches them alongside the raw pool values.
When evaluating a `pool_pct` condition the available (usable) value is:

```
usable     = current - reserved
usable_max = max - reserved
usable_pct = usable / usable_max   (clamped 0–1)
```

For example, if `hp = 120`, `hp_max = 200`, and `hp_reserved = 100`,
then `usable = 20`, `usable_max = 100`, and `usable_pct = 20%`. A
condition of "HP below 30%" would therefore be true in this state, even
though raw HP (60%) is well above 30%.

## Module index

Every `gmcp.<Name>.go` file follows the same shape: register in `init()`, emit
via `GMCPOut`. `gmcp.go` holds the shared plumbing. Read the file for its exact
payload schema and push triggers — this table is the map, not the schema.

| File | Package | What it carries |
|------|---------|-----------------|
| `gmcp.go` | — | Module registration, `GMCPOut`, shared helpers |
| `gmcp.Char.go` | `Char` | Vitals, stats, status — the core player feed |
| `gmcp.Automation.go` | `Char.Automation` | Triggers, macros, aliases, the action queue |
| `gmcp.Room.go` | `Room` | Current room: description, exits, occupants |
| `gmcp.Zone.go` | `Zone` | The `Zone.Map` snapshot the web mapper draws, plus `party` |
| `gmcp.Party.go` | `Party` | Party roster and member vitals |
| `gmcp.Comm.go` | `Comm` | Channel messages |
| `gmcp.Item.go` / `gmcp.Item_refs.go` | `Char.Items` | Inventory and equipment, plus item reference data |
| `gmcp.Quest.go` / `gmcp.Quest_refs.go` | `Quest` | Quest tracker state and quest reference data |
| `gmcp.Mob.go` | `Mob` | Mob presence and vitals for the room panel |
| `gmcp.Mutation.go` | `Mutation` | Mutation roster and reveal events |
| `gmcp.Dialogue.go` | `Dialogue` | NPC conversation state for the client |
| `gmcp.Action.go` | `Action` | Clickable action affordances |
| `gmcp.Commands.go` | `Commands` | Command state, modes, cooldowns |
| `gmcp.Behavior.go` | `Behavior` | Behaviour-tree state (admin/builder tooling) |
| `gmcp.Build.go` | `Build` | The admin web building tools' data feed |
| `gmcp.Game.go` | `Game` | Game-level metadata |
| `gmcp.World.go` | `World` | World-level state (weather, time) |
| `gmcp.Mudlet.go` | `Client.GUI` | Mudlet-specific package download support |

The `_refs` files exist because reference data (item and quest definitions) is
large and static: it is pushed once rather than on every update, and the live
payloads carry ids that index into it.

## Gotchas

- **The web client is the primary consumer of every one of these.** A payload
  change is a client change; check `_datafiles/html/public/static/js/gmcp.js`
  before altering a field name.
- **`Zone.Map` is fog-of-war filtered** by `Character.VisitedRooms` before it is
  sent — see `internal/mapper`'s `Snapshot`.
- **Emit only on change where you can.** These push on room change and round
  boundaries; adding an unconditional per-round emit to a large payload is a
  bandwidth regression that will not show up in a local test.
