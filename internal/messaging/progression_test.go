package messaging

import (
	"strings"
	"testing"
)

func TestFormatProgressionSkillNoTierChange(t *testing.T) {
	got := FormatProgression(ProgSkill, "Unarmed Combat", nil)
	if !strings.Contains(got, "SKILL ADVANCEMENT") {
		t.Errorf("missing SKILL ADVANCEMENT title: %q", got)
	}
	if !strings.Contains(got, "Unarmed Combat") {
		t.Errorf("missing skill name: %q", got)
	}
	if strings.Contains(got, "→") {
		t.Errorf("no tier change should mean no arrow, got %q", got)
	}
	if !strings.Contains(got, "━") {
		t.Errorf("missing banner rule: %q", got)
	}
}

func TestFormatProgressionStatNoTierChange(t *testing.T) {
	got := FormatProgression(ProgStat, "Strength", nil)
	if !strings.Contains(got, "STATISTIC INCREASED") {
		t.Errorf("missing STATISTIC INCREASED title: %q", got)
	}
	if !strings.Contains(got, "Strength") {
		t.Errorf("missing stat name: %q", got)
	}
}

func TestFormatProgressionTierCrossingIncludesThirdLine(t *testing.T) {
	got := FormatProgression(ProgSkill, "Unarmed Combat",
		&TierChange{From: "apprentice", To: "journeyman"})
	if !strings.Contains(got, "apprentice → journeyman") {
		t.Errorf("missing tier transition line: %q", got)
	}
}
