package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// MobDeathItemProcs fires on_kill procs and records the last-kill round
// (the Blackrazor hunger anchor — Task 11) for every player with damage
// attribution on the kill.
func MobDeathItemProcs(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.MobDeath)
	if !typeOk {
		return events.Continue
	}
	for uid := range evt.PlayerDamage {
		user := users.GetByUserId(uid)
		if user == nil {
			continue
		}
		user.Character.SetMiscData("pinnacle_last_kill_round", util.GetRoundCount())
		dispatchItemProcs("on_kill", user.Character, nil, nil, 0)
	}
	return events.Continue
}
