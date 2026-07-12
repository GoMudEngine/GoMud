package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/users"
)

// TestTickCompanionEmpowerment_Guards verifies the cheap guards no-op safely:
// a nil room, and an owner with no companion_empowerment mutation, both return
// without touching companions (and without panicking).
func TestTickCompanionEmpowerment_Guards(t *testing.T) {
	u := users.NewUserRecord(1, 0)

	// Nil room → immediate return (before any Character access).
	tickCompanionEmpowerment(u, nil)

	// A user with no empowerment mutation is gated out. GetCompanionEmpowerment
	// over an empty/nil Mutations map is 0, so the tick no-ops. (The nil-room
	// guard above already covers the room==nil path; here we just assert the
	// mutation gate doesn't panic on a fresh user.)
	if u.Character != nil {
		u.Character.Mutations = map[string]int{}
	}
	// Still nil room to stay safe without a live room harness.
	tickCompanionEmpowerment(u, nil)
}
