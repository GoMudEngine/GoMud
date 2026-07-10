package behaviortree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"gopkg.in/yaml.v2"
)

func parseTreeDef(t *testing.T, src string) TreeDef {
	t.Helper()
	var def TreeDef
	if err := yaml.Unmarshal([]byte(src), &def); err != nil {
		t.Fatalf("parseTreeDef: %v", err)
	}
	return def
}

func TestNodeDefHasEvent(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want bool
	}{
		{
			name: "top-level child branch",
			yaml: `
tree:
  type: selector
  children:
    - type: sequence
      event: player_enter
      children:
        - type: action
          do: attack
`,
			want: true,
		},
		{
			name: "nested under decorator child",
			yaml: `
tree:
  type: selector
  children:
    - type: decorator
      mod: invert
      child:
        type: sequence
        event: player_enter
        children:
          - type: action
            do: attack
`,
			want: true,
		},
		{
			name: "absent",
			yaml: `
tree:
  type: selector
  children:
    - type: action
      event: mob_idle
      do: attack
`,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			def := parseTreeDef(t, tc.yaml)
			if got := nodeDefHasEvent(def.Tree, "player_enter"); got != tc.want {
				t.Errorf("nodeDefHasEvent = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidateAutoAggroBehaviorGatesCore(t *testing.T) {
	dir := t.TempDir()

	gatedYAML := `
tree:
  type: selector
  children:
    - type: sequence
      event: player_enter
      children:
        - type: condition
          check: target_power_ratio_above
          value: 1.0
        - type: action
          do: attack
`
	plainYAML := `
tree:
  type: selector
  children:
    - type: action
      event: mob_combat_round
      do: attack
`
	mustWrite := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		return p
	}
	mustWrite("gated.yaml", gatedYAML)
	mustWrite("plain.yaml", plainYAML)
	perMobGated := mustWrite("77-shadow.yaml", gatedYAML)
	perMobPlain := mustWrite("78-shadow.yaml", plainYAML)

	newMob := func(id int, name string, autoAggro bool, archetype string) *mobs.Mob {
		m := &mobs.Mob{
			MobId:             mobs.MobId(id),
			AutoAggro:         autoAggro,
			BehaviorArchetype: archetype,
		}
		m.Character.Name = name
		return m
	}

	templates := []*mobs.Mob{
		newMob(1, "gated aggro", true, "gated"),     // WARN: hostile + gated archetype
		newMob(2, "plain aggro", true, "plain"),     // ok: archetype has no player_enter
		newMob(3, "gated passive", false, "gated"),  // ok: not auto-aggro
		newMob(4, "no archetype", true, ""),         // ok: no tree at all
		newMob(5, "missing file", true, "ghost"),    // ok here: missing archetype warned elsewhere
		newMob(77, "shadowed gated", true, "plain"), // WARN: per-mob tree has player_enter (shadows plain archetype)
		newMob(78, "shadowed plain", true, "gated"), // ok: per-mob tree without player_enter shadows gated archetype
	}

	behaviorPathFn := func(m *mobs.Mob) string {
		switch int(m.MobId) {
		case 77:
			return perMobGated
		case 78:
			return perMobPlain
		}
		return filepath.Join(dir, "does-not-exist", m.Character.Name+".yaml")
	}
	archetypePathFn := func(name string) string {
		return filepath.Join(dir, name+".yaml")
	}

	warnings := validateAutoAggroBehaviorGates(templates, behaviorPathFn, archetypePathFn)

	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d: %v", len(warnings), warnings)
	}
	joined := strings.Join(warnings, "\n")
	for _, wantSub := range []string{"gated aggro", "shadowed gated"} {
		if !strings.Contains(joined, wantSub) {
			t.Errorf("warnings missing mob %q: %v", wantSub, warnings)
		}
	}
	for _, badSub := range []string{"plain aggro", "gated passive", "no archetype", "missing file", "shadowed plain"} {
		if strings.Contains(joined, badSub) {
			t.Errorf("warnings should not include mob %q: %v", badSub, warnings)
		}
	}
}
