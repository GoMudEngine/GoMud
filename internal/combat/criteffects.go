package combat

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// DisarmResult represents the outcome of a disarm attempt
type DisarmResult struct {
	Success     bool
	Weapon      items.Item
	Message     string // For disarmer
	TargetMsg   string // For disarmed
	RoomMessage string // For observers
}

// AttemptCritDisarm attempts to disarm a target with a given percentage chance.
// Checks for PermaGear immunity and valid weapon before attempting.
// Returns DisarmResult with success status and messages.
func AttemptCritDisarm(source *characters.Character, target *characters.Character, disarmChance float64) DisarmResult {
	result := DisarmResult{
		Success: false,
	}

	// Check for PermaGear buff immunity
	if target.HasBuffFlag(buffs.PermaGear) {
		return result
	}

	// Check if target has a weapon equipped (0 = unarmed, <0 = disabled slot)
	if target.Equipment.Weapon.ItemId <= 0 {
		return result
	}

	// Roll percentile chance
	success, _ := dice.Percentile(disarmChance)
	if !success {
		return result
	}

	// Disarm successful!
	result.Success = true
	result.Weapon = target.Equipment.Weapon

	// Generate messages
	weaponName := result.Weapon.NameSimple()
	result.Message = fmt.Sprintf(`<ansi fg="yellow-bold">You disarm %s, loosening their grip on their %s!</ansi>`, target.Name, weaponName)
	result.TargetMsg = fmt.Sprintf(`<ansi fg="red-bold">%s disarms you! Your %s slips from your grasp!</ansi>`, source.Name, weaponName)
	result.RoomMessage = fmt.Sprintf(`<ansi fg="combat">%s disarms %s, knocking their %s loose!</ansi>`, source.Name, target.Name, weaponName)

	// Remove weapon from equipped and put in inventory
	target.RemoveFromBody(result.Weapon)
	target.StoreItem(result.Weapon)

	return result
}

// SetGrappleOpportunity sets a 1-round grapple opportunity timer on a character.
// This grants a +15% grapple bonus on the next grapple attempt.
func SetGrappleOpportunity(char *characters.Character) {
	char.TimerSet("grapple-opportunity", "1 round")
}

// HasGrappleOpportunity checks if a character has an active grapple opportunity timer.
func HasGrappleOpportunity(char *characters.Character) bool {
	return char.TimerExists("grapple-opportunity") && !char.TimerExpired("grapple-opportunity")
}

// GetGrappleOpportunityBonus returns the grapple opportunity multiplier.
// Returns 1.15 if the character has an active opportunity, 1.0 otherwise.
func GetGrappleOpportunityBonus(char *characters.Character) float64 {
	if HasGrappleOpportunity(char) {
		return 1.15
	}
	return 1.0
}

// ClearGrappleOpportunity removes the grapple opportunity timer from a character.
func ClearGrappleOpportunity(char *characters.Character) {
	if char.Timers != nil {
		delete(char.Timers, "grapple-opportunity")
	}
}
