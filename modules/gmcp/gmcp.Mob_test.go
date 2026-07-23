package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/llm"
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

// TestBuildMobUpdate_LLMProfileSentinelLeavesUntouched covers the "-" branch
// of the LLMProfileJSON sentinel: a payload that never mentions the field
// (as the Build.Mob.Update route pre-sets it before unmarshal) must leave
// whatever LLM profile the base template already has.
func TestBuildMobUpdate_LLMProfileSentinelLeavesUntouched(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90013, Zone: "Testzone",
		LLMProfile: &llm.LLMProfile{Model: "llama3.2", SystemPrompt: "Grumpy blacksmith."}}
	base.Character.Name = "Old Name"
	w.specs[90013] = base

	req := mobUpdateReq{MobId: 90013, Zone: "Testzone", Name: "Old Name",
		Description: "d", SpeciesId: 1, LLMProfileJSON: "-"}
	res := buildMobUpdate(w.deps(), req)
	if !res.Ok {
		t.Fatalf("update should succeed, got %+v", res)
	}
	got := w.saved[0]
	if got.LLMProfile == nil || got.LLMProfile.Model != "llama3.2" {
		t.Errorf("sentinel \"-\" must leave the existing LLM profile untouched, got %+v", got.LLMProfile)
	}
}

// TestBuildMobUpdate_LLMProfileEmptyStringClears covers the "" branch: an
// explicit empty string (as opposed to the "-" sentinel) clears an existing
// profile.
func TestBuildMobUpdate_LLMProfileEmptyStringClears(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90014, Zone: "Testzone",
		LLMProfile: &llm.LLMProfile{Model: "llama3.2"}}
	base.Character.Name = "Old Name"
	w.specs[90014] = base

	req := mobUpdateReq{MobId: 90014, Zone: "Testzone", Name: "Old Name",
		Description: "d", SpeciesId: 1, LLMProfileJSON: ""}
	res := buildMobUpdate(w.deps(), req)
	if !res.Ok {
		t.Fatalf("update should succeed, got %+v", res)
	}
	got := w.saved[0]
	if got.LLMProfile != nil {
		t.Errorf("empty llmProfileJson must clear the profile, got %+v", got.LLMProfile)
	}
}

// TestBuildMobUpdate_NonNilRelationshipsReplaceWholesale complements the
// nil-preserve assertion in TestBuildMobUpdate_RoundTripsAndPreserves: when
// the form DOES submit a Relationships list (non-nil), it replaces the base's
// list wholesale rather than merging or appending.
func TestBuildMobUpdate_NonNilRelationshipsReplaceWholesale(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90015, Zone: "Testzone",
		Relationships: []mobs.RelationshipYAMLEntry{{To: 42, Type: "friend"}, {To: 43, Type: "rival"}}}
	base.Character.Name = "Old Name"
	w.specs[90015] = base

	req := mobUpdateReq{MobId: 90015, Zone: "Testzone", Name: "Old Name",
		Description: "d", SpeciesId: 1,
		Relationships: []mobRelRow{{To: 99, Type: "friend"}}}
	res := buildMobUpdate(w.deps(), req)
	if !res.Ok {
		t.Fatalf("update should succeed, got %+v", res)
	}
	got := w.saved[0]
	if len(got.Relationships) != 1 || got.Relationships[0].To != 99 {
		t.Errorf("a non-nil Relationships payload must replace wholesale, got %+v", got.Relationships)
	}
}

// TestBuildMobUpdate_PreservesCustomizedItemInstances is the regression test
// for the item-instance-preservation fix: an equipped weapon carrying real
// per-instance customization (EnchantType/EnchantTier/Adjectives) and a
// carried item with non-default Uses/Blob must survive a save that echoes
// back the SAME item ids in those slots — an admin editing an unrelated field
// (e.g. just the name) must not silently revert an enchanted boss weapon (or
// any other customized instance) to a spec-default one.
func TestBuildMobUpdate_PreservesCustomizedItemInstances(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90016, Zone: "Testzone"}
	base.Character.Name = "Old Name"
	base.Character.Equipment.Weapon = items.Item{
		ItemId: 501, EnchantType: "flaming", EnchantTier: 2, Adjectives: []string{"flaming"},
	}
	base.Character.Items = []items.Item{
		{ItemId: 502, Uses: 3, Adjectives: []string{"rusty"}, Blob: "custom-blob"},
	}
	w.specs[90016] = base

	req := mobUpdateReq{MobId: 90016, Zone: "Testzone", Name: "Renamed",
		Description: "d", SpeciesId: 1,
		Equipment:    map[string]int{"weapon": 501}, // same id echoed back
		CarriedItems: []int{502},                    // same id echoed back
	}
	res := buildMobUpdate(w.deps(), req)
	if !res.Ok {
		t.Fatalf("update should succeed, got %+v", res)
	}
	got := w.saved[0]
	weapon := got.Character.Equipment.Weapon
	if weapon.ItemId != 501 || weapon.EnchantType != "flaming" || weapon.EnchantTier != 2 ||
		len(weapon.Adjectives) != 1 || weapon.Adjectives[0] != "flaming" {
		t.Errorf("equipped weapon instance customization was not preserved: %+v", weapon)
	}
	if len(got.Character.Items) != 1 {
		t.Fatalf("expected 1 carried item, got %+v", got.Character.Items)
	}
	carried := got.Character.Items[0]
	if carried.ItemId != 502 || carried.Uses != 3 || carried.Blob != "custom-blob" ||
		len(carried.Adjectives) != 1 || carried.Adjectives[0] != "rusty" {
		t.Errorf("carried item instance customization was not preserved: %+v", carried)
	}
}

// TestBuildMobUpdate_ChangedItemIdDropsOldCustomization complements the
// preservation test: submitting a DIFFERENT id in a slot/list must NOT carry
// forward the old instance's customization — it must be treated as a
// new/changed item, not a reuse of whatever used to be there.
func TestBuildMobUpdate_ChangedItemIdDropsOldCustomization(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90017, Zone: "Testzone"}
	base.Character.Name = "Old Name"
	base.Character.Equipment.Weapon = items.Item{
		ItemId: 501, EnchantType: "flaming", Adjectives: []string{"flaming"},
	}
	base.Character.Items = []items.Item{
		{ItemId: 502, Uses: 3, Adjectives: []string{"rusty"}},
	}
	w.specs[90017] = base

	req := mobUpdateReq{MobId: 90017, Zone: "Testzone", Name: "Old Name",
		Description: "d", SpeciesId: 1,
		Equipment:    map[string]int{"weapon": 999}, // different id
		CarriedItems: []int{888},                    // different id
	}
	res := buildMobUpdate(w.deps(), req)
	if !res.Ok {
		t.Fatalf("update should succeed, got %+v", res)
	}
	got := w.saved[0]
	weapon := got.Character.Equipment.Weapon
	if weapon.EnchantType == "flaming" || len(weapon.Adjectives) != 0 {
		t.Errorf("a changed slot id must not carry over the old instance's customization, got %+v", weapon)
	}
	if len(got.Character.Items) != 1 {
		t.Fatalf("expected 1 carried item, got %+v", got.Character.Items)
	}
	carried := got.Character.Items[0]
	if carried.Uses == 3 || (len(carried.Adjectives) == 1 && carried.Adjectives[0] == "rusty") {
		t.Errorf("a changed carried-item id must not carry over the old instance's customization, got %+v", carried)
	}
}
