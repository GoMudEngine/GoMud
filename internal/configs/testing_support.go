package configs

// setBalanceForTest replaces the module-level balance config with
// the provided instance. Intended for use from _test.go files in the
// configs package. Callers are responsible for saving/restoring if
// parallelism is a concern (these tests run sequentially).
func setBalanceForTest(b *Balance) {
	configData.Balance = *b
}
