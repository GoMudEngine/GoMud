package configs

import "testing"

// setBalanceForTest replaces the module-level balance config with
// the provided instance for the duration of the calling test. The
// prior balance is automatically restored via t.Cleanup when the
// test ends. Intended for use from _test.go files in the configs
// package.
func setBalanceForTest(t *testing.T, b *Balance) {
	t.Helper()
	original := configData.Balance
	configData.Balance = *b
	t.Cleanup(func() {
		configData.Balance = original
		// Re-validate to fill any zero'd defaults from the original.
		configData.Balance.Validate()
	})
}
