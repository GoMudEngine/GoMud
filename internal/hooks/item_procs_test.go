package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// seedStunBuff registers buff 84 (the 1-round stagger-Stun) into the buff
// registry for the duration of a test. seedAllRegistries seeds only buffs
// 100/101, so aoe_stun's AddBuff(84) would silently fail without this.
func seedStunBuff(t *testing.T) func() {
	t.Helper()
	return buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		84: {
			BuffId:        84,
			Name:          "Stunned",
			Description:   "Reeling — no meaningful attack or defense this round.",
			RoundInterval: 1,
			TriggerCount:  1,
		},
	})
}

// addTestMob registers a room-1 mob instance for aoe_stun targeting tests.
func addTestMob(instanceId int, nonCombatant bool) *mobs.Mob {
	m := &mobs.Mob{
		InstanceId: instanceId,
		HomeRoomId: 1,
		Character: characters.Character{
			Name:         "Test Beast",
			RoomId:       1,
			NonCombatant: nonCombatant,
			Buffs:        buffs.New(),
			Cooldowns:    map[string]int{},
		},
	}
	m.Character.HealthMax.Value = 50
	m.Character.Health = 50
	mobs.SetInstanceForTest(instanceId, m)
	if r := rooms.LoadRoom(1); r != nil {
		r.AddMob(instanceId)
	}
	return m
}

func TestProcAoeStun_StunsHostilesSkipsProtected(t *testing.T) {
	defer seedAllRegistries()()
	defer seedStunBuff(t)()
	enableItemProcs(t)

	// seedAllRegistries already places hostile mob instance 100 (Skeleton,
	// not a non-combatant) in room 1. Add a second hostile + a non-combatant.
	hostileA := mobs.GetInstance(100)
	if hostileA == nil {
		t.Fatal("expected seeded hostile mob instance 100")
	}
	hostileB := addTestMob(201, false)
	nonCombatant := addTestMob(202, true)

	// Owner: player 1 (already in room 1). Ensure the userId back-reference is
	// set so the party/charm ally logic can identify the owner.
	owner := users.GetByUserId(1).Character
	owner.SetUserId(1)

	room := rooms.LoadRoom(1)
	if ok := procAoeStun(owner, room, map[string]float64{}); !ok {
		t.Fatal("aoe_stun should execute (true) with hostile mobs present")
	}

	if !hostileA.Character.HasBuff(84) {
		t.Error("hostile mob 100 should be stunned (buff 84)")
	}
	if !hostileB.Character.HasBuff(84) {
		t.Error("hostile mob 201 should be stunned (buff 84)")
	}
	if nonCombatant.Character.HasBuff(84) {
		t.Error("non-combatant mob 202 must NOT be stunned")
	}
}

// TestProcAoeStun_SkipsCharmedCompanions proves charmed mobs are spared no
// matter whose companions they are — the owner's AND a non-party bystander's
// (player-cast HarmArea parity: bare IsCharmed()). Users 1 and 2 are seeded
// by seedAllRegistries with no party, so user 2 is a non-party bystander.
func TestProcAoeStun_SkipsCharmedCompanions(t *testing.T) {
	defer seedAllRegistries()()
	defer seedStunBuff(t)()
	enableItemProcs(t)

	hostile := mobs.GetInstance(100)
	ownerCompanion := addTestMob(203, false)
	ownerCompanion.Character.Charm(1, characters.CharmPermanent, "")
	bystanderCompanion := addTestMob(204, false)
	bystanderCompanion.Character.Charm(2, characters.CharmPermanent, "")

	owner := users.GetByUserId(1).Character
	owner.SetUserId(1)

	if ok := procAoeStun(owner, rooms.LoadRoom(1), map[string]float64{}); !ok {
		t.Fatal("aoe_stun should execute with a hostile present")
	}
	if !hostile.Character.HasBuff(84) {
		t.Error("hostile mob 100 should be stunned")
	}
	if ownerCompanion.Character.HasBuff(84) {
		t.Error("owner-charmed companion 203 must NOT be stunned")
	}
	if bystanderCompanion.Character.HasBuff(84) {
		t.Error("non-party bystander's companion 204 must NOT be stunned")
	}
}

// TestProcAoeStun_EmptyRoomReturnsFalse proves an empty room does not execute
// (so the caller does not burn the proc's cooldown).
func TestProcAoeStun_EmptyRoomReturnsFalse(t *testing.T) {
	defer seedAllRegistries()()
	defer seedStunBuff(t)()
	enableItemProcs(t)

	owner := users.GetByUserId(1).Character
	owner.SetUserId(1)
	owner.RoomId = 2 // room 2 has no mobs

	if ok := procAoeStun(owner, rooms.LoadRoom(2), map[string]float64{}); ok {
		t.Fatal("aoe_stun in an empty room should return false (no cooldown burn)")
	}
}

// TestProcAoeStun_MobOwnerIsNoOp proves a mob owner (GetUserId() == 0) stuns
// nothing and returns false — no Stage-2 mob wields an aoe_stun item.
func TestProcAoeStun_MobOwnerIsNoOp(t *testing.T) {
	defer seedAllRegistries()()
	defer seedStunBuff(t)()
	enableItemProcs(t)

	hostile := mobs.GetInstance(100)
	mobOwner := characters.New() // no userId assigned → GetUserId() == 0

	if ok := procAoeStun(mobOwner, rooms.LoadRoom(1), map[string]float64{}); ok {
		t.Fatal("aoe_stun from a mob owner should return false")
	}
	if hostile.Character.HasBuff(84) {
		t.Error("mob-owner aoe_stun must not stun anything")
	}
}

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
func TestProcApplyCondition_Bleed(t *testing.T) {
	defer seedAllRegistries()()
	target := characters.New()

	ok := procApplyCondition(target, map[string]float64{
		"condition": 1, // 1 = bleeding
		"duration":  6,
		"magnitude": 12,
	})
	if !ok {
		t.Fatal("apply_condition should execute")
	}
	if !target.HasCondition(characters.ConditionBleeding) {
		t.Fatal("target should be bleeding")
	}
	if got := target.GetConditionDuration(characters.ConditionBleeding); got != 6 {
		t.Fatalf("expected duration 6, got %d", got)
	}
	if got := target.GetConditionMagnitude(characters.ConditionBleeding); got != 12 {
		t.Fatalf("expected magnitude 12, got %v", got)
	}
}

func TestProcApplyCondition_NilAndUnknown(t *testing.T) {
	defer seedAllRegistries()()
	if procApplyCondition(nil, map[string]float64{"condition": 1}) {
		t.Fatal("nil target must not execute")
	}
	target := characters.New()
	if procApplyCondition(target, map[string]float64{"condition": 99}) {
		t.Fatal("unknown condition id must not execute (cooldown must not burn)")
	}
}

// TestDispatchOnGrappleProcs_Bleed proves the on_grapple dispatch reads the
// OTHER party's body armor (per procBearingItems' slot narrowing) and applies
// bleed to the opponent — mirrors TestDispatchOnHitProcs_Lifesteal.
func TestDispatchOnGrappleProcs_Bleed(t *testing.T) {
	defer seedAllRegistries()()
	enableItemProcs(t)
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999921: {ItemId: 999921, Name: "spiked harness", Type: items.Body,
			Procs: []items.ItemProc{{Trigger: "on_grapple", Chance: 100, Effect: "apply_condition",
				Params: map[string]float64{"condition": 1, "duration": 6, "magnitude": 12}}}},
	})()

	wearer := characters.New()
	wearer.Equipment.Body = items.New(999921)
	opponent := characters.New()

	dispatchItemProcs("on_grapple", wearer, opponent, nil, 0)

	if !opponent.HasCondition(characters.ConditionBleeding) {
		t.Fatal("on_grapple apply_condition expected the opponent to be bleeding")
	}
}

func TestProcStealPool_Conviction(t *testing.T) {
	defer seedAllRegistries()()
	caster := characters.New()
	caster.ConvictionMax.Value = 100
	caster.Conviction = 40
	target := characters.New()
	target.ConvictionMax.Value = 100
	target.Conviction = 50

	ok := procStealPool(caster, target, map[string]float64{"pool": 3, "amount_pct": 0.10})
	if !ok {
		t.Fatal("steal_pool should execute")
	}
	if target.Conviction != 40 { // 10% of target's max = 10 stolen
		t.Fatalf("target conviction expected 40, got %d", target.Conviction)
	}
	if caster.Conviction != 50 {
		t.Fatalf("caster conviction expected 50, got %d", caster.Conviction)
	}

	// clamps: target at 0, caster at max
	target.Conviction = 3
	caster.Conviction = 95
	procStealPool(caster, target, map[string]float64{"pool": 3, "amount_pct": 0.10})
	if target.Conviction != 0 || caster.Conviction != 98 {
		t.Fatalf("clamps wrong: target=%d caster=%d", target.Conviction, caster.Conviction)
	}
}

func TestProcStealPool_Guards(t *testing.T) {
	defer seedAllRegistries()()
	caster := characters.New()
	target := characters.New()
	if procStealPool(nil, target, map[string]float64{"pool": 3, "amount_pct": 0.1}) {
		t.Fatal("nil owner must not execute")
	}
	if procStealPool(caster, nil, map[string]float64{"pool": 3, "amount_pct": 0.1}) {
		t.Fatal("nil target must not execute")
	}
	if procStealPool(caster, target, map[string]float64{"pool": 3}) {
		t.Fatal("missing amount_pct must not execute")
	}
	target.Conviction = 0
	target.ConvictionMax.Value = 100
	if procStealPool(caster, target, map[string]float64{"pool": 3, "amount_pct": 0.1}) {
		t.Fatal("empty target pool must not execute (no cooldown burn)")
	}
	if procStealPool(caster, target, map[string]float64{"pool": 2, "amount_pct": 0.1}) {
		t.Fatal("unimplemented pool ids must not execute")
	}
}

// TestDispatchOnSpellHitProcs_StealPool proves the on_spell_hit dispatch reads
// the caster's WEAPON slot (per procBearingItems) and drains the target's
// conviction into the caster — the Staff of the Hollow Choir's sustain loop.
func TestDispatchOnSpellHitProcs_StealPool(t *testing.T) {
	defer seedAllRegistries()()
	enableItemProcs(t)
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999922: {ItemId: 999922, Name: "hollow staff", Type: items.Weapon, Hands: 2,
			Procs: []items.ItemProc{{Trigger: "on_spell_hit", Chance: 100, Effect: "steal_pool",
				Params: map[string]float64{"pool": 3, "amount_pct": 0.10}}}},
	})()

	caster := characters.New()
	caster.ConvictionMax.Value = 100
	caster.Conviction = 40
	caster.Equipment.Weapon = items.New(999922)
	target := characters.New()
	target.ConvictionMax.Value = 100
	target.Conviction = 50

	dispatchItemProcs("on_spell_hit", caster, target, nil, 25)

	if target.Conviction != 40 {
		t.Fatalf("on_spell_hit steal_pool expected target conviction 40, got %d", target.Conviction)
	}
	if caster.Conviction != 50 {
		t.Fatalf("on_spell_hit steal_pool expected caster conviction 50, got %d", caster.Conviction)
	}
}

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
