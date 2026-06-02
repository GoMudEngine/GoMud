package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TestShouldRecoverDisplacedHome covers the legacy MobIdle displacement
// guard. Regression target: chunk 3.6 off-hour pacing bug where scheduled
// NPCs ping-ponged between their schedule's current segment target and
// their original placement room because the schedule executor force-sets
// MaxWander=0 every tick while the legacy guard then "recovered" them
// home as if displaced.
func TestShouldRecoverDisplacedHome(t *testing.T) {
	cases := []struct {
		name string
		mob  *mobs.Mob
		want bool
	}{
		{
			name: "nil mob",
			mob:  nil,
			want: false,
		},
		{
			name: "wanderer (MaxWander > 0) never triggers the guard",
			mob: func() *mobs.Mob {
				m := &mobs.Mob{HomeRoomId: 100, MaxWander: 3}
				m.Character.RoomId = 200
				return m
			}(),
			want: false,
		},
		{
			name: "non-scheduled non-wanderer at home: no recovery needed",
			mob: func() *mobs.Mob {
				m := &mobs.Mob{HomeRoomId: 100, MaxWander: 0}
				m.Character.RoomId = 100
				return m
			}(),
			want: false,
		},
		{
			name: "non-scheduled non-wanderer displaced from home: guard fires (legacy behavior)",
			mob: func() *mobs.Mob {
				m := &mobs.Mob{HomeRoomId: 100, MaxWander: 0}
				m.Character.RoomId = 200
				return m
			}(),
			want: true,
		},
		{
			name: "scheduled mob at non-home segment target: guard SKIPPED (the bug fix)",
			mob: func() *mobs.Mob {
				// Mirrors Kerra evening: HomeRoomId=5101 (forge), but
				// schedule has paced her to 472 (tavern). Without the
				// skip, the legacy guard would yank her back to 5101
				// every other tick.
				m := &mobs.Mob{HomeRoomId: 5101, MaxWander: 0, ScheduleId: "thornwall_smith"}
				m.Character.RoomId = 472
				return m
			}(),
			want: false,
		},
		{
			name: "scheduled mob at home segment target: guard SKIPPED (schedule is the authority)",
			mob: func() *mobs.Mob {
				m := &mobs.Mob{HomeRoomId: 5101, MaxWander: 0, ScheduleId: "thornwall_smith"}
				m.Character.RoomId = 5101
				return m
			}(),
			want: false,
		},
		{
			name: "scheduled mob displaced from segment target: guard SKIPPED (schedule re-paths next tick)",
			mob: func() *mobs.Mob {
				m := &mobs.Mob{HomeRoomId: 5101, MaxWander: 0, ScheduleId: "thornwall_smith"}
				m.Character.RoomId = 9999 // pulled into a third room
				return m
			}(),
			want: false,
		},
		{
			name: "standalone patrol mob displaced: guard SKIPPED (patrol executor is the authority)",
			mob: func() *mobs.Mob {
				m := &mobs.Mob{HomeRoomId: 100, MaxWander: 0, PatrolId: "guard_loop"}
				m.Character.RoomId = 200
				return m
			}(),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRecoverDisplacedHome(tc.mob)
			if got != tc.want {
				t.Errorf("shouldRecoverDisplacedHome() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestMobGoalActedRound covers the "planner owns the tick" stamp reader.
// The stamp is written by behaviortree.actGoalPlanner via the
// "goalActedRound" TempData key whenever a goal planner issues a command.
// The idle handler compares the result against the current round (planner
// acted this tick → suppress legacy idle) and round-1 (planner moved the
// mob last idle round → don't yank it home mid-trip).
func TestMobGoalActedRound(t *testing.T) {
	t.Run("nil mob returns 0", func(t *testing.T) {
		if got := mobGoalActedRound(nil); got != 0 {
			t.Errorf("nil mob: got %d, want 0", got)
		}
	})

	t.Run("absent stamp returns 0", func(t *testing.T) {
		m := &mobs.Mob{}
		if got := mobGoalActedRound(m); got != 0 {
			t.Errorf("unstamped: got %d, want 0", got)
		}
	})

	t.Run("present uint64 stamp is returned", func(t *testing.T) {
		m := &mobs.Mob{}
		m.SetTempData("goalActedRound", uint64(777))
		if got := mobGoalActedRound(m); got != 777 {
			t.Errorf("stamped: got %d, want 777", got)
		}
	})

	t.Run("wrong-type stamp returns 0", func(t *testing.T) {
		m := &mobs.Mob{}
		m.SetTempData("goalActedRound", "not-a-uint64")
		if got := mobGoalActedRound(m); got != 0 {
			t.Errorf("wrong type: got %d, want 0", got)
		}
	})
}

// TestMobGoalPlannerRanRound covers the goal-planner dedup-marker reader. The
// marker is written by behaviortree.RunGoalPlanner via the "goalPlannerRanRound"
// TempData key every round the planner is dispatched. The idle handler compares
// it against the current round so archetype mobs whose btree already ran
// try_goal_planner aren't re-dispatched. Mirrors TestMobGoalActedRound.
func TestMobGoalPlannerRanRound(t *testing.T) {
	t.Run("nil mob returns 0", func(t *testing.T) {
		if got := mobGoalPlannerRanRound(nil); got != 0 {
			t.Errorf("nil mob: got %d, want 0", got)
		}
	})

	t.Run("absent stamp returns 0", func(t *testing.T) {
		m := &mobs.Mob{}
		if got := mobGoalPlannerRanRound(m); got != 0 {
			t.Errorf("unstamped: got %d, want 0", got)
		}
	})

	t.Run("present uint64 stamp is returned", func(t *testing.T) {
		m := &mobs.Mob{}
		m.SetTempData("goalPlannerRanRound", uint64(909))
		if got := mobGoalPlannerRanRound(m); got != 909 {
			t.Errorf("stamped: got %d, want 909", got)
		}
	})

	t.Run("wrong-type stamp returns 0", func(t *testing.T) {
		m := &mobs.Mob{}
		m.SetTempData("goalPlannerRanRound", "not-a-uint64")
		if got := mobGoalPlannerRanRound(m); got != 0 {
			t.Errorf("wrong type: got %d, want 0", got)
		}
	})
}
