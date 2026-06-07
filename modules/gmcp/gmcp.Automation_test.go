package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestBuildAutomationPayload_MacrosAndAliases(t *testing.T) {
	macros := map[string]string{"=1": "get all;wield sword"}
	aliases := map[string]string{"kk": "kick goblin"}
	p := buildAutomationPayload(macros, aliases, nil)
	if len(p.Macros) != 1 || p.Macros[0].Key != "=1" || p.Macros[0].Commands != "get all;wield sword" {
		t.Fatalf("macro not mapped: %+v", p.Macros)
	}
	if len(p.Aliases) != 1 || p.Aliases[0].Name != "kk" || p.Aliases[0].Command != "kick goblin" {
		t.Fatalf("alias not mapped: %+v", p.Aliases)
	}
}

func TestBuildAutomationPayload_Ticks(t *testing.T) {
	ticks := []users.UserTick{{Id: 1, Name: "sip", Commands: "drink health", IntervalSec: 30, Enabled: true}}
	p := buildAutomationPayload(nil, nil, ticks)
	if len(p.Ticks) != 1 || p.Ticks[0].Id != 1 || p.Ticks[0].IntervalSec != 30 || !p.Ticks[0].Enabled {
		t.Fatalf("tick not mapped: %+v", p.Ticks)
	}
}
