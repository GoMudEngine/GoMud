package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/facts"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// MobRoomChangeFactsAutoWithdraw withdraws active facts whose
// WithdrawOnRespawnOf matches the moving mob's template id. Fires
// on every RoomChange where MobInstanceId != 0; the cost is one
// map lookup (most facts won't have the bound field set).
func MobRoomChangeFactsAutoWithdraw(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok || evt.MobInstanceId == 0 {
		return events.Continue
	}
	mob := mobs.GetInstance(evt.MobInstanceId)
	if mob == nil {
		return events.Continue
	}
	facts.WithdrawAllBoundTo(int(mob.MobId))
	return events.Continue
}
