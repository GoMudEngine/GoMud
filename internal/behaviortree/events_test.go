package behaviortree

// Anti-drift gates for the event vocabulary, in BOTH directions (the
// AddPeriod lesson: pin by vocabulary, verify against the source of truth).

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"gopkg.in/yaml.v2"
)

var eventLiteralRe = regexp.MustCompile(`EventType:\s*"([a-z_]+)"`)

// firedEventsFromSource walks the repo's non-test Go source for
// EventType: "..." literals — the dispatch sites are the source of truth.
func firedEventsFromSource(t *testing.T) map[string]bool {
	t.Helper()
	chdirRepoRootForTest(t)
	fired := map[string]bool{}
	for _, root := range []string{"internal", "modules"} {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return nil
			}
			for _, m := range eventLiteralRe.FindAllStringSubmatch(string(data), -1) {
				fired[m[1]] = true
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(fired) < 10 {
		t.Fatalf("suspiciously few dispatch sites found (%d) — walk broken?", len(fired))
	}
	return fired
}

func TestEventVocabulary_MatchesDispatchSitesBothWays(t *testing.T) {
	fired := firedEventsFromSource(t)

	for ev := range fired {
		if !KnownBehaviorEvents[ev] {
			t.Errorf("event %q is fired by the engine but missing from KnownBehaviorEvents — add it (the editor would refuse it)", ev)
		}
	}
	for ev := range KnownBehaviorEvents {
		if !fired[ev] {
			t.Errorf("KnownBehaviorEvents lists %q but no dispatch site fires it — stale vocabulary entry", ev)
		}
	}
}

// Every event authored in live behavior YAML must be in the vocabulary —
// an unknown event is a silently dead node (the caravan-wagon mob_death
// incident this gate exists to prevent).
func TestEventVocabulary_CoversAllLiveBehaviorYAML(t *testing.T) {
	chdirRepoRootForTest(t)

	root := configs.GetFilePathsConfig().DataFiles.String() + `/behaviors`
	var walkEvents func(n NodeDef, file string)
	walkEvents = func(n NodeDef, file string) {
		if n.Event != "" && !KnownBehaviorEvents[n.Event] {
			t.Errorf("%s: event %q is not in the vocabulary — this node NEVER fires", file, n.Event)
		}
		for _, ch := range n.Children {
			walkEvents(ch, file)
		}
		if n.Child != nil {
			walkEvents(*n.Child, file)
		}
	}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".yaml" {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		var d TreeDef
		if yaml.Unmarshal(data, &d) != nil {
			return nil
		}
		walkEvents(d.Tree, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
