package warehouse

import "testing"

func TestAccrualDue(t *testing.T) {
	// 2 game-hours at rpd 900 -> every 75 rounds.
	if !accrualDue(150, 2, 900) || accrualDue(151, 2, 900) {
		t.Fatal("cadence math wrong")
	}
}

func TestRunAccrual_SeedsAndCaps(t *testing.T) {
	ResetForTest()
	runAccrual()
	w := WarehouseFor("The Confluence")
	if w.StockOf(40123) != 1 || w.AccruedCount < 1 {
		t.Fatalf("accrual didn't seed: %+v", w)
	}
	for i := 0; i < 1000; i++ {
		runAccrual()
	}
	if got := w.StockOf(40123); got != 400 {
		t.Fatalf("cap not enforced: %d", got)
	}
}
