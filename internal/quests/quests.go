package quests

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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
	Key         string   `yaml:"key" json:"key"`
	Values      []string `yaml:"values" json:"values"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
}

// QuestReward — every key is EXPLICITLY tagged with the key that has always
// bound it. The no-underscore keys are historical yaml.v2 tag-less binding
// (lowercased field name, no underscore handling), now pinned by tag; the
// snake_case five were tagged all along. This vocabulary is canonical — the
// 5c editor writer marshals exactly this. Keys are proven stable by the
// equivalence harness in unification_equivalence_test.go.
type QuestReward struct {
	QuestId       string `yaml:"questid,omitempty" json:"questid,omitempty"`         // new questId to give ( {id}-{step} format )
	Gold          int    `yaml:"gold,omitempty" json:"gold,omitempty"`               // zero or more gold to give
	ItemId        int    `yaml:"itemid,omitempty" json:"itemid,omitempty"`           // itemId to give
	BuffId        int    `yaml:"buffid,omitempty" json:"buffid,omitempty"`           // buffId to apply
	SkillInfo     string `yaml:"skillinfo,omitempty" json:"skillinfo,omitempty"`     // skill(s) to give, "skill:level[,skill:level]"
	StatInfo      string `yaml:"stat_info,omitempty" json:"stat_info,omitempty"`     // stat(s) to increase, "stat:amount[,...]"
	RecipeInfo    string `yaml:"recipe_info,omitempty" json:"recipe_info,omitempty"` // recipe(s) to grant, comma-separated recipe IDs
	ItemInfo      string `yaml:"item_info,omitempty" json:"item_info,omitempty"`     // item stockpile to grant, "itemid[:qty][,itemid[:qty]]"
	SpellId       string `yaml:"spellid,omitempty" json:"spellid,omitempty"`         // spell to teach on completion
	PlayerMessage string `yaml:"playermessage,omitempty" json:"playermessage,omitempty"`
	RoomMessage   string `yaml:"roommessage,omitempty" json:"roommessage,omitempty"`
	RoomId        int    `yaml:"roomid,omitempty" json:"roomid,omitempty"` // roomId to move player to
	RepFaction    string `yaml:"rep_faction,omitempty" json:"rep_faction,omitempty"`
	RepAmount     int    `yaml:"rep_amount,omitempty" json:"rep_amount,omitempty"`
}

// Quest is THE quest definition — the single parse of quest YAML (5c-pre
// unification). internal/questengine consumes these via GetAllQuests();
// nothing else parses the files.
type Quest struct {
	QuestId        int            `yaml:"questid" json:"questid"`
	Name           string         `yaml:"name" json:"name"`
	Description    string         `yaml:"description,omitempty" json:"description,omitempty"`
	Secret         bool           `yaml:"secret,omitempty" json:"secret,omitempty"` // marks progress without making it known to the player
	Steps          []QuestStep    `yaml:"steps" json:"steps"`
	Rewards        QuestReward    `yaml:"rewards,omitempty" json:"rewards,omitempty"`
	Triggers       []TriggerDef   `yaml:"triggers,omitempty" json:"triggers,omitempty"`
	Flags          []QuestFlagDef `yaml:"flags,omitempty" json:"flags,omitempty"`
	Repeatable     bool           `yaml:"repeatable,omitempty" json:"repeatable,omitempty"`           // completing clears progress so it can be re-taken (after CooldownRounds)
	CooldownRounds int            `yaml:"cooldown_rounds,omitempty" json:"cooldown_rounds,omitempty"` // rounds after completion before a repeatable quest can be re-taken
}

type QuestStep struct {
	Id          string `yaml:"id" json:"id"` // identifies this step, e.g. "start"
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Hint        string `yaml:"hint,omitempty" json:"hint,omitempty"`
	MapTarget   int    `yaml:"map_target,omitempty" json:"map_target,omitempty"` // room the minimap marker points at during this step (0 = infer from room_enter triggers, -1 = deliberate NO marker; see questengine.ResolveQuestTarget)
}

func (r *Quest) Id() int {
	return r.QuestId
}

// Validate implements fileloader.Loadable — it runs on EVERY parse, so a
// broken quest file fails the boot (and, later, an editor save) instead of
// loading half-formed. Body moved from questengine's old validateQuestDef in
// the 5c-pre unification; the checks are unchanged.
func (r *Quest) Validate() error {
	if r.QuestId < 1 {
		return fmt.Errorf("quest id must be > 0")
	}
	if r.Name == "" {
		return fmt.Errorf("quest %d: name cannot be empty", r.QuestId)
	}
	if len(r.Steps) == 0 {
		return fmt.Errorf("quest %d (%s): must have at least one step", r.QuestId, r.Name)
	}

	validEvents := map[string]bool{
		"room_enter": true, "item_give": true, "skill_use": true,
		"mob_death": true, "command": true, "item_gain": true,
		"dialogue": true, "quest_granted": true, "room_interact": true,
		"command_issued": true,
	}

	for i, t := range r.Triggers {
		if t.Event == "" {
			return fmt.Errorf("quest %d (%s): trigger %d has no event", r.QuestId, r.Name, i)
		}
		if !validEvents[t.Event] {
			return fmt.Errorf("quest %d (%s): trigger %d has invalid event %q", r.QuestId, r.Name, i, t.Event)
		}
		if len(t.Actions) == 0 {
			return fmt.Errorf("quest %d (%s): trigger %d has no actions", r.QuestId, r.Name, i)
		}
	}

	// Check for duplicate step IDs
	seen := make(map[string]bool)
	for _, s := range r.Steps {
		if s.Id == "" {
			return fmt.Errorf("quest %d (%s): step has empty id", r.QuestId, r.Name)
		}
		if seen[s.Id] {
			return fmt.Errorf("quest %d (%s): duplicate step id %q", r.QuestId, r.Name, s.Id)
		}
		seen[s.Id] = true
	}

	// Validate flag declarations
	flagKeys := make(map[string]bool)
	for _, f := range r.Flags {
		if f.Key == "" {
			return fmt.Errorf("quest %d (%s): flag has empty key", r.QuestId, r.Name)
		}
		if len(f.Values) == 0 {
			return fmt.Errorf("quest %d (%s): flag %q has no allowed values", r.QuestId, r.Name, f.Key)
		}
		fullKey := fmt.Sprintf("%d-%s", r.QuestId, f.Key)
		if flagKeys[fullKey] {
			return fmt.Errorf("quest %d (%s): duplicate flag key %q", r.QuestId, r.Name, f.Key)
		}
		flagKeys[fullKey] = true
	}

	// Validate grant tokens reference valid steps (only for this quest's own tokens)
	for i, t := range r.Triggers {
		for _, a := range t.Actions {
			if a.Grant != "" {
				parts := strings.SplitN(a.Grant, "-", 2)
				if len(parts) == 2 {
					// Only validate if the grant is for THIS quest
					questIdStr := fmt.Sprintf("%d", r.QuestId)
					if parts[0] == questIdStr {
						stepId := parts[1]
						if !seen[stepId] {
							return fmt.Errorf("quest %d (%s): trigger %d grants unknown step %q",
								r.QuestId, r.Name, i, a.Grant)
						}
					}
				}
			}
		}
	}

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

// GetQuestById returns the quest definition for a bare numeric id (no token
// parsing, no step check). The editor's load path — GetQuest requires a
// valid step suffix, which not every quest's tokens share.
func GetQuestById(questId int) *Quest {
	return quests[questId]
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

	dataPath := questsDataRoot()
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
