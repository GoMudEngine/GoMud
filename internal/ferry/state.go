package ferry

// VesselState is where a vessel is at a given round. When Docked,
// PortIdx is the berth. When at sea, PortIdx is the DESTINATION port.
type VesselState struct {
	Docked                bool
	PortIdx               int
	RoundsUntilTransition int
}

// hoursToRounds converts game-hours to rounds (integer truncation is fine
// — schedule precision of ±1 round is invisible to players).
func hoursToRounds(hours, roundsPerDay int) int {
	return hours * roundsPerDay / 24
}

// StateAt computes vessel state as a pure function of the round. This is
// the restart-safety property: no state is stored anywhere.
func StateAt(r Route, round uint64, roundsPerDay int) VesselState {
	lay := hoursToRounds(r.LayoverHours, roundsPerDay)
	cross := hoursToRounds(r.CrossingHours, roundsPerDay)
	cycle := 2 * (lay + cross)

	p := int((round + uint64(r.PhaseOffsetRounds)) % uint64(cycle))

	switch {
	case p < lay: // docked at port 0
		return VesselState{Docked: true, PortIdx: 0, RoundsUntilTransition: lay - p}
	case p < lay+cross: // at sea toward port 1
		return VesselState{Docked: false, PortIdx: 1, RoundsUntilTransition: lay + cross - p}
	case p < lay+cross+lay: // docked at port 1
		return VesselState{Docked: true, PortIdx: 1, RoundsUntilTransition: lay + cross + lay - p}
	default: // at sea toward port 0
		return VesselState{Docked: false, PortIdx: 0, RoundsUntilTransition: cycle - p}
	}
}

// NextDockedRound returns the earliest round >= fromRound at which the
// vessel is docked at portIdx. Precondition: portIdx must be 0 or 1 —
// the "unreachable" fallback below only holds for real ports, since a
// 2-port cycle always docks at both. Linear scan bounded by one cycle
// (<= a few hundred iterations) — called only on player asks,
// simplicity wins.
func NextDockedRound(r Route, portIdx int, fromRound uint64, roundsPerDay int) uint64 {
	cycle := uint64(2 * (hoursToRounds(r.LayoverHours, roundsPerDay) + hoursToRounds(r.CrossingHours, roundsPerDay)))
	for i := uint64(0); i <= cycle; i++ {
		s := StateAt(r, fromRound+i, roundsPerDay)
		if s.Docked && s.PortIdx == portIdx {
			return fromRound + i
		}
	}
	return fromRound // unreachable: a 2-port cycle always docks at both
}
