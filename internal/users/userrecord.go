package users

import (
	"crypto/md5"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/audio"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/prompt"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
	//
)

var (
	// immutable roles
	RoleGuest string = "guest"
	RoleUser  string = "user"
	RoleAdmin string = "admin"
)

type UserRecord struct {
	UserId         int                   `yaml:"userid"`
	Role           string                `yaml:"role"` // Roles group one or more admin commands
	Username       string                `yaml:"username"`
	Password       string                `yaml:"password"`
	Joined         time.Time             `yaml:"joined"`
	Macros         map[string]string     `yaml:"macros,omitempty"`  // Up to 10 macros, just string commands.
	Aliases        map[string]string     `yaml:"aliases,omitempty"` // string=>string remapping of commands
	Character      *characters.Character `yaml:"character,omitempty"`
	ItemStorage    Storage               `yaml:"itemstorage,omitempty"`
	ConfigOptions  map[string]any        `yaml:"configoptions,omitempty"`
	Inbox          Inbox                 `yaml:"inbox,omitempty"`
	Muted          bool                  `yaml:"muted,omitempty"`        // Cannot SEND custom communications to anyone but admin/mods
	Deafened       bool                  `yaml:"deafened,omitempty"`     // Cannot HEAR custom communications from anyone but admin/mods
	IsAI           bool                  `yaml:"isai,omitempty"`         // Flagged as an AI account
	ScreenReader   bool                  `yaml:"screenreader,omitempty"` // Are they using a screen reader? (We should remove excess symbols)
	EmailAddress   string                `yaml:"emailaddress,omitempty"` // Email address (if provided)
	TipsComplete   map[string]bool       `yaml:"tipscomplete,omitempty"` // Tips the user has followed/completed so they can be quiet
	EventLog        UserLog               `yaml:"-"` // Do not retain in user file (for now)
	LastMusic       string                `yaml:"-"` // Keeps track of the last music that was played
	LastWhisperFrom int                   `yaml:"-"` // UserId of last person who whispered to us (don't save)
	connectionId   uint64
	unsentText     string
	suggestText    string
	connectionTime time.Time
	lastInputRound uint64
	tempDataStore  map[string]any
	activePrompt   *prompt.Prompt
	isZombie       bool // are they a zombie currently?
	inputBlocked   bool // Whether input is currently intentionally turned off (for a certain category of commands)
}

func NewUserRecord(userId int, connectionId uint64) *UserRecord {

	c := configs.GetGamePlayConfig()

	u := &UserRecord{
		connectionId:   connectionId,
		UserId:         userId,
		Role:           RoleUser,
		Username:       "",
		Password:       "",
		Macros:         make(map[string]string),
		Character:      characters.New(),
		ConfigOptions:  map[string]any{},
		Joined:         time.Now(),
		connectionTime: time.Now(),
		tempDataStore:  make(map[string]any),
		EventLog:       UserLog{},
	}

	if c.Death.PermaDeath {
		u.Character.ExtraLives = int(c.LivesStart)
	}

	return u
}

func (u *UserRecord) ClientSettings() connections.ClientSettings {
	return connections.GetClientSettings(u.connectionId)
}

func (u *UserRecord) PasswordMatches(input string) bool {

	if input == u.Password {
		return true
	}

	if u.Password == util.Hash(input) {
		return true
	}

	// In case we reset the password to a plaintext string
	if input == util.Hash(u.Password) {
		return true
	}

	return false
}

func (u *UserRecord) AddCommandAlias(input string, output string) (addedAlias string, deletedAlias string) {

	if u.Aliases == nil {
		u.Aliases = map[string]string{}
	}

	input = strings.ToLower(strings.TrimSpace(input))
	if input == `alias` {
		return
	}

	if output == `` {
		delete(u.Aliases, input)
		return ``, input
	}

	if len(input) >= 64 {
		input = input[0:64]
	}

	if len(output) >= 64 {
		output = output[0:64]
	}

	u.Aliases[input] = strings.TrimSpace(output)

	return input, ``
}

func (u *UserRecord) TryCommandAlias(input string) string {

	if u.Aliases == nil {
		u.Aliases = map[string]string{}
		return input
	}

	if alias, ok := u.Aliases[strings.ToLower(input)]; ok {
		return alias
	}

	return input
}

func (u *UserRecord) ShorthandId() string {
	return fmt.Sprintf(`@%d`, u.UserId)
}

func (u *UserRecord) SetLastInputRound(rdNum uint64) {
	u.lastInputRound = rdNum
}

func (u *UserRecord) GetLastInputRound() uint64 {
	return u.lastInputRound
}

func (u *UserRecord) HasShop() bool {
	return len(u.Character.Shop) > 0
}

func (u *UserRecord) DidTip(tipName string, completed ...bool) bool {

	if u.TipsComplete == nil {
		u.TipsComplete = map[string]bool{}
	}

	if len(completed) > 0 {
		if completed[0] {
			u.TipsComplete[tipName] = completed[0]
		} else {
			delete(u.TipsComplete, tipName)
		}
		return completed[0]
	}

	return u.TipsComplete[tipName]
}

func (u *UserRecord) PlayMusic(musicFileOrId string) {

	v := 100
	if soundConfig := audio.GetFile(musicFileOrId); soundConfig.FilePath != `` {
		musicFileOrId = soundConfig.FilePath
		if soundConfig.Volume > 0 && soundConfig.Volume <= 100 {
			v = soundConfig.Volume
		}
	}

	events.AddToQueue(events.MSP{
		UserId:    u.UserId,
		SoundType: `MUSIC`,
		SoundFile: musicFileOrId,
		Volume:    v,
	})

}

func (u *UserRecord) PlaySound(soundId string, category string) {

	v := 100
	if soundConfig := audio.GetFile(soundId); soundConfig.FilePath != `` {
		soundId = soundConfig.FilePath
		if soundConfig.Volume > 0 && soundConfig.Volume <= 100 {
			v = soundConfig.Volume
		}
	}

	events.AddToQueue(events.MSP{
		UserId:    u.UserId,
		SoundType: `SOUND`,
		SoundFile: soundId,
		Volume:    v,
		Category:  category,
	})

}

func (u *UserRecord) Command(inputTxt string, waitSeconds ...float64) {

	readyTurn := util.GetTurnCount()
	if len(waitSeconds) > 0 {
		readyTurn += uint64(float64(configs.GetTimingConfig().SecondsToTurns(1)) * waitSeconds[0])
	}

	events.AddToQueue(events.Input{
		UserId:    u.UserId,
		InputText: inputTxt,
		ReadyTurn: readyTurn,
	})

}

func (u *UserRecord) BlockInput() {
	u.inputBlocked = true
}

func (u *UserRecord) UnblockInput() {
	u.inputBlocked = false
}

func (u *UserRecord) InputBlocked() bool {
	return u.inputBlocked
}

func (u *UserRecord) CommandFlagged(inputTxt string, flagData events.EventFlag, waitSeconds ...float64) {

	readyTurn := util.GetTurnCount()
	if len(waitSeconds) > 0 {
		readyTurn += uint64(float64(configs.GetTimingConfig().SecondsToTurns(1)) * waitSeconds[0])
	}

	if flagData&events.CmdBlockInput == events.CmdBlockInput {
		u.BlockInput()
	}

	events.AddToQueue(events.Input{
		UserId:    u.UserId,
		InputText: inputTxt,
		ReadyTurn: readyTurn,
		Flags:     flagData,
	})

}

func (u *UserRecord) AddBuff(buffId int, source string) {

	events.AddToQueue(events.Buff{
		UserId: u.UserId,
		BuffId: buffId,
		Source: source,
	})

}

func (u *UserRecord) SendText(txt string) {

	events.AddToQueue(events.Message{
		UserId: u.UserId,
		Text:   txt + "\n",
	})

}

func (u *UserRecord) SendWebClientCommand(txt string) {

	events.AddToQueue(events.WebClientCommand{
		ConnectionId: u.connectionId,
		Text:         txt,
	})

}

func (u *UserRecord) SetTempData(key string, value any) {

	if u.tempDataStore == nil {
		u.tempDataStore = make(map[string]any)
	}

	if value == nil {
		delete(u.tempDataStore, key)
		return
	}
	u.tempDataStore[key] = value
}

func (u *UserRecord) GetTempData(key string) any {

	if u.tempDataStore == nil {
		u.tempDataStore = make(map[string]any)
	}

	if value, ok := u.tempDataStore[key]; ok {
		return value
	}
	return nil
}

func (u *UserRecord) HasRolePermission(permissionId string, simpleMatch ...bool) bool {

	if u.Role == RoleAdmin {
		return true
	}

	if len(simpleMatch) == 0 {
		mudlog.Info("RoleCheck", "permissionId", permissionId, "userId", u.UserId, "username", u.Username, "characterName", u.Character.Name)
	}

	if u.Role == RoleUser {
		return false
	}

	roles := configs.GetRolesConfig()
	commandList, ok := roles[u.Role]
	if !ok {
		return false
	}

	permissionIdLen := len(permissionId)
	cmdLen := 0
	for _, cmdAccessId := range commandList {

		mudlog.Info("RoleCheck", "comparing", cmdAccessId, "to", permissionId)
		// room.info vs room
		if permissionId == cmdAccessId {
			return true
		}

		cmdLen = len(cmdAccessId)

		// For helpfiles we match any portion
		if len(simpleMatch) > 0 && simpleMatch[0] {

			// room vs room.info
			if permissionIdLen < cmdLen {
				if cmdAccessId[0:permissionIdLen] == permissionId {
					return true
				}
			} else if permissionIdLen > cmdLen {
				if permissionId[0:permissionIdLen] == cmdAccessId {
					return true
				}
			}
		}

		// If the permissionId is shorter than their permission on this, skip it
		if permissionIdLen < cmdLen {
			continue
		}

		if permissionId[0:cmdLen] == cmdAccessId {
			return true
		}
	}

	return false
}

func (u *UserRecord) SetConfigOption(key string, value any) {
	if u.ConfigOptions == nil {
		u.ConfigOptions = make(map[string]any)
	}
	if value == nil {
		delete(u.ConfigOptions, key)
		return
	}
	u.ConfigOptions[key] = value
}

func (u *UserRecord) GetConfigOption(key string) any {
	if u.ConfigOptions == nil {
		u.ConfigOptions = make(map[string]any)
	}
	if value, ok := u.ConfigOptions[key]; ok {
		return value
	}
	return nil
}

func (u *UserRecord) GetConnectTime() time.Time {
	return u.connectionTime
}

func (u *UserRecord) RoundTick() {

}

// The purpose of SetUnsentText(), GetUnsentText() is to
// Capture what the user is typing so that when we redraw the
// "prompt" or status bar, we can redraw what they were in the middle
// of typing.
// I don't like the idea of capturing it every time they hit a key though
// There is probably a better way.
func (u *UserRecord) SetUnsentText(t string, suggest string) {

	u.unsentText = t
	u.suggestText = suggest
}

func (u *UserRecord) GetUnsentText() (unsent string, suggestion string) {

	return u.unsentText, u.suggestText
}

// Replace a characters information with another.
func (u *UserRecord) ReplaceCharacter(replacement *characters.Character) {
	u.Character = replacement
}

func (u *UserRecord) SetUsername(un string) error {

	if err := ValidateName(un); err != nil {
		return err
	}

	u.Username = un

	// If no character name, just set it to username for now.
	if u.Character.Name == "" {
		u.Character.Name = u.TempName()
	}

	return nil
}

func (u *UserRecord) TempName() string {
	hasher := md5.New()
	hasher.Write([]byte([]byte(u.Username)))
	hashInBytes := hasher.Sum(nil)
	number := new(big.Int).SetBytes(hashInBytes)

	mod := new(big.Int)
	mod.SetInt64(9087919)

	return fmt.Sprintf("nameless-%d", number.Mod(number, mod).Int64())
}

func (u *UserRecord) SetCharacterName(cn string) error {

	if err := ValidateName(cn); err != nil {
		return err
	}

	u.Character.Name = cn

	return nil
}

func (u *UserRecord) SetPassword(pw string) error {

	validation := configs.GetValidationConfig()

	if len(pw) < int(validation.PasswordSizeMin) || len(pw) > int(validation.PasswordSizeMax) {
		return fmt.Errorf("password must be between %d and %d characters long", validation.PasswordSizeMin, validation.PasswordSizeMax)
	}

	u.Password = util.Hash(pw)
	return nil
}

func (u *UserRecord) ConnectionId() uint64 {
	return u.connectionId
}

// Prompt related functionality
func (u *UserRecord) StartPrompt(command string, rest string) (*prompt.Prompt, bool) {

	if u.activePrompt != nil {
		// If it's the same prompt, return the existing one
		if u.activePrompt.Command == command && u.activePrompt.Rest == rest {
			return u.activePrompt, false
		}
	}

	// If no prompt found or it seems like a new prompt, create a new one and replace the old
	u.activePrompt = prompt.New(command, rest)

	return u.activePrompt, true
}

func (u *UserRecord) GetPrompt() *prompt.Prompt {

	return u.activePrompt
}

func (u *UserRecord) ClearPrompt() {
	u.activePrompt = nil
}

func (u *UserRecord) GetOnlineInfo() OnlineInfo {
	c := configs.GetTimingConfig()
	afkRounds := uint64(c.SecondsToRounds(int(configs.GetNetworkConfig().AfkSeconds)))
	roundNow := util.GetRoundCount()

	connTime := u.GetConnectTime()

	oTime := time.Since(connTime)

	h := int(math.Floor(oTime.Hours()))
	m := int(math.Floor(oTime.Minutes())) - (h * 60)
	s := int(math.Floor(oTime.Seconds())) - (h * 60 * 60) - (m * 60)

	timeStr := ``
	if h > 0 {
		timeStr = fmt.Sprintf(`%dh%dm`, h, m)
	} else if m > 0 {
		timeStr = fmt.Sprintf(`%dm`, m)
	} else {
		timeStr = fmt.Sprintf(`%ds`, s)
	}

	isAfk := false
	if afkRounds > 0 && roundNow-u.GetLastInputRound() >= afkRounds {
		isAfk = true
	}

	isAI := false
	if cd := connections.Get(u.connectionId); cd != nil {
		isAI = cd.ConnType() == connections.ConnAI
	}

	return OnlineInfo{
		Username:      u.Username,
		CharacterName: u.Character.Name,
		Title:         skills.GetTitle(u.Character.Mutations, u.Character.GetAllSkillRanks(), u.Character.Stats),
		OnlineTime:    int64(oTime.Seconds()),
		OnlineTimeStr: timeStr,
		IsAFK:         isAfk,
		IsAI:          isAI,
		Role:          u.Role,
	}
}

func (u *UserRecord) WimpyCheck() {
	if currentWimpy := u.GetConfigOption(`wimpy`); currentWimpy != nil {
		healthPct := int(math.Floor(float64(u.Character.Health) / float64(u.Character.HealthMax.Value) * 100))
		if healthPct < currentWimpy.(int) {
			u.Command(`flee`, -1)
		}
	}
}

func (u *UserRecord) SwapToAlt(targetAltName string) bool {

	altNames := []string{}
	nameToAlt := map[string]characters.Character{}

	for _, char := range characters.LoadAlts(u.UserId) {
		altNames = append(altNames, char.Name)
		nameToAlt[char.Name] = char
	}

	match, closeMatch := util.FindMatchIn(targetAltName, altNames...)
	if match == `` {
		match = closeMatch
	}

	if match == `` {
		return false
	}

	selectedChar, ok := nameToAlt[match]
	if !ok {
		return false
	}

	retiredCharName := u.Character.Name

	newAlts := []characters.Character{}
	for _, altChar := range nameToAlt {
		if altChar.Name != selectedChar.Name {
			newAlts = append(newAlts, altChar)
		}
	}

	// add current char to the alts
	newAlts = append(newAlts, *u.Character)
	// Write them to disc
	characters.SaveAlts(u.UserId, newAlts)

	// Run validation on the new character
	selectedChar.Validate()

	// Set userId
	selectedChar.SetUserId(u.UserId)

	// Replace the current character (has already been written to alts)
	u.Character = &selectedChar

	SaveUser(*u)

	events.AddToQueue(events.CharacterChanged{UserId: u.UserId, LastCharacterName: retiredCharName, CharacterName: u.Character.Name})

	return true
}
