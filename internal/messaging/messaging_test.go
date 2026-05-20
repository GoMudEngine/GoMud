package messaging

import "testing"

func TestCategoryDefaultIsZero(t *testing.T) {
	if CategoryDefault != 0 {
		t.Fatalf("CategoryDefault must be zero-value (got %d)", CategoryDefault)
	}
}

func TestCategoryStringRoundTrip(t *testing.T) {
	cases := []Category{
		CategoryHitMelee, CategoryDodge, CategoryParry, CategoryBlock,
		CategoryGrappleFlow, CategorySurpriseAttack, CategoryRally,
		CategorySpellFold, CategorySpellElemental, CategorySpeech,
		CategoryWhisper, CategoryBroadcast, CategoryError,
		CategoryRoomDescription, CategorySkillProgress,
	}
	seen := map[string]Category{}
	for _, c := range cases {
		s := c.String()
		if s == "" || s == "Unknown" {
			t.Errorf("category %d returned %q", c, s)
		}
		if prev, ok := seen[s]; ok && prev != c {
			t.Errorf("category %q is ambiguous (%d and %d)", s, prev, c)
		}
		seen[s] = c
	}
}

func TestUnknownCategoryString(t *testing.T) {
	if Category(-1).String() != "Unknown" {
		t.Fatalf("negative Category should stringify to Unknown")
	}
}
