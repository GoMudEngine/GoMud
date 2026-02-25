// Package llm provides local LLM integration via Ollama for NPC dialogue.
// It has no dependency on internal/mobs to avoid circular imports.
package llm

// LLMProfile is embedded in a Mob to enable LLM-driven dialogue for that NPC.
type LLMProfile struct {
	Model        string `yaml:"model"`        // e.g. "llama3.2"
	SystemPrompt string `yaml:"systemprompt"` // NPC personality prompt
	MaxWords     int    `yaml:"maxwords"`     // response length cap; default 80
	CacheTTL     string `yaml:"cachettl"`     // gametime period; default "1h"
	DefaultMood  string `yaml:"defaultmood"`  // fallback mood for context injection
}

// ConversationContext is passed from ask.go / talk.go — no mobs/dialogue imports needed here.
type ConversationContext struct {
	MobName      string
	ZoneName     string
	PlayerName   string
	CurrentMood  string
	RecentTopics []string // last 5 topics from dialogue.GetMemory()
	QuestContext     []string // human-readable quest summaries relevant to this NPC
	PlayerCondition  string   // e.g. "healthy", "seriously wounded, has died 2 times"
	TutorialProgress string   // structured tutorial step summary for Sanctum Trials
}

// Ollama /api/chat wire types (stream: false)
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}
