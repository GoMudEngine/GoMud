package events

import (
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// EVENT DEFINITIONS FOLLOW
// NOTE: If you give an event the following receiver function: `UniqueID() string`
//
//	      It will become a "unique event", meaning only one can be in the event queue
//			 at a time matching the string return value.
//		 Example: See `RedrawPrompt`
//
// Used to apply or remove buffs
type Buff struct {
	UserId        int
	MobInstanceId int
	BuffId        int
	Source        string // optional source such as spell,
}

func (b Buff) Type() string { return `Buff` }

type BuffsTriggered struct {
	UserId        int
	MobInstanceId int
	BuffIds       []int
}

func (b BuffsTriggered) Type() string { return `BuffsTriggered` }

// Used for giving/taking quest progress
type Quest struct {
	UserId     int
	QuestToken string
}

func (q Quest) Type() string { return `Quest` }

// For special room-targetting actions
type RoomAction struct {
	RoomId       int
	SourceUserId int
	SourceMobId  int
	Action       string
	Details      any
	ReadyTurn    uint64
}

func (r RoomAction) Type() string { return `RoomAction` }

// Used for Input from players/mobs
type Input struct {
	UserId        int
	MobInstanceId int
	InputText     string
	ReadyTurn     uint64
	Flags         EventFlag
}

func (i Input) Type() string { return `Input` }

// When a skill is used by a player
type SkillUsed struct {
	UserId  int
	Skill   skills.SkillTag
	Details string // usually the specific sub-command of the skill
}

func (i SkillUsed) Type() string { return `SkillUsed` }

// Messages that are intended to reach all users on the system
type Broadcast struct {
	Text             string
	TextScreenReader string // optional text for screenreader friendliness
	IsCommunication  bool
	SourceIsMod      bool
	SkipLineRefresh  bool
}

func (b Broadcast) Type() string { return `Broadcast` }

type Message struct {
	UserId          int
	ExcludeUserIds  []int
	RoomId          int
	Text            string
	IsQuiet         bool // whether it can only be heard by superior "hearing"
	IsCommunication bool // If true, this is a communication such as "say" or "emote"
}

func (m Message) Type() string { return `Message` }

type Communication struct {
	SourceUserId        int    // User that sent the message
	SourceMobInstanceId int    // Mob that sent the message
	TargetUserId        int    // Sent to only 1 person
	CommType            string // say, party, broadcast, whisper, shout
	Name                string
	Message             string
}

func (m Communication) Type() string { return `Communication` }

// Special commands that only the webclient is equipped to handle
type WebClientCommand struct {
	ConnectionId uint64
	Text         string
}

func (w WebClientCommand) Type() string { return `WebClientCommand` }

// Messages that are intended to reach all users on the system
type System struct {
	Command     string
	Data        any
	Description string
}

func (s System) Type() string { return `System` }

// Payloads describing sound/music to play
type MSP struct {
	UserId    int
	SoundType string // SOUND or MUSIC
	SoundFile string
	Volume    int    // 1-100
	Category  string // special category/type for MSP string
}

func (m MSP) Type() string { return `MSP` }

// Fired whenever a mob or player changes rooms
type RoomChange struct {
	UserId        int
	MobInstanceId int
	FromRoomId    int
	ToRoomId      int
	Unseen        bool
}

func (r RoomChange) Type() string { return `RoomChange` }

// Fired every new round
type NewRound struct {
	RoundNumber uint64
	TimeNow     time.Time
}

func (n NewRound) Type() string { return `NewRound` }

// Each new turn (TurnMs in config.yaml)
type NewTurn struct {
	TurnNumber uint64
	TimeNow    time.Time
}

func (n NewTurn) Type() string { return `NewTurn` }

// Anytime a mob is idle
type MobIdle struct {
	MobInstanceId int
}

func (i MobIdle) Type() string { return `MobIdle` }

// Gained or lost an item
type EquipmentChange struct {
	UserId        int
	MobInstanceId int
	GoldChange    int
	BankChange    int
	ItemsWorn     []items.Item
	ItemsRemoved  []items.Item
}

func (i EquipmentChange) Type() string { return `EquipmentChange` }

// Gained or lost an item
type ItemOwnership struct {
	UserId        int
	MobInstanceId int
	Item          items.Item
	Gained        bool
}

func (i ItemOwnership) Type() string { return `ItemOwnership` }

// Triggered by a script
type ScriptedEvent struct {
	Name string
	Data map[string]any
}

func (s ScriptedEvent) Type() string { return `ScriptedEvent` }

// Entered the world
type PlayerSpawn struct {
	UserId        int
	ConnectionId  uint64
	RoomId        int
	Username      string
	CharacterName string
}

func (p PlayerSpawn) Type() string { return `PlayerSpawn` }

// Left the world
type PlayerDespawn struct {
	UserId        int
	RoomId        int
	Username      string
	CharacterName string
	TimeOnline    string
}

func (p PlayerDespawn) Type() string { return `PlayerDespawn` }

type Log struct {
	FollowAdd    connections.ConnectionId
	FollowRemove connections.ConnectionId
	Level        string
	Data         []any
}

func (l Log) Type() string { return `Log` }

type PlayerDrop struct {
	UserId int
	RoomId int
}

func (l PlayerDrop) Type() string { return `PlayerDrop` }

type PlayerDeath struct {
	UserId        int
	RoomId        int
	Username      string
	CharacterName string
	Permanent     bool
	KilledByUsers []int
}

func (l PlayerDeath) Type() string { return `PlayerDeath` }

type MobDeath struct {
	MobId         int
	InstanceId    int
	RoomId        int
	CharacterName string
	PlayerDamage  map[int]int
}

func (l MobDeath) Type() string { return `MobDeath` }

type DayNightCycle struct {
	IsSunrise bool
	Day       int
	Month     int
	Year      int
	Time      string
}

func (l DayNightCycle) Type() string { return `DayNightCycle` }

// MoonPhase is fired when a Witness crosses a new-moon or full-moon boundary.
type MoonPhase struct {
	MoonName  string // "Swiftmoon" | "The Wanderer" | "The Eye"
	PhaseName string // "new" | "full"
	IsFull    bool
	IsNew     bool
}

func (m MoonPhase) Type() string { return `MoonPhase` }

type Looking struct {
	UserId int
	RoomId int
	Target string
	Hidden bool
}

func (l Looking) Type() string { return `Looking` }

// Fired after creating a new character and giving the character a name.
type CharacterCreated struct {
	UserId        int
	CharacterName string
}

func (p CharacterCreated) Type() string { return `CharacterCreated` }

// Fired when a character alt change has occured.
type CharacterChanged struct {
	UserId            int
	LastCharacterName string
	CharacterName     string
}

func (p CharacterChanged) Type() string { return `CharacterChanged` }

type UserSettingChanged struct {
	UserId int
	Name   string
}

func (i UserSettingChanged) Type() string { return `UserSettingChanged` }

// Health, mana, etc.
type CharacterVitalsChanged struct {
	UserId int
}

func (p CharacterVitalsChanged) Type() string { return `CharacterVitalsChanged` }

// Health, mana, etc.
type CharacterTrained struct {
	UserId int
}

func (p CharacterTrained) Type() string { return `CharacterTrained` }

// any stats or healthmax etc. have changed
type CharacterStatsChanged struct {
	UserId int
}

func (p CharacterStatsChanged) Type() string { return `CharacterStatsChanged` }

// any stats or healthmax etc. have changed
type PartyUpdated struct {
	Action  string // create, disband, membership
	UserIds []int
}

func (p PartyUpdated) Type() string { return `PartyUpdated` }

type Party struct {
	LeaderUserId  int
	UserIds       []int
	InviteUserIds []int
	AutoAttackers []int
	Position      map[int]string
}

func (p Party) Type() string { return `Party` }

// Rebuilds mapper for a given RoomId
// NOTE: RoomId should USUALLY be the Room's Zone.RootRoomId
type RebuildMap struct {
	MapRootRoomId int
	SkipIfExists  bool
}

func (r RebuildMap) Type() string { return `RebuildMap` }
func (r RebuildMap) UniqueID() string {
	return `RebuildMap-` + strconv.Itoa(r.MapRootRoomId) + `-` + strconv.FormatBool(r.SkipIfExists)
}

type RedrawPrompt struct {
	UserId        int
	OnlyIfChanged bool
}

func (l RedrawPrompt) Type() string     { return `RedrawPrompt` }
func (l RedrawPrompt) UniqueID() string { return `RedrawPrompt-` + strconv.Itoa(l.UserId) }
