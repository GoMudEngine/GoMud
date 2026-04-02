# Quest Port Batch 4 (Final) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire up `room_interact` events in the quest engine, port Quests 9, 10, and 14 to quest engine triggers, and convert Quest 17 to pure lore discovery.

**Architecture:** First wire `room_interact` into `TryRoomScripts` so the quest engine fires before JS scripts. Then port each quest with triggers replacing JS. Q14's dungeon uses `room_enter`, `item_gain`, `room_interact` (strongbox), and `item_give` triggers.

**Tech Stack:** Go (room_interact wiring), YAML data files, existing quest engine infrastructure.

---

### Task 1: Wire Up room_interact Event

**Files:**
- Modify: `internal/usercommands/usercommands.go`

- [ ] **Step 1: Add room_interact notification to TryRoomScripts**

Read `internal/usercommands/usercommands.go`. Find the `TryRoomScripts` function (around line 504). Add a quest engine notification BEFORE the JS room script call. The function receives `alias` (command name like "push", "open", "get") and `rest` (the target like "stone", "strongbox", "ledger").

Fire the event with the command as `Verb` and the target as `Noun`:

```go
func TryRoomScripts(input, alias, rest string, userId int) (bool, error) {

	// Quest engine: room_interact notification
	// Fires before JS scripts so triggers can replace onCommand handlers.
	user := users.GetByUserId(userId)
	if user != nil {
		room := rooms.LoadRoom(user.Character.RoomId)
		if room != nil {
			bridge := questengine.NewGameBridge(user, room.RoomId)
			result := questengine.GetEngine().Notify("room_interact", questengine.EventDetails{
				UserId: user.UserId,
				RoomId: room.RoomId,
				Verb:   alias,
				Noun:   strings.ToLower(rest),
			}, bridge, bridge)
			if result.Handled {
				return true, nil
			}
		}
	}

	// Try direct command room script first
	cmdHandled, err := scripting.TryRoomCommand(alias, rest, userId)
	// ... rest of function unchanged
```

Add imports for `questengine`, `rooms`, and `strings` if not already present. The function already imports `users` (used later in the GMCP block).

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/usercommands.go
git commit -m "feat: wire up room_interact event in TryRoomScripts

Quest engine room_interact notification fires before JS room
scripts. Passes the command as Verb and target text as Noun.
If the quest engine handles it, JS scripts are skipped.
Enables room_interact triggers for strongbox, push stone, etc."
```

---

### Task 2: Container Conversions + Quest Item Checks

**Files:**
- Modify: `_datafiles/world/dogmud/rooms/thornwall_city/476.yaml`
- Modify: `_datafiles/world/dogmud/rooms/thornwall_city/492.yaml` (if exists, or find correct path)

- [ ] **Step 1: Add spawninfo for tithe ledger in room 476**

Read `_datafiles/world/dogmud/rooms/thornwall_city/476.yaml`. The tithe ledger (item 29) is currently only placed by the room script. Add spawninfo so it spawns on the floor:

```yaml
- itemid: 29
  respawnrate: 5 real minutes
```

- [ ] **Step 2: Convert tally stick in room 492**

Find the stash room (492) in `_datafiles/world/dogmud/rooms/thornwall_city/`. Read it. If item 40035 is in a static `containers:` section, empty the container and add spawninfo:

```yaml
- itemid: 40035
  container: crates
  respawnrate: 5 real minutes
```

- [ ] **Step 3: Delete stale room instance saves**

```bash
rm _datafiles/world/dogmud/rooms.instances/thornwall_city/476.yaml 2>/dev/null
rm _datafiles/world/dogmud/rooms.instances/thornwall_city/492.yaml 2>/dev/null
```

- [ ] **Step 4: Verify quest items not components**

Check items 29, 30, 40034, 40035, 40036, 40038 for `is_component`. Should not be set.

- [ ] **Step 5: Commit**

```bash
git add -A _datafiles/world/dogmud/rooms/thornwall_city/
git commit -m "fix: convert Q9/Q14 quest items to spawninfo respawn"
```

---

### Task 3: Port Quest 9 — The Temple's Tithe Audit

**Files:**
- Modify: `_datafiles/world/dogmud/quests/9-the_temples_tithe_audit.yaml`
- Modify: `_datafiles/world/dogmud/dialogue/thornwall_city/95.yaml`
- Disable: `_datafiles/world/dogmud/rooms/thornwall_city/476.js`
- Disable: `_datafiles/world/dogmud/mobs/thornwall_city/scripts/95-temple_priest_olen.js`

- [ ] **Step 1: Update Quest 9 YAML — collapse to [start, end], add triggers**

Collapse from [start, investigate, evidence, end] to [start, end]. Add item_give trigger on Olen (mob 95) for ledger (item 29). Add hints.

- [ ] **Step 2: Update Olen dialogue — fix dead step refs**

Update any `questRequired`/`questExcluded` references to dead steps (9-investigate, 9-evidence) to use 9-start/9-end. Verify SOP.

- [ ] **Step 3: Disable scripts**

```bash
mv _datafiles/world/dogmud/rooms/thornwall_city/476.js _datafiles/world/dogmud/rooms/thornwall_city/476.js.bak
mv _datafiles/world/dogmud/mobs/thornwall_city/scripts/95-temple_priest_olen.js _datafiles/world/dogmud/mobs/thornwall_city/scripts/95-temple_priest_olen.js.bak
```

- [ ] **Step 4: Build and commit**

```bash
go build ./...
git add -A _datafiles/world/dogmud/quests/9-* _datafiles/world/dogmud/dialogue/thornwall_city/95.yaml _datafiles/world/dogmud/rooms/thornwall_city/ _datafiles/world/dogmud/mobs/thornwall_city/scripts/95-*
git commit -m "feat: port Quest 9 (Tithe Audit) — ledger delivery via item_give trigger"
```

---

### Task 4: Port Quest 10 — The Drowning Post's Debt

**Files:**
- Modify: `_datafiles/world/dogmud/quests/10-the_drowning_posts_debt.yaml`
- Modify: `_datafiles/world/dogmud/dialogue/thornwall_city/96.yaml`
- Verify: `_datafiles/world/dogmud/dialogue/thornwall_city/94.yaml` (Velk)

- [ ] **Step 1: Update Quest 10 YAML — collapse to [start, report, end], add triggers**

Keep 3 steps: start (Marek gives notice), report (deliver to Velk), end (return to Marek). Add item_give trigger on Velk (mob 94) for notice (item 30). Add hints.

Note: Do NOT disable Velk's mob script yet — it also handles Q14. It will be disabled in Task 5 after Q14 triggers are in place.

- [ ] **Step 2: Update Marek dialogue — fix dead step refs**

Update any references to dead steps (10-investigate) to use the new step names. Ensure Marek's completion node checks `questRequired: ["10-report"]` and grants `10-end`. Verify SOP.

- [ ] **Step 3: Build and commit**

```bash
go build ./...
git add -A _datafiles/world/dogmud/quests/10-* _datafiles/world/dogmud/dialogue/thornwall_city/96.yaml _datafiles/world/dogmud/dialogue/thornwall_city/94.yaml
git commit -m "feat: port Quest 10 (Drowning Post) — notice delivery via item_give trigger"
```

---

### Task 5: Port Quest 14 — The Undertow

**Files:**
- Modify: `_datafiles/world/dogmud/quests/14-the_undertow.yaml`
- Verify: `_datafiles/world/dogmud/dialogue/thornwall_city/96.yaml` (Marek)
- Verify: `_datafiles/world/dogmud/dialogue/thornwall_city/94.yaml` (Velk)
- Verify: `_datafiles/world/dogmud/dialogue/thornwall_city/249.yaml` (Torvan)
- Disable: `_datafiles/world/dogmud/rooms/thornwall_city/485.js`
- Disable: `_datafiles/world/dogmud/rooms/thornwall_city/498.js`
- Disable: `_datafiles/world/dogmud/mobs/thornwall_city/scripts/94-guard_captain_velk.js`

- [ ] **Step 1: Update Quest 14 YAML — add all triggers**

Keep all 6 steps [start, explore, confront, evidence, report, end]. Add triggers:

1. `item_gain` on item 40035 → grants `14-explore`
2. `room_enter` on room 498 → grants `14-confront`
3. `room_interact` on room 498, noun "strongbox", verb "open"/"unlock"/"get"/"search"/"use" → with key: consume 40034, give 40036, grant `14-evidence`
4. `room_interact` on room 498, noun "strongbox" → without key: send "locked" text
5. `item_give` on Velk (mob 94) for item 40036 → grants `14-report`
6. `quest_granted` on `14-report` → auto-grants `14-end`

Also add a `room_enter` trigger on room 486 (first tunnel room) that sends a blocking message and teleports back to room 485 if the player doesn't have `14-start`. This replaces the cellar gate JS script.

Add hints for all 6 steps with directions through the dungeon.

NOTE on room_interact noun matching: The quest engine matches `Noun` field exactly. The `Noun` value from `TryRoomScripts` is `strings.ToLower(rest)`. For "open strongbox", `rest` = "strongbox". For "open strong box", `rest` = "strong box". Use the most common form. Multiple triggers can cover variants if needed.

- [ ] **Step 2: Verify dialogue SOP for Marek, Velk, Torvan**

Marek grants `14-start` + gives lantern (40038) via `givesItem`. Verify quest/task triggers. Velk's dialogue for Q14 handles `14-report`/`14-end` — verify it works with the new trigger-based flow (trigger grants 14-report, quest_granted chain grants 14-end, Velk's dialogue should show post-completion text). Torvan's dialogue is pre-combat flavor — verify it doesn't conflict.

- [ ] **Step 3: Disable scripts**

```bash
mv _datafiles/world/dogmud/rooms/thornwall_city/485.js _datafiles/world/dogmud/rooms/thornwall_city/485.js.bak
mv _datafiles/world/dogmud/rooms/thornwall_city/498.js _datafiles/world/dogmud/rooms/thornwall_city/498.js.bak
mv _datafiles/world/dogmud/mobs/thornwall_city/scripts/94-guard_captain_velk.js _datafiles/world/dogmud/mobs/thornwall_city/scripts/94-guard_captain_velk.js.bak
```

- [ ] **Step 4: Delete stale mob instance saves for Velk**

```bash
rm _datafiles/world/dogmud/mobs.instances/thornwall_city/94-guard_captain_velk-room473.yaml 2>/dev/null
```

- [ ] **Step 5: Build and commit**

```bash
go build ./...
git add -A _datafiles/world/dogmud/quests/14-* _datafiles/world/dogmud/rooms/thornwall_city/ _datafiles/world/dogmud/mobs/thornwall_city/scripts/94-* _datafiles/world/dogmud/dialogue/thornwall_city/
git commit -m "feat: port Quest 14 (The Undertow) — dungeon crawl with room_interact strongbox

6-step dungeon quest using item_gain (tally stick), room_enter
(operations room), room_interact (strongbox key/open), item_give
(ledger to Velk). Cellar gate via room_enter teleport-back.
Disabled JS room scripts (485, 498) and Velk's mob script."
```

---

### Task 6: Convert Quest 17 to Lore Discovery

**Files:**
- Modify or delete: `_datafiles/world/dogmud/quests/17-the_empty_cottage.yaml`
- Modify: `_datafiles/world/dogmud/rooms/ashwick/4023.js` → convert to room_interact triggers OR keep as lore-only JS

- [ ] **Step 1: Remove Quest 17 from quest system**

Either delete the quest YAML or set `secret: true` with no steps and no rewards. The room interactions should just give items as lore objects without granting quest tokens.

If using room_interact triggers (recommended): add triggers to a "lore" quest YAML that just gives items with flavor text but is marked secret:

```yaml
questid: 17
name: The Empty Cottage
description: Remnants of a life left behind in a cottage in Ashwick.
secret: true
steps: []
```

Actually, without steps, the quest engine can't grant tokens. Simplest: just delete the quest YAML and convert the room script to room_interact triggers that give items + send flavor text with no quest involvement.

- [ ] **Step 2: Add room_interact triggers for lore items**

Since Q17 has no quest, these can't be quest engine triggers (no quest to attach them to). Options:
- Keep the JS room script for flavor-only interactions
- Or add a lore quest with secret: true and minimal steps

Simplest: keep the room script `4023.js` as-is but remove the `GiveQuest` calls. The script already handles push stone and drawer interactions — just strip the quest token grants.

Edit `_datafiles/world/dogmud/rooms/ashwick/4023.js`: remove all `user.GiveQuest("17-...")` lines. Keep the `user.GiveItem()` and `user.SendText()` calls for lore.

- [ ] **Step 3: Delete or mark quest YAML as inactive**

Delete `_datafiles/world/dogmud/quests/17-the_empty_cottage.yaml`.

- [ ] **Step 4: Build and commit**

```bash
go build ./...
git add -A _datafiles/world/dogmud/quests/17-* _datafiles/world/dogmud/rooms/ashwick/4023.js
git commit -m "refactor: convert Quest 17 from quest to lore discovery

Removed quest YAML. Kept room script for flavor interactions
(push stone → letter, open drawer → recipe) but stripped quest
token grants. Items remain as lore objects."
```

---

### Task 7: Manual Test — All Quests

- [ ] **Step 1: Test room_interact wiring**

Go to a room with a container. Type `open <container>` or `push <noun>`. Verify no crashes. (Functional test comes with Q14 strongbox.)

- [ ] **Step 2: Test Quest 9**

Ask Olen → get ledger from room 476 → `give ledger olen` → verify rewards fire (gold).

- [ ] **Step 3: Test Quest 10**

Ask Marek → receive notice → `give notice velk` → return to Marek → verify end.

- [ ] **Step 4: Test Quest 14 — full dungeon crawl**

Requires Q10 completed. Ask Marek about the cellar → receive lantern → descend cellar (room 485) → navigate tunnels → pick up tally stick (room 492) → verify 14-explore → enter operations room (498) → verify 14-confront → defeat Torvan → get key → `open strongbox` → verify key consumed, ledger received, 14-evidence → `give ledger velk` → verify 14-report + 14-end + rewards.

- [ ] **Step 5: Test Quest 14 — cellar gate**

Without Q14: try to go down from room 485 → should be blocked.

- [ ] **Step 6: Test Quest 14 — strongbox without key**

Enter room 498 with 14-confront but no key → `open strongbox` → should see "locked" message.

- [ ] **Step 7: Test Quest 17 — lore only**

Visit cottage → push stone → verify letter received, no quest banner → open drawer → verify recipe received, no quest banner.

- [ ] **Step 8: Respawn test**

Take tithe ledger (room 476) and tally stick (room 492). Wait 5 minutes. Return — verify they reappeared.
