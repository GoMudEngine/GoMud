package templates

import (
	"sync"
	"testing"
)

// Reproduces the concurrent-map-write crash on templateConfigCache.
// Run with -race. Against a plain map this panics/races; with sync.Map it passes.
func TestTemplateConfigCache_ConcurrentAccess(t *testing.T) {
	const goroutines = 50
	const iters = 200
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				// userId 0 is the hot collision case (connect-splash before login)
				_ = getTemplateConfig(0)
				_ = getTemplateConfig(g) // varied ids, no seeded user -> nil user path
				if i%10 == 0 {
					ClearTemplateConfigCache(g)
				}
			}
		}(g)
	}
	wg.Wait()
}
