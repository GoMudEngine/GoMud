package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

type notAGainedEvent struct{}

func (notAGainedEvent) Type() string { return "dummy" }

func TestMutationGainedRevealWrongEventType(t *testing.T) {
	assert.Equal(t, events.Continue, MutationGained_Reveal(notAGainedEvent{}))
}

func TestDeepenFlourishText(t *testing.T) {
	got := deepenFlourishText("Chameleon Skin", 2, 3)
	assert.Contains(t, got, "Chameleon Skin")
	assert.Contains(t, got, "Level 2")
	assert.NotContains(t, got, "fully matured")

	got = deepenFlourishText("Chameleon Skin", 3, 3)
	assert.Contains(t, got, "fully matured")
}

func TestRevealCaption(t *testing.T) {
	got := revealCaption("Chameleon Skin", "Your skin drinks the colors around it.")
	// The caption is the screen-reader / degraded path — plain text, no ansi tags.
	assert.Contains(t, got, "A mutation emerges: Chameleon Skin.")
	assert.Contains(t, got, "Your skin drinks the colors around it.")
	assert.NotContains(t, got, "<ansi")
}

func TestFlattenDescription(t *testing.T) {
	in := "Your breathing reorganizes into a lattice of insectile spiracles --\ntireless,\nand indifferent to bad air.\n"
	got := flattenDescription(in)
	assert.NotContains(t, got, "\n")
	assert.Contains(t, got, "spiracles -- tireless, and indifferent")
}
