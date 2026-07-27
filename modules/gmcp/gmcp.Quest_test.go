package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/dialogue"
	"github.com/GoMudEngine/GoMud/internal/quests"
)

// fakeQuestWorld is an in-memory stand-in for the quests package + engine
// re-index, mirroring fakeMobWorld / fakeDialogueWorld.
type fakeQuestWorld struct {
	specs     map[int]*quests.Quest
	saved     []quests.Quest
	deleted   []int
	reindexed int
	refs      []questRefEntry
	nextId    int
}

func newFakeQuestWorld() *fakeQuestWorld {
	return &fakeQuestWorld{specs: map[int]*quests.Quest{}, nextId: 90000}
}

func fakeQuest(id int, name string) *quests.Quest {
	return &quests.Quest{QuestId: id, Name: name, Steps: []quests.QuestStep{
		{Id: "start"}, {Id: "end"},
	}, Triggers: []quests.TriggerDef{
		{Event: "room_enter", Room: 1, Actions: []quests.ActionDef{{Grant: "%d-start"}}},
	}}
}

func (w *fakeQuestWorld) deps() questDeps {
	v := quests.QuestValidators{
		StepExists:     func(string) bool { return true },
		MobExists:      func(int) bool { return true },
		ItemExists:     func(id int) bool { return id != 888 },
		RoomExists:     func(int) bool { return true },
		BuffExists:     func(int) bool { return true },
		SpellExists:    func(string) bool { return true },
		SkillExists:    func(string) bool { return true },
		StatExists:     func(string) bool { return true },
		RecipeExists:   func(string) bool { return true },
		FactionExists:  func(string) bool { return true },
		FlagDeclared:   func(string, string) bool { return true },
		DialogueGrants: func(string) bool { return false },
		MobHasDialogue: func(int) bool { return true },
	}
	return questDeps{
		load: func(id int) *quests.Quest { return w.specs[id] },
		all: func() []quests.Quest {
			out := []quests.Quest{}
			for _, q := range w.specs {
				out = append(out, *q)
			}
			return out
		},
		save: func(q quests.Quest) error {
			w.saved = append(w.saved, q)
			cp := q
			w.specs[q.QuestId] = &cp
			return nil
		},
		create: func(name string) (int, error) {
			w.nextId++
			w.specs[w.nextId] = fakeQuest(w.nextId, name)
			return w.nextId, nil
		},
		del:        func(id int) error { w.deleted = append(w.deleted, id); delete(w.specs, id); return nil },
		references: func(id int) []questRefEntry { return w.refs },
		reindex:    func() { w.reindexed++ },
		validators: v,
	}
}

func validUpdateReq(id int) quests.Quest {
	q := *fakeQuest(id, "Handler Test")
	q.Triggers[0].Actions[0].Grant = "" // no grant → orphan-step warnings expected
	return q
}

func TestBuildQuestUpdate_RefusesInvalidSavesNothing(t *testing.T) {
	w := newFakeQuestWorld()
	w.specs[90001] = fakeQuest(90001, "Handler Test")

	q := validUpdateReq(90001)
	q.Rewards.ItemId = 888 // fake says it doesn't exist
	res := buildQuestUpdate(w.deps(), q)
	if res.Ok {
		t.Fatal("bad registry ref must be refused")
	}
	if len(w.saved) != 0 || w.reindexed != 0 {
		t.Fatalf("refused update must not save (%d) or reindex (%d)", len(w.saved), w.reindexed)
	}
}

func TestBuildQuestUpdate_WarnOnlySavesWithWarnings(t *testing.T) {
	w := newFakeQuestWorld()
	w.specs[90002] = fakeQuest(90002, "Handler Test")

	res := buildQuestUpdate(w.deps(), validUpdateReq(90002))
	if !res.Ok {
		t.Fatalf("warning-only update should save, got %+v", res)
	}
	if len(w.saved) != 1 || w.reindexed != 1 {
		t.Fatalf("expected 1 save + 1 reindex, got %d/%d", len(w.saved), w.reindexed)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("orphan steps should surface as warnings")
	}
}

func TestBuildQuestListAndGet(t *testing.T) {
	w := newFakeQuestWorld()
	w.specs[90003] = fakeQuest(90003, "Visible Quest")
	w.specs[1000000] = fakeQuest(1000000, "Generic Template")

	rows := buildQuestList(w.deps())
	if len(rows) != 1 || rows[0].Id != 90003 || rows[0].StepCount != 2 || rows[0].TriggerCount != 1 {
		t.Fatalf("list shape wrong (template id must be hidden): %+v", rows)
	}

	detail, ok := buildQuestGet(w.deps(), 90003)
	if !ok || detail.Quest.QuestId != 90003 {
		t.Fatalf("get failed: %+v", detail)
	}
	// 10 events, 9 condition kinds, 23 action types — one vocab entry per
	// TriggerDef/Conditions/ActionDef field (count them, don't estimate).
	if len(detail.Enums.Events) != 10 || len(detail.Enums.Actions) != 23 || len(detail.Enums.Conditions) != 9 {
		t.Fatalf("vocabulary sizes wrong: events=%d conditions=%d actions=%d",
			len(detail.Enums.Events), len(detail.Enums.Conditions), len(detail.Enums.Actions))
	}
}

func TestBuildQuestCreate_ReindexesAndReturnsId(t *testing.T) {
	w := newFakeQuestWorld()
	res := buildQuestCreate(w.deps(), "Fresh Quest")
	if !res.Ok || res.QuestId == 0 {
		t.Fatalf("create should return an id, got %+v", res)
	}
	if w.reindexed != 1 {
		t.Fatalf("create must reindex once, got %d", w.reindexed)
	}
}

func TestBuildQuestDelete_GuardAndClean(t *testing.T) {
	w := newFakeQuestWorld()
	w.specs[90004] = fakeQuest(90004, "Guarded Quest")
	w.refs = []questRefEntry{{Kind: "dialogue", Where: "mob 9 (townzone)", Detail: `grantsQuest "90004-start"`}}

	if res := buildQuestDelete(w.deps(), 90004); res.Ok {
		t.Fatal("referenced quest must not delete")
	}
	if len(w.deleted) != 0 {
		t.Fatal("guarded delete must not reach deps.del")
	}

	w.refs = nil
	res := buildQuestDelete(w.deps(), 90004)
	if !res.Ok || len(w.deleted) != 1 || w.reindexed != 1 {
		t.Fatalf("clean delete should remove + reindex: %+v deleted=%v reindexed=%d", res, w.deleted, w.reindexed)
	}
}

// A dialogue file with no tree (Tree is a POINTER, nil for pattern-only
// NPCs — most of the 302 live files) must walk without panicking. Caught by
// the 5c E2E: the fake deps in the tests above bypass the real walker.
func TestWalkDialogueGates_NilTree(t *testing.T) {
	df := &dialogue.DialogueFile{MobId: 9, Patterns: []dialogue.Pattern{{GrantsQuest: "10-start"}}}
	seen := 0
	walkDialogueGates(df, func(where string, g dialogueGate) { seen++ })
	if seen != 1 {
		t.Fatalf("expected 1 gate visit for the pattern, got %d", seen)
	}
}
