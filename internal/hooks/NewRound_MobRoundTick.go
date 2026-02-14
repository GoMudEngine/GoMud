package hooks

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func MobRoundTick(e events.Event) events.ListenerReturn {

	//
	// Reduce existing hostility (if any)
	//
	mobs.ReduceHostility()

	//
	// Do mob round maintenance
	//
	for _, mobInstanceId := range mobs.GetAllMobInstanceIds() {

		mob := mobs.GetInstance(mobInstanceId)

		if mob == nil {
			continue
		}

		// Roundtick any cooldowns
		mob.Character.Cooldowns.RoundTick()

		// Stage 7.5: Attempt automatic recovery from prone (uses DEX)
		if attemptMade, success := mob.Character.AttemptRecovery(mob.Character.Stats.Dexterity.ValueAdj); attemptMade {
			// Send messages to the room so players can see NPCs trying to recover
			if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
				if success {
					room.SendText("<ansi fg=\"mobname\">" + mob.Character.Name + "</ansi> clambers to their feet in a rushed panic.")
				} else {
					room.SendText("<ansi fg=\"mobname\">" + mob.Character.Name + "</ansi> attempts to stand, but slips and falls in the chaos of battle.")
				}
			}
		}

		if mob.Character.Charmed != nil && mob.Character.Charmed.RoundsRemaining > 0 {
			mob.Character.Charmed.RoundsRemaining--
		}

		if triggeredBuffs := mob.Character.Buffs.Trigger(); len(triggeredBuffs) > 0 {

			//
			// Fire onTrigger for buff script
			//
			triggeredBuffIds := []int{}
			for _, buff := range triggeredBuffs {
				scripting.TryBuffScriptEvent(`onTrigger`, 0, mobInstanceId, buff.BuffId)
				triggeredBuffIds = append(triggeredBuffIds, buff.BuffId)
			}

			events.AddToQueue(events.BuffsTriggered{MobInstanceId: mobInstanceId, BuffIds: triggeredBuffIds})
		}

		// Do charm cleanup
		if mob.Character.IsCharmed() && mob.Character.Charmed.RoundsRemaining == 0 {
			cmd := mob.Character.Charmed.ExpiredCommand
			if charmedUserId := mob.Character.RemoveCharm(); charmedUserId > 0 {
				if charmedUser := users.GetByUserId(charmedUserId); charmedUser != nil {
					charmedUser.Character.TrackCharmed(mob.InstanceId, false)
				}
			}
			if cmd != `` {
				cmds := strings.Split(cmd, `;`)
				for _, cmd := range cmds {
					cmd = strings.TrimSpace(cmd)
					if len(cmd) > 0 {
						mob.Command(cmd)
					}
				}
			}
		}

		// Stage 7.5: Clear recovery penalty flag at end of round
		mob.Character.RecoveryPenaltyThisRound = false

		// Stage 8.6: Clear defense penalty flag at end of round
		mob.Character.DefensePenaltyNextRound = false

		// Recalculate all stats at the end of the round tick
		mob.Character.Validate()

		if mob.Character.Health <= 0 {
			// Mob died
			mob.Command(`suicide`)
		}

	}

	return events.Continue
}
