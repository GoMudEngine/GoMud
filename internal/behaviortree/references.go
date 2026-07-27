package behaviortree

// The archetype delete guard's evidence (5d): everything that still leans on
// an archetype name, reported verbatim. Deleting a referenced archetype
// would strand mobs (template refs), or leave the mutation shift system
// pulling toward / protecting a tree that no longer exists (shift tables).

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// ArchetypeReferences returns one verbatim line per live reference to the
// named archetype. Empty = safe to delete.
func ArchetypeReferences(name string) []string {
	out := []string{}

	for _, m := range mobs.AllMobTemplates() {
		if m.BehaviorArchetype == name {
			out = append(out, fmt.Sprintf("mob %d (%s, %s): behavior_archetype", int(m.MobId), m.Character.Name, m.Zone))
		}
	}
	sort.Strings(out)

	if shiftEligibleFrom[name] {
		out = append(out, "mutation shift system: FROM set (shiftEligibleFrom) — mobs with this archetype are shift-eligible")
	}
	if shiftTargetWhitelist[name] {
		out = append(out, "mutation shift system: TO whitelist (shiftTargetWhitelist) — shifted mobs can land on this archetype")
	}
	for cluster, target := range clusterArchetype {
		if target == name {
			out = append(out, fmt.Sprintf("mutation shift system: cluster %q pulls toward this archetype", cluster))
		}
	}

	return out
}
