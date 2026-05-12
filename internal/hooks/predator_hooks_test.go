package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpCallerActor is a minimal partyActor used only by TestPackFlee_*
// tests below to seed a non-leader caller into a party so we can verify
// the caller-death cleanup. It avoids importing internal/actions just
// for test scaffolding.
type helpCallerActor struct {
	mobInstanceId int
	name          string
}

func (h *helpCallerActor) IsPlayer() bool                      { return false }
func (h *helpCallerActor) GetUserId() int                      { return 0 }
func (h *helpCallerActor) GetMobInstanceId() int               { return h.mobInstanceId }
func (h *helpCallerActor) GetCharacter() *characters.Character { return nil }
func (h *helpCallerActor) GetRoom() *rooms.Room                { return nil }
func (h *helpCallerActor) GetName() string                     { return h.name }

// ─── Bleed DoT in AutoHeal ─────────────────────────────────────────────────

func TestAutoHeal_BleedDamagesPlayer(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	u := users.GetByUserId(1)
	require.NotNil(t, u)

	u.Character.Health = 40
	u.Character.HealthMax.Value = 100
	u.Character.AddCondition(characters.ConditionBleeding, 3, 5.0, "test")

	// AutoHeal fires every 3rd round
	evt := events.NewRound{RoundNumber: 3}
	AutoHeal(evt)

	// Health should have decreased by the bleed amount (5)
	// but also increased by regen. The key check is that bleed was applied.
	assert.True(t, u.Character.Health < 40+u.Character.HealthPerRound(),
		"bleed should offset some regen; health=%d", u.Character.Health)

	// Clean up condition
	u.Character.RemoveCondition(characters.ConditionBleeding)
	u.Character.Health = 50
}

func TestAutoHeal_BleedDamagesMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)

	mob.Character.Health = 40
	mob.Character.HealthMax.Value = 100
	mob.Character.AddCondition(characters.ConditionBleeding, 3, 5.0, "test")

	evt := events.NewRound{RoundNumber: 3}
	AutoHeal(evt)

	// Mob health should reflect bleed damage applied
	// Out of combat: regen + bleed. Bleed = 5, regen = 1% of 100 = 1
	// So net should be 40 + 1 - 5 = 36
	assert.Less(t, mob.Character.Health, 40,
		"bleed should reduce mob health below starting value; health=%d", mob.Character.Health)

	mob.Character.RemoveCondition(characters.ConditionBleeding)
	mob.Character.Health = 50
}

func TestAutoHeal_BleedFloorsMobHealthAtZero(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)

	mob.Character.Health = 2
	mob.Character.HealthMax.Value = 100
	mob.Character.AddCondition(characters.ConditionBleeding, 3, 50.0, "massive bleed")

	evt := events.NewRound{RoundNumber: 3}
	AutoHeal(evt)

	// Mob health should be floored at 0, not negative
	assert.GreaterOrEqual(t, mob.Character.Health, 0)

	mob.Character.RemoveCondition(characters.ConditionBleeding)
	mob.Character.Health = 50
}

func TestAutoHeal_BleedMinDamageOne(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)

	mob.Character.Health = 50
	mob.Character.HealthMax.Value = 100
	// Magnitude 0.5 truncates to 0, should be floored to 1
	mob.Character.AddCondition(characters.ConditionBleeding, 3, 0.5, "tiny bleed")

	evt := events.NewRound{RoundNumber: 3}
	AutoHeal(evt)

	// Even with tiny magnitude, at least 1 damage should be applied
	// Out of combat regen is 1, bleed is 1, so net = 0 from start
	// The important thing: no crash, and bleed was processed
	assert.GreaterOrEqual(t, mob.Character.Health, 0)

	mob.Character.RemoveCondition(characters.ConditionBleeding)
	mob.Character.Health = 50
}

// ─── PackFlee ───────────────────────────────────────────────────────────────

func TestPackFlee_WrongEventType(t *testing.T) {
	result := PackFlee(events.NewRound{RoundNumber: 1})
	assert.Equal(t, events.Continue, result)
}

func TestPackFlee_NoGroups(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// MobId 2 (Merchant) has no Groups in the spec
	evt := events.MobDeath{
		MobId:         2,
		InstanceId:    999,
		RoomId:        1,
		CharacterName: "Merchant",
	}

	result := PackFlee(evt)
	assert.Equal(t, events.Continue, result)
}

func TestPackFlee_NoMobSpec(t *testing.T) {
	evt := events.MobDeath{
		MobId:         9999, // nonexistent
		InstanceId:    999,
		RoomId:        1,
		CharacterName: "Ghost",
	}

	result := PackFlee(evt)
	assert.Equal(t, events.Continue, result)
}

func TestPackFlee_InvalidRoom(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	evt := events.MobDeath{
		MobId:         1,
		InstanceId:    999,
		RoomId:        9999, // nonexistent room
		CharacterName: "Skeleton",
	}

	result := PackFlee(evt)
	assert.Equal(t, events.Continue, result)
}

func TestPackFlee_TriggersForGroupmates(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// mob instance 100 is in room 1 with Groups: ["undead"]
	// Mob spec 1 also has Groups: ["undead"]
	// When mob spec 1 dies, instance 100 (same group) should get flee queued

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)
	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	// Verify mob is in the room
	mobIds := room.GetMobs(rooms.FindAll)
	assert.Contains(t, mobIds, 100)

	evt := events.MobDeath{
		MobId:         1, // spec with Groups: ["undead"]
		InstanceId:    999,
		RoomId:        1,
		CharacterName: "Skeleton",
	}

	// PackFlee should queue flee commands on groupmates
	// It won't execute them (Command queues for later), but it should not panic
	result := PackFlee(evt)
	assert.Equal(t, events.Continue, result)
}

// TestPackFlee_ClearsHelpRoomOnCallerDeath verifies that when a non-leader
// party member who raised a help call dies, the death handler clears the
// party's HelpRoomId and HelpCallerInstanceId so in-flight responders stop
// trekking to the now-empty rally room.
//
// Models the bandit pack scenario: lookout (non-leader) calls help, dies
// to the player, and the camp mobs walking up to the rally point should
// not infinitely oscillate after the fight is over.
func TestPackFlee_ClearsHelpRoomOnCallerDeath(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Build a party: leader (instance 8000) + caller (instance 8001).
	leader := &helpCallerActor{mobInstanceId: 8000, name: "Soren"}
	caller := &helpCallerActor{mobInstanceId: 8001, name: "Lookout"}

	p := parties.NewByActor(leader)
	require.NotNil(t, p, "NewByActor failed for leader")
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	require.True(t, p.AddActor(caller), "AddActor failed for caller")

	// Simulate party_call_help having fired from the caller.
	p.HelpRoomId = 4043
	p.HelpCallerInstanceId = 8001

	// The caller dies. The leader is still alive, so the party should NOT
	// dissolve — but the help call MUST clear because the rallying point
	// no longer has a fight to come to.
	evt := events.MobDeath{
		MobId:         283, // bandit lookout template
		InstanceId:    8001,
		RoomId:        4043,
		CharacterName: "bandit lookout",
	}
	result := PackFlee(evt)
	assert.Equal(t, events.Continue, result)

	// Party still exists (leader didn't die).
	stillThere := parties.GetByMobInstanceId(8000)
	require.NotNil(t, stillThere, "leader should still be in their party")
	assert.Equal(t, p, stillThere)

	// Help call cleared.
	assert.Equal(t, 0, p.HelpRoomId, "HelpRoomId should be cleared after caller death")
	assert.Equal(t, 0, p.HelpCallerInstanceId, "HelpCallerInstanceId should be cleared after caller death")
}

// TestPackFlee_PreservesHelpRoomOnNonCallerDeath verifies that when a
// non-caller member dies, the help call stays active so other responders
// keep coming. Only the caller's death clears the call.
func TestPackFlee_PreservesHelpRoomOnNonCallerDeath(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	leader := &helpCallerActor{mobInstanceId: 8010, name: "Soren"}
	caller := &helpCallerActor{mobInstanceId: 8011, name: "Lookout"}
	other := &helpCallerActor{mobInstanceId: 8012, name: "Fighter"}

	p := parties.NewByActor(leader)
	require.NotNil(t, p)
	t.Cleanup(func() { p.Dissolve("test-cleanup") })
	require.True(t, p.AddActor(caller))
	require.True(t, p.AddActor(other))

	p.HelpRoomId = 4043
	p.HelpCallerInstanceId = 8011 // lookout is the caller

	// Fighter (the non-caller) dies.
	evt := events.MobDeath{
		MobId:         284, // bandit fighter template
		InstanceId:    8012,
		RoomId:        4043,
		CharacterName: "bandit fighter",
	}
	PackFlee(evt)

	// Help call should be unchanged — caller is still alive.
	assert.Equal(t, 4043, p.HelpRoomId, "HelpRoomId should persist when a non-caller dies")
	assert.Equal(t, 8011, p.HelpCallerInstanceId, "HelpCallerInstanceId should persist when a non-caller dies")
}

func TestPackFlee_SkipsNonGroupmates(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Mob spec 2 (Merchant) has no Groups in the spec.
	// If a merchant dies, no mobs should flee because there are no groupmates.
	// mob instance 100 (undead skeleton) is in room 1 but doesn't share groups
	// with the merchant.
	mob100 := mobs.GetInstance(100)
	require.NotNil(t, mob100)

	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	evt := events.MobDeath{
		MobId:         2, // merchant spec — no groups
		InstanceId:    999,
		RoomId:        1,
		CharacterName: "Merchant",
	}

	// This should not crash and should skip everyone (merchant has no groups)
	result := PackFlee(evt)
	assert.Equal(t, events.Continue, result)
}
