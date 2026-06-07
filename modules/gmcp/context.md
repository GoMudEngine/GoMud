# GMCP Module Context

## Overview

The `modules/gmcp` package implements the GMCP (Generic Mud Communication
Protocol) sub-negotiation layer that pushes structured JSON payloads from
the server to GMCP-aware clients (web client, Mudlet, etc.). Each
`gmcp.<Name>.go` file is a self-contained module: it registers event
listeners in its `init()` function and emits `GMCPOut` events to deliver
payloads to connected clients. All modules follow the same
listener/payload pattern; this document currently focuses on the
`Char.Automation` module added in the web automation panel (Phase 1).

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
  "macros":  [{ "key": "=1", "commands": "wave;say hi" }, ...],
  "aliases": [{ "name": "ms",  "command": "cast mind-spike" }, ...],
  "ticks":   [{ "id": "abc123", "name": "My Timer", "commands": "forage;rest",
                "intervalSec": 30, "enabled": true }, ...]
}
```

All arrays are sorted by key/name/id for stable ordering. Fields:

| Field                  | Type     | Description                                   |
|------------------------|----------|-----------------------------------------------|
| `macros[].key`         | string   | Macro slot identifier (e.g. `"=1"`)          |
| `macros[].commands`    | string   | Semicolon-delimited command string            |
| `aliases[].name`       | string   | Alias name (e.g. `"ms"`)                     |
| `aliases[].command`    | string   | Expanded command string                       |
| `ticks[].id`           | string   | Unique identifier for the tick                |
| `ticks[].name`         | string   | Human-readable label shown in the panel       |
| `ticks[].commands`     | string   | Semicolon-delimited command string            |
| `ticks[].intervalSec`  | int      | Fire interval in seconds (minimum 1)          |
| `ticks[].enabled`      | bool     | Whether the tick is currently active          |

**Phase 3 addition:** a `triggers` array will be appended when the
trigger subsystem is implemented. The struct is commented accordingly
(`// Triggers added in Phase 3.` in `GMCPAutomation_Payload`).

### Push Triggers

The module registers two event listeners in `init()`:

| Event                        | When it fires                                      |
|------------------------------|----------------------------------------------------|
| `events.PlayerSpawn{}`       | Player logs in — sends the full current state      |
| `events.AutomationChanged{}` | Any macro or alias is added, edited, or removed    |

Both listeners call `sendAutomation(userId)`, which:
1. Looks up the `UserRecord` for the given `UserId`.
2. Skips silently if GMCP is not enabled for that connection
   (`isGMCPEnabled(connectionId)`).
3. Calls `buildAutomationPayload(user.Macros, user.Aliases)` to build a
   sorted, stable payload.
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

Ticks (Phase 2) are managed exclusively through the web panel via inbound
GMCP messages — there is no typed command for them. Triggers (Phase 3) will
use the same message names, gated by `kind`. The web client sends these as
binary GMCP frames to `gmcp.go`'s `HandleIAC` switch:

| Inbound message          | `kind` gate | Action                                         |
|--------------------------|-------------|------------------------------------------------|
| `Char.Automation.Set`    | `"tick"`    | Create or update a tick on the `UserRecord`    |
| `Char.Automation.Remove` | `"tick"`    | Delete a tick by `id` from the `UserRecord`    |

`Char.Automation.Set` payload shape:
```json
{ "kind": "tick", "id": "abc123", "name": "My Timer",
  "commands": "forage;rest", "intervalSec": 30, "enabled": true }
```

`Char.Automation.Remove` payload shape:
```json
{ "kind": "tick", "id": "abc123" }
```

After processing either handler emits `events.AutomationChanged{UserId}` to
trigger a re-push of the full `Char.Automation` payload to the client.

Phase 3 will add `kind == "trigger"` cases to both handlers. Until then,
unknown `kind` values are silently ignored.

See `docs/superpowers/specs/2026-06-07-web-client-automation-panel-design.md`
for the full Phase 2–3 design.

## Other GMCP Modules

The remaining `gmcp.<Name>.go` files (`gmcp.Char.go`, `gmcp.Zone.go`,
`gmcp.Room.go`, `gmcp.Party.go`, etc.) follow the same register-in-`init`
/ emit-`GMCPOut` pattern. See each file's source for its specific payload
schema and push triggers.
