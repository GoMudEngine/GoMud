package conversations

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

var (
	converseCheckCache   = map[string]bool{}
	conversations        = map[int]*Conversation{}
	conversationCounter  = map[string]int{}
	conversationUniqueId = 0
)

// Returns a non empty ConversationId if successful
func AttemptConversation(initiatorMobId int, initatorInstanceId int, initiatorName string, participantInstanceId int, participantName string, zone string, forceIndex ...int) int {

	initiatorName = strings.ToLower(initiatorName)
	participantName = strings.ToLower(participantName)
	zone = ZoneNameSanitize(zone)

	convFolder := string(configs.GetFilePathsConfig().DataFiles) + `/conversations`

	fileName := fmt.Sprintf("%s/%d.yaml", zone, initiatorMobId)

	filePath := util.FilePath(convFolder + `/` + fileName)

	var bytes []byte

	if _, err := os.Stat(filePath); err == nil {
		bytes, err = util.ReadFile(filePath)
		if err != nil {
			mudlog.Error("AttemptConversation()", "error", "Problem reading conversation datafile "+filePath+": "+err.Error())
			return 0
		}
	} else if pluginBytes, ok := readPluginConversationFile(zone, initiatorMobId); ok {
		// Fall back to a plugin-provided conversation file (disk takes priority).
		bytes = pluginBytes
	} else {
		return 0
	}

	var dataFile []ConversationData

	err := yaml.Unmarshal(bytes, &dataFile)
	if err != nil {
		mudlog.Error("AttemptConversation()", "error", "Problem unmarshalling conversation datafile "+filePath+": "+err.Error())
		return 0
	}

	// Actual chosen conversation index
	chosenIndex := 0

	if len(forceIndex) > 0 && forceIndex[0] >= 0 && forceIndex[0] < len(dataFile) {
		chosenIndex = forceIndex[0]
	} else {
		possibleConversations := []int{}
		for i, content := range dataFile {

			supportedNameList := content.Supported[initiatorName]
			if supportedNameList2, ok2 := content.Supported[`*`]; ok2 {
				supportedNameList = append(supportedNameList, supportedNameList2...)
			}

			if len(supportedNameList) > 0 {
				for _, supportedName := range supportedNameList {
					if supportedName == participantName || supportedName == `*` {
						possibleConversations = append(possibleConversations, i)
					}
				}
			}
		}

		if len(possibleConversations) == 0 {
			return 0
		}

		lowestCount := -1
		for _, index := range possibleConversations {
			val := conversationCounter[fmt.Sprintf(`%s:%d`, fileName, index)]
			if val < lowestCount || lowestCount == -1 {
				lowestCount = val
				chosenIndex = index
			}
		}
	}

	trackingTag := fmt.Sprintf(`%s:%d`, fileName, chosenIndex)
	conversationCounter[trackingTag] = conversationCounter[trackingTag] + 1

	conversationUniqueId++

	conversations[conversationUniqueId] = &Conversation{
		Id:             conversationUniqueId,
		MobInstanceId1: initatorInstanceId,
		MobInstanceId2: participantInstanceId,
		StartRound:     util.GetRoundCount(),
		// LastRound must start at the current round - the cleanup in
		// getConversation treats rounds since LastRound as staleness, and a
		// zero value makes a brand-new conversation look ancient and
		// eligible for deletion before its first action fires.
		LastRound:  util.GetRoundCount(),
		Position:   0,
		ActionList: dataFile[chosenIndex].Conversation,
	}

	return conversationUniqueId
}

func Destroy(conversationId int) {
	delete(conversations, conversationId)
}

func IsComplete(conversationId int) bool {
	c := getConversation(conversationId)
	if c == nil {
		return true
	}
	if c.Position >= len(c.ActionList) {
		Destroy(conversationId)
		return true
	}
	return false
}

func GetNextActions(convId int) (mob1 int, mob2 int, actions []string) {
	c := getConversation(convId)
	if c == nil {
		return 0, 0, []string{}
	}
	na := c.NextActions(util.GetRoundCount())

	return c.MobInstanceId1, c.MobInstanceId2, na
}

type Conversation struct {
	Id             int
	MobInstanceId1 int
	MobInstanceId2 int
	StartRound     uint64
	LastRound      uint64
	// What the actions are and where we are in them
	Position   int
	ActionList [][]string
}

func ZoneNameSanitize(zone string) string {
	if zone == "" {
		return ""
	}
	// Convert spaces to underscores
	zone = strings.ReplaceAll(zone, " ", "_")
	// Lowercase it all, and add a slash at the end
	return strings.ToLower(zone)
}

func HasConverseFile(mobId int, zone string) bool {

	zone = ZoneNameSanitize(zone)

	cacheKey := strconv.Itoa(mobId) + `-` + zone
	if result, ok := converseCheckCache[cacheKey]; ok {
		return result
	}

	convFolder := string(configs.GetFilePathsConfig().DataFiles) + `/conversations`

	fileName := fmt.Sprintf("%s/%d.yaml", zone, mobId)

	filePath := util.FilePath(convFolder + `/` + fileName)

	exists := false
	if _, err := os.Stat(filePath); err == nil {
		exists = true
	} else if hasPluginConversationFile(zone, mobId) {
		exists = true
	}

	converseCheckCache[cacheKey] = exists

	return exists
}

func (c *Conversation) NextActions(roundNow uint64) []string {

	if c.LastRound == roundNow {
		return []string{}
	}

	c.LastRound = roundNow

	pos := c.Position
	if c.Position >= len(c.ActionList) {
		return []string{}
	}

	c.Position++

	return append([]string{}, c.ActionList[pos]...)
}

func getConversation(conversationId int) *Conversation {

	if util.Rand(50) == 0 { // 2% chance to do a quick maintenance
		rNow := util.GetRoundCount()
		for id, info := range conversations {
			if rNow-info.LastRound > 10 {
				delete(conversations, id)
			}
		}
	}

	if conversation, ok := conversations[conversationId]; ok {
		return conversation
	}

	return nil
}
