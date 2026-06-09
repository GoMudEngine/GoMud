package templates

import (
	"sync"
	"testing"
)

// Reproduces the data race on the markdown package's global formatter.
// processMarkdown used to call markdown.SetFormatter (a global write) on every
// render, so two concurrent .md template renders raced on that write — the CI
// -race failure in inputhandlers (a /shutdown countdown and a /quit goodbye
// rendering at once). Run with -race: against the old per-render SetFormatter
// this races; with the sync.Once install it passes.
func TestProcessMarkdown_ConcurrentRender(t *testing.T) {
	const goroutines = 50
	const iters = 50
	const md = "# Heading\n\nSome **bold** and *italic* text.\n\n- one\n- two\n"

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				_ = processMarkdown(md)
			}
		}()
	}
	wg.Wait()
}
