package species

import (
	"testing"
)

func TestHasBodyPart_NilBodyParts_FailOpen(t *testing.T) {
	s := &Species{Name: "test", BodyParts: nil}
	if !s.HasBodyPart("arms") {
		t.Error("nil BodyParts should fail-open (return true)")
	}
}

func TestHasBodyPart_EmptySlice_GatesEverything(t *testing.T) {
	s := &Species{Name: "test", BodyParts: []string{}}
	if s.HasBodyPart("arms") {
		t.Error("explicit empty BodyParts should return false (incorporeal-style)")
	}
}

func TestHasBodyPart_PresentTag(t *testing.T) {
	s := &Species{Name: "test", BodyParts: []string{"arms", "legs"}}
	if !s.HasBodyPart("arms") {
		t.Error("present tag should return true")
	}
	if !s.HasBodyPart("legs") {
		t.Error("present tag should return true")
	}
}

func TestHasBodyPart_AbsentTag(t *testing.T) {
	s := &Species{Name: "test", BodyParts: []string{"arms", "legs"}}
	if s.HasBodyPart("tail") {
		t.Error("absent tag should return false")
	}
}

func TestHasAllBodyParts_EmptyRequirements(t *testing.T) {
	s := &Species{BodyParts: []string{"arms"}}
	if !s.HasAllBodyParts(nil) {
		t.Error("empty requirements should always return true")
	}
	if !s.HasAllBodyParts([]string{}) {
		t.Error("empty requirements should always return true")
	}
}

func TestHasAllBodyParts_AllPresent(t *testing.T) {
	s := &Species{BodyParts: []string{"arms", "hands", "legs"}}
	if !s.HasAllBodyParts([]string{"arms", "hands"}) {
		t.Error("all required parts present should return true")
	}
}

func TestHasAllBodyParts_SomeMissing(t *testing.T) {
	s := &Species{BodyParts: []string{"arms", "legs"}}
	if s.HasAllBodyParts([]string{"arms", "hands"}) {
		t.Error("missing required part should return false")
	}
}

func TestIsCanonicalBodyPart(t *testing.T) {
	valid := []string{"arms", "hands", "legs", "eyes", "mouth", "skin", "tail"}
	for _, v := range valid {
		if !IsCanonicalBodyPart(v) {
			t.Errorf("%q should be canonical", v)
		}
	}
	invalid := []string{"wings", "horns", "fins", "tentacle", ""}
	for _, v := range invalid {
		if IsCanonicalBodyPart(v) {
			t.Errorf("%q should NOT be canonical", v)
		}
	}
}
