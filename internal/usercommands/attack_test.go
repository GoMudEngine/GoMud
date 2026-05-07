package usercommands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
)

// TestAttack_PlayerAttackImmune_RebuffsAttack verifies that a mob with
// PlayerAttackImmune: true cannot be attacked by players — aggro must not be
// set on the user after attacking such a mob.
func TestAttack_PlayerAttackImmune_RebuffsAttack(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)

	// Seed a PlayerAttackImmune mob instance (NonCombatant: false so it can fight).
	immuneMob := &mobs.Mob{
		MobId:              5,
		InstanceId:         200,
		HomeRoomId:         1,
		Hostile:            true,
		NonCombatant:       false,
		PlayerAttackImmune: true,
		Character: characters.Character{
			Name:   "Caravan Guard",
			RoomId: 1,
			Health: 100,
			Buffs:  buffs.New(),
		},
	}
	immuneMob.Character.HealthMax.Value = 100
	mobs.SetInstanceForTest(200, immuneMob)
	defer mobs.SetInstanceForTest(200, nil)
	room.AddMob(200)
	defer room.RemoveMob(200)

	// Ensure no aggro before the attack.
	assert.Nil(t, user.Character.Aggro, "user should start with no aggro")

	handled, err := Attack("caravan guard", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)

	// The attack must be rebuffed — aggro must NOT be set.
	assert.Nil(t, user.Character.Aggro, "attack on PlayerAttackImmune mob must not set aggro")
}

func TestAttack_NonCombatantFiresEventOnce_ThenDedupes(t *testing.T) {
	originalTry := mobs.AttackRejectedTryMobBehavior
	var fireCount int
	mobs.AttackRejectedTryMobBehavior = func(instanceId int, ctx mobs.EventContext) bool {
		if ctx.EventType == "player_attack_rejected" {
			fireCount++
		}
		return true
	}
	defer func() { mobs.AttackRejectedTryMobBehavior = originalTry }()

	mob := &mobs.Mob{
		InstanceId: 12001,
		Character: characters.Character{
			RoomId: 1,
			Name:   "Test NPC",
		},
		NonCombatant: true,
	}
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{12001: mob})
	defer cleanup()

	util.SetRoundCountForTest(100)
	defer util.ResetRoundCountForTest()

	mobs.FireAttackRejected(mob, 42)
	mobs.FireAttackRejected(mob, 42)
	assert.Equal(t, 1, fireCount, "second call in same round should be deduped")

	util.SetRoundCountForTest(101)
	mobs.FireAttackRejected(mob, 42)
	assert.Equal(t, 2, fireCount, "call in a new round should fire again")
}

func TestAttackBumpsOpinion(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	user, room := getTestUserAndRoom(t)

	// Seed a fresh attackable mob instance backed by template mobId=1 (Skeleton).
	target := &mobs.Mob{
		MobId:      1,
		InstanceId: 250,
		HomeRoomId: 1,
		Hostile:    false,
		Character: characters.Character{
			Name:      "Skeleton",
			RoomId:    1,
			Health:    50,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
		},
	}
	target.Character.HealthMax.Value = 100
	mobs.SetInstanceForTest(250, target)
	defer mobs.SetInstanceForTest(250, nil)
	room.AddMob(250)
	defer room.RemoveMob(250)

	// Sanity: opinion starts at default (0 for unconfigured mob 1).
	if got := opinions.Get(1, user.UserId); got != 0 {
		t.Fatalf("pre-attack opinion = %d, want 0", got)
	}

	handled, err := Attack("#250", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)

	want := int(configs.GetBalanceConfig().OpinionAttackBump)
	if got := opinions.Get(1, user.UserId); got != want {
		t.Errorf("post-attack opinion = %d, want %d", got, want)
	}
}

func TestAttackOnSameTargetDoesNotDoubleBump(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	user, room := getTestUserAndRoom(t)

	target := &mobs.Mob{
		MobId:      1,
		InstanceId: 251,
		HomeRoomId: 1,
		Hostile:    false,
		Character: characters.Character{
			Name:      "Skeleton",
			RoomId:    1,
			Health:    50,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
		},
	}
	target.Character.HealthMax.Value = 100
	mobs.SetInstanceForTest(251, target)
	defer mobs.SetInstanceForTest(251, nil)
	room.AddMob(251)
	defer room.RemoveMob(251)

	if _, err := Attack("#251", user, room, 0); err != nil {
		t.Fatalf("first Attack: %v", err)
	}
	first := opinions.Get(1, user.UserId)

	// Second invocation — already aggroed on same target, should be a no-op.
	if _, err := Attack("#251", user, room, 0); err != nil {
		t.Fatalf("second Attack: %v", err)
	}
	second := opinions.Get(1, user.UserId)
	if second != first {
		t.Errorf("re-attack opinion changed: was %d, now %d (want stable)", first, second)
	}
}

// TestTargetSwitchBumpsNewMobOpinion verifies that switching combat
// targets via `attack #other` (which delegates to Target) also fires
// a fresh-aggression bump against the new mob's template. Without
// this hook, a player attacking mob A then switching to mob B would
// only bump A's opinion of them, not B's — an asymmetry the substrate
// shouldn't allow.
func TestTargetSwitchBumpsNewMobOpinion(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	user, room := getTestUserAndRoom(t)

	// Two mobs, different templates: Skeleton (mobId 1) and Merchant (mobId 2).
	mobA := &mobs.Mob{
		MobId:      1,
		InstanceId: 260,
		HomeRoomId: 1,
		Character: characters.Character{
			Name: "Skeleton", RoomId: 1, Health: 50,
			Buffs: buffs.New(), Cooldowns: map[string]int{},
		},
	}
	mobA.Character.HealthMax.Value = 100
	mobs.SetInstanceForTest(260, mobA)
	defer mobs.SetInstanceForTest(260, nil)
	room.AddMob(260)
	defer room.RemoveMob(260)

	mobB := &mobs.Mob{
		MobId:      2,
		InstanceId: 261,
		HomeRoomId: 1,
		Character: characters.Character{
			Name: "Merchant", RoomId: 1, Health: 50,
			Buffs: buffs.New(), Cooldowns: map[string]int{},
		},
	}
	mobB.Character.HealthMax.Value = 100
	mobs.SetInstanceForTest(261, mobB)
	defer mobs.SetInstanceForTest(261, nil)
	room.AddMob(261)
	defer room.RemoveMob(261)

	// Attack mobA first — bumps Skeleton (mobId 1).
	if _, err := Attack("#260", user, room, 0); err != nil {
		t.Fatalf("first Attack: %v", err)
	}
	bump := int(configs.GetBalanceConfig().OpinionAttackBump)
	if got := opinions.Get(1, user.UserId); got != bump {
		t.Fatalf("after attack on Skeleton, Get(1) = %d, want %d", got, bump)
	}

	// Kill mobA so the target switch goes through the free-switch path
	// (no skill roll). We're testing the opinion bump, not the switch logic.
	mobA.Character.Health = 0

	// Attack mobB — delegates to Target (already aggroed on mobA), which
	// should bump Merchant (mobId 2).
	if _, err := Attack("#261", user, room, 0); err != nil {
		t.Fatalf("target-switch Attack: %v", err)
	}
	if got := opinions.Get(2, user.UserId); got != bump {
		t.Errorf("after switch to Merchant, Get(2) = %d, want %d", got, bump)
	}
	// And mobA's opinion is unchanged — the switch shouldn't double-bump A.
	if got := opinions.Get(1, user.UserId); got != bump {
		t.Errorf("after switch, Skeleton opinion changed unexpectedly: %d, want %d", got, bump)
	}
}

func TestAttackRecordsAssaultCrimeOnFactionMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE", t.TempDir())
	if err := os.WriteFile(filepath.Join(dir, "thornwall_citizens.yaml"),
		[]byte(`faction_id: thornwall_citizens
display_name: "Citizens"
description: "x"
default_rep: 0
allies: []
enemies: []
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := factions.LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	factions.ClearCache()
	crimes.ClearCache()
	opinions.ClearCache()

	user, room := getTestUserAndRoom(t)

	// Seed a citizen mob in the room.
	target := &mobs.Mob{
		MobId:      100,
		InstanceId: 300,
		HomeRoomId: 1,
		Hostile:    false,
		Groups:     []string{"thornwall_citizens"},
		Character: characters.Character{
			Name:      "city beggar",
			RoomId:    1,
			Health:    50,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
		},
	}
	target.Character.HealthMax.Value = 100
	mobs.SetInstanceForTest(300, target)
	defer mobs.SetInstanceForTest(300, nil)
	room.AddMob(300)
	defer room.RemoveMob(300)

	if _, err := Attack("#300", user, room, 0); err != nil {
		t.Fatalf("Attack: %v", err)
	}

	got := crimes.AllForFaction("thornwall_citizens", false)
	if len(got) != 1 {
		t.Fatalf("expected 1 assault crime, got %d", len(got))
	}
	if got[0].Kind != crimes.KindAssault {
		t.Errorf("kind: %s", got[0].Kind)
	}
	if got[0].Perpetrator.Type != crimes.PerpPlayer || got[0].Perpetrator.Id != user.UserId {
		t.Errorf("perpetrator: %+v", got[0].Perpetrator)
	}

	wantRep := int(configs.GetBalanceConfig().CrimeRepDeltaAssault)
	if rep := factions.GetRep("thornwall_citizens", user.UserId); rep != wantRep {
		t.Errorf("rep: %d, want %d", rep, wantRep)
	}
}
