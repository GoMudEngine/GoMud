package dialogue

// PlayerState provides quest and inventory callbacks so the dialogue engine
// can gate options on player progress without importing characters/users.
// When nil is passed the engine skips all quest/item checks (backward compat).
type PlayerState struct {
	HasQuest     func(token string) bool
	HasItem      func(itemId int) bool
	RemoveItem   func(itemId int) bool
	GiveQuest    func(token string)
	GiveItem     func(itemId int)
	GetQuestFlag func(key string) string
	SetQuestFlag func(key, value string)
}

// QuestFlagSet describes a single key/value flag to write on the player's character.
type QuestFlagSet struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
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
	Keywords          []string          `yaml:"keywords"`
	Moods             []string          `yaml:"moods,omitempty"`
	Responses         []string          `yaml:"responses"`
	MoodChange        string            `yaml:"moodChange,omitempty"`
	QuestRequired     []string          `yaml:"questRequired,omitempty"`
	QuestExcluded     []string          `yaml:"questExcluded,omitempty"`
	GrantsQuest       string            `yaml:"grantsQuest,omitempty"`
	RequiresItem      int               `yaml:"requiresItem,omitempty"`
	GivesItem         int               `yaml:"givesItem,omitempty"`
	QuestFlagRequired map[string]string `yaml:"questFlagRequired,omitempty"`
	QuestFlagExcluded map[string]string `yaml:"questFlagExcluded,omitempty"`
	SetsQuestFlag     *QuestFlagSet     `yaml:"setsQuestFlag,omitempty"`
}

// TreeNode is a stateful conversation node gated by triggers and unlock requirements.
type TreeNode struct {
	Id                string            `yaml:"id"`
	Triggers          []string          `yaml:"triggers"`
	Requires          []string          `yaml:"requires,omitempty"`
	Text              string            `yaml:"text"`
	Hints             string            `yaml:"hints,omitempty"`
	Unlocks           []string          `yaml:"unlocks,omitempty"`
	MoodChange        string            `yaml:"moodChange,omitempty"`
	QuestRequired     []string          `yaml:"questRequired,omitempty"`
	QuestExcluded     []string          `yaml:"questExcluded,omitempty"`
	GrantsQuest       string            `yaml:"grantsQuest,omitempty"`
	RequiresItem      int               `yaml:"requiresItem,omitempty"`
	GivesItem         int               `yaml:"givesItem,omitempty"`
	QuestFlagRequired map[string]string `yaml:"questFlagRequired,omitempty"`
	QuestFlagExcluded map[string]string `yaml:"questFlagExcluded,omitempty"`
	SetsQuestFlag     *QuestFlagSet     `yaml:"setsQuestFlag,omitempty"`
}

// QuestGreeting is an alternative greeting shown when the player matches quest conditions.
type QuestGreeting struct {
	QuestRequired     []string          `yaml:"questRequired,omitempty"`
	QuestExcluded     []string          `yaml:"questExcluded,omitempty"`
	QuestFlagRequired map[string]string `yaml:"questFlagRequired,omitempty"`
	QuestFlagExcluded map[string]string `yaml:"questFlagExcluded,omitempty"`
	Text              string            `yaml:"text"`
	Hints             string            `yaml:"hints,omitempty"`
	GrantsQuest       string            `yaml:"grantsQuest,omitempty"`
	GivesItem         int               `yaml:"givesItem,omitempty"`
	RequiresItem      int               `yaml:"requiresItem,omitempty"`
	SetsQuestFlag     *QuestFlagSet     `yaml:"setsQuestFlag,omitempty"`
}

// TreeRoot holds the greeting delivered when a player first uses 'talk'.
type TreeRoot struct {
	Text     string          `yaml:"text"`
	Hints    string          `yaml:"hints,omitempty"`
	Variants []QuestGreeting `yaml:"variants,omitempty"`
}

// Tree holds all stateful conversation nodes for a mob.
type Tree struct {
	Root  TreeRoot   `yaml:"root"`
	Nodes []TreeNode `yaml:"nodes"`
}

// MemoryConfig controls how long per-player memory persists between visits.
type MemoryConfig struct {
	ExpiryPeriod string `yaml:"expiryPeriod"`
}

// DialogueFile is the top-level structure of a mob's dialogue YAML.
type DialogueFile struct {
	MobId       int          `yaml:"mobid"`
	Zone        string       `yaml:"zone"`
	DefaultMood string       `yaml:"defaultMood"`
	Patterns    []Pattern    `yaml:"patterns"`
	Tree        *Tree        `yaml:"tree,omitempty"`
	Memory      MemoryConfig `yaml:"memory"`
}
