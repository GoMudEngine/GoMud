package behaviortree

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// knownFields lists the structural YAML keys that should be stripped
// from Params before passing to conditions/actions.
var knownFields = map[string]struct{}{
	"type":     {},
	"event":    {},
	"children": {},
	"check":    {},
	"do":       {},
	"mod":      {},
	"child":    {},
}

// LoadTreeFromFile reads a YAML file and compiles it into a Node tree.
func LoadTreeFromFile(path string) (Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadTreeFromBytes(data)
}

// LoadArchetypeYAMLFromFile reads an archetype YAML file and returns
// both the compiled tree Node AND any chunk-4.2 goal_weights map
// declared at the top level.
func LoadArchetypeYAMLFromFile(path string) (Node, map[string]float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var def TreeDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, nil, fmt.Errorf("parse error: %w", err)
	}
	tree, err := compileNode(def.Tree, "root")
	if err != nil {
		return nil, nil, err
	}
	return tree, def.GoalWeights, nil
}

// LoadTreeFromBytes parses YAML bytes and compiles into a Node tree.
func LoadTreeFromBytes(data []byte) (Node, error) {
	var def TreeDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	return compileNode(def.Tree, "root")
}

// compileNode recursively converts a NodeDef into a concrete Node.
func compileNode(def NodeDef, path string) (Node, error) {
	var node Node
	var err error

	switch def.Type {
	case "selector":
		node, err = compileComposite(def, path, "selector")
	case "sequence":
		node, err = compileComposite(def, path, "sequence")
	case "condition":
		node, err = compileCondition(def, path)
	case "action":
		node, err = compileAction(def, path)
	case "decorator":
		node, err = compileDecorator(def, path)
	default:
		return nil, fmt.Errorf("%s: unknown node type %q", path, def.Type)
	}

	if err != nil {
		return nil, err
	}

	// Wrap in EventFilterNode if event is specified
	if def.Event != "" {
		node = &EventFilterNode{
			EventType: def.Event,
			Child:     node,
		}
	}

	return node, nil
}

func compileComposite(def NodeDef, path, kind string) (Node, error) {
	children := make([]Node, 0, len(def.Children))
	for i, childDef := range def.Children {
		childPath := fmt.Sprintf("%s.%s[%d]", path, kind, i)
		child, err := compileNode(childDef, childPath)
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}

	switch kind {
	case "selector":
		return &SelectorNode{Children: children}, nil
	default:
		return &SequenceNode{Children: children}, nil
	}
}

func compileCondition(def NodeDef, path string) (Node, error) {
	if def.Check == "" {
		return nil, fmt.Errorf("%s: condition node missing 'check' field", path)
	}
	fn := LookupCondition(def.Check)
	if fn == nil {
		return nil, fmt.Errorf("%s: unknown condition %q", path, def.Check)
	}
	return &ConditionNode{
		Name:   def.Check,
		Params: cleanParams(def),
		Fn:     fn,
	}, nil
}

func compileAction(def NodeDef, path string) (Node, error) {
	if def.Do == "" {
		return nil, fmt.Errorf("%s: action node missing 'do' field", path)
	}
	fn := LookupAction(def.Do)
	if fn == nil {
		return nil, fmt.Errorf("%s: unknown action %q", path, def.Do)
	}
	return &ActionNode{
		Name:   def.Do,
		Params: cleanParams(def),
		Fn:     fn,
	}, nil
}

func compileDecorator(def NodeDef, path string) (Node, error) {
	if def.Child == nil {
		return nil, fmt.Errorf("%s: decorator node missing 'child' field", path)
	}

	childPath := fmt.Sprintf("%s.decorator[0]", path)
	child, err := compileNode(*def.Child, childPath)
	if err != nil {
		return nil, err
	}

	params := cleanParams(def)

	switch def.Mod {
	case "cooldown":
		return &CooldownDecorator{
			Rounds:   getIntParam(params, "rounds"),
			StateKey: path + "_cooldown",
			Child:    child,
		}, nil
	case "repeat":
		return &RepeatDecorator{
			Times: getIntParam(params, "times"),
			Child: child,
		}, nil
	case "invert":
		return &InvertDecorator{
			Child: child,
		}, nil
	case "random":
		return &RandomDecorator{
			Percent: getIntParam(params, "percent"),
			Child:   child,
		}, nil
	case "delay":
		return &DelayDecorator{
			Rounds:   getIntParam(params, "rounds"),
			StateKey: path + "_delay",
			Child:    child,
		}, nil
	default:
		return nil, fmt.Errorf("%s: unknown decorator mod %q", path, def.Mod)
	}
}

// cleanParams returns a copy of the inline Params map with structural
// YAML keys removed, leaving only user-defined parameters.
func cleanParams(def NodeDef) map[string]any {
	result := make(map[string]any, len(def.Params))
	for k, v := range def.Params {
		if _, known := knownFields[k]; !known {
			result[k] = v
		}
	}
	return result
}
