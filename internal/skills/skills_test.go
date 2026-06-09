package skills

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/stats"
)

func TestGetTitle_IsTitleCased(t *testing.T) {
	var s stats.Statistics
	s.Strength.Value = 200
	s.Dexterity.Value = 100
	s.Perception.Value = 100
	s.Vitality.Value = 100
	s.Willpower.Value = 100
	s.Charisma.Value = 100

	got := GetTitle(map[string]int{}, map[string]int{}, s)
	if got != "Scrub Warrior" {
		t.Errorf("GetTitle = %q, want %q", got, "Scrub Warrior")
	}
}
