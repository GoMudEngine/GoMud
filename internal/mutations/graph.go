package mutations

import "fmt"

// MutationPrereq is a single prerequisite: an owned mutation id at a
// minimum level. A MinLevel of 0 is treated as 1.
type MutationPrereq struct {
	Id       string `yaml:"id"`
	MinLevel int    `yaml:"min_level"`
}

// KnownClusters is the closed set of design-side cluster tags. "generalist"
// is the central hub. Adding a cluster here is a deliberate design act.
var KnownClusters = map[string]bool{
	"colossus": true, "ironhide": true, "zealot": true, "manifester": true,
	"ethereal": true, "weaver": true, "trickster": true, "stalker": true,
	"ravener": true, "generalist": true,
}

var knownPoles = map[string]bool{"": true, "body": true, "belief": true}

// ValidateGraph panics if any loaded mutation references an unknown cluster,
// an unknown pole, or a prerequisite id that does not exist. Called at boot
// after LoadMutationFiles (same convention as ValidateBodyPartTags).
func ValidateGraph() {
	for _, spec := range allMutations {
		if !knownPoles[spec.Pole] {
			panic(fmt.Sprintf("mutation %q: unknown pole %q (want body|belief|\"\")",
				spec.MutationId, spec.Pole))
		}
		for _, cl := range spec.Clusters {
			if !KnownClusters[cl] {
				panic(fmt.Sprintf("mutation %q: unknown cluster %q", spec.MutationId, cl))
			}
		}
		for _, p := range spec.Prerequisites {
			if _, ok := allMutations[p.Id]; !ok {
				panic(fmt.Sprintf("mutation %q: prerequisite %q does not exist",
					spec.MutationId, p.Id))
			}
		}
	}
}

// ClustersForSkill returns the clusters a skill's use drifts toward (nil if none).
func ClustersForSkill(skill string) []string { return skillClusters[skill] }

// skillClusters maps a skill tag to the cluster(s) its use drifts toward.
// Skills not listed produce no drift. (Ironhide's tank signal comes from a
// damage-absorbed hook wired in a follow-on plan.)
var skillClusters = map[string][]string{
	"weapon-combat":  {"colossus"},
	"unarmed-combat": {"ravener"},
	"ranged-combat":  {"stalker"},
	"skullduggery":   {"stalker"},
	"spellcasting":   {"ethereal"},
	"rhetoric":       {"zealot"},
	"manifestation":  {"manifester"},
}

// OwnedGravity returns each cluster's pull from currently-owned mutations:
// sum of levels of owned mutations tagged with that cluster. Dual-cluster
// (bridge) mutations contribute to both.
func OwnedGravity(owned map[string]int) map[string]float64 {
	g := make(map[string]float64)
	for id, lvl := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, cl := range spec.Clusters {
			g[cl] += float64(lvl)
		}
	}
	return g
}

// PrereqsMet reports whether owned satisfies every prerequisite of spec.
func PrereqsMet(owned map[string]int, spec *MutationSpec) bool {
	for _, p := range spec.Prerequisites {
		min := p.MinLevel
		if min < 1 {
			min = 1
		}
		if owned[p.Id] < min {
			return false
		}
	}
	return true
}

// PoleDepth is the summed rarity×level of owned mutations on the given pole
// ("body" or "belief"). Drives the opposition decay curves.
func PoleDepth(owned map[string]int, pole string) float64 {
	var d float64
	for id, lvl := range owned {
		spec := GetMutation(id)
		if spec == nil || spec.Pole != pole {
			continue
		}
		d += float64(spec.Rarity) * float64(lvl)
	}
	return d
}
