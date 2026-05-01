package shops

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// makeTemplate returns a simple ShopInventory template for tests.
func makeTemplate() ShopInventory {
	return ShopInventory{
		Gold:         200,
		StartingGold: 200,
		Stock: []StockEntry{
			{ItemId: 10, RestockQty: 5, MaxStock: 15, Current: 0},
			{ItemId: 20, RestockQty: 0, MaxStock: 8, Current: 0}, // NPC-crafted
		},
	}
}

// writeShopFile writes a ShopInventory YAML to dir/{mobid}-room{roomid}.yaml
// and returns the path.
func writeShopFile(t *testing.T, dir string, mobId int, roomId int, inv ShopInventory) string {
	t.Helper()
	filename := fmt.Sprintf("%d-room%d.yaml", mobId, roomId)
	path := filepath.Join(dir, filename)
	data, err := yaml.Marshal(&inv)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))
	return path
}

// ---- Cache-level tests (no disk I/O via the real shopPath) ----

func TestRegisterShop_NoDisk_SeedsFromTemplate(t *testing.T) {
	ClearCache()
	tmpl := makeTemplate()

	inv := RegisterShop("testzone", 42, 100, tmpl)

	require.NotNil(t, inv)
	// Location fields set
	assert.Equal(t, "testzone", inv.Zone)
	assert.Equal(t, 42, inv.MobId)
	assert.Equal(t, 100, inv.RoomId)

	// Stocked item initialised to RestockQty
	entry := inv.GetStock(10)
	require.NotNil(t, entry)
	assert.Equal(t, 5, entry.Current, "stocked item should be seeded to RestockQty")

	// NPC-crafted item stays at 0
	crafted := inv.GetStock(20)
	require.NotNil(t, crafted)
	assert.Equal(t, 0, crafted.Current, "crafted item should start at 0")
}

func TestRegisterShop_ReturnsCachedOnSecondCall(t *testing.T) {
	ClearCache()
	tmpl := makeTemplate()

	first := RegisterShop("testzone", 42, 101, tmpl)
	second := RegisterShop("testzone", 42, 101, tmpl)

	assert.Same(t, first, second, "second call should return the same pointer")
}

func TestGetShopInventory_ReturnsCached(t *testing.T) {
	ClearCache()
	tmpl := makeTemplate()

	registered := RegisterShop("testzone", 42, 102, tmpl)
	got := GetShopInventory("testzone", 42, 102)

	assert.Same(t, registered, got)
}

func TestGetShopInventory_NotFound_ReturnsNil(t *testing.T) {
	ClearCache()
	// No registration, no disk file → should return nil.
	got := GetShopInventory("nonexistent_zone", 9999, 9999)
	assert.Nil(t, got)
}

func TestClearCache(t *testing.T) {
	ClearCache()
	tmpl := makeTemplate()
	RegisterShop("testzone", 42, 103, tmpl)

	ClearCache()

	// After clear there's no disk file for this path either, so nil.
	got := GetShopInventory("testzone", 42, 103)
	assert.Nil(t, got)
}

func TestRegisterShop_TemplateNotMutated(t *testing.T) {
	ClearCache()
	tmpl := makeTemplate()

	inv := RegisterShop("testzone", 42, 104, tmpl)
	inv.GetStock(10).Current = 99

	// Original template should be unchanged.
	assert.Equal(t, 0, tmpl.Stock[0].Current)
}

// ---- YAML round-trip tests (direct file I/O, no configs dependency) ----

func TestYAML_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	original := ShopInventory{
		Gold:         500,
		StartingGold: 500,
		LastRestock:  12345,
		Stock: []StockEntry{
			{ItemId: 10, RestockQty: 5, MaxStock: 15, Current: 7},
			{ItemId: 20, RestockQty: 0, MaxStock: 8, Current: 2},
		},
		KnownRecipes: []string{"iron_sword", "shield"},
	}

	path := writeShopFile(t, dir, 42, 100, original)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var loaded ShopInventory
	require.NoError(t, yaml.Unmarshal(data, &loaded))

	assert.Equal(t, original.Gold, loaded.Gold)
	assert.Equal(t, original.StartingGold, loaded.StartingGold)
	assert.Equal(t, original.LastRestock, loaded.LastRestock)
	assert.Equal(t, original.KnownRecipes, loaded.KnownRecipes)
	require.Len(t, loaded.Stock, 2)
	assert.Equal(t, 7, loaded.Stock[0].Current)
	assert.Equal(t, 2, loaded.Stock[1].Current)
}

func TestYAML_LocationFieldsNotPersisted(t *testing.T) {
	// Zone/MobId/RoomId have yaml:"-" so they must not appear in the file.
	dir := t.TempDir()

	inv := ShopInventory{
		Gold:         100,
		StartingGold: 100,
		Zone:         "should_not_appear",
		MobId:        42,
		RoomId:       100,
		Stock:        []StockEntry{},
	}

	path := writeShopFile(t, dir, 42, 100, inv)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	assert.NotContains(t, content, "should_not_appear")
	assert.NotContains(t, content, "mob_id")
	assert.NotContains(t, content, "room_id")
	assert.NotContains(t, content, "zone")
}

func TestRegisterShop_MigratesEmptyCraftSupport(t *testing.T) {
	const (
		testZone   = "testzone"
		testMobId  = 9001
		testRoomId = 1
	)

	// Remove any stale on-disk artifact from a previous run, then clean up
	// again when the test finishes (the migration path writes to disk).
	diskPath := shopPath(testZone, testMobId, testRoomId)
	os.Remove(diskPath) //nolint:errcheck // best-effort pre-cleanup
	ClearCache()
	t.Cleanup(func() {
		ClearCache()
		os.Remove(diskPath) //nolint:errcheck // best-effort post-cleanup
	})

	// Seed a cached shop with empty CraftSupport.
	tmplNoTag := makeTemplate()
	tmplNoTag.CraftSupport = ""
	inv := RegisterShop(testZone, testMobId, testRoomId, tmplNoTag)
	if inv.CraftSupport != "" {
		t.Fatalf("setup precondition: got %q, want \"\"", inv.CraftSupport)
	}

	// Re-register with a tagged template — should migrate the cached one.
	tmplWithTag := makeTemplate()
	tmplWithTag.CraftSupport = CraftSupportBlacksmithing
	inv2 := RegisterShop(testZone, testMobId, testRoomId, tmplWithTag)
	if inv2.CraftSupport != CraftSupportBlacksmithing {
		t.Errorf("got %q after migration, want %q", inv2.CraftSupport, CraftSupportBlacksmithing)
	}
}

func TestIsValidCraftSupport(t *testing.T) {
	for _, v := range ValidCraftSupports {
		if !IsValidCraftSupport(v) {
			t.Errorf("ValidCraftSupports contains %q but IsValidCraftSupport returns false", v)
		}
	}
	if IsValidCraftSupport("nonsense") {
		t.Error("IsValidCraftSupport(\"nonsense\") = true, want false")
	}
	if IsValidCraftSupport("") {
		t.Error("IsValidCraftSupport(\"\") = true, want false (empty is INVALID)")
	}
}

func TestYAML_PartialLoad_MissingFieldsAreZero(t *testing.T) {
	// A file written without last_restock or known_recipes should unmarshal
	// with zero values for those fields.
	dir := t.TempDir()

	minimal := `gold: 100
starting_gold: 100
inventory:
  - item_id: 10
    restock_qty: 5
    max_stock: 15
    current: 3
`
	path := filepath.Join(dir, "42-room100.yaml")
	require.NoError(t, os.WriteFile(path, []byte(minimal), 0644))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var loaded ShopInventory
	require.NoError(t, yaml.Unmarshal(data, &loaded))

	assert.Equal(t, 100, loaded.Gold)
	assert.Equal(t, uint64(0), loaded.LastRestock)
	assert.Nil(t, loaded.KnownRecipes)
	require.Len(t, loaded.Stock, 1)
	assert.Equal(t, 3, loaded.Stock[0].Current)
}
