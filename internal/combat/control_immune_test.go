package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func TestExecuteSkillMove_ControlImmuneNeverKnockedDown(t *testing.T) {
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"immovable": {MutationId: "immovable", Name: "Immovable", Rarity: 9,
			Pros: []mutations.MutationEffect{{Type: "flag", Target: "control-immune"}}},
	})
	defer cleanup()

	atk := characters.New()

	params := func(def *characters.Character) SkillMoveParams {
		return SkillMoveParams{
			Attacker: atk, Defender: def,
			AttackStat: 300, AttackSkill: 50, // overwhelming, so attacks land
			DefenseStat: 1, DefenseSkill: 1,
			DamagePercent: 0.5, KnockdownChance: 100, // guaranteed knockdown on a hit
			SkillRank: 10, DamageStat: 100,
		}
	}

	// Control-immune defender: NEVER knocked down, across many attempts.
	immune := characters.New()
	immune.HealthMax.Value = 100000
	immune.Health = 100000
	immune.Mutations = map[string]int{"immovable": 1}
	for i := 0; i < 30; i++ {
		if ExecuteSkillMove(params(immune)).KnockedDown {
			t.Fatal("control-immune defender was knocked down")
		}
	}

	// Normal defender: gets knocked down at least once (proves the mechanic
	// works and immunity is the difference).
	normal := characters.New()
	normal.HealthMax.Value = 100000
	normal.Health = 100000
	knockedAtLeastOnce := false
	for i := 0; i < 30 && !knockedAtLeastOnce; i++ {
		if ExecuteSkillMove(params(normal)).KnockedDown {
			knockedAtLeastOnce = true
		}
	}
	if !knockedAtLeastOnce {
		t.Fatal("normal defender was never knocked down — test setup can't distinguish immunity")
	}
}
