package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// bridgeTestQuestYAML is a single quest fixture that is valid for BOTH the
// legacy quests.Quest loader (questid/name/steps/flags — parsed by
// quests.LoadDataFiles) and the questengine.QuestDef loader
// (questid/name/steps/flags/triggers — parsed by questengine.LoadDataFiles).
// Real quests in this repo (e.g. 79-commission_vitalis_bandolier.yaml) are
// exactly this shape: a quest_granted trigger on "<id>-start" that only a
// dialogue grantsQuest can ever satisfy, which is the exact bug this test
// guards against.
const bridgeTestQuestYAML = `
questid: 999001
name: Test Bridge Quest
description: Fixture proving dialogue/legacy grants bridge into questengine quest_granted triggers.
steps:
  - id: start
    description: "Started."
  - id: end
    description: "Done."
flags:
  - key: fired-start
    values: ["yes"]
    description: "set by the quest_granted trigger on 999001-start"
  - key: fired-end
    values: ["yes"]
    description: "set by the quest_granted trigger on 999001-end"
triggers:
  - event: quest_granted
    quest_token: "999001-start"
    actions:
      - set_flag: {key: "999001-fired-start", value: "yes"}
  - event: quest_granted
    quest_token: "999001-end"
    actions:
      - set_flag: {key: "999001-fired-end", value: "yes"}
`

// setupBridgeTestQuest writes bridgeTestQuestYAML into a temp
// "<tmp>/quests/999001-test_bridge_quest.yaml" file, points
// FilePaths.DataFiles at tmp, and loads it through BOTH the legacy quests
// package and the questengine package (mirroring main.go's boot sequence,
// which loads the exact same quest YAML files into both systems). Returns a
// cleanup func that restores the original FilePaths.DataFiles config value.
func setupBridgeTestQuest(t *testing.T) func() {
	t.Helper()

	tmp := t.TempDir()
	questsDir := filepath.Join(tmp, "quests")
	require.NoError(t, os.MkdirAll(questsDir, 0755))

	fixturePath := filepath.Join(questsDir, "999001-test_bridge_quest.yaml")
	require.NoError(t, os.WriteFile(fixturePath, []byte(bridgeTestQuestYAML), 0644))

	origDataFiles := configs.GetFilePathsConfig().DataFiles.String()

	require.NoError(t, configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles": tmp,
	}))

	// Load the fixture into both systems, exactly like main.go's boot does
	// for the real _datafiles/world/dogmud/quests directory.
	quests.LoadDataFiles()
	questengine.LoadDataFiles()

	return func() {
		_ = configs.AddOverlayOverrides(map[string]any{
			"FilePaths.DataFiles": origDataFiles,
		})
	}
}

// TestHandleQuestUpdate_BridgesFreshDialogueGrantToQuestGrantedTrigger proves
// the fix for the bug that blocked the whole pinnacle commission system: a
// quest token granted via dialogue's grantsQuest (or any other legacy path
// that funnels through events.Quest -> HandleQuestUpdate) previously never
// notified the questengine, so a quest_granted trigger on that token (e.g.
// quest 79's charge_gold/learn_recipe actions on "79-start") silently never
// fired.
//
// This simulates the FRESH dialogue-grant path directly: HandleQuestUpdate
// is invoked with a QuestToken the character has never seen before (exactly
// what happens when dialogue.go's grantsQuest queues events.Quest{...}).
// GiveQuestToken must return true (fresh), and the bridge must have notified
// the questengine's quest_granted trigger, observable via the flag it sets.
func TestHandleQuestUpdate_BridgesFreshDialogueGrantToQuestGrantedTrigger(t *testing.T) {
	cleanupQuest := setupBridgeTestQuest(t)
	defer cleanupQuest()

	u := users.NewTestUser(501, "bridgeuser", "BridgeUser", 5001)
	cleanupUsers := users.SeedUsersForTest(map[int]*users.UserRecord{501: u})
	defer cleanupUsers()

	require.False(t, u.Character.HasQuest("999001-start"),
		"sanity: character must not already have this quest token")
	require.Equal(t, "", u.Character.GetQuestFlag("999001-fired-start"),
		"sanity: trigger flag must not be pre-set")

	result := HandleQuestUpdate(events.Quest{
		UserId:     501,
		QuestToken: "999001-start",
	})

	require.Equal(t, events.Continue, result)
	require.True(t, u.Character.HasQuest("999001-start"),
		"the token must be granted")
	require.Equal(t, "yes", u.Character.GetQuestFlag("999001-fired-start"),
		"a fresh dialogue-style grant must fire the quest_granted trigger's set_flag action — "+
			"this is the bridge under test")
}

// TestHandleQuestUpdate_DoesNotDoubleFireQuestengineInitiatedGrant proves the
// freshness guard: when the questengine ITSELF has already advanced the
// token synchronously (as bridge.GrantQuest does before queueing the legacy
// events.Quest for an "end" step — see internal/questengine/bridge.go:82),
// HandleQuestUpdate's GiveQuestToken call returns false (not fresh), and the
// bridge must NOT re-notify the questengine's quest_granted trigger a second
// time.
func TestHandleQuestUpdate_DoesNotDoubleFireQuestengineInitiatedGrant(t *testing.T) {
	cleanupQuest := setupBridgeTestQuest(t)
	defer cleanupQuest()

	u := users.NewTestUser(502, "bridgeuser2", "BridgeUser2", 5002)
	cleanupUsers := users.SeedUsersForTest(map[int]*users.UserRecord{502: u})
	defer cleanupUsers()

	// Get the character to "start" first (a legitimate prior grant), via the
	// same path as test 1.
	require.Equal(t, events.Continue, HandleQuestUpdate(events.Quest{
		UserId:     502,
		QuestToken: "999001-start",
	}))
	require.True(t, u.Character.HasQuest("999001-start"))

	// Simulate what questengine.GameBridge.GrantQuest does for an "end" step:
	// it calls GiveQuestToken itself, synchronously, BEFORE queueing the
	// legacy events.Quest that HandleQuestUpdate will process.
	require.True(t, u.Character.GiveQuestToken("999001-end"),
		"the simulated questengine-initiated grant must succeed (advances the token)")
	require.Equal(t, "", u.Character.GetQuestFlag("999001-fired-end"),
		"sanity: the end-trigger flag must not be set yet")

	// Now HandleQuestUpdate processes the legacy events.Quest for "999001-end"
	// that GrantQuest would have queued. Because the token was already
	// advanced above, GiveQuestToken here must return false, and the bridge
	// must skip notifying the quest_granted trigger a second time.
	result := HandleQuestUpdate(events.Quest{
		UserId:     502,
		QuestToken: "999001-end",
	})

	require.Equal(t, events.Continue, result)
	require.Equal(t, "", u.Character.GetQuestFlag("999001-fired-end"),
		"a questengine-pre-set (non-fresh) grant must NOT re-fire the quest_granted "+
			"trigger a second time — this is the double-fire guard")
}
