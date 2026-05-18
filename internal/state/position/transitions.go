package position

import "github.com/GoMudEngine/GoMud/internal/state"

// validTransitions enumerates every valid Position transition.
// Star-ish topology around Standing. ~75 edges across 14×14.
// Each row lists the valid SUCCESSOR states from the source state.
var validTransitions = state.TransitionTable[State]{
	Standing: {
		Prone, Supine, Clinch,
	},
	Prone: {
		Standing,
		// Someone mounts the prone target:
		Mount, SideControl, NorthSouth, Crucifix, BackGround,
	},
	Supine: {
		Standing,
		Guard, // pull guard when attacker engages
		// Someone mounts the supine target:
		Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix,
	},
	Clinch: {
		Standing,
		BackStanding,
		Mount, SideControl, Guard, HalfGuard, BackGround,
		// Clinch → KOB / NorthSouth / Crucifix are NOT direct;
		// reach via SideControl first.
	},
	BackStanding: {
		Standing,
		BackGround, // back-controller pulls down
		Clinch,     // controlled turns to face
	},
	Mount: {
		Standing,
		SideControl, KneeOnBelly, NorthSouth, Crucifix, BackGround,
		HalfGuard, Guard, // controlled escapes
	},
	SideControl: {
		Standing,
		Mount, KneeOnBelly, NorthSouth, Crucifix,
		BackGround,
		HalfGuard, Guard, Turtle, // controlled escapes
	},
	KneeOnBelly: {
		Standing,
		Mount, SideControl, NorthSouth,
		HalfGuard, Guard, Turtle, // controlled escapes
	},
	NorthSouth: {
		Standing,
		Mount, SideControl, Crucifix,
		Turtle, // controlled escapes
	},
	Crucifix: {
		Standing,
		SideControl, Mount,
	},
	BackGround: {
		Standing,
		Mount, SideControl,
		Turtle,
	},
	HalfGuard: {
		Standing,
		Guard,
		Mount, SideControl, // top passes
		BackGround,         // bottom takes back via sweep
	},
	Guard: {
		Standing,
		HalfGuard,
		Mount, SideControl, NorthSouth, // top passes
		BackGround,                     // bottom takes back
	},
	Turtle: {
		Standing,
		BackGround, // attacker hooks in
		SideControl, Mount,
	},
}

// Trigger reason constants for Position transitions. 4a names
// every trigger that 4b+ will fire from production code.
const (
	// Knockdowns / falls
	TriggerKnockdownFaceForward  = "knockdown_face_forward"  // → Prone
	TriggerKnockdownFaceBackward = "knockdown_face_backward" // → Supine
	TriggerKnockdownSpell        = "knockdown_spell"         // → Prone or Supine (caller picks)

	// Recovery
	TriggerRecoveryRoll = "recovery_roll" // → Standing (auto, gated by MinRecoveryRounds)
	TriggerStandCommand = "stand_command" // → Standing (explicit, stamina cost, bypasses min)

	// Grapple entry / break
	TriggerGrappleEntry = "grapple_entry" // Standing → Clinch
	TriggerGrappleBreak = "grapple_break" // any grapple → Standing

	// Takedowns from Clinch
	TriggerTakedownMount      = "takedown_mount"
	TriggerTakedownSide       = "takedown_side"
	TriggerTakedownGuardPull  = "takedown_guard_pull"
	TriggerTakedownHalfGuard  = "takedown_half_guard"
	TriggerTakedownBackGround = "takedown_back_ground"

	// Back-takes
	TriggerBackTakeStanding = "back_take_standing" // Clinch → BackStanding
	TriggerBackTakeGround   = "back_take_ground"   // various → BackGround
	TriggerBackPullDown     = "back_pull_down"      // BackStanding → BackGround

	// Controller-initiated transitions within ground subgraph
	TriggerPositionAdvance = "position_advance" // Mount ↔ SC ↔ KOB ↔ NS, etc.

	// TriggerPositionDegrade fires when the defender wins drift by a
	// moderate margin (|z| in [0.5, 1.0)) and the position regresses
	// to a less-dominant state per the spec §6.2 table.
	TriggerPositionDegrade = "position_degrade"

	// TriggerReversal fires when the defender wins drift big (|z| in
	// [1.0, 2.0)). Roles swap; position usually stays the same, with
	// realism exceptions Mount→Guard and BackGround→Mount per spec §6.3.
	TriggerReversal = "reversal"

	// TriggerControlledEscape fires when the defender wins drift
	// decisively (|z| >= 2.0). TransitionPair to Standing regardless
	// of current position. Replaces the chunk-4b "Controlled for 2
	// consecutive rounds" gate.
	TriggerControlledEscape = "controlled_escape"

	// Controlled-initiated escapes
	TriggerPositionEscape = "position_escape" // → Standing or up the chain (HalfGuard, Guard)

	// Defensive
	TriggerTurtleDefend = "turtle_defend" // ground state → Turtle
	TriggerGuardPull    = "guard_pull"    // Supine → Guard

	// Opportunistic top-side
	TriggerMountProneTarget = "mount_prone_target" // attacker takes Prone target

	// Arm-isolation
	TriggerArmIsolation = "arm_isolation" // → Crucifix

	// Cascades
	TriggerDeath = "death" // any → Standing (Life cascade)

	// 4b-placeholder (named here so 4a tests can reference; 4a
	// code paths never fire this — 4b implements rolls + thresholds)
	TriggerControlThresholdCrossed = "control_threshold_crossed"
)
