package statmods

// This contains centralized structs and constants regarding statmods
// Statmods are found in buffs, items, etc.
// They are used to augment in-game stats, calculations, etc.

// Statmods are a simple map of "name" to "modifier"
type StatMods map[string]int
type StatName string

var (
	// specific skills
	Picklock StatName = `picklock`

	// Not an exhaustive list, but ideally keep track of
	RacialBonusPrefix StatName = `racial-bonus-`

	// any statnames/prefixes here
	Casting            StatName = `casting`            // also used for `casting-` prefix followed by spell School
	CastingPrefix      StatName = `casting-`           // followed by spell School
	HealthRecovery     StatName = `healthrecovery`     // Augments HP recovery speed
	StaminaRecovery    StatName = `staminarecovery`    // Augments Stamina recovery speed
	ConvictionRecovery StatName = `convictionrecovery` // Augments Conviction recovery speed

	// Stat based
	Strength   StatName = `strength`
	Dexterity  StatName = `dexterity`
	Perception StatName = `perception`
	Vitality   StatName = `vitality`
	Willpower  StatName = `willpower`
	Charisma   StatName = `charisma`
	HealthMax  StatName = `healthmax`
	StaminaMax StatName = `staminamax`
)

func (s StatMods) Get(statName ...string) int {

	if len(s) == 0 {
		return 0
	}

	retAmt := 0

	for _, sn := range statName {
		if modAmt, ok := s[sn]; ok {
			retAmt += modAmt
		}
	}

	return retAmt
}

// Add accumulates statVal onto statName, allocating the map if it is nil.
//
// The receiver is a pointer specifically so the nil case works. With a value
// receiver, `s = make(StatMods)` assigned to the local parameter only — the new
// map never escaped, so Add on a nil StatMods silently discarded the value with
// no panic and no error. Every call site is an addressable field selector
// (spec.StatMods.Add(...)), so Go takes the address automatically and no caller
// needed changing.
func (s *StatMods) Add(statName string, statVal int) {
	if *s == nil {
		*s = make(StatMods)
	}

	// Missing keys read as the zero value, so this covers both branches.
	(*s)[statName] += statVal
}
