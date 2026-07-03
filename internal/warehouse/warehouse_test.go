package warehouse

import "testing"

// testItemCap mirrors the value ResetForTest pins itemCapFn to, so tests
// never depend on live config.
const testItemCap = 400

func TestCityFor_RegisteredZones(t *testing.T) {
	if _, ok := CityFor("New Plymouth Docks"); !ok {
		t.Fatal("NP Docks should be a warehouse city")
	}
	if _, ok := CityFor("The Confluence"); !ok {
		t.Fatal("The Confluence should be a warehouse city")
	}
	if _, ok := CityFor("Stillwater"); ok {
		t.Fatal("Stillwater must NOT be a warehouse city (spec: NP + Confluence)")
	}
}

func TestDeposit_AddsAndCaps(t *testing.T) {
	ResetForTest()
	// 40001 (iron ingot) is bucketed "base".
	if !Deposit("New Plymouth Docks", 40001, 3) {
		t.Fatal("deposit of a bucketed item should be accepted")
	}
	w := WarehouseFor("New Plymouth Docks")
	if w == nil || w.StockOf(40001) != 3 {
		t.Fatalf("expected 3 stocked, got %+v", w)
	}
	if w.CapturedCount != 3 {
		t.Fatalf("CapturedCount = %d, want 3", w.CapturedCount)
	}
	// Cap clamp: deposit beyond cap accepts only up to cap.
	Deposit("New Plymouth Docks", 40001, 1000)
	if got := w.StockOf(40001); got != testItemCap {
		t.Fatalf("stock = %d, want cap %d", got, testItemCap)
	}
}

func TestDeposit_RejectsUnbucketedAndUnknownZone(t *testing.T) {
	ResetForTest()
	if Deposit("New Plymouth Docks", 99999, 1) {
		t.Fatal("unbucketed item must be rejected")
	}
	if Deposit("Stillwater", 40001, 1) {
		t.Fatal("non-warehouse zone must be rejected")
	}
}
