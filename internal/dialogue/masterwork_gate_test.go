package dialogue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTreeAdvance_MasterworkGate exercises the masterworkRequired gate on a
// TreeNode (Veyra's "show me an item you crafted at skill 50+" intro gate).
// Mirrors TestIntegration_DialogueTreeWithQuestGates's PlayerState setup.
func TestTreeAdvance_MasterworkGate(t *testing.T) {
	df := &DialogueFile{
		MobId:       6000,
		Zone:        "test",
		DefaultMood: "neutral",
		Tree: &Tree{
			Nodes: []TreeNode{
				{
					Id:                 "masterwork-intro",
					Triggers:           []string{"masterwork", "commission"},
					Text:               "Ah, a fellow artisan of skill. I'll take your commission.",
					MasterworkRequired: 50,
				},
				{
					Id:       "masterwork-fallback",
					Triggers: []string{"masterwork", "commission"},
					Text:     "Come back when you've proven your craft.",
				},
			},
		},
	}

	userId := 600

	// hasMasterwork toggled by the test to flip HasOwnMasterwork's answer.
	hasMasterwork := false
	ps := &PlayerState{
		HasQuest:  func(token string) bool { return false },
		GiveQuest: func(token string) {},
		HasOwnMasterwork: func(skillMin int) bool {
			return hasMasterwork && skillMin <= 50
		},
	}

	// Below the gate: the masterwork node is hidden, falls through to the
	// unfiltered fallback node.
	text, _, _, advanced := TreeAdvance(df, 6000, userId, "commission", ps)
	assert.True(t, advanced, "some node should advance even without a masterwork")
	assert.Equal(t, "Come back when you've proven your craft.", text,
		"gate should hide the masterwork node when HasOwnMasterwork is false")

	// Above the gate: the masterwork node now matches first (it's earlier in
	// the node list and matches the same triggers).
	hasMasterwork = true
	ResetMemory(6000, userId)
	text, _, _, advanced = TreeAdvance(df, 6000, userId, "commission", ps)
	assert.True(t, advanced)
	assert.Equal(t, "Ah, a fellow artisan of skill. I'll take your commission.", text,
		"gate should reveal the masterwork node once HasOwnMasterwork returns true")

	// Backward compat: an old PlayerState with a nil HasOwnMasterwork callback
	// must not block the gated node (nil-guard skips the check entirely).
	hasMasterwork = false
	ResetMemory(6000, userId+1)
	legacyPs := &PlayerState{
		HasQuest:  func(token string) bool { return false },
		GiveQuest: func(token string) {},
	}
	text, _, _, advanced = TreeAdvance(df, 6000, userId+1, "commission", legacyPs)
	assert.True(t, advanced)
	assert.Equal(t, "Ah, a fellow artisan of skill. I'll take your commission.", text,
		"nil HasOwnMasterwork callback should skip the gate, not fail it (backward compat)")

	// Clean up
	ResetMemory(6000, userId)
	ResetMemory(6000, userId+1)
	delete(moodCache, 6000)
}
