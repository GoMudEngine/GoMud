# Web Client Inventory / Equipment Panel — Design

**Date:** 2026-06-07
**Status:** Approved (brainstorm) — ready for implementation plan
**Author:** brainstormed with the visual companion

## Goal

Add a **tabbed Inventory / Equipment panel** to the in-game web client,
GMCP-driven, with type icons and right-click actions that fire real commands.
It **replaces the reserved Scene-art slot** (`#panel-art`) in the dashboard
1:1 — scene-art is dropped from the overhaul. This is **sub-project #2** of the
web overhaul (see [[project_web_overhaul_sequence]]); sub-project #3
(Triggers/Tick/Macros card) is unchanged.

Reference the user liked: `inventory_image.png` (repo root) — a graphical
equipment + backpack grid with per-item context menus.

## Context

- The dashboard (sub-project #1, merged `cb3b8a55`) already has the panel
  framework, the reserved `#panel-art` slot, type tokens, and a GMCP dispatch
  (`GMCPStructs` / `GMCPUpdateHandlers`) in `webclient-pure.html`.
- GMCP `Char.Inventory` (`modules/gmcp/gmcp.Char.go`) already emits `Worn`
  (a fixed 10-slot struct) + `Backpack` (`items[]` + summary). Items carry
  `id` (`ShorthandId` = `!<ItemId>:<UUID>`), `name`, `type`, `subtype`,
  `uses`, `details[]`.
- GMCP `Char` is **event-driven** (listeners on `EquipmentChange`,
  `ItemOwnership`, `BuffsTriggered`, vitals, etc.) — NOT polled. So inventory
  updates push on equip/get/drop promptly (bounded by the turn flush).
- Equipment slots + canonical order live in `internal/characters/worn.go`
  (`type Worn struct`): Weapon, Offhand, ExtraArm1-4, Head, Neck, Shoulders,
  Body, Back, Belt, Wrist1-2, ExtraWrist1-4, Gloves, Ring, Ring2, Legs, Feet,
  Tail, ComponentBag. Mutation slots (ExtraArm/Wrist, Tail) exist only when the
  mutation is active.
- Item commands resolve targets by **name** (`FindInBackpack(rest)`), with
  diku/hash disambiguation (`N.item` / `item#N`). There is **no** instance-id
  targeting today.

## Locked decisions (from the brainstorm)

1. **Replaces scene-art 1:1** — the Inventory panel claims `#panel-art`.
2. **Item visuals: type-based icons now** — a small curated set of themeable
   medieval SVG icons keyed by item type/subtype (not per-item art, not
   text-only).
3. **Tabbed panel** — tabs: **Equipped** (default) · **Bandolier** ·
   **Components** · **Backpack**. Each tab is a tile grid of its contents.
4. **Equipped tab mirrors `eq`** — dynamic slot grid in `worn.go` order; base
   slots always shown; mutation slots shown only when active; empty slots
   render dashed with the slot label. Slots are labeled (the set is dynamic).
5. **No in-panel detail box** — right-click a tile → an item-type-aware action
   menu that fires real commands; all output goes to the **game feed**.
6. **Unambiguous targeting via an opaque item handle** — see "Item handle
   targeting" below. New server-side parser capability.
7. **Fold in the conditions-freshness fix** — make `Char.Conditions` push on
   condition change/expiry so the already-shipped Status panel isn't stale.

Visual source of truth:
`docs/superpowers/specs/2026-06-07-web-client-inventory-mockups/`
(`inventory-tabbed-v2.html`).

## Part A — GMCP `Char.Inventory` extension (server)

Rework the inventory payload so it carries everything the panel needs:

- **`Worn`** → change from the fixed 10-field struct to a **dynamic ordered
  list of slots** mirroring `eq`. Each entry: `{ slot: "<display label>",
  slotKey: "<weapon|offhand|extraarm1|…>", item: <Item|null> }`, emitted in
  `worn.go` order, **including the mutation slots only when the character has
  them** (so the client renders exactly the slots `eq` shows). Empty equippable
  slots are present with `item: null` so the client can draw the empty tile;
  disabled/non-existent slots are omitted.
- **`Bandolier`** → NEW: the potion-bandolier contents (`Character.PotionItems`)
  as an `items[]` list (+ capacity summary), present when a bandolier is worn.
- **`ComponentBag`** → NEW: the component-bag contents as an `items[]` list
  (+ capacity summary), present when a component bag is worn.
- **`Backpack`** → unchanged (`items[]` + count/max summary).
- Each item entry uses the existing item shape (`id`/handle, `name`, `type`,
  `subtype`, `uses`, `details[]`) — see Part B for the handle.
- **Push triggers:** ensure equip/remove/get/drop/use already fan out via
  `EquipmentChange` / `ItemOwnership` (they do). Add bandolier/component-bag
  mutations to the same push path if not already covered.

## Part B — Item handle targeting (server parser)

Add an **opaque per-instance handle** as a new way to target a specific item,
so the panel never grabs the wrong duplicate.

- **Handle = the item instance UUID** (already unique + random;
  `Item.UUID`). The GMCP item exposes this as an opaque `handle` (NOT the
  `!ItemId:UUID` shorthand, which leaks the template id — that's the
  "obfuscation": expose only the opaque instance token).
- **Parser:** extend the item-finding path so a target token with a reserved
  sigil (e.g. `@<handle>` — pick a sigil that can't collide with item names)
  resolves to the matching item **scoped to the actor's own reachable items**:
  backpack + worn + bandolier + component bag. Because resolution is scoped to
  the actor, a handle can only ever address an item that player owns (forgery
  is moot).
- **Commands:** the handle works for `look`, `identify`, `wear`/`wield`,
  `remove`, `drop`, `use`/`drink`/`eat` (the verbs the action menu fires).
- **Silent echo (chosen):** handle-targeted (panel-fired) commands are **not
  echoed** to the feed — the server suppresses the input echo
  (`EchoInputHandler` / input pipeline) for any command line whose target is a
  `@<handle>` token, so only the command's *result* appears (and that result
  still names the item). Manually-typed commands are unaffected. (The web
  client does not locally echo; the server is the only echo source, so the
  suppression must happen server-side.)
- **Open detail (spec-review):** whether the handle needs cryptographic
  signing/obfuscation beyond "opaque UUID, owner-scoped." Default: the raw
  UUID is already non-guessable and owner-scoped, so plain is sufficient;
  upgrade to a signed/encoded token only if we want to hide UUIDs entirely.

## Part C — Conditions-freshness fix (server)

The Status panel (shipped sub-project #1) only refreshes on `BuffsTriggered`
(buff-add), so buff **expiry** and other condition changes don't push until an
unrelated `Char` event — the observed 3-5s staleness. Fix: emit a
`Char.Conditions` GMCP update when conditions change or expire (e.g. on the
round-tick when the condition set differs from last sent, or on buff-removal
events). Scope: minimal — just close the freshness gap; no payload shape change.

## Part D — Client inventory panel

In `webclient-pure.html` + `dashboard.css` (or `dashboard.js`):

- Replace the `#panel-art` Scene slot contents with the **Inventory panel**:
  a tab strip (Equipped / Bandolier / Components / Backpack, Equipped active by
  default) + a tile-grid body per tab.
- **Tiles:** type-icon (Part E) on a brass-framed leather tile; equipped tiles
  carry a small slot label; empty equipped slots render dashed + labeled;
  duplicate stacks show `xN`. Hover shows the item name (title attr).
- **Render** from `GMCPStructs["Char"].Inventory` (Worn list / Bandolier /
  ComponentBag / Backpack); refresh on the `Char.Inventory` GMCP update
  handler. Keep the panel's pop-out / collapse / responsive behavior (it's a
  normal dashboard panel).
- **Right-click → action menu**, item-type-aware:
  - Always: **Look** (`look @<handle>`), **Identify** (cast the id spell:
    e.g. `cast identify @<handle>` — verify the exact id-spell invocation),
    **Drop** (`drop @<handle>`).
  - Worn items: **Remove** (`remove @<handle>`).
  - Backpack wearables: **Wear**/**Wield** (`wear`/`wield @<handle>`).
  - Consumables (potion/food/drink): **Drink**/**Eat**/**Use**.
  - Build the menu from the item's `type`/`subtype` + which tab/slot it's in.
- Commands fire via the existing `SendData(...)`; output appears in the feed.
- Left-click selects/highlights a tile (cosmetic); double-click could default
  to Look (optional).

## Part E — Type-icon set

A small curated set of monochrome **SVG** icons (themeable via `stroke`/`fill`
to the palette), keyed by item `type` (and a few `subtype`s where it matters —
sword/dagger/axe/wand/staff, shield, potion, ring, amulet, helm/body/legs/
feet/gloves/back/belt/shoulders, food, component, bag). Source a permissively-
licensed set (e.g. game-icons.net, CC-BY — attribute as required) or author
simple line icons; embed as inline SVG / a sprite. A generic fallback icon for
unknown types.

## Scope / boundaries

- **In:** Parts A-E + the conditions-freshness fix.
- **Out:** per-item unique art (type icons only for now); drag-and-drop
  equipping between tiles (right-click actions only for v1); the Triggers card
  (#3); scene-art (dropped).
- **Server work is required** (unlike sub-project #1): GMCP payload changes,
  the handle-targeting parser, the conditions fix. Pre-push SOP boot test will
  exercise these.

## Acceptance / verification

- `eq`-equivalent slots render in the Equipped tab in `worn.go` order, with
  mutation slots appearing/disappearing as the mutation toggles; empty slots
  labeled.
- Bandolier / Components / Backpack tabs show their real contents and update
  on change.
- Right-click actions fire the correct command against the exact clicked item
  (verified with duplicate-named items — the handle disambiguates); only the
  *result* lands in the feed (the `@<handle>` command line is NOT echoed); the
  item is described by name in that result.
- Equipping/dropping/using reflects in the panel promptly (no multi-second
  lag), and the Status panel now updates on buff expiry.
- `go build ./...` clean; server boots clean (data-file load + GMCP);
  `/webclient` loads with no console errors.

## Risks / open items

- **Handle sigil collision** — pick a target sigil (`@`, `#`, etc.) that can't
  be a legitimate item-name prefix; confirm the parser precedence so a handle
  is tried before name matching.
- **Identify invocation** — confirm the exact command to cast id on an item
  (`cast identify <target>` vs an `identify` command) before wiring the menu.
- **GMCP `Worn` shape change is breaking** for any other consumer of the
  current fixed struct (the leather mapper/vitals don't use it; check the
  web client's old equipment rendering is fully replaced).
- **Bandolier / Component Bag accessors** — confirm `Character.PotionItems`
  and the component-bag contents are readable for the payload.
