package behaviortree

import "testing"

// mockNode returns a fixed result.
type mockNode struct {
	result Result
	called bool
}

func (n *mockNode) Evaluate(ctx *EvalContext) Result {
	n.called = true
	return n.result
}

func TestSelector_ReturnsFirstSuccess(t *testing.T) {
	fail := &mockNode{result: Failure}
	pass := &mockNode{result: Success}
	skip := &mockNode{result: Success}

	sel := &SelectorNode{Children: []Node{fail, pass, skip}}
	result := sel.Evaluate(&EvalContext{})

	if result != Success {
		t.Errorf("expected Success, got %v", result)
	}
	if !fail.called || !pass.called {
		t.Error("first two children should have been called")
	}
	if skip.called {
		t.Error("third child should NOT have been called")
	}
}

func TestSelector_AllFail(t *testing.T) {
	sel := &SelectorNode{Children: []Node{
		&mockNode{result: Failure},
		&mockNode{result: Failure},
	}}
	if sel.Evaluate(&EvalContext{}) != Failure {
		t.Error("expected Failure when all children fail")
	}
}

func TestSequence_RunsAllOnSuccess(t *testing.T) {
	a := &mockNode{result: Success}
	b := &mockNode{result: Success}
	seq := &SequenceNode{Children: []Node{a, b}}

	if seq.Evaluate(&EvalContext{}) != Success {
		t.Error("expected Success")
	}
	if !a.called || !b.called {
		t.Error("both children should have been called")
	}
}

func TestSequence_StopsOnFailure(t *testing.T) {
	a := &mockNode{result: Success}
	b := &mockNode{result: Failure}
	c := &mockNode{result: Success}
	seq := &SequenceNode{Children: []Node{a, b, c}}

	if seq.Evaluate(&EvalContext{}) != Failure {
		t.Error("expected Failure")
	}
	if c.called {
		t.Error("third child should NOT have been called")
	}
}

func TestEventFilter_SkipsMismatch(t *testing.T) {
	inner := &mockNode{result: Success}
	filtered := &EventFilterNode{
		EventType: "player_ask",
		Child:     inner,
	}
	ctx := &EvalContext{Event: EventContext{EventType: "mob_idle"}}
	if filtered.Evaluate(ctx) != Failure {
		t.Error("expected Failure for mismatched event")
	}
	if inner.called {
		t.Error("child should not have been called")
	}
}

func TestEventFilter_PassesMatch(t *testing.T) {
	inner := &mockNode{result: Success}
	filtered := &EventFilterNode{
		EventType: "player_ask",
		Child:     inner,
	}
	ctx := &EvalContext{Event: EventContext{EventType: "player_ask"}}
	if filtered.Evaluate(ctx) != Success {
		t.Error("expected Success for matching event")
	}
	if !inner.called {
		t.Error("child should have been called")
	}
}

func TestLoadTree_SimpleSelector(t *testing.T) {
	yamlData := `
tree:
  type: selector
  children:
    - type: condition
      check: random_chance
      percent: 0
    - type: action
      do: set_state
      key: test
      value: passed
`
	node, err := LoadTreeFromBytes([]byte(yamlData))
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	state := NewBehaviorState()
	ctx := &EvalContext{MobState: state, Event: EventContext{EventType: "mob_idle"}}
	node.Evaluate(ctx)
	if state.GetString("test") != "passed" {
		t.Error("expected state 'test' to be 'passed'")
	}
}
