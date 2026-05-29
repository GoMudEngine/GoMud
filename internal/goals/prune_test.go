package goals

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// NOTE: package-global driven by serial tests via defer-restore. Do NOT
// mark tests in this file t.Parallel() while this global exists.
//
// pruneCtxScore lets tests drive a goal type's context score.
var pruneCtxScore = 1.0

func registerPruneType(t *testing.T) {
	t.Helper()
	resetRegistry()
	RegisterGoalType("prune-sat", GoalTypeMeta{
		Predicate:    func(g *Goal, m *mobs.Mob) bool { return true },
		ContextScore: func(g *Goal, m *mobs.Mob) float64 { return 1.0 },
	})
	RegisterGoalType("prune-live", GoalTypeMeta{
		Predicate:    func(g *Goal, m *mobs.Mob) bool { return false },
		ContextScore: func(g *Goal, m *mobs.Mob) float64 { return pruneCtxScore },
	})
}

func seedGoals(t *testing.T, mobId int, name string, gs ...*Goal) {
	t.Helper()
	ClearCache()
	mg := &MobGoals{MobId: mobId, NextGoalId: len(gs) + 1, Goals: gs}
	cacheStoreForTest(name, mg)
	if err := saveToDisk(mobId, name); err != nil {
		t.Fatalf("seed save: %v", err)
	}
}

func TestPrune_RemovesSatisfied(t *testing.T) {
	registerPruneType(t)
	seedGoals(t, 99301, "p-sat",
		&Goal{Id: "g1", Type: "prune-sat", Priority: 50, CreatedAt: time.Now().UTC()})
	recs := Prune(99301, "p-sat", nil, time.Now().UTC(), 1000)
	if len(recs) != 1 || recs[0].Reason != ReasonSatisfied {
		t.Fatalf("want 1 satisfied record, got %+v", recs)
	}
	if len(GoalsOf(99301, "p-sat")) != 0 {
		t.Error("satisfied goal not removed")
	}
}

func TestPrune_RemovesExpired(t *testing.T) {
	registerPruneType(t)
	seedGoals(t, 99302, "p-exp",
		&Goal{Id: "g1", Type: "prune-live", Priority: 50, CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)})
	recs := Prune(99302, "p-exp", nil, time.Now().UTC(), 1000)
	if len(recs) != 1 || recs[0].Reason != ReasonExpired {
		t.Fatalf("want 1 expired record, got %+v", recs)
	}
}

func TestPrune_AbandonsDormantPastThreshold(t *testing.T) {
	registerPruneType(t)
	pruneCtxScore = 0.0
	defer func() { pruneCtxScore = 1.0 }()
	seedGoals(t, 99303, "p-dorm",
		&Goal{Id: "g1", Type: "prune-live", Priority: 50, CreatedAt: time.Now().UTC(),
			DormantSinceRound: 100})
	recs := Prune(99303, "p-dorm", nil, time.Now().UTC(), 1000)
	if len(recs) != 1 || recs[0].Reason != ReasonAbandoned {
		t.Fatalf("want 1 abandoned record, got %+v", recs)
	}
}

func TestPrune_StampsDormancyButKeepsUnderThreshold(t *testing.T) {
	registerPruneType(t)
	pruneCtxScore = 0.0
	defer func() { pruneCtxScore = 1.0 }()
	seedGoals(t, 99304, "p-stamp",
		&Goal{Id: "g1", Type: "prune-live", Priority: 50, CreatedAt: time.Now().UTC()})
	recs := Prune(99304, "p-stamp", nil, time.Now().UTC(), 1000)
	if len(recs) != 0 {
		t.Fatalf("want no removals, got %+v", recs)
	}
	got := GoalsOf(99304, "p-stamp")
	if len(got) != 1 || got[0].DormantSinceRound != 1000 {
		t.Fatalf("want DormantSinceRound stamped to 1000, got %+v", got)
	}
}

func TestPrune_ClearsDormancyWhenLiveAgain(t *testing.T) {
	registerPruneType(t)
	pruneCtxScore = 1.0
	seedGoals(t, 99305, "p-clear",
		&Goal{Id: "g1", Type: "prune-live", Priority: 50, CreatedAt: time.Now().UTC(),
			DormantSinceRound: 100})
	recs := Prune(99305, "p-clear", nil, time.Now().UTC(), 1000)
	if len(recs) != 0 {
		t.Fatalf("want no removals, got %+v", recs)
	}
	got := GoalsOf(99305, "p-clear")
	if len(got) != 1 || got[0].DormantSinceRound != 0 {
		t.Fatalf("want DormantSinceRound cleared, got %+v", got)
	}
}

func TestPrune_BatchRemovesMultiple(t *testing.T) {
	registerPruneType(t)
	seedGoals(t, 99306, "p-batch",
		&Goal{Id: "g1", Type: "prune-sat", Priority: 50, CreatedAt: time.Now().UTC()},
		&Goal{Id: "g2", Type: "prune-live", Priority: 40, CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)})
	recs := Prune(99306, "p-batch", nil, time.Now().UTC(), 1000)
	if len(recs) != 2 {
		t.Fatalf("want 2 removals, got %+v", recs)
	}
	if len(GoalsOf(99306, "p-batch")) != 0 {
		t.Error("expected all dead goals removed")
	}
}

func TestGoal_DormantSinceRound_Persists(t *testing.T) {
	ClearCache()
	resetRegistry()
	mg := &MobGoals{MobId: 99201, NextGoalId: 2, Goals: []*Goal{
		{Id: "g1", Type: "revenge", Priority: 70, CreatedAt: time.Now().UTC(), DormantSinceRound: 4242},
	}}
	cacheStoreForTest("dormant-mob", mg)
	if err := saveToDisk(99201, "dormant-mob"); err != nil {
		t.Fatalf("save: %v", err)
	}
	ClearCache()
	got := GoalsOf(99201, "dormant-mob")
	if len(got) != 1 {
		t.Fatalf("want 1 goal, got %d", len(got))
	}
	if got[0].DormantSinceRound != 4242 {
		t.Errorf("DormantSinceRound not persisted: got %d", got[0].DormantSinceRound)
	}
}

func TestPrune_DormantAlreadyStamped_NotRestamped(t *testing.T) {
	registerPruneType(t)
	pruneCtxScore = 0.0
	defer func() { pruneCtxScore = 1.0 }()
	// Stamped at round 100; sweep at 200 (under the 600 default threshold).
	seedGoals(t, 99307, "p-nochurn",
		&Goal{Id: "g1", Type: "prune-live", Priority: 50, CreatedAt: time.Now().UTC(),
			DormantSinceRound: 100})
	recs := Prune(99307, "p-nochurn", nil, time.Now().UTC(), 200)
	if len(recs) != 0 {
		t.Fatalf("want no removals, got %+v", recs)
	}
	got := GoalsOf(99307, "p-nochurn")
	if len(got) != 1 || got[0].DormantSinceRound != 100 {
		t.Fatalf("DormantSinceRound must stay 100 (not re-stamped), got %+v", got)
	}
}
