package actions

import "testing"

// TestRaptorLegsKickBonus pins the Raptor Legs kick branch so it is verified
// without a full combat harness.
func TestRaptorLegsKickBonus(t *testing.T) {
	dmg, kd := raptorLegsKickBonus(map[string]int{}, 0.80, 20)
	if dmg != 0.80 || kd != 20 {
		t.Fatalf("no mutation → unchanged, got dmg=%v kd=%d", dmg, kd)
	}
	dmg2, kd2 := raptorLegsKickBonus(map[string]int{"raptor-legs": 1}, 0.80, 20)
	if !(dmg2 > 0.80 && kd2 > 20) {
		t.Fatalf("raptor-legs should raise kick damage + knockdown, got dmg=%v kd=%d", dmg2, kd2)
	}
}
