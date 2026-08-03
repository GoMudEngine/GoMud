package dialogue

import "testing"

// A PlayerState is a struct of optional callbacks. Six of its ten fields were
// already nil-guarded at the call site; four (HasQuest, HasItem, RemoveItem,
// GiveQuest) were invoked unguarded, so a partially-populated state panicked on
// those and only those. Guard every field consistently.

func TestCheckQuestGate_EmptyPlayerStateDoesNotPanic(t *testing.T) {
	ps := &PlayerState{} // every callback nil

	// Each of these exercises a different unguarded callback.
	cases := []struct {
		name          string
		questRequired []string
		questExcluded []string
		requiresItem  int
	}{
		{"questRequired uses HasQuest", []string{"10-start"}, nil, 0},
		{"questExcluded uses HasQuest", nil, []string{"10-end"}, 0},
		{"requiresItem uses HasItem", nil, nil, 12345},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("checkQuestGate panicked on a nil callback: %v", r)
				}
			}()
			checkQuestGate(tc.questRequired, tc.questExcluded, tc.requiresItem, nil, nil, 0, ps)
		})
	}
}

// A missing callback must not be read as "the player satisfies this gate".
// Absent information is not permission.
func TestCheckQuestGate_MissingCallbackFailsClosed(t *testing.T) {
	ps := &PlayerState{}

	if checkQuestGate([]string{"10-start"}, nil, 0, nil, nil, 0, ps) {
		t.Error("questRequired passed with no HasQuest callback; a gate must not open on missing information")
	}
	if checkQuestGate(nil, nil, 999, nil, nil, 0, ps) {
		t.Error("requiresItem passed with no HasItem callback")
	}
}

// questExcluded is the inverse: with no way to know whether the player holds
// the token, the node must remain available rather than vanish.
func TestCheckQuestGate_MissingCallbackKeepsExclusionOpen(t *testing.T) {
	ps := &PlayerState{}

	if !checkQuestGate(nil, []string{"10-end"}, 0, nil, nil, 0, ps) {
		t.Error("questExcluded hid the node with no HasQuest callback; exclusion should not fire on unknown state")
	}
}

func TestApplyQuestEffects_EmptyPlayerStateDoesNotPanic(t *testing.T) {
	ps := &PlayerState{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("applyQuestEffects panicked on a nil callback: %v", r)
		}
	}()

	applyQuestEffects(
		"10-start",
		4242,
		4243,
		&QuestFlagSet{Key: "10-branch", Value: "rhett"},
		[]RepBump{{Faction: "warren", Delta: 5}},
		100,
		ps,
	)
}

// A fully-populated state must still fire everything — the guards must not
// silently disable working callbacks.
func TestApplyQuestEffects_PopulatedStateFiresEverything(t *testing.T) {
	var removed, given, granted, flagged, bumped, gold bool

	ps := &PlayerState{
		HasQuest:     func(string) bool { return false },
		HasItem:      func(int) bool { return true },
		RemoveItem:   func(int) bool { removed = true; return true },
		GiveQuest:    func(string) { granted = true },
		GiveItem:     func(int) bool { given = true; return true },
		SetQuestFlag: func(string, string) { flagged = true },
		BumpRep:      func(string, int) { bumped = true },
		GiveGold:     func(int) { gold = true },
	}

	applyQuestEffects(
		"10-start",
		4242,
		4243,
		&QuestFlagSet{Key: "10-branch", Value: "rhett"},
		[]RepBump{{Faction: "warren", Delta: 5}},
		100,
		ps,
	)

	for name, fired := range map[string]bool{
		"RemoveItem": removed, "GiveQuest": granted, "GiveItem": given,
		"SetQuestFlag": flagged, "BumpRep": bumped, "GiveGold": gold,
	} {
		if !fired {
			t.Errorf("%s did not fire on a fully-populated PlayerState", name)
		}
	}
}
