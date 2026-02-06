package combat

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func PowerRanking(atkChar characters.Character, defChar characters.Character) float64 {

	attacks, dCount, dSides, dBonus, _ := atkChar.Equipment.Weapon.GetDiceRoll()
	atkDmg := attacks * (dCount*dSides + dBonus)

	attacks, dCount, dSides, dBonus, _ = defChar.Equipment.Weapon.GetDiceRoll()
	defDmg := attacks * (dCount*dSides + dBonus)

	pct := 0.0
	if defDmg == 0 {
		pct += 0.4
	} else {
		pct += 0.4 * float64(atkDmg) / float64(defDmg)
	}

	if defChar.Stats.Dexterity.ValueAdj == 0 {
		pct += 0.3
	} else {
		pct += 0.3 * float64(atkChar.Stats.Dexterity.ValueAdj) / float64(defChar.Stats.Dexterity.ValueAdj)
	}

	if defChar.HealthMax.Value == 0 {
		pct += 0.2
	} else {
		pct += 0.2 * float64(atkChar.HealthMax.Value) / float64(defChar.HealthMax.Value)
	}

	if defChar.GetDefense() == 0 {
		pct += 0.1
	} else {
		pct += 0.1 * float64(atkChar.GetDefense()) / float64(defChar.GetDefense())
	}

	return pct
}

func ChanceToTame(s *users.UserRecord, t *mobs.Mob) int {

	var MOD_SKILL_MIN int = 1   // Minimum base tame ability
	var MOD_SKILL_MAX int = 100 // Maximum base tame ability

	var MOD_SIZE_SMALL int = 0    // Modifier for small creatures
	var MOD_SIZE_MEDIUM int = -10 // Modifier for medium creatures
	var MOD_SIZE_LARGE int = -25  // Modifier for large creatures

	var MOD_LEVELDIFF_MIN int = -25 // Lowest level delta modifier
	var MOD_LEVELDIFF_MAX int = 25  // Highest level delta modifier

	var MOD_HEALTHPERCENT_MAX float64 = 50 // Highest possible bonus for target HP being reduced

	var FACTOR_IS_AGGRO float64 = .50 // Overall reduction of chance if target is aggro

	proficiencyModifier := s.Character.MobMastery.GetTame(int(t.MobId))

	if proficiencyModifier < MOD_SKILL_MIN {
		proficiencyModifier = MOD_SKILL_MIN
	} else if proficiencyModifier > MOD_SKILL_MAX {
		proficiencyModifier = MOD_SKILL_MAX
	}

	speciesInfo := species.GetSpecies(s.Character.SpeciesId)

	sizeModifier := 0
	switch speciesInfo.Size {
	case species.Large:
		sizeModifier = MOD_SIZE_LARGE
	case species.Small:
		sizeModifier = MOD_SIZE_SMALL
	case species.Medium:
	default:
		sizeModifier = MOD_SIZE_MEDIUM
	}

	levelDiff := s.Character.Level - t.Character.Level
	if levelDiff > MOD_LEVELDIFF_MAX {
		levelDiff = MOD_LEVELDIFF_MAX
	} else if levelDiff < MOD_LEVELDIFF_MIN {
		levelDiff = MOD_LEVELDIFF_MIN
	}

	healthModifier := MOD_HEALTHPERCENT_MAX - math.Ceil(float64(s.Character.Health)/float64(s.Character.HealthMax.Value)*MOD_HEALTHPERCENT_MAX)

	var aggroModifier float64 = 1
	if t.Character.IsAggro(s.UserId, 0) {
		aggroModifier = FACTOR_IS_AGGRO
	}

	return int(math.Ceil((float64(proficiencyModifier) + float64(levelDiff) + healthModifier + float64(sizeModifier)) * aggroModifier))
}

func AlignmentChange(killerAlignment int8, killedAlignment int8) int {

	isKillerGood := killerAlignment > characters.AlignmentNeutralHigh
	isKillerEvil := killerAlignment < characters.AlignmentNeutralLow
	isKillerNeutral := killerAlignment >= characters.AlignmentNeutralLow && killerAlignment <= characters.AlignmentNeutralHigh

	isKilledGood := killedAlignment > characters.AlignmentNeutralHigh
	isKilledEvil := killedAlignment < characters.AlignmentNeutralLow
	isKilledNeutral := killedAlignment >= characters.AlignmentNeutralLow && killedAlignment <= characters.AlignmentNeutralHigh

	// Normalize the delta to positive, then half, so 0-100
	deltaAbs := math.Abs(math.Max(float64(killerAlignment), float64(killedAlignment))-math.Min(float64(killerAlignment), float64(killedAlignment))) * 0.5

	changeAmt := 0
	if deltaAbs <= 10 {
		changeAmt = 0
	} else if deltaAbs <= 30 {
		changeAmt = 1
	} else if deltaAbs <= 60 {
		changeAmt = 2
	} else if deltaAbs <= 80 {
		changeAmt = 3
	} else {
		changeAmt = 4
	}

	factor := 0

	if isKillerGood {

		if isKilledGood { // good vs good is especially evil
			factor = -2
			changeAmt = int(math.Max(float64(changeAmt), 1)) // At least 1 when killing own kind
		} else if isKilledEvil { // good vs evil is good
			factor = 1
		} else if isKilledNeutral { // good vs neutral is evil
			factor = -1
		}

	} else if isKillerEvil {

		if isKilledGood { // evil vs good is evil
			factor = -1
		} else if isKilledEvil { // evil vs evil is especially good
			factor = 2
			changeAmt = int(math.Max(float64(changeAmt), 1)) // At least 1 when killing own kind
		} else if isKilledNeutral { // evil vs neutral is evil
			factor = -1
		}

	} else if isKillerNeutral {

		if isKilledGood { // neutral vs good is evil
			factor = -1
		} else if isKilledEvil { // neutral vs evil is good
			factor = 1
		} else if isKilledNeutral { // neutral vs evil is nothing
			factor = 0
		}

	}

	return factor * changeAmt
}
