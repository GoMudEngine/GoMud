package quests

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
)

const (
	QuestTokenSeparator = `-`
)

var (
	quests       map[int]*Quest = map[int]*Quest{}
	flagRegistry                = map[string][]string{}
)

type QuestFlagDef struct {
	Key         string   `yaml:"key"`
	Values      []string `yaml:"values"`
	Description string   `yaml:"description,omitempty"`
}

// QuestReward — LOADER GOTCHA: the quest loader uses gopkg.in/yaml.v2, which
// binds a TAG-LESS field to its lowercased name with NO underscore handling.
// So the tag-less fields below load ONLY from no-underscore yaml keys in a
// quest's `rewards:` block: use `itemid`, `skillinfo`, `buffid`,
// `playermessage`, `roommessage`, `roomid`, `spellid`, `questid`. snake_case
// keys (item_id, skill_info, player_message, ...) silently DO NOT load into
// these fields. Older quests follow the no-underscore convention; a few newer
// quests mistakenly used snake_case in their rewards block and those reward
// fields silently no-op (latent cleanup, tracked separately). Tagged
// exceptions DO take their snake_case key: stat_info, rep_faction, rep_amount.
type QuestReward struct {
	QuestId       string // new questId to give ( {id}-{step} format ); yaml key: questid
	Gold          int    // zero or more gold to give; yaml key: gold
	ItemId        int    // itemId to give; yaml key: itemid
	BuffId        int    // buffId to apply; yaml key: buffid
	SkillInfo     string // skill(s) to give, "skill:level[,skill:level]"; yaml key: skillinfo
	StatInfo      string `yaml:"stat_info,omitempty"`    // stat(s) to increase, "stat:amount[,...]"; yaml key: stat_info
	RecipeInfo    string `yaml:"recipe_info,omitempty"`  // recipe(s) to grant, comma-separated recipe IDs; yaml key: recipe_info
	ItemInfo      string `yaml:"item_info,omitempty"`    // item stockpile to grant, "itemid[:qty][,itemid[:qty]]"; yaml key: item_info
	SpellId       string // spell to teach on completion; yaml key: spellid
	PlayerMessage string // string to display to player; yaml key: playermessage
	RoomMessage   string // string to display to room; yaml key: roommessage
	RoomId        int    // roomId to move player to; yaml key: roomid
	RepFaction    string `yaml:"rep_faction"` // faction slug bumped on completion
	RepAmount     int    `yaml:"rep_amount"`  // rep delta applied on completion
}

type Quest struct {
	QuestId        int
	Name           string
	Description    string
	Secret         bool        // Secret quests are useful for marking some progress without making it known to the player
	Steps          []QuestStep // String identifiers for each step required to complete the quest
	Rewards        QuestReward
	Flags          []QuestFlagDef `yaml:"flags,omitempty"`
	Repeatable     bool           `yaml:"repeatable,omitempty"`      // if true, completing the quest clears its progress so it can be taken again (after CooldownRounds)
	CooldownRounds int            `yaml:"cooldown_rounds,omitempty"` // rounds that must pass after completion before a repeatable quest can be re-taken
}

type QuestStep struct {
	Id          string // A way to identify this step of the quest such as "start"
	Description string // A description of the step
	Hint        string // A hint to accomplish this step (optional)
}

func (r *Quest) Id() int {
	return r.QuestId
}

func (r *Quest) Validate() error {
	return nil
}

func (r *Quest) Filename() string {
	filename := util.ConvertForFilename(r.Name)
	return fmt.Sprintf("%d-%s.yaml", r.Id(), filename)
}

func (r *Quest) Filepath() string {
	return r.Filename()
}

func GetQuestCt(includeSecret bool) int {
	ret := 0
	for _, q := range quests {
		if includeSecret || !q.Secret {
			ret++
		}
	}
	return ret
}

func IsTokenAfter(currentToken string, nextToken string) bool {

	currentId, currentStep := TokenToParts(currentToken)
	nextId, nextStep := TokenToParts(nextToken)

	// If they don't have any progress yet, then they can only "start" a quest.
	if currentStep == `` {
		if nextStep == `start` {
			return true
		} else if nextStep == `end` {
			// If it's a single step quest, then they can end it.
			if questInfo := GetQuest(nextToken); questInfo != nil {
				if len(questInfo.Steps) == 1 {
					return true
				}
			}
		}
		return false
	}

	// If same, false
	if currentId != nextId || currentStep == nextStep {
		return false
	}

	// If currently at zero, whatever is offered must be next
	questInfo := GetQuest(currentToken)
	// If quest doesn't even exist, then no
	if questInfo == nil {
		return false
	}

	result := false
	startLooking := false

	for _, step := range questInfo.Steps {
		if step.Id == currentStep {
			startLooking = true
		}
		if startLooking {
			if step.Id == nextStep {
				result = true
				break
			}
		}
	}

	return result
}

func PartsToToken(questId int, questStep string) string {
	return fmt.Sprintf(`%d%s%s`, questId, QuestTokenSeparator, questStep)
}

func TokenToParts(questToken string) (questId int, questStep string) {
	parts := strings.Split(questToken, QuestTokenSeparator)
	var err error
	questId, err = strconv.Atoi(parts[0])
	if err != nil {
		mudlog.Warn("TokenToParts", "token", questToken, "error", err)
	}
	if len(parts) > 1 {
		questStep = parts[1]
	} else {
		questStep = `start`
	}

	return questId, questStep
}

func GetQuest(questToken string) *Quest {

	questId, questStep := TokenToParts(questToken)

	quest := quests[questId]
	if quest == nil {
		return nil
	}

	if questStep == `all+` {
		return quest
	}

	stepIsValid := true
	if len(questStep) > 0 {
		stepIsValid = false
		for _, step := range quest.Steps {
			if step.Id == questStep {
				stepIsValid = true
				break
			}
		}
	}

	if stepIsValid {
		return quest
	}

	return nil
}

func GetAllQuests() []Quest {
	ret := []Quest{}
	for _, q := range quests {
		ret = append(ret, *q)
	}
	return ret
}

func RegisterFlags(questId int, flags []QuestFlagDef) {
	for _, f := range flags {
		key := fmt.Sprintf("%d-%s", questId, f.Key)
		flagRegistry[key] = f.Values
	}
}

func ValidateFlag(key, value string) error {
	allowed, ok := flagRegistry[key]
	if !ok {
		return fmt.Errorf("undeclared quest flag %q (not defined in any quest's flags section)", key)
	}
	if value == "" {
		return nil
	}
	for _, v := range allowed {
		if v == value {
			return nil
		}
	}
	return fmt.Errorf("quest flag %q has invalid value %q (allowed: %v)", key, value, allowed)
}

func GetFlagRegistry() map[string][]string {
	out := make(map[string][]string, len(flagRegistry))
	for k, v := range flagRegistry {
		out[k] = v
	}
	return out
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/quests`
	tmpQuests, err := fileloader.LoadAllFlatFiles[int, *Quest](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
	}

	quests = tmpQuests

	flagRegistry = map[string][]string{}
	for _, q := range quests {
		RegisterFlags(q.QuestId, q.Flags)
	}

	mudlog.Info("quests.LoadDataFiles()", "loadedCount", len(quests), "Time Taken", time.Since(start))

}
