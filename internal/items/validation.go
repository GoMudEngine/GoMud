package items

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// ValidateVendorCategories panics-style validator for ItemSpec.VendorCategories
// integrity. Returns a non-nil error listing every offending item if any:
//   - has Value > 0 AND empty QuestToken AND empty VendorCategories, OR
//   - carries a VendorCategories value not present in validCategories.
//
// Caller behavior on non-nil error:
//   - Cold boot: panic.
//   - /reload: log Error and continue (data files in inconsistent state).
func ValidateVendorCategories(specs map[int]*ItemSpec, validCategories []string) error {
	type fault struct {
		itemId int
		name   string
		why    string
	}
	var faults []fault

	for id, spec := range specs {
		if spec == nil {
			continue
		}
		// Skip non-salable items.
		if spec.Value <= 0 {
			continue
		}
		if spec.QuestToken != "" {
			continue
		}
		if len(spec.VendorCategories) == 0 {
			faults = append(faults, fault{id, spec.Name, "missing vendor_categories"})
			continue
		}
		for _, c := range spec.VendorCategories {
			if !slices.Contains(validCategories, c) {
				faults = append(faults, fault{id, spec.Name,
					fmt.Sprintf("unknown vendor_category %q", c)})
			}
		}
	}

	if len(faults) == 0 {
		return nil
	}

	sort.Slice(faults, func(i, j int) bool { return faults[i].itemId < faults[j].itemId })
	var b strings.Builder
	fmt.Fprintf(&b, "items with bad vendor_categories (%d):\n", len(faults))
	for _, f := range faults {
		fmt.Fprintf(&b, "  - item %d (%q): %s\n", f.itemId, f.name, f.why)
	}
	fmt.Fprintf(&b, "Valid vendor_categories: %s\n", strings.Join(validCategories, ", "))
	return fmt.Errorf("%s", b.String())
}
