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

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
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

// clusterArchetype maps a mutation-graph cluster to the behavior archetype a
// mob drifts toward as it acquires that cluster's mutations (2026-07-12 NPC
// migration — replaces the provisional archetype_pull table). Clusters absent
// here (weaver, trickster, chrysifier) and zero-cluster/Center mutations
// produce no pull. Targets are TO-whitelist archetypes only.
var clusterArchetype = map[string]string{
	"colossus":   "tank_taunter",
	"ironhide":   "tank_taunter",
	"ravener":    "predator",
	"stalker":    "ambusher",
	"ethereal":   "pure_caster",
	"manifester": "defensive_caster",
	"zealot":     "defensive_caster",
}

// archetypeForSpec returns the archetype a mutation pulls toward, derived from
// its clusters (first mapped cluster wins — deterministic because a bridge's
// Clusters slice order is fixed by its YAML). "" = no pull.
func archetypeForSpec(spec *mutations.MutationSpec) string {
	if spec == nil {
		return ""
	}
	for _, cl := range spec.Clusters {
		if a, ok := clusterArchetype[cl]; ok {
			return a
		}
	}
	return ""
}

// archetypeShiftFlavor gives the room-visible line per shift target.
// Keys are TO-whitelist archetypes; anything absent falls back to the
// generic line. No hard numbers, no mechanics leakage (player-message
// SOP).
var archetypeShiftFlavor = map[string]string{
	"generic_fighter":  "squares up, settling into a fighter's stance",
	"predator":         "drops low, its movements turning predatory",
	"prey":             "grows skittish, eyes darting for escape routes",
	"combat_passive":   "stills, its aggression draining away",
	"tank_taunter":     "plants itself and swells with challenge",
	"defensive_caster": "draws inward, the air around it beginning to hum",
	"pure_caster":      "goes strangely calm, eyes lighting with gathered will",
	"ambusher":         "melts toward the shadows, patient and watching",
}

// mobHasPerMobTree reports whether the mob has a per-mob behavior tree —
// per-mob trees shadow archetypes entirely (TryMobBehavior resolution
// order), so such mobs never shift. Checks the engine caches before
// falling back to an os.Stat, mirroring TryMobBehavior.
func mobHasPerMobTree(mobId int, zone string, name string) bool {
	e := GetEngine()
	if e.GetTree(mobId) != nil {
		return true
	}
	if e.HasNoTree(mobId) {
		return false
	}
	_, err := os.Stat(GetBehaviorPath(mobId, zone, name))
	return err == nil
}

// strongestArchetypePull returns the pull of the rarest owned mutation
// that has one (alphabetical key tiebreak, matching the codebase's
// standard rarity sort). Mutations without a pull are ignored entirely —
// their rarity does not compete. Empty string = no pull owned.
func strongestArchetypePull(owned map[string]int) string {
	bestKey, bestRarity, bestPull := "", -1, ""
	for key := range owned {
		spec := mutations.GetMutation(key)
		if spec == nil {
			continue
		}
		pull := archetypeForSpec(spec)
		if pull == "" {
			continue
		}
		if spec.Rarity > bestRarity || (spec.Rarity == bestRarity && key < bestKey) {
			bestKey, bestRarity, bestPull = key, spec.Rarity, pull
		}
	}
	return bestPull
}

// ReevaluateArchetypeShift re-archetypes a mob toward its strongest
// mutation pull. Called from the mob mutation-ACQUISITION path (never
// deepening) right after the new mutation lands. Silent no-op unless
// all gates pass. Mid-combat is the normal case — the next btree event
// simply evaluates the new tree.
func ReevaluateArchetypeShift(mob *mobs.Mob) {
	if mob == nil {
		return
	}
	if !shiftEligibleFrom[mob.BehaviorArchetype] {
		return
	}
	if mobHasPerMobTree(int(mob.MobId), mob.Zone, mob.Character.Name) {
		return
	}
	target := strongestArchetypePull(mob.Character.Mutations)
	if target == "" || target == mob.BehaviorArchetype {
		return
	}

	prior := mob.BehaviorArchetype
	mob.BehaviorArchetype = target
	mob.BTreeState = nil // EnsureBTreeState lazily re-inits on the next event

	// Re-derive policies with the same author-override guard the spawn
	// path uses (mobs.go): explicit YAML policies stay; archetype-derived
	// ones follow the new role.
	if mob.SubmissionPolicy == "" {
		mob.Character.SubmissionPolicy = characters.DefaultSubmissionPolicyForArchetype(target)
	}
	if mob.SurrenderPolicy == "" {
		mob.Character.SurrenderPolicy = characters.DefaultSurrenderPolicyForArchetype(target)
	}

	// Additively merge the new archetype's default goals (learned goals
	// generally survive; a strictly-higher-priority default of the same
	// type can displace per Add's priority rules). Goal WEIGHTS need no
	// work — goals/lookup.go keys off mob.BehaviorArchetype dynamically.
	goals.MergeArchetypeDefaults(int(mob.MobId), util.ConvertForFilename(mob.Character.Name), mob)

	if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
		flavor, ok := archetypeShiftFlavor[target]
		if !ok {
			flavor = "carries itself differently, something fundamental shifted"
		}
		room.SendTextVisual(messaging.CategoryMutation, fmt.Sprintf(
			`<ansi fg="magenta"><ansi fg="mobname">%s</ansi> %s.</ansi>`,
			mob.Character.Name, flavor))
	}

	mudlog.Info("ArchetypeShift",
		"mobId", int(mob.MobId), "instanceId", mob.InstanceId,
		"name", mob.Character.Name, "from", prior, "to", target)
}
