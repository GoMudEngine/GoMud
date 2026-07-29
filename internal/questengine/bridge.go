package questengine

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/util"
	"runtime/debug"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
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

// GetQuestFlag returns the value of a quest flag from the player's character.
func (b *GameBridge) GetQuestFlag(key string) string {
	return b.user.Character.GetQuestFlag(key)
}

// GetGold returns the player's current gold total.
func (b *GameBridge) GetGold() int {
	return b.user.Character.Gold
}

// HasOwnMasterwork delegates to the character's own-crafted-item check.
func (b *GameBridge) HasOwnMasterwork(skillMin int) bool {
	return b.user.Character.HasOwnMasterwork(skillMin)
}

// --- ActionContext ---

// GetUserId returns the user's ID.
func (b *GameBridge) GetUserId() int {
	return b.user.UserId
}

// GrantQuest advances the player's quest progress to the given token.
// Sets the token immediately (for synchronous chain evaluation) and
// sends a quest progress banner to the player.
func (b *GameBridge) GrantQuest(token string) {
	if !b.user.Character.GiveQuestToken(token) {
		return
	}

	// Look up quest name for the banner
	_, stepName := quests.TokenToParts(token)
	questInfo := quests.GetQuest(token)
	questName := ""
	if questInfo != nil {
		questName = questInfo.Name
	}

	// Completion is handled entirely by HandleQuestUpdate via the events.Quest
	// event below: it sends the completion banner, writes the event log, and
	// applies all rewards. Emitting a banner here too would double it (the
	// "First Heat completed" duplicate seen in the 2026-06-29 newbie smoke and
	// the prior quest-60 report). So for "end" we ONLY queue the event and
	// return — do not also send a banner. "start"/"progress" do not queue an
	// event, so they emit their banner directly here (single, no duplicate).
	if stepName == "end" {
		events.AddToQueue(events.Quest{
			UserId:     b.user.UserId,
			QuestToken: token,
		})
		return
	}

	var bannerMsg string
	if stepName == "start" {
		bannerMsg = fmt.Sprintf("You have been given a new quest: <ansi fg=\"questname\">%s</ansi>!", questName)
	} else {
		bannerMsg = fmt.Sprintf("You've made progress on the quest: <ansi fg=\"questname\">%s</ansi>!", questName)
	}

	if questInfo != nil && !questInfo.Secret {
		questUpTxt, _ := templates.Process("character/questup", bannerMsg, b.user.UserId)
		b.user.SendText(messaging.CategorySystem, questUpTxt)
		b.user.EventLog.Add("quest", bannerMsg)
	}

	// Refresh the Char.Quests minimap marker on start/progress steps too. The
	// marker is driven by the GMCP quest-progress handler listening on
	// events.Quest; the "end" path above already queues one, but without this a
	// mid-quest step advance leaves the marker frozen on whatever step the
	// player loaded on. MarkerOnly so HandleQuestUpdate skips it — the banner
	// above is the single authoritative one (a plain events.Quest would double it).
	events.AddToQueue(events.Quest{
		UserId:     b.user.UserId,
		QuestToken: token,
		MarkerOnly: true,
	})
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
	b.user.SendText(messaging.CategoryLoot, fmt.Sprintf("You receive a <ansi fg=\"item\">%s</ansi>.", newItem.DisplayName()))
}

// GiveGold adds gold to the player's character and notifies the client.
func (b *GameBridge) GiveGold(amount int) {
	b.user.Character.Gold += amount
	b.user.SendText(messaging.CategoryLoot, fmt.Sprintf("You receive <ansi fg=\"gold\">%d gold</ansi>.", amount))
	events.AddToQueue(events.EquipmentChange{
		UserId:     b.user.UserId,
		GoldChange: amount,
	})
}

// ChargeGold deducts gold from the player's character, clamped so the
// player's gold never goes negative, and notifies the client.
func (b *GameBridge) ChargeGold(amount int) {
	if amount > b.user.Character.Gold {
		amount = b.user.Character.Gold
	}
	b.user.Character.Gold -= amount
	b.user.SendText(messaging.CategoryLoot, fmt.Sprintf("You pay <ansi fg=\"gold\">%d gold</ansi>.", amount))
	events.AddToQueue(events.EquipmentChange{
		UserId:     b.user.UserId,
		GoldChange: -amount,
	})
}

// SendText sends a categorized message directly to the player.
func (b *GameBridge) SendText(cat messaging.Category, text string) {
	b.user.SendText(cat, text)
}

// RoomText sends a message to everyone in the room except the triggering player.
func (b *GameBridge) RoomText(text string) {
	room := rooms.LoadRoom(b.roomId)
	if room == nil {
		mudlog.Error("GameBridge.RoomText", "error", fmt.Sprintf("room %d not found", b.roomId))
		return
	}
	room.SendText(messaging.CategoryNPCDialogue, text, b.user.UserId)
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
		b.user.SendText(messaging.CategorySystem, fmt.Sprintf("You have learned the <ansi fg=\"spellname\">%s</ansi> spell.", spellId))
	}
}

// TrainSkill sets the player's skill to the specified level.
func (b *GameBridge) TrainSkill(skill string, level int) {
	b.user.Character.SetSkill(skill, level)
}

// statGainMessages maps lowercase stat names to a flavourful gain phrase
// shown when a quest permanently increases that stat. No hard numbers.
var statGainMessages = map[string]string{
	"strength":   "You feel notably stronger — your muscles carry a new permanence.",
	"dexterity":  "Your movements sharpen; a new quickness settles into your limbs.",
	"perception": "The world comes into slightly sharper focus, details you once missed now clear.",
	"vitality":   "A deep resilience takes root — your body feels more enduring than before.",
	"willpower":  "Your mind firms, a quiet resolve that wasn't there before now anchoring you.",
	"charisma":   "Something in your bearing shifts; others may find you more compelling.",
}

// IncreaseStat permanently adds the given amount to the named stat's Training
// and notifies the player with a descriptive message.
func (b *GameBridge) IncreaseStat(stat string, amount int) {
	if amount <= 0 {
		return
	}
	if !b.user.Character.IncreaseStat(stat, amount) {
		mudlog.Error("GameBridge.IncreaseStat",
			"stat", stat, "amount", amount,
			"error", "unknown stat name — no change applied")
		return
	}
	msg, ok := statGainMessages[stat]
	if !ok {
		msg = "You feel a subtle but permanent improvement wash over you."
	}
	b.user.SendText(messaging.CategorySkillProgress,
		fmt.Sprintf("<ansi fg=\"yellow-bold\">%s</ansi>", msg))
}

// LearnRecipe grants the player a new crafting recipe, notifying them if it
// was newly learned. Silently skips recipes already in KnownRecipes.
func (b *GameBridge) LearnRecipe(recipe string) {
	if b.user.Character.LearnRecipe(recipe) {
		b.user.SendText(messaging.CategorySkillProgress, fmt.Sprintf(
			"<ansi fg=\"cyan\">You have learned the recipe for </ansi><ansi fg=\"itemname\">%s</ansi><ansi fg=\"cyan\">.</ansi>",
			recipe))
	}
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

// SetQuestFlag sets a quest flag on the player's character.
func (b *GameBridge) SetQuestFlag(key, value string) {
	b.user.Character.SetQuestFlag(key, value)
}

// BumpRep delegates to the factions package using this bridge's
// user as the rep target. No-op if the faction is unknown
// (factions.BumpRep handles that internally with a Warn log).
func (b *GameBridge) BumpRep(factionId string, delta int) {
	factions.BumpRep(factionId, b.user.UserId, delta)
}

// QueueNpcSay finds the mob in the room and makes it say each line.
func (b *GameBridge) GiveMutation() {
	if b.user.Character.Mutations == nil {
		b.user.Character.Mutations = make(map[string]int)
	}
	sp := species.GetSpecies(b.user.Character.SpeciesId)
	pool := mutations.GetWeightedPool(b.user.Character.Mutations, sp)
	if len(pool) == 0 {
		return
	}
	mutId := mutations.RollAcquisition(pool)
	if mutId == "" {
		return
	}
	if _, exists := b.user.Character.Mutations[mutId]; !exists {
		b.user.Character.Mutations[mutId] = 1
		events.AddToQueue(mutations.Gained{
			UserId:     b.user.UserId,
			MutationId: mutId,
			Rank:       1,
			IsNew:      true,
		})
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
// recoverQuestSequence contains a panic in one of QueueSequence's timer
// goroutines.
//
// These fire on bare timers rather than the tick loop, and their work is driven
// by content — quest YAML — not code, so a malformed sequence is a far more
// likely trigger than a code bug. Go cannot recover a child goroutine's panic
// from its parent, so without this a bad quest definition killed the server.
//
// Known residual gap: neither goroutine checks for server shutdown, so a
// sequence queued shortly before shutdown can still fire after users have been
// saved, silently losing the mutation. Closing that needs a shutdown signal
// reachable from this package (serverAlive lives in package main), or moving
// sequence delivery onto the event queue with a ReadyTurn so it participates in
// the tick loop.
func recoverQuestSequence(what string) {
	if r := recover(); r != nil {
		mudlog.Error("GameBridge.QueueSequence",
			"error", "panic in quest sequence goroutine",
			"stage", what,
			"panic", r,
			"stack", string(debug.Stack()))
	}
}

func (b *GameBridge) QueueSequence(s SequenceDef) {
	// Lock player movement during the sequence if configured
	if s.LockMessage != "" {
		b.user.SetTempData(`questSequenceLock`, s.LockMessage)
	}

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
					defer recoverQuestSequence("delayed dialogue line")

					<-time.After(time.Duration(delay) * time.Second)

					// Touching live user state from a timer goroutine, which
					// never holds the world lock otherwise.
					util.LockMud()
					defer util.UnlockMud()

					if u := users.GetByUserId(userId); u != nil {
						u.SendText(messaging.CategoryNPCDialogue, text)
					}
				}()
			} else {
				b.user.SendText(messaging.CategoryNPCDialogue, line.Text)
			}
		}
	}

	// Schedule on_complete after the longest delay + buffer.
	if len(s.OnComplete) > 0 || s.LockMessage != "" {
		waitSeconds := maxDelay + 3
		userId := b.user.UserId
		roomId := b.roomId
		onComplete := s.OnComplete
		lockMsg := s.LockMessage
		go func() {
			defer recoverQuestSequence("on_complete actions")

			time.Sleep(time.Duration(waitSeconds) * time.Second)

			// ExecuteAction runs arbitrary content-authored quest actions —
			// granting items, mutating the character, moving the player. Doing
			// that from an unlocked timer goroutine raced the tick loop over
			// the same objects.
			util.LockMud()
			defer util.UnlockMud()

			u := users.GetByUserId(userId)
			if u == nil {
				return
			}
			// Clear movement lock
			if lockMsg != "" {
				u.SetTempData(`questSequenceLock`, "")
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
