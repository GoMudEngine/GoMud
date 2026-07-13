package hooks_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	_ "github.com/GoMudEngine/GoMud/internal/hooks" // wire init() Life observers
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// TestDie_PlayerRespawns_NotStuckDowned is the regression guard for the
// new-player "downed" soft-lock (2026-07-13). A player driven to Health<=0
// and killed via Die() MUST complete the respawn cascade (Dead → Respawning →
// Alive) and come back with positive HP — never get stranded non-Alive at
// Health<=0, which re-exposes the vestigial IsDisabled()/"while downed" gate
// and bricks every command.
//
// The bug: users.CreateUser assigned UserRecord.UserId but never seeded the
// Character's private userId mirror, so a brand-new character played its whole
// first session with GetUserId()==0. Die() treats GetUserId()==0 as a mob and
// stops at Dead — so a fresh player who died before ever re-logging never
// respawned. This test locks the invariant at the death path.
func TestDie_PlayerRespawns_NotStuckDowned(t *testing.T) {
	c := characters.New()
	c.SetUserId(4242) // a properly-seeded player (post-CreateUser fix)
	c.HealthMax.Value = 100
	c.Health = 0 // combat dropped them to zero this round

	c.Die(state.ActorRef{}, life.TriggerHealthZero)

	if !c.IsAlive() {
		t.Fatalf("a player killed at 0 HP must respawn to Alive; stuck in Life state %v (soft-lock)", c.Life.State())
	}
	if c.Health <= 0 {
		t.Errorf("a respawned player must have positive HP; got %d", c.Health)
	}
}

// TestDie_ZeroUserIdStaysDead documents the exact mechanism the CreateUser fix
// prevents: a character whose userId is 0 is indistinguishable from a mob at
// the Die() branch and is intentionally left at Dead (mobs despawn from there).
// This is precisely why a *player* must always have a non-zero seeded userId —
// otherwise their death is a permanent soft-lock. If the Die() player/mob
// discriminator ever changes, revisit the new-player death path.
func TestDie_ZeroUserIdStaysDead(t *testing.T) {
	c := characters.New()
	// userId intentionally left 0 (the pre-fix new-character state).
	c.HealthMax.Value = 100
	c.Health = 0

	c.Die(state.ActorRef{}, life.TriggerHealthZero)

	if c.IsAlive() {
		t.Skip("Die() now revives userId-0 characters; the soft-lock discriminator changed — re-audit new-player death")
	}
	if !c.IsDead() {
		t.Errorf("a userId-0 character should remain Dead after Die(); got %v", c.Life.State())
	}
}
