package combat

// DefenseType represents the type of defense used (Stage 7.1)
type DefenseType string

const (
	DefenseNone  DefenseType = ""
	DefenseDodge DefenseType = "dodge"
	DefenseParry DefenseType = "parry"
	DefenseBlock DefenseType = "block"
)

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
	DamageToSource          int          // defaults 0
	DamageToSourceReduction int          // defaults 0
	DefenseUsed             DefenseType  // Which defense avoided the hit (Stage 7.1)
	DefenseAttempts         []DefenseType // Sequence of defenses attempted (Stage 7.1)
	DefenseZScore           float64      // Defense roll z-score (Stage 8.4)
	AttackZScore            float64      // Attack roll z-score (Stage 8.4)
	ParryCritDetected       bool         // Flag for parry crit (Stage 8.4)
	DodgeCritDetected       bool         // Flag for dodge crit (Stage 8.4)
	SwingEvents             []SwingEvent // Per-swing analytics (Stage 30.2)
	MessagesToSource        []string
	MessagesToTarget        []string
	MessagesToSourceRoom    []string
	MessagesToTargetRoom    []string
	MessagesToRoomOld       []string
}

func (a *AttackResult) SendToSource(msg string) {
	a.MessagesToSource = append(a.MessagesToSource, msg)
}

func (a *AttackResult) SendToTarget(msg string) {
	a.MessagesToTarget = append(a.MessagesToTarget, msg)
}

func (a *AttackResult) SendToSourceRoom(msg string) {
	a.MessagesToSourceRoom = append(a.MessagesToSourceRoom, msg)
}

func (a *AttackResult) SendToTargetRoom(msg string) {
	a.MessagesToTargetRoom = append(a.MessagesToTargetRoom, msg)
}

func (a *AttackResult) SendToRoomOld(msg string) {
	a.MessagesToRoomOld = append(a.MessagesToRoomOld, msg)
}
