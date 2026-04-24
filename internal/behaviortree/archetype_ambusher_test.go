package behaviortree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const ambusherYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/ambusher.yaml"

func TestArchetype_Ambusher_Loads(t *testing.T) {
	LoadArchetypeForTest(t, "ambusher", ambusherYAML)
	assert.NotNil(t, GetEngine().GetArchetype("ambusher"))
}

func TestArchetype_Ambusher_HandlesMobIdle(t *testing.T) {
	LoadArchetypeForTest(t, "ambusher", ambusherYAML)
	arch := GetEngine().GetArchetype("ambusher")
	if arch == nil {
		t.Fatal("archetype not loaded")
	}
	ctx := &EvalContext{
		InstanceId: 14001,
		RoomId:     1,
		Event: EventContext{
			EventType: "mob_idle",
		},
	}
	// Structural pass — expect no panic. Mob isn't in the test harness so
	// actAddBuff will return Failure at the mobs.GetInstance lookup; that's
	// fine, we're just asserting tree shape.
	_ = arch.Evaluate(ctx)
}
