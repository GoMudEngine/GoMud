package gmcp

import (
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {

	g := GMCPCommandsModule{
		plug: plugins.New(`gmcp.Commands`, `1.0`),
	}

	events.RegisterListener(events.PlayerSpawn{}, g.playerSpawnHandler)
	events.RegisterListener(events.CharacterVitalsChanged{}, g.vitalsChangedHandler)
	events.RegisterListener(events.CharacterChanged{}, g.charChangedHandler)
	events.RegisterListener(GMCPCommandsUpdate{}, g.buildAndSendGMCPPayload)
}

type GMCPCommandsModule struct {
	plug *plugins.Plugin
}

// Tell the system a wish to send specific GMCP Commands update data
type GMCPCommandsUpdate struct {
	UserId     int
	Identifier string
}

func (g GMCPCommandsUpdate) Type() string { return `GMCPCommandsUpdate` }

func (g GMCPCommandsModule) playerSpawnHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCommandsUpdate{
		UserId:     evt.UserId,
		Identifier: `Commands.List, Commands.State`,
	})

	return events.Continue
}

func (g GMCPCommandsModule) vitalsChangedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.CharacterVitalsChanged)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCommandsUpdate{
		UserId:     evt.UserId,
		Identifier: `Commands.State`,
	})

	return events.Continue
}

func (g GMCPCommandsModule) charChangedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.CharacterChanged)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCommandsUpdate{
		UserId:     evt.UserId,
		Identifier: `Commands.State`,
	})

	return events.Continue
}

func (g GMCPCommandsModule) buildAndSendGMCPPayload(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(GMCPCommandsUpdate)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "GMCPCommandsUpdate", "Actual Type", e.Type())
		return events.Cancel
	}

	if evt.UserId < 1 {
		return events.Continue
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return events.Cancel
	}

	for _, identifier := range strings.Split(evt.Identifier, `,`) {

		identifier = strings.TrimSpace(identifier)

		switch identifier {

		case `Commands.List`:
			events.AddToQueue(GMCPOut{
				UserId:  evt.UserId,
				Module:  `Commands.List`,
				Payload: buildCommandsList(),
			})

		case `Commands.State`:
			events.AddToQueue(GMCPOut{
				UserId:  evt.UserId,
				Module:  `Commands.State`,
				Payload: buildCommandsState(user),
			})

		default:
			mudlog.Error(`gmcp.Commands`, `error`, `Bad module requested`, `module`, identifier)
		}
	}

	return events.Continue
}

// buildCommandsList returns a sorted list of all non-admin commands with their flags.
func buildCommandsList() GMCPCommandsList {

	registry := usercommands.GetCommandRegistry()

	entries := make(GMCPCommandsList, 0, len(registry))
	for name, info := range registry {
		if info.AdminOnly {
			continue
		}
		entries = append(entries, GMCPCommandEntry{
			Name:              name,
			AllowedInCombat:   info.AllowedInCombat,
			AllowedWhenDowned: info.AllowedWhenDowned,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries
}

// buildCommandsState derives the current mode, active cooldowns, and blocked commands.
func buildCommandsState(user *users.UserRecord) GMCPCommandsState {

	// Determine mode
	mode := `exploration`
	if user.Character.Health <= 0 {
		mode = `downed`
	} else if user.Character.Aggro != nil {
		mode = `combat`
	}

	// Convert numeric cooldowns to qualitative labels
	cooldowns := map[string]string{}
	for name, rounds := range user.Character.Cooldowns {
		switch {
		case rounds <= 1:
			cooldowns[name] = `almost_ready`
		case rounds <= 5:
			cooldowns[name] = `cooling_down`
		default:
			cooldowns[name] = `not_ready`
		}
	}

	// Derive blocked commands from active conditions and position
	blocked := []string{}
	if user.Character.HasCondition(characters.ConditionGrappleController) {
		blocked = append(blocked, `flee`)
	}
	if user.Character.CombatPosition == characters.PositionGrounded {
		blocked = append(blocked, `stand`)
	}

	return GMCPCommandsState{
		Mode:      mode,
		Cooldowns: cooldowns,
		Blocked:   blocked,
	}
}

// /////////////////
// Commands.List
// /////////////////
type GMCPCommandEntry struct {
	Name              string `json:"name"`
	AllowedInCombat   bool   `json:"combat"`
	AllowedWhenDowned bool   `json:"downed"`
}

type GMCPCommandsList []GMCPCommandEntry

// /////////////////
// Commands.State
// /////////////////
type GMCPCommandsState struct {
	Mode      string            `json:"mode"`
	Cooldowns map[string]string `json:"cooldowns"`
	Blocked   []string          `json:"blocked"`
}
