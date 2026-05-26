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
