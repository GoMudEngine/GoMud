package banner

import (
	"strings"
	"testing"
)

func TestFormatSkillNoTier(t *testing.T) {
	got := Format(Skill, "Unarmed Combat", nil)
	if !strings.Contains(got, "SKILL ADVANCEMENT") {
		t.Errorf("missing SKILL ADVANCEMENT: %q", got)
	}
	if !strings.Contains(got, "Unarmed Combat") {
		t.Errorf("missing name: %q", got)
	}
	if strings.Contains(got, "→") {
		t.Errorf("no tier should mean no arrow: %q", got)
	}
}

func TestFormatStatNoTier(t *testing.T) {
	got := Format(Stat, "Strength", nil)
	if !strings.Contains(got, "STATISTIC INCREASED") {
		t.Errorf("missing STATISTIC INCREASED: %q", got)
	}
}

func TestFormatTierCrossingIncludesArrow(t *testing.T) {
	got := Format(Skill, "Unarmed Combat",
		&TierChange{From: "apprentice", To: "journeyman"})
	if !strings.Contains(got, "apprentice → journeyman") {
		t.Errorf("missing tier transition: %q", got)
	}
}
