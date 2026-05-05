package items

import (
	"strings"
	"testing"
)

func TestValidateVendorCategories_RejectsUntaggedSalableItem(t *testing.T) {
	specs := map[int]*ItemSpec{
		1: {ItemId: 1, Name: "x", Value: 5}, // no vendor_categories, no questtoken
	}
	err := ValidateVendorCategories(specs, []string{"alchemy", "blacksmithing"})
	if err == nil {
		t.Fatalf("expected error for untagged salable item")
	}
	if !strings.Contains(err.Error(), "1") {
		t.Errorf("error should mention itemId 1: %v", err)
	}
}

func TestValidateVendorCategories_AllowsQuestItem(t *testing.T) {
	specs := map[int]*ItemSpec{
		1: {ItemId: 1, Name: "quest token", Value: 0, QuestToken: "5-start"},
	}
	err := ValidateVendorCategories(specs, []string{"alchemy"})
	if err != nil {
		t.Errorf("quest items should be skipped: %v", err)
	}
}

func TestValidateVendorCategories_AllowsZeroValueItem(t *testing.T) {
	specs := map[int]*ItemSpec{
		1: {ItemId: 1, Name: "prop", Value: 0},
	}
	err := ValidateVendorCategories(specs, []string{"alchemy"})
	if err != nil {
		t.Errorf("zero-value items should be skipped: %v", err)
	}
}

func TestValidateVendorCategories_RejectsUnknownCategory(t *testing.T) {
	specs := map[int]*ItemSpec{
		1: {ItemId: 1, Name: "x", Value: 5, VendorCategories: []string{"madeup"}},
	}
	err := ValidateVendorCategories(specs, []string{"alchemy", "blacksmithing"})
	if err == nil {
		t.Fatalf("expected error for unknown category")
	}
	if !strings.Contains(err.Error(), "madeup") {
		t.Errorf("error should mention bad category: %v", err)
	}
}

func TestValidateVendorCategories_AcceptsTaggedItem(t *testing.T) {
	specs := map[int]*ItemSpec{
		1: {ItemId: 1, Name: "x", Value: 5, VendorCategories: []string{"alchemy"}},
	}
	err := ValidateVendorCategories(specs, []string{"alchemy", "blacksmithing"})
	if err != nil {
		t.Errorf("expected no error: %v", err)
	}
}
