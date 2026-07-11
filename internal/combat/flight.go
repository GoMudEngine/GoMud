package combat

// flightEdge returns the signed edge applied to the ATTACKER's side of a melee
// opposed roll from a flight mismatch: a flyer beats the earthbound both when
// attacking (strike from angle) and when defending (dodge earthbound). When
// both or neither fly, the advantage cancels to zero.
func flightEdge(attackerFlying, defenderFlying bool, edge int) int {
	if attackerFlying && !defenderFlying {
		return edge
	}
	if defenderFlying && !attackerFlying {
		return -edge
	}
	return 0
}
