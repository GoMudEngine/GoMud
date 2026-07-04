package health_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/economy/health"
	"github.com/GoMudEngine/GoMud/internal/warehouse"
)

// TestCaptureSnapshot_Warehouses verifies that CaptureSnapshot surfaces a
// WarehouseSnapshot row per registered warehouse city (Stage 3), with the
// per-item stock rows carrying their economy bucket alongside the raw
// item id/quantity, and the CapturedCount counter reflecting deposits
// made via warehouse.Deposit.
func TestCaptureSnapshot_Warehouses(t *testing.T) {
	warehouse.ResetForTest()
	t.Cleanup(warehouse.ResetForTest)

	warehouse.Deposit("The Confluence", 40123, 7)

	snap := health.CaptureSnapshot()

	var found *health.WarehouseSnapshot
	for i := range snap.Warehouses {
		if snap.Warehouses[i].Zone == "The Confluence" {
			found = &snap.Warehouses[i]
		}
	}
	if found == nil {
		t.Fatalf("no Confluence warehouse row in snapshot: %+v", snap.Warehouses)
	}
	if found.CapturedCount != 7 {
		t.Fatalf("CapturedCount = %d, want 7", found.CapturedCount)
	}

	var stock *health.WarehouseStockSnapshot
	for i := range found.Stock {
		if found.Stock[i].ItemId == 40123 {
			stock = &found.Stock[i]
		}
	}
	if stock == nil || stock.Current != 7 || stock.Bucket != "confluence" {
		t.Fatalf("bad stock row: %+v", stock)
	}
	if stock.Cap != int(configs.GetBalanceConfig().WarehouseItemCap) {
		t.Fatalf("Cap = %d, want live WarehouseItemCap %d", stock.Cap, configs.GetBalanceConfig().WarehouseItemCap)
	}
	if stock.ItemName == "" {
		t.Fatalf("ItemName is empty, want a resolved name or item-id fallback: %+v", stock)
	}
}
