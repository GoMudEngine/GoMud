package dialogue

// moodCache maps mob instance IDs to their current mood.
var moodCache = map[int]Mood{}

// GetMood returns the current mood for a mob instance.
// Falls back to defaultMood, then MoodNeutral if not set.
func GetMood(mobInstanceId int, defaultMood string) Mood {
	if mood, ok := moodCache[mobInstanceId]; ok {
		return mood
	}
	if defaultMood != "" {
		return Mood(defaultMood)
	}
	return MoodNeutral
}

// SetMood directly sets a mob instance's mood.
func SetMood(mobInstanceId int, mood Mood) {
	moodCache[mobInstanceId] = mood
}

// ShiftMood transitions a mob's mood to the target string if non-empty.
func ShiftMood(mobInstanceId int, target string, defaultMood string) {
	if target == "" {
		return
	}
	moodCache[mobInstanceId] = Mood(target)
}
