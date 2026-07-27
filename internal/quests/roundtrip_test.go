package quests

// The 5c writer's permanent safety net (5b pattern): for every live quest
// file, parse → marshal → parse → marshal must be a FIXED POINT (the two
// marshals byte-identical), sections must survive, and no non-zero reward
// may be lost. Authored files are NOT required to match the canonical
// marshal form — only to survive it — so this never forces content churn;
// it catches tag mistakes that would make a future editor save silently
// drop or rename data.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"gopkg.in/yaml.v2"
)

func TestRoundTrip_MarshalFixedPointEveryLiveFile(t *testing.T) {
	chdirRepoRootForTest(t)

	dir := configs.GetFilePathsConfig().DataFiles.String() + `/quests`
	files, err := filepath.Glob(dir + `/*.yaml`)
	if err != nil || len(files) == 0 {
		t.Fatalf("no quest files found under %s: %v", dir, err)
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		var q1 Quest
		if err := yaml.Unmarshal(data, &q1); err != nil {
			t.Fatalf("%s: parse: %v", f, err)
		}
		out1, err := yaml.Marshal(q1)
		if err != nil {
			t.Fatalf("%s: marshal: %v", f, err)
		}
		var q2 Quest
		if err := yaml.Unmarshal(out1, &q2); err != nil {
			t.Fatalf("%s: reparse of own marshal: %v", f, err)
		}
		out2, err := yaml.Marshal(q2)
		if err != nil {
			t.Fatalf("%s: re-marshal: %v", f, err)
		}
		if string(out1) != string(out2) {
			t.Errorf("%s: marshal is not a fixed point — a writer built on this form would churn on every save", f)
		}

		// Section survival.
		if len(q2.Steps) != len(q1.Steps) || len(q2.Triggers) != len(q1.Triggers) || len(q2.Flags) != len(q1.Flags) {
			t.Errorf("%s: sections lost in round-trip: steps %d→%d triggers %d→%d flags %d→%d",
				f, len(q1.Steps), len(q2.Steps), len(q1.Triggers), len(q2.Triggers), len(q1.Flags), len(q2.Flags))
		}
		for i := range q1.Triggers {
			if len(q2.Triggers[i].Actions) != len(q1.Triggers[i].Actions) {
				t.Errorf("%s: trigger %d lost actions in round-trip: %d→%d",
					f, i, len(q1.Triggers[i].Actions), len(q2.Triggers[i].Actions))
			}
		}

		// A reward the file pays must survive the round-trip — this is the
		// check that makes the silent-unpaid-reward class impossible for any
		// writer that marshals this struct.
		if q2.Rewards != q1.Rewards {
			t.Errorf("%s: REWARDS changed in round-trip:\nbefore=%+v\nafter =%+v", f, q1.Rewards, q2.Rewards)
		}
	}
}
