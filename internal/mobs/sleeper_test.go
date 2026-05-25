package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestOnSleeperWoken_StampsWakeRoundOnScheduledMob(t *testing.T) {
	util.SetRoundCountForTest(1000)
	defer util.SetRoundCountForTest(0)

	registerScheduleForTest(&Schedule{
		Id: "sleeper_test_sched",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 1, IdleCommands: []string{"x"}},
		},
	})
	defer unregisterScheduleForTest("sleeper_test_sched")

	mob := &Mob{InstanceId: 99999, ScheduleId: "sleeper_test_sched"}
	mob.Character.IsMob = true
	mob.Character.MobInstanceId = 99999
	SetInstanceForTest(99999, mob)
	defer SetInstanceForTest(99999, nil)

	OnSleeperWoken(&mob.Character)

	got := mob.Character.GetMiscData("schedule_wake_round")
	if got == nil {
		t.Fatalf("expected schedule_wake_round to be stamped, got nil")
	}
	if got.(uint64) != 1000 {
		t.Errorf("expected stamp value 1000, got %v", got)
	}
}

func TestOnSleeperWoken_NoOpForUnscheduledMob(t *testing.T) {
	util.SetRoundCountForTest(1000)
	defer util.SetRoundCountForTest(0)

	mob := &Mob{InstanceId: 99998, ScheduleId: ""}
	mob.Character.IsMob = true
	mob.Character.MobInstanceId = 99998
	SetInstanceForTest(99998, mob)
	defer SetInstanceForTest(99998, nil)

	OnSleeperWoken(&mob.Character)

	if mob.Character.GetMiscData("schedule_wake_round") != nil {
		t.Errorf("expected no stamp for unscheduled mob")
	}
}

func TestOnSleeperWoken_NoOpForPlayer(t *testing.T) {
	util.SetRoundCountForTest(1000)
	defer util.SetRoundCountForTest(0)

	var playerChar characters.Character
	playerChar.IsMob = false

	// Should not panic or stamp anything.
	OnSleeperWoken(&playerChar)
}
