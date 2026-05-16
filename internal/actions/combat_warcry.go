package actions

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
)

// WarcryResult reports the outcome of a warcry cooldown+buff application.
type WarcryResult struct {
	Executed      bool    // true if the warcry actually applied
	OnCooldown    bool    // blocked by shared special-move cooldown
	Crafting      bool    // blocked because the actor is mid-craft
	AlreadyActive bool    // blocked because the warcry buff is already on this actor
	Bonus         float64 // damage bonus the condition carries (0.05..0.20)
	Duration      int     // condition duration in rounds
}

// ExecuteWarcry performs the cooldown check + self-buff application shared by
// both the player "warcry" command and the mob "warcry" command. Callers handle
// any fan-out (party members, companions, room broadcast) and player-facing
// text.
func ExecuteWarcry(actor Actor) WarcryResult {
	char := actor.GetCharacter()

	// War cry is a noisy action — reveal if hidden.
	if char.IsHidden() {
		char.Awareness.TransitionToRevealing(state.TransitionReason{
			Trigger:  awareness.TriggerNoisyAction,
			Metadata: map[string]any{"command": "warcry"},
		})
	}

	// IsActing applies universally — any active activity (cast/craft/salvage)
	// blocks warcry. Mobs can craft/cast too and should not interrupt their
	// activity to warcry.
	if char.IsActing() {
		return WarcryResult{Crafting: true}
	}

	// Skip if the warcry buff is already active on this actor —
	// re-casting would just burn the cooldown for no new effect.
	if char.HasBuff(79) {
		return WarcryResult{AlreadyActive: true}
	}

	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return WarcryResult{OnCooldown: true}
	}

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

	char.AddCondition(characters.ConditionWarcry, duration, bonus, "warcry")
	char.AddBuff(79, false)

	// Set combat wait if in combat (matches player + mob behavior).
	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	return WarcryResult{
		Executed: true,
		Bonus:    bonus,
		Duration: duration,
	}
}
