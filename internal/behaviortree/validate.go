package behaviortree

import (
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// nodeDefHasEvent reports whether def or any descendant declares the
// given event.
func nodeDefHasEvent(def NodeDef, event string) bool {
	if def.Event == event {
		return true
	}
	for _, child := range def.Children {
		if nodeDefHasEvent(child, event) {
			return true
		}
	}
	if def.Child != nil && nodeDefHasEvent(*def.Child, event) {
		return true
	}
	return false
}

// treeFileHasEvent parses a behavior-tree YAML file (uncompiled — no
// action/condition lookup) and reports whether any node declares the
// given event. Missing or unparsable files report false; load errors
// are surfaced by the normal tree-loading path, not here.
func treeFileHasEvent(path string, event string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var def TreeDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return false
	}
	return nodeDefHasEvent(def.Tree, event)
}

// validateAutoAggroBehaviorGates is the testable core of
// ValidateAutoAggroBehaviorGates. Path resolution is injected so tests
// can point at fixture files. Mirrors TryMobBehavior's resolution
// order: a per-mob tree, when present, shadows the archetype tree.
func validateAutoAggroBehaviorGates(templates []*mobs.Mob, behaviorPathFn func(m *mobs.Mob) string, archetypePathFn func(name string) string) []string {
	warnings := []string{}
	archetypeGated := map[string]bool{}

	for _, m := range templates {
		if !m.AutoAggro {
			continue
		}

		// Per-mob tree shadows the archetype entirely (TryMobBehavior order).
		perMobPath := behaviorPathFn(m)
		if _, err := os.Stat(perMobPath); err == nil {
			if treeFileHasEvent(perMobPath, "player_enter") {
				warnings = append(warnings, fmt.Sprintf(
					"mob %d (%s): hostile/auto_aggro auto-attack on player entry preempts the player_enter branch in its behavior tree %s — either flip to hostile: false (btree owns engagement) or drop the branch",
					int(m.MobId), m.Character.Name, perMobPath))
			}
			continue
		}

		if m.BehaviorArchetype == "" {
			continue
		}
		gated, seen := archetypeGated[m.BehaviorArchetype]
		if !seen {
			gated = treeFileHasEvent(archetypePathFn(m.BehaviorArchetype), "player_enter")
			archetypeGated[m.BehaviorArchetype] = gated
		}
		if gated {
			warnings = append(warnings, fmt.Sprintf(
				"mob %d (%s): hostile/auto_aggro auto-attack on player entry preempts the %q archetype's player_enter branch — either flip to hostile: false (btree owns engagement) or use an ungated archetype",
				int(m.MobId), m.Character.Name, m.BehaviorArchetype))
		}
	}
	return warnings
}

// ValidateAutoAggroBehaviorGates checks every mob template that
// auto-attacks players on room entry (`hostile:` / `auto_aggro:`)
// against its behavior tree. The go.go entry auto-attack fires BEFORE
// the tree's player_enter handler, so any engagement gating there
// (power-ratio checks, hidden-ambush conditions) is silently bypassed —
// almost always an authoring mistake (see the 2026-05-13 Thornwall
// highwayman bug). Logs one warning per offending mob and returns the
// warning count. Call once at startup, after mobs.LoadDataFiles().
func ValidateAutoAggroBehaviorGates() int {
	warnings := validateAutoAggroBehaviorGates(
		mobs.AllMobTemplates(),
		func(m *mobs.Mob) string {
			return GetBehaviorPath(int(m.MobId), m.Zone, m.Character.Name)
		},
		GetArchetypePath,
	)
	for _, w := range warnings {
		mudlog.Warn("ValidateAutoAggroBehaviorGates", "warning", w)
	}
	return len(warnings)
}
