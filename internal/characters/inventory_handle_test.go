package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// testHandleItemId is a throwaway item spec id registered for handle tests so
// that name-based matching (items.FindMatchIn -> GetSpec().Name) has something
// to resolve against. The characters test package does not load real item data
// files, so we register a minimal spec in-test.
const testHandleItemId = 990001

func registerHandleTestSpec(t *testing.T) {
	t.Helper()
	items.RegisterTestItemSpec(&items.ItemSpec{
		ItemId:     testHandleItemId,
		Name:       "test widget",
		NameSimple: "widget",
		Type:       items.Object,
	})
}

func TestFindItem_HandleResolvesExactInstance(t *testing.T) {
	registerHandleTestSpec(t)

	item1 := items.New(testHandleItemId)
	item2 := items.New(testHandleItemId)

	// Two same-named backpack items must have distinct UUIDs.
	if item1.UUID.String() == item2.UUID.String() {
		t.Fatalf("expected distinct UUIDs, both were %s", item1.UUID.String())
	}

	c := &Character{Items: []items.Item{item1, item2}}

	// Handle targets the second instance specifically.
	got, src, found := c.FindItem(ItemHandleSigil + item2.UUID.String())
	if !found {
		t.Fatalf("FindItem by handle: expected found=true")
	}
	if got.UUID.String() != item2.UUID.String() {
		t.Fatalf("FindItem by handle: got UUID %s, want item2 %s", got.UUID.String(), item2.UUID.String())
	}
	if got.UUID.String() == item1.UUID.String() {
		t.Fatalf("FindItem by handle: resolved item1, expected item2")
	}
	if src != "backpack" {
		t.Fatalf("FindItem by handle: source = %q, want \"backpack\"", src)
	}

	// FindInBackpack handle branch resolves the same instance.
	gotBp, foundBp := c.FindInBackpack(ItemHandleSigil + item2.UUID.String())
	if !foundBp || gotBp.UUID.String() != item2.UUID.String() {
		t.Fatalf("FindInBackpack by handle: found=%v uuid=%s, want item2 %s", foundBp, gotBp.UUID.String(), item2.UUID.String())
	}
}

func TestFindItem_HandleBogusReturnsNotFound(t *testing.T) {
	registerHandleTestSpec(t)

	item1 := items.New(testHandleItemId)
	c := &Character{Items: []items.Item{item1}}

	if _, _, found := c.FindItem(ItemHandleSigil + "deadbeef"); found {
		t.Fatalf("FindItem with bogus handle: expected found=false")
	}
	if _, found := c.FindInBackpack(ItemHandleSigil + "deadbeef"); found {
		t.Fatalf("FindInBackpack with bogus handle: expected found=false")
	}
}

func TestFindItem_HandleWornAndBandolierAndComponents(t *testing.T) {
	registerHandleTestSpec(t)

	worn := items.New(testHandleItemId)
	potion := items.New(testHandleItemId)
	comp := items.New(testHandleItemId)

	c := &Character{
		PotionItems:    []items.Item{potion},
		ComponentItems: []items.Item{comp},
	}
	c.Equipment.Weapon = worn

	if got, src, found := c.FindItem(ItemHandleSigil + worn.UUID.String()); !found || got.UUID.String() != worn.UUID.String() || src != "worn" {
		t.Fatalf("worn handle: found=%v src=%q uuid=%s", found, src, got.UUID.String())
	}
	if got, src, found := c.FindItem(ItemHandleSigil + potion.UUID.String()); !found || got.UUID.String() != potion.UUID.String() || src != "bandolier" {
		t.Fatalf("bandolier handle: found=%v src=%q uuid=%s", found, src, got.UUID.String())
	}
	if got, src, found := c.FindItem(ItemHandleSigil + comp.UUID.String()); !found || got.UUID.String() != comp.UUID.String() || src != "components" {
		t.Fatalf("components handle: found=%v src=%q uuid=%s", found, src, got.UUID.String())
	}

	// FindOnBody resolves the worn instance by handle.
	if got, found := c.FindOnBody(ItemHandleSigil + worn.UUID.String()); !found || got.UUID.String() != worn.UUID.String() {
		t.Fatalf("FindOnBody worn handle: found=%v uuid=%s", found, got.UUID.String())
	}
}

func TestFindItem_NonHandleStillResolvesByName(t *testing.T) {
	registerHandleTestSpec(t)

	item1 := items.New(testHandleItemId)
	c := &Character{Items: []items.Item{item1}}

	got, _, found := c.FindItem("widget")
	if !found {
		t.Fatalf("FindItem by name: expected found=true for \"widget\"")
	}
	if got.ItemId != testHandleItemId {
		t.Fatalf("FindItem by name: got ItemId %d, want %d", got.ItemId, testHandleItemId)
	}

	gotBp, foundBp := c.FindInBackpack("widget")
	if !foundBp || gotBp.ItemId != testHandleItemId {
		t.Fatalf("FindInBackpack by name: found=%v itemId=%d", foundBp, gotBp.ItemId)
	}
}

func TestIsItemHandle(t *testing.T) {
	if h, ok := isItemHandle("@abc123"); !ok || h != "abc123" {
		t.Fatalf("isItemHandle(@abc123) = %q,%v", h, ok)
	}
	if h, ok := isItemHandle("  @abc123  "); !ok || h != "abc123" {
		t.Fatalf("isItemHandle trims: %q,%v", h, ok)
	}
	if _, ok := isItemHandle("dagger"); ok {
		t.Fatalf("isItemHandle(dagger) should be false")
	}
}
