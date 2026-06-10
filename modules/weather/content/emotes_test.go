package content

import (
	"testing"
	"testing/fstest"
)

const stormYAML = `weather: storm
outdoor:
  default:
    - "Thunder cracks directly overhead."
    - "A blinding fork of lightning splits the sky."
  forest:
    - "Wind tears at the branches; the whole canopy roars."
indoor:
  default:
    - "Rain hammers against the windows."
`

func loadTestTables(t *testing.T) Tables {
	t.Helper()
	fsys := fstest.MapFS{"emotes/storm.yaml": {Data: []byte(stormYAML)}}
	tables, err := LoadEmotes(fsys, "emotes")
	if err != nil {
		t.Fatal(err)
	}
	return tables
}

func TestPickSelectsByBiomeAndIndoor(t *testing.T) {
	tables := loadTestTables(t)
	first := func(n int) int { return 0 }

	if got := tables.Pick("storm", "forest", false, first); got != "Wind tears at the branches; the whole canopy roars." {
		t.Errorf("forest outdoor: %q", got)
	}
	if got := tables.Pick("storm", "desert", false, first); got != "Thunder cracks directly overhead." {
		t.Errorf("unknown biome should fall back to default: %q", got)
	}
	if got := tables.Pick("storm", "forest", true, first); got != "Rain hammers against the windows." {
		t.Errorf("indoor falls back to indoor default (never outdoor): %q", got)
	}
	if got := tables.Pick("fog", "forest", false, first); got != "" {
		t.Errorf("missing table must yield silence: %q", got)
	}
}

func TestPickUsesRoll(t *testing.T) {
	tables := loadTestTables(t)
	rolled := -1
	got := tables.Pick("storm", "default", false, func(n int) int { rolled = n; return 1 })
	if rolled != 2 {
		t.Errorf("roll should receive the line count, got %d", rolled)
	}
	if got != "A blinding fork of lightning splits the sky." {
		t.Errorf("roll result not honored: %q", got)
	}
}

func TestLoadEmotesRejectsMissingWeatherKey(t *testing.T) {
	fsys := fstest.MapFS{"emotes/bad.yaml": {Data: []byte("outdoor:\n  default: [\"x\"]\n")}}
	if _, err := LoadEmotes(fsys, "emotes"); err == nil {
		t.Fatal("emote table without 'weather' must be rejected")
	}
}

func TestLoadEmotesMissingDir(t *testing.T) {
	tables, err := LoadEmotes(fstest.MapFS{}, "emotes")
	if err != nil || len(tables) != 0 {
		t.Fatalf("missing dir should be empty tables, nil error: %v %v", tables, err)
	}
}

func TestPickClampsOutOfRangeRoll(t *testing.T) {
	tables := loadTestTables(t)
	if got := tables.Pick("storm", "default", false, func(n int) int { return n }); got != "Thunder cracks directly overhead." {
		t.Errorf("out-of-range roll should clamp to first line: %q", got)
	}
	if got := tables.Pick("storm", "default", false, func(n int) int { return -3 }); got != "Thunder cracks directly overhead." {
		t.Errorf("negative roll should clamp to first line: %q", got)
	}
}
