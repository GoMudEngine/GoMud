package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/position"
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
	setCombatPositionParallel(&m.Character, position.Standing)
	m.Character.Buffs = buffs.New()                      // Properly initialize buffs maps
	m.Character.SetAggro(1, 0, characters.DefaultAttack) // user 1 as generic target
	if cfg != nil {
		cfg(m)
	}
	return m
}

func TestCommandIsReady_UnknownCommand(t *testing.T) {
	m := newTestMob(t, nil)
	actor := &MobActor{Mob: m, Room: nil}
	assert.False(t, CommandIsReady(actor, "does_not_exist"))
}

func TestCommandIsReady_NilActor(t *testing.T) {
	assert.False(t, CommandIsReady(nil, "taunt"))
}

func TestCommandIsReady_Taunt_NoAggroFalse(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) { m.Character.EndAggro() })
	actor := &MobActor{Mob: m, Room: nil}
	assert.False(t, CommandIsReady(actor, "taunt"))
}

func TestCommandIsReady_Taunt_OnCooldownFalse(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) {
		m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
	})
	actor := &MobActor{Mob: m, Room: nil}
	assert.False(t, CommandIsReady(actor, "taunt"))
}

func TestCommandIsReady_Taunt_ReadyTrue(t *testing.T) {
	m := newTestMob(t, nil)
	actor := &MobActor{Mob: m, Room: nil}
	assert.True(t, CommandIsReady(actor, "taunt"))
}

func TestCommandIsReady_Rally_NoAggroStillTrue(t *testing.T) {
	// Rally doesn't require an aggro target.
	m := newTestMob(t, func(m *mobs.Mob) { m.Character.EndAggro() })
	actor := &MobActor{Mob: m, Room: nil}
	assert.True(t, CommandIsReady(actor, "rally"))
}

func TestCommandIsReady_Warcry_OnCooldownFalse(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) {
		m.Character.Cooldowns = characters.Cooldowns{"special-move": 1}
	})
	actor := &MobActor{Mob: m, Room: nil}
	assert.False(t, CommandIsReady(actor, "warcry"))
}

func TestCommandIsReady_Trip_TargetAlreadyProneFalse(t *testing.T) {
	m := newTestMob(t, nil)
	targetMob := &mobs.Mob{InstanceId: 200}
	targetMob.Character.Name = "Target"
	setCombatPositionParallel(&targetMob.Character, position.Prone)
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)
	m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
	actor := &MobActor{Mob: m, Room: nil}
	assert.False(t, CommandIsReady(actor, "trip"))
}

func TestCommandIsReady_Bash_NoShieldNoNaturalBashFalse(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) {
		m.Character.SpeciesId = 1 // human (no naturalbash)
	})
	actor := &MobActor{Mob: m, Room: nil}
	assert.False(t, CommandIsReady(actor, "bash"))
}

func TestCommandIsReady_Grapple_TargetAlreadyClinchedFalse(t *testing.T) {
	m := newTestMob(t, nil)
	targetMob := &mobs.Mob{InstanceId: 201}
	targetMob.Character.Name = "Target"
	setCombatPositionParallel(&targetMob.Character, position.Clinch)
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)
	m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
	actor := &MobActor{Mob: m, Room: nil}
	assert.False(t, CommandIsReady(actor, "grapple"))
}

func TestCommandIsReady_Kick_ReadyTrue(t *testing.T) {
	// Seed species 0 with legs so the anatomy gate passes for the default test mob.
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		0: {SpeciesId: 0, Name: "human", BodyParts: []string{"legs"}},
	})
	defer cleanup()
	m := newTestMob(t, nil)
	actor := &MobActor{Mob: m, Room: nil}
	assert.True(t, CommandIsReady(actor, "kick"))
}

// ─── IsCrafting (universal) ────────────────────────────────────────────────

func TestCommandIsReady_IsCrafting_BlocksEveryCommand(t *testing.T) {
	for _, cmd := range []string{"taunt", "rally", "warcry", "trip", "bash", "grapple", "kick"} {
		t.Run(cmd, func(t *testing.T) {
			m := newTestMob(t, func(m *mobs.Mob) {
				m.Character.Activity = activity.NewMachine()
				_ = m.Character.Activity.TransitionToCrafting(
					activity.CraftingData{RecipeId: "test", RoundsTotal: 1},
					state.TransitionReason{Trigger: activity.TriggerCraftBegin},
				)
			})
			actor := &MobActor{Mob: m, Room: nil}
			assert.False(t, CommandIsReady(actor, cmd),
				"crafting mob should NOT be ready for %s", cmd)
		})
	}
}

func seedBuffsForTest() func() {
	return buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		79: {BuffId: 79, Name: "Warcry", TriggerCount: 10, RoundInterval: 1},
		80: {BuffId: 80, Name: "Rally", TriggerCount: 10, RoundInterval: 1},
	})
}

// ─── Rally AlreadyActive ───────────────────────────────────────────────────

func TestCommandIsReady_Rally_BuffAlreadyActive_False(t *testing.T) {
	cleanup := seedBuffsForTest()
	defer cleanup()

	m := newTestMob(t, nil)
	m.Character.AddBuff(80, false)
	actor := &MobActor{Mob: m, Room: nil}
	assert.False(t, CommandIsReady(actor, "rally"))
}

func TestCommandIsReady_Rally_BuffNotActive_True(t *testing.T) {
	cleanup := seedBuffsForTest()
	defer cleanup()

	m := newTestMob(t, nil)
	actor := &MobActor{Mob: m, Room: nil}
	assert.True(t, CommandIsReady(actor, "rally"))
}

// ─── Warcry AlreadyActive ──────────────────────────────────────────────────

func TestCommandIsReady_Warcry_BuffAlreadyActive_False(t *testing.T) {
	cleanup := seedBuffsForTest()
	defer cleanup()

	m := newTestMob(t, nil)
	m.Character.AddBuff(79, false)
	actor := &MobActor{Mob: m, Room: nil}
	assert.False(t, CommandIsReady(actor, "warcry"))
}

func TestCommandIsReady_Warcry_BuffNotActive_True(t *testing.T) {
	cleanup := seedBuffsForTest()
	defer cleanup()

	m := newTestMob(t, nil)
	actor := &MobActor{Mob: m, Room: nil}
	assert.True(t, CommandIsReady(actor, "warcry"))
}
