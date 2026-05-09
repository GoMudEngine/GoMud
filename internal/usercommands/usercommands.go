package usercommands

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type CommandHelpItem struct {
	Command   string
	Type      string // command/skill
	Category  string
	AdminOnly bool
}

type CommandAccess struct {
	Func              UserCommand
	AllowedWhenDowned bool
	AllowedInCombat   bool
	AdminOnly         bool
}

// Signature of user command
type UserCommand func(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error)

type FunctionExporter interface {
	GetExportedFunction(funcName string) (any, bool)
}

func AddFunctionExporter(f FunctionExporter) {
	functionExporters = append(functionExporters, f)
}

var (
	functionExporters = []FunctionExporter{}

	userCommands map[string]CommandAccess = map[string]CommandAccess{
		`alias`:           {Alias, true, true, false},
		`afk`:             {AFK, true, true, false},
		`appraise`:        {Appraise, false, true, false},
		`ask`:             {Ask, false, true, false},
		`assess`:          {Assess, false, true, false},
		`assist`:          {Assist, false, true, false},
		`attack`:          {Attack, false, true, false},
		`ai-flag`:         {AiFlag, true, true, true},      // Admin only
		`ai-list`:         {AiList, true, true, true},      // Admin only
		`badcommands`:     {BadCommands, true, true, true}, // Admin only
		`bash`:            {Bash, false, true, false},
		`blinding-flash`:  {BlindingFlash, false, true, false},
		`blinding-spit`:   {BlindingSpit, false, true, false},
		`bug`:             {Bug, true, true, false},
		`biome`:           {Biome, true, true, false},
		`broadcast`:       {Broadcast, true, true, false},
		`character`:       {Character, true, true, false},
		`bank`:            {Bank, false, true, false},
		`break`:           {Break, false, true, false},
		`build`:           {Build, false, true, true}, // Admin only
		`buff`:            {Buff, false, true, true},  // Admin only
		`buy`:             {Buy, false, true, false},
		`caravan`:         {Caravan, true, true, true}, // Admin only
		`cancel`:          {Cancel, true, true, false},
		`cast`:            {Cast, false, true, false},
		`combatstats`:     {CombatStats, true, true, true}, // Admin only
		`craft`:           {Craft, false, false, false},    // Can't start crafting in combat
		`crime`:           {Crime, true, true, true},       // Admin only
		`cooldowns`:       {Cooldowns, true, true, false},
		`command`:         {Command, false, true, true}, // Admin only
		`conditions`:      {Conditions, true, true, false},
		`companion`:       {Companion, true, true, false},
		`companions`:      {Companion, true, true, false},
		`consider`:        {Consider, true, true, false},
		`deafen`:          {Deafen, true, true, true},   // Admin only
		`devtool`:         {Devtool, false, true, true}, // Admin only
		`default`:         {Default, false, true, false},
		`deletecharacter`: {DeleteCharacter, false, false, false}, // Player account deletion
		`defuse`:          {Defuse, false, true, false},
		`disenchant`:      {Disenchant, false, false, false}, // Can't disenchant in combat
		`dismiss`:         {Dismiss, false, true, false},
		`drop`:            {Drop, true, false, false}, // Can't drop items in combat
		`drink`:           {Drink, true, true, false},
		`eat`:             {Eat, true, true, false},
		`emote`:           {Emote, true, true, false},
		`equip`:           {Equip, false, false, false}, // Can't equip in combat
		`faction`:         {Faction, true, true, true},  // Admin only
		`flee`:            {Flee, false, true, false},
		`forage`:          {Forage, false, false, false}, // Can't forage in combat
		`gearup`:          {Gearup, false, false, false}, // Can't equip in combat
		`get`:             {Get, false, false, false},    // Can't pick up items in combat
		`give`:            {Give, false, true, false},
		`go`:              {Go, false, true, false},
		`grapple`:         {Grapple, false, true, false},
		`healing-gel`:     {HealingGel, false, true, false},
		`help`:            {Help, true, true, false},
		`hint`:            {Hint, true, true, false},
		`keyring`:         {KeyRing, true, true, false},
		`kick`:            {Kick, false, true, false},
		`stomp`:           {Kick, false, true, false},
		`knee`:            {Kick, false, true, false},
		`killstats`:       {Killstats, true, true, false},
		`bounty`:          {Bounty, true, true, true},   // Admin only
		`knowledge`:       {Knowledge, true, true, true}, // Admin only
		`history`:         {History, true, true, false},
		`inbox`:           {Inbox, true, true, false},
		`inventory`:       {Inventory, true, true, false},
		`item`:            {Item, true, true, true}, // Admin only
		`title`:           {Title, true, true, false},
		`list`:            {List, false, true, false},
		`locate`:          {Locate, true, true, true}, // Admin only
		`lock`:            {Lock, false, true, false},
		`look`:            {Look, true, true, false},
		`map`:             {Map, false, true, false},
		`macros`:          {Macros, true, true, false},
		`mob`:             {Mob, true, true, true},    // Admin only
		`modify`:          {Modify, true, true, true}, // Admin only
		`motd`:            {Motd, true, true, false},
		`mudmail`:         {Mudmail, true, true, true}, // Admin only
		`mute`:            {Mute, true, true, true},
		`mutations`:       {Mutations, true, true, false},
		`noop`:            {Noop, true, true, false},
		`offer`:           {Offer, false, true, false},
		`online`:          {Online, true, true, false},
		`opinion`:         {Opinion, true, true, true}, // Admin only
		`party`:           {Party, true, true, false},
		`pacifism-aura`:   {PacifismAura, false, true, false},
		`password`:        {Password, true, true, false},
		`paz`:             {Paz, true, true, true}, // Admin only
		`pet`:             {Pet, false, true, false},
		`picklock`:        {Picklock, false, true, false},
		`steal`:           {Steal, false, true, false},
		`plant`:           {Plant, false, true, false},
		`prepare`:         {Prepare, true, true, true}, // Admin only
		`print`:           {Print, true, true, false},
		`printline`:       {PrintLine, true, true, false},
		`put`:             {Put, false, false, false}, // Can't manipulate containers in combat
		`pvp`:             {Pvp, true, true, false},
		`quests`:          {Quests, true, true, false},
		`quit`:            {Quit, true, true, false},
		`questdebug`:      {QuestDebug, false, true, true}, // Admin only
		`questtoken`:      {QuestToken, false, true, true}, // Admin only
		`read`:            {Read, false, true, false},
		`reload`:          {Reload, true, true, true}, // Admin only
		`rep`:             {Report, true, true, false},
		`reply`:           {Reply, true, true, false},
		`remove`:          {Remove, false, false, false}, // Can't remove equipment in combat
		`report`:          {Report, true, true, false},
		`rename`:          {RenameSelf, false, true, false}, // Player rename — cooldown-gated
		`renameitem`:      {Rename, false, true, true},      // Admin only — renames an item in inventory
		`redescribe`:      {Redescribe, false, true, true},  // Admin only
		`room`:            {Room, false, true, true},        // Admin only
		`save`:            {Save, true, true, false},
		`salvage`:         {Salvage, false, false, false}, // Can't salvage in combat
		`say`:             {Say, true, true, false},
		`scan`:            {Scan, false, true, false},
		`search`:          {Search, false, true, false},
		`sell`:            {Sell, false, true, false},
		`server`:          {Server, false, true, true}, // Admin only
		`set`:             {Set, true, true, false},
		`setdesc`:         {SetDesc, true, true, false},
		`sethome`:         {SetHome, true, true, false},
		`setmotd`:         {SetMotd, true, true, true}, // Admin only
		`share`:           {Share, false, true, false},
		`shoot`:           {Shoot, false, true, false},
		`shout`:           {Shout, true, true, false},
		`show`:            {Show, true, true, false},
		`skills`:          {Skills, true, true, false},
		`skillset`:        {Skillset, false, true, true}, // Admin only
		`shadow`:          {Shadow, false, false, false},
		`sneak`:           {Sneak, false, false, false}, // Can't sneak in combat
		`sonic-shout`:     {SonicShout, false, true, false},
		`spawn`:           {Spawn, false, true, true}, // Admin only
		`spell`:           {Spell, true, true, true},  // Admin only
		`spells`:          {Spells, true, true, false},
		`sort`:            {Sort, false, false, false},  // Can't sort items in combat
		`stash`:           {Stash, false, false, false}, // Can't manipulate stash in combat
		`status`:          {Status, true, true, false},
		`stand`:           {Stand, true, true, false}, // Can stand when downed
		`submit`:          {Submit, false, true, false},
		`suggest`:         {Suggest, true, true, false},
		`storage`:         {Storage, false, false, false}, // Can't manipulate storage in combat
		`suicide`:         {Suicide, true, true, false},
		`syslogs`:         {SysLogs, true, true, true}, // Admin only
		`talk`:            {Talk, false, true, false},
		`target`:          {Target, false, true, false},
		`teleport`:        {Teleport, true, true, true}, // Admin only
		`toxic-bite`:      {ToxicBite, false, true, false},
		`throw`:           {Throw, false, true, false},
		`track`:           {Track, false, true, false},
		`taunt`:           {Taunt, false, true, false},
		`trip`:            {Trip, false, true, false},
		`warcry`:          {Warcry, false, true, false},
		`rally`:           {Rally, false, true, false},
		`tailsweep`:       {Trip, false, true, false},
		`unlock`:          {Unlock, false, true, false},
		`undeafen`:        {UnDeafen, true, true, true}, // Admin only
		`unmute`:          {UnMute, true, true, true},   // Admin only
		`use`:             {Use, false, true, false},
		`whisper`:         {Whisper, true, true, false},
		`who`:             {Who, true, true, false},
		`zap`:             {Zap, true, true, true},   // Admin only
		`zone`:            {Zone, false, true, true}, // Admin only
		// Special command only used upon creating a new account
		`start`:     {Start, false, true, false},
		`zombieact`: {ZombieAct, false, true, false},
	}

	selfKeywords = []string{
		`me`,
		`self`,
		`myself`,
	}
)

func GetExportedFunction(fName string) (any, bool) {
	for _, x := range functionExporters {
		if f, ok := x.GetExportedFunction(fName); ok {
			return f, ok
		}
	}
	return nil, false
}

// Returns a list of close match commands
func GetCmdSuggestions(text string, includeAdmin bool) []string {

	text = strings.ToLower(text)

	results := []string{}

	for _, info := range keywords.GetAllHelpTopicInfo() {
		if info.AdminOnly && !includeAdmin {
			continue
		}

		testCmd := strings.ToLower(info.Command)
		if testCmd != text && strings.HasPrefix(testCmd, text) {
			results = append(results, info.Command[len(text):])
		}
	}

	for alias, _ := range keywords.GetAllCommandAliases() {
		testCmd := strings.ToLower(alias)
		if testCmd != text && strings.HasPrefix(testCmd, text) {
			results = append(results, alias[len(text):])
		}
	}

	return results
}

func GetHelpSuggestions(text string, includeAdmin bool) []string {

	results := []string{}

	for _, cmd := range keywords.GetAllHelpTopics() {
		testCmd := strings.ToLower(cmd)
		if testCmd != text && strings.HasPrefix(testCmd, text) {
			results = append(results, cmd[len(text):])
		}
	}

	for alias, _ := range keywords.GetAllHelpAliases() {
		testCmd := strings.ToLower(alias)
		if testCmd != text && strings.HasPrefix(testCmd, text) {
			results = append(results, alias[len(text):])
		}
	}

	return results
}

func TryCommand(cmd string, rest string, userId int, flags events.EventFlag) (bool, error) {

	// Load user early so we can resolve user aliases before room scripts
	user := users.GetByUserId(userId)
	if user == nil {
		return false, fmt.Errorf(`user %d not found`, userId)
	}

	// Any input clears manual AFK (except the afk command itself)
	if user.ManualAFK && cmd != "afk" {
		user.ManualAFK = false
		user.AFKMessage = ""
		user.SendText(`<ansi fg="8">You are no longer AFK.</ansi>`)
	}

	// Do not allow scripts to intercept server commands
	if cmd != `server` {

		// Resolve system aliases (e.g., n→north)
		alias := keywords.TryCommandAlias(cmd)

		// Resolve user aliases (e.g., cs→cast conviction-spike)
		// so room scripts see the actual command, not the alias
		if userAlias := user.TryCommandAlias(cmd); userAlias != cmd {
			if strings.Contains(userAlias, ` `) {
				parts := strings.Split(userAlias, ` `)
				cmd = parts[0]
				if len(rest) > 0 {
					rest = strings.TrimPrefix(userAlias, cmd+` `) + ` ` + rest
				} else {
					rest = strings.TrimPrefix(userAlias, cmd+` `)
				}
			} else {
				cmd = userAlias
			}
			// Re-resolve system aliases after user alias expansion
			alias = keywords.TryCommandAlias(cmd)
		}

		// Re-resolve system aliases on the (possibly user-alias-expanded) cmd
		if sysAlias := keywords.TryCommandAlias(cmd); sysAlias != cmd {
			if strings.Contains(sysAlias, ` `) {
				parts := strings.Split(sysAlias, ` `)
				cmd = parts[0]
				if len(rest) > 0 {
					rest = strings.TrimPrefix(sysAlias, cmd+` `) + ` ` + rest
				} else {
					rest = strings.TrimPrefix(sysAlias, cmd+` `)
				}
			} else {
				cmd = sysAlias
			}
			alias = keywords.TryCommandAlias(cmd)
		}

		skipScript := flags.Has(events.CmdSkipScripts)
		if info, ok := userCommands[alias]; ok && info.AdminOnly {
			skipScript = true
		}

		// Room behavior tree: check before room scripts and command dispatch
		if !skipScript {
			if behaviortree.TryRoomBehavior(user.Character.RoomId, behaviortree.EventContext{
				EventType: "room_command",
				UserId:    user.UserId,
				RoomId:    user.Character.RoomId,
				Command:   cmd,
				Rest:      rest,
			}) {
				return true, nil // room tree intercepted the command
			}
		}

		if !skipScript {
			// Quest engine: room_interact notification
			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
				bridge := questengine.NewGameBridge(user, room.RoomId)
				result := questengine.GetEngine().Notify("room_interact", questengine.EventDetails{
					UserId: user.UserId,
					RoomId: room.RoomId,
					Verb:   alias,
					Noun:   strings.ToLower(rest),
				}, bridge, bridge)
				if result.Handled {
					return true, nil
				}
			}
		}

	}

	userDisabled := false

	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return false, fmt.Errorf(`room %d not found`, user.Character.RoomId)
	}

	rest = strings.TrimSpace(rest)

	// Figure out whether it was an exit entered
	exitName, _ := room.FindExitByName(cmd)
	if exitName != `` {
		rest = cmd
		cmd = "go"
	} else {

		userDisabled = user.Character.IsDisabled()

	}

	// Cancel any buffs they have that get cancelled based on them doing anything at all
	user.Character.CancelBuffsWithFlag(buffs.CancelOnAction)

	// Fold-casting intercept: while holding folds, most action commands are blocked.
	// Informational commands (AllowedWhenDowned=true) pass through.
	// 'cancel' always allowed (to stop casting).
	// 'flee' clears the cast and then proceeds.
	if user.Character.CastingState != nil {
		if cmd == `flee` {
			cs := user.Character.CastingState
			user.Character.CastingState = nil
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">You lose your concentration as you flee! %d conviction is lost.</ansi>`,
				cs.ConvictionSpent))
			room.SendTextVisual(fmt.Sprintf(
				`<ansi fg="username">%s</ansi> breaks their concentration.`,
				user.Character.Name), user.UserId)
			// Fall through — let the flee command execute normally
		} else if cmd != `cancel` {
			if cmdInfo, hasCmdInfo := userCommands[cmd]; !hasCmdInfo || !cmdInfo.AllowedWhenDowned {
				user.SendText(fmt.Sprintf(
					`<ansi fg="cyan">You are holding <ansi fg="cyan-bold">%d/%d</ansi> folds. Type <ansi fg="cyan-bold">cancel</ansi> to stop.</ansi>`,
					user.Character.CastingState.FoldsAccumulated,
					user.Character.CastingState.FoldsNeeded))
				return true, nil
			}
		}
	}

	// Experimental, not sure if will have unexpected consequences.
	// Turn keywords for targetting self into actual string of self
	if cmd == `look` || cmd == `cast` {
		for _, selfWord := range selfKeywords {
			wordLen := len(selfWord)
			if rest == selfWord {
				rest = user.Character.Name
				break
			} else if len(rest) >= wordLen+1 && rest[len(rest)-(wordLen+1):] == ` `+selfWord {
				rest = rest[:len(rest)-(wordLen+1)] + ` ` + user.Character.Name
				break
			}
		}
	}

	if cmdInfo, ok := userCommands[cmd]; ok {

		if !cmdInfo.AllowedWhenDowned {

			// If actually downed, prevent it (unless admin)
			if userDisabled && !cmdInfo.AdminOnly {
				user.SendText("You are unable to do that while downed.")
				return true, nil
			}

			// Disabled input affects commands which can't be performed when downed.
			if user.InputBlocked() {
				return true, nil
			}
		}

		// Check if command is allowed during combat
		if !cmdInfo.AllowedInCombat {
			// If in combat, prevent it (unless admin)
			if user.Character.Aggro != nil && !cmdInfo.AdminOnly {
				user.SendText("You can't do that while fighting!")
				return true, nil
			}
		}

		if !cmdInfo.AdminOnly || user.HasRolePermission(cmd, true) {

			start := time.Now()
			defer func() {
				util.TrackTime(`usr-cmd[`+cmd+`]`, time.Since(start).Seconds())
			}()

			if cmdInfo.AdminOnly {
				mudlog.Info("Admin Command", "cmd", cmd, "rest", rest, "userId", user.UserId)
			}

			// Run the command here
			handled, err := cmdInfo.Func(rest, user, room, flags)
			return handled, err

		}
	}

	if _, ok := actions.EmoteAliases[cmd]; ok {
		handled, err := Emote(cmd, user, room, flags)
		return handled, err
	}

	if user.Character.HasSpell(cmd) {
		castCmd := cmd
		if len(rest) > 0 {
			castCmd += ` ` + rest
		}
		return Cast(castCmd, user, room, flags)
	}

	// "go" attempt
	start := time.Now()
	defer func() {
		util.TrackTime(`usr-cmd[go]`, time.Since(start).Seconds())
	}()

	if handled, err := Go(cmd, user, room, flags); handled {
		return handled, err
	}
	// end "go" attempt

	return false, nil
}

// GetAllUserCommands returns the names of all registered user commands.
func GetAllUserCommands() []string {
	result := []string{}
	for cmd := range userCommands {
		result = append(result, cmd)
	}
	return result
}

// IsAdminCommand reports whether the named command is admin-only.
func IsAdminCommand(cmd string) bool {
	if info, ok := userCommands[cmd]; ok {
		return info.AdminOnly
	}
	return false
}

// GetCommandRegistry returns a snapshot of all registered commands for external use.
// Callers should filter AdminOnly entries as appropriate.
func GetCommandRegistry() map[string]CommandAccess {
	snapshot := make(map[string]CommandAccess, len(userCommands))
	for k, v := range userCommands {
		snapshot[k] = v
	}
	return snapshot
}

// Register mob commands from outside of the package
func RegisterCommand(command string, handlerFunc UserCommand, disabledWhenDowned bool, allowedInCombat bool, isAdminOnly bool) {
	userCommands[command] = CommandAccess{
		handlerFunc,
		disabledWhenDowned,
		allowedInCombat,
		isAdminOnly,
	}
}
