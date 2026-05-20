package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/messaging"
)

// categoryForAttack picks a messaging.Category for a swing's
// narration based on the AttackResult metadata. Used by the combat
// drainage points in NewRound_DoCombat_unified.go and
// NewRound_DoCombat_resolution.go that previously tagged everything
// as CategoryDefault (pass-through, no color) per the T11 deferral.
//
// Heuristic:
//   1. If a defense fired, prefer that defense's Category. The
//      defense is the round's most-narrated event in most templates.
//   2. Otherwise, classify by the first SwingEvent's AttackType
//      (weapon/unarmed/ranged) — a CategoryHit* for the right band.
//   3. Fall back to CategoryHitMelee as a sensible default — combat
//      prose should at least show in the combat hue band rather than
//      rendering uncolored.
//
// Heterogeneous drainage (e.g., the source's own room broadcast
// containing both the swing line AND a defense line) gets the
// dominant Category — not per-line perfect, but visibly distinct
// from pre-chunk-7 plain prose. Per-line tagging belongs at the
// combat package producer site and is left for a future pass.
//
// defenseSide is true for drainage that goes to the defender's
// side (MessagesToTarget, MessagesToTargetRoom). For that side, if
// a defense fired we prefer the defense color; the attacker's side
// (MessagesToSource, MessagesToSourceRoom) gets the hit Category
// even when a defense fired, because the attacker's prose is about
// the swing itself.
func categoryForAttack(res *combat.AttackResult, defenseSide bool) messaging.Category {
	if res == nil {
		return messaging.CategoryHitMelee
	}

	if defenseSide && res.DefenseUsed != combat.DefenseNone {
		switch res.DefenseUsed {
		case combat.DefenseDodge:
			return messaging.CategoryDodge
		case combat.DefenseParry:
			return messaging.CategoryParry
		case combat.DefenseBlock:
			return messaging.CategoryBlock
		}
	}

	// Classify by the first swing's attack type.
	for _, sw := range res.SwingEvents {
		switch sw.AttackType {
		case "weapon":
			return messaging.CategoryHitMelee
		case "unarmed":
			return messaging.CategoryHitUnarmed
		case "ranged":
			return messaging.CategoryHitRanged
		}
	}

	return messaging.CategoryHitMelee
}
