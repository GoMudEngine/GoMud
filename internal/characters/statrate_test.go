package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// TestStatProgressionRateWired is a wiring smoke: the knob exists, defaults
// positive, and is read. Behavioral rate tuning is validated in playtest.
func TestStatProgressionRateWired(t *testing.T) {
	if r := float64(configs.GetBalanceConfig().StatProgressionRate); r <= 0 {
		t.Errorf("StatProgressionRate should default > 0, got %v", r)
	}
}
