package questengine

import (
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// GameBridge wraps a UserRecord and a room ID to satisfy both PlayerState
// and ActionContext. The roomId is passed explicitly because some events
// fire after room changes.
type GameBridge struct {
	user   *users.UserRecord
	roomId int
}

// NewGameBridge creates a GameBridge for the given user positioned in roomId.
func NewGameBridge(user *users.UserRecord, roomId int) *GameBridge {
	return &GameBridge{user: user, roomId: roomId}
}

// --- PlayerState ---

// HasQuest delegates to the character's quest progress.
func (b *GameBridge) HasQuest(token string) bool {
	return b.user.Character.HasQuest(token)
}

// HasItem returns true if the player has at least one item with the given ID.
func (b *GameBridge) HasItem(itemId int) bool {
	for _, itm := range b.user.Character.Items {
		if itm.ItemId == itemId {
			return true
		}
	}
	return false
}

// GetRoomId returns the room ID this bridge was constructed with.
func (b *GameBridge) GetRoomId() int {
	return b.roomId
}

// --- ActionContext ---

// GetUserId returns the user's ID.
func (b *GameBridge) GetUserId() int {
	return b.user.UserId
}

// GrantQuest advances the player's quest progress to the given token.
func (b *GameBridge) GrantQuest(token string) {
	b.user.Character.GiveQuestToken(token)
}

// ConsumeItem removes the first matching item from the player's inventory.
func (b *GameBridge) ConsumeItem(itemId int) {
	for _, itm := range b.user.Character.Items {
		if itm.ItemId == itemId {
			b.user.Character.RemoveItem(itm)
			return
		}
	}
	mudlog.Error("GameBridge.ConsumeItem", "error", fmt.Sprintf("item %d not in inventory", itemId))
}

// GiveItem creates a new item and places it in the player's inventory.
func (b *GameBridge) GiveItem(itemId int) {
	newItem := items.New(itemId)
	if newItem.ItemId < 1 {
		mudlog.Error("GameBridge.GiveItem", "error", fmt.Sprintf("item %d not found", itemId))
		return
	}
	b.user.Character.StoreItem(newItem)
	b.user.SendText(fmt.Sprintf("You receive a <ansi fg=\"item\">%s</ansi>.", newItem.DisplayName()))
}

// GiveGold adds gold to the player's character and notifies the client.
func (b *GameBridge) GiveGold(amount int) {
	b.user.Character.Gold += amount
	b.user.SendText(fmt.Sprintf("You receive <ansi fg=\"gold\">%d gold</ansi>.", amount))
	events.AddToQueue(events.EquipmentChange{
		UserId:     b.user.UserId,
		GoldChange: amount,
	})
}

// SendText sends a message directly to the player.
func (b *GameBridge) SendText(text string) {
	b.user.SendText(text)
}

// RoomText sends a message to everyone in the room except the triggering player.
func (b *GameBridge) RoomText(text string) {
	room := rooms.LoadRoom(b.roomId)
	if room == nil {
		mudlog.Error("GameBridge.RoomText", "error", fmt.Sprintf("room %d not found", b.roomId))
		return
	}
	room.SendText(text, b.user.UserId)
}

// SpawnMob creates a new mob instance and places it in the target room.
func (b *GameBridge) SpawnMob(s SpawnDef) {
	mob := mobs.NewMobById(mobs.MobId(s.Id), s.Room)
	if mob == nil {
		mudlog.Error("GameBridge.SpawnMob", "error", fmt.Sprintf("mob %d not found", s.Id))
		return
	}
	room := rooms.LoadRoom(s.Room)
	if room == nil {
		mudlog.Error("GameBridge.SpawnMob", "error", fmt.Sprintf("room %d not found", s.Room))
		return
	}
	room.AddMob(mob.InstanceId)
}

// SpawnItem creates a new item and places it on the floor of the target room.
func (b *GameBridge) SpawnItem(s SpawnDef) {
	newItem := items.New(s.Id)
	if newItem.ItemId < 1 {
		mudlog.Error("GameBridge.SpawnItem", "error", fmt.Sprintf("item %d not found", s.Id))
		return
	}
	room := rooms.LoadRoom(s.Room)
	if room == nil {
		mudlog.Error("GameBridge.SpawnItem", "error", fmt.Sprintf("room %d not found", s.Room))
		return
	}
	room.AddItem(newItem, false)
}

// TeachSpell grants the player a new spell, notifying them if it was learned.
func (b *GameBridge) TeachSpell(spellId string) {
	if b.user.Character.LearnSpell(spellId) {
		b.user.SendText(fmt.Sprintf("You have learned the <ansi fg=\"spellname\">%s</ansi> spell.", spellId))
	}
}

// TrainSkill sets the player's skill to the specified level.
func (b *GameBridge) TrainSkill(skill string, level int) {
	b.user.Character.SetSkill(skill, level)
}

// ApplyBuff adds the given buff to the player.
func (b *GameBridge) ApplyBuff(bf BuffDef) {
	if err := b.user.Character.AddBuff(bf.Buff, false); err != nil {
		mudlog.Error("GameBridge.ApplyBuff", "buff", bf.Buff, "error", err)
	}
}

// Teleport moves the player to the specified room.
func (b *GameBridge) Teleport(roomId int) {
	if err := rooms.MoveToRoom(b.user.UserId, roomId); err != nil {
		mudlog.Error("GameBridge.Teleport", "room", roomId, "error", err)
	}
}

// LockExits locks all exits in the specified room.
func (b *GameBridge) LockExits(e ExitLock) {
	room := rooms.LoadRoom(e.Room)
	if room == nil {
		mudlog.Error("GameBridge.LockExits", "error", fmt.Sprintf("room %d not found", e.Room))
		return
	}
	for exitName := range room.Exits {
		room.SetExitLock(exitName, true)
	}
}

// UnlockExits unlocks all exits in the specified room.
func (b *GameBridge) UnlockExits(e ExitLock) {
	room := rooms.LoadRoom(e.Room)
	if room == nil {
		mudlog.Error("GameBridge.UnlockExits", "error", fmt.Sprintf("room %d not found", e.Room))
		return
	}
	for exitName := range room.Exits {
		room.SetExitLock(exitName, false)
	}
}

// QueueNpcSay finds the mob in the room and makes it say each line.
func (b *GameBridge) GiveMutation() {
	if b.user.Character.Mutations == nil {
		b.user.Character.Mutations = make(map[string]int)
	}
	pool := mutations.GetWeightedPool(b.user.Character.Mutations)
	if len(pool) == 0 {
		return
	}
	mutId := mutations.RollAcquisition(pool)
	if mutId == "" {
		return
	}
	if _, exists := b.user.Character.Mutations[mutId]; !exists {
		b.user.Character.Mutations[mutId] = 1
	}
}

// npcCommand builds a "say" or "emote" command string from a SayLineDef.
func npcCommand(line SayLineDef) string {
	if line.Emote {
		return fmt.Sprintf("emote %s", line.Text)
	}
	return fmt.Sprintf("say %s", line.Text)
}

// QueueNpcSay finds the mob in the room and schedules each line with delays.
func (b *GameBridge) QueueNpcSay(n NpcSayDef) {
	for _, line := range n.Lines {
		speakerMobId := n.Mob
		if line.Speaker != 0 {
			speakerMobId = line.Speaker
		}
		mob := findMobInRoom(speakerMobId, b.roomId)
		if mob == nil {
			mudlog.Error("GameBridge.QueueNpcSay", "error",
				fmt.Sprintf("mob spec %d not found in room %d", speakerMobId, b.roomId))
			continue
		}
		if line.Delay > 0 {
			mob.Command(npcCommand(line), float64(line.Delay))
		} else {
			mob.Command(npcCommand(line))
		}
	}
}

// QueueSequence schedules lines with per-line delays or a default
// delay_between interval. Speaker lines become mob commands; non-speaker
// lines are sent as delayed user messages. On-complete actions fire
// after the longest delay plus a buffer.
func (b *GameBridge) QueueSequence(s SequenceDef) {
	maxDelay := 0
	for _, line := range s.Lines {
		delay := line.Delay
		if delay == 0 {
			delay = s.DelayBetween
		}
		if delay > maxDelay {
			maxDelay = delay
		}

		if line.Speaker != 0 {
			mob := findMobInRoom(line.Speaker, b.roomId)
			if mob == nil {
				mudlog.Error("GameBridge.QueueSequence", "error",
					fmt.Sprintf("mob spec %d not found in room %d", line.Speaker, b.roomId))
				continue
			}
			if delay > 0 {
				mob.Command(npcCommand(line), float64(delay))
			} else {
				mob.Command(npcCommand(line))
			}
		} else {
			if delay > 0 {
				text := line.Text
				userId := b.user.UserId
				go func() {
					<-time.After(time.Duration(delay) * time.Second)
					if u := users.GetByUserId(userId); u != nil {
						u.SendText(text)
					}
				}()
			} else {
				b.user.SendText(line.Text)
			}
		}
	}

	// Schedule on_complete: mob.Command uses the turn system so
	// "delay: 34" in YAML may take longer than 34 real seconds.
	// We use a mob noop command scheduled at maxDelay+3 to sync
	// with the turn-based timing, then fire on_complete from
	// the noop's resolution via a polling goroutine.
	if len(s.OnComplete) > 0 {
		// Find any speaker mob to piggyback on their turn scheduling
		var anchorMob *mobs.Mob
		for _, line := range s.Lines {
			if line.Speaker != 0 {
				anchorMob = findMobInRoom(line.Speaker, b.roomId)
				if anchorMob != nil {
					break
				}
			}
		}

		if anchorMob != nil {
			// Schedule a noop after the last line; its lastCommandTurn
			// tells us when the mob will be done.
			anchorMob.Command("noop", float64(maxDelay+3))
			targetTurn := anchorMob.GetLastCommandTurn()

			userId := b.user.UserId
			roomId := b.roomId
			onComplete := s.OnComplete
			go func() {
				for util.GetTurnCount() < targetTurn {
					time.Sleep(500 * time.Millisecond)
				}
				u := users.GetByUserId(userId)
				if u == nil {
					return
				}
				bridge := NewGameBridge(u, roomId)
				for _, action := range onComplete {
					if err := ExecuteAction(action, bridge); err != nil {
						mudlog.Error("GameBridge.QueueSequence.OnComplete", "error", err)
					}
				}
			}()
		}
	}
}

// findMobInRoom searches the room's mob instances for one whose spec ID matches.
func findMobInRoom(mobSpecId int, roomId int) *mobs.Mob {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return nil
	}
	for _, instanceId := range room.GetMobs() {
		mob := mobs.GetInstance(instanceId)
		if mob == nil {
			continue
		}
		if int(mob.MobId) == mobSpecId {
			return mob
		}
	}
	return nil
}
