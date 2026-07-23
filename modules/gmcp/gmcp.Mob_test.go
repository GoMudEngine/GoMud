package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// fakeMobWorld is an in-memory stand-in for the mobs package + gmcp-layer
// zone lookup, mirroring fakeItemWorld in gmcp.Item_test.go.
type fakeMobWorld struct {
	specs   map[int]*mobs.Mob
	saved   []mobs.Mob
	deleted []int
	nextId  int
}

func newFakeMobWorld() *fakeMobWorld {
	return &fakeMobWorld{specs: map[int]*mobs.Mob{}, nextId: 90000}
}

func (w *fakeMobWorld) deps() mobDeps {
	return mobDeps{
		load: func(id int) *mobs.Mob { return w.specs[id] },
		save: func(m mobs.Mob) error {
			w.saved = append(w.saved, m)
			cp := m
			w.specs[int(m.MobId)] = &cp
			return nil
		},
		del: func(id int) error { w.deleted = append(w.deleted, id); delete(w.specs, id); return nil },
		create: func(zone string) (int, error) {
			w.nextId++
			m := mobs.Mob{MobId: mobs.MobId(w.nextId), Zone: zone}
			m.Character.Name = "New Mob"
			w.specs[w.nextId] = &m
			return w.nextId, nil
		},
		zoneExists: func(z string) bool { return z == "Testzone" },
		references: func(id int) []mobRefEntry { return nil },
		spawn:      func(mobId, roomId int) (string, error) { return "", nil },
	}
}

func TestBuildMobUpdate_RoundTripsAndPreserves(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90001, Zone: "Testzone",
		Relationships: []mobs.RelationshipYAMLEntry{{To: 42, Type: "friend"}},
		KnowsFacts:    []string{"fact_a"}}
	base.Character.Name = "Old Name"
	w.specs[90001] = base

	req := mobUpdateReq{MobId: 90001, Zone: "Testzone", Name: "Keen Guard",
		Description: "Sharp-eyed.", SpeciesId: 1, StatPool: 60, Archetype: "fighting",
		AutoAggro: true, AIProfile: "tactical", ActivityLevel: 10,
		IdleCommands: []string{"emote scans the road."}, Gold: 25,
		StatTraining: map[string]int{"strength": 10, "perception": 5},
		LootPool:     []int{40001}, Relationships: nil /* form didn't touch them */}
	res := buildMobUpdate(w.deps(), req)
	if !res.Ok {
		t.Fatalf("update should succeed, got %+v", res)
	}
	got := w.saved[0]
	if got.Character.Name != "Keen Guard" || !got.AutoAggro || got.AIProfile != "tactical" ||
		got.StatPool != 60 || got.Character.Gold != 25 ||
		got.Character.Stats.Strength.Training != 10 {
		t.Errorf("fields not round-tripped: %+v", got)
	}
	if len(got.Relationships) != 1 || len(got.KnowsFacts) != 1 {
		t.Errorf("form-absent fields must be preserved from base, got rel=%v facts=%v",
			got.Relationships, got.KnowsFacts)
	}
}

func TestBuildMobCreate_RejectsUnknownZone(t *testing.T) {
	w := newFakeMobWorld()
	if res := buildMobCreate(w.deps(), "Nowhere"); res.Ok {
		t.Fatal("create must reject an unknown zone")
	}
	res := buildMobCreate(w.deps(), "Testzone")
	if !res.Ok || res.MobId == 0 {
		t.Fatalf("create should return an id, got %+v", res)
	}
}

func TestBuildMobGet_MapsFields(t *testing.T) {
	w := newFakeMobWorld()
	m := &mobs.Mob{MobId: 90005, Zone: "Testzone", StatPool: 40, Archetype: "casting",
		ScheduleId: "tz_sched", MaxWander: 3}
	m.Character.Name = "Hermit"
	m.Character.Description = "Quiet."
	m.Character.SpeciesId = 1
	w.specs[90005] = m
	d, ok := buildMobGet(w.deps(), 90005)
	if !ok {
		t.Fatal("expected found")
	}
	if d.Name != "Hermit" || d.StatPool != 40 || d.Archetype != "casting" || d.ScheduleId != "tz_sched" {
		t.Errorf("detail wrong: %+v", d)
	}
}

// TestBuildMobUpdate_RejectsUnknownCrafterRecipe covers correction #8 from the
// implementer notes: crafterRecipeIds must be validated against the crafting
// package's recipe registry (an unknown id names the bad id in the error).
func TestBuildMobUpdate_RejectsUnknownCrafterRecipe(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90006, Zone: "Testzone"}
	base.Character.Name = "Old Name"
	w.specs[90006] = base

	req := mobUpdateReq{MobId: 90006, Zone: "Testzone", Name: "Tinker",
		Description: "Busy.", SpeciesId: 1,
		CrafterRecipeIds: []string{"definitely-not-a-real-recipe-id"}}
	res := buildMobUpdate(w.deps(), req)
	if res.Ok {
		t.Fatal("update must reject an unknown crafter recipe id")
	}
}

// TestBuildMobUpdate_RejectsInvalidLLMProfileJSON covers correction #7: a
// malformed LLMProfileJSON must surface as an error, not silently discard the
// profile.
func TestBuildMobUpdate_RejectsInvalidLLMProfileJSON(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90007, Zone: "Testzone"}
	base.Character.Name = "Old Name"
	w.specs[90007] = base

	req := mobUpdateReq{MobId: 90007, Zone: "Testzone", Name: "Chatty",
		Description: "Talks.", SpeciesId: 1, LLMProfileJSON: "{not valid json"}
	res := buildMobUpdate(w.deps(), req)
	if res.Ok {
		t.Fatal("update must reject malformed llmProfileJson")
	}
}
