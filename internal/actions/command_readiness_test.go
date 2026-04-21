package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

// newTestMob builds a minimal combat-ready mob with configurable state.
func newTestMob(t *testing.T, cfg func(*mobs.Mob)) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{
		MobId:      1,
		InstanceId: 100,
	}
	m.Character.Name = "Test"
	m.Character.Stamina = 999
	m.Character.StaminaMax.Value = 999
	m.Character.Conviction = 999
	m.Character.ConvictionMax.Value = 999
	m.Character.CombatPosition = characters.PositionStanding
	m.Character.SetAggro(1, 0, characters.DefaultAttack) // user 1 as generic target
	if cfg != nil {
		cfg(m)
	}
	return m
}

func TestCommandIsReady_UnknownCommand(t *testing.T) {
	m := newTestMob(t, nil)
	assert.False(t, CommandIsReady(m, "does_not_exist"))
}

func TestCommandIsReady_NilMob(t *testing.T) {
	assert.False(t, CommandIsReady(nil, "taunt"))
}

func TestCommandIsReady_Taunt_NoAggroFalse(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) { m.Character.EndAggro() })
	assert.False(t, CommandIsReady(m, "taunt"))
}

func TestCommandIsReady_Taunt_OnCooldownFalse(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) {
		m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
	})
	assert.False(t, CommandIsReady(m, "taunt"))
}

func TestCommandIsReady_Taunt_ReadyTrue(t *testing.T) {
	m := newTestMob(t, nil)
	assert.True(t, CommandIsReady(m, "taunt"))
}

func TestCommandIsReady_Rally_NoAggroStillTrue(t *testing.T) {
	// Rally doesn't require an aggro target.
	m := newTestMob(t, func(m *mobs.Mob) { m.Character.EndAggro() })
	assert.True(t, CommandIsReady(m, "rally"))
}

func TestCommandIsReady_Warcry_OnCooldownFalse(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) {
		m.Character.Cooldowns = characters.Cooldowns{"special-move": 1}
	})
	assert.False(t, CommandIsReady(m, "warcry"))
}

func TestCommandIsReady_Trip_TargetAlreadyProneFalse(t *testing.T) {
	m := newTestMob(t, nil)
	targetMob := &mobs.Mob{InstanceId: 200}
	targetMob.Character.Name = "Target"
	targetMob.Character.CombatPosition = characters.PositionProne
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)
	m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
	assert.False(t, CommandIsReady(m, "trip"))
}

func TestCommandIsReady_Bash_NoShieldNoNaturalBashFalse(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) {
		m.Character.SpeciesId = 1 // human (no naturalbash)
	})
	assert.False(t, CommandIsReady(m, "bash"))
}

func TestCommandIsReady_Grapple_TargetAlreadyClinchedFalse(t *testing.T) {
	m := newTestMob(t, nil)
	targetMob := &mobs.Mob{InstanceId: 201}
	targetMob.Character.Name = "Target"
	targetMob.Character.CombatPosition = characters.PositionClinched
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)
	m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
	assert.False(t, CommandIsReady(m, "grapple"))
}

func TestCommandIsReady_Kick_ReadyTrue(t *testing.T) {
	m := newTestMob(t, nil)
	assert.True(t, CommandIsReady(m, "kick"))
}
