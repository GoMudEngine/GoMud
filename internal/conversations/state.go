package conversations

// conversationPlan describes what the per-tick state machine wants to
// do for a single NPC. Pure-data, easy to unit-test.
type conversationPlan struct {
	ShouldFire   bool   // fire `say <text>` from this NPC this tick
	ShouldAbort  bool   // abort the conversation (out-of-range line_idx, etc.)
	Text         string // the text to say (set when ShouldFire)
	NextLineIdx  int    // shared line counter value AFTER this firing (set when ShouldFire)
	IsFinalLine  bool   // true when this firing completes the exchange (set when ShouldFire)
}

// computeConversationPlan is the pure decision logic for a single NPC's
// tick: given the exchange, the NPC's role ("A" or "B"), and the shared
// line index, decide whether to fire, wait, or abort.
func computeConversationPlan(ex Exchange, role string, lineIdx int) conversationPlan {
	if lineIdx < 0 || lineIdx >= len(ex.Lines) {
		return conversationPlan{ShouldAbort: true}
	}
	line := ex.Lines[lineIdx]
	if line.Speaker != role {
		// Not my turn — wait silently.
		return conversationPlan{}
	}
	return conversationPlan{
		ShouldFire:  true,
		Text:        line.Text,
		NextLineIdx: lineIdx + 1,
		IsFinalLine: lineIdx == len(ex.Lines)-1,
	}
}
