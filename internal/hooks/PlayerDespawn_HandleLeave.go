package hooks

import (
	"fmt"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// saveCompanionState copies live mob progression data into the CompanionInfo
// and then cleanly despawns the mob instance.
func saveCompanionState(user *users.UserRecord) {
	for i := range user.Character.Companions {
		comp := &user.Character.Companions[i]
		if comp.InstanceId == 0 {
			continue
		}
		mob := mobs.GetInstance(comp.InstanceId)
		if mob == nil {
			comp.InstanceId = 0
			continue
		}

		// Copy stat training from individual fields into the map.
		if comp.StatTraining == nil {
			comp.StatTraining = make(map[string]int)
		}
		comp.StatTraining["strength"] = mob.Character.Stats.Strength.Training
		comp.StatTraining["dexterity"] = mob.Character.Stats.Dexterity.Training
		comp.StatTraining["perception"] = mob.Character.Stats.Perception.Training
		comp.StatTraining["vitality"] = mob.Character.Stats.Vitality.Training
		comp.StatTraining["willpower"] = mob.Character.Stats.Willpower.Training
		comp.StatTraining["charisma"] = mob.Character.Stats.Charisma.Training

		// Copy skill progression maps (shallow copy is fine — we replace the map).
		comp.Skills = copyIntMap(mob.Character.Skills)
		comp.SkillUseCount = copyIntMap(mob.Character.SkillUseCount)
		comp.Mutations = copyIntMap(mob.Character.Mutations)
		comp.SpellBook = copyIntMap(mob.Character.SpellBook)
		comp.MutationProgress = mob.Character.MutationProgress

		// Snapshot carried + equipped gear so it survives logout. Shallow-copy
		// the Items slice (Item is a value type; the slice itself owns its
		// elements). Equipment is a struct of Item values — direct assignment
		// is a full copy.
		if len(mob.Character.Items) > 0 {
			comp.Items = make([]items.Item, len(mob.Character.Items))
			copy(comp.Items, mob.Character.Items)
		} else {
			comp.Items = nil
		}
		comp.Equipment = mob.Character.Equipment

		// Remove charm so the mob does not flag as charmed in any cleanup loops.
		mob.Character.RemoveCharm()

		// Pull mob out of its current room.
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			room.RemoveMob(mob.InstanceId)
		}

		mobs.DestroyInstance(mob.InstanceId)
		comp.InstanceId = 0
	}
}

// copyIntMap returns a shallow copy of a map[string]int (or nil if src is nil).
func copyIntMap(src map[string]int) map[string]int {
	if src == nil {
		return nil
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

//
// Some clean up
//

func HandleLeave(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerDespawn)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PlayerDespawn", "Actual Type", e.Type())
		return events.Cancel
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		mudlog.Error("HandleLeave", "error", fmt.Sprintf(`user %d not found`, evt.UserId))
		return events.Cancel
	}

	connId := user.ConnectionId()

	// Remove any zombie tracking for the user since they've been despawned from the world.
	if users.IsZombieConnection(connId) {
		users.RemoveZombieUser(evt.UserId)
	}

	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		mudlog.Error("HandleLeave", "error", fmt.Sprintf("room %d not found for user %d", user.Character.RoomId, evt.UserId))
		return events.Cancel
	}

	if currentParty := parties.Get(evt.UserId); currentParty != nil {
		// Settle the shared gold pool before the departing member is removed,
		// so their (and other members') pooled gold isn't lost on logout.
		payoutPartyGold(currentParty)
		currentParty.Leave(evt.UserId)
	}

	// Save companion progression and cleanly despawn companion instances.
	saveCompanionState(user)

	for _, mobInstId := range room.GetMobs(rooms.FindCharmed) {
		if mob := mobs.GetInstance(mobInstId); mob != nil {
			if mob.Character.IsCharmed(evt.UserId) {
				mob.Character.Charmed.Expire()
			}
		}
	}

	if _, ok := room.RemovePlayer(evt.UserId); ok {
		tplTxt, _ := templates.Process("player-despawn", user.Character.Name)
		sendVisualRoomText(room, messaging.CategoryLogout, tplTxt)
	}

	tplTxt, _ := templates.Process("goodbye", nil, evt.UserId)
	connections.SendTo([]byte(templates.AnsiParse(tplTxt)), connId)

	if err := users.LogOutUserByConnectionId(connId); err != nil {
		mudlog.Error("Log Out Error", "connectionId", connId, "error", err)
	}
	connections.Remove(connId)

	specialRooms := configs.GetSpecialRoomsConfig()
	testRoomId := rooms.GetOriginalRoom(user.Character.RoomId)
	for i := range specialRooms.TutorialRooms {
		roomId, _ := strconv.Atoi(specialRooms.TutorialRooms[i])
		if roomId == testRoomId {
			user.Character.RoomId = -1
			break
		}
	}

	users.SaveUser(user)

	return events.Continue
}
