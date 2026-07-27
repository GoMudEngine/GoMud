package quests

// The quest trigger DSL — moved here from internal/questengine as part of the
// 5c-pre unification so quest YAML has exactly one parse. These are pure data:
// evaluation lives in internal/questengine, which imports this package and
// aliases these names. Definitions are immutable after load — the engine
// tracks per-trigger identity (questId/trigId) in its own index wrapper, NOT
// on these structs.

// TriggerDef defines when and how a quest step fires.
type TriggerDef struct {
	Event      string      `yaml:"event" json:"event"`                                 // room_enter, item_give, skill_use, mob_death, command, command_issued, item_gain, dialogue, quest_granted, room_interact
	Room       int         `yaml:"room,omitempty" json:"room,omitempty"`               // room ID filter
	Mob        int         `yaml:"mob,omitempty" json:"mob,omitempty"`                 // mob ID filter
	Item       int         `yaml:"item,omitempty" json:"item,omitempty"`               // item ID filter
	Skill      string      `yaml:"skill,omitempty" json:"skill,omitempty"`             // skill name filter
	Command    string      `yaml:"command,omitempty" json:"command,omitempty"`         // command name filter
	Topic      string      `yaml:"topic,omitempty" json:"topic,omitempty"`             // dialogue topic filter
	QuestToken string      `yaml:"quest_token,omitempty" json:"quest_token,omitempty"` // for quest_granted event
	Noun       string      `yaml:"noun,omitempty" json:"noun,omitempty"`               // room noun filter
	Verb       string      `yaml:"verb,omitempty" json:"verb,omitempty"`               // room interaction verb filter
	Conditions Conditions  `yaml:"conditions,omitempty" json:"conditions,omitempty"`
	Actions    []ActionDef `yaml:"actions" json:"actions"`
}

// Conditions for trigger evaluation.
type Conditions struct {
	Has           []string          `yaml:"has,omitempty" json:"has,omitempty"`                       // player must have ALL these quest tokens
	Missing       []string          `yaml:"missing,omitempty" json:"missing,omitempty"`               // player must NOT have ANY of these tokens
	InRoom        int               `yaml:"in_room,omitempty" json:"in_room,omitempty"`               // player must be in this room
	HasItem       int               `yaml:"has_item,omitempty" json:"has_item,omitempty"`             // player must have this item
	MissingItem   int               `yaml:"missing_item,omitempty" json:"missing_item,omitempty"`     // player must NOT have this item
	HasFlag       map[string]string `yaml:"has_flag,omitempty" json:"has_flag,omitempty"`             // player must have ALL these flag key=value pairs
	MissingFlag   map[string]string `yaml:"missing_flag,omitempty" json:"missing_flag,omitempty"`     // player must NOT have ANY of these flag key=value pairs
	HasGold       int               `yaml:"has_gold,omitempty" json:"has_gold,omitempty"`             // player must have at least this much gold
	HasMasterwork int               `yaml:"has_masterwork,omitempty" json:"has_masterwork,omitempty"` // player must carry an own-crafted item at this craft skill or higher
}

// ActionDef is a single action to execute when a trigger fires.
// Only one field should be set per ActionDef.
type ActionDef struct {
	Grant         string            `yaml:"grant,omitempty" json:"grant,omitempty"`
	ConsumeItem   int               `yaml:"consume_item,omitempty" json:"consume_item,omitempty"`
	GiveItem      int               `yaml:"give_item,omitempty" json:"give_item,omitempty"`
	GiveGold      int               `yaml:"give_gold,omitempty" json:"give_gold,omitempty"`
	ChargeGold    int               `yaml:"charge_gold,omitempty" json:"charge_gold,omitempty"`
	NpcSay        *NpcSayDef        `yaml:"npc_say,omitempty" json:"npc_say,omitempty"`
	SendText      string            `yaml:"send_text,omitempty" json:"send_text,omitempty"`
	RoomText      string            `yaml:"room_text,omitempty" json:"room_text,omitempty"`
	SpawnMob      *SpawnDef         `yaml:"spawn_mob,omitempty" json:"spawn_mob,omitempty"`
	SpawnItem     *SpawnDef         `yaml:"spawn_item,omitempty" json:"spawn_item,omitempty"`
	LockExits     *ExitLock         `yaml:"lock_exits,omitempty" json:"lock_exits,omitempty"`
	UnlockExits   *ExitLock         `yaml:"unlock_exits,omitempty" json:"unlock_exits,omitempty"`
	TeachSpell    string            `yaml:"teach_spell,omitempty" json:"teach_spell,omitempty"`
	TrainSkill    *SkillDef         `yaml:"train_skill,omitempty" json:"train_skill,omitempty"`
	TrainStat     *StatDef          `yaml:"train_stat,omitempty" json:"train_stat,omitempty"`
	LearnRecipe   *RecipeDef        `yaml:"learn_recipe,omitempty" json:"learn_recipe,omitempty"`
	ApplyBuff     *BuffDef          `yaml:"apply_buff,omitempty" json:"apply_buff,omitempty"`
	Teleport      int               `yaml:"teleport,omitempty" json:"teleport,omitempty"`
	GiveMutation  bool              `yaml:"give_mutation,omitempty" json:"give_mutation,omitempty"` // roll and grant a random mutation
	SetFlag       *QuestFlagAction  `yaml:"set_flag,omitempty" json:"set_flag,omitempty"`
	Sequence      *SequenceDef      `yaml:"sequence,omitempty" json:"sequence,omitempty"`
	BumpRep       *BumpRepDef       `yaml:"bump_rep,omitempty" json:"bump_rep,omitempty"`
	DeclareBounty *DeclareBountyDef `yaml:"declare_bounty,omitempty" json:"declare_bounty,omitempty"`
}

// BumpRepDef parameters for the bump_rep action: which faction
// gets the bump and by how much.
type BumpRepDef struct {
	Faction string `yaml:"faction" json:"faction"`
	Delta   int    `yaml:"delta" json:"delta"`
}

// DeclareBountyDef parameters for the declare_bounty action.
// Either TargetPlayer (auto-fill with quest holder) OR explicit
// Target.Type+Target.Id must be set. Issuer mirrors the bounties
// package's tagged form; type=quest with id="<self>" auto-fills
// with the current quest id.
type DeclareBountyDef struct {
	Issuer struct {
		Type string `yaml:"type" json:"type"` // "faction" | "quest" | "npc"
		Id   string `yaml:"id" json:"id"`
	} `yaml:"issuer" json:"issuer"`
	TargetPlayer bool `yaml:"target_player,omitempty" json:"target_player,omitempty"`
	Target       *struct {
		Type string `yaml:"type" json:"type"` // "player" | "mob"
		Id   int    `yaml:"id" json:"id"`
	} `yaml:"target,omitempty" json:"target,omitempty"`
	Condition    string `yaml:"condition" json:"condition"` // "kill"
	ExpiryRounds uint64 `yaml:"expiry_rounds,omitempty" json:"expiry_rounds,omitempty"`
	GoldOverride int    `yaml:"gold_override,omitempty" json:"gold_override,omitempty"`
	RepOverride  int    `yaml:"rep_override,omitempty" json:"rep_override,omitempty"`
	Reason       string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

type QuestFlagAction struct {
	Key   string `yaml:"key" json:"key"`
	Value string `yaml:"value" json:"value"`
}

type NpcSayDef struct {
	Mob   int          `yaml:"mob" json:"mob"`
	Lines []SayLineDef `yaml:"lines" json:"lines"`
}

// SayLineDef can be a simple string or a {delay, text, speaker} struct.
type SayLineDef struct {
	Text    string `yaml:"text" json:"text"`
	Delay   int    `yaml:"delay,omitempty" json:"delay,omitempty"`     // delay in seconds before this line
	Speaker int    `yaml:"speaker,omitempty" json:"speaker,omitempty"` // mob ID override (0 = use parent NpcSayDef.Mob)
	Emote   bool   `yaml:"emote,omitempty" json:"emote,omitempty"`     // true = emote instead of say
}

type SpawnDef struct {
	Id   int `yaml:"id" json:"id"`     // mob or item ID
	Room int `yaml:"room" json:"room"` // room to spawn in
}

type ExitLock struct {
	Room         int  `yaml:"room" json:"room"`
	PlayerScoped bool `yaml:"player_scoped,omitempty" json:"player_scoped,omitempty"` // only affect this player
}

type SkillDef struct {
	Skill string `yaml:"skill" json:"skill"`
	Level int    `yaml:"level" json:"level"`
}

type StatDef struct {
	Stat   string `yaml:"stat" json:"stat"`
	Amount int    `yaml:"amount" json:"amount"`
}

type RecipeDef struct {
	Recipe string `yaml:"recipe" json:"recipe"`
}

type BuffDef struct {
	Buff   int    `yaml:"buff" json:"buff"`
	Source string `yaml:"source,omitempty" json:"source,omitempty"`
}

type SequenceDef struct {
	DelayBetween int          `yaml:"delay_between" json:"delay_between"` // seconds between lines
	Lines        []SayLineDef `yaml:"lines" json:"lines"`
	OnComplete   []ActionDef  `yaml:"on_complete,omitempty" json:"on_complete,omitempty"`   // actions to run after sequence
	LockMessage  string       `yaml:"lock_message,omitempty" json:"lock_message,omitempty"` // if set, block player movement during sequence with this message
}
