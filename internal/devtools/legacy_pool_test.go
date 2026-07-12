package devtools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// intendedCenterEnablers is the closed set of zero-cluster ("Center" /
// always-eligible) mutations the graph is allowed to carry. Any OTHER
// mutation YAML with no `clusters:` field is a legacy leftover that leaks
// into every drift/bloom/tutorial grant — the graph's affinityFor returns
// MaxFloat64 for a zero-cluster mutation, so it is ALWAYS eligible.
//
// This walks the filesystem (not the in-memory registry, which is empty in
// unit tests) — same pattern as TestHelpFileCompleteness_*.
var intendedCenterEnablers = map[string]bool{
	"hollow-bones": true, "prehensile-tail": true, "keen-senses": true,
	"rapid-healing": true, "thick-coat": true, "tremorsense": true,
	"precognition": true, "spiracle-lungs": true, "winged-flight": true,
	"tail": true,
}

func TestNoLegacyZeroClusterMutations(t *testing.T) {
	mutDir := filepath.Join(dataRoot(t), "mutations")
	for _, id := range listYAMLBasenames(t, mutDir) {
		body, err := os.ReadFile(filepath.Join(mutDir, id+".yaml"))
		if err != nil {
			t.Fatalf("cannot read mutation %s: %v", id, err)
		}
		hasClusters := false
		for _, line := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "clusters:") {
				hasClusters = true
				break
			}
		}
		if !hasClusters && !intendedCenterEnablers[id] {
			t.Errorf("legacy zero-cluster mutation still present (must be deleted or cluster-tagged): %s", id)
		}
	}
}
