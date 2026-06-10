package crawler

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestCrawlerPackageStaysPure mirrors sim's guardrail: the crawler package may
// depend on sim and the standard library, but never on the GoMud engine
// (internal/*). Engine access belongs in the engine/ adapter package.
func TestCrawlerPackageStaysPure(t *testing.T) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(p, "GoMudEngine/GoMud/internal") {
				t.Errorf("%s imports forbidden engine package %q (crawler must stay pure)", e.Name(), p)
			}
		}
	}
}
