package goals

import (
	"fmt"
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

func TestAdd_HappyPath(t *testing.T) {
	ClearCache()
	resetRegistry()
	g := &Goal{Type: "revenge", Priority: 70, Params: map[string]any{"k": "v"}}
	res, err := Add(99003, "addmob", g)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if res.Added == nil || res.Added.Id != "g1" {
		t.Errorf("expected Id=g1, got %+v", res.Added)
	}
	if res.Added.OwnerMobId != 99003 {
		t.Errorf("OwnerMobId not stamped: got %d", res.Added.OwnerMobId)
	}
	if res.Added.CreatedAt.IsZero() {
		t.Error("CreatedAt not stamped")
	}
	if len(res.Displaced) != 0 {
		t.Errorf("expected no displacements, got %v", res.Displaced)
	}
	got := GoalsOf(99003, "addmob")
	if len(got) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(got))
	}
}

func TestAdd_AssignsSequentialIds(t *testing.T) {
	ClearCache()
	resetRegistry()
	for i, want := range []string{"g1", "g2", "g3"} {
		res, err := Add(99004, "seqmob", &Goal{
			Type:     fmt.Sprintf("type%d", i),
			Priority: 50 - i, // descending so no conflicts within the test
		})
		if err != nil {
			t.Fatalf("Add %d: %v", i, err)
		}
		if res.Added.Id != want {
			t.Errorf("Add %d: id=%q, want %q", i, res.Added.Id, want)
		}
	}
}

func TestAdd_PersistsAcrossClearCache(t *testing.T) {
	ClearCache()
	resetRegistry()
	if _, err := Add(99005, "persistmob", &Goal{Type: "alpha", Priority: 50}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	ClearCache()
	got := GoalsOf(99005, "persistmob")
	if len(got) != 1 || got[0].Type != "alpha" {
		t.Fatalf("did not persist: %+v", got)
	}
}

