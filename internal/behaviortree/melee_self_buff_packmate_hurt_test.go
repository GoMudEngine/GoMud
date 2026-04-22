package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

const meleeSelfBuffYAML2 = "../../_datafiles/world/dogmud/behaviors/archetypes/melee_self_buff.yaml"

// TestMeleeSelfBuff_PackmateHurt_SetsAggroOnAttacker verifies the
// packmate_hurt handler engages the attacker by setting Aggro. The
// archetype's normal mob_combat_round cascade (including
// self-buff behavior) fires on the next tick.
func TestMeleeSelfBuff_PackmateHurt_SetsAggroOnAttacker(t *testing.T) {
	LoadArchetypeForTest(t, "melee_self_buff", meleeSelfBuffYAML2)

	mob, cleanup := seedArchetypeMob(t, 90301, map[string]int{})
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	ok := TryMobBehavior(mob.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	if !ok {
		t.Fatalf("TryMobBehavior(packmate_hurt): expected Success, got false")
	}

	// Drain perception-delay queue so mob.Command fires synchronously.
	DrainAllDelayedActionsForTest(t)

	// Assert aggro set on the attacker (via the attack action).
	if mob.Character.Aggro == nil {
		t.Fatalf("packmate_hurt handler should set mob.Aggro; got nil")
	}
	if mob.Character.Aggro.UserId != 42 {
		t.Fatalf("expected Aggro.UserId=42, got %d", mob.Character.Aggro.UserId)
	}
}
