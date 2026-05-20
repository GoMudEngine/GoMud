// Package messaging owns the centralized player-facing-text pipeline.
//
// Every Room.SendText / Room.SendTextVisual / UserRecord.SendText call
// flows through this package's pipeline (compose → normalize →
// anonymize → color → wrap → deliver). Sites identify their text by
// Category; the pipeline resolves the color, applies style
// normalization, sight-gates visual content using the Perception FSM,
// and wraps to each recipient's LineWidth preference.
//
// See docs/superpowers/specs/2026-05-19-messaging-framework-design.md
// for the design and rationale.
package messaging

// Category enumerates every recognized text class. Used by the
// pipeline to look up a color alias and to drive per-Category
// normalization-skip behavior. Adding a new Category is a 2-line
// change (constant here + alias in ansi-aliases.yaml).
type Category int

const (
	CategoryDefault Category = iota

	// Combat — hits.
	CategoryHitMelee
	CategoryHitBlunt
	CategoryHitNaturalSharp
	CategoryHitRanged
	CategoryHitCaster
	CategoryHitUnarmed

	// Combat — defense.
	CategoryDodge
	CategoryParry
	CategoryBlock

	// Combat — grapple.
	CategoryGrappleFlow
	CategoryGrappleHigh

	// Combat — submissions / death.
	CategorySubmission
	CategoryDeath

	// Combat — special moves.
	CategorySurpriseAttack
	CategoryKick
	CategoryTrip
	CategoryBash
	CategoryRally
	CategoryWarcry
	CategoryTauntSuccess
	CategoryTauntResist
	CategoryTauntFailure

	// Spells.
	CategorySpellFold
	CategorySpellDisruption
	CategorySpellElemental
	CategorySpellEnhancement
	CategorySpellMental
	CategorySpellVital
	CategorySpellManifestation

	// Social.
	CategorySpeech
	CategoryWhisper
	CategoryShout
	CategoryOOC
	CategoryNPCDialogue
	CategoryDialogueHint
	CategoryEmote
	CategoryMobIdle
	CategoryMobEmote

	// System / meta.
	CategoryBroadcast
	CategoryTip
	CategorySystem
	CategoryError
	CategoryWarning
	CategorySkillProgress
	CategoryLogin
	CategoryLogout

	// Environment.
	CategoryRoomDescription
	CategoryRoomEntry
	CategoryRoomExit
	CategoryWeather
	CategoryTimeOfDay

	// Other.
	CategoryLoot
	CategoryEquipment
	CategoryBuffApply
	CategoryBuffExpire
	CategoryMutation
	CategoryToxin
)

// String returns a stable identifier for the category — used in
// logging and to key the color alias lookup in color.go.
func (c Category) String() string {
	switch c {
	case CategoryDefault:
		return "default"
	case CategoryHitMelee:
		return "hit-melee"
	case CategoryHitBlunt:
		return "hit-blunt"
	case CategoryHitNaturalSharp:
		return "hit-natural-sharp"
	case CategoryHitRanged:
		return "hit-ranged"
	case CategoryHitCaster:
		return "hit-caster"
	case CategoryHitUnarmed:
		return "hit-unarmed"
	case CategoryDodge:
		return "dodge"
	case CategoryParry:
		return "parry"
	case CategoryBlock:
		return "block"
	case CategoryGrappleFlow:
		return "grapple-flow"
	case CategoryGrappleHigh:
		return "grapple-high"
	case CategorySubmission:
		return "submission"
	case CategoryDeath:
		return "death"
	case CategorySurpriseAttack:
		return "surprise"
	case CategoryKick:
		return "kick"
	case CategoryTrip:
		return "trip"
	case CategoryBash:
		return "bash"
	case CategoryRally:
		return "rally"
	case CategoryWarcry:
		return "warcry"
	case CategoryTauntSuccess:
		return "taunt-success"
	case CategoryTauntResist:
		return "taunt-resist"
	case CategoryTauntFailure:
		return "taunt-failure"
	case CategorySpellFold:
		return "spell-fold"
	case CategorySpellDisruption:
		return "spell-disruption"
	case CategorySpellElemental:
		return "spell-elemental"
	case CategorySpellEnhancement:
		return "spell-enhancement"
	case CategorySpellMental:
		return "spell-mental"
	case CategorySpellVital:
		return "spell-vital"
	case CategorySpellManifestation:
		return "spell-manifestation"
	case CategorySpeech:
		return "speech"
	case CategoryWhisper:
		return "whisper"
	case CategoryShout:
		return "shout"
	case CategoryOOC:
		return "ooc"
	case CategoryNPCDialogue:
		return "npc-dialogue"
	case CategoryDialogueHint:
		return "dialogue-hint"
	case CategoryEmote:
		return "emote"
	case CategoryMobIdle:
		return "mob-idle"
	case CategoryMobEmote:
		return "mob-emote"
	case CategoryBroadcast:
		return "broadcast"
	case CategoryTip:
		return "tip"
	case CategorySystem:
		return "system"
	case CategoryError:
		return "error"
	case CategoryWarning:
		return "warning"
	case CategorySkillProgress:
		return "skill-progress"
	case CategoryLogin:
		return "login"
	case CategoryLogout:
		return "logout"
	case CategoryRoomDescription:
		return "room-description"
	case CategoryRoomEntry:
		return "room-entry"
	case CategoryRoomExit:
		return "room-exit"
	case CategoryWeather:
		return "weather"
	case CategoryTimeOfDay:
		return "time-of-day"
	case CategoryLoot:
		return "loot"
	case CategoryEquipment:
		return "equipment"
	case CategoryBuffApply:
		return "buff-apply"
	case CategoryBuffExpire:
		return "buff-expire"
	case CategoryMutation:
		return "mutation"
	case CategoryToxin:
		return "toxin"
	}
	return "Unknown"
}
