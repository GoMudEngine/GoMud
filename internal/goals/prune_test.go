package goals

import (
	"testing"
	"time"
)

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
