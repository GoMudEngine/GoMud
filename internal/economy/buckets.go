// Package economy holds shared classification data for the supply
// pipeline. The item-bucket map is the Go-side mirror of
// docs/economy/mat-audit-matrix.md (Stage 3.0b classification).
//
// Used by:
//   - internal/shops.RestockBuckets (gates which slots a forager or
//     caravan delivery refills)
//   - internal/forager (each forager's bucket list)
//   - internal/caravan (caravan_load tracking)
//
// Keep in sync with docs/economy/mat-audit-matrix.md. Drift is caught
// by TestBucketMap_AuditMatrixCoverage in buckets_test.go.
package economy

// itemBucket maps item IDs to their supply bucket. Items not in the
// map (quest/specialty items, defer-to-3.0e cloth/leather) return ""
// from BucketFor.
var itemBucket = map[int]string{
	// Base bucket — universal feedstock (13 items)
	40001: "base", // iron ingot
	40003: "base", // wooden plank
	40006: "base", // glass vial
	40012: "base", // thread spool
	40013: "base", // bone needle
	40014: "base", // raw meat
	40015: "base", // wild vegetables
	40016: "base", // water flask
	40017: "base", // salt pouch
	40019: "base", // chain link
	40043: "base", // clay flask
	40044: "base", // sealed phial
	40045: "base", // crystalline decanter

	// Stillwater bucket (6 items)
	40051: "stillwater", // skitter-shrimp shell
	40053: "stillwater", // Stillwater black pearl
	40056: "stillwater", // marsh willow bark
	40057: "stillwater", // lake mint
	40058: "stillwater", // freshwater clam
	40059: "stillwater", // lake-iron nodule

	// Thornwall bucket (13 items) — in-shop crafted, not foraged
	40010: "thornwall", // Chrysalis Core
	40011: "thornwall", // Hive Fragment
	40018: "thornwall", // steel ingot
	40021: "thornwall", // copper wire
	40022: "thornwall", // silver wire
	40023: "thornwall", // gold wire
	40024: "thornwall", // polished stone
	40025: "thornwall", // raw gem
	40026: "thornwall", // gem dust
	40027: "thornwall", // chrysalis shard
	40028: "thornwall", // binding paste
	40029: "thornwall", // mutation catalyst
	40030: "thornwall", // chrysalis setting

	// Fernway bucket (8 items)
	40046: "fernway", // moonpetal
	40049: "fernway", // ironbark shaving
	40062: "fernway", // oak bark (3.0b)
	40063: "fernway", // shadowcap mushroom (3.0b)
	40064: "overlap", // wild hare meat — forager-deliverable to cooks (cooking chunk)
	40065: "fernway", // beeswax (3.0b)
	40066: "fernway", // blood-moss (3.0b)
	40067: "fernway", // pine pitch (3.0b)

	// Mid-tier overlap (11 items)
	40002: "overlap", // leather strip
	40004: "overlap", // healer's root
	40005: "overlap", // bitter thistle
	40007: "overlap", // cloth strip
	40008: "overlap", // spore sac
	40009: "overlap", // dustwalk herb
	40020: "overlap", // coal dust
	40047: "overlap", // veilbloom petal
	40048: "overlap", // serpent venom sac
	40050: "overlap", // putrid residue
	40068: "overlap", // sinew (3.0e)
}

// BucketFor returns the supply bucket for an item ID, or "" if the
// item is unbucketed (quest/specialty, deferred, or unknown).
func BucketFor(itemId int) string {
	return itemBucket[itemId]
}

// AllBuckets returns the canonical list of supply buckets used in
// the bucket-aware restock system. Stable order for callers that
// need determinism (tests, CLI output).
func AllBuckets() []string {
	return []string{"base", "stillwater", "thornwall", "fernway", "overlap"}
}
