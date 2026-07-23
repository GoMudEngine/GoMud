package itemvoices

import (
	"slices"
	"testing"
)

func TestAllVoiceIds(t *testing.T) {
	defer SeedVoicesForTest(map[string]*VoiceSpec{
		"blackrazor": {VoiceId: "blackrazor"},
		"aegis":      {VoiceId: "aegis"},
	})()
	got := AllVoiceIds()
	if !slices.Equal(got, []string{"aegis", "blackrazor"}) {
		t.Errorf("want sorted [aegis blackrazor], got %v", got)
	}
}

func TestVoiceLineSelection(t *testing.T) {
	defer SeedVoicesForTest(map[string]*VoiceSpec{
		"blackrazor": {
			VoiceId: "blackrazor",
			Lines: map[string][]string{
				"on_kill":           {"Yes... YES. Another.", "It drinks well tonight."},
				"on_idle":           {"*a low obsidian hum*"},
				"on_hunger_warning": {"I hunger, bearer."},
			},
		},
	})()

	v := GetVoice("blackrazor")
	if v == nil {
		t.Fatal("voice not found")
	}
	line := v.Line("on_kill")
	if line != "Yes... YES. Another." && line != "It drinks well tonight." {
		t.Fatalf("unexpected line %q", line)
	}
	if v.Line("on_equip") != "" {
		t.Fatal("missing event should return empty string")
	}
	if GetVoice("nope") != nil {
		t.Fatal("unknown voice should be nil")
	}
}

func TestVoiceValidation(t *testing.T) {
	bad := &VoiceSpec{VoiceId: ""}
	if err := bad.Validate(); err == nil {
		t.Fatal("empty voiceid accepted")
	}
	badEvent := &VoiceSpec{VoiceId: "x", Lines: map[string][]string{"on_sneeze": {"a"}}}
	if err := badEvent.Validate(); err == nil {
		t.Fatal("unknown event key accepted")
	}
	empty := &VoiceSpec{VoiceId: "y", Lines: map[string][]string{"on_kill": {}}}
	if err := empty.Validate(); err == nil {
		t.Fatal("event with zero lines accepted")
	}
	good := &VoiceSpec{VoiceId: "z", Lines: map[string][]string{"on_kill": {"a line"}}}
	if err := good.Validate(); err != nil {
		t.Fatalf("valid voice rejected: %v", err)
	}
}
