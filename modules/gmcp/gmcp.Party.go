package gmcp

import (
	"math"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
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
	g := GMCPPartyModule{
		plug: plugins.New(`gmcp.Comm`, `1.0`),
	}

	events.RegisterListener(events.RoomChange{}, g.roomChangeHandler)
	events.RegisterListener(events.PartyUpdated{}, g.onPartyChange)
	events.RegisterListener(PartyUpdateVitals{}, g.onUpdateVitals)
	events.RegisterListener(events.CharacterVitalsChanged{}, g.vitalsChangedHandler)

}

type GMCPPartyModule struct {
	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
	plug *plugins.Plugin
}

// This is a uniqu event so that multiple party members moving thorugh an area all at once don't queue up a bunch for just one party
type PartyUpdateVitals struct {
	LeaderId int
}

func (g PartyUpdateVitals) Type() string     { return `PartyUpdateVitals` }
func (g PartyUpdateVitals) UniqueID() string { return `PartyVitals-` + strconv.Itoa(g.LeaderId) }

func (g *GMCPPartyModule) vitalsChangedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.CharacterVitalsChanged)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "CharacterVitalsChanged", "Actual Type", e.Type())
		return events.Cancel
	}

	party := parties.Get(evt.UserId)
	if party == nil {
		return events.Continue
	}

	events.AddToQueue(PartyUpdateVitals{
		LeaderId: party.LeaderUserId,
	})

	return events.Continue
}

func (g *GMCPPartyModule) roomChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.RoomChange)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "RoomChange", "Actual Type", e.Type())
		return events.Cancel
	}

	if evt.MobInstanceId > 0 {
		return events.Continue
	}

	party := parties.Get(evt.UserId)
	if party == nil {
		return events.Continue
	}

	events.AddToQueue(PartyUpdateVitals{
		LeaderId: party.LeaderUserId,
	})

	return events.Continue
}

func (g *GMCPPartyModule) onUpdateVitals(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(PartyUpdateVitals)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PartyUpdateVitals", "Actual Type", e.Type())
		return events.Cancel
	}

	party := parties.Get(evt.LeaderId)
	if party == nil {
		return events.Cancel
	}

	payload, moduleName := g.GetPartyNode(party, `Party.Vitals`)

	for _, userId := range party.GetMembers() {

		events.AddToQueue(GMCPOut{
			UserId:  userId,
			Module:  moduleName,
			Payload: payload,
		})

	}

	return events.Continue
}

func (g *GMCPPartyModule) onPartyChange(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PartyUpdated)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PartyUpdated", "Actual Type", e.Type())
		return events.Cancel
	}

	if len(evt.UserIds) == 0 {
		return events.Cancel
	}

	var party *parties.Party
	for _, uId := range evt.UserIds {
		if party = parties.Get(uId); party != nil {
			break
		}
	}

	payload, moduleName := g.GetPartyNode(party, `Party`)

	inParty := map[int]struct{}{}
	if party != nil {
		for _, uId := range party.GetMembers() {
			inParty[uId] = struct{}{}
		}
		for _, uId := range party.GetInvited() {
			inParty[uId] = struct{}{}
		}
	}

	for _, userId := range evt.UserIds {

		if _, ok := inParty[userId]; ok {

			events.AddToQueue(GMCPOut{
				UserId:  userId,
				Module:  moduleName,
				Payload: payload,
			})

		} else {

			events.AddToQueue(GMCPOut{
				UserId:  userId,
				Module:  `Party`,
				Payload: GMCPPartyModule_Payload{},
			})

		}

	}

	return events.Continue
}

func (g *GMCPPartyModule) GetPartyNode(party *parties.Party, gmcpModule string) (data any, moduleName string) {

	all := gmcpModule == `Party`

	if party == nil {
		return GMCPPartyModule_Payload_Vitals{}, `Party`
	}

	partyPayload := GMCPPartyModule_Payload{
		Leader:  `None`,
		Members: []GMCPPartyModule_Payload_User{},
		Invited: []GMCPPartyModule_Payload_User{},
		Vitals:  map[string]GMCPPartyModule_Payload_Vitals{},
	}

	roomTitles := map[int]string{}

	for _, uId := range party.GetMembers() {

		if user := users.GetByUserId(uId); user != nil {

			hPct := int(math.Floor((float64(user.Character.Health) / float64(user.Character.HealthMax.Value)) * 100))
			if hPct < 0 {
				hPct = 0
			} else if hPct > 100 {
				hPct = 100
			}

			sPct := 0
			if user.Character.StaminaMax.Value > 0 {
				sPct = int(math.Floor((float64(user.Character.Stamina) / float64(user.Character.StaminaMax.Value)) * 100))
				if sPct < 0 {
					sPct = 0
				} else if sPct > 100 {
					sPct = 100
				}
			}

			cPct := 0
			if user.Character.ConvictionMax.Value > 0 {
				cPct = int(math.Floor((float64(user.Character.Conviction) / float64(user.Character.ConvictionMax.Value)) * 100))
				if cPct < 0 {
					cPct = 0
				} else if cPct > 100 {
					cPct = 100
				}
			}

			roomTitle, ok := roomTitles[user.Character.RoomId]
			if !ok {
				if uRoom := rooms.LoadRoom(user.Character.RoomId); uRoom != nil {
					roomTitle = uRoom.Title
					roomTitles[user.Character.RoomId] = roomTitle
				}
			}

			partyPayload.Vitals[user.Character.Name] = GMCPPartyModule_Payload_Vitals{
				HealthPercent:     hPct,
				StaminaPercent:    sPct,
				ConvictionPercent: cPct,
				Location:          roomTitle,
			}

			if gmcpModule == `Party.Vitals` {
				continue
			}

			if user.UserId == party.LeaderUserId {
				partyPayload.Leader = user.Character.Name
			}

			partyPayload.Members = append(partyPayload.Members,
				GMCPPartyModule_Payload_User{
					Name:     user.Character.Name,
					Status:   `In Party`,
					Position: party.GetRank(user.UserId),
				},
			)

		}

	}

	for _, uId := range party.GetInvited() {

		if user := users.GetByUserId(uId); user != nil {

			partyPayload.Vitals[user.Character.Name] = GMCPPartyModule_Payload_Vitals{
				HealthPercent: 0,
				Location:      ``,
			}

			if gmcpModule == `Party.Vitals` {
				continue
			}

			partyPayload.Invited = append(partyPayload.Invited,
				GMCPPartyModule_Payload_User{
					Name:     user.Character.Name,
					Status:   `Invited`,
					Position: ``,
				},
			)

		}

	}

	if gmcpModule == `Party.Vitals` {
		return partyPayload.Vitals, `Party.Vitals`
	}

	// If we reached this point and Char wasn't requested, we have a problem.
	if !all {
		mudlog.Error(`gmcp.Room`, `error`, `Bad module requested`, `module`, gmcpModule)
	}

	return partyPayload, `Party`

}

type GMCPPartyModule_Payload struct {
	Leader  string
	Members []GMCPPartyModule_Payload_User
	Invited []GMCPPartyModule_Payload_User
	Vitals  map[string]GMCPPartyModule_Payload_Vitals
}

type GMCPPartyModule_Payload_User struct {
	Name     string `json:"name"`
	Status   string `json:"status"`   // party/leader/invited
	Position string `json:"position"` // frontrank/middle/backrank
}

type GMCPPartyModule_Payload_Vitals struct {
	HealthPercent     int    `json:"health"`     // 1 = 1%, 23 = 23% etc.
	StaminaPercent    int    `json:"stamina"`    // 1 = 1%, 23 = 23% etc.
	ConvictionPercent int    `json:"conviction"` // 1 = 1%, 23 = 23% etc.
	Location          string `json:"location"`   // Title of room they are in
}
