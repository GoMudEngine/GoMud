package combat

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
)

// DefenseType represents the type of defense used (Stage 7.1)
type DefenseType string

const (
	DefenseNone  DefenseType = ""
	DefenseDodge DefenseType = "dodge"
	DefenseParry DefenseType = "parry"
	DefenseBlock DefenseType = "block"
)

// WeaponHitInfo tracks whether a specific weapon landed at least one hit
// during a combat round, for per-weapon skill progression.
type WeaponHitInfo struct {
	SkillTag string // e.g., "weapon-combat", "unarmed-combat", "ranged-combat"
	Hit      bool
	Crit     bool
	Fumble   bool
}

// SwingEvent captures per-swing analytics data for accurate hit rate tracking.
type SwingEvent struct {
	Hit           bool
	Crit          bool
	Fumble        bool
	DoubleFumble  bool
	DefenseCrit   bool
	Damage        int
	DamageReduced int
	DefenseUsed   DefenseType
	AttackZScore  float64
	DefenseZScore float64
	AttackType    string // "weapon", "unarmed", "ranged" — per-swing weapon type for analytics
}

// TaggedMessage pairs a combat narration line with the messaging
// Category that should color it. Multi-weapon characters (e.g., the
// 6-arm test character) produce heterogeneous batches — some swings
// hit, some are defended, some fumble — so per-line categorization
// is necessary for the new palette to render correctly. The hook
// drain iterates these tagged messages instead of guessing one
// Category for the whole batch.
type TaggedMessage struct {
	Category messaging.Category
	Text     string
}

type AttackResult struct {
	Hit                     bool  // defaults false
	Crit                    bool  // defaults false
	Fumble                  bool  // defaults false
	DoubleFumble            bool  // defaults false
	BuffSource              []int // defaults 0
	BuffTarget              []int // defaults 0
	DamageToTarget          int   // defaults 0
	DamageToTargetReduction int   // defaults 0
	DamageToSource          int             // defaults 0
	DamageToSourceReduction int             // defaults 0
	DefenseUsed             DefenseType     // Which defense avoided the hit (Stage 7.1)
	DefenseAttempts         []DefenseType   // Sequence of defenses attempted (Stage 7.1)
	DefenseZScore           float64         // Defense roll z-score (Stage 8.4)
	AttackZScore            float64         // Attack roll z-score (Stage 8.4)
	ParryCritDetected       bool            // Flag for parry crit → riposte
	DodgeCritDetected       bool            // Flag for dodge crit → auto-trip
	BlockCritDetected       bool            // Flag for block crit → auto-bash
	SwingEvents             []SwingEvent    // Per-swing analytics (Stage 30.2)
	WeaponHits              []WeaponHitInfo // Per-weapon hit tracking for skill progression
	DefenderWasAttacked     bool            // True if any swing was attempted against defender
	MessagesToSource        []TaggedMessage
	MessagesToTarget        []TaggedMessage
	MessagesToSourceRoom    []TaggedMessage
	MessagesToTargetRoom    []TaggedMessage
	MessagesToRoomOld       []TaggedMessage
}

func (a *AttackResult) SendToSource(cat messaging.Category, msg string) {
	a.MessagesToSource = append(a.MessagesToSource, TaggedMessage{Category: cat, Text: msg})
}

func (a *AttackResult) SendToTarget(cat messaging.Category, msg string) {
	a.MessagesToTarget = append(a.MessagesToTarget, TaggedMessage{Category: cat, Text: msg})
}

func (a *AttackResult) SendToSourceRoom(cat messaging.Category, msg string) {
	a.MessagesToSourceRoom = append(a.MessagesToSourceRoom, TaggedMessage{Category: cat, Text: msg})
}

func (a *AttackResult) SendToTargetRoom(cat messaging.Category, msg string) {
	a.MessagesToTargetRoom = append(a.MessagesToTargetRoom, TaggedMessage{Category: cat, Text: msg})
}

func (a *AttackResult) SendToRoomOld(cat messaging.Category, msg string) {
	a.MessagesToRoomOld = append(a.MessagesToRoomOld, TaggedMessage{Category: cat, Text: msg})
}

// CategoryForWeaponSubtype maps an items.ItemSubType to the right
// CategoryHit* damage-band. Used by the per-swing senders so each
// weapon's narration renders in its band color (warhammer → blunt,
// hook-spear → melee, knuckles → unarmed, etc.).
//
// Includes the literal "piercing" subtype as a synonym for Stabbing —
// the YAML data files use both spellings ("piercing" is the dominant
// term across spears / hooks / smuggler blades; the enum constant is
// historically "stabbing"). Keep both mapped to CategoryHitMelee until
// the YAML naming is normalized.
func CategoryForWeaponSubtype(sub items.ItemSubType) messaging.Category {
	switch sub {
	case items.Bludgeoning, items.Slam:
		return messaging.CategoryHitBlunt
	case items.Cleaving, items.Stabbing, items.Slashing, "piercing":
		return messaging.CategoryHitMelee
	case items.Claws, items.Bite, items.Sting, items.Gore:
		return messaging.CategoryHitNaturalSharp
	case items.Shooting, items.Whipping:
		return messaging.CategoryHitRanged
	case items.Wand, items.Sceptre, items.Staff:
		return messaging.CategoryHitCaster
	case items.Unarmed, items.Fist:
		return messaging.CategoryHitUnarmed
	}
	return messaging.CategoryHitMelee
}

// CategoryForDefenseVerb maps a defense verb ("dodge"/"parry"/"block")
// to the matching CategoryDodge/Parry/Block. Falls back to
// CategoryDodge for unknown verbs (e.g., "avoid"); not visually
// critical since unknown verbs already use the catch-all defense
// flavor.
func CategoryForDefenseVerb(verb string) messaging.Category {
	switch verb {
	case "dodge":
		return messaging.CategoryDodge
	case "parry":
		return messaging.CategoryParry
	case "block":
		return messaging.CategoryBlock
	}
	return messaging.CategoryDodge
}
