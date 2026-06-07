package gmcp

import (
	"sort"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// pendingSpawnRefresh tracks userIds that spawned but may not have received
// their initial Char.Automation push yet (e.g. a web client whose websocket
// GMCP frame raced the login burst). The next NewRound re-pushes for these
// users once and clears them, guaranteeing delivery after the connection is
// fully GMCP-ready. Mirrors the single-shot-plus-NewRound pattern in
// gmcp.World.go (World.Time). Steady-state cost is zero once drained.
var (
	pendingSpawnRefresh   = map[int]struct{}{}
	pendingSpawnRefreshMu sync.Mutex
)

func init() {

	g := GMCPAutomationModule{
		plug: plugins.New(`gmcp.Automation`, `1.0`),
	}

	// Push on login and whenever a macro/alias (and later tick/trigger) changes.
	events.RegisterListener(events.PlayerSpawn{}, g.playerSpawnHandler)
	events.RegisterListener(events.AutomationChanged{}, g.automationChangedHandler)
	// Re-push once on the round following spawn so the initial login push lands
	// reliably even if the very first GMCP frame raced the connection setup.
	events.RegisterListener(events.NewRound{}, g.newRoundHandler)
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

	// Push now (covers telnet/Mudlet clients that have negotiated GMCP)...
	g.sendAutomation(evt.UserId)

	// ...and queue a one-shot re-push for the next round, which guarantees
	// delivery for web clients once the connection is settled.
	pendingSpawnRefreshMu.Lock()
	pendingSpawnRefresh[evt.UserId] = struct{}{}
	pendingSpawnRefreshMu.Unlock()

	return events.Continue
}

func (g GMCPAutomationModule) newRoundHandler(e events.Event) events.ListenerReturn {

	if _, typeOk := e.(events.NewRound); !typeOk {
		return events.Continue
	}

	pendingSpawnRefreshMu.Lock()
	if len(pendingSpawnRefresh) == 0 {
		pendingSpawnRefreshMu.Unlock()
		return events.Continue
	}
	userIds := make([]int, 0, len(pendingSpawnRefresh))
	for userId := range pendingSpawnRefresh {
		userIds = append(userIds, userId)
	}
	pendingSpawnRefresh = map[int]struct{}{}
	pendingSpawnRefreshMu.Unlock()

	for _, userId := range userIds {
		g.sendAutomation(userId)
	}

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
