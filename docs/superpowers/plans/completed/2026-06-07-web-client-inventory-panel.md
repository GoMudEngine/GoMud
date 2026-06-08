# Web Client Inventory / Equipment Panel — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax. **Prefer the codegraph MCP** to confirm Go signatures/field names before editing — several integration points below say "confirm via codegraph"; do it.

**Goal:** A GMCP-driven, tabbed Inventory/Equipment panel in the web client (replacing the reserved Scene-art slot), with type icons and right-click actions that fire silent, handle-targeted commands into the game feed.

**Architecture:** Server first — expose an opaque per-item **handle** (instance UUID) over GMCP and teach the item finders to resolve `@<handle>` targets (owner-scoped), suppress the echo for those commands, extend `Char.Inventory` (dynamic `Worn` + Bandolier + Component Bag), and push `Char.Conditions` on buff expiry. Then the client renders the tabbed panel from that data and wires the action menu.

**Tech Stack:** Go (GoMud engine, GMCP module, characters/items packages), vanilla-JS/SVG web client. No new deps.

**Spec:** `docs/superpowers/specs/2026-06-07-web-client-inventory-panel-design.md`. Mockups: `docs/superpowers/specs/2026-06-07-web-client-inventory-mockups/`.

**Branch:** `feature/web-client-inventory-panel` (created; spec committed).

**Verification model:** Go tasks → `go build ./...` + targeted `go test` + a local **boot test** (data-file load + clean GMCP). Client tasks → `node --check` not applicable to HTML; structural self-review + a **human visual smoke** in `/webclient`. A dev server may hold ports 80/55555 — do NOT boot a second; the human smokes. GMCP/parser changes are exercised by the boot test and unit tests.

---

### Task 1: Expose an opaque item handle over GMCP

**Files:**
- Modify: `modules/gmcp/gmcp.Char.go` (the `GMCPCharModule_Payload_Inventory_Item` struct + `newInventory_Item`)

- [ ] **Step 1: Add a `handle` field to the GMCP item**

In `GMCPCharModule_Payload_Inventory_Item` (the struct with `Id`/`Name`/`Type`/`SubType`/`Uses`/`Details`), add:
```go
	Handle  string   `json:"handle"`   // opaque per-instance target token (item UUID)
```

In `newInventory_Item(itm items.Item)`, populate it from the instance UUID (confirm the field via codegraph — `Item.UUID` is a `uuid.UUID` per `ShorthandId()`):
```go
	d.Handle = itm.UUID.String()
```
Leave `Id` (the `!ItemId:UUID` shorthand) as-is for now; the client targets via `Handle`. (Optionally drop `Id` later to avoid leaking the template id — out of scope here.)

- [ ] **Step 2: Build + commit**

Run: `go build ./...` → expect exit 0.
```bash
git add modules/gmcp/gmcp.Char.go
git commit -m "feat(gmcp): expose opaque item handle (UUID) on inventory items"
```

---

### Task 2: Resolve `@<handle>` item targets in the finders (owner-scoped)

Teach the shared item finders to accept a handle token so every command (`look`/`wear`/`wield`/`remove`/`drop`/`use`/`drink`/`eat`/`identify` target) resolves the exact instance.

**Files:**
- Modify: `internal/characters/inventory.go` (`FindInBackpack`, `FindOnBody`, `FindItem`)
- Create: `internal/characters/inventory_handle_test.go`

- [ ] **Step 1: Add a handle-matching helper + a constant sigil**

Use codegraph to confirm `FindInBackpack`/`FindOnBody`/`FindItem` signatures and what `FindItem` searches (it returns `(items.Item, string, bool)` — item, source, found). Add a sigil constant and a matcher (new file or top of `inventory.go`):
```go
// ItemHandleSigil prefixes an opaque item handle target (e.g. "@<uuid>").
// Confirm '@' does not collide with an existing targeting/emote prefix; if it
// does, pick another reserved rune. (codegraph: search command target parsing.)
const ItemHandleSigil = "@"

func isItemHandle(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, ItemHandleSigil) && len(s) > len(ItemHandleSigil) {
		return s[len(ItemHandleSigil):], true
	}
	return "", false
}

func itemMatchesHandle(itm items.Item, handle string) bool {
	return itm.ItemId > 0 && itm.UUID.String() == handle
}
```

- [ ] **Step 2: Branch the finders on a handle token**

At the TOP of `FindInBackpack`, `FindOnBody`, and `FindItem`, before the name-matching logic, add a handle branch. For `FindInBackpack` (searches `c.Items` backpack), match against the backpack; for `FindOnBody` match against `c.GetAllWornItems()`; for `FindItem` (the unified finder) match across backpack + worn + `c.PotionItems` (bandolier) + the component-bag contents. Example for `FindItem`:
```go
func (c *Character) FindItem(itemName string) (items.Item, string, bool) {
	if h, ok := isItemHandle(itemName); ok {
		// owner-scoped: only this character's reachable items
		for _, itm := range c.Items {              // backpack
			if itemMatchesHandle(itm, h) { return itm, "backpack", true }
		}
		for _, itm := range c.GetAllWornItems() {   // equipped
			if itemMatchesHandle(itm, h) { return itm, "worn", true }
		}
		for _, itm := range c.PotionItems {         // bandolier
			if itemMatchesHandle(itm, h) { return itm, "bandolier", true }
		}
		// component bag contents — confirm the accessor via codegraph (Task 4)
		// for _, itm := range c.GetComponentBagContents() { ... }
		return items.Item{}, "", false
	}
	// ... existing name-matching logic unchanged ...
}
```
Do the analogous handle-first branch in `FindInBackpack` (backpack only) and `FindOnBody` (worn only) so commands that call those specifically still work.

- [ ] **Step 3: Failing test → implement → pass**

Write `inventory_handle_test.go`: build a `Character` with two same-named items in the backpack (distinct UUIDs), assert `FindItem("@"+uuidB)` returns the second one (not the first), and that a bogus handle returns `found=false`. Run it (fail → implement → pass):
```bash
go test ./internal/characters/ -run TestFindItem_ByHandle -v
```

- [ ] **Step 4: Build + commit**
```bash
go build ./... && go test ./internal/characters/ -run Handle
git add internal/characters/inventory.go internal/characters/inventory_handle_test.go
git commit -m "feat(items): resolve @<handle> item targets in finders (owner-scoped)"
```

---

### Task 3: Suppress the echo for handle-targeted commands (silent panel actions)

**Files:**
- Modify: `internal/inputhandlers/echo.go` (or the websocket input/echo path — confirm via codegraph which handler echoes full lines for ws clients)

- [ ] **Step 1: Skip echo when the input line targets a handle**

The web client doesn't echo locally; `EchoInputHandler` echoes input back. For a command whose argument is a `@<handle>` token, suppress the echo so only the result shows. In `EchoInputHandler` (confirm this is the ws echo point; the login handler notes ws clients skip per-char echo — verify the full-line path):
```go
func EchoInputHandler(clientInput *connections.ClientInput, sharedState map[string]any) (nextHandler bool) {
	if len(clientInput.DataIn) > 0 {
		// Suppress echo for handle-targeted (panel-fired) commands so the raw
		// @<handle> never appears in the feed (silent — result only).
		if !containsItemHandle(string(clientInput.DataIn)) {
			connections.SendTo(clientInput.DataIn, clientInput.ConnectionId)
		}
	}
	if !clientInput.EnterPressed { return false }
	connections.SendTo(term.CRLF, clientInput.ConnectionId)
	return true
}
```
where `containsItemHandle` checks for the `@`-sigil token (a simple `strings.Contains(line, " @") || strings.HasPrefix(line, "@")`-style match scoped to a handle-looking token). Keep it conservative so normal input is never suppressed. **If `EchoInputHandler` is per-character for ws (so `DataIn` isn't a full line), instead suppress at the point where the full command line is echoed for ws clients — codegraph: trace the ws input pipeline from `ClientInput` to the command processor.**

- [ ] **Step 2: Build + boot-confirm (human)**

`go build ./...`. Human boot test: fire a handle command (after Task 8 the panel does this; pre-Task-8, test by typing `look @<a real uuid>` — only the result shows, no echoed `look @...`). Commit:
```bash
git add internal/inputhandlers/echo.go
git commit -m "feat(input): suppress echo for @<handle>-targeted commands"
```

---

### Task 4: Extend `Char.Inventory` — dynamic Worn (eq order) + Bandolier + Component Bag

**Files:**
- Modify: `modules/gmcp/gmcp.Char.go` (inventory payload structs + the builder at ~377-410)

- [ ] **Step 1: New payload shapes**

Replace the fixed `GMCPCharModule_Payload_Inventory_Worn` struct with a dynamic slot list, and add bandolier + component bag:
```go
type GMCPCharModule_Payload_Inventory struct {
	Worn         []GMCPCharModule_Payload_Slot              `json:"Worn"`
	Bandolier    *GMCPCharModule_Payload_Container          `json:"Bandolier,omitempty"`
	ComponentBag *GMCPCharModule_Payload_Container          `json:"ComponentBag,omitempty"`
	Backpack     *GMCPCharModule_Payload_Inventory_Backpack `json:"Backpack,omitempty"`
}
type GMCPCharModule_Payload_Slot struct {
	Slot    string                                 `json:"slot"`     // display label, e.g. "Weapon", "Arm 3"
	SlotKey string                                 `json:"slotKey"`  // "weapon","offhand","extraarm1",...
	Item    *GMCPCharModule_Payload_Inventory_Item `json:"item"`     // null = empty slot
}
type GMCPCharModule_Payload_Container struct {
	Items   []GMCPCharModule_Payload_Inventory_Item `json:"items,omitempty"`
	Summary GMCPCharModule_Payload_Inventory_Backpack_Summary `json:"Summary,omitempty"`
}
```

- [ ] **Step 2: Build the dynamic Worn list in `eq` order**

In the inventory builder, emit slots in **`worn.go` order**: Weapon, Offhand, ExtraArm1-4, Head, Neck, Shoulders, Body, Back, Belt, Wrist1-2, ExtraWrist1-4, Gloves, Ring, Ring2, Legs, Feet, Tail, ComponentBag(slot). Include a base slot always (with `item:null` if empty); include a **mutation slot only when the character actually has it** — codegraph/confirm the predicate (e.g. Extra-Arms mutation level → which ExtraArm/ExtraWrist slots exist; Tail mutation → Tail slot). Mirror however the `eq`/`equip` command decides which slots to show (search `GetAllWornItems`/the equip listing for the existing "does this slot exist" logic and reuse it). Each filled slot's `Item` = `newInventory_Item(equipItem)`; empty = `nil`. Provide display labels (e.g. `ExtraArm1` → "Arm 3", `Wrist1` → "Wrist").

- [ ] **Step 3: Bandolier + component bag contents**

- Bandolier: from `c.PotionItems` (present when a bandolier is worn — gate on the bandolier being equipped; confirm the predicate). Map each to `newInventory_Item`.
- Component bag: **confirm where component-bag contents are stored** (codegraph: `is_component` routing in `inventory.go:150`, the `StoreItem`/component path — the contents may live in a dedicated slice or be filtered from backpack). Emit them as a `Container`. If the contents accessor doesn't exist, add a small `Character` helper `GetComponentBagContents() []items.Item` and use it here AND in Task 2's `FindItem` handle branch.
- Set each container's Summary count/max from the bag's capacity.

- [ ] **Step 4: Update the wantsGMCPPayload gating**

The builder gates sections on identifiers (`Char.Inventory.Backpack`, etc.). Add gating for `Char.Inventory.Worn` / `.Bandolier` / `.ComponentBag` consistent with the existing pattern, and ensure equip/ownership-change handlers request the right identifiers so the panel updates on equip/remove/get/drop.

- [ ] **Step 5: Build + boot + commit**

`go build ./...`; boot the server and confirm no panic + (optionally) inspect a `Char.Inventory` payload. Commit:
```bash
git add modules/gmcp/gmcp.Char.go internal/characters/  # if a helper was added
git commit -m "feat(gmcp): dynamic Worn (eq order) + bandolier + component bag in Char.Inventory"
```

---

### Task 5: Conditions-freshness fix (push `Char.Conditions` on expiry)

**Files:**
- Modify: `modules/gmcp/gmcp.Char.go` and/or the buff-removal path (`internal/characters/buffs.go`)

- [ ] **Step 1: Emit a Char.Conditions update when conditions change/expire**

Today GMCP only refreshes conditions on `BuffsTriggered` (add). Make buff **removal/expiry** also queue a `Char.Conditions` update. Options (pick the cleanest after codegraph):
- If a `BuffsRemoved`/expiry event exists, register a listener in `gmcp.Char.go` (like the `BuffsTriggered` one) that queues `GMCPCharUpdate{UserId, Identifier: "Char.Conditions"}`.
- Else, in `Character.RemoveBuff` (`buffs.go:135`) emit such an event / queue the update.
- Else, on the round-tick that ages buffs, if the condition set changed since last sent, queue the update.

Keep it minimal — no payload-shape change; just close the freshness gap.

- [ ] **Step 2: Build + commit**

`go build ./...`. (Human can later confirm the Status panel updates promptly on a buff expiring.)
```bash
git add modules/gmcp/gmcp.Char.go internal/characters/buffs.go
git commit -m "fix(gmcp): push Char.Conditions on buff removal/expiry (panel freshness)"
```

---

### Task 6: Type-icon set (client asset)

**Files:**
- Create: `_datafiles/html/public/static/js/item-icons.js` (an `ITEM_ICONS` map of type/subtype → inline SVG string) + license/attribution note

- [ ] **Step 1: Curate a small themeable SVG icon set**

Author or source (permissively licensed, e.g. game-icons.net CC-BY — include attribution) ~16-24 monochrome SVG icons keyed by item `type` and key `subtype`s: weapon (sword/dagger/axe/wand/sceptre/staff), offhand/shield, head, body, legs, feet, gloves, back, belt, shoulders, neck/amulet, ring, potion/drink, food, component, bag, plus a generic fallback. Expose:
```js
window.ITEM_ICONS = { /* "weapon-sword": "<svg…>", "potion": "<svg…>", … */ };
window.itemIconSVG = function (type, subtype) { /* returns best-match SVG string, fallback generic */ };
```
Icons use `currentColor`/`stroke` so the tile CSS tints them to the theme.

- [ ] **Step 2: Link it + commit**

Add `<script src="…/static/js/item-icons.js"></script>` to `webclient-pure.html` (before `dashboard.js`/the panel code).
```bash
git add _datafiles/html/public/static/js/item-icons.js _datafiles/html/public/webclient-pure.html
git commit -m "feat(webclient): themeable type-icon set for inventory tiles"
```

---

### Task 7: Client — Inventory panel structure + render (replace Scene slot)

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html` (the `#panel-art` Scene slot → Inventory panel + render JS + GMCP handler), `_datafiles/html/public/static/css/dashboard.css`

- [ ] **Step 1: Re-home the panel**

Replace `#panel-art`'s Scene content with the Inventory panel markup: keep `id="panel-art" data-panel="art"` (so the dashboard framework's slot/pop-out/persistence keep working — it just renders inventory now; remove the `dash-slot` class), retitle the header "Inventory", and give the body a tab strip + a `#inv-body` grid container:
```html
<div class="dash-panel-head"><span class="ph-title">Inventory</span>
  <span class="ph-btns"><span class="ph-btn ph-collapse">▾</span><span class="ph-btn ph-popout">⧉</span></span></div>
<div class="dash-panel-body">
  <div id="inv-tabs" class="inv-tabs">
    <button class="inv-tab brass active" data-tab="worn">Equipped</button>
    <button class="inv-tab brass" data-tab="bandolier">Bandolier</button>
    <button class="inv-tab brass" data-tab="components">Components</button>
    <button class="inv-tab brass" data-tab="backpack">Backpack</button>
  </div>
  <div id="inv-grid" class="inv-grid"></div>
</div>
```

- [ ] **Step 2: CSS (dashboard.css)**

Style `.inv-tabs` (brass tab strip), `.inv-tab.active` (pressed), `.inv-grid` (5-col grid), `.inv-cell`/`.inv-tile` (brass-framed leather tile, type-icon centered, `aspect-ratio:1`), `.inv-tile.empty` (dashed + faint slot label), `.inv-slab` (slot label), `.inv-qty` (`xN` stack badge), mutation-slot label tint (`--copper`/green). Reuse the mockup's styling (`inventory-tabbed-v2.html`).

- [ ] **Step 3: Render from GMCP + tab switching**

Add a `renderInventory()` that reads `GMCPStructs["Char"].Inventory` and draws the active tab into `#inv-grid`:
- `worn`: iterate `Inventory.Worn` (slot list) → a cell per slot: tile with `itemIconSVG(item.type, item.subtype)` if `item`, else `.empty` with the slot label; always show the `.inv-slab` slot label; mutation slots get the tinted label.
- `bandolier`/`components`/`backpack`: iterate the container `items[]` → icon tiles; collapse duplicate `(itemId+enchant)` into `xN` if desired.
- Store each tile's `data-handle`, `data-name`, `data-type`, `data-subtype`, and which collection/slot it's in (for Task 8's menu).
- Tab buttons switch the active tab (`data-tab`) and re-render. Default `worn`.
- Wire a `"Char.Inventory"` (and `"Char"`) GMCP update handler to call `renderInventory()` so the panel refreshes on changes. Build all DOM via `createElement`/`textContent` (+ icon SVG via a trusted local map) — never `innerHTML` with server strings.

- [ ] **Step 4: Verify + commit**

Human smoke: `/webclient` shows the Inventory panel in the old Scene slot; Equipped tab lists slots in `eq` order with icons; other tabs show their contents; equipping/dropping updates it. Structural self-review: no `innerHTML` with server data.
```bash
git add _datafiles/html/public/webclient-pure.html _datafiles/html/public/static/css/dashboard.css
git commit -m "feat(webclient): tabbed inventory panel rendered from GMCP (replaces scene slot)"
```

---

### Task 8: Client — right-click action menu firing handle commands

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html`, `_datafiles/html/public/static/css/dashboard.css`

- [ ] **Step 1: Item-type-aware context menu**

On `contextmenu` (right-click) of a tile, build a small floating menu near the cursor with actions derived from the tile's `data-type`/`data-subtype` + collection:
- Always: **Look** (`look @<handle>`), **Identify** (the id-spell invocation — confirm exact command, e.g. `cast identify @<handle>` vs `identify @<handle>`), **Drop** (`drop @<handle>`).
- Worn item: **Remove** (`remove @<handle>`).
- Backpack wearable (`type` weapon/offhand/wearable armor): **Wear**/**Wield** (`wield @<handle>` for weapons, `wear @<handle>` otherwise).
- Consumable (potion/food/drink): **Drink**/**Eat**/**Use** as appropriate.
Each menu item → `SendData(verb + " @" + handle)` (silent; Task 3 suppresses echo) then close the menu. Close on outside-click / Escape.

- [ ] **Step 2: CSS for `.inv-ctx` menu** (leather/brass dropdown, matching the mockup).

- [ ] **Step 3: Verify + commit**

Human smoke: right-click items in each tab → correct, type-appropriate actions; firing one targets the exact item (test with duplicate names) and shows only the result in the feed (no `@<handle>` echo); Remove works on equipped, Wear/Wield/Drink on backpack.
```bash
git add _datafiles/html/public/webclient-pure.html _datafiles/html/public/static/css/dashboard.css
git commit -m "feat(webclient): inventory action menu (handle-targeted, item-type-aware)"
```

---

### Task 9: Full smoke + final review

**Files:** None (verification + small fixes only)

- [ ] **Step 1: End-to-end human smoke** — `/webclient`: all four tabs; mutation slots appear when you have the mutation; equip/remove/get/drop/use update the panel promptly; actions fire correctly & silently; duplicate-name targeting is exact; Status panel now updates on buff expiry; pop-out/collapse/responsive still work for the panel.
- [ ] **Step 2: Server checks** — `go build ./...` clean; `go test ./internal/characters/ ./modules/gmcp/...`; boot the server clean (no GMCP/data-load panic).
- [ ] **Step 3: Leftover sweep** — grep `webclient-pure.html` for any dead Scene-art markup; confirm no `innerHTML` with server-derived inventory strings.
- [ ] **Step 4: Final whole-branch code review** (dispatch a reviewer), then **superpowers:finishing-a-development-branch** to merge to `master` (`--no-ff`, parked locally).

---

## Self-review

- **Spec coverage:** handle (T1) + finder resolution (T2) + silent echo (T3) + dynamic Worn/bandolier/component-bag GMCP (T4) + conditions freshness (T5) + type icons (T6) + panel render (T7) + action menu (T8) + smoke (T9) — all spec parts covered.
- **Placeholders:** none unresolved; the deliberately-flagged "confirm via codegraph" items (handle sigil collision, the ws echo-suppression point, the mutation-slot-exists predicate, component-bag contents accessor, the identify invocation) are real integration unknowns the implementer must verify against live code — each names the exact symbol/file to check rather than hand-waving.
- **Type/name consistency:** the `@`+UUID handle, `GMCPStructs["Char"].Inventory.{Worn,Bandolier,ComponentBag,Backpack}`, `item.handle/name/type/subtype`, `#panel-art` reuse, and the tab keys (`worn/bandolier/components/backpack`) are used consistently across tasks.
- **Ordering:** server data/targeting (T1-T5) lands before the client consumes it (T6-T8), so each client task can be smoke-tested against real GMCP.
