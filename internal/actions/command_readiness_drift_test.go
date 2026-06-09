package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// driftCase describes one point in the (command × gate) matrix.
type driftCase struct {
	name       string
	cmd        string
	mutate     func(*mobs.Mob)
	wantReady  bool
	wantReason string // Execute*-side flag when not ready; ignored when wantReady=true
}

// TestCommandReadinessDrift asserts CommandIsReady and each Execute*
// agree on readiness for a shared actor state. When they diverge, the
// btree's command_best_of can issue a command the command itself
// silently rejects. Surfaced once in T7 smoke testing of the tank_taunter
// archetype (taunt unregistered on mob side); this test is the guard.
//
// To add a new command or gate: add a row below. The helper at the
// bottom of this file dispatches to the right Execute* and reads the
// named result field.
//
// SCOPE LIMIT: This test checks boolean agreement ("is this command
// ready?"), not which specific reason flag Execute* returns. For the
// not-ready path, gate-ordering differences between CommandIsReady
// and Execute* can cause a case to fail for a cryptic reason-mismatch
// even when the boolean agrees; see the bash_cooldown note below for
// a concrete example.
//
// SYNC POINT: when adding a new gate to CommandIsReady or an
// Execute*, add the corresponding drift row here.
func TestCommandReadinessDrift(t *testing.T) {
	cleanup := seedBuffsForTest()
	defer cleanup()

	// Seed a legged species for the trip_ready / kick_ready "happy path" rows.
	// SpeciesId 0 is intentionally left unseeded so the no_legs rows below can
	// rely on the nil-species → no-legs behavior.
	// SpeciesId 7304 is a clawed species for the rake_ready / rake_notclawed rows.
	speciesCleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7300: {SpeciesId: 7300, Name: "legged-test", BodyParts: []string{"legs"}},
		7304: {SpeciesId: 7304, Name: "clawed-test", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Claws},
		7305: {SpeciesId: 7305, Name: "fanged-test", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Bite},
		7306: {SpeciesId: 7306, Name: "horned-test", BodyParts: []string{"legs", "mouth", "horns"}, NaturalAttack: items.Gore},
		7307: {SpeciesId: 7307, Name: "vampire-test", BodyParts: []string{"arms", "hands", "legs", "mouth"}, NaturalAttack: items.Claws, LifeDrain: true},
		// Hands-bearing beast-identity species for the _hashands drift rows.
		// These have the correct identity (clawed/fanged/horned) but also have
		// "hands", so all beast natural-weapon moves must be blocked.
		7310: {SpeciesId: 7310, Name: "hands-clawed-test", BodyParts: []string{"arms", "hands", "legs", "mouth"}, NaturalAttack: items.Claws},
		7311: {SpeciesId: 7311, Name: "hands-fanged-test", BodyParts: []string{"arms", "hands", "legs", "mouth"}, NaturalAttack: items.Bite},
		7312: {SpeciesId: 7312, Name: "hands-horned-test", BodyParts: []string{"arms", "hands", "legs", "mouth", "horns"}, NaturalAttack: items.Gore},
	})
	defer speciesCleanup()

	cases := []driftCase{
		// ─── taunt ────────────────────────────────────────────────
		{"taunt_ready", "taunt",
			func(m *mobs.Mob) { /* default has aggro */ },
			true, ""},
		{"taunt_crafting", "taunt",
			func(m *mobs.Mob) {
				setCraftingForTest(m)
			},
			false, "Crafting"},
		{"taunt_cooldown", "taunt",
			func(m *mobs.Mob) {
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"taunt_no_aggro", "taunt",
			func(m *mobs.Mob) { m.Character.EndAggro() },
			false, "NoTarget"},

		// ─── rally ────────────────────────────────────────────────
		{"rally_ready", "rally",
			nil,
			true, ""},
		{"rally_crafting", "rally",
			func(m *mobs.Mob) { setCraftingForTest(m) },
			false, "Crafting"},
		{"rally_cooldown", "rally",
			func(m *mobs.Mob) { m.Character.Cooldowns = characters.Cooldowns{"special-move": 3} },
			false, "OnCooldown"},
		{"rally_already_active", "rally",
			func(m *mobs.Mob) { m.Character.AddBuff(80, false) },
			false, "AlreadyActive"},

		// ─── warcry ───────────────────────────────────────────────
		{"warcry_ready", "warcry", nil, true, ""},
		{"warcry_crafting", "warcry",
			func(m *mobs.Mob) { setCraftingForTest(m) },
			false, "Crafting"},
		{"warcry_cooldown", "warcry",
			func(m *mobs.Mob) { m.Character.Cooldowns = characters.Cooldowns{"special-move": 3} },
			false, "OnCooldown"},
		{"warcry_already_active", "warcry",
			func(m *mobs.Mob) { m.Character.AddBuff(79, false) },
			false, "AlreadyActive"},

		// ─── trip ─────────────────────────────────────────────────
		{"trip_ready", "trip",
			func(m *mobs.Mob) {
				// SpeciesId 7300 has legs (seeded above) so the anatomy gate passes.
				m.Character.SpeciesId = 7300
				// Target must be standing (not prone). newTestMob
				// sets default aggro to user 1, but for trip we need
				// a real target mob that's not prone.
				targetMob := &mobs.Mob{InstanceId: 200}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			true, ""},
		{"trip_crafting", "trip",
			func(m *mobs.Mob) {
				setCraftingForTest(m)
				targetMob := &mobs.Mob{InstanceId: 201}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "Crafting"},
		{"trip_cooldown", "trip",
			func(m *mobs.Mob) {
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
				targetMob := &mobs.Mob{InstanceId: 202}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "OnCooldown"},
		{"trip_no_aggro", "trip",
			func(m *mobs.Mob) { m.Character.EndAggro() },
			false, "NoTarget"},

		// ─── bash ─────────────────────────────────────────────────
		// NOTE: bash_cooldown is intentionally omitted. CommandIsReady
		// rejects on the universal cooldown gate first, but ExecuteBash
		// rejects on NoShield first (default test mobs have no shield
		// and no naturalbash). Both agree on the readiness bool, but
		// the reason flag would differ. This test is a readiness-bool
		// agreement test, not a reason-flag agreement test. If
		// ExecuteBash's gate ordering ever changes to put cooldown
		// before NoShield, add the bash_cooldown row back.
		{"bash_crafting", "bash",
			func(m *mobs.Mob) {
				setCraftingForTest(m)
			},
			false, "Crafting"},
		{"bash_no_shield", "bash",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 1 // human, no naturalbash, no shield
			},
			false, "NoShield"},
		// SpeciesId 0 → species.GetSpecies(0) returns nil in test context →
		// HasBodyPart("arms") returns false, naturalBash=false. No shield
		// equipped either. Both gates fire; the shield gate fires first so
		// ExecuteBash returns NoShield=true (reused for the anatomy gate too).
		// CommandIsReady likewise returns false. Boolean agreement is the goal.
		{"bash_no_arms", "bash",
			func(m *mobs.Mob) {
				// SpeciesId stays 0 (nil species, no arms, not natural).
				// Default test mob already has no shield and aggro set.
			},
			false, "NoShield"},

		// ─── grapple ──────────────────────────────────────────────
		{"grapple_crafting", "grapple",
			func(m *mobs.Mob) {
				setCraftingForTest(m)
				targetMob := &mobs.Mob{InstanceId: 204}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "Crafting"},
		{"grapple_cooldown", "grapple",
			func(m *mobs.Mob) {
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
				targetMob := &mobs.Mob{InstanceId: 205}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "OnCooldown"},
		{"grapple_no_aggro", "grapple",
			func(m *mobs.Mob) { m.Character.EndAggro() },
			false, "NoTarget"},
		// SpeciesId 0 → species.GetSpecies(0) returns nil (no species loaded in
		// unit-test context) → HasBodyPart("arms") returns false. A valid
		// non-grappling target is registered so target resolution succeeds and
		// the arms gate is the sole blocking condition in CommandIsReady.
		// ExecuteGrapple checks HasBodyPart before target resolution, so it
		// also fires the arms gate and returns GrappleImmune:true.
		{"grapple_no_arms", "grapple",
			func(m *mobs.Mob) {
				targetMob := &mobs.Mob{InstanceId: 206}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "GrappleImmune"},

		// ─── trip anatomy gate ────────────────────────────────────
		// SpeciesId 0 → species.GetSpecies(0) returns nil (no species loaded in
		// unit-test context) → HasBodyPart("legs") returns false. A valid
		// non-prone target is registered so target resolution succeeds and the
		// legs gate is the sole blocking condition in CommandIsReady.
		// ExecuteTrip also reaches the legs gate (after target resolution) and
		// returns NoTarget:true (reused flag for the unreachable anatomy branch).
		{"trip_no_legs", "trip",
			func(m *mobs.Mob) {
				targetMob := &mobs.Mob{InstanceId: 207}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NoTarget"},

		// ─── kick ─────────────────────────────────────────────────
		{"kick_ready", "kick",
			func(m *mobs.Mob) {
				// SpeciesId 7300 has legs (seeded above) so the anatomy gate passes.
				m.Character.SpeciesId = 7300
			},
			true, ""},
		{"kick_crafting", "kick",
			func(m *mobs.Mob) {
				setCraftingForTest(m)
			},
			false, "Crafting"},
		{"kick_cooldown", "kick",
			func(m *mobs.Mob) {
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"kick_no_aggro", "kick",
			func(m *mobs.Mob) { m.Character.EndAggro() },
			false, "NoTarget"},

		// ─── kick anatomy gate ────────────────────────────────────
		// SpeciesId 0 → nil species → no legs. Default mob has aggro set;
		// CommandIsReady("kick") checks Aggro then HasBodyPart("legs") with no
		// target resolution, so the legs gate is the blocking condition.
		// ExecuteKick burns the cooldown, resolves the target (user 1 not found
		// in test context → NoTarget:true), which coincidentally matches the
		// reused flag for the anatomy gate. Boolean agreement is the goal.
		{"kick_no_legs", "kick",
			func(m *mobs.Mob) {
				// SpeciesId stays 0 (nil species, no legs). Default aggro to user 1.
			},
			false, "NoTarget"},

		// ─── rake ─────────────────────────────────────────────────
		// rake_ready: clawed species + default aggro (user 1, not found) →
		// CommandIsReady returns true (species gate passes, aggro non-nil).
		// Execute is skipped for ready cases.
		{"rake_ready", "rake",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7304 // clawed-test seeded above
			},
			true, ""},
		{"rake_cooldown", "rake",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7304
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"rake_no_aggro", "rake",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7304
				m.Character.EndAggro()
			},
			false, "NoTarget"},
		// rake_notclawed: SpeciesId 0 → nil species → not clawed. Default mob
		// has aggro to user 1. CommandIsReady returns false. ExecuteRake burns
		// the cooldown, resolves the target (user 1 not found → NoTarget first),
		// but with a registered target mob we can reach the NotClawed gate.
		// Use a registered mob target so target resolution succeeds and NotClawed fires.
		{"rake_notclawed", "rake",
			func(m *mobs.Mob) {
				// SpeciesId 0 → nil species → not clawed.
				targetMob := &mobs.Mob{InstanceId: 209}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NotClawed"},

		// ─── maul ─────────────────────────────────────────────────────────────
		// maul_ready: fanged species + default aggro (user 1, not found) →
		// CommandIsReady returns true (species gate passes, aggro non-nil).
		// Execute is skipped for ready cases.
		{"maul_ready", "maul",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7305 // fanged-test seeded above
			},
			true, ""},
		{"maul_cooldown", "maul",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7305
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"maul_no_aggro", "maul",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7305
				m.Character.EndAggro()
			},
			false, "NoTarget"},
		// maul_notfanged: SpeciesId 0 → nil species → not fanged. Default mob
		// has aggro to user 1. CommandIsReady returns false. ExecuteMaul burns
		// the cooldown, resolves the target (user 1 not found → NoTarget first),
		// but with a registered target mob we can reach the NotFanged gate.
		// Use a registered mob target so target resolution succeeds and NotFanged fires.
		{"maul_notfanged", "maul",
			func(m *mobs.Mob) {
				// SpeciesId 0 → nil species → not fanged.
				targetMob := &mobs.Mob{InstanceId: 210}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NotFanged"},

		// ─── pounce ───────────────────────────────────────────────────────────
		// pounce_ready: clawed legged species + default aggro (user 1, not found)
		// → CommandIsReady returns true (predator gate passes, aggro non-nil).
		// Execute is skipped for ready cases.
		{"pounce_ready", "pounce",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7304 // clawed-test seeded above (has legs + claws)
			},
			true, ""},
		{"pounce_cooldown", "pounce",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7304
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"pounce_no_aggro", "pounce",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7304
				m.Character.EndAggro()
			},
			false, "NoTarget"},
		// pounce_notpredator: SpeciesId 0 → nil species → not a quadruped predator.
		// Default mob has aggro to user 1. CommandIsReady returns false.
		// ExecutePounce burns the cooldown, resolves the target (user 1 not
		// found → NoTarget first), but with a registered target mob we can
		// reach the NotPredator gate.
		{"pounce_notpredator", "pounce",
			func(m *mobs.Mob) {
				// SpeciesId 0 → nil species → not a quadruped predator.
				targetMob := &mobs.Mob{InstanceId: 211}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NotPredator"},

		// pounce_grappling: SpeciesId 7304 (predator) but the actor is in
		// Clinch — CommandIsReady gates on !IsGrappling(), ExecutePounce gates
		// after target resolution. A registered target ensures ResolveAggroTarget
		// returns Found=true so the Grappling gate is what fires, not NoTarget.
		{"pounce_grappling", "pounce",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7304 // clawed-test: legs + claws → predator
				// Put the actor into a Clinch grapple so IsGrappling() returns true.
				setCombatPositionParallel(&m.Character, position.Clinch)
				// Register a target so ResolveAggroTarget returns Found=true and
				// the Grappling gate (not NoTarget) is the blocking condition.
				targetMob := &mobs.Mob{InstanceId: 215}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "Grappling"},

		// ─── gore ────────────────────────────────────────────────────────────
		// gore_ready: horned species + default aggro (user 1, not found) →
		// CommandIsReady returns true (horned gate passes, aggro non-nil).
		// Execute is skipped for ready cases.
		{"gore_ready", "gore",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7306 // horned-test seeded above
			},
			true, ""},
		{"gore_cooldown", "gore",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7306
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"gore_no_aggro", "gore",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7306
				m.Character.EndAggro()
			},
			false, "NoTarget"},
		// gore_nothorned: SpeciesId 0 → nil species → not horned.
		// Default mob has aggro to user 1. CommandIsReady returns false.
		// ExecuteGore burns the cooldown, resolves the target (user 1 not
		// found → NoTarget first), but with a registered target mob we can
		// reach the NotHorned gate.
		{"gore_nothorned", "gore",
			func(m *mobs.Mob) {
				// SpeciesId 0 → nil species → not horned.
				targetMob := &mobs.Mob{InstanceId: 212}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NotHorned"},

		// ─── drain ───────────────────────────────────────────────────────────
		// drain_ready: LifeDrain species + default aggro (user 1, not found) →
		// CommandIsReady returns true (lifedrain gate passes, aggro non-nil).
		// Execute is skipped for ready cases.
		{"drain_ready", "drain",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7307 // lifedrain-test seeded above
			},
			true, ""},
		{"drain_cooldown", "drain",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7307
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"drain_no_aggro", "drain",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7307
				m.Character.EndAggro()
			},
			false, "NoTarget"},
		// drain_notlifedrainer: SpeciesId 0 → nil species → no LifeDrain.
		// Default mob has aggro to user 1. CommandIsReady returns false.
		// ExecuteDrain burns the cooldown, resolves the target (user 1 not
		// found → NoTarget first), but with a registered target mob we can
		// reach the NotLifeDrainer gate.
		{"drain_notlifedrainer", "drain",
			func(m *mobs.Mob) {
				// SpeciesId 0 → nil species → no LifeDrain flag.
				targetMob := &mobs.Mob{InstanceId: 213}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NotLifeDrainer"},

		// ─── throttle ─────────────────────────────────────────────────────────
		// throttle_ready: fanged species + default aggro (user 1, not found) →
		// CommandIsReady returns true (species gate passes, aggro non-nil).
		// Execute is skipped for ready cases.
		{"throttle_ready", "throttle",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7305 // fanged-test seeded above
			},
			true, ""},
		{"throttle_cooldown", "throttle",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7305
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"throttle_no_aggro", "throttle",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7305
				m.Character.EndAggro()
			},
			false, "NoTarget"},
		// throttle_notfanged: SpeciesId 0 → nil species → not fanged. Default
		// mob has aggro to user 1. CommandIsReady returns false. ExecuteThrottle
		// burns the cooldown, resolves the target (user 1 not found → NoTarget
		// first), but with a registered target mob we can reach the NotFanged gate.
		{"throttle_notfanged", "throttle",
			func(m *mobs.Mob) {
				// SpeciesId 0 → nil species → not fanged.
				targetMob := &mobs.Mob{InstanceId: 214}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NotFanged"},

		// ─── _hashands rows ───────────────────────────────────────────────────
		// Each row seeds a species with the correct beast identity (clawed/fanged/
		// horned) but also with "hands", so CommandIsReady must return false and
		// the Execute* must return the move's Not<Identity> flag. This exercises
		// the true-beast gate added in P4-T1.

		{"rake_hashands", "rake",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7310 // hands+clawed-test
				targetMob := &mobs.Mob{InstanceId: 216}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NotClawed"},

		{"maul_hashands", "maul",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7311 // hands+fanged-test
				targetMob := &mobs.Mob{InstanceId: 217}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NotFanged"},

		{"gore_hashands", "gore",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7312 // hands+horned-test
				targetMob := &mobs.Mob{InstanceId: 218}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NotHorned"},

		{"throttle_hashands", "throttle",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7311 // hands+fanged-test
				targetMob := &mobs.Mob{InstanceId: 219}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NotFanged"},

		{"hamstring_hashands", "hamstring",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7311 // hands+fanged+legs-test
				targetMob := &mobs.Mob{InstanceId: 220}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NotBeast"},

		{"pounce_hashands", "pounce",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 7310 // hands+clawed+legs-test
				targetMob := &mobs.Mob{InstanceId: 221}
				targetMob.Character.Name = "Target"
				setCombatPositionParallel(&targetMob.Character, position.Standing)
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "NotPredator"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up test mobs after each case to avoid pollution.
			// The mutate function may set up target mobs via
			// mobs.SetInstanceForTest; we clean them all up here.
			defer func() {
				for id := 200; id <= 221; id++ {
					mobs.SetInstanceForTest(id, nil)
				}
			}()

			mob := newTestMob(t, tc.mutate)
			actor := &MobActor{Mob: mob, Room: nil}

			gotReady := CommandIsReady(actor, tc.cmd)
			assert.Equal(t, tc.wantReady, gotReady,
				"CommandIsReady(%s) for case %q", tc.cmd, tc.name)

			if tc.wantReady {
				// Don't run Execute* for happy-path — some have target-
				// resolution or other side effects that require more
				// setup. The !wantReady path is where drift matters.
				return
			}

			gotFlag := runExecuteAndReadFlag(tc.cmd, actor, tc.wantReason)
			assert.True(t, gotFlag,
				"Execute%s for case %q did not return %s=true", tc.cmd, tc.name, tc.wantReason)
		})
	}
}

// runExecuteAndReadFlag dispatches to the Execute* matching cmd and
// returns whether the named result field is true.
func runExecuteAndReadFlag(cmd string, actor Actor, flag string) bool {
	switch cmd {
	case "taunt":
		r := ExecuteTaunt(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		}
	case "rally":
		r := ExecuteRally(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "AlreadyActive":
			return r.AlreadyActive
		}
	case "warcry":
		r := ExecuteWarcry(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "AlreadyActive":
			return r.AlreadyActive
		}
	case "trip":
		r := ExecuteTrip(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		}
	case "bash":
		r := ExecuteBash(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoShield":
			return r.NoShield
		}
	case "grapple":
		r := ExecuteGrapple(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		case "GrappleImmune":
			return r.GrappleImmune
		}
	case "kick":
		r := ExecuteKick(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		}
	case "rake":
		r := ExecuteRake(actor)
		switch flag {
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		case "NotClawed":
			return r.NotClawed
		}
	case "maul":
		r := ExecuteMaul(actor)
		switch flag {
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		case "NotFanged":
			return r.NotFanged
		}
	case "pounce":
		r := ExecutePounce(actor)
		switch flag {
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		case "Grappling":
			return r.Grappling
		case "NotPredator":
			return r.NotPredator
		}
	case "gore":
		r := ExecuteGore(actor)
		switch flag {
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		case "NotHorned":
			return r.NotHorned
		}
	case "drain":
		r := ExecuteDrain(actor)
		switch flag {
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		case "NotLifeDrainer":
			return r.NotLifeDrainer
		}
	case "throttle":
		r := ExecuteThrottle(actor)
		switch flag {
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		case "NotFanged":
			return r.NotFanged
		}
	case "hamstring":
		r := ExecuteHamstring(actor)
		switch flag {
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		case "NotBeast":
			return r.NotBeast
		}
	}
	return false
}

// setCraftingForTest puts a mob into crafting state.
func setCraftingForTest(m *mobs.Mob) {
	m.Character.Activity = activity.NewMachine()
	_ = m.Character.Activity.TransitionToCrafting(
		activity.CraftingData{RecipeId: "test-recipe", RoundsTotal: 5},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin},
	)
}
