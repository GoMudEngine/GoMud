package position

// ShiftControl moves a ControlLevel by `delta` (positive = toward
// Controlled, negative = toward InControl), clamping to the
// [InControl, Controlled] range. Encodes the per-round drift
// arithmetic.
func ShiftControl(current ControlLevel, delta int) ControlLevel {
	// Enum order (post chunk-4a iota reorder):
	//   Neutral=0, InControl=1, LosingControl=2,
	//   BecomingControlled=3, Controlled=4
	//
	// Conceptual ordering (winning → losing):
	//   InControl < LosingControl < Neutral < BecomingControlled < Controlled
	//
	// "toward InControl" means moving leftward in the conceptual
	// ordering. The enum order doesn't match the conceptual order,
	// so we map to an integer rank and back.
	rank := controlRank(current)
	rank += delta // positive delta = toward Controlled (worse for the side)
	if rank < 0 {
		rank = 0
	}
	if rank > 4 {
		rank = 4
	}
	return controlFromRank(rank)
}

// controlRank maps ControlLevel to its position in the conceptual
// "winning → losing" gradient. 0 = InControl (best), 4 = Controlled
// (worst). Used by ShiftControl arithmetic.
func controlRank(c ControlLevel) int {
	switch c {
	case InControl:
		return 0
	case LosingControl:
		return 1
	case Neutral:
		return 2
	case BecomingControlled:
		return 3
	case Controlled:
		return 4
	}
	return 2 // safe default
}

// controlFromRank is the inverse of controlRank.
func controlFromRank(rank int) ControlLevel {
	switch rank {
	case 0:
		return InControl
	case 1:
		return LosingControl
	case 2:
		return Neutral
	case 3:
		return BecomingControlled
	case 4:
		return Controlled
	}
	return Neutral
}

// IsControllerLevel returns true if the given ControlLevel
// indicates the holder is in the "controller" role of the
// asymmetric grapple pair. By convention, ControlLevel ∈
// {InControl, LosingControl} = controller. Neutral is ambiguous
// (used for symmetric positions); caller's state context resolves.
func IsControllerLevel(c ControlLevel) bool {
	return c == InControl || c == LosingControl
}

// IsControlledLevel returns true if the given ControlLevel
// indicates the holder is in the "controlled" role.
func IsControlledLevel(c ControlLevel) bool {
	return c == BecomingControlled || c == Controlled
}

// MarginToDelta maps the |z-score| of an opposed roll outcome to
// the magnitude of ControlLevel shift per the 4b spec:
//
//	|z| range  | magnitude
//	0.0 – 0.5  | 0 (no shift)
//	0.5 – 1.0  | 1
//	1.0 – 2.0  | 2
//	≥ 2.0      | 3 (crit)
func MarginToDelta(absZScore float64) int {
	switch {
	case absZScore < 0.5:
		return 0
	case absZScore < 1.0:
		return 1
	case absZScore < 2.0:
		return 2
	default:
		return 3
	}
}

// ControlRankExported is the public-API wrapper for the internal
// controlRank helper. Used by btree primitives (T5) that compare
// ControlLevels in conceptual order.
func ControlRankExported(c ControlLevel) int {
	return controlRank(c)
}
