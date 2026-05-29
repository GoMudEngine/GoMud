package justice

import "testing"

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
