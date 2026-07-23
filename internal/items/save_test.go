package items

import (
	"os"
	"path/filepath"
	"testing"
)

// pointItemsAt redirects item file I/O at a temp dir for the test's duration
// (no config-override side effects).
func pointItemsAt(t *testing.T, dir string) {
	t.Helper()
	prev := itemsBasePath
	itemsBasePath = func() string { return dir }
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { itemsBasePath = prev })
}

func TestSaveItemSpec_RelocatesFileOnRename(t *testing.T) {
	dir := t.TempDir()
	pointItemsAt(t, dir)

	spec := ItemSpec{ItemId: 10001, Name: "Iron Sword", Type: Weapon, Description: "A sword.", Hands: 1, DamageMultiplier: 0.8}
	items[spec.ItemId] = &spec
	t.Cleanup(func() { delete(items, 10001) })

	if err := SaveItemSpec(spec); err != nil {
		t.Fatalf("initial save: %v", err)
	}
	oldPath := filepath.Join(dir, filepath.FromSlash(spec.Filepath()))
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("expected %s to exist: %v", oldPath, err)
	}

	// Rename -> filename changes -> old file removed, new written.
	renamed := spec
	renamed.Name = "Steel Sword"
	if err := SaveItemSpec(renamed); err != nil {
		t.Fatalf("rename save: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old file %s should be gone after rename", oldPath)
	}
	newPath := filepath.Join(dir, filepath.FromSlash(renamed.Filepath()))
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("new file %s should exist: %v", newPath, err)
	}
	if items[10001].Name != "Steel Sword" {
		t.Errorf("cache not updated: %q", items[10001].Name)
	}
}

// A non-canonical name must be normalized on save, or LoadDataFiles() panics
// on the next boot (AssertCanonical). This is the bug that bricked the server
// after the web editor created a "new item" and saved a lowercase rename.
func TestSaveItemSpec_CanonicalizesName(t *testing.T) {
	dir := t.TempDir()
	pointItemsAt(t, dir)

	spec := ItemSpec{ItemId: 10003, Name: "iron war hammer", Type: Weapon,
		DisplayName: "runed iron war hammer", Description: "A hammer.", Hands: 2, DamageMultiplier: 1.1}
	items[spec.ItemId] = &spec
	t.Cleanup(func() { delete(items, 10003) })

	if err := SaveItemSpec(spec); err != nil {
		t.Fatalf("save: %v", err)
	}
	got := items[10003]
	if got.Name != "Iron War Hammer" {
		t.Errorf("name not canonicalized: %q", got.Name)
	}
	if got.DisplayName != "Runed Iron War Hammer" {
		t.Errorf("display name not canonicalized: %q", got.DisplayName)
	}
	// A color-tagged display name must round-trip unchanged (Title() would
	// mangle it), so re-saving never corrupts a tagged name.
	tagged := ItemSpec{ItemId: 10004, Name: "Ember Blade", DisplayName: `<ansi fg="red">ember blade</ansi>`, Type: Weapon, Description: "d", Hands: 1, DamageMultiplier: 0.9}
	items[tagged.ItemId] = &tagged
	t.Cleanup(func() { delete(items, 10004) })
	if err := SaveItemSpec(tagged); err != nil {
		t.Fatalf("tagged save: %v", err)
	}
	if items[10004].DisplayName != `<ansi fg="red">ember blade</ansi>` {
		t.Errorf("color-tagged display name should be left untouched, got %q", items[10004].DisplayName)
	}
}

func TestCanonicalizeItemNames_PreservesMinorWords(t *testing.T) {
	s := ItemSpec{Name: "ring of the elephant"}
	CanonicalizeItemNames(&s)
	if s.Name != "Ring of the Elephant" {
		t.Errorf("smart title case wrong: %q", s.Name)
	}
}

// SaveItemSpec calls Validate(), which rejects an invalid proc — so a bad proc
// from the web editor returns an error (red toast) instead of persisting. If
// this ever fails, SaveItemSpec has stopped validating.
func TestSaveItemSpec_RejectsInvalidProc(t *testing.T) {
	dir := t.TempDir()
	pointItemsAt(t, dir)
	spec := ItemSpec{ItemId: 10009, Name: "Bad Blade", Type: Weapon, Description: "d", Hands: 1,
		NotSalable: true, DamageMultiplier: 1.0,
		Procs: []ItemProc{{Trigger: "on_wobble", Effect: "lifesteal", Chance: 50}}, // bad trigger
	}
	items[spec.ItemId] = &spec
	t.Cleanup(func() { delete(items, 10009) })
	if err := SaveItemSpec(spec); err == nil {
		t.Fatal("expected SaveItemSpec to reject an invalid proc trigger")
	}
}

func TestDeleteItemSpec_RemovesFileAndCache(t *testing.T) {
	dir := t.TempDir()
	pointItemsAt(t, dir)

	spec := ItemSpec{ItemId: 10002, Name: "Bronze Dagger", Type: Weapon, Description: "A dagger.", Hands: 1, DamageMultiplier: 0.5}
	items[spec.ItemId] = &spec
	if err := SaveItemSpec(spec); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, filepath.FromSlash(spec.Filepath()))
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	if err := DeleteItemSpec(10002); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("file %s should be gone", p)
	}
	if _, ok := items[10002]; ok {
		t.Error("cache entry should be gone")
	}
}
