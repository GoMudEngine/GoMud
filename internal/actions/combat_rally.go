package actions

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// RallyResult reports the outcome of a rally cooldown+buff application.
type RallyResult struct {
	Executed   bool    // true if the rally actually applied
	OnCooldown bool    // blocked by shared special-move cooldown
	Crafting   bool    // blocked because a player is mid-craft (player-only)
	Bonus      float64 // mitigation bonus the condition carries (0.05..0.20)
	Duration   int     // condition duration in rounds
}

// ExecuteRally performs the cooldown check + self-buff application shared by
// both the player "rally" command and the mob "rally" command. Callers handle
// any fan-out (party members, companions, room broadcast) and player-facing
// text.
func ExecuteRally(actor Actor) RallyResult {
	char := actor.GetCharacter()

	// IsCrafting applies to players only; mobs never craft.
	if actor.IsPlayer() && char.IsCrafting() {
		return RallyResult{Crafting: true}
	}

	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return RallyResult{OnCooldown: true}
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

	char.AddCondition(characters.ConditionRally, duration, bonus, "rally")
	char.AddBuff(80, false)

	// Set combat wait if in combat (matches player + mob behavior).
	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	return RallyResult{
		Executed: true,
		Bonus:    bonus,
		Duration: duration,
	}
}
