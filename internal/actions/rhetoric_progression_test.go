package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingActor wraps an Actor implementation and records OnSkillUse calls,
// so these tests assert the progression hook actually fires rather than
// asserting on a probabilistic skill-rank increase.
type recordingActor struct {
	Actor
	char       *characters.Character
	skillsUsed []string
}

func (r *recordingActor) GetCharacter() *characters.Character { return r.char }
func (r *recordingActor) OnSkillUse(skillName string) bool {
	r.skillsUsed = append(r.skillsUsed, skillName)
	return true
}

// TestRegression_WarcryRallyAwardRhetoric locks the fix for the 2026-07-20
// audit finding: warcry and rally left Rhetoric progression to their callers,
// unlike the other migrated special moves which award it inside actions/. The
// player wrappers implemented it; the mob wrappers never did, so mobs could
// warcry and rally indefinitely without ever building Rhetoric.
//
// Progression now lives in ExecuteWarcry/ExecuteRally, so every Actor
// implementation gets it. In combat it always fires, which is what these tests
// pin (the out-of-combat path is a 50% roll and is deliberately not asserted).
func TestRegression_WarcryRallyAwardRhetoric(t *testing.T) {
	newInCombatActor := func() *recordingActor {
		c := &characters.Character{
			Name:  "Test Subject",
			Aggro: &characters.Aggro{}, // non-nil Aggro => IsInCombat()
		}
		c.Validate()
		return &recordingActor{char: c}
	}

	t.Run("warcry_awards_rhetoric_in_combat", func(t *testing.T) {
		a := newInCombatActor()
		require.True(t, a.char.IsInCombat(), "precondition: actor must be in combat")

		res := ExecuteWarcry(a)
		require.True(t, res.Executed, "precondition: warcry must have executed")

		assert.Contains(t, a.skillsUsed, string(skills.Rhetoric),
			"warcry must award Rhetoric progression for every actor type")
	})

	t.Run("rally_awards_rhetoric_in_combat", func(t *testing.T) {
		a := newInCombatActor()
		require.True(t, a.char.IsInCombat(), "precondition: actor must be in combat")

		res := ExecuteRally(a)
		require.True(t, res.Executed, "precondition: rally must have executed")

		assert.Contains(t, a.skillsUsed, string(skills.Rhetoric),
			"rally must award Rhetoric progression for every actor type")
	})

	// A blocked execution must not award progression — otherwise a mob on
	// cooldown could farm Rhetoric by spamming the command.
	t.Run("blocked_warcry_awards_nothing", func(t *testing.T) {
		a := newInCombatActor()

		first := ExecuteWarcry(a)
		require.True(t, first.Executed)
		countAfterFirst := len(a.skillsUsed)

		second := ExecuteWarcry(a)
		require.False(t, second.Executed, "precondition: second warcry must be blocked")

		assert.Len(t, a.skillsUsed, countAfterFirst,
			"a blocked warcry must not award progression")
	})
}
