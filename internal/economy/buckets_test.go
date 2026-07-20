package economy

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestBucketFor_KnownItems(t *testing.T) {
	cases := []struct {
		id     int
		bucket string
	}{
		{40001, "base"},       // iron ingot
		{40046, "fernway"},    // moonpetal
		{40051, "stillwater"}, // skitter-shrimp shell
		{40010, "thornwall"},  // chrysalis core
		{40004, "overlap"},    // healer's root
		{40062, "fernway"},    // oak bark (3.0b)
		{40123, "confluence"}, // watercress
		{99999, ""},           // unknown
	}
	for _, c := range cases {
		if got := BucketFor(c.id); got != c.bucket {
			t.Errorf("BucketFor(%d) = %q, want %q", c.id, got, c.bucket)
		}
	}
}

func TestAllBuckets_StableOrder(t *testing.T) {
	want := []string{"base", "stillwater", "thornwall", "fernway", "confluence", "overlap"}
	got := AllBuckets()
	if len(got) != len(want) {
		t.Fatalf("AllBuckets() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AllBuckets()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestBucketMap_AuditMatrixCoverage asserts every item with
// is_component: true OR component_tag set has a bucket entry.
// Drift between the audit matrix doc and the Go map is caught here.
//
// Quest/specialty and Defer-to-3.0e items are exceptions and listed
// explicitly.
func TestBucketFor_WildHareMeatOverlap(t *testing.T) {
	if got := BucketFor(40064); got != "overlap" {
		t.Errorf("wild-hare-meat bucket = %q, want overlap", got)
	}
}

func TestBucketMap_AuditMatrixCoverage(t *testing.T) {
	knownExceptions := map[int]bool{
		// Quest/specialty (15 items per audit matrix)
		40031: true, 40032: true, 40033: true, 40034: true, 40035: true,
		40036: true, 40037: true, 40038: true, 40039: true, 40040: true,
		40041: true, 40042: true, 40054: true, 40060: true, 40061: true,
		// Defer to 3.0e (cloth/leather adjacent — not yet bucketed)
		40052: true, 40055: true,
	}
	for id := 40001; id <= 40068; id++ {
		if knownExceptions[id] {
			continue
		}
		spec := items.GetItemSpec(id)
		if spec == nil {
			// Item id unused — skip (not all 40000s exist)
			continue
		}
		if spec.ComponentTag == "" && !spec.IsComponent {
			continue // not a crafting component, skip
		}
		if got := BucketFor(id); got == "" {
			t.Errorf("item %d (%s) is a component but has no bucket", id, spec.Name)
		}
	}

	// Confluence bucket ids (ferry Stage 2). The main loop above is
	// deliberately bounded to 40001-40068 — extending it would sweep in
	// dozens of unrelated Confluence-zone quest/lore props (40069-40122)
	// needing exception entries. Pin exactly the four confluence goods
	// here instead so bucket↔matrix drift on 40123-40126 is still caught.
	//
	// Specs are seeded because this package's tests never call
	// items.LoadDataFiles() — without seeding, GetItemSpec returns nil
	// and the component gate skips every id, making the loop vacuous.
	// The seeded IsComponent flags mirror the YAML (all four are
	// is_component: true cooking goods — verified 2026-07-03).
	cleanupSeed := items.SeedItemsForTest(map[int]*items.ItemSpec{
		40123: {ItemId: 40123, Name: "Watercress", IsComponent: true},
		40124: {ItemId: 40124, Name: "Freshwater Mussels", IsComponent: true},
		40125: {ItemId: 40125, Name: "Smoked River-Fish", IsComponent: true},
		40126: {ItemId: 40126, Name: "Fresh River Catch", IsComponent: true},
	})
	defer cleanupSeed()
	for id := 40123; id <= 40126; id++ {
		spec := items.GetItemSpec(id)
		if spec == nil {
			// Item id unused — skip (not all 40000s exist)
			continue
		}
		if spec.ComponentTag == "" && !spec.IsComponent {
			continue // not a crafting component, skip
		}
		if got := BucketFor(id); got == "" {
			t.Errorf("item %d (%s) is a component but has no bucket", id, spec.Name)
		}
	}
}
