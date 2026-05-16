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
	if len(templates.GradientMessages) == 0 {
		t.Skip("YAML config not present in this environment")
	}
	if _, ok := templates.GradientMessages["controller"]; !ok {
		t.Errorf("expected controller block in gradient_messages")
	}
	if _, ok := templates.GradientMessages["controlled"]; !ok {
		t.Errorf("expected controlled block in gradient_messages")
	}
	if templates.StaminaWarning.Self == "" {
		t.Errorf("expected non-empty stamina_warning.self")
	}
}

// TestGradientMessage_NoOpOnSameLevel verifies that fireGradientMessages
// does not set a cooldown when from == to (degenerate no-op).
func TestGradientMessage_NoOpOnSameLevel(t *testing.T) {
	c := characters.New()
	c.Position = position.NewMachine()
	c.PerGrappleMessageCooldowns = map[string]bool{}

	fireGradientMessages(c, position.InControl, position.InControl)

	if len(c.PerGrappleMessageCooldowns) != 0 {
		t.Errorf("expected no cooldown set on same-level no-op, got %v",
			c.PerGrappleMessageCooldowns)
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

// TestControlLevelKey_RoundTrip verifies the YAML key mapping covers
// every ControlLevel and never returns "" for a real level.
func TestControlLevelKey_RoundTrip(t *testing.T) {
	cases := []struct {
		level position.ControlLevel
		want  string
	}{
		{position.InControl, "in_control"},
		{position.LosingControl, "losing_control"},
		{position.Neutral, "neutral"},
		{position.BecomingControlled, "becoming_controlled"},
		{position.Controlled, "controlled"},
	}
	for _, tc := range cases {
		got := controlLevelKey(tc.level)
		if got != tc.want {
			t.Errorf("controlLevelKey(%v) = %q, want %q", tc.level, got, tc.want)
		}
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
