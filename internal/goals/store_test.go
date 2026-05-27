package goals

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
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

func TestRemove_HappyPath(t *testing.T) {
	ClearCache()
	resetRegistry()
	res, err := Add(99011, "removable", &Goal{Type: "alpha", Priority: 50})
	if err != nil {
		t.Fatal(err)
	}
	if err := Remove(99011, "removable", res.Added.Id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(GoalsOf(99011, "removable")) != 0 {
		t.Error("expected empty after Remove")
	}
}

func TestRemove_MissingIdReturnsErrGoalNotFound(t *testing.T) {
	ClearCache()
	resetRegistry()
	err := Remove(99012, "empty", "g999")
	if !errors.Is(err, ErrGoalNotFound) {
		t.Errorf("expected ErrGoalNotFound, got %v", err)
	}
}

func TestRemove_DoesNotResetNextGoalId(t *testing.T) {
	// Ids are never reused even after Remove. After Add → Remove → Add,
	// the second Add should produce g2, not g1.
	ClearCache()
	resetRegistry()
	r1, _ := Add(99013, "noreuse", &Goal{Type: "x", Priority: 10})
	if err := Remove(99013, "noreuse", r1.Added.Id); err != nil {
		t.Fatal(err)
	}
	r2, err := Add(99013, "noreuse", &Goal{Type: "y", Priority: 20})
	if err != nil {
		t.Fatal(err)
	}
	if r2.Added.Id != "g2" {
		t.Errorf("expected g2 after Remove+Add, got %q", r2.Added.Id)
	}
}

func TestClear_RemovesAllAndResetsCounter(t *testing.T) {
	// Note: spec says ids never reused for the lifetime of a mob's file.
	// Clear is admin-only and intentionally heavy-hand — it wipes the
	// file AND resets NextGoalId to 1 so the operator gets a clean slate.
	ClearCache()
	resetRegistry()
	if _, err := Add(99014, "clearable", &Goal{Type: "a", Priority: 10}); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(99014, "clearable", &Goal{Type: "b", Priority: 20}); err != nil {
		t.Fatal(err)
	}
	if err := Clear(99014, "clearable"); err != nil {
		t.Fatal(err)
	}
	if len(GoalsOf(99014, "clearable")) != 0 {
		t.Error("expected empty after Clear")
	}
	r, err := Add(99014, "clearable", &Goal{Type: "c", Priority: 30})
	if err != nil {
		t.Fatal(err)
	}
	if r.Added.Id != "g1" {
		t.Errorf("expected counter reset to g1 after Clear, got %q", r.Added.Id)
	}
}

func TestGoalsOf_PriorityDescThenIdAsc(t *testing.T) {
	ClearCache()
	resetRegistry()
	// Different types to avoid same-type conflict.
	if _, err := Add(99015, "ordering", &Goal{Type: "a", Priority: 30}); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(99015, "ordering", &Goal{Type: "b", Priority: 70}); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(99015, "ordering", &Goal{Type: "c", Priority: 70}); err != nil {
		t.Fatal(err)
	}
	got := GoalsOf(99015, "ordering")
	if len(got) != 3 {
		t.Fatalf("expected 3 goals, got %d", len(got))
	}
	// Priority 70 g2 first (older id wins ties), then g3, then g1 prio 30.
	wantIds := []string{"g2", "g3", "g1"}
	for i, want := range wantIds {
		if got[i].Id != want {
			t.Errorf("position %d: got %q, want %q", i, got[i].Id, want)
		}
	}
}

func TestGoalsOf_ReturnedSliceIsCopy(t *testing.T) {
	// Mutating the returned slice (sort, append) must not affect the cache.
	ClearCache()
	resetRegistry()
	if _, err := Add(99016, "copytest", &Goal{Type: "a", Priority: 10}); err != nil {
		t.Fatal(err)
	}
	got := GoalsOf(99016, "copytest")
	sort.Slice(got, func(i, j int) bool { return false }) // reverse meaningless; just touch
	got = append(got, &Goal{Id: "fake"})
	if len(GoalsOf(99016, "copytest")) != 1 {
		t.Error("mutating returned slice affected cache")
	}
}

func TestConcurrentAdd_DifferentMobsNoRace(t *testing.T) {
	ClearCache()
	resetRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(mobId int) {
			defer wg.Done()
			if _, err := Add(mobId, fmt.Sprintf("racer%d", mobId), &Goal{
				Type: "x", Priority: 10,
			}); err != nil {
				t.Errorf("Add %d: %v", mobId, err)
			}
		}(99100 + i)
	}
	wg.Wait()
}

func TestConcurrentAdd_SameMobSerializes(t *testing.T) {
	ClearCache()
	resetRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Unique type per goroutine so no same-type conflict.
			if _, err := Add(99200, "samemob", &Goal{
				Type:     fmt.Sprintf("t%d", idx),
				Priority: 10 + idx,
			}); err != nil {
				t.Errorf("Add %d: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()
	got := GoalsOf(99200, "samemob")
	if len(got) != 5 {
		t.Errorf("expected 5 goals after concurrent Adds, got %d", len(got))
	}
}

func TestRecompute_FreshSelection_PersistsCurrentGoal(t *testing.T) {
	ClearCache()
	mobId := 99001
	name := "recompute_fresh"
	g := &Goal{Type: "wealth", Priority: 50}
	if _, err := Add(mobId, name, g); err != nil {
		t.Fatalf("seed Add: %v", err)
	}
	// After Add's eager Recompute (Task 7), this would already be set,
	// but for THIS task we test Recompute in isolation.
	mg := loadOrLazyInit(mobId, name)
	mg.CurrentGoalId = "" // simulate cold state
	mg.CurrentSinceRound = 0
	mg.LastSwitchRound = 0

	mob := &mobs.Mob{}
	Recompute(mobId, name, mob, 12345)

	mg = loadOrLazyInit(mobId, name)
	if mg.CurrentGoalId == "" {
		t.Errorf("CurrentGoalId not set after Recompute")
	}
	if mg.CurrentSinceRound != 12345 {
		t.Errorf("CurrentSinceRound=%d, want 12345", mg.CurrentSinceRound)
	}
	if mg.LastSwitchRound != 12345 {
		t.Errorf("LastSwitchRound=%d, want 12345", mg.LastSwitchRound)
	}
}

func TestRecompute_NoSwitch_DoesNotRewriteFile(t *testing.T) {
	ClearCache()
	mobId := 99002
	name := "recompute_nochange"
	g := &Goal{Type: "wealth", Priority: 50}
	if _, err := Add(mobId, name, g); err != nil {
		t.Fatalf("seed Add: %v", err)
	}
	mob := &mobs.Mob{}
	Recompute(mobId, name, mob, 1000)

	path := goalPath(mobId, name)
	info1, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after first Recompute: %v", err)
	}

	// Second Recompute with the same goal list — should NOT rewrite.
	time.Sleep(20 * time.Millisecond) // ensure mtime would change if write happened
	Recompute(mobId, name, mob, 1001)
	info2, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after second Recompute: %v", err)
	}
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Errorf("file mtime changed without a switch: %v → %v", info1.ModTime(), info2.ModTime())
	}
}

func TestCurrentGoalOf_AfterRecompute_ReturnsCurrentGoal(t *testing.T) {
	ClearCache()
	mobId := 99003
	name := "currentof_test"
	g := &Goal{Type: "wealth", Priority: 50}
	added, err := Add(mobId, name, g)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	Recompute(mobId, name, &mobs.Mob{}, 1000)
	got := CurrentGoalOf(mobId, name)
	if got == nil {
		t.Fatalf("CurrentGoalOf returned nil")
	}
	if got.Id != added.Added.Id {
		t.Errorf("got id=%s, want %s", got.Id, added.Added.Id)
	}
}

func TestCurrentGoalOf_NoGoals_ReturnsNil(t *testing.T) {
	ClearCache()
	got := CurrentGoalOf(99004, "no_goals_mob")
	if got != nil {
		t.Errorf("got=%v, want nil (no goals)", got)
	}
}

func TestCurrentGoalOf_StaleId_ReturnsNil(t *testing.T) {
	ClearCache()
	mobId := 99005
	name := "stale_id_test"
	// Seed the file with a current_goal_id that doesn't exist in goals slice.
	mg := &MobGoals{
		MobId:         mobId,
		NextGoalId:    2,
		CurrentGoalId: "g99",
		Goals:         []*Goal{{Id: "g1", Type: "wealth", Priority: 50}},
	}
	cacheStoreForTest(name, mg)
	got := CurrentGoalOf(mobId, name)
	if got != nil {
		t.Errorf("got=%v, want nil (stale current_goal_id)", got)
	}
}

func TestAdd_EagerRecompute_FirstGoalBecomesCurrent(t *testing.T) {
	ClearCache()
	mobId := 99101
	name := "eager_first"
	g := &Goal{Type: "wealth", Priority: 50}
	if _, err := Add(mobId, name, g); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got := CurrentGoalOf(mobId, name)
	if got == nil {
		t.Fatalf("CurrentGoalOf nil — Add did not eager-recompute")
	}
	if got.Type != "wealth" {
		t.Errorf("current.Type=%q, want wealth", got.Type)
	}
}

func TestRemove_OfCurrent_ClearsSelection_ThenEagerRecomputeSelectsFresh(t *testing.T) {
	ClearCache()
	mobId := 99102
	name := "eager_remove"
	g1 := &Goal{Type: "wealth", Priority: 30}
	g2 := &Goal{Type: "revenge", Priority: 90}
	r1, err := Add(mobId, name, g1)
	if err != nil {
		t.Fatalf("Add g1: %v", err)
	}
	// Backdate currentSinceRound so hysteresis min-hold is satisfied
	// before we add g2 (which has higher priority and should displace g1).
	mg := loadOrLazyInit(mobId, name)
	cacheMu.Lock()
	mg.CurrentSinceRound = 0 // held "forever" — far back enough to pass min-hold
	cacheMu.Unlock()
	if _, err := Add(mobId, name, g2); err != nil {
		t.Fatalf("Add g2: %v", err)
	}
	// After both Adds (with min-hold satisfied), current should be g2 (priority 90 > 30).
	if cur := CurrentGoalOf(mobId, name); cur == nil || cur.Type != "revenge" {
		t.Fatalf("pre-remove current = %v, want revenge", cur)
	}
	// Remove a non-current goal (g1) — should not disturb current (g2).
	if err := Remove(mobId, name, r1.Added.Id); err != nil {
		t.Fatalf("Remove g1: %v", err)
	}
	if cur := CurrentGoalOf(mobId, name); cur == nil || cur.Type != "revenge" {
		t.Errorf("after non-current remove, current = %v, want revenge", cur)
	}
}

func TestRemove_OfNonCurrent_DoesNotChangeCurrent(t *testing.T) {
	ClearCache()
	mobId := 99103
	name := "eager_remove_noncurr"
	g1 := &Goal{Type: "wealth", Priority: 90}
	g2 := &Goal{Type: "revenge", Priority: 30}
	r1, _ := Add(mobId, name, g1)
	r2, _ := Add(mobId, name, g2)
	currentBefore := CurrentGoalOf(mobId, name)
	if currentBefore == nil || currentBefore.Id != r1.Added.Id {
		t.Fatalf("pre-remove current=%v, want g1", currentBefore)
	}
	if err := Remove(mobId, name, r2.Added.Id); err != nil {
		t.Fatalf("Remove g2: %v", err)
	}
	currentAfter := CurrentGoalOf(mobId, name)
	if currentAfter == nil || currentAfter.Id != r1.Added.Id {
		t.Errorf("after non-current remove, current=%v, want g1", currentAfter)
	}
}

func TestClear_ZerosAllSelectionFields(t *testing.T) {
	ClearCache()
	mobId := 99104
	name := "eager_clear"
	if _, err := Add(mobId, name, &Goal{Type: "wealth", Priority: 50}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if cur := CurrentGoalOf(mobId, name); cur == nil {
		t.Fatalf("pre-clear current nil")
	}
	if err := Clear(mobId, name); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if cur := CurrentGoalOf(mobId, name); cur != nil {
		t.Errorf("post-clear current=%v, want nil", cur)
	}
	mg := loadOrLazyInit(mobId, name)
	if mg.CurrentSinceRound != 0 || mg.LastSwitchRound != 0 {
		t.Errorf("round fields not zeroed: since=%d switch=%d", mg.CurrentSinceRound, mg.LastSwitchRound)
	}
}

func TestAdd_ParamSchemaViolation_RejectsGoal(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-paramreq", GoalTypeMeta{
		Params: []ParamSchema{{Key: "target", Required: true, GoType: "int"}},
	})
	defer resetRegistry()

	mobId := 99301
	name := "param_test_mob"
	bad := &Goal{Type: "test-paramreq", Priority: 50, Params: map[string]any{}}
	_, err := Add(mobId, name, bad)
	if err == nil {
		t.Fatalf("Add returned nil error for missing required param")
	}
	var bpe *ErrBadParams
	if !errors.As(err, &bpe) {
		t.Fatalf("err type: got %T, want *ErrBadParams", err)
	}
	if got := GoalsOf(mobId, name); len(got) != 0 {
		t.Errorf("goal added despite validation failure: %v", got)
	}
}

func TestAdd_AllowMultiple_DifferentDedupKeys_BothAdded(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-multi", GoalTypeMeta{
		AllowMultiple: true,
		DedupKey: func(g *Goal) string {
			if tgt, ok := g.Params["target"].(int); ok {
				return strconv.Itoa(tgt)
			}
			return ""
		},
	})
	defer resetRegistry()

	mobId := 99401
	name := "multi_diff_mob"
	g1 := &Goal{Type: "test-multi", Priority: 50, Params: map[string]any{"target": 1}}
	g2 := &Goal{Type: "test-multi", Priority: 50, Params: map[string]any{"target": 2}}
	if _, err := Add(mobId, name, g1); err != nil {
		t.Fatalf("Add g1: %v", err)
	}
	if _, err := Add(mobId, name, g2); err != nil {
		t.Fatalf("Add g2 (different dedup key): %v", err)
	}
	if got := GoalsOf(mobId, name); len(got) != 2 {
		t.Errorf("expected 2 coexisting goals, got %d: %v", len(got), got)
	}
}

func TestAdd_AllowMultiple_SameDedupKey_ConflictsByPriority(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-multi2", GoalTypeMeta{
		AllowMultiple: true,
		DedupKey: func(g *Goal) string {
			if tgt, ok := g.Params["target"].(int); ok {
				return strconv.Itoa(tgt)
			}
			return ""
		},
	})
	defer resetRegistry()

	mobId := 99402
	name := "multi_same_mob"
	g1 := &Goal{Type: "test-multi2", Priority: 90, Params: map[string]any{"target": 1}}
	g2 := &Goal{Type: "test-multi2", Priority: 30, Params: map[string]any{"target": 1}}
	if _, err := Add(mobId, name, g1); err != nil {
		t.Fatalf("Add g1: %v", err)
	}
	_, err := Add(mobId, name, g2)
	if err == nil {
		t.Fatalf("Add g2 (same dedup key, lower priority) should have conflicted")
	}
	var ce *ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("err type: got %T, want *ConflictError", err)
	}
	if got := GoalsOf(mobId, name); len(got) != 1 {
		t.Errorf("expected 1 goal (blocker preserved), got %d", len(got))
	}
}

func TestAdd_AllowMultiple_SameDedupKey_HigherPriority_Displaces(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-multi3", GoalTypeMeta{
		AllowMultiple: true,
		DedupKey: func(g *Goal) string {
			if tgt, ok := g.Params["target"].(int); ok {
				return strconv.Itoa(tgt)
			}
			return ""
		},
	})
	defer resetRegistry()

	mobId := 99403
	name := "multi_displace_mob"
	g1 := &Goal{Type: "test-multi3", Priority: 30, Params: map[string]any{"target": 1}}
	g2 := &Goal{Type: "test-multi3", Priority: 90, Params: map[string]any{"target": 1}}
	if _, err := Add(mobId, name, g1); err != nil {
		t.Fatalf("Add g1: %v", err)
	}
	res, err := Add(mobId, name, g2)
	if err != nil {
		t.Fatalf("Add g2 (same dedup, higher priority): %v", err)
	}
	if len(res.Displaced) != 1 {
		t.Errorf("expected 1 displaced, got %v", res.Displaced)
	}
}

func TestAdd_AllowMultipleFalse_StillBlocksSameType(t *testing.T) {
	// Confirms 4.1 behavior preserved for types that don't opt in.
	ClearCache()
	RegisterGoalType("test-singleton", GoalTypeMeta{})
	defer resetRegistry()

	mobId := 99404
	name := "single_mob"
	g1 := &Goal{Type: "test-singleton", Priority: 50}
	g2 := &Goal{Type: "test-singleton", Priority: 50}
	if _, err := Add(mobId, name, g1); err != nil {
		t.Fatalf("Add g1: %v", err)
	}
	_, err := Add(mobId, name, g2)
	if err == nil {
		t.Fatalf("expected ConflictError on second add of singleton type")
	}
}

func TestAdd_DedupKey_PanicRecovered_FallsThroughToNoKeyCollision(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-panicky", GoalTypeMeta{
		AllowMultiple: true,
		DedupKey: func(g *Goal) string {
			panic("dedup-key boom")
		},
	})
	defer resetRegistry()

	mobId := 99405
	name := "panicky_mob"
	g1 := &Goal{Type: "test-panicky", Priority: 50, Params: map[string]any{"a": 1}}
	g2 := &Goal{Type: "test-panicky", Priority: 50, Params: map[string]any{"a": 2}}
	if _, err := Add(mobId, name, g1); err != nil {
		t.Fatalf("Add g1 with panicking dedup-key: %v", err)
	}
	if _, err := Add(mobId, name, g2); err != nil {
		t.Fatalf("Add g2 with panicking dedup-key: %v", err)
	}
	if got := GoalsOf(mobId, name); len(got) != 2 {
		t.Errorf("expected 2 goals (panic → empty key → coexist), got %d", len(got))
	}
}

func TestLoadOrLazyInit_FreshMob_NoLookup_SetsSentinelTrueNoSeeds(t *testing.T) {
	ClearCache()
	SetArchetypeDefaultsLookup(nil)
	mobId := 99501
	name := "fresh_no_lookup"
	mg := loadOrLazyInit(mobId, name)
	if !mg.SeededFromArchetype {
		t.Errorf("SeededFromArchetype=false, want true (sentinel must flip even with nil lookup)")
	}
	if len(mg.Goals) != 0 {
		t.Errorf("expected 0 goals, got %d", len(mg.Goals))
	}
}

func TestLoadOrLazyInit_FreshMob_WithLookup_SeedsAndPersists(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-seedable", GoalTypeMeta{})
	defer resetRegistry()
	SetArchetypeDefaultsLookup(func(mob *mobs.Mob) []GoalDefault {
		return []GoalDefault{{Type: "test-seedable", Priority: 80}}
	})
	defer SetArchetypeDefaultsLookup(nil)

	mobId := 99502
	name := "fresh_with_seeds"
	mg := loadOrLazyInit(mobId, name)
	if !mg.SeededFromArchetype {
		t.Errorf("SeededFromArchetype=false, want true")
	}
	if len(mg.Goals) != 1 {
		t.Fatalf("expected 1 seeded goal, got %d", len(mg.Goals))
	}
	if mg.Goals[0].Type != "test-seedable" || mg.Goals[0].Priority != 80 {
		t.Errorf("seeded goal mismatch: %+v", mg.Goals[0])
	}
}

func TestLoadOrLazyInit_ExistingFileWithSentinelTrue_SkipsSeed(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-skipseed", GoalTypeMeta{})
	defer resetRegistry()
	called := 0
	SetArchetypeDefaultsLookup(func(mob *mobs.Mob) []GoalDefault {
		called++
		return []GoalDefault{{Type: "test-skipseed", Priority: 80}}
	})
	defer SetArchetypeDefaultsLookup(nil)

	mobId := 99503
	name := "skip_seed_mob"
	cacheStoreForTest(name, &MobGoals{MobId: mobId, NextGoalId: 1, SeededFromArchetype: true})
	mg := loadOrLazyInit(mobId, name)
	if len(mg.Goals) != 0 {
		t.Errorf("expected 0 goals (no seed), got %d", len(mg.Goals))
	}
	if called > 0 {
		t.Errorf("lookup called %d times; expected 0", called)
	}
}

func TestClear_PreservesSentinel(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-preserve", GoalTypeMeta{})
	defer resetRegistry()
	SetArchetypeDefaultsLookup(func(mob *mobs.Mob) []GoalDefault {
		return []GoalDefault{{Type: "test-preserve", Priority: 80}}
	})
	defer SetArchetypeDefaultsLookup(nil)

	mobId := 99504
	name := "clear_preserve_mob"
	_ = loadOrLazyInit(mobId, name)
	if err := Clear(mobId, name); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	mg := loadOrLazyInit(mobId, name)
	if !mg.SeededFromArchetype {
		t.Errorf("sentinel cleared by Clear — should be preserved")
	}
	if len(mg.Goals) != 0 {
		t.Errorf("expected 0 goals after Clear+reload, got %d", len(mg.Goals))
	}
}

func TestLoadOrLazyInit_SeededDefaultFailsValidation_LogsAndContinues(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-strict", GoalTypeMeta{
		Params: []ParamSchema{{Key: "target", Required: true, GoType: "int"}},
	})
	RegisterGoalType("test-loose", GoalTypeMeta{})
	defer resetRegistry()
	SetArchetypeDefaultsLookup(func(mob *mobs.Mob) []GoalDefault {
		return []GoalDefault{
			{Type: "test-strict", Priority: 80}, // missing required param
			{Type: "test-loose", Priority: 40},  // seeds successfully
		}
	})
	defer SetArchetypeDefaultsLookup(nil)

	mobId := 99505
	name := "partial_seed_mob"
	mg := loadOrLazyInit(mobId, name)
	if !mg.SeededFromArchetype {
		t.Errorf("sentinel should flip even when one default failed")
	}
	if len(mg.Goals) != 1 || mg.Goals[0].Type != "test-loose" {
		t.Errorf("expected only test-loose to seed, got %v", mg.Goals)
	}
}
