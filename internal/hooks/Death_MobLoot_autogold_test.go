package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/parties"
)

// Auto-loot-on-kill routing: a killed mob's gold goes to the killer's party pool
// (settled/split later) or straight to a solo killer's purse.

func TestPlanAutoLootGold_Solo(t *testing.T) {
	plan := planAutoLootGold(map[int]int{9002: 50}, 200)
	if plan.toPartyPool || plan.userId != 9002 || plan.amount != 200 {
		t.Errorf("solo plan = %+v, want {userId 9002, amount 200}", plan)
	}
}

func TestPlanAutoLootGold_Party(t *testing.T) {
	p := parties.New(9001)
	if p == nil {
		t.Fatal("could not create test party for 9001")
	}
	defer p.Disband()

	plan := planAutoLootGold(map[int]int{9001: 50}, 200)
	if !plan.toPartyPool || plan.amount != 200 {
		t.Errorf("party plan = %+v, want {toPartyPool true, amount 200}", plan)
	}
}

func TestPlanAutoLootGold_NoPlayerKiller(t *testing.T) {
	if plan := planAutoLootGold(map[int]int{}, 200); plan.amount != 0 {
		t.Errorf("no-killer plan = %+v, want zero", plan)
	}
}

func TestPlanAutoLootGold_NoGold(t *testing.T) {
	if plan := planAutoLootGold(map[int]int{9002: 50}, 0); plan.amount != 0 {
		t.Errorf("no-gold plan = %+v, want zero", plan)
	}
}
