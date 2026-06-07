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
  "aliases": [{ "name": "ms",  "command": "cast mind-spike" }, ...]
}
```

Both arrays are sorted by key/name for stable ordering. Fields:

| Field              | Type     | Description                              |
|--------------------|----------|------------------------------------------|
| `macros[].key`     | string   | Macro slot identifier (e.g. `"=1"`)     |
| `macros[].commands`| string   | Semicolon-delimited command string       |
| `aliases[].name`   | string   | Alias name (e.g. `"ms"`)                |
| `aliases[].command`| string   | Expanded command string                  |

**Phase 2–3 additions:** `ticks` and `triggers` arrays will be appended
to this payload when those subsystems are implemented. The struct is
commented accordingly (`// Ticks/Triggers added in Phases 2-3.` in
`GMCPAutomation_Payload`).

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

### Inbound GMCP (Phases 2–3, Not Yet Built)

Ticks and triggers will be managed exclusively through the web panel via
inbound GMCP messages (no telnet command is planned for these). The
handlers will be added to the `HandleIAC` switch in `gmcp.go`:

| Inbound message              | Action                                              |
|------------------------------|-----------------------------------------------------|
| `Char.Automation.Set`        | Create or update a tick or trigger for the user     |
| `Char.Automation.Remove`     | Delete a tick or trigger by id                      |

After processing, each handler will emit `events.AutomationChanged{UserId}`
to trigger a re-push of the full `Char.Automation` payload.

See `docs/superpowers/specs/2026-06-07-web-client-automation-panel-design.md`
for the full Phase 2–3 design.

## Other GMCP Modules

The remaining `gmcp.<Name>.go` files (`gmcp.Char.go`, `gmcp.Zone.go`,
`gmcp.Room.go`, `gmcp.Party.go`, etc.) follow the same register-in-`init`
/ emit-`GMCPOut` pattern. See each file's source for its specific payload
schema and push triggers.
