package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// seedPackmateScenario creates a victim + N candidate mobs in the same
// starting state. Returns (victim, candidates slice). Caller sets
// per-candidate routines/state before calling FindPackmatesInRoom.
func seedPackmateScenario(t *testing.T, victimRoutine string, candidateCount int) (*Mob, []*Mob) {
	t.Helper()
	victim := &Mob{
		InstanceId: 1,
		Routine:    victimRoutine,
		Character: characters.Character{
			RoomId: 100,
			Health: 50,
		},
	}
	candidates := make([]*Mob, candidateCount)
	instances := map[int]*Mob{1: victim}
	for i := 0; i < candidateCount; i++ {
		id := 100 + i
		m := &Mob{
			InstanceId: id,
			Character: characters.Character{
				RoomId: 100,
				Health: 50,
			},
		}
		candidates[i] = m
		instances[id] = m
	}
	cleanup := SeedMobsForTest(nil, instances)
	t.Cleanup(cleanup)
	return victim, candidates
}

func TestFindPackmatesInRoom_SameRoutine(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "bandit_camp_guard", 2)
	candidates[0].Routine = "bandit_camp_guard"
	candidates[1].Routine = "different_routine"

	got := FindPackmatesInRoom(victim)

	assert.Len(t, got, 1)
	assert.Equal(t, candidates[0].InstanceId, got[0].InstanceId)
}

func TestFindPackmatesInRoom_RoutineLinks_VictimLinksToCandidate(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "watch_north_road", 1)
	victim.RoutineLinks = []string{"bandit_camp_guard"}
	candidates[0].Routine = "bandit_camp_guard"

	got := FindPackmatesInRoom(victim)
	assert.Len(t, got, 1)
}

func TestFindPackmatesInRoom_RoutineLinks_CandidateLinksToVictim(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "bandit_camp_guard", 1)
	candidates[0].Routine = "watch_north_road"
	candidates[0].RoutineLinks = []string{"bandit_camp_guard"}

	got := FindPackmatesInRoom(victim)
	assert.Len(t, got, 1)
}

func TestFindPackmatesInRoom_DifferentRoom_Excluded(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "wolf_pack", 1)
	candidates[0].Routine = "wolf_pack"
	candidates[0].Character.RoomId = 999

	got := FindPackmatesInRoom(victim)
	assert.Empty(t, got)
}

func TestFindPackmatesInRoom_CharmedExcluded(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "wolf_pack", 1)
	candidates[0].Routine = "wolf_pack"
	candidates[0].Character.Charm(42, 100, "")

	got := FindPackmatesInRoom(victim)
	assert.Empty(t, got, "charmed mob should not be a packmate")
}

func TestFindPackmatesInRoom_DeadExcluded(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "wolf_pack", 1)
	candidates[0].Routine = "wolf_pack"
	candidates[0].Character.Health = 0

	got := FindPackmatesInRoom(victim)
	assert.Empty(t, got, "dead mob should not be a packmate")
}

func TestFindPackmatesInRoom_NoRoutineVictim_EmptyResult(t *testing.T) {
	victim, candidates := seedPackmateScenario(t, "", 2)
	candidates[0].Routine = "wolf_pack"
	candidates[1].Routine = "bandit_camp_guard"

	got := FindPackmatesInRoom(victim)
	assert.Empty(t, got, "victim with no routine has no packmates")
}

func TestFindPackmatesInRoom_ExcludesVictimItself(t *testing.T) {
	victim, _ := seedPackmateScenario(t, "wolf_pack", 0)

	got := FindPackmatesInRoom(victim)
	assert.Empty(t, got, "victim must not match itself")
}
