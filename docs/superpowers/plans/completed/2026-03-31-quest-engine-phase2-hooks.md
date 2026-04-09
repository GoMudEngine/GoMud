# Quest Engine Phase 2: Hook Integration

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the Phase 1 quest engine into the live game by implementing the ActionContext/PlayerState bridges and adding Notify() calls at all hook points, then rework give.go for safe item delivery.

**Architecture:** The quest engine (`internal/questengine/`) already has the core loop, types, conditions, actions, guards, and logging. This phase adds a concrete `GameBridge` that implements `ActionContext` and `PlayerState` using real game objects (`users.UserRecord`, `rooms.Room`, `characters.Character`). Then each hook point (go.go, give.go, progression.go, combat helpers, ask.go, CheckItemQuests) gets a `questengine.Notify()` call. The `give.go` rework is the critical safety change — the quest engine intercepts item delivery BEFORE transfer.

**Tech Stack:** Go, YAML quest definitions, existing event system

---

## File Structure

| File | Responsibility |
|------|---------------|
| **Create:** `internal/questengine/bridge.go` | `GameBridge` struct implementing `ActionContext` + `PlayerState` using real game objects |
| **Create:** `internal/questengine/bridge_test.go` | Tests for bridge method dispatch (mock-free where possible, mock user/room where needed) |
| **Modify:** `internal/usercommands/give.go` | Intercept item_give BEFORE transfer, check `NotifyResult` |
| **Modify:** `internal/usercommands/go.go` | Add `room_enter` notification after successful room change |
| **Modify:** `internal/characters/progression.go` | Add `skill_use` notification in `OnSkillUse()` |
| **Modify:** `internal/hooks/ItemOwnership_CheckItemQuests.go` | Add `item_gain` notification |
| **Modify:** `internal/hooks/MobDeath_PackFlee.go` | Add `mob_death` notification |
| **Modify:** `internal/usercommands/ask.go` | Add `dialogue` notification |
| **Modify:** `internal/usercommands/buy.go` | Add `command` notification for `buy` |
| **Modify:** `internal/usercommands/equip.go` | Add `command` notification for `equip` |
| **Modify:** `internal/usercommands/track.go` | Add `command` notification for `track` |

---

### Task 1: GameBridge — PlayerState Interface

Implement the `PlayerState` half of the bridge. This wraps a `users.UserRecord` to satisfy the quest engine's condition-checking interface.

**Files:**
- Create: `internal/questengine/bridge.go`
- Create: `internal/questengine/bridge_test.go`

- [ ] **Step 1: Write the failing test for PlayerState methods**

In `bridge_test.go`:

```go
package questengine

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

func TestGameBridge_HasQuest(t *testing.T) {
	user := &users.UserRecord{
		UserId:    1,
		Character: &characters.Character{},
	}
	user.Character.GiveQuestToken("3-start")

	bridge := NewGameBridge(user, 100)

	assert.True(t, bridge.HasQuest("3-start"))
	assert.False(t, bridge.HasQuest("3-end"))
}

func TestGameBridge_HasItem(t *testing.T) {
	user := &users.UserRecord{
		UserId:    1,
		Character: &characters.Character{},
	}
	itm := items.Item{ItemId: 14}
	user.Character.StoreItem(itm)

	bridge := NewGameBridge(user, 100)

	assert.True(t, bridge.HasItem(14))
	assert.False(t, bridge.HasItem(999))
}

func TestGameBridge_GetRoomId(t *testing.T) {
	user := &users.UserRecord{
		UserId:    1,
		Character: &characters.Character{RoomId: 113},
	}
	bridge := NewGameBridge(user, 113)

	assert.Equal(t, 113, bridge.GetRoomId())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/questengine/ -run TestGameBridge -v`
Expected: FAIL — `NewGameBridge` undefined

- [ ] **Step 3: Write minimal implementation**

In `bridge.go`:

```go
package questengine

import (
	"github.com/GoMudEngine/GoMud/internal/users"
)

// GameBridge implements PlayerState and ActionContext using real game objects.
type GameBridge struct {
	user   *users.UserRecord
	roomId int // room where the event happened
}

// NewGameBridge creates a bridge for quest engine integration.
// roomId is passed explicitly because some events fire after room changes.
func NewGameBridge(user *users.UserRecord, roomId int) *GameBridge {
	return &GameBridge{user: user, roomId: roomId}
}

// ── PlayerState interface ──────────────────────────────────────────────

func (b *GameBridge) HasQuest(token string) bool {
	return b.user.Character.HasQuest(token)
}

func (b *GameBridge) HasItem(itemId int) bool {
	for _, itm := range b.user.Character.Items {
		if itm.ItemId == itemId {
			return true
		}
	}
	return false
}

func (b *GameBridge) GetRoomId() int {
	return b.roomId
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/questengine/ -run TestGameBridge -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/questengine/bridge.go internal/questengine/bridge_test.go
git commit -m "feat: quest engine GameBridge PlayerState implementation"
```

---

### Task 2: GameBridge — ActionContext Interface

Implement every `ActionContext` method on `GameBridge`. Each method delegates to the real game API.

**Files:**
- Modify: `internal/questengine/bridge.go`
- Modify: `internal/questengine/bridge_test.go`

- [ ] **Step 1: Write the failing test for GetUserId**

Append to `bridge_test.go`:

```go
func TestGameBridge_GetUserId(t *testing.T) {
	user := &users.UserRecord{
		UserId:    42,
		Character: &characters.Character{},
	}
	bridge := NewGameBridge(user, 100)
	assert.Equal(t, 42, bridge.GetUserId())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/questengine/ -run TestGameBridge_GetUserId -v`
Expected: FAIL — `GetUserId` not defined on `*GameBridge`

- [ ] **Step 3: Write full ActionContext implementation**

Append to `bridge.go`:

```go
import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ── ActionContext interface ────────────────────────────────────────────

func (b *GameBridge) GetUserId() int {
	return b.user.UserId
}

func (b *GameBridge) GrantQuest(token string) {
	b.user.Character.GiveQuestToken(token)
}

func (b *GameBridge) ConsumeItem(itemId int) {
	for _, itm := range b.user.Character.Items {
		if itm.ItemId == itemId {
			b.user.Character.RemoveItem(itm)
			return
		}
	}
}

func (b *GameBridge) GiveItem(itemId int) {
	newItem := items.New(itemId)
	if newItem.ItemId == 0 {
		mudlog.Error("QuestBridge", "action", "give_item", "error", fmt.Sprintf("item %d not found", itemId))
		return
	}
	b.user.Character.StoreItem(newItem)
	b.user.SendText(fmt.Sprintf("You receive a <ansi fg=\"item\">%s</ansi>.", newItem.DisplayName()))
}

func (b *GameBridge) GiveGold(amount int) {
	b.user.Character.Gold += amount
	b.user.SendText(fmt.Sprintf("You receive <ansi fg=\"gold\">%d gold</ansi>.", amount))
	events.AddToQueue(events.EquipmentChange{
		UserId:     b.user.UserId,
		GoldChange: amount,
	})
}

func (b *GameBridge) SendText(text string) {
	b.user.SendText(text)
}

func (b *GameBridge) RoomText(text string) {
	if room := rooms.LoadRoom(b.roomId); room != nil {
		room.SendText(text, b.user.UserId)
	}
}

func (b *GameBridge) SpawnMob(s SpawnDef) {
	mob := mobs.NewMobById(mobs.MobId(s.Id), s.Room)
	if mob == nil {
		mudlog.Error("QuestBridge", "action", "spawn_mob", "error", fmt.Sprintf("mob %d not found", s.Id))
		return
	}
	if room := rooms.LoadRoom(s.Room); room != nil {
		room.AddMob(mob.InstanceId)
	}
}

func (b *GameBridge) SpawnItem(s SpawnDef) {
	newItem := items.New(s.Id)
	if newItem.ItemId == 0 {
		mudlog.Error("QuestBridge", "action", "spawn_item", "error", fmt.Sprintf("item %d not found", s.Id))
		return
	}
	if room := rooms.LoadRoom(s.Room); room != nil {
		room.AddItem(newItem, false)
	}
}

func (b *GameBridge) TeachSpell(spellId string) {
	if b.user.Character.LearnSpell(spellId) {
		b.user.SendText(fmt.Sprintf("You learn the spell <ansi fg=\"spellname\">%s</ansi>!", spellId))
	}
}

func (b *GameBridge) TrainSkill(skill string, level int) {
	b.user.Character.SetSkill(skill, level)
}

func (b *GameBridge) ApplyBuff(bf BuffDef) {
	b.user.Character.AddBuff(bf.Buff, false)
}

func (b *GameBridge) Teleport(roomId int) {
	rooms.MoveToRoom(b.user.UserId, roomId)
}

func (b *GameBridge) LockExits(e ExitLock) {
	// Player-scoped exit locks are a future feature — for now, lock the room exits globally
	if room := rooms.LoadRoom(e.Room); room != nil {
		for exitName := range room.Exits {
			room.SetExitLock(exitName, true)
		}
	}
}

func (b *GameBridge) UnlockExits(e ExitLock) {
	if room := rooms.LoadRoom(e.Room); room != nil {
		for exitName := range room.Exits {
			room.SetExitLock(exitName, false)
		}
	}
}

func (b *GameBridge) QueueNpcSay(n NpcSayDef) {
	mob := findMobInRoom(n.Mob, b.roomId)
	if mob == nil {
		mudlog.Error("QuestBridge", "action", "npc_say", "error",
			fmt.Sprintf("mob %d not found in room %d", n.Mob, b.roomId))
		return
	}
	for _, line := range n.Lines {
		mob.Command(fmt.Sprintf("say %s", line.Text))
	}
}

func (b *GameBridge) QueueSequence(s SequenceDef) {
	// Phase 2 simple implementation: execute lines immediately, queue on_complete
	for _, line := range s.Lines {
		if line.Speaker > 0 {
			if mob := findMobInRoom(line.Speaker, b.roomId); mob != nil {
				mob.Command(fmt.Sprintf("say %s", line.Text))
			}
		} else {
			b.user.SendText(line.Text)
		}
	}
	// Execute on_complete actions synchronously
	for _, action := range s.OnComplete {
		ExecuteAction(action, b)
	}
}

// findMobInRoom searches for a mob with the given spec ID in the given room.
func findMobInRoom(mobId int, roomId int) *mobs.Mob {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return nil
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		mob := mobs.GetInstance(instId)
		if mob != nil && int(mob.MobId) == mobId {
			return mob
		}
	}
	return nil
}
```

Note: Merge the import blocks — the final file should have one `import` block with all needed packages.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/questengine/ -run TestGameBridge -v`
Expected: PASS

- [ ] **Step 5: Run full questengine package tests**

Run: `go test ./internal/questengine/ -v`
Expected: All existing tests still PASS (bridge doesn't change engine behavior)

- [ ] **Step 6: Commit**

```bash
git add internal/questengine/bridge.go internal/questengine/bridge_test.go
git commit -m "feat: quest engine GameBridge ActionContext implementation"
```

---

### Task 3: Hook — give.go Item Delivery Rework

This is the critical safety change. Intercept `item_give` events BEFORE transferring the item to the mob. If the quest engine handles it, consume the item from the player instead.

**Files:**
- Modify: `internal/usercommands/give.go` (lines ~148-213, the NPC item-give block)

- [ ] **Step 1: Add quest engine import to give.go**

Add to the import block:

```go
"github.com/GoMudEngine/GoMud/internal/questengine"
```

- [ ] **Step 2: Intercept item delivery before mob transfer**

In `give.go`, find the NPC item-give block (around line 175-202). Currently the code does:

```go
m.Character.StoreItem(giveItem)
user.Character.RemoveItem(giveItem)
// ... send messages ...
// ... fire onGive script ...
```

Replace the entire NPC item-give section (from line 175 where `m.Character.StoreItem(giveItem)` starts, through the onGive script fallback around line 213) with:

```go
					// ── Quest engine intercept ──────────────────────
					bridge := questengine.NewGameBridge(user, room.RoomId)
					result := questengine.GetEngine().Notify("item_give", questengine.EventDetails{
						UserId: user.UserId,
						RoomId: room.RoomId,
						MobId:  int(m.MobId),
						ItemId: giveItem.ItemId,
					}, bridge, bridge)

					if result.Handled && result.ConsumeItem {
						// Quest engine handled it — consume item from player, don't give to mob
						user.Character.RemoveItem(giveItem)

						user.SendText(
							fmt.Sprintf(`You give the <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, giveItem.DisplayName(), m.Character.Name),
						)
						room.SendText(
							fmt.Sprintf(`<ansi fg="username">%s</ansi> gave their <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, user.Character.Name, giveItem.DisplayName(), m.Character.Name),
							user.UserId,
						)
					} else {
						// Normal flow: transfer item to mob
						m.Character.StoreItem(giveItem)
						user.Character.RemoveItem(giveItem)

						user.SendText(
							fmt.Sprintf(`You give the <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, giveItem.DisplayName(), m.Character.Name),
						)
						room.SendText(
							fmt.Sprintf(`<ansi fg="username">%s</ansi> gave their <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, user.Character.Name, giveItem.DisplayName(), m.Character.Name),
							user.UserId,
						)

						events.AddToQueue(events.ItemOwnership{
							UserId: user.UserId,
							Item:   giveItem,
							Gained: false,
						})

						events.AddToQueue(events.ItemOwnership{
							MobInstanceId: m.InstanceId,
							Item:          giveItem,
							Gained:        true,
						})

						if handled, err := scripting.TryMobScriptEvent(`onGive`, m.InstanceId, user.UserId, `user`, map[string]any{`gold`: giveGoldAmount, `item`: giveItem}); err == nil {
							if handled {
								return true, nil
							}
						}

						m.Command(fmt.Sprintf(`emote considers the <ansi fg="itemname">%s</ansi> for a moment.`, giveItem.DisplayName()))
						m.Command(fmt.Sprintf(`gearup !%d`, giveItem.ItemId))
					}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/usercommands/`
Expected: Compiles without errors

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/give.go
git commit -m "feat: quest engine intercepts item_give before transfer in give.go"
```

---

### Task 4: Hook — room_enter in go.go

Add a `room_enter` notification after the player successfully moves to a new room.

**Files:**
- Modify: `internal/usercommands/go.go`

- [ ] **Step 1: Add quest engine import to go.go**

Add to the import block:

```go
"github.com/GoMudEngine/GoMud/internal/questengine"
```

- [ ] **Step 2: Add room_enter notification**

In `go.go`, find the block after `rooms.MoveToRoom(user.UserId, destRoom.RoomId)` succeeds (around line 246-248, inside the `} else {` after `MoveToRoom`). After the `scripting.TryRoomScriptEvent("onExit", ...)` call on line 250, add:

```go
			// Quest engine: room_enter notification
			bridge := questengine.NewGameBridge(user, destRoom.RoomId)
			questengine.GetEngine().Notify("room_enter", questengine.EventDetails{
				UserId: user.UserId,
				RoomId: destRoom.RoomId,
			}, bridge, bridge)
```

Place this AFTER the onExit script but BEFORE the look/message sends, so quest actions (like lock_exits or npc_say) execute before the player sees the room.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/usercommands/`
Expected: Compiles without errors

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/go.go
git commit -m "feat: quest engine room_enter hook in go.go"
```

---

### Task 5: Hook — skill_use in progression.go

Add a `skill_use` notification when `OnSkillUse` fires. This covers crafting, casting, foraging, combat, salvage, and any other skill usage.

**Files:**
- Modify: `internal/characters/progression.go`

- [ ] **Step 1: Add quest engine import**

Add to the import block:

```go
"github.com/GoMudEngine/GoMud/internal/questengine"
"github.com/GoMudEngine/GoMud/internal/users"
```

- [ ] **Step 2: Add skill_use notification at end of OnSkillUse**

In `progression.go`, at the end of `OnSkillUse` (before `return gained` on line 232), add:

```go
	// Quest engine: skill_use notification
	if userId > 0 {
		if u := users.GetByUserId(userId); u != nil {
			bridge := questengine.NewGameBridge(u, u.Character.RoomId)
			questengine.GetEngine().Notify("skill_use", questengine.EventDetails{
				UserId: userId,
				RoomId: u.Character.RoomId,
				Skill:  skillName,
			}, bridge, bridge)
		}
	}
```

The `userId > 0` check is important — mobs also call `OnSkillUse` with `userId=0` and we don't want to notify for NPC skill usage.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/characters/`
Expected: Compiles without errors

- [ ] **Step 4: Commit**

```bash
git add internal/characters/progression.go
git commit -m "feat: quest engine skill_use hook in OnSkillUse"
```

---

### Task 6: Hook — mob_death in MobDeath_PackFlee.go

Add a `mob_death` notification when a mob dies. We hook into the existing `MobDeath` event listener.

**Files:**
- Modify: `internal/hooks/MobDeath_PackFlee.go`

- [ ] **Step 1: Read the current PackFlee function**

Read `internal/hooks/MobDeath_PackFlee.go` to understand the full structure before modifying.

- [ ] **Step 2: Add quest engine notification**

Add import:

```go
"github.com/GoMudEngine/GoMud/internal/questengine"
"github.com/GoMudEngine/GoMud/internal/users"
```

At the top of the `PackFlee` function, after the type assertion succeeds and before the existing logic, add the quest notification. The `MobDeath` event has `PlayerDamage map[int]int` — each key is a userId who dealt damage. Notify for each player:

```go
	// Quest engine: notify all players who damaged this mob
	for userId := range evt.PlayerDamage {
		if u := users.GetByUserId(userId); u != nil {
			bridge := questengine.NewGameBridge(u, evt.RoomId)
			questengine.GetEngine().Notify("mob_death", questengine.EventDetails{
				UserId: userId,
				RoomId: evt.RoomId,
				MobId:  evt.MobId,
			}, bridge, bridge)
		}
	}
```

Insert this after the `evt` type assertion and `!ok` check (around line 14-16), before the existing pack flee logic.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/hooks/`
Expected: Compiles without errors

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/MobDeath_PackFlee.go
git commit -m "feat: quest engine mob_death hook in PackFlee"
```

---

### Task 7: Hook — item_gain in CheckItemQuests

Add an `item_gain` notification when a player gains an item.

**Files:**
- Modify: `internal/hooks/ItemOwnership_CheckItemQuests.go`

- [ ] **Step 1: Add quest engine notification**

Add imports:

```go
"github.com/GoMudEngine/GoMud/internal/questengine"
"github.com/GoMudEngine/GoMud/internal/users"
```

Inside the `if evt.Gained {` block (after the existing `QuestToken` check and `onFound` script call, around line 36), add:

```go
		// Quest engine: item_gain notification
		if u := users.GetByUserId(evt.UserId); u != nil {
			bridge := questengine.NewGameBridge(u, u.Character.RoomId)
			questengine.GetEngine().Notify("item_gain", questengine.EventDetails{
				UserId: evt.UserId,
				RoomId: u.Character.RoomId,
				ItemId: evt.Item.ItemId,
			}, bridge, bridge)
		}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/hooks/`
Expected: Compiles without errors

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/ItemOwnership_CheckItemQuests.go
git commit -m "feat: quest engine item_gain hook in CheckItemQuests"
```

---

### Task 8: Hook — dialogue in ask.go

Add a `dialogue` notification when a player asks an NPC about a topic.

**Files:**
- Modify: `internal/usercommands/ask.go`

- [ ] **Step 1: Read ask.go to find the right insertion point**

Read `internal/usercommands/ask.go` to understand where the dialogue topic is resolved and the NPC responds.

- [ ] **Step 2: Add quest engine notification**

Add import:

```go
"github.com/GoMudEngine/GoMud/internal/questengine"
```

After the dialogue system processes the ask and determines the topic (the point where the NPC has been identified and the topic string is known), add:

```go
	// Quest engine: dialogue notification
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("dialogue", questengine.EventDetails{
		UserId: user.UserId,
		RoomId: room.RoomId,
		MobId:  mobId,
		Topic:  topic,
	}, bridge, bridge)
```

The exact insertion point depends on the `ask.go` structure — place it after the mob is identified and topic is parsed, before or after the dialogue tree response. The quest engine should fire regardless of whether the dialogue tree matched.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/usercommands/`
Expected: Compiles without errors

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/ask.go
git commit -m "feat: quest engine dialogue hook in ask.go"
```

---

### Task 9: Hook — command notifications (buy, equip, track)

Add `command` notifications for specific commands used in quest triggers.

**Files:**
- Modify: `internal/usercommands/buy.go`
- Modify: `internal/usercommands/equip.go`
- Modify: `internal/usercommands/track.go`

- [ ] **Step 1: Read each file to find insertion points**

Read the top ~30 lines of `buy.go`, `equip.go`, and `track.go` to see their function signatures and determine where the command succeeds.

- [ ] **Step 2: Add command notification to buy.go**

Add import and, after a successful purchase (after the item is given and gold deducted), add:

```go
	// Quest engine: command notification
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "buy",
	}, bridge, bridge)
```

- [ ] **Step 3: Add command notification to equip.go**

Same pattern — after a successful equip:

```go
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "equip",
	}, bridge, bridge)
```

- [ ] **Step 4: Add command notification to track.go**

Same pattern — after a successful track:

```go
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "track",
	}, bridge, bridge)
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./internal/usercommands/`
Expected: Compiles without errors

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/buy.go internal/usercommands/equip.go internal/usercommands/track.go
git commit -m "feat: quest engine command hooks for buy, equip, track"
```

---

### Task 10: Full Build + Test Verification

Verify the entire project compiles and all existing tests pass with the new hooks.

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: Compiles without errors

- [ ] **Step 2: Run all questengine tests**

Run: `go test ./internal/questengine/ -v`
Expected: All PASS

- [ ] **Step 3: Run affected package tests**

Run: `go test ./internal/usercommands/ ./internal/hooks/ ./internal/characters/ -v -count=1`
Expected: All PASS (existing tests unaffected — the quest engine is empty with no registered quests, so all Notify calls are no-ops)

- [ ] **Step 4: Run full test suite**

Run: `go test ./... -count=1`
Expected: All PASS

- [ ] **Step 5: Commit (if any fixes were needed)**

Only if test failures required fixes:

```bash
git add -A
git commit -m "fix: resolve test issues from quest engine hook integration"
```

---

## Notes for Future Phases

**Phase 3** (Port Tutorial): Expand Quest 1 YAML with triggers (the spec has the full example), then live-test the tutorial end-to-end. Delete the JS scripts.

**Phase 4** (Port Remaining Quests): Expand all 17 quest YAMLs, write walkthrough tests, delete all quest JS scripts.

**Sequence action timing**: The current `QueueSequence` implementation in Task 2 executes lines immediately (no delays). A proper timed sequence using the round system is a Phase 3 enhancement — for now, the immediate execution is functionally correct.

**Player-scoped exit locks**: The current `LockExits`/`UnlockExits` implementation locks exits globally. True player-scoped locking requires room infrastructure changes — track as a Phase 3 item.
