package facts

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

func TestRegistry_SaveAndLoadRoundTrip(t *testing.T) {
	resetCaches()

	r := &Registry{
		Facts: []*Fact{
			{
				Id:           "king-dead",
				Description:  "The king is dead.",
				Significance: worldevents.Global,
				Tags:         []string{"politics", "death"},
				Status:       StatusActive,
			},
		},
	}

	if err := saveRegistry(r); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded := loadRegistryFromDisk()
	if loaded == nil || len(loaded.Facts) != 1 {
		t.Fatalf("expected 1 fact loaded, got %v", loaded)
	}
	if loaded.Facts[0].Id != "king-dead" {
		t.Errorf("fact id: got %q", loaded.Facts[0].Id)
	}
	if loaded.Facts[0].Significance != worldevents.Global {
		t.Errorf("significance mismatch: %v", loaded.Facts[0].Significance)
	}
}

func TestRegistry_LoadMissingFileReturnsNil(t *testing.T) {
	resetCaches()
	// Ensure the registry file doesn't exist
	_ = os.Remove(registryFilePath())
	if loadRegistryFromDisk() != nil {
		t.Errorf("expected nil for missing file")
	}
}
