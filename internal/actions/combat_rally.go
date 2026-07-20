package actions

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
)

// RallyResult reports the outcome of a rally cooldown+buff application.
type RallyResult struct {
	Executed      bool    // true if the rally actually applied
	OnCooldown    bool    // blocked by shared special-move cooldown
	Crafting      bool    // blocked because the actor is mid-craft
	AlreadyActive bool    // blocked because the rally buff is already on this actor
	Bonus         float64 // mitigation bonus the condition carries (0.05..0.20)
	Duration      int     // condition duration in rounds
}

// ExecuteRally performs the cooldown check + self-buff application shared by
// both the player "rally" command and the mob "rally" command. Callers handle
// any fan-out (party members, companions, room broadcast) and player-facing
// text.
func ExecuteRally(actor Actor) RallyResult {
	char := actor.GetCharacter()

	// Rallying is a noisy action — reveal if hidden.
	if char.IsHidden() {
		char.Awareness.TransitionToRevealing(state.TransitionReason{
			Trigger:  awareness.TriggerNoisyAction,
			Metadata: map[string]any{"command": "rally"},
		})
	}

	// IsActing applies universally — any active activity (cast/craft/salvage)
	// blocks rally. Mobs can craft/cast too and should not interrupt their
	// activity to rally.
	if char.IsActing() {
		return RallyResult{Crafting: true}
	}

	// Skip if the rally buff is already active on this actor —
	// re-casting would just burn the cooldown for no new effect.
	if char.HasBuff(80) {
		return RallyResult{AlreadyActive: true}
	}

	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return RallyResult{OnCooldown: true}
	}

	bonus, duration := ApplyRallyEffect(char)

	// Set combat wait if in combat (matches player + mob behavior).
	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	// See ExecuteWarcry: progression is owned here, not by the callers, so both
	// Actor implementations get it. The mob wrapper previously had none.
	awardRhetoricUse(actor, char)

	return RallyResult{
		Executed: true,
		Bonus:    bonus,
		Duration: duration,
	}
}

// ApplyRallyEffect computes the rally magnitude (rhetoric + charisma, then
// shout-amp scaled) and applies the rally condition + buff to char, returning
// the bonus and duration for the caller to fan out to allies. It performs NO
// cooldown or activity gating — ExecuteRally owns those. Exposed so the
// shout-stacking mutation (Resonant Larynx) can loose a rally as part of another
// shout in the same action, under that shout's single cooldown.
func ApplyRallyEffect(char *characters.Character) (float64, int) {
	// Magnitude: 0.05 + 0.15 * sqrt((rhetoric/75) * (charisma/175)), clamped.
	rhetoric := float64(char.GetSkillLevel(skills.Rhetoric))
	charisma := float64(char.Stats.Charisma.ValueAdj)
	bonus := 0.05 + 0.15*math.Sqrt((rhetoric/75.0)*(charisma/175.0))
	if bonus < 0.05 {
		bonus = 0.05
	}
	if bonus > 0.20 {
		bonus = 0.20
	}
	duration := 25

	// Booming Lungs (shout-amp) scales magnitude AND duration past the normal
	// cap — deliberately, that is the point of the amplification.
	if amp := mutations.GetShoutAmp(char.Mutations); amp > 0 {
		bonus *= 1.0 + amp
		duration = int(float64(duration) * (1.0 + amp))
	}

	char.AddCondition(characters.ConditionRally, duration, bonus, "rally")
	char.AddBuff(80, false)
	return bonus, duration
}
