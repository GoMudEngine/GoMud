package web

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Go test binaries run with CWD set to their own package directory, so any
// test that touches repo files has to climb back out. Same pattern as
// auth_test.go.
func diagramsRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

const (
	diagramsDirRel   = "_datafiles/html/public/diagrams"
	diagramsIndexRel = "_datafiles/html/public/architecture.html"
)

// TestDiagramArtifactsHaveNoTemplateDelimiters guards the hosting seam.
//
// serveTemplate parses EVERY .html file it serves as a text/template, including
// the generated archify artifacts. Those artifacts survive that round trip only
// because they contain no "{{" for the parser to latch onto. If a future
// archify version, or an authored node label, ever introduces one, the file
// stops being a diagram and becomes an HTTP 500. Catch it here rather than on
// the droplet.
func TestDiagramArtifactsHaveNoTemplateDelimiters(t *testing.T) {
	dir := filepath.Join(diagramsRepoRoot(t), filepath.FromSlash(diagramsDirRel))

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}

	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		checked++

		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", entry.Name(), err)
		}

		if idx := strings.Index(string(body), "{{"); idx >= 0 {
			start := idx - 40
			if start < 0 {
				start = 0
			}
			end := idx + 2 + 40
			if end > len(body) {
				end = len(body)
			}
			context := strings.ReplaceAll(string(body[start:end]), "\n", "\\n")
			t.Errorf("%s contains a Go template delimiter at byte %d: serveTemplate "+
				"would fail to parse it and return HTTP 500. Context: %q. This file is "+
				"a generated archify artifact, not hand-authored: find its source spec "+
				"under tools/archify/specs/, fix the offending label there, and "+
				"regenerate the artifact with `archify deliver` (see "+
				"docs/superpowers/plans/2026-08-04-archify-diagrams-tab.md for the exact "+
				"invocation) rather than editing the .html directly.",
				entry.Name(), idx, context)
		}
	}

	if checked == 0 {
		t.Fatalf("no .html artifacts found under %s: the diagrams are generated "+
			"archify artifacts and must be delivered into this directory from their "+
			"source specs (tools/archify/specs/, via `archify deliver`) before this "+
			"test can run.", diagramsDirRel)
	}
}

var diagramHrefPattern = regexp.MustCompile(`href="(/diagrams/[^"]+\.html)"`)

// TestDiagramIndexLinksResolve fails on a card pointing at a file that is not
// there, which would otherwise surface as a 404 only when a visitor clicked it.
func TestDiagramIndexLinksResolve(t *testing.T) {
	root := diagramsRepoRoot(t)

	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(diagramsIndexRel)))
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", diagramsIndexRel, err)
	}

	matches := diagramHrefPattern.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		t.Fatalf("%s matched no href=\"/diagrams/*.html\" links. Either the cards "+
			"were removed, or the href formatting changed and diagramHrefPattern "+
			"needs updating.", diagramsIndexRel)
	}

	for _, match := range matches {
		target := filepath.Join(root, "_datafiles", "html", "public",
			filepath.FromSlash(strings.TrimPrefix(match[1], "/")))
		if _, err := os.Stat(target); err != nil {
			t.Errorf("index card links %s but %s does not exist: %v", match[1], target, err)
		}
	}
}
