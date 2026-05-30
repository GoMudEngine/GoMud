package justice

import "testing"

// ---------------------------------------------------------------------------
// resolveArrest tests — pure helper, no live mobs/users required.
// ---------------------------------------------------------------------------

// (a) Surrender policy, first tick: guard should declare intent (stamp pending).
func TestResolveArrest_Surrender_FirstTick_Declares(t *testing.T) {
	out := resolveArrest(false, false, 0, 1000, 3)
	if out != arrestOutcomeDeclare {
		t.Errorf("first tick surrender: got %v, want Declare", out)
	}
}

// (b) Surrender, pending within grace: no-op.
func TestResolveArrest_Surrender_WithinGrace_NoOp(t *testing.T) {
	out := resolveArrest(false, true, 1000, 1001, 3)
	if out != arrestOutcomeNone {
		t.Errorf("within-grace surrender: got %v, want None", out)
	}
}

// (c) Surrender, pending past grace: haul off (executeArrest).
func TestResolveArrest_Surrender_PastGrace_HaulOff(t *testing.T) {
	out := resolveArrest(false, true, 1000, 1004, 3)
	if out != arrestOutcomeHaul {
		t.Errorf("past-grace surrender: got %v, want Haul", out)
	}
}

// (d) Resist policy: should attack regardless of pending state.
func TestResolveArrest_Resist_Attack(t *testing.T) {
	out := resolveArrest(true, false, 0, 1000, 3)
	if out != arrestOutcomeAttack {
		t.Errorf("resist policy: got %v, want Attack", out)
	}
}

// Resist with pending stamp: still attack (resist ignores grace window).
func TestResolveArrest_Resist_AlreadyPending_Attack(t *testing.T) {
	out := resolveArrest(true, true, 900, 1000, 3)
	if out != arrestOutcomeAttack {
		t.Errorf("resist+pending: got %v, want Attack", out)
	}
}

// Grace boundary: exactly at grace rounds — should NOT haul (< not <=).
func TestResolveArrest_Surrender_AtGraceBoundary_NoOp(t *testing.T) {
	// now - pending == grace (3), but we need >= grace to haul, so at exactly 3 it SHOULD haul.
	// The spec says nowRound-pending >= grace → haul.
	out := resolveArrest(false, true, 1000, 1003, 3)
	if out != arrestOutcomeHaul {
		t.Errorf("at-grace-boundary: got %v, want Haul (>= grace)", out)
	}
}

// One tick before grace expires: still no-op.
func TestResolveArrest_Surrender_OneTick_Before_Grace_NoOp(t *testing.T) {
	out := resolveArrest(false, true, 1000, 1002, 3)
	if out != arrestOutcomeNone {
		t.Errorf("one-tick-before-grace: got %v, want None", out)
	}
}

// ---------------------------------------------------------------------------
// firstFactionWithCell tests
// ---------------------------------------------------------------------------

func TestFirstFactionWithCell(t *testing.T) {
	origCell := cellRoomFn
	t.Cleanup(func() { cellRoomFn = origCell })

	cellRoomFn = func(faction string) int {
		if faction == "stillwater_guards" {
			return 5106
		}
		return 0 // citizens / others: no cell
	}

	got := firstFactionWithCell([]string{"stillwater_citizens", "stillwater_guards"})
	if got != "stillwater_guards" {
		t.Fatalf("firstFactionWithCell = %q, want stillwater_guards", got)
	}

	if firstFactionWithCell([]string{"stillwater_citizens"}) != "" {
		t.Fatalf("firstFactionWithCell should return \"\" when no faction owns a cell")
	}
}

// ---------------------------------------------------------------------------
// Original resolveWarn tests (unchanged).
// ---------------------------------------------------------------------------

func TestResolveWarn_FirstSighting_WarnsAndStamps(t *testing.T) {
	if out := resolveWarn(false, 0, 1000, 50); out != warnOutcomeWarn {
		t.Errorf("got %v, want warn", out)
	}
}

func TestResolveWarn_WithinGrace_NoOp(t *testing.T) {
	if out := resolveWarn(true, 1000, 1020, 50); out != warnOutcomeNone {
		t.Errorf("got %v, want none (within grace)", out)
	}
}

func TestResolveWarn_PastGrace_Escalates(t *testing.T) {
	if out := resolveWarn(true, 1000, 1060, 50); out != warnOutcomeAttack {
		t.Errorf("got %v, want attack (past grace)", out)
	}
}

func TestPruneStaleWarnStamps(t *testing.T) {
	md := map[string]any{
		"justice_warned_1":         uint64(100), // stale: now-100 > staleAfter(200)? 1000-100=900 > 200 -> pruned
		"justice_warned_2":         uint64(950), // fresh: 1000-950=50 <= 200 -> kept
		"justice_arrest_pending_3": uint64(100), // must be left alone
		"some_other_key":           uint64(100), // must be left alone
	}

	pruneStaleWarnStamps(md, 1000, 200) // now=1000, staleAfter=200 rounds

	if _, ok := md["justice_warned_1"]; ok {
		t.Fatalf("stale justice_warned_1 should have been pruned")
	}
	if _, ok := md["justice_warned_2"]; !ok {
		t.Fatalf("fresh justice_warned_2 must be kept")
	}
	if _, ok := md["justice_arrest_pending_3"]; !ok {
		t.Fatalf("arrest-pending stamp must not be touched")
	}
	if _, ok := md["some_other_key"]; !ok {
		t.Fatalf("unrelated key must not be touched")
	}
}

func TestMiscDataRound_ParsesNumericKinds(t *testing.T) {
	if r, ok := miscDataRound(map[string]any{"k": uint64(42)}, "k"); !ok || r != 42 {
		t.Errorf("uint64: got %d,%v", r, ok)
	}
	if r, ok := miscDataRound(map[string]any{"k": 42}, "k"); !ok || r != 42 {
		t.Errorf("int: got %d,%v", r, ok)
	}
	if r, ok := miscDataRound(map[string]any{"k": int64(42)}, "k"); !ok || r != 42 {
		t.Errorf("int64: got %d,%v", r, ok)
	}
	if r, ok := miscDataRound(map[string]any{"k": float64(42)}, "k"); !ok || r != 42 {
		t.Errorf("float64: got %d,%v", r, ok)
	}
	if _, ok := miscDataRound(map[string]any{}, "k"); ok {
		t.Error("missing key should return ok=false")
	}
}
