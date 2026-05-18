package position

// SubmissionType enumerates the named submissions available in
// chunk 4d. Each position-state has 0..N submissions available to
// each role (top-attack from the controller side, bottom-attack from
// the controlled side); the per-round submission tick picks which to
// attempt via a round-robin tracker on Character.
type SubmissionType int

const (
	SubNone SubmissionType = iota
	SubArmbar
	SubRNC // Rear Naked Choke
	SubTriangle
	SubAmericana
	SubKimura
	SubOmoplata
	SubAnaconda
)

func (t SubmissionType) String() string {
	switch t {
	case SubArmbar:
		return "armbar"
	case SubRNC:
		return "rear-naked choke"
	case SubTriangle:
		return "triangle choke"
	case SubAmericana:
		return "americana"
	case SubKimura:
		return "kimura"
	case SubOmoplata:
		return "omoplata"
	case SubAnaconda:
		return "anaconda choke"
	default:
		return "submission"
	}
}

// TopSubmissionsForPosition returns the canonical top-attack subs
// available to the InControl side of the given position. Nil for
// positions with no top-attack subs (Standing, Prone, Supine,
// Turtle, Clinch).
func TopSubmissionsForPosition(s State) []SubmissionType {
	switch s {
	case BackStanding:
		return []SubmissionType{SubRNC}
	case Mount:
		return []SubmissionType{SubAmericana, SubTriangle, SubArmbar}
	case SideControl:
		return []SubmissionType{SubKimura, SubAmericana}
	case KneeOnBelly:
		return []SubmissionType{SubArmbar}
	case NorthSouth:
		return []SubmissionType{SubKimura, SubAnaconda}
	case Crucifix:
		return []SubmissionType{SubArmbar}
	case BackGround:
		return []SubmissionType{SubRNC}
	case HalfGuard:
		return []SubmissionType{SubKimura}
	case Guard:
		// In the FSM the Guard-bottom is the controller; the
		// "top-attack" subs from Guard are the bottom-game subs
		// (Triangle / Armbar / Omoplata).
		return []SubmissionType{SubTriangle, SubArmbar, SubOmoplata}
	default:
		return nil
	}
}

// BottomSubmissionsForPosition returns the canonical bottom-attack
// subs available to the Controlled side of the given position. The
// "reversal sub" set — sparser than top-attack because being
// controlled is supposed to be disadvantageous.
func BottomSubmissionsForPosition(s State) []SubmissionType {
	switch s {
	case Mount:
		// Mount-bottom: hipping up to catch the attacking arm.
		return []SubmissionType{SubTriangle, SubArmbar}
	case SideControl:
		// SideControl-bottom: snatching the wrist for a kimura.
		return []SubmissionType{SubKimura}
	case NorthSouth:
		return []SubmissionType{SubKimura}
	case HalfGuard:
		// HalfGuard-bottom: kimura on the trapped arm.
		return []SubmissionType{SubKimura}
	default:
		return nil
	}
}

// IsTopSubEligible reports whether the controller side of a pair
// can attempt a submission at the given position + control level.
// Requires top-attack subs available AND ControlLevel of InControl
// or LosingControl.
func IsTopSubEligible(s State, cl ControlLevel) bool {
	if len(TopSubmissionsForPosition(s)) == 0 {
		return false
	}
	return cl == InControl || cl == LosingControl
}

// IsBottomSubEligible reports whether the controlled side of a pair
// can attempt a reversal submission at the given position + control
// level. Requires bottom-attack subs available AND ControlLevel of
// Controlled or BecomingControlled.
func IsBottomSubEligible(s State, cl ControlLevel) bool {
	if len(BottomSubmissionsForPosition(s)) == 0 {
		return false
	}
	return cl == Controlled || cl == BecomingControlled
}

// CrippleBodyPart returns the body part broken by a successful
// cripple-policy outcome for the given sub type. Chokes return "" —
// they don't break body parts, and cripple-policy degrades to
// subdue when the sub is a choke (see ResolveSubmissionOutcome).
func CrippleBodyPart(t SubmissionType) string {
	switch t {
	case SubArmbar, SubOmoplata:
		return "arm"
	case SubKimura, SubAmericana:
		return "shoulder"
	default:
		// Chokes (RNC, Triangle, Anaconda) — no body part
		return ""
	}
}

// SubmissionToVerb returns the third-person verb phrase for
// narration ("isolates and hyper-extends your arm" for an armbar,
// etc.). Used by Position_Messaging.go.
func SubmissionToVerb(t SubmissionType) string {
	switch t {
	case SubArmbar:
		return "isolates and hyper-extends the arm"
	case SubRNC:
		return "wraps an arm around the neck"
	case SubTriangle:
		return "wraps the legs around the neck and arm"
	case SubAmericana:
		return "cranks the shoulder backwards"
	case SubKimura:
		return "twists the shoulder into an unnatural angle"
	case SubOmoplata:
		return "traps the shoulder with the leg"
	case SubAnaconda:
		return "rolls and wraps the neck"
	default:
		return "applies a submission"
	}
}
