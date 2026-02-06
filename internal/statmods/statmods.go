package statmods

// This contains centralized structs and constants regarding statmods
// Statmods are found in buffs, items, etc.
// They are used to augment in-game stats, calculations, etc.

// Statmods are a simple map of "name" to "modifier"
type StatMods map[string]int
type StatName string

var (
	// specific skills
	Tame     StatName = `tame`
	Picklock StatName = `picklock`

	// Not an exhaustive list, but ideally keep track of
	RacialBonusPrefix StatName = `racial-bonus-`

	// any statnames/prefixes here
	Casting            StatName = `casting`            // also used for `casting-` prefix followed by spell School
	CastingPrefix      StatName = `casting-`           // followed by spell School
	XPScale            StatName = `xpscale`            // Used for scaling xp after kills
	HealthRecovery     StatName = `healthrecovery`     // Augments HP recovery speed
	ManaRecovery       StatName = `manarecovery`       // Augments MP recovery speed
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
	ManaMax    StatName = `manamax`
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

func (s StatMods) Add(statName string, statVal int) {
	if s == nil {
		s = make(StatMods)
	}

	if oldVal, ok := s[statName]; ok {
		s[statName] = oldVal + statVal
	} else {
		s[statName] = statVal
	}
}
