package spells

import (
	"math/rand"
	"os"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"gopkg.in/yaml.v2"
)

// CastingMessages holds the varied atmospheric messages for the casting system.
type CastingMessages struct {
	AlreadyCasting       []string `yaml:"already_casting"`
	CastStarted          []string `yaml:"cast_started"`
	ConcentrationSlipped []string `yaml:"concentration_slipped"`
}

var (
	castingMessages     *CastingMessages
	castingMessagesOnce sync.Once
)

// loadCastingMessages loads the YAML file once and caches the result.
func loadCastingMessages() *CastingMessages {
	castingMessagesOnce.Do(func() {
		path := string(configs.GetFilePathsConfig().DataFiles) + `/casting-messages.yaml`
		data, err := os.ReadFile(path)
		if err != nil {
			castingMessages = defaultCastingMessages()
			return
		}
		var cm CastingMessages
		if err := yaml.Unmarshal(data, &cm); err != nil {
			castingMessages = defaultCastingMessages()
			return
		}
		castingMessages = &cm
	})
	return castingMessages
}

// defaultCastingMessages returns hardcoded fallback messages if the YAML file is missing.
func defaultCastingMessages() *CastingMessages {
	return &CastingMessages{
		AlreadyCasting: []string{
			"You are already casting {spell}. Type cancel to release the folds.",
		},
		CastStarted: []string{
			"You gather your will and begin forming the image of {spell}...",
		},
		ConcentrationSlipped: []string{
			"You reach for the folds of {spell} but your concentration slips.",
		},
	}
}

// GetCastMessage picks a random message from the named category, substituting {spell} with spellName.
// category must be one of: "already_casting", "cast_started", "concentration_slipped".
func GetCastMessage(category, spellName string) string {
	cm := loadCastingMessages()

	var pool []string
	switch category {
	case "already_casting":
		pool = cm.AlreadyCasting
	case "cast_started":
		pool = cm.CastStarted
	case "concentration_slipped":
		pool = cm.ConcentrationSlipped
	}

	if len(pool) == 0 {
		return "Something stirs with " + spellName + "."
	}

	msg := pool[rand.Intn(len(pool))]
	return strings.ReplaceAll(msg, "{spell}", spellName)
}
