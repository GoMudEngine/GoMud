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
)

const (
	QuestTokenSeparator = `-`
)

var (
	quests map[int]*Quest = map[int]*Quest{}
)

type QuestReward struct {
	QuestId       string // new questId to give ( {id}-{step} format )
	Gold          int    // zero or more gold to give.
	ItemId        int    // itemId to give
	BuffId        int    // buffId to apply
	SkillInfo     string // skill to give, format: skillId:skillLevel such as "map:1"
	PlayerMessage string // string to display to player
	RoomMessage   string // string to display to room
	RoomId        int    // roomId to move player to
}

type Quest struct {
	QuestId     int
	Name        string
	Description string
	Secret      bool        // Secret quests are useful for marking some progress without making it known to the player
	Steps       []QuestStep // String identifiers for each step required to complete the quest
	Rewards     QuestReward
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

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	tmpQuests, err := fileloader.LoadAllFlatFiles[int, *Quest](configs.GetFilePathsConfig().DataFiles.String() + `/quests`)
	if err != nil {
		panic(err)
	}

	quests = tmpQuests

	mudlog.Info("quests.LoadDataFiles()", "loadedCount", len(quests), "Time Taken", time.Since(start))

}
