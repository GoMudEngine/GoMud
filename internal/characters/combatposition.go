package characters

// CombatPosition represents a character's current position in combat
// Stage 8.1: Only Standing and Prone are used (Clinched/Grounded in 8.2+)
type CombatPosition string

const (
	// PositionStanding - Default state, on feet, normal combat
	PositionStanding CombatPosition = "standing"

	// PositionProne - Knocked down (bash/trip), scrambling to stand
	// Replaces the old Prone boolean flag from Stage 7.5
	PositionProne CombatPosition = "prone"

	// PositionClinched - Standing grapple, locked up (Stage 8.2+)
	PositionClinched CombatPosition = "clinched"

	// PositionGrounded - Ground grapple, controller has advantage (Stage 8.2+)
	PositionGrounded CombatPosition = "grounded"
)

// IsGroundPosition returns true if the position involves being on the ground
// Used for applying ground-based penalties
func (p CombatPosition) IsGroundPosition() bool {
	return p == PositionProne || p == PositionGrounded
}

// IsGrapplePosition returns true if the position involves grappling (Stage 8.2+)
func (p CombatPosition) IsGrapplePosition() bool {
	return p == PositionClinched || p == PositionGrounded
}

// String returns the position as a string (for display)
func (p CombatPosition) String() string {
	return string(p)
}

// GetWorstPosition returns the "worse" position between two positions
// Used when multiple conditions apply (e.g., prone + grounded = show grounded)
// Ordering from best to worst: Standing > Prone > Clinched > Grounded
func GetWorstPosition(p1, p2 CombatPosition) CombatPosition {
	// Grounded is worst
	if p1 == PositionGrounded || p2 == PositionGrounded {
		return PositionGrounded
	}
	// Clinched is second worst
	if p1 == PositionClinched || p2 == PositionClinched {
		return PositionClinched
	}
	// Prone is third worst
	if p1 == PositionProne || p2 == PositionProne {
		return PositionProne
	}
	// Standing is best (default)
	return PositionStanding
}

// GetPositionColor returns the ANSI color tag for displaying the position
// Stage 8.1: Only Standing and Prone have colors (others added in 8.2+)
func (p CombatPosition) GetPositionColor() string {
	switch p {
	case PositionStanding:
		return "white" // Normal/neutral
	case PositionProne:
		return "yellow" // Warning - vulnerable
	case PositionClinched:
		return "orange" // Danger - restricted
	case PositionGrounded:
		return "red" // Critical - very vulnerable
	default:
		return "white"
	}
}
