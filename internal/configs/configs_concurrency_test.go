package configs

import (
	"sync"
	"testing"
)

// Reproduces the data race on configData where every getter ran
// configData.Validate() (a mutation) while holding only configDataLock.RLock().
// Two goroutines hitting any getter concurrently on first use both observed
// validated==false and both mutated configData. Run with -race: against the
// old read-locked lazy-validate this races; with ensureConfigValidated()
// (write-locked, double-checked) it passes.
//
// To keep exercising the first-use window, one goroutine periodically resets
// validated back to false under the write lock, forcing the getters to
// re-validate while the others are mid-read.
func TestConfigData_ConcurrentGetterValidation(t *testing.T) {
	const goroutines = 50
	const iters = 200

	// Start from the unvalidated state so the first wave of getters takes the
	// validate path concurrently.
	configDataLock.Lock()
	configData.validated = false
	configDataLock.Unlock()

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				_ = GetServerConfig()
				_ = GetGamePlayConfig()
				_ = GetFilePathsConfig()
				_ = GetBalanceConfig()
				_ = GetConfig()

				// Periodically force the validate window to reopen.
				if g == 0 && i%20 == 0 {
					configDataLock.Lock()
					configData.validated = false
					configDataLock.Unlock()
				}
			}
		}(g)
	}
	wg.Wait()
}
