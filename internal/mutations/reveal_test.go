package mutations_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// Compile-time proof Gained satisfies events.Event WITHOUT mutations
// importing events (events→skills→mutations would cycle; the external
// test package may import both).
var _ events.Event = mutations.Gained{}

func TestGainedType(t *testing.T) {
	if got := (mutations.Gained{}).Type(); got != `MutationGained` {
		t.Errorf(`Gained.Type() = %q, want "MutationGained"`, got)
	}
}
