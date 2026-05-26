package goals

import (
	"errors"
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

func TestAdd_SameTypeBlockedByHigherOrEqualPriority(t *testing.T) {
	ClearCache()
	resetRegistry()
	if _, err := Add(99006, "conflict1", &Goal{Type: "revenge", Priority: 70}); err != nil {
		t.Fatal(err)
	}
	_, err := Add(99006, "conflict1", &Goal{Type: "revenge", Priority: 70})
	var ce *ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConflictError, got %v", err)
	}
	if ce.BlockerGoalId != "g1" || ce.BlockerType != "revenge" || ce.BlockerPrio != 70 {
		t.Errorf("ConflictError fields wrong: %+v", ce)
	}
	if len(GoalsOf(99006, "conflict1")) != 1 {
		t.Error("blocked Add should not have appended")
	}
}

func TestAdd_SameTypeDisplacesLowerPriority(t *testing.T) {
	ClearCache()
	resetRegistry()
	if _, err := Add(99007, "conflict2", &Goal{Type: "revenge", Priority: 30}); err != nil {
		t.Fatal(err)
	}
	res, err := Add(99007, "conflict2", &Goal{Type: "revenge", Priority: 70})
	if err != nil {
		t.Fatalf("higher-prio Add: %v", err)
	}
	if len(res.Displaced) != 1 || res.Displaced[0] != "g1" {
		t.Errorf("expected displaced=[g1], got %v", res.Displaced)
	}
	got := GoalsOf(99007, "conflict2")
	if len(got) != 1 || got[0].Id != "g2" || got[0].Priority != 70 {
		t.Errorf("after displacement, expected single g2 priority=70; got %+v", got)
	}
}

func TestAdd_CrossTypeConflict_BothDirectionsDeclared(t *testing.T) {
	ClearCache()
	resetRegistry()
	RegisterGoalType("revenge", GoalTypeMeta{ConflictsWith: []string{"protection"}})
	RegisterGoalType("protection", GoalTypeMeta{ConflictsWith: []string{"revenge"}})

	if _, err := Add(99008, "crossA", &Goal{Type: "revenge", Priority: 50}); err != nil {
		t.Fatal(err)
	}
	_, err := Add(99008, "crossA", &Goal{Type: "protection", Priority: 40})
	var ce *ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConflictError, got %v", err)
	}
}

func TestAdd_CrossTypeConflict_SymmetrySafetyNet(t *testing.T) {
	// "revenge" declares protection as a conflict, but "protection"
	// does NOT declare revenge. Adding protection while revenge exists
	// should still be blocked, because the existing type's metadata
	// gets checked for the reverse edge by the safety net.
	//
	// Note: in this scenario the safety net path triggers because the
	// EXISTING goal type (revenge) has the declaration, and the NEW
	// goal type (protection) does not. isConflict's third branch (look
	// up the existing type's meta and check its ConflictsWith for the
	// new type) catches this.
	ClearCache()
	resetRegistry()
	RegisterGoalType("revenge", GoalTypeMeta{ConflictsWith: []string{"protection"}})
	RegisterGoalType("protection", GoalTypeMeta{ConflictsWith: nil})

	if _, err := Add(99009, "crossB", &Goal{Type: "revenge", Priority: 50}); err != nil {
		t.Fatal(err)
	}
	_, err := Add(99009, "crossB", &Goal{Type: "protection", Priority: 40})
	var ce *ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConflictError via symmetry safety net, got %v", err)
	}
}

func TestAdd_DisplacesMultipleConflicts(t *testing.T) {
	ClearCache()
	resetRegistry()
	RegisterGoalType("a", GoalTypeMeta{ConflictsWith: []string{"b", "c"}})
	RegisterGoalType("b", GoalTypeMeta{ConflictsWith: []string{"a"}})
	RegisterGoalType("c", GoalTypeMeta{ConflictsWith: []string{"a"}})

	if _, err := Add(99010, "multi", &Goal{Type: "b", Priority: 30}); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(99010, "multi", &Goal{Type: "c", Priority: 40}); err != nil {
		t.Fatal(err)
	}
	res, err := Add(99010, "multi", &Goal{Type: "a", Priority: 70})
	if err != nil {
		t.Fatalf("Add a: %v", err)
	}
	if len(res.Displaced) != 2 {
		t.Errorf("expected 2 displaced, got %v", res.Displaced)
	}
	got := GoalsOf(99010, "multi")
	if len(got) != 1 || got[0].Type != "a" {
		t.Errorf("expected only type=a remaining, got %+v", got)
	}
}

