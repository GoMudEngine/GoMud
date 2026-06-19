package seasons

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestSeasonsPackageStaysPure: seasons resolves season data and must never
// import the GoMud engine — engine access belongs in engine/.
func TestSeasonsPackageStaysPure(t *testing.T) {
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
				t.Errorf("%s imports forbidden engine package %q (seasons must stay pure)", e.Name(), p)
			}
		}
	}
}
