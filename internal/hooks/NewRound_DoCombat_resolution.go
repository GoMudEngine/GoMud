package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// File: NewRound_DoCombat_resolution.go
//
// Shared helper(s) used by the unified combat-round handler in
// NewRound_DoCombat_unified.go. Originally home to per-quadrant phase
// helpers extracted during the 1.2a god-function refactor; those have
// since been removed by Stage 2b of the combat-quadrant unification
// work. Only handleCombatWaitRound remains here because it is invoked
// from phase1WaitRound (in the unified handler) for every quadrant.

// handleCombatWaitRound handles the RoundsWaiting > 0 short-circuit
// shared by the unified combat handler across all four quadrants.
// Returns true if the caller should return immediately (i.e. the
// attacker is still waiting).
//
// attackerUser is non-nil when the attacker is a player (PvM/PvP).
// defenderUser is non-nil when the defender is a player (MvP/PvP).
// viewerUserId is the user ID passed to sendVisualRoomText and
// sendDarkRoomCombatFallback — always the user participant (0 if neither
// side is a player, e.g. MvM).
func handleCombatWaitRound(
	attackerChar *characters.Character,
	defenderChar *characters.Character,
	roleSource combat.SourceTarget,
	roleTarget combat.SourceTarget,
	attackerUser *users.UserRecord,
	defenderUser *users.UserRecord,
	attackerRoom *rooms.Room,
	defenderRoom *rooms.Room,
	viewerUserId int,
) bool {
	if attackerChar.Aggro.RoundsWaiting <= 0 {
		return false
	}
	mudlog.Debug(`RoundsWaiting`, `User`, attackerChar.Name, `Rounds`, attackerChar.Aggro.RoundsWaiting)
	attackerChar.Aggro.RoundsWaiting--

	roundResult := combat.GetWaitMessages(items.Wait, attackerChar, defenderChar, roleSource, roleTarget)

	for _, msg := range roundResult.MessagesToSource {
		if attackerUser != nil {
			attackerUser.SendText(msg)
		}
	}
	for _, msg := range roundResult.MessagesToTarget {
		if defenderUser != nil {
			defenderUser.SendText(msg)
		}
	}
	for _, msg := range roundResult.MessagesToSourceRoom {
		sendVisualRoomText(attackerRoom, msg, viewerUserId)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		sendVisualRoomText(defenderRoom, msg, viewerUserId)
	}
	sendDarkRoomCombatFallback(attackerRoom, viewerUserId)
	if defenderRoom != attackerRoom {
		sendDarkRoomCombatFallback(defenderRoom, viewerUserId)
	}
	return true
}
