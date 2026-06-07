package gmcp

import "testing"

func TestBuildAutomationPayload_MacrosAndAliases(t *testing.T) {
	macros := map[string]string{"=1": "get all;wield sword"}
	aliases := map[string]string{"kk": "kick goblin"}
	p := buildAutomationPayload(macros, aliases)
	if len(p.Macros) != 1 || p.Macros[0].Key != "=1" || p.Macros[0].Commands != "get all;wield sword" {
		t.Fatalf("macro not mapped: %+v", p.Macros)
	}
	if len(p.Aliases) != 1 || p.Aliases[0].Name != "kk" || p.Aliases[0].Command != "kick goblin" {
		t.Fatalf("alias not mapped: %+v", p.Aliases)
	}
}
