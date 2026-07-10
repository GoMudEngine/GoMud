package mutations

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestArchetypePullYAMLRoundtrip(t *testing.T) {
	src := "mutationid: test-pull\nname: Test Pull\nrarity: 5\narchetype_pull: generic_fighter\n"
	var m MutationSpec
	if err := yaml.Unmarshal([]byte(src), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.ArchetypePull != "generic_fighter" {
		t.Fatalf("ArchetypePull = %q, want %q", m.ArchetypePull, "generic_fighter")
	}
}

func TestAllSpecsReturnsSeeded(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"a": {MutationId: "a", Rarity: 3},
		"b": {MutationId: "b", Rarity: 7},
	})
	defer cleanup()

	specs := AllSpecs()
	if len(specs) != 2 {
		t.Fatalf("AllSpecs len = %d, want 2", len(specs))
	}
}
