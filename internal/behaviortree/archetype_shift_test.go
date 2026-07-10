package behaviortree

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func TestValidateArchetypePullsCore(t *testing.T) {
	exists := func(path string) bool {
		return strings.Contains(path, "generic_fighter") || strings.Contains(path, "predator")
	}

	// Valid pulls: whitelisted + file exists.
	good := []*mutations.MutationSpec{
		{MutationId: "no-pull", Rarity: 3},
		{MutationId: "brawn", Rarity: 9, ArchetypePull: "generic_fighter"},
		{MutationId: "fangs", Rarity: 5, ArchetypePull: "predator"},
	}
	if err := validateArchetypePulls(good, exists); err != nil {
		t.Fatalf("valid pulls: unexpected error %v", err)
	}

	// Non-whitelisted target (boss archetype).
	bad := []*mutations.MutationSpec{
		{MutationId: "hubris", Rarity: 9, ArchetypePull: "boss_soren"},
	}
	if err := validateArchetypePulls(bad, exists); err == nil {
		t.Fatal("non-whitelisted pull: expected error, got nil")
	}

	// Whitelisted but no archetype file on disk.
	missing := []*mutations.MutationSpec{
		{MutationId: "ghost", Rarity: 9, ArchetypePull: "pure_caster"},
	}
	if err := validateArchetypePulls(missing, exists); err == nil {
		t.Fatal("missing archetype file: expected error, got nil")
	}
}
