package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// wireCombatPhaseVetoes registers the seven Combat Phase veto callbacks
// against the fields they consult on this character. Reads existing
// Character/Mob fields directly; later chunks will replace these as their
// machines land (e.g., RegisterLifeCheck will read
// Character.LifeMachine.State() == Alive once the Life machine ships).
func wireCombatPhaseVetoes(c *characters.Character) {
	c.CombatPhase.RegisterCombatantVeto(func() bool {
		return c.IsCombatant()
	})
	c.CombatPhase.RegisterActivityCheck(func() bool {
		// Per chunk-3 per-activity policy (spec section "Per-activity
		// interrupt policy" — choice C from brainstorm): casting is
		// itself a combat action and is EXEMPT from the combat-entry
		// veto (cast continues into / during combat; damage triggers
		// concentration break via separate path). Crafting and
		// salvaging block combat entry — caller must cancel first.
		return !c.IsCrafting() && !c.IsSalvaging()
	})
	c.CombatPhase.RegisterLifeCheck(func() bool {
		return c.Health > 0
	})
	c.CombatPhase.RegisterPositionCheck(func() bool {
		return c.CombatPosition == characters.PositionStanding
	})

	// Target-side checks — look up the target character at call time.
	c.CombatPhase.RegisterTargetCombatantCheck(func(t state.ActorRef) bool {
		if t.IsPlayer() {
			if u := users.GetByUserId(t.UserId); u != nil {
				return u.Character.IsCombatant()
			}
		}
		if t.IsMob() {
			if m := mobs.GetInstance(t.MobInstanceId); m != nil {
				return m.Character.IsCombatant()
			}
		}
		// Unknown / stale ref — allow defensively so we don't silently
		// block attacks against targets that just left the registry.
		return true
	})
	c.CombatPhase.RegisterTargetLifeCheck(func(t state.ActorRef) bool {
		if t.IsPlayer() {
			if u := users.GetByUserId(t.UserId); u != nil {
				return u.Character.Health > 0
			}
		}
		if t.IsMob() {
			if m := mobs.GetInstance(t.MobInstanceId); m != nil {
				return m.Character.Health > 0
			}
		}
		return true
	})
	c.CombatPhase.RegisterTargetPresenceCheck(func(t state.ActorRef) bool {
		if t.IsPlayer() {
			if u := users.GetByUserId(t.UserId); u != nil {
				// Presence machine arrives in a later chunk. For now gate
				// on the NoAggroTarget grace buff (respawn-grace mechanism).
				return !u.Character.HasBuffFlag(buffs.NoAggroTarget)
			}
		}
		// Mobs do not have a grace period; always present if alive.
		return true
	})
}

func init() {
	characters.OnCharacterCreated(wireCombatPhaseVetoes)
}
