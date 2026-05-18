package position

import "testing"

func TestOutcomeTierBucketing(t *testing.T) {
	cases := []struct {
		name string
		absZ float64
		want OutcomeTier
	}{
		{"deep_hold", 0.0, TierHold},
		{"hold_just_under", 0.499, TierHold},
		{"one_step_low", 0.500, TierOneStep},
		{"one_step_mid", 0.75, TierOneStep},
		{"one_step_high", 0.999, TierOneStep},
		{"two_step_low", 1.000, TierTwoStep},
		{"two_step_mid", 1.5, TierTwoStep},
		{"two_step_high", 1.999, TierTwoStep},
		{"three_step_low", 2.000, TierThreeStep},
		{"three_step_high", 5.0, TierThreeStep},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := OutcomeTierFromAbsZ(c.absZ)
			if got != c.want {
				t.Errorf("OutcomeTierFromAbsZ(%v) = %v, want %v", c.absZ, got, c.want)
			}
		})
	}
}

func TestSubWindowGate(t *testing.T) {
	if SubWindowOpens(1.499) {
		t.Error("z=1.499 should not open sub window")
	}
	if !SubWindowOpens(1.500) {
		t.Error("z=1.500 should open sub window")
	}
	if !SubWindowOpens(3.0) {
		t.Error("z=3.0 should open sub window")
	}
}

func TestAdvancementTargetSimple(t *testing.T) {
	// Source position, tier, defender posture → expected target.
	// "Hold" means no position change (controller stays at source).
	cases := []struct {
		name           string
		source         State
		tier           OutcomeTier
		defenderPosture State // for Clinch's posture-based dispatch; irrelevant for others
		wantTarget     State
		wantHold       bool
	}{
		// BackStanding always advances toward BackGround
		{"backstanding_1step", BackStanding, TierOneStep, Standing, BackGround, false},
		{"backstanding_2step", BackStanding, TierTwoStep, Standing, BackGround, false},
		{"backstanding_3step", BackStanding, TierThreeStep, Standing, BackGround, false},

		// Mount: striking apex — 1/2 step is Hold, 3 step takes back
		{"mount_1step_hold", Mount, TierOneStep, Standing, Mount, true},
		{"mount_2step_hold", Mount, TierTwoStep, Standing, Mount, true},
		{"mount_3step_advance", Mount, TierThreeStep, Standing, BackGround, false},

		// SideControl → Mount (1/2), BackGround (3)
		{"sidecontrol_1step", SideControl, TierOneStep, Standing, Mount, false},
		{"sidecontrol_2step", SideControl, TierTwoStep, Standing, Mount, false},
		{"sidecontrol_3step", SideControl, TierThreeStep, Standing, BackGround, false},

		// KneeOnBelly → Mount (1/2), BackGround (3)
		{"kob_1step", KneeOnBelly, TierOneStep, Standing, Mount, false},
		{"kob_2step", KneeOnBelly, TierTwoStep, Standing, Mount, false},
		{"kob_3step", KneeOnBelly, TierThreeStep, Standing, BackGround, false},

		// NorthSouth → SideControl (1), Mount (2), BackGround (3)
		{"ns_1step", NorthSouth, TierOneStep, Standing, SideControl, false},
		{"ns_2step", NorthSouth, TierTwoStep, Standing, Mount, false},
		{"ns_3step", NorthSouth, TierThreeStep, Standing, BackGround, false},

		// Crucifix: terminal apex (sub-only). Always hold from advancement POV.
		{"crucifix_1step_hold", Crucifix, TierOneStep, Standing, Crucifix, true},
		{"crucifix_2step_hold", Crucifix, TierTwoStep, Standing, Crucifix, true},
		{"crucifix_3step_hold", Crucifix, TierThreeStep, Standing, Crucifix, true},

		// BackGround: terminal apex (sub-only).
		{"background_1step_hold", BackGround, TierOneStep, Standing, BackGround, true},
		{"background_2step_hold", BackGround, TierTwoStep, Standing, BackGround, true},
		{"background_3step_hold", BackGround, TierThreeStep, Standing, BackGround, true},

		// HalfGuard → SideControl (1), Mount (2), BackGround (3)
		{"halfguard_1step", HalfGuard, TierOneStep, Standing, SideControl, false},
		{"halfguard_2step", HalfGuard, TierTwoStep, Standing, Mount, false},
		{"halfguard_3step", HalfGuard, TierThreeStep, Standing, BackGround, false},

		// Guard (inverted; controller is bottom) → SideControl (1), Mount (2), BackGround (3)
		{"guard_1step", Guard, TierOneStep, Standing, SideControl, false},
		{"guard_2step", Guard, TierTwoStep, Standing, Mount, false},
		{"guard_3step", Guard, TierThreeStep, Standing, BackGround, false},

		// Turtle → BackGround at all tiers
		{"turtle_1step", Turtle, TierOneStep, Standing, BackGround, false},
		{"turtle_2step", Turtle, TierTwoStep, Standing, BackGround, false},
		{"turtle_3step", Turtle, TierThreeStep, Standing, BackGround, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target, hold := AdvancementTarget(c.source, c.tier, c.defenderPosture)
			if hold != c.wantHold {
				t.Errorf("AdvancementTarget(%v,%v) hold=%v, want %v", c.source, c.tier, hold, c.wantHold)
			}
			if !hold && target != c.wantTarget {
				t.Errorf("AdvancementTarget(%v,%v) target=%v, want %v", c.source, c.tier, target, c.wantTarget)
			}
		})
	}
}

func TestAdvancementTargetClinchPosture(t *testing.T) {
	cases := []struct {
		name            string
		defenderPosture State
		tier            OutcomeTier
		wantTarget      State
	}{
		// Clinch 1-step: posture dispatches
		{"clinch_prone_1step", Prone, TierOneStep, SideControl},
		{"clinch_supine_1step", Supine, TierOneStep, Mount},
		{"clinch_turtle_1step", Turtle, TierOneStep, BackGround},
		{"clinch_standing_1step", Standing, TierOneStep, Mount},

		// Clinch 2-step: always Mount
		{"clinch_prone_2step", Prone, TierTwoStep, Mount},
		{"clinch_standing_2step", Standing, TierTwoStep, Mount},

		// Clinch 3-step: always BackGround
		{"clinch_standing_3step", Standing, TierThreeStep, BackGround},
		{"clinch_prone_3step", Prone, TierThreeStep, BackGround},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target, hold := AdvancementTarget(Clinch, c.tier, c.defenderPosture)
			if hold {
				t.Errorf("Clinch should never Hold on advancement, got hold=true")
			}
			if target != c.wantTarget {
				t.Errorf("AdvancementTarget(Clinch,%v,posture=%v) = %v, want %v", c.tier, c.defenderPosture, target, c.wantTarget)
			}
		})
	}
}
