package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// allPositionStates is every value of the position.State enum. Adding a state
// without extending this list defeats the coverage test below, so the loop in
// TestPositionBucketCoversEveryState also asserts the enum has not grown.
var allPositionStates = []position.State{
	position.Standing,
	position.Prone,
	position.Supine,
	position.Clinch,
	position.BackStanding,
	position.Mount,
	position.SideControl,
	position.KneeOnBelly,
	position.NorthSouth,
	position.Crucifix,
	position.BackGround,
	position.HalfGuard,
	position.Guard,
	position.Turtle,
}

// TestPositionBucketCoversEveryState is the anti-drift gate for the bug where
// positionFields stored capitalized granular names ("Mount") while the summary
// keyed on lowercase buckets ("grounded"), so every position hit rate reported
// 0.0%. A new position.State with no bucket fails here instead of silently
// vanishing from combatstats.
func TestPositionBucketCoversEveryState(t *testing.T) {
	valid := map[string]bool{
		"standing": true, "prone": true, "clinched": true, "grounded": true,
	}

	for _, st := range allPositionStates {
		name := st.String()
		if name == "Unknown" {
			t.Fatalf("state %d stringified as Unknown — enum and String() disagree", int(st))
		}
		bucket := positionBucket(name)
		if bucket == "" {
			t.Errorf("position state %q has no summary bucket — add it to positionBucket", name)
			continue
		}
		if !valid[bucket] {
			t.Errorf("position state %q bucketed to %q, which is not a posMap key", name, bucket)
		}
	}

	// The enum is contiguous from Standing; if a state were appended, its
	// String() would be reachable at the next index and this catches it.
	next := position.State(len(allPositionStates))
	if s := next.String(); s != "Unknown" {
		t.Errorf("position.State gained a value (%q) not present in allPositionStates — "+
			"add it there and give it a bucket in positionBucket", s)
	}
}

// TestPositionBucketMapping pins the semantic groupings, not just coverage.
func TestPositionBucketMapping(t *testing.T) {
	cases := map[position.State]string{
		position.Standing:     "standing",
		position.Prone:        "prone",
		position.Supine:       "prone",
		position.Clinch:       "clinched",
		position.BackStanding: "clinched",
		position.Mount:        "grounded",
		position.Guard:        "grounded",
		position.Turtle:       "grounded",
	}
	for st, want := range cases {
		if got := positionBucket(st.String()); got != want {
			t.Errorf("positionBucket(%q) = %q, want %q", st.String(), got, want)
		}
	}

	if got := positionBucket("NotAPosition"); got != "" {
		t.Errorf("positionBucket on an unknown name = %q, want \"\" so the caller skips it", got)
	}
}

// TestComputeSummaryCountsPositions is the end-to-end proof: granular positions
// recorded on events must reach the summary's per-position hit rates. Before the
// fix these were all 0.0 regardless of input.
func TestComputeSummaryCountsPositions(t *testing.T) {
	events := []CombatEvent{
		{TargetPosition: position.Standing.String(), Hit: true},
		{TargetPosition: position.Standing.String(), Hit: false},
		{TargetPosition: position.Mount.String(), Hit: true},
		{TargetPosition: position.Guard.String(), Hit: true},
		{TargetPosition: position.Clinch.String(), Hit: true},
		{TargetPosition: position.Prone.String(), Hit: false},
	}

	s := computeSummary(events)

	if s.HitRateTargetStanding != 0.5 {
		t.Errorf("HitRateTargetStanding = %v, want 0.5 (1 hit of 2)", s.HitRateTargetStanding)
	}
	// Mount + Guard both bucket to grounded: 2 hits of 2.
	if s.HitRateTargetGrounded != 1.0 {
		t.Errorf("HitRateTargetGrounded = %v, want 1.0 (Mount+Guard, 2 hits of 2)", s.HitRateTargetGrounded)
	}
	if s.HitRateTargetClinched != 1.0 {
		t.Errorf("HitRateTargetClinched = %v, want 1.0", s.HitRateTargetClinched)
	}
	if s.HitRateTargetProne != 0.0 {
		t.Errorf("HitRateTargetProne = %v, want 0.0 (0 hits of 1)", s.HitRateTargetProne)
	}
}
