package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
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
