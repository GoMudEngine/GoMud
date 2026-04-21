package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSetAggro_GraceGuard_Untargetable verifies that when the registered
// untargetable-check callback returns true for a target userId,
// Character.SetAggro short-circuits without setting Aggro.
func TestSetAggro_GraceGuard_Untargetable(t *testing.T) {
	SetUserUntargetableCheck(func(userId int) bool {
		return userId == 42
	})
	t.Cleanup(func() { SetUserUntargetableCheck(nil) })

	mob := &Character{}
	mob.SetAggro(42, 0, DefaultAttack)

	assert.Nil(t, mob.Aggro, "SetAggro should short-circuit for untargetable user")
}

// TestSetAggro_GraceGuard_Targetable verifies that when the callback
// returns false, SetAggro behaves normally.
func TestSetAggro_GraceGuard_Targetable(t *testing.T) {
	SetUserUntargetableCheck(func(userId int) bool {
		return false
	})
	t.Cleanup(func() { SetUserUntargetableCheck(nil) })

	mob := &Character{}
	mob.SetAggro(42, 0, DefaultAttack)

	assert.NotNil(t, mob.Aggro, "SetAggro should set aggro when target is targetable")
	assert.Equal(t, 42, mob.Aggro.UserId)
}

// TestSetAggro_GraceGuard_NoCallbackRegistered verifies the guard is
// permissive when no callback is registered.
func TestSetAggro_GraceGuard_NoCallbackRegistered(t *testing.T) {
	SetUserUntargetableCheck(nil)

	mob := &Character{}
	mob.SetAggro(42, 0, DefaultAttack)

	assert.NotNil(t, mob.Aggro, "SetAggro with no callback should behave normally")
}

// TestSetAggro_GraceGuard_MobTargetBypasses verifies the guard only
// gates on userId. Targeting a mob (userId=0, mobInstanceId>0) isn't
// affected even if the callback would block some userId.
func TestSetAggro_GraceGuard_MobTargetBypasses(t *testing.T) {
	SetUserUntargetableCheck(func(userId int) bool { return true })
	t.Cleanup(func() { SetUserUntargetableCheck(nil) })

	mob := &Character{}
	mob.SetAggro(0, 200, DefaultAttack)

	assert.NotNil(t, mob.Aggro, "mob-target aggro should not be gated by user check")
	assert.Equal(t, 200, mob.Aggro.MobInstanceId)
}
