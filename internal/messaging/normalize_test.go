package messaging

import "testing"

func TestNormalizeCapitalizesSentenceStart(t *testing.T) {
	got := Normalize(CategoryHitMelee, "in control, you press forward")
	if got[:1] != "I" {
		t.Errorf("expected capitalized start, got %q", got)
	}
}

func TestNormalizeAAnAgreement(t *testing.T) {
	got := Normalize(CategoryHitMelee, "with a aggressive posture")
	if got != "With an aggressive posture." {
		t.Errorf("a/an: got %q", got)
	}
}

func TestNormalizeCollapsesDuplicateWord(t *testing.T) {
	got := Normalize(CategoryHitMelee, "negligible damage damage on")
	if got != "Negligible damage on." {
		t.Errorf("dup-word: got %q", got)
	}
}

func TestNormalizeAppendsEndPunctuation(t *testing.T) {
	got := Normalize(CategoryHitMelee, "you strike deeply")
	if got != "You strike deeply." {
		t.Errorf("end-punct: got %q", got)
	}
}

func TestNormalizeSkipsForRoomDescription(t *testing.T) {
	// Room descriptions manage their own capitalization and prose
	// shape; normalization is disabled for them.
	in := "the road winds west"
	got := Normalize(CategoryRoomDescription, in)
	if got != in {
		t.Errorf("CategoryRoomDescription must skip normalization, got %q", got)
	}
}

func TestNormalizeSkipsBannersWithBoxRule(t *testing.T) {
	in := "━━━ banner ━━━"
	got := Normalize(CategoryHitMelee, in)
	if got != in {
		t.Errorf("banner line must skip end-punct, got %q", in)
	}
}

func TestNormalizeDoesNotDoublePunctuate(t *testing.T) {
	in := "You strike deeply."
	got := Normalize(CategoryHitMelee, in)
	if got != "You strike deeply." {
		t.Errorf("must not double-punctuate, got %q", got)
	}
}

func TestNormalizeIdempotent(t *testing.T) {
	once := Normalize(CategoryHitMelee, "with a aggressive posture")
	twice := Normalize(CategoryHitMelee, once)
	if once != twice {
		t.Errorf("normalize must be idempotent: %q vs %q", once, twice)
	}
}
