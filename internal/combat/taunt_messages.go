package combat

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TauntIntensity represents the outcome type of a taunt attempt.
type TauntIntensity string

const (
	TauntHit      TauntIntensity = "hit"
	TauntMiss     TauntIntensity = "miss"
	TauntCritical TauntIntensity = "critical"
	TauntFumble   TauntIntensity = "fumble"
)

// TauntMessages holds messages for a single intensity level.
type TauntMessages struct {
	ToAttacker []string `yaml:"toattacker"`
	ToDefender []string `yaml:"todefender"`
	ToRoom     []string `yaml:"toroom"`
}

// TauntMessageGroup is the top-level YAML structure for taunt messages.
type TauntMessageGroup struct {
	OptionId string                            `yaml:"optionid"`
	Options  map[TauntIntensity]*TauntMessages `yaml:"options"`
}

func (t *TauntMessageGroup) Id() string       { return t.OptionId }
func (t *TauntMessageGroup) Filepath() string { return fmt.Sprintf("%s.yaml", t.OptionId) }
func (t *TauntMessageGroup) Validate() error {
	required := []TauntIntensity{TauntHit, TauntMiss, TauntCritical, TauntFumble}
	for _, r := range required {
		if _, ok := t.Options[r]; !ok {
			return fmt.Errorf("taunt messages %q: missing intensity %q", t.OptionId, r)
		}
	}
	return nil
}

var tauntMessages map[string]*TauntMessageGroup

// LoadTauntMessageFiles reads taunt message YAMLs from the data directory.
func LoadTauntMessageFiles() {
	start := time.Now()

	tmpAll, err := fileloader.LoadAllFlatFiles[string, *TauntMessageGroup](
		string(configs.GetFilePathsConfig().DataFiles) + `/taunt-messages`,
	)
	if err != nil {
		mudlog.Error("combat.LoadTauntMessageFiles()", "error", err)
		tauntMessages = make(map[string]*TauntMessageGroup)
		return
	}

	tauntMessages = tmpAll
	mudlog.Info("combat.LoadTauntMessageFiles()", "loadedCount", len(tauntMessages), "Time Taken", time.Since(start))
}

// GetTauntMessage returns a random message for the given intensity and
// perspective. Returns empty string if no messages are available.
func GetTauntMessage(intensity TauntIntensity, perspective string, source, target, sourceType, targetType, damageDesc string) string {
	group := tauntMessages["rhetoric"]
	if group == nil {
		return ""
	}

	msgs, ok := group.Options[intensity]
	if !ok || msgs == nil {
		return ""
	}

	var pool []string
	switch perspective {
	case "toattacker":
		pool = msgs.ToAttacker
	case "todefender":
		pool = msgs.ToDefender
	case "toroom":
		pool = msgs.ToRoom
	}

	if len(pool) == 0 {
		return ""
	}

	msg := pool[util.Rand(len(pool))]

	// Token replacement
	msg = strings.ReplaceAll(msg, "{source}", source)
	msg = strings.ReplaceAll(msg, "{target}", target)
	msg = strings.ReplaceAll(msg, "{sourcetype}", sourceType)
	msg = strings.ReplaceAll(msg, "{targettype}", targetType)
	msg = strings.ReplaceAll(msg, "{damage}", damageDesc)

	return msg
}
