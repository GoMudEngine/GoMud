package characters

// ArrestPolicy is the player's pre-decided response when a town guard attempts
// an arrest (Town Justice 5.1c). Surrender (default) = be jailed; Resist = the
// guard drops to lethal combat.
type ArrestPolicy string

const (
	ArrestSurrender ArrestPolicy = "surrender"
	ArrestResist    ArrestPolicy = "resist"
)

// ParseArrestPolicy parses a player-supplied policy string.
func ParseArrestPolicy(s string) (ArrestPolicy, bool) {
	switch ArrestPolicy(s) {
	case ArrestSurrender, ArrestResist:
		return ArrestPolicy(s), true
	}
	return "", false
}
