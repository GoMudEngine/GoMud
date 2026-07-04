package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
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

		// Sentient weapons savor the kill (paced by the shared chatter cooldown,
		// so this doesn't spam on multi-kill rounds). Sentient chatter is the
		// PinnacleItemsEnabled toggle's domain — same gate pinnacleUserTick reads.
		if bool(configs.GetConfig().GamePlay.PinnacleItemsEnabled) &&
			user.Character.Equipment.Weapon.ItemId > 0 {
			if wspec := user.Character.Equipment.Weapon.GetSpec(); wspec.VoiceId != "" {
				tryEmitVoice(user, nil, wspec, "on_kill")
			}
		}
	}
	return events.Continue
}
