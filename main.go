//go:generate go run cmd/generate/module-imports.go
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/GoMudEngine/GoMud/internal/dice"

	"github.com/GoMudEngine/GoMud/internal/audio"
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/flags"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/inputhandlers"
	"github.com/GoMudEngine/GoMud/internal/integrations/discord"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/language"
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/migration"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/version"
	"github.com/gorilla/websocket"

	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/enchantments"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/sealedcrate"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/economy/health"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/suggestions"
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
const VERSION = "0.12.0"

var (
	sigChan            = make(chan os.Signal, 1)
	workerShutdownChan = make(chan bool, 1)

	serverAlive atomic.Bool

	worldManager = NewWorld(sigChan)

	// Start a pool of worker goroutines
	wg sync.WaitGroup
)

// systemNPCAnchorRooms is the explicit list of rooms whose spawninfo
// must fire at boot rather than on first player visit. These rooms
// host long-running system NPCs (caravan master, foragers) whose
// state machines must run continuously regardless of player presence.
var systemNPCAnchorRooms = []int{
	465,  // Thornwall depot — Ketil 357 (caravan master) + wagon 374 + Hob 375 + Bran 376
	4123, // Stillwater Temple — Tova, Marsh forager 371
	3040, // Ironwind Steppe sanctuary — Halix, Steppe forager 372
	4197, // Forager's Camp, Fernway South — Kessa, Fernway forager 373
}

func main() {

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

	// Phase 1: Basic stderr logging (config not loaded yet)
	mudlog.SetupLogger(
		events.GetLogger(),
		os.Getenv(`LOG_LEVEL`),
		``,
		os.Getenv(`LOG_NOCOLOR`) == ``,
	)

	flags.HandleFlags(VERSION)

	configs.ReloadConfig()
	c := configs.GetConfig()

	// Phase 2: Reconfigure logging with file output if enabled in config.
	// Env vars take precedence over config for backward compatibility.
	{
		logCfg := configs.GetLoggingConfig()
		logLevel := os.Getenv(`LOG_LEVEL`)
		if logLevel == `` {
			logLevel = string(logCfg.LogLevel)
		}
		logPath := os.Getenv(`LOG_PATH`)
		if logPath == `` && bool(logCfg.LogToFile) {
			logPath = string(logCfg.LogFilePath)
		}
		mudlog.SetupLogger(
			events.GetLogger(),
			logLevel,
			logPath,
			os.Getenv(`LOG_NOCOLOR`) == ``,
		)
	}

	// Apply the global roll-spread factor from config to the dice package.
	// This is the single knob that scales stdDev for every stat-based roll.
	// See _datafiles/config.yaml Balance.RollSpread for the full explanation.
	dice.SetRollSpread(float64(configs.GetBalanceConfig().RollSpread))

	lastKnownVersion, err := version.Parse(string(configs.GetServerConfig().CurrentVersion))
	if err != nil {
		mudlog.Error("Versioning", "error", err)
		os.Exit(1)
	}

	currentVersion, _ := version.Parse(VERSION)

	if err = migration.Run(lastKnownVersion, currentVersion); err != nil {
		mudlog.Error("migration.Run()", "error", err)
		os.Exit(1)
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

	// Older versions of GoMud may not have this folder present.
	// Also deleting the folder is a quick way to reset instance state, so this corrects that if it happens.
	os.Mkdir(util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `rooms.instances`), os.ModeDir|0755)
	os.Mkdir(util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `mobs.instances`), os.ModeDir|0755)

	// Prune stale mob instance saves
	mobs.NukeSummonsInstances()
	mobs.PruneStaleInstances(int(configs.GetBalanceConfig().MobInstanceMaxAgeDays))

	// Register the plugin filesystem with the template system
	templates.RegisterFS(plugins.GetPluginRegistry())
	usercommands.AddFunctionExporter(plugins.GetPluginRegistry())

	inputhandlers.AddIACHandler(plugins.GetPluginRegistry())

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

	// Wire the rooms package's btree-eviction callback to behaviortree's
	// implementation. Avoids an internal/rooms → internal/behaviortree
	// import cycle (behaviortree already imports rooms).
	rooms.SetBTreeStateEvictor(behaviortree.EvictRoomBTreeState)

	// Wire companion follow-movement into MoveToRoom. Avoids the
	// internal/rooms → internal/hooks import cycle (hooks imports rooms).
	rooms.SetCompanionTransport(hooks.CompanionTransportCallback)

	// Register the grace-period untargetable-user check so SetAggro
	// can short-circuit against grace-protected respawning players
	// without characters/aggro.go needing to import users (would
	// create a cycle).
	characters.SetUserUntargetableCheck(func(userId int) bool {
		user := users.GetByUserId(userId)
		if user == nil {
			return false
		}
		return user.Character.HasBuffFlag(buffs.NoAggroTarget)
	})

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
	idx := users.NewUserIndex()
	if !idx.Exists() {
		// Since it doesn't exist yet, that's a good indication we should do a quick format migration check
		users.DoUserMigrations()
	}
	idx.Create()
	idx.Rebuild()
	mudlog.Info("UserIndex", "info", "User index recreated.")

	// Load the round count from the file
	if util.LoadRoundCount(c.FilePaths.DataFiles.String()+`/`+util.RoundCountFilename) == util.RoundCountMinimum {
		gametime.SetToDay(-3)
	}

	gametime.GetZodiac(1) // The first time this is called it randomizes all zodiacs

	mudlog.Info(`========================`)

	// Trigger the load plugins event
	plugins.Load(
		configs.GetFilePathsConfig().DataFiles.String(),
	)

	web.SetWebPlugin(plugins.GetPluginRegistry())

	// Audit command registry parity between user and mob command sets.
	// Logs warnings for any unexpected gaps (i.e. commands present on one side
	// but absent on the other that aren't in the intentional allowlists).
	actions.AuditCommandParity(
		usercommands.GetAllUserCommands(),
		mobcommands.GetAllMobCommands(),
	)

	//
	// Capture OS signals to gracefully shutdown the server
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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
		if p, err := strconv.Atoi(port); err == nil {
			if s := TelnetListenOnPort(``, p, &wg, int(c.Network.MaxTelnetConnections)); s != nil {
				allServerListeners = append(allServerListeners, s)
			}
		}
	}

	// Start AI port listener if configured
	if c.Network.AIPort > 0 {
		if s := TelnetListenOnPort(``, int(c.Network.AIPort), &wg, int(c.Network.MaxTelnetConnections), connections.ConnAI); s != nil {
			allServerListeners = append(allServerListeners, s)
		}
	}

	if c.Network.LocalPort > 0 {
		TelnetListenOnPort(`127.0.0.1`, int(c.Network.LocalPort), &wg, 0)
	}

	go worldManager.InputWorker(workerShutdownChan, &wg)
	go worldManager.MainWorker(workerShutdownChan, &wg)

	// Hourly economy-health snapshot capture. Runs while the server is
	// up; uses workerShutdownChan to halt cleanly. Pruning runs once
	// per day.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				mudlog.Error("PANIC", "where", "economy_snapshot_ticker", "error", r)
			}
		}()
		b := configs.GetBalanceConfig()
		hours := int(b.EconomySnapshotIntervalHours)
		if hours <= 0 {
			hours = 1
		}
		ticker := time.NewTicker(time.Duration(hours) * time.Hour)
		defer ticker.Stop()
		pruneTicker := time.NewTicker(24 * time.Hour)
		defer pruneTicker.Stop()
		for {
			select {
			case <-workerShutdownChan:
				return
			case <-ticker.C:
				snap := health.CaptureSnapshot()
				if err := health.WriteSnapshot(snap); err != nil {
					mudlog.Error("economy snapshot write", "error", err)
				}
			case <-pruneTicker.C:
				retention := int(configs.GetBalanceConfig().EconomySnapshotRetentionDays)
				if _, err := health.PruneSnapshots(retention); err != nil {
					mudlog.Error("economy snapshot prune", "error", err)
				}
			}
		}
	}()

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

	util.SaveRoundCount(c.FilePaths.DataFiles.String() + `/` + util.RoundCountFilename)

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
		defer func() {
			if r := recover(); r != nil {
				mudlog.Error("PANIC", "error", r)
				s := string(debug.Stack())
				for _, str := range strings.Split(s, "\n") {
					mudlog.Error("PANIC", "stack", str)
				}
			}
		}()
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

	loginHandler := inputhandlers.GetLoginPromptHandler()           // Get the configured handler func
	connDetails.AddInputHandler("LoginPromptHandler", loginHandler) // Add it with a unique name

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

	// Tell the client we intend to echo back what they type
	// So they shouldn't locally echo it

	connections.SendTo(
		term.TelnetWILL(term.TELNET_OPT_ECHO),
		connDetails.ConnectionId(),
	)
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

	// Show AI port notice if this is an AI connection
	if connDetails.ConnType() == connections.ConnAI {
		connections.SendTo([]byte("\r\nThis port is for AI clients. Human players, please connect on port 33333.\r\n\r\n"), connDetails.ConnectionId())
	}

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

			// If failed to read from the connection, switch to zombie state
			if userObject != nil {

				userObject.EventLog.Add(`conn`, `Disconnected`)

				if c.Network.ZombieSeconds > 0 {

					connDetails.SetState(connections.Zombie)
					worldManager.SendSetZombie(userObject.UserId, true)

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
				userObject = uo.(*users.UserRecord)
				// Remove it from shared state if no longer needed there
				delete(sharedState, "UserObject")
			} else {
				// This shouldn't happen if the completion function worked correctly
				mudlog.Error("Login process completed but UserObject not found in sharedState", "connectionId", clientInput.ConnectionId)
				connections.Remove(clientInput.ConnectionId) // Disconnect problematic connection
				break                                        // Exit the read loop for this connection
			}

			// Warn about AI/human port mismatch
			if connDetails.ConnType() == connections.ConnAI && !userObject.IsAI {
				connections.SendTo([]byte("\r\nWarning: This account is not flagged as AI but connected on the AI port.\r\n"), connDetails.ConnectionId())
			} else if connDetails.ConnType() == connections.ConnHuman && userObject.IsAI {
				connections.SendTo([]byte("\r\nWarning: This AI account is connected on the human port. Please use the AI port.\r\n"), connDetails.ConnectionId())
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

			// AI rate limiting: enforce commands-per-round limit
			if connDetails.ConnType() == connections.ConnAI {
				netCfg := configs.GetNetworkConfig()
				currentRound := int64(util.GetRoundCount())
				if !connDetails.AICommandAllowed(currentRound, int(netCfg.AICommandsPerRound)) {
					connections.SendTo(
						[]byte(fmt.Sprintf("Command dropped — AI rate limit (%d/round). Wait for the next round.\r\n", netCfg.AICommandsPerRound)),
						connDetails.ConnectionId(),
					)
					clientInput.Reset()
					if userObject != nil {
						userObject.SetUnsentText(``, ``)
					}
					continue
				}
			}

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

	loginHandler := inputhandlers.GetLoginPromptHandler()           // Get the configured handler func
	connDetails.AddInputHandler("LoginPromptHandler", loginHandler) // Add it with a unique name

	// Describes whatever the client sent us
	clientInput := &connections.ClientInput{
		ConnectionId: connDetails.ConnectionId(),
		DataIn:       []byte{},
		Buffer:       make([]byte, 0, connections.ReadBufferSize), // DataIn is appended to this buffer after processing
		EnterPressed: false,
		Clipboard:    []byte{},
		History:      connections.InputHistory{},
	}

	connections.SendTo(
		[]byte("!!SOUND(Off U="+configs.GetConfig().FilePaths.WebCDNLocation.String()+")"),
		clientInput.ConnectionId,
	)

	plugins.OnNetConnect(connDetails)

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

	// --- Send Initial Welcome/Splash ---
	// (This part was mostly correct before)
	splashTxt, _ := templates.Process("login/connect-splash", nil)
	connections.SendTo([]byte(templates.AnsiParse(splashTxt)), connDetails.ConnectionId())

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

	c := configs.GetConfig()

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {

			// If failed to read from the connection, switch to zombie state
			if userObject != nil {

				userObject.EventLog.Add(`conn`, `Disconnected`)

				if c.Network.ZombieSeconds > 0 {

					connDetails.SetState(connections.Zombie)
					worldManager.SendSetZombie(userObject.UserId, true)

				} else {

					worldManager.SendLeaveWorld(userObject.UserId)
					worldManager.SendLogoutConnectionId(connDetails.ConnectionId())

				}

			}

			mudlog.Warn("WS Read", "error", err)
			break
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
				userObject = uo.(*users.UserRecord)
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

			clientInput.Reset()
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

func TelnetListenOnPort(hostname string, portNum int, wg *sync.WaitGroup, maxConnections int, connType ...connections.ConnType) net.Listener {

	cType := connections.ConnHuman
	if len(connType) > 0 {
		cType = connType[0]
	}

	server, err := net.Listen("tcp", fmt.Sprintf("%s:%d", hostname, portNum))
	if err != nil {
		mudlog.Error("Error creating server", "error", err)
		return nil
	}

	// Start a goroutine to accept incoming connections, so that we can use a signal to stop the server
	go func() {

		netCfg := configs.GetNetworkConfig()

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

			// Enforce overall connection limit
			if maxConnections > 0 {
				if connections.ActiveConnectionCount() >= maxConnections {
					conn.Write([]byte(fmt.Sprintf("\n\n\n!!! Server is full (%d connections). Try again later. !!!\n\n\n", connections.ActiveConnectionCount())))
					conn.Close()
					continue
				}
			}

			// Enforce per-type connection limits
			if cType == connections.ConnAI {
				if connections.ActiveAIConnectionCount() >= int(netCfg.MaxAIConnections) {
					conn.Write([]byte("\n\n\n!!! AI connection pool is full. Try again later. !!!\n\n\n"))
					conn.Close()
					continue
				}
			} else {
				if int(netCfg.MaxHumanConnections) > 0 && connections.ActiveHumanConnectionCount() >= int(netCfg.MaxHumanConnections) {
					conn.Write([]byte("\n\n\n!!! Human connection pool is full. Try again later. !!!\n\n\n"))
					conn.Close()
					continue
				}
			}

			connDetails := connections.Add(conn, nil, cType)

			// AI connections get ANSI stripping and are tagged
			if cType == connections.ConnAI {
				connDetails.SetStripAnsi(true)
			}

			wg.Add(1)
			// hand off the connection to a handler goroutine so that we can continue handling new connections
			go handleTelnetConnection(
				connDetails,
				wg,
			)

		}
	}()

	return server
}

func loadAllDataFiles(isReload bool) {

	if isReload {

		defer func() {
			if r := recover(); r != nil {
				mudlog.Error("RELOAD FAILED", "err", r)
			}
		}()

	}

	// Load biomes before rooms since rooms reference biomes
	rooms.LoadBiomeDataFiles()
	spells.LoadSpellFiles()
	mutations.LoadMutationFiles()
	enchantments.LoadEnchantmentFiles()
	crafting.LoadRecipeFiles()
	rooms.LoadDataFiles()
	rooms.RebuildZonePlayerCount() // build the zone → player-count index
	buffs.LoadDataFiles() // Load buffs before items for cost calculation reasons
	items.LoadDataFiles()
	species.LoadDataFiles()
	mobs.LoadDataFiles()
	mobs.AuditMobNameCollisions(func(name string) (int, bool) {
		if userId, _ := users.CharacterNameSearch(name); userId > 0 {
			return userId, true
		}
		return 0, false
	})
	allMobTemplates := mobs.AllMobTemplates()
	adapted := make([]shops.ShopBearingMob, 0, len(allMobTemplates))
	for _, m := range allMobTemplates {
		adapted = append(adapted, m)
	}
	if err := shops.ValidateShopMobTags(adapted); err != nil {
		// Cold boot: fail fast — server refuses to start with bad tags.
		// Reload (`/reload`): the deferred recover() at the top of this
		// function would swallow a panic into a generic "RELOAD FAILED"
		// log. Bypass that by handling reload explicitly so the admin
		// gets a structured error pointing at the offender, plus a
		// remediation hint. Mob templates may now be in an inconsistent
		// state — fix the listed YAMLs and reload again.
		if isReload {
			mudlog.Error("shops.ValidateShopMobTags failed on reload",
				"error", err.Error(),
				"remediation", "fix the listed mob YAMLs (add or correct craft_support:) and run /reload again; mob templates may be inconsistent until then",
			)
		} else {
			panic(fmt.Sprintf("shops.ValidateShopMobTags failed:\n%v", err))
		}
	}
	// Build mob-template zone lookup for shop cache pre-warming.
	mobZoneByMobId := make(map[int]string, len(allMobTemplates))
	for _, m := range allMobTemplates {
		mobZoneByMobId[int(m.MobId)] = m.Zone
	}
	if n, err := shops.PrewarmFromPersistedFiles(func(mobId int) (string, bool) {
		z, ok := mobZoneByMobId[mobId]
		return z, ok
	}); err != nil {
		mudlog.Warn("shop cache prewarm", "error", err)
	} else {
		mudlog.Info("shop cache prewarmed (persisted)", "count", n)
	}
	// Walk every room's spawninfo and pre-register a shop entry for
	// each shop-bearing mob's spawn placement that doesn't already
	// have a persisted YAML. Without this, the dashboard only shows
	// shops in zones a player has actually visited (since mob spawn
	// triggers RegisterMobShop, and mobs spawn lazily on room
	// activation). The mob isn't actually spawned — only the cache
	// entry is seeded — so this is cheap.
	templateByMobId := make(map[int]*mobs.Mob, len(allMobTemplates))
	for _, m := range allMobTemplates {
		templateByMobId[int(m.MobId)] = m
	}
	prewarmedFromSpawn := 0
	for _, roomId := range rooms.GetAllRoomIds() {
		room := rooms.LoadRoom(roomId)
		if room == nil {
			continue
		}
		for _, si := range room.SpawnInfo {
			if si.MobId == 0 {
				continue
			}
			tmpl := templateByMobId[si.MobId]
			if tmpl == nil {
				continue
			}
			if !tmpl.HasShop() && !tmpl.IsCrafter() {
				continue
			}
			mobs.PrewarmShopForSpawnPlacement(tmpl, roomId)
			prewarmedFromSpawn++
		}
	}
	mudlog.Info("shop cache prewarmed (spawninfo)", "count", prewarmedFromSpawn)
	pets.LoadDataFiles()
	quests.LoadDataFiles()
	questengine.LoadDataFiles()
	templates.LoadAliases(plugins.GetPluginRegistry())
	keywords.LoadAliases(plugins.GetPluginRegistry())
	mutators.LoadDataFiles()

	// Force-spawn long-running system NPCs at boot. The shop prewarm
	// above seeds shop cache entries from spawninfo but does NOT call
	// room.Prepare(), so the actual mob instances would otherwise only
	// be created when a player walks into the room. For caravan +
	// forager NPCs anchored in low-traffic rooms, that means their
	// state machines never start.
	// NOTE: must come after mutators.LoadDataFiles() — Prepare() calls
	// r.Mutators.Update() which dereferences mutator definitions.
	preparedAnchors := 0
	for _, roomId := range systemNPCAnchorRooms {
		room := rooms.LoadRoom(roomId)
		if room == nil {
			mudlog.Warn("system anchor room not found", "roomId", roomId)
			continue
		}
		room.Prepare(false)
		preparedAnchors++
	}
	mudlog.Info("system NPC anchor rooms prepared", "count", preparedAnchors)

	// Load sealed crates from disk and attach them to their rooms.
	// The crates/ directory mirrors shops/ — one YAML per crate,
	// named "<roomid>-<label>.yaml". Missing directory means no
	// crates exist yet, which is fine.
	crateDir := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/crates`)
	if entries, err := os.ReadDir(crateDir); err == nil {
		loadedCrates := 0
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			path := util.FilePath(crateDir, `/`, e.Name())
			c, err := sealedcrate.LoadFrom(path)
			if err != nil {
				mudlog.Warn("sealedcrate load", "path", path, "error", err)
				continue
			}
			room := rooms.LoadRoom(c.RoomId())
			if room == nil {
				mudlog.Warn("sealedcrate room missing", "roomId", c.RoomId(), "path", path)
				continue
			}
			room.AttachSealedCrate(c)
			loadedCrates++
		}
		mudlog.Info("sealed crates loaded", "count", loadedCrates)
	} else if !os.IsNotExist(err) {
		mudlog.Warn("sealedcrate dir scan", "dir", crateDir, "error", err)
	}

	colorpatterns.LoadColorPatterns()
	combat.LoadTauntMessageFiles()
	audio.LoadAudioConfig()
	characters.CompileAdjectiveSwaps() // This should come after loading color patterns.
}
