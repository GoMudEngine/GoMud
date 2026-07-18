package questengine

type QuestFlagDef struct {
	Key         string   `yaml:"key"`
	Values      []string `yaml:"values"`
	Description string   `yaml:"description,omitempty"`
}

// QuestDef is the expanded quest definition loaded from YAML.
type QuestDef struct {
	QuestId     int            `yaml:"questid"`
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Secret      bool           `yaml:"secret,omitempty"`
	Linear      bool           `yaml:"linear,omitempty"` // true = steps must be granted in order (default true)
	Steps       []QuestStep    `yaml:"steps"`
	Rewards     QuestRewards   `yaml:"rewards,omitempty"`
	Triggers    []TriggerDef   `yaml:"triggers"`
	Flags       []QuestFlagDef `yaml:"flags,omitempty"`
}

type QuestStep struct {
	Id          string `yaml:"id"`
	Description string `yaml:"description,omitempty"`
	Hint        string `yaml:"hint,omitempty"`
	MapTarget   int    `yaml:"map_target,omitempty"` // room the minimap marker points at during this step (0 = infer/none)
}

type QuestRewards struct {
	Gold          int    `yaml:"gold,omitempty"`
	ItemId        int    `yaml:"item_id,omitempty"`
	BuffId        int    `yaml:"buff_id,omitempty"`
	SpellId       string `yaml:"spell_id,omitempty"`
	SkillInfo     string `yaml:"skill_info,omitempty"`
	StatInfo      string `yaml:"stat_info,omitempty"`
	PlayerMessage string `yaml:"player_message,omitempty"`
	RoomMessage   string `yaml:"room_message,omitempty"`
	RoomId        int    `yaml:"room_id,omitempty"`
	ChainQuest    string `yaml:"chain_quest,omitempty"` // next quest token to grant
}

// TriggerDef defines when and how a quest step fires.
type TriggerDef struct {
	Event      string      `yaml:"event"`                 // room_enter, item_give, skill_use, mob_death, command, item_gain, dialogue, quest_granted, room_interact
	Room       int         `yaml:"room,omitempty"`        // room ID filter
	Mob        int         `yaml:"mob,omitempty"`         // mob ID filter
	Item       int         `yaml:"item,omitempty"`        // item ID filter
	Skill      string      `yaml:"skill,omitempty"`       // skill name filter
	Command    string      `yaml:"command,omitempty"`     // command name filter
	Topic      string      `yaml:"topic,omitempty"`       // dialogue topic filter
	QuestToken string      `yaml:"quest_token,omitempty"` // for quest_granted event
	Noun       string      `yaml:"noun,omitempty"`        // room noun filter
	Verb       string      `yaml:"verb,omitempty"`        // room interaction verb filter
	Conditions Conditions  `yaml:"conditions,omitempty"`
	Actions    []ActionDef `yaml:"actions"`

	// Internal fields set during loading (not from YAML)
	questId int    // parent quest ID
	trigId  string // unique ID for visit tracking: "q{questId}-t{index}"
}

// Conditions for trigger evaluation.
type Conditions struct {
	Has           []string          `yaml:"has,omitempty"`            // player must have ALL these quest tokens
	Missing       []string          `yaml:"missing,omitempty"`        // player must NOT have ANY of these tokens
	InRoom        int               `yaml:"in_room,omitempty"`        // player must be in this room
	HasItem       int               `yaml:"has_item,omitempty"`       // player must have this item
	MissingItem   int               `yaml:"missing_item,omitempty"`   // player must NOT have this item
	HasFlag       map[string]string `yaml:"has_flag,omitempty"`       // player must have ALL these flag key=value pairs
	MissingFlag   map[string]string `yaml:"missing_flag,omitempty"`   // player must NOT have ANY of these flag key=value pairs
	HasGold       int               `yaml:"has_gold,omitempty"`       // player must have at least this much gold
	HasMasterwork int               `yaml:"has_masterwork,omitempty"` // player must carry an own-crafted item at this craft skill or higher
}

// ActionDef is a single action to execute when a trigger fires.
// Only one field should be set per ActionDef.
type ActionDef struct {
	Grant         string            `yaml:"grant,omitempty"`
	ConsumeItem   int               `yaml:"consume_item,omitempty"`
	GiveItem      int               `yaml:"give_item,omitempty"`
	GiveGold      int               `yaml:"give_gold,omitempty"`
	ChargeGold    int               `yaml:"charge_gold,omitempty"`
	NpcSay        *NpcSayDef        `yaml:"npc_say,omitempty"`
	SendText      string            `yaml:"send_text,omitempty"`
	RoomText      string            `yaml:"room_text,omitempty"`
	SpawnMob      *SpawnDef         `yaml:"spawn_mob,omitempty"`
	SpawnItem     *SpawnDef         `yaml:"spawn_item,omitempty"`
	LockExits     *ExitLock         `yaml:"lock_exits,omitempty"`
	UnlockExits   *ExitLock         `yaml:"unlock_exits,omitempty"`
	TeachSpell    string            `yaml:"teach_spell,omitempty"`
	TrainSkill    *SkillDef         `yaml:"train_skill,omitempty"`
	TrainStat     *StatDef          `yaml:"train_stat,omitempty"`
	LearnRecipe   *RecipeDef        `yaml:"learn_recipe,omitempty"`
	ApplyBuff     *BuffDef          `yaml:"apply_buff,omitempty"`
	Teleport      int               `yaml:"teleport,omitempty"`
	GiveMutation  bool              `yaml:"give_mutation,omitempty"` // roll and grant a random mutation
	SetFlag       *QuestFlagAction  `yaml:"set_flag,omitempty"`
	Sequence      *SequenceDef      `yaml:"sequence,omitempty"`
	BumpRep       *BumpRepDef       `yaml:"bump_rep,omitempty"`
	DeclareBounty *DeclareBountyDef `yaml:"declare_bounty,omitempty"`
}

// BumpRepDef parameters for the bump_rep action: which faction
// gets the bump and by how much.
type BumpRepDef struct {
	Faction string `yaml:"faction"`
	Delta   int    `yaml:"delta"`
}

// DeclareBountyDef parameters for the declare_bounty action.
// Either TargetPlayer (auto-fill with quest holder) OR explicit
// Target.Type+Target.Id must be set. Issuer mirrors the bounties
// package's tagged form; type=quest with id="<self>" auto-fills
// with the current quest id.
type DeclareBountyDef struct {
	Issuer struct {
		Type string `yaml:"type"` // "faction" | "quest" | "npc"
		Id   string `yaml:"id"`
	} `yaml:"issuer"`
	TargetPlayer bool `yaml:"target_player,omitempty"`
	Target       *struct {
		Type string `yaml:"type"` // "player" | "mob"
		Id   int    `yaml:"id"`
	} `yaml:"target,omitempty"`
	Condition    string `yaml:"condition"` // "kill"
	ExpiryRounds uint64 `yaml:"expiry_rounds,omitempty"`
	GoldOverride int    `yaml:"gold_override,omitempty"`
	RepOverride  int    `yaml:"rep_override,omitempty"`
	Reason       string `yaml:"reason,omitempty"`
}

type QuestFlagAction struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type NpcSayDef struct {
	Mob   int          `yaml:"mob"`
	Lines []SayLineDef `yaml:"lines"`
}

// SayLineDef can be a simple string or a {delay, text, speaker} struct.
type SayLineDef struct {
	Text    string `yaml:"text"`
	Delay   int    `yaml:"delay,omitempty"`   // delay in seconds before this line
	Speaker int    `yaml:"speaker,omitempty"` // mob ID override (0 = use parent NpcSayDef.Mob)
	Emote   bool   `yaml:"emote,omitempty"`   // true = emote instead of say
}

type SpawnDef struct {
	Id   int `yaml:"id"`   // mob or item ID
	Room int `yaml:"room"` // room to spawn in
}

type ExitLock struct {
	Room         int  `yaml:"room"`
	PlayerScoped bool `yaml:"player_scoped,omitempty"` // only affect this player
}

type SkillDef struct {
	Skill string `yaml:"skill"`
	Level int    `yaml:"level"`
}

type StatDef struct {
	Stat   string `yaml:"stat"`
	Amount int    `yaml:"amount"`
}

type RecipeDef struct {
	Recipe string `yaml:"recipe"`
}

type BuffDef struct {
	Buff   int    `yaml:"buff"`
	Source string `yaml:"source,omitempty"`
}

type SequenceDef struct {
	DelayBetween int          `yaml:"delay_between"` // seconds between lines
	Lines        []SayLineDef `yaml:"lines"`
	OnComplete   []ActionDef  `yaml:"on_complete,omitempty"`  // actions to run after sequence
	LockMessage  string       `yaml:"lock_message,omitempty"` // if set, block player movement during sequence with this message
}

// EventDetails carries context about the event that triggered evaluation.
type EventDetails struct {
	UserId     int
	RoomId     int
	MobId      int
	ItemId     int
	Skill      string
	Command    string
	Topic      string
	Noun       string
	Verb       string
	QuestToken string
}

// NotifyResult is returned by Notify to inform the caller how the event was handled.
type NotifyResult struct {
	Handled     bool // at least one trigger matched and executed
	ConsumeItem bool // the item should be consumed (not transferred to mob)
}
