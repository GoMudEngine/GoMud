package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// enableItemProcs flips the ItemProcsEnabled gate on in-memory for the test
// process. The hooks test env loads no config file, so the ConfigBool zero
// value is false and the proc gate would never open. AddOverlayOverrides is
// the same in-memory override mechanism warehouse/caravan tests use — it does
// NOT write a file. Idempotent + harmless to leave enabled (no other test
// equips proc-bearing items).
func enableItemProcs(t *testing.T) {
	t.Helper()
	if err := configs.AddOverlayOverrides(map[string]any{
		"GamePlay.ItemProcsEnabled": true,
	}); err != nil {
		t.Fatalf("failed to enable ItemProcsEnabled: %v", err)
	}
}

func TestProcGate_CooldownAndChance(t *testing.T) {
	defer seedAllRegistries()()
	enableItemProcs(t)

	c := characters.New()
	p := items.ItemProc{Trigger: "on_hit", Chance: 100, CooldownRounds: 10, Effect: "lifesteal"}

	if !procGateOpen(c, 12345, 0, p) {
		t.Fatal("100% chance, no cooldown recorded — gate should open")
	}
	markProcCooldown(c, 12345, 0, p)
	if procGateOpen(c, 12345, 0, p) {
		t.Fatal("gate should be closed during cooldown")
	}
}

func TestProcLifesteal(t *testing.T) {
	defer seedAllRegistries()()

	attacker := characters.New()
	attacker.HealthMax.Value = 200
	attacker.Health = 100

	healed := procLifesteal(attacker, 80, map[string]float64{"ratio": 0.25})
	if healed != 20 {
		t.Fatalf("expected 20 healed (25%% of 80), got %d", healed)
	}
	if attacker.Health != 120 {
		t.Fatalf("expected health 120, got %d", attacker.Health)
	}

	attacker.Health = 195
	procLifesteal(attacker, 80, map[string]float64{"ratio": 0.25})
	if attacker.Health != 200 {
		t.Fatalf("expected clamp at 200, got %d", attacker.Health)
	}
	_ = util.GetRoundCount()
}

func TestDispatchOnHitProcs_Lifesteal(t *testing.T) {
	defer seedAllRegistries()()
	enableItemProcs(t)
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999920: {ItemId: 999920, Name: "leech blade", Type: items.Weapon, Hands: 2,
			Procs: []items.ItemProc{{Trigger: "on_hit", Chance: 100, Effect: "lifesteal", Params: map[string]float64{"ratio": 0.25}}}},
	})()

	attacker := characters.New()
	attacker.HealthMax.Value = 200
	attacker.Health = 100
	attacker.Equipment.Weapon = items.New(999920)
	defender := characters.New()

	dispatchItemProcs("on_hit", attacker, defender, nil, 80)

	if attacker.Health != 120 {
		t.Fatalf("on_hit lifesteal expected 120 health, got %d", attacker.Health)
	}
}

// TestMobDeathItemProcs_RecordsLastKill proves the on_kill hook stamps the
// hunger-anchor MiscData key for every player with damage attribution. User 1
// is seeded by seedAllRegistries; a synthetic MobDeath drives the listener
// directly (constructing a full death flow is unnecessary here).
func TestMobDeathItemProcs_RecordsLastKill(t *testing.T) {
	defer seedAllRegistries()()
	enableItemProcs(t)

	u := users.GetByUserId(1)
	evt := events.MobDeath{MobId: 1, PlayerDamage: map[int]int{1: 50}}

	if ret := MobDeathItemProcs(evt); ret != events.Continue {
		t.Fatalf("expected events.Continue, got %v", ret)
	}

	got, ok := readMiscRound(u.Character.GetMiscData("pinnacle_last_kill_round"))
	if !ok || got != util.GetRoundCount() {
		t.Fatalf("expected last-kill round %d, got %v (ok=%v)", util.GetRoundCount(), got, ok)
	}
}
