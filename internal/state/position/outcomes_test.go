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

func TestDegradeTarget(t *testing.T) {
	cases := []struct {
		name       string
		source     State
		wantTarget State
		wantHold   bool
	}{
		// Symmetric source — no degrade target
		{"clinch_hold", Clinch, Clinch, true},

		// Standard degrades
		{"backstanding_to_clinch", BackStanding, Clinch, false},
		{"mount_to_sidecontrol", Mount, SideControl, false},
		{"sidecontrol_to_halfguard", SideControl, HalfGuard, false},
		{"kob_to_sidecontrol", KneeOnBelly, SideControl, false},
		{"ns_to_sidecontrol", NorthSouth, SideControl, false},
		{"crucifix_to_background", Crucifix, BackGround, false},
		{"background_to_mount", BackGround, Mount, false},
		{"halfguard_to_guard", HalfGuard, Guard, false},

		// Terminal degrade sources — defender must escape or reverse
		{"guard_hold", Guard, Guard, true},
		{"turtle_hold", Turtle, Turtle, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target, hold := DegradeTarget(c.source)
			if hold != c.wantHold {
				t.Errorf("DegradeTarget(%v) hold=%v, want %v", c.source, hold, c.wantHold)
			}
			if !hold && target != c.wantTarget {
				t.Errorf("DegradeTarget(%v) target=%v, want %v", c.source, target, c.wantTarget)
			}
		})
	}
}

func TestReversalTarget(t *testing.T) {
	cases := []struct {
		name       string
		source     State
		wantTarget State
		wantSwap   bool // role swap (always true for reversals)
	}{
		// Realism exceptions
		{"mount_reverse_to_guard", Mount, Guard, true},
		{"background_reverse_to_mount", BackGround, Mount, true},

		// Default: same position, roles swap
		{"clinch_reverse_same", Clinch, Clinch, true},
		{"sidecontrol_reverse_same", SideControl, SideControl, true},
		{"kob_reverse_same", KneeOnBelly, KneeOnBelly, true},
		{"ns_reverse_same", NorthSouth, NorthSouth, true},
		{"crucifix_reverse_same", Crucifix, Crucifix, true},
		{"halfguard_reverse_same", HalfGuard, HalfGuard, true},
		{"guard_reverse_same", Guard, Guard, true},
		{"turtle_reverse_same", Turtle, Turtle, true},
		{"backstanding_reverse_same", BackStanding, BackStanding, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target, swap := ReversalTarget(c.source)
			if target != c.wantTarget {
				t.Errorf("ReversalTarget(%v) target=%v, want %v", c.source, target, c.wantTarget)
			}
			if swap != c.wantSwap {
				t.Errorf("ReversalTarget(%v) swap=%v, want %v", c.source, swap, c.wantSwap)
			}
		})
	}
}

func TestResolveOutcomeHold(t *testing.T) {
	o := ResolveOutcome(Mount, 0.3, Standing)
	if o.Kind != OutcomeHold {
		t.Errorf("z=0.3 should resolve to Hold, got %v", o.Kind)
	}
	if o.SubWindow {
		t.Errorf("z=0.3 should not open sub window")
	}
}

func TestResolveOutcomeAdvance(t *testing.T) {
	// SideControl, z=0.7 (1-step) → Mount, no sub
	o := ResolveOutcome(SideControl, 0.7, Standing)
	if o.Kind != OutcomeAdvance {
		t.Errorf("z=0.7 from SideControl should be Advance, got %v", o.Kind)
	}
	if o.Target != Mount {
		t.Errorf("z=0.7 from SideControl should target Mount, got %v", o.Target)
	}
	if o.SubWindow {
		t.Errorf("z=0.7 should not open sub window")
	}
}

func TestResolveOutcomeAdvanceWithSub(t *testing.T) {
	// SideControl, z=1.7 (2-step, sub-window open) → Mount + sub
	o := ResolveOutcome(SideControl, 1.7, Standing)
	if o.Kind != OutcomeAdvance {
		t.Errorf("z=1.7 should be Advance, got %v", o.Kind)
	}
	if o.Target != Mount {
		t.Errorf("z=1.7 from SideControl should target Mount, got %v", o.Target)
	}
	if !o.SubWindow {
		t.Errorf("z=1.7 should open sub window")
	}
}

func TestResolveOutcomeAdvanceWithSubAtThreeStep(t *testing.T) {
	// SideControl, z=2.5 (3-step) → BackGround + sub
	o := ResolveOutcome(SideControl, 2.5, Standing)
	if o.Target != BackGround {
		t.Errorf("z=2.5 from SideControl should target BackGround, got %v", o.Target)
	}
	if !o.SubWindow {
		t.Errorf("z=2.5 should open sub window")
	}
}

func TestResolveOutcomeMountStrikingApex(t *testing.T) {
	// Mount, z=1.7 (2-step) → Hold (striking apex), but sub window opens
	o := ResolveOutcome(Mount, 1.7, Standing)
	if o.Kind != OutcomeHold {
		t.Errorf("z=1.7 from Mount should be Hold (striking apex), got %v", o.Kind)
	}
	if !o.SubWindow {
		t.Errorf("z=1.7 from Mount should still open sub window (independent gate)")
	}
}

func TestResolveOutcomeDegrade(t *testing.T) {
	// Mount, z=-0.7 (defender 1-step) → degrade to SideControl
	o := ResolveOutcome(Mount, -0.7, Standing)
	if o.Kind != OutcomeDegrade {
		t.Errorf("z=-0.7 from Mount should be Degrade, got %v", o.Kind)
	}
	if o.Target != SideControl {
		t.Errorf("z=-0.7 from Mount should degrade to SideControl, got %v", o.Target)
	}
}

func TestResolveOutcomeDegradeTerminalClinchHold(t *testing.T) {
	// Clinch, z=-0.7 → Hold (Clinch has no degrade target)
	o := ResolveOutcome(Clinch, -0.7, Standing)
	if o.Kind != OutcomeHold {
		t.Errorf("z=-0.7 from Clinch should Hold (no degrade target), got %v", o.Kind)
	}
}

func TestResolveOutcomeReversal(t *testing.T) {
	// Mount, z=-1.5 → Reversal to Guard
	o := ResolveOutcome(Mount, -1.5, Standing)
	if o.Kind != OutcomeReversal {
		t.Errorf("z=-1.5 from Mount should be Reversal, got %v", o.Kind)
	}
	if o.Target != Guard {
		t.Errorf("Mount reversal target should be Guard, got %v", o.Target)
	}
}

func TestResolveOutcomeEscape(t *testing.T) {
	// Mount, z=-2.5 → Escape to Standing
	o := ResolveOutcome(Mount, -2.5, Standing)
	if o.Kind != OutcomeEscape {
		t.Errorf("z=-2.5 from Mount should be Escape, got %v", o.Kind)
	}
	if o.Target != Standing {
		t.Errorf("Escape target should be Standing, got %v", o.Target)
	}
}
