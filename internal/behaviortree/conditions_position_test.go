package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// --- PO-041: mob_in_mount fires only when mob in Mount ---

func TestCondMobInMount_NotInMount_Failure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine() // already Standing
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condMobInMount(nil, ctx))
}

func TestCondMobInMount_InMount_Success(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine()
	_ = mob.Character.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 200}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = mob.Character.Position.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 200}},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condMobInMount(nil, ctx))
}

// --- PO-042: target_is_grappled fires when target is in any grapple state ---

func TestCondTargetIsGrappled_TargetGrappling_Success(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 201}
	target.Character.Name = "Target"
	target.Character.Position = position.NewMachine()
	_ = target.Character.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{MobInstanceId: mob.InstanceId}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condTargetIsGrappled(nil, ctx))
}

func TestCondTargetIsGrappled_TargetStanding_Failure(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 202}
	target.Character.Name = "Target"
	target.Character.Position = position.NewMachine() // Standing default
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condTargetIsGrappled(nil, ctx))
}

// --- PO-043: mob_is_prone fires only when mob is in Prone ---

func TestCondMobIsProne_InProne_Success(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine()
	_ = mob.Character.Position.TransitionToProne(
		position.ProneData{MinRecoveryRounds: 2},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
	)
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condMobIsProne(nil, ctx))
}

func TestCondMobIsProne_InStanding_Failure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine() // Standing default
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condMobIsProne(nil, ctx))
}

// --- Lookup verification: all 10 primitives registered ---

func TestPositionPrimitives_AllRegistered(t *testing.T) {
	names := []string{
		"mob_is_standing", "mob_is_prone", "mob_is_grappling",
		"mob_in_mount", "mob_in_guard", "mob_in_clinch", "mob_in_top_dominant",
		"target_is_standing", "target_is_prone", "target_is_grappled",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			if LookupCondition(name) == nil {
				t.Errorf("condition %q not registered", name)
			}
		})
	}
}
