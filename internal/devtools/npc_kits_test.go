package devtools

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// speciesPrimaryCluster is the executable encoding of the NPC-kits design §4:
// each kitted, cluster-anchored species and the body/belief cluster its kit must
// anchor to. Center-anchored species (skeleton→hollow-bones) and bare species are
// omitted — they are not anchoring-checked. Keep in sync with the design doc.
var speciesPrimaryCluster = map[string]string{
	"32-wraith": "ethereal", "33-spectre": "ethereal", "0-ghostly_spirit": "ethereal",
	"31-zombie": "ironhide", "34-vampire": "stalker", "35-flesh_golem": "colossus",
	"36-water_elemental": "ironhide", "37-earth_elemental": "colossus",
	"38-air_elemental": "ethereal", "39-fire_elemental": "ethereal",
	"40-magma_elemental": "ironhide", "41-sand_elemental": "stalker",
	"42-storm_elemental": "ethereal", "43-ice_elemental": "ironhide",
	"44-smoke_elemental": "ethereal", "23-aberration": "trickster",
	"99-ascended": "zealot", "16-slime": "ironhide", "15-fungal_colony": "weaver",
	"14-carnivorous_plant": "weaver", "20-orb": "ethereal", "4-troll": "ironhide",
}

var intrinsicLineRe = regexp.MustCompile(`^\s+([a-z0-9-]+):\s*\d+\s*$`)
var clustersLineRe = regexp.MustCompile(`^clusters:\s*\[(.*)\]`)

// mutationClusters reads the clusters tag list from a mutation YAML. The second
// return is false when the mutation file does not exist.
func mutationClusters(t *testing.T, root, id string) ([]string, bool) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root, "mutations", id+".yaml"))
	if err != nil {
		return nil, false // mutation does not exist
	}
	for _, ln := range strings.Split(string(body), "\n") {
		if m := clustersLineRe.FindStringSubmatch(strings.TrimSpace(ln)); m != nil {
			var out []string
			for _, c := range strings.Split(m[1], ",") {
				if c = strings.TrimSpace(c); c != "" {
					out = append(out, c)
				}
			}
			return out, true
		}
	}
	return nil, true // exists, zero-cluster (Center)
}

// speciesIntrinsicIDs returns the intrinsic mutation ids declared in a species file.
func speciesIntrinsicIDs(t *testing.T, path string) []string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var ids []string
	in := false
	for _, ln := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(ln, "intrinsic_mutations:") {
			in = true
			continue
		}
		if in {
			if m := intrinsicLineRe.FindStringSubmatch(ln); m != nil {
				ids = append(ids, m[1])
			} else if strings.TrimSpace(ln) != "" && !strings.HasPrefix(ln, " ") {
				in = false
			}
		}
	}
	return ids
}

// TestNPCKits_IDsLiveAndAnchored asserts every species intrinsic id is a live
// mutation, and every cluster-anchored species (per design §4) has a kit with at
// least one member in its declared primary cluster.
func TestNPCKits_IDsLiveAndAnchored(t *testing.T) {
	root := dataRoot(t)
	speciesDir := filepath.Join(root, "species")

	// (1) every intrinsic id must resolve to a live mutation YAML.
	speciesFiles, _ := filepath.Glob(filepath.Join(speciesDir, "*.yaml"))
	for _, f := range speciesFiles {
		for _, id := range speciesIntrinsicIDs(t, f) {
			if _, ok := mutationClusters(t, root, id); !ok {
				t.Errorf("species %s references non-existent mutation %q", filepath.Base(f), id)
			}
		}
	}

	// (2) each cluster-anchored species has a kit anchored to its primary cluster.
	for sp, cluster := range speciesPrimaryCluster {
		ids := speciesIntrinsicIDs(t, filepath.Join(speciesDir, sp+".yaml"))
		if len(ids) == 0 {
			t.Errorf("species %s: expected a cluster kit, has no intrinsic_mutations", sp)
			continue
		}
		anchored := false
		for _, id := range ids {
			cls, _ := mutationClusters(t, root, id)
			for _, c := range cls {
				if c == cluster {
					anchored = true
				}
			}
		}
		if !anchored {
			t.Errorf("species %s: kit %v not anchored to primary cluster %q", sp, ids, cluster)
		}
	}
}
