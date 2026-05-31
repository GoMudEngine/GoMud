package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestAttributeBountyKill(t *testing.T) {
	const target = 17 // the dead, wanted player

	// guard mob of the issuer faction landed the kill
	gk := attributeBountyKill(
		state.ActorRef{MobInstanceId: 5},
		target, "thornwall_guards", nil,
		func(int) []string { return []string{"thornwall_guards"} },
	)
	if gk.kind != killGuard || gk.mobInstanceId != 5 {
		t.Errorf("guard kill: got %+v", gk)
	}
	// third-party player landed the kill
	pk := attributeBountyKill(state.ActorRef{UserId: 42}, target, "thornwall_guards", nil, nil)
	if pk.kind != killPlayer || pk.userId != 42 {
		t.Errorf("player kill: got %+v", pk)
	}
	// non-issuer mob, but a third-party player damager exists -> player attribution
	dm := attributeBountyKill(
		state.ActorRef{MobInstanceId: 7},
		target, "thornwall_guards", map[int]int{42: 10},
		func(int) []string { return []string{"warren"} },
	)
	if dm.kind != killPlayer || dm.userId != 42 {
		t.Errorf("damager fallback: got %+v", dm)
	}
	// nobody eligible -> expire
	ex := attributeBountyKill(state.ActorRef{}, target, "thornwall_guards", nil, nil)
	if ex.kind != killNone {
		t.Errorf("none: got %+v", ex)
	}

	// SELF-KILL (suicide): the dead player is their own killer. They must NOT
	// collect their own bounty — exploit guard. Regression for the 5.1b smoke
	// BUG-1 (suicide paid the dying player their own bounties).
	self := attributeBountyKill(state.ActorRef{UserId: target}, target, "thornwall_guards", nil, nil)
	if self.kind != killNone {
		t.Errorf("self-kill must not pay: got %+v", self)
	}

	// Self-damage only in the damage map (player solely damaged themselves)
	// must also be non-attributable.
	selfDmg := attributeBountyKill(
		state.ActorRef{}, target, "thornwall_guards", map[int]int{target: 50}, nil,
	)
	if selfDmg.kind != killNone {
		t.Errorf("self-damage-only must not pay: got %+v", selfDmg)
	}

	// Mixed damage: a third-party out-damaged by the target still wins (only
	// the target is excluded, not other players).
	mixed := attributeBountyKill(
		state.ActorRef{}, target, "thornwall_guards", map[int]int{target: 100, 42: 10}, nil,
	)
	if mixed.kind != killPlayer || mixed.userId != 42 {
		t.Errorf("third-party should win once target is excluded: got %+v", mixed)
	}
}

// TestKillGuard_ClearsFactionRecord verifies that when the killGuard branch
// fires for a faction-issued bounty and TryClaim succeeds,
// clearFactionRecordFn is called with the issuing faction and the target's
// userId. This is the 5.2 "death pays the debt" path.
func TestKillGuard_ClearsFactionRecord(t *testing.T) {
	const (
		targetUserId  = 55
		hunterInstId  = 7
		bountyId      = 101
		issuerFaction = "thornwall_guards"
	)

	// Save and restore all seams.
	origOpen := openAgainstPlayerFn
	origGetMob := getMobInstanceFn
	origTryClaim := tryClaimFn
	origMarkExpired := markExpiredFn
	origClearRecord := clearFactionRecordFn
	origKillerFactions := killerMobFactionsFn
	t.Cleanup(func() {
		openAgainstPlayerFn = origOpen
		getMobInstanceFn = origGetMob
		tryClaimFn = origTryClaim
		markExpiredFn = origMarkExpired
		clearFactionRecordFn = origClearRecord
		killerMobFactionsFn = origKillerFactions
	})

	// Stub the bounty list: one faction-issued bounty against the target.
	openAgainstPlayerFn = func(userId int) []*bounties.Bounty {
		if userId != targetUserId {
			return nil
		}
		return []*bounties.Bounty{
			{
				Id:         bountyId,
				Issuer:     bounties.Issuer{Type: bounties.IssuerFaction, Id: issuerFaction},
				GoldReward: 50,
			},
		}
	}

	// Stub killerMobFactionsFn: the hunter instance belongs to the issuing faction.
	killerMobFactionsFn = func(instanceId int) []string {
		if instanceId == hunterInstId {
			return []string{issuerFaction}
		}
		return nil
	}

	// Stub getMobInstanceFn: return a minimal mob for the hunter instance.
	hunterMob := &mobs.Mob{}
	hunterMob.MobId = 300
	getMobInstanceFn = func(instanceId int) *mobs.Mob {
		if instanceId == hunterInstId {
			return hunterMob
		}
		return nil
	}

	// Stub TryClaim: always succeed and echo the bounty back.
	tryClaimFn = func(id int, claimer knowledge.Subject) (*bounties.Bounty, bool) {
		return &bounties.Bounty{Id: id, GoldReward: 50}, true
	}

	// Stub markExpired — must NOT be called on a successful claim.
	expiredCalled := false
	markExpiredFn = func(id int) { expiredCalled = true }

	// Capture clearFactionRecordFn calls.
	type clearCall struct {
		faction string
		userId  int
		reason  string
	}
	var clearCalls []clearCall
	clearFactionRecordFn = func(faction string, userId int, reason string) {
		clearCalls = append(clearCalls, clearCall{faction, userId, reason})
	}

	// Build a real user + life machine for the target player.
	u := users.NewTestUser(targetUserId, "wanted", "Wanted", 5001)
	u.Character.Life = life.NewMachine()

	cleanupUsers := users.SeedUsersForTest(map[int]*users.UserRecord{targetUserId: u})
	defer cleanupUsers()

	// Wire the bounty-resolve observer.
	wirePlayerDeathBountyResolve(u.Character)

	// Transition to Dead: killer is the hunter mob (instance 7).
	err := u.Character.Life.TransitionToDead(
		life.DeadData{
			Killer: state.ActorRef{MobInstanceId: hunterInstId},
		},
		state.TransitionReason{Trigger: life.TriggerHealthZero},
	)
	if err != nil {
		t.Fatalf("TransitionToDead failed: %v", err)
	}

	// clearFactionRecordFn must be called once with the right args.
	if len(clearCalls) != 1 {
		t.Fatalf("clearFactionRecordFn called %d times, want 1; calls: %+v", len(clearCalls), clearCalls)
	}
	if clearCalls[0].faction != issuerFaction {
		t.Errorf("faction=%q want %q", clearCalls[0].faction, issuerFaction)
	}
	if clearCalls[0].userId != targetUserId {
		t.Errorf("userId=%d want %d", clearCalls[0].userId, targetUserId)
	}
	if clearCalls[0].reason != "bounty claimed" {
		t.Errorf("reason=%q want %q", clearCalls[0].reason, "bounty claimed")
	}

	// markExpired must NOT have been called (claim succeeded).
	if expiredCalled {
		t.Error("markExpired must not be called when TryClaim succeeds")
	}
}
