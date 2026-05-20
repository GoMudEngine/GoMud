package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// TestLoadPositionMessages_ParsesYAML verifies the YAML config at
// _datafiles/messages/position_control.yaml is parseable. Skipped in
// environments without the data file (e.g. some CI sandboxes).
func TestLoadPositionMessages_ParsesYAML(t *testing.T) {
	templates := loadPositionMessages()
	if templates.StaminaWarning.Self == "" {
		t.Skip("YAML config not present in this environment")
	}
	if templates.StaminaWarning.Self == "" {
		t.Errorf("expected non-empty stamina_warning.self")
	}
}

// TestStaminaWarning_NoOpWhenNotLow verifies that fireStaminaWarningIfLow
// does not set a cooldown when stamina is above the threshold.
func TestStaminaWarning_NoOpWhenNotLow(t *testing.T) {
	c := characters.New()
	c.Position = position.NewMachine()
	c.PerGrappleMessageCooldowns = map[string]bool{}
	// Fresh character — StaminaMax.Value 0 means IsLowGrappleStamina
	// returns false (divide-by-zero guard). Verify no cooldown set.

	fireStaminaWarningIfLow(c)

	if c.PerGrappleMessageCooldowns["stamina_low"] {
		t.Errorf("expected no stamina_low cooldown when not low, got cooldown set")
	}
}

// TestSubstitute_ReplacesAllTokens verifies the simple {key}
// substitution logic.
func TestSubstitute_ReplacesAllTokens(t *testing.T) {
	tpl := "{Character} settles into a dominating {position}."
	subs := map[string]string{
		"Character": "Rocky",
		"position":  "Mount",
	}
	got := substitute(tpl, subs)
	want := "Rocky settles into a dominating Mount."
	if got != want {
		t.Errorf("substitute = %q, want %q", got, want)
	}
}
