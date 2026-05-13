package state

// ScheduleAt is an opaque scheduled-time handle. RoundScheduler
// produces these; Machine consumes them.
type ScheduleAt struct {
	scheduler *RoundScheduler
	round     int
}

// RoundScheduler fires scheduled transitions when its round
// counter advances past their target rounds.
//
// The engine wires the master round tick to RoundScheduler.Tick().
// Per-character RoundSchedulers do not exist — there's one
// global scheduler that all machines register against.
type RoundScheduler struct {
	currentRound int
	pending      []scheduledTransition
}

type scheduledTransition struct {
	round     int
	fire      func()
	cancelled *bool // pointer because copies of the entry share cancel state
}

// NewRoundScheduler returns a fresh scheduler at round 0.
func NewRoundScheduler() *RoundScheduler {
	return &RoundScheduler{}
}

// RoundsFromNow returns a ScheduleAt that fires after n round
// ticks (where 0 means "the next tick fires it").
func (s *RoundScheduler) RoundsFromNow(n int) ScheduleAt {
	return ScheduleAt{scheduler: s, round: s.currentRound + n}
}

// Tick advances the scheduler by one round and fires any
// scheduled transitions whose target round <= currentRound.
func (s *RoundScheduler) Tick() {
	s.currentRound++
	keep := s.pending[:0]
	for _, p := range s.pending {
		if *p.cancelled {
			continue // dropped
		}
		if p.round <= s.currentRound {
			p.fire()
			continue
		}
		keep = append(keep, p)
	}
	s.pending = keep
}

// ScheduleTransition registers a deferred TransitionTo.
// When the scheduler's round counter reaches at.round, the
// transition fires. Reuse of ScheduleTransition without
// CancelScheduled adds a second pending transition — callers
// who want at-most-one-pending must call CancelScheduled first.
//
// On the configured firing round, the framework calls
// TransitionTo with the supplied target and reason. If the
// transition is vetoed at fire time, the state remains
// unchanged; no error is returned (no caller to receive it).
// Cascades and observers fire normally on success.
func (m *Machine[S]) ScheduleTransition(to S, at ScheduleAt, reason TransitionReason) {
	if at.scheduler == nil {
		return
	}
	cancelled := false
	m.mu.Lock()
	m.cancelTokens = append(m.cancelTokens, &cancelled)
	m.mu.Unlock()
	at.scheduler.pending = append(at.scheduler.pending, scheduledTransition{
		round:     at.round,
		cancelled: &cancelled,
		fire: func() {
			_ = m.TransitionTo(to, reason)
		},
	})
}

// CancelScheduled marks all pending scheduled transitions on
// this machine as cancelled. Tick() will skip them.
func (m *Machine[S]) CancelScheduled() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.cancelTokens {
		*c = true
	}
	m.cancelTokens = nil
}
