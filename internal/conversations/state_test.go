package conversations

import (
	"testing"
)

func TestComputeNextLine_FiresLineWhenSpeakerMatches(t *testing.T) {
	ex := Exchange{
		Lines: []ConversationLine{
			{Speaker: "A", Text: "first"},
			{Speaker: "B", Text: "second"},
			{Speaker: "A", Text: "third"},
		},
	}
	// Role A, line_idx 0 → should fire (line 0 speaker is A)
	plan := computeConversationPlan(ex, "A", 0)
	if !plan.ShouldFire {
		t.Errorf("expected ShouldFire=true for A at line 0, got %+v", plan)
	}
	if plan.Text != "first" {
		t.Errorf("expected text 'first', got %q", plan.Text)
	}
	if plan.NextLineIdx != 1 {
		t.Errorf("expected NextLineIdx=1, got %d", plan.NextLineIdx)
	}
	if plan.IsFinalLine {
		t.Errorf("expected IsFinalLine=false at line 0 of 3, got true")
	}
}

func TestComputeNextLine_WaitsWhenSpeakerMismatches(t *testing.T) {
	ex := Exchange{
		Lines: []ConversationLine{
			{Speaker: "A", Text: "first"},
			{Speaker: "B", Text: "second"},
		},
	}
	// Role B, line_idx 0 → should NOT fire (line 0 speaker is A)
	plan := computeConversationPlan(ex, "B", 0)
	if plan.ShouldFire {
		t.Errorf("expected ShouldFire=false for B at line 0 (A's line), got %+v", plan)
	}
}

func TestComputeNextLine_DetectsFinalLine(t *testing.T) {
	ex := Exchange{
		Lines: []ConversationLine{
			{Speaker: "A", Text: "first"},
			{Speaker: "B", Text: "last"},
		},
	}
	plan := computeConversationPlan(ex, "B", 1)
	if !plan.IsFinalLine {
		t.Errorf("expected IsFinalLine=true at line 1 of 2, got %+v", plan)
	}
}

func TestComputeNextLine_OutOfRangeReturnsAbort(t *testing.T) {
	ex := Exchange{
		Lines: []ConversationLine{{Speaker: "A", Text: "only"}},
	}
	plan := computeConversationPlan(ex, "A", 999)
	if !plan.ShouldAbort {
		t.Errorf("expected ShouldAbort=true for out-of-range line_idx, got %+v", plan)
	}
}
