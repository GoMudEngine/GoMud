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
