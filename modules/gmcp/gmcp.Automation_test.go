package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestBuildAutomationPayload_MacrosAndAliases(t *testing.T) {
	macros := map[string]string{"=1": "get all;wield sword"}
	aliases := map[string]string{"kk": "kick goblin"}
	p := buildAutomationPayload(macros, aliases, nil, nil)
	if len(p.Macros) != 1 || p.Macros[0].Key != "=1" || p.Macros[0].Commands != "get all;wield sword" {
		t.Fatalf("macro not mapped: %+v", p.Macros)
	}
	if len(p.Aliases) != 1 || p.Aliases[0].Name != "kk" || p.Aliases[0].Command != "kick goblin" {
		t.Fatalf("alias not mapped: %+v", p.Aliases)
	}
}

func TestBuildAutomationPayload_Ticks(t *testing.T) {
	ticks := []users.UserTick{{Id: 1, Name: "sip", Commands: "drink health", IntervalSec: 30, Enabled: true}}
	p := buildAutomationPayload(nil, nil, ticks, nil)
	if len(p.Ticks) != 1 || p.Ticks[0].Id != 1 || p.Ticks[0].IntervalSec != 30 || !p.Ticks[0].Enabled {
		t.Fatalf("tick not mapped: %+v", p.Ticks)
	}
}

func TestBuildAutomationPayload_Triggers(t *testing.T) {
	trigs := []users.UserTrigger{{Id: 1, Name: "heal", Pattern: "*bleeding*", ThenCmds: "cast heal", Enabled: true,
		Condition: &users.TriggerCondition{SourceKind: "pool", SourceKey: "hp", Op: "below", Values: []string{"30"}}}}
	p := buildAutomationPayload(nil, nil, nil, trigs)
	if len(p.Triggers) != 1 || p.Triggers[0].Pattern != "*bleeding*" || p.Triggers[0].Condition == nil ||
		p.Triggers[0].Condition.SourceKey != "hp" {
		t.Fatalf("trigger not mapped: %+v", p.Triggers)
	}
}
