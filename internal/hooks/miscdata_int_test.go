package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TestGetMiscDataIntHandlesEveryNumericShape is a regression guard.
//
// MiscData is an any-typed bag written from many places and round-tripped
// through YAML, so the same logical value arrives as int (written directly),
// uint64 (anything storing util.GetRoundCount()), or float64 (after a
// save/load). getMiscDataInt previously handled only int and float64 and
// returned 0 for uint64 — which silently disabled ScheduleWakeGraceRounds,
// because OnSleeperWoken stamps the wake round as a uint64. Scheduled NPCs
// therefore fell straight back to sleep the tick after any wake.
func TestGetMiscDataIntHandlesEveryNumericShape(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want int
	}{
		{"int", int(1234), 1234},
		{"int64", int64(1234), 1234},
		{"uint64 (util.GetRoundCount)", uint64(1234), 1234},
		{"uint", uint(1234), 1234},
		{"float64 (after YAML round-trip)", float64(1234), 1234},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := characters.New()
			c.SetMiscData("k", tc.val)
			if got := getMiscDataInt(c, "k"); got != tc.want {
				t.Errorf("getMiscDataInt(%T) = %d, want %d", tc.val, got, tc.want)
			}
		})
	}

	c := characters.New()
	if got := getMiscDataInt(c, "absent"); got != 0 {
		t.Errorf("absent key = %d, want 0", got)
	}
	c.SetMiscData("s", "not a number")
	if got := getMiscDataInt(c, "s"); got != 0 {
		t.Errorf("non-numeric = %d, want 0", got)
	}
}

// TestWakeRoundStampIsReadable ties the two ends together: whatever
// OnSleeperWoken actually stores must be readable by the schedule executor's
// reader. Testing the pair, not each half, is what would have caught the
// original bug — both sides were individually "correct".
func TestWakeRoundStampIsReadable(t *testing.T) {
	c := characters.New()
	// Mirror mobs.OnSleeperWoken's write exactly.
	c.SetMiscData("schedule_wake_round", util.GetRoundCount())

	got := getMiscDataInt(c, "schedule_wake_round")
	if got != int(util.GetRoundCount()) {
		t.Fatalf("wake stamp wrote %v but reads back as %d — the sleep grace "+
			"cooldown silently does nothing when these disagree",
			util.GetRoundCount(), got)
	}
}
