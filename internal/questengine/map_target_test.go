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

// A step with map_target -1 means "deliberately no marker" — it must resolve to
// 0 (no marker) AND suppress room_enter inference, so a stray inferable target
// does not re-introduce a marker the author explicitly removed.
func TestResolveQuestTarget_DeliberateNoneSentinel(t *testing.T) {
	e := NewEngine()
	e.RegisterQuest(&QuestDef{
		QuestId: 9998,
		Steps:   []QuestStep{{Id: "nowhere", MapTarget: -1}},
		Triggers: []TriggerDef{
			// A room_enter gated on the step that WOULD infer room 4242 —
			// proves -1 overrides inference.
			{Event: "room_enter", Room: 4242, Conditions: Conditions{Has: []string{"9998-nowhere"}}},
		},
	})
	if got := e.ResolveQuestTarget(9998, "nowhere"); got != 0 {
		t.Fatalf("map_target -1 must resolve to 0 (no marker, inference suppressed); got %d", got)
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
