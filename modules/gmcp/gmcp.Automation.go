package gmcp

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {

	g := GMCPAutomationModule{
		plug: plugins.New(`gmcp.Automation`, `1.0`),
	}

	// Push on login and whenever a macro/alias (and later tick/trigger) changes.
	events.RegisterListener(events.PlayerSpawn{}, g.playerSpawnHandler)
	events.RegisterListener(events.AutomationChanged{}, g.automationChangedHandler)
}

type GMCPAutomationModule struct {
	plug *plugins.Plugin
}

func (g GMCPAutomationModule) playerSpawnHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	g.sendAutomation(evt.UserId)

	return events.Continue
}

func (g GMCPAutomationModule) automationChangedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.AutomationChanged)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "AutomationChanged", "Actual Type", e.Type())
		return events.Cancel
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	g.sendAutomation(evt.UserId)

	return events.Continue
}

// sendAutomation builds and pushes the Char.Automation payload for a user.
func (g GMCPAutomationModule) sendAutomation(userId int) {

	user := users.GetByUserId(userId)
	if user == nil {
		return
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return
	}

	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Char.Automation`,
		Payload: buildAutomationPayload(user.Macros, user.Aliases),
	})
}

// buildAutomationPayload converts the user's macro and alias maps into the
// stable, sorted GMCP payload structure.
func buildAutomationPayload(macros map[string]string, aliases map[string]string) GMCPAutomation_Payload {

	payload := GMCPAutomation_Payload{
		Macros:  make([]GMCPAutomation_Macro, 0, len(macros)),
		Aliases: make([]GMCPAutomation_Alias, 0, len(aliases)),
	}

	macroKeys := make([]string, 0, len(macros))
	for k := range macros {
		macroKeys = append(macroKeys, k)
	}
	sort.Strings(macroKeys)
	for _, k := range macroKeys {
		payload.Macros = append(payload.Macros, GMCPAutomation_Macro{
			Key:      k,
			Commands: macros[k],
		})
	}

	aliasKeys := make([]string, 0, len(aliases))
	for k := range aliases {
		aliasKeys = append(aliasKeys, k)
	}
	sort.Strings(aliasKeys)
	for _, k := range aliasKeys {
		payload.Aliases = append(payload.Aliases, GMCPAutomation_Alias{
			Name:    k,
			Command: aliases[k],
		})
	}

	return payload
}

// /////////////////
// Char.Automation
// /////////////////
type GMCPAutomation_Macro struct {
	Key      string `json:"key"`
	Commands string `json:"commands"`
}

type GMCPAutomation_Alias struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type GMCPAutomation_Payload struct {
	Macros  []GMCPAutomation_Macro `json:"macros"`
	Aliases []GMCPAutomation_Alias `json:"aliases"`
	// Ticks/Triggers added in Phases 2-3.
}
