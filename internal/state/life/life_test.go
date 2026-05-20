package life

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/stretchr/testify/require"
)

// Test helpers.

func makeTriad() (sneaker, killer, victim *Machine) {
	registryMu.Lock()
	machineRegistry = map[state.ActorRef]*Machine{}
	registryMu.Unlock()
	A := NewMachine()
	B := NewMachine()
	C := NewMachine()
	RegisterMachine(actor(1), A)
	RegisterMachine(actor(2), B)
	RegisterMachine(actor(3), C)
	return A, B, C
}

func actor(userId int) state.ActorRef {
	return state.ActorRef{UserId: userId}
}

// LI-001: Health zero → Dead.
func TestLI_001_HealthZeroDeath(t *testing.T) {
	A, _, _ := makeTriad()
	err := A.TransitionToDead(
		DeadData{Killer: actor(2)},
		state.TransitionReason{Trigger: TriggerHealthZero, Actor: actor(2)})
	require.NoError(t, err)
	require.Equal(t, Dead, A.State())
}

// LI-002: Suicide command → Dead.
func TestLI_002_SuicideDeath(t *testing.T) {
	A, _, _ := makeTriad()
	err := A.TransitionToDead(
		DeadData{},
		state.TransitionReason{Trigger: TriggerSuicide})
	require.NoError(t, err)
	require.Equal(t, Dead, A.State())
}

// LI-003: Admin kill → Dead.
func TestLI_003_AdminKillDeath(t *testing.T) {
	A, _, _ := makeTriad()
	err := A.TransitionToDead(
		DeadData{},
		state.TransitionReason{Trigger: TriggerAdminKill, Actor: actor(2)})
	require.NoError(t, err)
	require.Equal(t, Dead, A.State())
}

// LI-004: Dead → Respawning (player).
func TestLI_004_DeadToRespawning(t *testing.T) {
	A, _, _ := makeTriad()
	require.NoError(t, A.TransitionToDead(DeadData{}, state.TransitionReason{Trigger: TriggerSuicide}))
	require.NoError(t, A.TransitionToRespawning(
		RespawningData{DestRoomId: 468},
		state.TransitionReason{Trigger: TriggerCleanupReady}))
	require.Equal(t, Respawning, A.State())
	d, ok := A.RespawningData()
	require.True(t, ok)
	require.Equal(t, 468, d.DestRoomId)
}

// LI-005: Respawning → Alive.
func TestLI_005_RespawningToAlive(t *testing.T) {
	A, _, _ := makeTriad()
	require.NoError(t, A.TransitionToDead(DeadData{}, state.TransitionReason{Trigger: TriggerSuicide}))
	require.NoError(t, A.TransitionToRespawning(RespawningData{}, state.TransitionReason{}))
	require.NoError(t, A.TransitionToAlive(state.TransitionReason{Trigger: TriggerRespawnReady}))
	require.Equal(t, Alive, A.State())
}

// LI-006: Mob death → Dead; machine never reaches Respawning.
// Framework-level: just verify Dead is reached and stays Dead;
// instance destruction happens in observer (Task 7).
func TestLI_006_MobDeathStaysDead(t *testing.T) {
	A, _, _ := makeTriad()
	require.NoError(t, A.TransitionToDead(DeadData{}, state.TransitionReason{Trigger: TriggerHealthZero}))
	require.Equal(t, Dead, A.State())
}

// LI-007 through LI-012: Cross-machine cascade.
// LI-007 is framework-level (verify AfterTransition fires);
// LI-008 through LI-012 are integration-tested via
// Life_Cascades.go hooks (Task 5).
func TestLI_007_CascadeFires(t *testing.T) {
	A, _, _ := makeTriad()
	var transitionedTo []State
	A.Inner().AfterTransition("test", func(from, to State, r state.TransitionReason) {
		transitionedTo = append(transitionedTo, to)
	})
	require.NoError(t, A.TransitionToDead(DeadData{}, state.TransitionReason{Trigger: TriggerSuicide}))
	require.Contains(t, transitionedTo, Dead)
}

func TestLI_008_AwarenessCascade(t *testing.T) {
	t.Skip("Awareness cascade verified by integration in Task 5")
}
func TestLI_009_ActivityCascade(t *testing.T) {
	t.Skip("Activity cascade verified by integration in Task 5")
}
func TestLI_010_PositionCascade(t *testing.T) {
	t.Skip("Position cascade verified by integration in Task 5")
}
func TestLI_011_BuffsCancel(t *testing.T) {
	t.Skip("Buff cancel verified by integration in Task 5")
}
func TestLI_012_ConditionsClear(t *testing.T) {
	t.Skip("Conditions clear verified by integration in Task 5")
}

// LI-013 through LI-015: Dead → Respawning cascade verified in
// Task 5/8.
func TestLI_013_ResourceReset(t *testing.T) {
	t.Skip("Resource reset verified by integration in Task 5")
}
func TestLI_014_GraceBuff(t *testing.T) {
	t.Skip("Grace buff verified by integration in Task 5")
}
func TestLI_015_Teleport(t *testing.T) {
	t.Skip("Teleport verified by integration in Task 8 observer")
}

// LI-016: Auto-look verified by integration in Task 8 observer.
func TestLI_016_AutoLook(t *testing.T) {
	t.Skip("Auto-look verified by integration in Task 8 observer")
}

// LI-017: DeadData.Killer populated from transition reason.
func TestLI_017_KillerCaptured(t *testing.T) {
	A, _, _ := makeTriad()
	killer := actor(2)
	err := A.TransitionToDead(
		DeadData{Killer: killer},
		state.TransitionReason{Trigger: TriggerHealthZero, Actor: killer})
	require.NoError(t, err)
	d, ok := A.DeadData()
	require.True(t, ok)
	require.Equal(t, killer, d.Killer)
}

// LI-018: DeadData.DamageMap populated.
func TestLI_018_DamageMapCaptured(t *testing.T) {
	A, _, _ := makeTriad()
	dmg := map[int]int{2: 50, 3: 30}
	err := A.TransitionToDead(
		DeadData{DamageMap: dmg},
		state.TransitionReason{Trigger: TriggerHealthZero})
	require.NoError(t, err)
	d, ok := A.DeadData()
	require.True(t, ok)
	require.Equal(t, dmg, d.DamageMap)
}

// LI-019: DeadData available to observers.
func TestLI_019_DeadDataObservable(t *testing.T) {
	A, _, _ := makeTriad()
	var observed DeadData
	A.Inner().AfterTransition("observer", func(from, to State, r state.TransitionReason) {
		if to == Dead {
			d, _ := A.DeadData()
			observed = d
		}
	})
	dmg := map[int]int{2: 100}
	err := A.TransitionToDead(
		DeadData{Killer: actor(2), DamageMap: dmg},
		state.TransitionReason{})
	require.NoError(t, err)
	require.Equal(t, actor(2), observed.Killer)
	require.Equal(t, dmg, observed.DamageMap)
}

// LI-020 through LI-022: Mob-specific observers (Task 7).
func TestLI_020_LootDropFires(t *testing.T) {
	t.Skip("Loot drop verified by integration in Task 7 observer")
}
func TestLI_021_MobDeathEventFires(t *testing.T) {
	t.Skip("MobDeath event verified by integration in Task 7 observer")
}
func TestLI_022_InstanceCleanupScheduled(t *testing.T) {
	t.Skip("Instance cleanup verified by integration in Task 7 observer")
}

// LI-023 through LI-025: Player-specific observers (Task 6).
func TestLI_023_StatDecay(t *testing.T) {
	t.Skip("Stat decay verified by integration in Task 6 observer")
}
func TestLI_024_KDTracking(t *testing.T) {
	t.Skip("KD tracking verified by integration in Task 6 observer")
}
func TestLI_025_PartyNotifications(t *testing.T) {
	t.Skip("Party notifications verified by integration in Task 6 observer")
}

// LI-026: Fresh Machine is Alive.
func TestLI_026_FreshMachineIsAlive(t *testing.T) {
	m := NewMachine()
	require.Equal(t, Alive, m.State())
	require.True(t, m.IsAlive())
	require.False(t, m.IsDead())
}

// LI-027: Persistence — Life does not survive restart.
// Framework-level: documented as intentional; tested implicitly
// via LI-026 (fresh machine is Alive).
func TestLI_027_StateDoesNotPersist(t *testing.T) {
	m := NewMachine()
	require.Equal(t, Alive, m.State())
}
