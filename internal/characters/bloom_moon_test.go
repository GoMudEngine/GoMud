package characters

import (
	"math/rand"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// TestBloomSeedNewMutation_MoonFilter verifies the bloom pool only ever offers
// the Reflect-Skin flavor matching the current moon bucket — a non-matching
// flavor must never be granted.
func TestBloomSeedNewMutation_MoonFilter(t *testing.T) {
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"rs-1": {MutationId: "rs-1", Name: "RS1", Rarity: 5, MoonFlavor: 1},
		"rs-2": {MutationId: "rs-2", Name: "RS2", Rarity: 5, MoonFlavor: 2},
		"rs-3": {MutationId: "rs-3", Name: "RS3", Rarity: 5, MoonFlavor: 3},
		"rs-4": {MutationId: "rs-4", Name: "RS4", Rarity: 5, MoonFlavor: 4},
	})
	defer cleanup()

	byBucket := map[int]string{1: "rs-1", 2: "rs-2", 3: "rs-3", 4: "rs-4"}
	rng := rand.New(rand.NewSource(42))
	// A grant adds the mutation to the character, so use a FRESH character each
	// iteration; re-read the bucket each time (it advances with the game clock).
	for i := 0; i < 12; i++ {
		c := New()
		bucket := gametime.CurrentMoonFlavorBucket()
		id, _ := c.BloomSeedNewMutation(rng)
		if id != byBucket[bucket] {
			t.Fatalf("moon bucket %d: got %q, want %q (non-matching flavor leaked through the filter)", bucket, id, byBucket[bucket])
		}
	}
}
