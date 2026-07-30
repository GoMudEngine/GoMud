package questengine

import "testing"

// stubMobRooms installs a deterministic mob->rooms lookup for the duration of a
// test. The real lookup scans live mob instances; tests must not depend on a
// running world.
func stubMobRooms(t *testing.T, byMob map[int][]int) {
	t.Helper()
	prev := mobRoomLookup
	mobRoomLookup = func(mobId int) (int, bool) {
		rooms := byMob[mobId]
		if len(rooms) != 1 {
			// 0 instances (dead/unspawned) or 2+ (ambiguous) => not resolvable.
			return 0, false
		}
		return rooms[0], true
	}
	t.Cleanup(func() { mobRoomLookup = prev })
}

// The whole point of map_target_mob: a scheduled NPC moves during the day, so a
// static room is wrong for part of it. The marker must follow the NPC.
func TestResolveQuestTarget_MobTargetFollowsTheNPC(t *testing.T) {
	e := NewEngine()
	e.RegisterQuest(&QuestDef{
		QuestId: 10,
		Steps:   []QuestStep{{Id: "report", MapTargetMob: 96}},
	})

	// Daytime: Marek is behind the bar (472).
	stubMobRooms(t, map[int][]int{96: {472}})
	if got := e.ResolveQuestTarget(10, "report"); got != 472 {
		t.Fatalf("daytime: want 472, got %d", got)
	}

	// Night: same quest, same step — he is asleep upstairs (5102).
	stubMobRooms(t, map[int][]int{96: {5102}})
	if got := e.ResolveQuestTarget(10, "report"); got != 5102 {
		t.Fatalf("night: want 5102 (the marker must follow him), got %d", got)
	}
}

// map_target_mob takes precedence over a static map_target, and the static value
// remains as the fallback for when the NPC cannot be located.
func TestResolveQuestTarget_MobTargetBeatsStaticButFallsBack(t *testing.T) {
	e := NewEngine()
	e.RegisterQuest(&QuestDef{
		QuestId: 11,
		Steps:   []QuestStep{{Id: "s", MapTargetMob: 96, MapTarget: 472}},
	})

	stubMobRooms(t, map[int][]int{96: {5102}})
	if got := e.ResolveQuestTarget(11, "s"); got != 5102 {
		t.Fatalf("live mob should win over static map_target: want 5102, got %d", got)
	}

	// Mob dead / not spawned -> fall back to the authored static room.
	stubMobRooms(t, map[int][]int{})
	if got := e.ResolveQuestTarget(11, "s"); got != 472 {
		t.Fatalf("unresolvable mob should fall back to static: want 472, got %d", got)
	}
}

// Ambiguity must NOT be guessed at. A generic mob template with several live
// instances has no single correct answer, so the resolver declines and lets the
// static fallback (or nothing) stand. A wrong marker is worse than none.
func TestResolveQuestTarget_AmbiguousMobDeclines(t *testing.T) {
	e := NewEngine()
	e.RegisterQuest(&QuestDef{
		QuestId: 12,
		Steps: []QuestStep{
			{Id: "withFallback", MapTargetMob: 500, MapTarget: 4001},
			{Id: "noFallback", MapTargetMob: 500},
		},
	})
	stubMobRooms(t, map[int][]int{500: {4001, 4002, 4003}}) // three live instances

	if got := e.ResolveQuestTarget(12, "withFallback"); got != 4001 {
		t.Fatalf("ambiguous mob with static fallback: want 4001, got %d", got)
	}
	if got := e.ResolveQuestTarget(12, "noFallback"); got != 0 {
		t.Fatalf("ambiguous mob with no fallback must yield NO marker, got %d", got)
	}
}

// -1 stays the hard off switch. It must beat map_target_mob too, otherwise an
// author who deliberately removed a marker cannot keep it removed.
func TestResolveQuestTarget_MinusOneStillSuppressesMobTarget(t *testing.T) {
	e := NewEngine()
	e.RegisterQuest(&QuestDef{
		QuestId: 13,
		Steps:   []QuestStep{{Id: "s", MapTarget: -1, MapTargetMob: 96}},
	})
	stubMobRooms(t, map[int][]int{96: {472}})
	if got := e.ResolveQuestTarget(13, "s"); got != 0 {
		t.Fatalf("map_target -1 must suppress even a resolvable mob target, got %d", got)
	}
}

// A mob target must not resurrect a marker on the terminal step.
func TestResolveQuestTarget_MobTargetIgnoredOnEndStep(t *testing.T) {
	e := NewEngine()
	e.RegisterQuest(&QuestDef{
		QuestId: 14,
		Steps:   []QuestStep{{Id: "end", MapTargetMob: 96}},
	})
	stubMobRooms(t, map[int][]int{96: {472}})
	if got := e.ResolveQuestTarget(14, "end"); got != 0 {
		t.Fatalf("end step must never carry a marker, got %d", got)
	}
}

// An unresolvable mob with no static fallback still allows room_enter
// inference — the mob target should not shadow the existing mechanism.
func TestResolveQuestTarget_MobTargetFallsThroughToInference(t *testing.T) {
	e := NewEngine()
	e.RegisterQuest(&QuestDef{
		QuestId: 15,
		Steps:   []QuestStep{{Id: "s", MapTargetMob: 96}},
		Triggers: []TriggerDef{
			{Event: "room_enter", Room: 4242, Conditions: Conditions{Has: []string{"15-s"}}},
		},
	})
	stubMobRooms(t, map[int][]int{}) // not spawned
	if got := e.ResolveQuestTarget(15, "s"); got != 4242 {
		t.Fatalf("should fall through to room_enter inference: want 4242, got %d", got)
	}
}
