package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exactAdapter returns a Match only when the candidate equals want.
func exactAdapter(want string, kind Kind) adapter {
	return func(_ Scope, candidate string) (Match, bool) {
		if candidate == want {
			return Match{Kind: kind, Name: candidate}, true
		}
		return Match{}, false
	}
}

func TestResolveWith_LongestSpanWins(t *testing.T) {
	// Both a 2-word and a 1-word adapter can match; the 2-word span must win.
	adapters := []adapter{
		exactAdapter("hare paths", KindNoun),
		exactAdapter("hare", KindNoun),
	}
	m, ok := resolveWith([]string{"hare", "paths"}, Scope{}, adapters)
	require.True(t, ok)
	assert.Equal(t, "hare paths", m.Name)
}

func TestResolveWith_FallsBackToShorterSpan(t *testing.T) {
	adapters := []adapter{exactAdapter("hare", KindNoun)}
	m, ok := resolveWith([]string{"hare", "zzz"}, Scope{}, adapters)
	require.True(t, ok)
	assert.Equal(t, "hare", m.Name) // "hare zzz" misses, "hare" hits
}

func TestResolveWith_KindPriorityBreaksTies(t *testing.T) {
	// Two adapters match the SAME span; the first in order wins.
	adapters := []adapter{
		exactAdapter("bank clerk", KindMob),
		exactAdapter("bank clerk", KindNoun),
	}
	m, ok := resolveWith([]string{"bank", "clerk"}, Scope{}, adapters)
	require.True(t, ok)
	assert.Equal(t, KindMob, m.Kind)
}

func TestResolveWith_NoMatch(t *testing.T) {
	adapters := []adapter{exactAdapter("nope", KindNoun)}
	_, ok := resolveWith([]string{"totally", "absent"}, Scope{}, adapters)
	assert.False(t, ok)
}
