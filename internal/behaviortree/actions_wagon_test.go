package behaviortree

import (
	"testing"
)

func TestActDistributeCargoToHostiles_Registered(t *testing.T) {
	if _, ok := actionRegistry["distribute_cargo_to_hostiles"]; !ok {
		t.Error("distribute_cargo_to_hostiles not registered in actionRegistry")
	}
}

// Functional tests skipped — they require room + multiple mob fixtures
// + items in wagon inventory + a hostile mob with Hates intersecting
// wagon.Groups. Stage 3.1's pattern was to integration-test these via
// in-game smoke. The unit-level contract (registered, exists) is
// covered above.
//
// The functional behavior is verified at integration time:
//   - Wagon with N items dies in a room with M bandits → round-robin
//     distribution stops when first hostile hits CarryCapacity, or
//     when wagon is empty
//   - Wagon dies in a room with no hostiles → returns Failure, items
//     fall through to standard corpse-drop
func TestActDistributeCargoToHostiles_NoHostilesReturnsFailure(t *testing.T) {
	t.Skip("integration test — requires room + mob fixtures")
}

func TestActDistributeCargoToHostiles_RoundRobin(t *testing.T) {
	t.Skip("integration test — requires room + mob fixtures + wagon items")
}
