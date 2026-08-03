package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// refuseWhileBusy enforces activity exclusivity for focus-required commands:
// while the Activity machine is occupied (Crafting, Salvaging, or Casting),
// the command refuses with the standard focused-work message and the caller
// bails. REFUSE — not interrupt — is the project convention for everything
// except movement (go.go is the sole interrupter). This mirrors the gates
// already on cast, shoot, reload, sneak, and the 14 special-move verbs
// (actions.CommandIsReady's universal IsActing() check); the 2026-08-03
// audit found 13 active commands missing it.
func refuseWhileBusy(user *users.UserRecord, verb string) bool {
	if user.Character.IsActing() {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="red">You can't %s while focused on your work. Finish or be interrupted first.</ansi>`, verb))
		return true
	}
	return false
}
