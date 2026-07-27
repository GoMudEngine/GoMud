package gmcp

import (
	"fmt"
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
	// zoneExistsCalls records every zone string zoneExists was consulted
	// with, so a test can assert an empty zone is accepted WITHOUT ever
	// reaching the zoneExists call (buildMobCreate/buildMobUpdate must
	// short-circuit on "" before consulting it).
	zoneExistsCalls []string
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
		zoneExists: func(z string) bool { w.zoneExistsCalls = append(w.zoneExistsCalls, z); return z == "Testzone" },
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

// TestBuildMobCreate_EmptyZoneAllowedWithoutConsultingZoneExists covers the
// browser-eyeball fix round's zoneless-mob feature: a summon-only / not-yet-
// placed template can be created with no zone at all. The empty string must
// be accepted WITHOUT ever reaching zoneExists (only a non-empty, unknown
// zone like "Nowhere" above is rejected).
func TestBuildMobCreate_EmptyZoneAllowedWithoutConsultingZoneExists(t *testing.T) {
	w := newFakeMobWorld()
	res := buildMobCreate(w.deps(), "")
	if !res.Ok || res.MobId == 0 {
		t.Fatalf("create with an empty zone should succeed, got %+v", res)
	}
	for _, z := range w.zoneExistsCalls {
		if z == "" {
			t.Error("zoneExists must not be consulted for an empty zone")
		}
	}
}

// TestBuildMobUpdate_AllowsEmptyZoneWithoutConsultingZoneExists is the Update
// counterpart: clearing an existing mob's zone (or saving a mob that already
// has none) must succeed and must not consult zoneExists for "".
func TestBuildMobUpdate_AllowsEmptyZoneWithoutConsultingZoneExists(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90018, Zone: "Testzone"}
	base.Character.Name = "Old Name"
	w.specs[90018] = base

	req := mobUpdateReq{MobId: 90018, Zone: "", Name: "Old Name", Description: "d", SpeciesId: 1}
	res := buildMobUpdate(w.deps(), req)
	if !res.Ok {
		t.Fatalf("update clearing the zone should succeed, got %+v", res)
	}
	if w.saved[0].Zone != "" {
		t.Errorf("expected zone cleared, got %q", w.saved[0].Zone)
	}
	for _, z := range w.zoneExistsCalls {
		if z == "" {
			t.Error("zoneExists must not be consulted for an empty zone")
		}
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

// ---- Build.Mob.Delete + reference scan (Task 3) ----

func TestBuildMobDelete_BlocksWhenReferenced(t *testing.T) {
	w := newFakeMobWorld()
	m := &mobs.Mob{MobId: 90010, Zone: "Testzone"}
	m.Character.Name = "Referenced Mob"
	w.specs[90010] = m
	d := w.deps()
	d.references = func(id int) []mobRefEntry {
		return []mobRefEntry{{Kind: "room-spawn", Id: "room 101"}}
	}
	res := buildMobDelete(d, 90010)
	if res.Ok {
		t.Fatal("delete should be blocked when referenced")
	}
	if len(w.deleted) != 0 {
		t.Error("nothing should be deleted when blocked")
	}
	if len(res.MobRefs) != 1 || res.MobRefs[0].Id != "room 101" {
		t.Errorf("blocked result must carry references, got %+v", res.MobRefs)
	}
}

func TestBuildMobDelete_DeletesWhenClean(t *testing.T) {
	w := newFakeMobWorld()
	m := &mobs.Mob{MobId: 90011, Zone: "Testzone"}
	m.Character.Name = "Unused Mob"
	w.specs[90011] = m
	res := buildMobDelete(w.deps(), 90011)
	if !res.Ok {
		t.Fatalf("clean delete should succeed, got %+v", res)
	}
	if len(w.deleted) != 1 || w.deleted[0] != 90011 {
		t.Errorf("expected delete of 90011, got %+v", w.deleted)
	}
}

func TestScanMobReferences_FindsSpawnAndRelationship(t *testing.T) {
	refs := scanMobReferencesWith(9538, mobRefIterators{
		roomSpawns: func(yield func(roomId int, mobIds []int)) {
			yield(101, []int{9538})
			yield(102, []int{1})
		},
		mobRelationships: func(yield func(fromMobId int, name string, toIds []int)) {
			yield(9600, "Gossip Gert", []int{9538})
		},
	})
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %+v", refs)
	}
	if refs[0].Kind != "room-spawn" || refs[1].Kind != "relationship" {
		t.Errorf("wrong kinds: %+v", refs)
	}
}

// TestScanMobReferences_FindsQuestRef covers the quest iterator: verified via
// codegraph that internal/quests.Quest carries no mob-id fields at all (it's
// the legacy simple-reward model), but the real quest CONTENT authored in
// internal/questengine's expanded QuestDef DOES carry structured mob-id
// references (TriggerDef.Mob, NpcSay.Mob/.Lines[].Speaker, SpawnMob.Id,
// DeclareBounty.Target.Id when Target.Type=="mob") — scanMobReferences (the
// real wiring) collects those per quest. This test exercises the injected
// iterator only (not real quest data), matching the pure-function contract of
// scanMobReferencesWith.
func TestScanMobReferences_FindsQuestRef(t *testing.T) {
	refs := scanMobReferencesWith(9538, mobRefIterators{
		quests: func(yield func(questToken string, mobIds []int)) {
			yield("76 (Disc Questline)", []int{9538})
			yield("22 (Unrelated Quest)", []int{1})
		},
	})
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %+v", refs)
	}
	if refs[0].Kind != "quest" || refs[0].Id != "quest 76 (Disc Questline)" {
		t.Errorf("wrong quest ref: %+v", refs[0])
	}
}

// TestScanMobReferences_DedupsAndCoversMercAndConversation exercises the two
// mobRefIterators kinds not covered by the other scan tests (merc-shop,
// conversation), and confirms a source whose yielded id list contains the
// target mob id TWICE still produces exactly one ref — containsInt is a
// membership check, not a counter, so a source can't double-report.
func TestScanMobReferences_DedupsAndCoversMercAndConversation(t *testing.T) {
	refs := scanMobReferencesWith(9538, mobRefIterators{
		mercShops: func(yield func(sellerMobId int, name string, mobIds []int)) {
			yield(9601, "Traveling Slaver", []int{9538, 9538}) // duplicate id from one source
		},
		convPairs: func(yield func(pairFile string, mobIds []int)) {
			yield("conversations/pairs/9538_9600.yaml", []int{9538, 9600})
		},
	})
	if len(refs) != 2 {
		t.Fatalf("expected exactly 2 refs (one merc-shop, one conversation; the duplicate id must not double-count), got %+v", refs)
	}
	// scanMobReferencesWith emits in iterator order: convPairs before mercShops.
	if refs[0].Kind != "conversation" || refs[0].Id != "conversations/pairs/9538_9600.yaml" {
		t.Errorf("wrong conversation ref: %+v", refs[0])
	}
	if refs[1].Kind != "merc-shop" || refs[1].Id != "mob 9601 (Traveling Slaver) sells it" {
		t.Errorf("wrong merc-shop ref: %+v", refs[1])
	}
}

// ---- Build.Mob.Spawn (Task 4) ----

func TestBuildMobSpawn_ValidatesAndSpawns(t *testing.T) {
	w := newFakeMobWorld()
	m := &mobs.Mob{MobId: 90020, Zone: "Testzone"}
	m.Character.Name = "Spawnling"
	w.specs[90020] = m
	d := w.deps()
	spawned := []int{}
	d.spawn = func(mobId, roomId int) (string, error) {
		spawned = append(spawned, roomId)
		return "Spawnling", nil
	}
	res := buildMobSpawn(d, mobSpawnReq{MobId: 90020, RoomId: 101})
	if !res.Ok || res.MobId != 90020 {
		t.Fatalf("spawn should succeed, got %+v", res)
	}
	if res.Message != "Spawnling spawned in room 101" {
		t.Errorf("expected a descriptive success message, got %q", res.Message)
	}
	if len(spawned) != 1 || spawned[0] != 101 {
		t.Errorf("expected spawn in room 101, got %+v", spawned)
	}
	if res := buildMobSpawn(d, mobSpawnReq{MobId: 424242, RoomId: 101}); res.Ok {
		t.Error("unknown mob must be rejected")
	}
	if len(spawned) != 1 {
		t.Error("d.spawn must not be called for an unknown mob")
	}
}

// TestBuildMobSpawn_SurfacesSpawnError covers the other half of buildMobSpawn:
// a KNOWN mob whose d.spawn call itself fails (bad room id, instantiation
// failure, etc.) must come back as a non-Ok result naming the problem, not a
// silent success.
func TestBuildMobSpawn_SurfacesSpawnError(t *testing.T) {
	w := newFakeMobWorld()
	m := &mobs.Mob{MobId: 90021, Zone: "Testzone"}
	m.Character.Name = "Unspawnable"
	w.specs[90021] = m
	d := w.deps()
	d.spawn = func(mobId, roomId int) (string, error) {
		return "", fmt.Errorf("room %d not found", roomId)
	}
	res := buildMobSpawn(d, mobSpawnReq{MobId: 90021, RoomId: 999999})
	if res.Ok {
		t.Fatal("a spawn error from d.spawn must not report success")
	}
	if res.Error == "" {
		t.Error("expected a non-empty error message")
	}
}

// A crafter mob's CrafterSkill is compared verbatim against recipe.Skill in
// mobs.pickEligibleRecipe and handed to shops.EvaluateBuyRules on every sell.
// A typo there fails SILENTLY: the mob simply never crafts and its buy rules
// stop matching, with no panic and no log line. The editor must reject an
// unknown skill the same way it already rejects an unknown craft_support.
func TestBuildMobUpdate_GuardsCrafterSkill(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90030, Zone: "Testzone"}
	base.Character.Name = "Smith"
	w.specs[90030] = base

	req := func(skill string) mobUpdateReq {
		return mobUpdateReq{MobId: 90030, Zone: "Testzone", Name: "Smith",
			Description: "d", SpeciesId: 1, Crafter: true, CrafterSkill: skill}
	}

	if res := buildMobUpdate(w.deps(), req("blacksmithng")); res.Ok {
		t.Error("typo'd crafter skill must be rejected")
	}
	// "general" is a shop craft_support but never a recipe skill, so a
	// "general" crafter would match no recipe at all.
	if res := buildMobUpdate(w.deps(), req("general")); res.Ok {
		t.Error(`"general" is not a recipe skill and must be rejected for a crafter`)
	}
	if len(w.saved) != 0 {
		t.Errorf("no save on validation failure, got %d", len(w.saved))
	}
	if res := buildMobUpdate(w.deps(), req("blacksmithing")); !res.Ok {
		t.Errorf("valid crafter skill should be allowed: %+v", res)
	}
	// A non-crafter with no skill set stays legal.
	if res := buildMobUpdate(w.deps(), mobUpdateReq{MobId: 90030, Zone: "Testzone",
		Name: "Smith", Description: "d", SpeciesId: 1}); !res.Ok {
		t.Errorf("empty crafter skill should be allowed: %+v", res)
	}
}

// The mob load path interns every template description into a shared cache and
// replaces Character.Description with an `h:<hash>` token
// (characters.CacheDescription, called for every template at boot). The editor
// must never show that token to an admin — and must never let one be written
// back into a template YAML, where the prose would be unrecoverable after the
// next boot re-interned the token string itself.
func TestBuildMobGet_ResolvesInternedDescription(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90050, Zone: "Testzone"}
	base.Character.Name = "Ferryman"
	base.Character.Description = "A weathered ferryman poling the shallows."
	base.Character.CacheDescription()
	if base.Character.Description == "A weathered ferryman poling the shallows." {
		t.Fatal("precondition: CacheDescription should have replaced the prose with a token")
	}
	w.specs[90050] = base

	detail, ok := buildMobGet(w.deps(), 90050)
	if !ok {
		t.Fatal("get should succeed")
	}
	if detail.Description != "A weathered ferryman poling the shallows." {
		t.Errorf("editor must receive the prose, not the interned token; got %q", detail.Description)
	}
}

func TestBuildMobUpdate_ResolvesOrRefusesHashTokenDescription(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90051, Zone: "Testzone"}
	base.Character.Name = "Ferryman"
	base.Character.Description = "A weathered ferryman poling the shallows."
	base.Character.CacheDescription()
	token := base.Character.Description
	w.specs[90051] = base

	// A stale client echoing the interned token back unedited: resolve it.
	req := mobUpdateReq{MobId: 90051, Zone: "Testzone", Name: "Ferryman", Description: token}
	res := buildMobUpdate(w.deps(), req)
	if !res.Ok {
		t.Fatalf("resolvable token should save, got %+v", res)
	}
	if got := w.saved[len(w.saved)-1].Character.Description; got != "A weathered ferryman poling the shallows." {
		t.Errorf("saved template must hold the prose, got %q", got)
	}

	// An unresolvable token must be refused, never persisted.
	req.Description = "h:0000000000000000000000000000000000000000000000000000000000000000"
	savedBefore := len(w.saved)
	if res := buildMobUpdate(w.deps(), req); res.Ok {
		t.Fatal("unresolvable token must be refused")
	}
	if len(w.saved) != savedBefore {
		t.Fatal("refused save must not persist anything")
	}
}
