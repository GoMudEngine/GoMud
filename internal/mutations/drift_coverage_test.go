package mutations

import "testing"

// combatDriftClusters are the clusters reached via 6e combat behavior hooks
// (they have no skillClusters entry by design). Keep in sync with the drift
// hooks in internal/hooks/NewRound_DoCombat_unified.go.
var combatDriftClusters = []string{"ironhide", "trickster", "weaver"}

// TestAllClustersReachable guards that every design cluster has SOME emergent
// drift path — a skillClusters entry or a combat-behavior hook. Without this,
// a cluster is only reachable via re-spec phials (a later feature).
func TestAllClustersReachable(t *testing.T) {
	reached := map[string]bool{}
	for _, cls := range skillClusters {
		for _, c := range cls {
			reached[c] = true
		}
	}
	for _, c := range combatDriftClusters {
		reached[c] = true
	}
	for cluster := range KnownClusters {
		if cluster == "generalist" {
			continue // reached by being zero-cluster, not by drift
		}
		if !reached[cluster] {
			t.Errorf("cluster %q is unreachable — no skill or combat drift signal", cluster)
		}
	}
}
