package dialogue

// PlayerState provides quest and inventory callbacks so the dialogue engine
// can gate options on player progress without importing characters/users.
// When nil is passed the engine skips all quest/item checks (backward compat).
type PlayerState struct {
	HasQuest         func(token string) bool
	HasItem          func(itemId int) bool
	RemoveItem       func(itemId int) bool
	GiveQuest        func(token string)
	GiveItem         func(itemId int)
	GetQuestFlag     func(key string) string
	SetQuestFlag     func(key, value string)
	BumpRep          func(faction string, delta int)
	GiveGold         func(amount int)
	HasOwnMasterwork func(skillMin int) bool
}

// QuestFlagSet describes a single key/value flag to write on the player's character.
type QuestFlagSet struct {
	Key   string `yaml:"key" json:"key,omitempty"`
	Value string `yaml:"value" json:"value,omitempty"`
}

// RepBump describes a single faction-reputation change applied when a dialogue
// node matches. Lets an ask-path delivery node replicate a give-path's
// item_give bump_rep actions so both paths are equivalent.
type RepBump struct {
	Faction string `yaml:"faction" json:"faction,omitempty"`
	Delta   int    `yaml:"delta" json:"delta,omitempty"`
}

// Mood represents the current emotional state of an NPC instance.
type Mood string

const (
	MoodFriendly Mood = "friendly"
	MoodNeutral  Mood = "neutral"
	MoodHostile  Mood = "hostile"
	MoodAfraid   Mood = "afraid"
	MoodGrateful Mood = "grateful"
)

// Pattern is a single keyword-triggered response rule in a dialogue file.
type Pattern struct {
	Keywords           []string          `yaml:"keywords" json:"keywords,omitempty"`
	Moods              []string          `yaml:"moods,omitempty" json:"moods,omitempty"`
	Responses          []string          `yaml:"responses" json:"responses,omitempty"`
	MoodChange         string            `yaml:"moodChange,omitempty" json:"moodChange,omitempty"`
	QuestRequired      []string          `yaml:"questRequired,omitempty" json:"questRequired,omitempty"`
	QuestExcluded      []string          `yaml:"questExcluded,omitempty" json:"questExcluded,omitempty"`
	GrantsQuest        string            `yaml:"grantsQuest,omitempty" json:"grantsQuest,omitempty"`
	RequiresItem       int               `yaml:"requiresItem,omitempty" json:"requiresItem,omitempty"`
	GivesItem          int               `yaml:"givesItem,omitempty" json:"givesItem,omitempty"`
	QuestFlagRequired  map[string]string `yaml:"questFlagRequired,omitempty" json:"questFlagRequired,omitempty"`
	QuestFlagExcluded  map[string]string `yaml:"questFlagExcluded,omitempty" json:"questFlagExcluded,omitempty"`
	SetsQuestFlag      *QuestFlagSet     `yaml:"setsQuestFlag,omitempty" json:"setsQuestFlag,omitempty"`
	BumpsRep           []RepBump         `yaml:"bumpsRep,omitempty" json:"bumpsRep,omitempty"`
	GivesGold          int               `yaml:"givesGold,omitempty" json:"givesGold,omitempty"`
	MasterworkRequired int               `yaml:"masterworkRequired,omitempty" json:"masterworkRequired,omitempty"`
}

// TreeNode is a stateful conversation node gated by triggers and unlock requirements.
type TreeNode struct {
	Id                 string            `yaml:"id" json:"id,omitempty"`
	Triggers           []string          `yaml:"triggers" json:"triggers,omitempty"`
	Requires           []string          `yaml:"requires,omitempty" json:"requires,omitempty"`
	Text               string            `yaml:"text" json:"text,omitempty"`
	Hints              string            `yaml:"hints,omitempty" json:"hints,omitempty"`
	Unlocks            []string          `yaml:"unlocks,omitempty" json:"unlocks,omitempty"`
	MoodChange         string            `yaml:"moodChange,omitempty" json:"moodChange,omitempty"`
	QuestRequired      []string          `yaml:"questRequired,omitempty" json:"questRequired,omitempty"`
	QuestExcluded      []string          `yaml:"questExcluded,omitempty" json:"questExcluded,omitempty"`
	GrantsQuest        string            `yaml:"grantsQuest,omitempty" json:"grantsQuest,omitempty"`
	RequiresItem       int               `yaml:"requiresItem,omitempty" json:"requiresItem,omitempty"`
	GivesItem          int               `yaml:"givesItem,omitempty" json:"givesItem,omitempty"`
	QuestFlagRequired  map[string]string `yaml:"questFlagRequired,omitempty" json:"questFlagRequired,omitempty"`
	QuestFlagExcluded  map[string]string `yaml:"questFlagExcluded,omitempty" json:"questFlagExcluded,omitempty"`
	SetsQuestFlag      *QuestFlagSet     `yaml:"setsQuestFlag,omitempty" json:"setsQuestFlag,omitempty"`
	BumpsRep           []RepBump         `yaml:"bumpsRep,omitempty" json:"bumpsRep,omitempty"`
	GivesGold          int               `yaml:"givesGold,omitempty" json:"givesGold,omitempty"`
	MasterworkRequired int               `yaml:"masterworkRequired,omitempty" json:"masterworkRequired,omitempty"`
}

// QuestGreeting is an alternative greeting shown when the player matches quest conditions.
type QuestGreeting struct {
	QuestRequired      []string          `yaml:"questRequired,omitempty" json:"questRequired,omitempty"`
	QuestExcluded      []string          `yaml:"questExcluded,omitempty" json:"questExcluded,omitempty"`
	QuestFlagRequired  map[string]string `yaml:"questFlagRequired,omitempty" json:"questFlagRequired,omitempty"`
	QuestFlagExcluded  map[string]string `yaml:"questFlagExcluded,omitempty" json:"questFlagExcluded,omitempty"`
	Text               string            `yaml:"text" json:"text,omitempty"`
	Hints              string            `yaml:"hints,omitempty" json:"hints,omitempty"`
	GrantsQuest        string            `yaml:"grantsQuest,omitempty" json:"grantsQuest,omitempty"`
	GivesItem          int               `yaml:"givesItem,omitempty" json:"givesItem,omitempty"`
	RequiresItem       int               `yaml:"requiresItem,omitempty" json:"requiresItem,omitempty"`
	SetsQuestFlag      *QuestFlagSet     `yaml:"setsQuestFlag,omitempty" json:"setsQuestFlag,omitempty"`
	BumpsRep           []RepBump         `yaml:"bumpsRep,omitempty" json:"bumpsRep,omitempty"`
	GivesGold          int               `yaml:"givesGold,omitempty" json:"givesGold,omitempty"`
	MasterworkRequired int               `yaml:"masterworkRequired,omitempty" json:"masterworkRequired,omitempty"`
}

// TreeRoot holds the greeting delivered when a player first uses 'talk'.
type TreeRoot struct {
	Text     string          `yaml:"text" json:"text,omitempty"`
	Hints    string          `yaml:"hints,omitempty" json:"hints,omitempty"`
	Variants []QuestGreeting `yaml:"variants,omitempty" json:"variants,omitempty"`
}

// Tree holds all stateful conversation nodes for a mob.
type Tree struct {
	Root  TreeRoot   `yaml:"root" json:"root,omitempty"`
	Nodes []TreeNode `yaml:"nodes" json:"nodes,omitempty"`
}

// MemoryConfig controls how long per-player memory persists between visits.
type MemoryConfig struct {
	ExpiryPeriod string `yaml:"expiryPeriod" json:"expiryPeriod,omitempty"`
}

// DialogueFile is the top-level structure of a mob's dialogue YAML.
// Greeting is one ambient line an NPC offers when a player arrives in its
// room. Authored in 186 dialogue files since long before the engine read
// them — the struct is shaped to the existing YAML, not the other way round.
type Greeting struct {
	Text  string   `yaml:"text" json:"text,omitempty"`
	Moods []string `yaml:"moods,omitempty" json:"moods,omitempty"`
}

type DialogueFile struct {
	MobId       int          `yaml:"mobid" json:"mobid,omitempty"`
	Zone        string       `yaml:"zone" json:"zone,omitempty"`
	DefaultMood string       `yaml:"defaultMood" json:"defaultMood,omitempty"`
	Greetings   []Greeting   `yaml:"greetings,omitempty" json:"greetings,omitempty"`
	Patterns    []Pattern    `yaml:"patterns" json:"patterns,omitempty"`
	Tree        *Tree        `yaml:"tree,omitempty" json:"tree,omitempty"`
	Memory      MemoryConfig `yaml:"memory" json:"memory,omitempty"`
}
