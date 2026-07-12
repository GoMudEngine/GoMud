package mutations

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TestGetWeightedPool_MoonFilter proves the SHARED pool builder (used by the
// natural Chrysalis tick, the Awakening rite, and quest grants — not just Bloom)
// only offers the Reflect-Skin flavor matching the current moon bucket. Sweeps
// the game clock to exercise several buckets.
func TestGetWeightedPool_MoonFilter(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"rs-1": {MutationId: "rs-1", Name: "RS1", Rarity: 5, MoonFlavor: 1},
		"rs-2": {MutationId: "rs-2", Name: "RS2", Rarity: 5, MoonFlavor: 2},
		"rs-3": {MutationId: "rs-3", Name: "RS3", Rarity: 5, MoonFlavor: 3},
		"rs-4": {MutationId: "rs-4", Name: "RS4", Rarity: 5, MoonFlavor: 4},
	})
	defer cleanup()

	orig := util.GetRoundCount()
	defer util.SetRoundCountForTest(orig)

	byBucket := map[int]string{1: "rs-1", 2: "rs-2", 3: "rs-3", 4: "rs-4"}
	allIds := []string{"rs-1", "rs-2", "rs-3", "rs-4"}
	seen := map[int]bool{}

	for rc := uint64(1); rc < 20000 && len(seen) < 4; rc += 7 {
		util.SetRoundCountForTest(rc)
		b := gametime.CurrentMoonFlavorBucket()
		if seen[b] {
			continue
		}
		seen[b] = true

		inPool := map[string]bool{}
		for _, id := range GetWeightedPool(map[string]int{}, nil) {
			inPool[id] = true
		}
		for _, id := range allIds {
			if id == byBucket[b] && !inPool[id] {
				t.Fatalf("bucket %d: current-flavor %q missing from pool", b, id)
			}
			if id != byBucket[b] && inPool[id] {
				t.Fatalf("bucket %d: non-matching flavor %q leaked into pool", b, id)
			}
		}
	}
	if len(seen) < 2 {
		t.Fatalf("swept only %d distinct moon bucket(s); expected to exercise several", len(seen))
	}
	t.Logf("exercised %d distinct moon buckets", len(seen))
}
