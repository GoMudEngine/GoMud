package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
)

func TestAttributeBountyKill(t *testing.T) {
	// guard mob of the issuer faction landed the kill
	gk := attributeBountyKill(
		state.ActorRef{MobInstanceId: 5},
		"thornwall_guards", nil,
		func(int) []string { return []string{"thornwall_guards"} },
	)
	if gk.kind != killGuard || gk.mobInstanceId != 5 {
		t.Errorf("guard kill: got %+v", gk)
	}
	// third-party player landed the kill
	pk := attributeBountyKill(state.ActorRef{UserId: 42}, "thornwall_guards", nil, nil)
	if pk.kind != killPlayer || pk.userId != 42 {
		t.Errorf("player kill: got %+v", pk)
	}
	// non-issuer mob, but a player damager exists -> player attribution
	dm := attributeBountyKill(
		state.ActorRef{MobInstanceId: 7},
		"thornwall_guards", map[int]int{42: 10},
		func(int) []string { return []string{"warren"} },
	)
	if dm.kind != killPlayer || dm.userId != 42 {
		t.Errorf("damager fallback: got %+v", dm)
	}
	// nobody eligible -> expire
	ex := attributeBountyKill(state.ActorRef{}, "thornwall_guards", nil, nil)
	if ex.kind != killNone {
		t.Errorf("none: got %+v", ex)
	}
}
