package behaviortree

// The 5d writer's safety net (the 5b/5c pattern): for every live behavior
// file, TreeDef parse → marshal → parse → marshal must be a FIXED POINT, the
// recursive node count must survive, and the multiset of check/do/event
// values must be identical. Also pins the note/notes contract: a `note:` key
// on a node round-trips AND never leaks into runtime params.

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// chdirRepoRootForTest points the test at the real repo config + data tree
// (test binaries run with CWD = their own package dir).
func chdirRepoRootForTest(t *testing.T) {
	t.Helper()
	mudlog.SetupLogger(nil, `LOW`, ``, false)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := wd
	for range 6 {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		root = filepath.Dir(root)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
}

func countNodes(n NodeDef) int {
	c := 1
	for _, ch := range n.Children {
		c += countNodes(ch)
	}
	if n.Child != nil {
		c += countNodes(*n.Child)
	}
	return c
}

func collectVocab(n NodeDef, out *[]string) {
	if n.Check != "" {
		*out = append(*out, "check:"+n.Check)
	}
	if n.Do != "" {
		*out = append(*out, "do:"+n.Do)
	}
	if n.Event != "" {
		*out = append(*out, "event:"+n.Event)
	}
	for _, ch := range n.Children {
		collectVocab(ch, out)
	}
	if n.Child != nil {
		collectVocab(*n.Child, out)
	}
}

func TestRoundTrip_MarshalFixedPointEveryLiveBehaviorFile(t *testing.T) {
	chdirRepoRootForTest(t)

	root := configs.GetFilePathsConfig().DataFiles.String() + `/behaviors`
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && filepath.Ext(path) == ".yaml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil || len(files) < 70 {
		t.Fatalf("expected the live behavior tree (~79 files), found %d under %s: %v", len(files), root, err)
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		var d1 TreeDef
		if err := yaml.Unmarshal(data, &d1); err != nil {
			t.Fatalf("%s: parse: %v", f, err)
		}
		out1, err := yaml.Marshal(d1)
		if err != nil {
			t.Fatalf("%s: marshal: %v", f, err)
		}
		var d2 TreeDef
		if err := yaml.Unmarshal(out1, &d2); err != nil {
			t.Fatalf("%s: reparse of own marshal: %v", f, err)
		}
		out2, err := yaml.Marshal(d2)
		if err != nil {
			t.Fatalf("%s: re-marshal: %v", f, err)
		}
		if string(out1) != string(out2) {
			t.Errorf("%s: marshal is not a fixed point — a writer built on this form would churn on every save", f)
		}
		if countNodes(d2.Tree) != countNodes(d1.Tree) {
			t.Errorf("%s: node count changed in round-trip: %d→%d", f, countNodes(d1.Tree), countNodes(d2.Tree))
		}
		var v1, v2 []string
		collectVocab(d1.Tree, &v1)
		collectVocab(d2.Tree, &v2)
		sort.Strings(v1)
		sort.Strings(v2)
		if len(v1) != len(v2) {
			t.Errorf("%s: check/do/event vocabulary changed in round-trip", f)
		} else {
			for i := range v1 {
				if v1[i] != v2[i] {
					t.Errorf("%s: vocabulary mismatch at %d: %q vs %q", f, i, v1[i], v2[i])
					break
				}
			}
		}
		if len(d2.GoalWeights) != len(d1.GoalWeights) || len(d2.DefaultGoals) != len(d1.DefaultGoals) {
			t.Errorf("%s: archetype extras lost in round-trip", f)
		}
	}
}

// The note/notes contract: durable homes for design rationale (marshal drops
// # comments). A note must round-trip AND stay out of runtime params.
func TestNoteFields_RoundTripAndStayOutOfParams(t *testing.T) {
	src := []byte(`
notes: file-level rationale
tree:
  type: selector
  children:
    - type: action
      event: mob_idle
      do: flee
      note: panic example
      rounds: 3
`)
	var d TreeDef
	if err := yaml.Unmarshal(src, &d); err != nil {
		t.Fatal(err)
	}
	if d.Notes != "file-level rationale" {
		t.Fatalf("TreeDef.Notes not bound, got %q", d.Notes)
	}
	child := d.Tree.Children[0]
	if child.Note != "panic example" {
		t.Fatalf("NodeDef.Note not bound, got %q", child.Note)
	}
	params := cleanParams(child)
	if _, leaked := params["note"]; leaked {
		t.Fatal("note leaked into runtime params — add it to knownFields")
	}
	if params["rounds"] != 3 {
		t.Fatalf("real params must survive cleanParams, got %v", params)
	}

	out, err := yaml.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var d2 TreeDef
	if err := yaml.Unmarshal(out, &d2); err != nil {
		t.Fatal(err)
	}
	if d2.Notes != d.Notes || d2.Tree.Children[0].Note != child.Note {
		t.Fatal("note/notes did not round-trip")
	}
}
