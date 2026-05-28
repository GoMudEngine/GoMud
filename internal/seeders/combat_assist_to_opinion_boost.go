package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

const ruleNameCombatAssistToOpinion = "combat_assist_to_opinion_boost"
const combatAssistBump = 3
const combatAssistCooldownRounds uint64 = 50

func init() {
	Register(ruleNameCombatAssistToOpinion, combatAssistToOpinion, "PlayerAttackedMob")
}

// combatAssistToOpinion: on PlayerAttackedMob, if the attacked mob is
// currently in combat with another non-player mob, bump the beneficiary
// mob's opinion of the player (+3) with a 50-round per-pair cooldown.
//
// Filter: beneficiary mob must NOT be hostile to the player
// (factions.IsPeacefulToward — prevents fake credit for attacking one
// of two mutually-hostile mobs that happen to be fighting each other).
//
// Shares the PlayerAttackedMob event with rule 6 (aggressive_action_to_revenge)
// — the multi-consumer payoff of Option B's events-everywhere architecture.
//
// Firer: internal/actions/attack.go + internal/actions/taunt.go (Task 3).
func combatAssistToOpinion(event events.Event) {
	pa, ok := event.(events.PlayerAttackedMob)
	if !ok {
		return
	}
	if pa.UserId == 0 || pa.MobInstanceId == 0 {
		return
	}

	// The mob the player attacked — we inspect its Aggro to find its
	// current target (the beneficiary the player is helping).
	attackerMob := mobs.GetInstance(pa.MobInstanceId)
	if attackerMob == nil {
		return
	}

	// Resolve the mob being attacked by the attackerMob. If nil or pointing
	// at a player (UserId != 0), this is not a mob-vs-mob fight; skip.
	beneficiaryId := resolveAttackerMobTarget(attackerMob.Character.Aggro)
	if beneficiaryId == 0 {
		return // not currently fighting a mob
	}

	beneficiary := mobs.GetInstance(beneficiaryId)
	if beneficiary == nil {
		return
	}

	// Only grant credit when the beneficiary is peaceful toward the player.
	// This prevents fake credit for attacking one of two mutually-hostile
	// mobs (both hostile mobs would otherwise trigger an opinion bump for
	// every player who joins the fight on either side).
	if !factions.IsPeacefulToward(beneficiary, pa.UserId) {
		return
	}

	// Per-(beneficiary, attackerMob) cooldown: once per 50 rounds.
	if !applyCooldown(beneficiary, ruleNameCombatAssistToOpinion,
		instanceIdAsKey(attackerMob.InstanceId), combatAssistCooldownRounds) {
		return // cooldown active for this pair
	}

	opinions.Bump(int(beneficiary.MobId), pa.UserId, combatAssistBump)
}

// resolveAttackerMobTarget returns the MobInstanceId from an Aggro
// pointer when the aggro target is a mob (MobInstanceId > 0 and UserId
// == 0). Returns 0 if the aggro is nil, unset, or pointing at a player.
func resolveAttackerMobTarget(a *characters.Aggro) int {
	if a == nil {
		return 0
	}
	// Aggro.UserId > 0 means the target is a player, not a mob.
	if a.UserId != 0 {
		return 0
	}
	return a.MobInstanceId
}
