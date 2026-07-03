package warehouse

import "testing"

func TestWithdraw_DecrementsAndCounts(t *testing.T) {
	ResetForTest()
	Deposit("The Confluence", 40123, 5)
	got := Withdraw("The Confluence", 40123, 3)
	if got != 3 {
		t.Fatalf("Withdraw = %d, want 3", got)
	}
	w := WarehouseFor("The Confluence")
	if w.StockOf(40123) != 2 || w.DrawnCount != 3 {
		t.Fatalf("stock=%d drawn=%d, want 2/3", w.StockOf(40123), w.DrawnCount)
	}
}

func TestWithdraw_FloorsAtZeroStock(t *testing.T) {
	ResetForTest()
	Deposit("The Confluence", 40123, 2)
	if got := Withdraw("The Confluence", 40123, 10); got != 2 {
		t.Fatalf("partial withdraw = %d, want 2", got)
	}
	if got := Withdraw("The Confluence", 40123, 1); got != 0 {
		t.Fatalf("empty withdraw = %d, want 0", got)
	}
	if got := Withdraw("Stillwater", 40123, 1); got != 0 {
		t.Fatalf("unknown-zone withdraw = %d, want 0", got)
	}
}
