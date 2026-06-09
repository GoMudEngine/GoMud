package templates

import (
	"sync"
	"testing"
)

// Reproduces the data race on the package global forceAnsiFlags: SetAnsiFlag
// (the `server ansi ...` admin command) wrote it with no lock while
// Process/ProcessText/AnsiParse read it under ansiLock.RLock(). Run with -race:
// against the old unlocked write this races; with the write lock it passes.
func TestSetAnsiFlag_ConcurrentWithAnsiParse(t *testing.T) {
	// Save and restore the global so we don't perturb other tests.
	ansiLock.RLock()
	orig := forceAnsiFlags
	ansiLock.RUnlock()
	defer SetAnsiFlag(orig)

	const goroutines = 40
	const iters = 200
	flags := []AnsiFlag{AnsiTagsParse, AnsiTagsStrip, AnsiTagsMono, AnsiTagsDefault}

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				if g%4 == 0 {
					// Writer: flip the forced ansi flag (admin `server ansi`).
					SetAnsiFlag(flags[i%len(flags)])
				} else {
					// Readers: render through the flag.
					_ = AnsiParse(`<ansi fg="red">x</ansi>`)
				}
			}
		}(g)
	}
	wg.Wait()
}
