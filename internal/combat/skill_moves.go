package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// SkillMoveResult holds the outcome of a bash/kick/trip execution.
// Callers use these fields to dispatch messaging and analytics.
type SkillMoveResult struct {
	Hit         bool
	Damage      int
	KnockedDown bool
	TargetMaxHP int
}

// SkillMoveParams configures a skill move (bash/kick/trip) execution.
type SkillMoveParams struct {
	Attacker        *characters.Character
	Defender        *characters.Character
	AttackStat      int     // stat value for attack score (e.g. Strength or Dexterity)
	AttackSkill     int     // attacker's relevant skill level
	DefenseStat     int     // defender's Dexterity
	DefenseSkill    int     // defender's combat skill level
	DamagePercent   float64 // config knob (e.g. BashDamagePercent)
	KnockdownChance int     // config knob (e.g. BashKnockdownChance)
	SkillRank       int     // for SkillMultiplier in damage calc
	DamageStat           int     // stat for CalcRawDamage (always Strength)
	MitigationMultiplier float64 // 1.0 = full, 0.5 = half mitigation (stomp)

	// KnockdownToSupine: false (default) → defender falls face-forward
	// to Prone (TriggerKnockdownFaceForward). true → defender knocked
	// backward to Supine (TriggerKnockdownFaceBackward). Bash + charge
	// opt into the Supine path; trip / kick / hamstring / bite stay
	// face-forward.
	KnockdownToSupine bool
}

// ExecuteSkillMove performs the core combat resolution for bash/kick/trip.
// It handles the opposed roll, damage pipeline, knockdown determination,
// and applies HP reduction + prone status. Callers handle messaging and analytics.
func ExecuteSkillMove(p SkillMoveParams) SkillMoveResult {
	result := SkillMoveResult{}

	// Get target's max HP for damage descriptions
	result.TargetMaxHP = p.Defender.HealthMax.Value

	// Opposed roll: attacker skill+stat vs defender dex+combat skill
	attackerScore := float64(p.AttackSkill) + float64(p.AttackStat)
	defenderScore := float64(p.DefenseSkill) + float64(p.DefenseStat)
	attackSuccess, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

	// Damage pipeline: CalcRawDamage → ApplyMitigation → dice.RollStat
	rawDmg := CalcRawDamage(p.DamageStat, p.SkillRank, p.DamagePercent, ChannelPhysical)
	targetMitig := p.Defender.GetPhysicalMitigation()
	mitigMult := p.MitigationMultiplier
	if mitigMult <= 0 {
		mitigMult = 1.0
	}
	dmgMean := ApplyMitigation(rawDmg, targetMitig*mitigMult, MitigationCap(ChannelPhysical))
	dmgRoll := dice.RollStat(dmgMean)
	baseDamage := int(dmgRoll.Value)
	if baseDamage < 1 {
		baseDamage = 1
	}

	result.Hit = attackSuccess
	result.Damage = baseDamage

	if attackSuccess {
		// Knockdown roll — standardized to dice.RollStat(50)
		knockdownRoll := dice.RollStat(50)
		if knockdownRoll.Value < float64(p.KnockdownChance) {
			result.KnockedDown = true
		}

		// Apply damage
		p.Defender.Health -= baseDamage
		if p.Defender.Health < 0 {
			p.Defender.Health = 0
		}

		// Apply knockdown if rolled. Chunk 4b W4 cutover: fire the
		// FSM transition (Prone or Supine per KnockdownToSupine)
		// alongside the legacy CombatPosition / PositionRoundsMin
		// writes. If the FSM transition fails (e.g. target was
		// already grappling and not in Standing), the legacy fields
		// are NOT updated either so the two views stay consistent.
		if result.KnockedDown {
			var fsmErr error
			if p.KnockdownToSupine {
				fsmErr = p.Defender.Position.TransitionToSupine(
					position.SupineData{MinRecoveryRounds: 2},
					state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward},
				)
			} else {
				fsmErr = p.Defender.Position.TransitionToProne(
					position.ProneData{MinRecoveryRounds: 2},
					state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
				)
			}
			if fsmErr != nil {
				mudlog.Warn("ExecuteSkillMove: knockdown transition failed",
					"to_supine", p.KnockdownToSupine, "err", fsmErr)
			}
		}
	}

	return result
}
