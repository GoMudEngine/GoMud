package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/species"
)

func TestApplyIntrinsicMutations_NilSpecies(t *testing.T) {
	c := &Character{}
	c.ApplyIntrinsicMutations(nil)
	if len(c.Mutations) != 0 {
		t.Errorf("nil species should leave Mutations empty, got %d entries", len(c.Mutations))
	}
}

func TestApplyIntrinsicMutations_EmptyIntrinsic(t *testing.T) {
	c := &Character{}
	sp := &species.Species{IntrinsicMutations: nil}
	c.ApplyIntrinsicMutations(sp)
	if len(c.Mutations) != 0 {
		t.Errorf("empty intrinsic should leave Mutations empty, got %d entries", len(c.Mutations))
	}
}

func TestApplyIntrinsicMutations_AddsToEmpty(t *testing.T) {
	c := &Character{}
	sp := &species.Species{IntrinsicMutations: map[string]int{"tail": 1, "keen-eyes": 1}}
	c.ApplyIntrinsicMutations(sp)
	if c.Mutations["tail"] != 1 {
		t.Errorf("tail should be 1, got %d", c.Mutations["tail"])
	}
	if c.Mutations["keen-eyes"] != 1 {
		t.Errorf("keen-eyes should be 1, got %d", c.Mutations["keen-eyes"])
	}
}

func TestApplyIntrinsicMutations_MutationImmuneSkipped(t *testing.T) {
	c := &Character{}
	// A mutation_immune construct must gain NO intrinsics even if the species
	// carries a kit (machines are not touched by the Chrysalis).
	sp := &species.Species{MutationImmune: true, IntrinsicMutations: map[string]int{"titan-growth": 1}}
	c.ApplyIntrinsicMutations(sp)
	if len(c.Mutations) != 0 {
		t.Errorf("mutation_immune species must gain no mutations, got %d entries", len(c.Mutations))
	}
}

func TestApplyIntrinsicMutations_StacksAdditively(t *testing.T) {
	c := &Character{Mutations: map[string]int{"tail": 1}}
	sp := &species.Species{IntrinsicMutations: map[string]int{"tail": 1}}
	c.ApplyIntrinsicMutations(sp)
	if c.Mutations["tail"] != 2 {
		t.Errorf("tail should stack to 2, got %d", c.Mutations["tail"])
	}
}

func TestApplyIntrinsicMutations_ClampsToCap(t *testing.T) {
	c := &Character{Mutations: map[string]int{"tail": 4}}
	sp := &species.Species{IntrinsicMutations: map[string]int{"tail": 2}}
	c.ApplyIntrinsicMutations(sp)
	if c.Mutations["tail"] != 4 {
		t.Errorf("tail should clamp at 4, got %d", c.Mutations["tail"])
	}
}
