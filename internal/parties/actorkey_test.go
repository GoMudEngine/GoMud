package parties

import "testing"

// fakeActor satisfies actorIdentity (the minimal local interface used by
// ActorKeyFor). No import of internal/actions is needed — and is intentionally
// avoided because internal/actions already imports internal/parties, which
// would create an import cycle.
type fakeActor struct {
	userId        int
	mobInstanceId int
}

var _ actorIdentity = (*fakeActor)(nil)

func (f *fakeActor) IsPlayer() bool        { return f.userId > 0 }
func (f *fakeActor) GetUserId() int        { return f.userId }
func (f *fakeActor) GetMobInstanceId() int { return f.mobInstanceId }

func TestActorKey_PlayerActor(t *testing.T) {
	a := &fakeActor{userId: 42}
	key := ActorKeyFor(a)
	if string(key) != "user:42" {
		t.Errorf("got %q, want %q", key, "user:42")
	}
}

func TestActorKey_MobActor(t *testing.T) {
	a := &fakeActor{mobInstanceId: 1234}
	key := ActorKeyFor(a)
	if string(key) != "mob:1234" {
		t.Errorf("got %q, want %q", key, "mob:1234")
	}
}

func TestActorKey_DistinctForUserAndMob(t *testing.T) {
	user := &fakeActor{userId: 1}
	mob := &fakeActor{mobInstanceId: 1}
	if ActorKeyFor(user) == ActorKeyFor(mob) {
		t.Error("user and mob with same numeric id should produce distinct keys")
	}
}

func TestActorKey_NilActor(t *testing.T) {
	key := ActorKeyFor(nil)
	if key != "" {
		t.Errorf("nil actor should yield empty key, got %q", key)
	}
}
