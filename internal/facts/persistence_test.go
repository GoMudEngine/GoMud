package facts

import (
	"os"
	"sync"
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

func TestAwareness_SaveAndLoadRoundTrip(t *testing.T) {
	resetCaches()

	a := &Awareness{
		ObserverMobId:    114,
		ObserverName:     "old fen",
		HeardEvents:      []uint64{42, 56},
		KnownFacts:       []FactKnowledge{{FactId: "king-dead", Source: SourceWitnessed, LearnedRound: 100}},
		LastUpdatedRound: 100,
	}

	if err := saveAwareness(a); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded := loadAwarenessFromDisk(114, "old fen")
	if loaded == nil {
		t.Fatalf("expected non-nil load")
	}
	if loaded.ObserverMobId != 114 {
		t.Errorf("observer mob id mismatch: %d", loaded.ObserverMobId)
	}
	if len(loaded.HeardEvents) != 2 {
		t.Errorf("heard_events count: %d", len(loaded.HeardEvents))
	}
	if len(loaded.KnownFacts) != 1 || loaded.KnownFacts[0].FactId != "king-dead" {
		t.Errorf("known_facts: %v", loaded.KnownFacts)
	}
}

func TestAwareness_LoadOrLazyInitConcurrent(t *testing.T) {
	resetCaches()
	const N = 50
	var wg sync.WaitGroup
	var seen [N]*Awareness
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			seen[i] = loadOrLazyInitAwareness(123, "city beggar")
		}(i)
	}
	wg.Wait()

	first := seen[0]
	if first == nil {
		t.Fatalf("nil result")
	}
	for i := 1; i < N; i++ {
		if seen[i] != first {
			t.Errorf("goroutine %d got different pointer", i)
		}
	}
}
