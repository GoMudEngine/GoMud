package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type bountyKillKind int

const (
	killNone   bountyKillKind = iota
	killGuard                 // issuing-faction guard mob landed the killing blow
	killPlayer                // third-party player landed the kill or top damage
)

type bountyKill struct {
	kind          bountyKillKind
	mobInstanceId int
	userId        int
}

// attributeBountyKill decides who, if anyone, claims a bounty when the
// target player dies. issuerFaction is the bounty's issuing faction id
// (empty string for non-faction issuers). killerFactions resolves a mob
// instance's faction memberships (may be nil). Pure — no side effects.
func attributeBountyKill(
	killer state.ActorRef,
	issuerFaction string,
	damageMap map[int]int,
	killerFactions func(instanceId int) []string,
) bountyKill {
	// Guard-mob kill: killer is a mob that belongs to the issuing faction.
	if killer.IsMob() && killer.MobInstanceId > 0 && killerFactions != nil && issuerFaction != "" {
		for _, f := range killerFactions(killer.MobInstanceId) {
			if f == issuerFaction {
				return bountyKill{kind: killGuard, mobInstanceId: killer.MobInstanceId}
			}
		}
	}

	// Direct player kill.
	if killer.IsPlayer() && killer.UserId > 0 {
		return bountyKill{kind: killPlayer, userId: killer.UserId}
	}

	// Fallback: highest-damage player in the damage map.
	top, topDmg := 0, 0
	for uid, dmg := range damageMap {
		if dmg > topDmg {
			top, topDmg = uid, dmg
		}
	}
	if top > 0 {
		return bountyKill{kind: killPlayer, userId: top}
	}

	return bountyKill{kind: killNone}
}

// killerMobFactions resolves real mob-instance faction memberships via the
// factions registry. Used as the killerFactions callback in production.
func killerMobFactions(instanceId int) []string {
	inst := mobs.GetInstance(instanceId)
	if inst == nil {
		return nil
	}
	return factions.FactionsForMob(inst)
}

// wirePlayerDeathBountyResolve subscribes on the player's Alive→Dead
// transition and resolves any open bounties against that player.
// Mirrors wirePlayerDeathCleanup in Death_PlayerCleanup.go exactly.
func wirePlayerDeathBountyResolve(c *characters.Character) {
	c.Life.Inner().AfterTransition("player_death_bounty_resolve",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			// Only fire for player characters.
			if c.GetUserId() == 0 {
				return
			}
			userId := c.GetUserId()
			open := bounties.OpenAgainstPlayer(userId)
			if len(open) == 0 {
				return
			}

			d, _ := c.Life.DeadData()

			for _, b := range open {
				issuerFaction := ""
				if b.Issuer.Type == bounties.IssuerFaction {
					issuerFaction = b.Issuer.Id
				}

				bk := attributeBountyKill(d.Killer, issuerFaction, d.DamageMap, killerMobFactions)

				switch bk.kind {
				case killGuard:
					inst := mobs.GetInstance(bk.mobInstanceId)
					if inst == nil {
						bounties.MarkExpired(b.Id)
						continue
					}
					if _, ok := bounties.TryClaim(b.Id, knowledge.MobSubject(int(inst.MobId))); ok {
						inst.Character.Gold += b.GoldReward
					} else {
						// Claim lost to a race (already claimed/withdrawn);
						// don't leak the bounty open. Mirrors the killPlayer path.
						bounties.MarkExpired(b.Id)
					}

				case killPlayer:
					killer := users.GetByUserId(bk.userId)
					if killer == nil {
						bounties.MarkExpired(b.Id)
						continue
					}
					if claimed, ok := bounties.TryClaim(b.Id, knowledge.PlayerSubject(bk.userId)); ok {
						killer.Character.Gold += claimed.GoldReward
						if claimed.Issuer.Type == bounties.IssuerFaction {
							factions.BumpRep(claimed.Issuer.Id, bk.userId, claimed.RepReward)
						}
						// Mirror MobDeath_BountyClaim.go: CategoryLoot + gold figure.
						killer.SendText(messaging.CategoryLoot, fmt.Sprintf(
							"You collect a bounty: %dg.\r\n",
							claimed.GoldReward,
						))
					}

				default:
					bounties.MarkExpired(b.Id)
				}
			}
		})
}

func init() {
	characters.OnCharacterCreated(wirePlayerDeathBountyResolve)
}
