package dialogue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDialogue_NodeBumpsRep verifies a tree node can apply faction-reputation
// changes when it matches, so an ask-path delivery node can replicate the
// give-path's bump_rep actions (Tier C delivery quests: Q67/Q68/Q70/Q74). The
// rep change is applied via the PlayerState.BumpRep callback, mirroring how
// grantsQuest/requiresItem already route through PlayerState.
func TestDialogue_NodeBumpsRep(t *testing.T) {
	df := &DialogueFile{
		MobId: 6001, Zone: "test", DefaultMood: "neutral",
		Tree: &Tree{Nodes: []TreeNode{
			{
				Id:            "handoff",
				Triggers:      []string{"deliver"},
				QuestRequired: []string{"x-mid"},
				GrantsQuest:   "x-end",
				BumpsRep:      []RepBump{{Faction: "alpha", Delta: 20}, {Faction: "beta", Delta: -15}},
				Text:          "done",
			},
		}},
	}

	bumps := map[string]int{}
	granted := ""
	ps := &PlayerState{
		HasQuest:  func(tok string) bool { return tok == "x-mid" },
		GiveQuest: func(tok string) { granted = tok },
		BumpRep:   func(faction string, delta int) { bumps[faction] += delta },
	}

	_, _, _, advanced := TreeAdvance(df, 6001, 7001, "deliver", ps)
	assert.True(t, advanced, "the handoff node must advance")
	assert.Equal(t, "x-end", granted, "the node must grant its token")
	assert.Equal(t, 20, bumps["alpha"], "alpha rep must be bumped +20")
	assert.Equal(t, -15, bumps["beta"], "beta rep must be bumped -15")
}

// TestDialogue_NodeGivesGold verifies a tree node can award gold when it
// matches, so an ask-path delivery node can replicate a give-path's give_gold
// action (e.g. Q19/Q68/Q74 hand the player gold on hand-over).
func TestDialogue_NodeGivesGold(t *testing.T) {
	df := &DialogueFile{
		MobId: 6002, Zone: "test", DefaultMood: "neutral",
		Tree: &Tree{Nodes: []TreeNode{
			{
				Id:            "handoff",
				Triggers:      []string{"deliver"},
				QuestRequired: []string{"x-mid"},
				GrantsQuest:   "x-end",
				GivesGold:     350,
				Text:          "done",
			},
		}},
	}

	gold := 0
	ps := &PlayerState{
		HasQuest:  func(tok string) bool { return tok == "x-mid" },
		GiveQuest: func(string) {},
		GiveGold:  func(amount int) { gold += amount },
	}

	_, _, _, advanced := TreeAdvance(df, 6002, 7002, "deliver", ps)
	assert.True(t, advanced, "the handoff node must advance")
	assert.Equal(t, 350, gold, "the node must award 350 gold")
}
