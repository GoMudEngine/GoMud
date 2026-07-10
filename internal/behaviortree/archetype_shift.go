package behaviortree

// archetype_shift.go — mutation-driven archetype shift (2026-07-10).
//
// Design: docs/superpowers/specs/2026-07-10-mutation-archetype-shift-design.md
// Mobs that acquire a mutation carrying an archetype_pull may re-archetype
// mid-fight. FROM set protects authored behavior; TO whitelist is what any
// mob can credibly play. Pull table is PROVISIONAL pending the mutation-
// graph redesign.

import (
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// shiftEligibleFrom is the FROM set: only these archetypes (and
// archetype-less mobs, for which a shift grants an archetype) ever
// shift. Authored specialists — bosses, leader, casters, thief/
// lookout/scout, noncombat_* — never shift.
var shiftEligibleFrom = map[string]bool{
	"":                true,
	"generic_fighter": true,
	"predator":        true,
	"prey":            true,
	"combat_passive":  true,
}

// shiftTargetWhitelist is the TO set: archetypes any mob can credibly
// play. archer is excluded (needs a ranged weapon + ammo we can't
// conjure); ambusher is TO-only (any mob can take the Hidden buff,
// but authored ambushers keep their tuning).
var shiftTargetWhitelist = map[string]bool{
	"generic_fighter":  true,
	"predator":         true,
	"prey":             true,
	"combat_passive":   true,
	"tank_taunter":     true,
	"defensive_caster": true,
	"pure_caster":      true,
	"ambusher":         true,
}

// validateArchetypePulls is the testable core of ValidateArchetypePulls.
// fileExists is injected so tests don't depend on the config data path.
func validateArchetypePulls(specs []*mutations.MutationSpec, fileExists func(string) bool) error {
	for _, spec := range specs {
		if spec.ArchetypePull == "" {
			continue
		}
		if !shiftTargetWhitelist[spec.ArchetypePull] {
			return fmt.Errorf("mutation %q: archetype_pull %q is not in the shift target whitelist",
				spec.MutationId, spec.ArchetypePull)
		}
		if !fileExists(GetArchetypePath(spec.ArchetypePull)) {
			return fmt.Errorf("mutation %q: archetype_pull %q has no archetype file at %s",
				spec.MutationId, spec.ArchetypePull, GetArchetypePath(spec.ArchetypePull))
		}
	}
	return nil
}

// ValidateArchetypePulls panics at boot when any mutation's
// archetype_pull names a nonexistent archetype or one outside the
// target whitelist — same convention as the schedule validators;
// caught by the pre-push boot test. Call after mutations and behavior
// data files are loaded.
func ValidateArchetypePulls() {
	err := validateArchetypePulls(mutations.AllSpecs(), func(path string) bool {
		_, statErr := os.Stat(path)
		return statErr == nil
	})
	if err != nil {
		panic(err.Error())
	}
}
