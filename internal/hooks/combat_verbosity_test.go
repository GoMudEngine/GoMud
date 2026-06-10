package hooks

import (
	"strings"
	"testing"
)

func mkSwings(hits, misses int, worstDamage int) []swingStat {
	out := []swingStat{}
	for i := 0; i < hits; i++ {
		d := 1
		if i == 0 {
			d = worstDamage
		}
		out = append(out, swingStat{Hit: true, Damage: d})
	}
	for i := 0; i < misses; i++ {
		out = append(out, swingStat{Hit: false})
	}
	return out
}

func TestTally_ParticipantOutgoingHitsAndEnemyWhiffs(t *testing.T) {
	agg := newCombatTallies()
	// Viewer (user 7) attacked the wolf: 2 hits (worst 30), 1 miss. Wolf maxHP 100.
	agg.record(7, fighterRef{Key: "u:7", Name: "You-Ignored", IsMob: false},
		fighterRef{Key: "m:31", Name: "marsh wolf", IsMob: true},
		mkSwings(2, 1, 30), 100)
	// Wolf attacked viewer: 0 hits, 2 misses. Viewer maxHP 200.
	agg.record(7, fighterRef{Key: "m:31", Name: "marsh wolf", IsMob: true},
		fighterRef{Key: "u:7", Name: "You-Ignored", IsMob: false},
		mkSwings(0, 2, 0), 200)

	lines := agg.flushForViewer(7, "u:7")
	if len(lines) != 1 {
		t.Fatalf("expected 1 tally line, got %d: %v", len(lines), lines)
	}
	l := lines[0]
	if !strings.Contains(l, "You strike") || !strings.Contains(l, "marsh wolf") || !strings.Contains(l, "twice") {
		t.Errorf("outgoing segment wrong: %q", l)
	}
	// 30/100 = 30%% → "serious wounds" per GetDamageDescription thresholds
	if !strings.Contains(l, "serious wounds") {
		t.Errorf("expected worst-hit tier 'serious wounds': %q", l)
	}
	if !strings.Contains(l, "fails to land a blow") {
		t.Errorf("expected enemy whiff segment: %q", l)
	}
}

func TestTally_ParticipantIncomingHitsOmitted(t *testing.T) {
	agg := newCombatTallies()
	agg.record(7, fighterRef{Key: "u:7", Name: "x", IsMob: false},
		fighterRef{Key: "m:31", Name: "marsh wolf", IsMob: true},
		mkSwings(1, 0, 10), 100)
	// Wolf LANDED hits on the viewer — those showed in full prose (floor
	// rule), so the tally must NOT re-describe them.
	agg.record(7, fighterRef{Key: "m:31", Name: "marsh wolf", IsMob: true},
		fighterRef{Key: "u:7", Name: "x", IsMob: false},
		mkSwings(2, 0, 40), 200)

	lines := agg.flushForViewer(7, "u:7")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %v", lines)
	}
	if strings.Contains(lines[0], "fails to land") {
		t.Errorf("enemy landed hits; whiff text is wrong: %q", lines[0])
	}
	// Incoming hits already shown in full — no tier for them in the tally.
	if strings.Count(lines[0], "wounds")+strings.Count(lines[0], "injuries")+strings.Count(lines[0], "damage") > 1 {
		t.Errorf("incoming damage should not be re-described: %q", lines[0])
	}
}

func TestTally_WhiffRound(t *testing.T) {
	agg := newCombatTallies()
	agg.record(7, fighterRef{Key: "u:7", Name: "x", IsMob: false},
		fighterRef{Key: "m:31", Name: "marsh wolf", IsMob: true},
		mkSwings(0, 2, 0), 100)
	agg.record(7, fighterRef{Key: "m:31", Name: "marsh wolf", IsMob: true},
		fighterRef{Key: "u:7", Name: "x", IsMob: false},
		mkSwings(0, 1, 0), 200)

	lines := agg.flushForViewer(7, "u:7")
	if len(lines) != 1 || !strings.Contains(lines[0], "neither side draws blood") {
		t.Errorf("whiff round wording: %v", lines)
	}
}

func TestTally_SpectatorBothDirections(t *testing.T) {
	agg := newCombatTallies()
	// Viewer 9 watches Velk (user 4) fight the shambler.
	agg.record(9, fighterRef{Key: "u:4", Name: "Velk", IsMob: false},
		fighterRef{Key: "m:50", Name: "bog shambler", IsMob: true},
		mkSwings(3, 0, 35), 100)
	agg.record(9, fighterRef{Key: "m:50", Name: "bog shambler", IsMob: true},
		fighterRef{Key: "u:4", Name: "Velk", IsMob: false},
		mkSwings(1, 1, 12), 150)

	lines := agg.flushForViewer(9, "")
	if len(lines) != 1 {
		t.Fatalf("expected 1 spectator line, got %v", lines)
	}
	l := lines[0]
	if !strings.Contains(l, "Velk") || !strings.Contains(l, "bog shambler") {
		t.Errorf("names missing: %q", l)
	}
	if !strings.Contains(l, "three times") {
		t.Errorf("expected count word 'three times': %q", l)
	}
	// 35/100 → serious wounds; 12/150 = 8%% → light wounds
	if !strings.Contains(l, "serious wounds") || !strings.Contains(l, "light wounds") {
		t.Errorf("expected both direction tiers: %q", l)
	}
}

func TestTally_MultipleFightsSortedAndCleared(t *testing.T) {
	agg := newCombatTallies()
	agg.record(9, fighterRef{Key: "u:4", Name: "Velk", IsMob: false},
		fighterRef{Key: "m:50", Name: "bog shambler", IsMob: true}, mkSwings(1, 0, 10), 100)
	agg.record(9, fighterRef{Key: "u:5", Name: "Tova", IsMob: false},
		fighterRef{Key: "m:51", Name: "reed viper", IsMob: true}, mkSwings(1, 0, 10), 100)

	lines := agg.flushForViewer(9, "")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %v", lines)
	}
	// Deterministic order (sorted by pair key).
	again := newCombatTallies()
	again.record(9, fighterRef{Key: "u:5", Name: "Tova", IsMob: false},
		fighterRef{Key: "m:51", Name: "reed viper", IsMob: true}, mkSwings(1, 0, 10), 100)
	again.record(9, fighterRef{Key: "u:4", Name: "Velk", IsMob: false},
		fighterRef{Key: "m:50", Name: "bog shambler", IsMob: true}, mkSwings(1, 0, 10), 100)
	lines2 := again.flushForViewer(9, "")
	if lines[0] != lines2[0] || lines[1] != lines2[1] {
		t.Errorf("flush order must be deterministic:\n%v\n%v", lines, lines2)
	}

	// flush clears
	if rem := agg.flushForViewer(9, ""); len(rem) != 0 {
		t.Errorf("second flush must be empty, got %v", rem)
	}
}

func TestCountWord(t *testing.T) {
	cases := map[int]string{1: "", 2: " twice", 3: " three times", 4: " again and again", 7: " again and again"}
	for n, want := range cases {
		if got := countWord(n); got != want {
			t.Errorf("countWord(%d) = %q want %q", n, got, want)
		}
	}
}
