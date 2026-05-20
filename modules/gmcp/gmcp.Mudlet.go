package gmcp

import (
	"embed"
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var (
	//go:embed files/*
	files embed.FS
)

// MudletConfig holds the configuration for Mudlet clients
type MudletConfig struct {
	// Mapper configuration
	MapperVersion string `json:"mapper_version" yaml:"mapper_version"`
	MapperURL     string `json:"mapper_url" yaml:"mapper_url"`

	// UI configuration
	UIVersion string `json:"ui_version" yaml:"ui_version"`
	UIURL     string `json:"ui_url" yaml:"ui_url"`

	// Map data configuration
	MapVersion string `json:"map_version" yaml:"map_version"`
	MapURL     string `json:"map_url" yaml:"map_url"`

	// Discord Rich Presence configuration
	DiscordApplicationID string `json:"discord_application_id" yaml:"discord_application_id"`
	DiscordInviteURL     string `json:"discord_invite_url" yaml:"discord_invite_url"`
	DiscordLargeImageKey string `json:"discord_large_image_key" yaml:"discord_large_image_key"`
	DiscordDetails       string `json:"discord_details" yaml:"discord_details"`
	DiscordState         string `json:"discord_state" yaml:"discord_state"`
	DiscordSmallImageKey string `json:"discord_small_image_key" yaml:"discord_small_image_key"`
}

// GMCPMudletModule handles Mudlet-specific GMCP functionality
type GMCPMudletModule struct {
	plug        *plugins.Plugin
	config      MudletConfig
	mudletUsers map[int]bool // Track which users are using Mudlet clients
}

// GMCPMudletDetected is an event fired when a Mudlet client is detected
type GMCPMudletDetected struct {
	ConnectionId uint64
	UserId       int
}

func (g GMCPMudletDetected) Type() string { return `GMCPMudletDetected` }

// GMCPDiscordStatusRequest is an event fired when a client requests Discord status information
type GMCPDiscordStatusRequest struct {
	UserId int
}

func (g GMCPDiscordStatusRequest) Type() string { return `GMCPDiscordStatusRequest` }

// GMCPDiscordMessage is an event fired when a client sends a Discord-related GMCP message
type GMCPDiscordMessage struct {
	ConnectionId uint64
	Command      string
	Payload      []byte
}

func (g GMCPDiscordMessage) Type() string { return `GMCPDiscordMessage` }

func init() {
	// Create module with basic structure
	g := GMCPMudletModule{
		plug:        plugins.New(`gmcp.Mudlet`, `1.0`),
		mudletUsers: make(map[int]bool),
	}

	// Attach filesystem with proper error handling
	if err := g.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}

	// Register callbacks for load/save
	g.plug.Callbacks.SetOnLoad(g.load)
	g.plug.Callbacks.SetOnSave(g.save)

	// Register event listeners
	events.RegisterListener(events.PlayerSpawn{}, g.playerSpawnHandler)
	events.RegisterListener(events.PlayerDespawn{}, g.playerDespawnHandler)
	events.RegisterListener(GMCPMudletDetected{}, g.mudletDetectedHandler)
	events.RegisterListener(GMCPDiscordStatusRequest{}, g.discordStatusRequestHandler)
	events.RegisterListener(GMCPDiscordMessage{}, g.discordMessageHandler)
	events.RegisterListener(events.RoomChange{}, g.roomChangeHandler)
	events.RegisterListener(events.PartyUpdated{}, g.partyUpdateHandler)

	// Register the Mudlet-specific user commands
	g.plug.AddUserCommand("mudletmap", g.sendMapCommand, true, false)
	g.plug.AddUserCommand("mudletui", g.sendUICommand, false, false)
	g.plug.AddUserCommand("checkclient", g.checkClientCommand, true, false)
	g.plug.AddUserCommand("discord", g.discordCommand, true, false)
}

// Helper function to load a config string from the plugin's configuration
func loadConfigString(p *plugins.Plugin, key string) string {
	if val, ok := p.Config.Get(key).(string); ok {
		return val
	}
	return ""
}

// load handles loading configuration from the plugin's storage
func (g *GMCPMudletModule) load() {
	// Load config values directly from embedded config or overrides
	g.config.MapperVersion = loadConfigString(g.plug, "mapper_version")
	g.config.MapperURL = loadConfigString(g.plug, "mapper_url")
	g.config.UIVersion = loadConfigString(g.plug, "ui_version")
	g.config.UIURL = loadConfigString(g.plug, "ui_url")
	g.config.MapVersion = loadConfigString(g.plug, "map_version")
	g.config.MapURL = loadConfigString(g.plug, "map_url")
	g.config.DiscordApplicationID = loadConfigString(g.plug, "discord_application_id")
	g.config.DiscordInviteURL = loadConfigString(g.plug, "discord_invite_url")
	g.config.DiscordLargeImageKey = loadConfigString(g.plug, "discord_large_image_key")
	g.config.DiscordDetails = loadConfigString(g.plug, "discord_details")
	g.config.DiscordState = loadConfigString(g.plug, "discord_state")
	g.config.DiscordSmallImageKey = loadConfigString(g.plug, "discord_small_image_key")
}

// save handles saving configuration to the plugin's storage
func (g *GMCPMudletModule) save() {
	g.plug.WriteStruct(`mudlet_config`, g.config)
}

// Helper function to check if a user is using a Mudlet client
func (g *GMCPMudletModule) isMudletClient(userId int) bool {
	if userId < 1 {
		return false
	}

	// First check our cache of known Mudlet users
	if known, ok := g.mudletUsers[userId]; ok {
		return known
	}

	// If not in cache, check the connection
	connId := users.GetConnectionId(userId)
	if connId == 0 {
		return false
	}

	// Check the cache to see if this is a Mudlet client
	if gmcpData, ok := gmcpModule.cache.Get(connId); ok && gmcpData.Client.IsMudlet {
		// Store for future reference
		g.mudletUsers[userId] = true
		return true
	}

	return false
}

// Helper function to get user config option with default boolean value
func getUserBoolOption(user *users.UserRecord, key string, defaultValue bool) bool {
	val := user.GetConfigOption(key)
	if val == nil {
		return defaultValue
	}
	if boolVal, ok := val.(bool); ok {
		return boolVal
	}
	return defaultValue
}

// Helper to send GMCP event
func sendGMCP(userId int, module string, payload interface{}) {
	if userId < 1 {
		return
	}
	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  module,
		Payload: payload,
	})
}

// Helper function to create and send Discord Info message
func (g *GMCPMudletModule) sendDiscordInfo(userId int) {
	if userId < 1 {
		return
	}

	user := users.GetByUserId(userId)
	if user == nil {
		return
	}

	// Check if Discord.Info is enabled
	if !getUserBoolOption(user, "discord_enable_info", true) {
		mudlog.Debug("GMCP", "type", "Mudlet", "action", "Discord.Info package sending disabled for user", "userId", userId)
		return
	}

	// Send Discord Info payload
	payload := struct {
		ApplicationID string `json:"applicationid"`
		InviteURL     string `json:"inviteurl"`
	}{
		ApplicationID: g.config.DiscordApplicationID,
		InviteURL:     g.config.DiscordInviteURL,
	}

	sendGMCP(userId, "External.Discord.Info", payload)
	mudlog.Debug("GMCP", "type", "Mudlet", "action", "Sent Discord Info", "userId", userId)
}

// sendDiscordStatus sends the current Discord status information
func (g *GMCPMudletModule) sendDiscordStatus(userId int) {
	if userId < 1 {
		return
	}

	// Get the user record
	user := users.GetByUserId(userId)
	if user == nil {
		mudlog.Error("GMCP", "type", "Mudlet", "action", "Failed to get user record for Discord status", "userId", userId)
		return
	}

	// Check if Discord.Status is enabled
	if !getUserBoolOption(user, "discord_enable_status", true) {
		mudlog.Debug("GMCP", "type", "Mudlet", "action", "Discord.Status package sending disabled for user", "userId", userId)
		return
	}

	// Get the current room
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		mudlog.Error("GMCP", "type", "Mudlet", "action", "Failed to get room for Discord status", "userId", userId, "roomId", user.Character.RoomId)
		return
	}

	// Check display preferences
	showArea := getUserBoolOption(user, "discord_show_area", true)
	showParty := getUserBoolOption(user, "discord_show_party", true)
	showName := getUserBoolOption(user, "discord_show_name", true)

	// Build the details string based on preferences
	detailsStr := g.config.DiscordDetails
	if showName {
		detailsStr = user.Character.Name
	}

	// Create Discord Status payload
	payload := struct {
		Details       string `json:"details"`
		State         string `json:"state"`
		Game          string `json:"game"`
		LargeImageKey string `json:"large_image_key"`
		SmallImageKey string `json:"small_image_key"`
		StartTime     int64  `json:"starttime"`
		PartySize     int    `json:"partysize,omitempty"`
		PartyMax      int    `json:"partymax,omitempty"`
	}{
		Details:       detailsStr,
		State:         g.config.DiscordState,
		Game:          configs.GetServerConfig().MudName.String(),
		LargeImageKey: g.config.DiscordLargeImageKey,
		SmallImageKey: g.config.DiscordSmallImageKey,
		StartTime:     user.GetConnectTime().Unix(),
	}

	// Show area if enabled
	if showArea {
		payload.State = fmt.Sprintf("Exploring %s", room.Zone)
	}

	// Show party info if enabled and in a party
	if party := parties.Get(userId); party != nil && showParty {
		payload.PartySize = len(party.GetMembers())
		payload.PartyMax = 10
		if showArea {
			payload.State = fmt.Sprintf("Group in %s", room.Zone)
		} else {
			payload.State = "In group"
		}
	}

	// Send the Discord Status message
	sendGMCP(userId, "External.Discord.Status", payload)
	mudlog.Debug("GMCP", "type", "Mudlet", "action", "Sent Discord status update", "userId", userId, "zone", room.Zone)
}

// Send empty Discord status to clear it
func (g *GMCPMudletModule) clearDiscordStatus(userId int) {
	payload := struct {
		Details       string `json:"details"`
		State         string `json:"state"`
		Game          string `json:"game"`
		LargeImageKey string `json:"large_image_key"`
		SmallImageKey string `json:"small_image_key"`
	}{
		Details:       "",
		State:         "",
		Game:          "",
		LargeImageKey: "",
		SmallImageKey: "",
	}

	sendGMCP(userId, "External.Discord.Status", payload)
}

// Send Mudlet map configuration
func (g *GMCPMudletModule) sendMudletMapConfig(userId int) {
	if userId < 1 {
		return
	}

	mapConfig := map[string]string{
		"url": g.config.MapURL,
	}

	sendGMCP(userId, "Client.Map", mapConfig)
	mudlog.Debug("GMCP", "type", "Mudlet", "action", "Sent Mudlet map config", "userId", userId)
}

// Send Mudlet UI package installation message
func (g *GMCPMudletModule) sendMudletUIInstall(userId int) {
	if userId < 1 {
		return
	}

	payload := struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}{
		Version: g.config.UIVersion,
		URL:     g.config.UIURL,
	}

	sendGMCP(userId, "Client.GUI", payload)
	mudlog.Debug("GMCP", "type", "Mudlet", "action", "Sent Mudlet UI install config", "userId", userId)
}

// Send Mudlet UI package removal message
func (g *GMCPMudletModule) sendMudletUIRemove(userId int) {
	if userId < 1 {
		return
	}

	payload := struct {
		GoMudUI string `json:"gomudui"`
	}{
		GoMudUI: "remove",
	}

	sendGMCP(userId, "Client.GUI", payload)
	mudlog.Debug("GMCP", "type", "Mudlet", "action", "Sent Mudlet UI remove command", "userId", userId)
}

// Send Mudlet UI package update message
func (g *GMCPMudletModule) sendMudletUIUpdate(userId int) {
	if userId < 1 {
		return
	}

	payload := struct {
		GoMudUI string `json:"gomudui"`
	}{
		GoMudUI: "update",
	}

	sendGMCP(userId, "Client.GUI", payload)
	mudlog.Debug("GMCP", "type", "Mudlet", "action", "Sent Mudlet UI update command", "userId", userId)
}

// Send mapper configuration to Mudlet client
func (g *GMCPMudletModule) sendMudletConfig(userId int) {
	if userId < 1 {
		return
	}

	// Send mapper info
	payload := struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}{
		Version: g.config.MapperVersion,
		URL:     g.config.MapperURL,
	}
	sendGMCP(userId, "Client.GUI", payload)

	// Get the user record
	user := users.GetByUserId(userId)
	if user == nil {
		return
	}

	// Send Discord info if enabled
	g.sendDiscordInfo(userId)

	// Send Discord status
	g.sendDiscordStatus(userId)

	mudlog.Info("GMCP", "type", "Mudlet", "action", "Sent Mudlet package config", "userId", userId)
}

// playerSpawnHandler handles when a player connects
func (g *GMCPMudletModule) playerSpawnHandler(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PlayerSpawn", "Actual Type", e.Type())
		return events.Cancel
	}

	// Check if the client is Mudlet
	if gmcpData, ok := gmcpModule.cache.Get(evt.ConnectionId); ok && gmcpData.Client.IsMudlet {
		// Send Mudlet-specific GMCP
		g.sendMudletConfig(evt.UserId)
	}

	return events.Continue
}

// playerDespawnHandler handles when a player disconnects
func (g *GMCPMudletModule) playerDespawnHandler(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.PlayerDespawn)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PlayerDespawn", "Actual Type", e.Type())
		return events.Cancel
	}

	// Clean up the mudletUsers map entry for this user
	if evt.UserId > 0 {
		delete(g.mudletUsers, evt.UserId)
		mudlog.Debug("GMCP", "type", "Mudlet", "action", "Cleaned up Mudlet user entry", "userId", evt.UserId)
	}

	return events.Continue
}

// mudletDetectedHandler handles when a Mudlet client is detected
func (g *GMCPMudletModule) mudletDetectedHandler(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPMudletDetected)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "GMCPMudletDetected", "Actual Type", e.Type())
		return events.Cancel
	}

	if evt.UserId > 0 {
		g.sendMudletConfig(evt.UserId)
	}

	return events.Continue
}

// discordStatusRequestHandler handles Discord status requests
func (g *GMCPMudletModule) discordStatusRequestHandler(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPDiscordStatusRequest)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "GMCPDiscordStatusRequest", "Actual Type", e.Type())
		return events.Cancel
	}

	// Send both Discord info and status
	g.sendDiscordInfo(evt.UserId)
	g.sendDiscordStatus(evt.UserId)

	mudlog.Info("GMCP", "type", "Mudlet", "action", "Processed Discord status request", "userId", evt.UserId)
	return events.Continue
}

// discordMessageHandler handles Discord-related GMCP messages
func (g *GMCPMudletModule) discordMessageHandler(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPDiscordMessage)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "GMCPDiscordMessage", "Actual Type", e.Type())
		return events.Cancel
	}

	// Find the user ID for this connection
	userId := 0
	for _, user := range users.GetAllActiveUsers() {
		if user.ConnectionId() == evt.ConnectionId {
			userId = user.UserId
			break
		}
	}

	if userId == 0 {
		return events.Cancel
	}

	// Log the message
	mudlog.Info("Mudlet GMCP Discord", "type", evt.Command, "userId", userId, "payload", string(evt.Payload))

	// Handle different Discord commands
	switch evt.Command {
	case "Hello":
		g.sendDiscordInfo(userId)
	case "Get":
		user := users.GetByUserId(userId)
		if user != nil && user.Character != nil {
			events.AddToQueue(GMCPDiscordStatusRequest{
				UserId: userId,
			})
		}
	}

	return events.Continue
}

// roomChangeHandler updates Discord status when players change areas
func (g *GMCPMudletModule) roomChangeHandler(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.RoomChange)
	if !typeOk {
		return events.Cancel
	}

	// Only handle player movements (not mobs)
	if evt.UserId == 0 || evt.MobInstanceId > 0 {
		return events.Continue
	}

	// Check if this is a Mudlet client
	if !g.isMudletClient(evt.UserId) {
		return events.Continue
	}

	// Load rooms and check for zone change
	oldRoom := rooms.LoadRoom(evt.FromRoomId)
	newRoom := rooms.LoadRoom(evt.ToRoomId)
	if oldRoom == nil || newRoom == nil {
		return events.Continue
	}

	// Update Discord status on zone change
	if oldRoom.Zone != newRoom.Zone {
		g.sendDiscordStatus(evt.UserId)
	}

	return events.Continue
}

// partyUpdateHandler updates Discord status for party members
func (g *GMCPMudletModule) partyUpdateHandler(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.PartyUpdated)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PartyUpdated", "Actual Type", e.Type())
		return events.Cancel
	}

	// Update Discord status for all Mudlet users in the party
	for _, userId := range evt.UserIds {
		if g.isMudletClient(userId) {
			g.sendDiscordStatus(userId)
		}
	}

	return events.Continue
}

// Helper function for handling command toggles
func (g *GMCPMudletModule) handleToggleCommand(user *users.UserRecord, settingName string, value bool, enableMsg string, disableMsg string) {
	user.SetConfigOption(settingName, value)
	if value {
		user.SendText(messaging.CategorySystem, "\n<ansi fg=\"green\">" + enableMsg + "</ansi>\n")
	} else {
		user.SendText(messaging.CategorySystem, "\n<ansi fg=\"yellow\">" + disableMsg + "</ansi>\n")
	}

	// Update Discord status if this was a Discord-related setting
	if strings.HasPrefix(settingName, "discord_") {
		g.sendDiscordStatus(user.UserId)
	}
}

// sendUICommand handles UI-related commands
func (g *GMCPMudletModule) sendUICommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	// Only proceed if client is Mudlet
	connId := user.ConnectionId()
	if gmcpData, ok := gmcpModule.cache.Get(connId); !ok || !gmcpData.Client.IsMudlet {
		user.SendText(messaging.CategorySystem, "\n<ansi fg=\"red\">This command is only available for Mudlet clients.</ansi> You are currently using: " + gmcpData.Client.Name + "\n")
		return true, nil
	}

	// Process arguments
	args := strings.Fields(rest)
	if len(args) == 0 {
		// No arguments - show status and help
		mudName := configs.GetServerConfig().MudName.String()

		// Check prompt display status
		var promptStatus string
		if getUserBoolOption(user, "mudlet_ui_prompt_disabled", false) {
			promptStatus = "<ansi fg=\"red\">HIDDEN</ansi>"
		} else {
			promptStatus = "<ansi fg=\"green\">ENABLED</ansi>"
		}

		user.SendText(messaging.CategorySystem, "\n<ansi fg=\"cyan-bold\">" + mudName + " Mudlet UI Management</ansi>\n")
		user.SendText(messaging.CategorySystem, "<ansi fg=\"yellow-bold\">Status:</ansi>\n")
		user.SendText(messaging.CategorySystem, "  Login message display: " + promptStatus + "\n")
		user.SendText(messaging.CategorySystem, "<ansi fg=\"yellow-bold\">Available Commands:</ansi>\n")
		user.SendText(messaging.CategorySystem, "  <ansi fg=\"command\">mudletui install</ansi> - Install the Mudlet UI package\n")
		user.SendText(messaging.CategorySystem, "  <ansi fg=\"command\">mudletui remove</ansi>  - Remove the Mudlet UI package\n")
		user.SendText(messaging.CategorySystem, "  <ansi fg=\"command\">mudletui update</ansi>  - Manually check for updates to the Mudlet UI package\n")
		user.SendText(messaging.CategorySystem, "  <ansi fg=\"command\">mudletui hide</ansi>    - Hide login messages\n")
		user.SendText(messaging.CategorySystem, "  <ansi fg=\"command\">mudletui show</ansi>    - Enable login messages\n\n")
		user.SendText(messaging.CategorySystem, "For more information, type <ansi fg=\"command\">help mudletui</ansi>\n")
		return true, nil
	}

	// Handle specific commands
	switch args[0] {
	case "install":
		g.sendMudletUIInstall(user.UserId)
		user.SetConfigOption("mudlet_ui_prompt_disabled", true)
		user.SendText(messaging.CategorySystem, "\n<ansi fg=\"green\">UI installation package sent to your Mudlet client.</ansi> If it doesn't install automatically, you may need to accept the installation prompt in Mudlet.\n")

	case "remove":
		g.sendMudletUIRemove(user.UserId)
		user.SendText(messaging.CategorySystem, "\n<ansi fg=\"yellow\">UI removal command sent to your Mudlet client.</ansi>\n")

	case "update":
		g.sendMudletUIUpdate(user.UserId)
		user.SendText(messaging.CategorySystem, "\n<ansi fg=\"cyan\">Manual UI update check sent to your Mudlet client.</ansi>\n")

	case "hide":
		g.handleToggleCommand(user, "mudlet_ui_prompt_disabled", true,
			"The Mudlet UI prompt has been hidden.",
			"")
		user.SendText(messaging.CategorySystem, "You can use <ansi fg=\"command\">mudletui show</ansi> in the future if you want to see the prompts again.\n")

	case "show":
		g.handleToggleCommand(user, "mudlet_ui_prompt_disabled", false,
			"The Mudlet UI prompt has been re-enabled.",
			"")
		user.SendText(messaging.CategorySystem, "You can use <ansi fg=\"command\">mudletui hide</ansi> in the future if you want to hide the prompts again.\n")

	default:
		user.SendText(messaging.CategorySystem, "\nUsage: mudletui install|remove|update|hide|show\n\nType '<ansi fg=\"command\">help mudletui</ansi>' for more information.\n")
	}

	return true, nil
}

// sendMapCommand sends map configuration
func (g *GMCPMudletModule) sendMapCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	// Only send if the client is Mudlet
	connId := user.ConnectionId()
	if gmcpData, ok := gmcpModule.cache.Get(connId); ok && gmcpData.Client.IsMudlet {
		g.sendMudletMapConfig(user.UserId)
		return true, nil
	}
	return false, nil
}

// checkClientCommand checks if client is Mudlet and shows info
func (g *GMCPMudletModule) checkClientCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	// Check if client is Mudlet
	connId := user.ConnectionId()
	if gmcpData, ok := gmcpModule.cache.Get(connId); ok && gmcpData.Client.IsMudlet {
		// Skip if prompt is disabled
		if getUserBoolOption(user, "mudlet_ui_prompt_disabled", false) {
			return true, nil
		}

		// Show Mudlet help
		user.SendText(messaging.CategorySystem, "\n\n<ansi fg=\"cyan-bold\">We have detected you are using Mudlet as a client.</ansi>\n")
		usercommands.Help("mudletui", user, room, flags)
	}
	return true, nil
}

// discordCommand handles Discord-related settings
func (g *GMCPMudletModule) discordCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	// Only proceed if client is Mudlet
	connId := user.ConnectionId()
	if gmcpData, ok := gmcpModule.cache.Get(connId); !ok || !gmcpData.Client.IsMudlet {
		user.SendText(messaging.CategorySystem, "\n<ansi fg=\"red\">This command is only available for Mudlet clients.</ansi> You are currently using: " + gmcpData.Client.Name + "\n")
		return true, nil
	}

	// Process arguments
	args := strings.Fields(rest)
	if len(args) == 0 {
		user.SendText(messaging.CategorySystem, "\nUsage: discord area on|off|party on|off|name on|off|level on|off|info on|off|status on|off\n")
		return true, nil
	}

	// Handle different settings
	if len(args) >= 2 {
		switch args[0] {
		case "area":
			if args[1] == "on" {
				g.handleToggleCommand(user, "discord_show_area", true, "Area display in Discord status enabled.", "")
			} else if args[1] == "off" {
				g.handleToggleCommand(user, "discord_show_area", false, "Area display in Discord status disabled.", "")
			} else {
				user.SendText(messaging.CategorySystem, "\nUsage: discord area on|off\n")
			}

		case "party":
			if args[1] == "on" {
				g.handleToggleCommand(user, "discord_show_party", true, "Party display in Discord status enabled.", "")
			} else if args[1] == "off" {
				g.handleToggleCommand(user, "discord_show_party", false, "Party display in Discord status disabled.", "")
			} else {
				user.SendText(messaging.CategorySystem, "\nUsage: discord party on|off\n")
			}

		case "name":
			if args[1] == "on" {
				g.handleToggleCommand(user, "discord_show_name", true, "Character name display in Discord status enabled.", "")
			} else if args[1] == "off" {
				g.handleToggleCommand(user, "discord_show_name", false, "Character name display in Discord status disabled.", "")
			} else {
				user.SendText(messaging.CategorySystem, "\nUsage: discord name on|off\n")
			}

		case "level":
			if args[1] == "on" {
				g.handleToggleCommand(user, "discord_show_level", true, "Level display in Discord status enabled.", "")
			} else if args[1] == "off" {
				g.handleToggleCommand(user, "discord_show_level", false, "Level display in Discord status disabled.", "")
			} else {
				user.SendText(messaging.CategorySystem, "\nUsage: discord level on|off\n")
			}

		case "info":
			if args[1] == "on" {
				g.handleToggleCommand(user, "discord_enable_info", true, "Discord.Info package sending enabled.", "")
				g.sendDiscordInfo(user.UserId)
			} else if args[1] == "off" {
				g.handleToggleCommand(user, "discord_enable_info", false, "Discord.Info package sending disabled.", "")
				// Send empty Discord.Info payload
				sendGMCP(user.UserId, "External.Discord.Info", struct {
					ApplicationID string `json:"applicationid"`
					InviteURL     string `json:"inviteurl"`
				}{
					ApplicationID: "",
					InviteURL:     "",
				})
			} else {
				user.SendText(messaging.CategorySystem, "\nUsage: discord info on|off\n")
			}

		case "status":
			if args[1] == "on" {
				g.handleToggleCommand(user, "discord_enable_status", true, "Discord.Status package sending enabled.", "")
				g.sendDiscordStatus(user.UserId)
			} else if args[1] == "off" {
				g.handleToggleCommand(user, "discord_enable_status", false, "Discord.Status package sending disabled.", "")
				g.clearDiscordStatus(user.UserId)
			} else {
				user.SendText(messaging.CategorySystem, "\nUsage: discord status on|off\n")
			}

		default:
			user.SendText(messaging.CategorySystem, "\nUsage: discord area on|off|party on|off|name on|off|level on|off|info on|off|status on|off\n")
		}
	} else {
		user.SendText(messaging.CategorySystem, "\nUsage: discord area on|off|party on|off|name on|off|level on|off|info on|off|status on|off\n")
	}

	return true, nil
}
