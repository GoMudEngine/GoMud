package dialogue

// 5a: NPC greetings — the ambient welcome an NPC offers when a player arrives
// in its room. Selection and frequency live here; the arrival hook that calls
// them is in internal/usercommands/go.go beside the conversation boost.

// PickGreeting selects the greeting an NPC offers for its current mood:
// the first whose Moods contains the mood, else the first untagged line,
// else nothing — a line written for a mood the NPC is not in stays unsaid.
// Comparison is direct Mood conversion, the same idiom the pattern matcher
// uses (engine.go); no normalization exists in this package.
func PickGreeting(gs []Greeting, currentMood Mood) (string, bool) {
	for _, g := range gs {
		for _, m := range g.Moods {
			if Mood(m) == currentMood {
				return g.Text, true
			}
		}
	}
	for _, g := range gs {
		if len(g.Moods) == 0 {
			return g.Text, true
		}
	}
	return "", false
}

// HasGreeted reports whether this mob instance already greeted this player.
func HasGreeted(mobInstanceId, userId int) bool {
	return GetMemory(mobInstanceId, userId).Greeted
}

// MarkGreeted records the greeting. In-process memory, so the horizon is one
// server boot — and a respawned mob (new instance id) greets afresh, which
// reads correctly for a killed-and-returned shopkeeper.
func MarkGreeted(mobInstanceId, userId int) {
	GetMemory(mobInstanceId, userId).Greeted = true
}
