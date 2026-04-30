package mutators

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestMutatorSpec_RegenMultiplierField(t *testing.T) {
	src := `mutatorid: test-sanctuary
regenmultiplier: 5.0
`
	var spec MutatorSpec
	if err := yaml.Unmarshal([]byte(src), &spec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if spec.MutatorId != "test-sanctuary" {
		t.Errorf("MutatorId = %q, want test-sanctuary", spec.MutatorId)
	}
	if spec.RegenMultiplier != 5.0 {
		t.Errorf("RegenMultiplier = %v, want 5.0", spec.RegenMultiplier)
	}
}

func TestMutatorSpec_RegenMultiplierDefaultsZero(t *testing.T) {
	src := `mutatorid: nonregen-mutator`
	var spec MutatorSpec
	if err := yaml.Unmarshal([]byte(src), &spec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if spec.RegenMultiplier != 0 {
		t.Errorf("RegenMultiplier = %v, want 0 (unset)", spec.RegenMultiplier)
	}
}
