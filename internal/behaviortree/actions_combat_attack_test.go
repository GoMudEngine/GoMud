package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TestActAttack_MobAttacker_TargetsAttackerNotRandomPlayer is the
// regression test for the caravan-leader-aggros-player bug surfaced
// during the chunk 3.7 smoke (commit b7114767):
//
// When mob_hurt fires with ctx.Event.MobId set (a mob attacked the
// defender), actAttack must set Aggro on the attacking MOB — not
// fall back to "pick a random player in the room" via Event.UserId
// == 0. The bug caused Ketil's caravan to aggro players following
// it whenever a bandit lookout ambushed the crew.
//
// Pre-fix: actAttack ignored Event.MobId and either returned
// Failure (empty room) or hit a random player (room had a player).
// Post-fix: actAttack honors Event.MobId and sets Aggro on the
// actual attacker mob.
func TestActAttack_MobAttacker_TargetsAttackerNotRandomPlayer(t *testing.T) {
	// Defender mob (the one running the behavior tree).
	defender := &mobs.Mob{
		MobId:      mobs.MobId(357),
		InstanceId: 91000,
	}
	defender.Character.Name = "ketil"
	defender.Character.RoomId = 10001
	defender.Character.Health = 100
	defender.Character.Buffs = buffs.New()

	cleanup := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{357: defender},
		map[int]*mobs.Mob{91000: defender},
	)
	defer cleanup()

	// Event: a mob attacked us. UserId is 0; MobId is the attacker.
	const attackerInstanceId = 92000
	ctx := &EvalContext{
		InstanceId: defender.InstanceId,
		RoomId:     defender.Character.RoomId,
		Event: EventContext{
			EventType: "mob_hurt",
			UserId:    0,
			MobId:     attackerInstanceId,
			RoomId:    defender.Character.RoomId,
		},
	}

	result := actAttack(nil, ctx)
	if result != Success {
		t.Fatalf("actAttack returned %v, want Success when Event.MobId is set", result)
	}
	if defender.Character.Aggro == nil {
		t.Fatal("expected Aggro to be set; got nil")
	}
	if defender.Character.Aggro.UserId != 0 {
		t.Errorf("Aggro.UserId = %d, want 0 (no player should be targeted)", defender.Character.Aggro.UserId)
	}
	if defender.Character.Aggro.MobInstanceId != attackerInstanceId {
		t.Errorf("Aggro.MobInstanceId = %d, want %d (the attacking mob)",
			defender.Character.Aggro.MobInstanceId, attackerInstanceId)
	}
}
