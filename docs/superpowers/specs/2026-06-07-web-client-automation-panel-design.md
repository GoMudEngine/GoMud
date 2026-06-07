# Web Client Automation Panel — Design (Ticks · Triggers · Macros · Aliases)

**Date:** 2026-06-07
**Status:** Approved (brainstorm) — ready for implementation plan
**Author:** brainstormed with the visual companion

## Goal

Fill the reserved `#panel-trig` dashboard slot with a tabbed **automation panel**
covering four per-account QOL systems — **Ticks**, **Triggers**, **Macros**,
**Aliases** — using the same leather tiles + right-click model as the inventory
panel. This is **sub-project #3** of the web overhaul (see
[[project_web_overhaul_sequence]]) and, like the rest, stays **parked locally —
NOT pushed to prod** until the user decides to push the batch.

Reference: the inventory panel ([[project_web_client_inventory_panel]]) — its
tabbed-grid + right-click-action + GMCP-driven pattern is the template here.

## Context

- **Macros already exist** (server-side): `user.Macros` is `map[string]string`
  (number → `;`-separated commands). Listed by the `macros` command, set via
  `set macro N …`, invoked by `=N` (the web client already binds F1–F12 to
  `=N`, webclient-pure.html:1390).
- **Aliases already exist** (server-side): `user.Aliases` is `map[string]string`
  (name → command). Set/removed via `alias name=value` (empty value deletes;
  `user.AddCommandAlias`). There are also read-only built-in global aliases —
  the panel edits only the user's custom map.
- **Ticks & Triggers do NOT exist** — no server-side timer or trigger concept.
  Net-new.
- **GMCP is event-driven** (`Char.Inventory`, `Char.Conditions`, vitals) —
  pushed on login and on change. `Char.Conditions` already feeds the Status &
  Conditions panel (its freshness was fixed in the terminal-theming work).
- **The panel slot** `#panel-trig` (right column) is a placeholder today
  ("Triggers & timers — coming soon", webclient-pure.html:304) with CSS wired
  (`--f-trig`).
- **No-hard-numbers SOP** (CLAUDE.md): never surface raw pool figures to
  players. Pool conditions are therefore **percentage-only**.
- **Reserved pools:** a player can have a portion of a pool reserved
  (unusable). The vitals GMCP does not currently send the reserved portion —
  see the related followup [[project_webclient_vitals_reserved_pool_viz]].

## Locked decisions (from the brainstorm)

1. **Four tabs:** Ticks · Triggers · Macros · Aliases.
2. **Server-stores / client-executes** for all four ("Hard B"). Definitions
   persist per-account in the user record; the *runtime* (tick timers, trigger
   matching) runs on the client. Consistent QOL — no half-local/half-server
   split.
3. **Triggers are Tier-1:** a wildcard pattern + one optional if/else
   condition. The data model and editor are structured so **Tier-2** (regex,
   multiple chained conditions, more action types) bolts on later without a
   redesign.
4. **Ticks are real-time-second intervals** (no game-round unit in v1).
5. **Interaction model:** each tab is a compact **list** + a **"+ New" add
   icon**. **Right-click** a row → Edit / Duplicate / Remove. **Left-click** a
   row → *fire it now*: tick → run its commands; trigger → test-fire its action;
   macro → `=N`; alias → send the alias name (server expands it). Ticks &
   triggers each have an **enable/disable toggle** on the row.
6. **Add/Edit is a modal dialog** centered over the dashboard (dimmed backdrop),
   NOT crammed into the panel slot — the form size is independent of the slot.
7. **Trigger editor is a progressive builder:** starts minimal (Name + "When I
   see" pattern → commands). A **"+ Add ▾" dropdown** inserts the optional
   if/else clause. The dropdown's greyed entries (Highlight, Play sound, extra
   conditions) are the documented Tier-2 seam.
8. **Trigger condition sources (4 kinds);** operator + value adapt to the kind:
   - **Live pool** — my HP% / SP% / CP% → `is below / above / equals` → a
     **percent** (never a raw number).
   - **Condition/status** — my conditions → `include / exclude` → a status name
     (poisoned, bleeding, sleeping…). Source data = the `Char.Conditions` GMCP
     feed (no new plumbing).
   - **Pattern capture** — `$1`, `$2`… → `equals / contains` → text.
   - **Target** — my target → `is one of / is not one of` → a **list** of
     names, **OR semantics** (match any; never AND).
9. **Pool % is measured against the AVAILABLE (unreserved) pool:**
   `pct = current ÷ (maxTotal − reserved) × 100`. This reflects usable health
   the player can actually act on, matches the usable part of the vitals bar,
   and stays meaningful as the reserve changes. **Helpfiles MUST state this
   explicitly** (see Part G) so players know the threshold is "% of usable
   pool," not "% of the total bar."

## Part A — Panel UI (client)

Replace the `#panel-trig` placeholder with the automation panel in
`webclient-pure.html` + `dashboard.css`:

- **Tab strip:** Ticks · Triggers · Macros · Aliases (madder-underlined active
  tab, matching the right-column accent). Default tab: Ticks.
- **List body (per tab):** a "+ New" add button + one row per item. Each row:
  optional enable toggle (ticks/triggers), a type icon, a name, and a one-line
  summary (e.g. the pattern, the interval, the `=N`, the expansion). Rows scroll
  within the slot.
- **Left-click row → fire now** (per type, decision #5). **Right-click row →**
  context menu Edit / Duplicate / Remove (Duplicate is client-side: read the
  def, open a New editor pre-filled with a "(copy)" name).
- **Modal editor** (decision #6), one form per type:
  - **Tick:** Name · Commands (textarea, `;`-joined) · Every N seconds ·
    Enabled.
  - **Trigger:** Name · "When I see" pattern (`*` wildcards → `$1,$2…`) · "Do"
    section with the **"+ Add ▾"** builder (optional if/else clause per Part F)
    · Enabled.
  - **Macro:** Number / F-key · Commands.
  - **Alias:** Alias name · Expands-to command.
- **Render** from the `Char.Automation` GMCP payload (Part C); re-render on its
  update. Normal dashboard panel behaviors (collapse / pop-out / responsive)
  apply.
- **Theme:** leather tiles, brass add/save buttons, gold/madder accents, IBM
  Plex Mono for the monospace bits (patterns, commands) — consistent with the
  shipped panels.

Visual source of truth: `trig-panel-v2.html` and `trig-condition-sources.html`
in the companion session (reproduced by this spec; not committed).

## Part B — Data model (server)

Macros (`user.Macros`) and Aliases (`user.Aliases`) are unchanged. Add two new
per-account collections to the user record, persisted with the user save:

- **`user.Ticks`** — ordered list / map of:
  `{ id, name, commands string (";"-joined), intervalSec int, enabled bool }`
- **`user.Triggers`** — ordered list / map of:
  `{ id, name, pattern string, condition *Condition, thenCmds string,
     elseCmds string, enabled bool }`
  where `Condition` (nil = no condition) is:
  `{ sourceKind enum(pool|status|capture|target), sourceKey string,
     op enum(below|above|equals|contains|include|exclude|oneOf|notOneOf),
     values []string }`
  - `values` is a list to support the target OR-list and to leave room for
    Tier-2 multi-value; pool/capture use a single entry.
  - The struct is intentionally open for Tier-2 (add `regex bool`, a
    `conditions []Condition` with a join mode, `actions []Action`) without
    breaking the v1 shape.

IDs are server-assigned stable handles (used by edit/remove and by the panel,
the way item UUIDs are used by the inventory panel).

## Part C — GMCP exposure

A new GMCP package emits **`Char.Automation`** — a single payload with all four
lists:
`{ ticks:[…], triggers:[…], macros:[…], aliases:[…] }`, each entry carrying its
id + fields the panel needs. Pushed on **login** and whenever any of the four
collections changes (add/edit/remove of a tick/trigger; `set macro`; `alias`).
Mirrors the `Char.Inventory` event-driven pattern.

**Vitals/conditions for trigger conditions:** the trigger engine reads live
state from existing GMCP — `Char.Conditions` (status list) and the vitals feed
(pools). For decision #9 the vitals feed must expose, per pool, enough to
compute available-%: **`current` + `availableMax` (= maxTotal − reserved)**, or
`current` + `maxTotal` + `reserved`. The vitals GMCP does not send reserved
today — extend it (this overlaps [[project_webclient_vitals_reserved_pool_viz]];
the bar-rendering half of that followup can stay separate, but the
reserved/available number must ship here).

## Part D — Server commands (CRUD + persistence path)

Per the admin/command wiring checklist ([[feedback-admin-command-wiring-checklist]]):
**every new command needs handler + registration + helpfile.**

- **Macros / Aliases:** reuse existing `set macro` / `alias name=value` for
  write; add removal where missing (`alias name=` already removes; confirm a
  macro-clear path). GMCP read covers the list.
- **Ticks:** new `tick` command — bare `tick` lists; `tick add|edit|remove …`
  mutate. Usable from a plain telnet client, not just the web panel.
- **Triggers:** new `trigger` command — bare `trigger` lists; `trigger
  add|edit|remove …` mutate.
- **Web panel persistence transport:** the modal's Save builds and sends the
  corresponding command (mirroring how the inventory panel fires `@handle`
  commands), so there is **one canonical write path** shared by telnet and web.
  *Open item (resolve in plan):* if encoding a full trigger (pattern + condition
  + then/else) as a single command string proves too fiddly to quote robustly,
  evaluate an **inbound client→server GMCP set** message as the structured
  alternative — confirm whether the GoMud GMCP layer accepts inbound custom
  packages before committing either way.

## Part E — Client runtime (the "executes" half)

- **Ticks:** for each enabled tick, a JS timer (`setInterval`, `intervalSec`)
  `SendData`s its commands. Toggling enable starts/stops the timer; left-click
  fires immediately; editing restarts the timer. Timers are rebuilt from each
  `Char.Automation` push.
- **Trigger engine:**
  1. **Tap the incoming stream** — in the websocket message handler, before
     `term.write`, strip ANSI to plain text and split into lines. (GMCP frames
     are out-of-band and excluded.)
  2. **Match** each enabled trigger's pattern: `*` → non-greedy capture groups;
     substring match; captures populate `$1…$n`.
  3. **Evaluate** the optional condition against live state:
     - pool → compare the **available-%** (Part C / decision #9) to the value;
     - status → membership test against the `Char.Conditions` set;
     - capture → string compare on `$n`;
     - target → OR-membership of the current target name against the list.
  4. **Fire** the then- (or else-) commands via `SendData`, substituting `$n`.
  - Left-click test-fire: evaluate the condition against *current* live state
    and run the matching branch (captures empty), so the player can dry-run it.
- Guardrails: a per-trigger cooldown / max-fires-per-second so a runaway
  pattern can't flood the server; document the cap.

## Part F — Trigger condition model (Tier-1)

The "Do" section is, by default, a single commands box (`pattern → commands`).
**"+ Add ▾ → If/else condition"** inserts ONE condition clause:

`If [source] [operator] [value(s)] → then [commands] (· else [commands])`

Sources / operators / values per decision #8. The clause renders as plain
English in the editor. Removing the clause (✕) returns to the plain
commands box. Exactly one condition in v1; the "+ Add" menu's greyed entries
mark where Tier-2 grows.

## Part G — Helpfiles (explicit, per user request)

New helpfiles for `tick` and `trigger` (and updates to `macros`/`alias` help to
cross-reference the panel). The trigger/`tick` help **must explicitly state**:

- **Pool conditions are percentage of your *usable* (unreserved) pool, not the
  total.** Spell out an example: "If 50% of your health is reserved, 'HP below
  30%' fires at 30% of the *remaining* half — i.e. when your usable health bar
  is below 30%, not the whole bar." This is the user's explicit requirement.
- Wildcard syntax (`*` and `$1…$n` captures).
- That triggers/ticks run in the client (web/compatible clients), while the
  definitions are saved to the account and sync everywhere.
- The runaway-trigger cooldown cap.

## Scope / boundaries

- **In:** the 4-tab panel, modal editors, server storage + GMCP + CRUD commands
  for ticks/triggers, GMCP read of macros/aliases (+ reused write commands), the
  client tick/trigger runtime, the vitals-GMCP reserved/available addition,
  helpfiles.
- **Out (Tier-2, deferred but architected-for):** regex patterns, multiple
  chained conditions (AND/OR), non-command actions (highlight, sound, set
  variable), game-round tick units. The vitals **bar** reserved-viz rendering
  stays in [[project_webclient_vitals_reserved_pool_viz]] (only the
  reserved/available *number* ships here).
- **Server work IS required** (new storage, GMCP, commands, vitals extension).
  Pre-push SOP boot test exercises these.

## Phasing (build order — each its own task group, server-first)

1. **Panel shell + Macros & Aliases tabs.** Panel UI, tab framework, modal
   editor scaffold, `Char.Automation` GMCP carrying macros+aliases, read +
   reused write commands. Proves the panel end-to-end against systems that
   already work.
2. **Ticks.** `user.Ticks` storage + GMCP + `tick` CRUD command + client timer
   runtime + tick editor + helpfile.
3. **Triggers.** `user.Triggers` storage (+ Condition struct) + GMCP + `trigger`
   CRUD + the vitals reserved/available GMCP addition + the client match/condition
   engine + builder editor + helpfile (with the explicit reserved-% wording).

## Acceptance / verification

- Panel renders all four tabs in `#panel-trig`; lists reflect real account data
  and update on change. Collapse/pop-out work.
- Macros/Aliases: add/edit/remove from the panel persists (visible via `macros`
  / `alias` in-game) and round-trips on relog. Left-click fires `=N` / the
  alias.
- Ticks: a tick fires its commands every N seconds while enabled; toggle pauses
  it; left-click fires now; survives relog.
- Triggers: a wildcard pattern matches the live stream and fires; `$n`
  substitution works; each condition source (pool-available-%, status,
  capture, target-OR-list) evaluates correctly; left-click dry-runs; runaway
  cooldown holds.
- Pool conditions use available-% (verify with a reserved pool: a trigger at
  "HP below 30%" fires relative to usable health, not total).
- Helpfiles state the available-% rule explicitly.
- `go build ./...` clean; server boots clean (storage load + GMCP + new
  commands + helpfiles); `/webclient` loads with no console errors.

## Risks / open items

- **Persistence transport** (Part D open item): command-string encoding vs
  inbound GMCP for structured trigger saves — resolve in the plan after
  confirming GoMud's inbound-GMCP support.
- **Stream tap fidelity:** ANSI stripping + line assembly for trigger matching
  must handle partial/chunked websocket frames (buffer to line boundaries).
- **Vitals GMCP extension** must land for the pool-% (available) source; coordinate
  with [[project_webclient_vitals_reserved_pool_viz]] so the two don't diverge.
- **Runaway triggers:** the cooldown cap is mandatory, not optional.
- **Tier-2 shape:** keep the `Condition`/storage structs open (lists, nullable)
  so Tier-2 doesn't force a migration.
