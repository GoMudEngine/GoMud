package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/species"
)

// `mob := *m` in newMobByIdInternal shallow-copies the template, so every map
// and slice on it is shared with the template until it is explicitly copied.
// Character.Mutations was not copied, and three spawn-time writes land in it
// (SpawnMutations, the MutationChance roll, ApplyIntrinsicMutations). The
// intrinsic path is additive — `combined := c.Mutations[id] + intrinsicRank` —
// so ranks compounded across respawns until every instance of a species with
// an intrinsic kit was pinned at MutationMaxRank for the server's uptime.

const (
	stiSpeciesId  = 7701
	stiTemplateId = 7702
	stiIntrinsic  = "sti-intrinsic"
	stiCurated    = "sti-curated"
)

// seedTemplateIsolationFixture registers a species carrying an intrinsic
// mutation, a template that also lists a curated SpawnMutation, and the two
// mutation specs both reference. Returns a combined cleanup.
func seedTemplateIsolationFixture(t *testing.T) func() {
	t.Helper()

	cleanMutations := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		stiIntrinsic: {MutationId: stiIntrinsic, Name: "Sti Intrinsic", Rarity: 3},
		stiCurated:   {MutationId: stiCurated, Name: "Sti Curated", Rarity: 3},
	})
	cleanSpecies := species.SeedSpeciesForTest(map[int]*species.Species{
		stiSpeciesId: {
			SpeciesId:          stiSpeciesId,
			Name:               "test-intrinsic-species",
			BodyParts:          []string{"arms", "hands", "legs"},
			IntrinsicMutations: map[string]int{stiIntrinsic: 1},
		},
	})

	origMobs := mobs
	origInstances := mobInstances
	origCounter := instanceCounter

	mobs = map[int]*Mob{
		stiTemplateId: {
			MobId:          stiTemplateId,
			Zone:           "Testzone",
			StatPool:       10,
			ActivityLevel:  50,
			Groups:         []string{"testers", "spawnable"},
			SpawnMutations: []string{stiCurated},
			Character: characters.Character{
				Name:      "Isolation Dummy",
				SpeciesId: stiSpeciesId,
				Shop: characters.Shop{
					{ItemId: 100, Price: 50, Quantity: 5, QuantityMax: 5},
				},
			},
		},
	}
	mobInstances = map[int]*Mob{}
	instanceCounter = 8000

	// fileloader calls Validate() on every template at load; mirror that so
	// the fixture template has the same map-allocation state as a real one
	// (in particular a non-nil Mutations map, which is what made the bug
	// reachable).
	_ = mobs[stiTemplateId].Character.Validate()

	return func() {
		mobs = origMobs
		mobInstances = origInstances
		instanceCounter = origCounter
		cleanSpecies()
		cleanMutations()
	}
}

func TestSpawn_MutationsDoNotCompoundAcrossSpawns(t *testing.T) {
	cleanup := seedTemplateIsolationFixture(t)
	defer cleanup()

	template := mobs[stiTemplateId]

	first := NewMobByIdFresh(stiTemplateId, 1)
	if first == nil {
		t.Fatal("NewMobByIdFresh returned nil for the first spawn")
	}
	defer DestroyInstance(first.InstanceId)

	second := NewMobByIdFresh(stiTemplateId, 1)
	if second == nil {
		t.Fatal("NewMobByIdFresh returned nil for the second spawn")
	}
	defer DestroyInstance(second.InstanceId)

	third := NewMobByIdFresh(stiTemplateId, 1)
	if third == nil {
		t.Fatal("NewMobByIdFresh returned nil for the third spawn")
	}
	defer DestroyInstance(third.InstanceId)

	if got := first.Character.Mutations[stiIntrinsic]; got != 1 {
		t.Fatalf("precondition: first spawn intrinsic rank = %d, want 1", got)
	}
	if got := second.Character.Mutations[stiIntrinsic]; got != 1 {
		t.Errorf("second spawn intrinsic rank = %d, want 1 (rank compounded via the shared template map)", got)
	}
	if got := third.Character.Mutations[stiIntrinsic]; got != 1 {
		t.Errorf("third spawn intrinsic rank = %d, want 1 (rank compounded via the shared template map)", got)
	}

	// The curated SpawnMutations write is idempotent (always rank 1), so it
	// could never compound — but it must still not land in the template.
	if got := first.Character.Mutations[stiCurated]; got != 1 {
		t.Errorf("first spawn curated mutation rank = %d, want 1", got)
	}

	// The template itself must be untouched by any of the three spawns.
	if len(template.Character.Mutations) != 0 {
		t.Errorf("template Mutations polluted by spawning: %v, want empty", template.Character.Mutations)
	}
}

// Each instance must own its Shop backing array: ShopItem elements are mutated
// in place by Destock/Restock/StockItem, so a shared array meant two live
// merchants shared one stock counter and the template accumulated depletion.
func TestSpawn_ShopStockIsPerInstance(t *testing.T) {
	cleanup := seedTemplateIsolationFixture(t)
	defer cleanup()

	template := mobs[stiTemplateId]

	first := NewMobByIdFresh(stiTemplateId, 1)
	second := NewMobByIdFresh(stiTemplateId, 1)
	if first == nil || second == nil {
		t.Fatal("NewMobByIdFresh returned nil")
	}
	defer DestroyInstance(first.InstanceId)
	defer DestroyInstance(second.InstanceId)

	if !first.Character.Shop.Destock(characters.ShopItem{ItemId: 100}) {
		t.Fatal("Destock on the first merchant failed")
	}

	if got := first.Character.Shop[0].Quantity; got != 4 {
		t.Errorf("first merchant quantity = %d, want 4", got)
	}
	if got := second.Character.Shop[0].Quantity; got != 5 {
		t.Errorf("second merchant quantity = %d, want 5 (stock leaked between instances)", got)
	}
	if got := template.Character.Shop[0].Quantity; got != 5 {
		t.Errorf("template quantity = %d, want 5 (a sale depleted the template)", got)
	}
}

// Groups is appended to per-instance (bountyhunter tags a hunter with its
// issuer faction). Appending must never reach the template's backing array.
func TestSpawn_GroupsAppendIsPerInstance(t *testing.T) {
	cleanup := seedTemplateIsolationFixture(t)
	defer cleanup()

	template := mobs[stiTemplateId]
	// Give the template slice spare capacity — the exact shape that makes a
	// shared-backing-array append visible to a sibling instance.
	withSlack := make([]string, 2, 8)
	copy(withSlack, template.Groups)
	template.Groups = withSlack

	first := NewMobByIdFresh(stiTemplateId, 1)
	second := NewMobByIdFresh(stiTemplateId, 1)
	if first == nil || second == nil {
		t.Fatal("NewMobByIdFresh returned nil")
	}
	defer DestroyInstance(first.InstanceId)
	defer DestroyInstance(second.InstanceId)

	first.Groups = append(first.Groups, "faction-a")
	second.Groups = append(second.Groups, "faction-b")

	if got := first.Groups[len(first.Groups)-1]; got != "faction-a" {
		t.Errorf("first mob's faction tag = %q, want faction-a (overwritten by the sibling)", got)
	}
	if len(template.Groups) != 2 {
		t.Errorf("template Groups length = %d, want 2", len(template.Groups))
	}
}
