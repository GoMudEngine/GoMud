package combat

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

// ─── Integration: Combat Lifecycle (Pipeline) ────────────────────────────────
// Exercises: CalcRawDamage(), ApplyMitigation(), SkillMultiplier(),
// ResourceMultiplier(), dice.OpposedRollStat(), dice.RollStat()
//
// Note: calculateCombat() has deep dependencies (Aggro, rooms, etc.) that
// make it unsuitable for isolated tests. Instead, we exercise the full
// damage pipeline that calculateCombat() calls internally.

func TestIntegration_CombatLifecycle(t *testing.T) {
	// Simulate a multi-round combat through the damage pipeline
	atkStr := 120
	atkSkill := 20
	defDex := 100
	defPhysMit := 0.30 // 30% physical mitigation from gear

	atkHP := 200
	defHP := 200
	atkStamina := 100
	atkStaminaMax := 100

	rounds := 0
	for defHP > 0 && atkHP > 0 && rounds < 200 {
		rounds++

		// Attack roll: opposed stat check
		atkWins, _, _, _ := dice.OpposedRollStat(float64(atkStr), float64(defDex))
		if !atkWins {
			continue // miss
		}

		// Damage pipeline
		staminaMult := ResourceMultiplier(atkStamina, atkStaminaMax, 0.28)
		rawDmg := CalcRawDamage(atkStr, atkSkill, 1.0, ChannelPhysical)
		mitigated := ApplyMitigation(rawDmg, defPhysMit, MitigationCap(ChannelPhysical))
		finalDmg := mitigated * staminaMult

		// Roll for variance
		rolled := dice.RollStat(finalDmg)
		dmg := int(rolled.Value)
		if dmg < 1 {
			dmg = 1
		}

		defHP -= dmg

		// Stamina drain per attack
		if atkStamina > 0 {
			atkStamina -= 2
			if atkStamina < 0 {
				atkStamina = 0
			}
		}
	}

	// The fight should resolve within 200 rounds
	assert.Less(t, rounds, 200, "Combat should resolve in under 200 rounds")
	assert.LessOrEqual(t, defHP, 0, "Defender should be defeated")
}

func TestIntegration_CombatStaminaDepletion(t *testing.T) {
	// Verify ResourceMultiplier degrades as stamina drops
	fullMult := ResourceMultiplier(100, 100, 0.28)
	halfMult := ResourceMultiplier(50, 100, 0.28)
	emptyMult := ResourceMultiplier(0, 100, 0.28)

	assert.Equal(t, 1.0, fullMult, "Full stamina should give 1.0 multiplier")
	assert.Less(t, halfMult, fullMult, "Half stamina should reduce multiplier")
	assert.Less(t, emptyMult, halfMult, "Empty stamina should reduce further")
	assert.InDelta(t, 0.72, emptyMult, 0.01, "0% stamina should give ~0.72")

	// Verify the multiplier scales damage output
	baseDmg := CalcRawDamage(100, 20, 1.0, ChannelPhysical)
	fullDmg := baseDmg * fullMult
	depletedDmg := baseDmg * emptyMult

	assert.Greater(t, fullDmg, depletedDmg,
		"Damage at full stamina should exceed damage at empty stamina")
}

func TestIntegration_CombatSkillScaling(t *testing.T) {
	// Verify damage increases with skill rank through the full pipeline
	stat := 100
	itemMult := 1.0

	dmgRank0 := CalcRawDamage(stat, 0, itemMult, ChannelPhysical)
	dmgRank25 := CalcRawDamage(stat, 25, itemMult, ChannelPhysical)
	dmgRank50 := CalcRawDamage(stat, 50, itemMult, ChannelPhysical)

	assert.Less(t, dmgRank0, dmgRank25, "Rank 25 should deal more than rank 0")
	assert.Less(t, dmgRank25, dmgRank50, "Rank 50 should deal more than rank 25")

	// SkillMultiplier at rank 50 should be ~3.0
	mult50 := SkillMultiplier(50)
	assert.InDelta(t, 3.0, mult50, 0.01)

	// Damage at rank 50 should be ~3x damage at rank 0
	ratio := dmgRank50 / dmgRank0
	assert.InDelta(t, 3.0, ratio, 0.1,
		"Rank 50 damage should be ~3x rank 0 damage")
}

// ─── Integration: Damage Channels ────────────────────────────────────────────
// Exercises the 3-channel pipeline: Physical, Magical, Conviction

func TestIntegration_DamageChannels(t *testing.T) {
	// Physical: stat=100, rank=0, mult=1.0 → 100 * 1.0 * 1.0 * 0.30 = 30
	physRaw := CalcRawDamage(100, 0, 1.0, ChannelPhysical)
	assert.InDelta(t, 30.0, physRaw, 0.01, "Physical channel baseline")

	// Magical: stat=100, rank=0, mult=1.0 → 100 * 1.0 * 1.0 * 1.0 = 100
	magRaw := CalcRawDamage(100, 0, 1.0, ChannelMagical)
	assert.InDelta(t, 100.0, magRaw, 0.01, "Magical channel baseline")

	// Conviction: stat=100, rank=0, mult=0.5 → 100 * 1.0 * 0.5 * 1.0 = 50
	convRaw := CalcRawDamage(100, 0, 0.5, ChannelConviction)
	assert.InDelta(t, 50.0, convRaw, 0.01, "Conviction channel baseline")

	// A caster (high Willpower) vs a fighter (high Strength)
	casterWillpower := 150
	fighterStrength := 150

	casterDmg := CalcRawDamage(casterWillpower, 30, 1.0, ChannelMagical)
	fighterDmg := CalcRawDamage(fighterStrength, 30, 1.0, ChannelPhysical)

	// Magical has 1.0 scale vs Physical 0.30, so caster should deal more raw damage
	assert.Greater(t, casterDmg, fighterDmg,
		"Caster magical damage should exceed fighter physical damage at same stat/rank")

	// Verify dice.RollStat produces variance around the raw damage
	// Run multiple rolls and check they're distributed around the mean
	rolls := make([]float64, 200)
	for i := range rolls {
		result := dice.RollStat(physRaw)
		rolls[i] = result.Value
	}

	// Mean should be close to physRaw
	sum := 0.0
	for _, v := range rolls {
		sum += v
	}
	mean := sum / float64(len(rolls))
	assert.InDelta(t, physRaw, mean, physRaw*0.15,
		"Mean of rolled damage should be close to raw damage")
}

// ─── Integration: Defense and Mitigation ─────────────────────────────────────
// Exercises: GetPhysicalMitigation(), ApplyMitigation(), MitigationCap()

func TestIntegration_DefenseAndMitigation(t *testing.T) {
	// Create a character with equipment providing physical mitigation
	defender := characters.New()
	defender.Equipment.Body = items.Item{
		ItemId: 10,
		Spec: &items.ItemSpec{
			Type:               items.Body,
			PhysicalMitigation: 20,
			MagicalMitigation:  5,
		},
	}
	defender.Equipment.Legs = items.Item{
		ItemId: 11,
		Spec: &items.ItemSpec{
			Type:               items.Legs,
			PhysicalMitigation: 15,
			MagicalMitigation:  3,
		},
	}
	defender.Equipment.Head = items.Item{
		ItemId: 12,
		Spec: &items.ItemSpec{
			Type:               items.Head,
			PhysicalMitigation: 10,
		},
	}

	// Physical mitigation: 20 + 15 + 10 = 45 → 0.45
	physMit := defender.GetPhysicalMitigation()
	assert.InDelta(t, 0.45, physMit, 0.01,
		"Physical mitigation should sum equipment values")

	// Magical mitigation: 5 + 3 = 8 → 0.08
	magMit := defender.GetMagicalMitigation()
	assert.InDelta(t, 0.08, magMit, 0.01,
		"Magical mitigation should sum equipment values")

	// Apply mitigation to raw damage
	rawDmg := 100.0
	mitigated := ApplyMitigation(rawDmg, physMit, 0.75)
	expected := rawDmg * (1 - physMit)
	assert.InDelta(t, expected, mitigated, 0.01,
		"Mitigated damage should be raw * (1 - mitigation%)")

	// Test mitigation cap at 75%
	heavilyArmored := 0.90 // 90% mitigation from gear
	capped := ApplyMitigation(rawDmg, heavilyArmored, 0.75)
	assert.InDelta(t, 25.0, capped, 0.01,
		"Mitigation above 75% cap should be capped to 75%")

	// Verify each channel has the correct cap from config
	for _, ch := range []DamageChannel{ChannelPhysical, ChannelMagical, ChannelConviction} {
		cap := MitigationCap(ch)
		assert.Greater(t, cap, 0.0, "Mitigation cap should be positive")
		assert.LessOrEqual(t, cap, 1.0, "Mitigation cap should be <= 1.0")
	}
}

func TestIntegration_FullDamagePipeline(t *testing.T) {
	// End-to-end: stat → CalcRawDamage → ApplyMitigation → RollStat → final
	stat := 120
	skillRank := 25
	itemMult := 1.2
	physMitigation := 0.30 // 30% physical mitigation

	// Step 1: Raw damage
	rawDmg := CalcRawDamage(stat, skillRank, itemMult, ChannelPhysical)
	assert.Greater(t, rawDmg, 0.0, "Raw damage should be positive")

	// Step 2: Verify skill multiplier is applied
	expectedSkillMult := SkillMultiplier(skillRank)
	expectedRaw := float64(stat) * expectedSkillMult * itemMult * DamageScale(ChannelPhysical)
	assert.InDelta(t, expectedRaw, rawDmg, 0.01, "Raw damage should match formula")

	// Step 3: Apply mitigation
	mitigated := ApplyMitigation(rawDmg, physMitigation, MitigationCap(ChannelPhysical))
	assert.Less(t, mitigated, rawDmg, "Mitigated damage should be less than raw")
	assert.InDelta(t, rawDmg*0.70, mitigated, 0.01, "30% mitigation removes 30%")

	// Step 4: Roll for variance
	rolledValues := make([]float64, 100)
	for i := range rolledValues {
		result := dice.RollStat(mitigated)
		rolledValues[i] = result.Value
	}

	// All rolled values should be positive (damage can't go negative in practice)
	positiveCount := 0
	for _, v := range rolledValues {
		if v > 0 {
			positiveCount++
		}
	}
	assert.Greater(t, positiveCount, 80,
		"Most rolled values should be positive with these parameters")
}

func TestIntegration_OpposedCombatRolls(t *testing.T) {
	// Statistical test: stronger attacker should win more opposed rolls
	strongAtk := 150.0
	weakDef := 80.0

	wins := 0
	iterations := 1000
	for i := 0; i < iterations; i++ {
		atkWins, _, _, _ := dice.OpposedRollStat(strongAtk, weakDef)
		if atkWins {
			wins++
		}
	}

	winRate := float64(wins) / float64(iterations)
	assert.Greater(t, winRate, 0.60,
		"Attacker with 150 vs defender with 80 should win >60%% of the time")

	// Equal stats should produce roughly 50/50
	equalWins := 0
	for i := 0; i < iterations; i++ {
		atkWins, _, _, _ := dice.OpposedRollStat(100.0, 100.0)
		if atkWins {
			equalWins++
		}
	}

	equalRate := float64(equalWins) / float64(iterations)
	assert.InDelta(t, 0.50, equalRate, 0.10,
		"Equal stats should produce ~50%% win rate")
}

func TestIntegration_CritAndFumbleZScores(t *testing.T) {
	// Verify crit/fumble z-score thresholds via dice.RollStat
	crits := 0
	fumbles := 0
	iterations := 5000

	for i := 0; i < iterations; i++ {
		result := dice.RollStat(100.0)
		if result.ZScore >= 2.0 {
			crits++
		}
		if result.ZScore <= -2.0 {
			fumbles++
		}
	}

	critRate := float64(crits) / float64(iterations)
	fumbleRate := float64(fumbles) / float64(iterations)

	// Z-score >= 2.0: ~2.3% expected
	assert.InDelta(t, 0.023, critRate, 0.02,
		"Crit rate should be ~2.3%% (Z >= 2.0)")
	assert.InDelta(t, 0.023, fumbleRate, 0.02,
		"Fumble rate should be ~2.3%% (Z <= -2.0)")

	// At least some should occur in 5000 rolls
	assert.Greater(t, crits, 0, "At least one crit in 5000 rolls")
	assert.Greater(t, fumbles, 0, "At least one fumble in 5000 rolls")
}

func TestIntegration_ResourceMultiplierMonotonic(t *testing.T) {
	// Verify that ResourceMultiplier is monotonically non-increasing
	// as resource drops from 100% to 0%
	prev := ResourceMultiplier(100, 100, 0.28)
	for pct := 99; pct >= 0; pct-- {
		cur := ResourceMultiplier(pct, 100, 0.28)
		assert.LessOrEqual(t, cur, prev+0.001,
			"ResourceMultiplier should not increase as resource drops (at %d%%)", pct)
		prev = cur
	}

	// Verify it works for all three penalty types
	for _, penalty := range []float64{0.28, 0.28, 0.28} {
		full := ResourceMultiplier(100, 100, penalty)
		empty := ResourceMultiplier(0, 100, penalty)
		assert.Equal(t, 1.0, full)
		assert.InDelta(t, 1.0-penalty, empty, 0.01)
	}
}

func TestIntegration_SkillMultiplierCurve(t *testing.T) {
	// Verify the sqrt curve: mult = base + (max - base) * sqrt(rank / softCap)
	// base=1.0, max=3.0, softCap=50
	for rank := 0; rank <= 50; rank++ {
		got := SkillMultiplier(rank)
		expected := 1.0 + 2.0*math.Sqrt(float64(rank)/50.0)
		assert.InDelta(t, expected, got, 0.01,
			"SkillMultiplier(%d) should match sqrt curve", rank)
	}

	// Above soft cap should be clamped
	assert.InDelta(t, 3.0, SkillMultiplier(100), 0.01,
		"Above soft cap should be clamped to max")
}
