package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
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
		// Chunk 4b R5: FSM-driven — IsStanding() is the new source of
		// truth (covers the legacy CombatPosition == PositionStanding
		// check). Sunset of the legacy enum happens in S5.
		return c.IsStanding()
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
		// Chunk 5 T6: block Engaging when the target is Disconnected or
		// Despawning. AFK / Idle / Dormant targets ARE attackable — "if
		// you went AFK in a dangerous room, you deserve it." Dormant mobs
		// will auto-wake on attack via the T7 wake-on-attack hook.
		if t.IsPlayer() {
			if u := users.GetByUserId(t.UserId); u != nil {
				switch u.Character.Presence.State() {
				case presence.Disconnected, presence.Despawning:
					return false
				}
			}
		}
		if t.IsMob() {
			if m := mobs.GetInstance(t.MobInstanceId); m != nil {
				switch m.Character.Presence.State() {
				case presence.Disconnected, presence.Despawning:
					return false
				}
			}
		}
		// Unknown / stale ref — allow defensively.
		return true
	})
}

func init() {
	characters.OnCharacterCreated(wireCombatPhaseVetoes)
}
