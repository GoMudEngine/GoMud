package seasons

import (
	"strings"
	"testing"
	"testing/fstest"
)

const temperateYAML = `track: temperate
seasons:
  - name: winter
    months: [12, 1, 2]
    transitionDays: 6
    weatherWeightMultipliers: { snow: 3.0, heatwave: 0.0 }
    spawnWeightMultiplier: 0.9
    influence: { intensityDelta: -0.02 }
  - name: spring
    months: [3, 4, 5]
    transitionDays: 6
  - name: summer
    months: [6, 7, 8]
    transitionDays: 6
    weatherWeightMultipliers: { storm: 1.4 }
  - name: autumn
    months: [9, 10, 11]
    transitionDays: 6
`

func loadOne(t *testing.T, yaml string) Tracks {
	t.Helper()
	fsys := fstest.MapFS{"seasons/x.yaml": {Data: []byte(yaml)}}
	tracks, errs := Load(fsys, "seasons", 12, 365)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	return tracks
}

func TestLoadTemperate(t *testing.T) {
	tracks := loadOne(t, temperateYAML)
	tr, ok := tracks["temperate"]
	if !ok {
		t.Fatalf("track not keyed by name: %v", tracks)
	}
	if len(tr.Seasons) != 4 {
		t.Fatalf("expected 4 seasons, got %d", len(tr.Seasons))
	}
	w := tr.Seasons[0]
	if w.Name != "winter" || w.TransitionDays != 6 || w.SpawnWeightMultiplier != 0.9 {
		t.Errorf("winter mis-parsed: %+v", w)
	}
	if w.WeatherWeightMultipliers["snow"] != 3.0 || w.WeatherWeightMultipliers["heatwave"] != 0.0 {
		t.Errorf("multipliers mis-parsed: %+v", w.WeatherWeightMultipliers)
	}
	if w.Influence.IntensityDelta != -0.02 {
		t.Errorf("influence mis-parsed: %+v", w.Influence)
	}
	// Unset spawn multiplier defaults to 1.0.
	if tr.Seasons[1].SpawnWeightMultiplier != 1.0 {
		t.Errorf("unset spawnWeightMultiplier should default 1.0: %v", tr.Seasons[1].SpawnWeightMultiplier)
	}
}

func TestLoadEsotericFields(t *testing.T) {
	yaml := `track: stillness
seasons:
  - name: calm
    months: [1,2,3,4,5,6,7,8,9,10]
  - name: shattering
    months: [11,12]
    transitionDays: 2
    baseWeightScale: 0.0
    weatherWeightAdditions: { glassrain: 8, ashfall: 3 }
`
	tr := loadOne(t, yaml)["stillness"]
	calm, shat := tr.Seasons[0], tr.Seasons[1]
	if calm.BaseWeightScale != 1.0 {
		t.Errorf("unset baseWeightScale should default 1.0: %v", calm.BaseWeightScale)
	}
	if shat.BaseWeightScale != 0.0 {
		t.Errorf("explicit baseWeightScale 0 not honored: %v", shat.BaseWeightScale)
	}
	if shat.WeatherWeightAdditions["glassrain"] != 8 || shat.WeatherWeightAdditions["ashfall"] != 3 {
		t.Errorf("additions mis-parsed: %+v", shat.WeatherWeightAdditions)
	}
	if len(calm.WeatherWeightAdditions) != 0 {
		t.Errorf("calm should have no additions: %+v", calm.WeatherWeightAdditions)
	}
}

func TestLoadValidation(t *testing.T) {
	cases := []struct{ name, yaml, wantErr string }{
		{"gap", "track: t\nseasons:\n  - name: a\n    months: [1,2,3,4,5,6]\n", "claimed by no season"},
		{"overlap", "track: t\nseasons:\n  - name: a\n    months: [1,2,3,4,5,6,7]\n  - name: b\n    months: [7,8,9,10,11,12]\n", "claimed twice"},
		{"out of range", "track: t\nseasons:\n  - name: a\n    months: [1,2,3,4,5,6,13]\n  - name: b\n    months: [7,8,9,10,11,12]\n", "outside 1..12"},
		{"non-contiguous", "track: t\nseasons:\n  - name: a\n    months: [1,7,2,3,4,5]\n  - name: b\n    months: [6,8,9,10,11,12]\n", "contiguous"},
		{"no name", "track: t\nseasons:\n  - months: [1,2,3,4,5,6,7,8,9,10,11,12]\n", "missing a name"},
		{"no track key", "seasons:\n  - name: a\n    months: [1,2,3,4,5,6,7,8,9,10,11,12]\n", "missing required 'track'"},
		{"negative addition", "track: t\nseasons:\n  - name: a\n    months: [1,2,3,4,5,6,7,8,9,10,11,12]\n    weatherWeightAdditions: { glassrain: -1 }\n", "negative"},
		{"negative scale", "track: t\nseasons:\n  - name: a\n    months: [1,2,3,4,5,6,7,8,9,10,11,12]\n    baseWeightScale: -0.5\n", "negative"},
	}
	for _, c := range cases {
		fsys := fstest.MapFS{"seasons/x.yaml": {Data: []byte(c.yaml)}}
		tracks, errs := Load(fsys, "seasons", 12, 365)
		if len(errs) == 0 || !strings.Contains(errs[0].Error(), c.wantErr) {
			t.Errorf("%s: want error containing %q, got %v", c.name, c.wantErr, errs)
		}
		if len(tracks) != 0 {
			t.Errorf("%s: invalid track must be dropped, got %v", c.name, tracks)
		}
	}
}

func TestLoadMissingDirIsEmpty(t *testing.T) {
	tracks, errs := Load(fstest.MapFS{}, "seasons", 12, 365)
	if len(tracks) != 0 || len(errs) != 0 {
		t.Fatalf("missing dir: want empty/no errors, got %v %v", tracks, errs)
	}
}

func TestLoadRejectsDuplicateTrackNames(t *testing.T) {
	fsys := fstest.MapFS{
		"seasons/a.yaml": {Data: []byte("track: t\nseasons:\n  - name: a\n    months: [1,2,3,4,5,6,7,8,9,10,11,12]\n")},
		"seasons/b.yaml": {Data: []byte("track: t\nseasons:\n  - name: b\n    months: [1,2,3,4,5,6,7,8,9,10,11,12]\n")},
	}
	tracks, errs := Load(fsys, "seasons", 12, 365)
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "duplicate track") {
		t.Errorf("want duplicate-track error, got %v", errs)
	}
	if len(tracks) != 1 {
		t.Errorf("first track wins; got %d tracks", len(tracks))
	}
}
