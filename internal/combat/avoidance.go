package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// TrySpellDeflection rolls a defensive avoidance check for the target of a
// damage spell. Returns a damage multiplier: 1.0 (no deflection), the
// configured avoidance multiplier (partial deflection), or 0.0 (crit = full
// negation).
//
// Roll: Defender Perception + Spellcasting vs Attacker Willpower + Spellcasting.
//
// The defender always receives spellcasting and perception progression from
// the attempt regardless of outcome.
func TrySpellDeflection(attacker *characters.Character, defender *characters.Character, defenderUserId int) float64 {
	cfg := configs.GetBalanceConfig()
	skillWeight := float64(cfg.SkillWeight)

	atkSpellcasting := float64(attacker.GetSkillLevel(skills.Spellcasting)) * skillWeight
	attackScore := float64(attacker.Stats.Willpower.ValueAdj) + atkSpellcasting

	defSpellcasting := float64(defender.GetSkillLevel(skills.Spellcasting)) * skillWeight
	defenseScore := float64(defender.GetEffectivePerception()) + defSpellcasting

	success, _, _, defRoll := dice.OpposedRollStat(attackScore, defenseScore)

	defender.OnSkillUse(string(skills.Spellcasting), defenderUserId)
	defender.OnStatUse("perception", defenderUserId)

	if !success {
		if defRoll.ZScore >= 2.0 {
			return 0.0
		}
		return float64(cfg.SpellAvoidanceDamageMultiplier)
	}

	return 1.0
}

// TryStoicResolve rolls a defensive avoidance check for the target of a
// hostile rhetoric attack. Returns a damage multiplier: 1.0 (no resolve),
// the configured avoidance multiplier (partial), or 0.0 (crit = full
// negation).
//
// Roll: Defender Willpower + Rhetoric vs Attacker Charisma + Rhetoric.
//
// The defender always receives rhetoric and willpower progression from the
// attempt regardless of outcome.
func TryStoicResolve(attacker *characters.Character, defender *characters.Character, defenderUserId int) float64 {
	cfg := configs.GetBalanceConfig()
	skillWeight := float64(cfg.SkillWeight)

	atkRhetoric := float64(attacker.GetSkillLevel(skills.Rhetoric)) * skillWeight
	attackScore := float64(attacker.Stats.Charisma.ValueAdj) + atkRhetoric

	defRhetoric := float64(defender.GetSkillLevel(skills.Rhetoric)) * skillWeight
	defenseScore := float64(defender.Stats.Willpower.ValueAdj) + defRhetoric

	success, _, _, defRoll := dice.OpposedRollStat(attackScore, defenseScore)

	defender.OnSkillUse(string(skills.Rhetoric), defenderUserId)
	defender.OnStatUse("willpower", defenderUserId)

	if !success {
		if defRoll.ZScore >= 2.0 {
			return 0.0
		}
		return float64(cfg.RhetoricAvoidanceDamageMultiplier)
	}

	return 1.0
}
