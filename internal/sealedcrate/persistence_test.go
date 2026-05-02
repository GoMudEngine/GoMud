package sealedcrate

import (
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
		t.Errorf("expected error for missing file, got nil")
	}
	_ = os.Stat // keep import; harmless
}
