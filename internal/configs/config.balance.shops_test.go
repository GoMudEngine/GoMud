package configs

import "testing"

func TestValidateShops_StorageSeizureMinValueDefault(t *testing.T) {
	b := &Balance{}
	b.validateShops()
	if int(b.StorageSeizureMinValue) != 250 {
		t.Errorf("StorageSeizureMinValue default = %d, want 250", int(b.StorageSeizureMinValue))
	}
}

func TestValidateShops_StorageSeizureMinValuePreserved(t *testing.T) {
	b := &Balance{StorageSeizureMinValue: 1000}
	b.validateShops()
	if int(b.StorageSeizureMinValue) != 1000 {
		t.Errorf("StorageSeizureMinValue = %d, want 1000 (explicit value preserved)", int(b.StorageSeizureMinValue))
	}
}
