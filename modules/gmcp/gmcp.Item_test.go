package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// fakeItemWorld is an in-memory stand-in for the items package.
type fakeItemWorld struct {
	specs   map[int]*items.ItemSpec
	saved   []items.ItemSpec
	deleted []int
	nextId  int
}

func newFakeItemWorld() *fakeItemWorld {
	return &fakeItemWorld{specs: map[int]*items.ItemSpec{}, nextId: 10000}
}

func (w *fakeItemWorld) deps() itemDeps {
	return itemDeps{
		load: func(id int) *items.ItemSpec { return w.specs[id] },
		save: func(s items.ItemSpec) error {
			w.saved = append(w.saved, s)
			cp := s
			w.specs[s.ItemId] = &cp
			return nil
		},
		del: func(id int) error { w.deleted = append(w.deleted, id); delete(w.specs, id); return nil },
		create: func(s items.ItemSpec) (int, error) {
			w.nextId++
			s.ItemId = w.nextId
			cp := s
			w.specs[s.ItemId] = &cp
			return s.ItemId, nil
		},
		references: func(id int) []itemRef { return nil },
		ranges:     func(string) map[string][2]float64 { return map[string][2]float64{"damageMultiplier": {0.3, 3.5}} },
	}
}

func TestBuildItemUpdate_RoundTripsFields(t *testing.T) {
	w := newFakeItemWorld()
	w.specs[10001] = &items.ItemSpec{ItemId: 10001, Name: "Old", Type: items.Weapon, Description: "d", Hands: 1}
	res := buildItemUpdate(w.deps(), itemUpdateReq{
		ItemId: 10001, Name: "Keen Blade", Type: string(items.Weapon), Description: "Sharp.",
		Value: 55, Weight: 3, DamageMultiplier: 0.9, Hands: 1, RarityTier: 30,
		VendorCategories: []string{"blacksmithing"},
		StatMods:         map[string]int{"strength": 2},
	})
	if !res.Ok {
		t.Fatalf("update should succeed, got %+v", res)
	}
	if len(w.saved) != 1 {
		t.Fatalf("expected 1 save, got %d", len(w.saved))
	}
	got := w.saved[0]
	if got.Name != "Keen Blade" || got.Description != "Sharp." || got.DamageMultiplier != 0.9 || got.RarityTier != 30 {
		t.Errorf("fields not round-tripped: %+v", got)
	}
	if got.StatMods["strength"] != 2 {
		t.Errorf("statmods not round-tripped: %+v", got.StatMods)
	}
}

func TestBuildItemUpdate_RejectsEmptyNameOrType(t *testing.T) {
	w := newFakeItemWorld()
	w.specs[10001] = &items.ItemSpec{ItemId: 10001, Name: "X", Type: items.Weapon}
	if res := buildItemUpdate(w.deps(), itemUpdateReq{ItemId: 10001, Name: "", Type: "weapon"}); res.Ok {
		t.Error("empty name must be rejected")
	}
	if res := buildItemUpdate(w.deps(), itemUpdateReq{ItemId: 10001, Name: "X", Type: ""}); res.Ok {
		t.Error("empty type must be rejected")
	}
	if len(w.saved) != 0 {
		t.Errorf("no save on validation failure, got %d", len(w.saved))
	}
}

func TestBuildItemUpdate_PreservesUncoveredFields(t *testing.T) {
	w := newFakeItemWorld()
	// Reserve/voice fields aren't in the form payload — they must survive a Save.
	w.specs[10001] = &items.ItemSpec{ItemId: 10001, Name: "Old", Type: items.Weapon, Hands: 1,
		ReserveConvictionPct: 0.25, VoiceId: "aegis"}
	res := buildItemUpdate(w.deps(), itemUpdateReq{ItemId: 10001, Name: "New", Type: "weapon", Description: "d", NotSalable: true})
	if !res.Ok {
		t.Fatal(res)
	}
	got := w.saved[0]
	if got.ReserveConvictionPct != 0.25 || got.VoiceId != "aegis" {
		t.Errorf("form-omitted fields must survive Save, got reserve=%v voice=%q", got.ReserveConvictionPct, got.VoiceId)
	}
}

// A salable item with no valid vendor category bricks the next boot
// (ValidateVendorCategories panics), so the editor must refuse to save it.
func TestBuildItemUpdate_GuardsVendorCategories(t *testing.T) {
	w := newFakeItemWorld()
	w.specs[10001] = &items.ItemSpec{ItemId: 10001, Name: "X", Type: items.Weapon}

	// Salable, no categories -> rejected, nothing saved.
	if res := buildItemUpdate(w.deps(), itemUpdateReq{ItemId: 10001, Name: "Blade", Type: "weapon", Description: "d"}); res.Ok {
		t.Error("salable item with no vendor category must be rejected")
	}
	// Salable, unknown category -> rejected.
	if res := buildItemUpdate(w.deps(), itemUpdateReq{ItemId: 10001, Name: "Blade", Type: "weapon", Description: "d", VendorCategories: []string{"wizardry"}}); res.Ok {
		t.Error("unknown vendor category must be rejected")
	}
	if len(w.saved) != 0 {
		t.Errorf("no save on validation failure, got %d", len(w.saved))
	}
	// Not-salable OR a valid category -> allowed.
	if res := buildItemUpdate(w.deps(), itemUpdateReq{ItemId: 10001, Name: "Blade", Type: "weapon", Description: "d", NotSalable: true}); !res.Ok {
		t.Errorf("not-salable item should be allowed without categories: %+v", res)
	}
	if res := buildItemUpdate(w.deps(), itemUpdateReq{ItemId: 10001, Name: "Blade", Type: "weapon", Description: "d", VendorCategories: []string{"blacksmithing"}}); !res.Ok {
		t.Errorf("valid category should be allowed: %+v", res)
	}
}

func TestBuildItemCreate_AssignsIdAndStores(t *testing.T) {
	w := newFakeItemWorld()
	res := buildItemCreate(w.deps(), string(items.Weapon))
	if !res.Ok || res.ItemId == 0 {
		t.Fatalf("create should return an id, got %+v", res)
	}
	if w.specs[res.ItemId] == nil {
		t.Fatal("created item not stored")
	}
	if bad := buildItemCreate(w.deps(), ""); bad.Ok {
		t.Error("create with no type must be refused")
	}
}

func TestBuildItemGet_MapsFieldsAndEnums(t *testing.T) {
	w := newFakeItemWorld()
	w.specs[10005] = &items.ItemSpec{ItemId: 10005, Name: "Hammer", Type: items.Weapon, Subtype: items.Bludgeoning,
		Description: "Heavy.", Value: 40, Weight: 6, DamageMultiplier: 1.1, Hands: 2, RarityTier: 20}
	d, ok := buildItemGet(w.deps(), 10005)
	if !ok {
		t.Fatal("expected found")
	}
	if d.Name != "Hammer" || d.Type != string(items.Weapon) || d.DamageMultiplier != 1.1 || d.RarityTier != 20 {
		t.Errorf("detail wrong: %+v", d.itemUpdateReq)
	}
	if len(d.Types) == 0 || len(d.Subtypes) == 0 || len(d.Stats) == 0 {
		t.Error("detail must ship enum lists for the dropdowns")
	}
	if r, ok := d.Ranges["damageMultiplier"]; !ok || r[0] != 0.3 || r[1] != 3.5 {
		t.Errorf("detail must ship observed value ranges, got %+v", d.Ranges)
	}
	if _, ok := buildItemGet(w.deps(), 99999); ok {
		t.Error("missing item should return not-found")
	}
}

func TestBuildItemDelete_BlocksWhenReferenced(t *testing.T) {
	w := newFakeItemWorld()
	w.specs[10001] = &items.ItemSpec{ItemId: 10001, Name: "Used Sword", Type: items.Weapon}
	d := w.deps()
	d.references = func(id int) []itemRef { return []itemRef{{Kind: "mob", Id: "mob 9538"}} }
	res := buildItemDelete(d, 10001)
	if res.Ok {
		t.Fatal("delete should be blocked when referenced")
	}
	if len(w.deleted) != 0 {
		t.Error("nothing should be deleted when blocked")
	}
	if len(res.Refs) != 1 || res.Refs[0].Id != "mob 9538" {
		t.Errorf("blocked result must carry the references, got %+v", res.Refs)
	}
}

func TestBuildItemDelete_DeletesWhenClean(t *testing.T) {
	w := newFakeItemWorld()
	w.specs[10002] = &items.ItemSpec{ItemId: 10002, Name: "Unused", Type: items.Weapon}
	res := buildItemDelete(w.deps(), 10002) // fake references returns nil
	if !res.Ok {
		t.Fatalf("clean delete should succeed, got %+v", res)
	}
	if len(w.deleted) != 1 || w.deleted[0] != 10002 {
		t.Errorf("expected delete of 10002, got %+v", w.deleted)
	}
}

func TestScanItemReferencesWith_FindsMatchingSources(t *testing.T) {
	refs := scanItemReferencesWith(40163, refIterators{
		mobs: func(yield func(mobRef)) {
			yield(mobRef{id: 9538, name: "a wolf", ids: []int{40163}}) // match
			yield(mobRef{id: 9999, name: "other", ids: []int{111}})    // no match
		},
		recipes: func(yield func(string, int)) {
			yield("iron-dagger", 10001) // no match
		},
		quests: func(yield func(int, []int)) {
			yield(10, []int{40163}) // match
		},
		containers: func(yield func(int, []int)) {
			yield(5901, []int{222}) // no match
		},
	})
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs (mob 9538 + quest 10), got %+v", refs)
	}
	kinds := map[string]bool{}
	for _, r := range refs {
		kinds[r.Kind] = true
	}
	if !kinds["mob"] || !kinds["quest"] {
		t.Errorf("expected mob + quest kinds, got %+v", refs)
	}
}

func TestParseItemInfoIds(t *testing.T) {
	got := parseItemInfoIds("40163:2, 10001 ,junk,:5")
	if len(got) != 2 || got[0] != 40163 || got[1] != 10001 {
		t.Errorf("parseItemInfoIds wrong: %+v", got)
	}
}
