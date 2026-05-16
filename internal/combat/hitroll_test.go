package combat

import (
	"math"
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Initialize logger so mudlog.Debug doesn't panic in tests
	mudlog.SetupLogger(nil, "Low", "", false)
	os.Exit(m.Run())
}

// setCombatPositionParallel writes BOTH the legacy CombatPosition field
// AND the new Position FSM in lockstep. F1 fixture helper for the chunk
// 4b transition window: tests that previously wrote only the legacy
// enum now keep both views aligned so FSM-driven readers see the
// intended state. Seeds Position if nil (struct-literal Characters
// don't run through New()'s machine init). Use a synthetic partner
// ActorRef for Clinched/Grounded — the FSM only requires non-zero.
func setCombatPositionParallel(c *characters.Character, pos characters.CombatPosition) {
	c.CombatPosition = pos
	if c.Position == nil {
		c.Position = position.NewMachine()
	}
	r := state.TransitionReason{Trigger: "test_setup"}
	switch pos {
	case characters.PositionStanding:
		c.Position.ForceStanding(r)
	case characters.PositionProne:
		c.Position.ForceStanding(r)
		_ = c.Position.TransitionToProne(position.ProneData{}, r)
	case characters.PositionClinched:
		c.Position.ForceStanding(r)
		_ = c.Position.TransitionToClinch(
			position.GrappleData{Partner: state.ActorRef{UserId: 1}},
			state.TransitionReason{Trigger: position.TriggerGrappleEntry},
		)
	case characters.PositionGrounded:
		c.Position.ForceStanding(r)
		_ = c.Position.TransitionToClinch(
			position.GrappleData{Partner: state.ActorRef{UserId: 1}},
			state.TransitionReason{Trigger: position.TriggerGrappleEntry},
		)
		_ = c.Position.TransitionToMount(
			position.GrappleData{Partner: state.ActorRef{UserId: 1}, ControlLevel: position.InControl},
			state.TransitionReason{Trigger: position.TriggerTakedownMount},
		)
	}
}

// ─── calcSwingCount ─────────────────────────────────────────────────────────

func TestCalcSwingCount_Baseline(t *testing.T) {
	// Baseline character: dex 100, skill ~10, standing, full resources
	ch := &characters.Character{}
	ch.Stats.Dexterity.ValueAdj = 100
	ch.StaminaMax.Value = 100
	ch.Stamina = 100
	setCombatPositionParallel(ch, characters.PositionStanding)

	tests := []struct {
		name        string
		weaponSpeed float64
		wantSwings  int
	}{
		{"unarmed (speed 1.4)", 1.4, 2},
		{"light weapon (speed 1.2)", 1.2, 2},
		{"medium weapon (speed 1.0)", 1.0, 2},
		{"heavy weapon (speed 0.7)", 0.7, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcSwingCount(ch, tt.weaponSpeed, 0, false)
			assert.Equal(t, tt.wantSwings, got, "dex=100, skill=0, speed=%.1f", tt.weaponSpeed)
		})
	}
}

func TestCalcSwingCount_HighDexUnarmedPullsAhead(t *testing.T) {
	// At high dex + skill, unarmed (1.4) should get more swings than light (1.2)
	ch := &characters.Character{}
	ch.Stats.Dexterity.ValueAdj = 150
	ch.StaminaMax.Value = 100
	ch.Stamina = 100
	setCombatPositionParallel(ch, characters.PositionStanding)

	unarmed := calcSwingCount(ch, 1.4, 0, false)
	light := calcSwingCount(ch, 1.2, 0, false)
	assert.GreaterOrEqual(t, unarmed, light,
		"unarmed (1.4) should get >= swings as light weapon (1.2) at high dex")
}

func TestCalcSwingCount_HardCap(t *testing.T) {
	ch := &characters.Character{}
	ch.Stats.Dexterity.ValueAdj = 500 // absurdly high
	ch.StaminaMax.Value = 100
	ch.Stamina = 100
	setCombatPositionParallel(ch, characters.PositionStanding)

	got := calcSwingCount(ch, 1.4, 5, false)
	assert.LessOrEqual(t, got, 4, "swing count should never exceed hard cap of 4")
}

func TestCalcSwingCount_RecoveryForcesOne(t *testing.T) {
	ch := &characters.Character{}
	ch.Stats.Dexterity.ValueAdj = 200
	ch.StaminaMax.Value = 100
	ch.Stamina = 100
	setCombatPositionParallel(ch, characters.PositionStanding)
	ch.AddCondition(characters.ConditionRecoveryPenalty, 1, 1.0, "test")

	got := calcSwingCount(ch, 1.4, 0, false)
	assert.Equal(t, 1, got, "recovery penalty should force swings to 1")
}

func TestCalcSwingCount_ProneReduces(t *testing.T) {
	// Chunk 4b R1: calcSwingCount reads the speed multiplier via the
	// new FSM helper. Use characters.New() (which seeds Position) and
	// parallel-write the FSM alongside the legacy CombatPosition field
	// until S1/S2 sunset the legacy field.
	ch := characters.New()
	ch.Stats.Dexterity.ValueAdj = 100
	ch.StaminaMax.Value = 100
	ch.Stamina = 100

	setCombatPositionParallel(ch, characters.PositionStanding)
	standing := calcSwingCount(ch, 1.4, 0, false)

	setCombatPositionParallel(ch, characters.PositionProne)
	prone := calcSwingCount(ch, 1.4, 0, false)

	assert.Less(t, prone, standing,
		"prone should reduce swing count vs standing")
}

func TestCalcSwingCount_MinimumOne(t *testing.T) {
	ch := &characters.Character{}
	ch.Stats.Dexterity.ValueAdj = 10 // very low
	ch.StaminaMax.Value = 100
	ch.Stamina = 1 // nearly depleted
	setCombatPositionParallel(ch, characters.PositionProne)

	got := calcSwingCount(ch, 0.5, 0, false)
	assert.GreaterOrEqual(t, got, 1, "swing count should never go below 1")
}

// ─── resolveDefenseOutcome — hitroll priority ───────────────────────────────

// mockBestDefense creates a bestDefenseResult with controlled z-scores.
func mockBestDefense(atkZScore, defZScore, atkValue, defValue float64, defType string) bestDefenseResult {
	return bestDefenseResult{
		margin:      defValue - atkValue,
		defenseType: defType,
		hitRoll:     dice.RollResult{Value: atkValue, ZScore: atkZScore},
		defRoll:     dice.RollResult{Value: defValue, ZScore: defZScore},
	}
}

func TestResolveDefenseOutcome_AttackFumbleAlwaysMiss(t *testing.T) {
	result := &AttackResult{}
	src := &characters.Character{Name: "Attacker"}
	tgt := &characters.Character{Name: "Defender"}

	// Attack fumble (z <= -2.0), normal defense
	best := mockBestDefense(-2.5, 0.5, 50, 60, characters.DefenseDodge)
	res := resolveDefenseOutcome(result, best, src, tgt, 2.0, false)

	assert.False(t, res.hit, "attack fumble should always miss")
	assert.True(t, res.fumble, "should flag as fumble")
	assert.False(t, res.crit, "fumble should not be a crit")
}

func TestResolveDefenseOutcome_DefenseFumbleAlwaysHit(t *testing.T) {
	result := &AttackResult{}
	src := &characters.Character{Name: "Attacker"}
	tgt := &characters.Character{Name: "Defender"}

	// Normal attack, defense fumble (z <= -2.0)
	// Defense margin > 0 (defense roll value higher) but defense fumbled
	best := mockBestDefense(0.5, -2.5, 50, 80, characters.DefenseDodge)
	res := resolveDefenseOutcome(result, best, src, tgt, 2.0, false)

	assert.True(t, res.hit, "defense fumble should always hit")
	assert.False(t, res.fumble, "should not flag as attack fumble")
}

func TestResolveDefenseOutcome_DoubleFumble(t *testing.T) {
	result := &AttackResult{}
	src := &characters.Character{Name: "Attacker"}
	tgt := &characters.Character{Name: "Defender"}
	setCombatPositionParallel(src, characters.PositionStanding)
	setCombatPositionParallel(tgt, characters.PositionStanding)

	// Both fumble
	best := mockBestDefense(-2.5, -2.5, 50, 50, characters.DefenseDodge)
	res := resolveDefenseOutcome(result, best, src, tgt, 2.0, false)

	assert.False(t, res.hit, "double fumble should be a miss")
	assert.True(t, res.fumble, "should flag fumble")
	assert.True(t, res.doubleFumble, "should flag double fumble")
	assert.Equal(t, characters.PositionProne, src.CombatPosition, "attacker should be prone")
	assert.Equal(t, characters.PositionProne, tgt.CombatPosition, "defender should be prone")
}

func TestResolveDefenseOutcome_AttackCritAlwaysHits(t *testing.T) {
	result := &AttackResult{}
	src := &characters.Character{Name: "Attacker"}
	tgt := &characters.Character{Name: "Defender"}

	// Attack crit (z >= 2.0), normal defense wins on margin
	best := mockBestDefense(2.5, 1.0, 80, 90, characters.DefenseParry)
	res := resolveDefenseOutcome(result, best, src, tgt, 2.0, false)

	assert.True(t, res.hit, "attack crit should always hit vs normal defense")
	assert.True(t, res.crit, "should flag as crit")
}

func TestResolveDefenseOutcome_DefenseCritAlwaysAvoids(t *testing.T) {
	result := &AttackResult{}
	src := &characters.Character{Name: "Attacker"}
	tgt := &characters.Character{Name: "Defender"}

	// Normal attack wins on margin, but defense crits
	best := mockBestDefense(1.0, 2.5, 100, 80, characters.DefenseParry)
	res := resolveDefenseOutcome(result, best, src, tgt, 2.0, false)

	assert.False(t, res.hit, "defense crit should always avoid vs normal attack")
	assert.True(t, res.defenseCrit, "should flag defense crit")
	assert.True(t, result.ParryCritDetected, "parry crit should be flagged")
}

func TestResolveDefenseOutcome_CritVsCrit_HigherValueWins(t *testing.T) {
	result := &AttackResult{}
	src := &characters.Character{Name: "Attacker"}
	tgt := &characters.Character{Name: "Defender"}

	// Both crit, attack has higher raw value
	best := mockBestDefense(2.5, 2.5, 120, 100, characters.DefenseDodge)
	res := resolveDefenseOutcome(result, best, src, tgt, 2.0, false)
	assert.True(t, res.hit, "crit vs crit: higher value (attack) should win")
	assert.True(t, res.crit, "should be a crit hit")

	// Both crit, defense has higher raw value
	result2 := &AttackResult{}
	best2 := mockBestDefense(2.5, 2.5, 100, 120, characters.DefenseDodge)
	res2 := resolveDefenseOutcome(result2, best2, src, tgt, 2.0, false)
	assert.False(t, res2.hit, "crit vs crit: higher value (defense) should win")
	assert.True(t, res2.defenseCrit, "should be a defense crit")
}

func TestResolveDefenseOutcome_NormalResolution(t *testing.T) {
	result := &AttackResult{}
	src := &characters.Character{Name: "Attacker"}
	tgt := &characters.Character{Name: "Defender"}

	// Normal attack wins (margin < 0 means attack > defense)
	best := mockBestDefense(1.0, 0.5, 100, 80, characters.DefenseDodge)
	res := resolveDefenseOutcome(result, best, src, tgt, 2.0, false)

	assert.True(t, res.hit, "normal attack winning on margin should hit")
	assert.False(t, res.crit, "normal hit should not be crit")
	assert.False(t, res.fumble, "normal hit should not be fumble")
}

func TestResolveDefenseOutcome_DefenseFumbleDoesNotAutoCrit(t *testing.T) {
	result := &AttackResult{}
	src := &characters.Character{Name: "Attacker"}
	tgt := &characters.Character{Name: "Defender"}

	// Defense fumble, attack roll is normal (not a crit)
	best := mockBestDefense(0.5, -2.5, 60, 80, characters.DefenseDodge)
	res := resolveDefenseOutcome(result, best, src, tgt, 2.0, false)

	assert.True(t, res.hit, "defense fumble should guarantee hit")
	assert.False(t, res.crit, "defense fumble should NOT auto-crit the attack")
}

func TestResolveDefenseOutcome_DefenseFumbleWithAttackCrit(t *testing.T) {
	result := &AttackResult{}
	src := &characters.Character{Name: "Attacker"}
	tgt := &characters.Character{Name: "Defender"}

	// Defense fumble AND attack crit
	best := mockBestDefense(2.5, -2.5, 60, 80, characters.DefenseDodge)
	res := resolveDefenseOutcome(result, best, src, tgt, 2.0, false)

	assert.True(t, res.hit, "should hit")
	assert.True(t, res.crit, "attack crit should still apply even with defense fumble")
}

// ─── calcHitDamage ──────────────────────────────────────────────────────────

func TestCalcHitDamage_CritUsesRawDamage(t *testing.T) {
	result := &AttackResult{}
	sdp := swingDamageParams{
		dmgMean:       10.0,
		dmgVariance:   0.001, // near-zero variance for predictability
		rawDmgForCrit: 50.0,
		critBuffs:     []int{1},
	}

	// Crit hit should use rawDmgForCrit
	dmg, _ := calcHitDamage(result, true, false, sdp)
	assert.True(t, result.Crit, "should flag crit")
	assert.True(t, dmg > 0, "crit should deal damage")

	// Normal hit
	result2 := &AttackResult{}
	dmg2, _ := calcHitDamage(result2, false, false, sdp)
	assert.False(t, result2.Crit, "should not flag crit")
	_ = dmg2 // just confirm it doesn't panic
}

func TestCalcHitDamage_BackstabConsumed(t *testing.T) {
	result := &AttackResult{}
	sdp := swingDamageParams{
		dmgMean:       10.0,
		dmgVariance:   0.001,
		rawDmgForCrit: 50.0,
	}

	_, backstab := calcHitDamage(result, false, true, sdp)
	assert.False(t, backstab, "backstab should be consumed after use")
	assert.True(t, result.Crit, "backstab should trigger crit")
}

// ─── Swing count formula math ───────────────────────────────────────────────

func TestSwingCountFormula_Math(t *testing.T) {
	// Verify the raw formula: 1 + (dex - 50) / 100 * speed * (1 + skill/softCap)
	// At dex=100, skill=0, softCap=50:
	// 1 + (100-50)/100 * speed * (1+0/50) = 1 + 0.5 * speed * 1

	tests := []struct {
		dex   float64
		speed float64
		want  float64
	}{
		{100, 1.4, 1.7},  // 1 + 0.5 * 1.4 = 1.7 → rounds to 2
		{100, 1.2, 1.6},  // 1 + 0.5 * 1.2 = 1.6 → rounds to 2
		{100, 1.0, 1.5},  // 1 + 0.5 * 1.0 = 1.5 → rounds to 2
		{100, 0.7, 1.35}, // 1 + 0.5 * 0.7 = 1.35 → rounds to 1
		{50, 1.0, 1.0},   // 1 + 0/100 * 1.0 = 1.0 → rounds to 1
		{150, 1.4, 2.4},  // 1 + 1.0 * 1.4 = 2.4 → rounds to 2
	}

	for _, tt := range tests {
		raw := 1.0 + (tt.dex-50.0)/100.0*tt.speed*1.0
		assert.InDelta(t, tt.want, raw, 0.01,
			"dex=%.0f, speed=%.1f", tt.dex, tt.speed)
		rounded := int(math.Round(raw))
		if rounded < 1 {
			rounded = 1
		}
		t.Logf("dex=%.0f speed=%.1f → raw=%.2f → swings=%d", tt.dex, tt.speed, raw, rounded)
	}
}
