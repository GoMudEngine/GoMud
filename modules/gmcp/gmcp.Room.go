package gmcp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ////////////////////////////////////////////////////////////////////
// NOTE: The init function in Go is a special function that is
// automatically executed before the main function within a package.
// It is used to initialize variables, set up configurations, or
// perform any other setup tasks that need to be done before the
// program starts running.
// ////////////////////////////////////////////////////////////////////
func init() {

	//
	// We can use all functions only, but this demonstrates
	// how to use a struct
	//
	g := GMCPRoomModule{
		plug: plugins.New(`gmcp.Room`, `1.0`),
	}

	// Temporary for testing purposes.
	events.RegisterListener(events.RoomChange{}, g.roomChangeHandler)
	events.RegisterListener(events.PlayerDespawn{}, g.despawnHandler)
	events.RegisterListener(GMCPRoomUpdate{}, g.buildAndSendGMCPPayload)

}

type GMCPRoomModule struct {
	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
	plug *plugins.Plugin
}

// Tell the system a wish to send specific GMCP Update data
type GMCPRoomUpdate struct {
	UserId     int
	Identifier string
}

func (g GMCPRoomUpdate) Type() string { return `GMCPRoomUpdate` }

func (g *GMCPRoomModule) despawnHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerDespawn)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PlayerDespawn", "Actual Type", e.Type())
		return events.Cancel
	}

	// If this isn't a user changing rooms, just pass it along.
	if evt.UserId == 0 {
		return events.Continue
	}

	room := rooms.LoadRoom(evt.RoomId)
	if room == nil {
		return events.Continue
	}

	//
	// Send GMCP Updates for players leaving
	//
	for _, uid := range room.GetPlayers() {

		if uid == evt.UserId {
			continue
		}

		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}

		events.AddToQueue(GMCPOut{
			UserId:  uid,
			Payload: fmt.Sprintf(`Room.RemovePlayer "%s"`, evt.CharacterName),
		})

	}

	return events.Continue
}

func (g *GMCPRoomModule) roomChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.RoomChange)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "RoomChange", "Actual Type", e.Type())
		return events.Cancel
	}

	// Send updates to players in old/new rooms for
	// players or npcs (whichever changed)
	updateId := `Room.Info.Contents.Players`
	if evt.MobInstanceId > 0 {
		updateId = `Room.Info.Contents.Npcs`
	}

	if evt.FromRoomId != 0 {
		if oldRoom := rooms.LoadRoom(evt.FromRoomId); oldRoom != nil {
			for _, uId := range oldRoom.GetPlayers() {
				if uId == evt.UserId {
					continue
				}
				events.AddToQueue(GMCPRoomUpdate{
					UserId:     uId,
					Identifier: updateId,
				})
			}
		}
	}

	if evt.ToRoomId != 0 {
		if newRoom := rooms.LoadRoom(evt.ToRoomId); newRoom != nil {
			for _, uId := range newRoom.GetPlayers() {
				if uId == evt.UserId {
					continue
				}
				events.AddToQueue(GMCPRoomUpdate{
					UserId:     uId,
					Identifier: updateId,
				})
			}
		}
	}

	// If it's a mob changing rooms, don't need to send it its own room info
	if evt.UserId == 0 {
		return events.Continue
	}

	// Send update to the moved player about their new room.
	events.AddToQueue(GMCPRoomUpdate{
		UserId:     evt.UserId,
		Identifier: `Room.Info`,
	})

	return events.Continue
}

func (g *GMCPRoomModule) buildAndSendGMCPPayload(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(GMCPRoomUpdate)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "GMCPCharUpdate", "Actual Type", e.Type())
		return events.Cancel
	}

	if evt.UserId < 1 {
		return events.Continue
	}

	// Make sure they have this gmcp module enabled.
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return events.Cancel
	}

	if len(evt.Identifier) >= 4 {

		for _, identifier := range strings.Split(evt.Identifier, `,`) {

			identifier = strings.TrimSpace(identifier)

			identifierParts := strings.Split(strings.ToLower(identifier), `.`)
			for i := 0; i < len(identifierParts); i++ {
				identifierParts[i] = strings.Title(identifierParts[i])
			}

			requestedId := strings.Join(identifierParts, `.`)

			payload, moduleName := g.GetRoomNode(user, requestedId)

			events.AddToQueue(GMCPOut{
				UserId:  evt.UserId,
				Module:  moduleName,
				Payload: payload,
			})

		}

	}

	return events.Continue
}

func (g *GMCPRoomModule) GetRoomNode(user *users.UserRecord, gmcpModule string) (data any, moduleName string) {

	all := gmcpModule == `Room.Info`

	// Get the new room data... abort if doesn't exist.
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return GMCPRoomModule_Payload{}, `Room.Info`
	}

	payload := GMCPRoomModule_Payload{}

	////////////////////////////////////////////////
	// Room.Contents
	// Note: Process this first since we might be
	//       sending a subset of data
	////////////////////////////////////////////////

	////////////////////////////////////////////////
	// Room.Contents.Containers
	////////////////////////////////////////////////
	if all || g.wantsGMCPPayload(`Room.Info.Contents.Containers`, gmcpModule) {

		payload.Contents.Containers = []GMCPRoomModule_Payload_Contents_Container{}
		for name, container := range room.Containers {

			c := GMCPRoomModule_Payload_Contents_Container{
				Name:   name,
				Usable: len(container.Recipes) > 0,
			}

			if container.HasLock() {
				c.Locked = true
				lockId := fmt.Sprintf(`%d-%s`, room.RoomId, name)
				c.HasKey, c.HasPickCombo = user.Character.HasKey(lockId, int(container.Lock.Difficulty))
			}

			payload.Contents.Containers = append(payload.Contents.Containers, c)
		}

		if `Room.Info.Contents.Containers` == gmcpModule {
			return payload.Contents.Containers, `Room.Info.Contents.Containers`
		}
	}

	////////////////////////////////////////////////
	// Room.Contents.Items
	////////////////////////////////////////////////
	if all || g.wantsGMCPPayload(`Room.Info.Contents.Items`, gmcpModule) {
		payload.Contents.Items = []GMCPRoomModule_Payload_Contents_Item{}
		for _, itm := range room.Items {
			payload.Contents.Items = append(payload.Contents.Items, GMCPRoomModule_Payload_Contents_Item{
				Id:        itm.ShorthandId(),
				Name:      itm.Name(),
				QuestFlag: itm.GetSpec().QuestToken != ``,
			})
		}

		if `Room.Info.Contents.Items` == gmcpModule {
			return payload.Contents.Items, `Room.Info.Contents.Items`
		}
	}

	////////////////////////////////////////////////
	// Room.Contents.Players
	////////////////////////////////////////////////
	if all || g.wantsGMCPPayload(`Room.Info.Contents.Players`, gmcpModule) {
		payload.Contents.Players = []GMCPRoomModule_Payload_Contents_Character{}
		for _, uId := range room.GetPlayers() {

			// Exclude viewing player
			if uId == user.UserId {
				continue
			}

			u := users.GetByUserId(uId)
			if u == nil {
				continue
			}

			if u.Character.HasBuffFlag(buffs.Hidden) {
				continue
			}

			payload.Contents.Players = append(payload.Contents.Players, GMCPRoomModule_Payload_Contents_Character{
				Id:         u.ShorthandId(),
				Name:       u.Character.Name,
				Adjectives: u.Character.GetAdjectives(),
				Aggro:      u.Character.Aggro != nil,
			})
		}

		if `Room.Info.Contents.Players` == gmcpModule {
			return payload.Contents.Players, `Room.Info.Contents.Players`
		}
	}

	////////////////////////////////////////////////
	// Room.Contents.Npcs
	////////////////////////////////////////////////
	if all || g.wantsGMCPPayload(`Room.Info.Contents.Npcs`, gmcpModule) {
		payload.Contents.Npcs = []GMCPRoomModule_Payload_Contents_Character{}
		for _, mIId := range room.GetMobs() {
			mob := mobs.GetInstance(mIId)
			if mob == nil {
				continue
			}

			if mob.Character.HasBuffFlag(buffs.Hidden) {
				continue
			}

			c := GMCPRoomModule_Payload_Contents_Character{
				Id:         mob.ShorthandId(),
				Name:       mob.Character.Name,
				Adjectives: mob.Character.GetAdjectives(),
				Aggro:      mob.Character.Aggro != nil,
			}

			if len(mob.QuestFlags) > 0 {
				for _, qFlag := range mob.QuestFlags {
					if user.Character.HasQuest(qFlag) || (len(qFlag) >= 5 && qFlag[len(qFlag)-5:] == `start`) {
						c.QuestFlag = true
						break
					}
				}
			}

			payload.Contents.Npcs = append(payload.Contents.Npcs, c)
		}

		if `Room.Info.Contents.Npcs` == gmcpModule {
			return payload.Contents.Npcs, `Room.Info.Contents.Npcs`
		}

	}

	if !all && `Room.Info.Contents` == gmcpModule {
		return payload.Contents, `Room.Info.Contents`
	}

	////////////////////////////////////////////////
	// Room.Info
	// Note: This populates the root Room.Info data
	////////////////////////////////////////////////
	if all || g.wantsGMCPPayload(`Room.Info`, gmcpModule) {

		// Basic details
		payload.Id = room.RoomId
		payload.Name = room.Title
		payload.Description = room.Description
		payload.Area = room.Zone
		payload.Environment = room.GetBiome().Name
		payload.Details = []string{}

		// Coordinates
		payload.Coordinates = room.Zone
		m := mapper.GetMapper(room.RoomId)
		x, y, z, err := m.GetCoordinates(room.RoomId)
		if err != nil {
			payload.Coordinates += `, 999999999999999999, 999999999999999999, 999999999999999999`
		} else {
			payload.Coordinates += `, ` + strconv.Itoa(x) + `, ` + strconv.Itoa(y) + `, ` + strconv.Itoa(z)
		}

		// set exits
		payload.Exits = map[string]int{}
		payload.ExitsV2 = map[string]GMCPRoomModule_Payload_Contents_ExitInfo{}

		for exitName, exitInfo := range room.Exits {

			if exitInfo.Secret {
				if exitRoom := rooms.LoadRoom(exitInfo.RoomId); exitRoom != nil {
					if !exitRoom.HasVisited(user.UserId, rooms.VisitorUser) {
						continue
					}
				}
			}

			// Skip non compass directions?
			if !mapper.IsCompassDirection(exitName) {
				//continue
			}

			payload.Exits[exitName] = exitInfo.RoomId

			// Form the "exitV2"
			deltaX, deltaY, deltaZ := 0, 0, 0
			if len(exitInfo.MapDirection) > 0 {
				deltaX, deltaY, deltaZ = mapper.GetDelta(exitInfo.MapDirection)
			} else {
				deltaX, deltaY, deltaZ = mapper.GetDelta(exitName)
			}

			exitV2 := GMCPRoomModule_Payload_Contents_ExitInfo{
				RoomId:  exitInfo.RoomId,
				DeltaX:  deltaX,
				DeltaY:  deltaY,
				DeltaZ:  deltaZ,
				Details: []string{},
			}

			if exitInfo.Secret {
				exitV2.Details = append(exitV2.Details, `secret`)
			}

			if exitInfo.HasLock() {

				exitV2.Details = append(exitV2.Details, `locked`)

				lockId := fmt.Sprintf(`%d-%s`, room.RoomId, exitName)
				haskey, hascombo := user.Character.HasKey(lockId, int(exitInfo.Lock.Difficulty))

				if haskey {
					exitV2.Details = append(exitV2.Details, `player_has_key`)
				}

				if hascombo {
					exitV2.Details = append(exitV2.Details, `player_has_pick_combo`)
				}
			}

			payload.ExitsV2[exitName] = exitV2
		}
		// end exits

		// Set room details
		if len(room.SkillTraining) > 0 {
			payload.Details = append(payload.Details, `trainer`)
		}
		if room.IsBank {
			payload.Details = append(payload.Details, `bank`)
		}
		if room.IsStorage {
			payload.Details = append(payload.Details, `storage`)
		}
		if room.IsCharacterRoom {
			payload.Details = append(payload.Details, `character`)
		}

		// Indicate if this is an ephemeral room
		if rooms.IsEphemeralRoomId(room.RoomId) {
			payload.Details = append(payload.Details, `ephemeral`)
		}
		// end room details

	}

	// If we reached this point and Char wasn't requested, we have a problem.
	if !all {
		mudlog.Error(`gmcp.Room`, `error`, `Bad module requested`, `module`, gmcpModule)
	}

	return payload, `Room.Info`
}

// wantsGMCPPayload(`Room.Info.Contents`, `Room.Info`)
func (g *GMCPRoomModule) wantsGMCPPayload(packageToConsider string, packageRequested string) bool {

	if packageToConsider == packageRequested {
		return true
	}

	if len(packageToConsider) < len(packageRequested) {
		return false
	}

	if packageToConsider[0:len(packageRequested)] == packageRequested {
		return true
	}

	return false
}

type GMCPRoomModule_Payload struct {
	Id          int                                                 `json:"num"`
	Name        string                                              `json:"name"`
	Description string                                              `json:"description"`
	Area        string                                              `json:"area"`
	Environment string                                              `json:"environment"`
	Coordinates string                                              `json:"coords"`
	Exits       map[string]int                                      `json:"exits"`
	ExitsV2     map[string]GMCPRoomModule_Payload_Contents_ExitInfo `json:"exitsv2"`
	Details     []string                                            `json:"details"`
	Contents    GMCPRoomModule_Payload_Contents                     `json:"Contents"`
}

type GMCPRoomModule_Payload_Contents_ExitInfo struct {
	RoomId  int      `json:"num"`
	DeltaX  int      `json:"dx"`
	DeltaY  int      `json:"dy"`
	DeltaZ  int      `json:"dz"`
	Details []string `json:"details"`
}

// /////////////////
// Room.Contents
// /////////////////
type GMCPRoomModule_Payload_Contents struct {
	Players    []GMCPRoomModule_Payload_Contents_Character `json:"Players"`
	Npcs       []GMCPRoomModule_Payload_Contents_Character `json:"Npcs"`
	Items      []GMCPRoomModule_Payload_Contents_Item      `json:"Items"`
	Containers []GMCPRoomModule_Payload_Contents_Container `json:"Containers"`
}

type GMCPRoomModule_Payload_Contents_Character struct {
	Id         string   `json:"id"`
	Name       string   `json:"name"`
	Adjectives []string `json:"adjectives"`
	Aggro      bool     `json:"aggro"`
	QuestFlag  bool     `json:"quest_flag"`
}

type GMCPRoomModule_Payload_Contents_Item struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	QuestFlag bool   `json:"quest_flag"`
}

type GMCPRoomModule_Payload_Contents_Container struct {
	Name         string `json:"name"`
	Locked       bool   `json:"locked"`
	HasKey       bool   `json:"haskey"`
	HasPickCombo bool   `json:"haspickcombo"`
	Usable       bool   `json:"usable"`
}
