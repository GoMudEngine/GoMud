package dialogue

import "testing"

// A node carrying BOTH givesItem and grantsQuest must not grant the quest
// when the item cannot be delivered (GiveItem returns false). Otherwise the
// player holds the quest token, lacks the item, and the granting node is
// then hidden by questExcluded — an unrecoverable soft-lock. The engine
// must also leave requiresItem in the player's pack, skip flags/rep/gold,
// and leave tree memory untouched so the node re-fires once the player
// makes room. (Mirror of the questengine bridge fix from Tier 2.)

// failingGiveState returns a PlayerState whose GiveItem always fails, plus
// pointers to the observation flags.
func failingGiveState(delivered bool) (*PlayerState, *bool, *bool, *bool, *bool) {
	granted := false
	removed := false
	flagSet := false
	goldGiven := false
	ps := &PlayerState{
		HasQuest:   func(string) bool { return false },
		HasItem:    func(int) bool { return true },
		RemoveItem: func(int) bool { removed = true; return true },
		GiveQuest:  func(string) { granted = true },
		GiveItem:   func(int) bool { return delivered },
		GetQuestFlag: func(string) string {
			return ""
		},
		SetQuestFlag: func(string, string) { flagSet = true },
		GiveGold:     func(int) { goldGiven = true },
	}
	return ps, &granted, &removed, &flagSet, &goldGiven
}

func failedDeliveryFile() *DialogueFile {
	return &DialogueFile{
		MobId: 9999,
		Tree: &Tree{
			Root: TreeRoot{Text: "Hello."},
			Nodes: []TreeNode{
				{
					Id:            "reward",
					Triggers:      []string{"reward"},
					Text:          "Take this, and my thanks.",
					GrantsQuest:   "77-start",
					GivesItem:     4242,
					RequiresItem:  1111,
					SetsQuestFlag: &QuestFlagSet{Key: "77-k", Value: "v"},
					GivesGold:     10,
					Unlocks:       []string{"after"},
				},
			},
		},
	}
}

func TestTreeAdvance_GiveItemFails_NoQuestNoConsumption(t *testing.T) {
	df := failedDeliveryFile()
	ps, granted, removed, flagSet, goldGiven := failingGiveState(false)

	text, _, _, advanced := TreeAdvance(df, 800001, 42, "reward", ps)
	if !advanced {
		t.Fatal("expected the node to match and speak")
	}
	if text == "" {
		t.Fatal("expected node text to be returned")
	}
	if *granted {
		t.Error("quest granted despite failed item delivery")
	}
	if *removed {
		t.Error("requiresItem removed despite failed item delivery")
	}
	if *flagSet {
		t.Error("quest flag set despite failed item delivery")
	}
	if *goldGiven {
		t.Error("gold given despite failed item delivery")
	}

	// The node must NOT be consumed: memory untouched, unlocks not recorded.
	mem := GetMemory(800001, 42)
	if mem.UnlockedNodes["after"] {
		t.Error("node unlocks recorded despite failed item delivery")
	}

	// After the player makes room, the same ask must deliver everything.
	ps2, granted2, removed2, flagSet2, goldGiven2 := failingGiveState(true)
	_, _, _, advanced2 := TreeAdvance(df, 800001, 42, "reward", ps2)
	if !advanced2 {
		t.Fatal("expected the node to re-fire after making room")
	}
	if !*granted2 || !*removed2 || !*flagSet2 || !*goldGiven2 {
		t.Errorf("expected all effects on retry: granted=%v removed=%v flag=%v gold=%v",
			*granted2, *removed2, *flagSet2, *goldGiven2)
	}
}

func TestMatch_GiveItemFails_NoQuest(t *testing.T) {
	df := &DialogueFile{
		MobId: 9998,
		Patterns: []Pattern{
			{
				Keywords:    []string{"gift"},
				Responses:   []string{"Here you go."},
				GrantsQuest: "78-start",
				GivesItem:   4242,
			},
		},
	}
	ps, granted, _, _, _ := failingGiveState(false)
	_, _, ok := Match(df, 800002, "gift", ps)
	if !ok {
		t.Fatal("expected pattern to match")
	}
	if *granted {
		t.Error("pattern granted quest despite failed item delivery")
	}
}

func TestGreet_VariantGiveItemFails_NoQuest(t *testing.T) {
	df := &DialogueFile{
		MobId: 9997,
		Tree: &Tree{
			Root: TreeRoot{
				Text: "Default greeting.",
				Variants: []QuestGreeting{
					{
						Text:        "You look ready. Take this.",
						GrantsQuest: "79-start",
						GivesItem:   4242,
					},
				},
			},
		},
	}
	ps, granted, _, _, _ := failingGiveState(false)
	text, _, ok := Greet(df, 800003, 43, ps)
	if !ok || text == "" {
		t.Fatal("expected a greeting")
	}
	if *granted {
		t.Error("greeting variant granted quest despite failed item delivery")
	}
}

// Nil GiveItem callback with givesItem set keeps the legacy skip-checks
// contract: effects still apply (partial PlayerStates are test/JS territory;
// the production talk.go state always wires GiveItem).
func TestTreeAdvance_NilGiveItemCallback_StillGrants(t *testing.T) {
	df := failedDeliveryFile()
	granted := false
	ps := &PlayerState{
		HasQuest:  func(string) bool { return false },
		HasItem:   func(int) bool { return true },
		GiveQuest: func(string) { granted = true },
	}
	_, _, _, advanced := TreeAdvance(df, 800004, 44, "reward", ps)
	if !advanced {
		t.Fatal("expected the node to match")
	}
	if !granted {
		t.Error("nil GiveItem callback must not block the grant (skip-checks contract)")
	}
}
