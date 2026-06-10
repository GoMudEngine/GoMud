package sim

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestSimPackageStaysPure enforces the design spec's guardrail: the sim
// package must never import the GoMud engine. If this fails, move the
// engine-touching code into the engine/ adapter package instead.
func TestSimPackageStaysPure(t *testing.T) {
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
				t.Errorf("%s imports forbidden engine package %q (sim must stay pure)", e.Name(), p)
			}
		}
	}
}
