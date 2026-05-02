package sealedcrate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	tmp := t.TempDir()

	c := New(4038, 2000)
	c.Add(items.Item{ItemId: 40021})
	c.Add(items.Item{ItemId: 40028})

	if err := SaveTo(filepath.Join(tmp, "4038-fernway_shipment.yaml"), c); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	loaded, err := LoadFrom(filepath.Join(tmp, "4038-fernway_shipment.yaml"))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if loaded.RoomId() != 4038 {
		t.Errorf("loaded RoomId = %d, want 4038", loaded.RoomId())
	}
	if loaded.Capacity() != 2000 {
		t.Errorf("loaded Capacity = %d, want 2000", loaded.Capacity())
	}
	if loaded.Len() != 2 {
		t.Errorf("loaded Len = %d, want 2", loaded.Len())
	}
}

func TestLoadMissingFileReturnsError(t *testing.T) {
	tmp := t.TempDir()
	_, err := LoadFrom(filepath.Join(tmp, "does-not-exist.yaml"))
	if err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
	// Wrap chain must preserve os.ErrNotExist so callers can detect
	// "no such file" via errors.Is.
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected wrapped os.ErrNotExist, got %v", err)
	}
}

// TestSaveLoad_AfterPartialDrain pins the production lifecycle:
// Kessa drops items into a crate, the crate is saved to disk; the
// caravan later drains the crate, the empty crate is saved again.
// Loading the empty-state YAML must produce a correctly-empty crate
// rather than a stale-items copy or an unmarshal error.
func TestSaveLoad_AfterPartialDrain(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "4038-fernway_shipment.yaml")

	// Round 1: Kessa drops two items, save.
	c := New(4038, 2000)
	c.Add(items.Item{ItemId: 40046})
	c.Add(items.Item{ItemId: 40049})
	if err := SaveTo(path, c); err != nil {
		t.Fatalf("SaveTo (round 1): %v", err)
	}

	// Round 2: caravan drains, save again.
	drained := c.DrainAll()
	if len(drained) != 2 {
		t.Fatalf("DrainAll returned %d items, want 2", len(drained))
	}
	if err := SaveTo(path, c); err != nil {
		t.Fatalf("SaveTo (round 2): %v", err)
	}

	// Reload — should be empty, not stale.
	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom after drain: %v", err)
	}
	if loaded.Len() != 0 {
		t.Errorf("loaded Len after drain+save = %d, want 0", loaded.Len())
	}
	if loaded.RoomId() != 4038 {
		t.Errorf("loaded RoomId after drain = %d, want 4038", loaded.RoomId())
	}
}
