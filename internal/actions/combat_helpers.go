package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// AggroTarget represents the resolved target from a character's aggro.
type AggroTarget struct {
	Char          *characters.Character // Pointer to the target character
	Name          string                // Name of the target
	UserId        int                   // UserId if target is a player (0 if mob)
	MobInstanceId int                   // MobInstanceId if target is a mob (0 if player)
	Found         bool                  // True if target was successfully resolved
}

// ResolveAggroTarget resolves a character's current aggro target.
// Returns an AggroTarget struct with Found=true only if the target still exists.
// For players: Char is a pointer, UserId is set.
// For mobs: Char is a pointer (address of mob.Character), MobInstanceId is set.
func ResolveAggroTarget(aggro *characters.Aggro) AggroTarget {
	if aggro == nil {
		return AggroTarget{Found: false}
	}

	// Try mob target first
	if aggro.MobInstanceId > 0 {
		m := mobs.GetInstance(aggro.MobInstanceId)
		if m != nil {
			return AggroTarget{
				Char:          &m.Character,
				Name:          m.Character.Name,
				MobInstanceId: aggro.MobInstanceId,
				Found:         true,
			}
		}
	}

	// Then try player target
	if aggro.UserId > 0 {
		u := users.GetByUserId(aggro.UserId)
		if u != nil {
			return AggroTarget{
				Char:   u.Character,
				Name:   u.Character.Name,
				UserId: aggro.UserId,
				Found:  true,
			}
		}
	}

	return AggroTarget{Found: false}
}

// TryCombatCooldown checks if a character has a special move cooldown active.
// Returns true if the cooldown is active (move is blocked).
// Returns false if the cooldown was successfully set (move is allowed).
func TryCombatCooldown(char *characters.Character, cooldownRounds int) bool {
	if char == nil || char.Cooldowns == nil {
		return true // Blocked if character is invalid
	}

	// The Try method returns true if cooldown is NOT active (move allowed).
	// It returns false if cooldown IS active (move blocked).
	// We need to invert this for the caller.
	return !char.Cooldowns.Try("special-move", "1 rounds")
}

// RecordAndWait records a special combat move and marks the character
// as waiting one round (consumed by the special move).
// sourceType should be combat.User or combat.Mob.
func RecordAndWait(char *characters.Character, moveName string, sourceType combat.SourceTarget,
	targetChar *characters.Character, targetType combat.SourceTarget, hit bool, damage int, round uint64) {
	if char == nil {
		return
	}

	// Record the special move in combat analytics
	combat.RecordSpecialMove(sourceType, targetType, moveName, hit, damage, char, targetChar, round)

	// Mark that this character consumed their combat round
	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}
}
