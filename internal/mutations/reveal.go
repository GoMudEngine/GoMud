package mutations

// Gained is the player-facing mutation-reveal event (spec 2026-07-29).
// It deliberately does NOT import internal/events — events → skills →
// mutations would cycle — and satisfies events.Event structurally via
// Type(). Emit with events.AddToQueue(mutations.Gained{...}) from call
// sites (they all already import events).
type Gained struct {
	UserId     int
	MutationId string
	Rank       int  // new rank after the change (1 for acquisitions)
	IsNew      bool // false = an owned mutation deepened
}

func (Gained) Type() string { return `MutationGained` }
