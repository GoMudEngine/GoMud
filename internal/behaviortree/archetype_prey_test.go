package behaviortree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const preyYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/prey.yaml"

func TestArchetype_Prey_Loads(t *testing.T) {
	LoadArchetypeForTest(t, "prey", preyYAML)
	assert.NotNil(t, GetEngine().GetArchetype("prey"))
}
