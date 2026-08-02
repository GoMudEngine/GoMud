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

// SetConfigForTest replaces the entire package-level config with the
// provided instance for the duration of the calling test. The prior config
// is automatically restored via t.Cleanup when the test ends. Exported (unlike
// its sibling setBalanceForTest) because callers outside this package — e.g.
// internal/characters, which must chdir + configs.ReloadConfig() to read the
// real _datafiles/config.yaml — need it too. Self-registering the restore
// here (rather than leaving it to the caller) is the point: a caller that
// mutates configData and forgets its own cleanup leaks that mutation into
// every later test sharing the same test binary.
func SetConfigForTest(t *testing.T, c Config) {
	t.Helper()
	configDataLock.Lock()
	original := configData
	configData = c
	configDataLock.Unlock()
	t.Cleanup(func() {
		configDataLock.Lock()
		defer configDataLock.Unlock()
		configData = original
	})
}
