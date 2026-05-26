package goals

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestGoalsOf_EmptyWhenNothingPersisted(t *testing.T) {
	ClearCache()
	resetRegistry()
	if got := GoalsOf(99001, "ghost"); len(got) != 0 {
		t.Errorf("expected empty, got %d goals", len(got))
	}
}

func TestGoalsOf_LazyLoadFromDisk(t *testing.T) {
	ClearCache()
	resetRegistry()
	mg := &MobGoals{
		MobId:      99002,
		NextGoalId: 2,
		Goals: []*Goal{
			{Id: "g1", Type: "revenge", Priority: 70, CreatedAt: time.Now().UTC()},
		},
	}
	cacheStoreForTest("disk-mob", mg)
	if err := saveToDisk(99002, "disk-mob"); err != nil {
		t.Fatalf("save: %v", err)
	}
	ClearCache()
	got := GoalsOf(99002, "disk-mob")
	if len(got) != 1 {
		t.Fatalf("expected 1 goal after lazy load, got %d", len(got))
	}
	if got[0].OwnerMobId != 99002 {
		t.Errorf("OwnerMobId not stamped on load: got %d", got[0].OwnerMobId)
	}
}

func TestIsSatisfied_NoPredicateReturnsFalse(t *testing.T) {
	resetRegistry()
	g := &Goal{Id: "g1", Type: "noexist", OwnerMobId: 1}
	if IsSatisfied(g, nil) {
		t.Error("expected false when no predicate registered")
	}
}

func TestIsSatisfied_InvokesRegisteredPredicate(t *testing.T) {
	resetRegistry()
	called := false
	RegisterGoalType("pingtest", GoalTypeMeta{
		Predicate: func(g *Goal, m *mobs.Mob) bool {
			called = true
			return true
		},
	})
	g := &Goal{Id: "g1", Type: "pingtest", OwnerMobId: 1}
	if !IsSatisfied(g, nil) {
		t.Error("expected true from predicate")
	}
	if !called {
		t.Error("predicate was not invoked")
	}
}

func TestIsExpired_ZeroNeverExpires(t *testing.T) {
	g := &Goal{Id: "g1"}
	if IsExpired(g, time.Now()) {
		t.Error("zero ExpiresAt should never expire")
	}
}

func TestIsExpired_PastTime(t *testing.T) {
	g := &Goal{Id: "g1", ExpiresAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
	if !IsExpired(g, time.Now()) {
		t.Error("expected expired")
	}
}

func TestIsExpired_FutureTime(t *testing.T) {
	g := &Goal{Id: "g1", ExpiresAt: time.Now().Add(time.Hour)}
	if IsExpired(g, time.Now()) {
		t.Error("expected not expired")
	}
}

