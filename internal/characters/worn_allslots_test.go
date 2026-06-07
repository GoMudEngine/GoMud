package characters

import (
	"reflect"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// TestWornAllSlotsCoversEveryItemField is an anti-drift guard: it fails if an
// items.Item field is added to Worn without being registered in AllSlots().
// It enumerates Worn's items.Item fields by ADDRESS (so it is immune to label
// or field-name typos) and asserts AllSlots() covers exactly that set — no
// missing slot, no duplicate. AllSlots() is the single source of truth that
// GetAllWornItems, validateEquipmentItems, FindOnBody, and the GMCP Worn
// builder all iterate; a new slot left out of AllSlots() would silently keep a
// nil UUID and collide for @handle targeting (the bug class this guards).
func TestWornAllSlotsCoversEveryItemField(t *testing.T) {
	w := &Worn{}
	itemType := reflect.TypeOf(items.Item{})
	v := reflect.ValueOf(w).Elem()

	// want: address -> struct field name, for every items.Item field on Worn.
	want := map[uintptr]string{}
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Type() == itemType {
			want[v.Field(i).Addr().Pointer()] = v.Type().Field(i).Name
		}
	}

	// got: addresses AllSlots() reports, checking for duplicates as we go.
	got := map[uintptr]bool{}
	for _, s := range w.AllSlots() {
		ptr := reflect.ValueOf(s.Item).Pointer()
		if got[ptr] {
			t.Errorf("AllSlots() lists slot %q (key %q) more than once", s.Label, s.Key)
		}
		got[ptr] = true
		if _, ok := want[ptr]; !ok {
			t.Errorf("AllSlots() entry %q (key %q) does not point at any items.Item field on Worn", s.Label, s.Key)
		}
	}

	// Every items.Item field on Worn must be covered by AllSlots().
	for ptr, fieldName := range want {
		if !got[ptr] {
			t.Errorf("Worn field %q is an items.Item but is NOT registered in AllSlots() — add it (and it will then flow through GetAllWornItems / validation / FindOnBody / GMCP Worn)", fieldName)
		}
	}

	if len(got) != len(want) {
		t.Errorf("AllSlots() covers %d distinct item fields, Worn has %d items.Item fields", len(got), len(want))
	}
}
