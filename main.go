//go:generate go run cmd/generate/module-imports.go
package main

import (
	"fmt"
	"net"
	"os"
	"path"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoMudEngine/GoMud/internal/audio"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/conversations"
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/flags"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/inputhandlers"
	"github.com/GoMudEngine/GoMud/internal/integrations/discord"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/language"
	"github.com/GoMudEngine/GoMud/internal/migration"
	"github.com/GoMudEngine/GoMud/internal/modmanager"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/version"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"

	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/suggestions"
	"github.com/GoMudEngine/GoMud/internal/telemetry"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/web"
	_ "github.com/GoMudEngine/GoMud/modules"
	textLang "golang.org/x/text/language"
)

// Version of the binary
// Should be kept in lockstep with github releases
// When updating this version:
// 1. Expect to update the github release version
// 2. Consider whether any migration code is needed for breaking changes, particularly in datafiles (see internal/migration)
const VERSION = "0.9.10"

var (
	sigChan            = make(chan os.Signal, 1)
	workerShutdownChan = make(chan bool, 1)

	serverAlive atomic.Bool

	worldManager = NewWorld(sigChan)

	// Start a pool of worker goroutines
	wg sync.WaitGroup
)

func main() {

	// Dispatch the module manager subcommand before any server initialisation.
	if len(os.Args) >= 2 && os.Args[1] == "module" {
		modmanager.Run(os.Args[2:])
		return
	}

	serverStartTime := time.Now()

	// Capture panic and write msg/stack to logs
	defer func() {
		if r := recover(); r != nil {
			mudlog.Error("PANIC", "error", r)
			s := string(debug.Stack())
			for _, str := range strings.Split(s, "\n") {
				mudlog.Error("PANIC", "stack", str)
			}
		}
	}()

	// Setup logging
	mudlog.SetupLogger(
		events.GetLogger(),
		os.Getenv(`LOG_LEVEL`),
		os.Getenv(`LOG_PATH`),
		os.Getenv(`LOG_NOCOLOR`) == ``,
	)

	flags.HandleFlags(VERSION)

	// Register copyover contributors (must happen before Restore is called).
	copyover.Register(connections.CopyoverContributor())
	copyover.Register(users.CopyoverContributor())
	copyover.Register(util.CopyoverContributor())
	copyover.Register(gametime.CopyoverContributor())
	copyover.Register(copyover.TokenContributor())

	// Wire up the reconnect-token issuer so that connections can issue tokens
	// for WebSocket clients without creating an import cycle.
	connections.IssueWebSocketReconnectToken = func(connectionId connections.ConnectionId) (string, error) {
		u := users.GetByConnectionId(connectionId)
		if u == nil {
			return "", fmt.Errorf("no user for connection %d", connectionId)
		}
		return copyover.IssueReconnectToken(u.UserId)
	}

	configs.ReloadConfig()
	c := configs.GetConfig()

	lastKnownVersion, err := version.Parse(string(configs.GetServerConfig().CurrentVersion))
	if err != nil {
		mudlog.Error("Versioning", "error", err)
		os.Exit(1)
	}

	currentVersion, _ := version.Parse(VERSION)

	// if no copyover FD, run any migrations
	if flags.CopyoverFd() < 0 {
		if err = migration.Run(lastKnownVersion, currentVersion); err != nil {
			mudlog.Error("migration.Run()", "error", err)
			os.Exit(1)
		}
	}

	// Default i18n localize folders
	if len(c.Translation.LanguagePaths) == 0 {
		c.Translation.LanguagePaths = []string{
			path.Join("_datafiles", "localize"),
			path.Join(c.FilePaths.DataFiles.String(), "localize"),
		}
	}

	mudlog.Info(`========================`)
	//
	mudlog.Info(`  _____             `)
	mudlog.Info(` / ____|            `)
	mudlog.Info(`| |  __  ___        `)
	mudlog.Info(`| | |_ |/ _ \       `)
	mudlog.Info(`| |__| | (_) |      `)
	mudlog.Info(` \_____|\___/       `)
	mudlog.Info(` __  __           _ `)
	mudlog.Info(`|  \/  |         | |`)
	mudlog.Info(`| \  / |_   _  __| |`)
	mudlog.Info(`| |\/| | | | |/ _' |`)
	mudlog.Info(`| |  | | |_| | (_| |`)
	mudlog.Info(`|_|  |_|\__,_|\__,_|`)

	//
	mudlog.Info(`========================`)
	//
	/*
		cfgData := c.AllConfigData()
		cfgKeys := make([]string, 0, len(cfgData))
		for k := range cfgData {
			cfgKeys = append(cfgKeys, k)
		}

		// sort the keys
		slices.Sort(cfgKeys)
		for _, k := range cfgKeys {
			mudlog.Info("Config", "name", k, "value", cfgData[k])
		}
		//
		mudlog.Info(`========================`)
	*/

	// Older versions of GoMud may not have this folder present.
	// Also deleting the folder is a quick way to reset instance state, so this corrects that if it happens.
	os.Mkdir(util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `rooms.instances`), os.ModeDir|0755)

	// Register the plugin filesystem with the template system
	templates.RegisterFS(plugins.GetPluginRegistry())
	items.RegisterFS(plugins.GetPluginRegistry())
	mutators.RegisterFS(plugins.GetPluginRegistry())
	buffs.RegisterFS(plugins.GetPluginRegistry())
	buffs.RegisterFlagFS(plugins.GetPluginRegistry())
	pets.RegisterFS(plugins.GetPluginRegistry())
	quests.RegisterFS(plugins.GetPluginRegistry())
	mobs.RegisterFS(plugins.GetPluginRegistry())
	conversations.RegisterFS(plugins.GetPluginRegistry())
	usercommands.AddFunctionExporter(plugins.GetPluginRegistry())
	users.AddFunctionExporter(plugins.GetPluginRegistry())
	usercommands.SetRoomTagProvider(plugins.GetRegisteredRoomTags)
	web.SetRoomTagProvider(plugins.GetRegisteredRoomTags)

	inputhandlers.AddIACHandler(plugins.GetPluginRegistry())
	inputhandlers.AddTextPrefixHandler(plugins.GetPluginRegistry())
	//
	// System Configurations
	runtime.GOMAXPROCS(int(c.Server.MaxCPUCores))

	// Validate chosen world:
	if err := util.ValidateWorldFiles(`_datafiles/world/default`, c.FilePaths.DataFiles.String()); err != nil {
		mudlog.Error("World Validation", "error", err)
		os.Exit(1)
	}

	language.InitTranslation(language.BundleCfg{
		DefaultLanguage: textLang.Make(c.Translation.DefaultLanguage.String()),
		Language:        textLang.Make(c.Translation.Language.String()),
		LanguagePaths:   c.Translation.LanguagePaths,
	})

	hooks.RegisterListeners()

	telemetry.Load(configs.GetFilePathsConfig().DataFiles.String())
	telemetry.RegisterListeners()

	// Discord integration
	if webhookUrl := string(c.Integrations.Discord.WebhookUrl); webhookUrl != "" {
		discord.Init(webhookUrl)
		mudlog.Info("Discord", "info", "integration is enabled")
	} else {
		mudlog.Warn("Discord", "info", "integration is disabled")
	}

	mudlog.Info(
		"Starting server",
		"name", string(c.Server.MudName),
	)

	mudlog.Info(`========================`)

	// Load all the data files up front.
	loadAllDataFiles(false)

	mudlog.Info(`========================`)

	mudlog.Info("Mapper", "status", "precaching")
	timeStart := time.Now()
	mapper.PreCacheMaps()
	mudlog.Info("Mapper", "status", "done", "time taken", time.Since(timeStart))

	mudlog.Info(`========================`)

	// Create the user index
	isCopyover := flags.CopyoverFd() >= 0

	if !isCopyover {
		timeStart := time.Now()
		idx := users.InitUserIndex()
		if !idx.Exists() {
			// Since it doesn't exist yet, that's a good indication we should do a quick format migration check
			users.DoUserMigrations()
		}
		if idx.IsUpToDate() {
			mudlog.Info("UserIndex", "info", "User index up to date.", "users", idx.GetMetaData().RecordCount, "time taken", time.Since(timeStart))
		} else {
			idx.Create()
			idx.Rebuild()
			mudlog.Info("UserIndex", "info", "User index recreated.", "users", idx.GetMetaData().RecordCount, "time taken", time.Since(timeStart))
		}
	}

	users.GetCharacterIndex().Rebuild()
	mudlog.Info("CharacterIndex", "info", "Active character names indexed.", "characters", users.GetCharacterIndex().Len())

	// Load the round count from the file
	if !isCopyover {
		if util.LoadRoundCount(util.FilePath(c.FilePaths.DataFiles.String()+`/`+util.RoundCountFilename)) == util.RoundCountMinimum {
			gametime.SetToDay(-3)
		}
	}

	if isCopyover {
		if err := copyover.Restore(flags.CopyoverFd()); err != nil {
			mudlog.Error("copyover.Restore()", "error", err)
			os.Exit(1)
		}
		mudlog.Info("Copyover", "status", "state restored")
	}

	gametime.GetZodiac(1) // The first time this is called it randomizes all zodiacs

	scripting.Setup(int(c.Scripting.LoadTimeoutMs), int(c.Scripting.RoomTimeoutMs))

	mudlog.Info(`========================`)

	// Wire module admin registrar before loading plugins so that any admin
	// pages and API endpoints registered by modules are available immediately.
	plugins.SetAdminRegistrar(web.GetAdminRegistrar())

	// Trigger the load plugins event
	plugins.Load(
		configs.GetFilePathsConfig().DataFiles.String(),
	)

	mudlog.Info("CharacterIndex", "info", "Character name index complete.", "characters", users.GetCharacterIndex().Len())

	web.SetWebPlugin(plugins.GetPluginRegistry())

	//
	// Capture OS signals to gracefully shutdown the server
	registerShutdownSignals(sigChan)

	// for testing purposes, enable event debugging
	//events.SetDebug(true)

	//
	// Spin up server listeners
	//

	// Set the server to be alive
	serverAlive.Store(true)

	mudlog.Info(`========================`)
	web.Listen(&wg, HandleWebSocketConnection)

	allServerListeners := make([]net.Listener, 0, len(c.Network.TelnetPort))
	for _, port := range c.Network.TelnetPort {
		if p, err := strconv.Atoi(port); err == nil && p > 0 {
			if s := TelnetListenOnPort(``, p, &wg, int(c.Network.MaxTelnetConnections), connections.ConnHuman); s != nil {
				allServerListeners = append(allServerListeners, s)
			}
		}
	}

	if c.Network.LocalPort > 0 {
		TelnetListenOnPort(`127.0.0.1`, int(c.Network.LocalPort), &wg, 0, connections.ConnHuman)
	}

	if c.Network.AI.Port > 0 {
		if s := TelnetListenOnPort(``, int(c.Network.AI.Port), &wg, int(c.Network.AI.MaxConnections), connections.ConnAI); s != nil {
			allServerListeners = append(allServerListeners, s)
		}
	}

	if sshPort := int(c.Network.SSHPort); sshPort > 0 {
		hostKeyPath := c.FilePaths.SSHHostKeyFile.String()
		if hostKeyPath == `` {
			mudlog.Error("SSH", "error", "SSHPort is set but SSHHostKeyFile is not configured; SSH disabled")
		} else {
			hostKeyBytes, err := util.ReadFile(hostKeyPath)
			if err != nil {
				mudlog.Error("SSH", "error", "failed to read SSH host key", "path", hostKeyPath, "details", err)
			} else {
				signer, err := ssh.ParsePrivateKey(hostKeyBytes)
				if err != nil {
					mudlog.Error("SSH", "error", "failed to parse SSH host key", "details", err)
				} else {
					sshConfig := &ssh.ServerConfig{
						NoClientAuth: true,
					}
					sshConfig.AddHostKey(signer)
					if s := SSHListenOnPort(sshPort, sshConfig, &wg, int(c.Network.MaxSSHConnections)); s != nil {
						allServerListeners = append(allServerListeners, s)
					}
				}
			}
		}
	}

	go worldManager.InputWorker(workerShutdownChan, &wg)
	go worldManager.MainWorker(workerShutdownChan, &wg)

	startCopyoverSignalHandler()
	usercommands.SetCopyoverFunc(triggerCopyover)

	if isCopyover {
		for _, connId := range connections.GetAllConnectionIds() {
			cd := connections.Get(connId)
			if cd == nil || cd.State() != connections.LoggedIn {
				continue
			}
			u := users.GetByConnectionId(connId)
			if u == nil {
				continue
			}
			wg.Add(1)
			go resumeRestoredConnection(cd, u, &wg)
		}
		connections.Broadcast([]byte("\r\nCopyover complete.\r\n"))
		mudlog.Info("Copyover", "status", "complete")
	}

	mudlog.Info("Server Ready", "Time Taken", time.Since(serverStartTime))

	// block until a signal comes in
	<-sigChan

	tplTxt, err := templates.Process("goodbye", nil)
	if err != nil {
		mudlog.Error("Template Error", "error", err)
	}

	events.AddToQueue(events.Broadcast{
		Text: templates.AnsiParse(tplTxt),
	})

	serverAlive.Store(false) // immediately stop processing incoming connections

	util.SaveRoundCount(util.FilePath(c.FilePaths.DataFiles.String() + `/` + util.RoundCountFilename))

	// some last minute stats reporting
	totalConnections, totalDisconnections := connections.Stats()
	mudlog.Info(
		"Stopping server",
		"LifetimeConnections", totalConnections,
		"LifetimeDisconnects", totalDisconnections,
		"ActiveConnections", totalConnections-totalDisconnections,
	)

	// cleanup all connections
	connections.Cleanup()

	for _, s := range allServerListeners {
		s.Close()
	}

	web.Shutdown()

	// Final plugin save before shutting down
	plugins.Save()

	// Just a goroutine that spins its wheels until the program shuts down")
	go func() {
		for {
			mudlog.Warn("Waiting on workers")
			// sleep for 3 seconds
			time.Sleep(time.Duration(3) * time.Second)
		}
	}()

	// Close the channel, signalling to the worker threads to shutdown.
	close(workerShutdownChan)

	// Wait for all workers to finish their tasks.
	// Otherwise we end up getting flushed file saves incomplete.
	wg.Wait()

	// Give it a second to disaptch any final messages in the event queue
	// Example: discord server shutdown
	time.Sleep(1 * time.Second)
}

func resumeRestoredConnection(connDetails *connections.ConnectionDetails, userObject *users.UserRecord, wg *sync.WaitGroup) {
	defer wg.Done()

	mudlog.Info("Copyover", "resuming connection", connDetails.ConnectionId(), "userId", userObject.UserId)

	var sharedState map[string]any = make(map[string]any)

	connDetails.AddInputHandler("TelnetIACHandler", inputhandlers.TelnetIACHandler)
	connDetails.AddInputHandler("AnsiHandler", inputhandlers.AnsiHandler)
	connDetails.AddInputHandler("CleanserInputHandler", inputhandlers.CleanserInputHandler)
	connDetails.AddInputHandler("TextPrefixHandler", inputhandlers.TextPrefixHandler)
	connDetails.AddInputHandler("EchoInputHandler", inputhandlers.EchoInputHandler)
	connDetails.AddInputHandler("HistoryInputHandler", inputhandlers.HistoryInputHandler)

	if userObject.Role == users.RoleAdmin {
		connDetails.AddInputHandler("SystemCommandInputHandler", inputhandlers.SystemCommandInputHandler)
	}

	connDetails.AddInputHandler("SignalHandler", inputhandlers.SignalHandler, "AnsiHandler")

	worldManager.SendEnterWorld(userObject.UserId, userObject.Character.RoomId)

	inputBuffer := make([]byte, connections.ReadBufferSize)
	clientInput := &connections.ClientInput{
		ConnectionId: connDetails.ConnectionId(),
		DataIn:       []byte{},
		Buffer:       make([]byte, 0, connections.ReadBufferSize),
		EnterPressed: false,
		Clipboard:    []byte{},
		History:      connections.InputHistory{},
	}

	var sug suggestions.Suggestions
	lastInput := time.Now()
	c := configs.GetConfig()

	for {
		clientInput.EnterPressed = false
		clientInput.TabPressed = false
		clientInput.BSPressed = false

		n, err := connDetails.Read(inputBuffer)
		if err != nil {
			userObject.EventLog.Add(`conn`, `Disconnected`)

			if c.Network.LinkDeadSeconds > 0 {
				connDetails.SetState(connections.LinkDead)
				worldManager.SendSetLinkDead(userObject.UserId, true)
			} else {
				worldManager.SendLeaveWorld(userObject.UserId)
				worldManager.SendLogoutConnectionId(connDetails.ConnectionId())
			}

			mudlog.Warn("Telnet", "connectionID", connDetails.ConnectionId(), "error", err)
			connections.Remove(connDetails.ConnectionId())
			break
		}

		if connDetails.InputDisabled() {
			continue
		}

		clientInput.DataIn = inputBuffer[:n]
		okContinue, lastHandlerName, err := connDetails.HandleInput(clientInput, sharedState)
		if err != nil {
			mudlog.Warn("InputHandler Error", "handler", lastHandlerName, "error", err)
			continue
		}

		if !okContinue {
			_, suggested := userObject.GetUnsentText()

			redrawPrompt := false

			if clientInput.TabPressed {
				if sug.Count() < 1 {
					sug.Set(worldManager.GetAutoComplete(userObject.UserId, string(clientInput.Buffer)))
				}
				if sug.Count() > 0 {
					suggested = sug.Next()
					userObject.SetUnsentText(string(clientInput.Buffer), suggested)
					redrawPrompt = true
				}
			} else if clientInput.BSPressed {
				userObject.SetUnsentText(string(clientInput.Buffer), ``)
				if suggested != `` {
					suggested = ``
					sug.Clear()
					redrawPrompt = true
				}
			} else {
				if suggested != `` {
					if len(clientInput.Buffer) > 0 && clientInput.Buffer[len(clientInput.Buffer)-1] == term.ASCII_SPACE {
						clientInput.Buffer = append(clientInput.Buffer[0:len(clientInput.Buffer)-1], []byte(suggested)...)
						clientInput.Buffer = append(clientInput.Buffer[0:len(clientInput.Buffer)], []byte(` `)...)
						redrawPrompt = true
						userObject.SetUnsentText(string(clientInput.Buffer), ``)
						sug.Clear()
					} else {
						suggested = ``
						sug.Clear()
						userObject.SetUnsentText(string(clientInput.Buffer), suggested)
						redrawPrompt = true
					}
				}
				userObject.SetUnsentText(string(clientInput.Buffer), suggested)
			}

			if redrawPrompt {
				// The prompt redraw below repositions the terminal cursor at the
				// end of the input, so resync the logical cursor to match.
				clientInput.Cursor = len(clientInput.Buffer)
				pTxt := userObject.GetCommandPrompt()
				connections.SendTo([]byte(templates.AnsiParse(pTxt)), clientInput.ConnectionId)
			}

			continue
		}

		if clientInput.EnterPressed {
			c = configs.GetConfig()

			if time.Since(lastInput) < time.Duration(c.Timing.TurnMs)*time.Millisecond {
				clientInput.Reset()
				userObject.SetUnsentText(``, ``)
			} else {
				_, suggested := userObject.GetUnsentText()
				if len(suggested) > 0 {
					clientInput.Buffer = append(clientInput.Buffer, []byte(suggested)...)
					sug.Clear()
					userObject.SetUnsentText(string(clientInput.Buffer), ``)
					connections.SendTo([]byte(templates.AnsiParse(userObject.GetCommandPrompt())), clientInput.ConnectionId)
				}

				wi := WorldInput{
					FromId:    userObject.UserId,
					InputText: string(clientInput.Buffer),
				}
				worldManager.SendInput(wi)
				clientInput.Reset()
				userObject.SetUnsentText(``, ``)
				lastInput = time.Now()
			}

			time.Sleep(time.Duration(10) * time.Millisecond)
		}
	}
}

// Bounds for the connect-time client-type detection probe.
const (
	clientDetectTimeout      = 2 * time.Second        // overall budget before defaulting to non-Mudlet
	clientDetectReadDeadline = 200 * time.Millisecond // per-read slice so we can re-check the budget
)

// detectClientType probes a raw telnet client with an MNES NEW-ENVIRON request
// and synchronously waits (with a bounded timeout) for the reply so the caller
// can choose the correct echo baseline before the first prompt is shown.
//
// Replies are fed through the normal input handler chain, where TelnetIACHandler
// performs the NEW-ENVIRON negotiation and records IsMudlet/DetectionComplete on
// the connection's client settings. The login handler is intentionally not in
// the chain yet, so stray pre-login input is simply buffered (not echoed) and is
// preserved for the main loop. On timeout the client defaults to non-Mudlet.
//
// AI connections skip the probe entirely and keep the historical raw-telnet
// baseline, so automated clients are not delayed by the detection window.
func detectClientType(connDetails *connections.ConnectionDetails, clientInput *connections.ClientInput, sharedState map[string]any) {

	connId := connDetails.ConnectionId()

	if connDetails.ConnType() == connections.ConnAI {
		cs := connections.GetClientSettings(connId)
		cs.DetectionComplete = true
		connections.OverwriteClientSettings(connId, cs)
		return
	}

	// Ask the client to enable NEW-ENVIRON. A Mudlet client replies WILL, which
	// TelnetIACHandler answers with the MNES SEND request; Mudlet then returns
	// its CLIENT_NAME. Non-Mudlet clients reply WONT (or never reply -> timeout).
	connections.SendTo(term.TelnetRequestNewEnviron.BytesWithPayload(nil), connId)

	buf := make([]byte, connections.ReadBufferSize)
	deadline := time.Now().Add(clientDetectTimeout)

	for {
		if connections.GetClientSettings(connId).DetectionComplete {
			break
		}
		if !time.Now().Before(deadline) {
			break
		}

		_ = connDetails.SetReadDeadline(time.Now().Add(clientDetectReadDeadline))

		n, err := connDetails.Read(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue // no data this slice; re-check the overall budget
			}
			break // a real read error; let the main loop surface it
		}
		if n == 0 {
			continue
		}

		clientInput.DataIn = buf[:n]
		if _, lastHandler, herr := connDetails.HandleInput(clientInput, sharedState); herr != nil {
			mudlog.Warn("Client detection input", "handler", lastHandler, "connectionId", connId, "error", herr)
		}
	}

	// Restore blocking reads for the main loop.
	_ = connDetails.SetReadDeadline(time.Time{})

	// Ensure detection is flagged complete even if we timed out.
	cs := connections.GetClientSettings(connId)
	if !cs.DetectionComplete {
		cs.DetectionComplete = true
		connections.OverwriteClientSettings(connId, cs)
	}
}

func handleTelnetConnection(connDetails *connections.ConnectionDetails, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	mudlog.Info("New Connection", "connectionID", connDetails.ConnectionId(), "remoteAddr", connDetails.RemoteAddr().String())

	// Setup shared state map for this connection's handlers
	// Needs to be created BEFORE the first handler call
	var sharedState map[string]any = make(map[string]any)

	// Add starting handlers

	// Special escape handlers
	connDetails.AddInputHandler("TelnetIACHandler", inputhandlers.TelnetIACHandler)
	connDetails.AddInputHandler("AnsiHandler", inputhandlers.AnsiHandler)
	// Consider a macro handler at this point?
	// Text Processing
	connDetails.AddInputHandler("CleanserInputHandler", inputhandlers.CleanserInputHandler)
	connDetails.AddInputHandler("TextPrefixHandler", inputhandlers.TextPrefixHandler)

	// Get the configured login handler func. It is added to the input handler
	// chain further down, AFTER client-type detection, so that the connect-time
	// echo baseline is correct before the very first (username) prompt is shown.
	loginHandler := inputhandlers.GetLoginPromptHandler()

	// Turn off "line at a time", send chars as typed
	connections.SendTo(
		term.TelnetWILL(term.TELNET_OPT_SUP_GO_AHD),
		connDetails.ConnectionId(),
	)
	// Tell the client we expect chars as they are typed
	connections.SendTo(
		term.TelnetWONT(term.TELNET_OPT_LINE_MODE),
		connDetails.ConnectionId(),
	)

	// NOTE: The echo negotiation (WILL/WONT ECHO) is deliberately NOT sent here.
	// It depends on the client type, which we detect below before showing the
	// first prompt. Raw telnet clients rely on server-side echo (WILL ECHO),
	// while local-echo clients such as Mudlet must start at WONT ECHO. See the
	// detection block further down.

	// Request that the client report window size changes as they happen
	connections.SendTo(
		term.TelnetDO(term.TELNET_OPT_NAWS),
		connDetails.ConnectionId(),
	)

	// Send request to change charset
	connections.SendTo(
		term.TelnetRequestChangeCharset.BytesWithPayload(nil),
		connDetails.ConnectionId(),
	)

	// Send request to enable MSP
	connections.SendTo(
		term.MspEnable.BytesWithPayload(nil),
		connDetails.ConnectionId(),
	)

	connections.SendTo(
		term.TelnetSuppressGoAhead.BytesWithPayload(nil),
		connDetails.ConnectionId(),
	)

	// Offer EOR (End Of Record) support. Mudlet prefers EOR over GA and uses it
	// to delimit prompts; without it (and with SUPPRESS-GO-AHEAD set) Mudlet has
	// no prompt anchor, which can prevent its input-line password masking from
	// engaging even though it honors the ECHO option.
	connections.SendTo(
		term.TelnetWILL(term.TELNET_OPT_EOR),
		connDetails.ConnectionId(),
	)

	clientSetupCommands := "" + //term.AnsiAltModeStart.String() + // alternative mode (No scrollback)
		//term.AnsiCursorHide.String() + // Hide Cursor (Because we will manually echo back)
		//term.AnsiCharSetUTF8.String() + // UTF8 mode
		//term.AnsiReportMouseClick.String() + // Request client to capture and report mouse clicks
		term.AnsiRequestResolution.String() // Request resolution
		//""

	connections.SendTo(
		[]byte(clientSetupCommands),
		connDetails.ConnectionId(),
	)

	plugins.OnNetConnect(connDetails)

	// an input buffer for reading data sent over the network
	inputBuffer := make([]byte, connections.ReadBufferSize)

	// Describes whatever the client sent us
	clientInput := &connections.ClientInput{
		ConnectionId: connDetails.ConnectionId(),
		DataIn:       []byte{},
		Buffer:       make([]byte, 0, connections.ReadBufferSize), // DataIn is appended to this buffer after processing
		EnterPressed: false,
		Clipboard:    []byte{},
		History:      connections.InputHistory{},
	}

	if audioConfig := audio.GetFile(`intro`); audioConfig.FilePath != `` {
		v := 100
		if audioConfig.Volume > 0 && audioConfig.Volume <= 100 {
			v = audioConfig.Volume
		}
		connections.SendTo(
			term.MspCommand.BytesWithPayload([]byte("!!MUSIC("+audioConfig.FilePath+" V="+strconv.Itoa(v)+" L=-1 C=1)")),
			clientInput.ConnectionId,
		)
	}

	// --- Send Initial Welcome/Splash ---
	// (This part was mostly correct before)
	splashTxt, _ := templates.Process("login/connect-splash", nil)
	connections.SendTo([]byte(templates.AnsiParse(splashTxt)), connDetails.ConnectionId())

	if connDetails.ConnType() == connections.ConnAI {
		connections.SendTo([]byte("\r\nThis port is for AI clients. Human players, please use the standard telnet port.\r\n\r\n"), connDetails.ConnectionId())
	}

	// --- Detect the client type and pick the connect-time echo baseline ---
	// Local-echo clients such as Mudlet echo input themselves and read telnet
	// ECHO purely as a "mask this field" hint, so they must start at WONT ECHO
	// (and only see WILL ECHO transiently around password prompts). Raw telnet
	// clients rely on server-side echo, so they keep the historical WILL ECHO.
	detectClientType(connDetails, clientInput, sharedState)

	clientCS := connections.GetClientSettings(connDetails.ConnectionId())
	if clientCS.IsMudlet {
		// Mudlet does its own local echo -> baseline is WONT ECHO.
		connections.SendTo(term.TelnetWONT(term.TELNET_OPT_ECHO), connDetails.ConnectionId())
	} else {
		// Raw telnet keeps server-side echo (unchanged behavior).
		connections.SendTo(term.TelnetWILL(term.TELNET_OPT_ECHO), connDetails.ConnectionId())
	}

	// Now that the echo baseline is correct, add the login handler so the first
	// prompt is rendered with the right masking behavior.
	connDetails.AddInputHandler("LoginPromptHandler", loginHandler)

	// --- Trigger the Prompt Handler to initialize state and send the FIRST prompt ---
	// Create a dummy input that signifies "start the process" but has no actual user data/control codes.
	initialTriggerInput := &connections.ClientInput{
		ConnectionId: connDetails.ConnectionId(),
		// Ensure flags like EnterPressed are false
	}
	// Call the handler function directly ONCE.
	// This executes the `!ok` block inside the handler, which:
	// 1. Creates the PromptHandlerState in sharedState.
	// 2. Calls advanceAndSendPromptCustom -> sendPromptFunc for the *first* step (username).
	// 3. Returns false (which we ignore here, as we aren't in the main loop yet).
	loginHandler(initialTriggerInput, sharedState)

	var userObject *users.UserRecord
	var sug suggestions.Suggestions
	lastInput := time.Now()
	c := configs.GetConfig()

	for {

		clientInput.EnterPressed = false // Default state is always false
		clientInput.TabPressed = false   // Default state is always false
		clientInput.BSPressed = false    // Default state is always false

		n, err := connDetails.Read(inputBuffer)
		if err != nil {

			// If failed to read from the connection, switch to linkdead state
			if userObject != nil {

				userObject.EventLog.Add(`conn`, `Disconnected`)

				if c.Network.LinkDeadSeconds > 0 {

					connDetails.SetState(connections.LinkDead)
					worldManager.SendSetLinkDead(userObject.UserId, true)

				} else {

					worldManager.SendLeaveWorld(userObject.UserId)
					worldManager.SendLogoutConnectionId(connDetails.ConnectionId())

				}

			}

			mudlog.Warn("Telnet", "connectionID", connDetails.ConnectionId(), "error", err)

			connections.Remove(connDetails.ConnectionId())

			break
		}

		if connDetails.InputDisabled() {
			continue
		}

		clientInput.DataIn = inputBuffer[:n]
		// Input handler processes any special commands, transforms input, sets flags from input, etc
		okContinue, lastHandlerName, err := connDetails.HandleInput(clientInput, sharedState)
		// Was there an error? If so, we should probably just stop processing input
		if err != nil {
			mudlog.Warn("InputHandler Error", "handler", lastHandlerName, "error", err)
			// Decide if disconnect is needed based on error type
			continue
		}

		// If a handler aborted processing, just keep track of where we are so
		// far and jump back to waiting.
		if !okContinue {

			// if no user signed in, loop back
			if userObject == nil {
				continue
			}

			_, suggested := userObject.GetUnsentText()

			redrawPrompt := false

			if clientInput.TabPressed {

				if sug.Count() < 1 {
					sug.Set(worldManager.GetAutoComplete(userObject.UserId, string(clientInput.Buffer)))
				}

				if sug.Count() > 0 {
					suggested = sug.Next()
					userObject.SetUnsentText(string(clientInput.Buffer), suggested)
					redrawPrompt = true
				}

			} else if clientInput.BSPressed {
				// If a suggestion is pending, remove it
				// otherwise just do a normal backspace operation
				userObject.SetUnsentText(string(clientInput.Buffer), ``)
				if suggested != `` {
					suggested = ``
					sug.Clear()
					redrawPrompt = true
				}

			} else {

				if suggested != `` {

					// If they hit space, accept the suggestion
					if len(clientInput.Buffer) > 0 && clientInput.Buffer[len(clientInput.Buffer)-1] == term.ASCII_SPACE {
						clientInput.Buffer = append(clientInput.Buffer[0:len(clientInput.Buffer)-1], []byte(suggested)...)
						clientInput.Buffer = append(clientInput.Buffer[0:len(clientInput.Buffer)], []byte(` `)...)
						redrawPrompt = true
						userObject.SetUnsentText(string(clientInput.Buffer), ``)
						sug.Clear()
					} else {
						suggested = ``
						sug.Clear()
						// Otherwise, just keep the suggestion
						userObject.SetUnsentText(string(clientInput.Buffer), suggested)
						redrawPrompt = true
					}
				}

				userObject.SetUnsentText(string(clientInput.Buffer), suggested)
			}

			if redrawPrompt {
				// The prompt redraw below repositions the terminal cursor at the
				// end of the input, so resync the logical cursor to match.
				clientInput.Cursor = len(clientInput.Buffer)
				pTxt := userObject.GetCommandPrompt()
				if connections.IsWebsocket(clientInput.ConnectionId) {
					connections.SendTo([]byte(pTxt), clientInput.ConnectionId)
				} else {
					connections.SendTo([]byte(templates.AnsiParse(pTxt)), clientInput.ConnectionId)
				}
			}

			continue
		}

		// The prompt handler returns 'true' from its HandleInput func only when
		// the entire sequence is complete *and* successful (e.g., login or creation ok).
		// If it returns true, it means we should proceed to the logged-in state.
		if okContinue && lastHandlerName == "LoginPromptHandler" {

			// Prompt sequence finished successfully

			// Stop intro music if playing
			connections.SendTo(
				term.MspCommand.BytesWithPayload([]byte("!!MUSIC(Off)")),
				clientInput.ConnectionId,
			)

			// Retrieve the UserObject stored by the completion function
			if uo, exists := sharedState["UserObject"]; exists {
				var ok bool
				userObject, ok = uo.(*users.UserRecord)
				if !ok {
					mudlog.Error("UserObject type assertion failed", "connectionId", clientInput.ConnectionId)
					connections.Remove(clientInput.ConnectionId)
					break
				}
				// Remove it from shared state if no longer needed there
				delete(sharedState, "UserObject")
			} else {
				// This shouldn't happen if the completion function worked correctly
				mudlog.Error("Login process completed but UserObject not found in sharedState", "connectionId", clientInput.ConnectionId)
				connections.Remove(clientInput.ConnectionId) // Disconnect problematic connection
				break                                        // Exit the read loop for this connection
			}

			// Remove the prompt handler (it signaled completion by returning true)
			connDetails.RemoveInputHandler("LoginPromptHandler")
			// Replace it with a regular echo handler.
			connDetails.AddInputHandler("EchoInputHandler", inputhandlers.EchoInputHandler)
			// Add admin command handler
			connDetails.AddInputHandler("HistoryInputHandler", inputhandlers.HistoryInputHandler) // Put history tracking after login handling, since login handling aborts input until complete

			if userObject.Role == users.RoleAdmin {
				connDetails.AddInputHandler("SystemCommandInputHandler", inputhandlers.SystemCommandInputHandler)
			}

			// Add a signal handler (shortcut ctrl combos) after the AnsiHandler
			// This captures signals and replaces user input so should happen after AnsiHandler to ensure it happens before other processes.
			connDetails.AddInputHandler("SignalHandler", inputhandlers.SignalHandler, "AnsiHandler")

			connDetails.SetState(connections.LoggedIn)

			worldManager.SendEnterWorld(userObject.UserId, userObject.Character.RoomId)

			if connDetails.ConnType() == connections.ConnAI && !userObject.IsAI {
				connections.SendTo([]byte("\r\nWarning: this account is not flagged as AI but connected on the AI port.\r\n"), connDetails.ConnectionId())
			} else if connDetails.ConnType() == connections.ConnHuman && userObject.IsAI {
				connections.SendTo([]byte("\r\nWarning: this AI account is connected on a human port. Please use the AI port.\r\n"), connDetails.ConnectionId())
			}

			clientInput.Reset()
			continue
		}

		// If okContinue is false OR the last handler was *not* the prompt handler,
		// it means either an error occurred (handled above), a handler aborted (like IAC/ANSI),
		// or the prompt handler is still waiting for input for the current/next step.
		// The existing logic for handling Tab/Backspace suggestions AFTER the input handlers run
		// might need slight adjustment depending on exactly how you want suggestions during prompts.
		// For simplicity, you might disable suggestions during the prompt sequence.
		if !okContinue {
			if userObject == nil {
				continue
			}
		}

		// If they have pressed enter (submitted their input), and nothing else has handled/aborted
		if clientInput.EnterPressed {

			// Update config after enter presses
			// No need to update it every loop
			c = configs.GetConfig()

			if time.Since(lastInput) < time.Duration(c.Timing.TurnMs)*time.Millisecond {
				/*
					connections.SendTo(
						[]byte("Slow down! You're typing too fast! "+time.Since(lastInput).String()+"\n"),
						connDetails.ConnectionId(),
					)
				*/

				// Reset the buffer for future commands.
				clientInput.Reset()

				// Capturing and resetting the unsent text is purely to allow us to
				// Keep updating the prompt without losing the typed in text.
				userObject.SetUnsentText(``, ``)

			} else {

				_, suggested := userObject.GetUnsentText()

				if len(suggested) > 0 {
					// solidify it in the render for UX reasons

					clientInput.Buffer = append(clientInput.Buffer, []byte(suggested)...)
					sug.Clear()
					userObject.SetUnsentText(string(clientInput.Buffer), ``)

					if connections.IsWebsocket(clientInput.ConnectionId) {
						connections.SendTo([]byte(userObject.GetCommandPrompt()), clientInput.ConnectionId)
					} else {
						connections.SendTo([]byte(templates.AnsiParse(userObject.GetCommandPrompt())), clientInput.ConnectionId)
					}

				}

				if connDetails.ConnType() == connections.ConnAI {
					if !connDetails.AICommandAllowed(int64(util.GetRoundCount()), int(c.Network.AI.CommandsPerRound)) {
						connections.SendTo(
							[]byte(fmt.Sprintf("Command dropped — AI rate limit (%d/round). Wait for the next round.\r\n", int(c.Network.AI.CommandsPerRound))),
							connDetails.ConnectionId(),
						)
						clientInput.Reset()
						userObject.SetUnsentText(``, ``)
						time.Sleep(time.Duration(10) * time.Millisecond)
						continue
					}
				}

				wi := WorldInput{
					FromId:    userObject.UserId,
					InputText: string(clientInput.Buffer),
				}

				// Buffer should be processed as an in-game command
				worldManager.SendInput(wi)
				// Reset the buffer for future commands.
				clientInput.Reset()

				// Capturing and resetting the unsent text is purely to allow us to
				// Keep updating the prompt without losing the typed in text.
				userObject.SetUnsentText(``, ``)

				lastInput = time.Now()
			}

			time.Sleep(time.Duration(10) * time.Millisecond)
			//	time.Sleep(time.Duration(util.TurnMs) * time.Millisecond)
		}

	}

}

func HandleWebSocketConnection(conn *websocket.Conn) {

	var userObject *users.UserRecord
	connDetails := connections.Add(nil, conn)

	// Setup shared state map for this connection's handlers
	// Needs to be created BEFORE the first handler call
	var sharedState map[string]any = make(map[string]any)

	// Describes whatever the client sent us
	clientInput := &connections.ClientInput{
		ConnectionId: connDetails.ConnectionId(),
		DataIn:       []byte{},
		Buffer:       make([]byte, 0, connections.ReadBufferSize),
		EnterPressed: false,
		Clipboard:    []byte{},
		History:      connections.InputHistory{},
	}

	connections.SendTo(
		[]byte("!!SOUND(Off U="+configs.GetConfig().FilePaths.WebCDNLocation.String()+")"),
		clientInput.ConnectionId,
	)

	plugins.OnNetConnect(connDetails)

	loginHandler := inputhandlers.GetLoginPromptHandler()
	connDetails.AddInputHandler("LoginPromptHandler", loginHandler)

	if audioConfig := audio.GetFile(`intro`); audioConfig.FilePath != `` {
		v := 100
		if audioConfig.Volume > 0 && audioConfig.Volume <= 100 {
			v = audioConfig.Volume
		}
		connections.SendTo(
			[]byte("!!MUSIC("+audioConfig.FilePath+" V="+strconv.Itoa(v)+" L=-1 C=1)"),
			clientInput.ConnectionId,
		)
	}

	splashTxt, _ := templates.Process("login/connect-splash", nil)
	connections.SendTo([]byte(templates.AnsiParse(splashTxt)), connDetails.ConnectionId())

	initialTriggerInput := &connections.ClientInput{
		ConnectionId: connDetails.ConnectionId(),
	}
	loginHandler(initialTriggerInput, sharedState)

	c := configs.GetConfig()

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {

			// If failed to read from the connection, switch to linkdead state
			if userObject != nil {

				userObject.EventLog.Add(`conn`, `Disconnected`)

				if c.Network.LinkDeadSeconds > 0 {

					connDetails.SetState(connections.LinkDead)
					worldManager.SendSetLinkDead(userObject.UserId, true)

				} else {

					worldManager.SendLeaveWorld(userObject.UserId)
					worldManager.SendLogoutConnectionId(connDetails.ConnectionId())

				}

			}

			mudlog.Warn("WS Read", "error", err)
			connections.Remove(connDetails.ConnectionId())
			break
		}

		// Check for a copyover reconnect token before normal input handling.
		if userObject == nil {
			if userId, ok := copyover.ConsumeReconnectToken(string(message)); ok {
				tmpUser := users.GetByUserId(userId)
				if tmpUser != nil {
					loggedInUser, msg, loginErr := users.CopyoverReconnectUser(tmpUser, clientInput.ConnectionId)
					if loginErr != nil {
						if len(msg) > 0 {
							connections.SendTo([]byte(msg), clientInput.ConnectionId)
						}
						connections.Remove(clientInput.ConnectionId)
						return
					}
					if len(msg) > 0 {
						connections.SendTo([]byte(msg), clientInput.ConnectionId)
					}
					userObject = loggedInUser
					connDetails.RemoveInputHandler("LoginPromptHandler")
					connDetails.AddInputHandler("TextPrefixHandler", inputhandlers.TextPrefixHandler)
					connDetails.AddInputHandler("EchoInputHandler", inputhandlers.EchoInputHandler)
					connDetails.AddInputHandler("HistoryInputHandler", inputhandlers.HistoryInputHandler)
					if userObject.Role == users.RoleAdmin {
						connDetails.AddInputHandler("SystemCommandInputHandler", inputhandlers.SystemCommandInputHandler)
					}
					connDetails.AddInputHandler("SignalHandler", inputhandlers.SignalHandler, "AnsiHandler")
					connDetails.SetState(connections.LoggedIn)
					worldManager.SendEnterWorld(userObject.UserId, userObject.Character.RoomId)
					mudlog.Info("WebSocket copyover reconnect", "username", userObject.Username, "connectionId", clientInput.ConnectionId)
					clientInput.Reset()
					continue
				}
				// Token valid but user not in memory; fall through to normal login handling.
			}
		}

		clientInput.DataIn = message
		clientInput.Buffer = message
		clientInput.EnterPressed = true

		// Input handler processes any special commands, transforms input, sets flags from input, etc
		okContinue, lastHandlerName, err := connDetails.HandleInput(clientInput, sharedState)
		// Was there an error? If so, we should probably just stop processing input
		if err != nil {
			mudlog.Warn("InputHandler Error", "handler", lastHandlerName, "error", err)
			// Decide if disconnect is needed based on error type
			continue
		}

		// If okContinue is false OR the last handler was *not* the prompt handler,
		// it means either an error occurred (handled above), a handler aborted (like IAC/ANSI),
		// or the prompt handler is still waiting for input for the current/next step.
		// The existing logic for handling Tab/Backspace suggestions AFTER the input handlers run
		// might need slight adjustment depending on exactly how you want suggestions during prompts.
		// For simplicity, you might disable suggestions during the prompt sequence.
		if !okContinue {
			continue
		}

		// The prompt handler returns 'true' from its HandleInput func only when
		// the entire sequence is complete *and* successful (e.g., login or creation ok).
		// If it returns true, it means we should proceed to the logged-in state.
		if okContinue && lastHandlerName == "LoginPromptHandler" {

			// Prompt sequence finished successfully

			// Make sure web client text masking is off

			events.AddToQueue(events.WebClientCommand{
				ConnectionId: clientInput.ConnectionId,
				Text:         `TEXTMASK:false`,
			})

			// Stop intro music if playing
			connections.SendTo(
				[]byte("!!MUSIC(Off)"),
				clientInput.ConnectionId,
			)

			// Retrieve the UserObject stored by the completion function
			if uo, exists := sharedState["UserObject"]; exists {
				var ok bool
				userObject, ok = uo.(*users.UserRecord)
				if !ok {
					mudlog.Error("UserObject type assertion failed", "connectionId", clientInput.ConnectionId)
					connections.Remove(clientInput.ConnectionId)
					break
				}
				// Remove it from shared state if no longer needed there
				delete(sharedState, "UserObject")
			} else {
				// This shouldn't happen if the completion function worked correctly
				mudlog.Error("Login process completed but UserObject not found in sharedState", "connectionId", clientInput.ConnectionId)
				connections.Remove(clientInput.ConnectionId) // Disconnect problematic connection
				break                                        // Exit the read loop for this connection
			}

			// Remove the prompt handler (it signaled completion by returning true)
			connDetails.RemoveInputHandler("LoginPromptHandler")
			// Replace it with a regular echo handler.
			connDetails.AddInputHandler("TextPrefixHandler", inputhandlers.TextPrefixHandler)
			connDetails.AddInputHandler("EchoInputHandler", inputhandlers.EchoInputHandler)
			// Add admin command handler
			connDetails.AddInputHandler("HistoryInputHandler", inputhandlers.HistoryInputHandler) // Put history tracking after login handling, since login handling aborts input until complete

			if userObject.Role == users.RoleAdmin {
				connDetails.AddInputHandler("SystemCommandInputHandler", inputhandlers.SystemCommandInputHandler)
			}

			// Add a signal handler (shortcut ctrl combos) after the AnsiHandler
			// This captures signals and replaces user input so should happen after AnsiHandler to ensure it happens before other processes.
			connDetails.AddInputHandler("SignalHandler", inputhandlers.SignalHandler, "AnsiHandler")

			connDetails.SetState(connections.LoggedIn)

			worldManager.SendEnterWorld(userObject.UserId, userObject.Character.RoomId)

			clientInput.Reset()
			continue
		}

		if userObject == nil {
			continue
		}

		wi := WorldInput{
			FromId:    userObject.UserId,
			InputText: string(message),
		}

		// Buffer should be processed as an in-game command
		worldManager.SendInput(wi)

		c = configs.GetConfig()
	}
}

func TelnetListenOnPort(hostname string, portNum int, wg *sync.WaitGroup, maxConnections int, connType connections.ConnType) net.Listener {

	server, err := net.Listen("tcp", fmt.Sprintf("%s:%d", hostname, portNum))
	if err != nil {
		mudlog.Error("Error creating server", "error", err)
		return nil
	}

	// Start a goroutine to accept incoming connections, so that we can use a signal to stop the server
	go func() {

		// Loop to accept connections
		for {
			conn, err := server.Accept()

			if !serverAlive.Load() {
				mudlog.Warn("Connections disabled.")
				return
			}

			if err != nil {
				mudlog.Warn("Connection error", "error", err)
				continue
			}

			if maxConnections > 0 {
				activeCount := connections.ActiveConnectionCount()
				if connType == connections.ConnAI {
					activeCount = connections.ActiveAIConnectionCount()
				}
				if activeCount >= maxConnections {
					if connType == connections.ConnAI {
						conn.Write([]byte("\n\n\n!!! AI connection pool is full. Try again later. !!!\n\n\n"))
					} else {
						conn.Write([]byte(fmt.Sprintf("\n\n\n!!! Server is full (%d connections). Try again later. !!!\n\n\n", activeCount)))
					}
					conn.Close()
					continue
				}
			}

			wg.Add(1)
			connDetails := connections.Add(conn, nil, connType)
			if connType == connections.ConnAI {
				connDetails.SetStripAnsi(true)
			}
			// hand off the connection to a handler goroutine so that we can continue handling new connections
			go handleTelnetConnection(connDetails, wg)

		}
	}()

	return server
}

func SSHListenOnPort(portNum int, sshConfig *ssh.ServerConfig, wg *sync.WaitGroup, maxConnections int) net.Listener {

	server, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		mudlog.Error("Error creating SSH server", "error", err)
		return nil
	}

	mudlog.Info("SSH", "status", "listening", "port", portNum)

	go func() {
		for {
			conn, err := server.Accept()

			if !serverAlive.Load() {
				mudlog.Warn("SSH connections disabled.")
				return
			}

			if err != nil {
				mudlog.Warn("SSH connection error", "error", err)
				continue
			}

			if maxConnections > 0 {
				if connections.ActiveConnectionCount() >= maxConnections {
					conn.Write([]byte("\n\n\n!!! Server is full. Try again later. !!!\n\n\n"))
					conn.Close()
					continue
				}
			}

			wg.Add(1)
			go handleSSHHandshake(conn, sshConfig, wg)
		}
	}()

	return server
}

func handleSSHHandshake(conn net.Conn, sshConfig *ssh.ServerConfig, wg *sync.WaitGroup) {
	defer wg.Done()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, sshConfig)
	if err != nil {
		mudlog.Warn("SSH handshake failed", "remoteAddr", conn.RemoteAddr().String(), "error", err)
		return
	}
	defer sshConn.Close()

	// Discard all out-of-band requests (keepalive, etc.)
	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}

		ch, chanReqs, err := newChan.Accept()
		if err != nil {
			mudlog.Warn("SSH channel accept failed", "error", err)
			return
		}

		wg.Add(1)
		go handleSSHConnection(connections.AddSSH(ch, sshConn.RemoteAddr()), chanReqs, wg)
	}
}

func handleSSHConnection(connDetails *connections.ConnectionDetails, reqs <-chan *ssh.Request, wg *sync.WaitGroup) {
	defer wg.Done()

	mudlog.Info("New SSH Connection", "connectionID", connDetails.ConnectionId(), "remoteAddr", connDetails.RemoteAddr().String())

	// Handle SSH channel requests (pty-req, window-change, shell, exec, etc.)
	// We run this in a goroutine so the read loop below can start immediately.
	go func() {
		for req := range reqs {
			switch req.Type {
			case "pty-req":
				// Parse terminal dimensions from the pty-req payload.
				// Payload: string term, uint32 cols, uint32 rows, uint32 width-px, uint32 height-px, string modes
				if len(req.Payload) >= 12 {
					// Skip the terminal name string: 4-byte length prefix + string bytes
					nameLen := int(req.Payload[0])<<24 | int(req.Payload[1])<<16 | int(req.Payload[2])<<8 | int(req.Payload[3])
					offset := 4 + nameLen
					if offset+8 <= len(req.Payload) {
						cols := uint32(req.Payload[offset])<<24 | uint32(req.Payload[offset+1])<<16 | uint32(req.Payload[offset+2])<<8 | uint32(req.Payload[offset+3])
						rows := uint32(req.Payload[offset+4])<<24 | uint32(req.Payload[offset+5])<<16 | uint32(req.Payload[offset+6])<<8 | uint32(req.Payload[offset+7])
						if cols > 0 && rows > 0 {
							cs := connections.GetClientSettings(connDetails.ConnectionId())
							cs.Display.ScreenWidth = cols
							cs.Display.ScreenHeight = rows
							connections.OverwriteClientSettings(connDetails.ConnectionId(), cs)
						}
					}
				}
				if req.WantReply {
					req.Reply(true, nil)
				}
			case "window-change":
				// Payload: uint32 cols, uint32 rows, uint32 width-px, uint32 height-px
				if len(req.Payload) >= 8 {
					cols := uint32(req.Payload[0])<<24 | uint32(req.Payload[1])<<16 | uint32(req.Payload[2])<<8 | uint32(req.Payload[3])
					rows := uint32(req.Payload[4])<<24 | uint32(req.Payload[5])<<16 | uint32(req.Payload[6])<<8 | uint32(req.Payload[7])
					if cols > 0 && rows > 0 {
						cs := connections.GetClientSettings(connDetails.ConnectionId())
						cs.Display.ScreenWidth = cols
						cs.Display.ScreenHeight = rows
						connections.OverwriteClientSettings(connDetails.ConnectionId(), cs)
						connections.NotifyWindowChange(connDetails.ConnectionId(), cols, rows)
					}
				}
				if req.WantReply {
					req.Reply(true, nil)
				}
			case "shell", "exec":
				if req.WantReply {
					req.Reply(true, nil)
				}
			default:
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}
	}()

	var sharedState map[string]any = make(map[string]any)

	connDetails.AddInputHandler("AnsiHandler", inputhandlers.AnsiHandler)
	connDetails.AddInputHandler("CleanserInputHandler", inputhandlers.CleanserInputHandler)
	connDetails.AddInputHandler("TextPrefixHandler", inputhandlers.TextPrefixHandler)

	loginHandler := inputhandlers.GetLoginPromptHandler()
	connDetails.AddInputHandler("LoginPromptHandler", loginHandler)

	plugins.OnNetConnect(connDetails)

	if audioConfig := audio.GetFile(`intro`); audioConfig.FilePath != `` {
		v := 100
		if audioConfig.Volume > 0 && audioConfig.Volume <= 100 {
			v = audioConfig.Volume
		}
		connections.SendTo(
			term.MspCommand.BytesWithPayload([]byte("!!MUSIC("+audioConfig.FilePath+" V="+strconv.Itoa(v)+" L=-1 C=1)")),
			connDetails.ConnectionId(),
		)
	}

	splashTxt, _ := templates.Process("login/connect-splash", nil)
	connections.SendTo([]byte(templates.AnsiParse(splashTxt)), connDetails.ConnectionId())

	initialTriggerInput := &connections.ClientInput{
		ConnectionId: connDetails.ConnectionId(),
	}
	loginHandler(initialTriggerInput, sharedState)

	var userObject *users.UserRecord
	var sug suggestions.Suggestions
	lastInput := time.Now()
	c := configs.GetConfig()

	inputBuffer := make([]byte, connections.ReadBufferSize)
	clientInput := &connections.ClientInput{
		ConnectionId: connDetails.ConnectionId(),
		DataIn:       []byte{},
		Buffer:       make([]byte, 0, connections.ReadBufferSize),
		EnterPressed: false,
		Clipboard:    []byte{},
		History:      connections.InputHistory{},
	}

	for {
		clientInput.EnterPressed = false
		clientInput.TabPressed = false
		clientInput.BSPressed = false

		n, err := connDetails.Read(inputBuffer)
		if err != nil {
			if userObject != nil {
				userObject.EventLog.Add(`conn`, `Disconnected`)
				if c.Network.LinkDeadSeconds > 0 {
					connDetails.SetState(connections.LinkDead)
					worldManager.SendSetLinkDead(userObject.UserId, true)
				} else {
					worldManager.SendLeaveWorld(userObject.UserId)
					worldManager.SendLogoutConnectionId(connDetails.ConnectionId())
				}
			}
			mudlog.Warn("SSH", "connectionID", connDetails.ConnectionId(), "error", err)
			connections.Remove(connDetails.ConnectionId())
			break
		}

		if connDetails.InputDisabled() {
			continue
		}

		clientInput.DataIn = inputBuffer[:n]
		okContinue, lastHandlerName, err := connDetails.HandleInput(clientInput, sharedState)
		if err != nil {
			mudlog.Warn("InputHandler Error", "handler", lastHandlerName, "error", err)
			continue
		}

		if !okContinue {
			if userObject == nil {
				continue
			}

			_, suggested := userObject.GetUnsentText()
			redrawPrompt := false

			if clientInput.TabPressed {
				if sug.Count() < 1 {
					sug.Set(worldManager.GetAutoComplete(userObject.UserId, string(clientInput.Buffer)))
				}
				if sug.Count() > 0 {
					suggested = sug.Next()
					userObject.SetUnsentText(string(clientInput.Buffer), suggested)
					redrawPrompt = true
				}
			} else if clientInput.BSPressed {
				userObject.SetUnsentText(string(clientInput.Buffer), ``)
				if suggested != `` {
					suggested = ``
					sug.Clear()
					redrawPrompt = true
				}
			} else {
				if suggested != `` {
					if len(clientInput.Buffer) > 0 && clientInput.Buffer[len(clientInput.Buffer)-1] == term.ASCII_SPACE {
						clientInput.Buffer = append(clientInput.Buffer[0:len(clientInput.Buffer)-1], []byte(suggested)...)
						clientInput.Buffer = append(clientInput.Buffer[0:len(clientInput.Buffer)], []byte(` `)...)
						redrawPrompt = true
						userObject.SetUnsentText(string(clientInput.Buffer), ``)
						sug.Clear()
					} else {
						suggested = ``
						sug.Clear()
						userObject.SetUnsentText(string(clientInput.Buffer), suggested)
						redrawPrompt = true
					}
				}
				userObject.SetUnsentText(string(clientInput.Buffer), suggested)
			}

			if redrawPrompt {
				// The prompt redraw below repositions the terminal cursor at the
				// end of the input, so resync the logical cursor to match.
				clientInput.Cursor = len(clientInput.Buffer)
				pTxt := userObject.GetCommandPrompt()
				connections.SendTo([]byte(templates.AnsiParse(pTxt)), clientInput.ConnectionId)
			}
			continue
		}

		if okContinue && lastHandlerName == "LoginPromptHandler" {
			connections.SendTo(
				term.MspCommand.BytesWithPayload([]byte("!!MUSIC(Off)")),
				clientInput.ConnectionId,
			)

			if uo, exists := sharedState["UserObject"]; exists {
				var ok bool
				userObject, ok = uo.(*users.UserRecord)
				if !ok {
					mudlog.Error("UserObject type assertion failed", "connectionId", clientInput.ConnectionId)
					connections.Remove(clientInput.ConnectionId)
					break
				}
				delete(sharedState, "UserObject")
			} else {
				mudlog.Error("Login process completed but UserObject not found in sharedState", "connectionId", clientInput.ConnectionId)
				connections.Remove(clientInput.ConnectionId)
				break
			}

			connDetails.RemoveInputHandler("LoginPromptHandler")
			connDetails.AddInputHandler("EchoInputHandler", inputhandlers.EchoInputHandler)
			connDetails.AddInputHandler("HistoryInputHandler", inputhandlers.HistoryInputHandler)

			if userObject.Role == users.RoleAdmin {
				connDetails.AddInputHandler("SystemCommandInputHandler", inputhandlers.SystemCommandInputHandler)
			}

			connDetails.AddInputHandler("SignalHandler", inputhandlers.SignalHandler, "AnsiHandler")
			connDetails.SetState(connections.LoggedIn)
			worldManager.SendEnterWorld(userObject.UserId, userObject.Character.RoomId)
			clientInput.Reset()
			continue
		}

		if !okContinue {
			if userObject == nil {
				continue
			}
		}

		if clientInput.EnterPressed {
			c = configs.GetConfig()

			if time.Since(lastInput) < time.Duration(c.Timing.TurnMs)*time.Millisecond {
				clientInput.Reset()
				userObject.SetUnsentText(``, ``)
			} else {
				_, suggested := userObject.GetUnsentText()
				if len(suggested) > 0 {
					clientInput.Buffer = append(clientInput.Buffer, []byte(suggested)...)
					sug.Clear()
					userObject.SetUnsentText(string(clientInput.Buffer), ``)
					connections.SendTo([]byte(templates.AnsiParse(userObject.GetCommandPrompt())), clientInput.ConnectionId)
				}

				wi := WorldInput{
					FromId:    userObject.UserId,
					InputText: string(clientInput.Buffer),
				}
				worldManager.SendInput(wi)
				clientInput.Reset()
				userObject.SetUnsentText(``, ``)
				lastInput = time.Now()
			}

			time.Sleep(time.Duration(10) * time.Millisecond)
		}
	}
}

func loadAllDataFiles(isReload bool) {

	if isReload {

		defer func() {
			if r := recover(); r != nil {
				mudlog.Error("RELOAD FAILED", "err", r)
			}
		}()

	}

	// Force clear all cached VM's
	scripting.PruneVMs(true)

	gametime.LoadGameTimeConfigs()
	// Load biomes before rooms since rooms reference biomes
	rooms.LoadBiomeDataFiles()
	spells.LoadSpellFiles()
	rooms.LoadDataFiles()
	buffs.LoadFlagDataFiles() // Load buff flags before buffs so buff validation can check flags
	buffs.LoadDataFiles()     // Load buffs before items for cost calculation reasons
	items.LoadDataFiles()
	races.LoadDataFiles()
	skills.LoadDataFiles()           // skills before professions for cross-ref warnings
	skills.LoadProfessionDataFiles() // professions reference skills
	mobs.LoadDataFiles()
	pets.LoadDataFiles()
	quests.LoadDataFiles()
	templates.LoadAliases(plugins.GetPluginRegistry())
	keywords.LoadAliases(plugins.GetPluginRegistry())
	mutators.LoadDataFiles()
	colorpatterns.LoadColorPatterns()
	audio.LoadAudioConfig()
	characters.CompileAdjectiveSwaps() // This should come after loading color patterns.
}
