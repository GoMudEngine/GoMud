package dialogue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMatchWithFallbackInfo verifies the usedFallback flag: false when a
// specific keyword pattern matches, true when only the empty-keyword catch-all
// responds. This is what lets `ask <npc> quest` give a clear "no task" answer
// instead of generic filler (B2).
func TestMatchWithFallbackInfo(t *testing.T) {
	df := &DialogueFile{
		MobId: 8801, Zone: "test", DefaultMood: "neutral",
		Patterns: []Pattern{
			{Keywords: []string{"charm"}, Responses: []string{"Mind the charms."}},
			{Keywords: []string{""}, Responses: []string{"Speak up."}},
		},
	}

	resp, _, ok, usedFallback := MatchWithFallbackInfo(df, 8801, "charm", nil)
	assert.True(t, ok)
	assert.False(t, usedFallback, "a specific keyword match is not the fallback")
	assert.Equal(t, "Mind the charms.", resp)

	resp2, _, ok2, usedFallback2 := MatchWithFallbackInfo(df, 8801, "quest", nil)
	assert.True(t, ok2)
	assert.True(t, usedFallback2, "an unmatched topic falls to the catch-all")
	assert.Equal(t, "Speak up.", resp2)

	delete(moodCache, 8801)
}
