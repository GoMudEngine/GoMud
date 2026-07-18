package questengine

import "testing"

func newTargetTestEngine() *Engine {
	e := NewEngine()
	e.RegisterQuest(&QuestDef{
		QuestId: 32,
		Name:    "First Blood",
		Steps: []QuestStep{
			{Id: "start", MapTarget: 4201}, // explicit target
			{Id: "arrive"},                 // inferred via room_enter trigger
			{Id: "nowhere"},                // no target
			{Id: "end"},
		},
		Triggers: []TriggerDef{
			{
				Event:      "room_enter",
				Room:       4207,
				Conditions: Conditions{Has: []string{"32-arrive"}},
			},
		},
	})
	return e
}

func TestResolveQuestTarget_ExplicitMapTarget(t *testing.T) {
	e := newTargetTestEngine()
	if got := e.ResolveQuestTarget(32, "start"); got != 4201 {
		t.Fatalf("map_target step: want 4201, got %d", got)
	}
}

func TestResolveQuestTarget_RoomEnterInference(t *testing.T) {
	e := newTargetTestEngine()
	if got := e.ResolveQuestTarget(32, "arrive"); got != 4207 {
		t.Fatalf("room_enter inference: want 4207, got %d", got)
	}
}

func TestResolveQuestTarget_NoTarget(t *testing.T) {
	e := newTargetTestEngine()
	if got := e.ResolveQuestTarget(32, "nowhere"); got != 0 {
		t.Fatalf("no target: want 0, got %d", got)
	}
}

func TestResolveQuestTarget_TerminalAndUnknown(t *testing.T) {
	e := newTargetTestEngine()
	if got := e.ResolveQuestTarget(32, "end"); got != 0 {
		t.Fatalf("end step: want 0, got %d", got)
	}
	if got := e.ResolveQuestTarget(999, "start"); got != 0 {
		t.Fatalf("unknown quest: want 0, got %d", got)
	}
	if got := e.ResolveQuestTarget(32, ""); got != 0 {
		t.Fatalf("empty step: want 0, got %d", got)
	}
}
