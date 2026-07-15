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

// AutomationChanged fires when a user's macros/aliases/ticks/triggers change,
// so the Char.Automation GMCP payload can be re-pushed.
type AutomationChanged struct {
	UserId int
}

func (a AutomationChanged) Type() string { return `AutomationChanged` }

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

// ChannelMessage is a global chat-channel line for terminal fan-out. Recipients
// are filtered per-user by their channel toggle in ChannelMessage_SendToAll. The
// web/GMCP delivery goes through the separate Communication event.
type ChannelMessage struct {
	Channel      string // channel name, e.g. "newbie"
	SourceUserId int
	Name         string // sender display name
	Text         string // fully-formatted, ansi-tagged line (ends with CRLF)
}

func (c ChannelMessage) Type() string { return `ChannelMessage` }

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

// PatrolWaypointArrival fires once when a patrol-running mob reaches a
// waypoint room and enters the dwell phase. Consumers filter by
// ArrivalEvent name. Empty ArrivalEvent fires regardless — useful for
// debug subscribers but skipped by name-filtered consumers. Chunk 3.7.
type PatrolWaypointArrival struct {
	MobInstanceId int
	PatrolId      string
	WaypointIdx   int
	RoomId        int
	ArrivalEvent  string
}

func (e PatrolWaypointArrival) Type() string { return "PatrolWaypointArrival" }

// PatrolCompleted fires once when a oneshot patrol exhausts its last
// waypoint's dwell. Consumers (outer state machines) read this to
// advance their own state. The patrol executor clears the mob's
// PatrolId before emitting, so the mob is back in the "no patrol"
// state when the listener sees the event. Chunk 3.8.
type PatrolCompleted struct {
	MobInstanceId int
	PatrolId      string
	RoomId        int
}

func (e PatrolCompleted) Type() string { return "PatrolCompleted" }

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

// StorageItemSeized is emitted by the storage-fee hook when a stored slot is
// seized from a player who can't pay their bank-storage rent AND the slot's
// aggregate value (spec.Value * Count) clears the StorageSeizureMinValue floor.
// The auctions module listens and enqueues it onto the auction block. Sub-floor
// slots are disposed by the hook and never emit this event.
type StorageItemSeized struct {
	UserId int        // ex-owner; surplus after the lien returns here
	Item   items.Item // the seized item (a single representative of the stack)
	Count  int        // stack count seized (>=1); the winner receives all Count units
	Owed   int        // this lot's lien — unpaid rent to recoup from the sale before surplus
}

func (s StorageItemSeized) Type() string { return `StorageItemSeized` }

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
	// KillerMobInstanceId is the InstanceId of the mob that dealt the
	// killing blow, or 0 if the killer was a player (or attribution is
	// unclear). Populated from the dead mob's Aggro.MobInstanceId at
	// the moment of death — imperfect (last-aggro-target, not last-hit),
	// but workable for 4.5 seeder rules. Chunk 4.5.
	KillerMobInstanceId int
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

// MobAISignal is fired when a combat-relevant event occurs for a mob's AI system.
type MobAISignal struct {
	MobInstanceId int
	SignalType    string // "damage_taken", "combat_start", "target_fled", "action_complete", "player_entered"
	Detail        string
	RoomId        int
}

func (m MobAISignal) Type() string { return "MobAISignal" }

// PartyHelpRequested is fired when a party member calls for help via
// the party_call_help behavior tree action. Other party members'
// behavior trees can listen for this event to navigate to the
// rally room.
type PartyHelpRequested struct {
	PartyId        int  // internal numeric ID; see parties.Party.PartyIdInternal()
	CallerActorId  int  // user or mob instance ID; interpret via CallerIsPlayer
	CallerIsPlayer bool
	RallyRoomId    int
}

func (p PartyHelpRequested) Type() string { return "PartyHelpRequested" }

// PartyDissolved is fired when a party's leader dies or the party is
// explicitly disbanded. Member behavior trees can listen for this to
// react ("morale break" emote, panic flee, etc.) before reverting to
// solo behavior.
type PartyDissolved struct {
	PartyId        int
	Reason         string // "leader_died" | "disbanded" | "all_dead"
	MemberActorIds []int
}

func (p PartyDissolved) Type() string { return "PartyDissolved" }

// PlayerAttackedMob fires when a player initiates a combat action
// (attack, taunt, etc.) against a mob. Consumed by chunk-4.5 seeders:
//   - aggressive_action_to_revenge (rule 6) — seeds revenge-mob into
//     the attacked mob + non-hostile witnesses.
//   - combat_assist_to_opinion_boost (rule 9) — if the attacked mob
//     was already engaged with another non-player mob, bumps the
//     beneficiary's opinion of the attacking player.
//
// Chunk 4.5.
type PlayerAttackedMob struct {
	UserId        int // player initiating the attack
	MobInstanceId int // mob being attacked
}

func (p PlayerAttackedMob) Type() string { return "PlayerAttackedMob" }

// GiftOffered fires when a player runs `give <item> <mob>`. Fires
// unconditionally on every item give to a mob. No consumers in chunk
// 4.5 — reserved slot for future rules (analytics, tutorial hints)
// that want to react to the offer regardless of whether the mob keeps
// it.
//
// For opinion-boost purposes, use GiftAccepted instead — GiftOffered
// fires even on worthless-rock spam.
//
// Chunk 4.5.
type GiftOffered struct {
	UserId        int
	MobInstanceId int
	ItemId        int
}

func (g GiftOffered) Type() string { return "GiftOffered" }

// GiftAccepted fires when a mob receives (and does not return) an item
// offered by a player via the `give` command. Fired by the give-action
// handler immediately after a successful item-transfer to a mob. Mobs
// that run the equip-if-better btree path may subsequently return the
// item; the per-pair cooldown (100 rounds) in rule 7 absorbs any
// double-fire noise.
//
// Consumed by chunk-4.5 seeders:
//   - gift_to_opinion_boost (rule 7) — value-tiered opinion bump.
//
// Note: the "clean" approach would be to fire GiftAccepted only from
// the btree keep-branch, but that wiring is complex (no direct
// keep-or-return hook exists). Firing from give.go is pragmatic; rule
// 7's cooldown prevents spam-bumping. Chunk 4.5.
type GiftAccepted struct {
	UserId        int
	MobInstanceId int
	ItemId        int
}

func (g GiftAccepted) Type() string { return "GiftAccepted" }

// CastInterrupted fires when a player's in-progress spellcast is cancelled by
// an outside force (active interrupt, damage-broken concentration). Consumed by
// the GMCP layer so the web-client action queue can re-arm the cast.
type CastInterrupted struct {
	UserId  int
	SpellId string
}

func (c CastInterrupted) Type() string { return `CastInterrupted` }
