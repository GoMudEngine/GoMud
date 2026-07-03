package warehouse

import "testing"

func TestSaveLoad_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	SetBaseDirForTest(tmpDir)
	defer SetBaseDirForTest("")

	ResetForTest()
	Deposit("The Confluence", 40123, 5)
	if err := saveOne("The Confluence"); err != nil {
		t.Fatalf("save: %v", err)
	}
	ResetForTest()
	LoadAll()
	w := WarehouseFor("The Confluence")
	if w.StockOf(40123) != 5 || w.CapturedCount != 5 {
		t.Fatalf("round trip lost state: %+v", w)
	}
}

func TestSaveLoad_MissingFileYieldsZeroValue(t *testing.T) {
	tmpDir := t.TempDir()
	SetBaseDirForTest(tmpDir)
	defer SetBaseDirForTest("")

	ResetForTest()
	LoadAll()
	w := WarehouseFor("New Plymouth Docks")
	if w == nil || w.CapturedCount != 0 || len(w.Stock) != 0 {
		t.Fatalf("expected zero-value warehouse when no file present, got %+v", w)
	}
}
