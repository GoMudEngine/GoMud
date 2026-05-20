package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// wireMobKillCredit subscribes to Life Alive→Dead transitions on
// mob characters and awards kill credit to:
//   - Each player whose UserId appears in DeadData.DamageMap
//   - All party members of those players who are not already credited
//
// Also ends combat aggro for any killer who is currently targeting
// this mob.
//
// Migrated from internal/mobcommands/suicide.go lines 133-178 as
// part of chunk-2 Task 10.
func wireMobKillCredit(c *characters.Character) {
	c.Life.Inner().AfterTransition("mob_kill_credit",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			// Only fire for mob characters.
			if c.MobInstanceId == 0 {
				return
			}
			m := mobs.GetInstance(c.MobInstanceId)
			if m == nil {
				return
			}

			d, _ := c.Life.DeadData()
			if len(d.DamageMap) == 0 {
				return
			}

			inTraining := m.Character.Zone == `Training`

			// Award kill credit to direct attackers.
			for uid := range d.DamageMap {
				user := users.GetByUserId(uid)
				if user == nil {
					continue
				}

				// End combat aggro if the player is still targeting this mob.
				if user.Character.IsInCombat() {
					if user.Character.CurrentCombatTarget().MobInstanceId == m.InstanceId {
						user.Character.EndAggro()
					}
				}

				if !inTraining {
					user.Character.KD.AddMobKill(int(m.MobId))
					if user.Character.KD.GetMobKills(int(m.MobId)) == 1 {
						user.Character.OnFirstMobKill(user.UserId)
					}
				}
			}

			// Award kill credit to party members not already credited.
			partyMembersHandled := map[int]bool{}
			for uid := range d.DamageMap {
				partyMembersHandled[uid] = true
				p := parties.Get(uid)
				if p == nil {
					continue
				}
				for _, memberId := range p.GetMembers() {
					if partyMembersHandled[memberId] {
						continue
					}
					partyMembersHandled[memberId] = true
					partyUser := users.GetByUserId(memberId)
					if partyUser == nil {
						continue
					}
					if !inTraining {
						partyUser.Character.KD.AddMobKill(int(m.MobId))
						if partyUser.Character.KD.GetMobKills(int(m.MobId)) == 1 {
							partyUser.Character.OnFirstMobKill(partyUser.UserId)
						}
					}
				}
			}
		})
}

func init() {
	characters.OnCharacterCreated(wireMobKillCredit)
}
