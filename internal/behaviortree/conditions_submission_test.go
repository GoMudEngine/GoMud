package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// --- PO-076: mob_can_submit_top fires only when mob is controller of sub-eligible grapple ---

func TestCondMobCanSubmitTop_NotGrappling_Failure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine() // Standing default
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condMobCanSubmitTop(nil, ctx))
}

func TestCondMobCanSubmitTop_InMountAsController_Success(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine()
	_ = mob.Character.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = mob.Character.Position.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}, ControlLevel: position.InControl},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condMobCanSubmitTop(nil, ctx))
}

func TestCondMobCanSubmitTop_InMountAsControlled_Failure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine()
	_ = mob.Character.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	// Mount the mob as the controlled side (bottom)
	_ = mob.Character.Position.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}, ControlLevel: position.Controlled},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condMobCanSubmitTop(nil, ctx))
}

// --- PO-077: mob_can_submit_bottom fires only when mob is controlled side of bottom-sub-eligible grapple ---

func TestCondMobCanSubmitBottom_NotGrappling_Failure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine() // Standing default
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condMobCanSubmitBottom(nil, ctx))
}

func TestCondMobCanSubmitBottom_InMountAsControlled_Success(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine()
	_ = mob.Character.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	// Mount as controlled (bottom); Mount has bottom subs (Triangle, Armbar)
	_ = mob.Character.Position.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}, ControlLevel: position.Controlled},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condMobCanSubmitBottom(nil, ctx))
}

func TestCondMobCanSubmitBottom_InKneeOnBellyAsControlled_Failure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine()
	_ = mob.Character.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	// KneeOnBelly has no bottom subs, so condition should fail
	_ = mob.Character.Position.TransitionToKneeOnBelly(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}, ControlLevel: position.Controlled},
		state.TransitionReason{Trigger: position.TriggerPositionAdvance},
	)
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condMobCanSubmitBottom(nil, ctx))
}

func TestCondMobCanSubmitBottom_InMountAsController_Failure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine()
	_ = mob.Character.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	// Mount as controller (top); can't submit as bottom
	_ = mob.Character.Position.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}, ControlLevel: position.InControl},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condMobCanSubmitBottom(nil, ctx))
}

// --- PO-078: mob_submission_policy_is matches policy string param ---

func TestCondMobSubmissionPolicyIs_LethalMatch_Success(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.SubmissionPolicy = characters.PolicyLethal
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	params := map[string]any{"policy": "lethal"}
	assert.Equal(t, Success, condMobSubmissionPolicyIs(params, ctx))
}

func TestCondMobSubmissionPolicyIs_LethalMismatch_Failure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.SubmissionPolicy = characters.PolicySubdue
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	params := map[string]any{"policy": "lethal"}
	assert.Equal(t, Failure, condMobSubmissionPolicyIs(params, ctx))
}

func TestCondMobSubmissionPolicyIs_MercyMatch_Success(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.SubmissionPolicy = characters.PolicyMercy
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	params := map[string]any{"policy": "mercy"}
	assert.Equal(t, Success, condMobSubmissionPolicyIs(params, ctx))
}

func TestCondMobSubmissionPolicyIs_CrippleMatch_Success(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.SubmissionPolicy = characters.PolicyCripple
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	params := map[string]any{"policy": "cripple"}
	assert.Equal(t, Success, condMobSubmissionPolicyIs(params, ctx))
}

func TestCondMobSubmissionPolicyIs_InvalidPolicy_Failure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.SubmissionPolicy = characters.PolicySubdue
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	params := map[string]any{"policy": "invalid"}
	assert.Equal(t, Failure, condMobSubmissionPolicyIs(params, ctx))
}

func TestCondMobSubmissionPolicyIs_NoMob_Failure(t *testing.T) {
	ctx := &EvalContext{InstanceId: 999} // nonexistent
	params := map[string]any{"policy": "lethal"}
	assert.Equal(t, Failure, condMobSubmissionPolicyIs(params, ctx))
}

// --- Lookup verification: all 3 submission primitives registered ---

func TestSubmissionPrimitives_AllRegistered(t *testing.T) {
	names := []string{
		"mob_can_submit_top", "mob_can_submit_bottom", "mob_submission_policy_is",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			if LookupCondition(name) == nil {
				t.Errorf("condition %q not registered", name)
			}
		})
	}
}
