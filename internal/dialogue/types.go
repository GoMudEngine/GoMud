package dialogue

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
	Keywords   []string `yaml:"keywords"`
	Moods      []string `yaml:"moods,omitempty"`
	Responses  []string `yaml:"responses"`
	MoodChange string   `yaml:"moodChange,omitempty"`
}

// TreeNode is a stateful conversation node gated by triggers and unlock requirements.
type TreeNode struct {
	Id         string   `yaml:"id"`
	Triggers   []string `yaml:"triggers"`
	Requires   []string `yaml:"requires,omitempty"`
	Text       string   `yaml:"text"`
	Hints      string   `yaml:"hints,omitempty"`
	Unlocks    []string `yaml:"unlocks,omitempty"`
	MoodChange string   `yaml:"moodChange,omitempty"`
}

// TreeRoot holds the greeting delivered when a player first uses 'talk'.
type TreeRoot struct {
	Text  string `yaml:"text"`
	Hints string `yaml:"hints,omitempty"`
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
