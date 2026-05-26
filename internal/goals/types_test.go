package goals

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestGoal_YAMLRoundTrip(t *testing.T) {
	g := &Goal{
		Id:       "g1",
		Type:     "revenge",
		Priority: 70,
		Params: map[string]any{
			"target_player_name": "smoketester",
			"reason":             "killed brother",
			"observed_round":     12345,
			"intensity":          1.5,
			"public":             true,
		},
		CreatedAt: time.Date(2026, 5, 26, 14, 30, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 6, 25, 14, 30, 0, 0, time.UTC),
	}
	out, err := yaml.Marshal(g)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Goal
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Id != g.Id || got.Type != g.Type || got.Priority != g.Priority {
		t.Fatalf("scalar mismatch: %+v vs %+v", got, g)
	}
	if !got.CreatedAt.Equal(g.CreatedAt) || !got.ExpiresAt.Equal(g.ExpiresAt) {
		t.Fatalf("time mismatch: %v / %v vs %v / %v",
			got.CreatedAt, got.ExpiresAt, g.CreatedAt, g.ExpiresAt)
	}
	if got.Params["target_player_name"] != "smoketester" {
		t.Errorf("params string lost: %v", got.Params["target_player_name"])
	}
	if got.Params["public"] != true {
		t.Errorf("params bool lost: %v", got.Params["public"])
	}
}

func TestMobGoals_YAMLRoundTrip(t *testing.T) {
	mg := &MobGoals{
		MobId:      371,
		NextGoalId: 3,
		Goals: []*Goal{
			{Id: "g1", Type: "revenge", Priority: 70, CreatedAt: time.Now().UTC()},
			{Id: "g2", Type: "wealth-target", Priority: 30, CreatedAt: time.Now().UTC()},
		},
	}
	out, err := yaml.Marshal(mg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got MobGoals
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.MobId != 371 || got.NextGoalId != 3 || len(got.Goals) != 2 {
		t.Fatalf("round trip lost data: %+v", got)
	}
	if got.Goals[0].Id != "g1" || got.Goals[1].Id != "g2" {
		t.Errorf("goal order lost: %v / %v", got.Goals[0].Id, got.Goals[1].Id)
	}
}
